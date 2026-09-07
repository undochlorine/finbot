package domain

import "testing"

func TestParseYesNo(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		want   bool
		wantOK bool
	}{
		{name: "yes", in: Yes, want: true, wantOK: true},
		{name: "no", in: No, want: false, wantOK: true},
		{name: "YES", in: "YES", want: true, wantOK: true},
		{name: "maybe", in: "maybe"},
		{name: "empty", in: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseYesNo(tt.in)
			if ok != tt.wantOK {
				t.Fatalf("ok=%v want %v", ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
