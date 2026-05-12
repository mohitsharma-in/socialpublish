package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/yourorg/socialpublish/internal/ffmpeg"
	"github.com/yourorg/socialpublish/internal/platform"
	"github.com/yourorg/socialpublish/internal/storage"
	"github.com/yourorg/socialpublish/internal/store"
)

const (
	// TaskTranscode transcodes uploaded media.
	TaskTranscode = "media:transcode"
	// TaskPublish publishes one post target.
	TaskPublish = "post:publish"
	// TaskTokenRefresh refreshes a platform token.
	TaskTokenRefresh = "account:token_refresh"
	// TaskAnalyticsPoll polls platform analytics.
	TaskAnalyticsPoll = "analytics:poll"
	// TaskWebhookDeliver sends a webhook delivery.
	TaskWebhookDeliver = "webhook:deliver"
)

// Pool is the background worker pool.
type Pool struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// New creates a Pool with all handlers registered.
func New(redisAddr string, stores store.Stores, obj storage.ObjectStorage, adapters platform.Registry, ff *ffmpeg.Runner) *Pool {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Queues: map[string]int{
				"critical": 10,
				"default":  5,
				"low":      2,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				slog.Error("worker task failed", "type", task.Type(), "err", err)
			}),
		},
	)

	mux := asynq.NewServeMux()
	mux.Handle(TaskTranscode, NewTranscodeHandler(stores.Media, obj, ff))
	mux.Handle(TaskPublish, NewPublishHandler(stores.Posts, stores.Accounts, stores.Tokens, adapters, stores.Webhooks))
	mux.Handle(TaskTokenRefresh, NewTokenRefreshHandler(stores.Accounts, stores.Tokens, adapters))
	mux.Handle(TaskAnalyticsPoll, NewAnalyticsHandler(stores.Analytics, stores.Accounts, stores.Tokens, adapters))
	mux.Handle(TaskWebhookDeliver, NewWebhookDeliverHandler(stores.Webhooks))

	return &Pool{server: srv, mux: mux}
}

// Run starts the worker pool and blocks until ctx is done.
func (p *Pool) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := p.server.Run(p.mux); err != nil {
			errCh <- fmt.Errorf("worker pool: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		p.server.Shutdown()
		return nil
	}
}
