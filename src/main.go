package main

import (
	"VIO/internal/canvas"
	"VIO/internal/store"
	"VIO/internal/theme"
	"VIO/internal/views/settings"
	"VIO/internal/widgets"
	"fmt"
	"sync"
	"time"

	"github.com/rivo/tview"
)

func main() {
	// Must run before any widget is built
	theme.ApplyTheme()

	// Load local JSON data
	data, err := store.LoadAppData()
	if err != nil {
		panic(err)
	}

	canvasCfg, cfgExists, err := store.LoadCanvasConfig()
	if err != nil {
		panic(err)
	}
	if canvasCfg.PollMinutes <= 0 {
		canvasCfg.PollMinutes = 5
	}

	// Create new TUI Application
	app := tview.NewApplication()

	// Grab dashboard widgets
	mainWidgets, flex, syncStatus := widgets.BuildMainWidgets(data)
	syncStatus.SetConfigured(canvasCfg.HasCredentials())

	// Keep "synced 3m ago" and the Today panel current while VIO stays open
	go func() {
		for range time.Tick(30 * time.Second) {
			app.QueueUpdateDraw(func() {
				syncStatus.Refresh()
				widgets.RefreshMainWidgets(mainWidgets, data)
			})
		}
	}()

	var mu sync.Mutex
	lastSyncStatus := "Canvas sync is optional. Press [white::b]c[::-] from the dashboard to configure it."
	var stopPolling chan struct{}

	refreshDashboard := func() {
		widgets.RefreshMainWidgets(mainWidgets, data)
	}

	syncNow := func() {
		cfg := canvasCfg
		if !cfg.HasCredentials() {
			return
		}

		// QueueUpdateDraw blocks until the event loop runs the update, so it
		// must only be called from this goroutine: syncNow itself runs before
		// app.Run() at startup and on the UI thread when settings are saved.
		go func() {
			app.QueueUpdateDraw(syncStatus.Started)

			mu.Lock()
			snapshot := *data
			mu.Unlock()
			updated, err := canvas.SyncCoursesAndAssignments(cfg, snapshot)
			app.QueueUpdateDraw(func() {
				mu.Lock()
				defer mu.Unlock()

				syncStatus.Finished(err, time.Now())
				if err != nil {
					if widgets.IsOffline(err) {
						lastSyncStatus = fmt.Sprintf("[yellow]Canvas is unreachable (offline?):[-] %v", err)
					} else {
						lastSyncStatus = fmt.Sprintf("[red]Canvas sync failed:[-] %v", err)
					}
					return
				}

				// Sync only touches courses and assignments; keep tasks, notes and
				// manual assignments edited since it started
				updated.Tasks = data.Tasks
				updated.Notes = data.Notes
				updated.Courses = canvas.MergeCourses(data.Courses, updated.Courses)
				updated.Assignments = canvas.KeepManual(updated.Assignments, data.Assignments)
				*data = updated

				if err := store.SaveAppData(*data); err != nil {
					lastSyncStatus = fmt.Sprintf("[yellow]Canvas synced, but saving cache failed:[-] %v", err)
					refreshDashboard()
					return
				}

				refreshDashboard()
				lastSyncStatus = fmt.Sprintf("[green]Canvas synced[-] %s", time.Now().Format("3:04 PM"))
			})
		}()
	}

	startPolling := func() {
		if stopPolling != nil {
			close(stopPolling)
		}

		syncStatus.SetConfigured(canvasCfg.HasCredentials())
		if !canvasCfg.HasCredentials() {
			stopPolling = nil
			return
		}

		stopPolling = make(chan struct{})
		syncNow()

		ticker := time.NewTicker(canvasCfg.PollInterval())
		go func(stop <-chan struct{}) {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					syncNow()
				case <-stop:
					return
				}
			}
		}(stopPolling)
	}

	showDashboard := func() {
		app.SetRoot(flex, true).SetFocus(flex)
	}

	// Choose screen to display (subject to change with widget-click-selection
	openScreen := widgets.ScreenRouter(app, mainWidgets, flex, data)

	var openCanvasSettings func()
	openCanvasSettings = func() {
		page := settings.CanvasSettingsPage(
			app,
			canvasCfg,
			lastSyncStatus,
			func(newCfg store.CanvasConfig) {
				if err := store.SaveCanvasConfig(newCfg); err != nil {
					lastSyncStatus = fmt.Sprintf("[red]Failed to save Canvas config:[-] %v", err)
					openCanvasSettings()
					return
				}

				mu.Lock()
				canvasCfg = newCfg
				if canvasCfg.PollMinutes <= 0 {
					canvasCfg.PollMinutes = 5
				}
				lastSyncStatus = "[green]Canvas settings saved.[-]"
				mu.Unlock()

				startPolling()
				showDashboard()
			},
			func() {
				showDashboard()
			},
		)

		app.SetRoot(page, true).SetFocus(page)
	}

	// Handle navigation around widgets on main page
	widgets.HandleNavigation(flex, mainWidgets, openScreen, openCanvasSettings, app.Stop)

	if !cfgExists {
		modal := tview.NewModal().
			SetText("Configure Canvas now?\n\nThis is optional. You can skip it and use local JSON only.").
			AddButtons([]string{"Configure Canvas", "Skip For Now"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if buttonLabel == "Configure Canvas" {
					openCanvasSettings()
					return
				}
				showDashboard()
			})

		app.SetRoot(modal, true)
	} else {
		showDashboard()
		startPolling()
	}

	// Start program
	if err := app.SetRoot(flex, true).SetFocus(flex).Run(); err != nil {
		panic(err)
	}
}
