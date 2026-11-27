package risk

import (
	"fmt"
	"sort"

	"github.com/anuj193roblox-ship-it/prisk/internal/config"
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

// Input is everything the checks look at.
type Input struct {
	Files   []diff.FileChange // what the branch changed
	History []diff.Commit     // the base branch's past commits, newest first
	Author  string            // email of whoever wrote the branch
	Config  config.Config
}

// Every check has the same shape, so Score can loop over them.
// Order matters only for ties: equal points keep this order in the output.
var checks = []func(Input) Finding{sensitiveCheck, testGapCheck, sizeCheck, hotspotCheck, unfamiliarCheck}

// Score runs every check and adds up the points.
func Score(in Input) Result {
	var counted []diff.FileChange
	for _, f := range in.Files {
		if !isSkipped(f, in.Config.Skip) {
			counted = append(counted, f)
		}
	}
	in.Files = counted // checks only ever see files that count
	res := Result{Findings: []Finding{}}
	for _, check := range checks {
		if f := check(in); f.Points > 0 {
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
func sizeCheck(in Input) Finding {
	files := in.Files
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
func sensitiveCheck(in Input) Finding {
	var hits []string
	for _, f := range in.Files {
		if isSensitive(f.Path, in.Config.Sensitive) {
			hits = append(hits, f.Path)
		}
	}
	if len(hits) == 0 {
		return Finding{}
	}
	return Finding{"sensitive", 30, "touches sensitive code: " + listUpTo3(hits)}
}

// testGapCheck: new code with no new tests means nothing checks that it works.
func testGapCheck(in Input) Finding {
	source, tests := false, false
	for _, f := range in.Files {
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
