package storage

import (
	"context"
	"fmt"

	appconfig "github.com/yourorg/socialpublish/internal/config"
)

// S3Storage stores objects in S3-compatible storage.
type S3Storage struct {
	cfg appconfig.S3Config
}

// NewS3 creates S3-compatible object storage.
func NewS3(ctx context.Context, cfg appconfig.S3Config) (*S3Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	return &S3Storage{cfg: cfg}, nil
}

// PresignUpload returns a direct-upload URL.
func (s *S3Storage) PresignUpload(ctx context.Context, key string, contentType string) (string, map[string]string, error) {
	if key == "" {
		return "", nil, fmt.Errorf("presign upload: key is required")
	}
	return "", nil, fmt.Errorf("presign upload: S3 presigner not configured for bucket %s", s.cfg.Bucket)
}

// PublicURL returns a public object URL.
func (s *S3Storage) PublicURL(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("public URL: key is required")
	}
	if s.cfg.Endpoint != "" {
		return s.cfg.Endpoint + "/" + s.cfg.Bucket + "/" + key, nil
	}
	return "https://" + s.cfg.Bucket + ".s3." + s.cfg.Region + ".amazonaws.com/" + key, nil
}

// Delete deletes an object.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("delete object: key is required")
	}
	return fmt.Errorf("delete object: S3 client not configured for bucket %s", s.cfg.Bucket)
}
