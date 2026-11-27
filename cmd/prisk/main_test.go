package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/anuj193roblox-ship-it/prisk/internal/gittest"
)

// setupRepo makes a repo with two branches off main:
// "feature" adds 250 lines of login code with no tests; "docs" only edits the README.
func setupRepo(t *testing.T) {
	gittest.New(t)
	gittest.Write(t, "README.md", 5)
	gittest.Commit(t, "start")
	gittest.Git(t, "checkout", "-q", "-b", "feature")
	gittest.Write(t, "src/auth/login.go", 250)
	gittest.Commit(t, "add login")
	gittest.Git(t, "checkout", "-q", "-b", "docs", "main")
	gittest.Write(t, "README.md", 8)
	gittest.Commit(t, "docs")
}

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestScoreText(t *testing.T) {
	setupRepo(t)
	code, out, _ := runCLI("score", "main...feature")
	want := "Risk: HIGH (70/100)\n" +
		"  +30  touches sensitive code: src/auth/login.go\n" +
		"  +20  source code changed but no tests were added or changed\n" +
		"  +20  250 lines changed across 1 file\n"
	if code != 0 || out != want {
		t.Errorf("exit %d, output:\n%s\nwant exit 0, output:\n%s", code, out, want)
	}
}

func TestScoreDocsOnly(t *testing.T) {
	setupRepo(t)
	code, out, _ := runCLI("score", "main...docs")
	want := "Risk: LOW (0/100)\n  no risk signals found\n"
	if code != 0 || out != want {
		t.Errorf("exit %d, output:\n%s\nwant:\n%s", code, out, want)
	}
}

func TestScoreJSON(t *testing.T) {
	setupRepo(t)
	// The flag must work before or after the range.
	for _, args := range [][]string{
		{"score", "--json", "main...feature"},
		{"score", "main...feature", "--json"},
	} {
		code, out, errOut := runCLI(args...)
		if code != 0 {
			t.Fatalf("%v: exit %d, stderr %s", args, code, errOut)
		}
		var got struct {
			Score    int
			Level    string
			Findings []struct {
				Signal string
				Points int
				Reason string
			}
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("%v: not valid JSON: %v\n%s", args, err, out)
		}
		if got.Score != 70 || got.Level != "HIGH" || len(got.Findings) != 3 {
			t.Errorf("%v: got %+v", args, got)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	cases := [][]string{
		{},
		{"score"},
		{"nope", "main...feature"},
		{"score", "main...feature", "extra"},
		{"score", "--foo", "main...feature"},
	}
	for _, args := range cases {
		code, out, errOut := runCLI(args...)
		if code != 2 || out != "" || errOut == "" {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit 2 and a message on stderr only", args, code, out, errOut)
		}
	}
}

func TestOptionLikeRangeRejected(t *testing.T) {
	// "--" ends flags, so "--output=x" reaches the range check instead of the flag parser.
	code, _, errOut := runCLI("score", "--", "--output=x")
	if code != 2 || !strings.Contains(errOut, "invalid range") {
		t.Errorf("exit %d, stderr %q; want exit 2 and 'invalid range'", code, errOut)
	}
}

func TestGitErrors(t *testing.T) {
	setupRepo(t)
	code, out, errOut := runCLI("score", "main...nope")
	if code != 1 || out != "" || !strings.HasPrefix(errOut, "prisk: ") {
		t.Errorf("unknown branch: exit %d, stdout %q, stderr %q", code, out, errOut)
	}

	t.Chdir(t.TempDir()) // not a git repo
	code, _, errOut = runCLI("score", "main...feature")
	if code != 1 || !strings.HasPrefix(errOut, "prisk: ") {
		t.Errorf("not a repo: exit %d, stderr %q", code, errOut)
	}
}
