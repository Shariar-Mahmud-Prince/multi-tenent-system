package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/handler"
	"multi-tenant-domains/internal/middleware"
	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
)

type mockUserRepository struct {
	users []*model.User
}

func (m *mockUserRepository) CreateUser(ctx context.Context, user *model.User) error {
	for _, u := range m.users {
		if strings.EqualFold(u.Email, user.Email) {
			return repository.ErrEmailAlreadyTaken
		}
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users = append(m.users, user)
	return nil
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func TestAuthHandler_RegisterAndLoginFlow(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "super-secret-jwt-test-key-32b!",
	}
	repo := &mockUserRepository{}
	authH := handler.NewAuthHandler(repo, cfg)

	// 1. Register new user
	regPayload := model.RegisterUserRequest{
		Email:    "merchant@saas.local",
		Password: "SecurePassword123!",
		Role:     "owner",
	}
	body, _ := json.Marshal(regPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	authH.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on register, got %d: %s", rr.Code, rr.Body.String())
	}

	var authResp model.AuthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &authResp); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}

	if authResp.Token == "" {
		t.Errorf("expected non-empty JWT token on register")
	}
	if authResp.User.Email != "merchant@saas.local" {
		t.Errorf("expected email %q, got %q", "merchant@saas.local", authResp.User.Email)
	}

	// 2. Prevent duplicate registration
	reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rrDup := httptest.NewRecorder()
	authH.Register(rrDup, reqDup)

	if rrDup.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for duplicate registration, got %d", rrDup.Code)
	}

	// 3. Successful Login
	loginPayload := model.LoginRequest{
		Email:    "merchant@saas.local",
		Password: "SecurePassword123!",
	}
	loginBody, _ := json.Marshal(loginPayload)
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	rrLogin := httptest.NewRecorder()
	authH.Login(rrLogin, reqLogin)

	if rrLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login, got %d: %s", rrLogin.Code, rrLogin.Body.String())
	}

	// 4. Failed Login - Wrong Password
	badLoginPayload := model.LoginRequest{
		Email:    "merchant@saas.local",
		Password: "WrongPassword!",
	}
	badBody, _ := json.Marshal(badLoginPayload)
	reqBadLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(badBody))
	rrBadLogin := httptest.NewRecorder()
	authH.Login(rrBadLogin, reqBadLogin)

	if rrBadLogin.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for bad password, got %d", rrBadLogin.Code)
	}
}

func TestAuthHandler_Me(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "super-secret-jwt-test-key-32b!",
	}
	userID := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	repo := &mockUserRepository{
		users: []*model.User{
			{
				ID:           userID,
				Email:        "profile@test.local",
				PasswordHash: string(hash),
				Role:         "owner",
			},
		},
	}
	authH := handler.NewAuthHandler(repo, cfg)

	// Authenticated request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	claims := &middleware.UserClaims{
		UserID: userID,
		Role:   "owner",
	}
	ctx := middleware.WithUserContext(req.Context(), claims)
	rr := httptest.NewRecorder()

	authH.Me(rr, req.WithContext(ctx))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /me, got %d: %s", rr.Code, rr.Body.String())
	}

	var userResp model.UserResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &userResp); err != nil {
		t.Fatalf("failed to decode user response: %v", err)
	}
	if userResp.ID != userID || userResp.Email != "profile@test.local" {
		t.Errorf("unexpected profile data: %+v", userResp)
	}
}
