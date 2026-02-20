/*
Group API handlers.

This file contains:
- createGroup: Create a new group
- addToGroup: Add a user to a group
- leaveGroup: Leave a group
- setGroupName: Change group name
- setGroupPhoto: Set group photo
*/
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"wasatext/service/database"

	"github.com/gorilla/mux"
)

// CreateGroupRequest is the body for POST /groups
type CreateGroupRequest struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"memberIds"`
}

// AddToGroupRequest is the body for POST /groups/{groupId}/members
type AddToGroupRequest struct {
	UserID string `json:"userId"`
}

// SetGroupNameRequest is the body for PUT /groups/{groupId}/name
type SetGroupNameRequest struct {
	Name string `json:"name"`
}

// GroupResponse represents a group in API responses
type GroupResponse struct {
	GroupID  string         `json:"groupId"`
	Name     string         `json:"name"`
	HasPhoto bool           `json:"hasPhoto"`
	Members  []UserResponse `json:"members"`
}

/*
CreateGroup handles POST /groups
operationId: createGroup (needed to support PDF requirement)

From PDF:
"The user can create a new group with any number of other WASAText users
to start a conversation."
*/
func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Parse request body
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Step 3: Validate
	if req.Name == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Step 4: Create the group
	group, err := h.db.CreateGroup(req.Name, authUserID, req.MemberIDs)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 5: Convert to response format
	response := GroupResponse{
		GroupID:  group.ID,
		Name:     group.Name,
		HasPhoto: len(group.Photo) > 0,
	}

	for _, m := range group.Members {
		response.Members = append(response.Members, UserResponse{
			Identifier: m.ID,
			Name:       m.Name,
			HasPhoto:   len(m.Photo) > 0,
		})
	}

	// Step 6: Return the group
	writeJSON(w, http.StatusCreated, response)
}

/*
AddToGroup handles POST /groups/{groupId}/members
operationId: addToGroup

From PDF:
"Group members can add other users to the group, but users cannot
join groups on their own or even see groups they aren't a part of."
*/
func (h *Handler) AddToGroup(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]

	// Step 3: Parse request body
	var req AddToGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Step 4: Add the user to the group
	// The database function checks if the adder is a member
	err := h.db.AddUserToGroup(groupID, req.UserID, authUserID)
	if errors.Is(err, database.ErrGroupNotFound) {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, database.ErrNotGroupMember) {
		http.Error(w, "Not a member of this group", http.StatusForbidden)
		return
	}
	if errors.Is(err, database.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 5: Return success (201 Created)
	w.WriteHeader(http.StatusCreated)
}

/*
LeaveGroup handles DELETE /groups/{groupId}/members/me
operationId: leaveGroup

From PDF:
"Additionally, users have the option to leave a group at any time."
*/
func (h *Handler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]

	// Step 3: Remove the user from the group
	err := h.db.RemoveUserFromGroup(groupID, authUserID)
	if errors.Is(err, database.ErrGroupNotFound) {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, database.ErrNotGroupMember) {
		http.Error(w, "Not a member of this group", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 4: Return success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}

/*
SetGroupName handles PUT /groups/{groupId}/name
operationId: setGroupName

Allows group members to change the group name.
*/
func (h *Handler) SetGroupName(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]

	// Step 3: Check if user is a member
	isMember, err := h.db.IsGroupMember(groupID, authUserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !isMember {
		http.Error(w, "Not a member of this group", http.StatusForbidden)
		return
	}

	// Step 4: Parse request body
	var req SetGroupNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Step 5: Update the group name
	err = h.db.UpdateGroupName(groupID, req.Name)
	if errors.Is(err, database.ErrGroupNotFound) {
		http.Error(w, "Group not found", http.StatusNotFound)
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
SetGroupPhoto handles PUT /groups/{groupId}/photo
operationId: setGroupPhoto

Allows group members to set the group photo.
*/
func (h *Handler) SetGroupPhoto(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get group ID from URL
	vars := mux.Vars(r)
	groupID := vars["groupId"]

	// Step 3: Check if user is a member
	isMember, err := h.db.IsGroupMember(groupID, authUserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !isMember {
		http.Error(w, "Not a member of this group", http.StatusForbidden)
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

	// Step 5: Update the group photo
	err = h.db.UpdateGroupPhoto(groupID, photo)
	if errors.Is(err, database.ErrGroupNotFound) {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 6: Return success
	w.WriteHeader(http.StatusOK)
}
