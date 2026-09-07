package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestHygieneStoresPromptID(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().List(ctx, userID).Return([]domain.Bank{holiday}, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{
			Flow: domain.CommandAdd, Step: stepBank, PromptID: testPromptID,
		})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("/add"))
	require.Contains(t, sent, text.AskBank)
	require.Contains(t, sent, callbackAddPrefix+"7")
}

func TestHygieneAddCallbackEditsPrompt(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
			Flow: domain.CommandAdd, Step: stepBank, PromptID: testPromptID,
		}), true, nil)
		svc.EXPECT().Get(ctx, userID, testBankID).Return(holiday, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{
			Flow: domain.CommandAdd, Step: stepAmount, Name: "Holiday", BankID: testBankID, PromptID: testPromptID,
		})
		expectAnswerCallbackQuery(t, client)
		expectEditMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, callbackUpdate(callbackAddPrefix+"7"))
	require.Contains(t, sent, text.AskAddAmount("Holiday"))
	require.NotContains(t, sent, callbackAddPrefix+"7")
}

func TestHygieneAddAmountEditsAndSweeps(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
			Flow:     domain.CommandAdd,
			Step:     stepAmount,
			Name:     "Holiday",
			BankID:   testBankID,
			PromptID: testPromptID,
		}), true, nil)
		svc.EXPECT().
			Add(ctx, userID, testBankID, domain.Money(10000)).
			Return(domain.Bank{ID: testBankID, Name: "Holiday", Balance: 15000}, nil)
		expectDeleteMessages(t, client)
		expectEditMessage(t, client, &sent)
		cache.EXPECT().Delete(ctx, key).Return(nil)
	})
	b.ProcessUpdate(ctx, commandUpdateID("100", testUserMsgID))
	require.Contains(t, sent, text.Added("Holiday", "100.00", "150.00"))
	require.Contains(t, sent, `"inline_keyboard":[]`)
}

func TestHygieneCancelDeletesPrompt(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
			Flow: domain.CommandAdd, Step: stepBank, PromptID: testPromptID,
		}), true, nil)
		expectDeleteMessages(t, client)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("/cancel"))
	require.Contains(t, sent, text.Canceled)
}

func TestHygieneKeepsFeedbackBody(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	body := "please add history"
	var sent string
	notify := mocks.NewMockNotifier(t)

	b := newTestBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(ctx, key).Return(mustFSM(t, fsmState{
			Flow: commandFeedback, Step: stepText, PromptID: testPromptID,
		}), true, nil)
		notify.EXPECT().Notify(ctx, testAdminID, text.FeedbackForward(telegramUserID, "alice", body)).Return(nil)
		expectEditMessage(t, client, &sent)
		cache.EXPECT().Delete(ctx, key).Return(nil)
	})
	b.SetAdmin(testAdminID, notify)
	b.ProcessUpdate(ctx, commandUpdateID(body, testUserMsgID))
	require.Contains(t, sent, text.FeedbackThanks)
	require.Contains(t, sent, `"inline_keyboard":[]`)
}

func TestHygieneShortcutDoesNotDeleteCommand(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000}
	var sent string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().GetByName(ctx, userID, "Holiday").Return(holiday, nil)
		svc.EXPECT().
			Add(ctx, userID, testBankID, domain.Money(10000)).
			Return(domain.Bank{ID: testBankID, Name: "Holiday", Balance: 15000}, nil)
		cache.EXPECT().Delete(ctx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdateID("/add Holiday 100", testUserMsgID))
	require.Contains(t, sent, text.Added("Holiday", "100.00", "150.00"))
}
