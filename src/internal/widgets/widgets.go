package widgets

import (
	"VIO/internal/model"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rivo/tview"
)

// BuildMainWidgets returns the list of box widgets and the main layout.
func BuildMainWidgets(data *model.AppData) ([]tview.Primitive, *tview.Flex) {

	// Widgets

	// Dashboard panels
	wCalendar := tview.NewTextView().SetDynamicColors(true)
	wCalendar.SetBorder(true)
	wCalendar.SetTitle("[ 1 ]")

	wCourses := tview.NewTextView().SetDynamicColors(true)
	wCourses.SetBorder(true)
	wCourses.SetTitle("[ 2 ]")

	wTodo := tview.NewTextView().SetDynamicColors(true)
	wTodo.SetBorder(true)
	wTodo.SetTitle("[ 3 ]")

	wSchedule := tview.NewTextView().SetDynamicColors(true)
	wSchedule.SetBorder(true)
	wSchedule.SetTitle("[ 4 ]")

	wAssignments := tview.NewTextView().SetDynamicColors(true)
	wAssignments.SetBorder(true)
	wAssignments.SetTitle("[ 5 ]")

	wCalendar.SetText(buildCalendarSummary(data))
	wCourses.SetText(buildCoursesSummary(data))
	wTodo.SetText(buildTasksSummary(data))
	wSchedule.SetText(buildScheduleSummary(data))
	wAssignments.SetText(buildAssignmentsSummary(data))

	mainBody := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 1, 3, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(wCalendar, 0, 2, false).
				AddItem(wCourses, 0, 1, false).
				AddItem(wTodo, 0, 1, false),
			0, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(wSchedule, 0, 1, false).
				AddItem(wAssignments, 0, 1, false),
			0, 1, false)

	quitPadding := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(`


		`)

	quitText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[white::b][ q ] To QUIT    [ c ] Canvas Settings")

	quit := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(quitPadding, 2, 1, false).
		AddItem(quitText, 0, 2, false)

	title := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetText(`
__     __   _     ____    
\ \   / /  | |   / __ \ 
 \ \_/ /   | |  | |__| |
  \___/    |_|   \____/ 
`)

	header := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(quit, 0, 4, false). // narrow weight
		AddItem(title, 0, 6, false) // wider weight

	// Final layout with header and body
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 5, 0, false).
		AddItem(mainBody, 0, 1, false)

	widgets := []tview.Primitive{wCalendar, wCourses, wTodo, wSchedule, wAssignments}
	return widgets, flex
}

func buildCalendarSummary(data *model.AppData) string {
	now := time.Now()
	var dueLines []string

	for _, assignment := range data.Assignments {
		if assignment.DueAt == nil {
			continue
		}
		if assignment.DueAt.Month() != now.Month() || assignment.DueAt.Year() != now.Year() {
			continue
		}
		dueLines = append(dueLines, fmt.Sprintf("%02d  %s", assignment.DueAt.Day(), assignment.Name))
	}

	sort.Strings(dueLines)

	if len(dueLines) == 0 {
		return fmt.Sprintf("%s\n\nNo assignment due dates this month yet.", now.Month())
	}
	if len(dueLines) > 8 {
		dueLines = dueLines[:8]
	}

	return fmt.Sprintf("%s\n\n%s", now.Month(), strings.Join(dueLines, "\n"))
}

func buildCoursesSummary(data *model.AppData) string {
	if len(data.Courses) == 0 {
		return "Courses\n\nNo courses loaded."
	}

	var lines []string
	for _, course := range data.Courses {
		lines = append(lines, fmt.Sprintf("%s\n%s", course.Code, course.Name))
	}

	return strings.Join(lines, "\n\n")
}

func buildTasksSummary(data *model.AppData) string {
	if len(data.Tasks) == 0 {
		return "Tasks\n\nNo personal tasks yet."
	}

	counts := map[string]int{
		"in_progress": 0,
		"overdue":     0,
		"complete":    0,
	}

	for _, task := range data.Tasks {
		counts[dashboardTaskStatus(task)]++
	}

	return fmt.Sprintf(
		"IN PROGRESS: %d\nOVERDUE: %d\nCOMPLETE: %d",
		counts["in_progress"],
		counts["overdue"],
		counts["complete"],
	)
}

func dashboardTaskStatus(task model.Task) string {
	status := strings.ToLower(strings.TrimSpace(task.Status))
	if status == "complete" {
		return "complete"
	}
	if task.DueAt != nil && task.DueAt.Before(time.Now()) {
		return "overdue"
	}
	return "in_progress"
}

func buildScheduleSummary(data *model.AppData) string {
	today := time.Now()
	var lines []string

	for _, assignment := range data.Assignments {
		if assignment.DueAt == nil {
			continue
		}

		due := assignment.DueAt.In(time.Local)
		if sameLocalDay(due, today) {
			lines = append(lines, fmt.Sprintf(
				"%s\n%s",
				assignment.Name,
				due.Format("3:04 PM"),
			))
		}
	}

	if len(lines) == 0 {
		return "Today\n\nNothing due today."
	}

	return strings.Join(lines, "\n\n")
}

func buildAssignmentsSummary(data *model.AppData) string {
	now := time.Now()
	assignments := make([]model.Assignment, 0)

	for _, assignment := range data.Assignments {
		if assignment.DueAt == nil {
			continue
		}
		due := assignment.DueAt.In(time.Local)
		if due.After(now) {
			assignments = append(assignments, assignment)
		}
	}

	sort.Slice(assignments, func(i, j int) bool {
		return assignments[i].DueAt.In(time.Local).Before(assignments[j].DueAt.In(time.Local))
	})

	var lines []string
	for _, assignment := range assignments {
		due := assignment.DueAt.In(time.Local)
		lines = append(lines, fmt.Sprintf(
			"%s\n%s",
			assignment.Name,
			due.Format("Mon 1/2 3:04 PM"),
		))

		if len(lines) == 3 {
			break
		}
	}

	if len(lines) == 0 {
		return "Assignments\n\nNo upcoming assignments."
	}

	return strings.Join(lines, "\n\n")
}

// Redraw the dashboard panels after sync
func RefreshMainWidgets(widgets []tview.Primitive, data *model.AppData) {
	if len(widgets) < 5 {
		return
	}

	if tv, ok := widgets[0].(*tview.TextView); ok {
		tv.SetText(buildCalendarSummary(data))
	}
	if tv, ok := widgets[1].(*tview.TextView); ok {
		tv.SetText(buildCoursesSummary(data))
	}
	if tv, ok := widgets[2].(*tview.TextView); ok {
		tv.SetText(buildTasksSummary(data))
	}
	if tv, ok := widgets[3].(*tview.TextView); ok {
		tv.SetText(buildScheduleSummary(data))
	}
	if tv, ok := widgets[4].(*tview.TextView); ok {
		tv.SetText(buildAssignmentsSummary(data))
	}
}

func sameLocalDay(a, b time.Time) bool {
	la := a.In(time.Local)
	lb := b.In(time.Local)
	ay, am, ad := la.Date()
	by, bm, bd := lb.Date()
	return ay == by && am == bm && ad == bd
}
