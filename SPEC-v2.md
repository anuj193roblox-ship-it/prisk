# Spec: prisk v2 — history checks and team config

Builds on [SPEC.md](SPEC.md) (v1). Everything in v1 stays true unless this file says otherwise.

## Objective

v1 only looks at *what* changed. v2 also looks at the **history** behind it:

- **Hotspots:** has this file needed a lot of bug fixes lately? Code that broke recently tends to break again.
- **Familiarity:** has the PR's author worked on these files before? People make more mistakes in code they don't know.

And it lets each team set its own rules with a **config file**, because every company names its folders differently.

**User story:** "My team's risky code lives in `infra/` and `ledger/`, not `auth/`. I commit a config file once,
and from then on prisk flags PRs that touch those folders, the files we keep fixing, and code the author has never touched."

### Assumptions (correct any of these before we build)

1. The CLI stays the same: `prisk score <base>...<head> [--json]`. No new flags.
2. Still **zero third-party dependencies**, which is why the config file is JSON, not YAML (see Open Question 1).
3. "History" means the history of the **base** branch (for example `main`), not the PR's own commits.
   A PR that fixes its own bug five times shouldn't make its files look like hotspots.
4. Time windows are measured back from the **newest commit on the base branch**, not from today.
   The same range then always gives the same score, even when run months later.
5. The PR's author = the author email of the **head** commit. *(Shortcut: PRs with several authors
   are judged by the last one. Revisit in v3, where GitHub tells us the PR author directly.)*

## The two new checks

Both read the base branch's history: its newest **5,000 commits** (non-merge), limited to the **last 365 days**.
*(Shortcut: the 5,000 cap keeps huge repos fast. Raise it if a repo commits more than that in a year.)*

Both look at **source files only**, using the same rule as the test-gap check. Docs, config and test files
often ride along in fix commits for code (a `fix(proxy): …` commit that also updates a docs page),
so counting them flagged files that weren't the risky part. *(Changed after running v2 on real repos.)*

Author emails go through git's `.mailmap`, so one person committing from several emails counts as one person
in repos that have a `.mailmap`. *(Known limit: without one, a second email looks like a different author.)*

### 4. Hotspots: +15

- A **fix commit** is one whose message (first line) contains, as a whole word, any of:
  `fix`, `fixes`, `fixed`, `bug`, `bugfix`, `hotfix`, `revert`. Case doesn't matter.
  `fix: login crash` counts; `prefix the names` does not.
- A changed file is a **hotspot** if **3 or more** fix commits touched it in the **last 90 days**.
- +15 once if any changed file is a hotspot.
- Reason lists up to 3, most-fixed first: `changes bug-prone files: worker/queue.go (5 fixes in 90 days)`
- Skipped files (lockfiles, binaries, config `skip` globs) never count.

### 5. Unfamiliar author: +10

- Look only at changed files that **already existed** in the base history (brand-new files have no history to know).
- +10 if the author made **none** of the history commits that touched any of those files.
- Author emails are compared ignoring upper/lower case.
- Reason lists up to 3: `author hasn't changed these files in the last year: billing/invoice.go`
- If no changed file has history, the check doesn't fire.

### Scoring

Unchanged: `min(100, sum)`, LOW 0–29, MEDIUM 30–59, HIGH 60–100.
The maximum is now 105 (30 + 20 + 30 + 15 + 10), so the 100 cap really matters now.

## Config file: `.prisk.json`

Optional. Lives at the repo root.

```json
{
  "sensitive": ["infra", "ledger", ".github/workflows"],
  "skip": ["vendor/*", "*.pb.go", "docs/generated/*"]
}
```

