package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"safety-recorder/server/internal/config"
)

type namedServer struct {
	name   string
	server *http.Server
}

func newHTTPServers(cfg config.Config, privateHandler, publicHandler http.Handler) []namedServer {
	servers := make([]namedServer, 0, len(cfg.PrivateBindAddrs)+len(cfg.PublicBindAddrs))
	for _, addr := range cfg.PrivateBindAddrs {
		servers = append(servers, namedServer{
			name: "private api",
			server: &http.Server{
				Addr:              addr,
				Handler:           privateHandler,
				ReadHeaderTimeout: cfg.PrivateTimeouts.ReadHeaderTimeout,
				ReadTimeout:       cfg.PrivateTimeouts.ReadTimeout,
				WriteTimeout:      cfg.PrivateTimeouts.WriteTimeout,
				IdleTimeout:       cfg.PrivateTimeouts.IdleTimeout,
			},
		})
	}
	for _, addr := range cfg.PublicBindAddrs {
		servers = append(servers, namedServer{
			name: "public emergency viewer",
			server: &http.Server{
				Addr:              addr,
				Handler:           publicHandler,
				ReadHeaderTimeout: cfg.PublicTimeouts.ReadHeaderTimeout,
				ReadTimeout:       cfg.PublicTimeouts.ReadTimeout,
				WriteTimeout:      cfg.PublicTimeouts.WriteTimeout,
				IdleTimeout:       cfg.PublicTimeouts.IdleTimeout,
			},
		})
	}
	return servers
}

func startServer(errCh chan<- error, logger *slog.Logger, named namedServer) {
	go func() {
		logger.Info("starting "+named.name+" server", "addr", named.server.Addr)
		if err := named.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("%s server %s: %w", named.name, named.server.Addr, err)
		}
	}()
}

func shutdownServers(servers []namedServer) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var shutdownErr error
	for _, named := range servers {
		if err := named.server.Shutdown(shutdownCtx); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
	}
	return shutdownErr
}
