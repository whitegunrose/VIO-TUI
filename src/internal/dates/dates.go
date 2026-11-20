// Package dates reads the short, human date strings typed into VIO's forms:
// "fri 5pm", "tomorrow", "+3d", "next tue noon", "2026-11-20 17:00".
package dates

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Layout is the strict format VIO shows dates in and always accepts.
const Layout = "2006-01-02 15:04"

// Hint is a short example list for form labels and errors.
const Hint = "fri 5pm, tomorrow, +3d, next tue noon, 11/20, 2026-11-20 17:00"

// endOfDay is the time used when only a day is given.
const endOfDay = 23*60 + 59

var weekdays = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tues": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "weds": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

var months = map[string]time.Month{
	"jan": time.January, "january": time.January,
	"feb": time.February, "february": time.February,
	"mar": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"may": time.May,
	"jun": time.June, "june": time.June,
	"jul": time.July, "july": time.July,
	"aug": time.August, "august": time.August,
	"sep": time.September, "sept": time.September, "september": time.September,
	"oct": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"dec": time.December, "december": time.December,
}

var (
	isoDateRe  = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`)
	slashRe    = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})(?:/(\d{2}|\d{4}))?$`)
	offsetRe   = regexp.MustCompile(`^\+(\d+)\s*([hdw])$`)
	inRe       = regexp.MustCompile(`^in\s+(\d+)\s*(hours?|hrs?|h|days?|d|weeks?|wks?|w)$`)
	clockRe    = regexp.MustCompile(`^(\d{1,2})(?::(\d{2}))?\s*([ap])?\.?m?\.?$`)
	monthDayRe = regexp.MustCompile(`^([a-z]+)\s+(\d{1,2})(?:st|nd|rd|th)?(?:,?\s+(\d{4}))?$`)
)

// Parse reads text relative to now, in now's location. A day with no time
// means the end of that day (11:59 PM). A time with no day means the next
// time that clock time comes around (today, or tomorrow if it has passed).
func Parse(text string, now time.Time) (time.Time, error) {
	s := strings.ToLower(strings.TrimSpace(text))
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return time.Time{}, fmt.Errorf("enter a date, like %s", Hint)
	}

	if t, err := time.ParseInLocation(Layout, s, now.Location()); err == nil {
		return t, nil
	}

	// Relative offsets keep the current clock time: "+2h", "+3d", "in 2 weeks"
	if m := offsetRe.FindStringSubmatch(s); m != nil {
		return offsetFrom(now, m[1], m[2])
	}
	if m := inRe.FindStringSubmatch(s); m != nil {
		return offsetFrom(now, m[1], m[2][:1])
	}

	// Split into a day part and an optional trailing time part
	dayPart, clock, hasClock := splitClock(s)

	var day time.Time
	if dayPart == "" {
		if !hasClock {
			return time.Time{}, fmt.Errorf("can't read %q; try %s", text, Hint)
		}
		day = midnight(now)
		if atClock(day, clock).Before(now) {
			day = day.AddDate(0, 0, 1)
		}
	} else {
		d, err := parseDay(dayPart, now)
		if err != nil {
			return time.Time{}, fmt.Errorf("can't read %q; try %s", text, Hint)
		}
		day = d
	}

	if !hasClock {
		clock = endOfDay
	}
	return atClock(day, clock), nil
}

// atClock returns day's date at the given minutes after midnight, as a
// wall-clock time. Adding minutes to midnight instead would be an hour off
// on daylight-saving change days, which are 23 or 25 hours long.
func atClock(day time.Time, minutes int) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d, minutes/60, minutes%60, 0, 0, day.Location())
}

// maxOffsetDays caps "+Nd"-style offsets at about ten years. Without a cap,
// a huge hour count overflows time.Duration and gives a nonsense date.
const maxOffsetDays = 3660

func offsetFrom(now time.Time, digits, unit string) (time.Time, error) {
	n, err := strconv.Atoi(digits)
	days := n
	switch unit {
	case "h":
		days = n / 24
	case "w":
		days = n * 7
	}
	if err != nil || n > maxOffsetDays*24 || days > maxOffsetDays {
		return time.Time{}, fmt.Errorf("%s%s is too far ahead; use a date instead", digits, unit)
	}
	return addOffset(now, n, unit), nil
}

func addOffset(now time.Time, n int, unit string) time.Time {
	switch unit {
	case "h":
		return now.Add(time.Duration(n) * time.Hour)
	case "w":
		return now.AddDate(0, 0, 7*n)
	default:
		return now.AddDate(0, 0, n)
	}
}

func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// splitClock pulls a time off the end of s: "fri 5pm" → ("fri", 17:00).
// Words like "noon", "eod" and "at" are understood.
func splitClock(s string) (string, int, bool) {
	s = strings.TrimSuffix(s, " at")
	words := strings.Fields(s)
	if len(words) == 0 {
		return "", 0, false
	}

	switch words[len(words)-1] {
	case "noon":
		return trimAt(words[:len(words)-1]), 12 * 60, true
	case "midnight", "eod":
		return trimAt(words[:len(words)-1]), endOfDay, true
	case "morning":
		return trimAt(words[:len(words)-1]), 9 * 60, true
	case "tonight":
		return trimAt(append(words[:len(words)-1], "today")), 20 * 60, true
	}

	// Try the last one or two words as a clock ("5pm", "5 pm", "17:00")
	for take := 2; take >= 1; take-- {
		if len(words) < take {
			continue
		}
		candidate := strings.Join(words[len(words)-take:], "")
		if minutes, ok := parseClock(candidate); ok {
			rest := words[:len(words)-take]
			// A bare number after a month ("nov 20") is a day, not a time
			if take == 1 && len(rest) > 0 && !strings.ContainsAny(candidate, ":apm") {
				if _, isMonth := months[rest[len(rest)-1]]; isMonth {
					continue
				}
			}
			// A lone bare number ("5") is too ambiguous to read as a time
			if len(rest) == 0 && !strings.ContainsAny(candidate, ":apm") {
				continue
			}
			return trimAt(rest), minutes, true
		}
	}
	return s, 0, false
}

