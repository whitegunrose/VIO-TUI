package model

import (
	"reflect"
	"testing"
	"time"
)

// Fuzz tests check laws: rules that must hold for every input. See
// aidlc-docs/design-artifacts/testing_strategy.md. To search for new failing
// inputs: go test -run '^$' -fuzz FuzzParseMeeting -fuzztime 30s ./internal/model

// Law M1: ParseMeeting never panics.
// Law M2: a parsed meeting is valid: at least one day, days in week order
// with no repeats, and an end after its start.
// Law M3: a parsed meeting round-trips: ParseMeeting(m.String()) == m, so the
// meeting editor shows exactly what was saved.
func FuzzParseMeeting(f *testing.F) {
	for _, s := range []string{
		"Tue Thu 10:00 AM - 11:15 AM @ JD 1234", "MW 2:30-3:45pm", "TTh 10-11:15",
		"MWF 9am to 9:50am", "Thursday 13:00-14:15", "Tue 11:30 - 1", "mon, wed 1-2",
		"F 9 to 8", "Xyz 1-2", "", "@", "M 1-2 @ ", "Mon 12-12:01am", "TR 0:00-23:59",
		"Mon Wed 9am to 10:15am @ JD 1600 - Lab", "su sa 7 - 8 @ Gym @ North",
		"T 1-17", // regression, found by fuzzing: end became 29:00
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		m, err := ParseMeeting(s)
		if err != nil {
			return
		}
		if len(m.Days) == 0 {
			t.Fatalf("ParseMeeting(%q) has no days", s)
		}
		last := -1
		for _, d := range m.Days {
			i := dayIndex(d)
			if i <= last {
				t.Fatalf("ParseMeeting(%q) days %v not in week order or repeated", s, m.Days)
			}
			last = i
		}
		if _, _, ok := m.Minutes(); !ok {
			t.Fatalf("ParseMeeting(%q) = %+v, end not after start", s, m)
		}
		back, err := ParseMeeting(m.String())
		if err != nil {
			t.Fatalf("ParseMeeting(%q) gave %q, which doesn't parse back: %v", s, m.String(), err)
		}
		if !reflect.DeepEqual(back, m) {
			t.Fatalf("round trip changed the meeting:\n input %q\n first %+v\n shown %q\n back  %+v", s, m, m.String(), back)
		}
	})
}

// Law M4: overlap detection is symmetric and ignores back-to-back classes:
// two meetings on the same day conflict exactly when one starts before the
// other ends, whichever course is listed first.
func FuzzMeetingConflicts(f *testing.F) {
	f.Add(600, 675, 675, 750) // back to back
	f.Add(600, 675, 660, 735) // overlap
	f.Add(600, 675, 600, 675) // identical
	f.Fuzz(func(t *testing.T, s1, e1, s2, e2 int) {
		clamp := func(n int) int { return ((n % 1440) + 1440) % 1440 }
		s1, e1, s2, e2 = clamp(s1), clamp(e1), clamp(s2), clamp(e2)
		if e1 <= s1 || e2 <= s2 {
			return // not a valid meeting
		}
		a := Course{Meetings: []Meeting{{Days: []string{"Tue"}, Start: hhmm(s1), End: hhmm(e1)}}}
		b := Course{Meetings: []Meeting{{Days: []string{"Tue"}, Start: hhmm(s2), End: hhmm(e2)}}}
		want := max(s1, s2) < min(e1, e2)
		ab := len(MeetingConflicts([]Course{a, b})) == 1
		ba := len(MeetingConflicts([]Course{b, a})) == 1
		if ab != want || ba != want {
			t.Fatalf("%s-%s vs %s-%s: conflict a,b=%v b,a=%v, want %v", hhmm(s1), hhmm(e1), hhmm(s2), hhmm(e2), ab, ba, want)
		}
	})
}

// Law M5: Agenda is always in time order and only holds items from that day.
func FuzzAgendaOrder(f *testing.F) {
	f.Add(600, 675, 30, 1439)
	f.Fuzz(func(t *testing.T, start, end, taskMin, assignMin int) {
		clamp := func(n int) int { return ((n % 1440) + 1440) % 1440 }
		start, end, taskMin, assignMin = clamp(start), clamp(end), clamp(taskMin), clamp(assignMin)
		day := time.Date(2026, 11, 17, 0, 0, 0, 0, time.Local) // a Tuesday
		taskDue := day.Add(time.Duration(taskMin) * time.Minute)
		assignDue := day.Add(time.Duration(assignMin) * time.Minute)
		other := day.AddDate(0, 0, 1)
		d := AppData{
			Courses:     []Course{{ID: "c", Code: "COMP 440", Meetings: []Meeting{{Days: []string{"Tue"}, Start: hhmm(start), End: hhmm(end)}}}},
			Assignments: []Assignment{{ID: "a", CourseID: "c", DueAt: &assignDue}, {ID: "b", DueAt: &other}},
			Tasks:       []Task{{ID: "t", DueAt: &taskDue}},
		}
		items := d.Agenda(day.Add(12*time.Hour), true)
		for i, it := range items {
			if !SameLocalDay(it.At, day) {
				t.Fatalf("item %d %+v is not on %s", i, it, day.Format("Jan 2"))
			}
			if i > 0 && it.At.Before(items[i-1].At) {
				t.Fatalf("items out of order: %+v", items)
			}
		}
	})
}

func dayIndex(d string) int {
	for i, name := range dayOrder {
		if name == d {
			return i
		}
	}
	return -1
}
