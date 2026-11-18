package schedule

import (
	"VIO/internal/asciiart"
	"VIO/internal/model"
	"VIO/internal/store"
	"VIO/internal/theme"
	"VIO/internal/views/assignments"
	"VIO/internal/views/courses"
	"VIO/internal/views/tasks"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func SchedulePage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive {
	return DayPage(app, data, time.Now(), "MAIN", returnTo)
}

// DayPage shows one day's agenda: class meetings, assignments due and
// tasks due, in time order. + adds a task due that day; e, - and x edit,
// delete and complete the selected task; Enter opens any item. ←/→ (h/l)
// move between days and t jumps to today. returnLabel names the screen
// Esc goes back to (e.g. "MAIN", "CALENDAR").
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

	header := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(quit, 0, 2, false). // narrow weight
		AddItem(title, 0, 6, false) // wider weight

	// Main body: the day's agenda as a selectable list

	list := tview.NewList().
		SetHighlightFullLine(true).
		SetSelectedStyle(tcell.StyleDefault.Background(theme.ColSurface).Foreground(theme.ColFg)).
		SetMainTextStyle(tcell.StyleDefault.Foreground(theme.ColFg)).
		SetSecondaryTextStyle(tcell.StyleDefault.Foreground(theme.ColMuted)).
		SetWrapAround(false)
	theme.FramePanel(list.Box, "", true)

	mainBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(list, 0, 5, true).
		AddItem(tview.NewBox(), 0, 1, false)

	// Footer

	const help = "[ + ] ADD TASK   [ - ] REMOVE   [ E ] EDIT   [ X ] DONE   [ ENTER ] OPEN   [ < > ] DAY   [ T ] TODAY"
	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(help)
	setStatus := func(msg string) {
		if msg == "" {
			footer.SetText(help)
			return
		}
		footer.SetText(help + "\n\n" + msg)
	}

	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 0, 3, false).
		AddItem(mainBody, 0, 5, true).
		AddItem(footer, 0, 1, false)

	// State

	var items []model.AgendaItem

	render := func(keep int) {
		title.SetText(asciiart.GetMonthHeader(day.Month().String()) + asciiart.GetDayHeader(strconv.Itoa(day.Day())))
		list.SetTitle(" " + day.Format("Monday, January 2") + dayLabel(day) + " ")
		items = data.Agenda(day, true)
		fillAgenda(list, data, items)
		if keep >= len(items) {
			keep = len(items) - 1
		}
		if keep < 0 {
			keep = 0
		}
		list.SetCurrentItem(keep)
	}

	showPage := func() { app.SetRoot(page, true).SetFocus(list) }

	selected := func() (model.AgendaItem, bool) {
		i := list.GetCurrentItem()
		if i < 0 || i >= len(items) {
			return model.AgendaItem{}, false
		}
		return items[i], true
	}

	persist := func() {
		if err := store.SaveAppData(*data); err != nil {
			setStatus(theme.TagRed + "Saving failed: " + tview.Escape(err.Error()) + theme.TagReset)
			return
		}
		setStatus("")
	}

	selectTask := func(id string) {
		for i, it := range items {
			if it.Kind == model.ItemTask && data.Tasks[it.Index].ID == id {
				list.SetCurrentItem(i)
				return
			}
		}
	}

	openTaskForm := func(index int) {
		task := model.Task{Status: "in_progress"}
		formTitle := "[ NEW TASK ]"
		if index >= 0 {
			task = data.Tasks[index]
			formTitle = "[ EDIT TASK ]"
		} else {
			// Default the due date to the end of the day being viewed
			y, m, d := day.Date()
			due := time.Date(y, m, d, 23, 59, 0, 0, time.Local)
			task.DueAt = &due
		}
		form := tasks.TaskForm(data, task, formTitle, func(saved model.Task) {
			if index >= 0 {
				data.Tasks[index] = saved
			} else {
				data.Tasks = append(data.Tasks, saved)
			}
			persist()
			// A task moved to another day jumps the view there
			if saved.DueAt != nil && !model.SameLocalDay(*saved.DueAt, day) {
				day = saved.DueAt.In(time.Local)
			}
			render(0)
			selectTask(saved.ID)
			showPage()
		}, showPage)
		app.SetRoot(form, true).SetFocus(form)
	}

	onlyTasks := func(it model.AgendaItem, verb string) bool {
		if it.Kind == model.ItemTask {
			return true
		}
		what := "Class meetings"
		hint := "edit them from Courses (m)"
		if it.Kind == model.ItemAssignment {
			what = "Assignments"
			hint = "manage them from Assignments (a on the dashboard)"
		}
		setStatus(theme.TagYellow + what + " can't be " + verb + " here; " + hint + "." + theme.TagReset)
		return false
	}

	confirmDelete := func(index int) {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Delete task \"%s\"?", data.Tasks[index].Title)).
			AddButtons([]string{"Delete", "Cancel"}).
			SetDoneFunc(func(_ int, label string) {
				keep := list.GetCurrentItem()
				if label == "Delete" {
					data.Tasks = slices.Delete(data.Tasks, index, index+1)
					persist()
					render(keep)
				}
				showPage()
			})
		app.SetRoot(modal, true).SetFocus(modal)
	}

	openItem := func(it model.AgendaItem) {
		keep := list.GetCurrentItem()
		back := func() {
			render(keep)
			showPage()
		}
		var p tview.Primitive
		switch it.Kind {
		case model.ItemClass:
			p = courses.CourseDetailPage(data, it.Index, nil, back)
		case model.ItemAssignment:
			id := data.Assignments[it.Index].ID
			p = assignments.AssignmentDetailPage(app, data, id, func(string) { back() })
		case model.ItemTask:
			p = tasks.TaskDetailPage(app, data, data.Tasks[it.Index].ID, back)
		}
		app.SetRoot(p, true).SetFocus(p)
	}

	moveDay := func(delta int) {
		day = day.AddDate(0, 0, delta)
		setStatus("")
		render(0)
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			returnTo()
			return nil
		case tcell.KeyLeft:
			moveDay(-1)
			return nil
		case tcell.KeyRight:
			moveDay(1)
			return nil
		case tcell.KeyEnter:
			if it, ok := selected(); ok {
				openItem(it)
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'h':
				moveDay(-1)
				return nil
			case 'l':
				moveDay(1)
				return nil
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case 't', 'T':
				day = time.Now()
				render(0)
				return nil
			case '+', '=', 'a', 'A':
				openTaskForm(-1)
				return nil
			case 'e', 'E':
				if it, ok := selected(); ok && onlyTasks(it, "edited") {
					openTaskForm(it.Index)
				}
				return nil
			case '-', 'd', 'D':
				if it, ok := selected(); ok && onlyTasks(it, "removed") {
					confirmDelete(it.Index)
				}
				return nil
			case 'x', 'X', ' ':
				if it, ok := selected(); ok && onlyTasks(it, "checked off") {
					keep := list.GetCurrentItem()
					tasks.ToggleComplete(&data.Tasks[it.Index])
					persist()
					render(keep)
				}
				return nil
			}
		}
		return event
	})

	render(0)
	return page
}

