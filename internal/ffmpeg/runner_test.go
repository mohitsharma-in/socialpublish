package ffmpeg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTranscodeArgs(t *testing.T) {
	t.Parallel()

	args := transcodeArgs("in.mp4", "out.mp4", PresetYouTubeShort)

	require.Contains(t, args, "scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2")
	require.Contains(t, args, "libx264")
	require.Equal(t, "out.mp4", args[len(args)-1])
}

func TestAllPresets(t *testing.T) {
	t.Parallel()

	require.Len(t, AllPresets(), 4)
}
