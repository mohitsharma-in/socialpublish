package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yourorg/socialpublish/internal/api"
	appconfig "github.com/yourorg/socialpublish/internal/config"
	"github.com/yourorg/socialpublish/internal/queue"
	"github.com/yourorg/socialpublish/internal/storage"
	"github.com/yourorg/socialpublish/internal/store"
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

	stores := store.New(db)
	q := queue.NewAsynq(cfg.RedisAddr)
	defer q.Close()
	obj, err := storage.NewS3(ctx, cfg.S3)
	if err != nil {
		return fmt.Errorf("open object storage: %w", err)
	}

	srv := api.New(api.Config{
		ListenAddr:      cfg.ListenAddr,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}, stores, q, obj)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		slog.Info("api server starting", "addr", cfg.ListenAddr)
		return srv.Run(ctx)
	})
	return g.Wait()
}
