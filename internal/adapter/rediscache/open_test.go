package rediscache

import (
	"context"
	"testing"
	"time"
)

func TestOpenRejectsEmptyURL(t *testing.T) {
	_, err := Open(context.Background(), "  ", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenRejectsInvalidURL(t *testing.T) {
	_, err := Open(context.Background(), "://not-a-url", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCacheCanceledContextNoClient(t *testing.T) {
	c := &Cache{prefix: "t:"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, _, err := c.Get(ctx, "k"); err == nil {
		t.Fatal("expected get error")
	}
	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err == nil {
		t.Fatal("expected set error")
	}
	if err := c.Delete(ctx, "k"); err == nil {
		t.Fatal("expected delete error")
	}
}
