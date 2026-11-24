# VIO-TUI workflow: Chat decides, Claude Code builds

## Roles

- **Claude Chat** is for deciding: planning features, keeping the VIO Feature Roadmap up to date, design discussions, study guides.
- **Claude Code** is for building: it works in this repo, writes the Go code and tests, and debugs with real output. It never commits or pushes; it ends each session with git commands for me to run.
- **I** run all git commands and own `CLAUDE.md`'s rules.

## The loop

1. **Plan in Chat** until the feature is settled, then ask for a handoff spec in the template below.
2. **Create the branch** (I run this): `git switch main && git pull && git switch -c us-xx-short-name`
3. **Paste the spec into Claude Code.** It builds, tests, and ends with a Session report and git commands.
4. **Review and run the git commands** myself.
5. **Bring the Session report back to Chat** to update the roadmap and study guide.
6. **End of planning session:** ask Chat "Give me the CLAUDE.md additions from today." Paste short rules into `CLAUDE.md`, longer material into `docs/decisions.md` (or another `docs/` file), and commit them.

## Handoff spec template

```
# Handoff: US-xx <title>
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
| `CLAUDE.md` | Standing rules and current decisions Claude Code needs in almost every session. One line each. Keep the file short (aim under ~150 lines). |
| `docs/decisions.md` | Every settled decision with date, story and reason, including reversed ones. |
| `docs/architecture.md` | How the code is organized: packages, conventions, data flow. |
| `docs/testing.md` | Testing rules and the list of laws. |
| `docs/workflow.md` | This file: the loop and the spec template. |
| `docs/*.rst` | Package reference docs. |
| `aidlc-docs/` (gitignored) | Local plans, research, story drafts, session state. Never the only home of a rule. |
| The handoff spec | Anything that only matters for one feature. It doesn't go in `CLAUDE.md` unless it becomes a standing rule. |
