# Spec: prisk v1 — PR risk scorer (local CLI)

## Objective

`prisk` reads the changes in a git branch and gives them a risk score from 0 to 100,
with a reason for every point, so a reviewer knows which pull requests need careful attention.

**User:** a developer or reviewer on their own machine, inside a git repo.

**User story:** "Before I review or merge `feature`, I run `prisk score main...feature`
and see in two seconds whether it is LOW, MEDIUM or HIGH risk, and why."

v1 is the smallest useful version: three checks, run locally. History-based checks
(hotspots, author familiarity), the config file, the GitHub Action and learned weights
come in v2–v4 and are **out of scope** here.

### Assumptions (correct any of these before we build)

1. Language is Go, standard library only — **zero third-party dependencies** in v1.
2. v1 uses built-in default rules. The `.prisk.yml` config file comes in v2.
3. The project lives in its own folder `prisk/` and becomes its own git repo, later pushed to your GitHub.
4. It runs on Windows, macOS and Linux; it needs `git` on the PATH.
5. It only *reads* git data. It never runs, builds or imports code from the repo it scores.

## What v1 does

### Command

```
prisk score <base>...<head> [--json]
```

- `<base>...<head>` — a git range, e.g. `main...feature`. Three dots = compare `head`
  against the point where it branched off `base`, so only the branch's own changes count.
- `--json` — print machine-readable JSON instead of the text report.
- Runs in the current directory, which must be inside a git repo.

### The three checks (signals)

Each check returns `points` and a human-readable `reason`. A check that does not fire returns 0 points and is not shown.

**1. Size** — total lines changed (added + deleted), not counting skipped files.

| Lines changed | Points |
|---|---|
| 0–49 | 0 |
| 50–199 | 10 |
| 200–499 | 20 |
| 500+ | 30 |

Reason example: `412 lines changed across 9 files`

**2. Sensitive paths** — +30 (once, not per file) if any changed file sits under one of these folders:
`auth/`, `payments/`, `billing/`, `migrations/`, `security/`, `.github/workflows/`.

- Match by whole folder name at any depth: `src/auth/login.go` matches, `author/bio.go` does **not**.
- Reason lists up to 3 matching files: `touches sensitive code: auth/login.go, migrations/004.sql`

**3. Test gap** — +20 if at least one **source file** changed and **zero test files** changed.

- Source file = extension in: `.go .py .js .jsx .ts .tsx .java .kt .rb .rs .c .h .cpp .cs .php .swift`
- Test file = any of: name ends `_test.go`, `_test.py`, or starts `test_` (`.py`);
  name contains `.test.` or `.spec.`; or path has a folder named `test`, `tests` or `__tests__`.
- A test file is never also counted as a source file.
- Docs, config and images (`.md`, `.yml`, `.png`…) are neither, so a docs-only PR does not fire this.
- Reason: `source code changed but no tests were added or changed`

### Skipped files

Not counted by any check:
- **Binary files** (git reports `-` for their line counts).
- **Lockfiles**: `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `go.sum`, `Cargo.lock`, `poetry.lock`.

Renamed files are counted under their **new** path.

### Scoring

- `score = min(100, sum of points)` (v1 max is 80; the cap exists for v2's extra checks)
- Level: **LOW** 0–29, **MEDIUM** 30–59, **HIGH** 60–100

### Output

Text (default), findings sorted by points, highest first:

```
Risk: HIGH (70/100)
  +30  touches sensitive code: auth/login.go
  +20  source code changed but no tests were added or changed
  +20  412 lines changed across 9 files
