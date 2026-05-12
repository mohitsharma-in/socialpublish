package account

import "time"

// Platform identifies a supported connected-account platform.
type Platform string

const (
	// PlatformInstagram identifies Instagram.
	PlatformInstagram Platform = "instagram"
	// PlatformYouTube identifies YouTube.
	PlatformYouTube Platform = "youtube"
)

// Account is a connected social account.
type Account struct {
	ID             string    `json:"account_id"`
	Platform       Platform  `json:"platform"`
	PlatformUserID string    `json:"platform_user_id"`
	DisplayName    string    `json:"display_name"`
	AvatarURL      string    `json:"avatar_url,omitempty"`
	TokenExpiresAt time.Time `json:"token_expires_at,omitempty"`
	TokenHealthy   bool      `json:"token_healthy"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ConnectRequest starts or completes account connection.
type ConnectRequest struct {
	Platform     Platform `json:"platform"`
	OAuthCode    string   `json:"oauth_code,omitempty"`
	RedirectURI  string   `json:"redirect_uri,omitempty"`
	RefreshToken string   `json:"refresh_token,omitempty"`
}

// ConnectResponse returns the connected account.
type ConnectResponse struct {
	Account *Account `json:"account"`
}

// ListParams filters account listing.
type ListParams struct {
	Limit    int
	Cursor   string
	Platform Platform
}

// Page is a paginated account response.
type Page struct {
	Items      []*Account `json:"items"`
	NextCursor string     `json:"next_cursor,omitempty"`
	HasMore    bool       `json:"has_more"`
	Total      int        `json:"total"`
}

// GetItems returns the page items.
func (p *Page) GetItems() []*Account { return p.Items }

// GetNextCursor returns the next page cursor.
func (p *Page) GetNextCursor() string { return p.NextCursor }

// GetHasMore reports whether another page is available.
func (p *Page) GetHasMore() bool { return p.HasMore }
