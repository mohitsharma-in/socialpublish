package tenant

import "context"

type contextKey struct{}

// Workspace holds the authenticated workspace details.
type Workspace struct {
	ID   string
	Plan string
}

// FromContext extracts the Workspace from ctx.
func FromContext(ctx context.Context) (Workspace, bool) {
	w, ok := ctx.Value(contextKey{}).(Workspace)
	return w, ok
}

// WithWorkspace stores a Workspace in ctx.
func WithWorkspace(ctx context.Context, w Workspace) context.Context {
	return context.WithValue(ctx, contextKey{}, w)
}

// MustFromContext extracts the Workspace or panics.
func MustFromContext(ctx context.Context) Workspace {
	w, ok := FromContext(ctx)
	if !ok {
		panic("tenant: workspace not in context")
	}
	return w
}
