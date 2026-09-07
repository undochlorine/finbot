package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestViewCommandFlow(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{
		ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true,
	}
	gifts := domain.Bank{
		ID: 8, UserID: userID, Name: "Gifts", Balance: 1250, IncludeInTotal: false,
	}

	tests := []struct {
		name     string
		text     string
		setup    func(*mocks.MockService, *mocks.MockCache, *mocks.MockHTTPClient, *string)
		wantText []string
		wantCB   string
		notText  []string
	}{
		{
			name: "bank no banks",
			text: "/bank",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return(nil, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.NoBanks},
		},
		{
			name: "banks no banks",
			text: "/banks",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return(nil, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.NoBanks},
		},
		{
			name: "all no banks",
			text: "/all",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().All(ctx, userID).Return(nil, 0, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.NoBanks},
		},
		{
			name: "total no banks is zero",
			text: "/total",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().Total(ctx, userID).Return(domain.Money(0), nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.Total("0.00")},
			notText:  []string{text.NoBanks},
		},
		{
			name: "bank picker",
			text: "/bank",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return([]domain.Bank{holiday}, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandBank, Step: stepBank})
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.AskBank},
			wantCB:   callbackBankPrefix + "7",
		},
		{
			name: "bank name shortcut",
			text: "/bank Holiday",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.BankCard("Holiday", "50.00", true)},
		},
		{
			name: "bank spaced name shortcut",
			text: "/bank Holiday Fund",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				fund := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday Fund", Balance: 100, IncludeInTotal: false}
				svc.EXPECT().GetByName(ctx, userID, "Holiday Fund").Return(fund, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.BankCard("Holiday Fund", "1.00", false)},
		},
		{
			name: "bank unknown",
			text: "/bank Missing",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Missing").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.UnknownBank("Missing")},
		},
		{
			name: "bank mention shortcut",
			text: "/bank@finbot Holiday",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.BankCard("Holiday", "50.00", true)},
		},
		{
			name: "banks lists excluded separately from total flag",
			text: "/banks",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(ctx, userID).Return([]domain.Bank{holiday, gifts}, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{
				text.BankCard("Holiday", "50.00", true),
				text.BankCard("Gifts", "12.50", false),
			},
			notText: []string{text.Total("50.00"), text.Total("62.50")},
		},
		{
			name: "total is included banks only",
			text: "/total",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().Total(ctx, userID).Return(holiday.Balance, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{text.Total("50.00")},
			notText:  []string{"12.50", "Gifts"},
		},
		{
			name: "all lists excluded banks but total excludes them",
			text: "/all",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().All(ctx, userID).Return([]domain.Bank{holiday, gifts}, holiday.Balance, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: []string{
				text.BankCard("Holiday", "50.00", true),
				text.BankCard("Gifts", "12.50", false),
				text.Total("50.00"),
			},
			notText: []string{text.Total("62.50")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
				tt.setup(svc, cache, client, &sent)
			})
			b.ProcessUpdate(ctx, commandUpdate(tt.text))
			require.Contains(t, sent, "42")
			for _, want := range tt.wantText {
				require.Contains(t, sent, want)
			}
			if tt.wantCB != "" {
				require.Contains(t, sent, tt.wantCB)
			}
			for _, not := range tt.notText {
				require.NotContains(t, sent, not)
			}
		})
	}
}

func TestBankPendingNameShowsCard(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandBank, Step: stepBank}), true, nil)
		svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday"))
	require.Contains(t, sent, text.BankCard("Holiday", "50.00", true))
}

func TestBankCallbackShowsCard(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true}
	var sent string

	b := newNewBankBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandBank, Step: stepBank}), true, nil)
		svc.EXPECT().Get(ctx, userID, testBankID).Return(holiday, nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, includeCallbackUpdate(callbackBankPrefix+"7"))
	require.Contains(t, sent, text.BankCard("Holiday", "50.00", true))
}

func TestCommandAtStartViews(t *testing.T) {
	tests := []struct {
		cmd  string
		text string
		want bool
	}{
		{cmd: domain.CommandBank, text: "/bank", want: true},
		{cmd: domain.CommandBank, text: "/bank Holiday", want: true},
		{cmd: domain.CommandBank, text: "/bankfoo", want: false},
		{cmd: domain.CommandBanks, text: "/banks", want: true},
		{cmd: domain.CommandTotal, text: "/total", want: true},
		{cmd: domain.CommandAll, text: "/all", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			require.Equal(t, tt.want, commandAtStart(tt.cmd)(commandUpdate(tt.text)))
		})
	}
}
