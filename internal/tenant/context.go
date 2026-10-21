package tenant

import "context"

// ID identifies a tenant in the authenticated request context.
type ID string

// contextKey prevents tenant scope values from colliding with keys owned by other packages.
type contextKey struct{}

// WithID stores a tenant ID in a context for trusted application code to use.
func WithID(ctx context.Context, id ID) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// IDFromContext returns the tenant ID when one has been attached to the context.
func IDFromContext(ctx context.Context) (ID, bool) {
	id, ok := ctx.Value(contextKey{}).(ID)
	return id, ok && id != ""
}
