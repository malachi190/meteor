package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"meteor/internal/auth"
	"meteor/internal/health"
	"meteor/internal/logging"
	"meteor/internal/project"

	"github.com/redis/go-redis/v9"
)

// healthResponse is the small JSON body returned by the database-backed health endpoint.
type healthResponse struct {
	Status string `json:"status"`
}

// NewServer builds the HTTP handler layer and applies request logging to its routes.
func NewServer(service *health.Service, authService *auth.Service, projectService *project.Service, rateLimitClient redis.UniversalClient, logs *logging.Loggers, cookieSecure bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := service.Check(r.Context()); err != nil {
			logs.Errors.Error().Err(err).Msg("health check failed")
			writeHealth(w, http.StatusServiceUnavailable, "unavailable", logs.Errors)
			return
		}

		writeHealth(w, http.StatusOK, "ok", logs.Errors)
	})
	registerAuthRoutes(mux, authService, rateLimitClient, logs, cookieSecure)
	registerProjectRoutes(mux, projectService, authService, logs, cookieSecure)

	return requestLogging(mux, logs)
}

// writeHealth encodes the health status and logs failures while writing the response.
func writeHealth(w http.ResponseWriter, statusCode int, status string, logger zerolog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(healthResponse{Status: status}); err != nil {
		logger.Error().Err(err).Msg("write health response")
	}
}

// requestLogging records route metadata while omitting headers, query strings, and credentials.
func requestLogging(next http.Handler, logs *logging.Loggers) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		response := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(response, r)

		event := logs.Requests.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", response.status).
			Dur("duration", time.Since(startedAt))
		event.Msg("http request")

		if response.status >= http.StatusInternalServerError {
			logs.Errors.Error().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", response.status).
				Msg("request returned server error")
		}
	})
}

// statusWriter captures the first response status while preserving the underlying writer.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// WriteHeader records and forwards only the first status code for this response.
func (w *statusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

// Write defaults the status to 200 before forwarding the response body.
func (w *statusWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

// Unwrap exposes the original writer so standard HTTP helpers can reach its capabilities.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
