package clock

import "time"

type Real struct{}

func New() Real {
	return Real{}
}

func (Real) Now() time.Time {
	return time.Now().UTC()
}
