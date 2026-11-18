package calendar

import (
	"VIO/internal/asciiart"
	"VIO/internal/model"
	"VIO/internal/theme"
	"VIO/internal/views/schedule"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type calendarState struct {
	year        int
	month       time.Month
	selectedDay int
}

func (s *calendarState) selected() time.Time {
	return time.Date(s.year, s.month, s.selectedDay, 0, 0, 0, 0, time.Local)
}

func (s *calendarState) setSelected(t time.Time) {
	s.year, s.month, s.selectedDay = t.Year(), t.Month(), t.Day()
}

type viewMode int

const (
	weekMode viewMode = iota
	monthMode
)

func CalendarPage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive {
	state := &calendarState{}
	state.setSelected(time.Now())
	mode := weekMode

	// Header

	quitPadding := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(`


		`)

	quitText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[white::b][ ESC ] To RETURN TO MAIN")

	quit := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(quitPadding, 2, 1, false).
		AddItem(quitText, 0, 2, false)

	title := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	header := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(quit, 0, 4, false). // narrow weight
		AddItem(title, 0, 6, false) // wider weight

	// Week view: classes, timed tasks, and due assignments on a time grid
	week := newWeekGrid(data, state)
	theme.FramePanel(week.Box, "Week", true)

	weekBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(week, 0, 18, false).
		AddItem(tview.NewBox(), 0, 1, false)

	// Month view: day grid plus the selected day's details
	calendarBox := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false). // clip instead of wrapping when narrow
		SetTextAlign(tview.AlignCenter)
	theme.FramePanel(calendarBox.Box, "Month", true)

	detailBox := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true)
	theme.FramePanel(detailBox.Box, "Selected day", false)

	monthBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(calendarBox, 0, 5, false).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(detailBox, 0, 3, false).
		AddItem(tview.NewBox(), 0, 1, false)

	body := tview.NewPages().
		AddPage("week", weekBody, true, true).
		AddPage("month", monthBody, true, false)

	weekHelp := theme.KeyHints("←/→", "day", "n/p", "week", "j/k", "scroll", "enter", "open day", "t", "today", "v", "month view")
	monthHelp := theme.KeyHints("arrows", "move day", "n/p", "month", "enter", "open day", "t", "today", "v", "week view")

	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	render := func() {
		title.SetText(asciiart.GetMonthHeader(state.month.String()))
		if mode == weekMode {
			footer.SetText(weekHelp)
			return
		}
		calendarBox.SetText(renderCalendarGrid(data, state))
		detailBox.SetText(renderSelectedDayItems(data, state))
		footer.SetText(monthHelp)
	}
	render()

	// Bringing header, main body, and footer together
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 0, 2, false).
		AddItem(body, 0, 5, false).
		AddItem(footer, 0, 1, false)

	page.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			returnTo()
			return nil
		case tcell.KeyEnter:
			dayPage := schedule.DayPage(app, data, state.selected(), "CALENDAR", func() {
				app.SetRoot(page, true).SetFocus(page)
			})
			app.SetRoot(dayPage, true).SetFocus(dayPage)
			return nil
		case tcell.KeyLeft:
			moveSelectedDay(state, -1)
		case tcell.KeyRight:
			moveSelectedDay(state, 1)
		case tcell.KeyUp:
			if mode == weekMode {
				week.scrollBy(-2)
			} else {
				moveSelectedDay(state, -7)
			}
		case tcell.KeyDown:
			if mode == weekMode {
				week.scrollBy(2)
			} else {
				moveSelectedDay(state, 7)
			}
		case tcell.KeyRune:
			switch event.Rune() {
			case 'h':
				moveSelectedDay(state, -1)
			case 'l':
				moveSelectedDay(state, 1)
			case 'k':
				week.scrollBy(-2)
			case 'j':
				week.scrollBy(2)
			case 'n', 'N':
				if mode == weekMode {
					moveSelectedDay(state, 7)
				} else {
					changeMonth(state, 1)
				}
			case 'p', 'P':
				if mode == weekMode {
					moveSelectedDay(state, -7)
				} else {
					changeMonth(state, -1)
				}
			case 't', 'T':
				state.setSelected(time.Now())
			case 'v', 'V':
				if mode == weekMode {
					mode = monthMode
					body.SwitchToPage("month")
				} else {
					mode = weekMode
					body.SwitchToPage("week")
				}
			default:
				return event
			}
		default:
			return event
		}

		clampSelectedDay(state)
		render()
		return nil
	})

	return page
}

func moveSelectedDay(state *calendarState, delta int) {
	state.setSelected(state.selected().AddDate(0, 0, delta))
}

func changeMonth(state *calendarState, delta int) {
	next := time.Date(state.year, state.month, 1, 0, 0, 0, 0, time.Local).AddDate(0, delta, 0)
	state.year = next.Year()
	state.month = next.Month()
}

