package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/pkg/config"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(port int, handler http.Handler) *Server {
	envCfg := config.Instance()
	server := &http.Server{
		Addr:              ":" + intToString(port),
		Handler:           metricsMiddleware(requestIDMiddleware(authMiddleware(envCfg.APIAuthToken, handler))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return &Server{httpServer: server}
}

func (s *Server) Run() error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = sig
		return s.httpServer.Shutdown(ctx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func intToString(i int) string {
	return fmt.Sprintf("%d", i)
}
