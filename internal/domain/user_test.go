package domain

import "testing"

func TestKnownLocale(t *testing.T) {
	tests := []struct {
		locale string
		want   bool
	}{
		{locale: LocaleEN, want: true},
		{locale: LocaleRU, want: true},
		{locale: LocaleUK, want: true},
		{locale: LocaleMD, want: true},
		{locale: "", want: false},
		{locale: "fr", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			if got := KnownLocale(tt.locale); got != tt.want {
				t.Fatalf("KnownLocale(%q)=%v, want %v", tt.locale, got, tt.want)
			}
		})
	}
}
