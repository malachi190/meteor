package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"meteor/internal/auth"
	"meteor/internal/database"
	"meteor/internal/database/queries"
	"meteor/internal/tenant"
)

// AuthRepository owns auth SQL and keeps tenant-owned changes inside RLS-scoped transactions.
type AuthRepository struct {
	pool *pgxpool.Pool
}

// NewAuthRepository binds authentication persistence to the shared PostgreSQL pool.
func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

// CreateSignup atomically inserts the tenant, initial user, and pending verification token under RLS.
func (r *AuthRepository) CreateSignup(ctx context.Context, tenantName, email, normalizedEmail, passwordHash string, verificationHash []byte, verificationExpiry time.Time) (auth.Account, error) {
	tenantUUID, err := newTenantUUID()
	if err != nil {
		return auth.Account{}, fmt.Errorf("generate signup tenant ID: %w", err)
	}
	ctx = tenant.WithID(ctx, tenant.ID(tenantUUID.String()))
	var account auth.Account
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.CreateTenant(ctx, queries.CreateTenantParams{ID: tenantUUID, Name: tenantName}); err != nil {
			return fmt.Errorf("insert signup tenant: %w", err)
		}
		user, err := q.CreateDashboardUser(ctx, queries.CreateDashboardUserParams{
			TenantID: tenantUUID, Email: email, NormalizedEmail: normalizedEmail,
			PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("insert first dashboard user: %w", err)
		}
		if err := q.CreateEmailVerificationToken(ctx, queries.CreateEmailVerificationTokenParams{
			TokenHash: verificationHash, TenantID: tenantUUID, DashboardUserID: user.ID, ExpiresAt: pgtype.Timestamptz{Time: verificationExpiry, Valid: true},
		}); err != nil {
			return fmt.Errorf("insert email verification token: %w", err)
		}
		account = accountFromUser(user.ID, user.TenantID, user.Email, user.Role)
		return nil
	})
	return account, err
}

// FindLogin uses the restricted pre-auth lookup, then loads account details under tenant RLS.
func (r *AuthRepository) FindLogin(ctx context.Context, normalizedEmail string) (auth.LoginRecord, error) {
	row, err := queries.New(r.pool).ResolveDashboardLogin(ctx, normalizedEmail)
	if err != nil {
		return auth.LoginRecord{}, err
	}
	ctx = tenant.WithID(ctx, tenant.ID(row.TenantID.String()))
	var account auth.Account
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		user, err := queries.New(tx).GetDashboardUserByTenantAndID(ctx, queries.GetDashboardUserByTenantAndIDParams{
			TenantID: row.TenantID, ID: row.DashboardUserID,
		})
		if err != nil {
			return fmt.Errorf("load dashboard user for login: %w", err)
		}
		account = accountFromUser(user.ID, user.TenantID, user.Email, user.Role)
		return nil
	})
	if err != nil {
		return auth.LoginRecord{}, err
	}
	return auth.LoginRecord{Account: account, PasswordHash: row.PasswordHash, EmailVerified: row.EmailVerifiedAt.Valid}, nil
}

// CreateVerificationToken replaces older pending tokens and inserts the new hash in one transaction.
func (r *AuthRepository) CreateVerificationToken(ctx context.Context, tenantID, userID string, tokenHash []byte, expiresAt time.Time) error {
	tenantUUID, userUUID, err := authUUIDs(tenantID, userID)
	if err != nil {
		return err
	}
	ctx = tenant.WithID(ctx, tenant.ID(tenantID))
	return database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.ConsumePriorEmailVerificationTokens(ctx, queries.ConsumePriorEmailVerificationTokensParams{
			TenantID: tenantUUID, DashboardUserID: userUUID,
		}); err != nil {
			return fmt.Errorf("consume prior email verification tokens: %w", err)
		}
		if err := q.CreateEmailVerificationToken(ctx, queries.CreateEmailVerificationTokenParams{
			TokenHash: tokenHash, TenantID: tenantUUID, DashboardUserID: userUUID,
			ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		}); err != nil {
			return fmt.Errorf("insert email verification token: %w", err)
		}
		return nil
	})
}

// CreatePasswordResetToken consumes prior reset links before storing the replacement token hash.
func (r *AuthRepository) CreatePasswordResetToken(ctx context.Context, tenantID, userID string, tokenHash []byte, expiresAt time.Time) error {
	tenantUUID, userUUID, err := authUUIDs(tenantID, userID)
	if err != nil {
		return err
	}
	ctx = tenant.WithID(ctx, tenant.ID(tenantID))
	return database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.ConsumePriorPasswordResetTokens(ctx, queries.ConsumePriorPasswordResetTokensParams{
			TenantID: tenantUUID, DashboardUserID: userUUID,
		}); err != nil {
			return fmt.Errorf("consume prior password reset tokens: %w", err)
		}
		if err := q.CreatePasswordResetToken(ctx, queries.CreatePasswordResetTokenParams{
			TokenHash: tokenHash, TenantID: tenantUUID, DashboardUserID: userUUID,
			ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		}); err != nil {
			return fmt.Errorf("insert password reset token: %w", err)
		}
		return nil
	})
}

