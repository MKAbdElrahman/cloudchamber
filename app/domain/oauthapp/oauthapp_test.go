package oauthapp

import "testing"

func TestAccountName(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "Ada Lovelace", want: "Ada Lovelace"},
		{raw: "李", want: "Member"},
		{raw: "A", want: "Member"},
		{raw: "Name With Way Too Many Characters", want: "Name With Way Too Ma"},
	}

	for _, tt := range tests {
		got := accountName(tt.raw)
		if got.String() != tt.want {
			t.Errorf("accountName(%q) = %q, want %q", tt.raw, got.String(), tt.want)
		}
	}
}
