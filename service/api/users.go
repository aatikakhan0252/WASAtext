/*
Login and User API handlers.

This file contains:
- doLogin: Login/register a user
- setMyUserName: Change username
- setMyPhoto: Set profile photo
- searchUsers: Search for users
*/
package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"wasatext/service/database"

	"github.com/gorilla/mux"
)

// LoginRequest is the body for POST /session
type LoginRequest struct {
	Name string `json:"name"`
}

// LoginResponse is the response for POST /session
type LoginResponse struct {
	Identifier string `json:"identifier"`
}

// UsernameRequest is the body for PUT /users/{userId}/username
type UsernameRequest struct {
	Name string `json:"name"`
}

// UserResponse represents a user in API responses
type UserResponse struct {
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	HasPhoto   bool   `json:"hasPhoto,omitempty"`
}

// ErrorResponse is used for error messages
type ErrorResponse struct {
	Message string `json:"message"`
}

/*
DoLogin handles POST /session
operationId: doLogin

From PDF:
"If the user does not exist, it will be created, and an identifier is returned.
If the user exists, the user identifier is returned."

This is the SIMPLIFIED login - no password required!
*/
func (h *Handler) DoLogin(w http.ResponseWriter, r *http.Request) {
	// Step 1: Parse the request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Step 2: Validate the username (3-16 characters as per PDF)
	if len(req.Name) < 3 || len(req.Name) > 16 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Message: "Username must be between 3 and 16 characters",
		})
		return
	}

	// Step 3: Create or get the user
	userID, err := h.db.CreateUser(req.Name)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 4: Return the user identifier
	// Status 201 as specified in PDF
	writeJSON(w, http.StatusCreated, LoginResponse{
		Identifier: userID,
	})
}

/*
SetMyUserName handles PUT /users/{userId}/username
operationId: setMyUserName

From PDF:
"Users also have the ability to update their name, provided the new name
is not already in use by someone else."
*/
func (h *Handler) SetMyUserName(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get the user ID from the URL
	vars := mux.Vars(r)
	userID := vars["userId"]

	// Step 3: Make sure user is updating their own name
	if authUserID != userID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 4: Parse the request body
	var req UsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Step 5: Validate the new username
	if len(req.Name) < 3 || len(req.Name) > 16 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Message: "Username must be between 3 and 16 characters",
		})
		return
	}

	// Step 6: Update the username
	err := h.db.UpdateUserName(userID, req.Name)
	if errors.Is(err, database.ErrUsernameTaken) {
		writeJSON(w, http.StatusConflict, ErrorResponse{
			Message: "Username already taken",
		})
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 7: Return success
	w.WriteHeader(http.StatusOK)
}

/*
SetMyPhoto handles PUT /users/{userId}/photo
operationId: setMyPhoto

Allows user to upload/update their profile photo.
*/
func (h *Handler) SetMyPhoto(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get the user ID from URL
	vars := mux.Vars(r)
	userID := vars["userId"]

	// Step 3: Make sure user is updating their own photo
	if authUserID != userID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 4: Read the photo from request body
	photo, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read photo", http.StatusBadRequest)
		return
	}

	if len(photo) == 0 {
		http.Error(w, "No photo provided", http.StatusBadRequest)
		return
	}

	// Step 5: Update the photo in database
	err = h.db.UpdateUserPhoto(userID, photo)
	if errors.Is(err, database.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 6: Return success
	w.WriteHeader(http.StatusOK)
}

/*
SearchUsers handles GET /users
operationId: searchUsers (not in required list but needed for PDF requirement)

From PDF:
"The user can search for other users via the username and see all
the existing WASAText usernames."
*/
func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get the search query
	query := r.URL.Query().Get("search")

	// Step 3: Search for users
	users, err := h.db.SearchUsers(query)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 4: Convert to response format
	var response []UserResponse
	for _, u := range users {
		response = append(response, UserResponse{
			Identifier: u.ID,
			Name:       u.Name,
			HasPhoto:   len(u.Photo) > 0,
		})
	}

	// Step 5: Return the users
	writeJSON(w, http.StatusOK, response)
}

// writeJSON is a helper to write JSON responses
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
