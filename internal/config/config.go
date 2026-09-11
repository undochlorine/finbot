package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/caarlos0/env/v9"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/joho/godotenv"

	"finbot/internal/adapter/postgres"
	"finbot/internal/adapter/rediscache"
	"finbot/internal/adapter/telegram"
	"finbot/internal/service"
)

type LogConfig struct {
	Level string `env:"LEVEL" envDefault:"info"`
}

type Config struct {
	Log      LogConfig            `envPrefix:"LOG_"`
	Bot      telegram.Config      `envPrefix:"BOT_"`
	Admin    telegram.AdminConfig `envPrefix:"ADMIN_TELEGRAM_"`
	Postgres postgres.Config      `envPrefix:"DATABASE_"`
	Redis    rediscache.Config    `envPrefix:"REDIS_"`
	Service  service.Config
}

func ParseConfig(ctx context.Context) (Config, error) {
	var cfg Config

	if envFile := os.Getenv("ENVFILE"); len(envFile) > 0 {
		if err := godotenv.Load(envFile); err != nil {
			return Config{}, fmt.Errorf("loading .env file: %w", err)
		}
	}

	if err := clearEmptyOptionalEnv(); err != nil {
		return Config{}, err
	}

	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("env, parse: %w", err)
	}

	if err := cfg.ValidateWithContext(ctx); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func (c *Config) ValidateWithContext(ctx context.Context) error {
	for _, err := range []error{
		c.Log.ValidateWithContext(ctx),
		c.Bot.ValidateWithContext(ctx),
		c.Admin.ValidateWithContext(ctx),
		c.Postgres.ValidateWithContext(ctx),
		c.Redis.ValidateWithContext(ctx),
		c.Service.ValidateWithContext(ctx),
	} {
		if err != nil {
			return fmt.Errorf("validate config via lib: %w", err)
		}
	}
	return nil
}

func (c *LogConfig) ValidateWithContext(ctx context.Context) error {
	c.Level = strings.TrimSpace(c.Level)
	return validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.Level, validation.By(requireLogLevel)),
	)
}

func (c LogConfig) SlogLevel() slog.Level {
	switch strings.ToLower(strings.TrimSpace(c.Level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func requireLogLevel(value any) error {
	raw, ok := value.(string)
	if !ok {
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
	if raw == "" {
		raw = "info"
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
}

func clearEmptyOptionalEnv() error {
	for _, key := range []string{
		"LOG_LEVEL",
		"BOT_TTL",
		"TRIAL_DURATION",
		"ADMIN_TELEGRAM_ID",
		"DEFAULT_CURRENCY",
	} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			if err := os.Unsetenv(key); err != nil {
				return fmt.Errorf("unset %s: %w", key, err)
			}
		}
	}
	return nil
}
