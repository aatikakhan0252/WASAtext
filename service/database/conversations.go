/*
Database operations for Conversations.

This file handles getting conversation lists and conversation details.
*/
package database

import (
	"database/sql"

	"github.com/gofrs/uuid"
)

// GetConversations returns all conversations for a user, sorted by latest message
func (db *appdbimpl) GetConversations(userID string) ([]ConversationPreview, error) {
	// Query for all conversations the user is part of
	rows, err := db.db.Query(`
		SELECT 
			c.id,
			c.is_group,
			CASE 
				WHEN c.is_group = 1 THEN g.name
				ELSE (SELECT u.name FROM users u 
					  JOIN conversation_participants cp2 ON u.id = cp2.user_id 
					  WHERE cp2.conversation_id = c.id AND cp2.user_id != ?)
			END as name,
			CASE 
				WHEN c.is_group = 1 THEN g.photo
				ELSE (SELECT u.photo FROM users u 
					  JOIN conversation_participants cp2 ON u.id = cp2.user_id 
					  WHERE cp2.conversation_id = c.id AND cp2.user_id != ?)
			END as photo,
			(SELECT m.timestamp FROM messages m WHERE m.conversation_id = c.id ORDER BY m.timestamp DESC LIMIT 1) as last_msg_time,
			(SELECT m.content FROM messages m WHERE m.conversation_id = c.id ORDER BY m.timestamp DESC LIMIT 1) as last_msg_preview,
			(SELECT CASE WHEN m.photo IS NOT NULL THEN 1 ELSE 0 END FROM messages m WHERE m.conversation_id = c.id ORDER BY m.timestamp DESC LIMIT 1) as last_msg_is_photo
		FROM conversations c
		JOIN conversation_participants cp ON c.id = cp.conversation_id
		LEFT JOIN groups g ON c.group_id = g.id
		WHERE cp.user_id = ?
		ORDER BY last_msg_time DESC NULLS LAST
	`, userID, userID, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []ConversationPreview
	for rows.Next() {
		var conv ConversationPreview
		var photo sql.NullString
		var lastMsgTime sql.NullTime
		var lastMsgPreview sql.NullString
		var lastMsgIsPhoto sql.NullBool

		if err := rows.Scan(
			&conv.ID,
			&conv.IsGroup,
			&conv.Name,
			&photo,
			&lastMsgTime,
			&lastMsgPreview,
			&lastMsgIsPhoto,
		); err != nil {
			return nil, err
		}

		if photo.Valid {
			conv.Photo = []byte(photo.String)
		}
		if lastMsgTime.Valid {
			conv.LastMessageTime = lastMsgTime.Time
		}
		if lastMsgPreview.Valid {
			conv.LastMessagePreview = lastMsgPreview.String
		}
		if lastMsgIsPhoto.Valid {
			conv.LastMessageIsPhoto = lastMsgIsPhoto.Bool
		}

		conversations = append(conversations, conv)
	}

	return conversations, rows.Err()
}

// GetConversation returns a full conversation with all messages
func (db *appdbimpl) GetConversation(userID, conversationID string) (*Conversation, error) {
	// First, check if user is a participant
	var count int
	err := db.db.QueryRow(
		"SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = ? AND user_id = ?",
		conversationID, userID,
	).Scan(&count)

	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, ErrConversationNotFound
	}

	// Get conversation info
	var conv Conversation
	var isGroup bool
	var groupID sql.NullString

	err = db.db.QueryRow(
		"SELECT id, is_group, group_id FROM conversations WHERE id = ?",
		conversationID,
	).Scan(&conv.ID, &isGroup, &groupID)

	if err == sql.ErrNoRows {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}

	conv.IsGroup = isGroup

	// Get name and photo based on type
	if isGroup && groupID.Valid {
		group, err := db.GetGroup(groupID.String)
		if err != nil {
			return nil, err
		}
		conv.Name = group.Name
		conv.Photo = group.Photo
		conv.Members = group.Members
	} else {
		// Direct conversation - get the other user
		var otherUser User
		var photo sql.NullString

		err = db.db.QueryRow(`
			SELECT u.id, u.name, u.photo 
			FROM users u 
			JOIN conversation_participants cp ON u.id = cp.user_id 
			WHERE cp.conversation_id = ? AND cp.user_id != ?
		`, conversationID, userID).Scan(&otherUser.ID, &otherUser.Name, &photo)

		if err == nil {
			conv.Name = otherUser.Name
			if photo.Valid {
				conv.Photo = []byte(photo.String)
				otherUser.Photo = conv.Photo
			}
			conv.Members = []User{otherUser}
		}
	}

	// Get messages in reverse chronological order (as per PDF)
	messages, err := db.getConversationMessages(conversationID)
	if err != nil {
		return nil, err
	}
	conv.Messages = messages

	// Mark conversation as read (this updates message status)
	_ = db.MarkConversationAsRead(conversationID, userID)

	return &conv, nil
}

