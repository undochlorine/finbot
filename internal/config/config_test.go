package config

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"finbot/internal/adapter/postgres"
	"finbot/internal/adapter/rediscache"
	"finbot/internal/adapter/telegram"
	"finbot/internal/service"
)

const (
	testDatabaseURL = "postgres://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable"
	testRedisURL    = "redis://:finbot@127.0.0.1:6379/0"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{name: "missing token", env: map[string]string{}, wantErr: true},
		{name: "missing database url", env: map[string]string{"BOT_TOKEN": "tok"}, wantErr: true},
		{
			name: "blank database url",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": "   ",
			},
			wantErr: true,
		},
		{
			name: "invalid database url",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": "://not-a-url",
			},
			wantErr: true,
		},
		{
			name: "non postgres database url",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": "http://127.0.0.1:5432/finbot",
			},
			wantErr: true,
		},
		{
			name: "missing redis url",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
			},
			wantErr: true,
		},
		{
			name: "defaults",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"REDIS_URL":    testRedisURL,
			},
			want: validCfg("tok", testDatabaseURL, testRedisURL),
		},
		{
			name: "overrides",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"LOG_LEVEL":         "DEBUG",
				"TRIAL_DURATION":    "24h",
				"ADMIN_TELEGRAM_ID": "12345",
				"DEFAULT_CURRENCY":  "EUR",
				"DATABASE_URL":      testDatabaseURL,
				"REDIS_URL":         testRedisURL,
				"BOT_TTL":           "30m",
			},
			want: Config{
				Log:   LogConfig{Level: "DEBUG"},
				Bot:   telegram.Config{Token: "tok", TTL: "30m"},
				Admin: telegram.AdminConfig{ID: 12345},
				Postgres: postgres.Config{
					URL: testDatabaseURL,
				},
				Redis: rediscache.Config{URL: testRedisURL},
				Service: service.Config{
					Trial:   service.TrialConfig{Duration: "24h"},
					Default: service.DefaultConfig{Currency: "EUR"},
				},
			},
		},
		{
			name: "trims postgres urls",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": "  postgres://local/finbot  ",
				"REDIS_URL":    testRedisURL,
			},
			want: validCfg("tok", "postgres://local/finbot", testRedisURL),
		},
		{
			name: "trims redis urls",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"REDIS_URL":    "  redis://:finbot@127.0.0.1:6379/0  ",
			},
			want: validCfg("tok", testDatabaseURL, "redis://:finbot@127.0.0.1:6379/0"),
		},
		{
			name: "rediss scheme",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"REDIS_URL":    "rediss://cache.example:6379/0",
			},
			want: validCfg("tok", testDatabaseURL, "rediss://cache.example:6379/0"),
		},
		{
			name: "blank redis url",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"REDIS_URL":    "   ",
			},
			wantErr: true,
		},
		{
			name: "invalid redis url",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"REDIS_URL":    "://not-a-url",
			},
			wantErr: true,
		},
		{
			name: "non redis url",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"REDIS_URL":    "http://127.0.0.1:6379",
			},
			wantErr: true,
		},
		{
			name: "redis url empty host",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"REDIS_URL":    "redis:///0",
			},
			wantErr: true,
		},
		{
			name: "postgresql scheme",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": "postgresql://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable",
				"REDIS_URL":    testRedisURL,
			},
			want: validCfg("tok", "postgresql://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable", testRedisURL),
		},
		{
			name: "empty currency uses USD",
			env: map[string]string{
				"BOT_TOKEN":        "tok",
				"DATABASE_URL":     testDatabaseURL,
				"REDIS_URL":        testRedisURL,
				"DEFAULT_CURRENCY": "   ",
			},
			want: validCfg("tok", testDatabaseURL, testRedisURL),
		},
		{
			name: "currency kept as trimmed",
			env: map[string]string{
				"BOT_TOKEN":        "tok",
				"DATABASE_URL":     testDatabaseURL,
				"REDIS_URL":        testRedisURL,
				"DEFAULT_CURRENCY": " eur ",
			},
			want: func() Config {
				cfg := validCfg("tok", testDatabaseURL, testRedisURL)
				cfg.Service.Default.Currency = "eur"
				return cfg
			}(),
		},
		{
			name: "zero trial means none",
			env: map[string]string{
				"BOT_TOKEN":      "tok",
				"DATABASE_URL":   testDatabaseURL,
				"REDIS_URL":      testRedisURL,
				"TRIAL_DURATION": "0",
			},
			want: func() Config {
				cfg := validCfg("tok", testDatabaseURL, testRedisURL)
				cfg.Service.Trial.Duration = "0"
				return cfg
			}(),
		},
		{
			name: "zero admin id is unset",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"DATABASE_URL":      testDatabaseURL,
				"REDIS_URL":         testRedisURL,
				"ADMIN_TELEGRAM_ID": "0",
			},
			want: validCfg("tok", testDatabaseURL, testRedisURL),
		},
		{
			name: "invalid admin id",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"DATABASE_URL":      testDatabaseURL,
				"ADMIN_TELEGRAM_ID": "not-an-id",
			},
			wantErr: true,
		},
		{
			name: "negative admin id",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"DATABASE_URL":      testDatabaseURL,
				"ADMIN_TELEGRAM_ID": "-1",
			},
			wantErr: true,
		},
		{
			name: "invalid trial duration",
			env: map[string]string{
				"BOT_TOKEN":      "tok",
				"DATABASE_URL":   testDatabaseURL,
				"TRIAL_DURATION": "not-a-duration",
			},
			wantErr: true,
		},
		{
			name: "negative trial duration",
			env: map[string]string{
				"BOT_TOKEN":      "tok",
				"DATABASE_URL":   testDatabaseURL,
				"TRIAL_DURATION": "-1h",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"LOG_LEVEL":    "verbose",
			},
			wantErr: true,
		},
		{
			name: "zero bot ttl rejected",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"REDIS_URL":    testRedisURL,
				"BOT_TTL":      "0",
			},
			wantErr: true,
		},
		{
			name: "negative bot ttl rejected",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
				"REDIS_URL":    testRedisURL,
				"BOT_TTL":      "-1h",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetParseEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := ParseConfig(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseConfigENVFILE(t *testing.T) {
	resetParseEnv(t)

	path := filepath.Join(t.TempDir(), ".env")
	content := "BOT_TOKEN=from-file\nDATABASE_URL=" + testDatabaseURL + "\nREDIS_URL=" + testRedisURL + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENVFILE", path)
	if err := os.Unsetenv("DATABASE_URL"); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv("REDIS_URL"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BOT_TOKEN", "from-process")

	got, err := ParseConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Bot.Token != "from-process" {
		t.Fatalf("process env should win, got %q", got.Bot.Token)
	}
}

func TestParseConfigENVFILEMissing(t *testing.T) {
	resetParseEnv(t)
	t.Setenv("ENVFILE", filepath.Join(t.TempDir(), "missing.env"))

	_, err := ParseConfig(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLogConfigSlogLevel(t *testing.T) {
	if (LogConfig{Level: ""}).SlogLevel() != slog.LevelInfo {
		t.Fatal("empty defaults to info")
	}
	if (LogConfig{Level: "warn"}).SlogLevel() != slog.LevelWarn {
		t.Fatal("warn")
	}
}

func validCfg(token, databaseURL, redisURL string) Config {
	return Config{
		Log:      LogConfig{Level: "info"},
		Bot:      telegram.Config{Token: token, TTL: "1h"},
		Postgres: postgres.Config{URL: databaseURL},
		Redis:    rediscache.Config{URL: redisURL},
		Service: service.Config{
			Trial:   service.TrialConfig{Duration: "168h"},
			Default: service.DefaultConfig{Currency: "USD"},
		},
	}
}

func resetParseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ENVFILE", "")
	t.Setenv("BOT_TOKEN", "")
	t.Setenv("BOT_TTL", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("TRIAL_DURATION", "")
	t.Setenv("ADMIN_TELEGRAM_ID", "")
	t.Setenv("DEFAULT_CURRENCY", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
}
