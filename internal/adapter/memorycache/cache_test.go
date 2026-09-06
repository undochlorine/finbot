package memorycache

import (
	"bytes"
	"context"
	"testing"
	"time"

	"finbot/internal/ports"
)

type stubClock struct {
	now time.Time
}

func (s *stubClock) Now() time.Time { return s.now }

var _ ports.Clock = (*stubClock)(nil)

func fixedClock() *stubClock {
	return &stubClock{now: time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)}
}

func TestCache(t *testing.T) {
	now := fixedClock().now
	ttl := time.Minute

	tests := []struct {
		name         string
		put          bool
		putVal       []byte
		putTTL       time.Duration
		overwrite    bool
		overwriteVal []byte
		overwriteTTL time.Duration
		advance      time.Duration
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
			name:    "expired is miss",
			put:     true,
			putVal:  []byte("add"),
			putTTL:  ttl,
			advance: ttl,
			getKey:  "fsm:1",
		},
		{
			name:    "still valid before expiry",
			put:     true,
			putVal:  []byte("add"),
			putTTL:  ttl,
			advance: ttl - time.Nanosecond,
			getKey:  "fsm:1",
			want:    []byte("add"),
			wantOK:  true,
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
			name:         "overwrite with shorter ttl expires",
			put:          true,
			putVal:       []byte("add"),
			putTTL:       time.Hour,
			overwrite:    true,
			overwriteVal: []byte("spend"),
			overwriteTTL: ttl,
			advance:      ttl,
			getKey:       "fsm:1",
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
			clock := &stubClock{now: now}
			c := New(clock)
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
			clock.now = clock.now.Add(tt.advance)
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
				t.Fatalf("ok=%v, want %v", ok, tt.wantOK)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCacheIsolatesSlices(t *testing.T) {
	c := New(fixedClock())
	ctx := context.Background()

	in := []byte("add")
	if err := c.Set(ctx, "k", in, time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	in[0] = 'x'

	got, ok, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok || string(got) != "add" {
		t.Fatalf("got %q ok=%v, want add", got, ok)
	}
	got[0] = 'y'

	again, ok, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("get again: %v", err)
	}
	if !ok || string(again) != "add" {
		t.Fatalf("got %q ok=%v, want add", again, ok)
	}
}

func TestCacheCanceledContext(t *testing.T) {
	c := New(fixedClock())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, _, err := c.Get(ctx, "k"); err == nil {
		t.Fatal("get: expected error")
	}
	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err == nil {
		t.Fatal("set: expected error")
	}
	if err := c.Delete(ctx, "k"); err == nil {
		t.Fatal("delete: expected error")
	}
}
