package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestCancel(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)

	tests := []struct {
		name     string
		text     string
		setup    func(*mocks.MockCache, *mocks.MockHTTPClient, *string)
		wantText string
	}{
		{
			name: "clears pending flow",
			text: "/cancel",
			setup: func(cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandAdd, Step: stepBank}), true, nil)
				cache.EXPECT().Delete(ctx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Canceled,
		},
		{
			name: "mention form",
			text: "/cancel@finbot",
			setup: func(cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
					Flow: domain.CommandNewBank, Step: stepInclude, Name: "Holiday",
				}), true, nil)
				cache.EXPECT().Delete(ctx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Canceled,
		},
		{
			name: "idle",
			text: "/cancel",
			setup: func(cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				cache.EXPECT().Get(ctx, key).Return(nil, false, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.NothingToCancel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			b := newNewBankBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
				tt.setup(cache, client, &sent)
			})
			b.ProcessUpdate(ctx, commandUpdate(tt.text))
			require.Contains(t, sent, tt.wantText)
		})
	}
}

func TestSpendReplacesPendingAdd(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().List(ctx, userID).Return([]domain.Bank{holiday}, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandAdd, Step: stepBank})
		expectSendMessage(t, client, &sent)
		svc.EXPECT().List(ctx, userID).Return([]domain.Bank{holiday}, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandSpend, Step: stepBank})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("/add"))
	b.ProcessUpdate(ctx, commandUpdate("/spend"))
	require.Contains(t, sent, text.AskBank)
	require.Contains(t, sent, callbackSpendPrefix+"7")
	require.NotContains(t, sent, callbackAddPrefix+"7")
}

func TestStaleCallbackIgnored(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)

	b := newNewBankBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandSpend, Step: stepBank}), true, nil)
		expectAnswerCallbackQuery(t, client)
	})
	b.ProcessUpdate(ctx, includeCallbackUpdate(callbackAddPrefix+"7"))
}

func TestExpiredCallbackAsksToStartOver(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)

	tests := []struct {
		name string
		data string
	}{
		{name: "add", data: callbackAddPrefix + "7"},
		{name: "spend", data: callbackSpendPrefix + "7"},
		{name: "set", data: callbackSetPrefix + "7"},
		{name: "delete", data: callbackDeletePrefix + "7"},
		{name: "delete yes", data: callbackDeleteYes + "7"},
		{name: "bank", data: callbackBankPrefix + "7"},
		{name: "toggle", data: callbackTogglePrefix + "7"},
		{name: "newbank include", data: callbackNewBankIncludeYes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			b := newNewBankBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
				cache.EXPECT().Get(ctx, key).Return(nil, false, nil)
				expectAnswerCallbackQuery(t, client)
				expectSendMessage(t, client, &sent)
			})
			b.ProcessUpdate(ctx, includeCallbackUpdate(tt.data))
			require.Contains(t, sent, text.FlowExpired)
		})
	}
}

func TestCommandAtStartCancel(t *testing.T) {
	require.True(t, commandAtStart(commandCancel)(commandUpdate("/"+commandCancel)))
	require.True(t, commandAtStart(commandCancel)(commandUpdate("/"+commandCancel+"@finbot")))
	require.False(t, commandAtStart(commandCancel)(commandUpdate("/"+commandCancel+"foo")))
}
