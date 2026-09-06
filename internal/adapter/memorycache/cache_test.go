package memorycache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/memorycache/mocks"
)

func fixedNow() time.Time {
	return time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
}

func newTestClock(t *testing.T, now time.Time) (*mocks.MockClock, func(time.Duration)) {
	t.Helper()
	current := now
	clock := mocks.NewMockClock(t)
	clock.EXPECT().Now().RunAndReturn(func() time.Time { return current }).Maybe()
	return clock, func(d time.Duration) { current = current.Add(d) }
}

func TestCache(t *testing.T) {
	now := fixedNow()
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
			clock, advance := newTestClock(t, now)
			c := New(clock)
			ctx := context.Background()
			const storedKey = "fsm:1"

			if tt.put {
				require.NoError(t, c.Set(ctx, storedKey, tt.putVal, tt.putTTL))
			}
			if tt.overwrite {
				require.NoError(t, c.Set(ctx, storedKey, tt.overwriteVal, tt.overwriteTTL))
			}
			advance(tt.advance)
			if tt.del {
				require.NoError(t, c.Delete(ctx, storedKey))
			}

			got, ok, err := c.Get(ctx, tt.getKey)
			require.NoError(t, err)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCacheIsolatesSlices(t *testing.T) {
	clock, _ := newTestClock(t, fixedNow())
	c := New(clock)
	ctx := context.Background()

	in := []byte("add")
	require.NoError(t, c.Set(ctx, "k", in, time.Minute))
	in[0] = 'x'

	got, ok, err := c.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "add", string(got))
	got[0] = 'y'

	again, ok, err := c.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "add", string(again))
}

func TestCacheCanceledContext(t *testing.T) {
	clock, _ := newTestClock(t, fixedNow())
	c := New(clock)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := c.Get(ctx, "k")
	require.Error(t, err)
	require.Error(t, c.Set(ctx, "k", []byte("v"), time.Minute))
	require.Error(t, c.Delete(ctx, "k"))
}
