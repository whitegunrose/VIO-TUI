# CLAUDE.md

Standing rules and current decisions for Claude Code in VIO-TUI. Keep this file short. Longer material lives in the committed `docs/` files linked below. If something here conflicts with a handoff spec, stop and ask.

## Project

VIO-TUI is a keyboard-driven terminal app (Go + [tview](https://github.com/rivo/tview)/tcell) for students: dashboard panes (Calendar, Courses, Tasks, Today, Notes), a calendar week view with class meeting times, tasks, assignments, and optional Canvas LMS sync.

## Hard rules (never break these)

1. **Never commit or push.** Do not run `git commit`, `git push`, `git merge`, `git rebase`, `git reset --hard`, `git tag`, or anything else that writes history or touches the remote. Edit files, then end with a **Git commands** block for me to run myself (see format below). Read-only git (`status`, `diff`, `log`, `branch --show-current`) is fine.
2. **No Claude attribution, anywhere.** No `Co-Authored-By` trailers, no "Generated with Claude Code" lines, no Claude mentions in commit messages, PR text, code comments, or docs. Commit messages you suggest are plain, as if I wrote them.
3. **Features are user stories, `US-xx`,** numbered as in the VIO Feature Roadmap. Use the number in branch names, suggested commit messages and test names where it helps. Never invent a new US number: if work doesn't fit an existing story, say so and I'll assign one in Chat.
4. **Only build what the handoff spec says.** If the spec is unclear, or the code makes a rule in it impossible, stop and ask rather than guess. Note follow-up ideas at the end instead of building them.
5. **Anything you must follow lives in this file or a committed `docs/` file.** `aidlc-docs/` and `.kiro/` are gitignored local material: you may read them for background, but they are never the source of a rule, and committed code must not point readers to them.

## Commands

The Go module (`module VIO`, Go 1.24.5) lives in `src/`, not the repo root. Run all Go commands from there.

```sh
cd src
go run .              # run the TUI
go build -o ../vio .  # build binary (/vio is gitignored)
gofmt -l .            # must print nothing (CI fails otherwise)
go vet ./...
go test ./...         # unit tests + every fuzz target's seed corpus
go test -race ./...   # CI runs this on Linux
go test ./internal/model -run TestParseMeeting                        # one test
TZ=America/Los_Angeles go test -run '^$' -fuzz '^FuzzParse$' -fuzztime 30s ./internal/dates   # fuzz one target
```

Before saying a change is done: `gofmt -l .` is empty, and `go vet ./...` and `go test ./...` pass. Paste the real output of any failure, don't summarize it away.

## Testing (fuzz / property-based)

Full guide and the list of laws: [docs/testing.md](docs/testing.md).

- Every parser, normalizer, merge or ordering function gets **laws** (rules that hold for every input) checked by Go's built-in fuzzer, in the package's `fuzz_test.go`. Each law has an ID (D1, M3, S4...) listed in `docs/testing.md`. Add the ID in a comment above the check.
- Pair laws with example (table-driven) tests for the specific cases a spec names.
- When the fuzzer finds a failing input, add it to that target's `f.Add` seeds with a comment saying what it found, then fix the code. Every bug fix gets a regression test that fails on the old code.
- Never delete, skip or weaken a test or a law to get green. If a law turns out wrong, stop and ask.
- Fuzz with `TZ=America/Los_Angeles` so daylight-saving bugs can show up.
- `internal/views` and `main.go` wiring are not unit-tested. Keep logic out of them so it can be tested in `model`, `dates`, `store` or `canvas`.
- No third-party property-testing library: use Go's built-in `testing.F` (no `pgregory.net/rapid`).

## Current decisions

The full log with dates and reasons is in [docs/decisions.md](docs/decisions.md). The ones that most often affect code:

- **Undated Canvas assignments stay undated.** Never fill in a placeholder due date. They list last under Upcoming as "no due date yet". (Law S4.)
- **Task priority is optional per task** (US-22). A task with no priority is valid and must sort predictably.
- **Canvas sync** replaces the Canvas assignment list but keeps manual assignments (`Assignment.Manual`), the user's course fields (location, meetings, notes), and the live task and note lists.
- **Homebrew / release packaging is deferred as US-34.** Don't add GoReleaser, a tap, or release workflows unless a spec for US-34 says so.
- **Data safety:** write app files only through `store.writeFileAtomic`, never `os.WriteFile`. Model changes usually need matching changes in `store.normalizeAppData`.
- **Canvas token:** stored encrypted in `config.json`. Never log, print, or put it in test fixtures or error messages.
- **Dependencies:** ask before adding any new module to `go.mod`.

## Code map

Architecture details: [docs/architecture.md](docs/architecture.md). Read the section for a package before changing it.

- `src/main.go`: composition root, Canvas polling goroutine (applies results with `app.QueueUpdateDraw`)
- `src/internal/model`: `AppData` (single source of truth), meetings, agenda, assignment state
- `src/internal/store`: `data.json` / `config.json` persistence, normalization, atomic saves
- `src/internal/canvas`: Canvas REST client and sync/merge
- `src/internal/dates`: plain-language date parsing for forms
- `src/internal/widgets`: dashboard panes, navigation keys, screen router
- `src/internal/views/<feature>`: full-screen pages
- `src/internal/theme`: colors and borders (use `theme.Tag*` constants)
- `docs/*.rst`: package reference docs

## How a build session runs

The handoff loop and the spec template are in [docs/workflow.md](docs/workflow.md). In short:

1. I paste a handoff spec: US number, rules, acceptance criteria, branch.
2. Check you're on the spec's branch (`git branch --show-current`). If not, stop and give me the command to create or switch to it; don't switch yourself.
3. Build it with tests, run the checks above, and fix what fails.
4. End every session with a **Session report**:

```
## Session report: US-xx <title>
Done: <each acceptance criterion: met / not met, and how it's tested>
Files changed: <list>
Tests: <commands run and their result>
Laws added or changed: <IDs>
Open questions / follow-ups: <for Chat>
Suggested CLAUDE.md or docs changes: <if any; don't edit CLAUDE.md's rules yourself>

### Git commands
git status
git add <files>
git commit -m "US-xx: <summary>"
git push -u origin <branch>
```

The commit message has no trailers of any kind.

## Branches and merging

- One branch per story: `us-xx-short-name` (e.g. `us-22-assignment-overlay`), branched from an up-to-date `main`.
- Work reaches `main` through a pull request. I run every step myself, so the **Git commands** block ends with what gets the branch onto GitHub and merged: push the branch, open a PR into `main`, merge it, then update local `main` and delete the merged branch:

```
git push -u origin us-xx-short-name
# open the PR on GitHub (or: gh pr create --base main --fill), review, merge
git switch main && git pull
git branch -d us-xx-short-name
```
