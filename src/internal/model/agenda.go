package model

import (
	"sort"
	"time"
)

// Agenda item kinds.
const (
	ItemClass      = "class"
	ItemAssignment = "assignment"
	ItemTask       = "task"
)

// AgendaItem is one thing on a day: a class meeting, an assignment due, or
// a task due. Index points into the matching AppData slice (Courses,
// Assignments or Tasks); CourseIndex is the item's course in Courses, or -1.
type AgendaItem struct {
	Kind        string
	Index       int
	CourseIndex int
	Title       string
	At          time.Time // class start, or due time
	End         time.Time // class end; zero for due items
	Location    string
}

// Agenda lists everything on the given local day, in time order: class
// meetings for that weekday plus assignments and tasks due that day.
// Submitted assignments are left out unless includeDone is set.
func (d *AppData) Agenda(day time.Time, includeDone bool) []AgendaItem {
	day = day.In(time.Local)
	y, m, dd := day.Date()
	midnight := time.Date(y, m, dd, 0, 0, 0, 0, time.Local)

	courseIndex := make(map[string]int, len(d.Courses))
	for i, c := range d.Courses {
		courseIndex[c.ID] = i
	}
	lookup := func(id string) int {
		if i, ok := courseIndex[id]; ok {
			return i
		}
		return -1
	}

	var items []AgendaItem
	for ci, c := range d.Courses {
		for _, mt := range c.Meetings {
			if !mt.OnWeekday(midnight.Weekday()) {
				continue
			}
			s, e, ok := mt.Minutes()
			if !ok {
				continue
			}
			loc := mt.Location
			if loc == "" {
				loc = c.Location
			}
			items = append(items, AgendaItem{
				Kind: ItemClass, Index: ci, CourseIndex: ci,
				Title:    ShortCode(c.Code, c.Name),
				At:       atClock(midnight, s),
				End:      atClock(midnight, e),
				Location: loc,
			})
		}
	}

	for i, a := range d.Assignments {
		if a.DueAt == nil || !SameLocalDay(*a.DueAt, midnight) {
			continue
		}
		if a.Done() && !includeDone {
			continue
		}
		items = append(items, AgendaItem{
			Kind: ItemAssignment, Index: i, CourseIndex: lookup(a.CourseID),
			Title: a.Name, At: a.DueAt.In(time.Local),
		})
	}

	for i, t := range d.Tasks {
		if t.DueAt == nil || !SameLocalDay(*t.DueAt, midnight) {
			continue
		}
		items = append(items, AgendaItem{
			Kind: ItemTask, Index: i, CourseIndex: lookup(t.CourseID),
			Title: t.Title, At: t.DueAt.In(time.Local),
		})
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].At.Before(items[j].At) })
	return items
}

// atClock returns day's date at minutes after midnight, as a wall-clock
// time, so a 10:00 class is at 10:00 even on a daylight-saving change day.
func atClock(day time.Time, minutes int) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d, minutes/60, minutes%60, 0, 0, day.Location())
}

// SameLocalDay reports whether a and b fall on the same local calendar day.
func SameLocalDay(a, b time.Time) bool {
	ay, am, ad := a.In(time.Local).Date()
	by, bm, bd := b.In(time.Local).Date()
	return ay == by && am == bm && ad == bd
}

// CourseIndexByID returns the index of the course with the given ID, or -1.
func (d *AppData) CourseIndexByID(id string) int {
	for i, c := range d.Courses {
		if c.ID == id {
			return i
		}
	}
	return -1
}