func clampSelectedDay(state *calendarState) {
	days := daysInMonth(state.year, state.month)
	if state.selectedDay < 1 {
		state.selectedDay = 1
	}
	if state.selectedDay > days {
		state.selectedDay = days
	}
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func countItemsForDay(data *model.AppData, year int, month time.Month, day int) int {
	count := 0

	for _, a := range data.Assignments {
		if a.DueAt == nil {
			continue
		}
		if a.DueAt.Year() == year && a.DueAt.Month() == month && a.DueAt.Day() == day {
			count++
		}
	}

	for _, t := range data.Tasks {
		if t.DueAt == nil {
			continue
		}
		if t.DueAt.Year() == year && t.DueAt.Month() == month && t.DueAt.Day() == day {
			count++
		}
	}

	return count
}

// renderCalendarGrid draws the month as a box-drawn grid in the theme's
// border color. Each cell is 6 wide: the day number, then a ● count of
// things due.
func renderCalendarGrid(data *model.AppData, state *calendarState) string {
	const cellW = 6
	rule := func(left, mid, right string) string {
		seg := strings.Repeat("─", cellW)
		return theme.TagBorder + left + strings.Repeat(seg+mid, 6) + seg + right + theme.TagReset + "\n"
	}
	bar := theme.TagBorder + "│" + theme.TagReset

	var b strings.Builder
	b.WriteString(theme.TagMuted + " ")
	for _, name := range []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"} {
		b.WriteString(fmt.Sprintf("  %-3s ", name) + " ")
	}
	b.WriteString(theme.TagReset + "\n")

	firstDay := time.Date(state.year, state.month, 1, 0, 0, 0, 0, time.Local)
	startWeekday := int(firstDay.Weekday())
	totalDays := daysInMonth(state.year, state.month)
	now := time.Now()
	currentDay := 1

	b.WriteString(rule("╭", "┬", "╮"))
	for week := 0; week < 6; week++ {
		weekDays := make([]int, 7)
		for col := 0; col < 7; col++ {
			cellIndex := week*7 + col
			if cellIndex < startWeekday || currentDay > totalDays {
				continue
			}
			weekDays[col] = currentDay
			currentDay++
		}

		var dayRow, itemRow strings.Builder
		for _, day := range weekDays {
			dayRow.WriteString(bar)
			itemRow.WriteString(bar)
			if day == 0 {
				dayRow.WriteString(strings.Repeat(" ", cellW))
				itemRow.WriteString(strings.Repeat(" ", cellW))
				continue
			}
			selected := day == state.selectedDay
			today := day == now.Day() && state.month == now.Month() && state.year == now.Year()
			dayRow.WriteString(dayCell(day, selected, today))
			itemRow.WriteString(itemCell(countItemsForDay(data, state.year, state.month, day)))
		}
		b.WriteString(dayRow.String() + bar + "\n")
		b.WriteString(itemRow.String() + bar + "\n")

		if week < 5 {
			b.WriteString(rule("├", "┼", "┤"))
		}
	}
	b.WriteString(rule("╰", "┴", "╯"))

	return b.String()
}

func dayCell(day int, selected bool, today bool) string {
	switch {
	case selected:
		return fmt.Sprintf("  [#1a1b26:#7aa2f7:b]%2d[-:-:-]  ", day)
	case today:
		return fmt.Sprintf("  %s%s%2d%s  ", theme.TagAccent, theme.TagBold, day, theme.TagReset)
	default:
		return fmt.Sprintf("  %2d  ", day)
	}
}

func itemCell(count int) string {
	if count == 0 {
		return "      "
	}
	return fmt.Sprintf("  %s●%-2d%s ", theme.TagKey, count, theme.TagReset)
}

func renderSelectedDayItems(data *model.AppData, state *calendarState) string {
	selected := state.selected()
	var b strings.Builder
	b.WriteString(theme.TagAccent + theme.TagBold + selected.Format("Monday, January 2, 2006") + theme.TagReset + "\n")
	empty := true

	// Classes, in time order
	type class struct {
		start, end int
		line       string
	}
	var classes []class
	for i, course := range data.Courses {
		for _, m := range course.Meetings {
			if !m.OnWeekday(selected.Weekday()) {
				continue
			}
			s, e, ok := m.Minutes()
			if !ok {
				continue
			}
			line := theme.ColorTag(theme.CourseColor(i)) + "▌ " + theme.TagBold +
				tview.Escape(model.ShortCode(course.Code, course.Name)) + theme.TagReset +
				"\n  " + theme.TagMuted + model.FormatRange(s, e, false)
			if m.Location != "" {
				line += "  @ " + tview.Escape(m.Location)
			}
			classes = append(classes, class{s, e, line + theme.TagReset})
		}
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i].start < classes[j].start })
	if len(classes) > 0 {
		empty = false
		b.WriteString("\n" + theme.TagMuted + "Classes" + theme.TagReset + "\n")
		for _, c := range classes {
			b.WriteString(c.line + "\n")
		}
	}

	var due []string
	for _, a := range data.Assignments {
		if a.DueAt != nil && sameDay(*a.DueAt, selected) {
			due = append(due, theme.TagRed+"● "+theme.TagReset+tview.Escape(a.Name)+
				"\n  "+theme.TagMuted+"due "+a.DueAt.In(time.Local).Format("3:04 PM")+theme.TagReset)
		}
	}
	for _, t := range data.Tasks {
		if t.DueAt != nil && sameDay(*t.DueAt, selected) {
			due = append(due, theme.TagYellow+"● "+theme.TagReset+tview.Escape(t.Title)+
				"\n  "+theme.TagMuted+"task · due "+t.DueAt.In(time.Local).Format("3:04 PM")+" · "+taskStatusLabel(t)+theme.TagReset)
		}
	}
	if len(due) > 0 {
		empty = false
		b.WriteString("\n" + theme.TagMuted + "Due" + theme.TagReset + "\n")
		b.WriteString(strings.Join(due, "\n") + "\n")
	}

	if empty {
		b.WriteString("\n" + theme.TagMuted + "Nothing scheduled." + theme.TagReset)
	}
	return b.String()
}

func taskStatusLabel(task model.Task) string {
	if strings.EqualFold(strings.TrimSpace(task.Status), "complete") {
		return "complete"
	}
	if task.DueAt != nil && task.DueAt.Before(time.Now()) {
		return "overdue"
	}
	return "in progress"
}
