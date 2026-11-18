package assignments

import (
	"VIO/internal/dates"
	"VIO/internal/model"
	"VIO/internal/theme"
	"VIO/internal/views/detail"
	"VIO/internal/views/tasks"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rivo/tview"
)

// AssignmentDetailPage shows one assignment. done is called with "edit" or
// "delete" when those keys are pressed on a manual assignment, or "" on Esc.
func AssignmentDetailPage(app *tview.Application, data *model.AppData, id string, done func(action string)) tview.Primitive {
	i := slices.IndexFunc(data.Assignments, func(a model.Assignment) bool { return a.ID == id })
	if i < 0 {
		page, _, _ := detail.Page("Assignment", theme.TagMuted+"This assignment no longer exists."+theme.TagReset, "", nil, func() { done("") })
		return page
	}
	a := data.Assignments[i]

	hints := theme.KeyHints("j/k", "scroll", "esc", "back")
	if a.Manual {
		hints = theme.KeyHints("e", "edit", "-", "delete", "j/k", "scroll", "esc", "back")
	}

	onKey := func(r rune) bool {
		if !a.Manual {
			return false
		}
		switch r {
		case 'e', 'E':
			done("edit")
			return true
		case '-', 'd', 'D':
			done("delete")
			return true
		}
		return false
	}

	page, _, _ := detail.Page("Assignment", renderAssignmentDetail(data, a), hints, onKey, func() { done("") })
	return page
}

func renderAssignmentDetail(data *model.AppData, a model.Assignment) string {
	now := time.Now()
	var b strings.Builder

	b.WriteString(theme.TagBold + tview.Escape(a.Name) + theme.TagReset)
	if badge := StateBadge(a.State(now)); badge != "" {
		b.WriteString("   " + badge)
	}
	b.WriteString("\n\n")

	if ci := data.CourseIndexByID(a.CourseID); ci >= 0 {
		c := data.Courses[ci]
		b.WriteString(detail.Field("Course", theme.ColorTag(theme.CourseColor(ci))+"▌ "+theme.TagReset+
			tview.Escape(model.ShortCode(c.Code, c.Name))+theme.TagMuted+"  "+tview.Escape(c.Name)+theme.TagReset))
	}
	if a.DueAt != nil {
		due := a.DueAt.In(time.Local)
		b.WriteString(detail.Field("Due", due.Format("Mon Jan 2 2006, 3:04 PM")+theme.TagMuted+"  ("+dates.Relative(due, now)+")"+theme.TagReset))
	}
	if a.UnlockAt != nil {
		b.WriteString(detail.Field("Opens", a.UnlockAt.In(time.Local).Format("Mon Jan 2, 3:04 PM")))
	}
	if a.LockAt != nil {
		b.WriteString(detail.Field("Closes", a.LockAt.In(time.Local).Format("Mon Jan 2, 3:04 PM")))
	}
	if a.PointsPossible > 0 {
		b.WriteString(detail.Field("Points", fmt.Sprintf("%g", a.PointsPossible)))
	}

	var flags []string
	flags = append(flags, yesNo("submitted", a.HasSubmitted))
	if a.IsMissing {
		flags = append(flags, theme.TagRed+"missing"+theme.TagReset)
	}
	if a.IsLate {
		flags = append(flags, theme.TagYellow+"late"+theme.TagReset)
	}
	b.WriteString(detail.Field("Canvas", strings.Join(flags, theme.TagMuted+" · "+theme.TagReset)))
	if a.Manual {
		b.WriteString(detail.Field("Source", "added in VIO (editable)"))
	} else {
		b.WriteString(detail.Field("Source", "Canvas"+theme.TagMuted+" (read-only; updates on sync)"+theme.TagReset))
	}

	b.WriteString(detail.Section("Description"))
	if text := detail.PlainText(a.Description); text != "" {
		b.WriteString(tview.Escape(text) + "\n")
	} else {
		b.WriteString(theme.TagMuted + "No description." + theme.TagReset + "\n")
	}
	return b.String()
}

