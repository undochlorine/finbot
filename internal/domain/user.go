package domain

import "time"

type UserID int64

const (
	PlanTrial = "trial"
	PlanFree  = "free"
	LocaleEN  = "en"
)

type User struct {
	TelegramID      UserID
	Username        string
	LastActivityAt  time.Time
	Plan            string
	TrialEndsAt     time.Time
	DiscountPercent *int
	Locale          string
	ReferredBy      *UserID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
