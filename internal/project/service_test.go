package project

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// projectTestRepository records inputs to verify service validation and secret handling.
type projectTestRepository struct {
	createProjectName string
	createProject     Project
	createProjectErr  error
	createdKeyPrefix  string
	createdKeyHash    []byte
	createdKeyEnv     string
	createdKey        APIKey
}

// CreateProject captures the validated name and returns the configured project.
func (r *projectTestRepository) CreateProject(_ context.Context, name string) (Project, error) {
	r.createProjectName = name
	return r.createProject, r.createProjectErr
}

// ListProjects returns no rows because list behavior is not part of these service tests.
func (r *projectTestRepository) ListProjects(context.Context) ([]Project, error) { return nil, nil }

// GetProject returns no row because lookup behavior is not part of these service tests.
func (r *projectTestRepository) GetProject(context.Context, string) (Project, error) {
	return Project{}, nil
}

// CreateAPIKey captures the stored hash and returns metadata without a raw secret.
func (r *projectTestRepository) CreateAPIKey(_ context.Context, projectID, prefix string, hash []byte, environment string) (APIKey, error) {
	r.createdKeyPrefix = prefix
	r.createdKeyHash = append([]byte(nil), hash...)
	r.createdKeyEnv = environment
	r.createdKey = APIKey{ID: "33333333-3333-4333-8333-333333333333", ProjectID: projectID, Prefix: prefix, Environment: environment, CreatedAt: time.Now()}
	return r.createdKey, nil
}

// ListAPIKeys returns no rows because listing behavior is not part of these service tests.
func (r *projectTestRepository) ListAPIKeys(context.Context, string) ([]APIKey, error) {
	return nil, nil
}

// RevokeAPIKey is a no-op because revocation persistence is tested at the repository boundary.
func (r *projectTestRepository) RevokeAPIKey(context.Context, string, string) error { return nil }

// TestCreateProjectTrimsNamesAndRejectsEmptyNames checks project-name validation before persistence.
func TestCreateProjectTrimsNamesAndRejectsEmptyNames(t *testing.T) {
	repository := &projectTestRepository{createProject: Project{Name: "Meteor"}}
	service := NewService(repository)
	if _, err := service.CreateProject(context.Background(), "  Meteor  "); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if repository.createProjectName != "Meteor" {
		t.Fatalf("stored project name = %q, want trimmed value", repository.createProjectName)
	}
	if _, err := service.CreateProject(context.Background(), "  "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateProject(empty) error = %v, want ErrInvalidInput", err)
	}
}

// TestCreateProjectMapsDuplicateNamesToConflict keeps PostgreSQL uniqueness details out of the API.
func TestCreateProjectMapsDuplicateNamesToConflict(t *testing.T) {
	repository := &projectTestRepository{createProjectErr: &pgconn.PgError{Code: "23505"}}
	service := NewService(repository)
	if _, err := service.CreateProject(context.Background(), "Existing project"); !errors.Is(err, ErrConflict) {
		t.Fatalf("CreateProject(duplicate) error = %v, want ErrConflict", err)
	}
}

// TestCreateAPIKeyReturnsSecretOnceAndPersistsOnlyHash protects the one-time secret boundary.
func TestCreateAPIKeyReturnsSecretOnceAndPersistsOnlyHash(t *testing.T) {
	repository := &projectTestRepository{}
	service := NewService(repository)
	key, err := service.CreateAPIKey(context.Background(), "22222222-2222-4222-8222-222222222222", "test")
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if !strings.HasPrefix(key.Secret, "sk_test_") || !strings.HasPrefix(key.Prefix, "sk_test_") {
		t.Fatalf("key secret or prefix has wrong environment prefix: %+v", key)
	}
	hash := sha256.Sum256([]byte(key.Secret))
	if string(hash[:]) != string(repository.createdKeyHash) {
		t.Fatal("repository did not receive the SHA-256 hash of the returned secret")
	}
	if key.Secret == repository.createdKeyPrefix || repository.createdKeyEnv != "test" {
		t.Fatal("raw secret was stored as a prefix or wrong environment was persisted")
	}
}

// TestCreateAPIKeyRejectsInvalidEnvironment prevents keys outside test and live modes.
func TestCreateAPIKeyRejectsInvalidEnvironment(t *testing.T) {
	service := NewService(&projectTestRepository{})
	if _, err := service.CreateAPIKey(context.Background(), "22222222-2222-4222-8222-222222222222", "staging"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateAPIKey(staging) error = %v, want ErrInvalidInput", err)
	}
}

// TestCreateAPIKeyUsesLivePrefix checks live keys receive an environment-specific public prefix.
func TestCreateAPIKeyUsesLivePrefix(t *testing.T) {
	service := NewService(&projectTestRepository{})
	key, err := service.CreateAPIKey(context.Background(), "22222222-2222-4222-8222-222222222222", "live")
	if err != nil {
		t.Fatalf("CreateAPIKey(live) error = %v", err)
	}
	if !strings.HasPrefix(key.Secret, "sk_live_") || !strings.HasPrefix(key.Prefix, "sk_live_") {
		t.Fatalf("live key prefix = %q, want sk_live_", key.Prefix)
	}
}
