package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Config struct {
	Trial   TrialConfig   `envPrefix:"TRIAL_"`
	Default DefaultConfig `envPrefix:"DEFAULT_"`
}

type TrialConfig struct {
	Duration string `env:"DURATION" envDefault:"168h"`
}

type DefaultConfig struct {
	Currency string `env:"CURRENCY" envDefault:"USD"`
}

func (c *Config) ValidateWithContext(ctx context.Context) error {
	if err := c.Trial.ValidateWithContext(ctx); err != nil {
		return err
	}
	return c.Default.ValidateWithContext(ctx)
}

func (c *TrialConfig) ValidateWithContext(ctx context.Context) error {
	c.Duration = strings.TrimSpace(c.Duration)
	return validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.Duration, validation.By(parseNonNegativeDuration("TRIAL_DURATION"))),
	)
}

func (c *DefaultConfig) ValidateWithContext(ctx context.Context) error {
	c.Currency = strings.TrimSpace(c.Currency)
	if c.Currency == "" {
		c.Currency = "USD"
	}
	return validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.Currency, validation.Required),
	)
}

func (c TrialConfig) Parsed() (time.Duration, error) {
	d, err := time.ParseDuration(strings.TrimSpace(c.Duration))
	if err != nil {
		return 0, fmt.Errorf("TRIAL_DURATION: %w", err)
	}
	return d, nil
}

func (c DefaultConfig) Parsed() string {
	cur := strings.TrimSpace(c.Currency)
	if cur == "" {
		return "USD"
	}
	return cur
}

func parseNonNegativeDuration(name string) validation.RuleFunc {
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
		if parsed < 0 {
			return fmt.Errorf("%s must be >= 0", name)
		}
		return nil
	}
}
