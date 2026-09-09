package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"finbot/internal/adapter/clock"
	"finbot/internal/adapter/postgres"
	"finbot/internal/adapter/rediscache"
	"finbot/internal/adapter/telegram"
	"finbot/internal/config"
	"finbot/internal/domain"
	"finbot/internal/service"
)

const telegramHTTPTimeout = time.Minute

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

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			slog.Error("close postgres", slog.Any("err", cerr))
		}
	}()

	cache, err := rediscache.Open(ctx, cfg.RedisURL, "")
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	defer func() {
		if cerr := cache.Close(); cerr != nil {
			slog.Error("close redis", slog.Any("err", cerr))
		}
	}()

	clk := clock.New()
	svc := service.New(
		postgres.NewBankRepository(db),
		postgres.NewUserRepository(db),
		postgres.NewOperationRepository(db),
		postgres.NewTransactor(db),
		clk,
		cfg.TrialDuration,
		cfg.DefaultCurrency,
	)

	b, err := telegram.New(
		cfg.BotToken,
		svc,
		cache,
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
