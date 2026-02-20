/*
Database operations for Users.

This file contains all the functions that interact with the users table.
*/
package database

import (
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
)

// CreateUser creates a new user and returns their ID
// If the user already exists, returns their existing ID
func (db *appdbimpl) CreateUser(name string) (string, error) {
	// First, check if user already exists
	existingUser, err := db.GetUserByName(name)
	if err == nil && existingUser != nil {
		// User exists, return their ID (this is for login)
		return existingUser.ID, nil
	}

	// Generate a new unique ID
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	// Insert the new user
	_, err = db.db.Exec(
		"INSERT INTO users (id, name) VALUES (?, ?)",
		id.String(), name,
	)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

// GetUserByName finds a user by their username
func (db *appdbimpl) GetUserByName(name string) (*User, error) {
	var user User
	var photo sql.NullString

	err := db.db.QueryRow(
		"SELECT id, name, photo FROM users WHERE name = ?",
		name,
	).Scan(&user.ID, &user.Name, &photo)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	if photo.Valid {
		user.Photo = []byte(photo.String)
	}

	return &user, nil
}

// GetUserByID finds a user by their ID
func (db *appdbimpl) GetUserByID(id string) (*User, error) {
	var user User
	var photo sql.NullString

	err := db.db.QueryRow(
		"SELECT id, name, photo FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Name, &photo)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	if photo.Valid {
		user.Photo = []byte(photo.String)
	}

	return &user, nil
}

// UpdateUserName changes a user's username
// Returns error if the new name is already taken
func (db *appdbimpl) UpdateUserName(userID, newName string) error {
	// Check if name is already taken by another user
	existingUser, err := db.GetUserByName(newName)
	if err == nil && existingUser != nil && existingUser.ID != userID {
		return ErrUsernameTaken
	}

	// Update the username
	result, err := db.db.Exec(
		"UPDATE users SET name = ? WHERE id = ?",
		newName, userID,
	)
	if err != nil {
		return err
	}

	// Check if user was found
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateUserPhoto sets or updates a user's profile photo
func (db *appdbimpl) UpdateUserPhoto(userID string, photo []byte) error {
	result, err := db.db.Exec(
		"UPDATE users SET photo = ? WHERE id = ?",
		photo, userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// SearchUsers finds users matching a search query
// If query is empty, returns all users
func (db *appdbimpl) SearchUsers(query string) ([]User, error) {
	var rows *sql.Rows
	var err error

	if query == "" {
		// Return all users
		rows, err = db.db.Query("SELECT id, name, photo FROM users ORDER BY name")
	} else {
		// Search by partial name match
		rows, err = db.db.Query(
			"SELECT id, name, photo FROM users WHERE name LIKE ? ORDER BY name",
			"%"+query+"%",
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var photo sql.NullString

		if err := rows.Scan(&user.ID, &user.Name, &photo); err != nil {
			return nil, err
		}

		if photo.Valid {
			user.Photo = []byte(photo.String)
		}

		users = append(users, user)
	}

	return users, rows.Err()
}
