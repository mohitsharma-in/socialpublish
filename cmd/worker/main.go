package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	appconfig "github.com/mohitsharma-in/socialpublish/internal/config"
	"github.com/mohitsharma-in/socialpublish/internal/ffmpeg"
	"github.com/mohitsharma-in/socialpublish/internal/platform"
	"github.com/mohitsharma-in/socialpublish/internal/platform/instagram"
	"github.com/mohitsharma-in/socialpublish/internal/platform/youtube"
	"github.com/mohitsharma-in/socialpublish/internal/storage"
	"github.com/mohitsharma-in/socialpublish/internal/store"
	"github.com/mohitsharma-in/socialpublish/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	if err := run(); err != nil {
		slog.Error("worker exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := appconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	db, err := store.OpenDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	obj, err := storage.NewS3(ctx, cfg.S3)
	if err != nil {
		return fmt.Errorf("open object storage: %w", err)
	}
	stores := store.New(db, cfg.TokenEncryptionKey)
	adapters := platform.NewRegistry(instagram.New(instagram.Config{}, nil), youtube.New(youtube.Config{}, nil))
	pool := worker.New(cfg.RedisAddr, stores, obj, adapters, ffmpeg.New(ffmpeg.WithBinary(cfg.FFmpegBin)))
	return pool.Run(ctx)
}
