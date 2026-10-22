package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"meteor/internal/logging"
)

const authRateLimitPrefix = "meteor:rate_limit:auth:"

var fixedWindowScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`)

// authRateLimiter stores fixed-window counters in shared Redis so API instances enforce one limit.
type authRateLimiter struct {
	client redis.UniversalClient
	window time.Duration
}

// newAuthRateLimiter creates a Redis-backed fixed-window limiter.
func newAuthRateLimiter(client redis.UniversalClient, window time.Duration) *authRateLimiter {
	return &authRateLimiter{client: client, window: window}
}

// allow atomically counts one request for a route and remote IP and reports whether it is within limit.
func (l *authRateLimiter) allow(ctx context.Context, route string, r *http.Request, max int) (bool, error) {
	if l.client == nil || l.window <= 0 || max <= 0 {
		return false, errors.New("auth rate limiter is not configured")
	}
	identity := route + "\x00" + clientIP(r.RemoteAddr)
	digest := sha256.Sum256([]byte(identity))
	key := authRateLimitPrefix + hex.EncodeToString(digest[:])
	windowSeconds := int64(math.Ceil(l.window.Seconds()))
	count, err := fixedWindowScript.Run(ctx, l.client, []string{key}, windowSeconds).Int64()
	if err != nil {
		return false, err
	}
	return count <= int64(max), nil
}

// clientIP strips the port from RemoteAddr without trusting forwarded headers.
func clientIP(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddress)
}

// allowAuthRequest fails closed if Redis is unavailable and returns 429 after the shared limit is reached.
func allowAuthRequest(w http.ResponseWriter, r *http.Request, limiter *authRateLimiter, logs *logging.Loggers, route string, max int) bool {
	allowed, err := limiter.allow(r.Context(), route, r, max)
	if err != nil {
		logs.Errors.Error().Err(err).Str("route", route).Msg("shared auth rate limiter failed")
		writeAuthError(w, http.StatusServiceUnavailable, "rate_limit_unavailable", "Authentication is temporarily unavailable. Try again shortly.")
		return false
	}
	if allowed {
		return true
	}
	w.Header().Set("Retry-After", strconv.FormatInt(int64(math.Ceil(limiter.window.Seconds())), 10))
	writeAuthError(w, http.StatusTooManyRequests, "rate_limited", "Too many attempts. Try again shortly.")
	return false
}
