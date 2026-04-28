// Package api — Message API handlers.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

// SendMessageRequest is the body for POST /conversations/:conversationId/messages.
type SendMessageRequest struct {
	Content string `json:"content,omitempty"`
	ReplyTo string `json:"replyTo,omitempty"`
}

// ForwardMessageRequest is the body for POST .../forward.
type ForwardMessageRequest struct {
	TargetConversationID string `json:"targetConversationId"`
}

// CommentRequest is the body for POST .../comments.
type CommentRequest struct {
	Emoticon string `json:"emoticon"`
}

// SendMessage handles POST /conversations/:conversationId/messages (operationId: sendMessage).
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversationID := ps.ByName("conversationId")

	_, err := h.db.GetConversation(authUserID, conversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to get conversation")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("Content-Type")

	var content string
	var photo []byte
	var replyTo *string

	if strings.Contains(contentType, "multipart/form-data") {
		if parseErr := r.ParseMultipartForm(10 << 20); parseErr != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// Read text content from form field
		content = r.FormValue("content")

		file, _, fileErr := r.FormFile("photo")
		if fileErr == nil {
			var readErr error
			photo, readErr = io.ReadAll(file)
			if readErr != nil {
				h.logger.WithError(readErr).Error("failed to read photo file")
				http.Error(w, "Failed to read photo", http.StatusBadRequest)
				return
			}
			if closeErr := file.Close(); closeErr != nil {
				h.logger.WithError(closeErr).Error("failed to close photo file")
			}
		}

		if replyToVal := r.FormValue("replyTo"); replyToVal != "" {
			replyTo = &replyToVal
		}
	} else {
		var req SendMessageRequest
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		content = req.Content
		if req.ReplyTo != "" {
			replyTo = &req.ReplyTo
		}
	}

	if content == "" && len(photo) == 0 {
		http.Error(w, "Message must have content or photo", http.StatusBadRequest)
		return
	}

	msg, err := h.db.CreateMessage(conversationID, authUserID, content, photo, replyTo, false)
	if err != nil {
		h.logger.WithError(err).Error("failed to create message")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := MessageResponse{
		MessageID:       msg.ID,
		SenderID:        msg.SenderID,
		SenderName:      msg.SenderName,
		Content:         msg.Content,
		HasPhoto:        len(msg.Photo) > 0,
		Timestamp:       msg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		Status:          msg.Status,
		ReplyToContent:  msg.ReplyToContent,
		ReplyToHasPhoto: msg.ReplyToHasPhoto,
		Comments:        []CommentResponse{},
	}

	if msg.ReplyTo != nil {
		response.ReplyTo = *msg.ReplyTo
	}

	writeJSON(w, http.StatusCreated, response, h.logger)
}

// ForwardMessage handles POST .../forward (operationId: forwardMessage).
func (h *Handler) ForwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversationID := ps.ByName("conversationId")
	messageID := ps.ByName("messageId")

	_, err := h.db.GetConversation(authUserID, conversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to get conversation")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	originalMsg, err := h.db.GetMessage(messageID)
	if errors.Is(err, database.ErrMessageNotFound) {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to get message")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var req ForwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = h.db.GetConversation(authUserID, req.TargetConversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Target conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to get target conversation")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Forward with isForwarded = true
	msg, err := h.db.CreateMessage(req.TargetConversationID, authUserID, originalMsg.Content, originalMsg.Photo, nil, true)
	if err != nil {
		h.logger.WithError(err).Error("failed to forward message")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := MessageResponse{
		MessageID:   msg.ID,
		SenderID:    msg.SenderID,
		SenderName:  msg.SenderName,
		Content:     msg.Content,
		HasPhoto:    len(msg.Photo) > 0,
		Timestamp:   msg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		Status:      msg.Status,
		IsForwarded: true,
		Comments:    []CommentResponse{},
	}

	writeJSON(w, http.StatusCreated, response, h.logger)
}

// DeleteMessage handles DELETE .../messages/:messageId (operationId: deleteMessage).
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageID := ps.ByName("messageId")

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
		h.logger.WithError(err).Error("failed to delete message")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetMessagePhoto serves the photo of a message as binary.
func (h *Handler) GetMessagePhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageID := ps.ByName("messageId")

	msg, err := h.db.GetMessage(messageID)
	if errors.Is(err, database.ErrMessageNotFound) {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to get message")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(msg.Photo) == 0 {
		http.Error(w, "No photo", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "max-age=3600")
	if _, writeErr := w.Write(msg.Photo); writeErr != nil {
		h.logger.WithError(writeErr).Error("failed to write photo response")
	}
}

// CommentMessage handles POST .../comments (operationId: commentMessage).
func (h *Handler) CommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversationID := ps.ByName("conversationId")
	messageID := ps.ByName("messageId")

	_, err := h.db.GetConversation(authUserID, conversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to get conversation")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Emoticon == "" {
		http.Error(w, "Emoticon is required", http.StatusBadRequest)
		return
	}

	err = h.db.AddComment(messageID, authUserID, req.Emoticon)
	if errors.Is(err, database.ErrMessageNotFound) {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to add comment")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// UncommentMessage handles DELETE .../comments (operationId: uncommentMessage).
func (h *Handler) UncommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageID := ps.ByName("messageId")

	err := h.db.RemoveComment(messageID, authUserID)
	if errors.Is(err, database.ErrCommentNotFound) {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to remove comment")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
