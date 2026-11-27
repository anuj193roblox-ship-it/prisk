// Command prisk scores how risky a git branch's changes are.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/anuj193roblox-ship-it/prisk/internal/diff"
	"github.com/anuj193roblox-ship-it/prisk/internal/risk"
)

const usage = "usage: prisk score <base>...<head> [--json]"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main without the os calls, so tests can check output and exit codes.
// Exit codes: 0 ok, 1 git failed, 2 bad arguments.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "score" {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintln(stderr, usage) }
	asJSON := fs.Bool("json", false, "print JSON instead of text")

	// Go's flag parser stops at the first non-flag, so parse flags on both
	// sides of the range: "score --json main...x" and "score main...x --json".
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}
	rng := fs.Arg(0)
	if err := fs.Parse(fs.Args()[1:]); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fs.Usage()
		return 2
	}

	files, err := diff.Load(rng)
	if err != nil {
		fmt.Fprintln(stderr, "prisk:", err)
		if errors.Is(err, diff.ErrBadRange) {
			return 2
		}
		return 1
	}
	res := risk.Score(files)

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintln(stderr, "prisk:", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "Risk: %s (%d/100)\n", res.Level, res.Score)
	if len(res.Findings) == 0 {
		fmt.Fprintln(stdout, "  no risk signals found")
	}
	for _, f := range res.Findings {
		fmt.Fprintf(stdout, "  +%d  %s\n", f.Points, f.Reason)
	}
	return 0
}
