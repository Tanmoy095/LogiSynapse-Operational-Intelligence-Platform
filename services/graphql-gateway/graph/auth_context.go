package graph

import "context"

type Identity struct {
	UserID       string
	Email        string
	TenantID     string
	Role         string
	IsSuperAdmin bool
}

type identityKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}
