package health

import "context"

type Repository interface {
	CheckDatabase(context.Context) error
}

// Service contains the health-check use case and depends on a small repository interface.
type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Check(ctx context.Context) error {
	return s.repository.CheckDatabase(ctx)
}
