package widgets

import (
	"VIO/internal/model"
	"VIO/internal/theme"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rivo/tview"
)

// BuildMainWidgets returns the list of box widgets, the main layout, and
// the Canvas sync status line shown in the header.
func BuildMainWidgets(data *model.AppData) ([]tview.Primitive, *tview.Flex, *SyncStatus) {

	// Widgets

	// Dashboard panels
	newPanel := func(n int, name string) *tview.TextView {
		tv := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
		theme.StylePane(tv.Box, n, name)
		return tv
	}

	wCalendar := newPanel(1, "Calendar")
	wCourses := newPanel(2, "Courses")
	wTodo := newPanel(3, "Tasks")
	wSchedule := newPanel(4, "Today")
	wNotes := newPanel(5, "Notes")

	wCalendar.SetText(buildCalendarSummary(data))
	wCourses.SetText(buildCoursesSummary(data))
	wTodo.SetText(buildTasksSummary(data))
	wSchedule.SetText(buildScheduleSummary(data))
	wNotes.SetText(buildNotesSummary(data))

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
				AddItem(wNotes, 0, 1, false),
			0, 1, false)

	quitPadding := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(`


		`)

	quitText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[white::b][ q ] To QUIT    [ c ] Canvas Settings\n[ a ] Assignments")

	status := newSyncStatus()

	quit := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(quitPadding, 1, 0, false).
		AddItem(quitText, 2, 0, false).
		AddItem(status.view, 1, 0, false)

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

	widgets := []tview.Primitive{wCalendar, wCourses, wTodo, wSchedule, wNotes}
	return widgets, flex, status
}

func buildCalendarSummary(data *model.AppData) string {
	now := time.Now()
	type due struct {
		at   time.Time
		line string
	}
	var items []due
	missing := 0

	for _, assignment := range data.Assignments {
		if assignment.NeedsAttention(now) {
			missing++
		}
		if assignment.DueAt == nil || assignment.Done() {
			continue
		}
		at := assignment.DueAt.In(time.Local)
		if at.Month() != now.Month() || at.Year() != now.Year() {
			continue
		}
		line := fmt.Sprintf("%02d  %s", at.Day(), tview.Escape(assignment.Name))
		if assignment.NeedsAttention(now) {
			line = theme.TagRed + line + theme.TagReset
		} else if at.Before(now) {
			line = theme.TagMuted + line + theme.TagReset
		}
		items = append(items, due{at, line})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].at.Before(items[j].at) })

	head := theme.TagBold + now.Month().String() + theme.TagReset
	if missing > 0 {
		head += fmt.Sprintf("   %s● %d need attention%s %s(a)%s", theme.TagRed, missing, theme.TagReset, theme.TagMuted, theme.TagReset)
	}

	if len(items) == 0 {
		return head + "\n\n" + theme.TagMuted + "Nothing left to submit this month." + theme.TagReset
	}
	if len(items) > 8 {
		items = items[:8]
	}
	lines := make([]string, len(items))
	for i, it := range items {
		lines[i] = it.line
	}

	return head + "\n\n" + strings.Join(lines, "\n")
}

func buildCoursesSummary(data *model.AppData) string {
	if len(data.Courses) == 0 {
		return "No courses loaded."
	}

	var lines []string
	for _, course := range data.Courses {
		name := tview.Escape(course.Name)
		if strings.TrimSpace(course.Code) == "" || strings.EqualFold(strings.TrimSpace(course.Code), strings.TrimSpace(course.Name)) {
			lines = append(lines, theme.TagBold+name+theme.TagReset)
			continue
		}
		lines = append(lines, fmt.Sprintf("%s%s%s\n%s%s%s", theme.TagBold, name, theme.TagReset, theme.TagMuted, tview.Escape(course.Code), theme.TagReset))
	}

	if n := len(model.MeetingConflicts(data.Courses)); n > 0 {
		plural := "s"
		if n == 1 {
			plural = ""
		}
		lines = append(lines, fmt.Sprintf("%s⚠ %d class time overlap%s%s", theme.TagYellow, n, plural, theme.TagReset))
	}

	return strings.Join(lines, "\n\n")
}

func buildTasksSummary(data *model.AppData) string {
	if len(data.Tasks) == 0 {
		return "No personal tasks yet."
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

// buildScheduleSummary lists today's classes, assignments and tasks in
// time order, dimming what's already over.
func buildScheduleSummary(data *model.AppData) string {
	now := time.Now()
	items := data.Agenda(now, false)
	if len(items) == 0 {
		return theme.TagMuted + "Nothing scheduled today." + theme.TagReset
	}

	var lines []string
	for _, it := range items {
		color := theme.TagMuted
		if it.CourseIndex >= 0 {
			color = theme.ColorTag(theme.CourseColor(it.CourseIndex))
		}
		var when, what string
		past := it.At.Before(now)
		switch it.Kind {
		case model.ItemClass:
			when = it.At.Format("3:04") + "–" + it.End.Format("3:04 PM")
			what = color + "▌ " + theme.TagReset + theme.TagBold + tview.Escape(it.Title) + theme.TagReset
			if it.Location != "" {
				what += theme.TagMuted + " @ " + tview.Escape(it.Location) + theme.TagReset
			}
			past = it.End.Before(now)
		case model.ItemAssignment:
			when = "due " + it.At.Format("3:04 PM")
			what = theme.TagRed + "● " + theme.TagReset + tview.Escape(it.Title)
		case model.ItemTask:
			when = "due " + it.At.Format("3:04 PM")
			mark := theme.TagYellow + "● " + theme.TagReset
			if strings.EqualFold(data.Tasks[it.Index].Status, "complete") {
				mark = theme.TagGreen + "✓ " + theme.TagReset
				past = true
			}
			what = mark + tview.Escape(it.Title)
		}
		line := theme.TagMuted + fmt.Sprintf("%-16s", when) + theme.TagReset + what
		if past {
			line = theme.TagMuted + fmt.Sprintf("%-16s", when) + tview.Escape(it.Title) + theme.TagReset
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// Most recent notes: titles in a list, then the newest note's first lines
func buildNotesSummary(data *model.AppData) string {
	if len(data.Notes) == 0 {
		return theme.TagMuted + "No notes yet.\n\nPress 5 then Enter to open notes, n to write one." + theme.TagReset
	}

	notes := make([]model.Note, len(data.Notes))
	copy(notes, data.Notes)
	sort.SliceStable(notes, func(i, j int) bool {
		return notes[i].UpdatedAt.After(notes[j].UpdatedAt)
	})

	var lines []string
	for i, n := range notes {
		if i == 4 {
			lines = append(lines, fmt.Sprintf("%s+ %d more%s", theme.TagMuted, len(notes)-4, theme.TagReset))
			break
		}
		marker := theme.TagMuted + "  " + theme.TagReset
		if i == 0 {
			marker = theme.TagAccent + "▸ " + theme.TagReset
		}
		lines = append(lines, marker+tview.Escape(n.Title))
	}

	// Preview of the newest note's body, skipping the first line (it's the title)
	body := strings.Split(strings.TrimSpace(notes[0].Body), "\n")[1:]
	var preview []string
	for _, l := range body {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
			t = "• " + t[2:]
		}
		preview = append(preview, theme.TagMuted+tview.Escape(t)+theme.TagReset)
		if len(preview) == 4 {
			break
		}
	}
	if len(preview) > 0 {
		lines = append(lines, "")
		lines = append(lines, preview...)
	}

	return strings.Join(lines, "\n")
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
		tv.SetText(buildNotesSummary(data))
	}
}
