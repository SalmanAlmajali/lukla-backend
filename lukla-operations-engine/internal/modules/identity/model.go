// Package identity manages user accounts, login, sessions, and account status.
// Business rules for production operations must not be placed here.
package identity

import "time"

// User represents an application user account.
type User struct {
	ID           string
	Email        string
	PasswordHash string // never expose in responses or logs
	FullName     string
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserStatus represents the lifecycle state of a user account.
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

// Session holds an active user session created after successful login.
type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	IPAddress string
	UserAgent string
}
