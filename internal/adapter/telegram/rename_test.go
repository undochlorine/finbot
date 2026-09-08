package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestRenameCommandFlow(t *testing.T) {
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
			text: "/rename",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return(nil, nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.NoBanks,
		},
		{
			name: "picker",
			text: "/rename",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().List(anyCtx, userID).Return([]domain.Bank{holiday}, nil)
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: domain.CommandRename, Step: stepBank})
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskBank,
			wantCB:   []string{callbackRenamePrefix + "7"},
		},
		{
			name: "name shortcut asks new name",
			text: "/rename Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				expectFSMSet(t, cache, ctx, key, renameNameState(holiday))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskRenameName("Holiday"),
		},
		{
			name: "spaced name shortcut",
			text: "/rename Holiday Fund",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				fund := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday Fund"}
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday Fund").Return(fund, nil)
				expectFSMSet(t, cache, ctx, key, renameNameState(fund))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskRenameName("Holiday Fund"),
		},
		{
			name: "first-word fallback renames immediately",
			text: "/rename Holiday Trips",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday Trips").Return(domain.Bank{}, domain.ErrBankNotFound)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Trips").Return(domain.Bank{}, domain.ErrBankNotFound)
				svc.EXPECT().Rename(anyCtx, userID, testBankID, "Trips").Return(
					domain.Bank{ID: testBankID, Name: "Trips", Balance: 5000}, nil,
				)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.BankRenamed("Holiday", "Trips"),
		},
		{
			name: "first-word fallback keeps rest as new name",
			text: "/rename Holiday Fund Extra",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday Fund Extra").Return(domain.Bank{}, domain.ErrBankNotFound)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Fund Extra").Return(domain.Bank{}, domain.ErrBankNotFound)
				svc.EXPECT().Rename(anyCtx, userID, testBankID, "Fund Extra").Return(
					domain.Bank{ID: testBankID, Name: "Fund Extra", Balance: 5000}, nil,
				)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.BankRenamed("Holiday", "Fund Extra"),
		},
		{
			name: "first-word fallback duplicate stays on name step",
			text: "/rename Holiday Gifts",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts"}
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday Gifts").Return(domain.Bank{}, domain.ErrBankNotFound)
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				svc.EXPECT().GetByName(anyCtx, userID, "Gifts").Return(gifts, nil)
				expectFSMSet(t, cache, ctx, key, renameNameState(holiday))
				expectSendMessage(t, client, sent)
			},
			wantText: text.BankNameTaken("Gifts"),
		},
		{
			name: "unknown bank",
			text: "/rename Missing",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Missing").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: text.UnknownBank("Missing"),
		},
		{
			name: "unknown full name and first word",
			text: "/rename Missing Rest",
			setup: func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Missing Rest").Return(domain.Bank{}, domain.ErrBankNotFound)
				svc.EXPECT().GetByName(anyCtx, userID, "Missing").Return(domain.Bank{}, domain.ErrBankNotFound)
				expectSendMessage(t, client, sent)
			},
			wantText: text.UnknownBank("Missing Rest"),
		},
		{
			name: "mention shortcut",
			text: "/rename@finbot Holiday",
			setup: func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient, sent *string) {
				svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
				expectFSMSet(t, cache, ctx, key, renameNameState(holiday))
				expectSendMessage(t, client, sent)
			},
			wantText: text.AskRenameName("Holiday"),
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

func TestRenamePendingBankName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandRename, Step: stepBank}), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(holiday, nil)
		expectFSMSet(t, cache, ctx, key, renameNameState(holiday))
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday"))
	require.Contains(t, sent, text.AskRenameName("Holiday"))
}

func TestRenamePendingBankNameKeepsFullText(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandRename, Step: stepBank}), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday Fund").Return(domain.Bank{}, domain.ErrBankNotFound)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Holiday Fund"))
	require.Contains(t, sent, text.UnknownBank("Holiday Fund"))
}

func TestRenamePendingNewName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, renameNameState(holiday)), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Trips").Return(domain.Bank{}, domain.ErrBankNotFound)
		svc.EXPECT().Rename(anyCtx, userID, testBankID, "Trips").Return(
			domain.Bank{ID: testBankID, Name: "Trips", Balance: 5000}, nil,
		)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Trips"))
	require.Contains(t, sent, text.BankRenamed("Holiday", "Trips"))
}

func TestRenamePendingRecase(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, renameNameState(holiday)), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "holiday").Return(holiday, nil)
		svc.EXPECT().Rename(anyCtx, userID, testBankID, "holiday").Return(
			domain.Bank{ID: testBankID, Name: "holiday", Balance: 5000}, nil,
		)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("holiday"))
	require.Contains(t, sent, text.BankRenamed("Holiday", "holiday"))
}

func TestRenamePendingDuplicateName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	gifts := domain.Bank{ID: 8, UserID: userID, Name: "Gifts"}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, renameNameState(holiday)), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Gifts").Return(gifts, nil)
		expectFSMSet(t, cache, ctx, key, renameNameState(holiday))
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("Gifts"))
	require.Contains(t, sent, text.BankNameTaken("Gifts"))
}

func TestRenamePendingEmptyName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newTestBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, renameNameState(holiday)), true, nil)
		expectFSMSet(t, cache, ctx, key, renameNameState(holiday))
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("   "))
	require.Contains(t, sent, text.AskRenameName("Holiday"))
}

func TestRenameCallbackAsksName(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{Flow: domain.CommandRename, Step: stepBank}), true, nil)
		svc.EXPECT().Get(anyCtx, userID, testBankID).Return(holiday, nil)
		expectFSMSet(t, cache, ctx, key, renameNameState(holiday))
		expectAnswerCallbackQuery(t, client)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, callbackUpdate(callbackRenamePrefix+"7"))
	require.Contains(t, sent, text.AskRenameName("Holiday"))
}

func TestRenameHygieneEditsAndSweeps(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{
			Flow:     domain.CommandRename,
			Step:     stepName,
			Name:     holiday.Name,
			BankID:   holiday.ID,
			PromptID: testPromptID,
		}), true, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Trips").Return(domain.Bank{}, domain.ErrBankNotFound)
		svc.EXPECT().Rename(anyCtx, userID, testBankID, "Trips").Return(
			domain.Bank{ID: testBankID, Name: "Trips", Balance: 5000}, nil,
		)
		expectDeleteMessages(t, client)
		expectEditMessage(t, client, &sent)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
	})
	b.ProcessUpdate(ctx, commandUpdateID("Trips", testUserMsgID))
	require.Contains(t, sent, text.BankRenamed("Holiday", "Trips"))
	require.Contains(t, sent, `"inline_keyboard":[]`)
}

func TestCommandAtStartRename(t *testing.T) {
	require.True(t, commandAtStart(domain.CommandRename)(commandUpdate("/"+domain.CommandRename)))
	require.True(t, commandAtStart(domain.CommandRename)(commandUpdate("/"+domain.CommandRename+" Holiday")))
	require.False(t, commandAtStart(domain.CommandRename)(commandUpdate("/"+domain.CommandRename+"foo")))
}
