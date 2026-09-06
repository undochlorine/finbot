package telegram

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
)

func activityMiddleware(svc Service) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			from := sender(update)
			if from == nil || from.IsBot || from.ID == 0 {
				next(ctx, b, update)
				return
			}
			if _, err := svc.UpsertUser(ctx, domain.UserID(from.ID), from.Username); err != nil {
				slog.Error("upsert user", slog.Int64("telegram_id", from.ID), slog.Any("err", err))
				return
			}
			next(ctx, b, update)
		}
	}
}

func sender(update *models.Update) *models.User {
	if update == nil {
		return nil
	}
	switch {
	case update.Message != nil:
		return update.Message.From
	case update.EditedMessage != nil:
		return update.EditedMessage.From
	case update.CallbackQuery != nil:
		return &update.CallbackQuery.From
	default:
		return nil
	}
}
