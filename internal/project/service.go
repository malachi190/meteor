// Package project contains tenant-scoped project and project API-key use cases.
package project

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidInput = errors.New("invalid project or API key input")
	ErrNotFound     = errors.New("project or API key not found")
	ErrConflict     = errors.New("project or API key conflicts with an existing record")
)

// Project is the dashboard-safe representation of a tenant-owned project.
type Project struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKey contains metadata for a project key and includes its secret only at creation time.
type APIKey struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Prefix      string     `json:"prefix"`
	Environment string     `json:"environment"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	Secret      string     `json:"secret,omitempty"`
}

// Repository defines the tenant-scoped persistence needed by project use cases.
type Repository interface {
	CreateProject(context.Context, string) (Project, error)
	ListProjects(context.Context) ([]Project, error)
	GetProject(context.Context, string) (Project, error)
	CreateAPIKey(context.Context, string, string, []byte, string) (APIKey, error)
	ListAPIKeys(context.Context, string) ([]APIKey, error)
	RevokeAPIKey(context.Context, string, string) error
}

// Service validates project and key operations and handles one-time secret generation.
type Service struct {
	repository Repository
}

// NewService wires project use cases to their tenant-scoped repository.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// CreateProject validates a project name before persisting it for the authenticated tenant.
func (s *Service) CreateProject(ctx context.Context, name string) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return Project{}, ErrInvalidInput
	}
	created, err := s.repository.CreateProject(ctx, name)
	if isUniqueViolation(err) {
		return Project{}, ErrConflict
	}
	return created, err
}

// ListProjects returns the signed-in tenant's projects.
func (s *Service) ListProjects(ctx context.Context) ([]Project, error) {
	return s.repository.ListProjects(ctx)
}

// GetProject returns one project after repository tenant scoping has been applied.
func (s *Service) GetProject(ctx context.Context, projectID string) (Project, error) {
	if !looksLikeUUID(projectID) {
		return Project{}, ErrInvalidInput
	}
	return s.repository.GetProject(ctx, projectID)
}

// CreateAPIKey generates an environment-scoped secret and persists only its SHA-256 hash.
func (s *Service) CreateAPIKey(ctx context.Context, projectID, environment string) (APIKey, error) {
	if !looksLikeUUID(projectID) || (environment != "test" && environment != "live") {
		return APIKey{}, ErrInvalidInput
	}
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return APIKey{}, fmt.Errorf("generate project API key: %w", err)
	}
	secret := "sk_" + environment + "_" + base64.RawURLEncoding.EncodeToString(secretBytes)
	prefix := secret[:len("sk_"+environment+"_")+8]
	hash := sha256.Sum256([]byte(secret))
	created, err := s.repository.CreateAPIKey(ctx, projectID, prefix, hash[:], environment)
	if isUniqueViolation(err) {
		return APIKey{}, ErrConflict
	}
	if err != nil {
		return APIKey{}, err
	}
	created.Secret = secret
	return created, nil
}

// ListAPIKeys returns metadata and visible prefixes without any previously issued raw secrets.
func (s *Service) ListAPIKeys(ctx context.Context, projectID string) ([]APIKey, error) {
	if !looksLikeUUID(projectID) {
		return nil, ErrInvalidInput
	}
	return s.repository.ListAPIKeys(ctx, projectID)
}

// RevokeAPIKey revokes one key only when it belongs to the requested tenant-owned project.
func (s *Service) RevokeAPIKey(ctx context.Context, projectID, keyID string) error {
	if !looksLikeUUID(projectID) || !looksLikeUUID(keyID) {
		return ErrInvalidInput
	}
	return s.repository.RevokeAPIKey(ctx, projectID, keyID)
}

// looksLikeUUID rejects malformed IDs before they reach pgx parsing or database queries.
func looksLikeUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

// isUniqueViolation recognizes PostgreSQL duplicate-key errors without coupling the service to pgx.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
