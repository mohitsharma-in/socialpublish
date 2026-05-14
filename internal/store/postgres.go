package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
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

// Create inserts a new account.
func (s *accountStore) Create(ctx context.Context, workspaceID string, account *Account) error {
	const query = `INSERT INTO accounts (workspace_id, platform, platform_user_id, display_name, avatar_url, token_id, token_expires_at) VALUES (@workspace_id, @platform, @platform_user_id, @display_name, @avatar_url, @token_id, @token_expires_at) RETURNING id, created_at, updated_at`
	row := s.db.QueryRow(ctx, query, pgx.NamedArgs{
		"workspace_id": workspaceID, "platform": account.Platform, "platform_user_id": account.PlatformUserID,
		"display_name": account.DisplayName, "avatar_url": account.AvatarURL, "token_id": account.TokenID, "token_expires_at": account.TokenExpiresAt,
	})
	return row.Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)
}

// Delete removes an account.
func (s *accountStore) Delete(ctx context.Context, accountID string) error {
	const query = `DELETE FROM accounts WHERE id = @account_id`
	result, err := s.db.Exec(ctx, query, pgx.NamedArgs{"account_id": accountID})
	if err != nil {
		return fmt.Errorf("delete account %s: %w", accountID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("account %s not found", accountID)
	}
	return nil
}

// UpdateToken updates the token reference on an account.
func (s *accountStore) UpdateToken(ctx context.Context, accountID string, tokenID string, expiresAt *time.Time) error {
	const query = `UPDATE accounts SET token_id = @token_id, token_expires_at = @expires_at, token_healthy = true, updated_at = now() WHERE id = @account_id`
	_, err := s.db.Exec(ctx, query, pgx.NamedArgs{"account_id": accountID, "token_id": tokenID, "expires_at": expiresAt})
	if err != nil {
		return fmt.Errorf("update token for account %s: %w", accountID, err)
	}
	return nil
}

type mediaStore struct {
	db *pgxpool.Pool
}

// Get fetches media by ID.
func (s *mediaStore) Get(ctx context.Context, mediaID string) (*Media, error) {
	const query = `SELECT id, workspace_id, status, media_type, original_key, mime_type, size_bytes, duration_ms, formats, thumbnail_key, error_message, created_at, updated_at FROM media WHERE id = @media_id`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"media_id": mediaID})
	if err != nil {
		return nil, fmt.Errorf("query media %s: %w", mediaID, err)
	}
	media, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Media])
	if err != nil {
		return nil, fmt.Errorf("scan media %s: %w", mediaID, err)
	}
	return &media, nil
}

// List lists media for a workspace with cursor pagination.
func (s *mediaStore) List(ctx context.Context, workspaceID string, limit int, cursor string) ([]Media, string, error) {
	if limit <= 0 { limit = 20 }
	const query = `SELECT id, workspace_id, status, media_type, original_key, mime_type, size_bytes, duration_ms, formats, thumbnail_key, error_message, created_at, updated_at FROM media WHERE workspace_id = @workspace_id AND (@cursor = '' OR id::text < @cursor) ORDER BY created_at DESC LIMIT @lim`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"workspace_id": workspaceID, "cursor": cursor, "lim": limit + 1})
	if err != nil { return nil, "", fmt.Errorf("list media: %w", err) }
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[Media])
	if err != nil { return nil, "", fmt.Errorf("scan media: %w", err) }
	var next string
	if len(items) > limit { next = items[limit-1].ID; items = items[:limit] }
	return items, next, nil
}

