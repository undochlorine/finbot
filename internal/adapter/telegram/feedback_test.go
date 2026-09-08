package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

const testAdminID domain.UserID = 99

func TestFeedbackUnavailableWhenAdminUnset(t *testing.T) {
	ctx := context.Background()
	var sent string
	b := newCommandBot(t, ctx, true, &sent)
	b.ProcessUpdate(ctx, commandUpdate("/feedback"))
	require.Contains(t, sent, text.FeedbackUnavailable)
}

func TestFeedbackAskAndShortcut(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	body := "please add history"
	forward := text.FeedbackForward(telegramUserID, "alice", body)

	tests := []struct {
		name     string
		text     string
		setup    func(*testing.T, *mocks.MockCache, *mocks.MockHTTPClient, *mocks.MockNotifier, *string)
		wantText string
	}{
		{
			name: "asks for text",
			text: "/feedback",
			setup: func(t *testing.T, cache *mocks.MockCache, client *mocks.MockHTTPClient, _ *mocks.MockNotifier, sent *string) {
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: commandFeedback, Step: stepText})
				expectSendMessage(t, client, sent)
			},
			wantText: text.FeedbackAsk,
		},
		{
			name: "mention asks for text",
			text: "/feedback@finbot",
			setup: func(t *testing.T, cache *mocks.MockCache, client *mocks.MockHTTPClient, _ *mocks.MockNotifier, sent *string) {
				expectFSMSet(t, cache, ctx, key, fsmState{Flow: commandFeedback, Step: stepText})
				expectSendMessage(t, client, sent)
			},
			wantText: text.FeedbackAsk,
		},
		{
			name: "shortcut forwards payload",
			text: "/feedback " + body,
			setup: func(_ *testing.T, cache *mocks.MockCache, client *mocks.MockHTTPClient, notify *mocks.MockNotifier, sent *string) {
				notify.EXPECT().Notify(anyCtx, testAdminID, forward).Return(nil)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectSendMessage(t, client, sent)
			},
			wantText: text.FeedbackThanks,
		},
		{
			name: "notify error",
			text: "/feedback " + body,
			setup: func(_ *testing.T, _ *mocks.MockCache, client *mocks.MockHTTPClient, notify *mocks.MockNotifier, sent *string) {
				notify.EXPECT().Notify(anyCtx, testAdminID, forward).Return(errors.New("boom"))
				expectSendMessage(t, client, sent)
			},
			wantText: text.SomethingWentWrong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			notify := mocks.NewMockNotifier(t)
			b := newTestBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
				tt.setup(t, cache, client, notify, &sent)
			})
			b.SetAdmin(testAdminID, notify)
			b.ProcessUpdate(ctx, commandUpdate(tt.text))
			require.Contains(t, sent, tt.wantText)
		})
	}
}

func TestFeedbackPendingText(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	body := "please add history"
	var sent string
	notify := mocks.NewMockNotifier(t)

	b := newTestBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{Flow: commandFeedback, Step: stepText}), true, nil)
		notify.EXPECT().Notify(anyCtx, testAdminID, text.FeedbackForward(telegramUserID, "alice", body)).Return(nil)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.SetAdmin(testAdminID, notify)
	b.ProcessUpdate(ctx, commandUpdate(body))
	require.Contains(t, sent, text.FeedbackThanks)
}

func TestFeedbackPendingEmptyReasks(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string
	notify := mocks.NewMockNotifier(t)

	b := newTestBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{Flow: commandFeedback, Step: stepText}), true, nil)
		expectFSMSet(t, cache, ctx, key, fsmState{Flow: commandFeedback, Step: stepText})
		expectSendMessage(t, client, &sent)
	})
	b.SetAdmin(testAdminID, notify)
	b.ProcessUpdate(ctx, commandUpdate("   "))
	require.Contains(t, sent, text.FeedbackAsk)
}

func TestFeedbackForwardWithoutUsername(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	body := "please add history"
	var sent string
	notify := mocks.NewMockNotifier(t)

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().UpsertUser(anyCtx, userID, "").Return(domain.User{TelegramID: userID}, nil)
		notify.EXPECT().Notify(anyCtx, testAdminID, text.FeedbackForward(telegramUserID, "", body)).Return(nil)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, &sent)
	})
	b.SetAdmin(testAdminID, notify)
	b.ProcessUpdate(ctx, &models.Update{Message: &models.Message{
		From: &models.User{ID: telegramUserID},
		Chat: models.Chat{ID: telegramUserID, Type: "private"},
		Text: "/feedback " + body,
	}})
	require.Contains(t, sent, text.FeedbackThanks)
}

func TestFeedbackUnavailableWithPayload(t *testing.T) {
	ctx := context.Background()
	var sent string
	b := newCommandBot(t, ctx, true, &sent)
	b.ProcessUpdate(ctx, commandUpdate("/feedback please add history"))
	require.Contains(t, sent, text.FeedbackUnavailable)
}

func TestCommandAtStartFeedback(t *testing.T) {
	require.True(t, commandAtStart(commandFeedback)(commandUpdate("/"+commandFeedback)))
	require.True(t, commandAtStart(commandFeedback)(commandUpdate("/"+commandFeedback+"@finbot")))
	require.True(t, commandAtStart(commandFeedback)(commandUpdate("/"+commandFeedback+" hello")))
	require.False(t, commandAtStart(commandFeedback)(commandUpdate("/"+commandFeedback+"foo")))
}

func TestBotNotifySendsMessage(t *testing.T) {
	ctx := context.Background()
	var sent string
	b := newTestBot(t, ctx, func(_ *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient) {
		expectSendMessage(t, client, &sent)
	})
	require.NoError(t, b.Notify(ctx, testAdminID, "hello admin"))
	require.Contains(t, sent, "hello admin")
	require.Contains(t, sent, "99")
}
