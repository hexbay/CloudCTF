package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloudctf/backend/internal/background"
	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/config"
	"cloudctf/backend/internal/httpserver"
	"cloudctf/backend/internal/logging"
)

var version = "dev"

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.EnsureRuntimePaths(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare runtime paths: %v\n", err)
		os.Exit(1)
	}
	logger := logging.NewLogger(cfg)
	db, err := bootstrap.OpenDB(cfg.DatabaseURI)
	if err != nil {
		logger.Error("database initialization failed", "uri", cfg.DatabaseURI, "error", err)
		os.Exit(1)
	}
	if err := bootstrap.PrepareSchema(db); err != nil {
		logger.Error("database schema preparation failed", "uri", cfg.DatabaseURI, "error", err)
		os.Exit(1)
	}
	srv := httpserver.NewServer(httpserver.Dependencies{DB: db, Config: cfg, Logger: logger, Version: version})

	processCtx, processCancel := context.WithCancel(context.Background())
	defer processCancel()
	background.Start(processCtx, logger, srv.GetContainerGroupService())

	server := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("api server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	processCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("api server stopped")
}
