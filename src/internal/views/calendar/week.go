package calendar

import (
	"VIO/internal/model"
	"VIO/internal/theme"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const (
	slotMinutes = 30 // one grid row
	slotsPerDay = 24 * 60 / slotMinutes
	gutterWidth = 7 // hour labels on the left
	headerRows  = 4 // day names, rule, due strip, rule
)

// weekBlock is one class meeting or task drawn in a day column.
type weekBlock struct {
	start, end   int // minutes after midnight
	title, extra string
	sub          string // second line; the time range when empty
	color        tcell.Color
	lane, lanes  int // side-by-side position when blocks overlap
}

// weekGrid draws a Sunday–Saturday time grid. Blocks are a colored bar plus
// colored text with no background fill, so a transparent terminal's
// wallpaper still shows through.
type weekGrid struct {
	*tview.Box
	data        *model.AppData
	state       *calendarState
	scroll      int       // first visible slot
	visible     int       // slots that fit, set on each draw
	scrolledFor time.Time // week the auto-scroll last ran for
}

func newWeekGrid(data *model.AppData, state *calendarState) *weekGrid {
	return &weekGrid{Box: tview.NewBox(), data: data, state: state}
}

func (g *weekGrid) scrollBy(delta int) {
	g.scroll += delta
	g.clampScroll()
}

func (g *weekGrid) clampScroll() {
	maxScroll := slotsPerDay - g.visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if g.scroll > maxScroll {
		g.scroll = maxScroll
	}
	if g.scroll < 0 {
		g.scroll = 0
	}
}

func weekStart(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return d.AddDate(0, 0, -int(d.Weekday()))
}

func weekTitle(start time.Time) string {
	end := start.AddDate(0, 0, 6)
	if start.Month() == end.Month() {
		return fmt.Sprintf(" Week of %s %d – %d ", start.Format("Jan"), start.Day(), end.Day())
	}
	return fmt.Sprintf(" Week of %s – %s ", start.Format("Jan 2"), end.Format("Jan 2"))
}

func (g *weekGrid) Draw(screen tcell.Screen) {
	start := weekStart(g.state.selected())
	g.SetTitle(weekTitle(start))
	g.DrawForSubclass(screen, g)
	x, y, w, h := g.GetInnerRect()

	border := tcell.StyleDefault.Foreground(theme.ColBorder)
	muted := tcell.StyleDefault.Foreground(theme.ColMuted)
	fg := tcell.StyleDefault.Foreground(theme.ColFg)

	// Column widths: spread any leftover cells across the first columns so
	// the grid fills the panel
	avail := w - gutterWidth - 7 // minus one separator per column
	base, extra := avail/7, avail%7
	if base < 5 || h < headerRows+2 {
		put(screen, x, y, w, "Make the terminal larger to see the week.", muted)
		return
	}
	g.visible = h - headerRows

	days := make([]time.Time, 7)
	blocks := make([][]weekBlock, 7)
	for i := range days {
		days[i] = start.AddDate(0, 0, i)
		blocks[i] = g.blocksFor(days[i])
	}
	if !g.scrolledFor.Equal(start) {
		g.autoScroll(start, blocks)
		g.scrolledFor = start
	}
	g.clampScroll()

	colWidth := func(i int) int {
		if i < extra {
			return base + 1
		}
		return base
	}
	colX := func(i int) int {
		cx := x + gutterWidth + 1
		for k := 0; k < i; k++ {
			cx += colWidth(k) + 1
		}
		return cx
	}
	right := colX(6) + colWidth(6)
	now := time.Now()
	selected := g.state.selected()
	top := y + headerRows

	// Rules: horizontal under the day names and the due strip, then the
	// column separators with crossings
	for _, row := range []int{1, 3} {
		for cx := x; cx < right; cx++ {
			screen.SetContent(cx, y+row, '─', nil, border)
		}
	}
	for row := 0; row < h; row++ {
		ch := '│'
		if row == 1 || row == 3 {
			ch = '┼'
		}
		for i := 0; i < 7; i++ {
			screen.SetContent(colX(i)-1, y+row, ch, nil, border)
		}
	}

	// Day names
	for i, d := range days {
		label := strings.ToUpper(d.Format("Mon")) + " " + fmt.Sprint(d.Day())
		style := fg
		if sameDay(d, now) {
			style = tcell.StyleDefault.Foreground(theme.ColAccent).Bold(true)
		}
		if sameDay(d, selected) {
			if len(label)+2 <= colWidth(i) {
				label = " " + label + " "
			}
			style = tcell.StyleDefault.Background(theme.ColAccent).Foreground(theme.ColDark).Bold(true)
		}
		putCentered(screen, colX(i), y, colWidth(i), label, style)
	}

	// Due strip: assignments due that day
	put(screen, x+1, y+2, gutterWidth-1, "due", muted)
	for i, d := range days {
		due := assignmentsDue(g.data, d)
		if len(due) == 0 {
			continue
		}
		cx := colX(i)
		screen.SetContent(cx, y+2, '●', nil, tcell.StyleDefault.Foreground(theme.ColRed))
		text := due[0]
		if len(due) > 1 {
			text = fmt.Sprintf("%d due", len(due))
		}
		put(screen, cx+2, y+2, colWidth(i)-2, text, fg)
	}

	// Hour labels and faint hour lines
	for r := 0; r < g.visible; r++ {
		slot := g.scroll + r
		if slot >= slotsPerDay || (slot*slotMinutes)%60 != 0 {
			continue
		}
		putRight(screen, x, top+r, gutterWidth-1, hourLabel(slot*slotMinutes/60), muted)
		for i := range days {
			for cx := colX(i); cx < colX(i)+colWidth(i); cx++ {
				screen.SetContent(cx, top+r, '┈', nil, border)
			}
		}
	}

	// Current time line on today's column (blocks draw over it)
	for i, d := range days {
		if !sameDay(d, now) {
			continue
		}
		r := (now.Hour()*60+now.Minute())/slotMinutes - g.scroll
		if r < 0 || r >= g.visible {
			continue
		}
		red := tcell.StyleDefault.Foreground(theme.ColRed)
		for cx := colX(i); cx < colX(i)+colWidth(i); cx++ {
			screen.SetContent(cx, top+r, '━', nil, red)
		}
		putRight(screen, x, top+r, gutterWidth-1, "now", red)
	}

	// Blocks
	for i := range days {
		colW := colWidth(i)
		for _, b := range blocks[i] {
			laneW := colW / b.lanes
			bx := colX(i) + b.lane*laneW
			if b.lane == b.lanes-1 {
				laneW = colW - b.lane*laneW
			}
			firstRow := b.start / slotMinutes
			lastRow := (b.end+slotMinutes-1)/slotMinutes - 1
			visFirst := firstRow
			if visFirst < g.scroll {
				visFirst = g.scroll // keep the title on screen when scrolled into a block
			}
			textW := laneW - 1
			sub := b.sub
			if sub == "" {
				sub = model.FormatRange(b.start, b.end, textW < 19)
			}
			lines := []string{b.title, sub, b.extra}

			for row := visFirst; row <= lastRow; row++ {
				r := row - g.scroll
				if r >= g.visible {
					break
				}
				for cx := bx; cx < bx+laneW; cx++ {
					screen.SetContent(cx, top+r, ' ', nil, tcell.StyleDefault)
				}
				screen.SetContent(bx, top+r, '▌', nil, tcell.StyleDefault.Foreground(b.color))
				n := row - visFirst
				if n >= len(lines) || lines[n] == "" || textW < 1 {
					continue
				}
				style := muted
				if n == 0 {
					style = tcell.StyleDefault.Foreground(b.color).Bold(true)
				}
				put(screen, bx+1, top+r, textW, lines[n], style)
			}
		}
	}
}

// blocksFor collects the class meetings and timed tasks on one day.
func (g *weekGrid) blocksFor(d time.Time) []weekBlock {
	var out []weekBlock

	for i, course := range g.data.Courses {
		for _, m := range course.Meetings {
			if !m.OnWeekday(d.Weekday()) {
				continue
			}
			s, e, ok := m.Minutes()
			if !ok {
				continue
			}
			extra := ""
			if m.Location != "" {
				extra = "@ " + m.Location
			}
			out = append(out, weekBlock{
				start: s, end: e,
				title: model.ShortCode(course.Code, course.Name),
				extra: extra,
				color: theme.CourseColor(i),
			})
		}
	}

	for _, t := range g.data.Tasks {
		if t.StartsAt == nil || t.DueAt == nil {
			continue
		}
		st, due := t.StartsAt.In(time.Local), t.DueAt.In(time.Local)
		if !sameDay(st, d) {
			continue
		}
		s := st.Hour()*60 + st.Minute()
		e := due.Hour()*60 + due.Minute()
		sub := ""
		// Tasks are deadlines, not events: a long or multi-day one is a
		// one-hour marker at its start, labeled with when it's due
		if !sameDay(due, d) || e-s > 3*60 {
			e = s + 2*slotMinutes
			if sameDay(due, d) {
				sub = "due " + due.Format("3:04 PM")
			} else {
				sub = "due " + due.Format("Mon 3:04 PM")
			}
		}
		if e > 24*60 {
			e = 24 * 60
		}
		if e-s < slotMinutes {
			e = s + slotMinutes
			if e > 24*60 {
				e, s = 24*60, 24*60-slotMinutes
			}
		}
		title, color := t.Title, theme.ColYellow
		if strings.EqualFold(strings.TrimSpace(t.Status), "complete") {
			title, color = "✓ "+t.Title, theme.ColMuted
		}
		out = append(out, weekBlock{start: s, end: e, title: title, sub: sub, extra: "task", color: color})
	}

	assignLanes(out)
	return out
}

// assignLanes places overlapping blocks side by side.
func assignLanes(bs []weekBlock) {
	sort.SliceStable(bs, func(i, j int) bool {
		if bs[i].start == bs[j].start {
			return bs[i].end > bs[j].end
		}
		return bs[i].start < bs[j].start
	})

	clusterStart, clusterEnd := 0, -1
	var laneEnds []int
	finish := func(upto int) {
		for k := clusterStart; k < upto; k++ {
			bs[k].lanes = len(laneEnds)
		}
	}
	for i := range bs {
		if bs[i].start >= clusterEnd {
			finish(i)
			clusterStart = i
			laneEnds = laneEnds[:0]
		}
		placed := false
		for l, end := range laneEnds {
			if end <= bs[i].start {
				bs[i].lane, laneEnds[l], placed = l, bs[i].end, true
				break
			}
		}
		if !placed {
			bs[i].lane = len(laneEnds)
			laneEnds = append(laneEnds, bs[i].end)
		}
		if bs[i].end > clusterEnd {
			clusterEnd = bs[i].end
		}
	}
	finish(len(bs))
}

// autoScroll shows the week's earliest block (or 8 AM). When viewing this
// week later in the day, it scrolls down far enough to show the current time.
func (g *weekGrid) autoScroll(start time.Time, blocks [][]weekBlock) {
	earliest := 8 * 60
	for _, day := range blocks {
		for _, b := range day {
			if b.start < earliest {
				earliest = b.start
			}
		}
	}
	g.scroll = earliest/slotMinutes - 1

	now := time.Now()
	if !now.Before(start) && now.Before(start.AddDate(0, 0, 7)) {
		nowSlot := (now.Hour()*60 + now.Minute()) / slotMinutes
		if nowSlot >= g.scroll+g.visible {
			g.scroll = nowSlot - g.visible/3
		}
	}
}

func assignmentsDue(data *model.AppData, d time.Time) []string {
	type due struct {
		name string
		at   time.Time
	}
	var items []due
	for _, a := range data.Assignments {
		if a.DueAt != nil && sameDay(*a.DueAt, d) {
			name := a.Name
			if a.Done() {
				name = "✓ " + name
			} else if a.NeedsAttention(time.Now()) {
				name = "! " + name
			}
			items = append(items, due{name, *a.DueAt})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].at.Before(items[j].at) })
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.name
	}
	return names
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.In(time.Local).Date()
	by, bm, bd := b.In(time.Local).Date()
	return ay == by && am == bm && ad == bd
}

