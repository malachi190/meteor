package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// testRepository supplies controlled account and token behavior to auth service tests.
type testRepository struct {
	login             LoginRecord
	findErr           error
	createdAccount    Account
	createdPassword   string
	createdVerifyHash []byte
	createdVerifyAt   time.Time
	verifyErr         error
	resetErr          error
	resetHook         func(context.Context, string, string) error
	resetPasswordHash string
	resetTenantID     string
	resetUserID       string
	recordLoginCalls  int
}

// CreateSignup captures the inputs the service prepares before persisting a signup.
func (r *testRepository) CreateSignup(_ context.Context, tenantName, email, normalizedEmail, passwordHash string, tokenHash []byte, expiresAt time.Time) (Account, error) {
	r.createdAccount = Account{ID: "11111111-1111-4111-8111-111111111111", TenantID: "22222222-2222-4222-8222-222222222222", Email: email, Role: "owner"}
	r.createdPassword = passwordHash
	r.createdVerifyHash = append([]byte(nil), tokenHash...)
	r.createdVerifyAt = expiresAt
	if tenantName == "" || normalizedEmail != normalizeEmail(email) {
		return Account{}, errors.New("unexpected signup values")
	}
	return r.createdAccount, nil
}

// FindLogin returns the configured login fixture for a service test.
func (r *testRepository) FindLogin(context.Context, string) (LoginRecord, error) {
	return r.login, r.findErr
}

// CreateVerificationToken records only the hash and expiry passed to the fake repository.
func (r *testRepository) CreateVerificationToken(_ context.Context, _, _ string, hash []byte, expiry time.Time) error {
	r.createdVerifyHash = append([]byte(nil), hash...)
	r.createdVerifyAt = expiry
	return nil
}

// RecordLogin counts successful login activity updates without using a database.
func (r *testRepository) RecordLogin(context.Context, string, string) error {
	r.recordLoginCalls++
	return nil
}

// VerifyEmail returns the fixture account and configured token error.
func (r *testRepository) VerifyEmail(context.Context, []byte) (Account, error) {
	return r.createdAccount, r.verifyErr
}

// ResolveGitHubIdentity fails fast because these tests do not exercise provider linking.
func (r *testRepository) ResolveGitHubIdentity(context.Context, string) (Account, error) {
	return Account{}, errors.New("not used in this test")
}

// LinkGitHubIdentity fails fast because these tests do not exercise provider linking.
func (r *testRepository) LinkGitHubIdentity(context.Context, Account, string) (Account, error) {
	return Account{}, errors.New("not used in this test")
}

// CreateGitHubAccount fails fast because these tests do not exercise provider signup.
func (r *testRepository) CreateGitHubAccount(context.Context, string, string, string, string) (Account, error) {
	return Account{}, errors.New("not used in this test")
}

// CreatePasswordResetToken is an unused no-op for service tests focused on completed resets.
func (r *testRepository) CreatePasswordResetToken(context.Context, string, string, []byte, time.Time) error {
	return nil
}

// ResetPassword records the new hash and invokes the same revocation hook as the real repository.
func (r *testRepository) ResetPassword(_ context.Context, _ []byte, passwordHash string, revoke func(context.Context, string, string) error) error {
	r.resetHook = revoke
	r.resetPasswordHash = passwordHash
	r.resetTenantID, r.resetUserID = r.login.TenantID, r.login.ID
	if r.resetErr != nil {
		return r.resetErr
	}
	return revoke(context.Background(), r.resetTenantID, r.resetUserID)
}

// testSessions records the opaque-token hashes passed to the session boundary.
type testSessions struct {
	createdAccount Account
	createdHash    []byte
	resolveAccount Account
	resolveExpiry  *time.Time
	resolveErr     error
	revokedUser    string
}

// Create captures session inputs and returns a deterministic-enough future expiry.
func (s *testSessions) Create(_ context.Context, account Account, hash []byte) (time.Time, error) {
	s.createdAccount = account
	s.createdHash = append([]byte(nil), hash...)
	return time.Now().Add(time.Hour), nil
}

// Resolve returns the session result configured by the current test.
func (s *testSessions) Resolve(_ context.Context, _, _ string, _ []byte) (Account, *time.Time, error) {
	return s.resolveAccount, s.resolveExpiry, s.resolveErr
}

// Revoke is a no-op because service tests do not inspect single-session deletion.
func (s *testSessions) Revoke(context.Context, Account, []byte) error { return nil }

