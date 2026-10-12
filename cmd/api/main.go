package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"meteor/internal/config"
	"meteor/internal/database"
	"meteor/internal/health"
	"meteor/internal/httpapi"
	"meteor/internal/logging"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (runErr error) {
	cfg, configErr := config.Load()
	if cfg.ErrorLogPath == "" {
		return errors.New("ERROR_LOG_PATH is required to initialize logging")
	}

	logs, err := logging.New(cfg.ErrorLogPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not initialize the error log file")
		return fmt.Errorf("initialize logging: %w", err)
	}
	defer func() {
		if runErr != nil {
			logs.Errors.Error().Err(runErr).Msg("API stopped")
		}
		if err := logs.Close(); err != nil && runErr == nil {
			runErr = fmt.Errorf("close error log: %w", err)
		}
	}()

	if configErr != nil {
		return fmt.Errorf("load configuration: %w", configErr)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	service := health.NewService(database.NewHealthRepository(pool))
	handler := httpapi.NewServer(service, logs)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logs.Requests.Info().Str("address", server.Addr).Msg("Meteor API listening")
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("start HTTP server: %w", err)
	}
	return nil
}
