package telegram

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

func uniqueIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func emptyKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{}}
}

func sendMessage(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	message string,
	markup models.ReplyMarkup,
) int {
	if b == nil || chatID == 0 {
		return 0
	}
	params := &bot.SendMessageParams{ChatID: chatID, Text: message}
	if markup != nil {
		params.ReplyMarkup = markup
	}
	msg, err := b.SendMessage(ctx, params)
	if err != nil {
		slog.Error("send telegram message", slog.Any("err", err))
		return 0
	}
	return msg.ID
}

func editMessage(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	msgID int,
	message string,
	markup models.ReplyMarkup,
) {
	if b == nil || chatID == 0 || msgID == 0 {
		return
	}
	if markup == nil {
		markup = emptyKeyboard()
	}
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   msgID,
		Text:        message,
		ReplyMarkup: markup,
	})
	if err != nil {
		slog.Error("edit telegram message", slog.Any("err", err))
	}
}

func deleteMessages(ctx context.Context, b *bot.Bot, chatID int64, ids []int) {
	ids = uniqueIDs(ids)
	if b == nil || chatID == 0 || len(ids) == 0 {
		return
	}
	ok, err := b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chatID,
		MessageIDs: ids,
	})
	if err != nil {
		slog.Error("delete telegram messages", slog.Any("err", err))
		return
	}
	if !ok {
		slog.Error("delete telegram messages returned false")
	}
}

func (h *Bot) show(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	st fsmState,
	message string,
	markup models.ReplyMarkup,
) fsmState {
	if st.PromptID != 0 {
		editMessage(ctx, b, chatID, st.PromptID, message, markup)
		return st
	}
	st.PromptID = sendMessage(ctx, b, chatID, message, markup)
	return st
}

func (h *Bot) prompt(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	message string,
	markup models.ReplyMarkup,
) {
	h.saveFSM(ctx, userID, h.show(ctx, b, chatID, st, message, markup))
}

func (h *Bot) done(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	message string,
) {
	deleteMessages(ctx, b, chatID, st.SweepIDs)
	if st.PromptID != 0 {
		editMessage(ctx, b, chatID, st.PromptID, message, nil)
	} else {
		sendMessage(ctx, b, chatID, message, nil)
	}
	h.clearFSM(ctx, userID)
}

func (h *Bot) drop(ctx context.Context, b *bot.Bot, chatID int64, st fsmState) {
	deleteMessages(ctx, b, chatID, append(st.SweepIDs, st.PromptID))
}

func (h *Bot) replacePending(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID) {
	st, ok := h.loadFSM(ctx, userID)
	if !ok {
		return
	}
	h.drop(ctx, b, chatID, st)
	h.clearFSM(ctx, userID)
}

func (h *Bot) expireCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := callbackChatID(update)
	msgID := callbackMessageID(update)
	if msgID != 0 {
		editMessage(ctx, b, chatID, msgID, text.FlowExpired, nil)
		return
	}
	reply(ctx, b, chatID, text.FlowExpired, nil)
}

func (h *Bot) stripStale(ctx context.Context, b *bot.Bot, update *models.Update, st fsmState) {
	id := callbackMessageID(update)
	if id == 0 || id == st.PromptID {
		return
	}
	deleteMessages(ctx, b, callbackChatID(update), []int{id})
}

func callbackMessageID(update *models.Update) int {
	if update == nil || update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return 0
	}
	return update.CallbackQuery.Message.Message.ID
}

func userMessageID(update *models.Update) int {
	if update == nil || update.Message == nil {
		return 0
	}
	return update.Message.ID
}
