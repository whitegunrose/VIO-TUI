package dates

import (
	"testing"
	"time"
)

// withZone runs the test with time.Local set to a zone that has daylight
// saving, so DST cases are covered even where the machine runs in UTC.
func withZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Skipf("time zone %s not available: %v", name, err)
	}
	old := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = old })
	return loc
}

// Regression for the DST bug found by FuzzMonthDay: times on daylight-saving
// change days were an hour off because Parse added minutes to midnight.
func TestParseOnDSTChangeDays(t *testing.T) {
	la := withZone(t, "America/Los_Angeles")
	now := time.Date(2026, 11, 18, 10, 30, 0, 0, la)
	cases := []struct {
		in   string
		want time.Time
	}{
		{"3/14 5pm", time.Date(2027, 3, 14, 17, 0, 0, 0, la)},   // spring forward
		{"3/14", time.Date(2027, 3, 14, 23, 59, 0, 0, la)},      // was Mar 15, 12:59 AM
		{"11/7 5pm", time.Date(2027, 11, 7, 17, 0, 0, 0, la)},   // fall back
		{"11/7", time.Date(2027, 11, 7, 23, 59, 0, 0, la)},      // was 10:59 PM
		{"3/14/27 1am", time.Date(2027, 3, 14, 1, 0, 0, 0, la)}, // before the change
	}
	for _, c := range cases {
		got, err := Parse(c.in, now)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", c.in, err)
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("Parse(%q) = %s, want %s", c.in, got.Format("Jan 2 2006 15:04 MST"), c.want.Format("Jan 2 2006 15:04 MST"))
		}
	}
}

// Regression: Feb 29 typed without a year (next one is 2028), and offsets
// too large for time.Duration (they used to overflow into a garbage date).
func TestParseLeapDayAndHugeOffsets(t *testing.T) {
	got, err := Parse("2/29", now)
	if want := at(2028, 2, 29, 23, 59); err != nil || !got.Equal(want) {
		t.Errorf("Parse(\"2/29\") = %s, %v; want %s", got.Format(Layout), err, want.Format(Layout))
	}
	for _, in := range []string{"+999999999999h", "in 99999999999999999999 days", "+100000w", "+3661d"} {
		if got, err := Parse(in, now); err == nil {
			t.Errorf("Parse(%q) = %s, want an error", in, got.Format(Layout))
		}
	}
	if _, err := Parse("+3660d", now); err != nil {
		t.Errorf("Parse(\"+3660d\") (the limit) error: %v", err)
	}
}
