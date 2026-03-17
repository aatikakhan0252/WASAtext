package main

import (
	"io/fs"
	"net/http"
	"wasatext/webui"

	"github.com/julienschmidt/httprouter"
)

// registerWebUI registers the WebUI to serve the frontend files.
func registerWebUI(router *httprouter.Router) error {
	dist, err := fs.Sub(webui.Content, "dist")
	if err != nil {
		return err
	}

	// Serve static files. httprouter requires a named parameter for
	// catch-all routes; the pattern "/*filepath" matches everything.
	router.NotFound = http.FileServer(http.FS(dist))
	return nil
}
