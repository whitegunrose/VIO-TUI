package tasks

import (
	"VIO/internal/dates"
	"VIO/internal/model"
	"VIO/internal/store"
	"VIO/internal/theme"
	"VIO/internal/views/detail"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/rivo/tview"
)

// TaskDetailPage shows one task. From here: e edits it, x toggles
// complete, - deletes it. returnTo runs on Esc and after a delete.
func TaskDetailPage(app *tview.Application, data *model.AppData, taskID string, returnTo func()) tview.Primitive {
	var page tview.Primitive
	var body, footer *tview.TextView

	index := func() int {
		return slices.IndexFunc(data.Tasks, func(t model.Task) bool { return t.ID == taskID })
	}

	hints := theme.KeyHints("e", "edit", "x", "toggle complete", "-", "delete", "j/k", "scroll")
	show := func() { app.SetRoot(page, true).SetFocus(body) }
	refresh := func() {
		if i := index(); i >= 0 {
			body.SetText(renderTaskDetail(data, data.Tasks[i]))
		}
	}
	save := func() bool {
		if err := store.SaveAppData(*data); err != nil {
			footer.SetText(hints + "\n" + theme.TagRed + "Saving failed: " + tview.Escape(err.Error()) + theme.TagReset)
			return false
		}
		footer.SetText(hints)
		return true
	}

	onKey := func(r rune) bool {
		i := index()
		if i < 0 {
			returnTo()
			return true
		}
		switch r {
		case 'e', 'E':
			form := TaskForm(data, data.Tasks[i], "[ EDIT TASK ]", func(saved model.Task) {
				data.Tasks[i] = saved
				save()
				refresh()
				show()
			}, show)
			app.SetRoot(form, true).SetFocus(form)
			return true
		case 'x', 'X', ' ':
			ToggleComplete(&data.Tasks[i])
			save()
			refresh()
			return true
		case '-', 'd', 'D':
			modal := tview.NewModal().
				SetText(fmt.Sprintf("Delete task \"%s\"?", data.Tasks[i].Title)).
				AddButtons([]string{"Delete", "Cancel"}).
				SetDoneFunc(func(_ int, label string) {
					if label == "Delete" {
						data.Tasks = slices.Delete(data.Tasks, i, i+1)
						save()
						returnTo()
						return
					}
					show()
				})
			app.SetRoot(modal, true).SetFocus(modal)
			return true
		}
		return false
	}

	i := index()
	text := theme.TagMuted + "This task no longer exists." + theme.TagReset
	if i >= 0 {
		text = renderTaskDetail(data, data.Tasks[i])
	}
	page, body, footer = detail.Page("Task", text, hints, onKey, returnTo)
	return page
}

// ToggleComplete flips a task between complete and not complete. Reopened
// tasks go back to in progress (the store marks them overdue if past due).
func ToggleComplete(t *model.Task) {
	if effectiveTaskStatus(*t) == "complete" {
		t.Status = "in_progress"
		if t.DueAt != nil && t.DueAt.Before(time.Now()) {
			t.Status = "overdue"
		}
		return
	}
	t.Status = "complete"
}

func renderTaskDetail(data *model.AppData, t model.Task) string {
	now := time.Now()
	var b strings.Builder

	b.WriteString(theme.TagBold + tview.Escape(t.Title) + theme.TagReset + "\n\n")

	status := effectiveTaskStatus(t)
	statusText := map[string]string{
		"complete":    theme.TagGreen + "✓ complete" + theme.TagReset,
		"overdue":     theme.TagRed + "● overdue" + theme.TagReset,
		"in_progress": theme.TagYellow + "● in progress" + theme.TagReset,
	}[status]
	b.WriteString(detail.Field("Status", statusText))

	if ci := data.CourseIndexByID(t.CourseID); ci >= 0 {
		c := data.Courses[ci]
		b.WriteString(detail.Field("Course", theme.ColorTag(theme.CourseColor(ci))+"▌ "+theme.TagReset+
			tview.Escape(model.ShortCode(c.Code, c.Name))+theme.TagMuted+"  "+tview.Escape(c.Name)+theme.TagReset))
	}
	if t.DueAt != nil {
		due := t.DueAt.In(time.Local)
		b.WriteString(detail.Field("Due", due.Format("Mon Jan 2 2006, 3:04 PM")+theme.TagMuted+"  ("+dates.Relative(due, now)+")"+theme.TagReset))
	}
	if t.StartsAt != nil {
		b.WriteString(detail.Field("Starts", t.StartsAt.In(time.Local).Format("Mon Jan 2, 3:04 PM")))
	}

	b.WriteString(detail.Section("Description"))
	if strings.TrimSpace(t.Description) == "" {
		b.WriteString(theme.TagMuted + "No description. Press " + theme.TagKey + "e" + theme.TagMuted + " to add one." + theme.TagReset + "\n")
	} else {
		b.WriteString(tview.Escape(t.Description) + "\n")
	}
	return b.String()
}
