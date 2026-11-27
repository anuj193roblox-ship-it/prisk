# Plan: prisk v1

Implements [SPEC.md](../SPEC.md). Task checklist: [todo.md](todo.md).

## Components and dependency order

```
internal/diff  (git output → []FileChange)      no dependencies
internal/risk  ([]FileChange → score+findings)  depends on diff.FileChange only
cmd/prisk      (CLI: args, output, exit codes)  depends on both
```

`diff` and `risk` don't depend on each other's logic, only on the `FileChange` type,
so the type is defined first and both are built and tested on their own.
The CLI comes last because it only wires them together.

## Order

1. Skeleton: git repo, `go.mod`, an empty CLI that builds.
2. `diff`: parse git's output (pure function, easy to test), then run git.
3. `risk`: file rules (test? source? skipped? sensitive?), then the three checks, then the score.
4. CLI: text and JSON output, exit codes, end-to-end test on a real temporary git repo.
5. README, first commit, push to GitHub.

## Checkpoints

- **After task 2:** prisk can read any real branch's changes correctly.
- **After task 4:** all scoring rules pass their tests. The core logic is done.
- **After task 5:** success criteria 1–7 pass end to end. Stop and review before README and push.

## Risks

| Risk | Mitigation |
|---|---|
| Windows vs Unix paths | git always prints `/`, so use the `path` package (not `filepath`) on git paths. |
| Renamed files print in a tricky format | Use `git diff --numstat -z`: renames come out as clean, separate fields. Tested. |
| Branch name like `--output=x` makes git do something else | Reject ranges starting with `-` before calling git. Tested. |
| End-to-end test needs git | Test skips with a message if git isn't on the PATH. |
| Project folder is inside OneDrive, which can lock files while syncing | If builds fail with "access denied", pause OneDrive sync or move the folder. |
| Commits need a git name and email (not set on this machine) | Ask you before setting them (task 6). |
