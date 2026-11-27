// Package diff reads which files a branch changed, using git.
package diff

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrBadRange means the range was empty or looked like a git option.
var ErrBadRange = errors.New("invalid range")

// Load asks git which files changed in rng (for example "main...feature").
func Load(rng string) ([]FileChange, error) {
	// A range like "--output=x" would be read by git as an option, not a branch.
	if rng == "" || strings.HasPrefix(rng, "-") {
		return nil, ErrBadRange
	}
	// Arguments are passed one by one (no shell), and "--" marks the end of revisions.
	out, err := git("diff", "--numstat", "-z", "--no-ext-diff", rng, "--")
	if err != nil {
		return nil, err
	}
	return Parse(out)
}

// FileChange is one changed file and how many lines it added and deleted.
type FileChange struct {
	Path    string
	Added   int
	Deleted int
	Binary  bool // git can't count lines in binary files
}

// Parse reads the output of `git diff --numstat -z`.
// Each record is "added<TAB>deleted<TAB>path" ending in a NUL byte.
// For a rename the path is empty and the old and new paths follow as two more NUL-ended fields.
func Parse(out string) ([]FileChange, error) {
	fields := strings.Split(out, "\x00")
	var files []FileChange
	for i := 0; i < len(fields); i++ {
		rec := fields[i]
		if rec == "" {
			continue // the final NUL leaves one empty field
		}
		parts := strings.SplitN(rec, "\t", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("unexpected git output %q", rec)
		}
		fc := FileChange{Path: parts[2]}
		if fc.Path == "" {
			if i+2 >= len(fields) || fields[i+2] == "" {
				return nil, fmt.Errorf("rename record %q is missing its new path", rec)
			}
			fc.Path = fields[i+2]
			i += 2
		}
		if parts[0] == "-" && parts[1] == "-" {
			fc.Binary = true
		} else {
			added, err1 := strconv.Atoi(parts[0])
			deleted, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("bad line counts in %q", rec)
			}
			fc.Added, fc.Deleted = added, deleted
		}
		files = append(files, fc)
	}
	return files, nil
}
