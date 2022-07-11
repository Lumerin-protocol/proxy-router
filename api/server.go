package api

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Server struct {
	address               string
	server                http.Server
	log                   zap.SugaredLogger
	connectionsController http.Handler
	shutdownTimeout       time.Duration
}

func NewServer(address string, connectionsController http.Handler, logger *zap.SugaredLogger) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/connections", connectionsController.ServeHTTP)

	server := http.Server{Addr: address, Handler: mux}

	return &Server{
		address:               address,
		server:                server,
		connectionsController: connectionsController,
		shutdownTimeout:       5 * time.Second,
		log:                   *logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	return s.listenAndServe(ctx)
}

func (s *Server) listenAndServe(ctx context.Context) error {
	serverErr := make(chan error, 1)
	go func() {
		// Capture ListenAndServe errors such as "port already in use".
		// However, when a server is gracefully shutdown, it is safe to ignore errors
		// returned from this method (given the select logic below), because
		// Shutdown causes ListenAndServe to always return http.ErrServerClosed.
		if err := s.server.ListenAndServe(); err != nil {
			s.log.Error("Http server error", err)
		}
	}()
	var err error
	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
		defer cancel()
		err = s.server.Shutdown(ctx)
		s.log.Info("Server closed")
	case err = <-serverErr:
	}

	return err
}
