package postgres

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Config struct {
	URL string `env:"URL"`
}

func (c *Config) ValidateWithContext(ctx context.Context) error {
	c.URL = strings.TrimSpace(c.URL)
	return validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.URL, validation.By(requireURL("DATABASE_URL", "postgres", "postgresql"))),
	)
}

func requireURL(name string, schemes ...string) validation.RuleFunc {
	return func(value any) error {
		raw, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s is invalid", name)
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return fmt.Errorf("%s is required", name)
		}
		u, err := url.Parse(raw)
		if err != nil {
			return fmt.Errorf("%s is invalid: %w", name, err)
		}
		matched := false
		for _, scheme := range schemes {
			if u.Scheme == scheme {
				matched = true
				break
			}
		}
		if !matched || u.Host == "" {
			return fmt.Errorf("%s is invalid", name)
		}
		return nil
	}
}
