package schedule

import (
	"VIO/internal/asciiart"
	"VIO/internal/model"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func SchedulePage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive {
	return DayPage(app, data, time.Now(), "MAIN", returnTo)
}

// DayPage shows the assignments and tasks due on the given day.
// returnLabel names the screen Esc goes back to (e.g. "MAIN", "CALENDAR").
func DayPage(app *tview.Application, data *model.AppData, day time.Time, returnLabel string, returnTo func()) tview.Primitive {

	// Header

	quitPadding := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(`


		`)

	quitText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetText("[white::b][ ESC ] To RETURN TO " + returnLabel)

	quit := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(quitPadding, 2, 1, false).
		AddItem(quitText, 0, 2, false)

	title := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	currMonth := day.Month().String()
	currDay := strconv.Itoa(day.Day())

	title.SetText(asciiart.GetMonthHeader(currMonth) + asciiart.GetDayHeader(currDay))

	header := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(quit, 0, 2, false). // narrow weight
		AddItem(title, 0, 6, false) // wider weight
	// Main body

	// Schedule Box

	scheduleBody := tview.NewTextView().SetDynamicColors(true)
	scheduleBody.SetText(renderUpcomingFeed(data, day))

	mainBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(scheduleBody, 0, 5, false).
		AddItem(tview.NewBox(), 0, 1, false)

	// Footer

	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[ + ] ADD   [ - ] REMOVE   [ E ] EDIT")

	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 0, 3, false).
		AddItem(mainBody, 0, 5, false).
		AddItem(footer, 0, 1, false)

	page.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			returnTo()
			return nil
		}
		return event
	})

	return page
}

func renderUpcomingFeed(data *model.AppData, day time.Time) string {
	type item struct {
		label string
		due   time.Time
	}

	var items []item

	for _, assignment := range data.Assignments {
		if assignment.DueAt == nil || !sameLocalDay(*assignment.DueAt, day) {
			continue
		}

		items = append(items, item{
			label: fmt.Sprintf("Assignment: %s", assignment.Name),
			due:   assignment.DueAt.In(time.Local),
		})
	}

	for _, task := range data.Tasks {
		if task.DueAt == nil || !sameLocalDay(*task.DueAt, day) {
			continue
		}

		items = append(items, item{
			label: fmt.Sprintf("Task: %s (%s)", task.Title, strings.ReplaceAll(task.Status, "_", " ")),
			due:   task.DueAt.In(time.Local),
		})
	}

	if len(items) == 0 {
		return "No items for this day."
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].due.Before(items[j].due)
	})

	var lines []string
	for _, it := range items {
		lines = append(lines, fmt.Sprintf("%s\n%s", it.label, it.due.Format("Mon Jan 2, 3:04 PM")))
	}

	return strings.Join(lines, "\n\n")
}

func sameLocalDay(a, b time.Time) bool {
	la := a.In(time.Local)
	lb := b.In(time.Local)
	ay, am, ad := la.Date()
	by, bm, bd := lb.Date()
	return ay == by && am == bm && ad == bd
}
