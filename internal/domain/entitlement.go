package domain

import "time"

type Access string

const AccessFull Access = "full"

func Entitlement(_ User, _ time.Time) Access {
	return AccessFull
}
