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
		include, parsed := domain.ParseYesNo(strings.TrimSpace(raw))
		if !parsed {
			h.prompt(ctx, b, chatID, userID, st, text.NewBankAskInclude, includeKeyboard())
			return
		}
		h.createBankAndReply(ctx, b, chatID, userID, st, include)
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
		msg := text.NewBankAskName
		if askedName {
			msg = text.InvalidBankName
		}
		h.prompt(ctx, b, chatID, userID, st, msg, nil)
		return
	}
	if _, err := domain.NormalizeBankName(name); err != nil {
		st.Step = stepName
		h.prompt(ctx, b, chatID, userID, st, text.InvalidBankName, nil)
		return
	}
	existing, err := h.svc.GetByName(ctx, userID, name)
	if err == nil {
		st.Step = stepName
		h.prompt(ctx, b, chatID, userID, st, text.BankNameTaken(existing.Name), nil)
		return
	}
	if !errors.Is(err, domain.ErrBankNotFound) {
		replyErr(ctx, b, chatID, "check bank name", err)
		return
	}
	st.Step = stepInclude
	st.Name = name
	h.prompt(ctx, b, chatID, userID, st, text.NewBankAskInclude, includeKeyboard())
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
			h.done(ctx, b, chatID, userID, st, text.BankNameTaken(st.Name))
		case errors.Is(err, domain.ErrInvalidBankName):
			h.done(ctx, b, chatID, userID, st, text.InvalidBankName)
		default:
			slog.Error("create bank", slog.Any("err", err))
			h.done(ctx, b, chatID, userID, st, text.SomethingWentWrong)
		}
		return
	}
	h.done(ctx, b, chatID, userID, st, text.BankCreated(bank.Name, bank.IncludeInTotal))
}
