package socialpublish

import (
	"fmt"
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
type Client struct {
	cfg       config
	transport *transport

	accounts  account.Service
	media     media.Service
	posts     post.Service
	schedules schedule.Service
	analytics analytics.Service
}

// New creates a Client.
func New(opts ...Option) (*Client, error) {
	cfg := config{
		baseURL:      defaultBaseURL,
		timeout:      defaultTimeout,
		maxRetries:   3,
		retryWaitMin: 500 * time.Millisecond,
		retryWaitMax: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
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
