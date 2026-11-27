package risk

import (
	"testing"

	"github.com/anuj193roblox-ship-it/prisk/internal/config"
	"github.com/anuj193roblox-ship-it/prisk/internal/diff"
)

func TestIsTest(t *testing.T) {
	cases := map[string]bool{
		"auth/login_test.go":         true,
		"tests/test_login.py":        true,
		"app/test_utils.py":          true,
		"app/utils_test.py":          true,
		"web/Button.test.tsx":        true,
		"web/cart.spec.js":           true,
		"src/__tests__/cart.js":      true,
		"test/helpers.rb":            true,
		"pkg/a/b/tests/fixture.json": true,
		"auth/login.go":              false,
		"app/testing.go":             false, // "testing" isn't "test"
		"contest/entry.py":           false, // folder name contains "test" but isn't "test"
		"test_notes.md":              false, // test_ prefix only counts for Python
		"README.md":                  false,
	}
	for path, want := range cases {
		if got := isTest(path); got != want {
			t.Errorf("isTest(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestIsSource(t *testing.T) {
	cases := map[string]bool{
		"auth/login.go":      true,
		"web/App.tsx":        true,
		"lib/core.rs":        true,
		"auth/login_test.go": false, // tests are never source
		"README.md":          false,
		"config/app.yml":     false,
		"logo.png":           false,
		"Makefile":           false,
	}
	for path, want := range cases {
		if got := isSource(path); got != want {
			t.Errorf("isSource(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestIsSensitive(t *testing.T) {
	cases := map[string]bool{
		"auth/login.go":                 true,
		"src/auth/session.go":           true, // any depth
		"payments/stripe/charge.go":     true,
		"billing/invoice.py":            true,
		"db/migrations/004_users.sql":   true,
		"security/policy.md":            true,
		".github/workflows/ci.yml":      true,
		"author/bio.go":                 false, // "author" is not "auth"
		"auth.go":                       false, // a file named auth, not a folder
		"docs/auth-guide.md":            false,
		".github/ISSUE_TEMPLATE/bug.md": false, // only workflows are sensitive
		"app/github/workflows/x.yml":    false, // needs the dot: .github
	}
	for path, want := range cases {
		if got := isSensitive(path, config.DefaultSensitive); got != want {
			t.Errorf("isSensitive(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestIsSkipped(t *testing.T) {
	cases := []struct {
		f    diff.FileChange
		want bool
	}{
		{diff.FileChange{Path: "logo.png", Binary: true}, true},
		{diff.FileChange{Path: "package-lock.json", Added: 5000}, true},
		{diff.FileChange{Path: "web/yarn.lock"}, true},
		{diff.FileChange{Path: "go.sum"}, true},
		{diff.FileChange{Path: "go.mod"}, false},
		{diff.FileChange{Path: "package.json"}, false},
	}
	for _, c := range cases {
		if got := isSkipped(c.f, nil); got != c.want {
			t.Errorf("isSkipped(%+v) = %v, want %v", c.f, got, c.want)
		}
	}
}

func TestSkipPatterns(t *testing.T) {
	patterns := []string{"vendor/*", "*.pb.go", "docs/generated/*"}
	cases := map[string]bool{
		"vendor/lib.go":         true,
		"vendor/a/b/c.go":       true, // a pattern with "/" also matches parent folders
		"api/user.pb.go":        true, // no "/" in the pattern: matched against the file name
		"user.pb.go":            true,
		"docs/generated/x/y.md": true,
		"src/vendor/lib.go":     false, // "vendor/*" starts at the repo root
		"docs/guide.md":         false,
		"api/user.go":           false,
	}
	for p, want := range cases {
		if got := isSkipped(diff.FileChange{Path: p}, patterns); got != want {
			t.Errorf("isSkipped(%q) = %v, want %v", p, got, want)
		}
	}
}
