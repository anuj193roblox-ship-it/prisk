// Package gittest builds throwaway git repos for tests.
package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// New creates an empty repo on branch main in a temp folder and makes it the
// current directory for the rest of the test. Skips the test if git is missing.
func New(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	// Ignore the developer's own git settings so tests behave the same everywhere.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "test")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "test@example.com")
	}
	t.Chdir(t.TempDir())
	Git(t, "init", "-q", "-b", "main")
}

// Git runs a git command in the current directory and fails the test if it errors.
func Git(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// Write creates or overwrites a file with n lines, making folders as needed.
func Write(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("line\n", n)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// WriteText creates or overwrites a file with exact contents.
func WriteText(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Commit stages everything and commits it.
func Commit(t *testing.T, msg string) {
	t.Helper()
	Git(t, "add", "-A")
	Git(t, "commit", "-q", "-m", msg)
}

// CommitAs stages everything and commits it with a chosen author email and date.
func CommitAs(t *testing.T, msg, email string, when time.Time) {
	t.Helper()
	Git(t, "add", "-A")
	d := when.Format(time.RFC3339)
	cmd := exec.Command("git", "commit", "-q", "--allow-empty", "-m", msg)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_EMAIL="+email, "GIT_COMMITTER_EMAIL="+email,
		"GIT_AUTHOR_DATE="+d, "GIT_COMMITTER_DATE="+d)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}
