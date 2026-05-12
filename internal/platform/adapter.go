package platform

import (
	"context"
	"time"

	"github.com/mohitsharma-in/socialpublish/internal/store"
)

// PublishRequest is the normalized input every adapter receives.
type PublishRequest struct {
	AccountID   string
	AccessToken string
	MediaURL    string
	Target      *store.PostTarget
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

// Metrics is the normalized analytics payload from a platform.
type Metrics struct {
	Views    int64
	Likes    int64
	Comments int64
	Shares   int64
	Reach    int64
	Extra    map[string]any
}

// PlatformAdapter is the contract every platform must implement.
type PlatformAdapter interface {
	Platform() string
	Publish(ctx context.Context, req *PublishRequest) (*PublishResult, error)
	RefreshToken(ctx context.Context, refreshToken string) (*OAuthToken, error)
	FetchMetrics(ctx context.Context, accessToken string, platformPostID string) (*Metrics, error)
	ValidateTarget(target store.PostTarget) error
}
