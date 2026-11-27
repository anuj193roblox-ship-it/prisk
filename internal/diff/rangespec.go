package diff

import "strings"

// Split returns the two sides of a range: "main...feature" gives ("main", "feature").
// An empty side means HEAD, and a single revision is compared against HEAD.
func Split(rng string) (base, head string) {
	for _, sep := range []string{"...", ".."} {
		if b, h, ok := strings.Cut(rng, sep); ok {
			return orHEAD(b), orHEAD(h)
		}
	}
	return rng, "HEAD"
}

func orHEAD(rev string) string {
	if rev == "" {
		return "HEAD"
	}
	return rev
}
