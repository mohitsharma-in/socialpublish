# SocialPublish — Complete Implementation Plan
# SaaS Social Media Publishing SDK (Go) + Kubernetes Deployment

**Target audience**: This document is a complete, ordered implementation plan for Claude Opus / Codex.
**Review standard**: Senior Go engineer. Every code snippet is production-intentional. No shortcuts.

---

## Meta Instructions for Codex

- Go version: **1.22** (use `range over integer`, `slices`, `maps` stdlib packages)
- Module path: `github.com/yourorg/socialpublish`
- All errors wrapped with `%w`; never `errors.New` where context is available
- No `panic` in library code; panics only in `main()` for unrecoverable startup failures
- No `init()` doing real work — only stdlib registration (e.g. `pprof`)
- No global mutable state; pass dependencies explicitly
- Every exported function and type has a godoc comment
- All `context.Context` is first parameter, never stored in structs
- Use `any` not `interface{}`; use generics where it eliminates real duplication
- Use `errors.As` / `errors.Is` for error type checks — never type-assert errors directly
- No naked goroutines; use `golang.org/x/sync/errgroup` or a supervised worker pool
- Graceful shutdown via `signal.NotifyContext` — always drain in-flight work
- Database migrations via `golang-migrate/migrate` — never auto-migrate in production
- Tests use `testify/require` for fatal assertions, `testify/assert` for non-fatal
- HTTP handlers tested via `httptest.NewRecorder`, not real ports
- All durations are named constants or config fields — never magic numbers inline

---

## Part 1: Repository & Module Structure

```
socialpublish/
│
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml
│
├── sdk/                          ← PUBLIC SDK (what customers import)
│   ├── client.go
│   ├── options.go
│   ├── errors.go
│   ├── pagination.go
│   ├── webhook.go
│   └── services/
│       ├── media/
│       │   ├── service.go
│       │   ├── types.go
│       │   └── upload.go
│       ├── post/
│       │   ├── service.go
│       │   ├── builder.go
│       │   └── types.go
│       ├── account/
│       │   ├── service.go
│       │   └── types.go
│       ├── schedule/
│       │   ├── service.go
│       │   └── types.go
│       └── analytics/
│           ├── service.go
│           └── types.go
│
├── internal/                     ← SERVER-SIDE (not importable by SDK consumers)
│   ├── api/
│   │   ├── server.go             ← chi router wiring
│   │   ├── middleware/
│   │   │   ├── auth.go           ← JWT + API key validation
│   │   │   ├── ratelimit.go      ← per-workspace sliding window
│   │   │   ├── tenant.go         ← inject workspace into ctx
│   │   │   └── requestid.go
│   │   └── handler/
│   │       ├── media.go
│   │       ├── post.go
│   │       ├── account.go
│   │       ├── schedule.go
│   │       ├── analytics.go
│   │       └── webhook.go
│   ├── platform/
│   │   ├── adapter.go            ← PlatformAdapter interface
│   │   ├── registry.go           ← adapter registry (no global map)
│   │   ├── instagram/
│   │   │   ├── adapter.go
│   │   │   ├── container.go
│   │   │   └── types.go
│   │   └── youtube/
│   │       ├── adapter.go
│   │       ├── upload.go
│   │       └── types.go
│   ├── worker/
│   │   ├── pool.go               ← supervised worker pool
│   │   ├── transcode.go          ← FFmpeg job handler
│   │   ├── publish.go            ← platform publish job handler
│   │   ├── refresh.go            ← token refresh job handler
│   │   └── analytics.go          ← metrics poll job handler
│   ├── store/
│   │   ├── media.go
│   │   ├── post.go
│   │   ├── account.go
│   │   ├── schedule.go
│   │   ├── analytics.go
│   │   └── workspace.go
│   ├── queue/
│   │   ├── queue.go              ← Queue interface
│   │   └── asynq.go              ← asynq implementation
│   ├── token/
│   │   ├── store.go              ← TokenStore interface
│   │   └── postgres.go           ← AES-GCM encrypted token storage
│   ├── ffmpeg/
│   │   ├── runner.go
│   │   └── presets.go
│   ├── storage/
│   │   ├── storage.go            ← ObjectStorage interface
│   │   └── s3.go                 ← R2 / S3 implementation
│   └── tenant/
│       ├── context.go            ← workspace ctx key + accessors
│       └── types.go
│
├── migrations/                   ← SQL migrations (golang-migrate)
│   ├── 000001_create_workspaces.up.sql
│   ├── 000001_create_workspaces.down.sql
│   ├── 000002_create_accounts.up.sql
│   └── ...
│
├── deploy/
│   ├── k8s/
│   │   ├── namespace.yaml
│   │   ├── api/
│   │   │   ├── deployment.yaml
│   │   │   ├── service.yaml
│   │   │   └── hpa.yaml
│   │   ├── worker/
│   │   │   ├── deployment.yaml
│   │   │   └── hpa.yaml
│   │   ├── redis/
│   │   │   └── statefulset.yaml
│   │   ├── postgres/
│   │   │   └── statefulset.yaml  ← dev only; use managed in prod
│   │   ├── ingress/
│   │   │   └── ingress.yaml
│   │   ├── configmap.yaml
│   │   └── secrets.yaml          ← ExternalSecret (ESO), not plain Secret
│   └── helm/
│       └── socialpublish/        ← full Helm chart
│
└── cmd/
    ├── server/
    │   └── main.go               ← API server entrypoint
    ├── worker/
    │   └── main.go               ← Worker entrypoint
    └── migrate/
        └── main.go               ← Migration runner (run as K8s Job)
```

---

## Part 2: SaaS Data Model

### Core Concepts

```
Workspace  (= tenant / org)
  └── has many: Accounts (connected social accounts)
  └── has many: ApiKeys
  └── has many: Media
  └── has many: Posts
  └── has many: WebhookEndpoints
  └── has one:  Subscription (billing plan)

Post
  └── has many: Targets (one per platform account)
  └── belongs to: Workspace

Target
  └── belongs to: Post
  └── belongs to: Account
  └── has one:  TargetStatus (publish result)
```

### SQL Schema (write these migrations in order)

```sql
-- 000001: workspaces
CREATE TABLE workspaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    plan        TEXT NOT NULL DEFAULT 'free',  -- free | starter | pro | enterprise
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 000002: api_keys (hashed; never store plaintext)
CREATE TABLE api_keys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key_hash      TEXT NOT NULL UNIQUE,        -- bcrypt hash of "sp_live_xxx"
    key_prefix    TEXT NOT NULL,               -- "sp_live_" prefix for display
    name          TEXT NOT NULL,
    last_used_at  TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX api_keys_workspace_id_idx ON api_keys(workspace_id);

-- 000003: accounts (connected social accounts)
CREATE TABLE accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    platform        TEXT NOT NULL,   -- instagram | youtube
    platform_user_id TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    avatar_url      TEXT,
    token_id        UUID NOT NULL,   -- FK into encrypted_tokens
    token_expires_at TIMESTAMPTZ,
    token_healthy   BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, platform, platform_user_id)
);
CREATE INDEX accounts_workspace_id_idx ON accounts(workspace_id);

-- 000004: encrypted_tokens (AES-GCM; key from KMS/Vault)
CREATE TABLE encrypted_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ciphertext      BYTEA NOT NULL,
    key_version     INT NOT NULL DEFAULT 1,  -- for key rotation
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 000005: media
CREATE TABLE media (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'uploading',
    media_type      TEXT NOT NULL,  -- video | image
    original_key    TEXT NOT NULL,  -- S3/R2 object key
    mime_type       TEXT NOT NULL,
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    duration_ms     INT,
    formats         JSONB NOT NULL DEFAULT '{}',
    thumbnail_key   TEXT,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX media_workspace_id_status_idx ON media(workspace_id, status);

-- 000006: posts
CREATE TABLE posts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'draft',
    media_ids       UUID[] NOT NULL DEFAULT '{}',
    scheduled_at    TIMESTAMPTZ,
    published_at    TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX posts_workspace_status_idx     ON posts(workspace_id, status);
CREATE INDEX posts_scheduled_at_idx         ON posts(scheduled_at) WHERE status = 'scheduled';

-- 000007: post_targets
CREATE TABLE post_targets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id           UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    account_id        UUID NOT NULL REFERENCES accounts(id),
    platform          TEXT NOT NULL,
    format            TEXT NOT NULL,
    config            JSONB NOT NULL DEFAULT '{}',   -- platform-specific params
    status            TEXT NOT NULL DEFAULT 'pending',
    platform_post_id  TEXT,
    permalink         TEXT,
    failure_reason    TEXT,
    attempt_count     INT NOT NULL DEFAULT 0,
    last_attempted_at TIMESTAMPTZ,
    published_at      TIMESTAMPTZ
);
CREATE INDEX post_targets_post_id_idx ON post_targets(post_id);

-- 000008: webhook_endpoints
CREATE TABLE webhook_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    secret_hash     TEXT NOT NULL,   -- for HMAC signing; bcrypt of raw secret
    secret_enc      BYTEA NOT NULL,  -- AES-GCM encrypted raw secret (for signing)
    events          TEXT[] NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 000009: webhook_deliveries (audit log)
CREATE TABLE webhook_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id     UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    response_status INT,
    attempt_count   INT NOT NULL DEFAULT 0,
    delivered_at    TIMESTAMPTZ,
    next_retry_at   TIMESTAMPTZ
);
```

