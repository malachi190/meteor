package httpapi

import (
	"net/http"
)

const (
	allowedDashboardMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowedDashboardHeaders = "Authorization, Content-Type"
)

// dashboardCORS permits credentialed browser requests from the dashboard and configured public API origins.
func dashboardCORS(next http.Handler, dashboardOrigin, publicBaseURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if origin != dashboardOrigin && origin != publicBaseURL {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}

		header := w.Header()
		header.Add("Vary", "Origin")
		header.Set("Access-Control-Allow-Origin", origin)
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Methods", allowedDashboardMethods)
		header.Set("Access-Control-Allow-Headers", allowedDashboardHeaders)
		header.Set("Access-Control-Expose-Headers", "Retry-After")

		if r.Method == http.MethodOptions {
			header.Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