// getConversationMessages retrieves all messages for a conversation
func (db *appdbimpl) getConversationMessages(conversationID string) ([]Message, error) {
	rows, err := db.db.Query(`
		SELECT m.id, m.sender_id, u.name, m.content, m.photo, m.timestamp, m.status, m.reply_to
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.conversation_id = ?
		ORDER BY m.timestamp DESC
	`, conversationID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var content sql.NullString
		var photo sql.NullString
		var replyTo sql.NullString

		if err := rows.Scan(
			&msg.ID,
			&msg.SenderID,
			&msg.SenderName,
			&content,
			&photo,
			&msg.Timestamp,
			&msg.Status,
			&replyTo,
		); err != nil {
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

		// Get comments for this message
		comments, err := db.getMessageComments(msg.ID)
		if err != nil {
			return nil, err
		}
		msg.Comments = comments

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// getMessageComments retrieves all comments (reactions) on a message
func (db *appdbimpl) getMessageComments(messageID string) ([]Comment, error) {
	rows, err := db.db.Query(`
		SELECT c.user_id, u.name, c.emoticon
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.message_id = ?
	`, messageID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		if err := rows.Scan(&comment.UserID, &comment.UserName, &comment.Emoticon); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	return comments, rows.Err()
}

// GetOrCreateDirectConversation gets or creates a direct conversation between two users
func (db *appdbimpl) GetOrCreateDirectConversation(userID, otherUserID string) (string, error) {
	// Check if conversation already exists
	var convID string
	err := db.db.QueryRow(`
		SELECT cp1.conversation_id 
		FROM conversation_participants cp1
		JOIN conversation_participants cp2 ON cp1.conversation_id = cp2.conversation_id
		JOIN conversations c ON cp1.conversation_id = c.id
		WHERE cp1.user_id = ? AND cp2.user_id = ? AND c.is_group = 0
	`, userID, otherUserID).Scan(&convID)

	if err == nil {
		return convID, nil // Already exists
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// Create new conversation
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	tx, err := db.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Create conversation
	_, err = tx.Exec(
		"INSERT INTO conversations (id, is_group) VALUES (?, 0)",
		id.String(),
	)
	if err != nil {
		return "", err
	}

	// Add both participants
	_, err = tx.Exec(
		"INSERT INTO conversation_participants (conversation_id, user_id) VALUES (?, ?)",
		id.String(), userID,
	)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(
		"INSERT INTO conversation_participants (conversation_id, user_id) VALUES (?, ?)",
		id.String(), otherUserID,
	)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return id.String(), nil
}

// MarkConversationAsRead marks all messages in a conversation as read for a user
func (db *appdbimpl) MarkConversationAsRead(conversationID, userID string) error {
	// Update the last_read_time for this user
	_, err := db.db.Exec(`
		UPDATE conversation_participants 
		SET last_read_time = CURRENT_TIMESTAMP 
		WHERE conversation_id = ? AND user_id = ?
	`, conversationID, userID)
	if err != nil {
		return err
	}

	// Update message status to 'read' for messages sent by others
	// This is simplified - in real app, you'd track per-user read status
	_, err = db.db.Exec(`
		UPDATE messages 
		SET status = 'read' 
		WHERE conversation_id = ? AND sender_id != ? AND status != 'read'
	`, conversationID, userID)

	return err
}
