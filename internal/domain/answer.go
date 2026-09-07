package domain

import "strings"

const (
	Yes = "yes"
	No  = "no"
)

func ParseYesNo(s string) (bool, bool) {
	switch strings.ToLower(s) {
	case Yes:
		return true, true
	case No:
		return false, true
	default:
		return false, false
	}
}
