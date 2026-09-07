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

func parseIncludeCallback(data string) (bool, bool) {
	switch data {
	case callbackNewBankIncludeYes:
		return true, true
	case callbackNewBankIncludeNo:
		return false, true
	default:
		return false, false
	}
}

func includeKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: text.Yes, CallbackData: callbackNewBankIncludeYes},
			{Text: text.No, CallbackData: callbackNewBankIncludeNo},
		}},
	}
}

func (h *Bot) handleNewBank(ctx context.Context, b *bot.Bot, update *models.Update) {
	from := sender(update)
	if from == nil {
		return
	}
	h.progressNewBank(ctx, b, messageChatID(update), domain.UserID(from.ID), commandPayload(update.Message.Text), false)
}

func (h *Bot) continueNewBank(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	switch st.Step {
	case stepName:
		h.progressNewBank(ctx, b, chatID, userID, raw, true)
	case stepInclude:
		include, parsed := domain.ParseYesNo(strings.TrimSpace(raw))
		if !parsed {
			reply(ctx, b, chatID, text.NewBankAskInclude, includeKeyboard())
			return
		}
		h.createBankAndReply(ctx, b, chatID, userID, st.Name, include)
	}
}

func (h *Bot) handleNewBankCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.beginCallback(ctx, b, update) {
		return
	}
	include, ok := parseIncludeCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	st, ok := h.callbackFSM(ctx, b, update, domain.CommandNewBank, stepInclude)
	if !ok || st.Name == "" {
		return
	}
	h.createBankAndReply(ctx, b, callbackChatID(update), domain.UserID(update.CallbackQuery.From.ID), st.Name, include)
}

func (h *Bot) progressNewBank(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	name string,
	askedName bool,
) {
	name = strings.TrimSpace(name)
	if name == "" {
		h.saveFSM(ctx, userID, fsmState{Flow: domain.CommandNewBank, Step: stepName})
		msg := text.NewBankAskName
		if askedName {
			msg = text.InvalidBankName
		}
		reply(ctx, b, chatID, msg, nil)
		return
	}
	if _, err := domain.NormalizeBankName(name); err != nil {
		h.saveFSM(ctx, userID, fsmState{Flow: domain.CommandNewBank, Step: stepName})
		reply(ctx, b, chatID, text.InvalidBankName, nil)
		return
	}
	existing, err := h.svc.GetByName(ctx, userID, name)
	if err == nil {
		h.saveFSM(ctx, userID, fsmState{Flow: domain.CommandNewBank, Step: stepName})
		reply(ctx, b, chatID, text.BankNameTaken(existing.Name), nil)
		return
	}
	if !errors.Is(err, domain.ErrBankNotFound) {
		replyErr(ctx, b, chatID, "check bank name", err)
		return
	}
	h.saveFSM(ctx, userID, fsmState{Flow: domain.CommandNewBank, Step: stepInclude, Name: name})
	reply(ctx, b, chatID, text.NewBankAskInclude, includeKeyboard())
}

func (h *Bot) createBankAndReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	name string,
	include bool,
) {
	bank, err := h.svc.CreateBank(ctx, userID, name, include)
	h.clearFSM(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBankNameTaken):
			reply(ctx, b, chatID, text.BankNameTaken(name), nil)
		case errors.Is(err, domain.ErrInvalidBankName):
			reply(ctx, b, chatID, text.InvalidBankName, nil)
		default:
			replyErr(ctx, b, chatID, "create bank", err)
		}
		return
	}
	reply(ctx, b, chatID, text.BankCreated(bank.Name, bank.IncludeInTotal), nil)
}
