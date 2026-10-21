package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"meteor/internal/tenant"
)

// TestWithTenantFailsClosedWithoutContext confirms untrusted requests cannot open tenant work without scope.
func TestWithTenantFailsClosedWithoutContext(t *testing.T) {
	err := WithTenant(context.Background(), nil, func(context.Context, pgx.Tx) error { return nil })
	if !errors.Is(err, ErrTenantContextRequired) {
		t.Fatalf("WithTenant() error = %v, want ErrTenantContextRequired", err)
	}
}

// TestWithTenantRequiresCallback catches accidental calls that cannot commit useful work.
func TestWithTenantRequiresCallback(t *testing.T) {
	ctx := tenant.WithID(context.Background(), "22222222-2222-4222-8222-222222222222")
	if err := WithTenant(ctx, nil, nil); err == nil {
		t.Fatal("WithTenant() accepted a nil callback")
	}
}

// TestWithTenantSetsTransactionLocalContext verifies the RLS setting inside a live PostgreSQL transaction.
func TestWithTenantSetsTransactionLocalContext(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to verify PostgreSQL tenant transaction context")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("NewPool() could not connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	const tenantID = "22222222-2222-4222-8222-222222222222"
	err = WithTenant(tenant.WithID(ctx, tenant.ID(tenantID)), pool, func(ctx context.Context, tx pgx.Tx) error {
		var configuredTenant string
		if err := tx.QueryRow(ctx, "SELECT current_setting('app.tenant_id', true)").Scan(&configuredTenant); err != nil {
			return err
		}
		if configuredTenant != tenantID {
			return fmt.Errorf("transaction tenant setting = %q, want %q", configuredTenant, tenantID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant() error = %v", err)
	}
}
