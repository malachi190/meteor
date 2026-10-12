package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"meteor/internal/tenant"
)

var ErrTenantContextRequired = errors.New("tenant context is required")

// WithTenant runs fn in a transaction whose PostgreSQL RLS context matches the request tenant.
func WithTenant(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context, pgx.Tx) error) error {
	tenantID, ok := tenant.IDFromContext(ctx)
	if !ok {
		return ErrTenantContextRequired
	}
	if fn == nil {
		return errors.New("tenant transaction callback is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tenant transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", string(tenantID))
	if err != nil {
		return fmt.Errorf("set tenant database context: %w", err)
	}

	if err := fn(ctx, tx); err != nil {
		return fmt.Errorf("run tenant transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tenant transaction: %w", err)
	}
	return nil
}
