package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
)

const pollTimeout = time.Minute

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type Service interface {
	UpsertUser(ctx context.Context, userID domain.UserID, username string) (domain.User, error)
	GetByName(ctx context.Context, userID domain.UserID, name string) (domain.Bank, error)
	Get(ctx context.Context, userID domain.UserID, bankID int64) (domain.Bank, error)
	List(ctx context.Context, userID domain.UserID) ([]domain.Bank, error)
	CreateBank(ctx context.Context, userID domain.UserID, name string, includeInTotal bool) (domain.Bank, error)
	Add(ctx context.Context, userID domain.UserID, bankID int64, amount domain.Money) (domain.Bank, error)
	Spend(ctx context.Context, userID domain.UserID, bankID int64, amount domain.Money) (domain.Bank, error)
	Set(ctx context.Context, userID domain.UserID, bankID int64, amount domain.Money) (domain.Bank, error)
}

var _ HTTPClient = (*http.Client)(nil)

type Bot struct {
	inner *bot.Bot
	svc   Service
	cache Cache
}

func New(token string, svc Service, cache Cache, client HTTPClient, opts ...bot.Option) (*Bot, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}
	if svc == nil {
		return nil, fmt.Errorf("service is required")
	}
	if cache == nil {
		return nil, fmt.Errorf("cache is required")
	}
	if client == nil {
		return nil, fmt.Errorf("http client is required")
	}

	h := &Bot{svc: svc, cache: cache}
	defaults := []bot.Option{
		bot.WithErrorsHandler(func(err error) {
			slog.Error("telegram", slog.Any("err", err))
		}),
		bot.WithHTTPClient(pollTimeout, client),
		bot.WithMiddlewares(activityMiddleware(svc)),
		bot.WithDefaultHandler(h.handlePendingInput),
	}
	inner, err := bot.New(token, append(defaults, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("init telegram bot: %w", err)
	}
	h.inner = inner
	h.registerHandlers()
	return h, nil
}

func (b *Bot) Start(ctx context.Context) {
	slog.Info("telegram long polling started")
	b.inner.Start(ctx)
	slog.Info("telegram long polling stopped")
}

func (b *Bot) ProcessUpdate(ctx context.Context, update *models.Update) {
	b.inner.ProcessUpdate(ctx, update)
}
