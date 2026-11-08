<a id="readme-top"></a>

<!-- PROJECT SHIELDS -->
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![License][license-shield]][license-url]

<br />
<div align="center">

  <div align="center">
    <img src="docs/assets/images/vio_github_logo.webp" alt="VIO-TUI Logo" width="400">
  </div>

  <p align="center">
    Terminal User Interface for daily time-management tracking — built by a student software engineer, for student software engineers.
    <br />
    <a href="https://github.com/whitegunrose/VIO-TUI/wiki/Documentation"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="https://github.com/whitegunrose/VIO-TUI/issues/new?labels=bug&template=bug-report.md">Report Bug</a>
    ·
    <a href="https://github.com/whitegunrose/VIO-TUI/issues/new?labels=enhancement&template=feature-request.md">Request Feature</a>
  </p>
</div>

---

## Table of Contents

1. [About The Project](#about-the-project)
   - [Built With](#built-with)
2. [Getting Started](#getting-started)
   - [Prerequisites](#prerequisites)
   - [Installation](#installation)
3. [Usage](#usage)
4. [Roadmap](#roadmap)
5. [Contributing](#contributing)
6. [License](#license)
7. [Contact](#contact)
8. [Acknowledgments](#acknowledgments)

---
![](https://github.com/whitegunrose/VIO-TUI/blob/main/docs/assets/demos/demo.gif)
---

## About The Project

<!-- Replace the line below with an actual screenshot once available -->
<!-- ![VIO-TUI Screenshot](docs/screenshot.png) -->

VIO-TUI is a keyboard-driven Terminal User Interface application written in Go, designed to help student software engineers take control of their daily schedules. Managing lectures, coding sessions, deadlines, and personal tasks across multiple apps is overwhelming — VIO solves that by keeping everything in one distraction-free terminal window.

**Why VIO?**

- 🖥️ Stays out of your way — no GUI overhead, no mouse required
- ⚡ Instant startup and near-zero resource usage, so it never slows down your workflow
- 📋 Tracks your time blocks, tasks, and sessions in one unified view
- 🎓 Tailored to the rhythms of a student engineering life — classes, side projects, and deadlines all in one place

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Built With

[![Go][Go-badge]][Go-url]
[![tview][tview-badge]][tview-url]

> VIO-TUI is 100% Go. The TUI layer is powered by [tview](https://github.com/rivo/tview) — a rich interactive widget library built on top of [tcell](https://github.com/gdamore/tcell).

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Getting Started

Follow the steps below to get VIO-TUI running on your local machine.

### Prerequisites

- **Go 1.24+** — [Install Go](https://go.dev/dl/)

  ```sh
  go version   # verify installation
  ```

- A terminal with color support (any modern terminal on macOS, Linux, or Windows works)
- *(Optional)* A **Canvas LMS** account, if you want your courses and assignments synced automatically

### Installation

1. Clone the repository

   ```sh
   git clone https://github.com/whitegunrose/VIO-TUI.git
   cd VIO-TUI
   ```

2. Move into the Go module (it lives in `src/`, not the repo root)

   ```sh
   cd src
   ```

3. Run VIO directly

   ```sh
   go run .
   ```

   Or build a binary into the repo root and run that:

   ```sh
   go build -o ../vio .
   ../vio
   ```

4. *(Optional)* Move the binary onto your `PATH` for system-wide access

   ```sh
   mv ../vio /usr/local/bin/vio
   ```

### Where your data lives

On first launch VIO creates a `vio/` folder in your OS config directory:

| OS      | Location                               |
|---------|----------------------------------------|
| macOS   | `~/Library/Application Support/vio/`   |
| Linux   | `~/.config/vio/` (or `$XDG_CONFIG_HOME/vio/`) |
| Windows | `%AppData%\vio\`                       |

- **`data.json`** — your student info, courses, assignments, and tasks. It's seeded with sample data on first run so the dashboard isn't empty; edit or replace it with your own.
- **`config.json`** — Canvas settings. Your API token is encrypted with a key tied to your username and hostname, so this file won't work if copied to another machine — just re-enter your token there.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Usage

Launch VIO from any terminal:

```sh
vio        # or `go run .` from src/
```

On the very first launch you'll be asked whether to **configure Canvas** now. It's optional — choose *Skip For Now* to use local data only. You can set it up later from the dashboard.

### Dashboard

The dashboard shows five panels, each with a quick summary:

| # | Panel       | What it shows                                  |
|---|-------------|------------------------------------------------|
| 1 | Calendar    | The current month and what's due               |
| 2 | Courses     | Your enrolled courses                          |
| 3 | Tasks       | Personal to-dos and their status               |
| 4 | Schedule    | Class meeting times and locations              |
| 5 | Assignments | Upcoming coursework and due dates              |

| Key           | Action                                 |
|---------------|----------------------------------------|
| `1` – `5`     | Focus a panel                          |
| `Enter`       | Open the focused panel full-screen     |
| `c`           | Open Canvas settings                   |
| `q`           | Quit VIO                               |

### Inside a panel

| Key             | Action                                         |
|-----------------|------------------------------------------------|
| `Esc`           | Return to the dashboard                        |
| `←` / `→`       | *(Calendar)* Previous / next day               |
| `↑` / `↓`       | *(Calendar)* Previous / next week              |
| `p` / `n`       | *(Calendar)* Previous / next month             |

### Canvas sync

Press `c` on the dashboard to open Canvas settings, then fill in:

- **Canvas Domain** — your school's Canvas host, e.g. `myschool.instructure.com` (`https://` is added automatically)
- **Canvas API Key** — generate one in Canvas under **Account → Settings → Approved Integrations → + New Access Token**
- **Poll Minutes** — how often to re-sync (defaults to `5`)

Use `Tab` / `Shift+Tab` to move between fields, then select **Save**. VIO syncs immediately and then keeps polling in the background; the result of the last sync is shown in the settings page's *Status* box.

Sync pulls your **active** courses and their assignments:

- Courses are merged, so anything you've added yourself (location, meeting times, notes) is kept.
- Assignments are **replaced** with Canvas's list on every sync, so local edits to assignments will be overwritten.

### Editing your data

There's no in-app editor yet. To add tasks, set your name and school, or fill in class locations and meeting times, edit `data.json` (see [Where your data lives](#where-your-data-lives)) while VIO is closed. VIO cleans the file up on load — filling in missing fields, removing duplicate IDs, and marking past-due tasks as `overdue`. Valid task statuses are `in_progress`, `complete`, and `overdue`.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Roadmap

- [x] Core TUI shell and navigation
- [x] Daily task/time-block tracking
- [x] Canvas integration to import classes and assignments
- [ ] Persistent storage (save sessions across restarts)
- [ ] Weekly and monthly summary views
- [ ] Cross-platform binary releases (Linux, macOS, Windows)

<!-- See the [open issues](https://github.com/whitegunrose/VIO-TUI/issues) for the full list of proposed features and known bugs. -->

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Contributing

Contributions are welcome! If you have an idea that would make VIO better, open an issue tagged `enhancement` or send a pull request. Bug reports are just as valuable — use the [bug report template](https://github.com/whitegunrose/VIO-TUI/issues/new?labels=bug&template=bug-report.md).

### Workflow

1. Fork the project and clone your fork
2. Create a feature branch

   ```sh
   git checkout -b feature/AmazingFeature
   ```

3. Make your changes, then check them from `src/`

   ```sh
   cd src
   go vet ./...
   go build ./...
   go run .       # try it out in your terminal
   ```

4. Commit and push

   ```sh
   git commit -m "Add AmazingFeature"
   git push origin feature/AmazingFeature
   ```

5. Open a Pull Request against `main` describing what changed and why

### Project layout

```
src/
├── main.go                 # entry point: loads data, builds dashboard, runs Canvas polling
└── internal/
    ├── model/              # AppData, Course, Assignment, Task structs
    ├── store/              # data.json + encrypted config.json persistence
    ├── canvas/             # Canvas REST client and sync/merge logic
    ├── widgets/            # dashboard panels, key handling, panel → page routing
    ├── views/<feature>/    # full-screen pages (calendar, courses, tasks, schedule, assignments, settings)
    └── asciiart/           # large ASCII day/month headers for the calendar
```

Package reference docs live in [`docs/`](docs/).

### Things to keep in mind

- **Adding a model field?** Update the defaults and normalization in `internal/store/store.go` too — every load and save runs through `normalizeAppData`.
- **Adding or reordering a dashboard panel?** Keep `BuildMainWidgets`, `RefreshMainWidgets`, and the `ScreenRouter` switch in the same order.
- **New pages** follow the pattern `XxxPage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive` and call `returnTo` on `Esc`.
- **UI updates from goroutines** (e.g. network calls) must go through `app.QueueUpdateDraw`.
- **Colored text** uses tview tags like `[green]...[-]`, so `TextView`s need `SetDynamicColors(true)`.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## License

Distributed under the MIT License. See `LICENSE` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Contact

**whitegunrose** — [@whitegunrose](https://github.com/whitegunrose)

<!-- Project Link: [https://github.com/whitegunrose/VIO-TUI](https://github.com/whitegunrose/VIO-TUI) -->

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Acknowledgments

Resources and tools that made VIO-TUI possible:

- [tview](https://github.com/rivo/tview) — Rich interactive TUI widget library for Go
- [tcell](https://github.com/gdamore/tcell) — Low-level terminal handling that powers tview
- [Go Documentation](https://go.dev/doc/)
- [Best-README-Template](https://github.com/othneildrew/Best-README-Template) — README structure inspiration
- [Shields.io](https://shields.io) — Badge generation
- [Choose an Open Source License](https://choosealicense.com)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

<!-- MARKDOWN LINKS & BADGES -->
[contributors-shield]: https://img.shields.io/github/contributors/whitegunrose/VIO-TUI.svg?style=for-the-badge
[contributors-url]: https://github.com/whitegunrose/VIO-TUI/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/whitegunrose/VIO-TUI.svg?style=for-the-badge
[forks-url]: https://github.com/whitegunrose/VIO-TUI/network/members
[stars-shield]: https://img.shields.io/github/stars/whitegunrose/VIO-TUI.svg?style=for-the-badge
[stars-url]: https://github.com/whitegunrose/VIO-TUI/stargazers
[issues-shield]: https://img.shields.io/github/issues/whitegunrose/VIO-TUI.svg?style=for-the-badge
[issues-url]: https://github.com/whitegunrose/VIO-TUI/issues
[license-shield]: https://img.shields.io/github/license/whitegunrose/VIO-TUI.svg?style=for-the-badge
[license-url]: https://github.com/whitegunrose/VIO-TUI/blob/main/LICENSE
[Go-badge]: https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white
[Go-url]: https://go.dev/
[tview-badge]: https://img.shields.io/badge/tview-00ADD8?style=for-the-badge&logo=go&logoColor=white
[tview-url]: https://github.com/rivo/tview
