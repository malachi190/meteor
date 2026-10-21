package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthRepository exposes the database check needed by the health service.
type HealthRepository struct {
	pool *pgxpool.Pool
}

// NewHealthRepository adapts the shared pool to the health service interface.
func NewHealthRepository(pool *pgxpool.Pool) *HealthRepository {
	return &HealthRepository{pool: pool}
}

// CheckDatabase verifies PostgreSQL responds to a lightweight pool ping.
func (r *HealthRepository) CheckDatabase(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
