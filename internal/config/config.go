package config

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/joho/godotenv"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

const (
	defaultPort         = "8080"
	defaultErrorLogPath = "logs/meteor-errors.log"
)

// Config contains the settings the API needs after startup configuration is loaded.
type Config struct {
	Port         string `koanf:"port"`
	DatabaseURL  string `koanf:"database_url"`
	ErrorLogPath string `koanf:"error_log_path"`
}

// Load reads a local .env file when present, then loads environment values into Config.
func Load() (Config, error) {
	cfg := Config{Port: defaultPort, ErrorLogPath: defaultErrorLogPath}

	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return cfg, fmt.Errorf("load .env file: %w", err)
	}

	k := koanf.New(".")
	k.Set("port", defaultPort)
	k.Set("error_log_path", defaultErrorLogPath)
	provider := env.Provider("", ".", func(key string) string {
		switch strings.ToUpper(key) {
		case "PORT", "DATABASE_URL", "ERROR_LOG_PATH":
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
	if strings.TrimSpace(c.Port) == "" {
		return errors.New("PORT cannot be empty")
	}
	if strings.TrimSpace(c.ErrorLogPath) == "" {
		return errors.New("ERROR_LOG_PATH cannot be empty")
	}
	return nil
}
