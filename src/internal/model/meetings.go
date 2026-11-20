package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Meeting is one recurring class meeting, e.g. Tue/Thu 10:00–11:15 in JD 1234.
// Days are three-letter names ("Mon", "Thu"); Start and End are 24-hour
// "15:04" clock times in local time.
type Meeting struct {
	Days     []string `json:"days"`
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Location string   `json:"location,omitempty"`
}

var dayOrder = [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// OnWeekday reports whether the meeting happens on the given weekday.
func (m Meeting) OnWeekday(d time.Weekday) bool {
	for _, day := range m.Days {
		if day == dayOrder[d] {
			return true
		}
	}
	return false
}

// Minutes returns the start and end as minutes after midnight.
func (m Meeting) Minutes() (start, end int, ok bool) {
	s, okS := clockMinutes(m.Start)
	e, okE := clockMinutes(m.End)
	return s, e, okS && okE && e > s
}

// String renders the meeting in the same format ParseMeeting reads,
// e.g. "Tue Thu 10:00 AM – 11:15 AM @ JD 1234".
func (m Meeting) String() string {
	s, e, _ := m.Minutes()
	out := strings.Join(m.Days, " ") + " " + FormatRange(s, e, false)
	if m.Location != "" {
		out += " @ " + m.Location
	}
	return out
}

func clockMinutes(hhmm string) (int, bool) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func hhmm(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

// FormatClock renders minutes after midnight as "2:30 PM".
func FormatClock(minutes int) string {
	h, m := minutes/60, minutes%60
	suffix := "AM"
	if h >= 12 {
		suffix = "PM"
	}
	h12 := h % 12
	if h12 == 0 {
		h12 = 12
	}
	return fmt.Sprintf("%d:%02d %s", h12, m, suffix)
}

// FormatRange renders "2:30 PM – 3:45 PM", or "2:30–3:45" when compact.
func FormatRange(start, end int, compact bool) string {
	if compact {
		trim := func(s string) string {
			return strings.TrimSuffix(strings.TrimSuffix(s, " AM"), " PM")
		}
		return trim(FormatClock(start)) + "–" + trim(FormatClock(end))
	}
	return FormatClock(start) + " – " + FormatClock(end)
}

// FormatMeetings renders meetings one per line, for editing.
func FormatMeetings(meetings []Meeting) string {
	lines := make([]string, 0, len(meetings))
	for _, m := range meetings {
		lines = append(lines, m.String())
	}
	return strings.Join(lines, "\n")
}

// ParseMeetings reads one meeting per line. Blank lines and lines starting
// with # are skipped. Errors name the line they came from.
func ParseMeetings(text string) ([]Meeting, error) {
	var meetings []Meeting
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m, err := ParseMeeting(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		meetings = append(meetings, m)
	}
	return meetings, nil
}

var meetingRe = regexp.MustCompile(`(?i)^\s*(.+?)\s+(\d{1,2}(?::\d{2})?\s*(?:[ap]\.?m?\.?)?)\s*(?:-|–|—|to)\s*(\d{1,2}(?::\d{2})?\s*(?:[ap]\.?m?\.?)?)\s*(?:@\s*(.*?))?\s*$`)

// ParseMeeting reads a line like "Tue Thu 10:00 AM - 11:15 AM @ JD 1234".
// Days can be names (Mon, Thursday) or compact (MW, TTh, MWF, TR). AM/PM
// can be given once ("2:30-3:45pm"); with none, 7–11 reads as morning and
// 12–6 as afternoon, and 13–23 as 24-hour time.
func ParseMeeting(line string) (Meeting, error) {
	parts := meetingRe.FindStringSubmatch(line)
	if parts == nil {
		return Meeting{}, fmt.Errorf("expected days and a time range, like \"Tue Thu 10:00 AM - 11:15 AM @ JD 1234\"")
	}

	days, err := parseDays(parts[1])
	if err != nil {
		return Meeting{}, err
	}

	sh, sm, sap, err := parseClock(parts[2])
	if err != nil {
		return Meeting{}, err
	}
	eh, em, eap, err := parseClock(parts[3])
	if err != nil {
		return Meeting{}, err
	}

	if sap == 0 && eap == 0 {
		// Guess AM/PM only when both ends read as 12-hour times. If either
		// is already 24-hour (0 or 13–23), read both as 24-hour: "1-17" is
		// 01:00–17:00, not 1 PM to 29:00.
		if gs, ge := guessHalf(sh), guessHalf(eh); gs != 0 && ge != 0 {
			sap, eap = gs, ge
		}
	}
	var start, end int
	switch {
	case sap == 0 && eap == 0: // both 24-hour
		start, end = toMinutes(sh, sm, 0), toMinutes(eh, em, 0)
	case sap == 0: // only the end says AM/PM
		end = toMinutes(eh, em, eap)
		start = toMinutes(sh, sm, eap)
		if start >= end {
			start = toMinutes(sh, sm, otherHalf(eap))
		}
	case eap == 0: // only the start says AM/PM
		start = toMinutes(sh, sm, sap)
		end = toMinutes(eh, em, sap)
		if end <= start {
			end = toMinutes(eh, em, otherHalf(sap))
		}
	default:
		start, end = toMinutes(sh, sm, sap), toMinutes(eh, em, eap)
	}

	if end <= start {
		return Meeting{}, fmt.Errorf("end time %s is not after start time %s", FormatClock(end), FormatClock(start))
	}

	return Meeting{
		Days:     days,
		Start:    hhmm(start),
		End:      hhmm(end),
		Location: strings.TrimSpace(parts[4]),
	}, nil
}

var clockRe = regexp.MustCompile(`(?i)^(\d{1,2})(?::(\d{2}))?\s*(?:([ap])\.?m?\.?)?$`)

// parseClock returns hour, minute and 'a', 'p' or 0 when AM/PM is absent.
func parseClock(s string) (int, int, byte, error) {
	parts := clockRe.FindStringSubmatch(strings.TrimSpace(s))
	if parts == nil {
		return 0, 0, 0, fmt.Errorf("can't read time %q", s)
	}
	h, _ := strconv.Atoi(parts[1])
	m := 0
	if parts[2] != "" {
		m, _ = strconv.Atoi(parts[2])
	}
	var half byte
	if parts[3] != "" {
		half = strings.ToLower(parts[3])[0]
	}
	if m > 59 || h > 23 || (half != 0 && (h == 0 || h > 12)) {
		return 0, 0, 0, fmt.Errorf("can't read time %q", s)
	}
	return h, m, half, nil
}

func guessHalf(h int) byte {
	switch {
	case h == 0 || h >= 13:
		return 0 // already 24-hour
	case h >= 7 && h <= 11:
		return 'a'
	default:
		return 'p' // 12 through 6
	}
}

func otherHalf(half byte) byte {
	if half == 'a' {
		return 'p'
	}
	return 'a'
}

func toMinutes(h, m int, half byte) int {
	switch half {
	case 'a':
		if h == 12 {
			h = 0
		}
	case 'p':
		if h != 12 {
			h += 12
		}
	}
	return h*60 + m
}

var dayNames = map[string]string{
	"sun": "Sun", "sunday": "Sun", "su": "Sun",
	"mon": "Mon", "monday": "Mon", "mo": "Mon", "m": "Mon",
	"tue": "Tue", "tues": "Tue", "tuesday": "Tue", "tu": "Tue", "t": "Tue",
	"wed": "Wed", "wednesday": "Wed", "we": "Wed", "w": "Wed",
	"thu": "Thu", "thur": "Thu", "thurs": "Thu", "thursday": "Thu", "th": "Thu", "r": "Thu",
	"fri": "Fri", "friday": "Fri", "fr": "Fri", "f": "Fri",
	"sat": "Sat", "saturday": "Sat", "sa": "Sat",
}

func parseDays(s string) ([]string, error) {
	seen := map[string]bool{}
	tokens := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == ' ' || r == ',' || r == '/' || r == '&' || r == '+'
	})
	for _, tok := range tokens {
		if tok == "and" {
			continue
		}
		days, ok := dayToken(tok)
		if !ok {
			return nil, fmt.Errorf("unknown day %q", tok)
		}
		for _, d := range days {
			seen[d] = true
		}
	}

	var out []string
	for _, d := range dayOrder {
		if seen[d] {
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no days given")
	}
	return out, nil
}

// dayToken reads one word: a day name, or a compact run like "mwf" or "tth".
func dayToken(tok string) ([]string, bool) {
	if d, ok := dayNames[tok]; ok {
		return []string{d}, true
	}
	var out []string
	for i := 0; i < len(tok); {
		if i+1 < len(tok) {
			if d, ok := dayNames[tok[i:i+2]]; ok && len(tok[i:i+2]) == 2 {
				out = append(out, d)
				i += 2
				continue
			}
		}
		d, ok := dayNames[tok[i:i+1]]
		if !ok {
			return nil, false
		}
		out = append(out, d)
		i++
	}
	return out, len(out) > 0
}

// MeetingFromLegacy converts an old "Mon 12:34 PM" string (start time only)
// into a Meeting, assuming a 75-minute class.
func MeetingFromLegacy(s string) (Meeting, bool) {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return Meeting{}, false
	}
	days, ok := dayToken(strings.ToLower(fields[0]))
	if !ok {
		return Meeting{}, false
	}
	h, m, half, err := parseClock(strings.Join(fields[1:], " "))
	if err != nil {
		return Meeting{}, false
	}
	if half == 0 {
		half = guessHalf(h)
	}
	start := toMinutes(h, m, half)
	end := start + 75
	if end > 24*60-1 {
		end = 24*60 - 1
	}
	return Meeting{Days: days, Start: hhmm(start), End: hhmm(end)}, true
}

