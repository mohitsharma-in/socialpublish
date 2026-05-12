package queue

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
)

// AsynqQueue enqueues jobs through Asynq.
type AsynqQueue struct {
	client *asynq.Client
}

// NewAsynq creates an Asynq-backed queue.
func NewAsynq(redisAddr string) *AsynqQueue {
	return &AsynqQueue{client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})}
}

// Enqueue enqueues a task.
func (q *AsynqQueue) Enqueue(ctx context.Context, taskType string, payload []byte) error {
	if _, err := q.client.EnqueueContext(ctx, asynq.NewTask(taskType, payload)); err != nil {
		return fmt.Errorf("enqueue task %s: %w", taskType, err)
	}
	return nil
}

// Close closes the underlying client.
func (q *AsynqQueue) Close() error {
	if err := q.client.Close(); err != nil {
		return fmt.Errorf("close asynq client: %w", err)
	}
	return nil
}
