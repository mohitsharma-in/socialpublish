package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	appconfig "github.com/yourorg/socialpublish/internal/config"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	direction := "up"
	if len(args) > 0 {
		direction = args[0]
	}
	cfg, err := appconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	m, err := migrate.New("file://migrations", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	switch direction {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrate up: %w", err)
		}
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrate down: %w", err)
		}
	default:
		return fmt.Errorf("unknown migration direction %q", direction)
	}
	return nil
}
