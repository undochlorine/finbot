package clock

import (
	"testing"
	"time"
)

func TestRealNowUTC(t *testing.T) {
	got := New().Now()
	if got.Location() != time.UTC {
		t.Fatalf("location %s, want UTC", got.Location())
	}
}
