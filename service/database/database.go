// Package database handles all data storage for WASAText.
package database

import (
	"database/sql"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Message status constants.
const (
	StatusSent     = "sent"
	StatusReceived = "received"
	StatusRead     = "read"
)

// AppDatabase is the interface for all database operations.
type AppDatabase interface {
	// User operations
	CreateUser(name string) (string, error)
	GetUserByName(name string) (*User, error)
	GetUserByID(id string) (*User, error)
	UpdateUserName(userID, newName string) error
	UpdateUserPhoto(userID string, photo []byte) error
	GetUserPhoto(userID string) ([]byte, error)
	SearchUsers(query string) ([]User, error)

	// Conversation operations
	GetConversations(userID string) ([]ConversationPreview, error)
	GetConversation(userID, conversationID string) (*Conversation, error)
	GetOrCreateDirectConversation(userID, otherUserID string) (string, error)

	// Message operations
	CreateMessage(conversationID, senderID, content string, photo []byte, replyTo *string, isForwarded bool) (*Message, error)
	GetMessage(messageID string) (*Message, error)
	DeleteMessage(messageID, userID string) error
	UpdateMessageStatus(messageID, status string) error
	MarkConversationAsRead(conversationID, userID string) error

	// Comment (reaction) operations
	AddComment(messageID, userID, emoticon string) error
	RemoveComment(messageID, userID string) error

	// Group operations
	CreateGroup(name string, creatorID string, memberIDs []string) (*Group, error)
	GetGroup(groupID string) (*Group, error)
	GetGroupByConversationID(conversationID string) (*Group, error)
	AddUserToGroup(groupID, userID, adderID string) error
	RemoveUserFromGroup(groupID, userID string) error
	UpdateGroupName(groupID, name string) error
	UpdateGroupPhoto(groupID string, photo []byte) error
	GetGroupPhoto(groupID string) ([]byte, error)
	IsGroupMember(groupID, userID string) (bool, error)

	// Cleanup
	Close() error
}

// User represents a WASAText user.
type User struct {
	ID    string
	Name  string
	Photo []byte
}

// Group represents a WASAText group.
type Group struct {
	ID             string
	Name           string
	Photo          []byte
	Members        []User
	ConversationID string
}

// Message represents a message in a conversation.
type Message struct {
	ID              string
	SenderID        string
	SenderName      string
	Content         string
	Photo           []byte
	Timestamp       time.Time
	Status          string
	ReplyTo         *string
	ReplyToContent  string
	ReplyToHasPhoto bool
	IsForwarded     bool
	Comments        []Comment
}

// Comment represents a reaction on a message.
type Comment struct {
	UserID   string
	UserName string
	Emoticon string
}

// ConversationPreview is used for the conversation list.
type ConversationPreview struct {
	ID                 string
	IsGroup            bool
	Name               string
	Photo              []byte
	LastMessageTime    time.Time
	LastMessagePreview string
	LastMessageIsPhoto bool
}

// Conversation contains full conversation details with messages.
type Conversation struct {
	ID       string
	IsGroup  bool
	Name     string
	Photo    []byte
	GroupID  string
	Members  []User
	Messages []Message
}

// appdbimpl implements the AppDatabase interface.
type appdbimpl struct {
	db *sql.DB
}

// New creates a new database connection and initializes tables.
func New(filepath string) (AppDatabase, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := createTables(db); err != nil {
		return nil, err
	}

	return &appdbimpl{db: db}, nil
}

// createTables sets up all the database tables.
func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			photo BLOB
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			photo BLOB
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS group_members (
			group_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			PRIMARY KEY (group_id, user_id),
			FOREIGN KEY (group_id) REFERENCES groups(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			is_group BOOLEAN NOT NULL DEFAULT 0,
			group_id TEXT,
			FOREIGN KEY (group_id) REFERENCES groups(id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_participants (
			conversation_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			last_read_time DATETIME,
			PRIMARY KEY (conversation_id, user_id),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			content TEXT,
			photo BLOB,
			timestamp DATETIME NOT NULL,
			status TEXT NOT NULL DEFAULT 'sent',
			reply_to TEXT,
			is_forwarded BOOLEAN NOT NULL DEFAULT 0,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id),
			FOREIGN KEY (sender_id) REFERENCES users(id),
			FOREIGN KEY (reply_to) REFERENCES messages(id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			emoticon TEXT NOT NULL,
			PRIMARY KEY (message_id, user_id),
			FOREIGN KEY (message_id) REFERENCES messages(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return err
	}

	// Migration: add is_forwarded column if missing
	_, _ = db.Exec("ALTER TABLE messages ADD COLUMN is_forwarded BOOLEAN NOT NULL DEFAULT 0")

	return nil
}

// Close closes the database connection.
func (db *appdbimpl) Close() error {
	return db.db.Close()
}

// Common errors.
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUsernameTaken        = errors.New("username already taken")
	ErrGroupNotFound        = errors.New("group not found")
	ErrNotGroupMember       = errors.New("not a member of this group")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrNotMessageOwner      = errors.New("cannot delete messages sent by others")
	ErrCommentNotFound      = errors.New("comment not found")
)
