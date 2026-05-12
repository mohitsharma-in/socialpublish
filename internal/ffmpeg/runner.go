package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const defaultTimeout = 30 * time.Minute

// Runner executes FFmpeg jobs.
type Runner struct {
	bin     string
	timeout time.Duration
}

// Option configures a Runner.
type Option func(*Runner)

// WithBinary sets the FFmpeg binary path.
func WithBinary(bin string) Option {
	return func(r *Runner) { r.bin = bin }
}

// WithTimeout sets the command timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(r *Runner) { r.timeout = timeout }
}

// New creates an FFmpeg Runner.
func New(opts ...Option) *Runner {
	r := &Runner{bin: "ffmpeg", timeout: defaultTimeout}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Transcode runs FFmpeg for the given input, output, and preset.
func (r *Runner) Transcode(ctx context.Context, inputPath string, outputPath string, preset Preset) error {
	if inputPath == "" || outputPath == "" {
		return fmt.Errorf("ffmpeg transcode: input and output paths are required")
	}
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	args := transcodeArgs(inputPath, outputPath, preset)
	cmd := exec.CommandContext(ctx, r.bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg transcode %s to %s: %w: %s", inputPath, outputPath, err, stderr.String())
	}
	return nil
}

func transcodeArgs(inputPath string, outputPath string, preset Preset) []string {
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", preset.Width, preset.Height, preset.Width, preset.Height),
	}
	args = append(args, preset.VideoArgs...)
	args = append(args, preset.AudioArgs...)
	args = append(args, "-movflags", "+faststart", outputPath)
	return args
}
