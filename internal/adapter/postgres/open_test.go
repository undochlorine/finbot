package postgres

import (
	"context"
	"testing"
)

func TestOpenRejectsEmptyURL(t *testing.T) {
	_, err := Open(context.Background(), "  ")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenRejectsInvalidURL(t *testing.T) {
	_, err := Open(context.Background(), "://not-a-url")
	if err == nil {
		t.Fatal("expected error")
	}
}