---

## Part 3: SDK Design (Public API)

### 3.1 `sdk/client.go`

```go
package socialpublish

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/yourorg/socialpublish/sdk/services/account"
	"github.com/yourorg/socialpublish/sdk/services/analytics"
	"github.com/yourorg/socialpublish/sdk/services/media"
	"github.com/yourorg/socialpublish/sdk/services/post"
	"github.com/yourorg/socialpublish/sdk/services/schedule"
)

const (
	defaultBaseURL = "https://api.socialpublish.io"
	defaultTimeout = 30 * time.Second
	sdkVersion     = "0.1.0"
)

// Client is the root entry point for the SocialPublish SDK.
// It is safe for concurrent use. Create once; reuse everywhere.
type Client struct {
	cfg       config
	transport *transport // shared HTTP transport with auth + retry

	accounts  account.Service
	media     media.Service
	posts     post.Service
	schedules schedule.Service
	analytics analytics.Service
}

// New creates a Client. It reads SOCIALPUBLISH_API_KEY from the environment
// if WithAPIKey is not provided. Returns an error only for invalid config.
func New(opts ...Option) (*Client, error) {
	cfg := config{
		baseURL:      defaultBaseURL,
		timeout:      defaultTimeout,
		maxRetries:   3,
		retryWaitMin: 500 * time.Millisecond,
		retryWaitMax: 30 * time.Second,
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.apiKey == "" {
		cfg.apiKey = os.Getenv("SOCIALPUBLISH_API_KEY")
	}
	if cfg.apiKey == "" {
		return nil, fmt.Errorf("socialpublish: API key required; use WithAPIKey or set SOCIALPUBLISH_API_KEY")
	}
	cfg.baseURL = strings.TrimRight(cfg.baseURL, "/")

	t := newTransport(cfg)

	return &Client{
		cfg:       cfg,
		transport: t,
		accounts:  account.New(t),
		media:     media.New(t),
		posts:     post.New(t),
		schedules: schedule.New(t),
		analytics: analytics.New(t),
	}, nil
}

// Accounts returns the account management service.
func (c *Client) Accounts() account.Service { return c.accounts }

// Media returns the media upload and transcode service.
func (c *Client) Media() media.Service { return c.media }

// Posts returns the post lifecycle management service.
func (c *Client) Posts() post.Service { return c.posts }

// Schedules returns the scheduling and calendar service.
func (c *Client) Schedules() schedule.Service { return c.schedules }

// Analytics returns the analytics and metrics service.
func (c *Client) Analytics() analytics.Service { return c.analytics }
```

### 3.2 `sdk/options.go`

```go
package socialpublish

import (
	"net/http"
	"time"
)

// Option configures a Client.
type Option func(*config)

type config struct {
	apiKey       string
	baseURL      string
	timeout      time.Duration
	maxRetries   int
	retryWaitMin time.Duration
	retryWaitMax time.Duration
	httpClient   *http.Client  // nil = use default
}

// WithAPIKey sets the API key. Prefer SOCIALPUBLISH_API_KEY env for production.
func WithAPIKey(key string) Option { return func(c *config) { c.apiKey = key } }

// WithBaseURL overrides the API base URL. Useful for self-hosted deployments.
func WithBaseURL(url string) Option { return func(c *config) { c.baseURL = url } }

// WithTimeout sets the per-request HTTP timeout. Default: 30s.
func WithTimeout(d time.Duration) Option { return func(c *config) { c.timeout = d } }

// WithHTTPClient injects a custom *http.Client. The SDK wraps its transport,
// so auth headers are still injected. Use this for test instrumentation.
func WithHTTPClient(hc *http.Client) Option { return func(c *config) { c.httpClient = hc } }

// WithMaxRetries sets the maximum number of retry attempts for 429 and 5xx responses.
// Default: 3. Set 0 to disable retries.
func WithMaxRetries(n int) Option { return func(c *config) { c.maxRetries = n } }
```

### 3.3 `sdk/errors.go`

```go
package socialpublish

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Code classifies API errors. Use errors.As to extract, then switch on Code.
type Code string

const (
	CodeUnauthorized   Code = "unauthorized"
	CodeForbidden      Code = "forbidden"
	CodeNotFound       Code = "not_found"
	CodeRateLimit      Code = "rate_limit"
	CodeValidation     Code = "validation_error"
	CodePlatformError  Code = "platform_error"
	CodeMediaNotReady  Code = "media_not_ready"
	CodeTranscodeFail  Code = "transcode_failed"
	CodePublishFailed  Code = "publish_failed"
	CodeTokenExpired   Code = "token_expired"
	CodeQuotaExceeded  Code = "quota_exceeded"
	CodeInternal       Code = "internal_error"
)

// Error is the structured error type returned by all SDK methods.
// It implements the error interface and supports errors.Is / errors.As.
type Error struct {
	Code       Code              `json:"code"`
	Message    string            `json:"message"`
	HTTPStatus int               `json:"http_status"`
	RequestID  string            `json:"request_id"`
	Platform   string            `json:"platform,omitempty"`
	RetryAfter *time.Duration    `json:"-"`
	Detail     map[string]any    `json:"detail,omitempty"`
}

func (e *Error) Error() string {
	if e.Platform != "" {
		return fmt.Sprintf("[%s/%s] %s (request_id=%s)", e.Platform, e.Code, e.Message, e.RequestID)
	}
	return fmt.Sprintf("[%s] %s (request_id=%s)", e.Code, e.Message, e.RequestID)
}

// Is matches another *Error by Code only, enabling errors.Is(err, ErrNotFound).
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// IsRetryable reports whether the error is safe to retry.
func (e *Error) IsRetryable() bool {
	return e.HTTPStatus == http.StatusTooManyRequests || e.HTTPStatus >= 500
}

// Sentinel errors. Use with errors.Is(err, socialpublish.ErrNotFound).
var (
	ErrUnauthorized  = &Error{Code: CodeUnauthorized, HTTPStatus: 401}
	ErrForbidden     = &Error{Code: CodeForbidden, HTTPStatus: 403}
	ErrNotFound      = &Error{Code: CodeNotFound, HTTPStatus: 404}
	ErrRateLimit     = &Error{Code: CodeRateLimit, HTTPStatus: 429}
	ErrMediaNotReady = &Error{Code: CodeMediaNotReady}
	ErrTokenExpired  = &Error{Code: CodeTokenExpired}
	ErrPublishFailed = &Error{Code: CodePublishFailed}
)

// ValidationError wraps field-level validation failures.
// Returned when the API rejects the request payload (HTTP 422).
type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

// FieldError describes a single field validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		msgs[i] = f.Field + ": " + f.Message
	}
	return "validation failed: " + strings.Join(msgs, "; ")
}
```

### 3.4 `sdk/services/post/builder.go` — Fluent Builder

The builder is the SDK's most visible surface. Design it for discoverability in IDE autocomplete.

