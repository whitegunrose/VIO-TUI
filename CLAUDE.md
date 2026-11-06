# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

VIO-TUI is a keyboard-driven terminal app (Go + [tview](https://github.com/rivo/tview)/tcell) for students to track courses, assignments, tasks, and schedule, with optional Canvas LMS sync.

## Commands

The Go module (`module VIO`, Go 1.24) lives in `src/`, not the repo root — run all Go commands from there.

```sh
cd src
go run .              # run the TUI
go build -o ../vio .  # build binary (/vio is gitignored)
go vet ./...
go test ./...         # no tests exist yet
go test ./internal/store -run TestName   # single test, once tests exist
```

## Architecture

- **`main.go`** is the composition root: loads data + Canvas config, builds the dashboard, wires navigation callbacks, and owns the Canvas polling goroutine. Sync runs off the UI thread and applies results via `app.QueueUpdateDraw`, then calls `store.SaveAppData` and `widgets.RefreshMainWidgets`.
- **`internal/model`** — `AppData` (Student, Courses, Assignments, Tasks) is the single in-memory source of truth, passed as `*model.AppData` to every view.
- **`internal/store`** — persistence under `os.UserConfigDir()/vio/`:
  - `data.json` (`store.go`): created with sample starter data on first run. Both load and save pass through `normalizeAppData`, which fills missing fields from defaults, dedupes IDs, drops invalid course refs, normalizes task status to `complete` / `overdue` / `in_progress`, and marks past-due tasks `overdue`. Changes to the model usually need matching changes here.
  - `config.json` (`config.go`): Canvas domain, poll minutes, and the API token encrypted with AES-GCM using a key derived from username + hostname (so the config isn't portable between machines).
- **`internal/canvas`** — Canvas REST client with `Link`-header pagination. `SyncCoursesAndAssignments` merges courses (preserving user-entered fields like location/meeting times) but **replaces** the assignment list wholesale.
- **`internal/widgets`** — the dashboard and its routing:
  - `widgets.go` builds the 5 dashboard panels (Calendar, Courses, Tasks, Schedule, Assignments) and their summary text.
  - `navigation.go` handles dashboard keys: `1`–`5` focus a panel, `Enter` opens it, `c` opens Canvas settings, `q` quits.
  - `screenrouter.go` maps panel index → page constructor. The panel order in `BuildMainWidgets` must stay in sync with the `ScreenRouter` switch and `RefreshMainWidgets`.
- **`internal/views/<feature>`** — full-screen pages. Convention: `XxxPage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive`. Pages swap themselves in with `app.SetRoot(...)` and call `returnTo` on Esc. The `*_page.go` files are empty placeholders for future detail pages. Views currently only read `AppData`; persistence happens in `main.go` after a sync.
- **`internal/asciiart`** — large ASCII headers for day and month names used by the calendar views.

Text styling uses tview color tags (`[white::b]...[::-]`, `[red]...[-]`), so TextViews need `SetDynamicColors(true)`.

## Docs

- `docs/*.rst` hold reference docs for each package (canvas, model, store, widgets).
- `.kiro/` and `aidlc-docs/` are gitignored local AI-workflow material, not project source.
