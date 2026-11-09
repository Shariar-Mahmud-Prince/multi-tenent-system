package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"multi-tenant-domains/internal/config"
	"multi-tenant-domains/internal/middleware"
	"multi-tenant-domains/internal/model"
	"multi-tenant-domains/internal/repository"
)

// AuthHandler manages registration, login, and profile lookup.
type AuthHandler struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(userRepo repository.UserRepository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		respondJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
		return
	}

	user := model.User{
		Email:        req.Email,
		PasswordHash: string(hashed),
		Role:         req.Role,
	}

	if err := h.userRepo.CreateUser(r.Context(), &user); err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyTaken) {
			respondJSON(w, http.StatusConflict, map[string]string{"error": "An account with this email already exists"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create account: " + err.Error()})
		return
	}

	// Generate JWT
	token, err := middleware.GenerateJWT(user.ID, user.Role, h.cfg.JWTSecret, 7*24*time.Hour)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
		return
	}

	respondJSON(w, http.StatusCreated, model.AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		respondJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	user, err := h.userRepo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		// Generic invalid credentials to prevent email enumeration
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid email or password"})
		return
	}

	token, err := middleware.GenerateJWT(user.ID, user.Role, h.cfg.JWTSecret, 7*24*time.Hour)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
		return
	}

	respondJSON(w, http.StatusOK, model.AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}

// Me handles GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.UserFromContext(r.Context())
	if !ok || claims == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized context"})
		return
	}

	user, err := h.userRepo.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "User account not found"})
		return
	}

	respondJSON(w, http.StatusOK, user.ToResponse())
}
