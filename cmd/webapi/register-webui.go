package main

import (
	"net/http"
	"wasatext/webui"

	"github.com/gorilla/mux"
)

// registerWebUI registers the WebUI to serve the frontend files.
// The frontend files are embedded in the Go binary via the webui package.
func registerWebUI(router *mux.Router) error {
	// Serve static files directly from the embedded webui filesystem.
	// No "dist" subdirectory needed since this is a plain HTML/JS/CSS app.
	router.PathPrefix("/").Handler(http.FileServer(http.FS(webui.Content)))
	return nil
}
