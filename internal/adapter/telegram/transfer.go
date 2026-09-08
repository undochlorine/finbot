package telegram

import (
	"context"
	"errors"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
)

func (st fsmState) withTransferFrom(bank domain.Bank) fsmState {
	st.Flow = domain.CommandTransfer
	st.Step = stepTo
	st.Name = bank.Name
	st.BankID = bank.ID
	st.ToName = ""
	st.ToBankID = 0
	return st
}

func (st fsmState) withTransferTo(bank domain.Bank) fsmState {
	st.Flow = domain.CommandTransfer
	st.Step = stepAmount
	st.ToName = bank.Name
	st.ToBankID = bank.ID
	return st
}

func parseTransferTriple(payload string) (from, to, amount string, ok bool) {
	const parts = 3
	fields := strings.Fields(payload)
	if len(fields) != parts {
		return "", "", "", false
	}
	if _, err := domain.ParseMoney(fields[2]); err != nil {
		return "", "", "", false
	}
	return fields[0], fields[1], fields[2], true
}

func otherBanks(banks []domain.Bank, fromID int64) []domain.Bank {
	out := make([]domain.Bank, 0, len(banks))
	for _, bank := range banks {
		if bank.ID != fromID {
			out = append(out, bank)
		}
	}
	return out
}

func (h *Bot) handleTransfer(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandTransfer)
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	chatID := messageChatID(update)
	userID := domain.UserID(from.ID)
	h.replacePending(ctx, b, chatID, userID)
	if !h.hasTransferBanks(ctx, b, chatID, userID) {
		return
	}
	h.progressTransfer(ctx, b, chatID, userID, fsmState{}, commandPayload(update.Message.Text))
}

func (h *Bot) continueTransfer(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	switch st.Step {
	case stepBank:
		h.progressTransfer(ctx, b, chatID, userID, st, raw)
	case stepTo:
		h.applyTransferToName(ctx, b, chatID, userID, st, raw)
	case stepAmount:
		h.applyTransfer(ctx, b, chatID, userID, st, raw)
	}
}

func (h *Bot) hasTransferBanks(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID) bool {
	banks, err := h.svc.List(ctx, userID)
	if err != nil {
		replyErr(ctx, b, chatID, "list banks", err)
		return false
	}
	switch len(banks) {
	case 0:
		reply(ctx, b, chatID, copyFrom(ctx).NoBanks, nil)
		return false
	case 1:
		reply(ctx, b, chatID, copyFrom(ctx).NeedTwoBanks, nil)
		return false
	default:
		return true
	}
}

func (h *Bot) progressTransfer(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	payload string,
) {
	if h.tryTransferTriple(ctx, b, chatID, userID, st, payload) {
		return
	}
	name := strings.TrimSpace(payload)
	if name == "" {
		h.offerTransferFrom(ctx, b, chatID, userID, st)
		return
	}
	bank, ok := h.bankByName(ctx, b, chatID, userID, st, name)
	if !ok {
		return
	}
	h.askTransferTo(ctx, b, chatID, userID, st, bank)
}

func (h *Bot) tryTransferTriple(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	payload string,
) bool {
	fromName, toName, amountRaw, ok := parseTransferTriple(payload)
	if !ok {
		return false
	}
	fromBank, err := h.svc.GetByName(ctx, userID, fromName)
	if err != nil && !errors.Is(err, domain.ErrBankNotFound) {
		replyErr(ctx, b, chatID, "get bank by name", err)
		return true
	}
	if errors.Is(err, domain.ErrBankNotFound) {
		return false
	}
	toBank, err := h.svc.GetByName(ctx, userID, toName)
	if err != nil && !errors.Is(err, domain.ErrBankNotFound) {
		replyErr(ctx, b, chatID, "get bank by name", err)
		return true
	}
	if errors.Is(err, domain.ErrBankNotFound) {
		return false
	}
	h.applyTransfer(ctx, b, chatID, userID, st.withTransferFrom(fromBank).withTransferTo(toBank), amountRaw)
	return true
}

func (h *Bot) offerTransferFrom(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
) {
	banks, ok := h.loadBanks(ctx, b, chatID, userID)
	if !ok {
		return
	}
	st.Flow = domain.CommandTransfer
	st.Step = stepBank
	st.Name = ""
	st.BankID = 0
	st.ToName = ""
	st.ToBankID = 0
	h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).AskTransferFrom, idKeyboard(callbackTransferFromPrefix, banks))
}

func (h *Bot) askTransferTo(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	from domain.Bank,
) {
	others, ok := h.transferToBanks(ctx, b, chatID, userID, from.ID, st)
	if !ok {
		return
	}
	h.prompt(
		ctx, b, chatID, userID, st.withTransferFrom(from),
		copyFrom(ctx).AskTransferTo(from.Name),
		idKeyboard(callbackTransferToPrefix, others),
	)
}

