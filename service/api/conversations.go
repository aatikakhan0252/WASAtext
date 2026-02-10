/*
Conversation API handlers.

This file contains:
- getMyConversations: Get list of all conversations
- getConversation: Get a specific conversation with messages
- startConversation: Start a new direct conversation
*/
package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"wasatext/service/database"
)

// ConversationPreviewResponse is used for the conversation list
type ConversationPreviewResponse struct {
	ConversationID     string `json:"conversationId"`
	IsGroup            bool   `json:"isGroup"`
	Name               string `json:"name"`
	HasPhoto           bool   `json:"hasPhoto"`
	LastMessageTime    string `json:"lastMessageTimestamp,omitempty"`
	LastMessagePreview string `json:"lastMessagePreview,omitempty"`
	LastMessageIsPhoto bool   `json:"lastMessageIsPhoto"`
}

// ConversationResponse is the full conversation with messages
type ConversationResponse struct {
	ConversationID string            `json:"conversationId"`
	IsGroup        bool              `json:"isGroup"`
	Name           string            `json:"name"`
	HasPhoto       bool              `json:"hasPhoto"`
	Members        []UserResponse    `json:"members,omitempty"`
	Messages       []MessageResponse `json:"messages"`
}

// MessageResponse represents a message
type MessageResponse struct {
	MessageID  string            `json:"messageId"`
	SenderID   string            `json:"senderId"`
	SenderName string            `json:"senderName"`
	Content    string            `json:"content,omitempty"`
	HasPhoto   bool              `json:"hasPhoto"`
	Timestamp  string            `json:"timestamp"`
	Status     string            `json:"status"` // sent, received, read
	ReplyTo    string            `json:"replyTo,omitempty"`
	Comments   []CommentResponse `json:"comments"`
}

// CommentResponse represents a reaction
type CommentResponse struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
	Emoticon string `json:"emoticon"`
}

// StartConversationRequest is the body for POST /conversations
type StartConversationRequest struct {
	UserID string `json:"userId"` // User to start conversation with
}

/*
GetMyConversations handles GET /conversations
operationId: getMyConversations

From PDF:
"The user is presented with a list of conversations with other users or
with groups, sorted in reverse chronological order. Each element in the
list must display the username of the other person or the group name,
the user profile photo or the group photo, the date and time of the
latest message, the preview (snippet) of the text message, or an icon
for a photo message."
*/
func (h *Handler) GetMyConversations(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get conversations from database
	conversations, err := h.db.GetConversations(authUserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 3: Convert to response format
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

	// Step 4: Return the conversations
	writeJSON(w, http.StatusOK, response)
}

/*
GetConversation handles GET /conversations/{conversationId}
operationId: getConversation

From PDF:
"The user can open a conversation to view all exchanged messages,
displayed in reverse chronological order. Each message includes the
timestamp, the content (whether text or photo), and the sender's
username for received messages, or one/two checkmarks to indicate
the status of sent messages. Any reactions (comments) on messages
are also displayed, along with the names of the users who posted them."
*/
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Get conversation ID from URL
	vars := mux.Vars(r)
	conversationID := vars["conversationId"]

	// Step 3: Get conversation from database
	conv, err := h.db.GetConversation(authUserID, conversationID)
	if err == database.ErrConversationNotFound {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 4: Convert to response format
	response := ConversationResponse{
		ConversationID: conv.ID,
		IsGroup:        conv.IsGroup,
		Name:           conv.Name,
		HasPhoto:       len(conv.Photo) > 0,
	}

	// Add members
	for _, m := range conv.Members {
		response.Members = append(response.Members, UserResponse{
			Identifier: m.ID,
			Name:       m.Name,
			HasPhoto:   len(m.Photo) > 0,
		})
	}

	// Add messages
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

		// Add comments (reactions)
		for _, c := range msg.Comments {
			msgResp.Comments = append(msgResp.Comments, CommentResponse{
				UserID:   c.UserID,
				UserName: c.UserName,
				Emoticon: c.Emoticon,
			})
		}

		response.Messages = append(response.Messages, msgResp)
	}

	// Step 5: Return the conversation
	writeJSON(w, http.StatusOK, response)
}

/*
StartConversation handles POST /conversations
This allows a user to start a new conversation with another user.

From PDF:
"The user can start a new conversation with any other user of WASAText,
and this conversation will automatically be added to the list."
*/
func (h *Handler) StartConversation(w http.ResponseWriter, r *http.Request) {
	// Step 1: Check authentication
	authUserID := getUserIDFromAuth(r)
	if authUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 2: Parse request body
	var req StartConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Step 3: Check if the other user exists
	_, err := h.db.GetUserByID(req.UserID)
	if err == database.ErrUserNotFound {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 4: Get or create the conversation
	convID, err := h.db.GetOrCreateDirectConversation(authUserID, req.UserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Step 5: Return the conversation ID
	writeJSON(w, http.StatusCreated, map[string]string{
		"conversationId": convID,
	})
}
