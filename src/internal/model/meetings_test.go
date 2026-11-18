package model

import (
	"reflect"
	"testing"
)

func TestParseMeeting(t *testing.T) {
	tests := []struct {
		in   string
		want Meeting
	}{
		{"Tue Thu 10:00 AM - 11:15 AM @ JD 1234", Meeting{Days: []string{"Tue", "Thu"}, Start: "10:00", End: "11:15", Location: "JD 1234"}},
		{"MW 2:30-3:45pm", Meeting{Days: []string{"Mon", "Wed"}, Start: "14:30", End: "15:45"}},
		{"TTh 11:00-12:15 PM", Meeting{Days: []string{"Tue", "Thu"}, Start: "11:00", End: "12:15"}},
		{"Thursday 5:30 PM to 8:15 PM", Meeting{Days: []string{"Thu"}, Start: "17:30", End: "20:15"}},
		{"Wed 7:00-9:45pm", Meeting{Days: []string{"Wed"}, Start: "19:00", End: "21:45"}},
		{"mwf 9-9:50", Meeting{Days: []string{"Mon", "Wed", "Fri"}, Start: "09:00", End: "09:50"}},
		{"TR 13:00 - 14:15 @ Sierra Hall 101", Meeting{Days: []string{"Tue", "Thu"}, Start: "13:00", End: "14:15", Location: "Sierra Hall 101"}},
		{"Mon, Wed 10am – 11:15am", Meeting{Days: []string{"Mon", "Wed"}, Start: "10:00", End: "11:15"}},
	}
	for _, tt := range tests {
		got, err := ParseMeeting(tt.in)
		if err != nil {
			t.Errorf("ParseMeeting(%q) error: %v", tt.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ParseMeeting(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
		// Round trip through the editor format
		again, err := ParseMeeting(got.String())
		if err != nil || !reflect.DeepEqual(again, got) {
			t.Errorf("round trip of %q via %q gave %+v, %v", tt.in, got.String(), again, err)
		}
	}
}

func TestParseMeetingErrors(t *testing.T) {
	for _, in := range []string{"", "10:00-11:00", "Funday 10-11", "Mon 11:00 AM - 10:00 AM", "Mon 25:00-26:00"} {
		if _, err := ParseMeeting(in); err == nil {
			t.Errorf("ParseMeeting(%q) should fail", in)
		}
	}
}

func TestMeetingFromLegacy(t *testing.T) {
	got, ok := MeetingFromLegacy("Mon 12:34 PM")
	want := Meeting{Days: []string{"Mon"}, Start: "12:34", End: "13:49"}
	if !ok || !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, %v", got, ok)
	}
}

func TestShortCode(t *testing.T) {
	cases := map[[2]string]string{
		{"COMP 440-16707-FA2026", "COMP 440 DATABASE DESIGN"}:   "COMP 440",
		{"GEOL 110-17110-FA2026", ""}:                           "GEOL 110",
		{"PHYS 220AL", ""}:                                      "PHYS 220AL",
		{"26 to 27 RSO Officer Orientation", "RSO Orientation"}: "RSO Orientation",
	}
	for in, want := range cases {
		if got := ShortCode(in[0], in[1]); got != want {
			t.Errorf("ShortCode(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}
