package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"finbot/internal/adapter/clock"
	"finbot/internal/adapter/memorycache"
	"finbot/internal/adapter/sqlite"
	"finbot/internal/adapter/telegram"
	"finbot/internal/config"
	"finbot/internal/domain"
	"finbot/internal/service"
)

const (
	sqliteDirPerm       = 0o750
	telegramHTTPTimeout = time.Minute
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel})))

	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), sqliteDirPerm); err != nil {
		return fmt.Errorf("create sqlite directory: %w", err)
	}

	db, err := sqlite.Open(ctx, cfg.SQLitePath)
	if err != nil {
		return fmt.Errorf("sqlite: %w", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			slog.Error("close sqlite", slog.Any("err", cerr))
		}
	}()

	clk := clock.New()
	svc := service.New(
		sqlite.NewBankRepository(db),
		sqlite.NewUserRepository(db),
		clk,
		cfg.TrialDuration,
	)

	b, err := telegram.New(
		cfg.BotToken,
		svc,
		memorycache.New(clk),
		&http.Client{Timeout: telegramHTTPTimeout},
	)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	if cfg.AdminTelegramID != 0 {
		b.SetAdmin(domain.UserID(cfg.AdminTelegramID), b)
	}

	b.Start(ctx)
	return nil
}
