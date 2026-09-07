package telegram

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

func (h *Bot) registerHandlers() {
	for _, item := range []struct {
		cmd string
		fn  bot.HandlerFunc
	}{
		{commandStart, handleStart},
		{commandHelp, handleHelp},
		{domain.CommandNewBank, h.handleNewBank},
		{domain.CommandAdd, h.handleAdd},
		{domain.CommandSpend, h.handleSpend},
		{domain.CommandSet, h.handleSet},
		{domain.CommandDelete, h.handleDelete},
		{domain.CommandBank, h.handleBank},
		{domain.CommandToggle, h.handleToggle},
		{domain.CommandBanks, h.handleBanks},
		{domain.CommandTotal, h.handleTotal},
		{domain.CommandAll, h.handleAll},
		{commandCancel, h.handleCancel},
		{commandFeedback, h.handleFeedback},
	} {
		h.inner.RegisterHandlerMatchFunc(commandAtStart(item.cmd), item.fn)
	}
	for _, item := range []struct {
		prefix string
		fn     bot.HandlerFunc
	}{
		{callbackNewBankPrefix, h.handleNewBankCallback},
		{callbackAddPrefix, h.handleMoneyCallback},
		{callbackSpendPrefix, h.handleMoneyCallback},
		{callbackSetPrefix, h.handleMoneyCallback},
		{callbackDeletePrefix, h.handleDeleteCallback},
		{callbackBankPrefix, h.handleBankCallback},
		{callbackTogglePrefix, h.handleToggleCallback},
	} {
		h.inner.RegisterHandler(bot.HandlerTypeCallbackQueryData, item.prefix, bot.MatchTypePrefix, item.fn)
	}
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

func (h *Bot) handleCancel(ctx context.Context, b *bot.Bot, update *models.Update) {
	from := sender(update)
	if from == nil {
		return
	}
	userID := domain.UserID(from.ID)
	if _, ok := h.loadFSM(ctx, userID); !ok {
		reply(ctx, b, messageChatID(update), text.NothingToCancel, nil)
		return
	}
	h.clearFSM(ctx, userID)
	reply(ctx, b, messageChatID(update), text.Canceled, nil)
}

func (h *Bot) handlePendingInput(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}
	if strings.HasPrefix(update.Message.Text, "/") {
		return
	}
	from := sender(update)
	if from == nil {
		return
	}
	userID := domain.UserID(from.ID)
	st, ok := h.loadFSM(ctx, userID)
	if !ok {
		return
	}
	chatID := messageChatID(update)
	switch st.Flow {
	case domain.CommandNewBank:
		h.continueNewBank(ctx, b, chatID, userID, st, update.Message.Text)
	case domain.CommandAdd, domain.CommandSpend, domain.CommandSet:
		h.continueMoney(ctx, b, chatID, userID, st, update.Message.Text)
	case domain.CommandDelete:
		h.continueDelete(ctx, b, chatID, userID, st, update.Message.Text)
	case domain.CommandBank:
		h.continueBank(ctx, b, chatID, userID, st, update.Message.Text)
	case domain.CommandToggle:
		h.continueToggle(ctx, b, chatID, userID, st, update.Message.Text)
	case commandFeedback:
		h.progressFeedback(ctx, b, chatID, userID, from.Username, update.Message.Text)
	}
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

func replyErr(ctx context.Context, b *bot.Bot, chatID int64, op string, err error) {
	slog.Error(op, slog.Any("err", err))
	reply(ctx, b, chatID, text.SomethingWentWrong, nil)
}
