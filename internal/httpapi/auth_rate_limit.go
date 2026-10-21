package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// authRateLimiter tracks fixed-window request counts by route and client IP in one process.
type authRateLimiter struct {
	mu        sync.Mutex
	window    time.Duration
	started   map[string]time.Time
	requests  map[string]int
	lastSweep time.Time
}

// newAuthRateLimiter creates an empty fixed-window limiter with the requested window size.
func newAuthRateLimiter(window time.Duration) *authRateLimiter {
	return &authRateLimiter{window: window, started: make(map[string]time.Time), requests: make(map[string]int)}
}

// allow atomically checks and consumes one request slot for a route and remote address.
func (l *authRateLimiter) allow(route string, r *http.Request, max int) bool {
	key := route + ":" + clientIP(r.RemoteAddr)
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if started, ok := l.started[key]; !ok || now.Sub(started) >= l.window {
		l.started[key] = now
		l.requests[key] = 0
	}
	if len(l.started) > 10000 && now.Sub(l.lastSweep) > l.window {
		for existing, started := range l.started {
			if now.Sub(started) >= l.window {
				delete(l.started, existing)
				delete(l.requests, existing)
			}
		}
		l.lastSweep = now
	}
	if l.requests[key] >= max {
		return false
	}
	l.requests[key]++
	return true
}

// clientIP strips the port from RemoteAddr without trusting forwarded headers.
func clientIP(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddress)
}

// allowAuthRequest writes a retryable 429 response when a route has exceeded its local limit.
func allowAuthRequest(w http.ResponseWriter, r *http.Request, limiter *authRateLimiter, route string, max int) bool {
	if limiter.allow(route, r, max) {
		return true
	}
	w.Header().Set("Retry-After", "60")
	writeAuthError(w, http.StatusTooManyRequests, "rate_limited", "Too many attempts. Try again shortly.")
	return false
}
