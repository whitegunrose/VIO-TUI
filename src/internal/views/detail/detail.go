// Package detail is the shared layout for VIO's full-screen detail pages
// (one course, one assignment, one task): a framed, scrollable body with
// a key-hint footer.
package detail

import (
	"VIO/internal/theme"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Page builds a detail page. body is tview-tagged text; hints is the footer
// (usually theme.KeyHints output). onKey gets every rune key first and
// returns true when it handled it. Esc calls returnTo. j/k and the arrow
// keys scroll.
func Page(title, body, hints string, onKey func(r rune) bool, returnTo func()) (tview.Primitive, *tview.TextView, *tview.TextView) {
	text := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetScrollable(true).
		SetText(body)
	theme.FramePanel(text.Box, title, true)
	text.SetBorderPadding(1, 1, 2, 2)

	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(hints)

	back := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetText(theme.KeyHints("esc", "back"))

	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(back, 1, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(text, 0, 6, true).
			AddItem(tview.NewBox(), 0, 1, false),
			0, 1, true).
		AddItem(footer, 2, 0, false)

	text.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
			}
			if onKey != nil && onKey(event.Rune()) {
				return nil
			}
		}
		return event
	})

	return page, text, footer
}

// Field renders "Label   value" with a muted label.
func Field(label, value string) string {
	return theme.TagMuted + padRight(label, 10) + theme.TagReset + value + "\n"
}

// Section renders an accent-colored heading with a blank line before it.
func Section(name string) string {
	return "\n" + theme.TagAccent + theme.TagBold + name + theme.TagReset + "\n"
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s + " "
	}
	return s + strings.Repeat(" ", n-len(s))
}

var (
	breakRe  = regexp.MustCompile(`(?i)<\s*(br|/p|/div|/li|/h[1-6]|/tr)\s*/?>`)
	bulletRe = regexp.MustCompile(`(?i)<\s*li[^>]*>`)
	tagRe    = regexp.MustCompile(`<[^>]*>`)
	blankRe  = regexp.MustCompile(`\n{3,}`)
	entities = strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&rsquo;", "'", "&lsquo;", "'", "&ldquo;", `"`, "&rdquo;", `"`, "&ndash;", "–", "&mdash;", "—")
)

// PlainText turns a Canvas HTML description into readable plain text.
func PlainText(html string) string {
	s := breakRe.ReplaceAllString(html, "\n")
	s = bulletRe.ReplaceAllString(s, "• ")
	s = tagRe.ReplaceAllString(s, "")
	s = entities.Replace(s)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	s = strings.Join(lines, "\n")
	s = blankRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