```go
package post

import (
	"fmt"
	"time"
)

// Builder constructs a CreateRequest using a fluent API.
// Obtain one via NewPost(). Not safe for concurrent use.
type Builder struct {
	req    CreateRequest
	errors []error // collect all errors; surface them in Build()
}

// NewPost returns a fresh Builder.
//
// Example:
//
//	req, err := post.NewPost().
//	    WithMedia("med_abc").
//	    ForInstagram("acc_ig_001").AsReel("Caption #go").Done().
//	    ForYouTube("acc_yt_001").AsShort("Title", "Desc").WithPrivacy(PrivacyPublic).Done().
//	    ScheduleAt(time.Now().Add(24 * time.Hour)).
//	    Build()
func NewPost() *Builder {
	return &Builder{}
}

// WithMedia attaches one or more pre-uploaded media IDs to the post.
// At least one media ID is required.
func (b *Builder) WithMedia(ids ...string) *Builder {
	b.req.MediaIDs = append(b.req.MediaIDs, ids...)
	return b
}

// ScheduleAt sets the UTC publish time. Conflicts with PublishNow.
func (b *Builder) ScheduleAt(t time.Time) *Builder {
	if t.IsZero() {
		b.errors = append(b.errors, fmt.Errorf("builder: ScheduleAt: zero time is invalid"))
		return b
	}
	if time.Until(t) < time.Minute {
		b.errors = append(b.errors, fmt.Errorf("builder: ScheduleAt: time must be at least 1 minute in the future"))
		return b
	}
	b.req.ScheduledAt = &t
	return b
}

// PublishNow marks the post for immediate publishing after creation.
func (b *Builder) PublishNow() *Builder {
	b.req.PublishImmediately = true
	return b
}

// WithMetadata attaches an arbitrary key/value pair. Useful for tracking
// external IDs (e.g. your CMS article ID).
func (b *Builder) WithMetadata(key, value string) *Builder {
	if b.req.Metadata == nil {
		b.req.Metadata = make(map[string]string)
	}
	b.req.Metadata[key] = value
	return b
}

// ForInstagram returns a platform-specific sub-builder. Call Done() to return
// to the main Builder.
func (b *Builder) ForInstagram(accountID string) *InstagramBuilder {
	if accountID == "" {
		b.errors = append(b.errors, fmt.Errorf("builder: ForInstagram: accountID is required"))
	}
	return &InstagramBuilder{parent: b, accountID: accountID}
}

// ForYouTube returns a platform-specific sub-builder. Call Done() to return
// to the main Builder.
func (b *Builder) ForYouTube(accountID string) *YouTubeBuilder {
	if accountID == "" {
		b.errors = append(b.errors, fmt.Errorf("builder: ForYouTube: accountID is required"))
	}
	return &YouTubeBuilder{parent: b, accountID: accountID}
}

// Build validates the request and returns it.
// Returns the first accumulated builder error if any exist.
func (b *Builder) Build() (*CreateRequest, error) {
	if len(b.errors) > 0 {
		return nil, b.errors[0] // surface the first; programmer error
	}
	if len(b.req.MediaIDs) == 0 {
		return nil, fmt.Errorf("builder: at least one media ID is required")
	}
	if len(b.req.Targets) == 0 {
		return nil, fmt.Errorf("builder: at least one platform target is required")
	}
	if b.req.ScheduledAt != nil && b.req.PublishImmediately {
		return nil, fmt.Errorf("builder: ScheduleAt and PublishNow are mutually exclusive")
	}
	req := b.req // copy; builder should not be mutated after Build
	return &req, nil
}

// ── Instagram sub-builder ────────────────────────────────────────────────────

// InstagramBuilder configures a single Instagram publish target.
// All methods return *InstagramBuilder for chaining. Call Done() to commit.
type InstagramBuilder struct {
	parent    *Builder
	accountID string
	t         InstagramConfig
}

// AsReel configures the target as an Instagram Reel.
// caption max: 2200 characters (validated server-side).
func (b *InstagramBuilder) AsReel(caption string) *InstagramBuilder {
	b.t.Format = FormatReel
	b.t.Caption = caption
	return b
}

// AsStory configures the target as an Instagram Story.
func (b *InstagramBuilder) AsStory() *InstagramBuilder {
	b.t.Format = FormatStory
	return b
}

// AsCarousel configures the target as an Instagram carousel post.
func (b *InstagramBuilder) AsCarousel(caption string) *InstagramBuilder {
	b.t.Format = FormatCarousel
	b.t.Caption = caption
	return b
}

// WithCoverTimestamp sets the video cover frame offset in seconds.
func (b *InstagramBuilder) WithCoverTimestamp(secs float64) *InstagramBuilder {
	b.t.CoverTimestampSecs = &secs
	return b
}

// WithCollaborators adds Instagram usernames as collaborators (without the @ prefix).
func (b *InstagramBuilder) WithCollaborators(handles ...string) *InstagramBuilder {
	b.t.Collaborators = append(b.t.Collaborators, handles...)
	return b
}

// ShareToFeed controls whether a Reel also appears on the main feed. Default: true.
func (b *InstagramBuilder) ShareToFeed(v bool) *InstagramBuilder {
	b.t.ShareToFeed = &v
	return b
}

// Done commits this target to the parent Builder.
func (b *InstagramBuilder) Done() *Builder {
	b.parent.req.Targets = append(b.parent.req.Targets, Target{
		AccountID: b.accountID,
		Platform:  PlatformInstagram,
		Instagram: &b.t,
	})
	return b.parent
}

// ── YouTube sub-builder ──────────────────────────────────────────────────────

// YouTubeBuilder configures a single YouTube publish target.
type YouTubeBuilder struct {
	parent    *Builder
	accountID string
	t         YouTubeConfig
}

// AsShort configures the target as a YouTube Short (≤180s, 9:16 aspect).
// YouTube auto-classifies based on dimensions + duration; no special API flag needed.
// "#Shorts" is appended to description automatically.
func (b *YouTubeBuilder) AsShort(title, description string) *YouTubeBuilder {
	b.t.Format = FormatShort
	b.t.Title = title
	b.t.Description = description
	return b
}

// AsVideo configures the target as a standard YouTube video.
func (b *YouTubeBuilder) AsVideo(title, description string) *YouTubeBuilder {
	b.t.Format = FormatVideo
	b.t.Title = title
	b.t.Description = description
	return b
}

// WithPrivacy sets the video privacy. Default: PrivacyPublic.
func (b *YouTubeBuilder) WithPrivacy(p Privacy) *YouTubeBuilder {
	b.t.Privacy = p
	return b
}

// WithTags sets the video tags. Max 500 chars total across all tags.
func (b *YouTubeBuilder) WithTags(tags ...string) *YouTubeBuilder {
	b.t.Tags = append(b.t.Tags, tags...)
	return b
}

// WithCategory sets the YouTube category ID (e.g. 28 = Science & Technology).
func (b *YouTubeBuilder) WithCategory(id int) *YouTubeBuilder {
	b.t.CategoryID = id
	return b
}

// AddToPlaylist adds the video to the given playlist after publishing.
// Multiple calls are additive.
func (b *YouTubeBuilder) AddToPlaylist(playlistID string) *YouTubeBuilder {
	b.t.PlaylistIDs = append(b.t.PlaylistIDs, playlistID)
	return b
}

// MadeForKids sets the "made for kids" designation. Affects commenting and features.
func (b *YouTubeBuilder) MadeForKids(v bool) *YouTubeBuilder {
	b.t.MadeForKids = v
	return b
}

// NotifySubscribers controls whether subscribers receive a notification.
// Default: true. Set false for batch publishing or low-priority updates.
func (b *YouTubeBuilder) NotifySubscribers(v bool) *YouTubeBuilder {
	b.t.NotifySubscribers = v
	return b
}

// Done commits this target to the parent Builder.
func (b *YouTubeBuilder) Done() *Builder {
	b.parent.req.Targets = append(b.parent.req.Targets, Target{
		AccountID: b.accountID,
		Platform:  PlatformYouTube,
		YouTube:   &b.t,
	})
	return b.parent
}
```

### 3.5 `sdk/services/post/types.go`

