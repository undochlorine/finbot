package telegram

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

func (h *Bot) handleToggle(ctx context.Context, b *bot.Bot, update *models.Update) {
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	h.progressToggle(ctx, b, messageChatID(update), domain.UserID(from.ID), commandPayload(update.Message.Text))
}

func (h *Bot) continueToggle(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	if st.Step != stepBank {
		return
	}
	h.progressToggle(ctx, b, chatID, userID, raw)
}

func (h *Bot) progressToggle(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	payload string,
) {
	name := strings.TrimSpace(payload)
	if name == "" {
		h.offerBanks(ctx, b, chatID, userID, domain.CommandToggle)
		return
	}
	bank, err := h.svc.GetByName(ctx, userID, name)
	if errors.Is(err, domain.ErrBankNotFound) {
		reply(ctx, b, chatID, text.UnknownBank(name), nil)
		return
	}
	if err != nil {
		slog.Error("get bank by name", slog.Any("err", err))
		reply(ctx, b, chatID, text.SomethingWentWrong, nil)
		return
	}
	h.toggleAndReply(ctx, b, chatID, userID, bank.Name, bank.ID)
}

func (h *Bot) toggleAndReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	name string,
	bankID int64,
) {
	bank, err := h.svc.Toggle(ctx, userID, bankID)
	h.clearFSM(ctx, userID)
	if errors.Is(err, domain.ErrBankNotFound) {
		msg := text.SomethingWentWrong
		if name != "" {
			msg = text.UnknownBank(name)
		}
		reply(ctx, b, chatID, msg, nil)
		return
	}
	if err != nil {
		slog.Error("toggle bank", slog.Any("err", err))
		reply(ctx, b, chatID, text.SomethingWentWrong, nil)
		return
	}
	reply(ctx, b, chatID, text.Toggled(bank.Name, bank.Balance.Format(), bank.IncludeInTotal), nil)
}

func (h *Bot) handleToggleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.CallbackQuery == nil {
		return
	}
	h.answerCallback(ctx, b, update)
	bankID, ok := parseToggleCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	st, ok := h.loadCallbackFSM(ctx, b, update)
	if !ok || st.Flow != domain.CommandToggle || st.Step != stepBank {
		return
	}
	h.toggleAndReply(ctx, b, callbackChatID(update), domain.UserID(update.CallbackQuery.From.ID), "", bankID)
}

func parseToggleCallback(data string) (int64, bool) {
	rest, found := strings.CutPrefix(data, callbackTogglePrefix)
	if !found {
		return 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
