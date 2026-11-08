package calendar

import (
	"VIO/internal/asciiart"
	"VIO/internal/model"
	"VIO/internal/views/schedule"
	"fmt"
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

func CalendarPage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive {
	now := time.Now()
	state := &calendarState{
		year:        now.Year(),
		month:       now.Month(),
		selectedDay: now.Day(),
	}

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
	title.SetText(asciiart.GetMonthHeader(state.month.String()))

	header := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(quit, 0, 4, false). // narrow weight
		AddItem(title, 0, 6, false) // wider weight

	// Box on left-middle side of screen that shows days of the month
	calendarBox := tview.NewTextView()
	calendarBox.SetDynamicColors(true)
	calendarBox.SetBorder(true)
	calendarBox.SetTitle("[ CALENDAR ]")

	// Mini daily schedule menu that lists all meetings for the day
	detailBox := tview.NewTextView()
	detailBox.SetDynamicColors(true)
	detailBox.SetBorder(true)
	detailBox.SetTitle("[ SELECTED DAY ]")

	//dailyScheduleMini.SetText(renderTodayDue(data))

	render := func() {
		title.SetText(asciiart.GetMonthHeader(state.month.String()))
		calendarBox.SetText(renderCalendarGrid(data, state))
		detailBox.SetText(renderSelectedDayItems(data, state))
	}
	render()

	// Assembling the calendar box and mini daily schedule with all paddings
	mainBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(calendarBox, 0, 5, false).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(detailBox, 0, 3, false).
		AddItem(tview.NewBox(), 0, 1, false)

	// Footer below calendar and mini daily schedule menu
	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[ < ^ v > ] MOVE DAY    [ ENTER ] OPEN DAY    [ N ] NEXT MONTH    [ P ] PREV MONTH")

	// Bringing header, main body, and footer together
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 0, 2, false).
		AddItem(mainBody, 0, 5, false).
		AddItem(footer, 0, 1, false)

	// Listening for escape key to quit back to main page
	page.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			returnTo()
			return nil
		case tcell.KeyEnter:
			selected := time.Date(state.year, state.month, state.selectedDay, 0, 0, 0, 0, time.Local)
			dayPage := schedule.DayPage(app, data, selected, "CALENDAR", func() {
				app.SetRoot(page, true).SetFocus(page)
			})
			app.SetRoot(dayPage, true).SetFocus(dayPage)
			return nil
		case tcell.KeyLeft:
			moveSelectedDay(state, -1)
		case tcell.KeyRight:
			moveSelectedDay(state, 1)
		case tcell.KeyUp:
			moveSelectedDay(state, -7)
		case tcell.KeyDown:
			moveSelectedDay(state, 7)
		case tcell.KeyRune:
			switch event.Rune() {
			case 'n', 'N':
				changeMonth(state, 1)
			case 'p', 'P':
				changeMonth(state, -1)
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
	current := time.Date(state.year, state.month, state.selectedDay, 0, 0, 0, 0, time.Local)
	next := current.AddDate(0, 0, delta)
	state.year = next.Year()
	state.month = next.Month()
	state.selectedDay = next.Day()
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

func renderCalendarGrid(data *model.AppData, state *calendarState) string {
	const top = "+------+"
	const empty = "|      |"

	var b strings.Builder
	b.WriteString(" Sun     Mon     Tue     Wed     Thu     Fri     Sat   \n")

	firstDay := time.Date(state.year, state.month, 1, 0, 0, 0, 0, time.Local)
	startWeekday := int(firstDay.Weekday())
	totalDays := daysInMonth(state.year, state.month)
	now := time.Now()
	currentDay := 1

	for week := 0; week < 6; week++ {
		weekDays := make([]int, 7)

		for col := 0; col < 7; col++ {
			cellIndex := week*7 + col
			if cellIndex < startWeekday || currentDay > totalDays {
				weekDays[col] = 0
				continue
			}
			weekDays[col] = currentDay
			currentDay++
		}

		for i := 0; i < 7; i++ {
			b.WriteString(top)
		}
		b.WriteString("\n")

		for _, day := range weekDays {
			if day == 0 {
				b.WriteString(empty)
				continue
			}

			selected := day == state.selectedDay
			today := day == now.Day() && state.month == now.Month() && state.year == now.Year()
			b.WriteString(dayLine(day, selected, today))
		}
		b.WriteString("\n")

		for _, day := range weekDays {
			if day == 0 {
				b.WriteString(empty)
				continue
			}

			count := countItemsForDay(data, state.year, state.month, day)
			selected := day == state.selectedDay
			b.WriteString(itemLine(count, selected))
		}
		b.WriteString("\n")
	}

	for i := 0; i < 7; i++ {
		b.WriteString(top)
	}
	b.WriteString("\n")

	return b.String()
}

func emptyCellLine() string {
	return "|          |"
}

func dayLine(day int, selected bool, today bool) string {
	var dayText string

	switch {
	case selected && today:
		dayText = fmt.Sprintf("[black:green]%2d[-:-]", day)
	case selected:
		dayText = fmt.Sprintf("[black:white]%2d[-:-]", day)
	case today:
		dayText = fmt.Sprintf("[green]%2d[-]", day)
	default:
		dayText = fmt.Sprintf("%2d", day)
	}

	return fmt.Sprintf("|  %s  |", dayText)
}

func itemLine(count int, selected bool) string {
	label := ""
	if count > 0 {
		label = "***"
	}

	return fmt.Sprintf("| %-4s |", label)
}

func padCell(content string) string {
	return fmt.Sprintf("|%-10s|", content)
}

func renderSelectedDayItems(data *model.AppData, state *calendarState) string {
	var lines []string
	selectedDate := time.Date(state.year, state.month, state.selectedDay, 0, 0, 0, 0, time.Local)
	lines = append(lines, selectedDate.Format("Monday, January 2, 2006"))

	for _, a := range data.Assignments {
		if a.DueAt == nil {
			continue
		}
		if a.DueAt.Year() == state.year && a.DueAt.Month() == state.month && a.DueAt.Day() == state.selectedDay {
			lines = append(lines, fmt.Sprintf("\nAssignment:\n%s\nDue: %s", a.Name, a.DueAt.Format("3:04 PM")))
		}
	}

	for _, t := range data.Tasks {
		if t.DueAt == nil {
			continue
		}
		if t.DueAt.Year() == state.year && t.DueAt.Month() == state.month && t.DueAt.Day() == state.selectedDay {
			status := taskStatusLabel(t)
			lines = append(lines, fmt.Sprintf("\nTask:\n%s\nDue: %s\nStatus: %s", t.Title, t.DueAt.Format("3:04 PM"), status))
		}
	}

	if len(lines) == 1 {
		lines = append(lines, "\nNo items for this day.")
	}

	return strings.Join(lines, "\n")
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
