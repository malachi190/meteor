package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"meteor/internal/health"
	"meteor/internal/logging"
)

type healthResponse struct {
	Status string `json:"status"`
}

// NewServer builds the HTTP handler layer and applies request logging to its routes.
func NewServer(service *health.Service, logs *logging.Loggers) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := service.Check(r.Context()); err != nil {
			logs.Errors.Error().Err(err).Msg("health check failed")
			writeHealth(w, http.StatusServiceUnavailable, "unavailable", logs.Errors)
			return
		}

		writeHealth(w, http.StatusOK, "ok", logs.Errors)
	})

	return requestLogging(mux, logs)
}

func writeHealth(w http.ResponseWriter, statusCode int, status string, logger zerolog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(healthResponse{Status: status}); err != nil {
		logger.Error().Err(err).Msg("write health response")
	}
}

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

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
