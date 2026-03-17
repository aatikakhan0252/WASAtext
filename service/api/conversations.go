// Package api — Conversation API handlers.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"wasatext/service/database"

	"github.com/julienschmidt/httprouter"
)

// ConversationPreviewResponse is used for the conversation list.
type ConversationPreviewResponse struct {
	ConversationID     string `json:"conversationId"`
	IsGroup            bool   `json:"isGroup"`
	Name               string `json:"name"`
	HasPhoto           bool   `json:"hasPhoto"`
	LastMessageTime    string `json:"lastMessageTimestamp,omitempty"`
	LastMessagePreview string `json:"lastMessagePreview,omitempty"`
	LastMessageIsPhoto bool   `json:"lastMessageIsPhoto"`
}

// ConversationResponse is the full conversation with messages.
type ConversationResponse struct {
	ConversationID string            `json:"conversationId"`
	IsGroup        bool              `json:"isGroup"`
	Name           string            `json:"name"`
	HasPhoto       bool              `json:"hasPhoto"`
	Members        []UserResponse    `json:"members,omitempty"`
	Messages       []MessageResponse `json:"messages"`
}

// MessageResponse represents a message.
type MessageResponse struct {
	MessageID  string            `json:"messageId"`
	SenderID   string            `json:"senderId"`
	SenderName string            `json:"senderName"`
	Content    string            `json:"content,omitempty"`
	HasPhoto   bool              `json:"hasPhoto"`
	Timestamp  string            `json:"timestamp"`
	Status     string            `json:"status"`
	ReplyTo    string            `json:"replyTo,omitempty"`
	Comments   []CommentResponse `json:"comments"`
}

// CommentResponse represents a reaction.
type CommentResponse struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
	Emoticon string `json:"emoticon"`
}

// StartConversationRequest is the body for POST /conversations.
type StartConversationRequest struct {
	UserID string `json:"userId"`
}

// GetMyConversations handles GET /conversations (operationId: getMyConversations).
func (h *Handler) GetMyConversations(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversations, err := h.db.GetConversations(authUserID)
	if err != nil {
		h.logger.WithError(err).Error("failed to get conversations")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var response []ConversationPreviewResponse
	for _, c := range conversations {
		preview := ConversationPreviewResponse{
			ConversationID:     c.ID,
			IsGroup:            c.IsGroup,
			Name:               c.Name,
			HasPhoto:           len(c.Photo) > 0,
			LastMessagePreview: c.LastMessagePreview,
			LastMessageIsPhoto: c.LastMessageIsPhoto,
		}

		if !c.LastMessageTime.IsZero() {
			preview.LastMessageTime = c.LastMessageTime.Format("2006-01-02T15:04:05Z07:00")
		}

		response = append(response, preview)
	}

	writeJSON(w, http.StatusOK, response, h.logger)
}

// GetConversation handles GET /conversations/:conversationId (operationId: getConversation).
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversationID := ps.ByName("conversationId")

	conv, err := h.db.GetConversation(authUserID, conversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to get conversation")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := ConversationResponse{
		ConversationID: conv.ID,
		IsGroup:        conv.IsGroup,
		Name:           conv.Name,
		HasPhoto:       len(conv.Photo) > 0,
	}

	for _, m := range conv.Members {
		response.Members = append(response.Members, UserResponse{
			Identifier: m.ID,
			Name:       m.Name,
			HasPhoto:   len(m.Photo) > 0,
		})
	}

	for _, msg := range conv.Messages {
		msgResp := MessageResponse{
			MessageID:  msg.ID,
			SenderID:   msg.SenderID,
			SenderName: msg.SenderName,
			Content:    msg.Content,
			HasPhoto:   len(msg.Photo) > 0,
			Timestamp:  msg.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Status:     msg.Status,
		}

		if msg.ReplyTo != nil {
			msgResp.ReplyTo = *msg.ReplyTo
		}

		for _, c := range msg.Comments {
			msgResp.Comments = append(msgResp.Comments, CommentResponse{
				UserID:   c.UserID,
				UserName: c.UserName,
				Emoticon: c.Emoticon,
			})
		}

		response.Messages = append(response.Messages, msgResp)
	}

	writeJSON(w, http.StatusOK, response, h.logger)
}

// StartConversation handles POST /conversations (operationId: startConversation).
func (h *Handler) StartConversation(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req StartConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.db.GetUserByID(req.UserID)
	if errors.Is(err, database.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.WithError(err).Error("failed to look up user")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	convID, err := h.db.GetOrCreateDirectConversation(authUserID, req.UserID)
	if err != nil {
		h.logger.WithError(err).Error("failed to start conversation")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"conversationId": convID,
	}, h.logger)
}
