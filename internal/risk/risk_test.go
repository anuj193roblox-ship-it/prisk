package risk

import (
	"strings"
	"testing"

	"github.com/anuj193roblox-ship-it/prisk/internal/diff"
)

type fc = diff.FileChange

// points returns each finding's points keyed by signal name.
func points(r Result) map[string]int {
	m := map[string]int{}
	for _, f := range r.Findings {
		m[f.Signal] = f.Points
	}
	return m
}

func TestScoreSpecExamples(t *testing.T) {
	cases := []struct {
		name  string
		files []fc
		score int
		level string
		want  map[string]int
	}{
		{"auth change, no tests (spec #2)",
			[]fc{{Path: "src/auth/login.go", Added: 250}},
			70, "HIGH", map[string]int{"sensitive": 30, "test_gap": 20, "size": 20}},
		{"readme only (spec #3)",
			[]fc{{Path: "README.md", Added: 30}},
			0, "LOW", map[string]int{}},
		{"author/ is not auth/ (spec #4)",
			[]fc{{Path: "author/bio.go", Added: 10}},
			20, "LOW", map[string]int{"test_gap": 20}},
		{"lockfile ignored (spec #5)",
			[]fc{{Path: "package-lock.json", Added: 4000, Deleted: 1000}},
			0, "LOW", map[string]int{}},
		{"code with tests: no gap",
			[]fc{{Path: "cart.go", Added: 20}, {Path: "cart_test.go", Added: 20}},
			0, "LOW", map[string]int{}},
		{"tests only: no gap",
			[]fc{{Path: "cart_test.go", Added: 20}},
			0, "LOW", map[string]int{}},
		{"binary in auth/ is skipped",
			[]fc{{Path: "auth/logo.png", Binary: true}},
			0, "LOW", map[string]int{}},
		{"everything at once",
			[]fc{{Path: "payments/charge.go", Added: 600}},
			80, "HIGH", map[string]int{"sensitive": 30, "test_gap": 20, "size": 30}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Score(c.files)
			if r.Score != c.score || r.Level != c.level {
				t.Errorf("got %d %s, want %d %s", r.Score, r.Level, c.score, c.level)
			}
			got := points(r)
			if len(got) != len(c.want) {
				t.Errorf("findings %v, want %v", got, c.want)
			}
			for sig, p := range c.want {
				if got[sig] != p {
					t.Errorf("%s = %d points, want %d", sig, got[sig], p)
				}
			}
		})
	}
}

func TestSizeBoundaries(t *testing.T) {
	cases := map[int]int{0: 0, 49: 0, 50: 10, 199: 10, 200: 20, 499: 20, 500: 30, 9000: 30}
	for lines, want := range cases {
		// A docs file, so only the size check can fire.
		r := Score([]fc{{Path: "docs/guide.md", Added: lines}})
		if got := points(r)["size"]; got != want {
			t.Errorf("%d lines: size = %d, want %d", lines, got, want)
		}
	}
}

func TestSizeCountsAddedAndDeletedAcrossFiles(t *testing.T) {
	r := Score([]fc{{Path: "a.md", Added: 30, Deleted: 10}, {Path: "b.md", Deleted: 20}})
	if got := r.Findings[0].Reason; got != "60 lines changed across 2 files" {
		t.Errorf("reason = %q", got)
	}
}

func TestLevelBoundaries(t *testing.T) {
	cases := map[int]string{0: "LOW", 29: "LOW", 30: "MEDIUM", 59: "MEDIUM", 60: "HIGH", 100: "HIGH"}
	for score, want := range cases {
		if got := level(score); got != want {
			t.Errorf("level(%d) = %s, want %s", score, got, want)
		}
	}
}

func TestFindingsSortedHighestFirst(t *testing.T) {
	r := Score([]fc{{Path: "auth/a.go", Added: 600}})
	for i := 1; i < len(r.Findings); i++ {
		if r.Findings[i].Points > r.Findings[i-1].Points {
			t.Fatalf("not sorted: %+v", r.Findings)
		}
	}
}

func TestSensitiveReasonListsAtMostThree(t *testing.T) {
	r := Score([]fc{
		{Path: "auth/a.md"}, {Path: "auth/b.md"}, {Path: "auth/c.md"}, {Path: "auth/d.md"}, {Path: "auth/e.md"},
	})
	reason := r.Findings[0].Reason
	if !strings.Contains(reason, "auth/a.md, auth/b.md, auth/c.md") || !strings.HasSuffix(reason, "(+2 more)") {
		t.Errorf("reason = %q", reason)
	}
}

func TestNoFindingsIsEmptyListNotNil(t *testing.T) {
	// Matters for JSON: an empty list prints [], nil prints null.
	if r := Score(nil); r.Findings == nil {
		t.Error("Findings is nil, want empty slice")
	}
}
