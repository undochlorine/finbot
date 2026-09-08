package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestParseTransferTriple(t *testing.T) {
	tests := []struct {
		name       string
		payload    string
		wantFrom   string
		wantTo     string
		wantAmount string
		ok         bool
	}{
		{name: "empty"},
		{name: "from only", payload: "Holiday"},
		{name: "from and to", payload: "Holiday Gifts"},
		{name: "triple", payload: "Holiday Gifts 50", wantFrom: "Holiday", wantTo: "Gifts", wantAmount: "50", ok: true},
		{name: "decimal", payload: "Holiday Gifts 12.50", wantFrom: "Holiday", wantTo: "Gifts", wantAmount: "12.50", ok: true},
		{name: "extra spaces", payload: "  Holiday   Gifts  50  ", wantFrom: "Holiday", wantTo: "Gifts", wantAmount: "50", ok: true},
		{name: "from plus amount is not a triple", payload: "Holiday 50"},
		{name: "four tokens", payload: "Holiday Fund Gifts 50"},
		{name: "invalid amount", payload: "Holiday Gifts abc"},
		{name: "zero is a number", payload: "Holiday Gifts 0", wantFrom: "Holiday", wantTo: "Gifts", wantAmount: "0", ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, amount, ok := parseTransferTriple(tt.payload)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.wantFrom, from)
			require.Equal(t, tt.wantTo, to)
			require.Equal(t, tt.wantAmount, amount)
		})
	}
}

func TestTransferCommandFlow(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts", Balance: 2500}
	both := []domain.Bank{holiday, gifts}

	tests := []struct {
		name     string
		text     string
		setup    func(*mocks.MockService, *mocks.MockCache, *mocks.MockHTTPClient, *string)
		wantText string
		wantCB   []string
	}{
		{
			name: "no banks",
			text: "/transfer",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(nil, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.NoBanks,
		},
		{
			name: "one bank",
			text: "/transfer",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return([]domain.Bank{holiday}, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.NeedTwoBanks,
		},
		{
			name: "picker",
			text: "/transfer",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil).Times(2)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandTransfer, Step: stepBank})
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskTransferFrom,
			wantCB:   []string{callbackTransferFromPrefix + "7", callbackTransferFromPrefix + "8"},
		},
		{
			name: "from shortcut asks to",
			text: "/transfer Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil).Times(2)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{}.withTransferFrom(holiday))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskTransferTo("Holiday"),
			wantCB:   []string{callbackTransferToPrefix + "8"},
		},
		{
			name: "spaced from name",
			text: "/transfer Holiday Fund",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				fund := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday Fund", Balance: 5000}
				svc.EXPECT().List(anyCtx, userID).Return([]domain.Bank{fund, gifts}, nil).Times(2)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday Fund").Return(fund, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{}.withTransferFrom(fund))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskTransferTo("Holiday Fund"),
		},
		{
			name: "triple completes",
			text: "/transfer Holiday Gifts 50",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Gifts").Return(gifts, nil)
				svc.EXPECT().Transfer(anyCtx, userID, holiday.ID, gifts.ID, domain.Money(5000)).Return(
					domain.Bank{ID: holiday.ID, Name: "Holiday", Balance: 0},
					domain.Bank{ID: gifts.ID, Name: "Gifts", Balance: 7500},
					nil,
				)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Transferred("Holiday", "Gifts", "50.00", "0.00", "75.00"),
		},
		{
			name: "from plus amount is from name",
			text: "/transfer Holiday 50",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday 50").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: text.UnknownBank("Holiday 50"),
		},
		{
			name: "four tokens are from name",
			text: "/transfer Holiday Fund Gifts 50",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday Fund Gifts 50").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: text.UnknownBank("Holiday Fund Gifts 50"),
		},
		{
			name: "failed triple falls back to full remainder",
			text: "/transfer Holiday Missing 50",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Missing").Return(domain.Bank{}, domain.ErrBankNotFound)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday Missing 50").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: text.UnknownBank("Holiday Missing 50"),
		},
		{
			name: "same bank triple",
			text: "/transfer Holiday Holiday 50",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil).Times(2)
				svc.EXPECT().Transfer(anyCtx, userID, holiday.ID, holiday.ID, domain.Money(5000)).
					Return(domain.Bank{}, domain.Bank{}, domain.ErrSameBank)
				expectSendMessage(t, client, sent)
			},
			wantText: text.SameBank,
		},
		{
			name: "zero amount triple",
			text: "/transfer Holiday Gifts 0",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Gifts").Return(gifts, nil)
				svc.EXPECT().Transfer(anyCtx, userID, holiday.ID, gifts.ID, domain.Money(0)).
					Return(domain.Bank{}, domain.Bank{}, domain.ErrInvalidAmount)
				expectSendMessage(t, client, sent)
			},
			wantText: text.InvalidAmount,
		},
		{
			name: "unknown from",
			text: "/transfer Missing",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Missing").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: text.UnknownBank("Missing"),
		},
		{
			name: "mention shortcut",
			text: "/transfer@finbot Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(both, nil).Times(2)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{}.withTransferFrom(holiday))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskTransferTo("Holiday"),
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
			for _, cb := range tt.wantCB {
				require.Contains(t, sent, cb)
			}
		})
	}
}

