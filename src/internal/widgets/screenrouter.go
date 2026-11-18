package widgets

import (
	"VIO/internal/model"
	"VIO/internal/views/assignments"
	"VIO/internal/views/calendar"
	"VIO/internal/views/courses"
	"VIO/internal/views/notes"
	"VIO/internal/views/schedule"
	"VIO/internal/views/tasks"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// AssignmentsScreen is the router index for the Assignments page, opened
// with 'a' from the dashboard since panel 5 now shows notes.
const AssignmentsScreen = 5

// ScreenRouter returns a function that opens the correct screen based on index.
// Pass in the app, widgets array, and layout to allow switching back.
func ScreenRouter(app *tview.Application, widgets []tview.Primitive, layout *tview.Flex, data *model.AppData) func(index int) {
	return func(index int) {
		var screen tview.Primitive

		// Pages can edit data, so redraw the dashboard summaries on the way back
		back := func() {
			RefreshMainWidgets(widgets, data)
			focus := index
			if focus >= len(widgets) {
				focus = 0
			}
			app.SetRoot(layout, true).SetFocus(widgets[focus])
		}

		switch index {
		case 0:
			screen = calendar.CalendarPage(app, data, back)
		case 1:
			screen = courses.CoursesPage(app, data, back)
		case 2:
			screen = tasks.TasksPage(app, data, back)
		case 3:
			screen = schedule.SchedulePage(app, data, back)
		case 4:
			screen = notes.NotesPage(app, data, back)
		case AssignmentsScreen:
			screen = assignments.AssignmentsPage(app, data, back)
		default:
			tv := tview.NewTextView().
				SetTextAlign(tview.AlignCenter).
				SetDynamicColors(true).
				SetText(fmt.Sprintf("You selected [::b]Box %d[::-]!\n\n[gray]Press Esc to return", index+1))

			tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyEsc {
					back()
					return nil
				}
				return event
			})
			screen = tv
		}

		app.SetRoot(screen, true).SetFocus(screen)
	}
}
