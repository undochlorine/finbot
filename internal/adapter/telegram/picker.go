package telegram

import (
	"context"
	"errors"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

func (h *Bot) offerBanks(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	flow string,
) {
	banks, ok := h.loadBanks(ctx, b, chatID, userID)
	if !ok {
		return
	}
	st.Flow = flow
	st.Step = stepBank
	st.Name = ""
	st.BankID = 0
	h.prompt(ctx, b, chatID, userID, st, text.AskBank, bankKeyboard(flow, banks))
}

func (h *Bot) loadBanks(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
) ([]domain.Bank, bool) {
	banks, err := h.svc.List(ctx, userID)
	if err != nil {
		replyErr(ctx, b, chatID, "list banks", err)
		return nil, false
	}
	if len(banks) == 0 {
		reply(ctx, b, chatID, text.NoBanks, nil)
		return nil, false
	}
	return banks, true
}

func (h *Bot) bankByName(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	name string,
) (domain.Bank, bool) {
	bank, err := h.svc.GetByName(ctx, userID, name)
	if errors.Is(err, domain.ErrBankNotFound) {
		if st.PromptID == 0 {
			reply(ctx, b, chatID, text.UnknownBank(name), nil)
			return domain.Bank{}, false
		}
		banks, listErr := h.svc.List(ctx, userID)
		if listErr != nil {
			replyErr(ctx, b, chatID, "list banks", listErr)
			return domain.Bank{}, false
		}
		var markup models.ReplyMarkup
		if len(banks) > 0 {
			st.Step = stepBank
			markup = bankKeyboard(st.Flow, banks)
		}
		h.prompt(ctx, b, chatID, userID, st, text.UnknownBank(name), markup)
		return domain.Bank{}, false
	}
	if err != nil {
		replyErr(ctx, b, chatID, "get bank by name", err)
		return domain.Bank{}, false
	}
	return bank, true
}

func (h *Bot) bankByID(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	bankID int64,
) (domain.Bank, bool) {
	bank, err := h.svc.Get(ctx, userID, bankID)
	if err != nil {
		replyErr(ctx, b, chatID, "get bank", err)
		return domain.Bank{}, false
	}
	return bank, true
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
	case domain.CommandAdd:
		return callbackAddPrefix
	case domain.CommandSpend:
		return callbackSpendPrefix
	case domain.CommandDelete:
		return callbackDeletePrefix
	case domain.CommandBank:
		return callbackBankPrefix
	case domain.CommandToggle:
		return callbackTogglePrefix
	default:
		return callbackSetPrefix
	}
}
