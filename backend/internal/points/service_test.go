package points_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cu-points/backend/internal/points"
)

// ─── mocks ───────────────────────────────────────────────────────────────────

type mockRepo struct {
	balance    int
	balanceErr error
	earnErr    error
	spendErr   error
	newBalance int
}

func (m *mockRepo) GetBalance(_ context.Context, _ string) (int, error) {
	return m.balance, m.balanceErr
}
func (m *mockRepo) EarnAtomic(_ context.Context, _ string, _ int, _, _ string) error {
	return m.earnErr
}
func (m *mockRepo) SpendAtomic(_ context.Context, _, _ string, _ int) (int, error) {
	return m.newBalance, m.spendErr
}

type mockCache struct {
	used      bool
	isErr     error
	markErr   error
	markedID  string
}

func (m *mockCache) IsQRUsed(_ context.Context, _ string) (bool, error) {
	return m.used, m.isErr
}
func (m *mockCache) MarkQRUsed(_ context.Context, jti string) error {
	m.markedID = jti
	return m.markErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

const testSecret = "test-secret-minimum-32-characters-long"

func newService(repo points.Repository, cache points.CacheClient) *points.Service {
	return points.NewService(repo, cache, testSecret)
}

// makeExpiredToken creates a QR JWT that is already past its expiry.
func makeExpiredToken(secret string, userID string) string {
	type qrClaims struct {
		jwt.RegisteredClaims
		Type string `json:"type"`
	}
	claims := qrClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        "expired-jti",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-10 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Second)),
		},
		Type: "qr",
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return tok
}

// makeWrongTypeToken creates a valid JWT but with type != "qr".
func makeWrongTypeToken(secret string, userID string) string {
	type qrClaims struct {
		jwt.RegisteredClaims
		Type string `json:"type"`
	}
	claims := qrClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        "wrong-type-jti",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
		Type: "access", // wrong
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return tok
}

// ─── EarnPoints tests ─────────────────────────────────────────────────────────

