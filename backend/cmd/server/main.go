package main

import (
	"context"
	"errors"
	"fmt"
	"net"
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
	"cloudctf/backend/internal/updater"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "reset-admin-password" {
		if err := runResetAdminPassword(os.Args[2:], os.Stdin, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "failed to reset administrator password: %v\n", err)
			os.Exit(1)
		}
		return
	}

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.EnsureRuntimePaths(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare runtime paths: %v\n", err)
		os.Exit(1)
	}
	if updater.RunningInContainer() {
		if err := activatePersistentUpdate(version, cfg.UpdateDir); err != nil {
			fmt.Fprintf(os.Stderr, "failed to activate persistent update; using image version: %v\n", err)
		}
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
	restart := make(chan struct{}, 1)
	srv := httpserver.NewServer(httpserver.Dependencies{
		DB:      db,
		Config:  cfg,
		Logger:  logger,
		Version: version,
		RequestRestart: func() {
			select {
			case restart <- struct{}{}:
			default:
			}
		},
	})

	processCtx, processCancel := context.WithCancel(context.Background())
	defer processCancel()
	background.Start(processCtx, logger, srv.GetContainerGroupService())

	server := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		logger.Error("api server failed to listen", "addr", server.Addr, "error", err)
		os.Exit(1)
	}
	if updater.RunningInContainer() {
		if executable, executableErr := os.Executable(); executableErr == nil {
			if healthyErr := updater.MarkPersistentHealthy(version, cfg.UpdateDir, executable); healthyErr != nil {
				logger.Warn("failed to mark persistent update healthy", "error", healthyErr)
			}
		}
	}

	go func() {
		logger.Info("api server started", "addr", server.Addr)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	restartRequested := false
	select {
	case <-stop:
	case <-restart:
		restartRequested = true
		logger.Info("restart requested after online update")
	}

	processCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("api server stopped")
	if restartRequested {
		if err := restartCurrentProcess(version, cfg.UpdateDir); err != nil {
			logger.Error("failed to restart updated service", "error", err)
			os.Exit(1)
		}
	}
}
