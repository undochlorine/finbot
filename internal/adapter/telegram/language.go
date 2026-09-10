package telegram

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

func (h *Bot) handleLanguage(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, commandLanguage)
	from := sender(update)
	if from == nil {
		return
	}
	chatID := messageChatID(update)
	userID := domain.UserID(from.ID)
	h.replacePending(ctx, b, chatID, userID)
	h.offerLanguage(ctx, b, chatID, userID, commandLanguage)
}

func (h *Bot) offerLanguage(ctx context.Context, b *bot.Bot, chatID int64, userID domain.UserID, flow string) {
	h.prompt(ctx, b, chatID, userID, fsmState{Flow: flow, Step: stepPick}, text.AskLanguage, languageKeyboard())
}

func languageKeyboard() *models.InlineKeyboardMarkup {
	opts := text.LanguageOptions()
	rows := make([][]models.InlineKeyboardButton, 0, len(opts))
	for _, opt := range opts {
		rows = append(rows, []models.InlineKeyboardButton{{
			Text:         opt.Label,
			CallbackData: callbackLanguagePrefix + opt.Code,
		}})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func parseLanguageCallback(data string) (string, bool) {
	rest, ok := strings.CutPrefix(data, callbackLanguagePrefix)
	if !ok || !domain.KnownLocale(rest) {
		return "", false
	}
	return rest, true
}

func (h *Bot) handleLanguageCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, commandLanguage)
	if !h.beginCallback(ctx, b, update) {
		return
	}
	locale, ok := parseLanguageCallback(update.CallbackQuery.Data)
	if !ok {
		return
	}
	st, ok := h.loadCallbackFSM(ctx, b, update)
	if !ok {
		return
	}
	if st.Step != stepPick || (st.Flow != commandLanguage && st.Flow != commandStart) {
		h.stripStale(ctx, b, update, st)
		return
	}
	userID := domain.UserID(update.CallbackQuery.From.ID)
	chatID := callbackChatID(update)
	if err := h.svc.SetLocale(ctx, userID, locale); err != nil {
		replyErr(ctx, b, chatID, "set locale", err)
		return
	}
	ctx = withLocale(ctx, locale)
	msg := copyFrom(ctx).LanguageSetTo(locale)
	if st.Flow == commandStart {
		msg = copyFrom(ctx).Start
	}
	h.done(ctx, b, chatID, userID, st, msg)
}
