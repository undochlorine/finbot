package telegram

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompact(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "consecutive help keeps first", in: []string{"/help", "/help", "/help"}, want: []string{"/help"}},
		{name: "consecutive start keeps first", in: []string{"/start", "/start"}, want: []string{"/start"}},
		{name: "consecutive banks keeps first", in: []string{"/banks", "/banks"}, want: []string{"/banks"}},
		{name: "consecutive total keeps first", in: []string{"/total", "/total"}, want: []string{"/total"}},
		{name: "consecutive all keeps first", in: []string{"/all", "/all"}, want: []string{"/all"}},
		{name: "consecutive bank name keeps first", in: []string{"/bank Holiday", "/bank Holiday"}, want: []string{"/bank Holiday"}},
		{name: "bank name is case-insensitive", in: []string{"/bank Holiday", "/bank holiday"}, want: []string{"/bank Holiday"}},
		{name: "help at-mention matches help", in: []string{"/help@finbot", "/help"}, want: []string{"/help@finbot"}},
		{name: "help case-insensitive", in: []string{"/HELP", "/help"}, want: []string{"/HELP"}},

		{name: "consecutive empty newbank keeps first", in: []string{"/newbank", "/newbank"}, want: []string{"/newbank"}},
		{name: "consecutive empty add keeps first", in: []string{"/add", "/add"}, want: []string{"/add"}},
		{name: "consecutive empty spend keeps first", in: []string{"/spend", "/spend"}, want: []string{"/spend"}},
		{name: "consecutive empty set keeps first", in: []string{"/set", "/set"}, want: []string{"/set"}},
		{name: "consecutive empty delete keeps first", in: []string{"/delete", "/delete"}, want: []string{"/delete"}},
		{name: "consecutive empty toggle keeps first", in: []string{"/toggle", "/toggle"}, want: []string{"/toggle"}},
		{name: "consecutive empty bank keeps first", in: []string{"/bank", "/bank"}, want: []string{"/bank"}},
		{name: "consecutive empty rename keeps first", in: []string{"/rename", "/rename"}, want: []string{"/rename"}},
		{name: "consecutive empty transfer keeps first", in: []string{"/transfer", "/transfer"}, want: []string{"/transfer"}},
		{name: "consecutive empty feedback keeps first", in: []string{"/feedback", "/feedback"}, want: []string{"/feedback"}},
		{name: "consecutive empty cancel keeps first", in: []string{"/cancel", "/cancel"}, want: []string{"/cancel"}},

		{name: "empty add then spend keeps later flow", in: []string{"/add", "/spend"}, want: []string{"/spend"}},
		{name: "empty transfer then add keeps later flow", in: []string{"/transfer", "/add"}, want: []string{"/add"}},
		{name: "empty wizard chain keeps last flow", in: []string{"/add", "/spend", "/set"}, want: []string{"/set"}},
		{name: "empty newbank then named newbank keeps named", in: []string{"/newbank", "/newbank Holiday"}, want: []string{"/newbank Holiday"}},
		{name: "empty add then money shortcut keeps shortcut", in: []string{"/add", "/add Holiday 100"}, want: []string{"/add Holiday 100"}},
		{name: "empty add then bank name keeps bank", in: []string{"/add", "/bank Holiday"}, want: []string{"/bank Holiday"}},

		{name: "empty newbank then cancel keeps both", in: []string{"/newbank", "/cancel"}, want: []string{"/newbank", "/cancel"}},
		{name: "empty add then help keeps both", in: []string{"/add", "/help"}, want: []string{"/add", "/help"}},
		{name: "empty add then start keeps both", in: []string{"/add", "/start"}, want: []string{"/add", "/start"}},
		{name: "empty add then banks keeps both", in: []string{"/add", "/banks"}, want: []string{"/add", "/banks"}},
		{name: "empty add then total keeps both", in: []string{"/add", "/total"}, want: []string{"/add", "/total"}},
		{name: "empty add then all keeps both", in: []string{"/add", "/all"}, want: []string{"/add", "/all"}},
		{name: "empty add then typed keeps both", in: []string{"/add", "typed:42.00"}, want: []string{"/add", "typed:42.00"}},
		{name: "empty add then callback keeps both", in: []string{"/add", "callback:v1:add:7"}, want: []string{"/add", "callback:v1:add:7"}},

		{name: "consecutive newbank same name keeps first", in: []string{"/newbank Holiday", "/newbank Holiday"}, want: []string{"/newbank Holiday"}},
		{name: "newbank same name case-insensitive", in: []string{"/newbank Holiday", "/newbank holiday"}, want: []string{"/newbank Holiday"}},
		{name: "consecutive newbank different names keeps both", in: []string{"/newbank Holiday", "/newbank Gifts"}, want: []string{"/newbank Holiday", "/newbank Gifts"}},
		{name: "consecutive delete same name keeps first", in: []string{"/delete Holiday", "/delete Holiday"}, want: []string{"/delete Holiday"}},
		{name: "consecutive delete different names keeps both", in: []string{"/delete Holiday", "/delete Gifts"}, want: []string{"/delete Holiday", "/delete Gifts"}},

		{name: "identical add shortcuts both kept", in: []string{"/add Holiday 100", "/add Holiday 100"}, want: []string{"/add Holiday 100", "/add Holiday 100"}},
		{name: "identical transfer shortcuts both kept", in: []string{"/transfer Holiday Gifts 50", "/transfer Holiday Gifts 50"}, want: []string{"/transfer Holiday Gifts 50", "/transfer Holiday Gifts 50"}},
		{name: "identical spend shortcuts both kept", in: []string{"/spend Gifts 12.50", "/spend Gifts 12.50"}, want: []string{"/spend Gifts 12.50", "/spend Gifts 12.50"}},
		{name: "identical set shortcuts both kept", in: []string{"/set Live 0", "/set Live 0"}, want: []string{"/set Live 0", "/set Live 0"}},
		{name: "non-consecutive help keeps both", in: []string{"/help", "/banks", "/help"}, want: []string{"/help", "/banks", "/help"}},
		{name: "feedback bodies both kept", in: []string{"/feedback hello", "/feedback hello"}, want: []string{"/feedback hello", "/feedback hello"}},
		{name: "toggle shortcuts both kept", in: []string{"/toggle Holiday", "/toggle Holiday"}, want: []string{"/toggle Holiday", "/toggle Holiday"}},
		{name: "help then banks keeps order", in: []string{"/help", "/banks"}, want: []string{"/help", "/banks"}},
		{name: "burst help all help", in: []string{"/help", "/help", "/help", "/all", "/help"}, want: []string{"/help", "/all", "/help"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compact(jobsFromTokens(tt.in))
			require.Equal(t, tt.want, tokensFromJobs(got))
		})
	}
}

