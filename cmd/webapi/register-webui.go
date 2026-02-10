package main

import (
	"io/fs"
	"net/http"
	"wasatext/webui"

	"github.com/gorilla/mux"
)

// webuiContent is the embedded frontend content

// registerWebUI registers the WebUI path to the router
// This function allows serving the frontend files
func registerWebUI(router *mux.Router) error {
	dist, err := fs.Sub(webui.Content, "dist")
	if err != nil {
		return err
	}

	// Serve static files
	router.Handle("/", http.FileServer(http.FS(dist)))
	return nil
}
