package config

import (
	"log/slog"
	"testing"
	"time"
)

const testDatabaseURL = "postgres://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable"

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name:    "missing token",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name:    "missing database url",
			env:     map[string]string{"BOT_TOKEN": "tok"},
			wantErr: true,
		},
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
			name: "defaults",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": testDatabaseURL,
			},
			want: Config{
				BotToken:        "tok",
				LogLevel:        slog.LevelInfo,
				TrialDuration:   defaultTrialDuration,
				AdminTelegramID: 0,
				DefaultCurrency: defaultCurrency,
				DatabaseURL:     testDatabaseURL,
			},
		},
		{
			name: "overrides",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"LOG_LEVEL":         "DEBUG",
				"TRIAL_DURATION":    "24h",
				"ADMIN_TELEGRAM_ID": "12345",
				"DEFAULT_CURRENCY":  "EUR",
				"DATABASE_URL":      "postgres://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable",
				"POSTGRES_TEST_URL": "postgres://finbot:finbot@127.0.0.1:5432/finbot_test?sslmode=disable",
			},
			want: Config{
				BotToken:        "tok",
				LogLevel:        slog.LevelDebug,
				TrialDuration:   24 * time.Hour,
				AdminTelegramID: 12345,
				DefaultCurrency: "EUR",
				DatabaseURL:     "postgres://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable",
				PostgresTestURL: "postgres://finbot:finbot@127.0.0.1:5432/finbot_test?sslmode=disable",
			},
		},
		{
			name: "trims postgres urls",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"DATABASE_URL":      "  postgres://local/finbot  ",
				"POSTGRES_TEST_URL": "  postgres://local/finbot_test  ",
			},
			want: Config{
				BotToken:        "tok",
				LogLevel:        slog.LevelInfo,
				TrialDuration:   defaultTrialDuration,
				AdminTelegramID: 0,
				DefaultCurrency: defaultCurrency,
				DatabaseURL:     "postgres://local/finbot",
				PostgresTestURL: "postgres://local/finbot_test",
			},
		},
		{
			name: "postgresql scheme",
			env: map[string]string{
				"BOT_TOKEN":    "tok",
				"DATABASE_URL": "postgresql://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable",
			},
			want: Config{
				BotToken:        "tok",
				LogLevel:        slog.LevelInfo,
				TrialDuration:   defaultTrialDuration,
				AdminTelegramID: 0,
				DefaultCurrency: defaultCurrency,
				DatabaseURL:     "postgresql://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable",
			},
		},
		{
			name: "empty currency uses USD",
			env: map[string]string{
				"BOT_TOKEN":        "tok",
				"DATABASE_URL":     testDatabaseURL,
				"DEFAULT_CURRENCY": "   ",
			},
			want: Config{
				BotToken:        "tok",
				LogLevel:        slog.LevelInfo,
				TrialDuration:   defaultTrialDuration,
				AdminTelegramID: 0,
				DefaultCurrency: "USD",
				DatabaseURL:     testDatabaseURL,
			},
		},
		{
			name: "currency kept as trimmed",
			env: map[string]string{
				"BOT_TOKEN":        "tok",
				"DATABASE_URL":     testDatabaseURL,
				"DEFAULT_CURRENCY": " eur ",
			},
			want: Config{
				BotToken:        "tok",
				LogLevel:        slog.LevelInfo,
				TrialDuration:   defaultTrialDuration,
				AdminTelegramID: 0,
				DefaultCurrency: "eur",
				DatabaseURL:     testDatabaseURL,
			},
		},
		{
			name: "zero trial means none",
			env: map[string]string{
				"BOT_TOKEN":      "tok",
				"DATABASE_URL":   testDatabaseURL,
				"TRIAL_DURATION": "0",
			},
			want: Config{
				BotToken:        "tok",
				LogLevel:        slog.LevelInfo,
				TrialDuration:   0,
				AdminTelegramID: 0,
				DefaultCurrency: defaultCurrency,
				DatabaseURL:     testDatabaseURL,
			},
		},
		{
			name: "zero admin id is unset",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"DATABASE_URL":      testDatabaseURL,
				"ADMIN_TELEGRAM_ID": "0",
			},
			want: Config{
				BotToken:        "tok",
				LogLevel:        slog.LevelInfo,
				TrialDuration:   defaultTrialDuration,
				AdminTelegramID: 0,
				DefaultCurrency: defaultCurrency,
				DatabaseURL:     testDatabaseURL,
			},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BOT_TOKEN", "")
			t.Setenv("LOG_LEVEL", "")
			t.Setenv("TRIAL_DURATION", "")
			t.Setenv("ADMIN_TELEGRAM_ID", "")
			t.Setenv("DEFAULT_CURRENCY", "")
			t.Setenv("DATABASE_URL", "")
			t.Setenv("POSTGRES_TEST_URL", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := Load()
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

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		in      string
		want    slog.Level
		wantErr bool
	}{
		{in: "", want: slog.LevelInfo},
		{in: "warn", want: slog.LevelWarn},
		{in: "error", want: slog.LevelError},
		{in: "trace", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseLogLevel(tt.in)
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
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
