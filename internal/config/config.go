// Package config reads a team's .prisk.json settings.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// File is where the settings live, at the repo root.
const File = ".prisk.json"

// DefaultSensitive is used when the config doesn't list its own sensitive folders.
var DefaultSensitive = []string{"auth", "payments", "billing", "migrations", "security", ".github/workflows"}

type Config struct {
	Sensitive []string `json:"sensitive"` // folder names; replaces DefaultSensitive when set
	Skip      []string `json:"skip"`      // glob patterns for files every check ignores
}

// Default returns the settings used when a repo has no config file.
func Default() Config {
	return Config{Sensitive: DefaultSensitive}
}

// Parse reads and checks a config file. Empty input means "no file": the defaults.
// Mistakes are errors, not silently ignored, so a typo can't quietly switch a rule off.
func Parse(data []byte) (Config, error) {
	var cfg Config
	if len(bytes.TrimSpace(data)) > 0 {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&cfg); err != nil {
			return Config{}, err
		}
		if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return Config{}, errors.New("unexpected data after the settings object")
		}
	}
	if cfg.Sensitive == nil {
		return Config{Sensitive: DefaultSensitive, Skip: cfg.Skip}, validateSkip(cfg.Skip)
	}
	for i, dir := range cfg.Sensitive {
		dir = strings.Trim(dir, "/")
		if dir == "" {
			return Config{}, fmt.Errorf("sensitive: empty folder name at position %d", i+1)
		}
		cfg.Sensitive[i] = dir
	}
	return cfg, validateSkip(cfg.Skip)
}

func validateSkip(patterns []string) error {
	for _, p := range patterns {
		if _, err := path.Match(p, ""); err != nil {
			return fmt.Errorf("skip: bad pattern %q", p)
		}
	}
	return nil
}
