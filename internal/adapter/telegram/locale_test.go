package telegram

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"finbot/internal/domain"
)

func TestLogAttrsIncludesUserAndCommand(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := withCommand(withUser(context.Background(), domain.User{TelegramID: 7}), "add")
	log.Error("x", logAttrs(ctx, slog.String("err", "boom"))...)
	got := buf.String()
	if !bytes.Contains([]byte(got), []byte(`"user_id":7`)) {
		t.Fatalf("missing user_id: %s", got)
	}
	if !bytes.Contains([]byte(got), []byte(`"command":"add"`)) {
		t.Fatalf("missing command: %s", got)
	}
	if !bytes.Contains([]byte(got), []byte(`"err":"boom"`)) {
		t.Fatalf("missing extra: %s", got)
	}
}

func TestLogAttrsOmitsEmptyCommand(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	log.Error("x", logAttrs(context.Background(), slog.Any("err", errors.New("e")))...)
	got := buf.String()
	if bytes.Contains([]byte(got), []byte(`"command"`)) {
		t.Fatalf("unexpected command: %s", got)
	}
	if !bytes.Contains([]byte(got), []byte(`"user_id":0`)) {
		t.Fatalf("missing user_id: %s", got)
	}
}
