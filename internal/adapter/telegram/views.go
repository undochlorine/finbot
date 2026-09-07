package telegram

import (
	"context"
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
	bank, ok := h.bankByName(ctx, b, chatID, userID, name)
	if !ok {
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
	banks, ok := h.loadBanks(ctx, b, chatID, userID)
	if !ok {
		return
	}
	reply(ctx, b, chatID, formatBanks(banks), nil)
}

func (h *Bot) replyTotal(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID) {
	total, err := h.svc.Total(ctx, userID)
	if err != nil {
		replyErr(ctx, b, chatID, "total", err)
		return
	}
	reply(ctx, b, chatID, text.Total(total.Format()), nil)
}

func (h *Bot) replyAll(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID) {
	banks, total, err := h.svc.All(ctx, userID)
	if err != nil {
		replyErr(ctx, b, chatID, "all banks", err)
		return
	}
	if len(banks) == 0 {
		reply(ctx, b, chatID, text.NoBanks, nil)
		return
	}
	reply(ctx, b, chatID, text.All(formatBanks(banks), text.Total(total.Format())), nil)
}

func (h *Bot) handleBankCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.beginCallback(ctx, b, update) {
		return
	}
	bankID, ok := parsePrefixedID(update.CallbackQuery.Data, callbackBankPrefix)
	if !ok {
		return
	}
	if _, ok := h.callbackFSM(ctx, b, update, domain.CommandBank, stepBank); !ok {
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	chatID := callbackChatID(update)
	bank, ok := h.bankByID(ctx, b, chatID, userID, bankID)
	if !ok {
		return
	}
	h.replyBankCard(ctx, b, chatID, userID, bank, true)
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
