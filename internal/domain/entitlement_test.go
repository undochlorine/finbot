package domain

import (
	"testing"
	"time"
)

func TestEntitlementAlwaysFull(t *testing.T) {
	now := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	pct := 50
	hundred := 100

	tests := []struct {
		name string
		user User
	}{
		{name: "in trial", user: User{Plan: PlanTrial, TrialEndsAt: now.Add(time.Hour)}},
		{name: "expired trial", user: User{Plan: PlanFree, TrialEndsAt: now.Add(-time.Hour)}},
		{name: "paid", user: User{Plan: "paid", TrialEndsAt: now.Add(-time.Hour)}},
		{name: "50 percent off", user: User{Plan: PlanFree, TrialEndsAt: now.Add(-time.Hour), DiscountPercent: &pct}},
		{name: "100 percent whitelist", user: User{Plan: PlanFree, TrialEndsAt: now.Add(-time.Hour), DiscountPercent: &hundred}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Entitlement(tt.user, now); got != AccessFull {
				t.Fatalf("got %q, want %q", got, AccessFull)
			}
		})
	}
}
