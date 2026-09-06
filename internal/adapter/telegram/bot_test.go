package telegram

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/service"
)

const getMeOKBody = `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"finbot"}}`

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

func TestNew(t *testing.T) {
	svc := service.New(nil, nil, nil, 0)

	tests := []struct {
		name    string
		token   string
		svc     *service.Service
		client  HTTPClient
		wantErr string
	}{
		{name: "empty token", token: "", svc: svc, wantErr: "telegram bot token is required"},
		{name: "whitespace token", token: " \t", svc: svc, wantErr: "telegram bot token is required"},
		{name: "nil service", token: "123:token", wantErr: "service is required"},
		{name: "nil http client", token: "123:token", svc: svc, wantErr: "http client is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.token, tt.svc, tt.client)
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestNewGetMe(t *testing.T) {
	svc := service.New(nil, nil, nil, 0)

	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "valid token",
			status: http.StatusOK,
			body:   getMeOKBody,
		},
		{
			name:    "unauthorized token",
			status:  http.StatusUnauthorized,
			body:    `{"ok":false,"error_code":401,"description":"Unauthorized"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New("123:token", svc, expectGetMe(t, tt.status, tt.body))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStartReturnsWhenContextCanceled(t *testing.T) {
	b, err := New("123:token", service.New(nil, nil, nil, 0), expectGetMe(t, http.StatusOK, getMeOKBody))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		b.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Start did not return after context cancel")
	}
}
