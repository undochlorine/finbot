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
}
