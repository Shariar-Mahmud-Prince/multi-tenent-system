package authentication_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"multi-tenant-domains/internal/middleware"
)

func TestJWTTampering_AlgNoneAttack(t *testing.T) {
	secret := "secret-jwt-key-minimum-32-bytes!"
	mw := middleware.AuthMiddleware(secret)

	// Craft an unverified token with alg: none
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"` + uuid.New().String() + `","role":"admin"}`))
	tamperedToken := header + "." + claims + "."

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tamperedToken)
	rr := httptest.NewRecorder()

	mw(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("CRITICAL SECURITY VULNERABILITY! alg:none attack succeeded with status %d", rr.Code)
	}
}

func TestJWTTampering_PayloadModification(t *testing.T) {
	secret := "secret-jwt-key-minimum-32-bytes!"
	userID := uuid.New()

	// Generate legitimate token with role "user"
	validToken, err := middleware.GenerateJWT(userID, "user", secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Split into header.payload.signature
	parts := strings.Split(validToken, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid token format")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	var claimsMap map[string]interface{}
	_ = json.Unmarshal(payloadBytes, &claimsMap)
	// Tamper role to "admin"
	claimsMap["role"] = "admin"
	tamperedPayloadBytes, _ := json.Marshal(claimsMap)
	tamperedPayload := base64.RawURLEncoding.EncodeToString(tamperedPayloadBytes)

	// Reassemble with original signature
	tamperedToken := parts[0] + "." + tamperedPayload + "." + parts[2]

	mw := middleware.AuthMiddleware(secret)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tamperedToken)
	rr := httptest.NewRecorder()

	mw(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("CRITICAL SECURITY VULNERABILITY! Tampered payload was accepted with status %d", rr.Code)
	}
}
