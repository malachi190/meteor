package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 19 * 1024
	argonIterations  = 2
	argonParallelism = 1
	argonSaltBytes   = 16
	argonKeyBytes    = 32
	verificationTTL  = 24 * time.Hour
	passwordResetTTL = time.Hour
	sessionTTL       = 48 * time.Hour
	sessionRenewal   = 24 * time.Hour
)

var (
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrInvalidVerification   = errors.New("verification link is invalid or expired")
	ErrInvalidInput          = errors.New("invalid signup input")
	ErrVerificationDelivery  = errors.New("verification email could not be delivered")
	ErrInvalidPasswordReset  = errors.New("password reset link is invalid or expired")
	ErrPasswordResetDelivery = errors.New("password reset email could not be delivered")
)

// Account is the dashboard identity and tenant scope attached to an authenticated session.
type Account struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// LoginRecord combines the account fields needed for password authentication.
type LoginRecord struct {
	Account
	PasswordHash  string
	EmailVerified bool
}

// Repository defines the persistent auth operations used by the service layer.
type Repository interface {
	CreateSignup(context.Context, string, string, string, string, []byte, time.Time) (Account, error)
	FindLogin(context.Context, string) (LoginRecord, error)
	CreateVerificationToken(context.Context, string, string, []byte, time.Time) error
	RecordLogin(context.Context, string, string) error
	VerifyEmail(context.Context, []byte) (Account, error)
	ResolveGitHubIdentity(context.Context, string) (Account, error)
	LinkGitHubIdentity(context.Context, Account, string) (Account, error)
	CreateGitHubAccount(context.Context, string, string, string, string) (Account, error)
	CreatePasswordResetToken(context.Context, string, string, []byte, time.Time) error
	ResetPassword(context.Context, []byte, string, func(context.Context, string, string) error) error
}

// SessionStore holds active dashboard sessions outside PostgreSQL.
type SessionStore interface {
	Create(context.Context, Account, []byte) (time.Time, error)
	Resolve(context.Context, string, string, []byte) (Account, *time.Time, error)
	Revoke(context.Context, Account, []byte) error
	RevokeUser(context.Context, string, string) error
}

// VerificationMailer delivers account verification and password recovery links.
type VerificationMailer interface {
	SendVerification(context.Context, string, string) error
	SendPasswordReset(context.Context, string, string) error
}

// Service applies authentication policy while delegating persistence, sessions, and email.
type Service struct {
	repository Repository
	sessions   SessionStore
	mailer     VerificationMailer
	baseURL    string
	now        func() time.Time
	github     *githubProvider
}

// NewService wires auth dependencies and normalizes the public link base URL.
func NewService(repository Repository, sessions SessionStore, mailer VerificationMailer, publicBaseURL string) *Service {
	return &Service{repository: repository, sessions: sessions, mailer: mailer, baseURL: strings.TrimRight(publicBaseURL, "/"), now: time.Now}
}

// Signup creates a tenant, its first dashboard user, and a one-time verification token atomically.
func (s *Service) Signup(ctx context.Context, tenantName, email, password string) error {
	tenantName = strings.TrimSpace(tenantName)
	email = strings.TrimSpace(email)
	normalizedEmail := normalizeEmail(email)
	if tenantName == "" || len(tenantName) > 120 || !validEmail(email) || len(password) < 12 || len(password) > 128 {
		return ErrInvalidInput
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("hash signup password: %w", err)
	}
	verificationToken, verificationHash, err := newToken()
	if err != nil {
		return fmt.Errorf("create verification token: %w", err)
	}
	account, err := s.repository.CreateSignup(ctx, tenantName, email, normalizedEmail, passwordHash, verificationHash, s.now().Add(verificationTTL))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "dashboard_users_normalized_email_unique" {
			return s.resendVerification(ctx, email, normalizedEmail)
		}
		return err
	}
	return s.sendVerification(ctx, account.Email, verificationToken)
}

// ResendVerification sends a fresh token only when the address belongs to an unverified account.
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	return s.resendVerification(ctx, strings.TrimSpace(email), normalizeEmail(email))
}

// resendVerification avoids account enumeration while replacing prior verification tokens.
func (s *Service) resendVerification(ctx context.Context, email, normalizedEmail string) error {
	login, err := s.repository.FindLogin(ctx, normalizedEmail)
	if errors.Is(err, pgx.ErrNoRows) || login.EmailVerified || login.PasswordHash == "" {
		return nil
	}
	if err != nil {
		return fmt.Errorf("look up account for verification resend: %w", err)
	}
	token, hash, err := newToken()
	if err != nil {
		return fmt.Errorf("create verification token: %w", err)
	}
	if err := s.repository.CreateVerificationToken(ctx, login.TenantID, login.ID, hash, s.now().Add(verificationTTL)); err != nil {
		return fmt.Errorf("store verification token: %w", err)
	}
	if err := s.sendVerification(ctx, email, token); err != nil {
		return err
	}
	return nil
}

