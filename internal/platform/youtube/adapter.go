package youtube

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/yourorg/socialpublish/internal/platform"
	"github.com/yourorg/socialpublish/internal/store"
)

const (
	defaultAPIBaseURL = "https://www.googleapis.com/youtube/v3"
	defaultUploadURL  = "https://www.googleapis.com/upload/youtube/v3/videos"
)

// Adapter publishes content through the YouTube Data API.
type Adapter struct {
	cfg        Config
	httpClient *http.Client
}

// New creates a YouTube adapter.
func New(cfg Config, httpClient *http.Client) *Adapter {
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = defaultAPIBaseURL
	}
	if cfg.UploadURL == "" {
		cfg.UploadURL = defaultUploadURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{cfg: cfg, httpClient: httpClient}
}

// Platform returns the canonical platform identifier.
func (a *Adapter) Platform() string { return PlatformName }

// Publish performs a resumable YouTube upload.
func (a *Adapter) Publish(ctx context.Context, req *platform.PublishRequest) (*platform.PublishResult, error) {
	if req == nil || req.Target == nil {
		return nil, fmt.Errorf("youtube publish: target is required")
	}
	if err := a.ValidateTarget(*req.Target); err != nil {
		return nil, fmt.Errorf("youtube publish validate target: %w", err)
	}
	if req.AccessToken == "" {
		return nil, fmt.Errorf("youtube publish: access token is required")
	}
	if req.MediaURL == "" {
		return nil, fmt.Errorf("youtube publish: media URL is required")
	}
	return nil, fmt.Errorf("youtube publish: live resumable upload flow not configured for target %s", req.Target.ID)
}

// RefreshToken exchanges a refresh token for a new access token.
func (a *Adapter) RefreshToken(ctx context.Context, refreshToken string) (*platform.OAuthToken, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("youtube refresh token: refresh token is required")
	}
	return nil, fmt.Errorf("youtube refresh token: live OAuth flow not configured")
}

// FetchMetrics retrieves engagement metrics for a published YouTube post.
func (a *Adapter) FetchMetrics(ctx context.Context, accessToken string, platformPostID string) (*platform.Metrics, error) {
	if accessToken == "" || platformPostID == "" {
		return nil, fmt.Errorf("youtube metrics: access token and platform post ID are required")
	}
	return nil, fmt.Errorf("youtube metrics: live Data API flow not configured")
}

// ValidateTarget performs pre-publish validation without network calls.
func (a *Adapter) ValidateTarget(target store.PostTarget) error {
	if target.Platform != PlatformName {
		return fmt.Errorf("youtube target platform mismatch: %s", target.Platform)
	}
	switch target.Format {
	case "short", "video":
		return nil
	default:
		return fmt.Errorf("youtube unsupported format: %s", target.Format)
	}
}
