package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
)

const (
	telegramUserID    int64 = 42
	testBankID        int64 = 7
	getMeOKBody             = `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"finbot"}}`
	sendMessageOKBody       = `{"ok":true,"result":{"message_id":1,"date":1,"chat":{"id":42,"type":"private"}}}`
)

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Header:     make(http.Header),
	}
}

func expectGetMe(t *testing.T, status int, body string) *mocks.MockHTTPClient {
	t.Helper()
	resp := jsonResponse(status, body)
	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close getMe body: %v", err)
		}
	})

	client := mocks.NewMockHTTPClient(t)
	client.EXPECT().
		Do(mock.MatchedBy(func(req *http.Request) bool {
			return strings.Contains(req.URL.Path, "getMe")
		})).
		Return(resp, nil).
		Once()
	return client
}

func expectSetMyCommands(t *testing.T, client *mocks.MockHTTPClient, body *string) {
	t.Helper()
	resp := jsonResponse(http.StatusOK, `{"ok":true,"result":true}`)
	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close setMyCommands body: %v", err)
		}
	})

	exp := client.EXPECT().
		Do(mock.MatchedBy(func(req *http.Request) bool {
			return strings.Contains(req.URL.Path, "setMyCommands")
		}))
	if body != nil {
		exp.Run(func(req *http.Request) {
			b, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			*body = string(b)
		})
	}
	exp.Return(resp, nil).Once()
}

func expectSendMessage(t *testing.T, client *mocks.MockHTTPClient, sent *string) {
	t.Helper()
	resp := jsonResponse(http.StatusOK, sendMessageOKBody)
	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close sendMessage body: %v", err)
		}
	})

	client.EXPECT().
		Do(mock.MatchedBy(func(req *http.Request) bool {
			return strings.Contains(req.URL.Path, "sendMessage")
		})).
		Run(func(req *http.Request) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			*sent = string(body)
		}).
		Return(resp, nil).
		Once()
}

func expectAnswerCallbackQuery(t *testing.T, client *mocks.MockHTTPClient) {
	t.Helper()
	resp := jsonResponse(http.StatusOK, `{"ok":true,"result":true}`)
	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close answerCallbackQuery body: %v", err)
		}
	})
	client.EXPECT().
		Do(mock.MatchedBy(func(req *http.Request) bool {
			return strings.Contains(req.URL.Path, "answerCallbackQuery")
		})).
		Return(resp, nil).
		Once()
}

func expectFSMSet(t *testing.T, cache *mocks.MockCache, ctx context.Context, key string, want fsmState) {
	t.Helper()
	cache.EXPECT().
		Set(ctx, key, mock.MatchedBy(func(v []byte) bool {
			var got fsmState
			return json.Unmarshal(v, &got) == nil && got == want
		}), fsmTTL).
		Return(nil)
}

func expectNameAvailable(svc *mocks.MockService, ctx context.Context, userID domain.UserID, name string) {
	svc.EXPECT().GetByName(ctx, userID, name).Return(domain.Bank{}, domain.ErrBankNotFound)
}

func newTestBot(
	t *testing.T,
	ctx context.Context,
	setup func(*mocks.MockService, *mocks.MockCache, *mocks.MockHTTPClient),
) *Bot {
	t.Helper()

	svc := mocks.NewMockService(t)
	svc.EXPECT().
		UpsertUser(ctx, domain.UserID(telegramUserID), "alice").
		Return(domain.User{TelegramID: domain.UserID(telegramUserID), Username: "alice"}, nil).
		Maybe()

	cache := mocks.NewMockCache(t)
	client := expectGetMe(t, http.StatusOK, getMeOKBody)
	expectSetMyCommands(t, client, nil)
	setup(svc, cache, client)

	b, err := New("123:token", svc, cache, client, bot.WithNotAsyncHandlers())
	require.NoError(t, err)
	return b
}

func newCommandBot(t *testing.T, ctx context.Context, wantSend bool, sent *string) *Bot {
	t.Helper()
	return newTestBot(t, ctx, func(_ *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient) {
		if wantSend {
			expectSendMessage(t, client, sent)
		}
	})
}

func commandUpdate(text string) *models.Update {
	return &models.Update{Message: &models.Message{
		From: &models.User{ID: telegramUserID, Username: "alice"},
		Chat: models.Chat{ID: telegramUserID, Type: "private"},
		Text: text,
	}}
}

func callbackUpdate(data string) *models.Update {
	return &models.Update{CallbackQuery: &models.CallbackQuery{
		ID:   "cb1",
		From: models.User{ID: telegramUserID, Username: "alice"},
		Data: data,
		Message: models.MaybeInaccessibleMessage{
			Type: models.MaybeInaccessibleMessageTypeMessage,
			Message: &models.Message{
				Chat: models.Chat{ID: telegramUserID, Type: "private"},
			},
		},
	}}
}

func mustFSM(t *testing.T, st fsmState) []byte {
	t.Helper()
	raw, err := json.Marshal(st)
	require.NoError(t, err)
	return raw
}
