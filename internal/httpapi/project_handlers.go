package httpapi

import (
	"errors"
	"net/http"

	"meteor/internal/auth"
	"meteor/internal/logging"
	"meteor/internal/project"
)

// registerProjectRoutes installs session-protected project and API-key management endpoints.
func registerProjectRoutes(mux *http.ServeMux, projects *project.Service, authService *auth.Service, logs *logging.Loggers, cookieSecure bool) {
	mux.Handle("POST /v1/projects", requireDashboardSession(authService, logs, cookieSecure, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Name string `json:"name"`
		}
		if err := decodeJSON(w, r, &request); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
			return
		}
		created, err := projects.CreateProject(r.Context(), request.Name)
		if err != nil {
			writeProjectError(w, logs, "create project", err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"project": created})
	})))

	mux.Handle("GET /v1/projects", requireDashboardSession(authService, logs, cookieSecure, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items, err := projects.ListProjects(r.Context())
		if err != nil {
			writeProjectError(w, logs, "list projects", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"projects": items})
	})))

	mux.Handle("GET /v1/projects/{projectID}", requireDashboardSession(authService, logs, cookieSecure, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		item, err := projects.GetProject(r.Context(), r.PathValue("projectID"))
		if err != nil {
			writeProjectError(w, logs, "get project", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"project": item})
	})))

	mux.Handle("POST /v1/projects/{projectID}/keys", requireDashboardSession(authService, logs, cookieSecure, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Environment string `json:"environment"`
		}
		if err := decodeJSON(w, r, &request); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
			return
		}
		key, err := projects.CreateAPIKey(r.Context(), r.PathValue("projectID"), request.Environment)
		if err != nil {
			writeProjectError(w, logs, "create project API key", err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusCreated, map[string]any{"key": key})
	})))

	mux.Handle("GET /v1/projects/{projectID}/keys", requireDashboardSession(authService, logs, cookieSecure, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys, err := projects.ListAPIKeys(r.Context(), r.PathValue("projectID"))
		if err != nil {
			writeProjectError(w, logs, "list project API keys", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
	})))

	mux.Handle("DELETE /v1/projects/{projectID}/keys/{keyID}", requireDashboardSession(authService, logs, cookieSecure, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := projects.RevokeAPIKey(r.Context(), r.PathValue("projectID"), r.PathValue("keyID")); err != nil {
			writeProjectError(w, logs, "revoke project API key", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})))
}

// writeProjectError maps expected project errors to public statuses and logs unexpected failures.
func writeProjectError(w http.ResponseWriter, logs *logging.Loggers, operation string, err error) {
	switch {
	case errors.Is(err, project.ErrInvalidInput):
		writeAuthError(w, http.StatusBadRequest, "invalid_project_request", "Project or API key input is invalid.")
	case errors.Is(err, project.ErrNotFound):
		writeAuthError(w, http.StatusNotFound, "project_or_key_not_found", "The requested project or API key was not found.")
	case errors.Is(err, project.ErrConflict):
		writeAuthError(w, http.StatusConflict, "project_or_key_conflict", "A project or API key with those details already exists.")
	default:
		logs.Errors.Error().Err(err).Str("operation", operation).Msg("project operation failed")
		writeAuthError(w, http.StatusInternalServerError, "project_operation_failed", "Could not complete the project operation.")
	}
}
