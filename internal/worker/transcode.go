package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/yourorg/socialpublish/internal/ffmpeg"
	"github.com/yourorg/socialpublish/internal/storage"
	"github.com/yourorg/socialpublish/internal/store"
)

// TranscodePayload is serialized into media transcode jobs.
type TranscodePayload struct {
	MediaID    string        `json:"media_id"`
	InputPath  string        `json:"input_path"`
	OutputPath string        `json:"output_path"`
	Preset     ffmpeg.Preset `json:"preset"`
}

type transcodeHandler struct {
	media store.MediaStore
	obj   storage.ObjectStorage
	ff    *ffmpeg.Runner
}

// NewTranscodeHandler creates a media transcode handler.
func NewTranscodeHandler(media store.MediaStore, obj storage.ObjectStorage, ff *ffmpeg.Runner) asynq.Handler {
	return &transcodeHandler{media: media, obj: obj, ff: ff}
}

func (h *transcodeHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload TranscodePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: unmarshal transcode payload: %v", asynq.SkipRetry, err)
	}
	if err := h.ff.Transcode(ctx, payload.InputPath, payload.OutputPath, payload.Preset); err != nil {
		_ = h.media.MarkFailed(ctx, payload.MediaID, err.Error())
		return fmt.Errorf("transcode media %s: %w", payload.MediaID, err)
	}
	if err := h.media.MarkReady(ctx, payload.MediaID, map[string]any{payload.Preset.Name: payload.OutputPath}, ""); err != nil {
		return fmt.Errorf("mark media %s ready: %w", payload.MediaID, err)
	}
	return nil
}