// Create inserts a new media record.
func (s *mediaStore) Create(ctx context.Context, m *Media) error {
	const query = `INSERT INTO media (workspace_id, status, media_type, original_key, mime_type, size_bytes) VALUES (@workspace_id, @status, @media_type, @original_key, @mime_type, @size_bytes) RETURNING id, created_at, updated_at`
	row := s.db.QueryRow(ctx, query, pgx.NamedArgs{"workspace_id": m.WorkspaceID, "status": m.Status, "media_type": m.MediaType, "original_key": m.OriginalKey, "mime_type": m.MimeType, "size_bytes": m.SizeBytes})
	return row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

// Delete removes a media record.
func (s *mediaStore) Delete(ctx context.Context, mediaID string) error {
	result, err := s.db.Exec(ctx, `DELETE FROM media WHERE id = @media_id`, pgx.NamedArgs{"media_id": mediaID})
	if err != nil { return fmt.Errorf("delete media %s: %w", mediaID, err) }
	if result.RowsAffected() == 0 { return fmt.Errorf("media %s not found", mediaID) }
	return nil
}

// MarkReady records successful media processing.
func (s *mediaStore) MarkReady(ctx context.Context, mediaID string, formats map[string]any, thumbnailKey string) error {
	formatsJSON, err := json.Marshal(formats)
	if err != nil { return fmt.Errorf("marshal media formats: %w", err) }
	const query = `UPDATE media SET status = 'ready', formats = @formats, thumbnail_key = @thumbnail_key, updated_at = now() WHERE id = @media_id`
	_, err = s.db.Exec(ctx, query, pgx.NamedArgs{"media_id": mediaID, "formats": formatsJSON, "thumbnail_key": thumbnailKey})
	if err != nil { return fmt.Errorf("mark media %s ready: %w", mediaID, err) }
	return nil
}

// MarkFailed records media processing failure.
func (s *mediaStore) MarkFailed(ctx context.Context, mediaID string, reason string) error {
	const query = `UPDATE media SET status = 'failed', error_message = @reason, updated_at = now() WHERE id = @media_id`
	_, err := s.db.Exec(ctx, query, pgx.NamedArgs{"media_id": mediaID, "reason": reason})
	if err != nil { return fmt.Errorf("mark media %s failed: %w", mediaID, err) }
	return nil
}

// SetThumbnail updates the thumbnail key for a media record.
func (s *mediaStore) SetThumbnail(ctx context.Context, mediaID string, thumbnailKey string) error {
	_, err := s.db.Exec(ctx, `UPDATE media SET thumbnail_key = @key, updated_at = now() WHERE id = @id`, pgx.NamedArgs{"id": mediaID, "key": thumbnailKey})
	if err != nil { return fmt.Errorf("set thumbnail media %s: %w", mediaID, err) }
	return nil
}

type postStore struct {
	db *pgxpool.Pool
}

// Get fetches a post by ID.
func (s *postStore) Get(ctx context.Context, postID string) (*Post, error) {
	// Columns match migrations/000006_create_posts.up.sql. uuid[] -> text[] for []string scan.
	const query = `
SELECT
	id::text,
	workspace_id::text,
	status,
	COALESCE((SELECT array_agg(u::text) FROM unnest(COALESCE(media_ids, ARRAY[]::uuid[])) AS u), ARRAY[]::text[]) AS media_ids,
	scheduled_at,
	published_at,
	metadata,
	created_at,
	updated_at
FROM posts
WHERE id = @post_id::uuid`
	row := s.db.QueryRow(ctx, query, pgx.NamedArgs{"post_id": postID})
	var p Post
	if err := row.Scan(
		&p.ID,
		&p.WorkspaceID,
		&p.Status,
		&p.MediaIDs,
		&p.ScheduledAt,
		&p.PublishedAt,
		&p.Metadata,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan post %s: %w", postID, err)
	}
	return &p, nil
}

// GetTarget fetches a post target by ID.
func (s *postStore) GetTarget(ctx context.Context, targetID string) (*PostTarget, error) {
	const query = `SELECT id, post_id, account_id, platform, format, config, status, platform_post_id, permalink, failure_reason, attempt_count, last_attempted_at, published_at FROM post_targets WHERE id = @target_id`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"target_id": targetID})
	if err != nil {
		return nil, fmt.Errorf("query post target %s: %w", targetID, err)
	}
	target, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[PostTarget])
	if err != nil {
		return nil, fmt.Errorf("scan post target %s: %w", targetID, err)
	}
	return &target, nil
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

// List lists posts for a workspace with cursor pagination.
func (s *postStore) List(ctx context.Context, workspaceID string, limit int, cursor string) ([]Post, string, error) {
	if limit <= 0 { limit = 20 }
	const query = `SELECT id::text, workspace_id::text, status, COALESCE((SELECT array_agg(u::text) FROM unnest(COALESCE(media_ids, ARRAY[]::uuid[])) AS u), ARRAY[]::text[]) AS media_ids, scheduled_at, published_at, metadata, created_at, updated_at FROM posts WHERE workspace_id = @workspace_id::uuid AND (@cursor = '' OR id::text < @cursor) ORDER BY created_at DESC LIMIT @lim`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"workspace_id": workspaceID, "cursor": cursor, "lim": limit + 1})
	if err != nil { return nil, "", fmt.Errorf("list posts: %w", err) }
	var items []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Status, &p.MediaIDs, &p.ScheduledAt, &p.PublishedAt, &p.Metadata, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, "", fmt.Errorf("scan post: %w", err)
		}
		items = append(items, p)
	}
	var next string
	if len(items) > limit { next = items[limit-1].ID; items = items[:limit] }
	return items, next, nil
}

