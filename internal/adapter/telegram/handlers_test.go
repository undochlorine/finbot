package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestCommandAtStart(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		text string
		upd  *models.Update
		want bool
	}{
		{name: "exact", cmd: "start", text: "/start", want: true},
		{name: "args", cmd: "start", text: "/start payload", want: true},
		{name: "mention", cmd: "start", text: "/start@finbot", want: true},
		{name: "mention args", cmd: "start", text: "/start@finbot payload", want: true},
		{name: "case", cmd: "start", text: "/START", want: true},
		{name: "prefix collision", cmd: "start", text: "/startfoo", want: false},
		{name: "other command", cmd: "start", text: "/help", want: false},
		{name: "plain text", cmd: "start", text: "start", want: false},
		{name: "help exact", cmd: "help", text: "/help", want: true},
		{name: "help mention", cmd: "help", text: "/help@finbot", want: true},
		{name: "nil update", cmd: "start", want: false},
		{name: "nil message", cmd: "start", upd: &models.Update{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upd := tt.upd
			if upd == nil && tt.text != "" {
				upd = &models.Update{Message: &models.Message{Text: tt.text}}
			}
			require.Equal(t, tt.want, commandAtStart(tt.cmd)(upd))
		})
	}
}

func TestStartAndHelpReply(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		text     string
		wantText string
		wantSend bool
	}{
		{name: "start", text: "/start", wantText: text.Start, wantSend: true},
		{name: "start with payload", text: "/start payload", wantText: text.Start, wantSend: true},
		{name: "help", text: "/help", wantText: text.Help, wantSend: true},
		{name: "unknown command", text: "/unknown", wantSend: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			b := newCommandBot(t, ctx, tt.wantSend, &sent)
			b.ProcessUpdate(ctx, commandUpdate(tt.text))
			if !tt.wantSend {
				require.Empty(t, sent)
				return
			}
			require.Contains(t, sent, tt.wantText)
			require.Contains(t, sent, "42")
		})
	}
}

func TestStartReferral(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		text    string
		call    bool
		ref     domain.UserID
		callErr error
	}{
		{name: "valid payload", text: "/start 99", call: true, ref: 99},
		{name: "self", text: "/start 42"},
		{name: "zero", text: "/start 0"},
		{name: "non numeric", text: "/start nope"},
		{name: "repo error still welcomes", text: "/start 99", call: true, ref: 99, callErr: errors.New("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			var svc *mocks.MockService
			b := newTestBot(t, ctx, func(s *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient) {
				svc = s
				if tt.call {
					s.EXPECT().SetReferredByIfEmpty(anyCtx, domain.UserID(telegramUserID), tt.ref).Return(tt.callErr)
				}
				expectSendMessage(t, client, &sent)
			})
			b.ProcessUpdate(ctx, commandUpdate(tt.text))
			require.Contains(t, sent, text.Start)
			if !tt.call {
				require.True(t, svc.AssertNotCalled(t, "SetReferredByIfEmpty"))
			}
		})
	}
}
