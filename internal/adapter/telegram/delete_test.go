package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestDeleteCommandFlow(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}

	tests := []struct {
		name     string
		text     string
		setup    func(*mocks.MockService, *mocks.MockCache, *mocks.MockHTTPClient, *string)
		wantText string
		wantCB   []string
	}{
		{
			name: "no banks",
			text: "/delete",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return(nil, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.NoBanks,
		},
		{
			name: "picker",
			text: "/delete",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return([]domain.Bank{holiday}, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandDelete, Step: stepBank})
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskBank,
			wantCB:   []string{callbackDeletePrefix + "7"},
		},
		{
			name: "name shortcut asks confirm",
			text: "/delete Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
				expectFSMSet(t, cache, ctx, key, confirmState(holiday))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskDeleteConfirm("Holiday"),
			wantCB:   []string{callbackDeleteYes + "7", callbackDeleteNo + "7"},
		},
		{
			name: "spaced name shortcut",
			text: "/delete Holiday Fund",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				fund := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday Fund"}
				svc.EXPECT().GetByName(ctx, userID, "Holiday Fund").Return(fund, nil)
				expectFSMSet(t, cache, ctx, key, confirmState(fund))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskDeleteConfirm("Holiday Fund"),
			wantCB:   []string{callbackDeleteYes + "7", callbackDeleteNo + "7"},
		},
		{
			name: "unknown bank",
			text: "/delete Missing",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Missing").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: text.UnknownBank("Missing"),
		},
		{
			name: "mention shortcut",
			text: "/delete@finbot Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
				expectFSMSet(t, cache, ctx, key, confirmState(holiday))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskDeleteConfirm("Holiday"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
				tt.setup(svc, cache, client, &sent)
			})
			b.ProcessUpdate(ctx, commandUpdate(tt.text))
			require.Contains(t, sent, tt.wantText)
			require.Contains(t, sent, "42")
			for _, cb := range tt.wantCB {
				require.Contains(t, sent, cb)
			}
		})
	}
}

func TestDeletePendingBankName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandDelete, Step: stepBank}), true, nil)
		svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
		expectFSMSet(t, cache, ctx, key, confirmState(holiday))
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday"))
	require.Contains(t, sent, text.AskDeleteConfirm("Holiday"))
}

func TestDeletePendingConfirmYes(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, confirmState(holiday)), true, nil)
		svc.EXPECT().Delete(ctx, userID, testBankID).Return(nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate(domain.Yes))
	require.Contains(t, sent, text.BankDeleted("Holiday"))
}

func TestDeletePendingConfirmNo(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newNewBankBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, confirmState(holiday)), true, nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate(domain.No))
	require.Contains(t, sent, text.DeleteCancelled)
}

func TestDeletePendingConfirmInvalid(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newNewBankBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, confirmState(holiday)), true, nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("maybe"))
	require.Contains(t, sent, text.AskDeleteConfirm("Holiday"))
}

func TestDeleteCallbackAsksConfirm(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandDelete, Step: stepBank}), true, nil)
		svc.EXPECT().Get(ctx, userID, testBankID).Return(holiday, nil)
		expectFSMSet(t, cache, ctx, key, confirmState(holiday))
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, includeCallbackUpdate(callbackDeletePrefix+"7"))
	require.Contains(t, sent, text.AskDeleteConfirm("Holiday"))
	require.Contains(t, sent, callbackDeleteYes+"7")
}

func TestDeleteCallbackYesDeletes(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, confirmState(holiday)), true, nil)
		svc.EXPECT().Delete(ctx, userID, testBankID).Return(nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, includeCallbackUpdate(callbackDeleteYes+"7"))
	require.Contains(t, sent, text.BankDeleted("Holiday"))
}

func TestDeleteCallbackNoCancels(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newNewBankBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, confirmState(holiday)), true, nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, includeCallbackUpdate(callbackDeleteNo+"7"))
	require.Contains(t, sent, text.DeleteCancelled)
}

func TestDeleteNotFound(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, confirmState(holiday)), true, nil)
		svc.EXPECT().Delete(ctx, userID, testBankID).Return(domain.ErrBankNotFound)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate(domain.Yes))
	require.Contains(t, sent, text.UnknownBank("Holiday"))
}

func TestCommandAtStartDelete(t *testing.T) {
	require.True(t, commandAtStart(domain.CommandDelete)(commandUpdate("/"+domain.CommandDelete)))
	require.True(t, commandAtStart(domain.CommandDelete)(commandUpdate("/"+domain.CommandDelete+" Holiday")))
	require.False(t, commandAtStart(domain.CommandDelete)(commandUpdate("/"+domain.CommandDelete+"foo")))
}