// RevokeUser captures the tenant and user whose sessions were invalidated.
func (s *testSessions) RevokeUser(_ context.Context, tenantID, userID string) error {
	s.revokedUser = tenantID + "/" + userID
	return nil
}

// testMailer captures generated links without sending email outside the test.
type testMailer struct {
	verificationLink string
	resetLink        string
	err              error
}

// SendVerification captures a generated link without making a network request.
func (m *testMailer) SendVerification(_ context.Context, _, link string) error {
	m.verificationLink = link
	return m.err
}

// SendPasswordReset captures a generated recovery link without making a network request.
func (m *testMailer) SendPasswordReset(_ context.Context, _, link string) error {
	m.resetLink = link
	return m.err
}

// TestSignupValidatesInputAndSendsOneTimeVerification ensures signup hashes credentials and delivers a verification link.
func TestSignupValidatesInputAndSendsOneTimeVerification(t *testing.T) {
	repository, mailer := &testRepository{}, &testMailer{}
	service := NewService(repository, &testSessions{}, mailer, "https://meteor.example/")

	if err := service.Signup(context.Background(), " Workspace ", "User@example.com", "a-long-password-123"); err != nil {
		t.Fatalf("Signup() error = %v", err)
	}
	if !strings.HasPrefix(repository.createdPassword, "$argon2id$") {
		t.Fatalf("stored password format = %q, want Argon2id", repository.createdPassword)
	}
	if len(repository.createdVerifyHash) != 32 || repository.createdVerifyAt.Before(time.Now()) {
		t.Fatalf("verification token hash/expiry not stored correctly")
	}
	if !strings.HasPrefix(mailer.verificationLink, "https://meteor.example/v1/auth/verify-email?token=") {
		t.Fatalf("verification link = %q", mailer.verificationLink)
	}
	if strings.Contains(repository.createdPassword, "a-long-password-123") {
		t.Fatal("plaintext password was stored")
	}

	if err := service.Signup(context.Background(), "", "not-an-email", "short"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Signup(invalid input) error = %v, want ErrInvalidInput", err)
	}
}

// TestLoginRequiresVerifiedEmailAndCreatesSession checks login's account gate and session handoff.
func TestLoginRequiresVerifiedEmailAndCreatesSession(t *testing.T) {
	passwordHash, err := hashPassword("a-long-password-123")
	if err != nil {
		t.Fatal(err)
	}
	account := Account{ID: "11111111-1111-4111-8111-111111111111", TenantID: "22222222-2222-4222-8222-222222222222", Email: "user@example.com", Role: "owner"}
	repository := &testRepository{login: LoginRecord{Account: account, PasswordHash: passwordHash, EmailVerified: true}}
	sessions := &testSessions{}
	service := NewService(repository, sessions, &testMailer{}, "http://localhost:8080")

	gotAccount, token, _, err := service.Login(context.Background(), " USER@example.com ", "a-long-password-123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if gotAccount != account || token == "" || len(sessions.createdHash) != 32 || repository.recordLoginCalls != 1 {
		t.Fatalf("Login() did not record login and create a hashed session")
	}

	repository.login.EmailVerified = false
	if _, _, _, err := service.Login(context.Background(), account.Email, "a-long-password-123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login(unverified account) error = %v, want ErrInvalidCredentials", err)
	}
}

// TestAuthenticateSessionRejectsMalformedTokens prevents malformed cookie values reaching Redis.
func TestAuthenticateSessionRejectsMalformedTokens(t *testing.T) {
	service := NewService(&testRepository{}, &testSessions{}, &testMailer{}, "http://localhost:8080")
	if _, _, err := service.AuthenticateSession(context.Background(), "not-a-session-token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("AuthenticateSession() error = %v, want ErrInvalidCredentials", err)
	}
}

// TestResetPasswordRevokesAllUserSessions checks password reset invokes the user-wide revocation hook.
func TestResetPasswordRevokesAllUserSessions(t *testing.T) {
	repository := &testRepository{login: LoginRecord{Account: Account{ID: "user-id", TenantID: "tenant-id"}}}
	sessions := &testSessions{}
	service := NewService(repository, sessions, &testMailer{}, "http://localhost:8080")
	token, _, err := newToken()
	if err != nil {
		t.Fatal(err)
	}

	if err := service.ResetPassword(context.Background(), token, "new-long-password-123"); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if sessions.revokedUser != "tenant-id/user-id" || !strings.HasPrefix(repository.resetPasswordHash, "$argon2id$") {
		// The fake repository calls the service hook only after its password update succeeds.
		t.Fatalf("ResetPassword() did not revoke the user's sessions: %q", sessions.revokedUser)
	}
}
