package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	pbAuth "github.com/karimiku/smart-stay-platform/pkg/genproto/auth"
)

// contextKey is a type-safe context key
type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RoleKey   contextKey = "role"
)

// AuthMiddleware validates JWT tokens and sets user information in the context
type AuthMiddleware struct {
	authClient pbAuth.AuthServiceClient
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authClient pbAuth.AuthServiceClient) *AuthMiddleware {
	return &AuthMiddleware{
		authClient: authClient,
	}
}

// RequireAuth is a middleware that requires authentication
// Returns 401 Unauthorized if the token is invalid
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract token from Authorization header
		token := extractBearerToken(r)
		if token == "" {
			respondUnauthorized(w, "Missing or invalid authorization header")
			return
		}

		// 2. Validate token with Auth Service
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		validateRes, err := m.authClient.Validate(ctx, &pbAuth.ValidateRequest{
			AccessToken: token,
		})
		if err != nil {
			log.Printf("[Auth] Token validation error: %v", err)
			respondUnauthorized(w, "Invalid token")
			return
		}

		if !validateRes.Valid {
			respondUnauthorized(w, "Invalid or expired token")
			return
		}

		// 3. Set user information in context
		ctx = context.WithValue(r.Context(), UserIDKey, validateRes.UserId)
		ctx = context.WithValue(ctx, RoleKey, validateRes.Role)

		// 4. Execute next handler
		next(w, r.WithContext(ctx))
	}
}

// OptionalAuth is a middleware that optionally validates authentication
// If a token is present, it validates it; otherwise, it passes through
func (m *AuthMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			validateRes, err := m.authClient.Validate(ctx, &pbAuth.ValidateRequest{
				AccessToken: token,
			})
			if err == nil && validateRes.Valid {
				ctx = context.WithValue(r.Context(), UserIDKey, validateRes.UserId)
				ctx = context.WithValue(ctx, RoleKey, validateRes.Role)
				r = r.WithContext(ctx)
			}
		}

		next(w, r)
	}
}

// RequireRole is a middleware that requires specific roles
func (m *AuthMiddleware) RequireRole(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetRole(r)
			if !ok {
				respondForbidden(w, "Role information not found")
				return
			}

			// Check if role is allowed
			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					next(w, r)
					return
				}
			}

			respondForbidden(w, "Insufficient permissions")
		})
	}
}

// extractBearerToken extracts Bearer token from Authorization header or Cookie
// Security: Strictly validates token format
func extractBearerToken(r *http.Request) string {
	// First, try to get token from Authorization header (for API clients)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
	const bearerPrefix = "Bearer "
		if strings.HasPrefix(authHeader, bearerPrefix) {
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
			if token != "" {
				return token
			}
		}
	}

	// Fallback to Cookie (for browser clients)
	cookie, err := r.Cookie("auth_token")
	if err == nil && cookie != nil && cookie.Value != "" {
		return cookie.Value
	}

		return ""
}

// respondUnauthorized returns 401 Unauthorized response
// Security: Does not return detailed error information (prevents information leakage)
func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}

// respondForbidden returns 403 Forbidden response
func respondForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}

// GetUserID retrieves user_id from context
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	return userID, ok
}

// GetRole retrieves role from context
func GetRole(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(RoleKey).(string)
	return role, ok
}

