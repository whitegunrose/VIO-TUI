package tasks

import (
	"VIO/internal/dates"
	"VIO/internal/model"
	"VIO/internal/theme"
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"
)

// TaskForm builds the add/edit form. onSave receives the updated task;
// onCancel runs on Esc or Cancel. Other pages (like the day schedule)
// use it to add and edit tasks too.
func TaskForm(data *model.AppData, task model.Task, title string, onSave func(model.Task), onCancel func()) tview.Primitive {
	courseIDs := []string{""}
	courseLabels := []string{"None"}
	courseIndex := 0
	for _, course := range data.Courses {
		if course.ID == task.CourseID {
			courseIndex = len(courseIDs)
		}
		courseIDs = append(courseIDs, course.ID)
		courseLabels = append(courseLabels, fmt.Sprintf("%s - %s", course.Code, course.Name))
	}

	dueText := ""
	if task.DueAt != nil {
		dueText = task.DueAt.In(time.Local).Format(dates.Layout)
	}

	statusValues := []string{"in_progress", "complete"}
	statusIndex := 0
	if task.Status == "complete" {
		statusIndex = 1
	}

	titleField := tview.NewInputField().SetLabel("Title").SetText(task.Title).SetFieldWidth(40)
	courseField := tview.NewDropDown().SetLabel("Course").SetOptions(courseLabels, nil).SetCurrentOption(courseIndex)
	dueField := tview.NewInputField().SetLabel("Due").SetText(dueText).SetFieldWidth(30).
		SetPlaceholder("fri 5pm, tomorrow, +3d")
	statusField := tview.NewDropDown().SetLabel("Status").SetOptions([]string{"In progress", "Complete"}, nil).SetCurrentOption(statusIndex)
	descriptionField := tview.NewTextArea().SetLabel("Description").SetText(task.Description, false).SetSize(4, 40)

	errorText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Live preview of how the due text reads
	preview := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	updatePreview := func(text string) {
		preview.SetText(DuePreview(text, time.Now()))
	}
	dueField.SetChangedFunc(updatePreview)
	updatePreview(dueText)

	form := tview.NewForm().
		AddFormItem(titleField).
		AddFormItem(courseField).
		AddFormItem(dueField).
		AddFormItem(statusField).
		AddFormItem(descriptionField)

	form.AddButton("Save", func() {
		name := strings.TrimSpace(titleField.GetText())
		if name == "" {
			errorText.SetText(theme.TagRed + "Title is required." + theme.TagReset)
			return
		}

		due, err := dates.Parse(dueField.GetText(), time.Now())
		if err != nil {
			errorText.SetText(theme.TagRed + tview.Escape(err.Error()) + theme.TagReset)
			return
		}

		course, _ := courseField.GetCurrentOption()
		status, _ := statusField.GetCurrentOption()

		// Let the store recompute the start time when the due date moves
		if task.DueAt == nil || !task.DueAt.Equal(due) {
			task.StartsAt = nil
		}
		if task.ID == "" {
			task.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
		}

		task.Title = name
		task.CourseID = courseIDs[course]
		task.DueAt = &due
		task.Status = statusValues[status]
		task.Description = strings.TrimSpace(descriptionField.GetText())

		onSave(task)
	})
	form.AddButton("Cancel", onCancel)
	form.SetCancelFunc(onCancel)

	theme.FramePanel(form.Box, strings.Trim(title, "[] "), true)

	return CenterForm(form, 18, preview, errorText)
}

// DuePreview renders the "reads as" line under a due-date field.
func DuePreview(text string, now time.Time) string {
	if strings.TrimSpace(text) == "" {
		return theme.TagMuted + "e.g. " + dates.Hint + theme.TagReset
	}
	t, err := dates.Parse(text, now)
	if err != nil {
		return theme.TagYellow + "Can't read that yet. Try " + dates.Hint + theme.TagReset
	}
	return theme.TagMuted + "reads as " + theme.TagReset + theme.TagGreen + dates.Describe(t, now) + theme.TagReset
}

// CenterForm centers a form of the given height on screen with the
// preview and error lines and a key hint below it.
func CenterForm(form *tview.Form, height int, preview, errorText *tview.TextView) tview.Primitive {
	hints := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(theme.KeyHints("tab", "next field", "esc", "cancel"))

	body := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(form, height, 0, true).
		AddItem(preview, 1, 0, false).
		AddItem(errorText, 1, 0, false).
		AddItem(hints, 1, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	return tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(body, 80, 0, true).
		AddItem(tview.NewBox(), 0, 1, false)
}
