package assignments

import (
	"VIO/internal/dates"
	"VIO/internal/model"
	"VIO/internal/store"
	"VIO/internal/theme"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// AssignmentsPage lists assignments in three groups: ones that need
// attention (missing or past due), upcoming ones, and (toggled with s)
// submitted ones. Canvas assignments are read-only; assignments added
// here with + are manual and can be edited and deleted.
func AssignmentsPage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive {

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
 ▗▄▖  ▗▄▄▖ ▗▄▄▖▗▄▄▄▖ ▗▄▄▖▗▖  ▗▖▗▖  ▗▖▗▄▄▄▖▗▖  ▗▖▗▄▄▄▖▗▄▄▖
▐▌ ▐▌▐▌   ▐▌     █  ▐▌   ▐▛▚▖▐▌▐▛▚▞▜▌▐▌   ▐▛▚▖▐▌  █ ▐▌
▐▛▀▜▌ ▝▀▚▖ ▝▀▚▖  █  ▐▌▝▜▌▐▌ ▝▜▌▐▌  ▐▌▐▛▀▀▘▐▌ ▝▜▌  █  ▝▀▚▖
▐▌ ▐▌▗▄▄▞▘▗▄▄▞▘▗▄█▄▖▝▚▄▞▘▐▌  ▐▌▐▌  ▐▌▐▙▄▄▖▐▌  ▐▌  █ ▗▄▄▞▘
`)

	header := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(quit, 0, 3, false).
		AddItem(title, 0, 6, false)

	// Main body: one list, with group headings as unselectable rows

	list := tview.NewList().
		SetHighlightFullLine(true).
		SetSelectedStyle(tcell.StyleDefault.Background(theme.ColSurface).Foreground(theme.ColFg)).
		SetMainTextStyle(tcell.StyleDefault.Foreground(theme.ColFg)).
		SetSecondaryTextStyle(tcell.StyleDefault.Foreground(theme.ColMuted)).
		SetWrapAround(false)
	theme.FramePanel(list.Box, "Assignments", true)

	mainBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(list, 0, 6, true).
		AddItem(tview.NewBox(), 0, 1, false)

	// Footer

	help := func(showDone bool) string {
		toggle := "show submitted"
		if showDone {
			toggle = "hide submitted"
		}
		return theme.KeyHints("enter", "open", "+", "add", "e", "edit", "-", "remove", "s", toggle, "j/k", "move") +
			"\n" + theme.TagMuted + "Canvas assignments are read-only; + adds your own." + theme.TagReset
	}
	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 0, 2, false).
		AddItem(mainBody, 0, 5, true).
		AddItem(footer, 0, 1, false)

	// State

	showDone := false
	var rowIDs []string // assignment ID per list row; "" for headings
	lastRow := 0

	setStatus := func(msg string) {
		text := help(showDone)
		if msg != "" {
			text += "\n" + msg
		}
		footer.SetText(text)
	}

	render := func(selectID string) {
		rowIDs = fillAssignments(list, data, showDone)
		row := firstSelectable(rowIDs, 0, 1)
		if selectID != "" {
			if i := slices.Index(rowIDs, selectID); i >= 0 {
				row = i
			}
		}
		lastRow = row
		list.SetCurrentItem(row)
		setStatus("")
	}

	// Skip heading rows, moving in the direction the cursor went
	list.SetChangedFunc(func(i int, _, _ string, _ rune) {
		if i >= 0 && i < len(rowIDs) && rowIDs[i] == "" {
			dir := 1
			if i < lastRow {
				dir = -1
			}
			next := firstSelectable(rowIDs, i, dir)
			if next == i || rowIDs[next] == "" {
				next = lastRow
			}
			lastRow = next
			list.SetCurrentItem(next)
			return
		}
		lastRow = i
	})

	selected := func() int {
		row := list.GetCurrentItem()
		if row < 0 || row >= len(rowIDs) || rowIDs[row] == "" {
			return -1
		}
		return slices.IndexFunc(data.Assignments, func(a model.Assignment) bool { return a.ID == rowIDs[row] })
	}

	showPage := func() { app.SetRoot(page, true).SetFocus(list) }

	persist := func() bool {
		if err := store.SaveAppData(*data); err != nil {
			setStatus(theme.TagRed + "Saving failed: " + tview.Escape(err.Error()) + theme.TagReset)
			return false
		}
		return true
	}

	openForm := func(index int) {
		a := model.Assignment{Manual: true}
		formTitle := "New assignment"
		if index >= 0 {
			a = data.Assignments[index]
			formTitle = "Edit assignment"
		}
		if len(data.Courses) == 0 {
			setStatus(theme.TagYellow + "Add a course first: assignments belong to a course." + theme.TagReset)
			return
		}
		form := AssignmentForm(data, a, formTitle, func(saved model.Assignment) {
			if index >= 0 {
				data.Assignments[index] = saved
			} else {
				data.Assignments = append(data.Assignments, saved)
			}
			persist()
			render(saved.ID)
			showPage()
		}, showPage)
		app.SetRoot(form, true).SetFocus(form)
	}

	readOnly := func(index int) bool {
		if data.Assignments[index].Manual {
			return false
		}
		setStatus(theme.TagYellow + "\"" + tview.Escape(data.Assignments[index].Name) +
			"\" comes from Canvas, so it can't be changed here." + theme.TagReset)
		return true
	}

	confirmDelete := func(index int) {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Delete assignment \"%s\"?", data.Assignments[index].Name)).
			AddButtons([]string{"Delete", "Cancel"}).
			SetDoneFunc(func(_ int, label string) {
				if label == "Delete" {
					data.Assignments = slices.Delete(data.Assignments, index, index+1)
					persist()
					render("")
				}
				showPage()
			})
		app.SetRoot(modal, true).SetFocus(modal)
	}

	openDetail := func(index int) {
		id := data.Assignments[index].ID
		var detailPage tview.Primitive
		detailPage = AssignmentDetailPage(app, data, id, func(action string) {
			i := slices.IndexFunc(data.Assignments, func(a model.Assignment) bool { return a.ID == id })
			switch {
			case action == "edit" && i >= 0:
				form := AssignmentForm(data, data.Assignments[i], "Edit assignment", func(saved model.Assignment) {
					data.Assignments[i] = saved
					persist()
					render(saved.ID)
					showPage()
				}, func() { app.SetRoot(detailPage, true).SetFocus(detailPage) })
				app.SetRoot(form, true).SetFocus(form)
			case action == "delete" && i >= 0:
				confirmDelete(i)
			default:
				render(id)
				showPage()
			}
		})
		app.SetRoot(detailPage, true).SetFocus(detailPage)
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			returnTo()
			return nil
		case tcell.KeyEnter:
			if i := selected(); i >= 0 {
				openDetail(i)
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case '+', '=', 'a', 'A':
				openForm(-1)
				return nil
			case 'e', 'E':
				if i := selected(); i >= 0 && !readOnly(i) {
					openForm(i)
				}
				return nil
			case '-', 'd', 'D':
				if i := selected(); i >= 0 && !readOnly(i) {
					confirmDelete(i)
				}
				return nil
			case 's', 'S':
				showDone = !showDone
				id := ""
				if i := selected(); i >= 0 {
					id = data.Assignments[i].ID
				}
				render(id)
				return nil
			}
		}
		return event
	})

	render("")
	return page
}

// firstSelectable finds the first non-heading row from start going in dir,
// or start if there is none.
func firstSelectable(rowIDs []string, start, dir int) int {
	for i := start; i >= 0 && i < len(rowIDs); i += dir {
		if rowIDs[i] != "" {
			return i
		}
	}
	return start
}

// fillAssignments rebuilds the list and returns the assignment ID of each
// row ("" for heading and placeholder rows).
func fillAssignments(list *tview.List, data *model.AppData, showDone bool) []string {
	now := time.Now()
	var attention, upcoming, done []int
	for i, a := range data.Assignments {
		switch {
		case a.Done():
			done = append(done, i)
		case a.NeedsAttention(now):
			attention = append(attention, i)
		default: // includes undated assignments, which byDue sorts last
			upcoming = append(upcoming, i)
		}
	}
	byDue := func(idx []int, newestFirst bool) {
		sort.SliceStable(idx, func(x, y int) bool {
			a, b := data.Assignments[idx[x]].DueAt, data.Assignments[idx[y]].DueAt
			if a == nil || b == nil {
				return b == nil && a != nil
			}
			if newestFirst {
				return a.After(*b)
			}
			return a.Before(*b)
		})
	}
	byDue(attention, false)
	byDue(upcoming, false)
	byDue(done, true)

	list.Clear()
	var ids []string
	heading := func(label, tag string, n int) {
		list.AddItem(fmt.Sprintf("%s%s%s %s(%d)%s", tag, theme.TagBold, label, theme.TagMuted, n, theme.TagReset), "", 0, nil)
		ids = append(ids, "")
	}
	add := func(i int) {
		main, secondary := assignmentRow(data, data.Assignments[i], now)
		list.AddItem(main, secondary, 0, nil)
		ids = append(ids, data.Assignments[i].ID)
	}

	if len(attention) > 0 {
		heading("NEEDS ATTENTION", theme.TagRed, len(attention))
		for _, i := range attention {
			add(i)
		}
	}
	heading("UPCOMING", theme.TagAccent, len(upcoming))
	for _, i := range upcoming {
		add(i)
	}
	if len(upcoming) == 0 {
		list.AddItem(theme.TagMuted+"  Nothing upcoming."+theme.TagReset, "", 0, nil)
		ids = append(ids, "")
	}
	if showDone {
		heading("SUBMITTED", theme.TagGreen, len(done))
		for _, i := range done {
			add(i)
		}
	} else if len(done) > 0 {
		list.AddItem(fmt.Sprintf("%s  %d submitted hidden · press s to show%s", theme.TagMuted, len(done), theme.TagReset), "", 0, nil)
		ids = append(ids, "")
	}
	return ids
}

// assignmentRow renders the two list lines for one assignment.
func assignmentRow(data *model.AppData, a model.Assignment, now time.Time) (string, string) {
	ci := data.CourseIndexByID(a.CourseID)
	course := a.CourseID
	color := theme.TagMuted
	if ci >= 0 {
		course = model.ShortCode(data.Courses[ci].Code, data.Courses[ci].Name)
		color = theme.ColorTag(theme.CourseColor(ci))
	}

	name := tview.Escape(a.Name)
	if a.Done() {
		name = theme.TagMuted + name + theme.TagReset
	}
	main := "  " + color + "▌ " + theme.TagReset + name
	if badge := StateBadge(a.State(now)); badge != "" {
		main += "  " + badge
	}
	if a.Manual {
		main += theme.TagMuted + "  · manual" + theme.TagReset
	}

	parts := []string{tview.Escape(course)}
	if a.DueAt != nil {
		due := a.DueAt.In(time.Local)
		parts = append(parts, "due "+due.Format("Mon Jan 2, 3:04 PM")+" ("+dates.Relative(due, now)+")")
	} else {
		parts = append(parts, "no due date yet")
	}
	if a.PointsPossible > 0 {
		parts = append(parts, fmt.Sprintf("%.0f pts", a.PointsPossible))
	}
	return main, "      " + strings.Join(parts, " · ")
}

// StateBadge renders a colored label for an assignment state.
func StateBadge(state string) string {
	switch state {
	case model.AssignmentMissing:
		return theme.TagRed + theme.TagBold + "MISSING" + theme.TagReset
	case model.AssignmentOverdue:
		return theme.TagRed + "past due" + theme.TagReset
	case model.AssignmentLate:
		return theme.TagYellow + "late" + theme.TagReset
	case model.AssignmentSubmitted:
		return theme.TagGreen + "✓ submitted" + theme.TagReset
	default:
		return ""
	}
}
