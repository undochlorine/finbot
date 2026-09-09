package rediscache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DefaultPrefix   = "finbot:v1:"
	DialTimeout     = 3 * time.Second
	ReadTimeout     = 400 * time.Millisecond
	WriteTimeout    = 400 * time.Millisecond
	PoolTimeout     = time.Second
	PoolSize        = 10
	MinIdleConns    = 2
	ConnMaxIdleTime = 5 * time.Minute
	ConnMaxLifetime = 30 * time.Minute
	MaxRetries      = 1
)

func Open(ctx context.Context, url, prefix string) (*Cache, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("redis url is required")
	}

	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	opt.DialTimeout = DialTimeout
	opt.ReadTimeout = ReadTimeout
	opt.WriteTimeout = WriteTimeout
	opt.PoolTimeout = PoolTimeout
	opt.PoolSize = PoolSize
	opt.MinIdleConns = MinIdleConns
	opt.ConnMaxIdleTime = ConnMaxIdleTime
	opt.ConnMaxLifetime = ConnMaxLifetime
	opt.MaxRetries = MaxRetries

	client := redis.NewClient(opt)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, closeWith(client, fmt.Errorf("ping redis: %w", err))
	}
	return &Cache{client: client, prefix: normalizePrefix(prefix)}, nil
}

func (c *Cache) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("close redis: %w", err)
	}
	return nil
}

func normalizePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return DefaultPrefix
	}
	return prefix
}

func closeWith(client *redis.Client, err error) error {
	if cerr := client.Close(); cerr != nil {
		return fmt.Errorf("%w: close: %w", err, cerr)
	}
	return err
}
