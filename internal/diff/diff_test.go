package diff

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		in   string // raw output of `git diff --numstat -z`
		want []FileChange
	}{
		{"empty", "", nil},
		{"one file", "12\t3\tapp/theme.go\x00",
			[]FileChange{{Path: "app/theme.go", Added: 12, Deleted: 3}}},
		{"two files", "1\t0\ta.go\x0040\t2\tauth/login.go\x00",
			[]FileChange{{Path: "a.go", Added: 1}, {Path: "auth/login.go", Added: 40, Deleted: 2}}},
		{"path with spaces", "5\t5\tmy docs/read me.md\x00",
			[]FileChange{{Path: "my docs/read me.md", Added: 5, Deleted: 5}}},
		{"binary", "-\t-\tlogo.png\x00",
			[]FileChange{{Path: "logo.png", Binary: true}}},
		// -z prints renames as an empty path, then old and new paths as their own fields.
		{"rename keeps new path", "2\t1\t\x00old/name.go\x00new/name.go\x00",
			[]FileChange{{Path: "new/name.go", Added: 2, Deleted: 1}}},
		{"rename then normal file", "0\t0\t\x00a.go\x00b.go\x007\t0\tc.go\x00",
			[]FileChange{{Path: "b.go"}, {Path: "c.go", Added: 7}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.in)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.in, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q)\n got  %+v\n want %+v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	bad := []string{
		"garbage\x00",                 // no tabs
		"x\t3\ta.go\x00",              // non-number count
		"1\t1\t\x00only-old-path\x00", // rename missing its new path
	}
	for _, in := range bad {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) = nil error, want an error", in)
		}
	}
}
