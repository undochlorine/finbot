package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestStartAsksLanguageWhenUnset(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBotUser(t, ctx, domain.User{
		TelegramID: userID,
		Username:   "alice",
	}, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		expectFSMSet(t, cache, ctx, key, fsmState{Flow: commandStart, Step: stepPick})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("/start"))
	require.Contains(t, sent, text.AskLanguage)
	for _, opt := range text.LanguageOptions() {
		require.Contains(t, sent, opt.Label)
		require.Contains(t, sent, callbackLanguagePrefix+opt.Code)
	}
	require.NotContains(t, sent, text.Start)
}

func TestStartWelcomeWhenLocaleSet(t *testing.T) {
	ctx := context.Background()
	var sent string
	b := newTestBotUser(t, ctx, domain.User{
		TelegramID: domain.UserID(telegramUserID),
		Username:   "alice",
		Locale:     domain.LocaleRU,
	}, func(_ *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient) {
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("/start"))
	require.Contains(t, sent, text.For(domain.LocaleRU).Start)
	require.NotContains(t, sent, callbackLanguagePrefix)
}

func TestLanguageCommandOffersButtons(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	var sent string

	b := newTestBot(t, ctx, func(_ *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		expectFSMSet(t, cache, ctx, key, fsmState{Flow: commandLanguage, Step: stepPick})
		expectSendMessage(t, client, &sent)
	})
	b.ProcessUpdate(ctx, commandUpdate("/language"))
	require.Contains(t, sent, text.AskLanguage)
	require.Contains(t, sent, `"text":"English"`)
	require.Contains(t, sent, `"text":"Русский"`)
	require.Contains(t, sent, `"text":"Українська"`)
	require.Contains(t, sent, `"text":"Moldovenească"`)
}

func TestLanguageCallbackSetsLocale(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)

	tests := []struct {
		name     string
		flow     string
		locale   string
		wantText string
	}{
		{
			name:     "start then english welcome",
			flow:     commandStart,
			locale:   domain.LocaleEN,
			wantText: text.Start,
		},
		{
			name:     "language then russian confirm",
			flow:     commandLanguage,
			locale:   domain.LocaleRU,
			wantText: text.For(domain.LocaleRU).LanguageSetTo(domain.LocaleRU),
		},
		{
			name:     "language then ukrainian confirm",
			flow:     commandLanguage,
			locale:   domain.LocaleUK,
			wantText: text.For(domain.LocaleUK).LanguageSetTo(domain.LocaleUK),
		},
		{
			name:     "language then moldavian confirm",
			flow:     commandLanguage,
			locale:   domain.LocaleMD,
			wantText: text.For(domain.LocaleMD).LanguageSetTo(domain.LocaleMD),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sent string
			b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
				cache.EXPECT().Get(anyCtx, key).Return(mustFSM(t, fsmState{
					Flow: tt.flow, Step: stepPick, PromptID: testPromptID,
				}), true, nil)
				svc.EXPECT().SetLocale(anyCtx, userID, tt.locale).Return(nil)
				cache.EXPECT().Delete(anyCtx, key).Return(nil)
				expectAnswerCallbackQuery(t, client)
				expectEditMessage(t, client, &sent)
			})
			b.ProcessUpdate(ctx, callbackUpdate(callbackLanguagePrefix+tt.locale))
			require.Contains(t, sent, tt.wantText)
		})
	}
}

func TestLanguageCallbackInvalidIgnored(t *testing.T) {
	ctx := context.Background()
	var sent string
	var svc *mocks.MockService
	b := newTestBot(t, ctx, func(s *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc = s
		expectAnswerCallbackQuery(t, client)
	})
	b.ProcessUpdate(ctx, callbackUpdate(callbackLanguagePrefix+"fr"))
	require.Empty(t, sent)
	require.True(t, svc.AssertNotCalled(t, "SetLocale"))
}

func TestParseYesNoUsesCatalog(t *testing.T) {
	ctx := withLocale(context.Background(), domain.LocaleRU)
	got, ok := parseYesNo(ctx, "Да")
	require.True(t, ok)
	require.True(t, got)
	got, ok = parseYesNo(ctx, "нет")
	require.True(t, ok)
	require.False(t, got)
	_, ok = parseYesNo(ctx, "maybe")
	require.False(t, ok)
}
