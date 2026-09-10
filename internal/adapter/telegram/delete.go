package telegram

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
)

const deleteActionPick = "pick"

func (h *Bot) handleDelete(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandDelete)
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
		yes, parsed := parseYesNo(ctx, raw)
		if !parsed {
			h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).AskDeleteConfirm(st.Name), deleteConfirmKeyboard(ctx, st.BankID))
			return
		}
		if yes {
			h.deleteBankAndReply(ctx, b, chatID, userID, st)
			return
		}
		h.done(ctx, b, chatID, userID, st, copyFrom(ctx).DeleteCancelled)
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
		copyFrom(ctx).AskDeleteConfirm(bank.Name),
		deleteConfirmKeyboard(ctx, bank.ID),
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
		h.done(ctx, b, chatID, userID, st, copyFrom(ctx).UnknownBank(st.Name))
		return
	}
	if err != nil {
		logHandlerErr(userID, domain.CommandDelete, "delete bank", err)
		h.done(ctx, b, chatID, userID, st, copyFrom(ctx).SomethingWentWrong)
		return
	}
	h.done(ctx, b, chatID, userID, st, copyFrom(ctx).BankDeleted(st.Name))
}

func (h *Bot) handleDeleteCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandDelete)
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
		h.done(ctx, b, chatID, userID, st, copyFrom(ctx).DeleteCancelled)
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

func deleteConfirmKeyboard(ctx context.Context, bankID int64) *models.InlineKeyboardMarkup {
	id := strconv.FormatInt(bankID, 10)
	c := copyFrom(ctx)
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: c.Yes, CallbackData: callbackDeleteYes + id},
			{Text: c.No, CallbackData: callbackDeleteNo + id},
		}},
	}
}

func deleteConfirmState(bank domain.Bank) fsmState {
	return fsmState{}.withDeleteConfirm(bank)
}
