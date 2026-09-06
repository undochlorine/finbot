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

type Service interface {
	UpsertUser(ctx context.Context, userID domain.UserID, username string) (domain.User, error)
}

var _ HTTPClient = (*http.Client)(nil)

type Bot struct {
	inner *bot.Bot
}

func New(token string, svc Service, client HTTPClient, opts ...bot.Option) (*Bot, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}
	if svc == nil {
		return nil, fmt.Errorf("service is required")
	}
	if client == nil {
		return nil, fmt.Errorf("http client is required")
	}

	defaults := []bot.Option{
		bot.WithErrorsHandler(func(err error) {
			slog.Error("telegram", slog.Any("err", err))
		}),
		bot.WithHTTPClient(pollTimeout, client),
		bot.WithMiddlewares(activityMiddleware(svc)),
	}
	inner, err := bot.New(token, append(defaults, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("init telegram bot: %w", err)
	}
	registerHandlers(inner)
	return &Bot{inner: inner}, nil
}

func (b *Bot) Start(ctx context.Context) {
	slog.Info("telegram long polling started")
	b.inner.Start(ctx)
	slog.Info("telegram long polling stopped")
}

func (b *Bot) ProcessUpdate(ctx context.Context, update *models.Update) {
	b.inner.ProcessUpdate(ctx, update)
}
