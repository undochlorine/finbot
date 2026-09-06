package domain

import (
	"math"
	"strconv"
	"strings"
)

const (
	centsPerUnit      = 100
	maxFractionDigits = 2
	decimalBase       = 10
)

type Money int64

func ParseMoney(raw string) (Money, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, ErrInvalidAmount
	}

	neg := false
	switch s[0] {
	case '-':
		neg = true
		s = s[1:]
	case '+':
		s = s[1:]
	}
	if s == "" {
		return 0, ErrInvalidAmount
	}

	whole, frac, hasDot := strings.Cut(s, ".")
	if !isDigits(whole) {
		return 0, ErrInvalidAmount
	}
	if hasDot {
		if len(frac) == 0 || len(frac) > maxFractionDigits || !isDigits(frac) {
			return 0, ErrInvalidAmount
		}
	}

	wholeVal, err := strconv.ParseInt(whole, decimalBase, 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}

	var fracVal int64
	if hasDot {
		fracVal = int64(frac[0] - '0')
		fracVal *= decimalBase
		if len(frac) == maxFractionDigits {
			fracVal += int64(frac[1] - '0')
		}
	}

	return centsFromParts(wholeVal, fracVal, neg)
}

func (m Money) Format() string {
	v := int64(m)
	neg := v < 0
	var whole, frac int64
	if v == math.MinInt64 {
		whole = -(math.MinInt64 / centsPerUnit)
		frac = -(math.MinInt64 % centsPerUnit)
	} else {
		if neg {
			v = -v
		}
		whole = v / centsPerUnit
		frac = v % centsPerUnit
	}

	out := strconv.FormatInt(whole, decimalBase) + "." + pad2(frac)
	if neg {
		return "-" + out
	}
	return out
}

func centsFromParts(whole, frac int64, neg bool) (Money, error) {
	maxWhole := int64(math.MaxInt64 / centsPerUnit)
	if whole > maxWhole {
		return 0, ErrInvalidAmount
	}
	if whole == maxWhole {
		maxFrac := int64(math.MaxInt64 % centsPerUnit)
		if neg {
			maxFrac = -(math.MinInt64 % centsPerUnit)
		}
		if frac > maxFrac {
			return 0, ErrInvalidAmount
		}
	}

	cents := whole*centsPerUnit + frac
	if neg {
		cents = -cents
	}
	return Money(cents), nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func pad2(n int64) string {
	s := strconv.FormatInt(n, decimalBase)
	if len(s) == 1 {
		return "0" + s
	}
	return s
}
