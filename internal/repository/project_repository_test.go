package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"meteor/internal/database"
	"meteor/internal/project"
	"meteor/internal/tenant"
)

// TestProjectAndKeyOperationsStayWithinTenant checks RLS-backed CRUD and cross-tenant denial.
func TestProjectAndKeyOperationsStayWithinTenant(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to verify project and API-key tenant isolation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)

	tenantRepo := NewTenantRepository(pool)
	firstTenant, err := tenantRepo.CreateTenant(ctx, fmt.Sprintf("project-test-a-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("create first test tenant: %v", err)
	}
	secondTenant, err := tenantRepo.CreateTenant(ctx, fmt.Sprintf("project-test-b-%d", time.Now().UnixNano()))
	if err != nil {
		cleanupTestTenant(t, pool, firstTenant.ID.String())
		t.Fatalf("create second test tenant: %v", err)
	}
	t.Cleanup(func() {
		cleanupTestTenant(t, pool, firstTenant.ID.String())
		cleanupTestTenant(t, pool, secondTenant.ID.String())
	})

	service := project.NewService(NewProjectRepository(pool))
	firstContext := tenant.WithID(ctx, tenant.ID(firstTenant.ID.String()))
	secondContext := tenant.WithID(ctx, tenant.ID(secondTenant.ID.String()))
	createdProject, err := service.CreateProject(firstContext, "Integration project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	key, err := service.CreateAPIKey(firstContext, createdProject.ID, "test")
	if err != nil {
		t.Fatalf("create project key: %v", err)
	}
	if key.Secret == "" {
		t.Fatal("new API key did not return its one-time secret")
	}
	liveKey, err := service.CreateAPIKey(firstContext, createdProject.ID, "live")
	if err != nil {
		t.Fatalf("create live project key: %v", err)
	}
	if liveKey.Secret == "" || liveKey.Environment != "live" {
		t.Fatal("live key did not return its one-time secret and environment")
	}

	keys, err := service.ListAPIKeys(firstContext, createdProject.ID)
	if err != nil {
		t.Fatalf("list project keys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("listed keys = %+v, want both test and live keys", keys)
	}
	for _, listed := range keys {
		if listed.Secret != "" {
			t.Fatal("listing returned a raw API key secret")
		}
	}
	if err := service.RevokeAPIKey(firstContext, createdProject.ID, key.ID); err != nil {
		t.Fatalf("revoke project key: %v", err)
	}
	keys, err = service.ListAPIKeys(firstContext, createdProject.ID)
	if err != nil || len(keys) != 2 {
		t.Fatalf("listed revoked key = %+v, error = %v", keys, err)
	}
	var revoked bool
	for _, listed := range keys {
		if listed.ID == key.ID {
			revoked = listed.RevokedAt != nil
		}
	}
	if !revoked {
		t.Fatalf("revoked key %s was not marked revoked", key.ID)
	}
	if _, err := service.GetProject(secondContext, createdProject.ID); !errors.Is(err, project.ErrNotFound) {
		t.Fatalf("cross-tenant project lookup error = %v, want ErrNotFound", err)
	}
	if _, err := service.ListAPIKeys(secondContext, createdProject.ID); !errors.Is(err, project.ErrNotFound) {
		t.Fatalf("cross-tenant key listing error = %v, want ErrNotFound", err)
	}
}

// cleanupTestTenant removes integration-test rows with the same RLS scope used for normal writes.
func cleanupTestTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tenantContext := tenant.WithID(ctx, tenant.ID(tenantID))
	err := database.WithTenant(tenantContext, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "DELETE FROM projects WHERE tenant_id = $1", tenantID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
		return err
	})
	if err != nil {
		t.Errorf("clean up test tenant %s: %v", tenantID, err)
	}
}
