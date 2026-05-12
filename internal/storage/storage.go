package storage

import "context"

// ObjectStorage reads and writes media objects.
type ObjectStorage interface {
	PresignUpload(ctx context.Context, key string, contentType string) (string, map[string]string, error)
	PublicURL(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}
