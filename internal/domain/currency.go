package domain

import (
	"strings"

	"golang.org/x/text/currency"
)

const (
	CurrencyUSD  = "USD"
	CurrencyEUR  = "EUR"
	CurrencyUSDT = "USDT"
)

func ParseCurrency(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if code == CurrencyUSDT {
		return CurrencyUSDT, nil
	}
	unit, err := currency.ParseISO(code)
	if err != nil {
		return "", ErrInvalidCurrency
	}
	return unit.String(), nil
}

type CurrencyTotal struct {
	Currency string
	Amount   Money
}

func FormatMoney(m Money, code string) string {
	amount := m.Format()
	switch strings.ToUpper(code) {
	case CurrencyUSD:
		return withSymbol(amount, "$")
	case CurrencyEUR:
		return withSymbol(amount, "€")
	default:
		return amount + " " + code
	}
}

func CurrencyUnit(code string) string {
	switch strings.ToUpper(code) {
	case CurrencyUSD:
		return "$"
	case CurrencyEUR:
		return "€"
	default:
		return code
	}
}

func withSymbol(amount, symbol string) string {
	if strings.HasPrefix(amount, "-") {
		return "-" + symbol + amount[1:]
	}
	return symbol + amount
}
