package queue

import "context"

// Queue enqueues asynchronous background work.
type Queue interface {
	Enqueue(ctx context.Context, taskType string, payload []byte) error
}
