package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Stores groups all persistence dependencies.
type Stores struct {
	Workspaces WorkspaceStore
	APIKeys    APIKeyStore
	Accounts   AccountStore
	Media      MediaStore
	Posts      PostStore
	Analytics  AnalyticsStore
	Webhooks   WebhookStore
	Tokens     TokenStore
}

// WorkspaceStore persists workspaces and rate-limit state.
type WorkspaceStore interface {
	Get(ctx context.Context, workspaceID string) (*Workspace, error)
	Allow(ctx context.Context, workspaceID string, limit int) (int, time.Time, bool)
}

// APIKeyStore validates API keys.
type APIKeyStore interface {
	FindByHash(ctx context.Context, keyHash string) (*APIKey, error)
	TouchLastUsed(ctx context.Context, keyID string) error
}

// AccountStore persists connected accounts.
type AccountStore interface {
	Get(ctx context.Context, accountID string) (*Account, error)
	List(ctx context.Context, workspaceID string) ([]Account, error)
}

// MediaStore persists media records.
type MediaStore interface {
	Get(ctx context.Context, mediaID string) (*Media, error)
	MarkReady(ctx context.Context, mediaID string, formats map[string]any, thumbnailKey string) error
	MarkFailed(ctx context.Context, mediaID string, reason string) error
}

// PostStore persists posts and targets.
type PostStore interface {
	Get(ctx context.Context, postID string) (*Post, error)
	GetTarget(ctx context.Context, targetID string) (*PostTarget, error)
	SetTargetFailed(ctx context.Context, targetID string, reason string) error
	SetTargetPublished(ctx context.Context, targetID string, platformPostID string, permalink string) error
}

// AnalyticsStore persists analytics snapshots.
type AnalyticsStore interface {
	Record(ctx context.Context, snapshot AnalyticsSnapshot) error
}

// WebhookStore persists webhook settings and delivery work.
type WebhookStore interface {
	EnqueueDelivery(ctx context.Context, params WebhookDeliveryParams) error
}

// TokenStore persists encrypted platform tokens.
type TokenStore interface {
	Decrypt(ctx context.Context, tokenID string) (string, error)
	Save(ctx context.Context, token string) (string, error)
}

// OpenDB opens a pgx connection pool.
func OpenDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pgx pool: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

// New creates pgx-backed stores.
func New(db *pgxpool.Pool) Stores {
	return Stores{
		Workspaces: &workspaceStore{db: db},
		APIKeys:    &apiKeyStore{db: db},
		Accounts:   &accountStore{db: db},
		Media:      &mediaStore{db: db},
		Posts:      &postStore{db: db},
		Analytics:  &analyticsStore{db: db},
		Webhooks:   &webhookStore{db: db},
		Tokens:     &tokenStore{db: db},
	}
}
