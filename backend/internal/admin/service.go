package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cu-points/backend/internal/points"
)

// Service handles business logic for administrative operations.
type Service struct {
	db     *pgxpool.Pool
	points *points.Service
}

// NewService creates a new admin Service.
// pointsSvc is used for all balance mutations so that earn logic is not duplicated here.
func NewService(db *pgxpool.Pool, pointsSvc *points.Service) *Service {
	return &Service{db: db, points: pointsSvc}
}

// AdminTransaction is a transaction record as seen by an administrator.
type AdminTransaction struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	UserEmail   string `json:"user_email"`
	PartnerID   string `json:"partner_id,omitempty"`
	Amount      int    `json:"amount"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// Student is a user record as seen by an administrator.
type Student struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	StudentID string `json:"student_id,omitempty"`
	Balance   int    `json:"balance"`
	CreatedAt string `json:"created_at"`
}

// Stats holds aggregated system metrics shown on the admin dashboard.
type Stats struct {
	TotalStudents     int `json:"total_students"`
	TotalPointsIssued int `json:"total_points_issued"`
	TotalPointsSpent  int `json:"total_points_spent"`
	ActivePartners    int `json:"active_partners"`
}

// GrantPoints credits the given amount to the student's balance and records an
// admin_grant transaction. Delegates to points.Service.EarnPoints so that all
// balance mutation logic lives in one place.
func (s *Service) GrantPoints(ctx context.Context, userID string, amount int, description string) error {
	return s.points.EarnPoints(ctx, points.EarnRequest{
		UserID:      userID,
		Amount:      amount,
		Type:        "admin_grant",
		Description: description,
	})
}

// ListTransactions returns a paginated slice of all transactions in the system
// (newest first) and the total row count. txType filters by transaction type when non-empty.
func (s *Service) ListTransactions(ctx context.Context, limit, offset int, txType string) ([]AdminTransaction, int, error) {
	var total int
	var err error

	if txType != "" {
		err = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE type = $1`, txType).Scan(&total)
	} else {
		err = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListTransactions: count: %w", err)
	}

	var query string
	var args []any

	if txType != "" {
		query = `
			SELECT t.id,
			       t.user_id,
			       u.email,
			       COALESCE(t.partner_id::text, ''),
			       t.amount,
			       t.type,
			       COALESCE(t.description, ''),
			       t.created_at
			FROM transactions t
			JOIN users u ON u.id = t.user_id
			WHERE t.type = $3
			ORDER BY t.created_at DESC
			LIMIT $1 OFFSET $2`
		args = []any{limit, offset, txType}
	} else {
		query = `
			SELECT t.id,
			       t.user_id,
			       u.email,
			       COALESCE(t.partner_id::text, ''),
			       t.amount,
			       t.type,
			       COALESCE(t.description, ''),
			       t.created_at
			FROM transactions t
			JOIN users u ON u.id = t.user_id
			ORDER BY t.created_at DESC
			LIMIT $1 OFFSET $2`
		args = []any{limit, offset}
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListTransactions: query: %w", err)
	}
	defer rows.Close()

	var txs []AdminTransaction
	for rows.Next() {
		var t AdminTransaction
		var createdAt time.Time
		if err := rows.Scan(&t.ID, &t.UserID, &t.UserEmail, &t.PartnerID,
			&t.Amount, &t.Type, &t.Description, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("service.ListTransactions: scan: %w", err)
		}
		t.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("service.ListTransactions: rows: %w", err)
	}
	return txs, total, nil
}

// ListStudents returns students with optional search and pagination.
// search filters by email or name (case-insensitive); empty string returns all.
// Results are sorted by balance DESC when no search term, by name when searching.
func (s *Service) ListStudents(ctx context.Context, search string, limit, offset int) ([]Student, int, error) {
	var total int
	var err error

	if search != "" {
		err = s.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM users
			 WHERE role = 'student'
			   AND (email ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%')`,
			search,
		).Scan(&total)
	} else {
		err = s.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM users WHERE role = 'student'`,
		).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListStudents: count: %w", err)
	}

	var query string
	var args []any

	if search != "" {
		query = `
			SELECT id, email, name, COALESCE(student_id, ''), balance, created_at
			FROM users
			WHERE role = 'student'
			  AND (email ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%')
			ORDER BY name
			LIMIT $2 OFFSET $3`
		args = []any{search, limit, offset}
	} else {
		query = `
			SELECT id, email, name, COALESCE(student_id, ''), balance, created_at
			FROM users
			WHERE role = 'student'
			ORDER BY balance DESC
			LIMIT $1 OFFSET $2`
		args = []any{limit, offset}
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListStudents: query: %w", err)
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var st Student
		var createdAt time.Time
		if err := rows.Scan(&st.ID, &st.Email, &st.Name, &st.StudentID, &st.Balance, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("service.ListStudents: scan: %w", err)
		}
		st.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		students = append(students, st)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("service.ListStudents: rows: %w", err)
	}
	return students, total, nil
}

// GetStats returns aggregated system statistics for the admin dashboard.
func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	var stats Stats

	// Single query for all transaction aggregates.
	err := s.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0),
			COALESCE(ABS(SUM(amount) FILTER (WHERE amount < 0)), 0)
		FROM transactions`,
	).Scan(&stats.TotalPointsIssued, &stats.TotalPointsSpent)
	if err != nil {
		return nil, fmt.Errorf("service.GetStats: transaction aggregates: %w", err)
	}

	err = s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE role = 'student'`,
	).Scan(&stats.TotalStudents)
	if err != nil {
		return nil, fmt.Errorf("service.GetStats: total students: %w", err)
	}

	err = s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM partners WHERE is_active = true`,
	).Scan(&stats.ActivePartners)
	if err != nil {
		return nil, fmt.Errorf("service.GetStats: active partners: %w", err)
	}

	return &stats, nil
}
