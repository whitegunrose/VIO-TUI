package notes

import (
	"VIO/internal/model"
	"VIO/internal/store"
	"VIO/internal/theme"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NotesPage lists notes on the left and shows the selected one on the right.
// Enter edits in place, Esc saves and closes the editor (or returns to main).
func NotesPage(app *tview.Application, data *model.AppData, returnTo func()) tview.Primitive {

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
▗▖  ▗▖ ▗▄▖ ▗▄▄▄▖▗▄▄▄▖ ▗▄▄▖
▐▛▚▖▐▌▐▌ ▐▌  █  ▐▌   ▐▌   
▐▌ ▝▜▌▐▌ ▐▌  █  ▐▛▀▀▘ ▝▀▚▖
▐▌  ▐▌▝▚▄▞▘  █  ▐▙▄▄▖▗▄▄▞▘
		`)

	header := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(quit, 0, 4, false).
		AddItem(title, 0, 6, false)

	// Note list (left), preview or editor (right)

	list := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedStyle(tcell.StyleDefault.Background(theme.ColSurface).Foreground(theme.ColFg)).
		SetMainTextStyle(tcell.StyleDefault.Foreground(theme.ColFg))
	list.SetBorder(true).SetTitle(" Notes ").SetTitleAlign(tview.AlignLeft).SetBorderPadding(0, 0, 1, 1)

	preview := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetScrollable(true)

	editor := tview.NewTextArea().
		SetPlaceholder("# Title on the first line\n\nStart typing…").
		SetPlaceholderStyle(tcell.StyleDefault.Foreground(theme.ColMuted))

	right := tview.NewPages().
		AddPage("preview", preview, true, true).
		AddPage("editor", editor, true, false)
	right.SetBorder(true).SetTitleAlign(tview.AlignLeft).SetBorderPadding(0, 0, 1, 1)

	mainBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(list, 0, 8, true).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(right, 0, 16, false).
		AddItem(tview.NewBox(), 0, 1, false)

	listHelp := theme.KeyHints("enter", "edit", "n", "new", "d", "delete", "j/k", "move", "esc", "back")
	editHelp := theme.KeyHints("esc", "save and close") + theme.TagMuted + "the first line becomes the title"
	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(listHelp)

	setStatus := func(msg string) {
		help := listHelp
		if editor.HasFocus() {
			help = editHelp
		}
		if msg == "" {
			footer.SetText(help)
			return
		}
		footer.SetText(help + "\n\n" + msg)
	}

	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 0, 2, false).
		AddItem(mainBody, 0, 5, true).
		AddItem(footer, 0, 1, false)

	// Focus colors: whichever side has the keyboard gets the accent frame
	setFocusSide := func(editing bool) {
		active, inactive := list.Box, right.Box
		if editing {
			active, inactive = right.Box, list.Box
		}
		active.SetBorderColor(theme.ColAccent).SetTitleColor(theme.ColAccent)
		inactive.SetBorderColor(theme.ColBorder).SetTitleColor(theme.ColMuted)
	}

	// State

	var editingID string // ID of the note open in the editor, "" if none
	pendingDelete := -1  // list index waiting for a second d press

	persist := func() {
		if err := store.SaveAppData(*data); err != nil {
			setStatus(fmt.Sprintf("%sSaving notes failed:%s %v", theme.TagRed, theme.TagReset, err))
			return
		}
		setStatus("")
	}

	indexOf := func(id string) int {
		for i, n := range data.Notes {
			if n.ID == id {
				return i
			}
		}
		return -1
	}

	showPreview := func(i int) {
		pendingDelete = -1
		if i < 0 || i >= len(data.Notes) {
			right.SetTitle(" Preview ")
			preview.SetText(theme.TagMuted + "No notes yet. Press " + theme.TagKey + "n" + theme.TagMuted + " to write one.")
			return
		}
		right.SetTitle(" Preview ")
		preview.SetText(renderNote(data, &data.Notes[i])).ScrollToBeginning()
	}

	render := func(selectID string) {
		sort.SliceStable(data.Notes, func(i, j int) bool {
			return data.Notes[i].UpdatedAt.After(data.Notes[j].UpdatedAt)
		})
		list.Clear()
		for _, n := range data.Notes {
			list.AddItem(tview.Escape(n.Title), "", 0, nil)
		}
		sel := 0
		if idx := indexOf(selectID); idx >= 0 {
			sel = idx
		}
		if len(data.Notes) > 0 {
			list.SetCurrentItem(sel)
		}
		showPreview(sel)
	}

	startEdit := func(i int) {
		if i < 0 || i >= len(data.Notes) {
			return
		}
		editingID = data.Notes[i].ID
		editor.SetText(data.Notes[i].Body, false)
		right.SetTitle(" Editing ")
		right.SwitchToPage("editor")
		setFocusSide(true)
		app.SetFocus(editor)
		setStatus("")
	}

	finishEdit := func() {
		i := indexOf(editingID)
		editingID = ""
		body := editor.GetText()
		keepID := ""

		if i >= 0 {
			if strings.TrimSpace(body) == "" {
				// Closing an empty note discards it
				data.Notes = append(data.Notes[:i], data.Notes[i+1:]...)
			} else {
				n := &data.Notes[i]
				keepID = n.ID
				if body != n.Body {
					n.Body = body
					n.Title = titleFrom(body)
					n.UpdatedAt = time.Now()
				}
			}
			persist()
		}

		right.SwitchToPage("preview")
		setFocusSide(false)
		app.SetFocus(list)
		render(keepID)
		setStatus("")
	}

	newNote := func() {
		n := model.Note{
			ID:        fmt.Sprintf("note-%d", time.Now().UnixNano()),
			Title:     "Untitled",
			UpdatedAt: time.Now(),
		}
		data.Notes = append(data.Notes, n)
		render(n.ID)
		startEdit(indexOf(n.ID))
	}

	// Input

	list.SetChangedFunc(func(i int, _, _ string, _ rune) { showPreview(i) })
	list.SetSelectedFunc(func(i int, _, _ string, _ rune) { startEdit(i) })

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			returnTo()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'n', 'N', '+':
				newNote()
				return nil
			case 'd', 'D', '-':
				i := list.GetCurrentItem()
				if i < 0 || i >= len(data.Notes) {
					return nil
				}
				if pendingDelete != i {
					pendingDelete = i
					setStatus(theme.TagRed + "Press d again to delete " + theme.TagReset + tview.Escape(data.Notes[i].Title))
					return nil
				}
				data.Notes = append(data.Notes[:i], data.Notes[i+1:]...)
				persist()
				render("")
				return nil
			case 'e', 'E':
				startEdit(list.GetCurrentItem())
				return nil
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			}
		}
		return event
	})

	editor.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			finishEdit()
			return nil
		}
		return event
	})

	render("")
	setFocusSide(false)
	return page
}

func titleFrom(body string) string {
	first := strings.SplitN(strings.TrimSpace(body), "\n", 2)[0]
	first = strings.TrimSpace(strings.TrimLeft(first, "# "))
	if first == "" {
		return "Untitled"
	}
	if r := []rune(first); len(r) > 40 {
		return string(r[:39]) + "…"
	}
	return first
}

// renderNote is a small Markdown-ish renderer: # headings in accent,
// - and * list items as bullets, everything else as written.
func renderNote(data *model.AppData, n *model.Note) string {
	var b strings.Builder

	meta := "edited " + n.UpdatedAt.In(time.Local).Format("Mon Jan 2, 3:04 PM")
	if n.CourseID != "" {
		label := n.CourseID
		for _, c := range data.Courses {
			if c.ID == n.CourseID {
				label = c.Code
				break
			}
		}
		meta = theme.TagKey + tview.Escape(label) + theme.TagMuted + " · " + meta
	}
	b.WriteString(theme.TagMuted + meta + theme.TagReset + "\n\n")

	for _, line := range strings.Split(n.Body, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "#"):
			b.WriteString(theme.TagAccent + theme.TagBold + tview.Escape(strings.TrimLeft(t, "# ")) + theme.TagReset)
		case strings.HasPrefix(t, "- "), strings.HasPrefix(t, "* "):
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			b.WriteString(indent + theme.TagAccent + "•" + theme.TagReset + " " + tview.Escape(t[2:]))
		default:
			b.WriteString(tview.Escape(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}
