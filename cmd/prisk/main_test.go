package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

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

// reasons runs prisk --json on rng and returns each finding's reason by signal.
func reasons(t *testing.T, rng string) map[string]string {
	t.Helper()
	code, out, errOut := runCLI("score", rng, "--json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	var res struct {
		Findings []struct{ Signal, Reason string }
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	for _, f := range res.Findings {
		m[f.Signal] = f.Reason
	}
	return m
}

// fixThreeTimes makes three fix commits to path on the current branch.
func fixThreeTimes(t *testing.T, path string) {
	for i := 2; i <= 4; i++ {
		gittest.Write(t, path, i)
		gittest.Commit(t, "fix: queue stalls")
	}
}

func TestHotspotComesFromBaseHistory(t *testing.T) {
	gittest.New(t)
	gittest.Write(t, "worker/queue.md", 1)
	gittest.Commit(t, "start")
	fixThreeTimes(t, "worker/queue.md")
	gittest.Git(t, "checkout", "-q", "-b", "pr")
	gittest.Write(t, "worker/queue.md", 10)
	gittest.Commit(t, "tweak queue")

	if got := reasons(t, "main...pr")["hotspot"]; got != "changes bug-prone files: worker/queue.md (3 fixes in 90 days)" {
		t.Errorf("hotspot reason = %q", got)
	}
}

func TestFixesOnThePRBranchDontCount(t *testing.T) {
	gittest.New(t)
	gittest.Write(t, "worker/queue.md", 1)
	gittest.Commit(t, "start")
	gittest.Git(t, "checkout", "-q", "-b", "pr")
	fixThreeTimes(t, "worker/queue.md")

	if got, ok := reasons(t, "main...pr")["hotspot"]; ok {
		t.Errorf("PR's own fixes made a hotspot: %q", got)
	}
}

func TestUnfamiliarAuthor(t *testing.T) {
	gittest.New(t)
	gittest.Write(t, "billing/invoice.md", 1)
	gittest.CommitAs(t, "start", "someone-else@example.com", time.Now())
	gittest.Git(t, "checkout", "-q", "-b", "pr")
	gittest.Write(t, "billing/invoice.md", 3)
	gittest.Commit(t, "change invoice") // by test@example.com

	want := "author hasn't changed these files in the last year: billing/invoice.md"
	if got := reasons(t, "main...pr")["unfamiliar"]; got != want {
		t.Errorf("unfamiliar reason = %q, want %q", got, want)
	}
}

func TestConfigIsReadFromBase(t *testing.T) {
	gittest.New(t)
	gittest.WriteText(t, ".prisk.json", `{"sensitive": ["infra"]}`)
	gittest.Commit(t, "add config")
	gittest.Git(t, "checkout", "-q", "-b", "pr")
	gittest.Write(t, "infra/main.tf", 3)
	gittest.Write(t, "auth/notes.md", 3)
	gittest.Commit(t, "infra change")

	if got := reasons(t, "main...pr")["sensitive"]; got != "touches sensitive code: infra/main.tf" {
		t.Errorf("sensitive reason = %q, want only infra/ flagged", got)
	}
}

func TestConfigOnlyOnPRBranchIsIgnored(t *testing.T) {
	gittest.New(t)
	gittest.Write(t, "README.md", 1)
	gittest.Commit(t, "start")
	gittest.Git(t, "checkout", "-q", "-b", "pr")
	// The PR tries to switch off the sensitive check that would flag it.
	gittest.WriteText(t, ".prisk.json", `{"sensitive": []}`)
	gittest.Write(t, "auth/login.md", 3)
	gittest.Commit(t, "sneaky")

	if _, ok := reasons(t, "main...pr")["sensitive"]; !ok {
		t.Error("a config added by the PR itself switched off the sensitive check")
	}
}

func TestBadConfigOnBase(t *testing.T) {
	gittest.New(t)
	gittest.WriteText(t, ".prisk.json", `{"sensitve": ["infra"]}`)
	gittest.Commit(t, "typo in config")
	gittest.Git(t, "checkout", "-q", "-b", "pr")
	gittest.Write(t, "a.md", 1)
	gittest.Commit(t, "change")

	code, out, errOut := runCLI("score", "main...pr")
	if code != 1 || out != "" || !strings.HasPrefix(errOut, "prisk: .prisk.json: ") {
		t.Errorf("exit %d, stdout %q, stderr %q; want exit 1 and a .prisk.json error", code, out, errOut)
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
