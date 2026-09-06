package service

import "time"

type Clock interface {
	Now() time.Time
}
