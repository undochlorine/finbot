package telegram

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestCurrencyChoices(t *testing.T) {
	defaults := []string{domain.CurrencyEUR, domain.CurrencyUSD, domain.CurrencyUSDT}
	tests := []struct {
		name string
		last string
		want []string
	}{
		{name: "missing", want: defaults},
		{name: "invalid", last: "BTC", want: defaults},
		{name: "blank", last: "  ", want: defaults},
		{name: "usd", last: "usd", want: defaults},
		{name: "eur", last: domain.CurrencyEUR, want: defaults},
		{name: "usdt", last: domain.CurrencyUSDT, want: defaults},
		{name: "uah", last: "uah", want: []string{domain.CurrencyEUR, domain.CurrencyUSD, domain.CurrencyUSDT, "UAH"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, currencyChoices(tt.last))
		})
	}
}

func TestNewBankInvalidCurrencyStays(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{
			Flow:     domain.CommandNewBank,
			Step:     stepCurrency,
			Name:     "Holiday",
			Include:  true,
			PromptID: testPromptID,
		}), true, nil)
		svc.EXPECT().LastCurrency(anyCtx, userID).Return(domain.CurrencyUSD, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{
			Flow:     domain.CommandNewBank,
			Step:     stepCurrency,
			Name:     "Holiday",
			Include:  true,
			PromptID: testPromptID,
			SweepIDs: []int{testUserMsgID},
		})
		expectEditMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdateID("btc", testUserMsgID))
	require.Contains(t, sent, text.InvalidCurrency)
	requireCallbackOrder(t, sent, domain.CurrencyEUR, domain.CurrencyUSD, domain.CurrencyUSDT)
	require.NotContains(t, sent, callbackCCYPrefix+"UAH")
}

func TestNewBankCurrencyCreatesAndSweeps(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{
			Flow:     domain.CommandNewBank,
			Step:     stepCurrency,
			Name:     "Holiday",
			Include:  false,
			PromptID: testPromptID,
		}), true, nil)
		svc.EXPECT().
			CreateBank(anyCtx, userID, "Holiday", false, "UAH").
			Return(domain.Bank{Name: "Holiday", Currency: "UAH"}, nil)
		expectDeleteMessages(t, client)
		expectEditMessage(t, client, &sent)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
	})
	b.ProcessUpdate(ctx, commandUpdateID("uah", testUserMsgID))
	require.Contains(t, sent, text.BankCreated("Holiday", "UAH", false))
	require.NotContains(t, sent, "0.00")
}

func TestNewBankCancelOnCurrencyWritesNothing(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{
			Flow: domain.CommandNewBank, Step: stepCurrency, Name: "Holiday", Include: true, PromptID: testPromptID,
		}), true, nil)
		expectDeleteMessages(t, client)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("/cancel"))
	require.Contains(t, sent, text.Canceled)
}

func TestCurrencyCommandFlow(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{
		ID: testBankID, UserID: userID, Name: "Holiday", Balance: 10000, Currency: domain.CurrencyUSD,
	}

	tests := []struct {
		name     string
		text     string
		setup    func(*mocks.MockService, *mocks.MockCache, *mocks.MockHTTPClient, *string)
		wantText string
		order    []string
		noExtra  string
	}{
		{
			name: "empty asks for a bank",
			text: "/currency",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return([]domain.Bank{holiday}, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandCurrency, Step: stepBank})
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskBank,
		},
		{
			name: "named bank opens the picker",
			text: "/currency Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().LastCurrency(anyCtx, userID).Return(domain.CurrencyEUR, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{}.withCurrencyPick(holiday))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskCurrency,
			order:    []string{domain.CurrencyEUR, domain.CurrencyUSD, domain.CurrencyUSDT},
			noExtra:  callbackCCYPrefix + "UAH",
		},
		{
			name: "bank and code skips to confirm",
			text: "/currency Holiday UAH",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday UAH").Return(domain.Bank{}, domain.ErrBankNotFound)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{}.withCurrencyConfirm(holiday, "UAH"))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskCurrencyConfirm("Holiday", "$100.00", "100.00 UAH"),
		},
		{
			name: "bad code is not a bank name",
			text: "/currency Holiday BTC",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday BTC").Return(domain.Bank{}, domain.ErrBankNotFound)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.InvalidCurrency,
			noExtra:  text.UnknownBank("Holiday BTC"),
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
			if len(tt.order) > 0 {
				requireCallbackOrder(t, sent, tt.order...)
			}
			if tt.noExtra != "" {
				require.NotContains(t, sent, tt.noExtra)
			}
			if tt.name == "bank and code skips to confirm" {
				require.Contains(t, sent, callbackCurrencyYes+"7")
				require.Contains(t, sent, callbackCurrencyNo+"7")
				require.NotContains(t, sent, callbackCCYPrefix)
			}
		})
	}
}

func TestCurrencySameCodeWritesNoConfirm(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{
		ID: testBankID, UserID: userID, Name: "Holiday", Balance: 10000, Currency: domain.CurrencyUSD,
	}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{}.withCurrencyPick(holiday)), true, nil)
		svc.EXPECT().Get(anyCtx, userID, testBankID).Return(holiday, nil)
		svc.EXPECT().ChangeCurrency(anyCtx, userID, testBankID, domain.CurrencyUSD).Return(holiday, nil)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, callbackUpdate(callbackCCYPrefix+domain.CurrencyUSD))
	require.Contains(t, sent, text.CurrencyAlready("Holiday", "$"))
	require.NotContains(t, sent, "Confirm?")
}

func TestCurrencyConfirmYesAndNo(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{
		ID: testBankID, UserID: userID, Name: "Holiday", Balance: 10000, Currency: domain.CurrencyUSD,
	}
	changed := holiday
	changed.Currency = "UAH"

	tests := []struct {
		name     string
		data     string
		setup    func(*mocks.MockService)
		wantText string
	}{
		{
			name: "yes writes the new label",
			data: callbackCurrencyYes + "7",
			setup: func(svc *mocks.MockService) {
				svc.EXPECT().ChangeCurrency(anyCtx, userID, testBankID, "UAH").Return(changed, nil)
			},
			wantText: text.CurrencyChanged("Holiday", "UAH", "100.00 UAH"),
		},
		{
			name: "no leaves the currency",
			data: callbackCurrencyNo + "7",
			setup: func(svc *mocks.MockService) {
				svc.EXPECT().Get(anyCtx, userID, testBankID).Return(holiday, nil)
			},
			wantText: text.CurrencyKept("$"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
				cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{}.withCurrencyConfirm(holiday, "UAH")), true, nil)
				tt.setup(svc)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectAnswerCallbackQuery(t, client)
				expectSendMessage(t, client, &sent)
			})
			b.ProcessUpdate(ctx, callbackUpdate(tt.data))
			require.Contains(t, sent, tt.wantText)
		})
	}
}

func TestCommandAtStartCurrency(t *testing.T) {
	require.True(t, commandAtStart(domain.CommandCurrency)(commandUpdate("/"+domain.CommandCurrency)))
	require.True(t, commandAtStart(domain.CommandCurrency)(commandUpdate("/"+domain.CommandCurrency+" Holiday UAH")))
	require.False(t, commandAtStart(domain.CommandCurrency)(commandUpdate("/"+domain.CommandCurrency+"foo")))
}

func requireCallbackOrder(t *testing.T, body string, codes ...string) {
	t.Helper()
	prev := -1
	for _, code := range codes {
		at := strings.Index(body, callbackCCYPrefix+code)
		require.Greater(t, at, prev, code)
		prev = at
	}
}