// ResetPassword consumes the link, updates the password, and revokes sessions in the tenant transaction.
func (r *AuthRepository) ResetPassword(ctx context.Context, tokenHash []byte, passwordHash string, revokeUserSessions func(context.Context, string, string) error) error {
	lookup, err := queries.New(r.pool).ResolvePasswordResetToken(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.ErrInvalidPasswordReset
	}
	if err != nil {
		return fmt.Errorf("resolve password reset token: %w", err)
	}
	tenantUUID, userUUID := lookup.TenantID, lookup.DashboardUserID
	ctx = tenant.WithID(ctx, tenant.ID(tenantUUID.String()))
	return database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		consumedUserID, err := q.ConsumePasswordResetToken(ctx, queries.ConsumePasswordResetTokenParams{
			TenantID: tenantUUID, DashboardUserID: userUUID, TokenHash: tokenHash,
		})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && consumedUserID != userUUID) {
			return auth.ErrInvalidPasswordReset
		}
		if err != nil {
			return fmt.Errorf("consume password reset token: %w", err)
		}
		updated, err := q.UpdateDashboardPassword(ctx, queries.UpdateDashboardPasswordParams{
			TenantID: tenantUUID, ID: userUUID, PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("update dashboard password: %w", err)
		}
		if updated != 1 {
			return auth.ErrInvalidPasswordReset
		}
		if err := revokeUserSessions(ctx, tenantUUID.String(), userUUID.String()); err != nil {
			return fmt.Errorf("revoke dashboard session after password reset: %w", err)
		}
		return nil
	})
}

// RecordLogin updates both the dashboard user's and tenant's last-login timestamps.
func (r *AuthRepository) RecordLogin(ctx context.Context, tenantID, userID string) error {
	tenantUUID, userUUID, err := authUUIDs(tenantID, userID)
	if err != nil {
		return err
	}
	ctx = tenant.WithID(ctx, tenant.ID(tenantID))
	return database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.UpdateDashboardUserLastLogin(ctx, queries.UpdateDashboardUserLastLoginParams{TenantID: tenantUUID, ID: userUUID}); err != nil {
			return fmt.Errorf("update dashboard user login time: %w", err)
		}
		if _, err := q.UpdateTenantLastLogin(ctx, tenantUUID); err != nil {
			return fmt.Errorf("update tenant login time: %w", err)
		}
		return nil
	})
}

// VerifyEmail consumes the one-time token and marks the corresponding user verified under RLS.
func (r *AuthRepository) VerifyEmail(ctx context.Context, verificationHash []byte) (auth.Account, error) {
	lookup, err := queries.New(r.pool).ResolveEmailVerificationToken(ctx, verificationHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.Account{}, auth.ErrInvalidVerification
		}
		return auth.Account{}, err
	}
	tenantUUID, userUUID := lookup.TenantID, lookup.DashboardUserID
	ctx = tenant.WithID(ctx, tenant.ID(tenantUUID.String()))
	var account auth.Account
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		consumedUserID, err := q.ConsumeEmailVerificationToken(ctx, queries.ConsumeEmailVerificationTokenParams{
			TenantID: tenantUUID, TokenHash: verificationHash,
		})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && consumedUserID != userUUID) {
			return auth.ErrInvalidVerification
		}
		if err != nil {
			return fmt.Errorf("consume verification token: %w", err)
		}
		user, err := q.MarkDashboardUserEmailVerified(ctx, queries.MarkDashboardUserEmailVerifiedParams{TenantID: tenantUUID, ID: userUUID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return auth.ErrInvalidVerification
			}
			return fmt.Errorf("mark dashboard user email verified: %w", err)
		}
		if _, err := q.UpdateDashboardUserLastLogin(ctx, queries.UpdateDashboardUserLastLoginParams{TenantID: tenantUUID, ID: userUUID}); err != nil {
			return fmt.Errorf("update dashboard user login time: %w", err)
		}
		if _, err := q.UpdateTenantLastLogin(ctx, tenantUUID); err != nil {
			return fmt.Errorf("update tenant login time: %w", err)
		}
		account = accountFromUser(user.ID, user.TenantID, user.Email, user.Role)
		return nil
	})
	return account, err
}

