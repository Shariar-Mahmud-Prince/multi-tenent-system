package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type userContextKey string

const (
	authUserKey userContextKey = "authenticated_user"
)

// UserClaims defines JWT token payload.
type UserClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a signed token for an authenticated user.
func GenerateJWT(userID uuid.UUID, role string, secret string, duration time.Duration) (string, error) {
	claims := UserClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "saas-multi-tenant-platform",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signed, nil
}

// ParseJWT validates the JWT string and returns claims.
func ParseJWT(tokenString string, secret string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// AuthMiddleware protects routes requiring valid user credentials.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "Authorization header is required (Bearer <token>)",
				})
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid Authorization header format. Must be: Bearer <token>",
				})
				return
			}

			claims, err := ParseJWT(parts[1], secret)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "Invalid or expired authentication token",
					"details": err.Error(),
				})
				return
			}

			// Invalidate token if user ID is empty
			if claims.UserID == uuid.Nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid token subject",
				})
				return
			}

			ctx := context.WithValue(r.Context(), authUserKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WithUserContext injects claims into a context (useful for testing or system invocation).
func WithUserContext(ctx context.Context, claims *UserClaims) context.Context {
	return context.WithValue(ctx, authUserKey, claims)
}

// RequireRole ensures the authenticated user possesses at least one of the specified roles.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := UserFromContext(r.Context())
			if !ok || claims == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
				return
			}

			allowed := false
			for _, role := range allowedRoles {
				if strings.EqualFold(claims.Role, role) {
					allowed = true
					break
				}
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden: insufficient role permissions"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UserFromContext extracts the authenticated user claims from context.
func UserFromContext(ctx context.Context) (*UserClaims, bool) {
	val := ctx.Value(authUserKey)
	if val == nil {
		return nil, false
	}
	claims, ok := val.(*UserClaims)
	return claims, ok
}
