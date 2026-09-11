package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Config struct {
	Token string `env:"TOKEN"`
	TTL   string `env:"TTL" envDefault:"1h"`
}

type AdminConfig struct {
	ID int64 `env:"ID"`
}

func (c *Config) ValidateWithContext(ctx context.Context) error {
	c.Token = strings.TrimSpace(c.Token)
	c.TTL = strings.TrimSpace(c.TTL)
	return validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.Token, validation.Required.Error("BOT_TOKEN is required")),
		validation.Field(&c.TTL, validation.By(parsePositiveDuration("BOT_TTL"))),
	)
}

func (c *AdminConfig) ValidateWithContext(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.ID, validation.Min(int64(0)).Error("ADMIN_TELEGRAM_ID must be >= 0")),
	)
}

func (c Config) SessionTTL() (time.Duration, error) {
	d, err := time.ParseDuration(strings.TrimSpace(c.TTL))
	if err != nil {
		return 0, fmt.Errorf("BOT_TTL: %w", err)
	}
	return d, nil
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
