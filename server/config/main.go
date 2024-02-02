package config

import (
	"errors"
	"flag"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	StartStandalone    string
	Command            string
	DatabaseConnection string `env:"DATABASE_URI"`
	MartAddress        string `env:"RUN_ADDRESS"`
	AccrualAddress     string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

const layer = `server`

// Init Заполняет данными
// структуру конфигурации
func (c *Config) Init() error {
	envErr := env.Parse(c)
	if envErr != nil {
		return envErr
	}

	flag.StringVar(&c.MartAddress, "a", "", "address and port to run server mart")
	flag.StringVar(&c.DatabaseConnection, "d", "", "db connection")
	flag.StringVar(&c.AccrualAddress, "r", "", "address and port to run accrual server")
	flag.StringVar(&c.StartStandalone, "standalone", "n", "working mode y/n, default n")
	flag.StringVar(&c.Command, "command", "start", "action command start/stop, default start")
	flag.Parse()

	if c.DatabaseConnection == `` || c.MartAddress == `` || c.AccrualAddress == `` {
		return errors.New(`empty params`)
	}

	return nil
}

func (c *Config) Get() Config {
	return *c
}
