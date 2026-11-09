package fx

import (
	"context"
	"fmt"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Config struct {
	FreshTTL    string `env:"FRESH_TTL" envDefault:"1h"`
	CacheTTL    string `env:"CACHE_TTL" envDefault:"24h"`
	HTTPTimeout string `env:"HTTP_TIMEOUT" envDefault:"3s"`
}

func (c *Config) ValidateWithContext(ctx context.Context) error {
	c.FreshTTL = strings.TrimSpace(c.FreshTTL)
	c.CacheTTL = strings.TrimSpace(c.CacheTTL)
	c.HTTPTimeout = strings.TrimSpace(c.HTTPTimeout)
	if err := validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.FreshTTL, validation.By(parsePositiveDuration("FX_FRESH_TTL"))),
		validation.Field(&c.CacheTTL, validation.By(parsePositiveDuration("FX_CACHE_TTL"))),
		validation.Field(&c.HTTPTimeout, validation.By(parsePositiveDuration("FX_HTTP_TIMEOUT"))),
	); err != nil {
		return err
	}

	fresh, cache, _, err := c.durations()
	if err != nil {
		return err
	}
	if cache < fresh {
		return fmt.Errorf("FX_CACHE_TTL must be >= FX_FRESH_TTL")
	}
	return nil
}

func (c Config) durations() (fresh, cache, timeout time.Duration, err error) {
	fresh, err = time.ParseDuration(strings.TrimSpace(c.FreshTTL))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("FX_FRESH_TTL: %w", err)
	}
	cache, err = time.ParseDuration(strings.TrimSpace(c.CacheTTL))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("FX_CACHE_TTL: %w", err)
	}
	timeout, err = time.ParseDuration(strings.TrimSpace(c.HTTPTimeout))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("FX_HTTP_TIMEOUT: %w", err)
	}
	return fresh, cache, timeout, nil
}

func parsePositiveDuration(name string) validation.RuleFunc {
	return func(value any) error {
		raw, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s is invalid", name)
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return fmt.Errorf("%s is required", name)
		}
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if parsed <= 0 {
			return fmt.Errorf("%s must be > 0", name)
		}
		return nil
	}
}
