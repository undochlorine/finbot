package config

import (
	"log/slog"
	"testing"
	"time"
)

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
			name: "defaults",
			env:  map[string]string{"BOT_TOKEN": "tok"},
			want: Config{
				BotToken:        "tok",
				SQLitePath:      defaultSQLitePath,
				LogLevel:        slog.LevelInfo,
				TrialDuration:   defaultTrialDuration,
				AdminTelegramID: 0,
			},
		},
		{
			name: "overrides",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"SQLITE_PATH":       "/var/lib/finbot/finbot.db",
				"LOG_LEVEL":         "DEBUG",
				"TRIAL_DURATION":    "24h",
				"ADMIN_TELEGRAM_ID": "12345",
			},
			want: Config{
				BotToken:        "tok",
				SQLitePath:      "/var/lib/finbot/finbot.db",
				LogLevel:        slog.LevelDebug,
				TrialDuration:   24 * time.Hour,
				AdminTelegramID: 12345,
			},
		},
		{
			name: "zero trial means none",
			env: map[string]string{
				"BOT_TOKEN":      "tok",
				"TRIAL_DURATION": "0",
			},
			want: Config{
				BotToken:        "tok",
				SQLitePath:      defaultSQLitePath,
				LogLevel:        slog.LevelInfo,
				TrialDuration:   0,
				AdminTelegramID: 0,
			},
		},
		{
			name: "zero admin id is unset",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"ADMIN_TELEGRAM_ID": "0",
			},
			want: Config{
				BotToken:        "tok",
				SQLitePath:      defaultSQLitePath,
				LogLevel:        slog.LevelInfo,
				TrialDuration:   defaultTrialDuration,
				AdminTelegramID: 0,
			},
		},
		{
			name: "invalid admin id",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"ADMIN_TELEGRAM_ID": "not-an-id",
			},
			wantErr: true,
		},
		{
			name: "negative admin id",
			env: map[string]string{
				"BOT_TOKEN":         "tok",
				"ADMIN_TELEGRAM_ID": "-1",
			},
			wantErr: true,
		},
		{
			name: "invalid trial duration",
			env: map[string]string{
				"BOT_TOKEN":      "tok",
				"TRIAL_DURATION": "not-a-duration",
			},
			wantErr: true,
		},
		{
			name: "negative trial duration",
			env: map[string]string{
				"BOT_TOKEN":      "tok",
				"TRIAL_DURATION": "-1h",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			env: map[string]string{
				"BOT_TOKEN": "tok",
				"LOG_LEVEL": "verbose",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BOT_TOKEN", "")
			t.Setenv("SQLITE_PATH", "")
			t.Setenv("LOG_LEVEL", "")
			t.Setenv("TRIAL_DURATION", "")
			t.Setenv("ADMIN_TELEGRAM_ID", "")
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
