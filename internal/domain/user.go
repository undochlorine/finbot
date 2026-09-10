package domain

import "time"

type UserID int64

const (
	PlanTrial = "trial"
	PlanFree  = "free"
	LocaleEN  = "en"
	LocaleRU  = "ru"
	LocaleUK  = "uk"
	LocaleMD  = "md"
)

func KnownLocale(locale string) bool {
	switch locale {
	case LocaleEN, LocaleRU, LocaleUK, LocaleMD:
		return true
	default:
		return false
	}
}

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
