package accrual

import (
	"bytes"
	"encoding/json"
	"errors"
	"go-diploma/server/config"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

type Accrual struct {
	Address string
	DSN     string
	BinPath string
	Pid     int
}

const protocol = `http://`

var ErrEmptyAddr = errors.New(`host or port not found`)

func ValidateURL(str string) (string, error) {
	if str[0:4] != `http` {
		str = protocol + str
	}
	u, err := url.Parse(str)
	if err != nil {
		return str, err
	}
	if u.Host == `` {
		return u.Host, ErrEmptyAddr
	}
	return u.Host, nil
}

/**
 * Создаем объект для работы с accrual.
 */

func (a *Accrual) Init(address string, dsn string, binPath string) error {
	var err error
	a.Address, err = ValidateURL(address)
	a.DSN = dsn
	a.BinPath = binPath
	if err != nil {
		return errors.New(`address is not valid`)
	}
	return nil
}

func (a *Accrual) Start() error {
	cmd := exec.Command(
		a.BinPath,
		`-a`, a.Address,
		`-d`, a.DSN,
	)
	err := cmd.Start()
	if err != nil {
		return err
	}
	a.Pid = cmd.Process.Pid

	return nil
}
func (a *Accrual) Stop() error {
	p, err := os.FindProcess(a.Pid)
	if err != nil {
		return err
	}
	err = p.Signal(syscall.SIGKILL)
	if err != nil {
		return err
	}
	return nil
}

/**
 * Получаем инфо по заказу по его ID.
 */

var ErrUnexpectedResponse = errors.New(`unexpected response`)

func (a *Accrual) GetOrderInfo(orderNum string) (config.GetOrderData, error) {
	accrualURL := protocol + a.Address + `/api/orders/` + orderNum
	request, err := http.NewRequest("GET", accrualURL, nil)
	if err != nil {
		return config.GetOrderData{}, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return config.GetOrderData{}, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return config.GetOrderData{}, err
	}

	if response.StatusCode != http.StatusOK {
		return config.GetOrderData{}, ErrUnexpectedResponse
	}

	var resp config.GetOrderData
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return config.GetOrderData{}, err
	}

	return resp, nil
}

/**
 * Сохраняем заказ
 */

func (a *Accrual) SetOrderInfo(data config.SetOrderData) error {
	marshaledOrder, err := json.Marshal(data)
	if err != nil {
		return err
	}

	accrualURL := protocol + a.Address + `/api/orders`
	request, err := http.NewRequest("POST", accrualURL, bytes.NewBuffer(marshaledOrder))
	if err != nil {
		return err
	}
	request.Header.Set("content-type", "application/json")
	request.Header.Set("cache-control", "no-cache")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusAccepted {
		return errors.New(`unexpected response: ` + strconv.Itoa(response.StatusCode))
	}
	err = response.Body.Close()
	if err != nil {
		return err
	}

	return nil
}

/**
 * Сохраняем данные для расчета вознаграждения за заказ
 */

func (a *Accrual) SetNewAccrualType(data config.NewAccrualType) error {
	marshaledAccType, err := json.Marshal(data)
	if err != nil {
		return err
	}

	accrualURL := protocol + a.Address + `/api/goods`
	request, err := http.NewRequest("POST", accrualURL, bytes.NewBuffer(marshaledAccType))
	if err != nil {
		return err
	}
	request.Header.Set("content-type", "application/json")
	request.Header.Set("cache-control", "no-cache")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return errors.New(`unexpected response: ` + strconv.Itoa(response.StatusCode))
	}
	err = response.Body.Close()
	if err != nil {
		return err
	}

	return nil
}

func (a *Accrual) Prepare(orders []config.SetOrderData, types []config.NewAccrualType) error {
	for _, order := range orders {
		err := a.SetOrderInfo(order)
		if err != nil {
			return err
		}
	}

	for _, singleType := range types {
		err := a.SetNewAccrualType(singleType)
		if err != nil {
			return err
		}
	}

	return nil
}
