// Package main is a health check utility for the WASAText server.
package main

import (
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()

	if len(os.Args) < 2 {
		logger.Info("Usage: healthcheck <url>")
		return
	}

	url := os.Args[1]

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		logger.WithError(err).Error("failed to create request")
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.WithError(err).Error("healthcheck failed")
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.WithError(closeErr).Error("error closing response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		logger.WithField("status", resp.StatusCode).Error("healthcheck failed")
		return
	}

	logger.Info("healthcheck passed")
}
