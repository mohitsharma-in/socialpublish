package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type workspaceStore struct {
	db *pgxpool.Pool
}

// Get fetches a workspace by ID.
func (s *workspaceStore) Get(ctx context.Context, workspaceID string) (*Workspace, error) {
	const query = `SELECT id, name, slug, plan, created_at, updated_at FROM workspaces WHERE id = @workspace_id`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"workspace_id": workspaceID})
	if err != nil {
		return nil, fmt.Errorf("query workspace %s: %w", workspaceID, err)
	}
	ws, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Workspace])
	if err != nil {
		return nil, fmt.Errorf("scan workspace %s: %w", workspaceID, err)
	}
	return &ws, nil
}

// Allow reports whether a workspace is within its rate limit.
func (s *workspaceStore) Allow(ctx context.Context, workspaceID string, limit int) (int, time.Time, bool) {
	reset := time.Now().Add(time.Minute).Truncate(time.Minute)
	return limit, reset, true
}

type apiKeyStore struct {
	db *pgxpool.Pool
}

// FindByHash fetches an API key by hash.
func (s *apiKeyStore) FindByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	const query = `SELECT id, workspace_id, key_hash, key_prefix, name, last_used_at, expires_at, created_at FROM api_keys WHERE key_hash = @key_hash`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"key_hash": keyHash})
	if err != nil {
		return nil, fmt.Errorf("query api key: %w", err)
	}
	key, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[APIKey])
	if err != nil {
		return nil, fmt.Errorf("scan api key: %w", err)
	}
	return &key, nil
}

// TouchLastUsed records API key usage.
func (s *apiKeyStore) TouchLastUsed(ctx context.Context, keyID string) error {
	const query = `UPDATE api_keys SET last_used_at = now() WHERE id = @key_id`
	if _, err := s.db.Exec(ctx, query, pgx.NamedArgs{"key_id": keyID}); err != nil {
		return fmt.Errorf("touch api key %s: %w", keyID, err)
	}
	return nil
}

type accountStore struct {
	db *pgxpool.Pool
}

// Get fetches an account by ID.
func (s *accountStore) Get(ctx context.Context, accountID string) (*Account, error) {
	const query = `SELECT id, workspace_id, platform, platform_user_id, display_name, COALESCE(avatar_url, '') AS avatar_url, token_id, token_expires_at, token_healthy, created_at, updated_at FROM accounts WHERE id = @account_id`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"account_id": accountID})
	if err != nil {
		return nil, fmt.Errorf("query account %s: %w", accountID, err)
	}
	account, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Account])
	if err != nil {
		return nil, fmt.Errorf("scan account %s: %w", accountID, err)
	}
	return &account, nil
}

// List fetches accounts for a workspace.
func (s *accountStore) List(ctx context.Context, workspaceID string) ([]Account, error) {
	const query = `SELECT id, workspace_id, platform, platform_user_id, display_name, COALESCE(avatar_url, '') AS avatar_url, token_id, token_expires_at, token_healthy, created_at, updated_at FROM accounts WHERE workspace_id = @workspace_id ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"workspace_id": workspaceID})
	if err != nil {
		return nil, fmt.Errorf("query accounts for workspace %s: %w", workspaceID, err)
	}
	accounts, err := pgx.CollectRows(rows, pgx.RowToStructByName[Account])
	if err != nil {
		return nil, fmt.Errorf("scan accounts for workspace %s: %w", workspaceID, err)
	}
	return accounts, nil
}

type mediaStore struct {
	db *pgxpool.Pool
}

// Get fetches media by ID.
func (s *mediaStore) Get(ctx context.Context, mediaID string) (*Media, error) {
	return nil, fmt.Errorf("get media %s: %w", mediaID, ErrNotImplemented)
}

// MarkReady records successful media processing.
func (s *mediaStore) MarkReady(ctx context.Context, mediaID string, formats map[string]any, thumbnailKey string) error {
	formatsJSON, err := json.Marshal(formats)
	if err != nil {
		return fmt.Errorf("marshal media formats: %w", err)
	}
	const query = `UPDATE media SET status = 'ready', formats = @formats, thumbnail_key = @thumbnail_key, updated_at = now() WHERE id = @media_id`
	if _, err := s.db.Exec(ctx, query, pgx.NamedArgs{"media_id": mediaID, "formats": formatsJSON, "thumbnail_key": thumbnailKey}); err != nil {
		return fmt.Errorf("mark media %s ready: %w", mediaID, err)
	}
	return nil
}

// MarkFailed records media processing failure.
func (s *mediaStore) MarkFailed(ctx context.Context, mediaID string, reason string) error {
	const query = `UPDATE media SET status = 'failed', error_message = @reason, updated_at = now() WHERE id = @media_id`
	if _, err := s.db.Exec(ctx, query, pgx.NamedArgs{"media_id": mediaID, "reason": reason}); err != nil {
		return fmt.Errorf("mark media %s failed: %w", mediaID, err)
	}
	return nil
}

type postStore struct {
	db *pgxpool.Pool
}

// Get fetches a post by ID.
func (s *postStore) Get(ctx context.Context, postID string) (*Post, error) {
	return nil, fmt.Errorf("get post %s: %w", postID, ErrNotImplemented)
}

// GetTarget fetches a post target by ID.
func (s *postStore) GetTarget(ctx context.Context, targetID string) (*PostTarget, error) {
	return nil, fmt.Errorf("get target %s: %w", targetID, ErrNotImplemented)
}

// SetTargetFailed records target publish failure.
func (s *postStore) SetTargetFailed(ctx context.Context, targetID string, reason string) error {
	const query = `UPDATE post_targets SET status = 'failed', failure_reason = @reason, attempt_count = attempt_count + 1, last_attempted_at = now() WHERE id = @target_id`
	if _, err := s.db.Exec(ctx, query, pgx.NamedArgs{"target_id": targetID, "reason": reason}); err != nil {
		return fmt.Errorf("set target %s failed: %w", targetID, err)
	}
	return nil
}

// SetTargetPublished records successful target publishing.
func (s *postStore) SetTargetPublished(ctx context.Context, targetID string, platformPostID string, permalink string) error {
	const query = `UPDATE post_targets SET status = 'published', platform_post_id = @platform_post_id, permalink = @permalink, published_at = now() WHERE id = @target_id`
	if _, err := s.db.Exec(ctx, query, pgx.NamedArgs{"target_id": targetID, "platform_post_id": platformPostID, "permalink": permalink}); err != nil {
		return fmt.Errorf("set target %s published: %w", targetID, err)
	}
	return nil
}

type analyticsStore struct {
	db *pgxpool.Pool
}

// Record records an analytics snapshot.
func (s *analyticsStore) Record(ctx context.Context, snapshot AnalyticsSnapshot) error {
	return fmt.Errorf("record analytics snapshot: %w", ErrNotImplemented)
}

type webhookStore struct {
	db *pgxpool.Pool
}

// EnqueueDelivery stores or enqueues a webhook delivery.
func (s *webhookStore) EnqueueDelivery(ctx context.Context, params WebhookDeliveryParams) error {
	return nil
}

type tokenStore struct {
	db *pgxpool.Pool
}

// Decrypt returns a decrypted token.
func (s *tokenStore) Decrypt(ctx context.Context, tokenID string) (string, error) {
	return "", fmt.Errorf("decrypt token %s: %w", tokenID, ErrNotImplemented)
}

// Save persists an encrypted token.
func (s *tokenStore) Save(ctx context.Context, token string) (string, error) {
	return "", fmt.Errorf("save token: %w", ErrNotImplemented)
}
