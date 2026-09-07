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
	chatID := messageChatID(update)
	userID := domain.UserID(from.ID)
	h.replacePending(ctx, b, chatID, userID)
	h.progressDelete(ctx, b, chatID, userID, fsmState{}, commandPayload(update.Message.Text))
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
		h.progressDelete(ctx, b, chatID, userID, st, raw)
	case stepConfirm:
		yes, parsed := domain.ParseYesNo(strings.TrimSpace(raw))
		if !parsed {
			h.prompt(ctx, b, chatID, userID, st, text.AskDeleteConfirm(st.Name), deleteConfirmKeyboard(st.BankID))
			return
		}
		if yes {
			h.deleteBankAndReply(ctx, b, chatID, userID, st)
			return
		}
		h.done(ctx, b, chatID, userID, st, text.DeleteCancelled)
	}
}

func (h *Bot) progressDelete(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	payload string,
) {
	name := strings.TrimSpace(payload)
	if name == "" {
		h.offerBanks(ctx, b, chatID, userID, st, domain.CommandDelete)
		return
	}
	bank, ok := h.bankByName(ctx, b, chatID, userID, st, name)
	if !ok {
		return
	}
	h.askDeleteConfirm(ctx, b, chatID, userID, st, bank)
}

func (h *Bot) askDeleteConfirm(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	bank domain.Bank,
) {
	h.prompt(
		ctx,
		b,
		chatID,
		userID,
		st.withDeleteConfirm(bank),
		text.AskDeleteConfirm(bank.Name),
		deleteConfirmKeyboard(bank.ID),
	)
}

func (h *Bot) deleteBankAndReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
) {
	err := h.svc.Delete(ctx, userID, st.BankID)
	if errors.Is(err, domain.ErrBankNotFound) {
		h.done(ctx, b, chatID, userID, st, text.UnknownBank(st.Name))
		return
	}
	if err != nil {
		slog.Error("delete bank", slog.Any("err", err))
		h.done(ctx, b, chatID, userID, st, text.SomethingWentWrong)
		return
	}
	h.done(ctx, b, chatID, userID, st, text.BankDeleted(st.Name))
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
		if ok {
			h.stripStale(ctx, b, update, st)
		}
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
		h.askDeleteConfirm(ctx, b, chatID, userID, st, bank)
	case domain.Yes:
		if st.Step != stepConfirm || st.BankID != bankID {
			return
		}
		h.deleteBankAndReply(ctx, b, chatID, userID, st)
	case domain.No:
		if st.Step != stepConfirm || st.BankID != bankID {
			return
		}
		h.done(ctx, b, chatID, userID, st, text.DeleteCancelled)
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
	return fsmState{}.withDeleteConfirm(bank)
}