// dayLabel adds " · today" / " · tomorrow" / " · yesterday" when it applies.
func dayLabel(day time.Time) string {
	now := time.Now()
	switch {
	case model.SameLocalDay(day, now):
		return " · today"
	case model.SameLocalDay(day, now.AddDate(0, 0, 1)):
		return " · tomorrow"
	case model.SameLocalDay(day, now.AddDate(0, 0, -1)):
		return " · yesterday"
	}
	return ""
}

func fillAgenda(list *tview.List, data *model.AppData, items []model.AgendaItem) {
	list.Clear()
	if len(items) == 0 {
		list.AddItem(theme.TagMuted+"Nothing scheduled. Press + to add a task for this day."+theme.TagReset, "", 0, nil)
		return
	}
	now := time.Now()
	for _, it := range items {
		color := theme.TagMuted
		if it.CourseIndex >= 0 {
			color = theme.ColorTag(theme.CourseColor(it.CourseIndex))
		}
		var main, sub string
		switch it.Kind {
		case model.ItemClass:
			main = color + "▌ " + theme.TagReset + theme.TagBold + tview.Escape(it.Title) + theme.TagReset +
				theme.TagMuted + "  class" + theme.TagReset
			sub = "   " + it.At.Format("3:04 PM") + " – " + it.End.Format("3:04 PM")
			if it.Location != "" {
				sub += "  @ " + tview.Escape(it.Location)
			}
		case model.ItemAssignment:
			a := data.Assignments[it.Index]
			main = theme.TagRed + "● " + theme.TagReset + tview.Escape(it.Title) + "  " + assignments.StateBadge(a.State(now))
			sub = "   assignment · due " + it.At.Format("3:04 PM") + courseSuffix(data, it.CourseIndex)
		case model.ItemTask:
			t := data.Tasks[it.Index]
			mark := theme.TagYellow + "● " + theme.TagReset
			name := tview.Escape(it.Title)
			if strings.EqualFold(t.Status, "complete") {
				mark = theme.TagGreen + "✓ " + theme.TagReset
				name = theme.TagMuted + name + theme.TagReset
			}
			main = mark + name
			sub = "   task · due " + it.At.Format("3:04 PM") + courseSuffix(data, it.CourseIndex)
		}
		list.AddItem(main, sub, 0, nil)
	}
}

func courseSuffix(data *model.AppData, ci int) string {
	if ci < 0 {
		return ""
	}
	return " · " + tview.Escape(model.ShortCode(data.Courses[ci].Code, data.Courses[ci].Name))
}
