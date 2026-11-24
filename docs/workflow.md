# VIO-TUI workflow: Claude Chat plans and builds

## Roles

- **Claude Chat** does both the planning and the building: it plans features, keeps the VIO Feature Roadmap up to date, writes the Go code and tests in this repo, runs the checks, and keeps the study guides current. It never commits or pushes; it ends each build with git commands for me to run.
- **I** approve plans, run all git commands and own `CLAUDE.md`'s rules.
- Claude Code is no longer used (since 2026-11-24).

## The loop

1. **Plan in Chat** until the story is settled; Chat writes the plan and I approve it.
2. **Create the branch** (I run this): `git switch main && git pull && git switch -c us-xx-short-name`
3. **Chat builds it** in this folder: code, tests, `gofmt -l .`, `go vet ./...`, `go test ./...`, then a Session report and git commands.
4. **Review and run the git commands** myself: commit, push the branch, open a pull request into `main` on GitHub and merge it, then `git switch main && git pull` and `git branch -d us-xx-short-name`.
5. **Chat updates the roadmap, stories and study guide** for what shipped.
6. **New standing rules** go into `CLAUDE.md` (short) or `docs/decisions.md` (with the reason) and are committed with the story or in a docs PR.

## Plan template

```
# Plan: US-xx <title>
Branch: us-xx-short-name
Goal: <one sentence: what the user can do afterwards>

Rules:
- <decisions that constrain the build, e.g. "priority is optional; no priority sorts last">

Acceptance criteria:
- [ ] <observable behavior, testable>
- [ ] <...>

Laws / tests:
- <new or changed laws, with IDs if known>

Out of scope:
- <what not to build>

Files likely touched: <optional>
```

## What goes where

| Put it in | When |
|---|---|
| `CLAUDE.md` | Standing rules and current decisions Claude needs in almost every build session. One line each. Keep the file short (aim under ~150 lines). |
| `docs/decisions.md` | Every settled decision with date, story and reason, including reversed ones. |
| `docs/architecture.md` | How the code is organized: packages, conventions, data flow. |
| `docs/testing.md` | Testing rules and the list of laws. |
| `docs/workflow.md` | This file: the loop and the spec template. |
| `docs/*.rst` | Package reference docs. |
| `aidlc-docs/` (gitignored) | Local plans, research, story drafts, session state. Never the only home of a rule. |
| The story's plan | Anything that only matters for one feature. It doesn't go in `CLAUDE.md` unless it becomes a standing rule. |
