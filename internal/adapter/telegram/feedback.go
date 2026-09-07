package telegram

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

func (h *Bot) handleFeedback(ctx context.Context, b *bot.Bot, update *models.Update) {
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	h.progressFeedback(
		ctx,
		b,
		messageChatID(update),
		domain.UserID(from.ID),
		from.Username,
		commandPayload(update.Message.Text),
	)
}

func (h *Bot) progressFeedback(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	username string,
	body string,
) {
	if h.adminID == 0 || h.notify == nil {
		reply(ctx, b, chatID, text.FeedbackUnavailable, nil)
		return
	}
	body = strings.TrimSpace(body)
	if body == "" {
		h.saveFSM(ctx, userID, fsmState{Flow: commandFeedback, Step: stepText})
		reply(ctx, b, chatID, text.FeedbackAsk, nil)
		return
	}
	if err := h.notify.Notify(ctx, h.adminID, text.FeedbackForward(int64(userID), username, body)); err != nil {
		replyErr(ctx, b, chatID, "forward feedback", err)
		return
	}
	h.clearFSM(ctx, userID)
	reply(ctx, b, chatID, text.FeedbackThanks, nil)
}