func jobsFromTokens(tokens []string) []pendingJob {
	jobs := make([]pendingJob, 0, len(tokens))
	for _, tok := range tokens {
		jobs = append(jobs, jobFromToken(tok))
	}
	return jobs
}

func jobFromToken(tok string) pendingJob {
	switch {
	case strings.HasPrefix(tok, "callback:"):
		return pendingJob{update: callbackUpdate(strings.TrimPrefix(tok, "callback:"))}
	case strings.HasPrefix(tok, "typed:"):
		return pendingJob{update: commandUpdate(strings.TrimPrefix(tok, "typed:"))}
	default:
		return pendingJob{update: commandUpdate(tok)}
	}
}

func tokensFromJobs(jobs []pendingJob) []string {
	out := make([]string, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, tokenFromJob(job))
	}
	return out
}

func tokenFromJob(job pendingJob) string {
	u := job.update
	if u == nil {
		return ""
	}
	if u.CallbackQuery != nil {
		return "callback:" + u.CallbackQuery.Data
	}
	if u.Message != nil && u.Message.Text != "" && u.Message.Text[0] != '/' {
		return "typed:" + u.Message.Text
	}
	if u.Message != nil {
		return u.Message.Text
	}
	return ""
}

func TestCompactKeepsJobsWithoutUpdate(t *testing.T) {
	jobs := []pendingJob{{run: func() {}}, {update: commandUpdate("/help")}}
	got := compact(jobs)
	require.Len(t, got, 2)
}
