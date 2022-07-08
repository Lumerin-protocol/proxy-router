package api

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

type Server struct {
	address               string
	server                http.Server
	log                   zap.SugaredLogger
	connectionsController http.Handler
}

func NewServer(address string, connectionsController http.Handler) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/connections", connectionsController.ServeHTTP)

	server := http.Server{Addr: address, Handler: mux}

	return &Server{
		address:               address,
		server:                server,
		connectionsController: connectionsController,
	}
}

func (s *Server) Run(ctx context.Context) {
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			s.log.Error("Http server error", err)
		}
	}()

	<-ctx.Done()

	if err := s.server.Shutdown(context.Background()); err != nil {
		s.log.Fatal("Shutdown failed %+s", err)
	}
}
