package auth

import "context"

// contextKey is a private key type to avoid collisions with other packages.
type contextKey string

const (
	userIDKey    contextKey = "gk/user-id"
	userEmailKey contextKey = "gk/user-email"
)

// WithUser returns a child context carrying the authenticated user's id and email.
func WithUser(ctx context.Context, userID, email string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	if email != "" {
		ctx = context.WithValue(ctx, userEmailKey, email)
	}
	return ctx
}

// UserIDFrom extracts the user id from context set by WithUser.
func UserIDFrom(ctx context.Context) (string, bool) {
	v, _ := ctx.Value(userIDKey).(string)
	return v, v != ""
}

// UserEmailFrom extracts the user email from context set by WithUser.
func UserEmailFrom(ctx context.Context) (string, bool) {
	v, _ := ctx.Value(userEmailKey).(string)
	return v, v != ""
}

// UserFrom returns both id and email when available.
func UserFrom(ctx context.Context) (id, email string, ok bool) {
	id, ok = UserIDFrom(ctx)
	if !ok {
		return "", "", false
	}
	email, _ = UserEmailFrom(ctx)
	return id, email, true
}