// sendVerification builds the public confirmation URL and hands delivery to the mail adapter.
func (s *Service) sendVerification(ctx context.Context, email, token string) error {
	if s.mailer == nil {
		return ErrVerificationDelivery
	}
	link := s.baseURL + "/v1/auth/verify-email?token=" + url.QueryEscape(token)
	if err := s.mailer.SendVerification(ctx, email, link); err != nil {
		return fmt.Errorf("%w: %v", ErrVerificationDelivery, err)
	}
	return nil
}

// RequestPasswordReset gives the account owner a one-time link without confirming whether the email exists.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalizedEmail := normalizeEmail(email)
	if !validEmail(strings.TrimSpace(email)) {
		return nil
	}
	login, err := s.repository.FindLogin(ctx, normalizedEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = verifyPassword(dummyPasswordHash, "")
		return nil
	}
	if err != nil {
		return fmt.Errorf("look up account for password reset: %w", err)
	}
	if !login.EmailVerified || login.PasswordHash == "" {
		_ = verifyPassword(dummyPasswordHash, "")
		return nil
	}
	plainToken, tokenHash, err := newToken()
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	expiresAt := s.now().Add(passwordResetTTL)
	if err := s.repository.CreatePasswordResetToken(ctx, login.TenantID, login.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("store password reset token: %w", err)
	}
	if s.mailer == nil {
		return ErrPasswordResetDelivery
	}
	link := s.baseURL + "/v1/auth/password-reset?token=" + url.QueryEscape(plainToken)
	if err := s.mailer.SendPasswordReset(ctx, login.Email, link); err != nil {
		return fmt.Errorf("%w: %v", ErrPasswordResetDelivery, err)
	}
	return nil
}

// ResetPassword changes the password once and invalidates the token and every active session.
func (s *Service) ResetPassword(ctx context.Context, plainToken, password string) error {
	if len(password) < 12 || len(password) > 128 {
		return ErrInvalidInput
	}
	_, tokenHash, err := hashToken(plainToken)
	if err != nil {
		return ErrInvalidPasswordReset
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	if err := s.repository.ResetPassword(ctx, tokenHash, passwordHash, func(ctx context.Context, tenantID, userID string) error {
		return s.sessions.RevokeUser(ctx, tenantID, userID)
	}); err != nil {
		if errors.Is(err, ErrInvalidPasswordReset) {
			return ErrInvalidPasswordReset
		}
		return fmt.Errorf("reset dashboard password: %w", err)
	}
	return nil
}

// Login checks the verified account credentials, records activity, and creates an opaque session.
func (s *Service) Login(ctx context.Context, email, password string) (Account, string, time.Time, error) {
	login, err := s.repository.FindLogin(ctx, normalizeEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		_ = verifyPassword(dummyPasswordHash, password)
		return Account{}, "", time.Time{}, ErrInvalidCredentials
	}
	if err != nil {
		return Account{}, "", time.Time{}, fmt.Errorf("look up dashboard login: %w", err)
	}
	if login.PasswordHash == "" {
		_ = verifyPassword(dummyPasswordHash, password)
		return Account{}, "", time.Time{}, ErrInvalidCredentials
	}
	passwordMatches := verifyPassword(login.PasswordHash, password)
	if !login.EmailVerified || !passwordMatches {
		return Account{}, "", time.Time{}, ErrInvalidCredentials
	}
	plainToken, tokenHash, err := newSessionToken(login.Account)
	if err != nil {
		return Account{}, "", time.Time{}, fmt.Errorf("create dashboard session token: %w", err)
	}
	if err := s.repository.RecordLogin(ctx, login.TenantID, login.ID); err != nil {
		return Account{}, "", time.Time{}, fmt.Errorf("record dashboard login: %w", err)
	}
	expiresAt, err := s.sessions.Create(ctx, login.Account, tokenHash)
	if err != nil {
		return Account{}, "", time.Time{}, fmt.Errorf("create dashboard session: %w", err)
	}
	return login.Account, plainToken, expiresAt, nil
}

// VerifyEmail consumes a one-time token and starts a session for the newly verified account.
func (s *Service) VerifyEmail(ctx context.Context, plainToken string) (Account, string, time.Time, error) {
	if plainToken == "" || len(plainToken) > 128 {
		return Account{}, "", time.Time{}, ErrInvalidVerification
	}
	_, verificationHash, err := hashToken(plainToken)
	if err != nil {
		return Account{}, "", time.Time{}, ErrInvalidVerification
	}
	account, err := s.repository.VerifyEmail(ctx, verificationHash)
	if err != nil {
		if errors.Is(err, ErrInvalidVerification) || errors.Is(err, pgx.ErrNoRows) {
			return Account{}, "", time.Time{}, ErrInvalidVerification
		}
		return Account{}, "", time.Time{}, fmt.Errorf("verify dashboard email: %w", err)
	}
	sessionToken, expiresAt, err := s.createSession(ctx, account)
	if err != nil {
		return Account{}, "", time.Time{}, err
	}
	return account, sessionToken, expiresAt, nil
}

// AuthenticateSession parses the cookie token and resolves its server-side session record.
func (s *Service) AuthenticateSession(ctx context.Context, plainToken string) (Account, *time.Time, error) {
	if plainToken == "" || len(plainToken) > 128 {
		return Account{}, nil, ErrInvalidCredentials
	}
	tenantID, userID, secret, err := parseSessionToken(plainToken)
	if err != nil {
		return Account{}, nil, ErrInvalidCredentials
	}
	_, tokenHash, err := hashToken(secret)
	if err != nil {
		return Account{}, nil, ErrInvalidCredentials
	}
	account, renewedUntil, err := s.sessions.Resolve(ctx, tenantID, userID, tokenHash)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return Account{}, nil, ErrInvalidCredentials
		}
		return Account{}, nil, fmt.Errorf("resolve dashboard session: %w", err)
	}
	return account, renewedUntil, nil
}