// Create inserts a new post.
func (s *postStore) Create(ctx context.Context, p *Post) error {
	metaJSON, _ := json.Marshal(p.Metadata)
	const query = `INSERT INTO posts (workspace_id, status, media_ids, scheduled_at, metadata) VALUES (@workspace_id::uuid, @status, @media_ids::uuid[], @scheduled_at, @metadata) RETURNING id::text, created_at, updated_at`
	row := s.db.QueryRow(ctx, query, pgx.NamedArgs{"workspace_id": p.WorkspaceID, "status": p.Status, "media_ids": p.MediaIDs, "scheduled_at": p.ScheduledAt, "metadata": metaJSON})
	return row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

// Update patches post fields.
func (s *postStore) Update(ctx context.Context, postID string, fields map[string]any) error {
	metaJSON, _ := json.Marshal(fields)
	_, err := s.db.Exec(ctx, `UPDATE posts SET metadata = metadata || @meta::jsonb, updated_at = now() WHERE id = @id::uuid`, pgx.NamedArgs{"id": postID, "meta": metaJSON})
	if err != nil { return fmt.Errorf("update post %s: %w", postID, err) }
	return nil
}

// Delete removes a post.
func (s *postStore) Delete(ctx context.Context, postID string) error {
	result, err := s.db.Exec(ctx, `DELETE FROM posts WHERE id = @id::uuid`, pgx.NamedArgs{"id": postID})
	if err != nil { return fmt.Errorf("delete post %s: %w", postID, err) }
	if result.RowsAffected() == 0 { return fmt.Errorf("post %s not found", postID) }
	return nil
}

// SetStatus updates a post's status.
func (s *postStore) SetStatus(ctx context.Context, postID string, status string) error {
	_, err := s.db.Exec(ctx, `UPDATE posts SET status = @status, updated_at = now() WHERE id = @id::uuid`, pgx.NamedArgs{"id": postID, "status": status})
	if err != nil { return fmt.Errorf("set post %s status: %w", postID, err) }
	return nil
}

// ListTargets returns all targets for a post.
func (s *postStore) ListTargets(ctx context.Context, postID string) ([]PostTarget, error) {
	const query = `SELECT id, post_id, account_id, platform, format, config, status, platform_post_id, permalink, failure_reason, attempt_count, last_attempted_at, published_at FROM post_targets WHERE post_id = @post_id`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"post_id": postID})
	if err != nil { return nil, fmt.Errorf("list targets for post %s: %w", postID, err) }
	return pgx.CollectRows(rows, pgx.RowToStructByName[PostTarget])
}

// CreateTarget inserts a post target.
func (s *postStore) CreateTarget(ctx context.Context, t *PostTarget) error {
	configJSON, _ := json.Marshal(t.Config)
	const query = `INSERT INTO post_targets (post_id, account_id, platform, format, config) VALUES (@post_id, @account_id, @platform, @format, @config) RETURNING id`
	row := s.db.QueryRow(ctx, query, pgx.NamedArgs{"post_id": t.PostID, "account_id": t.AccountID, "platform": t.Platform, "format": t.Format, "config": configJSON})
	return row.Scan(&t.ID)
}

// ListScheduled lists scheduled posts in a time range.
func (s *postStore) ListScheduled(ctx context.Context, workspaceID string, from time.Time, to time.Time) ([]Post, error) {
	const query = `SELECT id::text, workspace_id::text, status, COALESCE((SELECT array_agg(u::text) FROM unnest(COALESCE(media_ids, ARRAY[]::uuid[])) AS u), ARRAY[]::text[]) AS media_ids, scheduled_at, published_at, metadata, created_at, updated_at FROM posts WHERE workspace_id = @workspace_id::uuid AND status = 'scheduled' AND scheduled_at >= @from AND scheduled_at <= @to ORDER BY scheduled_at`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"workspace_id": workspaceID, "from": from, "to": to})
	if err != nil { return nil, fmt.Errorf("list scheduled: %w", err) }
	var items []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Status, &p.MediaIDs, &p.ScheduledAt, &p.PublishedAt, &p.Metadata, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan scheduled post: %w", err)
		}
		items = append(items, p)
	}
	return items, nil
}

type analyticsStore struct {
	db *pgxpool.Pool
}

