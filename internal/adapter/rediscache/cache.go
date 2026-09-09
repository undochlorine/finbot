package rediscache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"finbot/internal/ports"
)

var _ ports.Cache = (*Cache)(nil)

type Cache struct {
	client *redis.Client
	prefix string
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, fmt.Errorf("cache get: %w", err)
	}

	got, err := c.client.Get(ctx, c.key(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cache get: %w", err)
	}
	return bytes.Clone(got), true, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	if ttl <= 0 {
		return c.Delete(ctx, key)
	}
	if err := c.client.Set(ctx, c.key(key), value, ttl).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}
	if err := c.client.Del(ctx, c.key(key)).Err(); err != nil {
		return fmt.Errorf("cache delete: %w", err)
	}
	return nil
}

func (c *Cache) key(k string) string {
	return c.prefix + k
}
