package httpapi

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestAuthRateLimiterSharesCountersAcrossInstances checks distinct limiter objects use one Redis quota.
func TestAuthRateLimiterSharesCountersAcrossInstances(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("set TEST_REDIS_URL to verify shared auth rate limits against Redis")
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("parse TEST_REDIS_URL: %v", err)
	}
	firstClient := redis.NewClient(options)
	secondClient := redis.NewClient(options)
	t.Cleanup(func() { _ = firstClient.Close() })
	t.Cleanup(func() { _ = secondClient.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := firstClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("Redis PING failed: %v", err)
	}

	firstInstance := newAuthRateLimiter(firstClient, time.Minute)
	secondInstance := newAuthRateLimiter(secondClient, time.Minute)
	route := fmt.Sprintf("test-login-%d", time.Now().UnixNano())
	request := httptest.NewRequest("POST", "/v1/auth/login", nil)
	request.RemoteAddr = "198.51.100.24:41000"

	allowed, err := firstInstance.allow(ctx, route, request, 1)
	if err != nil || !allowed {
		t.Fatalf("first allow() = (%v, %v), want (true, nil)", allowed, err)
	}
	allowed, err = secondInstance.allow(ctx, route, request, 1)
	if err != nil || allowed {
		t.Fatalf("second instance allow() = (%v, %v), want (false, nil)", allowed, err)
	}

	request.RemoteAddr = "198.51.100.25:41000"
	allowed, err = secondInstance.allow(ctx, route, request, 1)
	if err != nil || !allowed {
		t.Fatalf("allow() for a different IP = (%v, %v), want (true, nil)", allowed, err)
	}
}
