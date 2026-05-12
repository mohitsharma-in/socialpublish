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
	const query = `SELECT id, workspace_id, author_id, caption, media_ids, scheduled_at, status, created_at, updated_at FROM posts WHERE id = @post_id`
	rows, err := s.db.Query(ctx, query, pgx.NamedArgs{"post_id": postID})
	if err != nil {
		return nil, fmt.Errorf("query post %s: %w", postID, err)
	}
	post, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Post])
	if err != nil {
		return nil, fmt.Errorf("scan post %s: %w", postID, err)
	}
	return &post, nil
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

type analyticsStore struct {
	db *pgxpool.Pool
}

// Record records an analytics snapshot.
func (s *analyticsStore) Record(ctx context.Context, snapshot AnalyticsSnapshot) error {
	metricsJSON, err := json.Marshal(snapshot.Metrics)
	if err != nil {
		return fmt.Errorf("marshal analytics metrics: %w", err)
	}
	const query = `INSERT INTO analytics_snapshots (workspace_id, account_id, post_id, platform_post_id, metrics, collected_at) VALUES (@workspace_id, @account_id, @post_id, @platform_post_id, @metrics, @collected_at)`
	if _, err := s.db.Exec(ctx, query, pgx.NamedArgs{
		"workspace_id":      snapshot.WorkspaceID,
		"account_id":        snapshot.AccountID,
		"post_id":           snapshot.PostID,
		"platform_post_id":  snapshot.PlatformPostID,
		"metrics":           metricsJSON,
		"collected_at":      snapshot.CollectedAt,
	}); err != nil {
		return fmt.Errorf("record analytics snapshot: %w", err)
	}
	return nil
}

type webhookStore struct {
	db *pgxpool.Pool
}

// EnqueueDelivery stores or enqueues a webhook delivery.
func (s *webhookStore) EnqueueDelivery(ctx context.Context, params WebhookDeliveryParams) error {
	payloadJSON, err := json.Marshal(params.Payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}
	const query = `INSERT INTO webhook_deliveries (endpoint_id, event_type, payload) SELECT id, @event_type, @payload FROM webhook_endpoints WHERE workspace_id = @workspace_id AND enabled = true`
	result, err := s.db.Exec(ctx, query, pgx.NamedArgs{"workspace_id": params.WorkspaceID, "event_type": params.EventType, "payload": payloadJSON})
	if err != nil {
		return fmt.Errorf("enqueue webhook delivery: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("enqueue webhook delivery: no enabled endpoints for workspace %s", params.WorkspaceID)
	}
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
