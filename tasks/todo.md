# prisk v1 tasks

Each task: write the tests first, make them pass, run `go test ./... && go vet ./... && gofmt -l .`

- [x] **1. Project skeleton**
  - Acceptance: `git init`; `go.mod` with module `github.com/anuj193roblox-ship-it/prisk`; `cmd/prisk/main.go` prints usage and exits 2; `.gitignore` ignores `prisk.exe`.
  - Verify: `go build ./...` and `go vet ./...` succeed; running with no args prints usage, exit code 2.
  - Files: `go.mod`, `cmd/prisk/main.go`, `.gitignore`

- [x] **2. Parse git's numstat output**
  - Acceptance: `diff.Parse(out string) ([]FileChange, error)` handles normal files, binary files (`-`), renames (new path kept), empty output, malformed lines (error).
  - Verify: table-driven tests in `diff_test.go` pass.
  - Files: `internal/diff/diff.go`, `internal/diff/diff_test.go`

- [x] **3. Run git safely**
  - Acceptance: `diff.Load(rng string)` rejects ranges starting with `-`, runs `git diff --numstat -z <rng>`, returns git's error text on failure.
  - Verify: test for the `-` rejection; test against a temp git repo (skipped if no git).
  - Files: `internal/diff/diff.go`, `internal/diff/diff_test.go`

- [x] **4. File rules**
  - Acceptance: `isSkipped`, `isTest`, `isSource`, `isSensitive` match every rule in the spec, including `author/` ≠ `auth/` and test files never counted as source.
  - Verify: table-driven tests pass.
  - Files: `internal/risk/files.go`, `internal/risk/files_test.go`

- [x] **5. Checks and score**
  - Acceptance: `risk.Score([]FileChange) Result` runs Size, Sensitive, TestGap; caps at 100; sets level; sorts findings by points (highest first); skipped files ignored.
  - Verify: tests for spec criteria 2–5 at data level, plus boundaries (49/50, 499/500 lines; 29/30, 59/60 score).
  - Files: `internal/risk/risk.go`, `internal/risk/risk_test.go`

- [x] **6. CLI output and exit codes** — checkpoint before this: review core logic
  - Acceptance: `prisk score <range> [--json]` prints the spec's text or JSON; exit codes 0/1/2 as in the spec.
  - Verify: end-to-end test builds a temp repo with branches for spec criteria 2, 3, 6, 7 and checks output and exit codes.
  - Files: `cmd/prisk/main.go`, `cmd/prisk/main_test.go`
  - Notes: Go's `flag` stops at the first non-flag, so `score main...x --json` needs the remaining args parsed again.
    Put logic in `run(args, stdout, stderr) int` so tests check exit codes without building a binary.

- [ ] **7. README, first commit, push** — ask first: git name/email, creating the GitHub repo
  - Acceptance: README has what/why, install, usage, real example output; code is committed and on GitHub.
  - Verify: fresh `go install github.com/anuj193roblox-ship-it/prisk/cmd/prisk@latest` works.
  - Files: `README.md`
