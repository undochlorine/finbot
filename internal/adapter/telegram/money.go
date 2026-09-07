package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

func splitBankAmount(payload string) (name, amount string) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", ""
	}
	i := strings.LastIndex(payload, " ")
	if i < 0 {
		return payload, ""
	}
	head := strings.TrimSpace(payload[:i])
	last := strings.TrimSpace(payload[i+1:])
	if head == "" {
		return payload, ""
	}
	if _, err := domain.ParseMoney(last); err != nil {
		return payload, ""
	}
	return head, last
}

func (h *Bot) handleAdd(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.startMoney(ctx, b, update, flowAdd)
}

func (h *Bot) handleSpend(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.startMoney(ctx, b, update, flowSpend)
}

func (h *Bot) handleSet(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.startMoney(ctx, b, update, flowSet)
}

func (h *Bot) startMoney(ctx context.Context, b *bot.Bot, update *models.Update, flow string) {
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	h.progressMoney(ctx, b, messageChatID(update), domain.UserID(from.ID), flow, commandPayload(update.Message.Text))
}

func (h *Bot) continueMoney(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	switch st.Step {
	case stepBank:
		h.progressMoney(ctx, b, chatID, userID, st.Flow, raw)
	case stepAmount:
		h.applyMoney(ctx, b, chatID, userID, st.Flow, st.BankID, st.Name, raw)
	}
}

func (h *Bot) progressMoney(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	flow string,
	payload string,
) {
	name, amountRaw := splitBankAmount(payload)
	if name == "" {
		h.offerBanks(ctx, b, chatID, userID, flow)
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
	if amountRaw == "" {
		h.saveAmountStep(ctx, userID, flow, bank)
		reply(ctx, b, chatID, askAmountText(flow, bank.Name), nil)
		return
	}
	h.applyMoney(ctx, b, chatID, userID, flow, bank.ID, bank.Name, amountRaw)
}

func (h *Bot) offerBanks(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	flow string,
) {
	banks, err := h.svc.List(ctx, userID)
	if err != nil {
		slog.Error("list banks", slog.Any("err", err))
		reply(ctx, b, chatID, text.SomethingWentWrong, nil)
		return
	}
	if len(banks) == 0 {
		reply(ctx, b, chatID, text.NoBanks, nil)
		return
	}
	h.saveFSM(ctx, userID, fsmState{Flow: flow, Step: stepBank})
	reply(ctx, b, chatID, text.AskBank, bankKeyboard(flow, banks))
}

func (h *Bot) applyMoney(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	flow string,
	bankID int64,
	name string,
	amountRaw string,
) {
	amount, err := domain.ParseMoney(amountRaw)
	if err != nil {
		h.saveFSM(ctx, userID, amountState(flow, bankID, name))
		reply(ctx, b, chatID, text.InvalidAmount, nil)
		return
	}
	bank, err := h.changeBalance(ctx, userID, flow, bankID, amount)
	if errors.Is(err, domain.ErrInvalidAmount) {
		h.saveFSM(ctx, userID, amountState(flow, bankID, name))
		reply(ctx, b, chatID, text.InvalidAmount, nil)
		return
	}
	if errors.Is(err, domain.ErrBankNotFound) {
		h.clearFSM(ctx, userID)
		reply(ctx, b, chatID, text.UnknownBank(name), nil)
		return
	}
	if err != nil {
		slog.Error("change balance", slog.Any("err", err))
		reply(ctx, b, chatID, text.SomethingWentWrong, nil)
		return
	}
	h.clearFSM(ctx, userID)
	reply(ctx, b, chatID, successText(flow, bank, amount), nil)
}

func (h *Bot) changeBalance(
	ctx context.Context,
	userID domain.UserID,
	flow string,
	bankID int64,
	amount domain.Money,
) (domain.Bank, error) {
	var (
		bank domain.Bank
		err  error
	)
	switch flow {
	case flowAdd:
		bank, err = h.svc.Add(ctx, userID, bankID, amount)
	case flowSpend:
		bank, err = h.svc.Spend(ctx, userID, bankID, amount)
	default:
		bank, err = h.svc.Set(ctx, userID, bankID, amount)
	}
	if err != nil {
		return domain.Bank{}, fmt.Errorf("change balance: %w", err)
	}
	return bank, nil
}

func (h *Bot) handleMoneyCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.CallbackQuery == nil {
		return
	}
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	}); err != nil {
		slog.Error("answer callback query", slog.Any("err", err))
	}
	flow, bankID, ok := parseMoneyCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	st, ok := h.loadFSM(ctx, userID)
	if !ok || st.Flow != flow || st.Step != stepBank {
		return
	}
	bank, err := h.svc.Get(ctx, userID, bankID)
	if err != nil {
		slog.Error("get bank", slog.Any("err", err))
		reply(ctx, b, callbackChatID(update), text.SomethingWentWrong, nil)
		return
	}
	h.saveAmountStep(ctx, userID, flow, bank)
	reply(ctx, b, callbackChatID(update), askAmountText(flow, bank.Name), nil)
}

func (h *Bot) saveAmountStep(ctx context.Context, userID domain.UserID, flow string, bank domain.Bank) {
	h.saveFSM(ctx, userID, amountState(flow, bank.ID, bank.Name))
}

func amountState(flow string, bankID int64, name string) fsmState {
	return fsmState{Flow: flow, Step: stepAmount, Name: name, BankID: bankID}
}

func parseMoneyCallback(data string) (flow string, bankID int64, ok bool) {
	for _, p := range []struct {
		prefix string
		flow   string
	}{
		{callbackAddPrefix, flowAdd},
		{callbackSpendPrefix, flowSpend},
		{callbackSetPrefix, flowSet},
	} {
		rest, found := strings.CutPrefix(data, p.prefix)
		if !found {
			continue
		}
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			return "", 0, false
		}
		return p.flow, id, true
	}
	return "", 0, false
}

func bankKeyboard(flow string, banks []domain.Bank) *models.InlineKeyboardMarkup {
	prefix := callbackPrefix(flow)
	rows := make([][]models.InlineKeyboardButton, 0, len(banks))
	for _, bank := range banks {
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         bank.Name,
			CallbackData: prefix + strconv.FormatInt(bank.ID, 10),
		}})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func callbackPrefix(flow string) string {
	switch flow {
	case flowAdd:
		return callbackAddPrefix
	case flowSpend:
		return callbackSpendPrefix
	default:
		return callbackSetPrefix
	}
}

func askAmountText(flow, name string) string {
	switch flow {
	case flowAdd:
		return text.AskAddAmount(name)
	case flowSpend:
		return text.AskSpendAmount(name)
	default:
		return text.AskSetAmount(name)
	}
}

func successText(flow string, bank domain.Bank, amount domain.Money) string {
	switch flow {
	case flowAdd:
		return text.Added(bank.Name, amount.Format(), bank.Balance.Format())
	case flowSpend:
		return text.Spent(bank.Name, amount.Format(), bank.Balance.Format())
	default:
		return text.SetTo(bank.Name, bank.Balance.Format())
	}
}
