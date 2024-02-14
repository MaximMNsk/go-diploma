package main

import (
	_ "context"
	"errors"
	conf "go-diploma/server/config"
	"go-diploma/server/logger"
	serv "go-diploma/server/server"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	var config conf.Config
	args := os.Args
	log := logger.CreateLogger()

	log.Info(`Parse config`)
	err := config.Init()
	if err != nil {
		log.Error(err.Error())
	}

	if config.Command == `stop` {
		log.Info(`Command stop received. Shutting down server`)
		stopped := gracefulShutdown(config, log)
		if !stopped {
			log.Error(`do not stop`)
		}
		return
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGABRT, syscall.SIGINT)
	go func() {
		for {
			select {
			case <-exit:
				log.Info(`Signal received. Shutting down server`)
				stopped := gracefulShutdown(config, log)
				if !stopped {
					log.Error(`do not stop`)
				}
				time.Sleep(2 * time.Second)
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}
	}()

	if config.StartStandalone == `y` {
		log.Info(`Standalone mode`)
		cmd := exec.Command(
			args[0],
			`-a`, config.MartAddress,
			`-d`, config.DatabaseConnection,
			`-r`, config.AccrualAddress,
		)
		err := cmd.Start()
		if err != nil {
			log.Info(err.Error())
		}
		return
	}

	log.Info(`Starting app`)

	var server serv.Server
	log.Info(`Init server`)
	err = server.New(config, log)
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Info(`Start server`)
	err = server.Start()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error(err.Error())
		return
	} else {
		log.Info(`Server stopped`)
		return
	}
}

func gracefulShutdown(c conf.Config, l *zap.Logger) bool {
	shutdownURL := `http://` + c.MartAddress + `/app/shutdown`
	l.Info(`Shutdown by: ` + shutdownURL)
	request, err := http.NewRequest(http.MethodPost, shutdownURL, nil)
	if err != nil {
		l.Error(err.Error())
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		l.Error(err.Error())
	}
	l.Info(`Status code: ` + strconv.Itoa(response.StatusCode))
	err = response.Body.Close()

	return err == nil && response.StatusCode == http.StatusOK
}
