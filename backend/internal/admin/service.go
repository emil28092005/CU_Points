package admin

import (
	"context"
	"fmt"

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
// It includes the associated user email for quick identification.
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
// (newest first) and the total row count for pagination metadata.
func (s *Service) ListTransactions(ctx context.Context, limit, offset int) ([]AdminTransaction, int, error) {
	var total int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListTransactions: count: %w", err)
	}

	rows, err := s.db.Query(ctx, `
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
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListTransactions: query: %w", err)
	}
	defer rows.Close()

	var txs []AdminTransaction
	for rows.Next() {
		var t AdminTransaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.UserEmail, &t.PartnerID,
			&t.Amount, &t.Type, &t.Description, &t.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("service.ListTransactions: scan: %w", err)
		}
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("service.ListTransactions: rows: %w", err)
	}
	return txs, total, nil
}

// ListStudents returns all users with role=student, ordered by name.
func (s *Service) ListStudents(ctx context.Context) ([]Student, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, email, name, COALESCE(student_id, ''), balance
		 FROM users
		 WHERE role = 'student'
		 ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("service.ListStudents: %w", err)
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var st Student
		if err := rows.Scan(&st.ID, &st.Email, &st.Name, &st.StudentID, &st.Balance); err != nil {
			return nil, fmt.Errorf("service.ListStudents: scan: %w", err)
		}
		students = append(students, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("service.ListStudents: rows: %w", err)
	}
	return students, nil
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
