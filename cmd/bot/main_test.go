package main

import (
	"context"
	"strings"
	"testing"
)

func TestRunMissingToken(t *testing.T) {
	t.Setenv("BOT_TOKEN", "")
	t.Setenv("DATABASE_URL", "postgres://finbot:finbot@127.0.0.1:5432/finbot?sslmode=disable")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("TRIAL_DURATION", "")
	t.Setenv("ADMIN_TELEGRAM_ID", "")
	t.Setenv("REDIS_URL", "")

	err := run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "BOT_TOKEN is required") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestRunMissingDatabaseURL(t *testing.T) {
	t.Setenv("BOT_TOKEN", "123:token")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("TRIAL_DURATION", "")
	t.Setenv("ADMIN_TELEGRAM_ID", "")
	t.Setenv("REDIS_URL", "")

	err := run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestRunInvalidDatabaseURL(t *testing.T) {
	t.Setenv("BOT_TOKEN", "123:token")
	t.Setenv("DATABASE_URL", "://not-a-url")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("TRIAL_DURATION", "")
	t.Setenv("ADMIN_TELEGRAM_ID", "")
	t.Setenv("REDIS_URL", "")

	err := run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL is invalid") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestRunUnreachablePostgres(t *testing.T) {
	t.Setenv("BOT_TOKEN", "123:token")
	t.Setenv("DATABASE_URL", "postgres://finbot:finbot@127.0.0.1:1/finbot?sslmode=disable")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("TRIAL_DURATION", "")
	t.Setenv("ADMIN_TELEGRAM_ID", "")
	t.Setenv("REDIS_URL", "")

	err := run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "postgres:") {
		t.Fatalf("got %q", err.Error())
	}
}
