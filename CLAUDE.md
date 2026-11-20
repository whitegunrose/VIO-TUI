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
go test ./...         # tests: internal/model, internal/dates, internal/canvas, internal/store, internal/widgets
go test ./internal/model -run TestParseMeeting   # single test
```

## Architecture

- **`main.go`** is the composition root: loads data + Canvas config, builds the dashboard, wires navigation callbacks, and owns the Canvas polling goroutine. Sync runs off the UI thread and applies results via `app.QueueUpdateDraw`, then calls `store.SaveAppData` and `widgets.RefreshMainWidgets`.
- **`internal/model`** — `AppData` (Student, Courses, Assignments, Tasks, Notes) is the single in-memory source of truth, passed as `*model.AppData` to every view.
- **`internal/store`** — persistence under `os.UserConfigDir()/vio/`:
  - `data.json` (`store.go`): created with sample starter data on first run. Both load and save pass through `normalizeAppData`, which fills missing fields from defaults, dedupes IDs, drops invalid course refs, normalizes task status to `complete` / `overdue` / `in_progress`, and marks past-due tasks `overdue`. It also migrates legacy `Course.MeetingTimes` strings ("Mon 12:34 PM") into structured `Course.Meetings`. Changes to the model usually need matching changes here.
  - Saves are crash-safe (`atomic.go`): `writeFileAtomic` writes a hidden temp file in the same folder, fsyncs it, renames it over the target and fsyncs the folder. Each `data.json` save first copies the version it replaces to `data.json.bak` (only if that version is valid JSON and differs). If `data.json` can't be decoded, `LoadAppData` returns `*store.CorruptDataError`; `main.go`'s `loadOrRecover` (`recover.go`) shows a modal offering `store.RestoreBackup`, which moves the bad file aside as `data.json.corrupt-<time>` and restores the backup. Always write app files through `writeFileAtomic`, never `os.WriteFile`.
  - `config.json` (`config.go`): Canvas domain, poll minutes, and the API token encrypted with AES-GCM using a key derived from username + hostname (so the config isn't portable between machines).
- **`internal/canvas`** — Canvas REST client with `Link`-header pagination. `SyncCoursesAndAssignments` merges courses (preserving user-entered fields: location, meetings, notes; `main.go` reapplies them via `canvas.MergeCourses` for edits made mid-sync) but **replaces** the assignment list wholesale.
- **`internal/widgets`** — the dashboard and its routing:
  - `status.go` is the Canvas sync line in the dashboard header (`SyncStatus`: not set up / syncing / synced Xm ago / offline / failed). `main.go` updates it around each sync and refreshes it every 30s; call its methods on the UI goroutine.
  - `widgets.go` builds the 5 dashboard panels (Calendar, Courses, Tasks, Today, Notes) and their summary text.
  - `navigation.go` handles dashboard keys: `1`–`5` focus a panel, `Enter` opens it, `a` opens Assignments (it has no panel), `c` opens Canvas settings, `q` quits.
  - `screenrouter.go` maps panel index → page constructor (index 5, `AssignmentsScreen`, is the Assignments page). The panel order in `BuildMainWidgets` must stay in sync with the `ScreenRouter` switch and `RefreshMainWidgets`.
- **`internal/views/<feature>`** — full-screen pages. Convention: `XxxPage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive`. Pages swap themselves in with `app.SetRoot(...)` and call `returnTo` on Esc. Detail pages (`courses/course_page.go`, `assignments/assignment_page.go`, `tasks/task_page.go`) open with Enter and share the framed, scrollable layout in `views/detail`. The day page is `schedule.DayPage` (used by the dashboard's Today panel and Enter on a calendar day): an agenda from `AppData.Agenda` where `+`/`e`/`-`/`x` add, edit, delete and complete tasks and ←/→ change day. Most views only read `AppData`; the tasks page (`views/tasks`), notes page (`views/notes`), and courses page (`views/courses`, `m` edits a course's meeting times; `model.MeetingConflicts` flags overlapping classes on the course page and dashboard) edit `data.Tasks` / `data.Notes` / `Course.Meetings` and save with `store.SaveAppData` themselves. The calendar page defaults to a week view (`views/calendar/week.go`, a custom `tview.Box` that draws its own grid with no background fills so transparent terminals show through); `v` toggles the month grid. Canvas sync in `main.go` keeps the live task and note lists when applying results, and `ScreenRouter` refreshes dashboard summaries when a page returns.
- **`internal/model/agenda.go`** — `AppData.Agenda(day, includeDone)` merges a day's class meetings, assignments due and tasks due into time-ordered `AgendaItem`s; the Today panel and day page both use it. `models.go` gives `Assignment.State(now)` (missing / late / overdue / open / submitted), `Done()` and `NeedsAttention(now)` from Canvas's submission flags. `Assignment.Manual` marks assignments added in VIO: they're editable, and `canvas.KeepManual` keeps them through syncs (Canvas assignments stay read-only).
- **`internal/dates`** — `dates.Parse(text, now)` reads form dates: "fri 5pm", "tomorrow", "+3d", "next tue noon", "11/20", "2026-11-20 17:00" (a day alone means 11:59 PM; "next X" is X in next Monday-start week). `tasks.DuePreview` shows the "reads as …" line under due fields.
- **`internal/model/meetings.go`** — `Meeting` (days, 24-hour start/end, location) plus `ParseMeeting`/`ParseMeetings` for the one-per-line editor format ("Tue Thu 10:00 AM - 11:15 AM @ JD 1234", also "MW 2:30-3:45pm"), `FormatRange`, and `ShortCode` ("COMP 440-16707-FA2026" → "COMP 440").
- **`internal/theme`** — palette, rounded borders, and color tags. `theme.ApplyTheme()` runs first in `main()` because tview copies `tview.Styles` into each widget at construction. Dashboard panels use `theme.StylePane`, page panels `theme.FramePanel`, and `theme.CourseColor(i)` gives each course (by index in `data.Courses`) its color; prefer the `theme.Tag*` constants over hard-coded color tags.
- **`internal/asciiart`** — large ASCII headers for day and month names used by the calendar views.

Text styling uses tview color tags (`[white::b]...[::-]`, `[red]...[-]`), so TextViews need `SetDynamicColors(true)`.

## Docs

- `docs/*.rst` hold reference docs for each package (canvas, model, store, widgets).
- `.kiro/` and `aidlc-docs/` are gitignored local AI-workflow material, not project source.
