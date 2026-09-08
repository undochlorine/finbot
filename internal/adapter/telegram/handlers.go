package telegram

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
)

func (h *Bot) registerHandlers() {
	for _, item := range []struct {
		cmd string
		fn  bot.HandlerFunc
	}{
		{commandStart, h.handleStart},
		{commandHelp, h.handleHelp},
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

func (h *Bot) handleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, commandStart)
	from := sender(update)
	if from != nil && update != nil && update.Message != nil {
		if ref, ok := parseReferral(commandPayload(update.Message.Text), from.ID); ok {
			if err := h.svc.SetReferredByIfEmpty(ctx, domain.UserID(from.ID), domain.UserID(ref)); err != nil {
				logHandlerErr(domain.UserID(from.ID), commandStart, "set referred by", err)
			}
		}
	}
	reply(ctx, b, messageChatID(update), copyFrom(ctx).Start, nil)
}

func (h *Bot) handleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, commandHelp)
	reply(ctx, b, messageChatID(update), copyFrom(ctx).Help, nil)
}

func (h *Bot) handleCancel(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, commandCancel)
	from := sender(update)
	if from == nil {
		return
	}
	userID := domain.UserID(from.ID)
	st, ok := h.loadFSM(ctx, userID)
	if !ok {
		reply(ctx, b, messageChatID(update), copyFrom(ctx).NothingToCancel, nil)
		return
	}
	chatID := messageChatID(update)
	h.drop(ctx, b, chatID, st)
	h.clearFSM(ctx, userID)
	reply(ctx, b, chatID, copyFrom(ctx).Canceled, nil)
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
	ctx = withCommand(ctx, st.Flow)
	chatID := messageChatID(update)
	msgID := userMessageID(update)
	switch st.Flow {
	case domain.CommandNewBank:
		st.note(msgID)
		h.continueNewBank(ctx, b, chatID, userID, st, update.Message.Text)
	case domain.CommandAdd, domain.CommandSpend, domain.CommandSet:
		st.note(msgID)
		h.continueMoney(ctx, b, chatID, userID, st, update.Message.Text)
	case domain.CommandDelete:
		st.note(msgID)
		h.continueDelete(ctx, b, chatID, userID, st, update.Message.Text)
	case domain.CommandBank:
		st.note(msgID)
		h.continueBank(ctx, b, chatID, userID, st, update.Message.Text)
	case domain.CommandToggle:
		st.note(msgID)
		h.continueToggle(ctx, b, chatID, userID, st, update.Message.Text)
	case commandFeedback:
		h.progressFeedback(ctx, b, chatID, userID, st, from.Username, update.Message.Text)
	}
}

func messageChatID(update *models.Update) int64 {
	if update == nil || update.Message == nil {
		return 0
	}
	return update.Message.Chat.ID
}

func reply(ctx context.Context, b *bot.Bot, chatID int64, message string, markup models.ReplyMarkup) {
	sendMessage(ctx, b, chatID, message, markup)
}

func replyErr(ctx context.Context, b *bot.Bot, chatID int64, op string, err error) {
	slog.Error(op, logAttrs(ctx, slog.Any("err", err))...)
	reply(ctx, b, chatID, copyFrom(ctx).SomethingWentWrong, nil)
}

func parseReferral(payload string, selfID int64) (int64, bool) {
	if payload == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(payload, 10, 64)
	if err != nil || id <= 0 || id == selfID {
		return 0, false
	}
	return id, true
}
