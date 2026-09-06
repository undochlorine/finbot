package telegram

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/text"
)

func registerHandlers(inner *bot.Bot) {
	inner.RegisterHandlerMatchFunc(commandAtStart("start"), handleStart)
	inner.RegisterHandlerMatchFunc(commandAtStart("help"), handleHelp)
}

func commandAtStart(name string) bot.MatchFunc {
	return func(update *models.Update) bool {
		if update == nil || update.Message == nil {
			return false
		}
		msg := update.Message.Text
		if !strings.HasPrefix(msg, "/") {
			return false
		}
		cmd, _, _ := strings.Cut(msg[1:], " ")
		cmd, _, _ = strings.Cut(cmd, "@")
		return strings.EqualFold(cmd, name)
	}
}

func handleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	reply(ctx, b, update, text.Start)
}

func handleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	reply(ctx, b, update, text.Help)
}

func reply(ctx context.Context, b *bot.Bot, update *models.Update, message string) {
	if b == nil || update == nil || update.Message == nil {
		return
	}
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   message,
	}); err != nil {
		slog.Error("send telegram message", slog.Any("err", err))
	}
}
