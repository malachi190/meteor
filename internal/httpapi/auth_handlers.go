package httpapi

import (
	"encoding/json"
	"errors"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"meteor/internal/auth"
	"meteor/internal/logging"
	"meteor/internal/tenant"

	"github.com/redis/go-redis/v9"
)

const dashboardSessionCookie = "meteor_session"
const githubStateCookie = "meteor_github_state"
const githubWorkspaceCookie = "meteor_github_workspace"

// authResponse is the success envelope shared by authenticated auth endpoints.
type authResponse struct {
	User auth.Account `json:"user"`
}

// authErrorResponse keeps public auth errors in a stable, non-sensitive JSON shape.
type authErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// registerAuthRoutes installs public auth flows and session-protected account endpoints.
func registerAuthRoutes(mux *http.ServeMux, service *auth.Service, rateLimitClient redis.UniversalClient, logs *logging.Loggers, cookieSecure bool) {
	limiter := newAuthRateLimiter(rateLimitClient, time.Minute)
	mux.HandleFunc("GET /v1/auth/github", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "github_start", 10) {
			return
		}
		authorizationURL, state, workspace, err := service.StartGitHub(r.URL.Query().Get("tenant_name"))
		if errors.Is(err, auth.ErrInvalidInput) {
			writeAuthError(w, http.StatusBadRequest, "workspace_required", "Provide a workspace name to continue with GitHub.")
			return
		}
		if err != nil {
			if !errors.Is(err, auth.ErrGitHubUnavailable) {
				logs.Errors.Error().Err(err).Msg("start GitHub OAuth failed")
			}
			writeAuthError(w, http.StatusServiceUnavailable, "github_unavailable", "GitHub sign-in is unavailable.")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: githubStateCookie, Value: state, Path: "/v1/auth/github", MaxAge: 600, HttpOnly: true, Secure: cookieSecure, SameSite: http.SameSiteLaxMode})
		http.SetCookie(w, &http.Cookie{Name: githubWorkspaceCookie, Value: workspace, Path: "/v1/auth/github", MaxAge: 600, HttpOnly: true, Secure: cookieSecure, SameSite: http.SameSiteLaxMode})
		http.Redirect(w, r, authorizationURL, http.StatusFound)
	})

	mux.HandleFunc("GET /v1/auth/github/callback", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "github_callback", 20) {
			clearOAuthCookie(w, githubStateCookie, cookieSecure)
			clearOAuthCookie(w, githubWorkspaceCookie, cookieSecure)
			return
		}
		stateCookie, stateErr := r.Cookie(githubStateCookie)
		workspaceCookie, workspaceErr := r.Cookie(githubWorkspaceCookie)
		clearOAuthCookie(w, githubStateCookie, cookieSecure)
		clearOAuthCookie(w, githubWorkspaceCookie, cookieSecure)
		state, workspace := "", ""
		if stateErr == nil {
			state = stateCookie.Value
		}
		if workspaceErr == nil {
			workspace = workspaceCookie.Value
		}
		account, token, expiresAt, err := service.GitHubCallback(r.Context(), r.URL.Query().Get("code"), r.URL.Query().Get("state"), state, workspace)
		if errors.Is(err, auth.ErrGitHubState) {
			writeAuthError(w, http.StatusBadRequest, "github_state_invalid", "GitHub sign-in expired. Start again.")
			return
		}
		if errors.Is(err, auth.ErrWorkspaceRequired) {
			writeAuthError(w, http.StatusBadRequest, "workspace_required", "Provide a workspace name to finish creating a new tenant.")
			return
		}
		if errors.Is(err, auth.ErrGitHubUnavailable) {
			writeAuthError(w, http.StatusServiceUnavailable, "github_unavailable", "GitHub sign-in is unavailable.")
			return
		}
		if errors.Is(err, auth.ErrGitHubIdentity) {
			writeAuthError(w, http.StatusForbidden, "verified_email_required", "GitHub must provide a verified primary email address.")
			return
		}
		if err != nil {
			logs.Errors.Error().Err(err).Msg("GitHub OAuth callback failed")
			writeAuthError(w, http.StatusBadGateway, "github_signin_failed", "Could not sign in with GitHub. Start again.")
			return
		}
		setDashboardCookie(w, token, expiresAt, cookieSecure)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		_, _ = w.Write([]byte("<!doctype html><html><body><h1>Signed in</h1><p>Your Meteor dashboard session is ready. You can close this page.</p><p>" + html.EscapeString(account.Email) + "</p></body></html>"))
	})

	mux.HandleFunc("POST /v1/auth/signup", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "signup", 5) {
			return
		}
		var request struct {
			TenantName string `json:"tenant_name"`
			Email      string `json:"email"`
			Password   string `json:"password"`
		}
		if err := decodeJSON(w, r, &request); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
			return
		}
		if err := service.Signup(r.Context(), request.TenantName, request.Email, request.Password); err != nil {
			if errors.Is(err, auth.ErrInvalidInput) {
				writeAuthError(w, http.StatusBadRequest, "invalid_signup", "Provide a workspace name, valid email, and password of 12 to 128 bytes.")
				return
			}
			if errors.Is(err, auth.ErrVerificationDelivery) {
				logs.Errors.Error().Err(err).Msg("verification email delivery failed")
				writeJSON(w, http.StatusAccepted, map[string]string{"status": "verification_required"})
				return
			}
			logs.Errors.Error().Err(err).Msg("signup failed")
			writeAuthError(w, http.StatusServiceUnavailable, "signup_unavailable", "Signup is temporarily unavailable. Try again later.")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "verification_required"})
	})

	mux.HandleFunc("POST /v1/auth/resend-verification", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "resend_verification", 3) {
			return
		}
		var request struct {
			Email string `json:"email"`
		}
		if err := decodeJSON(w, r, &request); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
			return
		}
		if err := service.ResendVerification(r.Context(), request.Email); err != nil {
			if errors.Is(err, auth.ErrVerificationDelivery) {
				logs.Errors.Error().Err(err).Msg("verification email delivery failed")
				writeJSON(w, http.StatusAccepted, map[string]string{"status": "if_eligible_check_email"})
				return
			}
			logs.Errors.Error().Err(err).Msg("resend verification failed")
			writeAuthError(w, http.StatusServiceUnavailable, "verification_unavailable", "Verification email is temporarily unavailable.")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "if_eligible_check_email"})
	})

	mux.HandleFunc("POST /v1/auth/password-reset/request", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "password_reset_request", 3) {
			return
		}
		var request struct {
			Email string `json:"email"`
		}
		if err := decodeJSON(w, r, &request); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
			return
		}
		if err := service.RequestPasswordReset(r.Context(), request.Email); err != nil {
			if errors.Is(err, auth.ErrPasswordResetDelivery) {
				logs.Errors.Error().Err(err).Msg("password reset email delivery failed")
				writeJSON(w, http.StatusAccepted, map[string]string{"status": "if_eligible_check_email"})
				return
			}
			logs.Errors.Error().Err(err).Msg("password reset request failed")
			writeAuthError(w, http.StatusServiceUnavailable, "password_reset_unavailable", "Password reset is temporarily unavailable.")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "if_eligible_check_email"})
	})

	mux.HandleFunc("GET /v1/auth/password-reset", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "password_reset_page", 20) {
			return
		}
		token := r.URL.Query().Get("token")
		if token == "" || len(token) > 128 {
			writeAuthError(w, http.StatusBadRequest, "invalid_password_reset", "Password reset link is invalid.")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!doctype html><html><head><meta charset=\"utf-8\"><title>Reset Meteor password</title></head><body><h1>Choose a new password</h1><form method=\"post\" action=\"/v1/auth/password-reset\"><input type=\"hidden\" name=\"token\" value=\"" + html.EscapeString(token) + "\"><label>New password <input type=\"password\" name=\"password\" minlength=\"12\" maxlength=\"128\" required autocomplete=\"new-password\"></label><button type=\"submit\">Reset password</button></form></body></html>"))
	})

	mux.HandleFunc("POST /v1/auth/password-reset", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "password_reset", 10) {
			return
		}
		isForm := strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		var token, password string
		if isForm {
			r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
			if err := r.ParseForm(); err != nil {
				writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
				return
			}
			token, password = r.PostForm.Get("token"), r.PostForm.Get("password")
		} else {
			var request struct {
				Token    string `json:"token"`
				Password string `json:"password"`
			}
			if err := decodeJSON(w, r, &request); err != nil {
				writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
				return
			}
			token, password = request.Token, request.Password
		}
		if err := service.ResetPassword(r.Context(), token, password); err != nil {
			if errors.Is(err, auth.ErrInvalidPasswordReset) {
				writeAuthError(w, http.StatusBadRequest, "invalid_password_reset", "Password reset link is invalid or expired.")
				return
			}
			if errors.Is(err, auth.ErrInvalidInput) {
				writeAuthError(w, http.StatusBadRequest, "invalid_password", "Choose a password of 12 to 128 bytes.")
				return
			}
			logs.Errors.Error().Err(err).Msg("password reset failed")
			writeAuthError(w, http.StatusInternalServerError, "password_reset_failed", "Could not reset the password right now.")
			return
		}
		clearDashboardCookie(w, cookieSecure)
		if isForm {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
			_, _ = w.Write([]byte("<!doctype html><html><body><h1>Password updated</h1><p>All active sessions were signed out. Sign in with your new password.</p></body></html>"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "password_reset"})
	})

	mux.HandleFunc("POST /v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "login", 10) {
			return
		}
		var request struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := decodeJSON(w, r, &request); err != nil {
			writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
			return
		}
		account, token, expiresAt, err := service.Login(r.Context(), request.Email, request.Password)
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeAuthError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect, or the account is not verified.")
			return
		}
		if err != nil {
			logs.Errors.Error().Err(err).Msg("login failed")
			writeAuthError(w, http.StatusInternalServerError, "login_failed", "Could not sign in right now.")
			return
		}
		setDashboardCookie(w, token, expiresAt, cookieSecure)
		writeJSON(w, http.StatusOK, authResponse{User: account})
	})

	mux.HandleFunc("GET /v1/auth/verify-email", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "verification_page", 20) {
			return
		}
		token := r.URL.Query().Get("token")
		if token == "" || len(token) > 128 {
			writeAuthError(w, http.StatusBadRequest, "invalid_verification", "Verification link is invalid.")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!doctype html><html><head><meta charset=\"utf-8\"><title>Verify Meteor account</title></head><body><h1>Verify your email</h1><form method=\"post\" action=\"/v1/auth/verify-email\"><input type=\"hidden\" name=\"token\" value=\"" + html.EscapeString(token) + "\"><button type=\"submit\">Verify email</button></form></body></html>"))
	})

	mux.HandleFunc("POST /v1/auth/verify-email", func(w http.ResponseWriter, r *http.Request) {
		if !allowAuthRequest(w, r, limiter, logs, "verification", 10) {
			return
		}
		var token string
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
			if err := r.ParseForm(); err != nil {
				writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
				return
			}
			token = r.PostForm.Get("token")
		} else {
			var request struct {
				Token string `json:"token"`
			}
			if err := decodeJSON(w, r, &request); err != nil {
				writeAuthError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
				return
			}
			token = request.Token
		}
		account, sessionToken, expiresAt, err := service.VerifyEmail(r.Context(), token)
		if errors.Is(err, auth.ErrInvalidVerification) {
			writeAuthError(w, http.StatusBadRequest, "invalid_verification", "Verification link is invalid or expired.")
			return
		}
		if err != nil {
			logs.Errors.Error().Err(err).Msg("email verification failed")
			writeAuthError(w, http.StatusInternalServerError, "verification_failed", "Could not verify the email right now.")
			return
		}
		setDashboardCookie(w, sessionToken, expiresAt, cookieSecure)
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
			_, _ = w.Write([]byte("<!doctype html><html><body><h1>Email verified</h1><p>Your Meteor account is ready. You can close this page.</p></body></html>"))
			return
		}
		writeJSON(w, http.StatusOK, authResponse{User: account})
	})

	mux.Handle("GET /v1/auth/me", requireDashboardSession(service, logs, cookieSecure, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account, _ := auth.AccountFromContext(r.Context())
		writeJSON(w, http.StatusOK, authResponse{User: account})
	})))

	mux.Handle("POST /v1/auth/logout", requireDashboardSession(service, logs, cookieSecure, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account, _ := auth.AccountFromContext(r.Context())
		token, _ := readDashboardCookie(r)
		if err := service.Logout(r.Context(), account, token); err != nil {
			logs.Errors.Error().Err(err).Msg("logout failed")
			writeAuthError(w, http.StatusInternalServerError, "logout_failed", "Could not sign out right now.")
			return
		}
		clearDashboardCookie(w, cookieSecure)
		w.WriteHeader(http.StatusNoContent)
	})))
}

