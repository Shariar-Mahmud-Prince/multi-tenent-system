package model

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents an authenticated account in the system.
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserResponse is the safe public representation of a user.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse formats user model for serialization.
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}

// CheckPassword compares a plaintext password against the stored bcrypt hash.
func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

// RegisterUserRequest contains input for creating an account.
type RegisterUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}

// Validate checks user registration input.
func (r *RegisterUserRequest) Validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	if r.Email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(r.Email); err != nil {
		return errors.New("invalid email address format")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if r.Role == "" {
		r.Role = "owner"
	} else if r.Role != "owner" && r.Role != "admin" && r.Role != "member" {
		return errors.New("role must be 'owner', 'admin', or 'member'")
	}
	return nil
}

// LoginRequest contains authentication credentials.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks login credentials.
func (r *LoginRequest) Validate() error {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

// AuthResponse returns token and user info.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
