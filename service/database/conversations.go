// Package database — Database operations for Conversations.
package database

import (
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// GetConversations returns all conversations for a user, sorted by latest message.
func (db *appdbimpl) GetConversations(userID string) ([]ConversationPreview, error) {
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
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			logrus.WithError(closeErr).Error("failed to close rows")
		}
	}()

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

// GetConversation returns a full conversation with all messages.
func (db *appdbimpl) GetConversation(userID, conversationID string) (*Conversation, error) {
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

	var conv Conversation
	var isGroup bool
	var groupID sql.NullString

	err = db.db.QueryRow(
		"SELECT id, is_group, group_id FROM conversations WHERE id = ?",
		conversationID,
	).Scan(&conv.ID, &isGroup, &groupID)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}

	conv.IsGroup = isGroup

	if isGroup && groupID.Valid {
		conv.GroupID = groupID.String
		group, groupErr := db.GetGroup(groupID.String)
		if groupErr != nil {
			return nil, groupErr
		}
		conv.Name = group.Name
		conv.Photo = group.Photo
		conv.Members = group.Members
	} else {
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

	messages, msgErr := db.getConversationMessages(conversationID, userID)
	if msgErr != nil {
		return nil, msgErr
	}
	conv.Messages = messages

	// Mark as read for this user + update message statuses
	if markErr := db.MarkConversationAsRead(conversationID, userID); markErr != nil {
		logrus.WithError(markErr).Warn("failed to mark conversation as read")
	}

	return &conv, nil
}

// getConversationMessages retrieves all messages for a conversation with computed status.
func (db *appdbimpl) getConversationMessages(conversationID, viewerID string) ([]Message, error) {
	rows, err := db.db.Query(`
		SELECT m.id, m.sender_id, u.name, m.content, m.photo, m.timestamp, m.status, m.reply_to, m.is_forwarded
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.conversation_id = ?
		ORDER BY m.timestamp ASC
	`, conversationID)

	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			logrus.WithError(closeErr).Error("failed to close rows")
		}
	}()

	// Get participant count for computing group read receipts
	var participantCount int
	if countErr := db.db.QueryRow(
		"SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = ?",
		conversationID,
	).Scan(&participantCount); countErr != nil {
		return nil, countErr
	}

	var messages []Message
	for rows.Next() {
		var msg Message
		var content sql.NullString
		var photo sql.NullString
		var replyTo sql.NullString
		var isForwarded sql.NullBool

		if err := rows.Scan(
			&msg.ID,
			&msg.SenderID,
			&msg.SenderName,
			&content,
			&photo,
			&msg.Timestamp,
			&msg.Status,
			&replyTo,
			&isForwarded,
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
		if isForwarded.Valid {
			msg.IsForwarded = isForwarded.Bool
		}

		// Compute effective status for this message
		if msg.SenderID != viewerID {
			// Messages from others: viewer is reading them now, so they are "read"
			msg.Status = StatusRead
		} else {
			// Messages from self: compute based on how many recipients have read
			msg.Status = db.computeMessageStatus(msg.ID, msg.SenderID, conversationID, participantCount)
		}

		db.fillReplySnippet(&msg)

		comments, commentErr := db.getMessageComments(msg.ID)
		if commentErr != nil {
			return nil, commentErr
		}
		msg.Comments = comments

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// computeMessageStatus determines the display status for a sent message:
// - "sent": no recipient has opened the conversation since the message was sent
// - "received": at least one (but not all) recipients have read it (for groups)
// - "read": ALL recipients have opened the conversation after the message timestamp
func (db *appdbimpl) computeMessageStatus(messageID, senderID, conversationID string, participantCount int) string {
	// Get the message timestamp
	var msgTimestamp sql.NullTime
	if err := db.db.QueryRow(
		"SELECT timestamp FROM messages WHERE id = ?",
		messageID,
	).Scan(&msgTimestamp); err != nil || !msgTimestamp.Valid {
		return StatusSent
	}

	// Count how many NON-SENDER participants have a last_read_time >= message timestamp
	var readCount int
	if err := db.db.QueryRow(`
		SELECT COUNT(*) FROM conversation_participants 
		WHERE conversation_id = ? AND user_id != ? AND last_read_time >= ?
	`, conversationID, senderID, msgTimestamp.Time).Scan(&readCount); err != nil {
		return StatusSent
	}

	otherParticipants := participantCount - 1
	if otherParticipants <= 0 {
		return StatusSent
	}

	if readCount >= otherParticipants {
		return StatusRead
	}
	if readCount > 0 {
		return StatusReceived
	}
	return StatusSent
}

// getMessageComments retrieves all comments (reactions) on a message.
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
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			logrus.WithError(closeErr).Error("failed to close rows")
		}
	}()

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

// GetOrCreateDirectConversation gets or creates a direct conversation between two users.
func (db *appdbimpl) GetOrCreateDirectConversation(userID, otherUserID string) (string, error) {
	var convID string
	err := db.db.QueryRow(`
		SELECT cp1.conversation_id 
		FROM conversation_participants cp1
		JOIN conversation_participants cp2 ON cp1.conversation_id = cp2.conversation_id
		JOIN conversations c ON cp1.conversation_id = c.id
		WHERE cp1.user_id = ? AND cp2.user_id = ? AND c.is_group = 0
	`, userID, otherUserID).Scan(&convID)

	if err == nil {
		return convID, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	tx, err := db.db.Begin()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(
		"INSERT INTO conversations (id, is_group) VALUES (?, 0)",
		id.String(),
	)
	if err != nil {
		return "", err
	}

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

// MarkConversationAsRead updates this user's last_read_time to now.
func (db *appdbimpl) MarkConversationAsRead(conversationID, userID string) error {
	_, err := db.db.Exec(`
		UPDATE conversation_participants 
		SET last_read_time = CURRENT_TIMESTAMP 
		WHERE conversation_id = ? AND user_id = ?
	`, conversationID, userID)
	return err
}