// requireDashboardSession validates the cookie and attaches trusted account and tenant context.
func requireDashboardSession(service *auth.Service, logs *logging.Loggers, cookieSecure bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := readDashboardCookie(r)
		if err != nil {
			clearDashboardCookie(w, cookieSecure)
			writeAuthError(w, http.StatusUnauthorized, "unauthenticated", "Sign in to continue.")
			return
		}
		account, renewedUntil, err := service.AuthenticateSession(r.Context(), token)
		if errors.Is(err, auth.ErrInvalidCredentials) {
			clearDashboardCookie(w, cookieSecure)
			writeAuthError(w, http.StatusUnauthorized, "unauthenticated", "Sign in to continue.")
			return
		}
		if err != nil {
			logs.Errors.Error().Err(err).Msg("resolve dashboard session")
			writeAuthError(w, http.StatusInternalServerError, "session_unavailable", "Could not validate the dashboard session.")
			return
		}
		if renewedUntil != nil {
			setDashboardCookie(w, token, *renewedUntil, cookieSecure)
		}
		ctx := auth.WithAccount(r.Context(), account)
		ctx = tenant.WithID(ctx, tenant.ID(account.TenantID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// decodeJSON bounds request size, rejects unknown fields, and permits only one JSON value.
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return ensureJSONEnd(decoder)
}

// ensureJSONEnd rejects trailing values or malformed data after the decoded request object.
func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

// writeJSON emits a JSON response with an explicit status and content type.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeAuthError emits the standard auth error envelope without exposing internal details.
func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	var response authErrorResponse
	response.Error.Code = code
	response.Error.Message = message
	writeJSON(w, status, response)
}

// setDashboardCookie stores the opaque session token in a browser-restricted cookie.
func setDashboardCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: dashboardSessionCookie, Value: token, Path: "/", Expires: expiresAt,
		MaxAge: max(1, int(time.Until(expiresAt).Seconds())), HttpOnly: true,
		Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

// clearDashboardCookie expires the browser's dashboard session cookie.
func clearDashboardCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: dashboardSessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

// clearOAuthCookie expires one short-lived OAuth state cookie at its original path.
func clearOAuthCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/v1/auth/github", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

// readDashboardCookie returns a non-empty dashboard session token from the request.
func readDashboardCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(dashboardSessionCookie)
	if err != nil || cookie.Value == "" {
		return "", errors.New("dashboard session cookie is missing")
	}
	return cookie.Value, nil
}
