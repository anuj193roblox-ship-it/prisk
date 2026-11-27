package risk

import (
	"strings"
	"testing"
	"time"

	"github.com/anuj193roblox-ship-it/prisk/internal/config"
	"github.com/anuj193roblox-ship-it/prisk/internal/diff"
)

// now is the newest commit on the base branch; all windows count back from it.
var now = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

func daysAgo(n int) time.Time { return now.AddDate(0, 0, -n) }

// fixes makes n fix commits touching path, by email, n days ago.
func fixes(path string, n, days int, email string) []diff.Commit {
	var cs []diff.Commit
	for i := 0; i < n; i++ {
		cs = append(cs, diff.Commit{Email: email, Time: daysAgo(days), Subject: "fix: something", Files: []string{path}})
	}
	return cs
}

// history puts a commit dated "now" first, so the windows are anchored, then the rest.
func history(rest ...[]diff.Commit) []diff.Commit {
	h := []diff.Commit{{Email: "ops@x.com", Time: now, Subject: "chore: bump"}}
	for _, r := range rest {
		h = append(h, r...)
	}
	return h
}

func run(files []fc, h []diff.Commit, author string) Result {
	return Score(Input{Files: files, History: h, Author: author, Config: config.Default()})
}

func finding(r Result, signal string) (Finding, bool) {
	for _, f := range r.Findings {
		if f.Signal == signal {
			return f, true
		}
	}
	return Finding{}, false
}

func TestIsFix(t *testing.T) {
	cases := map[string]bool{
		"fix: crash on login":  true,
		"Fixed the cart total": true,
		"fixes #12":            true,
		"bugfix: null pointer": true,
		"HOTFIX payments":      true,
		`Revert "add cache"`:   true,
		"bug in parser":        true,
		"prefix the names":     false,
		"add test fixture":     false,
		"debug logging":        false,
		"feat: dark mode":      false,
	}
	for subject, want := range cases {
		if got := isFix(subject); got != want {
			t.Errorf("isFix(%q) = %v, want %v", subject, got, want)
		}
	}
}

func TestHotspot(t *testing.T) {
	file := []fc{{Path: "worker/queue.md", Added: 5}} // .md so only history checks can fire
	cases := []struct {
		name string
		h    []diff.Commit
		want int
	}{
		{"3 recent fixes", history(fixes("worker/queue.md", 3, 10, "me@x.com")), 15},
		{"2 recent fixes", history(fixes("worker/queue.md", 2, 10, "me@x.com")), 0},
		{"fix exactly 90 days ago counts", history(fixes("worker/queue.md", 3, 90, "me@x.com")), 15},
		{"3 fixes older than 90 days", history(fixes("worker/queue.md", 3, 91, "me@x.com")), 0},
		{"fixes to another file", history(fixes("worker/other.md", 3, 10, "me@x.com")), 0},
		{"non-fix commits", history([]diff.Commit{
			{Email: "me@x.com", Time: daysAgo(1), Subject: "prefix the names", Files: []string{"worker/queue.md"}},
			{Email: "me@x.com", Time: daysAgo(2), Subject: "prefix the names", Files: []string{"worker/queue.md"}},
			{Email: "me@x.com", Time: daysAgo(3), Subject: "prefix the names", Files: []string{"worker/queue.md"}},
		}), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := run(file, c.h, "me@x.com")
			if got := points(r)["hotspot"]; got != c.want {
				t.Errorf("hotspot = %d, want %d; findings %+v", got, c.want, r.Findings)
			}
		})
	}
}

