// Package api — Login and User API handlers.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

// LoginRequest is the body for POST /session.
type LoginRequest struct {
	Name string `json:"name"`
}

// LoginResponse is the response for POST /session.
type LoginResponse struct {
	Identifier string `json:"identifier"`
}

// UsernameRequest is the body for PUT /users/:userId/username.
type UsernameRequest struct {
	Name string `json:"name"`
}

// UserResponse represents a user in API responses.
type UserResponse struct {
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	HasPhoto   bool   `json:"hasPhoto,omitempty"`
}

// ErrorResponse is used for error messages.
type ErrorResponse struct {
	Message string `json:"message"`
}

// DoLogin handles POST /session (operationId: doLogin).
func (h *Handler) DoLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Name) < 3 || len(req.Name) > 16 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Message: "Username must be between 3 and 16 characters",
		}, h.logger)
		return
	}

	userID, err := h.db.CreateUser(req.Name)
	if err != nil {
		h.logger.WithError(err).Error("failed to create user")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, LoginResponse{
		Identifier: userID,
	}, h.logger)
}

// SetMyUserName handles PUT /users/:userId/username (operationId: setMyUserName).
func (h *Handler) SetMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := ps.ByName("userId")

	if authUserID != userID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Name) < 3 || len(req.Name) > 16 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Message: "Username must be between 3 and 16 characters",
		}, h.logger)
		return
	}

	err := h.db.UpdateUserName(userID, req.Name)
	if errors.Is(err, database.ErrUsernameTaken) {
		writeJSON(w, http.StatusConflict, ErrorResponse{
			Message: "Username already taken",
		}, h.logger)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to update username")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SetMyPhoto handles PUT /users/:userId/photo (operationId: setMyPhoto).
func (h *Handler) SetMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := ps.ByName("userId")

	if authUserID != userID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	photo, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read photo", http.StatusBadRequest)
		return
	}

	if len(photo) == 0 {
		http.Error(w, "No photo provided", http.StatusBadRequest)
		return
	}

	err = h.db.UpdateUserPhoto(userID, photo)
	if errors.Is(err, database.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to update user photo")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SearchUsers handles GET /users (operationId: searchUsers).
func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("search")

	users, err := h.db.SearchUsers(query)
	if err != nil {
		h.logger.WithError(err).Error("failed to search users")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var response []UserResponse
	for _, u := range users {
		response = append(response, UserResponse{
			Identifier: u.ID,
			Name:       u.Name,
			HasPhoto:   len(u.Photo) > 0,
		})
	}

	writeJSON(w, http.StatusOK, response, h.logger)
}

// GetUserPhoto handles GET /users/:userId/photo — serves photo as binary.
func (h *Handler) GetUserPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userID := ps.ByName("userId")

	photo, err := h.db.GetUserPhoto(userID)
	if err != nil {
		http.Error(w, "No photo", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	if _, writeErr := w.Write(photo); writeErr != nil {
		h.logger.WithError(writeErr).Error("failed to write photo response")
	}
}
