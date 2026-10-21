package health

import "context"

// Repository is the database availability check consumed by the health service.
type Repository interface {
	CheckDatabase(context.Context) error
}

// Service contains the health-check use case and depends on a small repository interface.
type Service struct {
	repository Repository
}

// NewService wraps the repository dependency used by the health endpoint.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// Check reports whether the configured health repository can reach PostgreSQL.
func (s *Service) Check(ctx context.Context) error {
	return s.repository.CheckDatabase(ctx)
}
