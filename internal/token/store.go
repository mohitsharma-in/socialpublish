package token

import "context"

// Store persists encrypted platform tokens.
type Store interface {
	Decrypt(ctx context.Context, tokenID string) (string, error)
	Save(ctx context.Context, token string) (string, error)
}
