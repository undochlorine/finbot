package telegram

import (
	"context"
	"errors"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
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

func includeKeyboard(ctx context.Context) *models.InlineKeyboardMarkup {
	c := copyFrom(ctx)
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: c.Yes, CallbackData: callbackNewBankIncludeYes},
			{Text: c.No, CallbackData: callbackNewBankIncludeNo},
		}},
	}
}

func (h *Bot) handleNewBank(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandNewBank)
	from := sender(update)
	if from == nil {
		return
	}
	chatID := messageChatID(update)
	userID := domain.UserID(from.ID)
	h.replacePending(ctx, b, chatID, userID)
	h.progressNewBank(ctx, b, chatID, userID, fsmState{}, commandPayload(update.Message.Text), false)
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
		h.progressNewBank(ctx, b, chatID, userID, st, raw, true)
	case stepInclude:
		include, parsed := parseYesNo(ctx, raw)
		if !parsed {
			h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).NewBankAskInclude, includeKeyboard(ctx))
			return
		}
		h.createBankAndReply(ctx, b, chatID, userID, st, include)
	}
}

func (h *Bot) handleNewBankCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandNewBank)
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
	h.createBankAndReply(ctx, b, callbackChatID(update), domain.UserID(update.CallbackQuery.From.ID), st, include)
}

func (h *Bot) progressNewBank(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	name string,
	askedName bool,
) {
	name = strings.TrimSpace(name)
	st.Flow = domain.CommandNewBank
	if name == "" {
		st.Step = stepName
		msg := copyFrom(ctx).NewBankAskName
		if askedName {
			msg = copyFrom(ctx).InvalidBankName
		}
		h.prompt(ctx, b, chatID, userID, st, msg, nil)
		return
	}
	if _, err := domain.NormalizeBankName(name); err != nil {
		st.Step = stepName
		h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).InvalidBankName, nil)
		return
	}
	existing, err := h.svc.GetByName(ctx, userID, name)
	if err == nil {
		st.Step = stepName
		h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).BankNameTaken(existing.Name), nil)
		return
	}
	if !errors.Is(err, domain.ErrBankNotFound) {
		replyErr(ctx, b, chatID, "check bank name", err)
		return
	}
	st.Step = stepInclude
	st.Name = name
	h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).NewBankAskInclude, includeKeyboard(ctx))
}

func (h *Bot) createBankAndReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	include bool,
) {
	bank, err := h.svc.CreateBank(ctx, userID, st.Name, include)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBankNameTaken):
			h.done(ctx, b, chatID, userID, st, copyFrom(ctx).BankNameTaken(st.Name))
		case errors.Is(err, domain.ErrInvalidBankName):
			h.done(ctx, b, chatID, userID, st, copyFrom(ctx).InvalidBankName)
		default:
			logHandlerErr(userID, domain.CommandNewBank, "create bank", err)
			h.done(ctx, b, chatID, userID, st, copyFrom(ctx).SomethingWentWrong)
		}
		return
	}
	h.done(ctx, b, chatID, userID, st, copyFrom(ctx).BankCreated(bank.Name, bank.IncludeInTotal))
}
