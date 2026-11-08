package tasks

import (
	"VIO/internal/model"
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"
)

const dueLayout = "2006-01-02 15:04"

// taskForm builds the add/edit form. onSave receives the updated task; onCancel runs on Esc or Cancel.
func taskForm(data *model.AppData, task model.Task, title string, onSave func(model.Task), onCancel func()) tview.Primitive {
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
		dueText = task.DueAt.In(time.Local).Format(dueLayout)
	}

	statusValues := []string{"in_progress", "complete"}
	statusIndex := 0
	if task.Status == "complete" {
		statusIndex = 1
	}

	titleField := tview.NewInputField().SetLabel("Title").SetText(task.Title).SetFieldWidth(40)
	courseField := tview.NewDropDown().SetLabel("Course").SetOptions(courseLabels, nil).SetCurrentOption(courseIndex)
	dueField := tview.NewInputField().SetLabel("Due (YYYY-MM-DD HH:MM)").SetText(dueText).SetFieldWidth(20)
	statusField := tview.NewDropDown().SetLabel("Status").SetOptions([]string{"In progress", "Complete"}, nil).SetCurrentOption(statusIndex)
	descriptionField := tview.NewTextArea().SetLabel("Description").SetText(task.Description, false).SetSize(4, 40)

	errorText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	form := tview.NewForm().
		AddFormItem(titleField).
		AddFormItem(courseField).
		AddFormItem(dueField).
		AddFormItem(statusField).
		AddFormItem(descriptionField)

	form.AddButton("Save", func() {
		name := strings.TrimSpace(titleField.GetText())
		if name == "" {
			errorText.SetText("[red]Title is required.[-]")
			return
		}

		due, err := parseDue(dueField.GetText())
		if err != nil {
			errorText.SetText("[red]" + err.Error() + "[-]")
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

	form.SetBorder(true)
	form.SetTitle(title)

	body := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(form, 18, 0, true).
		AddItem(errorText, 1, 0, false).
		AddItem(tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText("[ TAB ] NEXT FIELD   [ ESC ] CANCEL"), 1, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	return tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(body, 80, 0, true).
		AddItem(tview.NewBox(), 0, 1, false)
}

// parseDue accepts "YYYY-MM-DD HH:MM", or just "YYYY-MM-DD" for end of day.
func parseDue(text string) (time.Time, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return time.Time{}, fmt.Errorf("Due date is required.")
	}
	if due, err := time.ParseInLocation(dueLayout, text, time.Local); err == nil {
		return due, nil
	}
	if day, err := time.ParseInLocation("2006-01-02", text, time.Local); err == nil {
		return day.Add(23*time.Hour + 59*time.Minute), nil
	}
	return time.Time{}, fmt.Errorf("Due date must look like 2026-11-08 17:00 or 2026-11-08.")
}
