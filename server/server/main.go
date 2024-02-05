package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go-diploma/internal/accrual"
	"go-diploma/internal/utils/hash/sha1hash"
	"go-diploma/internal/utils/luhnalgorithm"
	"go-diploma/server/compress/gzipapp"
	"go-diploma/server/config"
	"go-diploma/server/cookie"
	"go-diploma/server/storage/database"
	"go.uber.org/zap"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

type Server struct {
	Routers         chi.Router
	Config          config.Config
	Logger          *zap.Logger
	DB              database.Database
	Http            http.Server
	Accrual         accrual.Accrual
	ShutdownProcess bool
}

func (s *Server) New(c config.Config, l *zap.Logger) error {
	var err error
	s.Config = c
	s.Routers = chi.NewRouter()
	s.Logger = l
	err = s.DB.Init(context.Background(), c.DatabaseConnection)
	s.Http = http.Server{Addr: s.Config.MartAddress, Handler: s.Routers}
	s.Accrual = accrual.Accrual{}
	s.ShutdownProcess = false
	return err
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
		})
	})

	accrualPath := filepath.Join(s.Config.LocalConfig.App.RootPath, s.Config.LocalConfig.App.AccrualPath)
	err = s.Accrual.Init(s.Config.AccrualAddress, s.Config.DatabaseConnection, accrualPath)
	if err != nil {
		return err
	}
	err = s.Accrual.Start()
	if err != nil {
		return err
	}

	err = s.Http.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop() error {
	var err error
	shutdownCtx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = s.Http.Shutdown(shutdownCtx)
	err = s.Accrual.Stop()
	s.DB.Close()
	return err
}

func (s *Server) Shutdown(res http.ResponseWriter, req *http.Request) {
	s.ShutdownProcess = true
	res.WriteHeader(http.StatusOK)
	s.Stop()
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
	user.pwdHash = sha1hash.Hash(user.Password)
	s.Logger.Debug(user.pwdHash)

	_, err = s.DB.Pool.Exec(
		req.Context(),
		`insert into public.users (login, password_hash) values ($1, $2)`,
		user.Login, user.pwdHash,
	)
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

	row := s.DB.Pool.QueryRow(
		req.Context(),
		`select id from public.users where login = $1`,
		user.Login,
	)
	var userID int
	err = row.Scan(&userID)
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
	user.pwdHash = sha1hash.Hash(user.Password)
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

	contentBody, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		s.Logger.Error(err.Error())
		http.Error(res, `inconsistent body`, http.StatusBadRequest)
		return
	}

	orderNum := string(contentBody)

	isLuhn, err := luhnalgorithm.IsLuhnValid(orderNum)
	if err != nil {
		err := fmt.Errorf(`luhn calc error: %w`, err)
		s.Logger.Error(err.Error())
		http.Error(res, `bad request`, http.StatusBadRequest)
		return
	}
	if !isLuhn {
		err := errors.New(`wrong format`)
		s.Logger.Error(err.Error())
		http.Error(res, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// get by order
	// if order exists and user is equal - return 200
	// if order exists and user is not equal - return 409
	// if not exists:
	// save to db
	// get from accrual info about order
	//

	s.Logger.Info(`successfully saved`)
	res.WriteHeader(http.StatusOK)
}

// GetOrders
// Получение текущего баланса пользователя
func (s *Server) GetOrders(res http.ResponseWriter, req *http.Request) {

}

// GetBalance
// Получение текущего баланса пользователя
func (s *Server) GetBalance(res http.ResponseWriter, req *http.Request) {

}

// Withdraw
// Запрос на списание средств
func (s *Server) Withdraw(res http.ResponseWriter, req *http.Request) {

}

// Withdrawals
// Запрос на списание средств
func (s *Server) Withdrawals(res http.ResponseWriter, req *http.Request) {

}
