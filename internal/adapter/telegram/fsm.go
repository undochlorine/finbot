package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
	"finbot/internal/text"
)

const (
	commandStart  = "start"
	commandHelp   = "help"
	commandCancel = "cancel"

	fsmTTL = 10 * time.Minute

	stepName    = "name"
	stepInclude = "include"
	stepBank    = "bank"
	stepAmount  = "amount"
	stepConfirm = "confirm"

	callbackNewBankPrefix     = "v1:" + domain.CommandNewBank + ":"
	callbackNewBankIncludeYes = callbackNewBankPrefix + stepInclude + ":1"
	callbackNewBankIncludeNo  = callbackNewBankPrefix + stepInclude + ":0"

	callbackAddPrefix    = "v1:" + domain.CommandAdd + ":"
	callbackSpendPrefix  = "v1:" + domain.CommandSpend + ":"
	callbackSetPrefix    = "v1:" + domain.CommandSet + ":"
	callbackDeletePrefix = "v1:" + domain.CommandDelete + ":"
	callbackDeleteYes    = callbackDeletePrefix + domain.Yes + ":"
	callbackDeleteNo     = callbackDeletePrefix + domain.No + ":"
	callbackBankPrefix   = "v1:" + domain.CommandBank + ":"
	callbackTogglePrefix = "v1:" + domain.CommandToggle + ":"
)

type fsmState struct {
	Flow   string `json:"flow"`
	Step   string `json:"step"`
	Name   string `json:"name,omitempty"`
	BankID int64  `json:"bank_id,omitempty"`
}

func fsmKey(userID domain.UserID) string {
	return fmt.Sprintf("fsm:%d", userID)
}

func parsePrefixedID(data, prefix string) (int64, bool) {
	rest, found := strings.CutPrefix(data, prefix)
	if !found {
		return 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func (h *Bot) loadFSM(ctx context.Context, userID domain.UserID) (fsmState, bool) {
	raw, ok, err := h.cache.Get(ctx, fsmKey(userID))
	if err != nil {
		slog.Error("fsm get", slog.Any("err", err))
		return fsmState{}, false
	}
	if !ok {
		return fsmState{}, false
	}
	var st fsmState
	if err := json.Unmarshal(raw, &st); err != nil {
		slog.Error("fsm decode", slog.Any("err", err))
		return fsmState{}, false
	}
	return st, true
}

func (h *Bot) saveFSM(ctx context.Context, userID domain.UserID, st fsmState) {
	raw, err := json.Marshal(st)
	if err != nil {
		slog.Error("fsm encode", slog.Any("err", err))
		return
	}
	if err := h.cache.Set(ctx, fsmKey(userID), raw, fsmTTL); err != nil {
		slog.Error("fsm set", slog.Any("err", err))
	}
}

func (h *Bot) clearFSM(ctx context.Context, userID domain.UserID) {
	if err := h.cache.Delete(ctx, fsmKey(userID)); err != nil {
		slog.Error("fsm delete", slog.Any("err", err))
	}
}

func (h *Bot) beginCallback(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update == nil || update.CallbackQuery == nil {
		return false
	}
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	}); err != nil {
		slog.Error("answer callback query", slog.Any("err", err))
	}
	return true
}

func (h *Bot) loadCallbackFSM(ctx context.Context, b *bot.Bot, update *models.Update) (fsmState, bool) {
	st, ok := h.loadFSM(ctx, domain.UserID(update.CallbackQuery.From.ID))
	if !ok {
		reply(ctx, b, callbackChatID(update), text.FlowExpired, nil)
		return fsmState{}, false
	}
	return st, true
}

func (h *Bot) callbackFSM(
	ctx context.Context,
	b *bot.Bot,
	update *models.Update,
	flow, step string,
) (fsmState, bool) {
	st, ok := h.loadCallbackFSM(ctx, b, update)
	if !ok || st.Flow != flow || st.Step != step {
		return fsmState{}, false
	}
	return st, true
}

func callbackChatID(update *models.Update) int64 {
	if update == nil || update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return 0
	}
	return update.CallbackQuery.Message.Message.Chat.ID
}
