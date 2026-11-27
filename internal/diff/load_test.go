package diff

import (
	"errors"
	"reflect"
	"testing"

	"github.com/anuj193roblox-ship-it/prisk/internal/gittest"
)

func TestLoadRejectsOptionLikeRanges(t *testing.T) {
	for _, rng := range []string{"", "-p", "--output=/tmp/x"} {
		if _, err := Load(rng); !errors.Is(err, ErrBadRange) {
			t.Errorf("Load(%q) error = %v, want ErrBadRange", rng, err)
		}
	}
}

func TestLoadReadsBranchChanges(t *testing.T) {
	gittest.New(t)
	gittest.Write(t, "README.md", 2)
	gittest.Commit(t, "start")
	gittest.Git(t, "checkout", "-q", "-b", "feature")
	gittest.Write(t, "auth/login.go", 3)
	gittest.Commit(t, "add login")
	// A later commit on main must not show up in main...feature.
	gittest.Git(t, "checkout", "-q", "main")
	gittest.Write(t, "other.go", 9)
	gittest.Commit(t, "unrelated main work")

	got, err := Load("main...feature")
	if err != nil {
		t.Fatal(err)
	}
	want := []FileChange{{Path: "auth/login.go", Added: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// Checks the rename format in TestParse against what real git prints.
func TestLoadRename(t *testing.T) {
	gittest.New(t)
	gittest.Write(t, "auth/login.go", 3)
	gittest.Commit(t, "start")
	gittest.Git(t, "checkout", "-q", "-b", "feature")
	gittest.Git(t, "mv", "auth/login.go", "auth/session.go")
	gittest.Commit(t, "rename")

	got, err := Load("main...feature")
	if err != nil {
		t.Fatal(err)
	}
	want := []FileChange{{Path: "auth/session.go"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadUnknownBranch(t *testing.T) {
	gittest.New(t)
	gittest.Write(t, "README.md", 1)
	gittest.Commit(t, "start")
	if _, err := Load("main...nope"); err == nil {
		t.Error("want an error for an unknown branch")
	}
}
