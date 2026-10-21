package database

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestNewPoolRejectsInvalidURL covers local configuration errors without requiring a database.
func TestNewPoolRejectsInvalidURL(t *testing.T) {
	if pool, err := NewPool(context.Background(), "://not-a-database-url"); err == nil {
		pool.Close()
		t.Fatal("NewPool() accepted an invalid database URL")
	}
}

// TestNewPoolConnectsToConfiguredDatabase verifies a real PostgreSQL handshake when opted in.
func TestNewPoolConnectsToConfiguredDatabase(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to verify a live PostgreSQL connection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("NewPool() could not connect to PostgreSQL: %v", err)
	}
	defer pool.Close()
}