```go
package post

import "time"

// Platform identifies a social media platform.
type Platform string

const (
	PlatformInstagram Platform = "instagram"
	PlatformYouTube   Platform = "youtube"
)

// Format identifies the content format within a platform.
type Format string

const (
	FormatReel     Format = "reel"
	FormatStory    Format = "story"
	FormatCarousel Format = "carousel"
	FormatShort    Format = "short"
	FormatVideo    Format = "video"
)

// Privacy controls YouTube video visibility.
type Privacy string

const (
	PrivacyPublic   Privacy = "public"
	PrivacyUnlisted Privacy = "unlisted"
	PrivacyPrivate  Privacy = "private"
)

// Status represents the post lifecycle state.
type Status string

const (
	StatusDraft      Status = "draft"
	StatusScheduled  Status = "scheduled"
	StatusPublishing Status = "publishing"
	StatusPublished  Status = "published"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// InstagramConfig holds Instagram-specific publish parameters.
type InstagramConfig struct {
	Format             Format   `json:"format"`
	Caption            string   `json:"caption,omitempty"`
	CoverTimestampSecs *float64 `json:"cover_timestamp_secs,omitempty"`
	Collaborators      []string `json:"collaborators,omitempty"`
	LocationID         string   `json:"location_id,omitempty"`
	ShareToFeed        *bool    `json:"share_to_feed,omitempty"`
}

// YouTubeConfig holds YouTube-specific publish parameters.
type YouTubeConfig struct {
	Format            Format   `json:"format"`
	Title             string   `json:"title"`
	Description       string   `json:"description,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Privacy           Privacy  `json:"privacy"`
	CategoryID        int      `json:"category_id,omitempty"`
	PlaylistIDs       []string `json:"playlist_ids,omitempty"`
	MadeForKids       bool     `json:"made_for_kids"`
	NotifySubscribers bool     `json:"notify_subscribers"`
}

// Target is a single platform + account publishing destination.
type Target struct {
	AccountID string           `json:"account_id"`
	Platform  Platform         `json:"platform"`
	Instagram *InstagramConfig `json:"instagram,omitempty"`
	YouTube   *YouTubeConfig   `json:"youtube,omitempty"`
}

// CreateRequest is the input to Service.Create. Build one via NewPost().
type CreateRequest struct {
	MediaIDs           []string          `json:"media_ids"`
	Targets            []Target          `json:"targets"`
	ScheduledAt        *time.Time        `json:"scheduled_at,omitempty"`
	PublishImmediately bool              `json:"publish_immediately,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// TargetStatus is the per-platform publish outcome for a post.
type TargetStatus struct {
	AccountID      string     `json:"account_id"`
	Platform       Platform   `json:"platform"`
	Status         Status     `json:"status"`
	PlatformPostID string     `json:"platform_post_id,omitempty"`
	Permalink      string     `json:"permalink,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	FailureReason  string     `json:"failure_reason,omitempty"`
	AttemptCount   int        `json:"attempt_count"`
}

// Post is the full post resource returned by the API.
type Post struct {
	ID          string            `json:"post_id"`
	Status      Status            `json:"status"`
	MediaIDs    []string          `json:"media_ids"`
	Targets     []TargetStatus    `json:"targets"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	PublishedAt *time.Time        `json:"published_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// IsFullyPublished reports whether every target has succeeded.
func (p *Post) IsFullyPublished() bool {
	for i := range p.Targets {
		if p.Targets[i].Status != StatusPublished {
			return false
		}
	}
	return len(p.Targets) > 0
}

// TargetFor returns a pointer to the status of the given platform, or nil.
func (p *Post) TargetFor(platform Platform) *TargetStatus {
	for i := range p.Targets {
		if p.Targets[i].Platform == platform {
			return &p.Targets[i]
		}
	}
	return nil
}

// UpdateRequest fields are all optional; omit to leave unchanged.
type UpdateRequest struct {
	MediaIDs    []string          `json:"media_ids,omitempty"`
	Targets     []Target          `json:"targets,omitempty"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ListParams filters and paginates post listing.
type ListParams struct {
	Limit    int       // 1–100; default 20
	Cursor   string
	Status   Status
	Platform Platform
	From     *time.Time
	To       *time.Time
}

// Page is a paginated list of posts.
type Page struct {
	Items      []*Post `json:"items"`
	NextCursor string  `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
	Total      int     `json:"total"`
}

// Implement sdk.Pager for the generic iterator.
func (p *Page) GetItems() []*Post   { return p.Items }
func (p *Page) GetNextCursor() string { return p.NextCursor }
func (p *Page) GetHasMore() bool    { return p.HasMore }
```

### 3.6 `sdk/pagination.go` — Generic Iterator

```go
package socialpublish

import "context"

// Pager is implemented by all paginated response types.
type Pager[T any] interface {
	GetItems() []T
	GetNextCursor() string
	GetHasMore() bool
}

// FetchFn is a function that fetches one page given a cursor.
type FetchFn[T any] func(ctx context.Context, cursor string) (Pager[T], error)

// Iter walks all pages lazily. Fetch the next page only when the buffer
// is exhausted. Zero allocations per item after the first page load.
//
//	iter := socialpublish.Iter(ctx, func(ctx context.Context, cursor string) (socialpublish.Pager[*post.Post], error) {
//	    return client.Posts().List(ctx, &post.ListParams{Cursor: cursor, Limit: 50})
//	})
//	for iter.Next() {
//	    p := iter.Item()
//	}
//	if err := iter.Err(); err != nil { ... }
type Iter[T any] struct {
	ctx    context.Context
	fetch  FetchFn[T]
	cursor string
	items  []T
	pos    int
	done   bool
	err    error
}

// NewIter creates an Iter. Call Next() before Item().
func NewIter[T any](ctx context.Context, fn FetchFn[T]) *Iter[T] {
	return &Iter[T]{ctx: ctx, fetch: fn}
}

// Next advances to the next item. Returns false when exhausted or on error.
func (it *Iter[T]) Next() bool {
	if it.err != nil {
		return false
	}
	// still have items buffered
	if it.pos < len(it.items) {
		it.pos++
		return true
	}
	// exhausted and no more pages
	if it.done {
		return false
	}
	// fetch the next page
	page, err := it.fetch(it.ctx, it.cursor)
	if err != nil {
		it.err = err
		return false
	}
	it.items = page.GetItems()
	it.cursor = page.GetNextCursor()
	it.done = !page.GetHasMore()
	it.pos = 0
	if len(it.items) == 0 {
		return false
	}
	it.pos = 1
	return true
}

// Item returns the current item. Only valid after Next() returns true.
func (it *Iter[T]) Item() T { return it.items[it.pos-1] }

// Err returns the first error encountered during iteration, if any.
func (it *Iter[T]) Err() error { return it.err }
```

---

## Part 4: Internal Server

### 4.1 API Server (`internal/api/server.go`)

```go
package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/yourorg/socialpublish/internal/api/handler"
	"github.com/yourorg/socialpublish/internal/api/middleware"
	"github.com/yourorg/socialpublish/internal/store"
	"github.com/yourorg/socialpublish/internal/queue"
	"github.com/yourorg/socialpublish/internal/storage"
)

