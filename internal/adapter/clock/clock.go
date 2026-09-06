package clock

import (
	"time"

	"finbot/internal/ports"
)

var _ ports.Clock = Real{}

type Real struct{}

func New() Real {
	return Real{}
}

func (Real) Now() time.Time {
	return time.Now().UTC()
}
