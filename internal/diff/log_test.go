package diff

import (
	"reflect"
	"testing"
	"time"

	"github.com/anuj193roblox-ship-it/prisk/internal/gittest"
)

func TestParseLog(t *testing.T) {
	// What `git log --format=%x1e%ae%x1f%ct%x1f%s --name-only` prints.
	in := "\x1ea@x.com\x1f1700000000\x1ffix: crash on login\n\nsrc/a.go\nsrc/b.go\n" +
		"\x1eb@x.com\x1f1690000000\x1fempty commit\n" +
		"\x1ec@x.com\x1f1680000000\x1fstart\n\nmy docs/read me.md\n"
	got, err := ParseLog(in)
	if err != nil {
		t.Fatal(err)
	}
	want := []Commit{
		{Email: "a@x.com", Time: time.Unix(1700000000, 0), Subject: "fix: crash on login", Files: []string{"src/a.go", "src/b.go"}},
		{Email: "b@x.com", Time: time.Unix(1690000000, 0), Subject: "empty commit"},
		{Email: "c@x.com", Time: time.Unix(1680000000, 0), Subject: "start", Files: []string{"my docs/read me.md"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestParseLogErrors(t *testing.T) {
	for _, in := range []string{
		"\x1ea@x.com\x1fnot-a-number\x1fsubject\n",
		"\x1eonly-one-field\n",
	} {
		if _, err := ParseLog(in); err == nil {
			t.Errorf("ParseLog(%q) = nil error, want an error", in)
		}
	}
}

func TestLogAndAuthorOnRealRepo(t *testing.T) {
	gittest.New(t)
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	gittest.Write(t, "a.go", 1)
	gittest.CommitAs(t, "start", "a@x.com", t0)
	gittest.Write(t, "a.go", 2)
	gittest.Write(t, "b.go", 1)
	gittest.CommitAs(t, "fix: crash", "b@x.com", t0.Add(24*time.Hour))

	got, err := Log("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d commits, want 2: %+v", len(got), got)
	}
	newest := got[0]
	if newest.Email != "b@x.com" || newest.Subject != "fix: crash" ||
		!newest.Time.Equal(t0.Add(24*time.Hour)) || !reflect.DeepEqual(newest.Files, []string{"a.go", "b.go"}) {
		t.Errorf("newest commit = %+v", newest)
	}

	author, err := Author("HEAD")
	if err != nil || author != "b@x.com" {
		t.Errorf("Author = %q, %v; want b@x.com", author, err)
	}
}

// .mailmap tells git that several emails are one person; Log and Author must respect it.
func TestMailmapMergesEmails(t *testing.T) {
	gittest.New(t)
	gittest.WriteText(t, ".mailmap", "Dev <dev@work.com> <dev@home.com>\n")
	gittest.CommitAs(t, "from home", "dev@home.com", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	got, err := Log("HEAD")
	if err != nil || got[0].Email != "dev@work.com" {
		t.Errorf("Log email = %q, %v; want dev@work.com", got[0].Email, err)
	}
	if author, err := Author("HEAD"); err != nil || author != "dev@work.com" {
		t.Errorf("Author = %q, %v; want dev@work.com", author, err)
	}
}

func TestLogRejectsOptionLikeRevision(t *testing.T) {
	if _, err := Log("--all"); err != ErrBadRange {
		t.Errorf("err = %v, want ErrBadRange", err)
	}
}

func TestShow(t *testing.T) {
	gittest.New(t)
	gittest.WriteText(t, ".prisk.json", `{"sensitive":["infra"]}`)
	gittest.Commit(t, "add config")
	gittest.WriteText(t, ".prisk.json", `changed on disk, not committed`)

	data, err := Show("HEAD", ".prisk.json")
	if err != nil || string(data) != `{"sensitive":["infra"]}` {
		t.Errorf("Show = %q, %v; want the committed contents", data, err)
	}
	data, err = Show("HEAD", "missing.json")
	if err != nil || data != nil {
		t.Errorf("missing file: Show = %q, %v; want nil, nil", data, err)
	}
}
