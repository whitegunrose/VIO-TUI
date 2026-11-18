package dates

import (
	"testing"
	"time"
)

// Wednesday, Nov 18 2026, 10:30 AM
var now = time.Date(2026, time.November, 18, 10, 30, 0, 0, time.Local)

func at(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.Local)
}

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want time.Time
	}{
		{"2026-11-20 17:00", at(2026, 11, 20, 17, 0)},
		{"2026-11-20", at(2026, 11, 20, 23, 59)},
		{"2026-11-20 5pm", at(2026, 11, 20, 17, 0)},
		{"today", at(2026, 11, 18, 23, 59)},
		{"tomorrow", at(2026, 11, 19, 23, 59)},
		{"tmr 9am", at(2026, 11, 19, 9, 0)},
		{"tonight", at(2026, 11, 18, 20, 0)},
		{"fri", at(2026, 11, 20, 23, 59)},
		{"Fri 5pm", at(2026, 11, 20, 17, 0)},
		{"friday at 5:30 PM", at(2026, 11, 20, 17, 30)},
		{"fri 5", at(2026, 11, 20, 17, 0)},
		{"fri 9", at(2026, 11, 20, 9, 0)},
		{"wed", at(2026, 11, 18, 23, 59)}, // today counts
		{"next wed", at(2026, 11, 25, 23, 59)},
		{"next fri noon", at(2026, 11, 27, 12, 0)},
		{"next tue", at(2026, 11, 24, 23, 59)},
		{"next sun", at(2026, 11, 29, 23, 59)},
		{"next mon", at(2026, 11, 23, 23, 59)},
		{"mon eod", at(2026, 11, 23, 23, 59)},
		{"+3d", at(2026, 11, 21, 10, 30)},
		{"+2h", at(2026, 11, 18, 12, 30)},
		{"+1w", at(2026, 11, 25, 10, 30)},
		{"in 2 days", at(2026, 11, 20, 10, 30)},
		{"5pm", at(2026, 11, 18, 17, 0)},
		{"9am", at(2026, 11, 19, 9, 0)}, // already passed today
		{"17:00", at(2026, 11, 18, 17, 0)},
		{"11/20", at(2026, 11, 20, 23, 59)},
		{"1/5", at(2027, 1, 5, 23, 59)}, // next one to come
		{"11/20/26 3pm", at(2026, 11, 20, 15, 0)},
		{"nov 20", at(2026, 11, 20, 23, 59)},
		{"December 3rd 2pm", at(2026, 12, 3, 14, 0)},
		{"dec 3 2027", at(2027, 12, 3, 23, 59)},
	}
	for _, c := range cases {
		got, err := Parse(c.in, now)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", c.in, err)
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("Parse(%q) = %s, want %s", c.in, got.Format(Layout), c.want.Format(Layout))
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{"", "5", "someday", "2026-02-30", "13/40", "fri 25pm", "next blursday"} {
		if got, err := Parse(in, now); err == nil {
			t.Errorf("Parse(%q) = %s, want error", in, got.Format(Layout))
		}
	}
}

func TestRelative(t *testing.T) {
	cases := map[time.Duration]string{
		10 * time.Second:      "now",
		45 * time.Minute:      "in 45m",
		-3 * time.Hour:        "3h ago",
		5 * 24 * time.Hour:    "in 5 days",
		-50 * 24 * time.Hour:  "50 days ago",
		30*time.Hour + 1:      "in 30h",
		-90 * time.Second:     "1m ago",
		7*24*time.Hour + 3600: "in 7 days",
	}
	for d, want := range cases {
		if got := Relative(now.Add(d), now); got != want {
			t.Errorf("Relative(%v) = %q, want %q", d, got, want)
		}
	}
}
