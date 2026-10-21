package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisSessionStoreRoundTrip verifies Redis connectivity and the session create, resolve, and revoke path.
func TestRedisSessionStoreRoundTrip(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("set TEST_REDIS_URL to verify a live Redis connection and session lifecycle")
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("parse TEST_REDIS_URL: %v", err)
	}
	client := redis.NewClient(options)
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("Redis PING failed: %v", err)
	}

	store := NewRedisSessionStore(client)
	account := Account{ID: "11111111-1111-4111-8111-111111111111", TenantID: "22222222-2222-4222-8222-222222222222", Email: "test@example.com", Role: "owner"}
	secret := []byte("test-session-secret")
	tokenHash := sha256.Sum256(secret)
	if _, err := store.Create(ctx, account, tokenHash[:]); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	resolved, renewedUntil, err := store.Resolve(ctx, account.TenantID, account.ID, tokenHash[:])
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved != account || renewedUntil != nil {
		t.Fatalf("Resolve() = (%+v, %v), want stored account without renewal", resolved, renewedUntil)
	}

	wrongHash := sha256.Sum256([]byte("wrong-session-secret"))
	if _, _, err := store.Resolve(ctx, account.TenantID, account.ID, wrongHash[:]); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Resolve(wrong token) error = %v, want ErrInvalidCredentials", err)
	}
	if err := store.Revoke(ctx, account, tokenHash[:]); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if _, _, err := store.Resolve(ctx, account.TenantID, account.ID, tokenHash[:]); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Resolve(revoked token) error = %v, want ErrInvalidCredentials", err)
	}
}
