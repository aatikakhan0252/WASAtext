package main

import (
	"io/fs"
	"net/http"
	"wasatext/webui"

	"github.com/gorilla/mux"
)

// registerWebUI registers the WebUI to serve the frontend files.
// The frontend files are built by Vite into the dist/ directory
// and embedded in the Go binary via the webui package.
func registerWebUI(router *mux.Router) error {
	// Get the dist subdirectory from the embedded filesystem
	dist, err := fs.Sub(webui.Content, "dist")
	if err != nil {
		return err
	}

	// Serve static files from the dist directory
	router.PathPrefix("/").Handler(http.FileServer(http.FS(dist)))
	return nil
}
