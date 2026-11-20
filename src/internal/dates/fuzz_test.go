package dates

import (
	"fmt"
	"testing"
	"time"
)

// Fuzz tests check laws: rules that must hold for every input, not just the
// examples in dates_test.go. See aidlc-docs/design-artifacts/testing_strategy.md.
//
// `go test` runs each one on its seed inputs only. To search for new failing
// inputs, run one fuzzer at a time, e.g.
//
//	go test -run '^$' -fuzz FuzzParse -fuzztime 30s ./internal/dates

// Law D1: Parse never panics, whatever is typed.
// Law D2: a successful result is in now's time zone, on a whole minute.
func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"2026-11-20 17:00", "2026-11-20 5pm", "today", "tmr 9am", "tonight",
		"friday at 5:30 PM", "fri 5", "next wed", "next fri noon", "mon eod",
		"+3d", "+2h", "in 2 days", "9am", "17:00", "11/20", "11/20/26 3pm",
		"nov 20", "December 3rd 2pm", "dec 3 2027", "", "5", "someday",
		"2026-02-30", "13/40", "fri 25pm", "next blursday", "at", "  at  5pm ",
		"in 0 days", "+999999999999h", "2/29", "3/14 5pm",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got, err := Parse(s, now)
		if err != nil {
			return
		}
		if got.Location() != now.Location() {
			t.Errorf("Parse(%q) location = %v, want %v", s, got.Location(), now.Location())
		}
		if got.Second() != 0 || got.Nanosecond() != 0 {
			t.Errorf("Parse(%q) = %s, not on a whole minute", s, got)
		}
	})
}

// Law D3: what VIO displays in a form reads back unchanged.
// Formatting any minute with Layout and parsing it gives the same moment.
func FuzzLayoutRoundTrip(f *testing.F) {
	// Inputs map to year 1970 + y%200, month 1 + mo%12, day 1 + d%31
	f.Add(56, 10, 19, 17, 0) // 2026-11-20 17:00
	f.Add(58, 1, 28, 23, 59) // 2028-02-29 23:59
	f.Add(56, 2, 7, 2, 30)   // 2026-03-08 02:30, inside the spring-forward gap in US zones
	f.Add(56, 10, 0, 1, 30)  // 2026-11-01 01:30, the repeated hour in US zones
	f.Fuzz(func(t *testing.T, y, mo, d, h, mi int) {
		want := time.Date(1970+abs(y)%200, time.Month(1+abs(mo)%12), 1+abs(d)%31, abs(h)%24, abs(mi)%60, 0, 0, time.Local)
		text := want.Format(Layout)
		got, err := Parse(text, now)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", text, err)
		}
		if got.Format(Layout) != text {
			t.Errorf("Parse(%q) = %q, want the same text back", text, got.Format(Layout))
		}
	})
}

// Law D4: any month/day that exists in some year ("2/29", "12/31") can be
// typed without a year, and reads as the next time that date comes around.
// Law D5: a typed time lands at that clock time on that day, including on
// daylight-saving change days (run with a DST time zone to cover them).
func FuzzMonthDay(f *testing.F) {
	// The fuzz inputs map to a date as month = 1 + mo%12, day = 1 + d%days,
	// so seed (m-1, d-1) means m/d
	f.Add(10, 19) // 11/20
	f.Add(0, 4)   // 1/5
	f.Add(11, 30) // 12/31
	f.Add(1, 27)  // 2/28
	f.Add(1, 28)  // 2/29 (regression: no Feb 29 within one year)
	f.Add(2, 13)  // 3/14 (regression, found by fuzzing: Mar 14 2027 is a DST change day)
	f.Add(10, 6)  // 11/7 (regression: Nov 7 2027 is the other DST change day)
	f.Fuzz(func(t *testing.T, mo, d int) {
		month := time.Month(1 + abs(mo)%12)
		// 2028 is a leap year, so this is the longest each month can be
		day := 1 + abs(d)%daysIn(2028, month)
		text := fmt.Sprintf("%d/%d 5pm", month, day)
		got, err := Parse(text, now)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", text, err)
		}
		if got.Month() != month || got.Day() != day || got.Hour() != 17 || got.Minute() != 0 {
			t.Errorf("Parse(%q) = %s, wrong month or day", text, got.Format(Layout))
		}
		if got.Before(midnight(now)) || got.After(midnight(now).AddDate(4, 0, 1)) {
			t.Errorf("Parse(%q) = %s, not the next one to come", text, got.Format(Layout))
		}
	})
}

func abs(n int) int {
	if n < 0 {
		if n == -n { // the most negative int
			return 0
		}
		return -n
	}
	return n
}

func daysIn(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
