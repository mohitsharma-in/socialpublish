package instagram

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/yourorg/socialpublish/internal/platform"
	"github.com/yourorg/socialpublish/internal/store"
)

const defaultGraphBaseURL = "https://graph.facebook.com/v20.0"

// Adapter publishes content through the Instagram Graph API.
type Adapter struct {
	cfg        Config
	httpClient *http.Client
}

// New creates an Instagram adapter.
func New(cfg Config, httpClient *http.Client) *Adapter {
	if cfg.GraphBaseURL == "" {
		cfg.GraphBaseURL = defaultGraphBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{cfg: cfg, httpClient: httpClient}
}

// Platform returns the canonical platform identifier.
func (a *Adapter) Platform() string { return PlatformName }

// Publish performs the Instagram container publish flow.
func (a *Adapter) Publish(ctx context.Context, req *platform.PublishRequest) (*platform.PublishResult, error) {
	if req == nil || req.Target == nil {
		return nil, fmt.Errorf("instagram publish: target is required")
	}
	if err := a.ValidateTarget(*req.Target); err != nil {
		return nil, fmt.Errorf("instagram publish validate target: %w", err)
	}
	if req.AccessToken == "" {
		return nil, fmt.Errorf("instagram publish: access token is required")
	}
	if req.MediaURL == "" {
		return nil, fmt.Errorf("instagram publish: media URL is required")
	}
	return nil, fmt.Errorf("instagram publish: live Graph API flow not configured for target %s", req.Target.ID)
}

// RefreshToken exchanges a refresh token for a new access token.
func (a *Adapter) RefreshToken(ctx context.Context, refreshToken string) (*platform.OAuthToken, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("instagram refresh token: refresh token is required")
	}
	return nil, fmt.Errorf("instagram refresh token: live Graph API flow not configured")
}

// FetchMetrics retrieves engagement metrics for a published Instagram post.
func (a *Adapter) FetchMetrics(ctx context.Context, accessToken string, platformPostID string) (*platform.Metrics, error) {
	if accessToken == "" || platformPostID == "" {
		return nil, fmt.Errorf("instagram metrics: access token and platform post ID are required")
	}
	return nil, fmt.Errorf("instagram metrics: live Graph API flow not configured")
}

// ValidateTarget performs pre-publish validation without network calls.
func (a *Adapter) ValidateTarget(target store.PostTarget) error {
	if target.Platform != PlatformName {
		return fmt.Errorf("instagram target platform mismatch: %s", target.Platform)
	}
	switch target.Format {
	case "reel", "story", "carousel":
		return nil
	default:
		return fmt.Errorf("instagram unsupported format: %s", target.Format)
	}
}
