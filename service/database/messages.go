/*
Database operations for Messages and Comments (reactions).

This file handles sending, forwarding, deleting messages,
and adding/removing reactions.
*/
package database

import (
	"database/sql"
	"time"

	"github.com/gofrs/uuid"
)

// CreateMessage creates a new message in a conversation
func (db *appdbimpl) CreateMessage(conversationID, senderID, content string, photo []byte, replyTo *string) (*Message, error) {
	// Generate message ID
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	timestamp := time.Now()

	// Handle nullable fields
	var contentVal interface{}
	if content != "" {
		contentVal = content
	}

	var photoVal interface{}
	if photo != nil {
		photoVal = photo
	}

	var replyToVal interface{}
	if replyTo != nil && *replyTo != "" {
		replyToVal = *replyTo
	}

	// Insert the message
	_, err = db.db.Exec(`
		INSERT INTO messages (id, conversation_id, sender_id, content, photo, timestamp, status, reply_to)
		VALUES (?, ?, ?, ?, ?, ?, 'sent', ?)
	`, id.String(), conversationID, senderID, contentVal, photoVal, timestamp, replyToVal)

	if err != nil {
		return nil, err
	}

	// Update message status to 'received' for other participants
	// (In a real app, this would happen when they fetch their conversations)
	go db.updateMessageStatusForRecipients(id.String(), conversationID, senderID)

	// Get sender name
	sender, err := db.GetUserByID(senderID)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:         id.String(),
		SenderID:   senderID,
		SenderName: sender.Name,
		Content:    content,
		Photo:      photo,
		Timestamp:  timestamp,
		Status:     "sent",
		ReplyTo:    replyTo,
		Comments:   []Comment{},
	}, nil
}

// updateMessageStatusForRecipients marks message as received
func (db *appdbimpl) updateMessageStatusForRecipients(messageID, conversationID, senderID string) {
	// Check if all other participants have "seen" the message in their list
	// For simplicity, we mark as received immediately
	_, _ = db.db.Exec(
		"UPDATE messages SET status = 'received' WHERE id = ?",
		messageID,
	)
}

// GetMessage retrieves a single message by ID
func (db *appdbimpl) GetMessage(messageID string) (*Message, error) {
	var msg Message
	var content sql.NullString
	var photo sql.NullString
	var replyTo sql.NullString

	err := db.db.QueryRow(`
		SELECT m.id, m.sender_id, u.name, m.content, m.photo, m.timestamp, m.status, m.reply_to
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.id = ?
	`, messageID).Scan(
		&msg.ID,
		&msg.SenderID,
		&msg.SenderName,
		&content,
		&photo,
		&msg.Timestamp,
		&msg.Status,
		&replyTo,
	)

	if err == sql.ErrNoRows {
		return nil, ErrMessageNotFound
	}
	if err != nil {
		return nil, err
	}

	if content.Valid {
		msg.Content = content.String
	}
	if photo.Valid {
		msg.Photo = []byte(photo.String)
	}
	if replyTo.Valid {
		msg.ReplyTo = &replyTo.String
	}

	// Get comments
	comments, err := db.getMessageComments(messageID)
	if err != nil {
		return nil, err
	}
	msg.Comments = comments

	return &msg, nil
}

// DeleteMessage deletes a message (only the sender can delete their own messages)
func (db *appdbimpl) DeleteMessage(messageID, userID string) error {
	// First, check if the message exists and belongs to the user
	var senderID string
	err := db.db.QueryRow(
		"SELECT sender_id FROM messages WHERE id = ?",
		messageID,
	).Scan(&senderID)

	if err == sql.ErrNoRows {
		return ErrMessageNotFound
	}
	if err != nil {
		return err
	}

	if senderID != userID {
		return ErrNotMessageOwner
	}

	// Delete all comments on this message first
	_, err = db.db.Exec("DELETE FROM comments WHERE message_id = ?", messageID)
	if err != nil {
		return err
	}

	// Delete the message
	_, err = db.db.Exec("DELETE FROM messages WHERE id = ?", messageID)
	return err
}

// UpdateMessageStatus updates the status of a message
func (db *appdbimpl) UpdateMessageStatus(messageID, status string) error {
	result, err := db.db.Exec(
		"UPDATE messages SET status = ? WHERE id = ?",
		status, messageID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrMessageNotFound
	}

	return nil
}

// AddComment adds a reaction (comment) to a message
func (db *appdbimpl) AddComment(messageID, userID, emoticon string) error {
	// Check if message exists
	_, err := db.GetMessage(messageID)
	if err != nil {
		return err
	}

	// Insert or replace the comment (one reaction per user per message)
	_, err = db.db.Exec(`
		INSERT OR REPLACE INTO comments (message_id, user_id, emoticon)
		VALUES (?, ?, ?)
	`, messageID, userID, emoticon)

	return err
}

// RemoveComment removes a user's reaction from a message
func (db *appdbimpl) RemoveComment(messageID, userID string) error {
	result, err := db.db.Exec(
		"DELETE FROM comments WHERE message_id = ? AND user_id = ?",
		messageID, userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrCommentNotFound
	}

	return nil
}
