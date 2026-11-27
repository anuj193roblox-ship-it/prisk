package risk

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/anuj193roblox-ship-it/prisk/internal/diff"
)

// \b means "word boundary": "fix:" matches, "prefix" and "fixture" don't.
var fixWord = regexp.MustCompile(`(?i)\b(fix|fixes|fixed|bug|bugfix|hotfix|revert)\b`)

func isFix(subject string) bool { return fixWord.MatchString(subject) }

// newest is the time of the newest commit in the history. Windows count back from it rather than
// from today, so the same range always gets the same score.
func newest(history []diff.Commit) time.Time {
	var t time.Time
	for _, c := range history {
		if c.Time.After(t) {
			t = c.Time
		}
	}
	return t
}

// hotspotCheck: files that needed many fixes recently tend to break again.
func hotspotCheck(in Input) Finding {
	cutoff := newest(in.History).AddDate(0, 0, -90)
	fixCount := map[string]int{}
	for _, c := range in.History {
		if c.Time.Before(cutoff) || !isFix(c.Subject) {
			continue
		}
		for _, f := range c.Files {
			fixCount[f]++
		}
	}
	var hot []string
	for _, f := range in.Files {
		if fixCount[f.Path] >= 3 {
			hot = append(hot, f.Path)
		}
	}
	if len(hot) == 0 {
		return Finding{}
	}
	sort.Slice(hot, func(i, j int) bool {
		if fixCount[hot[i]] != fixCount[hot[j]] {
			return fixCount[hot[i]] > fixCount[hot[j]]
		}
		return hot[i] < hot[j]
	})
	labels := make([]string, len(hot))
	for i, p := range hot {
		labels[i] = fmt.Sprintf("%s (%d fixes in 90 days)", p, fixCount[p])
	}
	return Finding{"hotspot", 15, "changes bug-prone files: " + listUpTo3(labels)}
}

// unfamiliarCheck: people make more mistakes in code they've never worked on.
func unfamiliarCheck(in Input) Finding {
	if in.Author == "" {
		return Finding{}
	}
	cutoff := newest(in.History).AddDate(-1, 0, 0)
	changed := map[string]bool{}
	for _, f := range in.Files {
		changed[f.Path] = true
	}
	existing := map[string]bool{} // changed files that have history: only those can be "known"
	for _, c := range in.History {
		if c.Time.Before(cutoff) {
			continue
		}
		for _, f := range c.Files {
			if !changed[f] {
				continue
			}
			if strings.EqualFold(c.Email, in.Author) {
				return Finding{} // the author has worked on at least one of these files
			}
			existing[f] = true
		}
	}
	if len(existing) == 0 {
		return Finding{}
	}
	paths := make([]string, 0, len(existing))
	for p := range existing {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return Finding{"unfamiliar", 10, "author hasn't changed these files in the last year: " + listUpTo3(paths)}
}

// listUpTo3 joins the first three items and counts the rest: "a, b, c (+2 more)".
func listUpTo3(items []string) string {
	s := strings.Join(items[:min(3, len(items))], ", ")
	if len(items) > 3 {
		s += fmt.Sprintf(" (+%d more)", len(items)-3)
	}
	return s
}
