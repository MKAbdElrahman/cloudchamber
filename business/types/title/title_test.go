package title

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	got, err := Parse("  Lecture notes  ")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.String() != "Lecture notes" {
		t.Fatalf("title: %q", got.String())
	}

	if _, err := Parse("   "); err == nil {
		t.Fatal("blank title accepted")
	}
	if _, err := Parse(stringsRepeat("a", 81)); err == nil {
		t.Fatal("long title accepted")
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for range n {
		out = append(out, s...)
	}
	return string(out)
}
