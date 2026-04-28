// Package database — Database operations for Groups.
package database

import (
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// CreateGroup creates a new group and adds the creator and initial members.
func (db *appdbimpl) CreateGroup(name string, creatorID string, memberIDs []string) (*Group, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	tx, err := db.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", id.String(), name)
	if err != nil {
		return nil, err
	}

	convID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(
		"INSERT INTO conversations (id, is_group, group_id) VALUES (?, 1, ?)",
		convID.String(), id.String(),
	)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		"INSERT INTO group_members (group_id, user_id) VALUES (?, ?)",
		id.String(), creatorID,
	)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		"INSERT INTO conversation_participants (conversation_id, user_id) VALUES (?, ?)",
		convID.String(), creatorID,
	)
	if err != nil {
		return nil, err
	}

	for _, memberID := range memberIDs {
		if memberID == creatorID {
			continue
		}

		_, err = tx.Exec(
			"INSERT INTO group_members (group_id, user_id) VALUES (?, ?)",
			id.String(), memberID,
		)
		if err != nil {
			return nil, err
		}

		_, err = tx.Exec(
			"INSERT INTO conversation_participants (conversation_id, user_id) VALUES (?, ?)",
			convID.String(), memberID,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return db.GetGroup(id.String())
}

// GetGroup retrieves a group by ID with all its members.
func (db *appdbimpl) GetGroup(groupID string) (*Group, error) {
	var group Group
	var photo sql.NullString

	err := db.db.QueryRow(
		"SELECT id, name, photo FROM groups WHERE id = ?",
		groupID,
	).Scan(&group.ID, &group.Name, &photo)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}

	if photo.Valid {
		group.Photo = []byte(photo.String)
	}

	// Get associated conversation ID
	_ = db.db.QueryRow(
		"SELECT id FROM conversations WHERE group_id = ?",
		groupID,
	).Scan(&group.ConversationID)

	rows, err := db.db.Query(`
		SELECT u.id, u.name, u.photo 
		FROM users u 
		JOIN group_members gm ON u.id = gm.user_id 
		WHERE gm.group_id = ?
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			logrus.WithError(closeErr).Error("failed to close rows")
		}
	}()

	for rows.Next() {
		var user User
		var userPhoto sql.NullString

		if err := rows.Scan(&user.ID, &user.Name, &userPhoto); err != nil {
			return nil, err
		}

		if userPhoto.Valid {
			user.Photo = []byte(userPhoto.String)
		}

		group.Members = append(group.Members, user)
	}

	return &group, rows.Err()
}

// GetGroupByConversationID retrieves a group by its conversation ID.
func (db *appdbimpl) GetGroupByConversationID(conversationID string) (*Group, error) {
	var groupID string
	err := db.db.QueryRow(
		"SELECT group_id FROM conversations WHERE id = ? AND is_group = 1",
		conversationID,
	).Scan(&groupID)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}

	return db.GetGroup(groupID)
}

// GetGroupPhoto retrieves just the photo bytes for a group.
func (db *appdbimpl) GetGroupPhoto(groupID string) ([]byte, error) {
	var photo sql.NullString
	err := db.db.QueryRow(
		"SELECT photo FROM groups WHERE id = ?",
		groupID,
	).Scan(&photo)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	if !photo.Valid || len(photo.String) == 0 {
		return nil, ErrGroupNotFound
	}
	return []byte(photo.String), nil
}

// AddUserToGroup adds a user to a group.
func (db *appdbimpl) AddUserToGroup(groupID, userID, adderID string) error {
	isMember, err := db.IsGroupMember(groupID, adderID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotGroupMember
	}

	_, err = db.GetUserByID(userID)
	if err != nil {
		return err
	}

	var convID string
	err = db.db.QueryRow(
		"SELECT id FROM conversations WHERE group_id = ?",
		groupID,
	).Scan(&convID)
	if err != nil {
		return err
	}

	_, err = db.db.Exec(
		"INSERT OR IGNORE INTO group_members (group_id, user_id) VALUES (?, ?)",
		groupID, userID,
	)
	if err != nil {
		return err
	}

	_, err = db.db.Exec(
		"INSERT OR IGNORE INTO conversation_participants (conversation_id, user_id) VALUES (?, ?)",
		convID, userID,
	)

	return err
}

// RemoveUserFromGroup removes a user from a group (for leaving).
func (db *appdbimpl) RemoveUserFromGroup(groupID, userID string) error {
	isMember, err := db.IsGroupMember(groupID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotGroupMember
	}

	var convID string
	err = db.db.QueryRow(
		"SELECT id FROM conversations WHERE group_id = ?",
		groupID,
	).Scan(&convID)
	if err != nil {
		return err
	}

	_, err = db.db.Exec(
		"DELETE FROM group_members WHERE group_id = ? AND user_id = ?",
		groupID, userID,
	)
	if err != nil {
		return err
	}

	_, err = db.db.Exec(
		"DELETE FROM conversation_participants WHERE conversation_id = ? AND user_id = ?",
		convID, userID,
	)

	return err
}

// UpdateGroupName changes the group's name.
func (db *appdbimpl) UpdateGroupName(groupID, name string) error {
	result, err := db.db.Exec(
		"UPDATE groups SET name = ? WHERE id = ?",
		name, groupID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrGroupNotFound
	}

	return nil
}

// UpdateGroupPhoto sets or updates the group's photo.
func (db *appdbimpl) UpdateGroupPhoto(groupID string, photo []byte) error {
	result, err := db.db.Exec(
		"UPDATE groups SET photo = ? WHERE id = ?",
		photo, groupID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrGroupNotFound
	}

	return nil
}

// IsGroupMember checks if a user is a member of a group.
func (db *appdbimpl) IsGroupMember(groupID, userID string) (bool, error) {
	var count int
	err := db.db.QueryRow(
		"SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?",
		groupID, userID,
	).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
