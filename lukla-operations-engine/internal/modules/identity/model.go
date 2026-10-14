// Package identity manages user accounts, login, sessions, and account status.
// Business rules for production operations must not be placed here.
package identity

import "time"

// User represents an application user account.
type User struct {
	ID           string     `json:"id"`
	EmployeeCode *string    `json:"employee_code,omitempty"`
	FullName     string     `json:"full_name"`
	Email        *string    `json:"email,omitempty"`
	PhoneNumber  *string    `json:"phone_number,omitempty"`
	PasswordHash *string    `json:"-"` // never expose in JSON responses
	RoleID       int64      `json:"role_id"`
	RoleCode     string     `json:"role_code"`
	RoleName     string     `json:"role_name"`
	ClientID     *string    `json:"client_id,omitempty"`
	CanAccessApp bool       `json:"can_access_app"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// LoginRequest represents incoming login credentials.
// Sesuai catatan: field username harus berupa email valid.
type LoginRequest struct {
	Username string `json:"username" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginData represents the payload returned in standard data envelope.
type LoginData struct {
	Token string `json:"token"`
}

// UserResponse represents the user profile returned by GET /auth/me.
type UserResponse struct {
	ID              string  `json:"id"`
	Username        string  `json:"username"`
	Name            string  `json:"name"`
	Email           string  `json:"email"`
	Role            string  `json:"role"`
	ClientID        *string `json:"clientId"`
	RoleDescription string  `json:"roleDescription"`
}
