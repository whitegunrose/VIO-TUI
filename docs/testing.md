# VIO-TUI testing

The committed testing guide. It carries over the parts of the local testing strategy (`aidlc-docs/design-artifacts/testing_strategy.md`, gitignored) that anyone working on the repo must follow. When a law is added, changed or found broken, update the table here in the same change.

## What we protect first

| Risk | Where |
|---|---|
| Losing or corrupting `data.json` | `internal/store` |
| Wrong due dates | `internal/dates`, the forms |
| Wrong class times or missed overlaps | `internal/model` (meetings, agenda) |
| Sync losing local work | `internal/canvas`, `main.go` |
| Crashing on bad input (hand-edited files, odd Canvas responses, typos) | every parser |
| Leaking the Canvas token | `internal/store/config.go` |

## Layers

| Layer | Run with |
|---|---|
| Unit and table-driven tests | `go test ./...` |
| Fuzz / property-based tests: one `fuzz_test.go` per package; seeds run with every `go test` | `go test -run '^$' -fuzz '^FuzzName$' -fuzztime 60s ./internal/<pkg>` (one target at a time) |
| Integration with fakes: Canvas against `httptest`, store in `t.TempDir()` | `go test ./internal/canvas ./internal/store` |
| Race checker | `go test -race ./...` |
| Smoke test, by hand before a release: open VIO, add and complete a task, sync Canvas, quit, reopen | manual |

Not unit-tested on purpose: `internal/views` and `main.go` wiring. Keep logic out of them.

## Laws

A law is a rule that must hold for every input, checked by a fuzz target. Give each new law the next ID for its package (D = dates, M = model, S = store, C = canvas) and write the ID in a comment in the test.

| # | Law | Test |
|---|---|---|
| D1 | `dates.Parse` never panics | `FuzzParse` |
| D2 | A parsed date is in the local time zone, on a whole minute | `FuzzParse` |
| D3 | What a form displays (`2026-11-20 17:00`) reads back unchanged | `FuzzLayoutRoundTrip` |
| D4 | Any month/day that exists can be typed without a year and gives the next one | `FuzzMonthDay` |
| D5 | A date with a time lands on that day at that clock time, including DST change days | `FuzzMonthDay`, `TestParseOnDSTChangeDays` |
| M1 | `ParseMeeting` never panics | `FuzzParseMeeting` |
| M2 | A parsed meeting has days in week order, no repeats, and ends after it starts | `FuzzParseMeeting` |
| M3 | A meeting round-trips through the editor text | `FuzzParseMeeting` |
| M4 | Overlap detection is symmetric and ignores back-to-back classes | `FuzzMeetingConflicts` |
| M5 | A day's agenda is in time order and only holds that day's items | `FuzzAgendaOrder` |
| S1 | Normalizing is idempotent and reports no change on clean data | `FuzzNormalize` |
| S2 | After normalizing, IDs are unique and every course reference is real | `FuzzNormalize` |
| S3 | Loading any bytes never panics; an unreadable file is reported and left untouched; save then load gives the same data | `FuzzLoad` |
| S4 | An assignment with no due date stays undated | `FuzzNormalize`, `TestNormalizeKeepsUndatedAssignment` |
| C1 | Reading a `Link` header never panics, and the next URL comes from inside it | `FuzzNextLink` |
| C2 | Merging keeps Canvas's course list and order, and each course keeps the user's room, meetings and notes | `FuzzMergeCourses` |

Laws still without a fuzz test:

- Sync is all-or-nothing: a failed sync leaves `AppData` unchanged (example test only: `TestSyncErrorLeavesDataUnchanged`).
- A failed save leaves the previous `data.json` intact (example test only).
- `KeepManual` keeps every manual assignment and no stale Canvas one, for any mix.

## Rules

- New parsing, normalizing, merging or ordering code comes with at least one law and a fuzz target, plus example tests for the cases the spec names.
- When the fuzzer finds a failing input, Go saves it under `internal/<pkg>/testdata/fuzz/<FuzzName>/`. Copy it into the target's `f.Add` seeds with a comment saying what it found, so it runs on every `go test`.
- Every bug fix gets a regression test that fails on the old code.
- Never delete, skip or weaken a test or a law to get green. If a bug can't be fixed right away, ask first; an approved skip names this file and the fix removes it.
- Daylight-saving bugs show up only in a zone with DST: fuzz with `TZ=America/Los_Angeles`.
- A new fuzz target is also added to the matrix in `.github/workflows/fuzz.yml`.

## CI

- `.github/workflows/ci.yml` (every push and PR): `gofmt -l`, `go vet`, `go build`, `go test` on Linux, macOS and Windows; `go test -race` on Linux.
- `.github/workflows/fuzz.yml` (Mondays 09:17 UTC, and on demand): each fuzz target for 60s in `America/Los_Angeles`; a failing input is uploaded as an artifact.