// Config holds all server configuration. Load from env / config file.
type Config struct {
	ListenAddr      string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Server wraps the HTTP server with dependencies.
type Server struct {
	cfg    Config
	router *chi.Mux
	http   *http.Server
}

// New assembles the server. All dependencies are explicit — no globals.
func New(
	cfg Config,
	stores store.Stores,
	q queue.Queue,
	obj storage.ObjectStorage,
) *Server {
	r := chi.NewRouter()

	// ── Global middleware ────────────────────────────────────────────────────
	r.Use(chimw.RequestID)      // sets X-Request-ID
	r.Use(middleware.RequestID) // copies chi request ID to our context key
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)      // catch panics in handlers (not in workers)
	r.Use(middleware.Logger)

	// ── Unauthenticated routes ───────────────────────────────────────────────
	r.Get("/health", handler.Health)
	r.Get("/readyz", handler.Readyz(stores))

	// ── Authenticated API (v1) ───────────────────────────────────────────────
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.Authenticate(stores.APIKeys))
		r.Use(middleware.InjectTenant(stores.Workspaces))
		r.Use(middleware.RateLimit(stores.Workspaces))

		// Accounts
		r.Route("/accounts", func(r chi.Router) {
			ah := handler.NewAccount(stores.Accounts, stores.Tokens)
			r.Get("/", ah.List)
			r.Post("/connect", ah.Connect)
			r.Get("/{accountID}", ah.Get)
			r.Delete("/{accountID}", ah.Delete)
			r.Get("/{accountID}/status", ah.Status)
		})

		// Media
		r.Route("/media", func(r chi.Router) {
			mh := handler.NewMedia(stores.Media, obj, q)
			r.Post("/upload", mh.Upload)
			r.Get("/", mh.List)
			r.Get("/{mediaID}", mh.Get)
			r.Delete("/{mediaID}", mh.Delete)
			r.Post("/{mediaID}/thumbnail", mh.SetThumbnail)
		})

		// Posts
		r.Route("/posts", func(r chi.Router) {
			ph := handler.NewPost(stores.Posts, stores.Media, q)
			r.Post("/", ph.Create)
			r.Get("/", ph.List)
			r.Get("/{postID}", ph.Get)
			r.Patch("/{postID}", ph.Update)
			r.Delete("/{postID}", ph.Delete)
			r.Post("/{postID}/publish", ph.Publish)
			r.Post("/{postID}/cancel", ph.Cancel)
			r.Post("/{postID}/duplicate", ph.Duplicate)
		})

		// Schedule
		r.Route("/schedule", func(r chi.Router) {
			sh := handler.NewSchedule(stores.Posts)
			r.Get("/", sh.Calendar)
			r.Get("/queue", sh.Queue)
			r.Get("/next-available", sh.NextAvailable)
		})

		// Analytics
		r.Route("/analytics", func(r chi.Router) {
			ah := handler.NewAnalytics(stores.Analytics)
			r.Get("/posts/{postID}", ah.Post)
			r.Get("/accounts/{accountID}", ah.Account)
			r.Get("/summary", ah.Summary)
		})

		// Webhooks
		r.Route("/webhooks", func(r chi.Router) {
			wh := handler.NewWebhook(stores.Webhooks, stores.Tokens)
			r.Post("/", wh.Create)
			r.Get("/", wh.List)
			r.Delete("/{webhookID}", wh.Delete)
			r.Post("/{webhookID}/test", wh.Test)
		})
	})

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		// Always set this — never leave BaseContext nil in production.
		BaseContext: func(_ net.Listener) context.Context {
			return context.Background()
		},
	}

	return &Server{cfg: cfg, router: r, http: srv}
}

// Run starts the server and blocks until ctx is cancelled, then drains.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	return s.http.Shutdown(shutCtx)
}
```

### 4.2 `cmd/server/main.go` — Entrypoint

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yourorg/socialpublish/internal/api"
	"github.com/yourorg/socialpublish/internal/store"
	"github.com/yourorg/socialpublish/internal/queue"
	"github.com/yourorg/socialpublish/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// Signal-aware root context — all child goroutines derive from this.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig() // reads env; panics on missing required fields

	db, err := store.OpenDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	stores := store.New(db)
	q := queue.NewAsynq(cfg.RedisAddr)
	obj, err := storage.NewS3(ctx, cfg.S3)
	if err != nil {
		return fmt.Errorf("open object storage: %w", err)
	}

	srv := api.New(api.Config{
		ListenAddr:      cfg.ListenAddr,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}, stores, q, obj)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("api server starting", "addr", cfg.ListenAddr)
		return srv.Run(ctx)
	})

	return g.Wait()
}
```

### 4.3 Worker Pool (`internal/worker/pool.go`)

```go
package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/yourorg/socialpublish/internal/platform"
	"github.com/yourorg/socialpublish/internal/store"
	"github.com/yourorg/socialpublish/internal/storage"
	"github.com/yourorg/socialpublish/internal/ffmpeg"
)

// Task type constants. Keep these stable — they're persisted in Redis.
const (
	TaskTranscode       = "media:transcode"
	TaskPublish         = "post:publish"
	TaskTokenRefresh    = "account:token_refresh"
	TaskAnalyticsPoll   = "analytics:poll"
	TaskWebhookDeliver  = "webhook:deliver"
)

// Pool is the background worker pool. It processes jobs from the queue.
type Pool struct {
	server   *asynq.Server
	mux      *asynq.ServeMux
}

// New creates a Pool with all handlers registered.
func New(
	redisAddr string,
	stores store.Stores,
	obj storage.ObjectStorage,
	adapters platform.Registry,
	ff *ffmpeg.Runner,
) *Pool {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			// Concurrency per task type — tune for your instance size.
			Queues: map[string]int{
				"critical": 10, // webhook delivery, token refresh
				"default":  5,  // publish
				"low":      2,  // analytics poll, transcode (CPU-heavy)
			},
			// Retry config is set per-job at enqueue time.
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				slog.Error("worker task failed",
					"type", task.Type(),
					"err", err,
				)
			}),
		},
	)

	mux := asynq.NewServeMux()
	mux.Handle(TaskTranscode,      NewTranscodeHandler(stores.Media, obj, ff))
	mux.Handle(TaskPublish,        NewPublishHandler(stores.Posts, stores.Accounts, stores.Tokens, adapters, stores.Webhooks))
	mux.Handle(TaskTokenRefresh,   NewTokenRefreshHandler(stores.Accounts, stores.Tokens, adapters))
	mux.Handle(TaskAnalyticsPoll,  NewAnalyticsHandler(stores.Analytics, stores.Accounts, stores.Tokens, adapters))
	mux.Handle(TaskWebhookDeliver, NewWebhookDeliverHandler(stores.Webhooks))

	return &Pool{server: srv, mux: mux}
}

// Run starts the worker pool. Blocks until ctx is done.
func (p *Pool) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := p.server.Run(p.mux); err != nil {
			errCh <- fmt.Errorf("worker pool: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		p.server.Shutdown()
		return nil
	}
}
```

### 4.4 Publish Worker (`internal/worker/publish.go`)

```go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/yourorg/socialpublish/internal/platform"
	"github.com/yourorg/socialpublish/internal/store"
	"github.com/yourorg/socialpublish/internal/token"
)

// PublishPayload is serialised into the job queue.
type PublishPayload struct {
	PostID   string `json:"post_id"`
	TargetID string `json:"target_id"`
}

type publishHandler struct {
	posts    store.PostStore
	accounts store.AccountStore
	tokens   token.Store
	adapters platform.Registry
	webhooks store.WebhookStore
}

func NewPublishHandler(
	posts store.PostStore,
	accounts store.AccountStore,
	tokens token.Store,
	adapters platform.Registry,
	webhooks store.WebhookStore,
) asynq.Handler {
	return &publishHandler{posts: posts, accounts: accounts, tokens: tokens, adapters: adapters, webhooks: webhooks}
}

func (h *publishHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var p PublishPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		// Malformed payload — do not retry.
		return fmt.Errorf("%w: unmarshal publish payload: %v", asynq.SkipRetry, err)
	}

	target, err := h.posts.GetTarget(ctx, p.TargetID)
	if err != nil {
		return fmt.Errorf("get target %s: %w", p.TargetID, err)
	}

	acc, err := h.accounts.Get(ctx, target.AccountID)
	if err != nil {
		return fmt.Errorf("get account %s: %w", target.AccountID, err)
	}

	accessToken, err := h.tokens.Decrypt(ctx, acc.TokenID)
	if err != nil {
		return fmt.Errorf("decrypt token for account %s: %w", acc.ID, err)
	}

	adapter, ok := h.adapters.Get(target.Platform)
	if !ok {
		return fmt.Errorf("%w: no adapter for platform %s", asynq.SkipRetry, target.Platform)
	}

	result, err := adapter.Publish(ctx, &platform.PublishRequest{
		AccountID:   acc.ID,
		AccessToken: accessToken,
		Target:      target,
	})
	if err != nil {
		// Mark the target as failed; increment attempt count.
		_ = h.posts.SetTargetFailed(ctx, p.TargetID, err.Error())
		// Propagate so asynq can decide to retry based on queue config.
		return fmt.Errorf("adapter publish: %w", err)
	}

	if err := h.posts.SetTargetPublished(ctx, p.TargetID, result.PlatformPostID, result.Permalink); err != nil {
		// Target is published but we failed to record it.
		// Log loudly; do NOT retry the publish (idempotency guard needed in adapters).
		slog.Error("failed to persist publish result — platform post is live",
			"target_id", p.TargetID,
			"platform_post_id", result.PlatformPostID,
			"err", err,
		)
		return fmt.Errorf("%w: persist publish result: %v", asynq.SkipRetry, err)
	}

	// Fire webhook asynchronously — failure here must not fail the publish job.
	h.webhooks.EnqueueDelivery(ctx, store.WebhookDeliveryParams{
		WorkspaceID: acc.WorkspaceID,
		EventType:   "post.published",
		Payload: map[string]any{
			"post_id":          p.PostID,
			"target_id":        p.TargetID,
			"platform":         target.Platform,
			"platform_post_id": result.PlatformPostID,
			"permalink":        result.Permalink,
		},
	})

	return nil
}
```

