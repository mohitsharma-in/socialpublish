package ffmpeg

// Preset describes one FFmpeg output profile.
type Preset struct {
	Name      string
	Width     int
	Height    int
	VideoArgs []string
	AudioArgs []string
}

var (
	// PresetInstagramReel encodes a 9:16 Instagram Reel.
	PresetInstagramReel = Preset{
		Name:   "instagram_reel",
		Width:  1080,
		Height: 1920,
		VideoArgs: []string{
			"-c:v", "libx264",
			"-profile:v", "high",
			"-pix_fmt", "yuv420p",
			"-r", "30",
			"-b:v", "8M",
		},
		AudioArgs: []string{"-c:a", "aac", "-b:a", "128k"},
	}

	// PresetInstagramStory encodes a 9:16 Instagram Story.
	PresetInstagramStory = Preset{
		Name:      "instagram_story",
		Width:     1080,
		Height:    1920,
		VideoArgs: []string{"-c:v", "libx264", "-pix_fmt", "yuv420p", "-r", "30", "-b:v", "6M"},
		AudioArgs: []string{"-c:a", "aac", "-b:a", "128k"},
	}

	// PresetYouTubeShort encodes a 9:16 YouTube Short.
	PresetYouTubeShort = Preset{
		Name:      "youtube_short",
		Width:     1080,
		Height:    1920,
		VideoArgs: []string{"-c:v", "libx264", "-profile:v", "high", "-pix_fmt", "yuv420p", "-r", "30", "-b:v", "10M"},
		AudioArgs: []string{"-c:a", "aac", "-b:a", "192k"},
	}

	// PresetYouTubeVideo encodes a 16:9 YouTube video.
	PresetYouTubeVideo = Preset{
		Name:      "youtube_video",
		Width:     1920,
		Height:    1080,
		VideoArgs: []string{"-c:v", "libx264", "-profile:v", "high", "-pix_fmt", "yuv420p", "-r", "30", "-b:v", "12M"},
		AudioArgs: []string{"-c:a", "aac", "-b:a", "192k"},
	}
)

// AllPresets returns every supported preset.
func AllPresets() []Preset {
	return []Preset{PresetInstagramReel, PresetInstagramStory, PresetYouTubeShort, PresetYouTubeVideo}
}
