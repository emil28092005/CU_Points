package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/cu-points/backend/internal/auth"
)

// mockRepo is a test double for UserRepository.
// Populate user and/or repoErr before each test case.
type mockRepo struct {
	user    *auth.UserRecord
	repoErr error
}

func (m *mockRepo) GetUserByEmail(_ context.Context, _ string) (*auth.UserRecord, error) {
	return m.user, m.repoErr
}

func (m *mockRepo) GetUserByID(_ context.Context, _ string) (*auth.UserRecord, error) {
	return m.user, m.repoErr
}

// hashPassword hashes the given plain-text password using bcrypt minimum cost for speed.
func hashPassword(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	return string(h)
}

// newTestService builds a Service wired to the given mock repo.
func newTestService(repo auth.UserRepository) *auth.Service {
	jwtMgr := auth.NewJWTManager(
		"test-secret-minimum-32-characters-long",
		15*time.Minute,
		168*time.Hour,
	)
	return auth.NewService(repo, jwtMgr)
}

func TestService_Login_Success(t *testing.T) {
	repo := &mockRepo{
		user: &auth.UserRecord{
			ID:           "a5b66288-4a97-410b-9e30-a7cf61cdabab",
			Email:        "student@cu.ru",
			PasswordHash: hashPassword(t, "password123"),
			Role:         "student",
		},
	}
	svc := newTestService(repo)

	pair, err := svc.Login(context.Background(), auth.LoginRequest{
		Email:    "student@cu.ru",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	repo := &mockRepo{
		user: &auth.UserRecord{
			ID:           "a5b66288-4a97-410b-9e30-a7cf61cdabab",
			Email:        "student@cu.ru",
			PasswordHash: hashPassword(t, "password123"),
			Role:         "student",
		},
	}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), auth.LoginRequest{
		Email:    "student@cu.ru",
		Password: "wrongpassword",
	})

	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestService_Login_UserNotFound(t *testing.T) {
	repo := &mockRepo{repoErr: auth.ErrNotFound}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), auth.LoginRequest{
		Email:    "nobody@cu.ru",
		Password: "password123",
	})

	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}
