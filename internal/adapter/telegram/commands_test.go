package telegram

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestMenuCommandsListsMVPCommands(t *testing.T) {
	want := []struct {
		command     string
		description string
	}{
		{commandStart, text.CmdDescStart},
		{commandHelp, text.CmdDescHelp},
		{domain.CommandNewBank, text.CmdDescNewBank},
		{domain.CommandAdd, text.CmdDescAdd},
		{domain.CommandSpend, text.CmdDescSpend},
		{domain.CommandSet, text.CmdDescSet},
		{domain.CommandDelete, text.CmdDescDelete},
		{domain.CommandBank, text.CmdDescBank},
		{domain.CommandToggle, text.CmdDescToggle},
		{domain.CommandRename, text.CmdDescRename},
		{domain.CommandBanks, text.CmdDescBanks},
		{domain.CommandTotal, text.CmdDescTotal},
		{domain.CommandAll, text.CmdDescAll},
		{commandCancel, text.CmdDescCancel},
		{commandFeedback, text.CmdDescFeedback},
	}

	got := menuCommands()
	require.Len(t, got, len(want))
	for i, cmd := range want {
		require.Equal(t, cmd.command, got[i].Command)
		require.Equal(t, cmd.description, got[i].Description)
		require.NotEmpty(t, got[i].Description)
	}
}

func TestMenuCommandsMatchHelp(t *testing.T) {
	for _, cmd := range menuCommands() {
		require.Contains(t, text.Help, "/"+cmd.Command+" - "+cmd.Description)
	}
}

func TestNewRegistersMenuCommands(t *testing.T) {
	var body string
	client := mocks.NewMockHTTPClient(t)

	getMeResp := jsonResponse(http.StatusOK, getMeOKBody)
	t.Cleanup(func() {
		if err := getMeResp.Body.Close(); err != nil {
			t.Errorf("close getMe body: %v", err)
		}
	})
	client.EXPECT().
		Do(mock.MatchedBy(func(req *http.Request) bool {
			return strings.Contains(req.URL.Path, "getMe")
		})).
		Return(getMeResp, nil).
		Once()
	expectSetMyCommands(t, client, &body)

	_, err := New("123:token", mocks.NewMockService(t), mocks.NewMockCache(t), client)
	require.NoError(t, err)

	for _, cmd := range menuCommands() {
		require.Contains(t, body, `"command":"`+cmd.Command+`"`)
		require.Contains(t, body, `"description":"`+cmd.Description+`"`)
	}
}

func TestNewSetMyCommandsError(t *testing.T) {
	client := expectGetMe(t, http.StatusOK, getMeOKBody)
	resp := jsonResponse(http.StatusBadRequest, `{"ok":false,"error_code":400,"description":"Bad Request"}`)
	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close setMyCommands body: %v", err)
		}
	})
	client.EXPECT().
		Do(mock.MatchedBy(func(req *http.Request) bool {
			return strings.Contains(req.URL.Path, "setMyCommands")
		})).
		Return(resp, nil).
		Once()

	_, err := New("123:token", mocks.NewMockService(t), mocks.NewMockCache(t), client)
	require.Error(t, err)
	require.Contains(t, err.Error(), "register telegram commands")
}
