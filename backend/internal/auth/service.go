package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidCredentials is returned for both an unknown email and a wrong password.
// Using a single sentinel prevents callers from distinguishing the two cases,
// which would otherwise allow email enumeration.
var ErrInvalidCredentials = errors.New("invalid email or password")

// Service contains business logic for authentication.
// All password and token operations live here; the handler only parses HTTP.
type Service struct {
	repo UserRepository
	jwt  *JWTManager
}

// NewService creates a new auth Service with the given repository and JWT manager.
func NewService(repo UserRepository, jwt *JWTManager) *Service {
	return &Service{repo: repo, jwt: jwt}
}

// LoginRequest holds credentials submitted by the user on the login form.
type LoginRequest struct {
	Email    string
	Password string
}

// TokenPair holds the access and refresh tokens returned after a successful login.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Login validates credentials and returns a JWT token pair on success.
// Returns ErrInvalidCredentials for both an unknown email and a wrong password
// so callers cannot distinguish between the two cases (anti-enumeration).
func (s *Service) Login(ctx context.Context, req LoginRequest) (*TokenPair, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Run a dummy bcrypt comparison so that response time is constant
			// regardless of whether the email exists in the database.
			bcrypt.CompareHashAndPassword([]byte("$2a$10$dummyhashpadding000000000000000000000000000000000000000"), []byte(req.Password)) //nolint:errcheck
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("service.Login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("service.Login: %w", err)
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("service.Login: %w", err)
	}

	slog.Info("user logged in", "user_id", user.ID, "role", user.Role)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Refresh validates a refresh token and returns a new access token.
// The user's role is re-fetched from the database so that role changes take effect immediately
// rather than persisting until the old refresh token expires.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (string, error) {
	claims, err := s.jwt.ParseToken(refreshToken)
	if err != nil {
		return "", ErrInvalidCredentials
	}
	if claims.Type != "refresh" {
		return "", ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("service.Refresh: %w", err)
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return "", fmt.Errorf("service.Refresh: %w", err)
	}

	return accessToken, nil
}

// ValidateToken parses an access token and returns its claims.
// Returns ErrInvalidCredentials if the token is invalid, expired, or not an access token.
// Used by other services that need to inspect token claims (e.g. extracting user_id).
func (s *Service) ValidateToken(token string) (*Claims, error) {
	claims, err := s.jwt.ParseToken(token)
	if err != nil {
		return nil, fmt.Errorf("service.ValidateToken: %w", err)
	}
	if claims.Type != "access" {
		return nil, fmt.Errorf("service.ValidateToken: %w", ErrInvalidCredentials)
	}
	return claims, nil
}
