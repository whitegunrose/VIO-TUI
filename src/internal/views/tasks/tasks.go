package tasks

import (
	"VIO/internal/model"
	"VIO/internal/store"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TasksPage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive {

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
		SetDynamicColors(true).
		SetText(`
▗▄▄▄▖▗▄▖  ▗▄▄▖▗▖ ▗▖ ▗▄▄▖
  █ ▐▌ ▐▌▐▌   ▐▌▗▞▘▐▌   
  █ ▐▛▀▜▌ ▝▀▚▖▐▛▚▖  ▝▀▚▖
  █ ▐▌ ▐▌▗▄▄▞▘▐▌ ▐▌▗▄▄▞▘
		`)

	header := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(quit, 0, 4, false). // narrow weight
		AddItem(title, 0, 6, false) // wider weight

	// IN PROGRESS, OVERDUE, and COMPLETE task columns

	statuses := []string{"in_progress", "overdue", "complete"}
	columns := []*tview.List{
		newTaskList("[ IN PROGRESS ]"),
		newTaskList("[ OVERDUE ]"),
		newTaskList("[ COMPLETE ]"),
	}
	// Task IDs shown in each column, in list order
	columnIDs := make([][]string, len(columns))
	focused := 0

	// Main body with paddings

	mainBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(columns[0], 0, 8, true).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(columns[1], 0, 8, false).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(columns[2], 0, 7, false).
		AddItem(tview.NewBox(), 0, 1, false)

	// Footer below all 3 task columns
	const footerHelp = "[ < > ] SWITCH COLUMN   [ ^ v ] SELECT   [ ENTER ] OPEN   [ + ] ADD   [ - ] REMOVE   [ E ] EDIT   [ X ] DONE"
	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(footerHelp)

	setStatus := func(msg string) {
		footer.SetText(footerHelp + "\n\n" + msg)
	}

	// Bringing header, main body, and footer together with paddings
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 0, 2, false).
		AddItem(mainBody, 0, 5, true).
		AddItem(footer, 0, 1, false)

	render := func() {
		for i, list := range columns {
			columnIDs[i] = fillTaskColumn(list, data, statuses[i])
		}
	}

	highlightColumn := func(index int) {
		focused = index
		for i, list := range columns {
			color := tcell.ColorWhite
			if i == index {
				color = tcell.ColorSpringGreen
			}
			list.SetBorderColor(color)
			list.SetTitleColor(color)
		}
	}

	focusColumn := func(index int) {
		highlightColumn(index)
		app.SetFocus(columns[index])
	}

	showPage := func() {
		app.SetRoot(page, true)
		focusColumn(focused)
	}

	// selectedTask returns the index in data.Tasks of the highlighted task, or -1
	selectedTask := func() int {
		ids := columnIDs[focused]
		current := columns[focused].GetCurrentItem()
		if current < 0 || current >= len(ids) {
			return -1
		}
		return slices.IndexFunc(data.Tasks, func(t model.Task) bool {
			return t.ID == ids[current]
		})
	}

	persist := func() {
		if err := store.SaveAppData(*data); err != nil {
			setStatus(fmt.Sprintf("[red]Saving tasks failed:[-] %v", err))
			return
		}
		setStatus("")
	}

	// Jump to wherever a task landed after it was saved
	selectByID := func(id string) {
		for i, ids := range columnIDs {
			if j := slices.Index(ids, id); j >= 0 {
				focused = i
				columns[i].SetCurrentItem(j)
				return
			}
		}
	}

	openForm := func(index int) {
		task := model.Task{Status: "in_progress"}
		formTitle := "[ NEW TASK ]"
		if index >= 0 {
			task = data.Tasks[index]
			formTitle = "[ EDIT TASK ]"
		}

		form := TaskForm(data, task, formTitle, func(saved model.Task) {
			if index >= 0 {
				data.Tasks[index] = saved
			} else {
				data.Tasks = append(data.Tasks, saved)
			}
			persist()
			render()
			selectByID(saved.ID)
			showPage()
		}, showPage)

		app.SetRoot(form, true).SetFocus(form)
	}

	confirmDelete := func(index int) {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Delete task \"%s\"?", data.Tasks[index].Title)).
			AddButtons([]string{"Delete", "Cancel"}).
			SetDoneFunc(func(_ int, label string) {
				if label == "Delete" {
					data.Tasks = slices.Delete(data.Tasks, index, index+1)
					persist()
					render()
				}
				showPage()
			})

		app.SetRoot(modal, true).SetFocus(modal)
	}

	render()

	page.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			returnTo()
			return nil
		case tcell.KeyLeft:
			focusColumn((focused + len(columns) - 1) % len(columns))
			return nil
		case tcell.KeyRight:
			focusColumn((focused + 1) % len(columns))
			return nil
		case tcell.KeyEnter:
			if index := selectedTask(); index >= 0 {
				id := data.Tasks[index].ID
				detailPage := TaskDetailPage(app, data, id, func() {
					render()
					selectByID(id)
					showPage()
				})
				app.SetRoot(detailPage, true).SetFocus(detailPage)
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case '+', '=', 'a', 'A':
				openForm(-1)
				return nil
			case '-', 'd', 'D':
				if index := selectedTask(); index >= 0 {
					confirmDelete(index)
				}
				return nil
			case 'e', 'E':
				if index := selectedTask(); index >= 0 {
					openForm(index)
				}
				return nil
			case 'x', 'X', ' ':
				if index := selectedTask(); index >= 0 {
					id := data.Tasks[index].ID
					ToggleComplete(&data.Tasks[index])
					persist()
					render()
					selectByID(id)
					focusColumn(focused)
				}
				return nil
			}
		}
		return event
	})

	highlightColumn(focused)

	return page
}

func newTaskList(title string) *tview.List {
	list := tview.NewList().
		SetSelectedFocusOnly(true).
		SetWrapAround(false)
	list.SetBorder(true)
	list.SetTitle(title)
	return list
}

// fillTaskColumn rebuilds a column's list and returns the task IDs in display order.
func fillTaskColumn(list *tview.List, data *model.AppData, status string) []string {
	filtered := make([]model.Task, 0)
	for _, task := range data.Tasks {
		if effectiveTaskStatus(task) == status {
			filtered = append(filtered, task)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		left := filtered[i].DueAt
		right := filtered[j].DueAt
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return left.Before(*right)
	})

	courseCodes := make(map[string]string, len(data.Courses))
	for _, course := range data.Courses {
		courseCodes[course.ID] = course.Code
	}

	current := list.GetCurrentItem()
	list.Clear()

	ids := make([]string, 0, len(filtered))
	for _, task := range filtered {
		var details []string
		if task.CourseID != "" {
			code := courseCodes[task.CourseID]
			if code == "" {
				code = task.CourseID
			}
			details = append(details, code)
		}
		if task.DueAt != nil {
			details = append(details, "Due: "+task.DueAt.Format("Jan 2, 3:04 PM"))
		}

		list.AddItem(task.Title, "[gray]"+strings.Join(details, " · ")+"[-]", 0, nil)
		ids = append(ids, task.ID)
	}

	if len(ids) == 0 {
		list.AddItem("[gray]No tasks here yet.[-]", "", 0, nil)
	}

	list.SetCurrentItem(current)
	return ids
}

func effectiveTaskStatus(task model.Task) string {
	status := strings.ToLower(strings.TrimSpace(task.Status))
	if status == "complete" {
		return "complete"
	}
	if task.DueAt != nil && task.DueAt.Before(time.Now()) {
		return "overdue"
	}
	return "in_progress"
}
