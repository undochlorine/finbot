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

const currencyActionPick = "pick"

func currencyChoices(last string) []string {
	codes := []string{domain.CurrencyEUR, domain.CurrencyUSD, domain.CurrencyUSDT}
	code, err := domain.ParseCurrency(last)
	if err != nil {
		return codes
	}
	switch code {
	case domain.CurrencyEUR, domain.CurrencyUSD, domain.CurrencyUSDT:
		return codes
	default:
		return append(codes, code)
	}
}

func currencyKeyboard(last string) *models.InlineKeyboardMarkup {
	codes := currencyChoices(last)
	row := make([]models.InlineKeyboardButton, 0, len(codes))
	for _, code := range codes {
		row = append(row, models.InlineKeyboardButton{
			Text:         domain.CurrencyUnit(code),
			CallbackData: callbackCCYPrefix + code,
		})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{row}}
}

func parseCCYCallback(data string) (string, bool) {
	raw, ok := strings.CutPrefix(data, callbackCCYPrefix)
	if !ok || raw == "" {
		return "", false
	}
	code, err := domain.ParseCurrency(raw)
	if err != nil || code != raw {
		return "", false
	}
	return code, true
}

func (st fsmState) withCurrencyPick(bank domain.Bank) fsmState {
	st.Flow = domain.CommandCurrency
	st.Step = stepCurrency
	st.Name = bank.Name
	st.BankID = bank.ID
	st.Currency = ""
	return st
}

func (st fsmState) withCurrencyConfirm(bank domain.Bank, code string) fsmState {
	st.Flow = domain.CommandCurrency
	st.Step = stepConfirm
	st.Name = bank.Name
	st.BankID = bank.ID
	st.Currency = code
	return st
}

func (h *Bot) handleCurrency(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandCurrency)
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	chatID := messageChatID(update)
	userID := domain.UserID(from.ID)
	h.replacePending(ctx, b, chatID, userID)
	h.progressCurrency(ctx, b, chatID, userID, fsmState{}, commandPayload(update.Message.Text), true)
}

func (h *Bot) continueCurrency(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	switch st.Step {
	case stepBank:
		h.progressCurrency(ctx, b, chatID, userID, st, raw, false)
	case stepCurrency:
		h.applyCurrencyCode(ctx, b, chatID, userID, st, raw, true)
	case stepConfirm:
		yes, parsed := parseYesNo(ctx, raw)
		if !parsed {
			bank, ok := h.bankByID(ctx, b, chatID, userID, st.BankID)
			if !ok {
				return
			}
			h.prompt(
				ctx, b, chatID, userID, st,
				currencyConfirmCopy(ctx, bank, st.Currency),
				currencyConfirmKeyboard(ctx, bank.ID),
			)
			return
		}
		if yes {
			h.changeCurrencyAndReply(ctx, b, chatID, userID, st)
			return
		}
		h.declineCurrency(ctx, b, chatID, userID, st)
	}
}

func (h *Bot) progressCurrency(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	payload string,
	tryCode bool,
) {
	name := strings.TrimSpace(payload)
	if name == "" {
		h.offerBanks(ctx, b, chatID, userID, st, domain.CommandCurrency)
		return
	}
	bank, err := h.svc.GetByName(ctx, userID, name)
	switch {
	case err == nil:
		h.askBankCurrency(ctx, b, chatID, userID, st, bank)
	case !errors.Is(err, domain.ErrBankNotFound):
		replyErr(ctx, b, chatID, "get bank by name", err)
	case tryCode && h.currencyByFirstWord(ctx, b, chatID, userID, st, name):
		return
	default:
		h.unknownBank(ctx, b, chatID, userID, st, name)
	}
}

