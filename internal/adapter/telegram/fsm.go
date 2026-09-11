package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"finbot/internal/domain"
)

const (
	commandStart    = "start"
	commandHelp     = "help"
	commandLanguage = "language"
	commandCancel   = "cancel"
	commandFeedback = "feedback"

	stepName    = "name"
	stepInclude = "include"
	stepBank    = "bank"
	stepAmount  = "amount"
	stepConfirm = "confirm"
	stepText    = "text"
	stepTo      = "to"
	stepPick    = "pick"

	callbackNewBankPrefix     = "v1:" + domain.CommandNewBank + ":"
	callbackNewBankIncludeYes = callbackNewBankPrefix + stepInclude + ":1"
	callbackNewBankIncludeNo  = callbackNewBankPrefix + stepInclude + ":0"

	callbackAddPrefix          = "v1:" + domain.CommandAdd + ":"
	callbackSpendPrefix        = "v1:" + domain.CommandSpend + ":"
	callbackSetPrefix          = "v1:" + domain.CommandSet + ":"
	callbackDeletePrefix       = "v1:" + domain.CommandDelete + ":"
	callbackDeleteYes          = callbackDeletePrefix + domain.Yes + ":"
	callbackDeleteNo           = callbackDeletePrefix + domain.No + ":"
	callbackBankPrefix         = "v1:" + domain.CommandBank + ":"
	callbackTogglePrefix       = "v1:" + domain.CommandToggle + ":"
	callbackRenamePrefix       = "v1:" + domain.CommandRename + ":"
	callbackTransferPrefix     = "v1:" + domain.CommandTransfer + ":"
	callbackTransferFromPrefix = callbackTransferPrefix + "from:"
	callbackTransferToPrefix   = callbackTransferPrefix + "to:"
	callbackLanguagePrefix     = "v1:" + commandLanguage + ":"
)

type fsmState struct {
	Flow     string `json:"flow"`
	Step     string `json:"step"`
	Name     string `json:"name,omitempty"`
	BankID   int64  `json:"bank_id,omitempty"`
	ToName   string `json:"to_name,omitempty"`
	ToBankID int64  `json:"to_bank_id,omitempty"`
	PromptID int    `json:"prompt_id,omitempty"`
	SweepIDs []int  `json:"sweep_ids,omitempty"`
}

func (st *fsmState) note(id int) {
	if id == 0 || id == st.PromptID {
		return
	}
	for _, existing := range st.SweepIDs {
		if existing == id {
			return
		}
	}
	st.SweepIDs = append(st.SweepIDs, id)
}

func (st fsmState) withAmount(flow string, bank domain.Bank) fsmState {
	st.Flow = flow
	st.Step = stepAmount
	st.Name = bank.Name
	st.BankID = bank.ID
	return st
}

func (st fsmState) withDeleteConfirm(bank domain.Bank) fsmState {
	st.Flow = domain.CommandDelete
	st.Step = stepConfirm
	st.Name = bank.Name
	st.BankID = bank.ID
	return st
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
		slog.Error("fsm get", logAttrs(ctx, slog.Any("err", err))...)
		return fsmState{}, false
	}
	if !ok {
		return fsmState{}, false
	}
	var st fsmState
	if err := json.Unmarshal(raw, &st); err != nil {
		slog.Error("fsm decode", logAttrs(ctx, slog.Any("err", err))...)
		return fsmState{}, false
	}
	return st, true
}

func (h *Bot) saveFSM(ctx context.Context, userID domain.UserID, st fsmState) {
	raw, err := json.Marshal(st)
	if err != nil {
		slog.Error("fsm encode", logAttrs(ctx, slog.Any("err", err))...)
		return
	}
	if err := h.cache.Set(ctx, fsmKey(userID), raw, h.fsmTTL); err != nil {
		slog.Error("fsm set", logAttrs(ctx, slog.Any("err", err))...)
	}
}

func (h *Bot) clearFSM(ctx context.Context, userID domain.UserID) {
	if err := h.cache.Delete(ctx, fsmKey(userID)); err != nil {
		slog.Error("fsm delete", logAttrs(ctx, slog.Any("err", err))...)
	}
}

func (h *Bot) beginCallback(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update == nil || update.CallbackQuery == nil {
		return false
	}
	if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	}); err != nil {
		slog.Error("answer callback query", logAttrs(ctx, slog.Any("err", err))...)
	}
	return true
}

func (h *Bot) loadCallbackFSM(ctx context.Context, b *bot.Bot, update *models.Update) (fsmState, bool) {
	st, ok := h.loadFSM(ctx, domain.UserID(update.CallbackQuery.From.ID))
	if !ok {
		h.expireCallback(ctx, b, update)
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
	if !ok {
		return fsmState{}, false
	}
	if st.Flow != flow || st.Step != step {
		h.stripStale(ctx, b, update, st)
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
