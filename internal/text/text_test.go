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