func trimAt(words []string) string {
	if len(words) > 0 && words[len(words)-1] == "at" {
		words = words[:len(words)-1]
	}
	return strings.Join(words, " ")
}

// parseClock reads "5pm", "5:30pm", "17:00", "9am". Times without am/pm
// are read as 24-hour.
func parseClock(s string) (int, bool) {
	m := clockRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	h, _ := strconv.Atoi(m[1])
	min := 0
	if m[2] != "" {
		min, _ = strconv.Atoi(m[2])
	}
	if min > 59 {
		return 0, false
	}
	switch m[3] {
	case "a":
		if h < 1 || h > 12 {
			return 0, false
		}
		if h == 12 {
			h = 0
		}
	case "p":
		if h < 1 || h > 12 {
			return 0, false
		}
		if h != 12 {
			h += 12
		}
	default:
		if h > 23 {
			return 0, false
		}
		// A bare hour like "fri 5" reads the way class times do:
		// 7–11 is morning, 12–6 is afternoon
		if m[2] == "" && h >= 1 && h <= 6 {
			h += 12
		}
	}
	return h*60 + min, true
}

// parseDay reads the day part and returns that day at midnight.
func parseDay(s string, now time.Time) (time.Time, error) {
	today := midnight(now)
	loc := now.Location()

	switch s {
	case "today", "tod":
		return today, nil
	case "tomorrow", "tmr", "tmrw", "tom":
		return today.AddDate(0, 0, 1), nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	}

	// "fri", "this fri": the next one coming (today counts).
	// "next fri": that day in next week, with weeks starting Monday.
	words := strings.Fields(s)
	if wd, ok := weekdays[words[len(words)-1]]; ok && len(words) <= 2 {
		mod := ""
		if len(words) == 2 {
			mod = words[0]
			if mod != "next" && mod != "this" {
				return time.Time{}, fmt.Errorf("unknown day")
			}
		}
		delta := (int(wd) - int(today.Weekday()) + 7) % 7
		if mod == "next" {
			// "next tue" is the Tuesday of next week (weeks start Monday)
			daysToNextMonday := (8 - int(today.Weekday())) % 7
			if daysToNextMonday == 0 {
				daysToNextMonday = 7
			}
			delta = daysToNextMonday + (int(wd)+6)%7
		}
		return today.AddDate(0, 0, delta), nil
	}

	if m := isoDateRe.FindStringSubmatch(s); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		return validDate(y, time.Month(mo), d, loc)
	}

	// "11/20" or "11/20/2026": with no year, the next 11/20 to come
	if m := slashRe.FindStringSubmatch(s); m != nil {
		mo, _ := strconv.Atoi(m[1])
		d, _ := strconv.Atoi(m[2])
		if m[3] != "" {
			y, _ := strconv.Atoi(m[3])
			if y < 100 {
				y += 2000
			}
			return validDate(y, time.Month(mo), d, loc)
		}
		return nextOccurrence(today, time.Month(mo), d)
	}

	// "nov 20", "november 20th", "nov 20 2026"
	if m := monthDayRe.FindStringSubmatch(s); m != nil {
		mo, ok := months[m[1]]
		if !ok {
			return time.Time{}, fmt.Errorf("unknown month")
		}
		d, _ := strconv.Atoi(m[2])
		if m[3] != "" {
			y, _ := strconv.Atoi(m[3])
			return validDate(y, mo, d, loc)
		}
		return nextOccurrence(today, mo, d)
	}

	return time.Time{}, fmt.Errorf("unknown day")
}

func validDate(y int, m time.Month, d int, loc *time.Location) (time.Time, error) {
	t := time.Date(y, m, d, 0, 0, 0, 0, loc)
	if t.Year() != y || t.Month() != m || t.Day() != d {
		return time.Time{}, fmt.Errorf("no such date")
	}
	return t, nil
}

// nextOccurrence returns the next m/d on or after today. Feb 29 can be up
// to 8 years away (leap years skip 2100, 2200, 2300), so it looks that far.
func nextOccurrence(today time.Time, m time.Month, d int) (time.Time, error) {
	for y := today.Year(); y <= today.Year()+8; y++ {
		t, err := validDate(y, m, d, today.Location())
		if err == nil && !t.Before(today) {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("no such date")
}

// Describe renders t for a "reads as" preview: "Fri Nov 20, 5:00 PM (in 2 days)".
func Describe(t, now time.Time) string {
	return t.Format("Mon Jan 2 2006, 3:04 PM") + " (" + Relative(t, now) + ")"
}

// Relative renders a short distance from now: "in 3 days", "2h ago", "now".
func Relative(t, now time.Time) string {
	d := t.Sub(now)
	future := d >= 0
	if !future {
		d = -d
	}
	var s string
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		s = fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		s = fmt.Sprintf("%dh", int(d.Hours()))
	default:
		s = fmt.Sprintf("%d days", int(d.Hours()/24))
	}
	if future {
		return "in " + s
	}
	return s + " ago"
}
