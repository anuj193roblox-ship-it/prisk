// Package risk scores a list of changed files. It never calls git or touches
// the disk, so every rule can be tested with plain data.
package risk

import (
	"path" // not path/filepath: git always prints "/" even on Windows
	"strings"

	"github.com/anuj193roblox-ship-it/prisk/internal/diff"
)

// Generated files: huge diffs that say nothing about risk.
var lockfiles = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true,
	"go.sum": true, "Cargo.lock": true, "poetry.lock": true,
}

var sourceExts = map[string]bool{
	".go": true, ".py": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true,
	".java": true, ".kt": true, ".rb": true, ".rs": true, ".c": true, ".h": true,
	".cpp": true, ".cs": true, ".php": true, ".swift": true,
}

// isSkipped reports whether every check should ignore f: binaries, lockfiles, and the config's skip patterns.
func isSkipped(f diff.FileChange, patterns []string) bool {
	if f.Binary || lockfiles[path.Base(f.Path)] {
		return true
	}
	for _, pat := range patterns {
		if !strings.Contains(pat, "/") {
			if ok, _ := path.Match(pat, path.Base(f.Path)); ok {
				return true
			}
			continue
		}
		// Like .gitignore: "vendor/*" also skips vendor/a/b.go, by matching each parent folder in turn.
		for p := f.Path; p != "."; p = path.Dir(p) {
			if ok, _ := path.Match(pat, p); ok {
				return true
			}
		}
	}
	return false
}

func isTest(p string) bool {
	name := path.Base(p)
	if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_test.py") ||
		(strings.HasPrefix(name, "test_") && strings.HasSuffix(name, ".py")) ||
		strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") {
		return true
	}
	for _, dir := range strings.Split(path.Dir(p), "/") {
		if dir == "test" || dir == "tests" || dir == "__tests__" {
			return true
		}
	}
	return false
}

func isSource(p string) bool {
	return !isTest(p) && sourceExts[path.Ext(p)]
}

func isSensitive(p string, sensitiveDirs []string) bool {
	// Wrap the folder path in slashes so "auth" only matches a whole folder name:
	// "/src/auth/" contains "/auth/", but "/author/" does not.
	dirs := "/" + path.Dir(p) + "/"
	for _, d := range sensitiveDirs {
		if strings.Contains(dirs, "/"+d+"/") {
			return true
		}
	}
	return false
}
