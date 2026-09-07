package telegram

import (
	"context"
	"errors"
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
	bank, ok := h.bankByName(ctx, b, chatID, userID, name)
	if !ok {
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
	h.saveFSM(ctx, userID, deleteConfirmState(bank))
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
		replyErr(ctx, b, chatID, "delete bank", err)
		return
	}
	reply(ctx, b, chatID, text.BankDeleted(name), nil)
}

func (h *Bot) cancelDelete(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID) {
	h.clearFSM(ctx, userID)
	reply(ctx, b, chatID, text.DeleteCancelled, nil)
}

func (h *Bot) handleDeleteCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.beginCallback(ctx, b, update) {
		return
	}
	action, bankID, ok := parseDeleteCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	st, ok := h.loadCallbackFSM(ctx, b, update)
	if !ok || st.Flow != domain.CommandDelete {
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	chatID := callbackChatID(update)
	switch action {
	case deleteActionPick:
		if st.Step != stepBank {
			return
		}
		bank, ok := h.bankByID(ctx, b, chatID, userID, bankID)
		if !ok {
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
		id, found := parsePrefixedID(data, p.prefix)
		if !found {
			continue
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

func deleteConfirmState(bank domain.Bank) fsmState {
	return fsmState{Flow: domain.CommandDelete, Step: stepConfirm, Name: bank.Name, BankID: bank.ID}
}
