package accrual

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAccrual_SetNewAccrualType(t *testing.T) {
	type args struct {
		address string
		accType newAccrualType
	}

	tests := []struct {
		name      string
		arguments args
	}{
		{
			name: `Test set new accrual type`,
			arguments: args{
				address: `http://localhost:8081/`,
				accType: newAccrualType{
					Match:      `test`,
					Reward:     5,
					RewardType: `%`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var acc Accrual
			err := acc.New(tt.arguments.address)
			require.NoError(t, err)

			err = acc.SetNewAccrualType(tt.arguments.accType)
			require.NoError(t, err)
		})
	}
}

func TestAccrual_SetOrderInfo(t *testing.T) {
	type args struct {
		address string
		order   SetOrderData
	}

	tests := []struct {
		name      string
		arguments args
	}{
		{
			name: `Test set order accrual`,
			arguments: args{
				address: `http://localhost:8081/`,
				order: SetOrderData{
					OrderNum: "4561261212345467",
					Goods: []Good{
						{
							Description: `First good`,
							Price:       5000,
						},
						{
							Description: `Second good`,
							Price:       7000,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var acc Accrual
			err := acc.New(tt.arguments.address)
			require.NoError(t, err)

			err = acc.SetOrderInfo(tt.arguments.order)
			require.NoError(t, err)
		})
	}
}

func TestAccrual_GetOrderInfo(t *testing.T) {
	type args struct {
		address  string
		orderNum string
	}
	type want GetOrderData

	tests := []struct {
		name      string
		arguments args
		wanted    want
	}{
		{
			name: `Test get order accrual: no orders`,
			arguments: args{
				address:  `http://localhost:8081/`,
				orderNum: "0",
			},
			wanted: want{
				OrderNum: "0",
				Status:   ``,
			},
		},
		{
			name: `Test get order accrual`,
			arguments: args{
				address:  `http://localhost:8081/`,
				orderNum: "4561261212345467",
			},
			wanted: want{
				OrderNum: "4561261212345467",
				Status:   `PROCESSED`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var acc Accrual
			err := acc.New(tt.arguments.address)
			require.NoError(t, err)

			_, err = acc.GetOrderInfo(tt.wanted.OrderNum)
			if tt.name == `Test get order accrual: no orders` {
				assert.Error(t, err)
			}
			if tt.name == `Test get order accrual` {
				require.NoError(t, err)
			}
		})
	}
}
