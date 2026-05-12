package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	appconfig "github.com/yourorg/socialpublish/internal/config"
	"github.com/yourorg/socialpublish/internal/ffmpeg"
	"github.com/yourorg/socialpublish/internal/platform"
	"github.com/yourorg/socialpublish/internal/platform/instagram"
	"github.com/yourorg/socialpublish/internal/platform/youtube"
	"github.com/yourorg/socialpublish/internal/storage"
	"github.com/yourorg/socialpublish/internal/store"
	"github.com/yourorg/socialpublish/internal/worker"
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
	stores := store.New(db)
	adapters := platform.NewRegistry(instagram.New(instagram.Config{}, nil), youtube.New(youtube.Config{}, nil))
	pool := worker.New(cfg.RedisAddr, stores, obj, adapters, ffmpeg.New(ffmpeg.WithBinary(cfg.FFmpegBin)))
	return pool.Run(ctx)
}
