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

// newTestJWT returns a JWTManager configured with the test secret.
func newTestJWT() *auth.JWTManager {
	return auth.NewJWTManager(
		"test-secret-minimum-32-characters-long",
		15*time.Minute,
		168*time.Hour,
	)
}

// ─── Login ────────────────────────────────────────────────────────────────────

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

func TestService_Login_RepoError(t *testing.T) {
	repo := &mockRepo{repoErr: errors.New("db error")}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), auth.LoginRequest{
		Email:    "user@cu.ru",
		Password: "pass",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Must NOT be ErrInvalidCredentials — we don't want to mask infra errors.
	if errors.Is(err, auth.ErrInvalidCredentials) {
		t.Error("unexpected ErrInvalidCredentials for non-ErrNotFound repo error")
	}
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestService_Refresh_Success(t *testing.T) {
	jwtMgr := newTestJWT()
	user := &auth.UserRecord{
		ID:   "user-1",
		Role: "student",
	}
	repo := &mockRepo{user: user}
	svc := auth.NewService(repo, jwtMgr)

	// Generate a real refresh token via the JWT manager.
	refreshToken, err := jwtMgr.GenerateRefreshToken(user.ID)
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	accessToken, err := svc.Refresh(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if accessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestService_Refresh_InvalidToken(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Refresh(context.Background(), "not.a.valid.token")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestService_Refresh_WrongTokenType(t *testing.T) {
	jwtMgr := newTestJWT()
	svc := auth.NewService(&mockRepo{}, jwtMgr)

	// Use an access token where a refresh token is expected.
	accessToken, _ := jwtMgr.GenerateAccessToken("user-1", "student")

	_, err := svc.Refresh(context.Background(), accessToken)
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestService_Refresh_UserNotFound(t *testing.T) {
	jwtMgr := newTestJWT()
	repo := &mockRepo{repoErr: auth.ErrNotFound}
	svc := auth.NewService(repo, jwtMgr)

	refreshToken, _ := jwtMgr.GenerateRefreshToken("deleted-user")

	_, err := svc.Refresh(context.Background(), refreshToken)
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

// ─── ValidateToken ────────────────────────────────────────────────────────────

func TestService_ValidateToken_Success(t *testing.T) {
	jwtMgr := newTestJWT()
	svc := auth.NewService(&mockRepo{}, jwtMgr)

	accessToken, _ := jwtMgr.GenerateAccessToken("user-1", "student")

	claims, err := svc.ValidateToken(accessToken)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Errorf("expected subject=user-1, got %s", claims.Subject)
	}
}

func TestService_ValidateToken_InvalidToken(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.ValidateToken("garbage.token.value")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_ValidateToken_RefreshTokenRejected(t *testing.T) {
	jwtMgr := newTestJWT()
	svc := auth.NewService(&mockRepo{}, jwtMgr)

	refreshToken, _ := jwtMgr.GenerateRefreshToken("user-1")

	_, err := svc.ValidateToken(refreshToken)
	if err == nil {
		t.Fatal("expected error for refresh token passed to ValidateToken")
	}
}

// ─── JWT round-trip ───────────────────────────────────────────────────────────

func TestJWTManager_AccessToken_RoundTrip(t *testing.T) {
	mgr := newTestJWT()

	token, err := mgr.GenerateAccessToken("user-42", "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := mgr.ParseToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Subject != "user-42" {
		t.Errorf("subject: want user-42, got %s", claims.Subject)
	}
	if claims.Role != "admin" {
		t.Errorf("role: want admin, got %s", claims.Role)
	}
	if claims.Type != "access" {
		t.Errorf("type: want access, got %s", claims.Type)
	}
}

func TestJWTManager_RefreshToken_RoundTrip(t *testing.T) {
	mgr := newTestJWT()

	token, err := mgr.GenerateRefreshToken("user-7")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := mgr.ParseToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Subject != "user-7" {
		t.Errorf("subject: want user-7, got %s", claims.Subject)
	}
	if claims.Type != "refresh" {
		t.Errorf("type: want refresh, got %s", claims.Type)
	}
}

func TestJWTManager_ParseToken_Invalid(t *testing.T) {
	mgr := newTestJWT()

	_, err := mgr.ParseToken("not.a.valid.jwt")
	if err == nil {
		t.Fatal("expected error for invalid JWT, got nil")
	}
}

func TestJWTManager_ParseToken_WrongSecret(t *testing.T) {
	mgr1 := newTestJWT()
	mgr2 := auth.NewJWTManager("other-secret-that-is-at-least-32-chars-long", 15*time.Minute, 168*time.Hour)

	token, _ := mgr1.GenerateAccessToken("user-1", "student")

	_, err := mgr2.ParseToken(token)
	if err == nil {
		t.Fatal("expected error when parsing with wrong secret")
	}
}
