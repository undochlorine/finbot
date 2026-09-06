package domain

import "testing"

func TestNormalizeBankName(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "ascii lower", in: "Live", want: "live"},
		{name: "trims", in: "  Gifts  ", want: "gifts"},
		{name: "unicode", in: "Подарки", want: "подарки"},
		{name: "empty", in: "", wantErr: true},
		{name: "spaces only", in: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeBankName(tt.in)
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
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
