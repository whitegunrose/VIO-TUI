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
	mainWidgets, flex := widgets.BuildMainWidgets(data)

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

		go func() {
			updated, err := canvas.SyncCoursesAndAssignments(cfg, *data)
			app.QueueUpdateDraw(func() {
				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					lastSyncStatus = fmt.Sprintf("[red]Canvas sync failed:[-] %v", err)
					return
				}

				// Sync only touches courses and assignments; keep tasks and notes edited since it started
				updated.Tasks = data.Tasks
				updated.Notes = data.Notes
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
