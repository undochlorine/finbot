package text

import (
	"strings"
	"testing"
)

func TestHelpListsMVPCommands(t *testing.T) {
	commands := []string{
		"/start",
		"/help",
		"/newbank",
		"/add",
		"/spend",
		"/set",
		"/delete",
		"/bank",
		"/toggle",
		"/banks",
		"/total",
		"/all",
	}
	for _, cmd := range commands {
		if !strings.Contains(Help, cmd+" -") {
			t.Errorf("help missing %s", cmd)
		}
	}
}

func TestStartPointsToHelp(t *testing.T) {
	if !strings.Contains(Start, "/help") {
		t.Fatal("start should point to /help")
	}
	if !strings.Contains(Start, "/newbank") {
		t.Fatal("start should mention a command shortcut")
	}
}

func TestHelpMentionsCommandShortcuts(t *testing.T) {
	if !strings.Contains(Help, "/newbank Holiday fund") {
		t.Fatal("help should show a command-plus-name example")
	}
	if !strings.Contains(Help, "spaces") {
		t.Fatal("help should mention that bank names may contain spaces")
	}
	if !strings.Contains(Help, "/add Holiday 100") {
		t.Fatal("help should show an /add shortcut")
	}
	if !strings.Contains(Help, "/spend Gifts 12.50") {
		t.Fatal("help should show a /spend shortcut")
	}
	if !strings.Contains(Help, "/set Live 0") {
		t.Fatal("help should show a /set shortcut")
	}
	if !strings.Contains(Help, "/delete Holiday") {
		t.Fatal("help should show a /delete shortcut")
	}
	if !strings.Contains(Help, "/bank Holiday") {
		t.Fatal("help should show a /bank shortcut")
	}
}

func TestBankCreated(t *testing.T) {
	tests := []struct {
		name     string
		bank     string
		included bool
		want     string
	}{
		{name: "included", bank: "Holiday", included: true, want: "counts toward your total"},
		{name: "excluded", bank: "Gifts", included: false, want: "does not count toward your total"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BankCreated(tt.bank, tt.included)
			if !strings.Contains(got, tt.bank) {
				t.Errorf("missing name %q in %q", tt.bank, got)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("missing %q in %q", tt.want, got)
			}
		})
	}
}

func TestBankNameTaken(t *testing.T) {
	got := BankNameTaken("Holiday")
	if !strings.Contains(got, "Holiday") {
		t.Fatalf("missing name: %q", got)
	}
}

func TestMoneyCopy(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want []string
	}{
		{name: "unknown", got: UnknownBank("Holiday"), want: []string{"Holiday", "/banks"}},
		{name: "ask add", got: AskAddAmount("Holiday"), want: []string{"Holiday"}},
		{name: "ask spend", got: AskSpendAmount("Gifts"), want: []string{"Gifts"}},
		{name: "ask set", got: AskSetAmount("Live"), want: []string{"Live"}},
		{name: "added", got: Added("Holiday", "100.00", "150.00"), want: []string{"Holiday", "100.00", "150.00"}},
		{name: "spent", got: Spent("Gifts", "12.50", "87.50"), want: []string{"Gifts", "12.50", "87.50"}},
		{name: "set", got: SetTo("Live", "0.00"), want: []string{"Live", "0.00"}},
		{name: "ask delete", got: AskDeleteConfirm("Holiday"), want: []string{"Holiday"}},
		{name: "deleted", got: BankDeleted("Holiday"), want: []string{"Holiday"}},
		{name: "bank included", got: BankCard("Holiday", "50.00", true), want: []string{"Holiday", "50.00", "in total"}},
		{name: "bank excluded", got: BankCard("Gifts", "12.50", false), want: []string{"Gifts", "12.50", "not in total"}},
		{name: "total", got: Total("123.45"), want: []string{"123.45"}},
		{name: "empty total", got: Total("0.00"), want: []string{"0.00"}},
		{
			name: "all",
			got:  All("Holiday: 50.00 (in total)\nGifts: 12.50 (not in total)", Total("50.00")),
			want: []string{"Holiday", "50.00", "in total", "Gifts", "12.50", "not in total", "50.00"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, w := range tt.want {
				if !strings.Contains(tt.got, w) {
					t.Errorf("missing %q in %q", w, tt.got)
				}
			}
		})
	}
}
