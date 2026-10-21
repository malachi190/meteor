package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrGitHubUnavailable = errors.New("GitHub sign-in is not configured")
	ErrGitHubState       = errors.New("GitHub sign-in state is invalid or expired")
	ErrGitHubIdentity    = errors.New("GitHub did not provide a verified email")
	ErrWorkspaceRequired = errors.New("workspace name is required for GitHub signup")
)

// GitHubSettings contains the OAuth client and signed-state configuration.
type GitHubSettings struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	StateSecret  string
}

// githubProvider holds the client and secret needed to complete GitHub OAuth requests.
type githubProvider struct {
	settings GitHubSettings
	client   *http.Client
}

// ConfigureGitHub installs the OAuth provider after startup configuration is loaded.
func (s *Service) ConfigureGitHub(settings GitHubSettings) {
	if settings.ClientID == "" || settings.ClientSecret == "" || settings.CallbackURL == "" || len(settings.StateSecret) < 32 {
		s.github = nil
		return
	}
	s.github = &githubProvider{settings: settings, client: &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// StartGitHub builds an authorization URL and signs state plus the optional workspace name.
func (s *Service) StartGitHub(tenantName string) (authorizationURL, state, signedWorkspace string, err error) {
	if s.github == nil {
		return "", "", "", ErrGitHubUnavailable
	}
	tenantName = strings.TrimSpace(tenantName)
	if len(tenantName) > 120 {
		return "", "", "", ErrInvalidInput
	}
	var rawState [32]byte
	if _, err := rand.Read(rawState[:]); err != nil {
		return "", "", "", fmt.Errorf("create OAuth state: %w", err)
	}
	state = base64.RawURLEncoding.EncodeToString(rawState[:])
	signedWorkspace, err = s.github.signWorkspace(tenantName, state)
	if err != nil {
		return "", "", "", err
	}
	query := url.Values{
		"client_id":    {s.github.settings.ClientID},
		"redirect_uri": {s.github.settings.CallbackURL},
		"scope":        {"read:user user:email"},
		"state":        {state},
	}
	return "https://github.com/login/oauth/authorize?" + query.Encode(), state, signedWorkspace, nil
}

// GitHubCallback validates OAuth state and identity before linking or creating an account.
func (s *Service) GitHubCallback(ctx context.Context, code, receivedState, cookieState, signedWorkspace string) (Account, string, time.Time, error) {
	if s.github == nil {
		return Account{}, "", time.Time{}, ErrGitHubUnavailable
	}
	if code == "" || len(code) > 2048 || len(receivedState) != 43 || len(cookieState) != 43 || subtle.ConstantTimeCompare([]byte(receivedState), []byte(cookieState)) != 1 {
		return Account{}, "", time.Time{}, ErrGitHubState
	}
	tenantName, ok := s.github.verifyWorkspace(signedWorkspace, receivedState)
	if !ok {
		return Account{}, "", time.Time{}, ErrGitHubState
	}
	subject, email, err := s.github.fetchVerifiedIdentity(ctx, code)
	if err != nil {
		return Account{}, "", time.Time{}, err
	}
	account, err := s.repository.ResolveGitHubIdentity(ctx, subject)
	if err == nil {
		return s.createOAuthSession(ctx, account)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Account{}, "", time.Time{}, fmt.Errorf("resolve GitHub identity: %w", err)
	}

	login, err := s.repository.FindLogin(ctx, normalizeEmail(email))
	if err == nil {
		return s.linkGitHubAndCreateSession(ctx, login.Account, subject)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Account{}, "", time.Time{}, fmt.Errorf("resolve account by verified GitHub email: %w", err)
	}
	if tenantName == "" {
		return Account{}, "", time.Time{}, ErrWorkspaceRequired
	}

	account, err = s.repository.CreateGitHubAccount(ctx, tenantName, email, normalizeEmail(email), subject)
	if err != nil {
		return Account{}, "", time.Time{}, fmt.Errorf("create GitHub signup: %w", err)
	}
	plainToken, expiresAt, err := s.createSession(ctx, account)
	if err != nil {
		return Account{}, "", time.Time{}, err
	}
	return account, plainToken, expiresAt, nil
}

// createOAuthSession replaces the user's existing session after successful OAuth sign-in.
func (s *Service) createOAuthSession(ctx context.Context, account Account) (Account, string, time.Time, error) {
	plainToken, expiresAt, err := s.createSession(ctx, account)
	if err != nil {
		return Account{}, "", time.Time{}, err
	}
	return account, plainToken, expiresAt, nil
}

// linkGitHubAndCreateSession attaches the verified GitHub identity before creating a session.
func (s *Service) linkGitHubAndCreateSession(ctx context.Context, account Account, subject string) (Account, string, time.Time, error) {
	linked, err := s.repository.LinkGitHubIdentity(ctx, account, subject)
	if err != nil {
		return Account{}, "", time.Time{}, fmt.Errorf("link GitHub identity: %w", err)
	}
	return s.createOAuthSession(ctx, linked)
}

// signWorkspace binds a new-workspace name to one OAuth state value using an HMAC.
func (g *githubProvider) signWorkspace(tenantName, state string) (string, error) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(tenantName + "\x00" + state))
	mac := hmac.New(sha256.New, []byte(g.settings.StateSecret))
	if _, err := mac.Write([]byte(payload)); err != nil {
		return "", err
	}
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// verifyWorkspace returns a workspace name only when its signature matches the OAuth state.
func (g *githubProvider) verifyWorkspace(value, state string) (string, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return "", false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(g.settings.StateSecret))
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	fields := strings.SplitN(string(payload), "\x00", 2)
	if len(fields) != 2 || len(fields[0]) > 120 || subtle.ConstantTimeCompare([]byte(fields[1]), []byte(state)) != 1 {
		return "", false
	}
	return fields[0], true
}