| Key | Meaning | If missing |
|---|---|---|
| `sensitive` | Folders that count as sensitive. **Replaces** the built-in list. Same whole-folder matching as v1. | v1's built-in list |
| `skip` | Extra files to ignore in every check, as [glob patterns](https://pkg.go.dev/path#Match). A pattern containing `/` is matched against the full path and each parent folder (like `.gitignore`), so `vendor/*` skips everything under `vendor/`; one without `/` is matched against the file name. | Nothing extra skipped |

- **Read from the base branch, not the PR** (`git show <base>:.prisk.json`).
  Otherwise a PR could edit the config to turn off the checks that would flag it.
  A change to `.prisk.json` takes effect once it's merged into `main`.
- Unknown keys, wrong types or broken JSON are an **error** (exit 1), not silently ignored,
  so a typo like `"sensitve"` can't quietly switch a rule off.
- A bad glob pattern is an error too.

## Output

Same formats as v1. New findings use signals `"hotspot"` and `"unfamiliar"` in JSON.

```
Risk: HIGH (95/100)
  +30  touches sensitive code: ledger/post.go
  +20  source code changed but no tests were added or changed
  +20  310 lines changed across 4 files
  +15  changes bug-prone files: ledger/post.go (4 fixes in 90 days)
  +10  author hasn't changed these files in the last year: ledger/post.go
```

### Errors and exit codes

Same as v1, plus:

| Situation | Message | Exit code |
|---|---|---|
| `.prisk.json` on base is invalid | `prisk: .prisk.json: <what's wrong>` | 1 |

## Design changes

```
internal/diff     + Log(base): reads base history into []Commit{Email, Time, Subject, Files}
                  + Show(rev, path): reads one file at a revision (for the config)
internal/config   NEW: parse and validate .prisk.json (pure: bytes in, Config out)
internal/risk     Score takes one Input struct: files, history, author, config.
                  Still pure: no git, no disk.
cmd/prisk         wires it together
```

- `git log` is called with `-c core.quotePath=false` so file names with non-English characters come out as-is.
  *(Shortcut: file names containing a line break are not supported.)*
- The range is split to find base and head: `A...B` and `A..B` give base `A`, head `B`.
  An empty side means `HEAD`. A range with no dots (`prisk score abc123`) gives base `abc123`, head `HEAD`.

## Testing Strategy

As in v1: tests first, table-driven, pure logic tested on plain data, end-to-end tests on real throwaway repos.
The `gittest` helper gets a way to make commits **with a chosen date and author**, so tests can check
"fix commits older than 90 days don't count" and "a different author made this commit".

## Workflow

This time we work like a real team:
1. All v2 work happens on the `v2` branch (already created).
2. When done, push the branch and **open a pull request** on GitHub.
3. Run prisk on its own PR (`prisk score main...v2`) and put the result in the PR description.
4. Review it, then merge it into `main`.

## Boundaries

Same as v1. Also:
- **Never** read the config from the PR's head.
- **Ask first:** opening the PR, merging it, changing any points or thresholds above.

## Success Criteria

v2 is done when all of these are true:

1. `go test ./...` and `go vet ./...` pass, and `gofmt -l .` prints nothing. All v1 tests still pass.
2. A file with 3 fix commits on `main` in the last 90 days → `hotspot` +15, reason shows the count.
3. 2 fix commits, or 3 fix commits older than 90 days → no hotspot. `prefix` in a message is not a fix.
4. Fix commits made on the PR branch itself don't make a hotspot.
5. Author never touched the changed existing files → `unfamiliar` +10. Author touched one of them → no finding.
   Only brand-new files → no finding. `Dev@X.com` and `dev@x.com` count as the same person.
6. `.prisk.json` on `main` with `"sensitive": ["infra"]` → `infra/` is flagged and `auth/` no longer is.
   A `skip` pattern makes a file invisible to every check.
7. A `.prisk.json` that exists only on the PR branch has no effect.
8. Broken JSON, an unknown key, or a bad glob → exit 1 with `prisk: .prisk.json: ...`.
9. All five checks firing → score is exactly 100.
9b. Docs, config and test files never trigger `hotspot` or `unfamiliar`. A `.mailmap` entry merges two emails into one author.
10. Running on the `headroom` repo (~3,800 commits) takes under 3 seconds.
11. README documents the new checks and the config file. The v2 PR is open on GitHub with prisk's score of itself.

## Open Questions

1. **Config format: JSON or YAML?** I recommend **JSON**. It keeps prisk at zero dependencies, which is a nice
   thing to say in interviews, and Go reads it in one line. YAML looks friendlier and is what most CI tools use,
   but it needs a third-party library. We can add YAML later without breaking JSON configs.
2. **Are the new points OK?** Hotspot +15 and unfamiliar +10 are lower than sensitive (+30) on purpose:
   they're good hints but noisier. v4 will tune them from real data.
