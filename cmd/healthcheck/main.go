package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: healthcheck <url>")
		os.Exit(1)
	}

	url := os.Args[1]
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Healthcheck failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Healthcheck failed: status %d\n", resp.StatusCode)
		os.Exit(1)
	}

	fmt.Println("Healthcheck passed")
}
