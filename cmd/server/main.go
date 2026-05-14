package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	"github.com/mohitsharma-in/socialpublish/internal/api"
	"github.com/mohitsharma-in/socialpublish/internal/api/middleware"
	appconfig "github.com/mohitsharma-in/socialpublish/internal/config"
	"github.com/mohitsharma-in/socialpublish/internal/queue"
	"github.com/mohitsharma-in/socialpublish/internal/storage"
	"github.com/mohitsharma-in/socialpublish/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	if err := run(); err != nil {
		slog.Error("server exited with error", "err", err)
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

	stores := store.New(db, cfg.TokenEncryptionKey)
	q := queue.NewAsynq(cfg.RedisAddr)
	defer q.Close()
	obj, err := storage.NewS3(ctx, cfg.S3)
	if err != nil {
		return fmt.Errorf("open object storage: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer rdb.Close()
	limiter := middleware.NewRedisRateLimiter(rdb)

	srv := api.New(api.Config{
		ListenAddr:      cfg.ListenAddr,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		ReadinessCheck: func(ctx context.Context) error {
			return db.Ping(ctx)
		},
	}, stores, q, obj, limiter)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		slog.Info("api server starting", "addr", cfg.ListenAddr)
		return srv.Run(ctx)
	})
	return g.Wait()
}