---

## Part 5: Platform Adapters

### 5.1 `internal/platform/adapter.go`

```go
package platform

import (
	"context"
	"time"

	"github.com/yourorg/socialpublish/internal/store"
)

// PublishRequest is the normalised input every adapter receives.
type PublishRequest struct {
	AccountID   string
	AccessToken string
	MediaURL    string // CDN URL of the transcoded variant
	Target      store.PostTarget
}

// PublishResult is what every adapter returns on success.
type PublishResult struct {
	PlatformPostID string
	Permalink      string
}

// OAuthToken is the refreshed credential pair from RefreshToken.
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// PlatformAdapter is the contract every platform must implement.
// Implementations must be safe for concurrent use.
type PlatformAdapter interface {
	// Platform returns the canonical platform identifier.
	Platform() string

	// Publish performs the full publish flow. Must be idempotent:
	// if the platform post already exists (due to a retry), return its ID.
	Publish(ctx context.Context, req *PublishRequest) (*PublishResult, error)

	// RefreshToken exchanges a refresh token for a new access token.
	RefreshToken(ctx context.Context, refreshToken string) (*OAuthToken, error)

	// FetchMetrics retrieves engagement metrics for a published post.
	FetchMetrics(ctx context.Context, accessToken, platformPostID string) (*Metrics, error)

	// ValidateTarget performs pre-publish validation of target config.
	// Called at post creation time; must not make network calls.
	ValidateTarget(target store.PostTarget) error
}

// Metrics is the normalised analytics payload from a platform.
type Metrics struct {
	Views    int64
	Likes    int64
	Comments int64
	Shares   int64
	Reach    int64
	// Platform-specific extras stored as JSON in the DB.
	Extra map[string]any
}

// Registry holds all registered adapters. Inject as a dependency.
type Registry struct {
	adapters map[string]PlatformAdapter
}

// NewRegistry builds a registry from the provided adapters.
func NewRegistry(adapters ...PlatformAdapter) Registry {
	m := make(map[string]PlatformAdapter, len(adapters))
	for _, a := range adapters {
		m[a.Platform()] = a
	}
	return Registry{adapters: m}
}

// Get returns the adapter for the given platform, and whether it was found.
func (r Registry) Get(platform string) (PlatformAdapter, bool) {
	a, ok := r.adapters[platform]
	return a, ok
}
```

---

## Part 6: SaaS Multi-Tenancy

### Tenant Context (`internal/tenant/context.go`)

```go
package tenant

import "context"

// key is unexported to prevent collisions with other packages.
type contextKey struct{}

// Workspace holds the authenticated workspace details.
type Workspace struct {
	ID   string
	Plan string // free | starter | pro | enterprise
}

// FromContext extracts the Workspace from ctx.
// Returns zero value and false if not present.
func FromContext(ctx context.Context) (Workspace, bool) {
	w, ok := ctx.Value(contextKey{}).(Workspace)
	return w, ok
}

// WithWorkspace stores a Workspace in ctx.
func WithWorkspace(ctx context.Context, w Workspace) context.Context {
	return context.WithValue(ctx, contextKey{}, w)
}

// MustFromContext extracts the Workspace or panics.
// Only use in handler code where the auth middleware guarantees its presence.
func MustFromContext(ctx context.Context) Workspace {
	w, ok := FromContext(ctx)
	if !ok {
		panic("tenant: workspace not in context — auth middleware not applied")
	}
	return w
}
```

### Rate Limit Middleware (`internal/api/middleware/ratelimit.go`)

