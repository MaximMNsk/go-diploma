package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go-diploma/internal/accrual"
	"go-diploma/internal/utils/hash/sha1hash"
	"go-diploma/server/compress/gzipapp"
	"go-diploma/server/config"
	"go-diploma/server/cookie"
	"go-diploma/server/storage/database"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

type Server struct {
	Routers         chi.Router
	Config          config.Config
	Logger          *zap.Logger
	DB              database.Database
	HTTP            http.Server
	Accrual         accrual.Accrual
	StopChan        chan struct{}
	ShutdownProcess bool
}

func (s *Server) New(c config.Config, l *zap.Logger) error {
	var err error
	s.Config = c
	s.Routers = chi.NewRouter()
	s.Logger = l
	err = s.DB.Init(context.Background(), c.DatabaseConnection)
	if err != nil {
		return err
	}
	s.HTTP = http.Server{Addr: s.Config.MartAddress, Handler: s.Routers}
	accrualPath := filepath.Join(s.Config.LocalConfig.App.RootPath, s.Config.LocalConfig.App.AccrualPath)
	err = s.Accrual.Init(s.Config.AccrualAddress, s.Config.DatabaseConnection, accrualPath)
	if err != nil {
		return err
	}
	s.ShutdownProcess = false
	return nil
}

func (s *Server) Start() error {
	err := s.DB.PrepareDB()
	if err != nil {
		return err
	}

	s.Routers.With(gzipapp.GzipHandler)
	s.Routers.Route(`/`, func(r chi.Router) {
		s.Routers.Group(func(r chi.Router) {
			r.Post(`/app/shutdown`, s.Shutdown)
			r.Post(`/api/user/register`, s.UserRegister)
			r.Post(`/api/user/login`, s.UserLogin)
		})
		s.Routers.Group(func(r chi.Router) {
			r.Use(cookie.AuthChecker)
			r.Post(`/api/user/orders`, s.SaveOrder)
			r.Get(`/api/user/orders`, s.GetOrders)
			r.Get(`/api/user/balance`, s.GetBalance)
			r.Post(`/api/user/balance/withdraw`, s.Withdraw)
			r.Get(`/api/user/withdrawals`, s.Withdrawals)
		})
	})

	if s.Config.Mode == `full` {
		err = s.Accrual.Start()
		if err != nil {
			return err
		}
		err = s.Accrual.Prepare(s.Config.LocalConfig.Accrual.Orders, s.Config.LocalConfig.Accrual.Goods)
		if err != nil {
			return err
		}
	}
	go s.StartUpdateBackground()
	err = s.HTTP.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop() error {
	var err error
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = s.HTTP.Shutdown(shutdownCtx)
	if err != nil {
		s.Logger.Error(err.Error())
	}
	s.StopUpdateBackground()
	err = s.Accrual.Stop()
	if err != nil {
		s.Logger.Error(err.Error())
	}
	s.DB.Close()
	return nil
}

func (s *Server) Shutdown(res http.ResponseWriter, req *http.Request) {
	s.ShutdownProcess = true
	res.WriteHeader(http.StatusOK)
	_ = s.Stop()
}

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	pwdHash  string
}

