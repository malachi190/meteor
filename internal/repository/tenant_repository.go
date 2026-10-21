package repository

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"meteor/internal/database"
	"meteor/internal/database/queries"
	"meteor/internal/tenant"
)

// TenantRepository creates tenants while satisfying the tenant RLS policy.
type TenantRepository struct {
	pool *pgxpool.Pool
}

// NewTenantRepository binds tenant onboarding operations to the shared PostgreSQL pool.
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

// CreateTenant generates the tenant ID before starting its RLS-scoped insert.
func (r *TenantRepository) CreateTenant(ctx context.Context, name string) (queries.CreateTenantRow, error) {
	tenantID, err := newTenantUUID()
	if err != nil {
		return queries.CreateTenantRow{}, fmt.Errorf("generate tenant ID: %w", err)
	}

	ctx = tenant.WithID(ctx, tenant.ID(tenantID.String()))
	var created queries.CreateTenantRow
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		tenantRecord, err := queries.New(tx).CreateTenant(ctx, queries.CreateTenantParams{
			ID:   tenantID,
			Name: name,
		})
		if err != nil {
			return fmt.Errorf("insert tenant: %w", err)
		}
		created = tenantRecord
		return nil
	})
	if err != nil {
		return queries.CreateTenantRow{}, fmt.Errorf("create tenant: %w", err)
	}
	return created, nil
}

// newTenantUUID creates a random version-four UUID before the RLS-scoped tenant insert.
func newTenantUUID() (pgtype.UUID, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return pgtype.UUID{}, err
	}

	// Set the RFC 4122 version 4 and variant bits on the random bytes.
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	return pgtype.UUID{Bytes: id, Valid: true}, nil
}
