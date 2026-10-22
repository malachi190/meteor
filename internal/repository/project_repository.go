package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"meteor/internal/database"
	"meteor/internal/database/queries"
	"meteor/internal/project"
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
func (r *ProjectRepository) CreateProject(ctx context.Context, name string) (project.Project, error) {
	tenantID, err := tenantUUIDFromContext(ctx)
	if err != nil {
		return project.Project{}, err
	}

	var createdProject queries.Project
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		created, err := queries.New(tx).CreateProject(ctx, queries.CreateProjectParams{
			TenantID: tenantID,
			Name:     name,
		})
		if err != nil {
			return fmt.Errorf("create project: %w", err)
		}
		createdProject = created
		return nil
	})
	if err != nil {
		return project.Project{}, err
	}
	return projectFromRow(createdProject), nil
}

// ListProjects returns only projects belonging to the tenant in the request context.
func (r *ProjectRepository) ListProjects(ctx context.Context) ([]project.Project, error) {
	tenantID, err := tenantUUIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var rows []queries.Project
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		rows, err = queries.New(tx).ListProjectsByTenant(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("list projects: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	projects := make([]project.Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, projectFromRow(row))
	}
	return projects, nil
}

// GetProject returns one project only when it belongs to the request tenant.
func (r *ProjectRepository) GetProject(ctx context.Context, projectID string) (project.Project, error) {
	tenantID, err := tenantUUIDFromContext(ctx)
	if err != nil {
		return project.Project{}, err
	}

	projectUUID, err := parseProjectUUID(projectID)
	if err != nil {
		return project.Project{}, err
	}

	var row queries.Project
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		row, err = queries.New(tx).GetProjectByTenantAndID(ctx, queries.GetProjectByTenantAndIDParams{
			TenantID: tenantID,
			ID:       projectUUID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return project.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get project: %w", err)
		}
		return nil
	})
	if err != nil {
		return project.Project{}, err
	}
	return projectFromRow(row), nil
}

// CreateAPIKey stores a key prefix and hash only after verifying the project belongs to the tenant.
func (r *ProjectRepository) CreateAPIKey(ctx context.Context, projectID, prefix string, keyHash []byte, environment string) (project.APIKey, error) {
	tenantID, err := tenantUUIDFromContext(ctx)
	if err != nil {
		return project.APIKey{}, err
	}
	projectUUID, err := parseProjectUUID(projectID)
	if err != nil {
		return project.APIKey{}, err
	}
	var row queries.CreateAPIKeyRow
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.GetProjectByTenantAndID(ctx, queries.GetProjectByTenantAndIDParams{TenantID: tenantID, ID: projectUUID}); errors.Is(err, pgx.ErrNoRows) {
			return project.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("verify API key project: %w", err)
		}
		row, err = q.CreateAPIKey(ctx, queries.CreateAPIKeyParams{
			TenantID: tenantID, ProjectID: projectUUID, Prefix: prefix, KeyHash: keyHash, Env: environment,
		})
		if err != nil {
			return fmt.Errorf("insert project API key: %w", err)
		}
		return nil
	})
	if err != nil {
		return project.APIKey{}, err
	}
	return apiKeyFromRow(row.ID, row.ProjectID, row.Prefix, row.Env, row.CreatedAt, row.LastUsedAt, row.RevokedAt), nil
}

// ListAPIKeys verifies project ownership before returning its key metadata under RLS.
func (r *ProjectRepository) ListAPIKeys(ctx context.Context, projectID string) ([]project.APIKey, error) {
	tenantID, err := tenantUUIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	projectUUID, err := parseProjectUUID(projectID)
	if err != nil {
		return nil, err
	}
	var rows []queries.ListAPIKeysByTenantProjectRow
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.GetProjectByTenantAndID(ctx, queries.GetProjectByTenantAndIDParams{TenantID: tenantID, ID: projectUUID}); errors.Is(err, pgx.ErrNoRows) {
			return project.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("verify API key project: %w", err)
		}
		rows, err = q.ListAPIKeysByTenantProject(ctx, queries.ListAPIKeysByTenantProjectParams{TenantID: tenantID, ProjectID: projectUUID})
		if err != nil {
			return fmt.Errorf("list project API keys: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	keys := make([]project.APIKey, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, apiKeyFromRow(row.ID, row.ProjectID, row.Prefix, row.Env, row.CreatedAt, row.LastUsedAt, row.RevokedAt))
	}
	return keys, nil
}

// RevokeAPIKey marks an active key revoked after confirming its project is tenant-owned.
func (r *ProjectRepository) RevokeAPIKey(ctx context.Context, projectID, keyID string) error {
	tenantID, err := tenantUUIDFromContext(ctx)
	if err != nil {
		return err
	}
	projectUUID, err := parseProjectUUID(projectID)
	if err != nil {
		return err
	}
	keyUUID, err := parseProjectUUID(keyID)
	if err != nil {
		return err
	}
	return database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.GetProjectByTenantAndID(ctx, queries.GetProjectByTenantAndIDParams{TenantID: tenantID, ID: projectUUID}); errors.Is(err, pgx.ErrNoRows) {
			return project.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("verify API key project: %w", err)
		}
		updated, err := q.RevokeAPIKey(ctx, queries.RevokeAPIKeyParams{TenantID: tenantID, ProjectID: projectUUID, ID: keyUUID})
		if err != nil {
			return fmt.Errorf("revoke project API key: %w", err)
		}
		if updated != 1 {
			return project.ErrNotFound
		}
		return nil
	})
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

// parseProjectUUID converts a project or key identifier to pgx's UUID representation.
func parseProjectUUID(value string) (pgtype.UUID, error) {
	var parsed pgtype.UUID
	if err := parsed.Scan(value); err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse ID: %w", err)
	}
	return parsed, nil
}

// projectFromRow converts a generated SQL row into the project service's JSON-safe model.
func projectFromRow(row queries.Project) project.Project {
	return project.Project{ID: row.ID.String(), TenantID: row.TenantID.String(), Name: row.Name, CreatedAt: row.CreatedAt.Time}
}

// apiKeyFromRow converts generated database fields while omitting the one-time raw secret.
func apiKeyFromRow(id, projectID pgtype.UUID, prefix, environment string, created, lastUsed, revoked pgtype.Timestamptz) project.APIKey {
	key := project.APIKey{ID: id.String(), ProjectID: projectID.String(), Prefix: prefix, Environment: environment, CreatedAt: created.Time}
	if lastUsed.Valid {
		value := lastUsed.Time
		key.LastUsedAt = &value
	}
	if revoked.Valid {
		value := revoked.Time
		key.RevokedAt = &value
	}
	return key
}
