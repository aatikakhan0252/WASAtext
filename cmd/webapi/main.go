/*
Package main is the entry point for the WASAText web API server.

What is this file?
This is the MAIN file - the starting point of our Go backend application.
When you run the program, Go looks for "func main()" and starts there.

What does it do?
1. Sets up the database (where we store users, messages, groups)
2. Creates the API router (handles incoming HTTP requests)
3. Starts the web server (listens for requests)
*/
package main

import (
	"log"
	"net/http"
	"os"

	"wasatext/service/api"
	"wasatext/service/database"
	"fmt"
)

func main() {
	fmt.Println("Starting WASAText server...")
	// Step 1: Get the port to listen on (default: 3000)
	// Environment variables are settings you can change without modifying code
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Step 2: Initialize the database
	// We use SQLite - a simple file-based database (great for learning!)
	db, err := database.New("wasatext.db")
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer db.Close() // Close the database when the program ends

	// Step 3: Create the API handler
	// This contains all our API endpoints (doLogin, sendMessage, etc.)
	apiHandler := api.New(db)

	// Step 4: Create the router with CORS middleware
	// Router = matches URLs to handler functions
	// CORS = allows requests from any website (as required by PDF)
	router := api.NewRouter(apiHandler)

	// Step 5: Start the server
	log.Printf("WASAText server starting on port %s...", port)
	log.Printf("API available at http://localhost:%s/api", port)

	// ListenAndServe starts the web server
	// It will keep running until you stop it (Ctrl+C)
	err = http.ListenAndServe(":"+port, router)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