func hourLabel(h int) string {
	switch {
	case h == 0:
		return "12 AM"
	case h < 12:
		return fmt.Sprintf("%d AM", h)
	case h == 12:
		return "12 PM"
	default:
		return fmt.Sprintf("%d PM", h-12)
	}
}

// put draws text from x, cutting it to maxW cells with an ellipsis.
func put(screen tcell.Screen, x, y, maxW int, text string, style tcell.Style) {
	if maxW <= 0 {
		return
	}
	if runewidth.StringWidth(text) > maxW {
		text = runewidth.Truncate(text, maxW, "…")
	}
	for _, r := range text {
		screen.SetContent(x, y, r, nil, style)
		x += runewidth.RuneWidth(r)
	}
}

func putRight(screen tcell.Screen, x, y, w int, text string, style tcell.Style) {
	tw := runewidth.StringWidth(text)
	if tw > w {
		put(screen, x, y, w, text, style)
		return
	}
	put(screen, x+w-tw, y, tw, text, style)
}

func putCentered(screen tcell.Screen, x, y, w int, text string, style tcell.Style) {
	tw := runewidth.StringWidth(text)
	if tw > w {
		put(screen, x, y, w, text, style)
		return
	}
	put(screen, x+(w-tw)/2, y, tw, text, style)
}