// UserRegister
// Регистрация пользователя
func (s *Server) UserRegister(res http.ResponseWriter, req *http.Request) {
	if s.ShutdownProcess {
		http.Error(res, "503 service unavailable", http.StatusServiceUnavailable)
		return
	}

	contentBody, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `inconsistent body`, http.StatusBadRequest)
		return
	}

	var user User
	err = json.Unmarshal(contentBody, &user)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `inconsistent request`, http.StatusBadRequest)
		return
	}
	user.pwdHash, err = sha1hash.Hash(user.Password)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `inconsistent request`, http.StatusBadRequest)
		return
	}

	s.Logger.Debug(user.pwdHash)

	var userID int
	row := s.DB.Pool.QueryRow(
		req.Context(),
		`insert into public.users (login, password_hash) values ($1, $2) returning id`,
		user.Login, user.pwdHash,
	)
	err = row.Scan(&userID)
	if err != nil {
		var insertErr *pgconn.PgError
		if errors.As(err, &insertErr) {
			if insertErr.Code == `23505` {
				s.Logger.Warn(err.Error())
				http.Error(res, `duplicate user`, http.StatusConflict)
				return
			}
		}
		s.Logger.Error(err.Error())
		http.Error(res, `internal error`, http.StatusInternalServerError)
		return
	}

	_, err = s.DB.Pool.Exec(
		req.Context(),
		`insert into public.accruals (user_id) values ($1)`,
		userID,
	)

	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `internal error`, http.StatusInternalServerError)
		return
	}

	s.Logger.Info(`user saved`)

	jwtString, err := cookie.BuildJWTString(userID)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `internal error`, http.StatusInternalServerError)
		return
	}

	regCookie := &http.Cookie{
		Name:    `token`,
		Value:   jwtString,
		Expires: time.Now().Add(cookie.TokenExp),
		Path:    `/`,
	}
	http.SetCookie(res, regCookie)

	s.Logger.Info(`user successfully created`)
	res.WriteHeader(http.StatusOK)
}

// UserLogin
// Аутентификация пользователя
func (s *Server) UserLogin(res http.ResponseWriter, req *http.Request) {
	if s.ShutdownProcess {
		http.Error(res, "503 service unavailable", http.StatusServiceUnavailable)
		return
	}

	contentBody, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `inconsistent body`, http.StatusBadRequest)
		return
	}

	var user User
	err = json.Unmarshal(contentBody, &user)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `inconsistent request`, http.StatusBadRequest)
		return
	}
	user.pwdHash, err = sha1hash.Hash(user.Password)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `inconsistent request`, http.StatusBadRequest)
		return
	}
	s.Logger.Debug(user.pwdHash)

	row := s.DB.Pool.QueryRow(
		req.Context(),
		`select id, password_hash from public.users where login = $1`,
		user.Login,
	)
	var pwdHash string
	var userID int
	err = row.Scan(&userID, &pwdHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.Logger.Warn(`Decline user authority`)
			http.Error(res, `unauthorized`, http.StatusUnauthorized)
			return
		}
		s.Logger.Error(err.Error())
		http.Error(res, `internal error`, http.StatusInternalServerError)
		return

	}

	if user.pwdHash != pwdHash {
		s.Logger.Warn(`Decline user authority`)
		http.Error(res, `unauthorized`, http.StatusUnauthorized)
		return
	}

	jwtString, err := cookie.BuildJWTString(userID)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `internal error`, http.StatusInternalServerError)
		return
	}

	regCookie := &http.Cookie{
		Name:    `token`,
		Value:   jwtString,
		Expires: time.Now().Add(cookie.TokenExp),
		Path:    `/`,
	}
	http.SetCookie(res, regCookie)

	s.Logger.Info(`user successfully authorized`)
	res.WriteHeader(http.StatusOK)
}

