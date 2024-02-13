package accrual

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-diploma/server/config"
	"path/filepath"
	"testing"
)

var Conf config.Config
var Acc Accrual

func TestMain(m *testing.M) {
	_ = Conf.Init()
	_ = Acc.Init(Conf.LocalConfig.Test.AccrualAddress, Conf.LocalConfig.Test.DB, filepath.Join(Conf.LocalConfig.App.RootPath+Conf.LocalConfig.App.AccrualPath))
	_ = Acc.Start()
	defer Acc.Stop()
}

func TestAccrual_SetNewAccrualType(t *testing.T) {
	_ = Conf.Init()
	Conf.Get()

	type args struct {
		accType config.NewAccrualType
	}

	tests := []struct {
		name      string
		arguments args
	}{
		{
			name: `Test set new accrual type`,
			arguments: args{
				accType: config.NewAccrualType{
					Match:      `test`,
					Reward:     5,
					RewardType: `%`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Acc.SetNewAccrualType(tt.arguments.accType)
			require.NoError(t, err)
		})
	}
}

func TestAccrual_SetOrderInfo(t *testing.T) {
	Conf.Get()

	type args struct {
		order config.SetOrderData
	}

	tests := []struct {
		name      string
		arguments args
	}{
		{
			name: `Test set order accrual`,
			arguments: args{
				order: config.SetOrderData{
					OrderNum: "4561261212345467",
					Goods: []config.Good{
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
			err := Acc.SetOrderInfo(tt.arguments.order)
			require.NoError(t, err)
		})
	}
}

func TestAccrual_GetOrderInfo(t *testing.T) {
	Conf.Get()

	type args struct {
		orderNum string
	}
	type want config.GetOrderData

	tests := []struct {
		name      string
		arguments args
		wanted    want
	}{
		{
			name: `Test get order accrual: no orders`,
			arguments: args{
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
			_, err := Acc.GetOrderInfo(tt.wanted.OrderNum)
			if tt.name == `Test get order accrual: no orders` {
				assert.Error(t, err)
			}
			if tt.name == `Test get order accrual` {
				require.NoError(t, err)
			}
		})
	}
}
