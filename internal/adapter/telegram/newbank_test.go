package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestCommandPayload(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{name: "empty", want: ""},
		{name: "command only", msg: "/newbank", want: ""},
		{name: "args", msg: "/newbank Holiday yes", want: "Holiday yes"},
		{name: "mention args", msg: "/newbank@finbot Holiday", want: "Holiday"},
		{name: "trims", msg: "/newbank   Holiday  ", want: "Holiday"},
		{name: "plain text", msg: "Holiday", want: "Holiday"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, commandPayload(tt.msg))
		})
	}
}

func TestNewBankFlow(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)

	tests := []struct {
		name     string
		text     string
		setup    func(*mocks.MockService, *mocks.MockCache, *mocks.MockHTTPClient, *string)
		wantText string
		wantKB   bool
	}{
		{
			name: "asks for name",
			text: "/newbank",
			setup: func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepName})
				expectSendMessage(t, client, sent)
			},
			wantText: text.NewBankAskName,
		},
		{
			name: "name shortcut asks include",
			text: "/newbank Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				expectNameAvailable(svc, ctx, userID, "Holiday")
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepInclude, Name: "Holiday"})
				expectSendMessage(t, client, sent)
			},
			wantText: text.NewBankAskInclude,
			wantKB:   true,
		},
		{
			name: "yes after name stays in the name",
			text: "/newbank Holiday yes",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				expectNameAvailable(svc, ctx, userID, "Holiday yes")
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepInclude, Name: "Holiday yes"})
				expectSendMessage(t, client, sent)
			},
			wantText: text.NewBankAskInclude,
			wantKB:   true,
		},
		{
			name: "no after name stays in the name",
			text: "/newbank Gifts no",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				expectNameAvailable(svc, ctx, userID, "Gifts no")
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepInclude, Name: "Gifts no"})
				expectSendMessage(t, client, sent)
			},
			wantText: text.NewBankAskInclude,
			wantKB:   true,
		},
		{
			name: "spaced name is kept",
			text: "/newbank My Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				expectNameAvailable(svc, ctx, userID, "My Holiday")
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepInclude, Name: "My Holiday"})
				expectSendMessage(t, client, sent)
			},
			wantText: text.NewBankAskInclude,
			wantKB:   true,
		},
		{
			name: "yes alone is the bank name",
			text: "/newbank " + domain.Yes,
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				expectNameAvailable(svc, ctx, userID, domain.Yes)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepInclude, Name: domain.Yes})
				expectSendMessage(t, client, sent)
			},
			wantText: text.NewBankAskInclude,
			wantKB:   true,
		},
		{
			name: "duplicate name as soon as it is entered",
			text: "/newbank Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().
					GetByName(ctx, userID, "Holiday").
					Return(domain.Bank{Name: "Holiday"}, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepName})
				expectSendMessage(t, client, sent)
			},
			wantText: text.BankNameTaken("Holiday"),
		},
		{
			name: "empty name after command",
			text: "/newbank    ",
			setup: func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepName})
				expectSendMessage(t, client, sent)
			},
			wantText: text.NewBankAskName,
		},
		{
			name: "mention shortcut",
			text: "/newbank@finbot Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				expectNameAvailable(svc, ctx, userID, "Holiday")
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepInclude, Name: "Holiday"})
				expectSendMessage(t, client, sent)
			},
			wantText: text.NewBankAskInclude,
			wantKB:   true,
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
			if tt.wantKB {
				require.Contains(t, sent, callbackNewBankIncludeYes)
				require.Contains(t, sent, callbackNewBankIncludeNo)
			}
		})
	}
}

func TestNewBankPendingName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandNewBank, Step: stepName}), true, nil)
		expectNameAvailable(svc, ctx, userID, "Holiday")
		expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepInclude, Name: "Holiday"})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday"))
	require.Contains(t, sent, text.NewBankAskInclude)
	require.Contains(t, sent, callbackNewBankIncludeYes)
}

func TestNewBankPendingNameInvalid(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandNewBank, Step: stepName}), true, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepName})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("   "))
	require.Contains(t, sent, text.InvalidBankName)
}

func TestNewBankPendingNameTaken(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandNewBank, Step: stepName}), true, nil)
		svc.EXPECT().
			GetByName(ctx, userID, "Holiday").
			Return(domain.Bank{Name: "Holiday"}, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepName})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday"))
	require.Contains(t, sent, text.BankNameTaken("Holiday"))
	require.NotContains(t, sent, callbackNewBankIncludeYes)
}

func TestNewBankPendingNameKeepsSpacesAndYes(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandNewBank, Step: stepName}), true, nil)
		expectNameAvailable(svc, ctx, userID, "Holiday yes")
		expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandNewBank, Step: stepInclude, Name: "Holiday yes"})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday yes"))
	require.Contains(t, sent, text.NewBankAskInclude)
}

func TestNewBankPendingIncludeText(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
			Flow: domain.CommandNewBank, Step: stepInclude, Name: "Holiday",
		}), true, nil)
		svc.EXPECT().
			CreateBank(ctx, userID, "Holiday", false).
			Return(domain.Bank{ID: 1, UserID: userID, Name: "Holiday"}, nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate(domain.No))
	require.Contains(t, sent, text.BankCreated("Holiday", false))
}

func TestNewBankCallbackCreatesBank(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
			Flow: domain.CommandNewBank, Step: stepInclude, Name: "Holiday",
		}), true, nil)
		svc.EXPECT().
			CreateBank(ctx, userID, "Holiday", true).
			Return(domain.Bank{ID: 1, UserID: userID, Name: "Holiday", IncludeInTotal: true}, nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, callbackUpdate(callbackNewBankIncludeYes))
	require.Contains(t, sent, text.BankCreated("Holiday", true))
}

func TestNewBankCallbackDuplicate(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
			Flow: domain.CommandNewBank, Step: stepInclude, Name: "Holiday",
		}), true, nil)
		svc.EXPECT().
			CreateBank(ctx, userID, "Holiday", true).
			Return(domain.Bank{}, domain.ErrBankNameTaken)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, callbackUpdate(callbackNewBankIncludeYes))
	require.Contains(t, sent, text.BankNameTaken("Holiday"))
}

func TestCommandAtStartNewBank(t *testing.T) {
	require.True(t, commandAtStart(domain.CommandNewBank)(commandUpdate("/"+domain.CommandNewBank)))
	require.True(t, commandAtStart(domain.CommandNewBank)(commandUpdate("/"+domain.CommandNewBank+" Holiday "+domain.Yes)))
	require.False(t, commandAtStart(domain.CommandNewBank)(commandUpdate("/"+domain.CommandNewBank+"foo")))
}
