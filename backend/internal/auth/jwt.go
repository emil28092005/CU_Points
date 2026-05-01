package auth

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims are the JWT payload fields used by this service.
// Both access and refresh tokens use this struct; the Type field distinguishes them.
// Every token includes a unique JWTID (jti) for future revocation support.
type Claims struct {
	jwt.RegisteredClaims        // carries sub (user_id), exp, iat, jti
	Role                 string `json:"role,omitempty"` // populated only in access tokens
	Type                 string `json:"type"`           // "access" or "refresh"
}

// JWTManager generates and validates JWT tokens.
type JWTManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewJWTManager creates a JWTManager with the given HMAC secret and TTL durations.
func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// GenerateAccessToken creates a signed HS256 access token for the given user.
// Claims include: sub (user_id), role, jti (unique ID), iat, exp.
func (m *JWTManager) GenerateAccessToken(userID, role string) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", fmt.Errorf("jwt.GenerateAccessToken: generate jti: %w", err)
	}

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTTL)),
		},
		Role: role,
		Type: "access",
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("jwt.GenerateAccessToken: sign: %w", err)
	}
	return signed, nil
}

// GenerateRefreshToken creates a signed HS256 refresh token for the given user.
// Claims include: sub (user_id), jti (unique ID), iat, exp.
// The role is intentionally omitted — it is always re-fetched from the DB on use.
func (m *JWTManager) GenerateRefreshToken(userID string) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", fmt.Errorf("jwt.GenerateRefreshToken: generate jti: %w", err)
	}

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshTTL)),
		},
		Type: "refresh",
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("jwt.GenerateRefreshToken: sign: %w", err)
	}
	return signed, nil
}

// ParseToken parses and cryptographically validates a JWT string.
// It verifies the HMAC signature and token expiry but does NOT check the Type field —
// callers are responsible for asserting the expected type ("access" or "refresh").
func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt.ParseToken: unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt.ParseToken: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt.ParseToken: invalid token")
	}
	return claims, nil
}

// newJTI generates a cryptographically random UUID v4 string for use as a JWT ID.
func newJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Set version 4 and variant bits per RFC 4122
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
