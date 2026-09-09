//go:build integration

package rediscache

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

const defaultTestURL = "redis://:finbot@127.0.0.1:6379/0"

func TestOpenAppliesPoolAndTimeouts(t *testing.T) {
	c := mustOpen(t, testPrefix(t))
	opt := c.client.Options()
	if opt.DialTimeout != DialTimeout {
		t.Fatalf("DialTimeout = %s, want %s", opt.DialTimeout, DialTimeout)
	}
	if opt.ReadTimeout != ReadTimeout {
		t.Fatalf("ReadTimeout = %s, want %s", opt.ReadTimeout, ReadTimeout)
	}
	if opt.WriteTimeout != WriteTimeout {
		t.Fatalf("WriteTimeout = %s, want %s", opt.WriteTimeout, WriteTimeout)
	}
	if opt.PoolTimeout != PoolTimeout {
		t.Fatalf("PoolTimeout = %s, want %s", opt.PoolTimeout, PoolTimeout)
	}
	if opt.PoolSize != PoolSize {
		t.Fatalf("PoolSize = %d, want %d", opt.PoolSize, PoolSize)
	}
	if opt.MinIdleConns != MinIdleConns {
		t.Fatalf("MinIdleConns = %d, want %d", opt.MinIdleConns, MinIdleConns)
	}
	if opt.ConnMaxIdleTime != ConnMaxIdleTime {
		t.Fatalf("ConnMaxIdleTime = %s, want %s", opt.ConnMaxIdleTime, ConnMaxIdleTime)
	}
	if opt.ConnMaxLifetime != ConnMaxLifetime {
		t.Fatalf("ConnMaxLifetime = %s, want %s", opt.ConnMaxLifetime, ConnMaxLifetime)
	}
	if opt.MaxRetries != MaxRetries {
		t.Fatalf("MaxRetries = %d, want %d", opt.MaxRetries, MaxRetries)
	}
}