var shortCodeRe = regexp.MustCompile(`^([A-Za-z]{2,5})\s?(\d{2,4}[A-Za-z]{0,3})\b`)

// ShortCode trims a Canvas code like "COMP 440-16707-FA2026" to "COMP 440".
func ShortCode(code, name string) string {
	for _, s := range []string{code, name} {
		if parts := shortCodeRe.FindStringSubmatch(strings.TrimSpace(s)); parts != nil {
			return strings.ToUpper(parts[1]) + " " + strings.ToUpper(parts[2])
		}
	}
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return strings.TrimSpace(code)
}

// Conflict is two class meetings that overlap on the same weekday.
type Conflict struct {
	Day        string // "Tue"
	CourseA    int    // index into the courses slice
	CourseB    int
	MeetingA   Meeting
	MeetingB   Meeting
	Start, End int // the overlapping minutes
}

// MeetingConflicts finds every pair of meetings (across all courses, or
// within one) that overlap on a shared day. Back-to-back classes, where one
// ends as the next starts, don't count.
func MeetingConflicts(courses []Course) []Conflict {
	type slot struct {
		course     int
		meeting    Meeting
		start, end int
	}
	byDay := map[string][]slot{}
	for ci, c := range courses {
		for _, m := range c.Meetings {
			s, e, ok := m.Minutes()
			if !ok {
				continue
			}
			for _, d := range m.Days {
				byDay[d] = append(byDay[d], slot{ci, m, s, e})
			}
		}
	}

	var out []Conflict
	for _, day := range dayOrder {
		slots := byDay[day]
		for i := 0; i < len(slots); i++ {
			for j := i + 1; j < len(slots); j++ {
				a, b := slots[i], slots[j]
				start, end := max(a.start, b.start), min(a.end, b.end)
				if start < end {
					out = append(out, Conflict{
						Day: day, CourseA: a.course, CourseB: b.course,
						MeetingA: a.meeting, MeetingB: b.meeting,
						Start: start, End: end,
					})
				}
			}
		}
	}
	return out
}
