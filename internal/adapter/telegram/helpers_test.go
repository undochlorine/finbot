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
	testPromptID            = 1
	testUserMsgID           = 99
	getMeOKBody             = `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"finbot"}}`
	sendMessageOKBody       = `{"ok":true,"result":{"message_id":1,"date":1,"chat":{"id":42,"type":"private"}}}`
	apiOKBody               = `{"ok":true,"result":true}`
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
	expectAPI(t, client, "setMyCommands", apiOKBody, body)
}

func expectSendMessage(t *testing.T, client *mocks.MockHTTPClient, sent *string) {
	t.Helper()
	expectAPI(t, client, "sendMessage", sendMessageOKBody, sent)
}

func expectEditMessage(t *testing.T, client *mocks.MockHTTPClient, sent *string) {
	t.Helper()
	expectAPI(t, client, "editMessageText", sendMessageOKBody, sent)
}

func expectDeleteMessages(t *testing.T, client *mocks.MockHTTPClient) {
	t.Helper()
	expectAPI(t, client, "deleteMessage", apiOKBody, nil)
}

func expectAPI(t *testing.T, client *mocks.MockHTTPClient, path, respBody string, captured *string) {
	t.Helper()
	resp := jsonResponse(http.StatusOK, respBody)
	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close %s body: %v", path, err)
		}
	})

	exp := client.EXPECT().
		Do(mock.MatchedBy(func(req *http.Request) bool {
			return strings.Contains(req.URL.Path, path)
		}))
	if captured != nil {
		exp.Run(func(req *http.Request) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			*captured = string(body)
		})
	}
	exp.Return(resp, nil).Once()
}

func expectAnswerCallbackQuery(t *testing.T, client *mocks.MockHTTPClient) {
	t.Helper()
	expectAPI(t, client, "answerCallbackQuery", apiOKBody, nil)
}

func expectFSMSet(t *testing.T, cache *mocks.MockCache, ctx context.Context, key string, want fsmState) {
	t.Helper()
	cache.EXPECT().
		Set(ctx, key, mock.MatchedBy(func(v []byte) bool {
			var got fsmState
			if json.Unmarshal(v, &got) != nil {
				return false
			}
			if got.Flow != want.Flow || got.Step != want.Step || got.Name != want.Name || got.BankID != want.BankID {
				return false
			}
			if want.PromptID != 0 && got.PromptID != want.PromptID {
				return false
			}
			if want.SweepIDs != nil && !equalInts(got.SweepIDs, want.SweepIDs) {
				return false
			}
			return true
		}), fsmTTL).
		Return(nil)
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

	cache.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, nil).Maybe()

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
	return commandUpdateID(text, 0)
}

func commandUpdateID(text string, msgID int) *models.Update {
	return &models.Update{Message: &models.Message{
		ID:   msgID,
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
				ID:   testPromptID,
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