// fetchVerifiedIdentity exchanges the code and accepts only GitHub's verified primary email.
func (g *githubProvider) fetchVerifiedIdentity(ctx context.Context, code string) (string, string, error) {
	form := url.Values{
		"client_id":     {g.settings.ClientID},
		"client_secret": {g.settings.ClientSecret},
		"code":          {code},
		"redirect_uri":  {g.settings.CallbackURL},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("create GitHub token request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := g.client.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("exchange GitHub OAuth code: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", errors.New("GitHub OAuth token exchange failed")
	}
	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&token); err != nil || token.AccessToken == "" || !strings.EqualFold(token.TokenType, "bearer") {
		return "", "", errors.New("GitHub OAuth token response is invalid")
	}

	userRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", "", fmt.Errorf("create GitHub user request: %w", err)
	}
	setGitHubHeaders(userRequest, token.AccessToken)
	userResponse, err := g.client.Do(userRequest)
	if err != nil {
		return "", "", fmt.Errorf("fetch GitHub user: %w", err)
	}
	defer userResponse.Body.Close()
	if userResponse.StatusCode != http.StatusOK {
		return "", "", errors.New("GitHub user lookup failed")
	}
	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(userResponse.Body, 1<<20)).Decode(&user); err != nil || user.ID <= 0 {
		return "", "", errors.New("GitHub user response is invalid")
	}

	emailRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", "", fmt.Errorf("create GitHub email request: %w", err)
	}
	setGitHubHeaders(emailRequest, token.AccessToken)
	emailResponse, err := g.client.Do(emailRequest)
	if err != nil {
		return "", "", fmt.Errorf("fetch GitHub emails: %w", err)
	}
	defer emailResponse.Body.Close()
	if emailResponse.StatusCode != http.StatusOK {
		return "", "", errors.New("GitHub email lookup failed")
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(io.LimitReader(emailResponse.Body, 1<<20)).Decode(&emails); err != nil {
		return "", "", errors.New("GitHub email response is invalid")
	}
	for _, email := range emails {
		if email.Primary && email.Verified && validEmail(email.Email) {
			return fmt.Sprint(user.ID), email.Email, nil
		}
	}
	return "", "", ErrGitHubIdentity
}

// setGitHubHeaders adds the API version and bearer token required by GitHub endpoints.
func setGitHubHeaders(request *http.Request, accessToken string) {
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "Meteor-API")
}
