package memorycache

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"finbot/internal/ports"
)

var _ ports.Cache = (*Cache)(nil)

type Clock interface {
	Now() time.Time
}

type item struct {
	value     []byte
	expiresAt time.Time
}

type Cache struct {
	clock Clock
	mu    sync.Mutex
	items map[string]item
}

func New(clock Clock) *Cache {
	return &Cache{
		clock: clock,
		items: make(map[string]item),
	}
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, fmt.Errorf("cache get: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	it, ok := c.items[key]
	if !ok || !c.clock.Now().Before(it.expiresAt) {
		delete(c.items, key)
		return nil, false, nil
	}
	return bytes.Clone(it.value), true, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		delete(c.items, key)
		return nil
	}
	c.items[key] = item{
		value:     bytes.Clone(value),
		expiresAt: c.clock.Now().Add(ttl),
	}
	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}
