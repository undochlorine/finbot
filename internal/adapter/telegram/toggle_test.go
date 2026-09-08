package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestToggleCommandFlow(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{
		ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true,
	}
	gifts := domain.Bank{
		ID: 8, UserID: userID, Name: "Gifts", Balance: 1250, IncludeInTotal: false,
	}
	holidayOut := holiday
	holidayOut.IncludeInTotal = false
	giftsIn := gifts
	giftsIn.IncludeInTotal = true

	tests := []struct {
		name     string
		text     string
		setup    func(*mocks.MockService, *mocks.MockCache, *mocks.MockHTTPClient, *string)
		wantText string
		wantCB   string
	}{
		{
			name: "no banks",
			text: "/toggle",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(nil, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.NoBanks,
		},
		{
			name: "picker",
			text: "/toggle",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return([]domain.Bank{holiday}, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandToggle, Step: stepBank})
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskBank,
			wantCB:   callbackTogglePrefix + "7",
		},
		{
			name: "name shortcut excludes without changing balance",
			text: "/toggle Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().Toggle(anyCtx, userID, testBankID).Return(holidayOut, nil)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Toggled("Holiday", "50.00", false),
		},
		{
			name: "name shortcut includes without changing balance",
			text: "/toggle Gifts",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Gifts").Return(gifts, nil)
				svc.EXPECT().Toggle(anyCtx, userID, gifts.ID).Return(giftsIn, nil)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Toggled("Gifts", "12.50", true),
		},
		{
			name: "spaced name shortcut",
			text: "/toggle Holiday Fund",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				fund := domain.Bank{
					ID: testBankID, UserID: userID, Name: "Holiday Fund", Balance: 100, IncludeInTotal: true,
				}
				out := fund
				out.IncludeInTotal = false
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday Fund").Return(fund, nil)
				svc.EXPECT().Toggle(anyCtx, userID, testBankID).Return(out, nil)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Toggled("Holiday Fund", "1.00", false),
		},
		{
			name: "unknown bank",
			text: "/toggle Missing",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Missing").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: text.UnknownBank("Missing"),
		},
		{
			name: "mention shortcut",
			text: "/toggle@finbot Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().Toggle(anyCtx, userID, testBankID).Return(holidayOut, nil)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Toggled("Holiday", "50.00", false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
				tt.setup(svc, cache, client, &sent)
			})
			b.ProcessUpdate(ctx, commandUpdate(tt.text))
			require.Contains(t, sent, tt.wantText)
			require.Contains(t, sent, "42")
			if tt.wantCB != "" {
				require.Contains(t, sent, tt.wantCB)
			}
		})
	}
}

func TestTogglePendingBankName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{
		ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true,
	}
	out := holiday
	out.IncludeInTotal = false
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandToggle, Step: stepBank}), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
		svc.EXPECT().Toggle(anyCtx, userID, testBankID).Return(out, nil)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday"))
	require.Contains(t, sent, text.Toggled("Holiday", "50.00", false))
}

func TestToggleCallbackFlipsAndReplies(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{
		ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true,
	}
	out := holiday
	out.IncludeInTotal = false
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandToggle, Step: stepBank}), true, nil)
		svc.EXPECT().Toggle(anyCtx, userID, testBankID).Return(out, nil)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, callbackUpdate(callbackTogglePrefix+"7"))
	require.Contains(t, sent, text.Toggled("Holiday", "50.00", false))
}

func TestToggleNotFound(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
		svc.EXPECT().Toggle(anyCtx, userID, testBankID).Return(domain.Bank{}, domain.ErrBankNotFound)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("/toggle Holiday"))
	require.Contains(t, sent, text.UnknownBank("Holiday"))
}

func TestToggleThenTotalUsesNewIncludedSet(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{
		ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true,
	}
	out := holiday
	out.IncludeInTotal = false
	var toggled, total string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
		svc.EXPECT().Toggle(anyCtx, userID, testBankID).Return(out, nil)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, &toggled)
		svc.EXPECT().Total(anyCtx, userID).Return(domain.Money(0), nil)
		expectSendMessage(t, client, &total)
	})
	b.ProcessUpdate(ctx, commandUpdate("/toggle Holiday"))
	b.ProcessUpdate(ctx, commandUpdate("/total"))
	require.Contains(t, toggled, text.Toggled("Holiday", "50.00", false))
	require.Contains(t, total, text.Total("0.00"))
}

func TestCommandAtStartToggle(t *testing.T) {
	require.True(t, commandAtStart(domain.CommandToggle)(commandUpdate("/"+domain.CommandToggle)))
	require.True(t, commandAtStart(domain.CommandToggle)(commandUpdate("/"+domain.CommandToggle+" Holiday")))
	require.False(t, commandAtStart(domain.CommandToggle)(commandUpdate("/"+domain.CommandToggle+"foo")))
}
