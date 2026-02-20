/*
Message API handlers.

This file contains:
- sendMessage: Send a new message
- forwardMessage: Forward a message to another conversation
- deleteMessage: Delete a sent message
- commentMessage: Add a reaction to a message
- uncommentMessage: Remove a reaction from a message
*/
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"wasatext/service/database"

	"github.com/gorilla/mux"
)

// SendMessageRequest is the body for POST /conversations/{id}/messages
type SendMessageRequest struct {
	Content string `json:"content,omitempty"`
	ReplyTo string `json:"replyTo,omitempty"`
}

// ForwardMessageRequest is the body for POST /conversations/{id}/messages/{msgId}/forward
type ForwardMessageRequest struct {
	TargetConversationID string `json:"targetConversationId"`
}

// CommentRequest is the body for POST /conversations/{id}/messages/{msgId}/comments
type CommentRequest struct {
	Emoticon string `json:"emoticon"`
}

/*
SendMessage handles POST /conversations/{conversationId}/messages
operationId: sendMessage

From PDF:
"The user can send a new message, reply to an existing one..."
Messages can be text or photo/GIF.
*/
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get conversation ID from URL
	vars := mux.Vars(r)
	conversationID := vars["conversationId"]

	// Step 3: Check if user is part of this conversation
	_, err := h.db.GetConversation(authUserID, conversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 4: Parse the request based on content type
	contentType := r.Header.Get("Content-Type")

	var content string
	var photo []byte
	var replyTo *string

	if strings.Contains(contentType, "multipart/form-data") {
		// Photo/GIF upload
		err := r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("photo")
		if err == nil { //nolint:goerr113
			defer file.Close()
			photo, _ = io.ReadAll(file)
		}

		if replyToVal := r.FormValue("replyTo"); replyToVal != "" {
			replyTo = &replyToVal
		}
	} else {
		// JSON text message
		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		content = req.Content
		if req.ReplyTo != "" {
			replyTo = &req.ReplyTo
		}
	}

	// Step 5: Validate - must have content or photo
	if content == "" && len(photo) == 0 {
		http.Error(w, "Message must have content or photo", http.StatusBadRequest)
		return
	}

	// Step 6: Create the message
	msg, err := h.db.CreateMessage(conversationID, authUserID, content, photo, replyTo)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 7: Return the created message
	response := MessageResponse{
		MessageID:  msg.ID,
		SenderID:   msg.SenderID,
		SenderName: msg.SenderName,
		Content:    msg.Content,
		HasPhoto:   len(msg.Photo) > 0,
		Timestamp:  msg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		Status:     msg.Status,
	}

	if msg.ReplyTo != nil {
		response.ReplyTo = *msg.ReplyTo
	}

	writeJSON(w, http.StatusCreated, response)
}

/*
ForwardMessage handles POST /conversations/{conversationId}/messages/{messageId}/forward
operationId: forwardMessage

From PDF:
"The user can... forward a message..."
*/
func (h *Handler) ForwardMessage(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get IDs from URL
	vars := mux.Vars(r)
	conversationID := vars["conversationId"]
	messageID := vars["messageId"]

	// Step 3: Check if user is part of source conversation
	_, err := h.db.GetConversation(authUserID, conversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 4: Get the message to forward
	originalMsg, err := h.db.GetMessage(messageID)
	if errors.Is(err, database.ErrMessageNotFound) {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 5: Parse the target conversation
	var req ForwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Step 6: Check if user is part of target conversation
	_, err = h.db.GetConversation(authUserID, req.TargetConversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Target conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 7: Create a new message in the target conversation
	// (forwarding creates a copy)
	msg, err := h.db.CreateMessage(req.TargetConversationID, authUserID, originalMsg.Content, originalMsg.Photo, nil)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 8: Return the forwarded message
	response := MessageResponse{
		MessageID:  msg.ID,
		SenderID:   msg.SenderID,
		SenderName: msg.SenderName,
		Content:    msg.Content,
		HasPhoto:   len(msg.Photo) > 0,
		Timestamp:  msg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		Status:     msg.Status,
	}

	writeJSON(w, http.StatusCreated, response)
}

/*
DeleteMessage handles DELETE /conversations/{conversationId}/messages/{messageId}
operationId: deleteMessage

From PDF:
"The user can... delete any sent messages."
Only the sender can delete their own messages.
*/
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get IDs from URL
	vars := mux.Vars(r)
	messageID := vars["messageId"]

	// Step 3: Delete the message
	err := h.db.DeleteMessage(messageID, authUserID)
	if errors.Is(err, database.ErrMessageNotFound) {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, database.ErrNotMessageOwner) {
		http.Error(w, "Cannot delete messages sent by others", http.StatusForbidden)
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
CommentMessage handles POST /conversations/{conversationId}/messages/{messageId}/comments
operationId: commentMessage

From PDF:
"Users can also react to messages (a.k.a. comment them) with an emoticon..."
*/
func (h *Handler) CommentMessage(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get IDs from URL
	vars := mux.Vars(r)
	conversationID := vars["conversationId"]
	messageID := vars["messageId"]

	// Step 3: Check if user is part of this conversation
	_, err := h.db.GetConversation(authUserID, conversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 4: Parse the request
	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Emoticon == "" {
		http.Error(w, "Emoticon is required", http.StatusBadRequest)
		return
	}

	// Step 5: Add the comment
	err = h.db.AddComment(messageID, authUserID, req.Emoticon)
	if errors.Is(err, database.ErrMessageNotFound) {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 6: Return success (201 Created)
	w.WriteHeader(http.StatusCreated)
}

/*
UncommentMessage handles DELETE /conversations/{conversationId}/messages/{messageId}/comments
operationId: uncommentMessage

From PDF:
"...and delete their reactions at any time (a.k.a. uncomment)."
*/
func (h *Handler) UncommentMessage(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get message ID from URL
	vars := mux.Vars(r)
	messageID := vars["messageId"]

	// Step 3: Remove the comment
	err := h.db.RemoveComment(messageID, authUserID)
	if errors.Is(err, database.ErrCommentNotFound) {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 4: Return success (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}
