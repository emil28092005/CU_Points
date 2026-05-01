// Package middleware provides HTTP middleware: JWT verification, role guard, request logging.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is an unexported type for context keys in this package,
// preventing collisions with keys set by other packages.
type contextKey string

const (
	userIDKey   contextKey = "user_id"
	userRoleKey contextKey = "user_role"
	jtiKey      contextKey = "jti"
)

// tokenClaims mirrors the JWT payload fields the middleware needs to inspect.
// Defined locally so the middleware does not import the auth package.
type tokenClaims struct {
	jwt.RegisteredClaims        // provides Subject (user_id), JWTID (jti), ExpiresAt
	Role                 string `json:"role"`
	Type                 string `json:"type"`
}

// UserIDFromContext retrieves the authenticated user's ID stored by Auth middleware.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// ContextWithUserID returns a copy of ctx carrying the given userID.
// Intended only for handler unit tests that bypass Auth middleware.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserRoleFromContext retrieves the authenticated user's role stored by Auth middleware.
func UserRoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userRoleKey).(string)
	return v
}

// JTIFromContext retrieves the JWT ID (jti) stored by Auth middleware.
// Useful for token revocation checks in downstream handlers.
func JTIFromContext(ctx context.Context) string {
	v, _ := ctx.Value(jtiKey).(string)
	return v
}

// Auth returns middleware that validates the Bearer JWT in the Authorization header.
// On success it injects user_id, role, and jti into the request context.
// Rejects tokens that are expired, have a bad signature, or are not of type "access"
// (prevents refresh tokens from being used on protected endpoints).
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := bearerToken(r)
			if err != nil {
				http.Error(w, `{"error":"missing or invalid Authorization header"}`, http.StatusUnauthorized)
				return
			}

			claims := &tokenClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("middleware.Auth: unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			// Explicitly block refresh tokens from reaching protected endpoints.
			if claims.Type != "access" {
				http.Error(w, `{"error":"access token required"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.Subject)
			ctx = context.WithValue(ctx, userRoleKey, claims.Role)
			ctx = context.WithValue(ctx, jtiKey, claims.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerToken extracts the token string from the Authorization: Bearer <token> header.
func bearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return "", fmt.Errorf("middleware.bearerToken: missing Bearer prefix")
	}
	return strings.TrimPrefix(h, "Bearer "), nil
}
