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

const deleteActionPick = "pick"

func (h *Bot) handleDelete(ctx context.Context, b *bot.Bot, update *models.Update) {
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	h.progressDelete(ctx, b, messageChatID(update), domain.UserID(from.ID), commandPayload(update.Message.Text))
}

func (h *Bot) continueDelete(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	switch st.Step {
	case stepBank:
		h.progressDelete(ctx, b, chatID, userID, raw)
	case stepConfirm:
		yes, parsed := domain.ParseYesNo(strings.TrimSpace(raw))
		if !parsed {
			reply(ctx, b, chatID, text.AskDeleteConfirm(st.Name), deleteConfirmKeyboard(st.BankID))
			return
		}
		if yes {
			h.deleteBankAndReply(ctx, b, chatID, userID, st.Name, st.BankID)
			return
		}
		h.cancelDelete(ctx, b, chatID, userID)
	}
}

func (h *Bot) progressDelete(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	payload string,
) {
	name := strings.TrimSpace(payload)
	if name == "" {
		h.offerBanks(ctx, b, chatID, userID, domain.CommandDelete)
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
	h.askDeleteConfirm(ctx, b, chatID, userID, bank)
}

func (h *Bot) askDeleteConfirm(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	bank domain.Bank,
) {
	h.saveFSM(ctx, userID, confirmState(bank))
	reply(ctx, b, chatID, text.AskDeleteConfirm(bank.Name), deleteConfirmKeyboard(bank.ID))
}

func (h *Bot) deleteBankAndReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	name string,
	bankID int64,
) {
	err := h.svc.Delete(ctx, userID, bankID)
	h.clearFSM(ctx, userID)
	if errors.Is(err, domain.ErrBankNotFound) {
		reply(ctx, b, chatID, text.UnknownBank(name), nil)
		return
	}
	if err != nil {
		slog.Error("delete bank", slog.Any("err", err))
		reply(ctx, b, chatID, text.SomethingWentWrong, nil)
		return
	}
	reply(ctx, b, chatID, text.BankDeleted(name), nil)
}

func (h *Bot) cancelDelete(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID) {
	h.clearFSM(ctx, userID)
	reply(ctx, b, chatID, text.DeleteCancelled, nil)
}

func (h *Bot) handleDeleteCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.CallbackQuery == nil {
		return
	}
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	}); err != nil {
		slog.Error("answer callback query", slog.Any("err", err))
	}
	action, bankID, ok := parseDeleteCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	st, ok := h.loadFSM(ctx, userID)
	if !ok || st.Flow != domain.CommandDelete {
		return
	}
	chatID := callbackChatID(update)
	switch action {
	case deleteActionPick:
		if st.Step != stepBank {
			return
		}
		bank, err := h.svc.Get(ctx, userID, bankID)
		if err != nil {
			slog.Error("get bank", slog.Any("err", err))
			reply(ctx, b, chatID, text.SomethingWentWrong, nil)
			return
		}
		h.askDeleteConfirm(ctx, b, chatID, userID, bank)
	case domain.Yes:
		if st.Step != stepConfirm || st.BankID != bankID {
			return
		}
		h.deleteBankAndReply(ctx, b, chatID, userID, st.Name, bankID)
	case domain.No:
		if st.Step != stepConfirm || st.BankID != bankID {
			return
		}
		h.cancelDelete(ctx, b, chatID, userID)
	}
}

func parseDeleteCallback(data string) (action string, bankID int64, ok bool) {
	for _, p := range []struct {
		prefix string
		action string
	}{
		{callbackDeleteYes, domain.Yes},
		{callbackDeleteNo, domain.No},
		{callbackDeletePrefix, deleteActionPick},
	} {
		rest, found := strings.CutPrefix(data, p.prefix)
		if !found {
			continue
		}
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			return "", 0, false
		}
		return p.action, id, true
	}
	return "", 0, false
}

func deleteConfirmKeyboard(bankID int64) *models.InlineKeyboardMarkup {
	id := strconv.FormatInt(bankID, 10)
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: text.Yes, CallbackData: callbackDeleteYes + id},
			{Text: text.No, CallbackData: callbackDeleteNo + id},
		}},
	}
}

func confirmState(bank domain.Bank) fsmState {
	return fsmState{Flow: domain.CommandDelete, Step: stepConfirm, Name: bank.Name, BankID: bank.ID}
}
