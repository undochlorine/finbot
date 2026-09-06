package domain

import "time"

type UserID int64

const (
	PlanTrial = "trial"
	PlanFree  = "free"
)

type User struct {
	TelegramID      UserID
	Username        string
	LastActivityAt  time.Time
	Plan            string
	TrialEndsAt     time.Time
	DiscountPercent *int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
