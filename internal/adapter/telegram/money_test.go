package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

const testBankID int64 = 7

func TestSplitBankAmount(t *testing.T) {
	tests := []struct {
		name       string
		payload    string
		wantName   string
		wantAmount string
	}{
		{name: "empty"},
		{name: "whitespace", payload: "   "},
		{name: "name only", payload: "Holiday", wantName: "Holiday"},
		{name: "name and amount", payload: "Holiday 100", wantName: "Holiday", wantAmount: "100"},
		{name: "spaced name and amount", payload: "Holiday Fund 100", wantName: "Holiday Fund", wantAmount: "100"},
		{name: "decimal amount", payload: "Gifts 12.50", wantName: "Gifts", wantAmount: "12.50"},
		{name: "zero amount", payload: "Live 0", wantName: "Live", wantAmount: "0"},
		{name: "negative amount", payload: "Live -10", wantName: "Live", wantAmount: "-10"},
		{name: "numeric name only", payload: "100", wantName: "100"},
		{name: "name that looks like amount plus word", payload: "Holiday Fund", wantName: "Holiday Fund"},
		{name: "invalid last token stays in name", payload: "Holiday abc", wantName: "Holiday abc"},
		{name: "trims", payload: "  Holiday   100  ", wantName: "Holiday", wantAmount: "100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotAmount := splitBankAmount(tt.payload)
			require.Equal(t, tt.wantName, gotName)
			require.Equal(t, tt.wantAmount, gotAmount)
		})
	}
}

func TestMoneyCommandFlow(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}

	tests := []struct {
		name     string
		text     string
		setup    func(*mocks.MockService, *mocks.MockCache, *mocks.MockHTTPClient, *string)
		wantText string
		wantCB   string
	}{
		{
			name: "add with no banks",
			text: "/add",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return(nil, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.NoBanks,
		},
		{
			name: "spend with no banks",
			text: "/spend",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return(nil, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.NoBanks,
		},
		{
			name: "set with no banks",
			text: "/set",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return(nil, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.NoBanks,
		},
		{
			name: "add picker",
			text: "/add",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return([]domain.Bank{holiday}, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: flowAdd, Step: stepBank})
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskBank,
			wantCB:   callbackAddPrefix + "7",
		},
		{
			name: "spend picker",
			text: "/spend",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return([]domain.Bank{holiday}, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: flowSpend, Step: stepBank})
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskBank,
			wantCB:   callbackSpendPrefix + "7",
		},
		{
			name: "set picker",
			text: "/set",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return([]domain.Bank{holiday}, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: flowSet, Step: stepBank})
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskBank,
			wantCB:   callbackSetPrefix + "7",
		},
		{
			name: "add name only asks amount",
			text: "/add Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{
					Flow: flowAdd, Step: stepAmount, Name: "Holiday", BankID: testBankID,
				})
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskAddAmount("Holiday"),
		},
		{
			name: "add name and amount",
			text: "/add Holiday 100",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().
					Add(ctx, userID, testBankID, domain.Money(10000)).
					Return(domain.Bank{ID: testBankID, Name: "Holiday", Balance: 15000}, nil)
				cache.EXPECT().Delete(ctx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Added("Holiday", "100.00", "150.00"),
		},
		{
			name: "add spaced name and amount",
			text: "/add Holiday Fund 100",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				fund := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday Fund", Balance: 0}
				svc.EXPECT().GetByName(ctx, userID, "Holiday Fund").Return(fund, nil)
				svc.EXPECT().
					Add(ctx, userID, testBankID, domain.Money(10000)).
					Return(domain.Bank{ID: testBankID, Name: "Holiday Fund", Balance: 10000}, nil)
				cache.EXPECT().Delete(ctx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Added("Holiday Fund", "100.00", "100.00"),
		},
		{
			name: "spend name and amount",
			text: "/spend Gifts 12.50",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				gifts := domain.Bank{ID: testBankID, UserID: userID, Name: "Gifts", Balance: 2000}
				svc.EXPECT().GetByName(ctx, userID, "Gifts").Return(gifts, nil)
				svc.EXPECT().
					Spend(ctx, userID, testBankID, domain.Money(1250)).
					Return(domain.Bank{ID: testBankID, Name: "Gifts", Balance: 750}, nil)
				cache.EXPECT().Delete(ctx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Spent("Gifts", "12.50", "7.50"),
		},
		{
			name: "set name and amount",
			text: "/set Live 0",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				live := domain.Bank{ID: testBankID, UserID: userID, Name: "Live", Balance: 900}
				svc.EXPECT().GetByName(ctx, userID, "Live").Return(live, nil)
				svc.EXPECT().
					Set(ctx, userID, testBankID, domain.Money(0)).
					Return(domain.Bank{ID: testBankID, Name: "Live", Balance: 0}, nil)
				cache.EXPECT().Delete(ctx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.SetTo("Live", "0.00"),
		},
		{
			name: "set negative amount",
			text: "/set Live -10",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				live := domain.Bank{ID: testBankID, UserID: userID, Name: "Live", Balance: 0}
				svc.EXPECT().GetByName(ctx, userID, "Live").Return(live, nil)
				svc.EXPECT().
					Set(ctx, userID, testBankID, domain.Money(-1000)).
					Return(domain.Bank{ID: testBankID, Name: "Live", Balance: -1000}, nil)
				cache.EXPECT().Delete(ctx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.SetTo("Live", "-10.00"),
		},
		{
			name: "unknown bank",
			text: "/add Missing",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Missing").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: text.UnknownBank("Missing"),
		},
		{
			name: "add negative amount re-prompts",
			text: "/add Holiday -1",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().
					Add(ctx, userID, testBankID, domain.Money(-100)).
					Return(domain.Bank{}, domain.ErrInvalidAmount)
				expectFSMSet(t, cache, ctx, key, fsmState{
					Flow: flowAdd, Step: stepAmount, Name: "Holiday", BankID: testBankID,
				})
				expectSendMessage(t, client, sent)
			},
			wantText: text.InvalidAmount,
		},
		{
			name: "mention shortcut",
			text: "/add@finbot Holiday 100",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().
					Add(ctx, userID, testBankID, domain.Money(10000)).
					Return(domain.Bank{ID: testBankID, Name: "Holiday", Balance: 15000}, nil)
				cache.EXPECT().Delete(ctx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.Added("Holiday", "100.00", "150.00"),
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
			if tt.wantCB != "" {
				require.Contains(t, sent, tt.wantCB)
			}
		})
	}
}

func TestMoneyPendingBankName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 0}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: flowAdd, Step: stepBank}), true, nil)
		svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{
			Flow: flowAdd, Step: stepAmount, Name: "Holiday", BankID: testBankID,
		})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday"))
	require.Contains(t, sent, text.AskAddAmount("Holiday"))
}

