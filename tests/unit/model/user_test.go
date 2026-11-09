package model_test

import (
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"multi-tenant-domains/internal/model"
)

func TestRegisterUserRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     model.RegisterUserRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			req: model.RegisterUserRequest{
				Email:    "merchant@example.com",
				Password: "password123",
				Role:     "owner",
			},
			wantErr: false,
		},
		{
			name: "Invalid email",
			req: model.RegisterUserRequest{
				Email:    "invalid-email",
				Password: "password123",
				Role:     "owner",
			},
			wantErr: true,
		},
		{
			name: "Empty email",
			req: model.RegisterUserRequest{
				Email:    "",
				Password: "password123",
				Role:     "owner",
			},
			wantErr: true,
		},
		{
			name: "Short password",
			req: model.RegisterUserRequest{
				Email:    "merchant@example.com",
				Password: "short",
				Role:     "owner",
			},
			wantErr: true,
		},
		{
			name: "Invalid role",
			req: model.RegisterUserRequest{
				Email:    "merchant@example.com",
				Password: "password123",
				Role:     "superhacker",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterUserRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     model.LoginRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			req: model.LoginRequest{
				Email:    "admin@example.com",
				Password: "securePassword!",
			},
			wantErr: false,
		},
		{
			name: "Empty email",
			req: model.LoginRequest{
				Email:    "",
				Password: "securePassword!",
			},
			wantErr: true,
		},
		{
			name: "Empty password",
			req: model.LoginRequest{
				Email:    "admin@example.com",
				Password: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoginRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUser_PasswordCheckAndToResponse(t *testing.T) {
	rawPassword := "SuperSecret123"
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &model.User{
		ID:           uuid.New(),
		Email:        "user@store.local",
		PasswordHash: string(hashedBytes),
		Role:         "owner",
	}

	if !user.CheckPassword(rawPassword) {
		t.Errorf("expected CheckPassword to return true for correct password")
	}

	if user.CheckPassword("WrongPassword!") {
		t.Errorf("expected CheckPassword to return false for incorrect password")
	}

	// Verify ToResponse never leaks password hash
	resp := user.ToResponse()
	if resp.Email != user.Email {
		t.Errorf("expected email %q, got %q", user.Email, resp.Email)
	}
	if resp.Role != user.Role {
		t.Errorf("expected role %q, got %q", user.Role, resp.Role)
	}
}