func TestEarnPoints_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := newService(repo, &mockCache{})

	err := svc.EarnPoints(context.Background(), points.EarnRequest{
		UserID:      "user-1",
		Amount:      100,
		Type:        "earn",
		Description: "test",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestEarnPoints_ZeroAmount(t *testing.T) {
	svc := newService(&mockRepo{}, &mockCache{})

	err := svc.EarnPoints(context.Background(), points.EarnRequest{
		UserID:  "user-1",
		Amount:  0,
		Type:    "earn",
	})
	if err == nil {
		t.Fatal("expected error for zero amount, got nil")
	}
}

func TestEarnPoints_NegativeAmount(t *testing.T) {
	svc := newService(&mockRepo{}, &mockCache{})

	err := svc.EarnPoints(context.Background(), points.EarnRequest{
		UserID:  "user-1",
		Amount:  -50,
		Type:    "earn",
	})
	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
}

func TestEarnPoints_RepoError(t *testing.T) {
	repoErr := errors.New("db is down")
	repo := &mockRepo{earnErr: repoErr}
	svc := newService(repo, &mockCache{})

	err := svc.EarnPoints(context.Background(), points.EarnRequest{
		UserID:  "user-1",
		Amount:  100,
		Type:    "earn",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── GenerateQRToken tests ────────────────────────────────────────────────────

func TestGenerateQRToken_ReturnsToken(t *testing.T) {
	svc := newService(&mockRepo{}, &mockCache{})

	token, err := svc.GenerateQRToken(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestGenerateQRToken_DifferentEachCall(t *testing.T) {
	svc := newService(&mockRepo{}, &mockCache{})

	t1, _ := svc.GenerateQRToken(context.Background(), "user-1")
	t2, _ := svc.GenerateQRToken(context.Background(), "user-1")
	if t1 == t2 {
		t.Error("expected different tokens on successive calls (distinct jti)")
	}
}

// ─── SpendPoints tests ────────────────────────────────────────────────────────

func TestSpendPoints_Success(t *testing.T) {
	svc := newService(
		&mockRepo{balance: 500, newBalance: 400},
		&mockCache{},
	)

	token, err := svc.GenerateQRToken(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	newBalance, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if newBalance != 400 {
		t.Errorf("expected newBalance=400, got %d", newBalance)
	}
}

func TestSpendPoints_InsufficientBalance(t *testing.T) {
	svc := newService(
		&mockRepo{balance: 50},
		&mockCache{},
	)

	token, _ := svc.GenerateQRToken(context.Background(), "user-1")

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if !errors.Is(err, points.ErrInsufficientBalance) {
		t.Errorf("expected ErrInsufficientBalance, got: %v", err)
	}
}

func TestSpendPoints_QRAlreadyUsed(t *testing.T) {
	svc := newService(
		&mockRepo{balance: 500},
		&mockCache{used: true},
	)

	token, _ := svc.GenerateQRToken(context.Background(), "user-1")

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if !errors.Is(err, points.ErrQRAlreadyUsed) {
		t.Errorf("expected ErrQRAlreadyUsed, got: %v", err)
	}
}

func TestSpendPoints_InvalidToken(t *testing.T) {
	svc := newService(&mockRepo{balance: 500}, &mockCache{})

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   "not.a.valid.jwt",
		Amount:    100,
		PartnerID: "partner-1",
	})
	if !errors.Is(err, points.ErrInvalidQRToken) {
		t.Errorf("expected ErrInvalidQRToken, got: %v", err)
	}
}

func TestSpendPoints_ExpiredToken(t *testing.T) {
	svc := newService(&mockRepo{balance: 500}, &mockCache{})

	expiredToken := makeExpiredToken(testSecret, "user-1")

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   expiredToken,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if !errors.Is(err, points.ErrInvalidQRToken) {
		t.Errorf("expected ErrInvalidQRToken for expired token, got: %v", err)
	}
}

func TestSpendPoints_WrongSigningMethod(t *testing.T) {
	// Build a QR token signed with RS256 to trigger the "unexpected signing method" check.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	type qrClaims struct {
		jwt.RegisteredClaims
		Type string `json:"type"`
	}
	claims := qrClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ID:        "jti-1",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
		Type: "qr",
	}
	rs256Token, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(rsaKey)

	svc := newService(&mockRepo{balance: 500}, &mockCache{})

	_, err = svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   rs256Token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if !errors.Is(err, points.ErrInvalidQRToken) {
		t.Errorf("expected ErrInvalidQRToken for RS256-signed token, got: %v", err)
	}
}

func TestSpendPoints_WrongTokenType(t *testing.T) {
	svc := newService(&mockRepo{balance: 500}, &mockCache{})

	wrongToken := makeWrongTypeToken(testSecret, "user-1")

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   wrongToken,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if !errors.Is(err, points.ErrInvalidQRToken) {
		t.Errorf("expected ErrInvalidQRToken for wrong type token, got: %v", err)
	}
}

func TestSpendPoints_MarksTokenUsedAfterSuccess(t *testing.T) {
	cache := &mockCache{}
	svc := newService(
		&mockRepo{balance: 500, newBalance: 400},
		cache,
	)

	token, _ := svc.GenerateQRToken(context.Background(), "user-1")
	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cache.markedID == "" {
		t.Error("expected MarkQRUsed to be called after successful spend")
	}
}

func TestSpendPoints_CacheCheckError(t *testing.T) {
	svc := newService(
		&mockRepo{balance: 500},
		&mockCache{isErr: errors.New("redis down")},
	)
	token, _ := svc.GenerateQRToken(context.Background(), "user-1")

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if err == nil {
		t.Fatal("expected error from cache check, got nil")
	}
	// Should NOT be a domain-level error — it's an infrastructure error.
	if errors.Is(err, points.ErrInsufficientBalance) || errors.Is(err, points.ErrQRAlreadyUsed) {
		t.Errorf("unexpected domain error for cache failure: %v", err)
	}
}

func TestSpendPoints_GetBalanceError(t *testing.T) {
	svc := newService(
		&mockRepo{balanceErr: errors.New("db error"), balance: 0},
		&mockCache{},
	)
	token, _ := svc.GenerateQRToken(context.Background(), "user-1")

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if err == nil {
		t.Fatal("expected error from GetBalance, got nil")
	}
}

func TestSpendPoints_SpendAtomicNonConstraintError(t *testing.T) {
	svc := newService(
		&mockRepo{balance: 500, spendErr: errors.New("connection reset")},
		&mockCache{},
	)
	token, _ := svc.GenerateQRToken(context.Background(), "user-1")

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if err == nil {
		t.Fatal("expected error from SpendAtomic, got nil")
	}
	if errors.Is(err, points.ErrInsufficientBalance) {
		t.Error("non-constraint error should not map to ErrInsufficientBalance")
	}
}

func TestSpendPoints_MarkQRUsedError(t *testing.T) {
	svc := newService(
		&mockRepo{balance: 500, newBalance: 400},
		&mockCache{markErr: errors.New("redis write failed")},
	)
	token, _ := svc.GenerateQRToken(context.Background(), "user-1")

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if err == nil {
		t.Fatal("expected error from MarkQRUsed, got nil")
	}
}

func TestSpendPoints_DbConstraintError_ReturnsInsufficientBalance(t *testing.T) {
	// Simulate a scenario where the pre-check passes (balance == amount)
	// but the DB CHECK constraint fires (e.g. concurrent spend).
	svc := newService(
		&mockRepo{balance: 100, spendErr: errors.New("check constraint violation")},
		&mockCache{},
	)

	token, _ := svc.GenerateQRToken(context.Background(), "user-1")

	_, err := svc.SpendPoints(context.Background(), points.SpendRequest{
		QRToken:   token,
		Amount:    100,
		PartnerID: "partner-1",
	})
	if !errors.Is(err, points.ErrInsufficientBalance) {
		t.Errorf("expected ErrInsufficientBalance from constraint error, got: %v", err)
	}
}
