package risk

import (
	"fmt"
	"sort"
	"strings"

	"github.com/anuj193roblox-ship-it/prisk/internal/diff"
)

// Finding is one reason the change is risky, worth some points.
type Finding struct {
	Signal string `json:"signal"`
	Points int    `json:"points"`
	Reason string `json:"reason"`
}

// Result is the overall verdict.
type Result struct {
	Score    int       `json:"score"`
	Level    string    `json:"level"`
	Findings []Finding `json:"findings"`
}

// Every check has the same shape, so Score can loop over them.
// Order matters only for ties: equal points keep this order in the output.
var checks = []func([]diff.FileChange) Finding{sensitiveCheck, testGapCheck, sizeCheck}

// Score runs every check on the changed files and adds up the points.
func Score(files []diff.FileChange) Result {
	var counted []diff.FileChange
	for _, f := range files {
		if !isSkipped(f) {
			counted = append(counted, f)
		}
	}
	res := Result{Findings: []Finding{}}
	for _, check := range checks {
		if f := check(counted); f.Points > 0 {
			res.Findings = append(res.Findings, f)
			res.Score += f.Points
		}
	}
	res.Score = min(res.Score, 100)
	res.Level = level(res.Score)
	sort.SliceStable(res.Findings, func(i, j int) bool {
		return res.Findings[i].Points > res.Findings[j].Points
	})
	return res
}

func level(score int) string {
	switch {
	case score >= 60:
		return "HIGH"
	case score >= 30:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// sizeCheck: bigger changes are harder to review, so more slips through.
func sizeCheck(files []diff.FileChange) Finding {
	lines := 0
	for _, f := range files {
		lines += f.Added + f.Deleted
	}
	pts := 0
	switch {
	case lines >= 500:
		pts = 30
	case lines >= 200:
		pts = 20
	case lines >= 50:
		pts = 10
	}
	noun := "files"
	if len(files) == 1 {
		noun = "file"
	}
	return Finding{"size", pts, fmt.Sprintf("%d lines changed across %d %s", lines, len(files), noun)}
}

// sensitiveCheck: mistakes in logins, money or data hurt the most.
func sensitiveCheck(files []diff.FileChange) Finding {
	var hits []string
	for _, f := range files {
		if isSensitive(f.Path) {
			hits = append(hits, f.Path)
		}
	}
	if len(hits) == 0 {
		return Finding{}
	}
	reason := "touches sensitive code: " + strings.Join(hits[:min(3, len(hits))], ", ")
	if len(hits) > 3 {
		reason += fmt.Sprintf(" (+%d more)", len(hits)-3)
	}
	return Finding{"sensitive", 30, reason}
}

// testGapCheck: new code with no new tests means nothing checks that it works.
func testGapCheck(files []diff.FileChange) Finding {
	source, tests := false, false
	for _, f := range files {
		if isTest(f.Path) {
			tests = true
		} else if isSource(f.Path) {
			source = true
		}
	}
	if source && !tests {
		return Finding{"test_gap", 20, "source code changed but no tests were added or changed"}
	}
	return Finding{}
}
