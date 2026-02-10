/*
Package main is the entry point for the WASAText web API server.

It sets up the database, API router, and starts the HTTP server.
This package follows the project structure guidelines for the WASA course.
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"wasatext/service/api"
	"wasatext/service/database"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

// Main entry point
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run performs the server setup and execution
func run() error {
	fmt.Println("Starting WASAText server...")

	// Just a dummy usage of rate limiter dependency to force vendoring
	_ = rate.NewLimiter(1, 5)

	// Step 1: Get the port to listen on (default: 3000)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Step 2: Initialize the database
	db, err := database.New("wasatext.db")
	if err != nil {
		return fmt.Errorf("error initializing database: %w", err)
	}
	defer db.Close()

	// Step 3: Create the API handler
	apiHandler := api.New(db)

	// Step 4: Create the router
	router := api.NewRouter(apiHandler)

	// Register WebUI
	// This serves the frontend files (if embedded)
	if err := registerWebUI(router.(*mux.Router)); err != nil {
		// Log warning but don't fail, as webui might be optional during dev
		log.Printf("Warning: failed to register WebUI: %v", err)
	}

	// Step 5: Start the server
	log.Printf("WASAText server starting on port %s...", port)
	log.Printf("API available at http://localhost:%s/api", port)

	err = http.ListenAndServe(":"+port, router)
	if err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}
