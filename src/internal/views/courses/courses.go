package courses

import (
	"VIO/internal/model"
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

	// Course info

	courseInfo := tview.NewTextView().
		SetDynamicColors(true)
	courseInfo.SetBorder(true)
	courseInfo.SetTitle("[ COURSES ]")

	courseInfo.SetText(renderCourses(data))

	// Main body

	mainBody := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(schoolInfo, 0, 3, false).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(courseInfo, 0, 7, true). // focused so it scrolls with j/k and arrows
		AddItem(tview.NewBox(), 0, 1, false)

	// Footer below course and school info
	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[ ^ v ] or [ J K ] SCROLL    [ G ] TOP    [ SHIFT+G ] BOTTOM")

	// Bringing header, main body, and footer together with paddings
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 0, 2, false).
		AddItem(mainBody, 0, 5, true).
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

func renderCourses(data *model.AppData) string {
	if len(data.Courses) == 0 {
		return "No courses loaded yet."
	}

	var lines []string
	for _, course := range data.Courses {
		line := fmt.Sprintf("[white::b]%s[::-]\n%s", course.Code, course.Name)

		if course.Term != "" {
			line += fmt.Sprintf("\n[gray]%s[-]", course.Term)
		}

		if course.Workflow != "" {
			line += fmt.Sprintf("\n[gray]state: %s[-]", course.Workflow)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n\n")
}
