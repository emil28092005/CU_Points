package users

import "context"

// Service handles business logic for the users domain.
type Service struct {
	repo *Repository
}

// NewService creates a new users Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Profile represents a student's public profile and current balance.
type Profile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	StudentID string `json:"student_id,omitempty"`
	Balance   int    `json:"balance"`
}

// Transaction represents a single point-earning or point-spending event.
type Transaction struct {
	ID          string `json:"id"`
	Amount      int    `json:"amount"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	PartnerID   string `json:"partner_id,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// GetProfile returns the profile and current balance for the given user.
func (s *Service) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	return s.repo.GetByID(ctx, userID)
}

// GetTransactions returns a paginated list of transactions for the given user
// (newest first) together with the total row count for pagination metadata.
func (s *Service) GetTransactions(ctx context.Context, userID string, limit, offset int) ([]Transaction, int, error) {
	total, err := s.repo.CountTransactions(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	txs, err := s.repo.ListTransactions(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return txs, total, nil
}
