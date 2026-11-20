package model

import (
	"testing"
	"time"
)

// Regression: on a daylight-saving change day, class times in the agenda
// were an hour off because they were computed as midnight + minutes.
func TestAgendaClassTimesOnDSTChangeDays(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skipf("time zone not available: %v", err)
	}
	old := time.Local
	time.Local = la
	t.Cleanup(func() { time.Local = old })

	d := AppData{Courses: []Course{{ID: "c", Code: "COMP 440", Meetings: []Meeting{
		{Days: []string{"Sun"}, Start: "10:00", End: "11:15"},
	}}}}
	for _, day := range []time.Time{
		time.Date(2027, 3, 14, 12, 0, 0, 0, la), // spring forward
		time.Date(2027, 11, 7, 12, 0, 0, 0, la), // fall back
	} {
		items := d.Agenda(day, true)
		if len(items) != 1 {
			t.Fatalf("%s: got %d items, want 1", day.Format("Jan 2"), len(items))
		}
		if h, m := items[0].At.Hour(), items[0].At.Minute(); h != 10 || m != 0 {
			t.Errorf("%s: class starts %02d:%02d, want 10:00", day.Format("Jan 2"), h, m)
		}
		if h, m := items[0].End.Hour(), items[0].End.Minute(); h != 11 || m != 15 {
			t.Errorf("%s: class ends %02d:%02d, want 11:15", day.Format("Jan 2"), h, m)
		}
	}
}
