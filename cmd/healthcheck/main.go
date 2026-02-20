/*
Package main is a health check utility for the WASAText server.

It checks if the server is responding to HTTP requests.
*/
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("Usage: healthcheck <url>")
		return
	}

	url := os.Args[1]
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		log.Printf("Healthcheck failed: %v", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Healthcheck failed: status %d", resp.StatusCode)
		return
	}

	log.Println("Healthcheck passed")
}
