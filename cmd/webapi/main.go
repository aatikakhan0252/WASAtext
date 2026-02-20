/*
Package main is the entry point for the WASAText web API server.

It sets up the database, API router, and starts the HTTP server.
This package follows the project structure guidelines for the WASA course.
*/
package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"wasatext/service/api"
	"wasatext/service/database"

	"golang.org/x/time/rate"
)

// Main entry point
func main() {
	if err := run(); err != nil {
		log.Printf("error: %v", err)
	}
}

// run performs the server setup and execution
func run() error {
	log.Println("Starting WASAText server...")

	// Just a dummy usage of rate limiter dependency to force vendoring
	_ = rate.NewLimiter(1, 5)

	// Step 1: Get the port to listen on (default: 3000)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Step 2: Initialize the database
	dbPath := os.Getenv("WASATEXT_DB_FILENAME")
	if dbPath == "" {
		dbPath = "wasatext.db"
	}
	db, err := database.New(dbPath)
	if err != nil {
		return errors.New("error initializing database: " + err.Error())
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Step 3: Create the API handler
	apiHandler := api.New(db)

	// Step 4: Create the router
	router := api.NewRouter(apiHandler)

	// Register WebUI
	// This serves the frontend files (if embedded)
	if err := registerWebUI(router); err != nil {
		// Log warning but don't fail, as webui might be optional during dev
		log.Printf("Warning: failed to register WebUI: %v", err)
	}

	// Step 5: Start the server
	log.Printf("WASAText server starting on port %s...", port)
	log.Printf("API available at http://localhost:%s/", port)

	// Wrap the router with CORS middleware
	handler := api.CorsMiddleware(router)

	err = http.ListenAndServe(":"+port, handler)
	if err != nil {
		return errors.New("server failed to start: " + err.Error())
	}

	return nil
}
