package telegram

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/text"
)

func (h *Bot) registerHandlers() {
	h.inner.RegisterHandlerMatchFunc(commandAtStart("start"), handleStart)
	h.inner.RegisterHandlerMatchFunc(commandAtStart("help"), handleHelp)
	h.inner.RegisterHandlerMatchFunc(commandAtStart("newbank"), h.handleNewBank)
	h.inner.RegisterHandler(
		bot.HandlerTypeCallbackQueryData,
		callbackNewBankPrefix,
		bot.MatchTypePrefix,
		h.handleNewBankCallback,
	)
}

func commandPayload(msg string) string {
	if msg == "" || !strings.HasPrefix(msg, "/") {
		return strings.TrimSpace(msg)
	}
	_, rest, found := strings.Cut(msg, " ")
	if !found {
		return ""
	}
	return strings.TrimSpace(rest)
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
	reply(ctx, b, messageChatID(update), text.Start, nil)
}

func handleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	reply(ctx, b, messageChatID(update), text.Help, nil)
}

func messageChatID(update *models.Update) int64 {
	if update == nil || update.Message == nil {
		return 0
	}
	return update.Message.Chat.ID
}

func reply(ctx context.Context, b *bot.Bot, chatID int64, message string, markup models.ReplyMarkup) {
	if b == nil || chatID == 0 {
		return
	}
	params := &bot.SendMessageParams{ChatID: chatID, Text: message}
	if markup != nil {
		params.ReplyMarkup = markup
	}
	if _, err := b.SendMessage(ctx, params); err != nil {
		slog.Error("send telegram message", slog.Any("err", err))
	}
}
