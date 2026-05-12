package store

import "time"

// Workspace is a tenant or organization.
type Workspace struct {
	ID        string
	Name      string
	Slug      string
	Plan      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// APIKey is a hashed API key record.
type APIKey struct {
	ID          string
	WorkspaceID string
	KeyHash     string
	KeyPrefix   string
	Name        string
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
}

// Account is a connected social account.
type Account struct {
	ID             string
	WorkspaceID    string
	Platform       string
	PlatformUserID string
	DisplayName    string
	AvatarURL      string
	TokenID        string
	TokenExpiresAt *time.Time
	TokenHealthy   bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Media is an uploaded asset.
type Media struct {
	ID           string
	WorkspaceID  string
	Status       string
	MediaType    string
	OriginalKey  string
	MimeType     string
	SizeBytes    int64
	DurationMS   *int
	Formats      map[string]any
	ThumbnailKey string
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Post is a post resource.
type Post struct {
	ID          string
	WorkspaceID string
	Status      string
	MediaIDs    []string
	ScheduledAt *time.Time
	PublishedAt *time.Time
	Metadata    map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PostTarget is one platform destination for a post.
type PostTarget struct {
	ID              string
	PostID          string
	AccountID       string
	Platform        string
	Format          string
	Config          map[string]any
	Status          string
	PlatformPostID  string
	Permalink       string
	FailureReason   string
	AttemptCount    int
	LastAttemptedAt *time.Time
	PublishedAt     *time.Time
}

// AnalyticsSnapshot stores normalized platform metrics.
type AnalyticsSnapshot struct {
	ID             string
	WorkspaceID    string
	AccountID      string
	PostID         string
	PlatformPostID string
	Metrics        map[string]any
	CollectedAt    time.Time
}

// WebhookEndpoint is a configured webhook target.
type WebhookEndpoint struct {
	ID          string
	WorkspaceID string
	URL         string
	SecretHash  string
	SecretEnc   []byte
	Events      []string
	Enabled     bool
	CreatedAt   time.Time
}

// WebhookDeliveryParams describes a webhook delivery to enqueue.
type WebhookDeliveryParams struct {
	WorkspaceID string
	EventType   string
	Payload     map[string]any
}
