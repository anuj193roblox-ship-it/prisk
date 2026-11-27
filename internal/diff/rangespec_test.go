package diff

import "testing"

func TestSplit(t *testing.T) {
	cases := []struct{ in, base, head string }{
		{"main...feature", "main", "feature"},
		{"main..feature", "main", "feature"},
		{"HEAD~10...HEAD", "HEAD~10", "HEAD"},
		{"...feature", "HEAD", "feature"},
		{"main...", "main", "HEAD"},
		{"abc123", "abc123", "HEAD"},
	}
	for _, c := range cases {
		base, head := Split(c.in)
		if base != c.base || head != c.head {
			t.Errorf("Split(%q) = (%q, %q), want (%q, %q)", c.in, base, head, c.base, c.head)
		}
	}
}
