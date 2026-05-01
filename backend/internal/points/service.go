package points

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInsufficientBalance is returned when a student's balance is lower than the requested spend amount.
var ErrInsufficientBalance = errors.New("insufficient balance")

// ErrQRAlreadyUsed is returned when a QR token has already been redeemed.
var ErrQRAlreadyUsed = errors.New("QR token already used")

// ErrInvalidQRToken is returned when the QR JWT is malformed, expired, or has the wrong type.
var ErrInvalidQRToken = errors.New("invalid or expired QR token")

const qrTokenTTL = 5 * time.Minute

// qrClaims are the JWT payload fields for one-time QR spend tokens.
type qrClaims struct {
	jwt.RegisteredClaims
	Type string `json:"type"` // always "qr"
}

// Service contains the critical business logic for points operations.
// This is the most important file in the project — all balance mutations live here.
type Service struct {
	repo   Repository
	cache  CacheClient
	secret []byte
}

// NewService creates a new points Service.
// secret must be the same HMAC secret used for all JWTs in this application.
func NewService(repo Repository, cache CacheClient, secret string) *Service {
	return &Service{repo: repo, cache: cache, secret: []byte(secret)}
}

// EarnRequest holds the data needed to credit a student's balance.
// Type must be "earn" or "admin_grant" — enforced by the DB CHECK constraint.
type EarnRequest struct {
	UserID      string
	Amount      int
	Type        string // "earn" or "admin_grant"
	Description string
}

// EarnPoints credits the given amount to the student's balance and records a
// transaction of the specified type atomically. This is the single place where
// all credit logic lives — admin grants, future LMS integrations, etc. must
// call this method rather than touching the DB directly.
func (s *Service) EarnPoints(ctx context.Context, req EarnRequest) error {
	if req.Amount <= 0 {
		return fmt.Errorf("service.EarnPoints: amount must be positive")
	}
	if err := s.repo.EarnAtomic(ctx, req.UserID, req.Amount, req.Type, req.Description); err != nil {
		return fmt.Errorf("service.EarnPoints: %w", err)
	}
	return nil
}

// SpendRequest holds the data needed to debit a student's balance at a partner.
type SpendRequest struct {
	QRToken   string
	Amount    int
	PartnerID string
}

// SpendPoints debits the given amount from the student's balance
// and records a spend transaction atomically in a single DB transaction.
// Returns the student's new balance on success.
// Returns ErrInvalidQRToken if the token is malformed or expired.
// Returns ErrQRAlreadyUsed if the QR token has been redeemed before.
// Returns ErrInsufficientBalance if balance < amount.
func (s *Service) SpendPoints(ctx context.Context, req SpendRequest) (int, error) {
	// Step 1: validate QR JWT and extract student_id and jti.
	claims, err := s.parseQRToken(req.QRToken)
	if err != nil {
		return 0, ErrInvalidQRToken
	}
	studentID := claims.Subject
	jti := claims.ID

	// Step 2: one-time-use check — reject if already redeemed.
	used, err := s.cache.IsQRUsed(ctx, jti)
	if err != nil {
		return 0, fmt.Errorf("service.SpendPoints: cache check: %w", err)
	}
	if used {
		return 0, ErrQRAlreadyUsed
	}

	// Step 3: pre-check balance for a clear error message before hitting the DB.
	// The DB CHECK (balance >= 0) is the authoritative guard; this is a fast-fail.
	balance, err := s.repo.GetBalance(ctx, studentID)
	if err != nil {
		return 0, fmt.Errorf("service.SpendPoints: get balance: %w", err)
	}
	if balance < req.Amount {
		return 0, ErrInsufficientBalance
	}

	// Step 4–5: debit balance and insert spend transaction atomically.
	// SpendAtomic uses a DB transaction; the balance CHECK constraint is the
	// last line of defence against concurrent overdrafts.
	newBalance, err := s.repo.SpendAtomic(ctx, studentID, req.PartnerID, req.Amount)
	if err != nil {
		// Propagate balance constraint violation with a domain error.
		if isConstraintError(err) {
			return 0, ErrInsufficientBalance
		}
		return 0, fmt.Errorf("service.SpendPoints: spend atomic: %w", err)
	}

	// Step 6: mark token as used only after the DB commit succeeds.
	// If MarkQRUsed fails, the spend already committed — log but don't rollback.
	if err := s.cache.MarkQRUsed(ctx, jti); err != nil {
		return 0, fmt.Errorf("service.SpendPoints: mark qr used: %w", err)
	}

	return newBalance, nil
}

// GenerateQRToken creates a one-time JWT for the student to present at a partner terminal.
// The token encodes the student's user_id and a unique jti; TTL is 5 minutes.
func (s *Service) GenerateQRToken(ctx context.Context, userID string) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", fmt.Errorf("service.GenerateQRToken: generate jti: %w", err)
	}

	claims := qrClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(qrTokenTTL)),
		},
		Type: "qr",
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("service.GenerateQRToken: sign: %w", err)
	}
	return token, nil
}

// parseQRToken validates the JWT signature/expiry and asserts type="qr".
func (s *Service) parseQRToken(tokenStr string) (*qrClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &qrClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(*qrClaims)
	if !ok || claims.Type != "qr" {
		return nil, fmt.Errorf("not a QR token")
	}
	return claims, nil
}

// newJTI generates a cryptographically random UUID v4 string for use as a JWT ID.
func newJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// isConstraintError reports whether err contains a PostgreSQL balance CHECK violation.
func isConstraintError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "check")
}
