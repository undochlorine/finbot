package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMissingToken(t *testing.T) {
	t.Setenv("BOT_TOKEN", "")
	t.Setenv("SQLITE_PATH", filepath.Join(t.TempDir(), "finbot.db"))
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("TRIAL_DURATION", "")
	t.Setenv("ADMIN_TELEGRAM_ID", "")

	err := run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "BOT_TOKEN is required") {
		t.Fatalf("got %q", err.Error())
	}
}

func TestRunSQLiteDirMissing(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(parent, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Setenv("BOT_TOKEN", "123:token")
	t.Setenv("SQLITE_PATH", filepath.Join(parent, "finbot.db"))
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("TRIAL_DURATION", "")
	t.Setenv("ADMIN_TELEGRAM_ID", "")

	err := run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create sqlite directory") {
		t.Fatalf("got %q", err.Error())
	}
}
