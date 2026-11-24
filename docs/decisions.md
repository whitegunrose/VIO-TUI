# VIO-TUI decisions

Settled decisions from planning in Chat. Newest first. Each entry: date, story, the decision, and why. `CLAUDE.md` lists only the ones that most often affect code; this file is the full record. When a decision is reversed, don't delete it: add a new entry and mark the old one "Replaced by …".

Format:

```
## YYYY-MM-DD · US-xx · <short title>
Decision: …
Why: …
Affects: <packages, laws, files>
```

---

## 2026-11-24 · (workflow) · Chat decides, Claude Code builds
Decision: Planning, the VIO Feature Roadmap and study guides live in Claude Chat. Claude Code builds from a handoff spec and never commits or pushes; I run git myself. No Claude attribution in any commit, PR or file. Rules Claude Code must follow live in `CLAUDE.md` or committed `docs/`, never only in `aidlc-docs/`.
Why: Claude Code can't see Chat history; `aidlc-docs/` is gitignored.
Affects: `CLAUDE.md`, `docs/workflow.md`

## 2026-11-20 · US-?? (TODO) · Undated Canvas assignments stay undated
Decision: An assignment with no due date in Canvas keeps no due date until the professor sets one; the next sync picks it up. The Assignments page lists them last under Upcoming as "no due date yet".
Why: `normalizeAppData` used to give them a sample date, which showed fake deadlines.
Affects: `internal/store` (normalize), Assignments page; law S4, `TestNormalizeKeepsUndatedAssignment`

## 2026-11-20 · US-32 · Fuzz/property-based testing with Go's built-in fuzzer
Decision: Core logic is checked by laws in `fuzz_test.go` per package, using `testing.F`; seeds run in CI on every push, full fuzzing weekly in `.github/workflows/fuzz.yml`.
Why: Parsers and normalizers take arbitrary user and Canvas input; fuzzing found eight real bugs on day one.
Affects: `docs/testing.md`

## 2026-11-19 · US-22 · Task priority is optional per task
Decision: Priority applies to personal tasks as well as Canvas assignments, so both lists sort the same way, but it is optional on each task.
Why: Open question 1 in the roadmap plan.
Affects: `internal/model`, `internal/store` (normalize), tasks views. TODO: record the priority levels and how "no priority" sorts once settled.

## 2026-11-19 · US-34 · Homebrew release deferred
Decision: GoReleaser + Homebrew tap is deferred until VIO has a tagged release.
Why: TODO: add the reason from the Homebrew research notes.
Affects: nothing until US-34 is picked up.

## 2026-11-19 · US-33 · Crash-safe saves
Decision: All app files are written through `store.writeFileAtomic`; each `data.json` save keeps a `data.json.bak`; a corrupt file offers a restore.
Why: All of a student's tasks and notes live in `data.json`.
Affects: `internal/store/atomic.go`, `recover.go`

## 2026-11-18 · US-05 (TODO: confirm) · Canvas sync keeps local work
Decision: Sync replaces Canvas assignments wholesale but keeps manual assignments (`Assignment.Manual`), the user's course fields (location, meetings, notes) and the live task and note lists. Canvas assignments are read-only in VIO.
Why: A sync must never lose something the user typed.
Affects: `internal/canvas`, `main.go`; law C2

## TODO · Canvas auth
Decision: The API token is stored AES-GCM encrypted in `config.json` with a key from username + hostname, so the config isn't portable between machines.
TODO: confirm whether that's the intended long-term design (vs. the OS keychain), and the auth method (personal access token only, or OAuth later).