func yesNo(label string, v bool) string {
	if v {
		return theme.TagGreen + "✓ " + label + theme.TagReset
	}
	return theme.TagMuted + "not " + label + theme.TagReset
}

// AssignmentForm adds or edits a manual assignment.
func AssignmentForm(data *model.AppData, a model.Assignment, title string, onSave func(model.Assignment), onCancel func()) tview.Primitive {
	courseIDs := make([]string, 0, len(data.Courses))
	courseLabels := make([]string, 0, len(data.Courses))
	courseIndex := 0
	for _, c := range data.Courses {
		if c.ID == a.CourseID {
			courseIndex = len(courseIDs)
		}
		courseIDs = append(courseIDs, c.ID)
		courseLabels = append(courseLabels, fmt.Sprintf("%s - %s", c.Code, c.Name))
	}

	dueText := ""
	if a.DueAt != nil {
		dueText = a.DueAt.In(time.Local).Format(dates.Layout)
	}
	pointsText := ""
	if a.PointsPossible > 0 {
		pointsText = strconv.FormatFloat(a.PointsPossible, 'f', -1, 64)
	}

	nameField := tview.NewInputField().SetLabel("Name").SetText(a.Name).SetFieldWidth(40)
	courseField := tview.NewDropDown().SetLabel("Course").SetOptions(courseLabels, nil).SetCurrentOption(courseIndex)
	dueField := tview.NewInputField().SetLabel("Due").SetText(dueText).SetFieldWidth(30).
		SetPlaceholder("fri 5pm, tomorrow, +3d")
	pointsField := tview.NewInputField().SetLabel("Points").SetText(pointsText).SetFieldWidth(8).
		SetAcceptanceFunc(tview.InputFieldFloat)
	submittedField := tview.NewCheckbox().SetLabel("Submitted").SetChecked(a.HasSubmitted)
	descriptionField := tview.NewTextArea().SetLabel("Description").SetText(a.Description, false).SetSize(4, 40)

	errorText := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetDynamicColors(true)
	preview := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetDynamicColors(true)
	dueField.SetChangedFunc(func(text string) { preview.SetText(tasks.DuePreview(text, time.Now())) })
	preview.SetText(tasks.DuePreview(dueText, time.Now()))

	form := tview.NewForm().
		AddFormItem(nameField).
		AddFormItem(courseField).
		AddFormItem(dueField).
		AddFormItem(pointsField).
		AddFormItem(submittedField).
		AddFormItem(descriptionField)

	form.AddButton("Save", func() {
		name := strings.TrimSpace(nameField.GetText())
		if name == "" {
			errorText.SetText(theme.TagRed + "Name is required." + theme.TagReset)
			return
		}
		due, err := dates.Parse(dueField.GetText(), time.Now())
		if err != nil {
			errorText.SetText(theme.TagRed + tview.Escape(err.Error()) + theme.TagReset)
			return
		}
		points := 0.0
		if p := strings.TrimSpace(pointsField.GetText()); p != "" {
			points, err = strconv.ParseFloat(p, 64)
			if err != nil || points < 0 {
				errorText.SetText(theme.TagRed + "Points must be a number." + theme.TagReset)
				return
			}
		}
		course, _ := courseField.GetCurrentOption()
		if course < 0 || course >= len(courseIDs) {
			errorText.SetText(theme.TagRed + "Pick a course." + theme.TagReset)
			return
		}

		if a.ID == "" {
			a.ID = fmt.Sprintf("manual-%d", time.Now().UnixNano())
		}
		a.Manual = true
		a.Name = name
		a.CourseID = courseIDs[course]
		a.DueAt = &due
		a.PointsPossible = points
		a.HasSubmitted = submittedField.IsChecked()
		a.Description = strings.TrimSpace(descriptionField.GetText())
		onSave(a)
	})
	form.AddButton("Cancel", onCancel)
	form.SetCancelFunc(onCancel)
	theme.FramePanel(form.Box, title, true)

	return tasks.CenterForm(form, 20, preview, errorText)
}
