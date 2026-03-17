// Package api — Group API handlers.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

// CreateGroupRequest is the body for POST /groups.
type CreateGroupRequest struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"memberIds"`
}

// AddToGroupRequest is the body for POST /groups/:groupId/members.
type AddToGroupRequest struct {
	UserID string `json:"userId"`
}

// SetGroupNameRequest is the body for PUT /groups/:groupId/name.
type SetGroupNameRequest struct {
	Name string `json:"name"`
}

// GroupResponse represents a group in API responses.
type GroupResponse struct {
	GroupID  string         `json:"groupId"`
	Name     string         `json:"name"`
	HasPhoto bool           `json:"hasPhoto"`
	Members  []UserResponse `json:"members"`
}

// CreateGroup handles POST /groups (operationId: createGroup).
func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	group, err := h.db.CreateGroup(req.Name, authUserID, req.MemberIDs)
	if err != nil {
		h.logger.WithError(err).Error("failed to create group")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

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

	writeJSON(w, http.StatusCreated, response, h.logger)
}

// AddToGroup handles POST /groups/:groupId/members (operationId: addToGroup).
func (h *Handler) AddToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupID := ps.ByName("groupId")

	var req AddToGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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
		h.logger.WithError(err).Error("failed to add user to group")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// LeaveGroup handles DELETE /groups/:groupId/members/me (operationId: leaveGroup).
func (h *Handler) LeaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupID := ps.ByName("groupId")

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
		h.logger.WithError(err).Error("failed to leave group")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetGroupName handles PUT /groups/:groupId/name (operationId: setGroupName).
func (h *Handler) SetGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupID := ps.ByName("groupId")

	isMember, err := h.db.IsGroupMember(groupID, authUserID)
	if err != nil {
		h.logger.WithError(err).Error("failed to check group membership")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !isMember {
		http.Error(w, "Not a member of this group", http.StatusForbidden)
		return
	}

	var req SetGroupNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	err = h.db.UpdateGroupName(groupID, req.Name)
	if errors.Is(err, database.ErrGroupNotFound) {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to update group name")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SetGroupPhoto handles PUT /groups/:groupId/photo (operationId: setGroupPhoto).
func (h *Handler) SetGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupID := ps.ByName("groupId")

	isMember, err := h.db.IsGroupMember(groupID, authUserID)
	if err != nil {
		h.logger.WithError(err).Error("failed to check group membership")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !isMember {
		http.Error(w, "Not a member of this group", http.StatusForbidden)
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

	err = h.db.UpdateGroupPhoto(groupID, photo)
	if errors.Is(err, database.ErrGroupNotFound) {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to update group photo")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
