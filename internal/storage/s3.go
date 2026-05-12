package storage

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/credentials"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/presign"

	appconfig "github.com/mohitsharma-in/socialpublish/internal/config"
)

// S3Storage stores objects in S3-compatible storage.
type S3Storage struct {
	cfg       appconfig.S3Config
	client    *s3.Client
	presigner *presign.PresignClient
}

// NewS3 creates S3-compatible object storage.
func NewS3(ctx context.Context, cfg appconfig.S3Config) (*S3Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("s3 region is required")
	}

	loaderOptions := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		loaderOptions = append(loaderOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	if cfg.Endpoint != "" {
		loaderOptions = append(loaderOptions, config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: strings.TrimRight(cfg.Endpoint, "/"), SigningRegion: cfg.Region}, nil
			},
		)))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, loaderOptions...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	clientOptions := func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.EndpointResolver = aws.EndpointResolverWithOptionsFunc(
				func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: strings.TrimRight(cfg.Endpoint, "/"), SigningRegion: cfg.Region}, nil
				},
			)
		}
	}

	client := s3.NewFromConfig(awsCfg, clientOptions)
	return &S3Storage{cfg: cfg, client: client, presigner: presign.NewPresignClient(client)}, nil
}

// PresignUpload returns a direct upload URL and required headers.
func (s *S3Storage) PresignUpload(ctx context.Context, key string, contentType string) (string, map[string]string, error) {
	if key == "" {
		return "", nil, fmt.Errorf("presign upload: key is required")
	}

	params := &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	req, err := s.presigner.PresignPutObject(ctx, params, presign.WithExpires(15*time.Minute))
	if err != nil {
		return "", nil, fmt.Errorf("presign upload: %w", err)
	}

	headers := map[string]string{}
	if contentType != "" {
		headers["Content-Type"] = contentType
	}

	return req.URL, headers, nil
}

// PublicURL returns a public object URL.
func (s *S3Storage) PublicURL(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("public URL: key is required")
	}
	return buildObjectURL(s.cfg, key)
}

// Delete deletes an object.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("delete object: key is required")
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object %s: %w", key, err)
	}
	return nil
}

func buildObjectURL(cfg appconfig.S3Config, key string) (string, error) {
	if cfg.Endpoint != "" {
		endpoint := strings.TrimRight(cfg.Endpoint, "/")
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return "", fmt.Errorf("parse endpoint: %w", err)
		}
		parsed.Path = path.Join(parsed.Path, cfg.Bucket, key)
		return parsed.String(), nil
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.Bucket, cfg.Region, key), nil
}
