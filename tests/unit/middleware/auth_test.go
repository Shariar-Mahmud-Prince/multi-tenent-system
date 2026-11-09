package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"multi-tenant-domains/internal/middleware"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!!"
	userID := uuid.New()
	role := "owner"

	token, err := middleware.GenerateJWT(userID, role, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	var capturedClaims *middleware.UserClaims
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.UserFromContext(r.Context())
		if !ok || claims == nil {
			t.Errorf("failed to extract claims from context")
		}
		capturedClaims = claims
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.AuthMiddleware(secret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	mw(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
	if capturedClaims == nil || capturedClaims.UserID != userID || capturedClaims.Role != role {
		t.Errorf("unexpected claims: %+v", capturedClaims)
	}
}

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!!"
	mw := middleware.AuthMiddleware(secret)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	rr := httptest.NewRecorder()

	mw(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing auth header, got %d", rr.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!!"
	userID := uuid.New()

	// Generate expired token (-1 hour)
	token, err := middleware.GenerateJWT(userID, "owner", secret, -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	mw := middleware.AuthMiddleware(secret)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	mw(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for expired token, got %d", rr.Code)
	}
}

func TestAuthMiddleware_InvalidSignature(t *testing.T) {
	secretA := "secret-key-A-----------------32b"
	secretB := "secret-key-B-----------------32b"
	userID := uuid.New()

	// Signed with secretA
	token, err := middleware.GenerateJWT(userID, "owner", secretA, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Middleware verifies with secretB
	mw := middleware.AuthMiddleware(secretB)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	mw(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid signature, got %d", rr.Code)
	}
}

func TestRequireRole(t *testing.T) {
	adminClaims := &middleware.UserClaims{
		UserID: uuid.New(),
		Role:   "admin",
	}

	ownerClaims := &middleware.UserClaims{
		UserID: uuid.New(),
		Role:   "owner",
	}

	mw := middleware.RequireRole("admin")
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test with admin role -> should pass
	reqAdmin := httptest.NewRequest(http.MethodGet, "/admin", nil)
	ctxAdmin := middleware.WithUserContext(reqAdmin.Context(), adminClaims)
	rrAdmin := httptest.NewRecorder()
	mw(dummyHandler).ServeHTTP(rrAdmin, reqAdmin.WithContext(ctxAdmin))

	if rrAdmin.Code != http.StatusOK {
		t.Errorf("expected 200 OK for admin role, got %d", rrAdmin.Code)
	}

	// Test with owner role -> should receive 403 Forbidden
	reqOwner := httptest.NewRequest(http.MethodGet, "/admin", nil)
	ctxOwner := middleware.WithUserContext(reqOwner.Context(), ownerClaims)
	rrOwner := httptest.NewRecorder()
	mw(dummyHandler).ServeHTTP(rrOwner, reqOwner.WithContext(ctxOwner))

	if rrOwner.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for non-admin role, got %d", rrOwner.Code)
	}
}