func TestHotspotReasonMostFixedFirst(t *testing.T) {
	files := []fc{{Path: "b.md"}, {Path: "a.md"}, {Path: "c.md"}, {Path: "d.md"}}
	h := history(fixes("b.md", 3, 5, "me@x.com"), fixes("a.md", 5, 5, "me@x.com"),
		fixes("c.md", 4, 5, "me@x.com"), fixes("d.md", 3, 5, "me@x.com"))
	f, ok := finding(run(files, h, "me@x.com"), "hotspot")
	want := "changes bug-prone files: a.md (5 fixes in 90 days), c.md (4 fixes in 90 days), b.md (3 fixes in 90 days) (+1 more)"
	if !ok || f.Reason != want {
		t.Errorf("reason = %q\nwant     %q", f.Reason, want)
	}
}

func TestHotspotIgnoresSkippedFiles(t *testing.T) {
	r := run([]fc{{Path: "package-lock.json"}}, history(fixes("package-lock.json", 5, 1, "me@x.com")), "me@x.com")
	if _, ok := finding(r, "hotspot"); ok {
		t.Errorf("lockfile flagged as hotspot: %+v", r.Findings)
	}
}

func TestUnfamiliar(t *testing.T) {
	touch := func(path, email string, days int) []diff.Commit {
		return []diff.Commit{{Email: email, Time: daysAgo(days), Subject: "feat: work", Files: []string{path}}}
	}
	cases := []struct {
		name   string
		files  []fc
		h      []diff.Commit
		author string
		want   int
	}{
		{"never touched it", []fc{{Path: "billing/a.md"}}, history(touch("billing/a.md", "other@x.com", 10)), "me@x.com", 10},
		{"touched it before", []fc{{Path: "billing/a.md"}}, history(touch("billing/a.md", "me@x.com", 10)), "me@x.com", 0},
		{"email case ignored", []fc{{Path: "billing/a.md"}}, history(touch("billing/a.md", "Me@X.com", 10)), "me@x.com", 0},
		{"touched one of two", []fc{{Path: "a.md"}, {Path: "b.md"}},
			history(touch("a.md", "other@x.com", 10), touch("b.md", "me@x.com", 10)), "me@x.com", 0},
		{"brand-new file", []fc{{Path: "new.md"}}, history(touch("old.md", "other@x.com", 10)), "me@x.com", 0},
		{"last touched over a year ago counts as new", []fc{{Path: "a.md"}},
			history(touch("a.md", "other@x.com", 366)), "me@x.com", 0},
		{"no author known", []fc{{Path: "a.md"}}, history(touch("a.md", "other@x.com", 10)), "", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := run(c.files, c.h, c.author)
			if got := points(r)["unfamiliar"]; got != c.want {
				t.Errorf("unfamiliar = %d, want %d; findings %+v", got, c.want, r.Findings)
			}
		})
	}
}

func TestUnfamiliarReason(t *testing.T) {
	h := history([]diff.Commit{{Email: "other@x.com", Time: daysAgo(3), Subject: "feat",
		Files: []string{"a.md", "b.md", "c.md", "d.md"}}})
	f, ok := finding(run([]fc{{Path: "d.md"}, {Path: "a.md"}, {Path: "c.md"}, {Path: "b.md"}, {Path: "new.md"}}, h, "me@x.com"), "unfamiliar")
	want := "author hasn't changed these files in the last year: a.md, b.md, c.md (+1 more)"
	if !ok || f.Reason != want {
		t.Errorf("reason = %q\nwant     %q", f.Reason, want)
	}
}

func TestAllFiveChecksCapAt100(t *testing.T) {
	r := run([]fc{{Path: "payments/ledger.go", Added: 600}},
		history(fixes("payments/ledger.go", 3, 5, "other@x.com")), "me@x.com")
	if r.Score != 100 || r.Level != "HIGH" || len(r.Findings) != 5 {
		t.Errorf("got %d %s with %d findings %+v; want 100 HIGH with 5", r.Score, r.Level, len(r.Findings), r.Findings)
	}
	if !strings.HasPrefix(r.Findings[0].Reason, "touches sensitive code") {
		t.Errorf("first finding = %q, want the +30 sensitive one", r.Findings[0].Reason)
	}
}