// SaveOrder
// Загрузка номера заказа
func (s *Server) SaveOrder(res http.ResponseWriter, req *http.Request) {
	if s.ShutdownProcess {
		http.Error(res, "503 service unavailable", http.StatusServiceUnavailable)
		return
	}

	userID := req.Context().Value(cookie.UserNum(`UserID`)).(int)

	contentBody, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `inconsistent body`, http.StatusBadRequest)
		return
	}

	orderNum := string(contentBody)
	err = goluhn.Validate(orderNum)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `inconsistent body`, http.StatusUnprocessableEntity)
		return
	}

	var savedOrderUserID int
	err = s.DB.Pool.QueryRow(
		req.Context(),
		`select user_id from public.orders where number = $1`,
		orderNum,
	).Scan(&savedOrderUserID)

	isSavedOrder := true

	if savedOrderUserID == 0 {
		isSavedOrder = false
		s.Logger.Debug(`saving is allowed`)
		s.Logger.Info(`saved order user id: ` + strconv.Itoa(savedOrderUserID))
	}

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.Logger.Error(err.Error())
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if userID == savedOrderUserID && isSavedOrder {
		s.Logger.Warn(`order was uploaded by current user`)
		res.WriteHeader(http.StatusOK)
		return
	}

	if userID != savedOrderUserID && isSavedOrder {
		s.Logger.Warn(`order was uploaded by another user`)
		http.Error(res, `another user`, http.StatusConflict)
		return
	}

	isEmptyAccrual := false
	accrualData, err := s.Accrual.GetOrderInfo(orderNum)
	if err != nil {
		isEmptyAccrual = true
		s.Logger.Warn(err.Error())
	}

	if isEmptyAccrual {
		_, err = s.DB.Pool.Exec(
			req.Context(),
			`insert into public.orders (user_id, number) values ($1, $2)`,
			userID,
			orderNum,
		)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		_, err = s.DB.Pool.Exec(
			req.Context(),
			`insert into public.orders (user_id, number, status, accrual) values ($1, $2, $3, $4)`,
			userID,
			orderNum,
			accrualData.Status,
			accrualData.Accrual,
		)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if !isEmptyAccrual {
		_, err = s.DB.Pool.Exec(
			req.Context(),
			`update public.accruals
				 set
	                      current_balance = current_balance + $2
				where
				    user_id = $1`,
			userID,
			accrualData.Accrual,
		)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	s.Logger.Info(`successfully saved order: ` + orderNum)
	res.WriteHeader(http.StatusAccepted)
}

// GetOrders
// Получение текущего баланса пользователя
func (s *Server) GetOrders(res http.ResponseWriter, req *http.Request) {
	if s.ShutdownProcess {
		http.Error(res, "503 service unavailable", http.StatusServiceUnavailable)
		return
	}

	userID := req.Context().Value(cookie.UserNum(`UserID`)).(int)

	rows, err := s.DB.Pool.Query(req.Context(),
		`select 
    			number, 
    			status, 
    			round(cast(accrual as numeric), 2) as accrual, 
    			uploaded_at 
			from public.orders 
			where user_id = $1`,
		userID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.Logger.Error(err.Error())
			http.Error(res, "internal error", http.StatusInternalServerError)
			return
		}
		s.Logger.Error(err.Error())
		http.Error(res, "", http.StatusNoContent)
		return
	}

	var savedOrders []config.GetOrderData
	for rows.Next() {
		var savedOrder config.GetOrderData
		err := rows.Scan(&savedOrder.OrderNum, &savedOrder.Status, &savedOrder.Accrual, &savedOrder.UploadedAt)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(res, "internal error", http.StatusInternalServerError)
			return
		}
		savedOrders = append(savedOrders, savedOrder)
	}

	marshaledOrders, err := json.Marshal(savedOrders)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, "internal error", http.StatusInternalServerError)
		return
	}

	res.Header().Add(`Content-Type`, `application/json`)
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(marshaledOrders)
	if err != nil {
		s.Logger.Error(err.Error())
		return
	}
	s.Logger.Info(`success GetOrders`)
}

type Balance struct {
	Balance   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

// GetBalance
// Получение текущего баланса пользователя
func (s *Server) GetBalance(res http.ResponseWriter, req *http.Request) {
	if s.ShutdownProcess {
		http.Error(res, "503 service unavailable", http.StatusServiceUnavailable)
		return
	}

	userID := req.Context().Value(cookie.UserNum(`UserID`)).(int)

	var bal Balance
	err := s.DB.Pool.QueryRow(
		req.Context(),
		`select round(cast(current_balance as numeric), 2) as current_balance, 
       				round(cast(total_withdrawn as numeric), 2) as total_withdrawn 
			from public.accruals 
			where user_id = $1`,
		userID,
	).Scan(&bal.Balance, &bal.Withdrawn)
	if err != nil {
		s.Logger.Error(err.Error())
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(res, "", http.StatusNoContent)
		} else {
			http.Error(res, "", http.StatusInternalServerError)
		}
		return
	}

	marshaled, err := json.Marshal(bal)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	res.Header().Add(`Content-Type`, `application/json`)
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(marshaled)
	if err != nil {
		s.Logger.Error(err.Error())
		return
	}
	s.Logger.Info(`success GetBalance`)
}