// Record records an analytics snapshot.
func (s *analyticsStore) Record(ctx context.Context, snapshot AnalyticsSnapshot) error {
	metricsJSON, err := json.Marshal(snapshot.Metrics)
	if err != nil { return fmt.Errorf("marshal analytics metrics: %w", err) }
	const query = `INSERT INTO analytics_snapshots (workspace_id, account_id, post_id, platform_post_id, metrics, collected_at) VALUES (@workspace_id, @account_id, @post_id, @platform_post_id, @metrics, @collected_at)`
	_, err = s.db.Exec(ctx, query, pgx.NamedArgs{
		"workspace_id": snapshot.WorkspaceID, "account_id": snapshot.AccountID,
		"post_id": snapshot.PostID, "platform_post_id": snapshot.PlatformPostID,
		"metrics": metricsJSON, "collected_at": snapshot.CollectedAt,
	})
	if err != nil { return fmt.Errorf("record analytics snapshot: %w", err) }
	return nil
}

// GetPostMetrics returns the latest snapshot for a post.
func (s *analyticsStore) GetPostMetrics(ctx context.Context, postID string) (*AnalyticsSnapshot, error) {
	const query = `SELECT id, workspace_id, account_id, post_id, platform_post_id, metrics, collected_at FROM analytics_snapshots WHERE post_id = @post_id ORDER BY collected_at DESC LIMIT 1`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"post_id": postID})
	if err != nil { return nil, fmt.Errorf("get post metrics: %w", err) }
	snap, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[AnalyticsSnapshot])
	if err != nil { return nil, fmt.Errorf("scan post metrics: %w", err) }
	return &snap, nil
}

// GetAccountMetrics returns the latest snapshot for an account.
func (s *analyticsStore) GetAccountMetrics(ctx context.Context, accountID string) (*AnalyticsSnapshot, error) {
	const query = `SELECT id, workspace_id, account_id, post_id, platform_post_id, metrics, collected_at FROM analytics_snapshots WHERE account_id = @account_id ORDER BY collected_at DESC LIMIT 1`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"account_id": accountID})
	if err != nil { return nil, fmt.Errorf("get account metrics: %w", err) }
	snap, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[AnalyticsSnapshot])
	if err != nil { return nil, fmt.Errorf("scan account metrics: %w", err) }
	return &snap, nil
}

// GetSummary returns aggregated metrics for a workspace.
func (s *analyticsStore) GetSummary(ctx context.Context, workspaceID string, from *time.Time, to *time.Time) (*AnalyticsSnapshot, error) {
	const query = `SELECT COALESCE(id, '') AS id, workspace_id, COALESCE(account_id, '') AS account_id, COALESCE(post_id, '') AS post_id, '' AS platform_post_id, metrics, collected_at FROM analytics_snapshots WHERE workspace_id = @workspace_id ORDER BY collected_at DESC LIMIT 1`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"workspace_id": workspaceID})
	if err != nil { return nil, fmt.Errorf("get summary: %w", err) }
	snap, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[AnalyticsSnapshot])
	if err != nil { return nil, fmt.Errorf("scan summary: %w", err) }
	return &snap, nil
}

type webhookStore struct {
	db *pgxpool.Pool
}

// Create inserts a webhook endpoint.
func (s *webhookStore) Create(ctx context.Context, e *WebhookEndpoint) error {
	const query = `INSERT INTO webhook_endpoints (workspace_id, url, secret_hash, secret_enc, events, enabled) VALUES (@workspace_id, @url, @secret_hash, @secret_enc, @events, @enabled) RETURNING id, created_at`
	row := s.db.QueryRow(ctx, query, pgx.NamedArgs{"workspace_id": e.WorkspaceID, "url": e.URL, "secret_hash": e.SecretHash, "secret_enc": e.SecretEnc, "events": e.Events, "enabled": e.Enabled})
	return row.Scan(&e.ID, &e.CreatedAt)
}

