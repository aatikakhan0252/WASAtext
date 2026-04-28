// Package database — Database operations for Messages and Comments.
package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

// CreateMessage creates a new message in a conversation.
func (db *appdbimpl) CreateMessage(conversationID, senderID, content string, photo []byte, replyTo *string, isForwarded bool) (*Message, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	timestamp := time.Now()

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

	forwardedInt := 0
	if isForwarded {
		forwardedInt = 1
	}

	_, err = db.db.Exec(`
		INSERT INTO messages (id, conversation_id, sender_id, content, photo, timestamp, status, reply_to, is_forwarded)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id.String(), conversationID, senderID, contentVal, photoVal, timestamp, StatusSent, replyToVal, forwardedInt)

	if err != nil {
		return nil, err
	}

	sender, err := db.GetUserByID(senderID)
	if err != nil {
		return nil, err
	}

	msg := &Message{
		ID:          id.String(),
		SenderID:    senderID,
		SenderName:  sender.Name,
		Content:     content,
		Photo:       photo,
		Timestamp:   timestamp,
		Status:      StatusSent,
		ReplyTo:     replyTo,
		IsForwarded: isForwarded,
		Comments:    []Comment{},
	}

	// Fill reply snippet
	if replyTo != nil && *replyTo != "" {
		db.fillReplySnippet(msg)
	}

	return msg, nil
}

// fillReplySnippet populates replyToContent and replyToHasPhoto.
func (db *appdbimpl) fillReplySnippet(msg *Message) {
	if msg.ReplyTo == nil || *msg.ReplyTo == "" {
		return
	}
	var content sql.NullString
	var photo sql.NullString
	err := db.db.QueryRow(
		"SELECT content, photo FROM messages WHERE id = ?",
		*msg.ReplyTo,
	).Scan(&content, &photo)
	if err != nil {
		return
	}
	if content.Valid {
		msg.ReplyToContent = content.String
	}
	if photo.Valid && len(photo.String) > 0 {
		msg.ReplyToHasPhoto = true
	}
}

// GetMessage retrieves a single message by ID.
func (db *appdbimpl) GetMessage(messageID string) (*Message, error) {
	var msg Message
	var content sql.NullString
	var photo sql.NullString
	var replyTo sql.NullString
	var isForwarded sql.NullBool

	err := db.db.QueryRow(`
		SELECT m.id, m.sender_id, u.name, m.content, m.photo, m.timestamp, m.status, m.reply_to, m.is_forwarded
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
		&isForwarded,
	)

	if errors.Is(err, sql.ErrNoRows) {
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
	if isForwarded.Valid {
		msg.IsForwarded = isForwarded.Bool
	}

	db.fillReplySnippet(&msg)

	comments, err := db.getMessageComments(messageID)
	if err != nil {
		return nil, err
	}
	msg.Comments = comments

	return &msg, nil
}

// DeleteMessage deletes a message (only the sender can delete their own messages).
func (db *appdbimpl) DeleteMessage(messageID, userID string) error {
	var senderID string
	err := db.db.QueryRow(
		"SELECT sender_id FROM messages WHERE id = ?",
		messageID,
	).Scan(&senderID)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrMessageNotFound
	}
	if err != nil {
		return err
	}

	if senderID != userID {
		return ErrNotMessageOwner
	}

	_, err = db.db.Exec("DELETE FROM comments WHERE message_id = ?", messageID)
	if err != nil {
		return err
	}

	_, err = db.db.Exec("DELETE FROM messages WHERE id = ?", messageID)
	return err
}

// UpdateMessageStatus updates the status of a message.
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

// AddComment adds a reaction (comment) to a message.
func (db *appdbimpl) AddComment(messageID, userID, emoticon string) error {
	_, err := db.GetMessage(messageID)
	if err != nil {
		return err
	}

	_, err = db.db.Exec(`
		INSERT OR REPLACE INTO comments (message_id, user_id, emoticon)
		VALUES (?, ?, ?)
	`, messageID, userID, emoticon)

	return err
}

// RemoveComment removes a user's reaction from a message.
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