func (h *Bot) currencyByFirstWord(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	payload string,
) bool {
	bankName, raw, ok := splitRenameArgs(payload)
	if !ok {
		return false
	}
	bank, err := h.svc.GetByName(ctx, userID, bankName)
	if errors.Is(err, domain.ErrBankNotFound) {
		return false
	}
	if err != nil {
		replyErr(ctx, b, chatID, "get bank by name", err)
		return true
	}
	code, err := domain.ParseCurrency(raw)
	if err != nil {
		h.rejectCurrency(ctx, b, chatID, userID, st, false)
		return true
	}
	h.chooseCurrency(ctx, b, chatID, userID, st, bank, code)
	return true
}

func (h *Bot) askBankCurrency(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	bank domain.Bank,
) {
	h.promptCurrency(ctx, b, chatID, userID, st.withCurrencyPick(bank), false)
}

func (h *Bot) askNewBankCurrency(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	include bool,
) {
	st.Flow = domain.CommandNewBank
	st.Step = stepCurrency
	st.Include = include
	h.promptCurrency(ctx, b, chatID, userID, st, false)
}

func (h *Bot) promptCurrency(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	invalid bool,
) {
	last, err := h.svc.LastCurrency(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		replyErr(ctx, b, chatID, "last currency", err)
		return
	}
	msg := copyFrom(ctx).AskCurrency
	if invalid {
		msg = copyFrom(ctx).InvalidCurrency
	}
	h.prompt(ctx, b, chatID, userID, st, msg, currencyKeyboard(last))
}

func (h *Bot) applyCurrencyCode(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
	stay bool,
) {
	code, err := domain.ParseCurrency(raw)
	if err != nil {
		h.rejectCurrency(ctx, b, chatID, userID, st, stay)
		return
	}
	if st.Flow == domain.CommandNewBank {
		h.createBankAndReply(ctx, b, chatID, userID, st, code)
		return
	}
	bank, ok := h.bankByID(ctx, b, chatID, userID, st.BankID)
	if !ok {
		return
	}
	h.chooseCurrency(ctx, b, chatID, userID, st, bank, code)
}

func (h *Bot) chooseCurrency(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	bank domain.Bank,
	code string,
) {
	if bank.Currency == code {
		h.sameCurrency(ctx, b, chatID, userID, st, bank, code)
		return
	}
	h.askCurrencyConfirm(ctx, b, chatID, userID, st, bank, code)
}

func (h *Bot) sameCurrency(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	bank domain.Bank,
	code string,
) {
	if _, err := h.svc.ChangeCurrency(ctx, userID, bank.ID, code); err != nil {
		h.currencyFailed(ctx, b, chatID, userID, st, bank.Name, err)
		return
	}
	h.done(ctx, b, chatID, userID, st, copyFrom(ctx).CurrencyAlready(bank.Name, domain.CurrencyUnit(code)))
}

func (h *Bot) askCurrencyConfirm(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	bank domain.Bank,
	code string,
) {
	next := st.withCurrencyConfirm(bank, code)
	h.prompt(ctx, b, chatID, userID, next, currencyConfirmCopy(ctx, bank, code), currencyConfirmKeyboard(ctx, bank.ID))
}

func currencyConfirmCopy(ctx context.Context, bank domain.Bank, code string) string {
	return copyFrom(ctx).AskCurrencyConfirm(
		bank.Name,
		domain.FormatMoney(bank.Balance, bank.Currency),
		domain.FormatMoney(bank.Balance, code),
	)
}

func (h *Bot) changeCurrencyAndReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
) {
	bank, err := h.svc.ChangeCurrency(ctx, userID, st.BankID, st.Currency)
	if err != nil {
		h.currencyFailed(ctx, b, chatID, userID, st, st.Name, err)
		return
	}
	h.done(ctx, b, chatID, userID, st, copyFrom(ctx).CurrencyChanged(
		bank.Name,
		domain.CurrencyUnit(bank.Currency),
		domain.FormatMoney(bank.Balance, bank.Currency),
	))
}

