package telegram

import (
	"context"
	"errors"
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
	bank, ok := h.bankByName(ctx, b, chatID, userID, name)
	if !ok {
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
		replyErr(ctx, b, chatID, "toggle bank", err)
		return
	}
	reply(ctx, b, chatID, text.Toggled(bank.Name, bank.Balance.Format(), bank.IncludeInTotal), nil)
}

func (h *Bot) handleToggleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.beginCallback(ctx, b, update) {
		return
	}
	bankID, ok := parsePrefixedID(update.CallbackQuery.Data, callbackTogglePrefix)
	if !ok {
		return
	}
	if _, ok := h.callbackFSM(ctx, b, update, domain.CommandToggle, stepBank); !ok {
		return
	}
	h.toggleAndReply(ctx, b, callbackChatID(update), domain.UserID(update.CallbackQuery.From.ID), "", bankID)
}
