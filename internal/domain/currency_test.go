package domain

import (
	"errors"
	"testing"
)

func TestParseCurrency(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{name: "usd lower", in: "usd", want: CurrencyUSD},
		{name: "eur padded", in: " eur ", want: CurrencyEUR},
		{name: "uah lower", in: "uah", want: "UAH"},
		{name: "usdt mixed", in: "Usdt", want: CurrencyUSDT},
		{name: "metal", in: "xau", want: "XAU"},
		{name: "dollar sign", in: "$", wantErr: ErrInvalidCurrency},
		{name: "euro sign", in: "€", wantErr: ErrInvalidCurrency},
		{name: "name", in: "dollar", wantErr: ErrInvalidCurrency},
		{name: "btc", in: "BTC", wantErr: ErrInvalidCurrency},
		{name: "two letters", in: "US", wantErr: ErrInvalidCurrency},
		{name: "digits", in: "UAH1", wantErr: ErrInvalidCurrency},
		{name: "internal space", in: "U SD", wantErr: ErrInvalidCurrency},
		{name: "empty", in: "", wantErr: ErrInvalidCurrency},
		{name: "whitespace", in: "   ", wantErr: ErrInvalidCurrency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurrency(tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name string
		in   Money
		code string
		want string
	}{
		{name: "usd", in: 1000, code: CurrencyUSD, want: "$10.00"},
		{name: "usd negative", in: -1000, code: CurrencyUSD, want: "-$10.00"},
		{name: "usd zero", in: 0, code: CurrencyUSD, want: "$0.00"},
		{name: "usd cent", in: 1, code: CurrencyUSD, want: "$0.01"},
		{name: "usd lower still symbol", in: 1000, code: "usd", want: "$10.00"},
		{name: "eur", in: 1000, code: CurrencyEUR, want: "€10.00"},
		{name: "eur negative", in: -1000, code: CurrencyEUR, want: "-€10.00"},
		{name: "eur zero", in: 0, code: CurrencyEUR, want: "€0.00"},
		{name: "eur lower still symbol", in: 1000, code: "eur", want: "€10.00"},
		{name: "uah", in: 1000, code: "UAH", want: "10.00 UAH"},
		{name: "usdt", in: 1000, code: CurrencyUSDT, want: "10.00 USDT"},
		{name: "uah negative", in: -1000, code: "UAH", want: "-10.00 UAH"},
		{name: "jpy two decimals", in: 1000, code: "JPY", want: "10.00 JPY"},
		{name: "unknown", in: 1000, code: "FOO", want: "10.00 FOO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMoney(tt.in, tt.code)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
