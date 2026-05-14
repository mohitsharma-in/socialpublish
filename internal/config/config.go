package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultListenAddr = "0.0.0.0:8080"
	defaultRedisAddr  = "127.0.0.1:6379"
	defaultFFmpegBin  = "ffmpeg"
)

// Config holds runtime configuration.
type Config struct {
	ListenAddr         string
	DatabaseURL        string
	RedisAddr          string
	FFmpegBin          string
	TokenEncryptionKey string
	S3                 S3Config
}

// S3Config holds object storage configuration.
type S3Config struct {
	Region          string
	Endpoint        string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		ListenAddr:         getEnv("LISTEN_ADDR", defaultListenAddr),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		RedisAddr:          getEnv("REDIS_ADDR", defaultRedisAddr),
		FFmpegBin:          getEnv("FFMPEG_BIN", defaultFFmpegBin),
		TokenEncryptionKey: os.Getenv("TOKEN_ENCRYPTION_KEY"),
		S3: S3Config{
			Region:          os.Getenv("S3_REGION"),
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			Bucket:          os.Getenv("S3_BUCKET"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
		},
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.TokenEncryptionKey) != 32 {
		return Config{}, fmt.Errorf("TOKEN_ENCRYPTION_KEY is required and must be exactly 32 bytes")
	}
	return cfg, nil
}

// Duration reads a duration environment variable.
func Duration(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s duration: %w", name, err)
	}
	return d, nil
}

// Int reads an integer environment variable.
func Int(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s int: %w", name, err)
	}
	return i, nil
}

func getEnv(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
