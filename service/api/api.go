// Package api provides the HTTP API handlers for WASAText.
package api

import (
	"encoding/json"
	"net/http"

	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// Message status constants — no magic strings.
const (
	StatusSent     = "sent"
	StatusReceived = "received"
	StatusRead     = "read"
)

// Handler contains all API handler methods.
type Handler struct {
	db     database.AppDatabase
	logger *logrus.Logger
}

// New creates a new API handler.
func New(db database.AppDatabase, logger *logrus.Logger) *Handler {
	return &Handler{db: db, logger: logger}
}

// NewRouter creates a new httprouter.Router with all routes registered.
func NewRouter(h *Handler) *httprouter.Router {
	r := httprouter.New()

	// LOGIN
	r.POST("/session", h.DoLogin)

	// USERS
	r.GET("/users", h.SearchUsers)
	r.PUT("/users/:userId/username", h.SetMyUserName)
	r.PUT("/users/:userId/photo", h.SetMyPhoto)
	r.GET("/users/:userId/photo", h.GetUserPhoto)

	// CONVERSATIONS
	r.GET("/conversations", h.GetMyConversations)
	r.POST("/conversations", h.StartConversation)
	r.GET("/conversations/:conversationId", h.GetConversation)

	// MESSAGES
	r.POST("/conversations/:conversationId/messages", h.SendMessage)
	r.DELETE("/conversations/:conversationId/messages/:messageId", h.DeleteMessage)
	r.POST("/conversations/:conversationId/messages/:messageId/forward", h.ForwardMessage)
	r.GET("/conversations/:conversationId/messages/:messageId/photo", h.GetMessagePhoto)

	// COMMENTS (REACTIONS)
	r.POST("/conversations/:conversationId/messages/:messageId/comments", h.CommentMessage)
	r.DELETE("/conversations/:conversationId/messages/:messageId/comments", h.UncommentMessage)

	// GROUPS
	r.POST("/groups", h.CreateGroup)
	r.POST("/groups/:groupId/members", h.AddToGroup)
	r.DELETE("/groups/:groupId/members/me", h.LeaveGroup)
	r.PUT("/groups/:groupId/name", h.SetGroupName)
	r.PUT("/groups/:groupId/photo", h.SetGroupPhoto)
	r.GET("/groups/:groupId/photo", h.GetGroupPhotoHandler)

	return r
}

// CorsMiddleware handles CORS headers.
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "1")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getUserIDFromAuth extracts the user ID from the Authorization header.
// Format: "Bearer <user-identifier>"
func getUserIDFromAuth(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	const bearerPrefix = "Bearer "
	if len(auth) > len(bearerPrefix) && auth[:len(bearerPrefix)] == bearerPrefix {
		return auth[len(bearerPrefix):]
	}
	return ""
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}, logger *logrus.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.WithError(err).Error("failed to encode JSON response")
	}
}