// List returns all webhook endpoints for a workspace.
func (s *webhookStore) List(ctx context.Context, workspaceID string) ([]WebhookEndpoint, error) {
	const query = `SELECT id, workspace_id, url, secret_hash, secret_enc, events, enabled, created_at FROM webhook_endpoints WHERE workspace_id = @workspace_id ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"workspace_id": workspaceID})
	if err != nil { return nil, fmt.Errorf("list webhooks: %w", err) }
	return pgx.CollectRows(rows, pgx.RowToStructByName[WebhookEndpoint])
}

// Get returns a single webhook endpoint.
func (s *webhookStore) Get(ctx context.Context, endpointID string) (*WebhookEndpoint, error) {
	const query = `SELECT id, workspace_id, url, secret_hash, secret_enc, events, enabled, created_at FROM webhook_endpoints WHERE id = @id`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"id": endpointID})
	if err != nil { return nil, fmt.Errorf("get webhook: %w", err) }
	ep, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[WebhookEndpoint])
	if err != nil { return nil, fmt.Errorf("scan webhook: %w", err) }
	return &ep, nil
}

// Delete removes a webhook endpoint.
func (s *webhookStore) Delete(ctx context.Context, endpointID string) error {
	result, err := s.db.Exec(ctx, `DELETE FROM webhook_endpoints WHERE id = @id`, pgx.NamedArgs{"id": endpointID})
	if err != nil { return fmt.Errorf("delete webhook %s: %w", endpointID, err) }
	if result.RowsAffected() == 0 { return fmt.Errorf("webhook %s not found", endpointID) }
	return nil
}

// EnqueueDelivery stores a webhook delivery for all matching enabled endpoints.
func (s *webhookStore) EnqueueDelivery(ctx context.Context, params WebhookDeliveryParams) error {
	payloadJSON, err := json.Marshal(params.Payload)
	if err != nil { return fmt.Errorf("marshal webhook payload: %w", err) }
	const query = `INSERT INTO webhook_deliveries (endpoint_id, event_type, payload) SELECT id, @event_type, @payload FROM webhook_endpoints WHERE workspace_id = @workspace_id AND enabled = true`
	_, err = s.db.Exec(ctx, query, pgx.NamedArgs{"workspace_id": params.WorkspaceID, "event_type": params.EventType, "payload": payloadJSON})
	if err != nil { return fmt.Errorf("enqueue webhook delivery: %w", err) }
	return nil
}

// ListPendingDeliveries returns undelivered webhook deliveries.
func (s *webhookStore) ListPendingDeliveries(ctx context.Context, endpointID string, limit int) ([]WebhookDelivery, error) {
	if limit <= 0 { limit = 10 }
	const query = `SELECT id, endpoint_id, event_type, payload, response_status, attempt_count, delivered_at, next_retry_at FROM webhook_deliveries WHERE endpoint_id = @endpoint_id AND delivered_at IS NULL ORDER BY id LIMIT @lim`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"endpoint_id": endpointID, "lim": limit})
	if err != nil { return nil, fmt.Errorf("list pending deliveries: %w", err) }
	return pgx.CollectRows(rows, pgx.RowToStructByName[WebhookDelivery])
}

// MarkDelivered records a successful webhook delivery.
func (s *webhookStore) MarkDelivered(ctx context.Context, deliveryID string, statusCode int) error {
	_, err := s.db.Exec(ctx, `UPDATE webhook_deliveries SET delivered_at = now(), response_status = @status, attempt_count = attempt_count + 1 WHERE id = @id`, pgx.NamedArgs{"id": deliveryID, "status": statusCode})
	if err != nil { return fmt.Errorf("mark delivered %s: %w", deliveryID, err) }
	return nil
}


type tokenStore struct {
	db  *pgxpool.Pool
	key []byte
}

// Decrypt returns a decrypted token.
func (s *tokenStore) Decrypt(ctx context.Context, tokenID string) (string, error) {
	if len(s.key) != 32 {
		return "", fmt.Errorf("decrypt token %s: invalid encryption key", tokenID)
	}
	const query = `SELECT ciphertext FROM encrypted_tokens WHERE id = @token_id`
	row := s.db.QueryRow(ctx, query, pgx.NamedArgs{"token_id": tokenID})
	var ciphertext []byte
	if err := row.Scan(&ciphertext); err != nil {
		return "", fmt.Errorf("query encrypted token %s: %w", tokenID, err)
	}
	plaintext, err := decryptToken(s.key, ciphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt token %s: %w", tokenID, err)
	}
	return plaintext, nil
}

// Save persists an encrypted token.
func (s *tokenStore) Save(ctx context.Context, token string) (string, error) {
	if len(s.key) != 32 {
		return "", fmt.Errorf("save token: invalid encryption key")
	}
	ciphertext, err := encryptToken(s.key, token)
	if err != nil {
		return "", fmt.Errorf("encrypt token: %w", err)
	}
	const query = `INSERT INTO encrypted_tokens (ciphertext) VALUES (@ciphertext) RETURNING id`
	row := s.db.QueryRow(ctx, query, pgx.NamedArgs{"ciphertext": ciphertext})
	var tokenID string
	if err := row.Scan(&tokenID); err != nil {
		return "", fmt.Errorf("insert encrypted token: %w", err)
	}
	return tokenID, nil
}

func encryptToken(key []byte, token string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nonce, nonce, []byte(token), nil)
	return ciphertext, nil
}

func decryptToken(key []byte, data []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aead.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