func (h *Bot) declineCurrency(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
) {
	bank, ok := h.bankByID(ctx, b, chatID, userID, st.BankID)
	if !ok {
		return
	}
	h.done(ctx, b, chatID, userID, st, copyFrom(ctx).CurrencyKept(domain.CurrencyUnit(bank.Currency)))
}

func (h *Bot) rejectCurrency(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	stay bool,
) {
	if stay {
		h.promptCurrency(ctx, b, chatID, userID, st, true)
		return
	}
	if st.PromptID != 0 {
		h.done(ctx, b, chatID, userID, st, copyFrom(ctx).InvalidCurrency)
		return
	}
	reply(ctx, b, chatID, copyFrom(ctx).InvalidCurrency, nil)
}

func (h *Bot) currencyFailed(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	name string,
	err error,
) {
	if errors.Is(err, domain.ErrBankNotFound) {
		h.done(ctx, b, chatID, userID, st, copyFrom(ctx).UnknownBank(name))
		return
	}
	logHandlerErr(userID, domain.CommandCurrency, "change currency", err)
	h.done(ctx, b, chatID, userID, st, copyFrom(ctx).SomethingWentWrong)
}

func (h *Bot) handleCurrencyCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, domain.CommandCurrency)
	if !h.beginCallback(ctx, b, update) {
		return
	}
	action, bankID, ok := parseCurrencyCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	st, ok := h.loadCallbackFSM(ctx, b, update)
	if !ok || st.Flow != domain.CommandCurrency {
		if ok {
			h.stripStale(ctx, b, update, st)
		}
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	chatID := callbackChatID(update)
	switch action {
	case currencyActionPick:
		if st.Step != stepBank {
			h.stripStale(ctx, b, update, st)
			return
		}
		bank, found := h.bankByID(ctx, b, chatID, userID, bankID)
		if !found {
			return
		}
		h.askBankCurrency(ctx, b, chatID, userID, st, bank)
	case domain.Yes:
		if st.Step != stepConfirm || st.BankID != bankID {
			h.stripStale(ctx, b, update, st)
			return
		}
		h.changeCurrencyAndReply(ctx, b, chatID, userID, st)
	case domain.No:
		if st.Step != stepConfirm || st.BankID != bankID {
			h.stripStale(ctx, b, update, st)
			return
		}
		h.declineCurrency(ctx, b, chatID, userID, st)
	}
}

func (h *Bot) handleCCYCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.beginCallback(ctx, b, update) {
		return
	}
	code, ok := parseCCYCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	st, ok := h.loadCallbackFSM(ctx, b, update)
	if !ok {
		return
	}
	if st.Step != stepCurrency || (st.Flow != domain.CommandNewBank && st.Flow != domain.CommandCurrency) {
		h.stripStale(ctx, b, update, st)
		return
	}
	if st.Flow == domain.CommandNewBank && st.Name == "" {
		h.stripStale(ctx, b, update, st)
		return
	}
	ctx = withCommand(ctx, st.Flow)
	userID := domain.UserID(update.CallbackQuery.From.ID)
	chatID := callbackChatID(update)
	h.applyCurrencyCode(ctx, b, chatID, userID, st, code, true)
}

func parseCurrencyCallback(data string) (action string, bankID int64, ok bool) {
	for _, p := range []struct {
		prefix string
		action string
	}{
		{callbackCurrencyYes, domain.Yes},
		{callbackCurrencyNo, domain.No},
		{callbackCurrencyPrefix, currencyActionPick},
	} {
		id, found := parsePrefixedID(data, p.prefix)
		if !found {
			continue
		}
		return p.action, id, true
	}
	return "", 0, false
}

func currencyConfirmKeyboard(ctx context.Context, bankID int64) *models.InlineKeyboardMarkup {
	id := strconv.FormatInt(bankID, 10)
	c := copyFrom(ctx)
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			{Text: c.Yes, CallbackData: callbackCurrencyYes + id},
			{Text: c.No, CallbackData: callbackCurrencyNo + id},
		}},
	}
}