// Withdraw
// Запрос на списание средств
func (s *Server) Withdraw(res http.ResponseWriter, req *http.Request) {
	if s.ShutdownProcess {
		http.Error(res, "503 service unavailable", http.StatusServiceUnavailable)
		return
	}

	userID := req.Context().Value(cookie.UserNum(`UserID`)).(int)

	contentBody, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, ``, http.StatusInternalServerError)
		return
	}

	var w Withdrawal
	err = json.Unmarshal(contentBody, &w)
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, ``, http.StatusInternalServerError)
		return
	}

	s.Logger.Info(`try to withdraw sum: ` + strconv.Itoa(int(w.Sum)) + ` by order: ` + w.Order)

	var acc float32
	err = s.DB.Pool.QueryRow(req.Context(),
		//`select round(cast(accrual as numeric), 2) as accrual from public.orders where number = $1`,
		//w.Order,
		`select round(cast(current_balance as numeric), 2) as accrual from accruals where user_id = $1`,
		userID,
	).Scan(&acc)
	if err != nil {
		s.Logger.Error(err.Error())
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(res, `no orders`, http.StatusUnprocessableEntity)
			return
		} else {
			http.Error(res, ``, http.StatusInternalServerError)
			return
		}
	}

	if w.Sum > acc {
		s.Logger.Error(`insufficient funds`)
		http.Error(res, `insufficient funds`, http.StatusPaymentRequired)
		return
	}

	tx, err := s.DB.Pool.Begin(req.Context())
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, ``, http.StatusInternalServerError)
		return
	}

	eg := errgroup.Group{}

	eg.Go(func() error {
		//_, err = tx.Exec(req.Context(), `update public.orders set accrual = accrual-$1 where number = $2`,
		//	w.Sum, w.Order)
		//if err != nil {
		//	return err
		//}
		_, err = tx.Exec(
			req.Context(),
			`insert into public.withdrawals (user_id, sum, order_number)
					values ($1, $2, $3)`,
			userID, w.Sum, w.Order)
		if err != nil {
			return err
		}
		_, err = tx.Exec(req.Context(),
			`update public.accruals set current_balance = current_balance-$1,
	                      total_withdrawn = total_withdrawn+$2
	                  where user_id = $3`,
			w.Sum, w.Sum, userID)
		if err != nil {
			return err
		}
		return nil
	})

	if err = eg.Wait(); err != nil {
		s.Logger.Error(err.Error())
		errRollback := tx.Rollback(req.Context())
		if errRollback != nil {
			s.Logger.Error(errRollback.Error())
		}
		http.Error(res, ``, http.StatusInternalServerError)
		return
	}
	if err = tx.Commit(req.Context()); err != nil {
		http.Error(res, ``, http.StatusInternalServerError)
		s.Logger.Error(err.Error())
	}

	res.WriteHeader(http.StatusOK)
}

type Withdrawal struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

