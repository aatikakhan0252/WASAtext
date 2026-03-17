// Package main is the entry point for the WASAText web API server.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"wasatext/service/api"
	"wasatext/service/database"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := run(); err != nil {
		logger := logrus.New()
		logger.WithError(err).Error("application exited with error")
	}
}

func run() error {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	logger.Info("Starting WASAText server...")

	// Get the port to listen on (default: 3000).
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Initialize the database.
	dbPath := os.Getenv("WASATEXT_DB_FILENAME")
	if dbPath == "" {
		dbPath = "wasatext.db"
	}

	// Ensure the parent directory for the database file exists.
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
			return errors.New("error creating database directory: " + mkErr.Error())
		}
	}

	db, err := database.New(dbPath)
	if err != nil {
		return errors.New("error initializing database: " + err.Error())
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.WithError(closeErr).Error("error closing database")
		}
	}()

	// Create the API handler and router.
	apiHandler := api.New(db, logger)
	router := api.NewRouter(apiHandler)

	// Register WebUI (serves the embedded frontend files).
	if regErr := registerWebUI(router); regErr != nil {
		logger.WithError(regErr).Warn("failed to register WebUI")
	}

	// Wrap the router with CORS middleware.
	handler := api.CorsMiddleware(router)

	// Create the HTTP server.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown: listen for interrupt signals.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		logger.WithField("port", port).Info("WASAText server listening")
		if listenErr := srv.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			errCh <- listenErr
		}
		close(errCh)
	}()

	// Wait for shutdown signal or server error.
	select {
	case sig := <-quit:
		logger.WithField("signal", sig.String()).Info("shutting down server")
	case listenErr := <-errCh:
		if listenErr != nil {
			return errors.New("server failed: " + listenErr.Error())
		}
	}

	// Shutdown with a timeout.
	const shutdownTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
		return errors.New("server shutdown failed: " + shutdownErr.Error())
	}

	logger.Info("server stopped gracefully")
	return nil
}