// Logout revokes the session represented by the supplied opaque token.
func (s *Service) Logout(ctx context.Context, account Account, plainToken string) error {
	_, _, secret, err := parseSessionToken(plainToken)
	if err != nil {
		return ErrInvalidCredentials
	}
	_, tokenHash, err := hashToken(secret)
	if err != nil {
		return ErrInvalidCredentials
	}
	return s.sessions.Revoke(ctx, account, tokenHash)
}

// InvalidateUserSession logs out the user's current session after a role or account change.
func (s *Service) InvalidateUserSession(ctx context.Context, tenantID, userID string) error {
	return s.sessions.RevokeUser(ctx, tenantID, userID)
}

// createSession creates the random bearer token and stores only its hash in the session backend.
func (s *Service) createSession(ctx context.Context, account Account) (string, time.Time, error) {
	plainToken, tokenHash, err := newSessionToken(account)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create dashboard session token: %w", err)
	}
	expiresAt, err := s.sessions.Create(ctx, account, tokenHash)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create dashboard session: %w", err)
	}
	return plainToken, expiresAt, nil
}

// normalizeEmail applies the canonical comparison form used by auth lookups.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validEmail accepts a single plain mailbox address within the schema's length limit.
func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email && len(email) <= 320
}

// newToken returns a URL-safe random token and the SHA-256 hash stored by repositories.
func newToken() (string, []byte, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", nil, err
	}
	plain := base64.RawURLEncoding.EncodeToString(raw[:])
	_, hash, err := hashToken(plain)
	return plain, hash, err
}

// newSessionToken binds a random secret to the tenant and user IDs needed for session lookup.
func newSessionToken(account Account) (string, []byte, error) {
	if !isUUID(account.TenantID) || !isUUID(account.ID) {
		return "", nil, errors.New("session account identifiers are invalid")
	}
	secret, tokenHash, err := newToken()
	if err != nil {
		return "", nil, err
	}
	return strings.ToLower(account.TenantID) + "." + strings.ToLower(account.ID) + "." + secret, tokenHash, nil
}

// parseSessionToken validates the token's public ID prefix and extracts its secret.
func parseSessionToken(token string) (tenantID, userID, secret string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || !isUUID(parts[0]) || !isUUID(parts[1]) {
		return "", "", "", errors.New("session token has an invalid format")
	}
	if _, _, err := hashToken(parts[2]); err != nil {
		return "", "", "", errors.New("session token secret has an invalid format")
	}
	return strings.ToLower(parts[0]), strings.ToLower(parts[1]), parts[2], nil
}

// isUUID checks the canonical UUID text shape before it is used in cache keys or SQL parameters.
func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, char := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
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

// hashToken validates the encoded token and returns its SHA-256 digest for storage or lookup.
func hashToken(token string) (string, []byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return "", nil, errors.New("token has an invalid format")
	}
	hash := sha256.Sum256(decoded)
	return token, hash[:], nil
}

// hashPassword creates an Argon2id password hash with the service's current work factors.
func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyBytes)
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonIterations, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
	return encoded, nil
}

var dummyPasswordHash = func() string {
	salt := make([]byte, argonSaltBytes)
	key := argon2.IDKey([]byte("meteor-invalid-account-password"), salt, argonIterations, argonMemory, argonParallelism, argonKeyBytes)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonIterations, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
}()

// verifyPassword parses the encoded Argon2id parameters and compares the derived key in constant time.
func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || !strings.HasPrefix(parts[2], "v=") {
		return false
	}
	version, err := strconv.ParseUint(strings.TrimPrefix(parts[2], "v="), 10, 32)
	if err != nil {
		return false
	}
	var memory, iterations, parallelism uint64
	for _, parameter := range strings.Split(parts[3], ",") {
		pair := strings.SplitN(parameter, "=", 2)
		if len(pair) != 2 {
			return false
		}
		value, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return false
		}
		switch pair[0] {
		case "m":
			memory = value
		case "t":
			iterations = value
		case "p":
			parallelism = value
		default:
			return false
		}
	}
	saltText, keyText := parts[4], parts[5]
	if version != 19 || memory < 19*1024 || memory > 256*1024 || iterations < 1 || iterations > 10 || parallelism < 1 || parallelism > 8 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(saltText)
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(keyText)
	if err != nil || len(expected) < 16 || len(expected) > 64 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, uint32(iterations), uint32(memory), uint8(parallelism), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
