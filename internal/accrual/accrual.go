package accrual

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type Accrual struct {
	address string
}

func IsUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != "" && u.Port() != ""
}

/**
 * Создаем объект для работы с accrual
 */

func (a *Accrual) New(address string) error {
	if !IsUrl(address) {
		return errors.New(`address is not valid`)
	}
	a.address = address
	return nil
}

/**
 * Получаем инфо по заказу по его ID
 */

type GetOrderData struct {
	OrderNum string  `json:"order"`
	Status   string  `json:"status"`
	Accrual  float32 `json:"accrual,omitempty"`
}

func (a *Accrual) GetOrderInfo(orderNum string) (GetOrderData, error) {
	accrualUrl := a.address + `api/orders/` + orderNum
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

	accrualUrl := a.address + `api/orders`
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

	accrualUrl := a.address + `api/goods`
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
