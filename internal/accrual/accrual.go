package accrual

import (
	"bytes"
	"encoding/json"
	"errors"
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

const webScheme = `http://`

var ErrEmptyAddr = errors.New(`host or port not found`)

func ValidateURL(str string) (string, error) {
	if str[0:4] != `http` {
		str = webScheme + str
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

func AccrualDaemon(stop chan struct{}) {

}

/**
 * Получаем инфо по заказу по его ID.
 */

type GetOrderData struct {
	OrderNum string  `json:"order"`
	Status   string  `json:"status"`
	Accrual  float32 `json:"accrual,omitempty"`
}

func (a *Accrual) GetOrderInfo(orderNum string) (GetOrderData, error) {
	accrualUrl := webScheme + a.Address + `/api/orders/` + orderNum
	request, err := http.NewRequest("GET", accrualUrl, nil)
	if err != nil {
		return GetOrderData{}, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return GetOrderData{}, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return GetOrderData{}, err
	}

	if response.StatusCode != http.StatusOK {
		return GetOrderData{}, errors.New(`unexpected response: ` + strconv.Itoa(response.StatusCode))
	}

	var resp GetOrderData
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return GetOrderData{}, err
	}

	return resp, nil
}

/**
 * Сохраняем заказ
 */

type Good struct {
	Description string  `json:"description"`
	Price       float32 `json:"price"`
}
type SetOrderData struct {
	OrderNum string `json:"order"`
	Goods    []Good `json:"goods"`
}

func (a *Accrual) SetOrderInfo(data SetOrderData) error {
	marshaledOrder, err := json.Marshal(data)
	if err != nil {
		return err
	}

	accrualUrl := webScheme + a.Address + `/api/orders`
	request, err := http.NewRequest("POST", accrualUrl, bytes.NewBuffer(marshaledOrder))
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

	return nil
}

/**
 * Сохраняем данные для расчета вознаграждения за заказ
 */

type newAccrualType struct {
	Match      string  `json:"match"`
	Reward     float32 `json:"reward"`
	RewardType string  `json:"reward_type"`
}

func (a *Accrual) SetNewAccrualType(data newAccrualType) error {
	marshaledAccType, err := json.Marshal(data)
	if err != nil {
		return err
	}

	accrualUrl := webScheme + a.Address + `/api/goods`
	request, err := http.NewRequest("POST", accrualUrl, bytes.NewBuffer(marshaledAccType))
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

	return nil
}
