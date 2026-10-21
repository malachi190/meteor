package auth

import "context"

// accountContextKey keeps authenticated account values separate from other context entries.
type accountContextKey struct{}

// WithAccount stores an account resolved from a valid server-side session.
func WithAccount(ctx context.Context, account Account) context.Context {
	return context.WithValue(ctx, accountContextKey{}, account)
}

// AccountFromContext returns the account attached by authenticated HTTP middleware.
func AccountFromContext(ctx context.Context) (Account, bool) {
	account, ok := ctx.Value(accountContextKey{}).(Account)
	return account, ok && account.ID != "" && account.TenantID != ""
}
