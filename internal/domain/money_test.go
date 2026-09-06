package domain

import (
	"math"
	"testing"
)

func TestParseMoney(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Money
		wantErr bool
	}{
		{name: "whole", in: "100", want: 10000},
		{name: "one decimal", in: "100.5", want: 10050},
		{name: "two decimals", in: "100.50", want: 10050},
		{name: "zero", in: "0", want: 0},
		{name: "zero cents", in: "0.00", want: 0},
		{name: "one cent", in: "0.01", want: 1},
		{name: "one tenth", in: "0.1", want: 10},
		{name: "negative", in: "-12.50", want: -1250},
		{name: "positive sign", in: "+3.2", want: 320},
		{name: "trimmed", in: "  8.09  ", want: 809},
		{name: "leading zeros", in: "0100.50", want: 10050},
		{name: "max int64 cents", in: "92233720368547758.07", want: math.MaxInt64},
		{name: "min int64 cents", in: "-92233720368547758.08", want: math.MinInt64},
		{name: "empty", in: "", wantErr: true},
		{name: "whitespace only", in: "  ", wantErr: true},
		{name: "letters", in: "abc", wantErr: true},
		{name: "three decimals", in: "1.234", wantErr: true},
		{name: "trailing dot", in: "100.", wantErr: true},
		{name: "leading dot", in: ".50", wantErr: true},
		{name: "two dots", in: "1.2.3", wantErr: true},
		{name: "comma", in: "1,00", wantErr: true},
		{name: "scientific", in: "1e2", wantErr: true},
		{name: "currency", in: "$100", wantErr: true},
		{name: "sign only", in: "-", wantErr: true},
		{name: "plus only", in: "+", wantErr: true},
		{name: "internal space", in: "1 00", wantErr: true},
		{name: "positive overflow frac", in: "92233720368547758.08", wantErr: true},
		{name: "positive overflow whole", in: "92233720368547759", wantErr: true},
		{name: "negative overflow", in: "-92233720368547758.09", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMoney(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMoneyFormat(t *testing.T) {
	tests := []struct {
		name string
		in   Money
		want string
	}{
		{name: "two decimals", in: 12345, want: "123.45"},
		{name: "trailing zero", in: 12340, want: "123.40"},
		{name: "zero", in: 0, want: "0.00"},
		{name: "one cent", in: 1, want: "0.01"},
		{name: "negative", in: -1250, want: "-12.50"},
		{name: "max", in: math.MaxInt64, want: "92233720368547758.07"},
		{name: "min", in: math.MinInt64, want: "-92233720368547758.08"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.Format()
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseMoneyRoundTrip(t *testing.T) {
	values := []Money{0, 1, -1, 100, -1250, math.MaxInt64, math.MinInt64}
	for _, v := range values {
		got, err := ParseMoney(v.Format())
		if err != nil {
			t.Fatalf("ParseMoney(%q): %v", v.Format(), err)
		}
		if got != v {
			t.Fatalf("round-trip %d: got %d", v, got)
		}
	}
}