```go
package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/yourorg/socialpublish/internal/tenant"
)

// PlanLimits maps a plan name to allowed requests per minute.
var PlanLimits = map[string]int{
	"free":       60,
	"starter":    300,
	"pro":        1000,
	"enterprise": 5000,
}

// RateLimiter is a sliding window rate limiter backed by Redis.
type RateLimiter interface {
	// Allow reports whether the workspace is within its rate limit.
	// It returns the remaining quota and the reset time.
	Allow(ctx context.Context, workspaceID string, limit int) (remaining int, reset time.Time, ok bool)
}

// RateLimit returns a middleware that enforces per-workspace rate limits.
func RateLimit(limiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ws := tenant.MustFromContext(r.Context())
			limit := PlanLimits[ws.Plan]
			if limit == 0 {
				limit = PlanLimits["free"]
			}

			remaining, reset, ok := limiter.Allow(r.Context(), ws.ID, limit)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))

			if !ok {
				w.Header().Set("Retry-After", strconv.FormatInt(time.Until(reset).Milliseconds()/1000+1, 10))
				http.Error(w, `{"code":"rate_limit","message":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

---

## Part 7: Hosting & Kubernetes Deployment

### 7.1 Infrastructure Overview

```
Production Topology (single-region; replicate for multi-region)

  ┌─────────────────────────────────────────────────────────────────────┐
  │  Kubernetes Cluster (EKS / GKE / AKS)                               │
  │                                                                      │
  │  ┌─────────────┐     ┌───────────────────┐     ┌─────────────────┐  │
  │  │  Ingress    │────▶│  API Deployment   │────▶│  Worker         │  │
  │  │  (nginx /   │     │  (3+ replicas)    │     │  Deployment     │  │
  │  │   Traefik)  │     │  HPA: 3–20        │     │  (2+ replicas)  │  │
  │  └─────────────┘     └───────────────────┘     └─────────────────┘  │
  │         │                    │                         │             │
  │  ┌──────▼──────┐    ┌────────▼──────┐        ┌────────▼──────────┐  │
  │  │  cert-manager│   │  PostgreSQL   │        │  Redis (asynq     │  │
  │  │  (TLS)      │   │  (managed:    │        │   queue + rate     │  │
  │  └─────────────┘   │  RDS/CloudSQL)│        │   limit counters)  │  │
  │                    └───────────────┘        └───────────────────┘  │
  │                                                                      │
  │  ┌────────────────────────────────────────────────────────────────┐  │
  │  │  External Services                                              │  │
  │  │  Cloudflare R2 (media storage)   Vault / AWS Secrets Manager   │  │
  │  │  (no egress fees for reads)      (token encryption keys)       │  │
  │  └────────────────────────────────────────────────────────────────┘  │
  └─────────────────────────────────────────────────────────────────────┘
```

### 7.2 Namespace and RBAC

```yaml
# deploy/k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: socialpublish
  labels:
    app.kubernetes.io/name: socialpublish
```

### 7.3 ConfigMap

```yaml
# deploy/k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: socialpublish-config
  namespace: socialpublish
data:
  LISTEN_ADDR: "0.0.0.0:8080"
  LOG_LEVEL: "info"
  REDIS_ADDR: "redis-service.socialpublish.svc.cluster.local:6379"
  S3_REGION: "auto"
  S3_ENDPOINT: "https://<account>.r2.cloudflarestorage.com"
  S3_BUCKET: "socialpublish-media"
  FFMPEG_BIN: "/usr/bin/ffmpeg"
  # Never put secrets here. Use ExternalSecret.
```

### 7.4 Secrets via External Secrets Operator (ESO)

```yaml
# deploy/k8s/secrets.yaml
# Requires External Secrets Operator installed in the cluster.
# References AWS Secrets Manager; swap backend for Vault/GCP Secret Manager.
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: socialpublish-secrets
  namespace: socialpublish
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: socialpublish-secrets
    creationPolicy: Owner
  data:
    - secretKey: DATABASE_URL
      remoteRef:
        key: socialpublish/prod
        property: database_url
    - secretKey: TOKEN_ENCRYPTION_KEY
      remoteRef:
        key: socialpublish/prod
        property: token_encryption_key
    - secretKey: INSTAGRAM_APP_ID
      remoteRef:
        key: socialpublish/prod
        property: instagram_app_id
    - secretKey: INSTAGRAM_APP_SECRET
      remoteRef:
        key: socialpublish/prod
        property: instagram_app_secret
    - secretKey: YOUTUBE_CLIENT_ID
      remoteRef:
        key: socialpublish/prod
        property: youtube_client_id
    - secretKey: YOUTUBE_CLIENT_SECRET
      remoteRef:
        key: socialpublish/prod
        property: youtube_client_secret
    - secretKey: S3_ACCESS_KEY_ID
      remoteRef:
        key: socialpublish/prod
        property: s3_access_key_id
    - secretKey: S3_SECRET_ACCESS_KEY
      remoteRef:
        key: socialpublish/prod
        property: s3_secret_access_key
```

### 7.5 API Deployment

```yaml
# deploy/k8s/api/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: socialpublish-api
  namespace: socialpublish
  labels:
    app: socialpublish-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: socialpublish-api
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0        # zero-downtime deployments
  template:
    metadata:
      labels:
        app: socialpublish-api
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port:   "9090"
        prometheus.io/path:   "/metrics"
    spec:
      terminationGracePeriodSeconds: 40  # > server WriteTimeout (30s)
      containers:
        - name: api
          image: ghcr.io/yourorg/socialpublish-api:${IMAGE_TAG}
          imagePullPolicy: Always
          ports:
            - name: http
              containerPort: 8080
            - name: metrics
              containerPort: 9090
          envFrom:
            - configMapRef:
                name: socialpublish-config
            - secretRef:
                name: socialpublish-secrets
          resources:
            requests:
              cpu:    "250m"
              memory: "256Mi"
            limits:
              cpu:    "1000m"
              memory: "512Mi"
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
            failureThreshold: 3
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 15
            periodSeconds: 20
            failureThreshold: 3
          lifecycle:
            preStop:
              exec:
                # Give the LB time to drain before the process exits.
                command: ["/bin/sleep", "5"]
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app
                      operator: In
                      values: [socialpublish-api]
                topologyKey: kubernetes.io/hostname
```

### 7.6 API HorizontalPodAutoscaler

```yaml
# deploy/k8s/api/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: socialpublish-api
  namespace: socialpublish
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: socialpublish-api
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 65
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Pods
          value: 3
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300  # 5 min cool-down prevents flapping
```

### 7.7 Worker Deployment

```yaml
# deploy/k8s/worker/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: socialpublish-worker
  namespace: socialpublish
  labels:
    app: socialpublish-worker
spec:
  replicas: 2
  selector:
    matchLabels:
      app: socialpublish-worker
  template:
    metadata:
      labels:
        app: socialpublish-worker
    spec:
      terminationGracePeriodSeconds: 120  # workers may be mid-transcode
      containers:
        - name: worker
          image: ghcr.io/yourorg/socialpublish-worker:${IMAGE_TAG}
          imagePullPolicy: Always
          envFrom:
            - configMapRef:
                name: socialpublish-config
            - secretRef:
                name: socialpublish-secrets
          resources:
            requests:
              cpu:    "500m"
              memory: "512Mi"
            limits:
              cpu:    "2000m"
              memory: "2Gi"     # FFmpeg needs memory for video buffering
          # FFmpeg binary must be in the container image.
          # Use a multi-stage Dockerfile: builder + ffmpeg layer.
```

### 7.8 Migration Job (run before every deploy)

```yaml
# deploy/k8s/migrate-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: socialpublish-migrate-${BUILD_ID}
  namespace: socialpublish
spec:
  backoffLimit: 1
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: migrate
          image: ghcr.io/yourorg/socialpublish-migrate:${IMAGE_TAG}
          command: ["/app/migrate", "up"]
          envFrom:
            - secretRef:
                name: socialpublish-secrets
          resources:
            requests:
              cpu:    "100m"
              memory: "64Mi"
            limits:
              cpu:    "200m"
              memory: "128Mi"
```

### 7.9 Ingress

```yaml
# deploy/k8s/ingress/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: socialpublish-ingress
  namespace: socialpublish
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/proxy-body-size: "5g"      # allow large video uploads
    nginx.ingress.kubernetes.io/proxy-read-timeout: "300"  # 5 min for uploads
    nginx.ingress.kubernetes.io/proxy-send-timeout: "300"
    nginx.ingress.kubernetes.io/proxy-request-buffering: "off" # stream uploads
spec:
  ingressClassName: nginx
  tls:
    - hosts: [api.socialpublish.io]
      secretName: socialpublish-tls
  rules:
    - host: api.socialpublish.io
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: socialpublish-api
                port:
                  number: 8080
```

---

## Part 8: Dockerfile

```dockerfile
# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO disabled — pure Go binary, no libc dependency.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ── FFmpeg layer (worker only) ────────────────────────────────────────────────
FROM jrottenberg/ffmpeg:6.1-alpine AS ffmpeg

# ── API final image ───────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS api
COPY --from=builder /out/server /app/server
EXPOSE 8080 9090
ENTRYPOINT ["/app/server"]

# ── Worker final image (needs FFmpeg) ─────────────────────────────────────────
FROM alpine:3.19 AS worker
RUN apk --no-cache add ca-certificates
COPY --from=ffmpeg /usr/local/bin/ffmpeg /usr/bin/ffmpeg
COPY --from=builder /out/worker /app/worker
EXPOSE 9090
ENTRYPOINT ["/app/worker"]

# ── Migrate final image ───────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS migrate
COPY --from=builder /out/migrate /app/migrate
ENTRYPOINT ["/app/migrate"]
```

---

## Part 9: CI/CD Pipeline (GitHub Actions skeleton)

```yaml
# .github/workflows/deploy.yaml
name: Build and Deploy

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_BASE: ghcr.io/${{ github.repository }}

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_DB: sp_test
          POSTGRES_USER: sp
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 5s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        options: --health-cmd "redis-cli ping" --health-interval 5s
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true
      - name: Run linter
        uses: golangci/golangci-lint-action@v4
        with:
          version: v1.57
      - name: Run tests
        run: go test -race -coverprofile=coverage.out ./...
        env:
          DATABASE_URL: postgres://sp:test@localhost:5432/sp_test?sslmode=disable
          REDIS_ADDR: localhost:6379

  build-and-push:
    needs: test
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push API image
        uses: docker/build-push-action@v5
        with:
          context: .
          target: api
          push: true
          tags: ${{ env.IMAGE_BASE }}-api:${{ github.sha }},${{ env.IMAGE_BASE }}-api:latest
          cache-from: type=gha
          cache-to: type=gha,mode=max
      - name: Build and push Worker image
        uses: docker/build-push-action@v5
        with:
          context: .
          target: worker
          push: true
          tags: ${{ env.IMAGE_BASE }}-worker:${{ github.sha }},${{ env.IMAGE_BASE }}-worker:latest
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy:
    needs: build-and-push
    runs-on: ubuntu-latest
    environment: production
    steps:
      - uses: actions/checkout@v4
      - name: Run migration job
        run: |
          kubectl apply -f deploy/k8s/migrate-job.yaml
          kubectl wait --for=condition=complete job/socialpublish-migrate-${{ github.sha }} \
            --timeout=120s -n socialpublish
      - name: Deploy API
        run: |
          kubectl set image deployment/socialpublish-api \
            api=${{ env.IMAGE_BASE }}-api:${{ github.sha }} \
            -n socialpublish
          kubectl rollout status deployment/socialpublish-api -n socialpublish --timeout=300s
      - name: Deploy Worker
        run: |
          kubectl set image deployment/socialpublish-worker \
            worker=${{ env.IMAGE_BASE }}-worker:${{ github.sha }} \
            -n socialpublish
          kubectl rollout status deployment/socialpublish-worker -n socialpublish --timeout=300s
```

---

## Part 10: Implementation Order for Codex

Implement in this exact sequence. Each phase compiles and passes tests before the next starts.

### Phase 1 — Scaffold (Day 1)
1. `go mod init github.com/yourorg/socialpublish`
2. `go.mod` dependencies (list below)
3. All `types.go` files (zero logic — just structs, constants, interfaces)
4. `sdk/errors.go` — full error types with `Is()` methods
5. `internal/tenant/context.go`
6. SQL migrations (all 9, `.up.sql` and `.down.sql`)

### Phase 2 — HTTP Transport + SDK Client (Day 1–2)
7. `sdk/internal/transport.go` — retry, auth injection, error parsing
8. `sdk/options.go` + `sdk/client.go`
9. `sdk/pagination.go`
10. Unit tests for transport (use `httptest.NewServer`)

### Phase 3 — Post Builder (Day 2)
11. `sdk/services/post/builder.go` — builder with both sub-builders
12. `sdk/services/post/types.go` — full type set
13. Unit tests for builder (table-driven; test all validation paths)

### Phase 4 — Remaining SDK Services (Day 2–3)
14. `sdk/services/media/` — service + upload + types
15. `sdk/services/account/` — service + types
16. `sdk/services/schedule/` — service + types
17. `sdk/services/analytics/` — service + types
18. `sdk/webhook.go` — signature verification + router

### Phase 5 — Database Store Layer (Day 3–4)
19. `internal/store/` — all store interfaces first, then postgres implementations
20. Use `pgx/v5` directly (not `database/sql`) — named parameters, row scanning
21. Integration tests using `testcontainers-go` (real Postgres, real Redis)

### Phase 6 — Platform Adapters (Day 4–5)
22. `internal/platform/adapter.go` + `registry.go`
23. `internal/platform/instagram/` — container flow, polling, retries
24. `internal/platform/youtube/` — resumable upload, chunk loop, poll
25. Unit tests with recorded HTTP fixtures (no real API calls in CI)

### Phase 7 — FFmpeg Runner (Day 5)
26. `internal/ffmpeg/presets.go` — all 4 presets as typed constants
27. `internal/ffmpeg/runner.go` — `exec.CommandContext`, chunk output, cleanup
28. Integration test: transcode a 5-second test video, verify output dimensions

### Phase 8 — Workers (Day 6)
29. `internal/worker/pool.go`
30. `internal/worker/transcode.go`
31. `internal/worker/publish.go`
32. `internal/worker/refresh.go`
33. `internal/worker/analytics.go`
34. `internal/worker/webhook.go`

### Phase 9 — HTTP Handlers + Middleware (Day 6–7)
35. All middleware (`auth`, `tenant`, `ratelimit`, `requestid`, `logger`)
36. All handlers — each thin: validate → call store/worker → encode response
37. `internal/api/server.go` route wiring
38. Handler integration tests using `httptest`

### Phase 10 — Entrypoints + Config (Day 7)
39. `cmd/server/main.go` — signal handling, errgroup, graceful shutdown
40. `cmd/worker/main.go`
41. `cmd/migrate/main.go`
42. `internal/config/` — env var loading with validation

### Phase 11 — Deployment (Day 8)
43. `Dockerfile` (multi-stage, all three targets)
44. All K8s manifests
45. GitHub Actions workflow
46. `Makefile` targets: `test`, `lint`, `build`, `docker-build`, `k8s-apply`

---

## Part 11: Go Module Dependencies

```
# go.mod — pin these exact major versions

require (
    # HTTP router
    github.com/go-chi/chi/v5          v5.0.12

    # Database
    github.com/jackc/pgx/v5           v5.5.5

    # Job queue
    github.com/hibiken/asynq          v0.24.1

    # Redis client (used by asynq + rate limiter)
    github.com/redis/go-redis/v9      v9.5.1

    # AWS SDK (for S3/R2)
    github.com/aws/aws-sdk-go-v2              v1.26.1
    github.com/aws/aws-sdk-go-v2/config       v1.27.9
    github.com/aws/aws-sdk-go-v2/service/s3   v1.53.1

    # Migrations
    github.com/golang-migrate/migrate/v4      v4.17.1

    # Testing
    github.com/stretchr/testify               v1.9.0
    github.com/testcontainers/testcontainers-go v0.29.1

    # Concurrency
    golang.org/x/sync                         v0.6.0

    # Crypto (for HMAC, AES-GCM)
    # stdlib crypto — no external dep needed

    # Metrics
    github.com/prometheus/client_golang       v1.19.0
)
```

---

## Part 12: Linter Config (`.golangci.yml`)

```yaml
linters:
  enable:
    - errcheck      # never ignore errors
    - govet         # go vet checks
    - staticcheck   # comprehensive static analysis
    - unused        # no dead code
    - gosimple
    - gofmt
    - goimports
    - misspell
    - bodyclose     # always close response bodies
    - contextcheck  # correct context propagation
    - noctx         # no http.NewRequest without context
    - exhaustive    # exhaustive switch on typed consts
    - testifylint   # correct testify usage
    - godot         # godoc comments end with period

linters-settings:
  errcheck:
    check-type-assertions: true   # catch unchecked .(Type) assertions
    check-blank: true             # catch _ = someErr
  godot:
    scope: all

issues:
  exclude-rules:
    # Allow _ error ignores in defer chains where error is not actionable
    - path: "cmd/"
      linters: [errcheck]
      text: "defer.*Close"
```

---

## Part 13: Senior Go Review Checklist

Before submitting any implementation for review, verify every item:

**Correctness**
- [ ] Every error from every function call is handled — never `_`
- [ ] All `resp.Body.Close()` calls are in `defer` immediately after checking `err`
- [ ] HTTP response bodies are `io.LimitReader` before `io.ReadAll`
- [ ] Context is propagated to every blocking call (DB, HTTP, queue)
- [ ] No goroutine is started without a clear owner responsible for its lifecycle
- [ ] No `time.Sleep` in loops — use `time.NewTicker` with `select { case <-ctx.Done() }`
- [ ] Mutexes are never copied after first use
- [ ] No `sync.Mutex` embedded in a struct that is passed by value

**API Design**
- [ ] All public types have godoc comments ending with a period
- [ ] Option funcs use `func(*config)` not `func(*config) error` (errors belong at `New()`)
- [ ] Builder errors accumulate and surface at `Build()`, not at each call site
- [ ] No public `init()` side effects
- [ ] Zero values are useful (e.g. `Privacy{}` defaults gracefully)

**Concurrency**
- [ ] Worker handlers are stateless; all state via parameters, not receiver fields
- [ ] `errgroup.WithContext` propagates cancellation to all goroutines
- [ ] Graceful shutdown: API drains in-flight requests; workers finish current task
- [ ] `signal.NotifyContext` used in `main()`; not `signal.Notify` + manual channel

**Database**
- [ ] All queries use `$1` parameters — never `fmt.Sprintf` into SQL
- [ ] All migrations have matching `.down.sql`
- [ ] Transactions wrap multi-step DB operations
- [ ] `pgx.Rows` is always `defer rows.Close()`'d and `rows.Err()` is checked

**Security**
- [ ] API keys are bcrypt-hashed in DB; never stored plaintext
- [ ] OAuth tokens are AES-GCM encrypted; key loaded from KMS/Vault
- [ ] Webhook secrets are HMAC-signed with timestamp replay protection
- [ ] All user-controlled strings are parameterised in SQL
- [ ] No credentials in logs, even at debug level
- [ ] Image tags in K8s use SHA digest, not `latest` in prod

**Testing**
- [ ] Table-driven tests with `t.Run(name, func(t *testing.T) {...})`
- [ ] `t.Parallel()` on unit tests that have no shared mutable state
- [ ] Mocks implement the same interface as the real impl — not hand-rolled structs
- [ ] Integration tests use `t.Cleanup` to tear down containers, never `defer` at package level
- [ ] No `time.Sleep` in tests — use polling with `require.Eventually`
```
