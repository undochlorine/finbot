package telegram

import (
	"context"
	"errors"
	"log/slog"
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
	chatID := messageChatID(update)
	userID := domain.UserID(from.ID)
	h.replacePending(ctx, b, chatID, userID)
	h.progressToggle(ctx, b, chatID, userID, fsmState{}, commandPayload(update.Message.Text))
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
	h.progressToggle(ctx, b, chatID, userID, st, raw)
}

func (h *Bot) progressToggle(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	payload string,
) {
	name := strings.TrimSpace(payload)
	if name == "" {
		h.offerBanks(ctx, b, chatID, userID, st, domain.CommandToggle)
		return
	}
	bank, ok := h.bankByName(ctx, b, chatID, userID, st, name)
	if !ok {
		return
	}
	st.Name = bank.Name
	st.BankID = bank.ID
	h.toggleAndReply(ctx, b, chatID, userID, st)
}

func (h *Bot) toggleAndReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
) {
	bank, err := h.svc.Toggle(ctx, userID, st.BankID)
	if errors.Is(err, domain.ErrBankNotFound) {
		msg := text.SomethingWentWrong
		if st.Name != "" {
			msg = text.UnknownBank(st.Name)
		}
		h.done(ctx, b, chatID, userID, st, msg)
		return
	}
	if err != nil {
		slog.Error("toggle bank", slog.Any("err", err))
		h.done(ctx, b, chatID, userID, st, text.SomethingWentWrong)
		return
	}
	h.done(ctx, b, chatID, userID, st, text.Toggled(bank.Name, bank.Balance.Format(), bank.IncludeInTotal))
}

func (h *Bot) handleToggleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.beginCallback(ctx, b, update) {
		return
	}
	bankID, ok := parsePrefixedID(update.CallbackQuery.Data, callbackTogglePrefix)
	if !ok {
		return
	}
	st, ok := h.callbackFSM(ctx, b, update, domain.CommandToggle, stepBank)
	if !ok {
		return
	}
	st.BankID = bankID
	h.toggleAndReply(ctx, b, callbackChatID(update), domain.UserID(update.CallbackQuery.From.ID), st)
}