```

No findings: `Risk: LOW (0/100)` followed by `  no risk signals found`.

JSON (`--json`):

```json
{
  "score": 70,
  "level": "HIGH",
  "findings": [
    {"signal": "sensitive", "points": 30, "reason": "touches sensitive code: auth/login.go"}
  ]
}
```

### Errors and exit codes

| Situation | Message (stderr) | Exit code |
|---|---|---|
| Success | — | 0 |
| Wrong arguments / no range | usage text | 2 |
| Range starts with `-` | `invalid range` (stops git option injection) | 2 |
| Not a git repo, unknown branch, git missing | git's own error, prefixed `prisk:` | 1 |

## Tech Stack

- Go (latest stable; needs installing — see Open Questions)
- Standard library only: `os/exec`, `flag`, `encoding/json`, `testing`, `path`, `strings`, `sort`
- Git is called as a subprocess: `git diff --numstat -z <range>`

## Commands

```
Build:   go build -o prisk.exe ./cmd/prisk
Test:    go test ./...
Vet:     go vet ./...
Format:  gofmt -l .        (must print nothing)
Run:     ./prisk.exe score main...feature
```

## Project Structure

```
prisk/
  SPEC.md              this file
  README.md            what it is, how to install and use it
  go.mod               module definition
  cmd/prisk/main.go    CLI: parse arguments, wire the pieces, print, exit code
  internal/diff/       run git, parse its output into []FileChange
  internal/risk/       the three checks + scoring (pure functions, no git, no I/O)
```

`internal/risk` never calls git. It takes plain data in and returns plain data out,
which is what makes it easy to test.

## Code Style

`gofmt` formatting, small functions, plain data types, errors returned (never ignored).

```go
type FileChange struct {
	Path    string
	Added   int
	Deleted int
}

type Finding struct {
	Signal string `json:"signal"`
	Points int    `json:"points"`
	Reason string `json:"reason"`
}

// Each check has the same shape so scoring can loop over them.
func SizeCheck(files []FileChange) Finding { ... }
```

- Names: `CamelCase` for exported, short lowercase for locals.
- Comments explain *why*, not *what*.
- Git is called with `exec.Command("git", "diff", "--numstat", "-z", rng)`, args passed separately, never built into a shell string.

## Testing Strategy

- Framework: Go's built-in `testing` package, **table-driven tests** (a list of input → expected output cases).
- `internal/diff`: parser tested on literal git output strings, including binary files and renames.
- `internal/risk`: every check tested on hand-made `[]FileChange` lists, including the edge cases above
  (`author/` is not `auth/`, docs-only PR, test-only PR, exactly 50 and 500 lines, lockfile skipped).
- One end-to-end test: creates a temporary git repo, commits on a branch, runs the full command, checks the output.
- Tests are written **before** the code that makes them pass (TDD).

## Boundaries

- **Always:** run `go test ./...` and `gofmt` before every commit; keep `internal/risk` free of I/O; return errors with context.
- **Ask first:** adding any dependency; changing points or thresholds in this spec; changing the CLI shape.
- **Never:** execute or import code from the repo being scored; build shell command strings; make network calls; commit secrets.

## Success Criteria

v1 is done when all of these are true:

1. `go test ./...`, `go vet ./...` pass and `gofmt -l .` prints nothing.
2. On a branch that changes `src/auth/login.go` (+250 lines) with no tests, prisk prints
   `Risk: HIGH (70/100)` with the three reasons above.
3. On a branch that only edits `README.md`, prisk prints `Risk: LOW (0/100)`.
4. A branch changing `author/bio.go` does **not** trigger the sensitive check.
5. A 5,000-line `package-lock.json` change adds 0 size points.
6. `--json` output parses as valid JSON with `score`, `level`, `findings`.
7. Outside a git repo, or with `--foo` as the range, prisk exits non-zero with a clear message.
8. The README shows install, usage and a real example output.

## Open Questions

1. **Install Go?** It isn't on this machine. I can install it with `winget install GoLang.Go` — OK to do that?
2. **GitHub username?** Needed for the module name `github.com/<you>/prisk`.
3. **Are the points and thresholds OK** as a starting point? (We'll tune them with real data in v4.)
