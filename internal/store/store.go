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

// WorkspaceStore persists workspaces.
type WorkspaceStore interface {
	Get(ctx context.Context, workspaceID string) (*Workspace, error)
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
	Create(ctx context.Context, workspaceID string, account *Account) error
	Delete(ctx context.Context, accountID string) error
	UpdateToken(ctx context.Context, accountID string, tokenID string, expiresAt *time.Time) error
}

// MediaStore persists media records.
type MediaStore interface {
	Get(ctx context.Context, mediaID string) (*Media, error)
	List(ctx context.Context, workspaceID string, limit int, cursor string) ([]Media, string, error)
	Create(ctx context.Context, m *Media) error
	Delete(ctx context.Context, mediaID string) error
	MarkReady(ctx context.Context, mediaID string, formats map[string]any, thumbnailKey string) error
	MarkFailed(ctx context.Context, mediaID string, reason string) error
	SetThumbnail(ctx context.Context, mediaID string, thumbnailKey string) error
}

// PostStore persists posts and targets.
type PostStore interface {
	Get(ctx context.Context, postID string) (*Post, error)
	List(ctx context.Context, workspaceID string, limit int, cursor string) ([]Post, string, error)
	Create(ctx context.Context, p *Post) error
	Update(ctx context.Context, postID string, fields map[string]any) error
	Delete(ctx context.Context, postID string) error
	SetStatus(ctx context.Context, postID string, status string) error
	GetTarget(ctx context.Context, targetID string) (*PostTarget, error)
	ListTargets(ctx context.Context, postID string) ([]PostTarget, error)
	CreateTarget(ctx context.Context, t *PostTarget) error
	SetTargetFailed(ctx context.Context, targetID string, reason string) error
	SetTargetPublished(ctx context.Context, targetID string, platformPostID string, permalink string) error
	ListScheduled(ctx context.Context, workspaceID string, from time.Time, to time.Time) ([]Post, error)
}

// AnalyticsStore persists analytics snapshots.
type AnalyticsStore interface {
	Record(ctx context.Context, snapshot AnalyticsSnapshot) error
	GetPostMetrics(ctx context.Context, postID string) (*AnalyticsSnapshot, error)
	GetAccountMetrics(ctx context.Context, accountID string) (*AnalyticsSnapshot, error)
	GetSummary(ctx context.Context, workspaceID string, from *time.Time, to *time.Time) (*AnalyticsSnapshot, error)
}

// WebhookStore persists webhook settings and delivery work.
type WebhookStore interface {
	Create(ctx context.Context, e *WebhookEndpoint) error
	List(ctx context.Context, workspaceID string) ([]WebhookEndpoint, error)
	Get(ctx context.Context, endpointID string) (*WebhookEndpoint, error)
	Delete(ctx context.Context, endpointID string) error
	EnqueueDelivery(ctx context.Context, params WebhookDeliveryParams) error
	ListPendingDeliveries(ctx context.Context, endpointID string, limit int) ([]WebhookDelivery, error)
	MarkDelivered(ctx context.Context, deliveryID string, statusCode int) error
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
func New(db *pgxpool.Pool, tokenKey string) Stores {
	return Stores{
		Workspaces: &workspaceStore{db: db},
		APIKeys:    &apiKeyStore{db: db},
		Accounts:   &accountStore{db: db},
		Media:      &mediaStore{db: db},
		Posts:      &postStore{db: db},
		Analytics:  &analyticsStore{db: db},
		Webhooks:   &webhookStore{db: db},
		Tokens:     &tokenStore{db: db, key: []byte(tokenKey)},
	}
}
