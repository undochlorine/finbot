package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
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
	h.startMoney(ctx, b, update, domain.CommandAdd)
}

func (h *Bot) handleSpend(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.startMoney(ctx, b, update, domain.CommandSpend)
}

func (h *Bot) handleSet(ctx context.Context, b *bot.Bot, update *models.Update) {
	h.startMoney(ctx, b, update, domain.CommandSet)
}

func (h *Bot) startMoney(ctx context.Context, b *bot.Bot, update *models.Update, flow string) {
	ctx = withCommand(ctx, flow)
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	chatID := messageChatID(update)
	userID := domain.UserID(from.ID)
	h.replacePending(ctx, b, chatID, userID)
	h.progressMoney(ctx, b, chatID, userID, fsmState{Flow: flow}, commandPayload(update.Message.Text))
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
		h.progressMoney(ctx, b, chatID, userID, st, raw)
	case stepAmount:
		h.applyMoney(ctx, b, chatID, userID, st, raw)
	}
}

func (h *Bot) progressMoney(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	payload string,
) {
	name, amountRaw := splitBankAmount(payload)
	if name == "" {
		h.offerBanks(ctx, b, chatID, userID, st, st.Flow)
		return
	}
	bank, ok := h.bankByName(ctx, b, chatID, userID, st, name)
	if !ok {
		return
	}
	if amountRaw == "" {
		h.prompt(ctx, b, chatID, userID, st.withAmount(st.Flow, bank), askAmountText(ctx, st.Flow, bank.Name), nil)
		return
	}
	st.Name = bank.Name
	st.BankID = bank.ID
	h.applyMoney(ctx, b, chatID, userID, st, amountRaw)
}

func (h *Bot) applyMoney(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	amountRaw string,
) {
	amount, err := domain.ParseMoney(amountRaw)
	if err != nil {
		h.repromptAmount(ctx, b, chatID, userID, st)
		return
	}
	bank, err := h.changeBalance(ctx, userID, st.Flow, st.BankID, amount)
	if errors.Is(err, domain.ErrInvalidAmount) {
		h.repromptAmount(ctx, b, chatID, userID, st)
		return
	}
	if errors.Is(err, domain.ErrBankNotFound) {
		h.done(ctx, b, chatID, userID, st, copyFrom(ctx).UnknownBank(st.Name))
		return
	}
	if err != nil {
		logHandlerErr(userID, st.Flow, "change balance", err)
		h.done(ctx, b, chatID, userID, st, copyFrom(ctx).SomethingWentWrong)
		return
	}
	h.done(ctx, b, chatID, userID, st, successText(ctx, st.Flow, bank, amount))
}

func (h *Bot) repromptAmount(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
) {
	bank := domain.Bank{ID: st.BankID, Name: st.Name}
	h.prompt(ctx, b, chatID, userID, st.withAmount(st.Flow, bank), copyFrom(ctx).InvalidAmount, nil)
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
	case domain.CommandAdd:
		bank, err = h.svc.Add(ctx, userID, bankID, amount)
	case domain.CommandSpend:
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
	flow, bankID, ok := parseMoneyCallback(update.CallbackQuery.Data)
	if ok {
		ctx = withCommand(ctx, flow)
	}
	if !h.beginCallback(ctx, b, update) {
		return
	}
	if !ok {
		return
	}
	st, ok := h.callbackFSM(ctx, b, update, flow, stepBank)
	if !ok {
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	chatID := callbackChatID(update)
	bank, ok := h.bankByID(ctx, b, chatID, userID, bankID)
	if !ok {
		return
	}
	h.prompt(ctx, b, chatID, userID, st.withAmount(flow, bank), askAmountText(ctx, flow, bank.Name), nil)
}

func parseMoneyCallback(data string) (flow string, bankID int64, ok bool) {
	for _, p := range []struct {
		prefix string
		flow   string
	}{
		{callbackAddPrefix, domain.CommandAdd},
		{callbackSpendPrefix, domain.CommandSpend},
		{callbackSetPrefix, domain.CommandSet},
	} {
		id, found := parsePrefixedID(data, p.prefix)
		if !found {
			continue
		}
		return p.flow, id, true
	}
	return "", 0, false
}

func askAmountText(ctx context.Context, flow, name string) string {
	c := copyFrom(ctx)
	switch flow {
	case domain.CommandAdd:
		return c.AskAddAmount(name)
	case domain.CommandSpend:
		return c.AskSpendAmount(name)
	default:
		return c.AskSetAmount(name)
	}
}

func successText(ctx context.Context, flow string, bank domain.Bank, amount domain.Money) string {
	c := copyFrom(ctx)
	switch flow {
	case domain.CommandAdd:
		return c.Added(bank.Name, amount.Format(), bank.Balance.Format())
	case domain.CommandSpend:
		return c.Spent(bank.Name, amount.Format(), bank.Balance.Format())
	default:
		return c.SetTo(bank.Name, bank.Balance.Format())
	}
}
