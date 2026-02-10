/*
Package api provides the HTTP API handlers for WASAText.

What is an API Handler?
When a request comes in (like POST /session), the router calls
the appropriate handler function. That function:
1. Reads the request data
2. Validates it
3. Calls the database
4. Returns a response

This file sets up the router and CORS middleware.
*/
package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"wasatext/service/database"
)

// Handler contains all API handler methods
type Handler struct {
	db database.AppDatabase
}

// New creates a new API handler
func New(db database.AppDatabase) *Handler {
	return &Handler{db: db}
}

// NewRouter creates a new router with all routes and CORS middleware
func NewRouter(h *Handler) http.Handler {
	// Gorilla Mux is a popular Go router
	// It matches URLs to handler functions
	r := mux.NewRouter()

	// ===========================================
	// LOGIN API (from PDF - doLogin)
	// ===========================================
	r.HandleFunc("/session", h.DoLogin).Methods("POST", "OPTIONS")

	// ===========================================
	// USER APIs
	// ===========================================
	r.HandleFunc("/users", h.SearchUsers).Methods("GET", "OPTIONS")
	r.HandleFunc("/users/{userId}/username", h.SetMyUserName).Methods("PUT", "OPTIONS")
	r.HandleFunc("/users/{userId}/photo", h.SetMyPhoto).Methods("PUT", "OPTIONS")

	// ===========================================
	// CONVERSATION APIs
	// ===========================================
	r.HandleFunc("/conversations", h.GetMyConversations).Methods("GET", "OPTIONS")
	r.HandleFunc("/conversations", h.StartConversation).Methods("POST", "OPTIONS")
	r.HandleFunc("/conversations/{conversationId}", h.GetConversation).Methods("GET", "OPTIONS")

	// ===========================================
	// MESSAGE APIs
	// ===========================================
	r.HandleFunc("/conversations/{conversationId}/messages", h.SendMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/conversations/{conversationId}/messages/{messageId}", h.DeleteMessage).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/conversations/{conversationId}/messages/{messageId}/forward", h.ForwardMessage).Methods("POST", "OPTIONS")

	// ===========================================
	// COMMENT (REACTION) APIs
	// ===========================================
	r.HandleFunc("/conversations/{conversationId}/messages/{messageId}/comments", h.CommentMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/conversations/{conversationId}/messages/{messageId}/comments", h.UncommentMessage).Methods("DELETE", "OPTIONS")

	// ===========================================
	// GROUP APIs
	// ===========================================
	r.HandleFunc("/groups", h.CreateGroup).Methods("POST", "OPTIONS")
	r.HandleFunc("/groups/{groupId}/members", h.AddToGroup).Methods("POST", "OPTIONS")
	r.HandleFunc("/groups/{groupId}/members/me", h.LeaveGroup).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/groups/{groupId}/name", h.SetGroupName).Methods("PUT", "OPTIONS")
	r.HandleFunc("/groups/{groupId}/photo", h.SetGroupPhoto).Methods("PUT", "OPTIONS")

	// Serve static files (Frontend)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./webui")))

	// Wrap with CORS middleware
	return corsMiddleware(r)
}

/*
CORS Middleware

What is CORS?
CORS = Cross-Origin Resource Sharing

When your frontend (running on localhost:5000) makes a request to
your backend (running on localhost:3000), the browser blocks it
by default for security reasons. This is called "cross-origin".

CORS headers tell the browser: "It's OK to accept requests from
other websites/origins."

From the PDF:
- Allow ALL origins
- Set Max-Age to 1 second
*/
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers (as specified in PDF)
		w.Header().Set("Access-Control-Allow-Origin", "*")                                // Allow ALL origins
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS") // Allowed HTTP methods
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")     // Allowed request headers
		w.Header().Set("Access-Control-Max-Age", "1")                                     // Cache preflight for 1 second (PDF requirement)

		// Handle preflight requests
		// Preflight = browser sends OPTIONS request first to check if actual request is allowed
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Continue to actual handler
		next.ServeHTTP(w, r)
	})
}

// Helper function to get user ID from Authorization header
// Format: "Bearer <user-identifier>"
func getUserIDFromAuth(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}