func TestOpenEmptyPrefixUsesDefault(t *testing.T) {
	c, err := Open(context.Background(), testURL(), "  ")
	if err != nil {
		t.Fatalf("open redis: %v (start local Redis with docker compose up -d)", err)
	}
	logical := "fsm:prefix-check:" + strings.ReplaceAll(t.Name(), "/", "_")
	rawKey := DefaultPrefix + logical
	t.Cleanup(func() {
		if _, err := c.client.Del(context.Background(), rawKey).Result(); err != nil {
			t.Errorf("cleanup key: %v", err)
		}
		if err := c.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})
	if c.prefix != DefaultPrefix {
		t.Fatalf("prefix = %q, want %q", c.prefix, DefaultPrefix)
	}

	ctx := context.Background()
	if err := c.Set(ctx, logical, []byte("v"), time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := c.client.Get(ctx, rawKey).Bytes()
	if err != nil {
		t.Fatalf("raw get: %v", err)
	}
	if string(got) != "v" {
		t.Fatalf("raw value = %q, want %q", got, "v")
	}
}

func TestOpenFailsWhenUnreachable(t *testing.T) {
	_, err := Open(context.Background(), "redis://:finbot@127.0.0.1:1/0", testPrefix(t))
	if err == nil {
		t.Fatal("expected error when redis is unreachable")
	}
}

func TestCache(t *testing.T) {
	ttl := time.Minute
	expireTTL := 200 * time.Millisecond
	expireWait := 500 * time.Millisecond

	tests := []struct {
		name         string
		put          bool
		putVal       []byte
		putTTL       time.Duration
		overwrite    bool
		overwriteVal []byte
		overwriteTTL time.Duration
		wait         time.Duration
		del          bool
		getKey       string
		want         []byte
		wantOK       bool
	}{
		{
			name:   "hit",
			put:    true,
			putVal: []byte("add"),
			putTTL: ttl,
			getKey: "fsm:1",
			want:   []byte("add"),
			wantOK: true,
		},
		{
			name:   "unknown key",
			getKey: "missing",
		},
		{
			name:   "other key is miss",
			put:    true,
			putVal: []byte("add"),
			putTTL: ttl,
			getKey: "other",
		},
		{
			name:   "delete is miss",
			put:    true,
			putVal: []byte("add"),
			putTTL: ttl,
			del:    true,
			getKey: "fsm:1",
		},
		{
			name:   "expired is miss",
			put:    true,
			putVal: []byte("add"),
			putTTL: expireTTL,
			wait:   expireWait,
			getKey: "fsm:1",
		},
		{
			name:   "still valid before expiry",
			put:    true,
			putVal: []byte("add"),
			putTTL: ttl,
			getKey: "fsm:1",
			want:   []byte("add"),
			wantOK: true,
		},
		{
			name:   "zero ttl is miss",
			put:    true,
			putVal: []byte("add"),
			getKey: "fsm:1",
		},
		{
			name:   "negative ttl is miss",
			put:    true,
			putVal: []byte("add"),
			putTTL: -time.Second,
			getKey: "fsm:1",
		},
		{
			name:         "overwrite",
			put:          true,
			putVal:       []byte("add"),
			putTTL:       ttl,
			overwrite:    true,
			overwriteVal: []byte("spend"),
			overwriteTTL: ttl,
			getKey:       "fsm:1",
			want:         []byte("spend"),
			wantOK:       true,
		},
		{
			name:   "empty value is a hit",
			put:    true,
			putVal: []byte{},
			putTTL: ttl,
			getKey: "fsm:1",
			want:   []byte{},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mustOpen(t, testPrefix(t))
			ctx := context.Background()
			const storedKey = "fsm:1"

			if tt.put {
				if err := c.Set(ctx, storedKey, tt.putVal, tt.putTTL); err != nil {
					t.Fatalf("set: %v", err)
				}
			}
			if tt.overwrite {
				if err := c.Set(ctx, storedKey, tt.overwriteVal, tt.overwriteTTL); err != nil {
					t.Fatalf("overwrite: %v", err)
				}
			}
			if tt.wait > 0 {
				time.Sleep(tt.wait)
			}
			if tt.del {
				if err := c.Delete(ctx, storedKey); err != nil {
					t.Fatalf("delete: %v", err)
				}
			}

			got, ok, err := c.Get(ctx, tt.getKey)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCacheKeyIsolation(t *testing.T) {
	c := mustOpen(t, testPrefix(t))
	ctx := context.Background()

	if err := c.Set(ctx, "fsm:111", []byte("alice"), time.Minute); err != nil {
		t.Fatalf("set alice: %v", err)
	}
	if err := c.Set(ctx, "fsm:222", []byte("bob"), time.Minute); err != nil {
		t.Fatalf("set bob: %v", err)
	}

	got, ok, err := c.Get(ctx, "fsm:111")
	if err != nil || !ok || string(got) != "alice" {
		t.Fatalf("alice get = %q ok=%v err=%v", got, ok, err)
	}
	got, ok, err = c.Get(ctx, "fsm:222")
	if err != nil || !ok || string(got) != "bob" {
		t.Fatalf("bob get = %q ok=%v err=%v", got, ok, err)
	}

	if err := c.Delete(ctx, "fsm:111"); err != nil {
		t.Fatalf("delete alice: %v", err)
	}
	_, ok, err = c.Get(ctx, "fsm:111")
	if err != nil || ok {
		t.Fatalf("alice after delete ok=%v err=%v, want miss", ok, err)
	}
	got, ok, err = c.Get(ctx, "fsm:222")
	if err != nil || !ok || string(got) != "bob" {
		t.Fatalf("bob after alice delete = %q ok=%v err=%v", got, ok, err)
	}
}

func TestCacheCanceledContext(t *testing.T) {
	c := mustOpen(t, testPrefix(t))
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

func mustOpen(t *testing.T, prefix string) *Cache {
	t.Helper()
	c, err := Open(context.Background(), testURL(), prefix)
	if err != nil {
		t.Fatalf("open redis: %v (start local Redis with docker compose up -d)", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		keys, err := c.client.Keys(ctx, c.prefix+"*").Result()
		if err == nil && len(keys) > 0 {
			if _, err := c.client.Del(ctx, keys...).Result(); err != nil {
				t.Errorf("cleanup keys: %v", err)
			}
		}
		if err := c.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})
	return c
}

func testURL() string {
	if u := strings.TrimSpace(os.Getenv("REDIS_TEST_URL")); u != "" {
		return u
	}
	return defaultTestURL
}

func testPrefix(t *testing.T) string {
	t.Helper()
	return "finbot:test:" + strings.ReplaceAll(t.Name(), "/", "_") + ":"
}
