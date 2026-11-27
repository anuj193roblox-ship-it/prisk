package diff

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Commit is one past commit: who made it, when, its message's first line, and the files it touched.
type Commit struct {
	Email   string
	Time    time.Time
	Subject string
	Files   []string
}

// maxCommits keeps huge repos fast.
// shortcut: only the newest 5,000 commits are read; raise this if a repo makes more than that in a year.
const maxCommits = "5000"

// Log reads the history of rev, newest first, skipping merge commits.
func Log(rev string) ([]Commit, error) {
	if strings.HasPrefix(rev, "-") {
		return nil, ErrBadRange
	}
	// \x1e starts each commit and \x1f separates its fields: characters that never appear in emails or messages.
	// %aE (capital E) applies .mailmap, so one person with several emails counts as one person.
	// File names come out unescaped because git() turns off core.quotePath for every call.
	out, err := git("log", "--no-merges", "--max-count="+maxCommits,
		"--format=%x1e%aE%x1f%ct%x1f%s", "--name-only", rev, "--")
	if err != nil {
		return nil, err
	}
	return ParseLog(out)
}

// ParseLog reads the output of the git log command in Log.
// shortcut: a file name containing a line break would be split in two; such names are vanishingly rare.
func ParseLog(out string) ([]Commit, error) {
	var commits []Commit
	for _, rec := range strings.Split(out, "\x1e") {
		if rec == "" {
			continue
		}
		lines := strings.Split(rec, "\n")
		fields := strings.Split(lines[0], "\x1f")
		if len(fields) != 3 {
			return nil, fmt.Errorf("unexpected git log line %q", lines[0])
		}
		secs, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad commit time %q", fields[1])
		}
		c := Commit{Email: fields[0], Time: time.Unix(secs, 0), Subject: fields[2]}
		for _, f := range lines[1:] {
			if f != "" {
				c.Files = append(c.Files, f)
			}
		}
		commits = append(commits, c)
	}
	return commits, nil
}

// Author returns the author email of commit rev.
func Author(rev string) (string, error) {
	if strings.HasPrefix(rev, "-") {
		return "", ErrBadRange
	}
	out, err := git("log", "-1", "--format=%aE", rev, "--") // %aE applies .mailmap, as in Log
	return strings.TrimSpace(out), err
}

// Show returns a file's contents as of rev, or nil if the file doesn't exist there.
// Callers must check rev is valid first (Load does), since a bad rev also reads as "no file".
func Show(rev, path string) ([]byte, error) {
	out, err := exec.Command("git", "cat-file", "blob", rev+":"+path).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("running git: %w", err)
	}
	return out, nil
}

// git runs a git command (args[0] is the command name, e.g. "log") and returns its output,
// or git's own error message prefixed with that command name.
func git(args ...string) (string, error) {
	// Settings go here, not in args, so args[0] stays the command name.
	// quotePath=false prints non-English file names as-is instead of escaped.
	out, err := exec.Command("git", append([]string{"-c", "core.quotePath=false"}, args...)...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("running git: %w", err)
	}
	return string(out), nil
}
