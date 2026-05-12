package post

import (
	"fmt"
	"time"
)

const minimumScheduleDelay = time.Minute

// Builder constructs a CreateRequest using a fluent API.
type Builder struct {
	req    CreateRequest
	errors []error
}

// NewPost returns a fresh Builder.
func NewPost() *Builder {
	return &Builder{}
}

// WithMedia attaches one or more pre-uploaded media IDs to the post.
func (b *Builder) WithMedia(ids ...string) *Builder {
	b.req.MediaIDs = append(b.req.MediaIDs, ids...)
	return b
}

// ScheduleAt sets the UTC publish time.
func (b *Builder) ScheduleAt(t time.Time) *Builder {
	if t.IsZero() {
		b.errors = append(b.errors, fmt.Errorf("builder: ScheduleAt: zero time is invalid"))
		return b
	}
	if time.Until(t) < minimumScheduleDelay {
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

// WithMetadata attaches an arbitrary key/value pair.
func (b *Builder) WithMetadata(key, value string) *Builder {
	if b.req.Metadata == nil {
		b.req.Metadata = make(map[string]string)
	}
	b.req.Metadata[key] = value
	return b
}

// ForInstagram returns a platform-specific sub-builder.
func (b *Builder) ForInstagram(accountID string) *InstagramBuilder {
	if accountID == "" {
		b.errors = append(b.errors, fmt.Errorf("builder: ForInstagram: accountID is required"))
	}
	return &InstagramBuilder{parent: b, accountID: accountID}
}

// ForYouTube returns a platform-specific sub-builder.
func (b *Builder) ForYouTube(accountID string) *YouTubeBuilder {
	if accountID == "" {
		b.errors = append(b.errors, fmt.Errorf("builder: ForYouTube: accountID is required"))
	}
	return &YouTubeBuilder{parent: b, accountID: accountID}
}

// Build validates the request and returns it.
func (b *Builder) Build() (*CreateRequest, error) {
	if len(b.errors) > 0 {
		return nil, b.errors[0]
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
	req := b.req
	return &req, nil
}

// InstagramBuilder configures a single Instagram publish target.
type InstagramBuilder struct {
	parent    *Builder
	accountID string
	t         InstagramConfig
}

// AsReel configures the target as an Instagram Reel.
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

// WithCollaborators adds Instagram usernames as collaborators.
func (b *InstagramBuilder) WithCollaborators(handles ...string) *InstagramBuilder {
	b.t.Collaborators = append(b.t.Collaborators, handles...)
	return b
}

// ShareToFeed controls whether a Reel also appears on the main feed.
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

// YouTubeBuilder configures a single YouTube publish target.
type YouTubeBuilder struct {
	parent    *Builder
	accountID string
	t         YouTubeConfig
}

// AsShort configures the target as a YouTube Short.
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

// WithPrivacy sets the video privacy.
func (b *YouTubeBuilder) WithPrivacy(p Privacy) *YouTubeBuilder {
	b.t.Privacy = p
	return b
}

// WithTags sets the video tags.
func (b *YouTubeBuilder) WithTags(tags ...string) *YouTubeBuilder {
	b.t.Tags = append(b.t.Tags, tags...)
	return b
}

// WithCategory sets the YouTube category ID.
func (b *YouTubeBuilder) WithCategory(id int) *YouTubeBuilder {
	b.t.CategoryID = id
	return b
}

// AddToPlaylist adds the video to the given playlist after publishing.
func (b *YouTubeBuilder) AddToPlaylist(playlistID string) *YouTubeBuilder {
	b.t.PlaylistIDs = append(b.t.PlaylistIDs, playlistID)
	return b
}

// MadeForKids sets the made-for-kids designation.
func (b *YouTubeBuilder) MadeForKids(v bool) *YouTubeBuilder {
	b.t.MadeForKids = v
	return b
}

// NotifySubscribers controls whether subscribers receive a notification.
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
