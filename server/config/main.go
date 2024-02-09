package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
	"os"
	"path/filepath"
)

type Config struct {
	StartStandalone    string
	Command            string
	Mode               string
	DatabaseConnection string `env:"DATABASE_URI"`
	MartAddress        string `env:"RUN_ADDRESS"`
	AccrualAddress     string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	LocalConfig        LocalCfg
}

type LocalCfg struct {
	App struct {
		RootPath       string `json:"rootPath"`
		MartPath       string `json:"martPath"`
		AccrualPath    string `json:"accrualPath"`
		MigrationsPath string `json:"migrationsPath"`
	} `json:"app"`
	Logger struct {
	} `json:"logger"`
	Test struct {
		DB             string `json:"db"`
		MartAddress    string `json:"martAddress"`
		AccrualAddress string `json:"accrualAddress"`
	} `json:"test"`
	Accrual struct {
		Orders []SetOrderData   `json:"orders"`
		Goods  []NewAccrualType `json:"goods"`
	} `json:"accrual"`
}

type Good struct {
	Description string  `json:"description"`
	Price       float32 `json:"price"`
}
type SetOrderData struct {
	OrderNum string `json:"order"`
	Goods    []Good `json:"goods"`
}

type NewAccrualType struct {
	Match      string  `json:"match"`
	Reward     float32 `json:"reward"`
	RewardType string  `json:"reward_type"`
}

type GetOrderData struct {
	OrderNum   string      `json:"number"`
	Status     string      `json:"status"`
	Accrual    float32     `json:"accrual,omitempty"`
	UploadedAt interface{} `json:"uploaded_at"`
}

var (
	ErrEnv           = errors.New(`config env error`)
	ErrFile          = errors.New(`config file error`)
	ErrConfigConsist = errors.New(`not all params filled for start app`)
)

// Init Заполняет данными
// структуру конфигурации
func (c *Config) Init() error {
	envErr := env.Parse(c)
	if envErr != nil {
		return fmt.Errorf(envErr.Error()+` : %w`, ErrEnv)
	}

	if c.DatabaseConnection == `` || c.MartAddress == `` || c.AccrualAddress == `` {
		flag.StringVar(&c.MartAddress, "a", "", "address and port to run server mart")
		flag.StringVar(&c.DatabaseConnection, "d", "", "db connection")
		flag.StringVar(&c.AccrualAddress, "r", "", "address and port to run accrual server")
	}
	flag.StringVar(&c.StartStandalone, "standalone", "n", "working mode y/n, default n")
	flag.StringVar(&c.Mode, "mode", "easy", "running mode easy/full, default easy")
	flag.StringVar(&c.Command, "command", "start", "action command start/stop, default start")
	flag.Parse()

	if c.Mode == `full` {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}
		fileConfig, errFile := os.Open(filepath.Join(pwd, `./config.json`))
		if errFile != nil {
			return fmt.Errorf(errFile.Error()+` : %w`, ErrFile)
		}
		decoder := json.NewDecoder(fileConfig)
		errDecode := decoder.Decode(&c.LocalConfig)
		if errDecode != nil {
			return fmt.Errorf(errDecode.Error()+` : %w`, ErrFile)
		}
		err = fileConfig.Close()
		if err != nil {
			return err
		}
	}

	if c.DatabaseConnection == `` || c.MartAddress == `` || c.AccrualAddress == `` {
		return ErrConfigConsist
	}

	return nil
}

func (c *Config) Get() Config {
	return *c
}