func (h *Bot) transferToBanks(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	fromID int64,
	st fsmState,
) ([]domain.Bank, bool) {
	banks, err := h.svc.List(ctx, userID)
	if err != nil {
		replyErr(ctx, b, chatID, "list banks", err)
		return nil, false
	}
	others := otherBanks(banks, fromID)
	if len(others) == 0 {
		h.finishOrReply(ctx, b, chatID, userID, st, copyFrom(ctx).NeedTwoBanks)
		return nil, false
	}
	return others, true
}

func (h *Bot) applyTransferToName(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	name := strings.TrimSpace(raw)
	if name == "" {
		h.askTransferTo(ctx, b, chatID, userID, st, domain.Bank{ID: st.BankID, Name: st.Name})
		return
	}
	bank, err := h.svc.GetByName(ctx, userID, name)
	if errors.Is(err, domain.ErrBankNotFound) {
		h.rejectTo(ctx, b, chatID, userID, st, copyFrom(ctx).UnknownBank(name))
		return
	}
	if err != nil {
		replyErr(ctx, b, chatID, "get bank by name", err)
		return
	}
	if bank.ID == st.BankID {
		h.rejectTo(ctx, b, chatID, userID, st, copyFrom(ctx).SameBank)
		return
	}
	h.askTransferAmount(ctx, b, chatID, userID, st, bank)
}

func (h *Bot) askTransferAmount(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	to domain.Bank,
) {
	st = st.withTransferTo(to)
	h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).AskTransferAmount(st.Name, to.Name), nil)
}

func (h *Bot) applyTransfer(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	amountRaw string,
) {
	amount, err := domain.ParseMoney(amountRaw)
	if err != nil {
		h.transferFailed(ctx, b, chatID, userID, st, domain.ErrInvalidAmount)
		return
	}
	from, to, err := h.svc.Transfer(ctx, userID, st.BankID, st.ToBankID, amount)
	if err != nil {
		h.transferFailed(ctx, b, chatID, userID, st, err)
		return
	}
	h.done(ctx, b, chatID, userID, st, copyFrom(ctx).Transferred(
		from.Name, to.Name, amount.Format(), from.Balance.Format(), to.Balance.Format(),
	))
}

func (h *Bot) transferFailed(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	err error,
) {
	c := copyFrom(ctx)
	switch {
	case errors.Is(err, domain.ErrInvalidAmount):
		if st.PromptID != 0 && st.Step == stepAmount {
			h.prompt(ctx, b, chatID, userID, st, c.InvalidAmount, nil)
			return
		}
		h.finishOrReply(ctx, b, chatID, userID, st, c.InvalidAmount)
	case errors.Is(err, domain.ErrSameBank):
		h.rejectTo(ctx, b, chatID, userID, st, c.SameBank)
	case errors.Is(err, domain.ErrCurrencyMismatch):
		h.rejectTo(ctx, b, chatID, userID, st, c.CurrencyMismatch)
	case errors.Is(err, domain.ErrBankNotFound):
		h.finishOrReply(ctx, b, chatID, userID, st, c.UnknownBank(st.Name))
	default:
		logHandlerErr(userID, domain.CommandTransfer, "transfer", err)
		h.finishOrReply(ctx, b, chatID, userID, st, c.SomethingWentWrong)
	}
}

func (h *Bot) rejectTo(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	message string,
) {
	if st.PromptID == 0 && st.Step != stepTo {
		reply(ctx, b, chatID, message, nil)
		return
	}
	others, ok := h.transferToBanks(ctx, b, chatID, userID, st.BankID, st)
	if !ok {
		return
	}
	st.Step = stepTo
	st.ToName = ""
	st.ToBankID = 0
	h.prompt(ctx, b, chatID, userID, st, message, idKeyboard(callbackTransferToPrefix, others))
}

func (h *Bot) finishOrReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	message string,
) {
	if st.PromptID == 0 {
		reply(ctx, b, chatID, message, nil)
		return
	}
	h.done(ctx, b, chatID, userID, st, message)
}

func (h *Bot) handleTransferCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandTransfer)
	if !h.beginCallback(ctx, b, update) {
		return
	}
	data := update.CallbackQuery.Data
	fromID, isFrom := parsePrefixedID(data, callbackTransferFromPrefix)
	toID, isTo := parsePrefixedID(data, callbackTransferToPrefix)
	if !isFrom && !isTo {
		return
	}
	step := stepBank
	if isTo {
		step = stepTo
	}
	st, ok := h.callbackFSM(ctx, b, update, domain.CommandTransfer, step)
	if !ok {
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	chatID := callbackChatID(update)
	if isFrom {
		bank, ok := h.bankByID(ctx, b, chatID, userID, fromID)
		if !ok {
			return
		}
		h.askTransferTo(ctx, b, chatID, userID, st, bank)
		return
	}
	bank, ok := h.bankByID(ctx, b, chatID, userID, toID)
	if !ok {
		return
	}
	if bank.ID == st.BankID {
		h.rejectTo(ctx, b, chatID, userID, st, copyFrom(ctx).SameBank)
		return
	}
	h.askTransferAmount(ctx, b, chatID, userID, st, bank)
}
