# prisk

**Score how risky a pull request is, and say why.**

Reviewers get more pull requests than they can read carefully, and in the queue every PR looks the same.
But a 3-line change to login code is far more dangerous than 300 lines of docs.
`prisk` reads a branch's changes and gives them a risk score from 0 to 100, with a reason for every point,
so reviewers know where to spend their attention.

```
$ prisk score main...feature
Risk: HIGH (70/100)
  +30  touches sensitive code: src/auth/login.go
  +20  source code changed but no tests were added or changed
  +20  250 lines changed across 1 file
```

## Install

Needs [Go](https://go.dev/dl/) 1.27+ and `git`.

```
go install github.com/anuj193roblox-ship-it/prisk/cmd/prisk@latest
```

## Usage

Run it inside a git repo:

```
prisk score <base>...<head>          # text report
prisk score <base>...<head> --json   # JSON, for scripts and CI
```

`main...feature` (three dots) compares `feature` against the point where it branched off `main`,
so only the branch's own changes are scored, not whatever landed on `main` since.

```json
{
  "score": 70,
  "level": "HIGH",
  "findings": [
    { "signal": "sensitive", "points": 30, "reason": "touches sensitive code: src/auth/login.go" },
    { "signal": "test_gap", "points": 20, "reason": "source code changed but no tests were added or changed" },
    { "signal": "size", "points": 20, "reason": "250 lines changed across 1 file" }
  ]
}
```

| Exit code | Meaning |
|---|---|
| 0 | Scored successfully |
| 1 | git failed (not a repo, unknown branch) |
| 2 | Bad arguments |

## How the score works

Three checks, each adding points with a reason. Total is capped at 100.
**LOW** 0–29 · **MEDIUM** 30–59 · **HIGH** 60–100

| Check | Fires when | Points |
|---|---|---|
| **Sensitive paths** | Any file under `auth/`, `payments/`, `billing/`, `migrations/`, `security/`, `.github/workflows/` (whole folder names, any depth, so `author/` doesn't count) | +30 |
| **Test gap** | Source code changed but no test files did | +20 |
| **Size** | Lines added + deleted: 50+ / 200+ / 500+ | +10 / +20 / +30 |

Lockfiles (`package-lock.json`, `go.sum`, …) and binary files are ignored: they're large but say nothing about risk.

### Why a simple rubric instead of machine learning?

- **It explains itself.** "+30 because it touches auth/" is something a reviewer can check and trust. A bare probability isn't.
- **It works on day one**, in any repo, with no training data.
- Research on predicting bugs from changes ("just-in-time defect prediction") finds that size, spread and
  touching risky areas are among the strongest signals, so a rubric built on them is a reasonable baseline.
  Learning the weights from a repo's own history is on the roadmap.

## Design

```
cmd/prisk       CLI: arguments, output, exit codes
internal/diff   runs `git diff --numstat -z` and parses it into a list of changed files
internal/risk   the checks and scoring: pure functions, no git, no disk
```

- **Read-only and safe.** prisk only reads git metadata. It never builds, runs or imports code from the repo it scores,
  and ranges starting with `-` are rejected so a branch name can't be smuggled in as a git option.
- **No dependencies.** Standard library only.
- **Tested.** Rules are table-tested on plain data. End-to-end tests build real throwaway git repos.

## Development

```
go test ./...
go vet ./...
gofmt -l .
```

[SPEC.md](SPEC.md) is the v1 specification; [tasks/](tasks/) holds the build plan.

## Roadmap

- [x] **v1**: local CLI with size, sensitive-path and test-gap checks
- [ ] **v2**: hotspots (files that change and break often) and author familiarity, from `git log`; `.prisk.yml` config
- [ ] **v3**: GitHub Action that posts the score as a PR comment
- [ ] **v4**: learn weights from the repo's history of reverted and fixed changes, and measure how well the score predicts them
