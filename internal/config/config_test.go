package config

import (
	"reflect"
	"testing"
)

func TestParseValid(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Config
	}{
		{"no file", "", Config{Sensitive: DefaultSensitive}},
		{"empty object", "{}", Config{Sensitive: DefaultSensitive}},
		{"spec example",
			`{"sensitive": ["infra", "ledger", ".github/workflows"], "skip": ["vendor/*", "*.pb.go"]}`,
			Config{Sensitive: []string{"infra", "ledger", ".github/workflows"}, Skip: []string{"vendor/*", "*.pb.go"}}},
		{"slashes trimmed", `{"sensitive": ["auth/", "/infra/"]}`,
			Config{Sensitive: []string{"auth", "infra"}}},
		{"empty list means nothing is sensitive", `{"sensitive": []}`,
			Config{Sensitive: []string{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Parse([]byte(c.in))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	for name, in := range map[string]string{
		"typo in key":       `{"sensitve": ["auth"]}`,
		"wrong type":        `{"sensitive": "auth"}`,
		"broken JSON":       `{"sensitive": [`,
		"bad glob":          `{"skip": ["[a-"]}`,
		"empty folder name": `{"sensitive": ["/"]}`,
		"two objects":       `{} {}`,
	} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: Parse(%q) = nil error, want an error", name, in)
		}
	}
}
