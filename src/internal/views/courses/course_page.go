package courses

import (
	"VIO/internal/dates"
	"VIO/internal/model"
	"VIO/internal/theme"
	"VIO/internal/views/detail"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rivo/tview"
)

// CourseDetailPage shows everything about one course: meeting times (and
// any overlaps with other classes), open assignments, open tasks, and
// notes linked to it. index is the course's position in data.Courses.
// onEditMeetings, when not nil, is offered on the m key.
func CourseDetailPage(data *model.AppData, index int, onEditMeetings func(), returnTo func()) tview.Primitive {
	hints := theme.KeyHints("j/k", "scroll", "esc", "back")
	var onKey func(rune) bool
	if onEditMeetings != nil {
		hints = theme.KeyHints("m", "edit meeting times", "j/k", "scroll", "esc", "back")
		onKey = func(r rune) bool {
			if r == 'm' || r == 'M' {
				onEditMeetings()
				return true
			}
			return false
		}
	}
	page, _, _ := detail.Page("Course", renderCoursePage(data, index), hints, onKey, returnTo)
	return page
}

func renderCoursePage(data *model.AppData, index int) string {
	if index < 0 || index >= len(data.Courses) {
		return theme.TagMuted + "This course no longer exists." + theme.TagReset
	}
	now := time.Now()
	c := data.Courses[index]
	color := theme.ColorTag(theme.CourseColor(index))

	var b strings.Builder
	b.WriteString(color + "▌ " + theme.TagReset + theme.TagBold + tview.Escape(c.Name) + theme.TagReset + "\n\n")
	if c.Code != "" && !strings.EqualFold(c.Code, c.Name) {
		b.WriteString(detail.Field("Code", tview.Escape(c.Code)))
	}
	if c.Term != "" {
		b.WriteString(detail.Field("Term", tview.Escape(c.Term)))
	}
	if c.Location != "" {
		b.WriteString(detail.Field("Room", tview.Escape(c.Location)))
	}
	if c.CanvasID != 0 {
		b.WriteString(detail.Field("Canvas", fmt.Sprintf("course %d · %s", c.CanvasID, tview.Escape(c.Workflow))))
	}

	b.WriteString(detail.Section("Meeting times"))
	if len(c.Meetings) == 0 {
		b.WriteString(theme.TagMuted + "None yet." + theme.TagReset + "\n")
	}
	for _, m := range c.Meetings {
		s, e, _ := m.Minutes()
		b.WriteString(color + "▌ " + theme.TagReset + strings.Join(m.Days, " ") + "  " + model.FormatRange(s, e, false))
		if m.Location != "" {
			b.WriteString(theme.TagMuted + "  @ " + tview.Escape(m.Location) + theme.TagReset)
		}
		b.WriteString("\n")
	}
	b.WriteString(conflictLines(data, index))

	// Open assignments for this course, soonest first
	var open []model.Assignment
	for _, a := range data.Assignments {
		if a.CourseID == c.ID && !a.Done() {
			open = append(open, a)
		}
	}
	sort.SliceStable(open, func(i, j int) bool {
		if open[i].DueAt == nil || open[j].DueAt == nil {
			return open[j].DueAt == nil && open[i].DueAt != nil
		}
		return open[i].DueAt.Before(*open[j].DueAt)
	})
	b.WriteString(detail.Section(fmt.Sprintf("Assignments to do (%d)", len(open))))
	if len(open) == 0 {
		b.WriteString(theme.TagMuted + "All caught up." + theme.TagReset + "\n")
	}
	for _, a := range open {
		mark := theme.TagMuted + "○ " + theme.TagReset
		if a.NeedsAttention(now) {
			mark = theme.TagRed + "● " + theme.TagReset
		}
		line := mark + tview.Escape(a.Name)
		if a.DueAt != nil {
			due := a.DueAt.In(time.Local)
			line += theme.TagMuted + "  due " + due.Format("Mon Jan 2, 3:04 PM") + " (" + dates.Relative(due, now) + ")" + theme.TagReset
		}
		b.WriteString(line + "\n")
	}

	// Open tasks for this course
	var taskLines []string
	for _, t := range data.Tasks {
		if t.CourseID != c.ID || strings.EqualFold(t.Status, "complete") {
			continue
		}
		line := theme.TagYellow + "● " + theme.TagReset + tview.Escape(t.Title)
		if t.DueAt != nil {
			line += theme.TagMuted + "  due " + t.DueAt.In(time.Local).Format("Mon Jan 2, 3:04 PM") + theme.TagReset
		}
		taskLines = append(taskLines, line)
	}
	b.WriteString(detail.Section(fmt.Sprintf("Tasks (%d)", len(taskLines))))
	if len(taskLines) == 0 {
		b.WriteString(theme.TagMuted + "No open tasks for this course." + theme.TagReset + "\n")
	} else {
		b.WriteString(strings.Join(taskLines, "\n") + "\n")
	}

	// Linked notes
	var noteLines []string
	for _, n := range data.Notes {
		if n.CourseID == c.ID {
			noteLines = append(noteLines, theme.TagAccent+"▸ "+theme.TagReset+tview.Escape(n.Title)+
				theme.TagMuted+"  "+n.UpdatedAt.In(time.Local).Format("Jan 2")+theme.TagReset)
		}
	}
	if len(noteLines) > 0 {
		b.WriteString(detail.Section(fmt.Sprintf("Notes (%d)", len(noteLines))))
		b.WriteString(strings.Join(noteLines, "\n") + "\n")
	}

	if strings.TrimSpace(c.Notes) != "" {
		b.WriteString(detail.Section("Course notes"))
		b.WriteString(tview.Escape(c.Notes) + "\n")
	}
	return b.String()
}

// conflictLines lists overlaps between this course's meetings and any
// other meeting, or "" when there are none.
func conflictLines(data *model.AppData, index int) string {
	var lines []string
	for _, cf := range model.MeetingConflicts(data.Courses) {
		var other int
		switch index {
		case cf.CourseA:
			other = cf.CourseB
		case cf.CourseB:
			other = cf.CourseA
		default:
			continue
		}
		with := "itself"
		if other != index {
			o := data.Courses[other]
			with = theme.ColorTag(theme.CourseColor(other)) + tview.Escape(model.ShortCode(o.Code, o.Name)) + theme.TagYellow
		}
		lines = append(lines, fmt.Sprintf("%s⚠ overlaps %s on %s, %s%s",
			theme.TagYellow, with, cf.Day, model.FormatRange(cf.Start, cf.End, false), theme.TagReset))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
