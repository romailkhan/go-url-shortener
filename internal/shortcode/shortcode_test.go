package shortcode

import "testing"

func TestRandomLength(t *testing.T) {
	t.Parallel()
	s, err := Random(12)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 12 {
		t.Fatalf("got len %d, want 12", len(s))
	}
}
