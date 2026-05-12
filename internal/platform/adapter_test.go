package platform_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mohitsharma-in/socialpublish/internal/platform"
	"github.com/mohitsharma-in/socialpublish/internal/platform/instagram"
	"github.com/mohitsharma-in/socialpublish/internal/platform/youtube"
	"github.com/mohitsharma-in/socialpublish/internal/store"
)

func TestRegistryAndAdapterValidation(t *testing.T) {
	t.Parallel()

	ig := instagram.New(instagram.Config{}, nil)
	yt := youtube.New(youtube.Config{}, nil)
	registry := platform.NewRegistry(ig, yt)

	adapter, ok := registry.Get(instagram.PlatformName)
	require.True(t, ok)
	require.NoError(t, adapter.ValidateTarget(store.PostTarget{Platform: instagram.PlatformName, Format: "reel"}))
	require.Error(t, adapter.ValidateTarget(store.PostTarget{Platform: instagram.PlatformName, Format: "video"}))

	adapter, ok = registry.Get(youtube.PlatformName)
	require.True(t, ok)
	require.NoError(t, adapter.ValidateTarget(store.PostTarget{Platform: youtube.PlatformName, Format: "short"}))
	require.Error(t, adapter.ValidateTarget(store.PostTarget{Platform: youtube.PlatformName, Format: "story"}))
}