// Withdrawals
// Получение информации о выводе средств
func (s *Server) Withdrawals(res http.ResponseWriter, req *http.Request) {
	if s.ShutdownProcess {
		http.Error(res, "503 service unavailable", http.StatusServiceUnavailable)
		return
	}

	userID := req.Context().Value(cookie.UserNum(`UserID`)).(int)

	rows, err := s.DB.Pool.Query(
		req.Context(),
		`select round(cast(sum as numeric), 2) as sum, 
       				order_number 
			from public.withdrawals 
			where user_id = $1`,
		userID,
	)
	if err != nil {
		s.Logger.Error(err.Error())
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(res, "no withdrawals", http.StatusNoContent)
		} else {
			http.Error(res, "", http.StatusInternalServerError)
		}
		return
	}

	var ws []Withdrawal
	for rows.Next() {
		var w Withdrawal
		err = rows.Scan(&w.Sum, &w.Order)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(res, "", http.StatusInternalServerError)
			return
		}
		ws = append(ws, w)
	}
	if len(ws) == 0 {
		s.Logger.Warn(`no Withdrawals for user: ` + strconv.Itoa(userID))
		http.Error(res, "no withdrawals", http.StatusNoContent)
		return
	}

	marshaled, err := json.Marshal(ws)
	if err != nil {
		s.Logger.Warn(err.Error())
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	res.Header().Add(`Content-Type`, `application/json`)
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(marshaled)
	if err != nil {
		s.Logger.Error(err.Error())
		return
	}
	s.Logger.Info(`success GetWithdrawals`)
}

/** TODO make funOut: отсюда порождать горутины, которые будут ходить в accrual с учетом 429 ответа оттуда */
func (s *Server) StartUpdateBackground() {
	s.Logger.Info(`start updater`)

	ctx := context.Background()
	s.StopChan = make(chan struct{})
	defer close(s.StopChan)

	for {
		var sleeper time.Duration
		sleeper = 1
		select {
		case <-s.StopChan:
			s.Logger.Debug(`stop background updater`)
			return
		default:
			time.Sleep(sleeper * time.Second)
		}

		unhandledOrders, err := s.GetUnhandledOrders()
		if err != nil {
			s.Logger.Warn(err.Error())
			continue
		}
		if len(unhandledOrders) == 0 {
			continue
		}

		for _, orderNum := range unhandledOrders {
			info, err := s.Accrual.GetOrderInfo(orderNum)
			if err != nil {
				s.Logger.Error(err.Error())
				sleeper = 5
				continue
			}

			tx, err := s.DB.Pool.Begin(ctx)
			if err != nil {
				s.Logger.Error(err.Error())
				continue
			}

			eg := errgroup.Group{}
			eg.Go(func() error {
				_, err = tx.Exec(ctx, `update public.orders set status = $1, accrual = $2 where number = $3`,
					info.Status, info.Accrual, orderNum)
				if err != nil {
					return err
				}
				_, err = tx.Exec(ctx,
					`update public.accruals set current_balance = current_balance + $1
                       		where user_id = (select user_id from public.orders where number = $2)`,
					info.Accrual, orderNum)
				if err != nil {
					return err
				}
				return nil
			})

			if err = eg.Wait(); err != nil {
				s.Logger.Error(err.Error())

				errRollback := tx.Rollback(ctx)
				if errRollback != nil {
					s.Logger.Error(errRollback.Error())
				}
				continue
			}
			if err = tx.Commit(ctx); err != nil {
				s.Logger.Error(err.Error())
			}

		}
	}
}

func (s *Server) StopUpdateBackground() {
	go func() {
		s.StopChan <- struct{}{}
	}()
}

type UnhandledOrders []string

func (s *Server) GetUnhandledOrders() (UnhandledOrders, error) {
	var unhandledOrders UnhandledOrders
	rows, err := s.DB.Pool.Query(
		context.Background(),
		`update public.orders set status = 'PROCESSING' where status in ('NEW', 'PROCESSING') returning number`,
	)
	emptySlice := make([]string, 0)
	if err != nil {
		return emptySlice, err
	}
	for rows.Next() {
		var unhandledOrder string
		err = rows.Scan(&unhandledOrder)
		if err != nil {
			return emptySlice, err
		}
		unhandledOrders = append(unhandledOrders, unhandledOrder)
	}

	return unhandledOrders, nil
}
