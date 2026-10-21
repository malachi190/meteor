package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"meteor/internal/database"
	"meteor/internal/database/queries"
	"meteor/internal/tenant"
)

// ProjectRepository performs project queries inside tenant-scoped transactions.
type ProjectRepository struct {
	pool *pgxpool.Pool
}

// NewProjectRepository binds tenant-scoped project operations to the shared pool.
func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

// CreateProject inserts a project for the tenant attached to the request context.
func (r *ProjectRepository) CreateProject(ctx context.Context, name string) (queries.Project, error) {
	tenantID, err := tenantUUIDFromContext(ctx)
	if err != nil {
		return queries.Project{}, err
	}

	var project queries.Project
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		created, err := queries.New(tx).CreateProject(ctx, queries.CreateProjectParams{
			TenantID: tenantID,
			Name:     name,
		})
		if err != nil {
			return fmt.Errorf("create project: %w", err)
		}
		project = created
		return nil
	})
	return project, err
}

// ListProjects returns only projects belonging to the tenant in the request context.
func (r *ProjectRepository) ListProjects(ctx context.Context) ([]queries.Project, error) {
	tenantID, err := tenantUUIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var projects []queries.Project
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		projects, err = queries.New(tx).ListProjectsByTenant(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("list projects: %w", err)
		}
		return nil
	})
	return projects, err
}

// GetProject returns one project only when it belongs to the request tenant.
func (r *ProjectRepository) GetProject(ctx context.Context, projectID string) (queries.Project, error) {
	tenantID, err := tenantUUIDFromContext(ctx)
	if err != nil {
		return queries.Project{}, err
	}

	var projectUUID pgtype.UUID
	if err := projectUUID.Scan(projectID); err != nil {
		return queries.Project{}, fmt.Errorf("parse project ID: %w", err)
	}

	var project queries.Project
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		project, err = queries.New(tx).GetProjectByTenantAndID(ctx, queries.GetProjectByTenantAndIDParams{
			TenantID: tenantID,
			ID:       projectUUID,
		})
		if err != nil {
			return fmt.Errorf("get project: %w", err)
		}
		return nil
	})
	return project, err
}

// tenantUUIDFromContext requires the trusted tenant scope and converts it for typed SQL queries.
func tenantUUIDFromContext(ctx context.Context) (pgtype.UUID, error) {
	id, ok := tenant.IDFromContext(ctx)
	if !ok {
		return pgtype.UUID{}, database.ErrTenantContextRequired
	}

	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(string(id)); err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse tenant ID: %w", err)
	}
	return tenantUUID, nil
}
