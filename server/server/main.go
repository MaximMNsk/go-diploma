package server

import (
	"context"
	"github.com/go-chi/chi/v5"
	"go-diploma/server/config"
	"go-diploma/server/database"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Server struct {
	Routers         chi.Router
	Config          config.Config
	Logger          *zap.Logger
	DB              database.Database
	Http            http.Server
	ShutdownProcess bool
}

func (s *Server) New(c config.Config, l *zap.Logger) error {
	var err error
	s.Config = c
	s.Routers = chi.NewRouter()
	s.Logger = l
	err = s.DB.Connect(context.Background(), c.DatabaseConnection)
	s.Http = http.Server{Addr: s.Config.MartAddress, Handler: s.Routers}
	s.ShutdownProcess = false
	return err
}

func (s *Server) Start() error {
	s.Routers.Route(`/`, func(r chi.Router) {
		s.Routers.Group(func(r chi.Router) {
			s.Routers.Post(`/app/shutdown`, s.Shutdown)
			s.Routers.Get(`/app/ok`, s.Ok)
		})
	})

	err := s.Http.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop() error {
	shutdownCtx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	err := s.Http.Shutdown(shutdownCtx)
	if err != nil {
		return err
	}
	s.DB.Close()
	//os.Exit(1)
	return nil
}

func (s *Server) Shutdown(res http.ResponseWriter, req *http.Request) {
	s.ShutdownProcess = true
	res.WriteHeader(http.StatusOK)
	s.Stop()
}
func (s *Server) Ok(res http.ResponseWriter, req *http.Request) {
	if s.ShutdownProcess {
		http.Error(res, "503 service unavailable", http.StatusServiceUnavailable)
		return
	}
	res.WriteHeader(http.StatusOK)
}
