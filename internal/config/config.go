package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLogLevel      = "info"
	defaultTrialDuration = 168 * time.Hour
	defaultCurrency      = "USD"
	dotEnvPath           = ".env"
)

type Config struct {
	BotToken        string
	LogLevel        slog.Level
	TrialDuration   time.Duration
	AdminTelegramID int64
	DefaultCurrency string
	DatabaseURL     string
	PostgresTestURL string
	RedisURL        string
	RedisTestURL    string
}

func Load() (Config, error) {
	if err := applyDotEnv(dotEnvPath); err != nil {
		return Config{}, err
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("BOT_TOKEN is required")
	}

	databaseURL, err := parseDatabaseURL(os.Getenv("DATABASE_URL"))
	if err != nil {
		return Config{}, err
	}

	level, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return Config{}, err
	}

	trial, err := parseTrialDuration(os.Getenv("TRIAL_DURATION"))
	if err != nil {
		return Config{}, err
	}

	adminID, err := parseAdminTelegramID(os.Getenv("ADMIN_TELEGRAM_ID"))
	if err != nil {
		return Config{}, err
	}

	redisURL, err := parseRedisURL(os.Getenv("REDIS_URL"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		BotToken:        token,
		LogLevel:        level,
		TrialDuration:   trial,
		AdminTelegramID: adminID,
		DefaultCurrency: parseDefaultCurrency(os.Getenv("DEFAULT_CURRENCY")),
		DatabaseURL:     databaseURL,
		PostgresTestURL: strings.TrimSpace(os.Getenv("POSTGRES_TEST_URL")),
		RedisURL:        redisURL,
		RedisTestURL:    strings.TrimSpace(os.Getenv("REDIS_TEST_URL")),
	}, nil
}

func parseDatabaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("DATABASE_URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}
	if (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		return "", fmt.Errorf("DATABASE_URL is invalid")
	}
	return raw, nil
}

func parseRedisURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("REDIS_URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("REDIS_URL is invalid: %w", err)
	}
	if (u.Scheme != "redis" && u.Scheme != "rediss") || u.Host == "" {
		return "", fmt.Errorf("REDIS_URL is invalid")
	}
	return raw, nil
}

func parseDefaultCurrency(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultCurrency
	}
	return raw
}

func parseLogLevel(raw string) (slog.Level, error) {
	if raw == "" {
		raw = defaultLogLevel
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
}

func parseTrialDuration(raw string) (time.Duration, error) {
	if raw == "" {
		return defaultTrialDuration, nil
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("TRIAL_DURATION: %w", err)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("TRIAL_DURATION must be >= 0")
	}
	return parsed, nil
}

func parseAdminTelegramID(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("ADMIN_TELEGRAM_ID must be a Telegram user id")
	}
	if id < 0 {
		return 0, fmt.Errorf("ADMIN_TELEGRAM_ID must be >= 0")
	}
	return id, nil
}

func applyDotEnv(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // path is the fixed local .env file
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s: invalid line %q", path, line)
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}
	return nil
}
