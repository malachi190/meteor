package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthRepository exposes the database check needed by the health service.
type HealthRepository struct {
	pool *pgxpool.Pool
}

func NewHealthRepository(pool *pgxpool.Pool) *HealthRepository {
	return &HealthRepository{pool: pool}
}

func (r *HealthRepository) CheckDatabase(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
