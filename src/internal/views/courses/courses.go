package courses

import (
	"VIO/internal/model"
	"VIO/internal/store"
	"VIO/internal/theme"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func CoursesPage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive {

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

	// School and student info

	schoolTitle := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(`



 ▗▄▄▖ ▗▄▄▖▗▖ ▗▖▗▖  ▗▖
▐▌   ▐▌   ▐▌ ▐▌▐▛▚▖▐▌
▐▌    ▝▀▚▖▐▌ ▐▌▐▌ ▝▜▌
▝▚▄▄▖▗▄▄▞▘▝▚▄▞▘▐▌  ▐▌
                     


		`)

	studentInfo := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(fmt.Sprintf("\n%s\n%s\n", data.Student.Name, data.Student.School))

	schoolInfo := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(schoolTitle, 0, 1, false).
		AddItem(studentInfo, 0, 1, false)

	// Course list (left) and details or meeting-times editor (right)

	list := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedStyle(tcell.StyleDefault.Background(theme.ColSurface).Foreground(theme.ColFg)).
		SetMainTextStyle(tcell.StyleDefault.Foreground(theme.ColFg))
	theme.FramePanel(list.Box, "Courses", true)

	detail := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetScrollable(true)

	editor := tview.NewTextArea().
		SetPlaceholder("Tue Thu 10:00 AM - 11:15 AM @ JD 1234\nMW 2:30-3:45pm\nThursday 5:30 PM to 8:15 PM").
		SetPlaceholderStyle(tcell.StyleDefault.Foreground(theme.ColMuted))

	right := tview.NewPages().
		AddPage("detail", detail, true, true).
		AddPage("editor", editor, true, false)
	theme.FramePanel(right.Box, "Details", false)

	mainBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(schoolInfo, 0, 3, false).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(list, 0, 4, true).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(right, 0, 5, false).
		AddItem(tview.NewBox(), 0, 1, false)

	listHelp := theme.KeyHints("j/k", "select", "m", "edit meeting times", "g/G", "top/bottom", "esc", "back")
	editHelp := theme.KeyHints("esc", "save", "ctrl+x", "discard") +
		theme.TagMuted + "one meeting per line: days  start - end  @ room" + theme.TagReset

	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(listHelp)

	// Bringing header, main body, and footer together with paddings
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 0, 2, false).
		AddItem(mainBody, 0, 5, true).
		AddItem(footer, 0, 1, false)

	// State and behavior

	editing := -1 // index of the course whose meetings are open, -1 if none

	showDetail := func(i int) {
		detail.SetText(renderCourseDetail(data, i)).ScrollToBeginning()
	}

	for i, course := range data.Courses {
		list.AddItem(listLabel(i, course), "", 0, nil)
	}
	showDetail(0)
	list.SetChangedFunc(func(i int, _, _ string, _ rune) { showDetail(i) })

	startEdit := func() {
		i := list.GetCurrentItem()
		if i < 0 || i >= len(data.Courses) {
			return
		}
		editing = i
		course := data.Courses[i]
		editor.SetText(model.FormatMeetings(course.Meetings), true)
		right.SetTitle(" Meeting times · " + model.ShortCode(course.Code, course.Name) + " ")
		right.SwitchToPage("editor")
		theme.SetPaneFocus(right.Box, true)
		theme.SetPaneFocus(list.Box, false)
		app.SetFocus(editor)
		footer.SetText(editHelp)
	}

	stopEdit := func() {
		editing = -1
		right.SetTitle(" Details ")
		right.SwitchToPage("detail")
		theme.SetPaneFocus(right.Box, false)
		theme.SetPaneFocus(list.Box, true)
		app.SetFocus(list)
		footer.SetText(listHelp)
		showDetail(list.GetCurrentItem())
	}

	list.SetSelectedFunc(func(int, string, string, rune) { startEdit() })

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			returnTo()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case 'g':
				list.SetCurrentItem(0)
				return nil
			case 'G':
				list.SetCurrentItem(list.GetItemCount() - 1)
				return nil
			case 'm', 'M', 'e', 'E':
				startEdit()
				return nil
			}
		}
		return event
	})

	editor.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlX:
			stopEdit()
			return nil
		case tcell.KeyEsc:
			meetings, err := model.ParseMeetings(editor.GetText())
			if err != nil {
				footer.SetText(editHelp + "\n\n" + theme.TagRed + tview.Escape(err.Error()) + theme.TagReset)
				return nil
			}
			data.Courses[editing].Meetings = meetings
			if err := store.SaveAppData(*data); err != nil {
				footer.SetText(editHelp + "\n\n" + theme.TagRed + "Saving failed: " + tview.Escape(err.Error()) + theme.TagReset)
				return nil
			}
			stopEdit()
			return nil
		}
		return event
	})

	return page
}

func listLabel(i int, course model.Course) string {
	return theme.ColorTag(theme.CourseColor(i)) + "▌ " + theme.TagReset + tview.Escape(model.ShortCode(course.Code, course.Name))
}

func renderCourseDetail(data *model.AppData, i int) string {
	if i < 0 || i >= len(data.Courses) {
		return theme.TagMuted + "No courses loaded yet." + theme.TagReset
	}
	course := data.Courses[i]
	color := theme.ColorTag(theme.CourseColor(i))

	var b strings.Builder
	b.WriteString(color + theme.TagBold + tview.Escape(course.Name) + theme.TagReset + "\n")
	if course.Code != "" && !strings.EqualFold(course.Code, course.Name) {
		b.WriteString(theme.TagMuted + tview.Escape(course.Code) + theme.TagReset + "\n")
	}
	if course.Term != "" {
		b.WriteString(theme.TagMuted + tview.Escape(course.Term) + theme.TagReset + "\n")
	}

	b.WriteString("\n" + theme.TagAccent + "Meeting times" + theme.TagReset + "\n")
	if len(course.Meetings) == 0 {
		b.WriteString(theme.TagMuted + "None yet. Press " + theme.TagKey + "m" + theme.TagMuted +
			" to add your class times so they show on the calendar." + theme.TagReset + "\n")
	}
	for _, m := range course.Meetings {
		start, end, _ := m.Minutes()
		b.WriteString(color + "▌ " + theme.TagReset + strings.Join(m.Days, " ") + "  " + model.FormatRange(start, end, false))
		if m.Location != "" {
			b.WriteString(theme.TagMuted + "  @ " + tview.Escape(m.Location) + theme.TagReset)
		}
		b.WriteString("\n")
	}
	for _, legacy := range course.MeetingTimes {
		b.WriteString(theme.TagMuted + "Couldn't read: " + tview.Escape(legacy) + theme.TagReset + "\n")
	}

	if course.Location != "" {
		b.WriteString("\n" + theme.TagAccent + "Location" + theme.TagReset + "\n" + tview.Escape(course.Location) + "\n")
	}
	if course.Workflow != "" {
		b.WriteString("\n" + theme.TagMuted + "Canvas state: " + tview.Escape(course.Workflow) + theme.TagReset + "\n")
	}
	if strings.TrimSpace(course.Notes) != "" {
		b.WriteString("\n" + theme.TagAccent + "Notes" + theme.TagReset + "\n" + tview.Escape(course.Notes) + "\n")
	}
	return b.String()
}
