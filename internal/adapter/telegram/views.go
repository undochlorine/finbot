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

func (h *Bot) handleBank(ctx context.Context, b *bot.Bot, update *models.Update) {
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	h.progressBank(ctx, b, messageChatID(update), domain.UserID(from.ID), commandPayload(update.Message.Text), false)
}

func (h *Bot) handleBanks(ctx context.Context, b *bot.Bot, update *models.Update) {
	from := sender(update)
	if from == nil {
		return
	}
	h.replyBanks(ctx, b, messageChatID(update), domain.UserID(from.ID))
}

func (h *Bot) handleTotal(ctx context.Context, b *bot.Bot, update *models.Update) {
	from := sender(update)
	if from == nil {
		return
	}
	h.replyTotal(ctx, b, messageChatID(update), domain.UserID(from.ID))
}

func (h *Bot) handleAll(ctx context.Context, b *bot.Bot, update *models.Update) {
	from := sender(update)
	if from == nil {
		return
	}
	h.replyAll(ctx, b, messageChatID(update), domain.UserID(from.ID))
}

func (h *Bot) continueBank(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	raw string,
) {
	if st.Step != stepBank {
		return
	}
	h.progressBank(ctx, b, chatID, userID, raw, true)
}

func (h *Bot) progressBank(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	payload string,
	fromFSM bool,
) {
	name := strings.TrimSpace(payload)
	if name == "" {
		h.offerBanks(ctx, b, chatID, userID, domain.CommandBank)
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
	h.replyBankCard(ctx, b, chatID, userID, bank, fromFSM)
}

func (h *Bot) replyBankCard(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	bank domain.Bank,
	clear bool,
) {
	if clear {
		h.clearFSM(ctx, userID)
	}
	reply(ctx, b, chatID, bankCard(bank), nil)
}

func (h *Bot) replyBanks(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID) {
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
	reply(ctx, b, chatID, formatBanks(banks), nil)
}

func (h *Bot) replyTotal(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID) {
	total, err := h.svc.Total(ctx, userID)
	if err != nil {
		slog.Error("total", slog.Any("err", err))
		reply(ctx, b, chatID, text.SomethingWentWrong, nil)
		return
	}
	reply(ctx, b, chatID, text.Total(total.Format()), nil)
}

func (h *Bot) replyAll(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID) {
	banks, total, err := h.svc.All(ctx, userID)
	if err != nil {
		slog.Error("all banks", slog.Any("err", err))
		reply(ctx, b, chatID, text.SomethingWentWrong, nil)
		return
	}
	if len(banks) == 0 {
		reply(ctx, b, chatID, text.NoBanks, nil)
		return
	}
	reply(ctx, b, chatID, text.All(formatBanks(banks), text.Total(total.Format())), nil)
}

func (h *Bot) handleBankCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.CallbackQuery == nil {
		return
	}
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	}); err != nil {
		slog.Error("answer callback query", slog.Any("err", err))
	}
	bankID, ok := parseBankCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	st, ok := h.loadFSM(ctx, userID)
	if !ok || st.Flow != domain.CommandBank || st.Step != stepBank {
		return
	}
	bank, err := h.svc.Get(ctx, userID, bankID)
	if err != nil {
		slog.Error("get bank", slog.Any("err", err))
		reply(ctx, b, callbackChatID(update), text.SomethingWentWrong, nil)
		return
	}
	h.replyBankCard(ctx, b, callbackChatID(update), userID, bank, true)
}

func parseBankCallback(data string) (int64, bool) {
	rest, found := strings.CutPrefix(data, callbackBankPrefix)
	if !found {
		return 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func formatBanks(banks []domain.Bank) string {
	lines := make([]string, 0, len(banks))
	for _, bank := range banks {
		lines = append(lines, bankCard(bank))
	}
	return strings.Join(lines, "\n")
}

func bankCard(bank domain.Bank) string {
	return text.BankCard(bank.Name, bank.Balance.Format(), bank.IncludeInTotal)
}
