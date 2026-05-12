package post

import "time"

// Platform identifies a social media platform.
type Platform string

const (
	// PlatformInstagram identifies Instagram.
	PlatformInstagram Platform = "instagram"
	// PlatformYouTube identifies YouTube.
	PlatformYouTube Platform = "youtube"
)

// Format identifies the content format within a platform.
type Format string

const (
	// FormatReel is an Instagram Reel.
	FormatReel Format = "reel"
	// FormatStory is an Instagram Story.
	FormatStory Format = "story"
	// FormatCarousel is an Instagram carousel.
	FormatCarousel Format = "carousel"
	// FormatShort is a YouTube Short.
	FormatShort Format = "short"
	// FormatVideo is a YouTube video.
	FormatVideo Format = "video"
)

// Privacy controls YouTube video visibility.
type Privacy string

const (
	// PrivacyPublic makes a YouTube video public.
	PrivacyPublic Privacy = "public"
	// PrivacyUnlisted makes a YouTube video unlisted.
	PrivacyUnlisted Privacy = "unlisted"
	// PrivacyPrivate makes a YouTube video private.
	PrivacyPrivate Privacy = "private"
)

// Status represents the post lifecycle state.
type Status string

const (
	// StatusDraft indicates a draft post.
	StatusDraft Status = "draft"
	// StatusScheduled indicates a scheduled post.
	StatusScheduled Status = "scheduled"
	// StatusPublishing indicates publish work is in progress.
	StatusPublishing Status = "publishing"
	// StatusPublished indicates all publish work is complete.
	StatusPublished Status = "published"
	// StatusFailed indicates publishing failed.
	StatusFailed Status = "failed"
	// StatusCancelled indicates publishing was cancelled.
	StatusCancelled Status = "cancelled"
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

// Target is a single platform and account publishing destination.
type Target struct {
	AccountID string           `json:"account_id"`
	Platform  Platform         `json:"platform"`
	Instagram *InstagramConfig `json:"instagram,omitempty"`
	YouTube   *YouTubeConfig   `json:"youtube,omitempty"`
}

// CreateRequest is the input to Service.Create.
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

// UpdateRequest fields are all optional; omit to leave unchanged.
type UpdateRequest struct {
	MediaIDs    []string          `json:"media_ids,omitempty"`
	Targets     []Target          `json:"targets,omitempty"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ListParams filters and paginates post listing.
type ListParams struct {
	Limit    int
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

// GetItems returns the page items.
func (p *Page) GetItems() []*Post { return p.Items }

// GetNextCursor returns the next page cursor.
func (p *Page) GetNextCursor() string { return p.NextCursor }

// GetHasMore reports whether another page is available.
func (p *Page) GetHasMore() bool { return p.HasMore }

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
