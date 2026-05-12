package post

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderBuildsMultiPlatformPost(t *testing.T) {
	t.Parallel()

	scheduledAt := time.Now().Add(2 * time.Hour).UTC()
	req, err := NewPost().
		WithMedia("med_1", "med_2").
		ForInstagram("acc_ig").AsReel("Caption").WithCollaborators("alice").ShareToFeed(true).Done().
		ForYouTube("acc_yt").AsShort("Title", "Description").WithPrivacy(PrivacyPublic).WithTags("go", "sdk").AddToPlaylist("pl_1").MadeForKids(false).NotifySubscribers(true).Done().
		WithMetadata("cms_id", "post_42").
		ScheduleAt(scheduledAt).
		Build()

	require.NoError(t, err)
	require.Equal(t, []string{"med_1", "med_2"}, req.MediaIDs)
	require.Len(t, req.Targets, 2)
	require.Equal(t, "post_42", req.Metadata["cms_id"])
	require.NotNil(t, req.ScheduledAt)
	assert.Equal(t, scheduledAt, *req.ScheduledAt)

	ig := req.Targets[0]
	require.Equal(t, PlatformInstagram, ig.Platform)
	require.NotNil(t, ig.Instagram)
	assert.Equal(t, FormatReel, ig.Instagram.Format)
	assert.Equal(t, "Caption", ig.Instagram.Caption)
	require.NotNil(t, ig.Instagram.ShareToFeed)
	assert.True(t, *ig.Instagram.ShareToFeed)

	yt := req.Targets[1]
	require.Equal(t, PlatformYouTube, yt.Platform)
	require.NotNil(t, yt.YouTube)
	assert.Equal(t, FormatShort, yt.YouTube.Format)
	assert.Equal(t, PrivacyPublic, yt.YouTube.Privacy)
	assert.Equal(t, []string{"go", "sdk"}, yt.YouTube.Tags)
}

func TestBuilderValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		build   func() (*CreateRequest, error)
		wantErr string
	}{
		{
			name: "missing media",
			build: func() (*CreateRequest, error) {
				return NewPost().ForInstagram("acc").AsStory().Done().Build()
			},
			wantErr: "at least one media ID is required",
		},
		{
			name: "missing target",
			build: func() (*CreateRequest, error) {
				return NewPost().WithMedia("med").Build()
			},
			wantErr: "at least one platform target is required",
		},
		{
			name: "schedule zero time",
			build: func() (*CreateRequest, error) {
				return NewPost().WithMedia("med").ForInstagram("acc").AsStory().Done().ScheduleAt(time.Time{}).Build()
			},
			wantErr: "zero time is invalid",
		},
		{
			name: "schedule too soon",
			build: func() (*CreateRequest, error) {
				return NewPost().WithMedia("med").ForInstagram("acc").AsStory().Done().ScheduleAt(time.Now()).Build()
			},
			wantErr: "time must be at least 1 minute in the future",
		},
		{
			name: "schedule and publish now",
			build: func() (*CreateRequest, error) {
				return NewPost().WithMedia("med").ForInstagram("acc").AsStory().Done().ScheduleAt(time.Now().Add(2 * time.Hour)).PublishNow().Build()
			},
			wantErr: "ScheduleAt and PublishNow are mutually exclusive",
		},
		{
			name: "instagram account required",
			build: func() (*CreateRequest, error) {
				return NewPost().WithMedia("med").ForInstagram("").AsStory().Done().Build()
			},
			wantErr: "ForInstagram: accountID is required",
		},
		{
			name: "youtube account required",
			build: func() (*CreateRequest, error) {
				return NewPost().WithMedia("med").ForYouTube("").AsVideo("title", "desc").Done().Build()
			},
			wantErr: "ForYouTube: accountID is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tt.build()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