func TestTransferPendingFromName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts", Balance: 2500}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandTransfer, Step: stepBank}), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
		svc.EXPECT().List(anyCtx, userID).Return([]domain.Bank{holiday, gifts}, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{}.withTransferFrom(holiday))
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday"))
	require.Contains(t, sent, text.AskTransferTo("Holiday"))
}

func TestTransferPendingToName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts", Balance: 2500}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{}.withTransferFrom(holiday)), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Gifts").Return(gifts, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{}.withTransferFrom(holiday).withTransferTo(gifts))
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Gifts"))
	require.Contains(t, sent, text.AskTransferAmount("Holiday", "Gifts"))
}

func TestTransferPendingSameBank(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts", Balance: 2500}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{}.withTransferFrom(holiday)), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
		svc.EXPECT().List(anyCtx, userID).Return([]domain.Bank{holiday, gifts}, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{}.withTransferFrom(holiday))
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday"))
	require.Contains(t, sent, text.SameBank)
	require.Contains(t, sent, callbackTransferToPrefix+"8")
}

func TestTransferPendingAmount(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts", Balance: 2500}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{}.withTransferFrom(holiday).withTransferTo(gifts)), true, nil)
		svc.EXPECT().Transfer(anyCtx, userID, holiday.ID, gifts.ID, domain.Money(5000)).Return(
			domain.Bank{ID: holiday.ID, Name: "Holiday", Balance: 0},
			domain.Bank{ID: gifts.ID, Name: "Gifts", Balance: 7500},
			nil,
		)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("50"))
	require.Contains(t, sent, text.Transferred("Holiday", "Gifts", "50.00", "0.00", "75.00"))
}

func TestTransferCallbackFromAsksTo(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts", Balance: 2500}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandTransfer, Step: stepBank}), true, nil)
		svc.EXPECT().Get(anyCtx, userID, testBankID).Return(holiday, nil)
		svc.EXPECT().List(anyCtx, userID).Return([]domain.Bank{holiday, gifts}, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{}.withTransferFrom(holiday))
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, callbackUpdate(callbackTransferFromPrefix+"7"))
	require.Contains(t, sent, text.AskTransferTo("Holiday"))
	require.Contains(t, sent, callbackTransferToPrefix+"8")
	require.NotContains(t, sent, callbackTransferToPrefix+"7")
}

func TestTransferCallbackToAsksAmount(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts", Balance: 2500}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{}.withTransferFrom(holiday)), true, nil)
		svc.EXPECT().Get(anyCtx, userID, gifts.ID).Return(gifts, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{}.withTransferFrom(holiday).withTransferTo(gifts))
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, callbackUpdate(callbackTransferToPrefix+"8"))
	require.Contains(t, sent, text.AskTransferAmount("Holiday", "Gifts"))
}

func TestTransferHygieneEditsAndSweeps(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts", Balance: 2500}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{
			Flow:     domain.CommandTransfer,
			Step:     stepAmount,
			Name:     holiday.Name,
			BankID:   holiday.ID,
			ToName:   gifts.Name,
			ToBankID: gifts.ID,
			PromptID: testPromptID,
		}), true, nil)
		svc.EXPECT().Transfer(anyCtx, userID, holiday.ID, gifts.ID, domain.Money(5000)).Return(
			domain.Bank{ID: holiday.ID, Name: "Holiday", Balance: 0},
			domain.Bank{ID: gifts.ID, Name: "Gifts", Balance: 7500},
			nil,
		)
		expectDeleteMessages(t, client)
		expectEditMessage(t, client, &sent)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
	})
	b.ProcessUpdate(ctx, commandUpdateID("50", testUserMsgID))
	require.Contains(t, sent, text.Transferred("Holiday", "Gifts", "50.00", "0.00", "75.00"))
	require.Contains(t, sent, `"inline_keyboard":[]`)
}

func TestCommandAtStartTransfer(t *testing.T) {
	require.True(t, commandAtStart(domain.CommandTransfer)(commandUpdate("/"+domain.CommandTransfer)))
	require.True(t, commandAtStart(domain.CommandTransfer)(commandUpdate("/"+domain.CommandTransfer+" Holiday Gifts 50")))
	require.False(t, commandAtStart(domain.CommandTransfer)(commandUpdate("/"+domain.CommandTransfer+"foo")))
}