func TestMoneyPendingAmount(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
			Flow: flowAdd, Step: stepAmount, Name: "Holiday", BankID: testBankID,
		}), true, nil)
		svc.EXPECT().
			Add(ctx, userID, testBankID, domain.Money(10000)).
			Return(domain.Bank{ID: testBankID, Name: "Holiday", Balance: 15000}, nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("100"))
	require.Contains(t, sent, text.Added("Holiday", "100.00", "150.00"))
}

func TestMoneyPendingAmountInvalid(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newNewBankBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
			Flow: flowSpend, Step: stepAmount, Name: "Gifts", BankID: testBankID,
		}), true, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{
			Flow: flowSpend, Step: stepAmount, Name: "Gifts", BankID: testBankID,
		})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("abc"))
	require.Contains(t, sent, text.InvalidAmount)
}

func TestMoneyCallbackAsksAmount(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 0}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: flowAdd, Step: stepBank}), true, nil)
		svc.EXPECT().Get(ctx, userID, testBankID).Return(holiday, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{
			Flow: flowAdd, Step: stepAmount, Name: "Holiday", BankID: testBankID,
		})
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, includeCallbackUpdate(callbackAddPrefix+"7"))
	require.Contains(t, sent, text.AskAddAmount("Holiday"))
}

func TestCommandAtStartMoney(t *testing.T) {
	require.True(t, commandAtStart("add")(commandUpdate("/add")))
	require.True(t, commandAtStart("add")(commandUpdate("/add Holiday 100")))
	require.True(t, commandAtStart("spend")(commandUpdate("/spend")))
	require.True(t, commandAtStart("set")(commandUpdate("/set Live 0")))
	require.False(t, commandAtStart("add")(commandUpdate("/addfoo")))
}
