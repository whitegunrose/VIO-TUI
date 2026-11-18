package theme

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Palette (Tokyo Night). Change these to re-theme the whole app.
// ColBg is the terminal's own background so VIO matches your terminal theme.
var (
	ColBg      = tcell.ColorDefault
	ColSurface = tcell.NewHexColor(0x283457) // selected rows
	ColBorder  = tcell.NewHexColor(0x3b4261)
	ColFg      = tcell.NewHexColor(0xc0caf5)
	ColMuted   = tcell.NewHexColor(0x565f89)
	ColAccent  = tcell.NewHexColor(0x7aa2f7) // focus, headings
	ColKey     = tcell.NewHexColor(0xbb9af7) // key hints, tags
	ColRed     = tcell.NewHexColor(0xf7768e)
	ColYellow  = tcell.NewHexColor(0xe0af68)
	ColDark    = tcell.NewHexColor(0x1a1b26) // text on accent backgrounds
)

// CourseColors gives each course its own color, in course order.
var CourseColors = []tcell.Color{
	tcell.NewHexColor(0x7aa2f7), // blue
	tcell.NewHexColor(0x9ece6a), // green
	tcell.NewHexColor(0xbb9af7), // purple
	tcell.NewHexColor(0xff9e64), // orange
	tcell.NewHexColor(0x7dcfff), // cyan
	tcell.NewHexColor(0xf7768e), // pink
	tcell.NewHexColor(0x73daca), // teal
	tcell.NewHexColor(0xe0af68), // yellow
}

// CourseColor returns the color for the course at index i in data.Courses.
func CourseColor(i int) tcell.Color {
	if i < 0 {
		i = -i
	}
	return CourseColors[i%len(CourseColors)]
}

// ColorTag turns a color into a tview tag like "[#7aa2f7]".
func ColorTag(c tcell.Color) string {
	return fmt.Sprintf("[#%06x]", c.Hex())
}

// Color tags for TextViews with SetDynamicColors(true).
const (
	TagBorder = "[#3b4261]"
	TagMuted  = "[#565f89]"
	TagAccent = "[#7aa2f7]"
	TagKey    = "[#bb9af7]"
	TagRed    = "[#f7768e]"
	TagYellow = "[#e0af68]"
	TagGreen  = "[#9ece6a]"
	TagCyan   = "[#7dcfff]"
	TagBold   = "[::b]"
	TagReset  = "[-:-:-]"
)

// ApplyTheme must run before any widget is created: tview copies
// tview.Styles into each primitive when it is constructed.
func ApplyTheme() {
	tview.Styles = tview.Theme{
		PrimitiveBackgroundColor:    ColBg,
		ContrastBackgroundColor:     ColSurface,
		MoreContrastBackgroundColor: ColBorder,
		BorderColor:                 ColBorder,
		TitleColor:                  ColMuted,
		GraphicsColor:               ColBorder,
		PrimaryTextColor:            ColFg,
		SecondaryTextColor:          ColMuted,
		TertiaryTextColor:           ColKey,
		InverseTextColor:            tcell.NewHexColor(0x1a1b26),
		ContrastSecondaryTextColor:  ColAccent,
	}

	// Rounded corners. The *Focus runes match the normal ones so focus is
	// shown by color, not by tview's default switch to double lines.
	b := &tview.Borders
	b.Horizontal, b.Vertical = '─', '│'
	b.TopLeft, b.TopRight = '╭', '╮'
	b.BottomLeft, b.BottomRight = '╰', '╯'
	b.HorizontalFocus, b.VerticalFocus = '─', '│'
	b.TopLeftFocus, b.TopRightFocus = '╭', '╮'
	b.BottomLeftFocus, b.BottomRightFocus = '╰', '╯'
}

// StylePane gives a dashboard panel the standard frame:
// rounded border and a left-aligned " 1 · Calendar " title.
func StylePane(box *tview.Box, n int, name string) {
	box.SetBorder(true).
		SetTitle(fmt.Sprintf(" %d · %s ", n, name)).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1)
	SetPaneFocus(box, false)
}

// FramePanel gives any page panel the dashboard frame: rounded border and
// a left-aligned " Title " that is accent-colored while focused.
func FramePanel(box *tview.Box, title string, focused bool) {
	box.SetBorder(true).
		SetTitle(" "+title+" ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1)
	SetPaneFocus(box, focused)
}

// SetPaneFocus colors a panel's frame for the focused or unfocused state.
func SetPaneFocus(box *tview.Box, focused bool) {
	if focused {
		box.SetBorderColor(ColAccent).SetTitleColor(ColAccent)
	} else {
		box.SetBorderColor(ColBorder).SetTitleColor(ColMuted)
	}
}

// KeyHints renders key/label pairs: KeyHints("enter", "edit", "n", "new").
func KeyHints(pairs ...string) string {
	s := ""
	for i := 0; i+1 < len(pairs); i += 2 {
		s += fmt.Sprintf("%s%s%s %s   ", TagKey, pairs[i], TagReset, pairs[i+1])
	}
	return s
}
