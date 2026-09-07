package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

func parseYesNo(s string) (bool, bool) {
	switch strings.ToLower(s) {
	case "yes":
		return true, true
	case "no":
		return false, true
	default:
		return false, false
	}
}

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
	if !ok || st.Flow != flowNewBank {
		return
	}
	chatID := messageChatID(update)
	switch st.Step {
	case stepName:
		h.progressNewBank(ctx, b, chatID, userID, update.Message.Text, true)
	case stepInclude:
		include, parsed := parseYesNo(strings.TrimSpace(update.Message.Text))
		if !parsed {
			reply(ctx, b, chatID, text.NewBankAskInclude, includeKeyboard())
			return
		}
		h.createBankAndReply(ctx, b, chatID, userID, st.Name, include)
	}
}

func (h *Bot) handleNewBankCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.CallbackQuery == nil {
		return
	}
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	}); err != nil {
		slog.Error("answer callback query", slog.Any("err", err))
	}
	include, ok := parseIncludeCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	st, ok := h.loadFSM(ctx, userID)
	if !ok || st.Flow != flowNewBank || st.Step != stepInclude || st.Name == "" {
		return
	}
	h.createBankAndReply(ctx, b, callbackChatID(update), userID, st.Name, include)
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
		h.saveFSM(ctx, userID, fsmState{Flow: flowNewBank, Step: stepName})
		msg := text.NewBankAskName
		if askedName {
			msg = text.InvalidBankName
		}
		reply(ctx, b, chatID, msg, nil)
		return
	}
	if _, err := domain.NormalizeBankName(name); err != nil {
		h.saveFSM(ctx, userID, fsmState{Flow: flowNewBank, Step: stepName})
		reply(ctx, b, chatID, text.InvalidBankName, nil)
		return
	}
	existing, err := h.svc.GetByName(ctx, userID, name)
	if err == nil {
		h.saveFSM(ctx, userID, fsmState{Flow: flowNewBank, Step: stepName})
		reply(ctx, b, chatID, text.BankNameTaken(existing.Name), nil)
		return
	}
	if !errors.Is(err, domain.ErrBankNotFound) {
		slog.Error("check bank name", slog.Any("err", err))
		reply(ctx, b, chatID, text.SomethingWentWrong, nil)
		return
	}
	h.saveFSM(ctx, userID, fsmState{Flow: flowNewBank, Step: stepInclude, Name: name})
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
			slog.Error("create bank", slog.Any("err", err))
			reply(ctx, b, chatID, text.SomethingWentWrong, nil)
		}
		return
	}
	reply(ctx, b, chatID, text.BankCreated(bank.Name, bank.IncludeInTotal), nil)
}

func (h *Bot) loadFSM(ctx context.Context, userID domain.UserID) (fsmState, bool) {
	raw, ok, err := h.cache.Get(ctx, fsmKey(userID))
	if err != nil {
		slog.Error("fsm get", slog.Any("err", err))
		return fsmState{}, false
	}
	if !ok {
		return fsmState{}, false
	}
	var st fsmState
	if err := json.Unmarshal(raw, &st); err != nil {
		slog.Error("fsm decode", slog.Any("err", err))
		return fsmState{}, false
	}
	return st, true
}

func (h *Bot) saveFSM(ctx context.Context, userID domain.UserID, st fsmState) {
	raw, err := json.Marshal(st)
	if err != nil {
		slog.Error("fsm encode", slog.Any("err", err))
		return
	}
	if err := h.cache.Set(ctx, fsmKey(userID), raw, fsmTTL); err != nil {
		slog.Error("fsm set", slog.Any("err", err))
	}
}

func (h *Bot) clearFSM(ctx context.Context, userID domain.UserID) {
	if err := h.cache.Delete(ctx, fsmKey(userID)); err != nil {
		slog.Error("fsm delete", slog.Any("err", err))
	}
}

func callbackChatID(update *models.Update) int64 {
	if update == nil || update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return 0
	}
	return update.CallbackQuery.Message.Message.Chat.ID
}
