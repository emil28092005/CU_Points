package partners

import "context"

// Service handles business logic for the partners domain.
type Service struct {
	repo *Repository
}

// NewService creates a new partners Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Partner represents a participating business.
type Partner struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	MaxSpendPct int    `json:"max_spend_pct"`
}

// ListActive returns all partners with is_active = true.
func (s *Service) ListActive(ctx context.Context) ([]Partner, error) {
	return s.repo.ListActive(ctx)
}

// GetByID returns a partner by its primary key.
// Returns ErrNotFound (from repository) if the partner does not exist.
func (s *Service) GetByID(ctx context.Context, id string) (*Partner, error) {
	return s.repo.GetByID(ctx, id)
}