// ResolveGitHubIdentity finds the linked account through the restricted lookup, then loads it under RLS.
func (r *AuthRepository) ResolveGitHubIdentity(ctx context.Context, subject string) (auth.Account, error) {
	lookup, err := queries.New(r.pool).ResolveGitHubIdentity(ctx, subject)
	if err != nil {
		return auth.Account{}, err
	}
	tenantID := lookup.TenantID.String()
	ctx = tenant.WithID(ctx, tenant.ID(tenantID))
	var account auth.Account
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		user, err := queries.New(tx).GetDashboardUserByTenantAndID(ctx, queries.GetDashboardUserByTenantAndIDParams{
			TenantID: lookup.TenantID, ID: lookup.DashboardUserID,
		})
		if err != nil {
			return fmt.Errorf("load linked GitHub user: %w", err)
		}
		account = accountFromUser(user.ID, user.TenantID, user.Email, user.Role)
		return nil
	})
	return account, err
}

// LinkGitHubIdentity attaches a verified provider subject and records successful account activity.
func (r *AuthRepository) LinkGitHubIdentity(ctx context.Context, account auth.Account, subject string) (auth.Account, error) {
	tenantUUID, userUUID, err := authUUIDs(account.TenantID, account.ID)
	if err != nil {
		return auth.Account{}, err
	}
	ctx = tenant.WithID(ctx, tenant.ID(account.TenantID))
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.MarkDashboardUserEmailVerified(ctx, queries.MarkDashboardUserEmailVerifiedParams{TenantID: tenantUUID, ID: userUUID}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("mark matched GitHub email verified: %w", err)
		}
		if _, err := q.CreateUserIdentity(ctx, queries.CreateUserIdentityParams{
			TenantID: tenantUUID, DashboardUserID: userUUID, Provider: "github", ProviderSubject: subject,
		}); err != nil {
			return fmt.Errorf("insert GitHub identity: %w", err)
		}
		if _, err := q.UpdateDashboardUserLastLogin(ctx, queries.UpdateDashboardUserLastLoginParams{TenantID: tenantUUID, ID: userUUID}); err != nil {
			return fmt.Errorf("update dashboard user login time: %w", err)
		}
		if _, err := q.UpdateTenantLastLogin(ctx, tenantUUID); err != nil {
			return fmt.Errorf("update tenant login time: %w", err)
		}
		return nil
	})
	return account, err
}

// CreateGitHubAccount atomically creates a verified tenant owner and their provider identity.
func (r *AuthRepository) CreateGitHubAccount(ctx context.Context, tenantName, email, normalizedEmail, subject string) (auth.Account, error) {
	tenantUUID, err := newTenantUUID()
	if err != nil {
		return auth.Account{}, fmt.Errorf("generate GitHub signup tenant ID: %w", err)
	}
	ctx = tenant.WithID(ctx, tenant.ID(tenantUUID.String()))
	var account auth.Account
	err = database.WithTenant(ctx, r.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := queries.New(tx)
		if _, err := q.CreateTenant(ctx, queries.CreateTenantParams{ID: tenantUUID, Name: tenantName}); err != nil {
			return fmt.Errorf("insert GitHub signup tenant: %w", err)
		}
		user, err := q.CreateVerifiedGitHubUser(ctx, queries.CreateVerifiedGitHubUserParams{
			TenantID: tenantUUID, Email: email, NormalizedEmail: normalizedEmail,
		})
		if err != nil {
			return fmt.Errorf("insert GitHub dashboard user: %w", err)
		}
		if _, err := q.CreateUserIdentity(ctx, queries.CreateUserIdentityParams{
			TenantID: tenantUUID, DashboardUserID: user.ID, Provider: "github", ProviderSubject: subject,
		}); err != nil {
			return fmt.Errorf("insert GitHub identity: %w", err)
		}
		if _, err := q.UpdateDashboardUserLastLogin(ctx, queries.UpdateDashboardUserLastLoginParams{TenantID: tenantUUID, ID: user.ID}); err != nil {
			return fmt.Errorf("update dashboard user login time: %w", err)
		}
		if _, err := q.UpdateTenantLastLogin(ctx, tenantUUID); err != nil {
			return fmt.Errorf("update tenant login time: %w", err)
		}
		account = accountFromUser(user.ID, user.TenantID, user.Email, user.Role)
		return nil
	})
	return account, err
}

// accountFromUser converts the database UUID fields into the auth service's account representation.
func accountFromUser(id, tenantID pgtype.UUID, email, role string) auth.Account {
	return auth.Account{ID: id.String(), TenantID: tenantID.String(), Email: email, Role: role}
}

// authUUIDs validates and converts both IDs before a tenant-owned auth query is issued.
func authUUIDs(tenantID, userID string) (pgtype.UUID, pgtype.UUID, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return tenantUUID, userUUID, nil
}

// parseUUID converts a UUID string to pgx's nullable-aware UUID type.
func parseUUID(value string) (pgtype.UUID, error) {
	var parsed pgtype.UUID
	if err := parsed.Scan(value); err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse UUID: %w", err)
	}
	return parsed, nil
}
