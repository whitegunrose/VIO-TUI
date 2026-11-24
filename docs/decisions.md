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

## 2026-11-24 · (no story yet) · Canvas auth: current design and direction
Decision: Today the personal access token is stored AES-GCM encrypted in `config.json` (file mode 0600) with a key derived from username + hostname, so the config isn't portable between machines. OAuth sign-in is to be explored later. Moving the token to the OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service), with the encrypted file kept as a fallback, is recommended but not yet approved.
Why: The file key is built from values that aren't secret, so it mainly guards against the file being copied elsewhere; the keychain encrypts the token with the user's login and, on macOS, limits which apps can read it. OAuth needs a developer key from each school's Canvas admin (or a global key from Instructure) and short-lived tokens with refresh.
Affects: `internal/store/config.go`, the Canvas settings page. TODO: assign stories in Chat once approved; a keychain library is a new `go.mod` dependency and needs approval.

## 2026-11-24 · (workflow) · Branches and pull requests
Decision: One branch per story, named `us-xx-short-name`, from an up-to-date `main`. Work merges into `main` through a pull request; I run every git step, and Claude Code's Git commands block includes the push, PR and post-merge cleanup commands.
Why: Keeps each story reviewable on its own, and I stay the only author on the repo.
Affects: `CLAUDE.md` (Branches and merging), `docs/workflow.md`

## 2026-11-24 · US-32 · Go's built-in fuzzer only
Decision: Property-based and fuzz tests use Go's `testing.F`. No third-party property-testing library (e.g. `pgregory.net/rapid`).
Why: The built-in fuzzer already covers every law and needs no new dependency.
Affects: `docs/testing.md`, every `fuzz_test.go`

## 2026-11-24 · (workflow) · Chat decides, Claude Code builds
Decision: Planning, the VIO Feature Roadmap and study guides live in Claude Chat. Claude Code builds from a handoff spec and never commits or pushes; I run git myself. No Claude attribution in any commit, PR or file. Rules Claude Code must follow live in `CLAUDE.md` or committed `docs/`, never only in `aidlc-docs/`.
Why: Claude Code can't see Chat history; `aidlc-docs/` is gitignored.
Affects: `CLAUDE.md`, `docs/workflow.md`

## 2026-11-20 · US-24 · Fuzzy matcher written in-house
Decision: Search (`/`) uses our own fuzzy matcher, `internal/sift`. No new dependency.
Why: Keeps `go.mod` small; a prototype scored well and ran ~5 µs per item.
Affects: new package `internal/sift` (when US-24 is built)

## 2026-11-20 · US-22 · Assignment overlay details
Decision:
- Overlay entries whose assignment leaves Canvas are kept 30 days (`orphaned_at` stamp, cleared if the assignment comes back), stamped only after a fully successful sync.
- Estimate is in hours, typed like `+2h`, `2h` or `in 2 hours`; stored as minutes.
- "Started" is a badge only; it doesn't change an assignment's state or group. Its look is provisional.
- Default order inside each Assignments group and Tasks column: priority high → medium → low → none, then due date.
Why: Settled while planning Phase 3.
Affects: `internal/model`, `internal/store` (normalize), `internal/canvas` (sync), assignments and tasks views

## 2026-11-20 · US-18 · Undated Canvas assignments stay undated
Decision: An assignment with no due date in Canvas keeps no due date until the professor sets one; the next sync picks it up. The Assignments page lists them last under Upcoming as "no due date yet".
Why: `normalizeAppData` used to give them a sample date, which showed fake deadlines.
Affects: `internal/store` (normalize), Assignments page; law S4, `TestNormalizeKeepsUndatedAssignment`

## 2026-11-20 · US-32 · Fuzz/property-based testing with Go's built-in fuzzer
Decision: Core logic is checked by laws in `fuzz_test.go` per package, using `testing.F`; seeds run in CI on every push, full fuzzing weekly in `.github/workflows/fuzz.yml`.
Why: Parsers and normalizers take arbitrary user and Canvas input; fuzzing found eight real bugs on day one.
Affects: `docs/testing.md`

## 2026-11-19 · US-22 · Task priority is optional per task
Decision: Priority applies to personal tasks as well as Canvas assignments, so both lists sort the same way, but it is optional on each task. Canvas has no priority field, so it's a VIO-only field: assignments get it through the US-22 overlay, tasks through an optional `Priority` field. Levels: none / low / medium / high. Unset is the default and sorts last (see the 2026-11-20 US-22 entry).
Why: Open question 1 in the roadmap plan.
Affects: `internal/model`, `internal/store` (normalize), tasks views.

## 2026-11-19 · US-34 · Homebrew release deferred
Decision: GoReleaser + Homebrew tap is deferred until VIO has a tagged release.
Why: Homebrew needs a stable tagged release (VIO has none yet) and a `vio --version` for the formula's test; the official `homebrew/core` also needs about 225 stars. The plan when US-34 starts: add `vio --version`, tag `v0.1.0`, publish a from-source formula in a `whitegunrose/homebrew-tap` repo, then automate with GoReleaser.
Affects: nothing until US-34 is picked up.

## 2026-11-19 · US-33 · Crash-safe saves
Decision: All app files are written through `store.writeFileAtomic`; each `data.json` save keeps a `data.json.bak`; a corrupt file offers a restore.
Why: All of a student's tasks and notes live in `data.json`.
Affects: `internal/store/atomic.go`, `recover.go`

## 2026-11-18 · US-05 · Canvas sync keeps local work
Decision: Sync replaces Canvas assignments wholesale but keeps manual assignments (`Assignment.Manual`), the user's course fields (location, meetings, notes) and the live task and note lists. Canvas assignments are read-only in VIO.
Why: A sync must never lose something the user typed.
Affects: `internal/canvas`, `main.go`; law C2
