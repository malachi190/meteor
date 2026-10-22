package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"

	"github.com/joho/godotenv"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

const (
	defaultPort            = "8080"
	defaultErrorLogPath    = "logs/meteor-errors.log"
	defaultPublicBaseURL   = "http://localhost:8080"
	defaultDashboardOrigin = "http://localhost:3000"
)

// Config contains the settings the API needs after startup configuration is loaded.
type Config struct {
	Port               string `koanf:"port"`
	DatabaseURL        string `koanf:"database_url"`
	RedisURL           string `koanf:"redis_url"`
	ErrorLogPath       string `koanf:"error_log_path"`
	PublicBaseURL      string `koanf:"public_base_url"`
	DashboardOrigin    string `koanf:"dashboard_origin"`
	ResendAPIKey       string `koanf:"resend_api_key"`
	ResendFrom         string `koanf:"resend_from"`
	CookieSecure       bool   `koanf:"cookie_secure"`
	GitHubClientID     string `koanf:"github_client_id"`
	GitHubClientSecret string `koanf:"github_client_secret"`
	GitHubCallbackURL  string `koanf:"github_callback_url"`
	GitHubStateSecret  string `koanf:"github_state_secret"`
}

// Load reads a local .env file when present, then loads environment values into Config.
func Load() (Config, error) {
	cfg := Config{
		Port: defaultPort, ErrorLogPath: defaultErrorLogPath,
		PublicBaseURL: defaultPublicBaseURL, DashboardOrigin: defaultDashboardOrigin,
	}

	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return cfg, fmt.Errorf("load .env file: %w", err)
	}

	k := koanf.New(".")
	k.Set("port", defaultPort)
	k.Set("error_log_path", defaultErrorLogPath)
	k.Set("public_base_url", defaultPublicBaseURL)
	k.Set("dashboard_origin", defaultDashboardOrigin)
	provider := env.Provider("", ".", func(key string) string {
		switch strings.ToUpper(key) {
		case "PORT", "DATABASE_URL", "REDIS_URL", "ERROR_LOG_PATH", "PUBLIC_BASE_URL", "DASHBOARD_ORIGIN", "RESEND_API_KEY", "RESEND_FROM", "COOKIE_SECURE", "GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET", "GITHUB_CALLBACK_URL", "GITHUB_STATE_SECRET":
			return strings.ToLower(key)
		default:
			return ""
		}
	})
	if err := k.Load(provider, nil); err != nil {
		return cfg, fmt.Errorf("load environment configuration: %w", err)
	}
	if err := k.Unmarshal("", &cfg); err != nil {
		return cfg, fmt.Errorf("decode application configuration: %w", err)
	}
	if strings.TrimSpace(cfg.ErrorLogPath) == "" {
		cfg.ErrorLogPath = defaultErrorLogPath
	}

	return cfg, nil
}

// Validate checks settings that are required before the API can start.
func (c Config) Validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL is required")
	}
	if strings.TrimSpace(c.RedisURL) == "" {
		return errors.New("REDIS_URL is required")
	}
	if strings.TrimSpace(c.Port) == "" {
		return errors.New("PORT cannot be empty")
	}
	if strings.TrimSpace(c.ErrorLogPath) == "" {
		return errors.New("ERROR_LOG_PATH cannot be empty")
	}
	if err := validateOrigin("PUBLIC_BASE_URL", c.PublicBaseURL); err != nil {
		return err
	}
	if err := validateDashboardOrigin(c.DashboardOrigin); err != nil {
		return err
	}
	if (strings.TrimSpace(c.ResendAPIKey) == "") != (strings.TrimSpace(c.ResendFrom) == "") {
		return errors.New("RESEND_API_KEY and RESEND_FROM must be configured together")
	}
	githubFields := []string{c.GitHubClientID, c.GitHubClientSecret, c.GitHubCallbackURL, c.GitHubStateSecret}
	githubConfigured := 0
	for _, value := range githubFields {
		if strings.TrimSpace(value) != "" {
			githubConfigured++
		}
	}
	if githubConfigured != 0 && githubConfigured != len(githubFields) {
		return errors.New("GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, GITHUB_CALLBACK_URL, and GITHUB_STATE_SECRET must be configured together")
	}
	if len(c.GitHubStateSecret) > 0 && len(c.GitHubStateSecret) < 32 {
		return errors.New("GITHUB_STATE_SECRET must be at least 32 characters")
	}
	return nil
}

// validateDashboardOrigin requires one exact HTTP origin for credentialed dashboard requests.
func validateDashboardOrigin(origin string) error {
	return validateOrigin("DASHBOARD_ORIGIN", origin)
}

// validateOrigin requires an exact HTTP origin without a path, query, or fragment.
func validateOrigin(name, origin string) error {
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must be an http(s) origin without a path, query, or fragment", name)
	}
	return nil
}
