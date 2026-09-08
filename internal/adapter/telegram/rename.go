package telegram

import (
	"context"
	"errors"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
)

func (st fsmState) withRenameName(bank domain.Bank) fsmState {
	st.Flow = domain.CommandRename
	st.Step = stepName
	st.Name = bank.Name
	st.BankID = bank.ID
	return st
}

func renameNameState(bank domain.Bank) fsmState {
	return fsmState{}.withRenameName(bank)
}

func (h *Bot) handleRename(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandRename)
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	chatID := messageChatID(update)
	userID := domain.UserID(from.ID)
	h.replacePending(ctx, b, chatID, userID)
	h.progressRename(ctx, b, chatID, userID, fsmState{}, commandPayload(update.Message.Text), true)
}

func (h *Bot) continueRename(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	switch st.Step {
	case stepBank:
		h.progressRename(ctx, b, chatID, userID, st, raw, false)
	case stepName:
		h.applyRenameName(ctx, b, chatID, userID, st, raw)
	}
}

func (h *Bot) progressRename(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	payload string,
	tryOldNew bool,
) {
	name := strings.TrimSpace(payload)
	if name == "" {
		h.offerBanks(ctx, b, chatID, userID, st, domain.CommandRename)
		return
	}
	bank, err := h.svc.GetByName(ctx, userID, name)
	switch {
	case err == nil:
		h.askRenameName(ctx, b, chatID, userID, st, bank)
	case !errors.Is(err, domain.ErrBankNotFound):
		replyErr(ctx, b, chatID, "get bank by name", err)
	case tryOldNew && h.renameByFirstWord(ctx, b, chatID, userID, st, name):
		return
	default:
		h.unknownBank(ctx, b, chatID, userID, st, name)
	}
}

func (h *Bot) renameByFirstWord(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	payload string,
) bool {
	oldName, newName, ok := splitRenameArgs(payload)
	if !ok {
		return false
	}
	bank, err := h.svc.GetByName(ctx, userID, oldName)
	if err == nil {
		h.applyRenameName(ctx, b, chatID, userID, st.withRenameName(bank), newName)
		return true
	}
	if !errors.Is(err, domain.ErrBankNotFound) {
		replyErr(ctx, b, chatID, "get bank by name", err)
		return true
	}
	return false
}

func splitRenameArgs(payload string) (oldName, newName string, ok bool) {
	oldName, rest, found := strings.Cut(payload, " ")
	newName = strings.TrimSpace(rest)
	return oldName, newName, found && oldName != "" && newName != ""
}

func (h *Bot) askRenameName(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	bank domain.Bank,
) {
	h.prompt(ctx, b, chatID, userID, st.withRenameName(bank), copyFrom(ctx).AskRenameName(bank.Name), nil)
}

func (h *Bot) applyRenameName(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	name := strings.TrimSpace(raw)
	if name == "" {
		h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).AskRenameName(st.Name), nil)
		return
	}
	existing, err := h.svc.GetByName(ctx, userID, name)
	if err == nil && existing.ID != st.BankID {
		h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).BankNameTaken(existing.Name), nil)
		return
	}
	if err != nil && !errors.Is(err, domain.ErrBankNotFound) {
		replyErr(ctx, b, chatID, "check bank name", err)
		return
	}
	h.renameAndReply(ctx, b, chatID, userID, st, name)
}

func (h *Bot) renameAndReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	newName string,
) {
	bank, err := h.svc.Rename(ctx, userID, st.BankID, newName)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBankNameTaken):
			h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).BankNameTaken(newName), nil)
		case errors.Is(err, domain.ErrInvalidBankName):
			h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).AskRenameName(st.Name), nil)
		case errors.Is(err, domain.ErrBankNotFound):
			h.done(ctx, b, chatID, userID, st, copyFrom(ctx).UnknownBank(st.Name))
		default:
			logHandlerErr(userID, domain.CommandRename, "rename bank", err)
			h.done(ctx, b, chatID, userID, st, copyFrom(ctx).SomethingWentWrong)
		}
		return
	}
	h.done(ctx, b, chatID, userID, st, copyFrom(ctx).BankRenamed(st.Name, bank.Name))
}

func (h *Bot) handleRenameCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandRename)
	if !h.beginCallback(ctx, b, update) {
		return
	}
	bankID, ok := parsePrefixedID(update.CallbackQuery.Data, callbackRenamePrefix)
	if !ok {
		return
	}
	st, ok := h.callbackFSM(ctx, b, update, domain.CommandRename, stepBank)
	if !ok {
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	chatID := callbackChatID(update)
	bank, ok := h.bankByID(ctx, b, chatID, userID, bankID)
	if !ok {
		return
	}
	h.askRenameName(ctx, b, chatID, userID, st, bank)
}
