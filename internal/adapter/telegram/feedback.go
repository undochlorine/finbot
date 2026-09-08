package telegram

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
)

func (h *Bot) handleFeedback(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctx = withCommand(ctx, commandFeedback)
	from := sender(update)
	if from == nil || update == nil || update.Message == nil {
		return
	}
	chatID := messageChatID(update)
	userID := domain.UserID(from.ID)
	h.replacePending(ctx, b, chatID, userID)
	h.progressFeedback(
		ctx,
		b,
		chatID,
		userID,
		fsmState{},
		from.Username,
		commandPayload(update.Message.Text),
	)
}

func (h *Bot) progressFeedback(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	userID domain.UserID,
	st fsmState,
	username string,
	body string,
) {
	if h.adminID == 0 || h.notify == nil {
		reply(ctx, b, chatID, copyFrom(ctx).FeedbackUnavailable, nil)
		return
	}
	body = strings.TrimSpace(body)
	if body == "" {
		st.Flow = commandFeedback
		st.Step = stepText
		h.prompt(ctx, b, chatID, userID, st, copyFrom(ctx).FeedbackAsk, nil)
		return
	}
	if err := h.notify.Notify(ctx, h.adminID, copyFrom(ctx).FeedbackForward(int64(userID), username, body)); err != nil {
		replyErr(ctx, b, chatID, "forward feedback", err)
		return
	}
	h.done(ctx, b, chatID, userID, st, copyFrom(ctx).FeedbackThanks)
}
