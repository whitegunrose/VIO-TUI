package model

import (
	"testing"
	"time"
)

func TestMeetingConflicts(t *testing.T) {
	courses := []Course{
		{Code: "COMP 440", Meetings: []Meeting{{Days: []string{"Tue", "Thu"}, Start: "10:00", End: "11:15"}}},
		{Code: "COMP 467", Meetings: []Meeting{{Days: []string{"Thu"}, Start: "11:00", End: "12:15"}}},
		{Code: "GEOL 102", Meetings: []Meeting{{Days: []string{"Tue"}, Start: "11:15", End: "12:30"}}}, // back-to-back, not a conflict
	}
	got := MeetingConflicts(courses)
	if len(got) != 1 {
		t.Fatalf("got %d conflicts, want 1: %+v", len(got), got)
	}
	c := got[0]
	if c.Day != "Thu" || c.CourseA != 0 || c.CourseB != 1 || c.Start != 11*60 || c.End != 11*60+15 {
		t.Errorf("unexpected conflict %+v", c)
	}
}

func TestAssignmentState(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local)
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	cases := []struct {
		a    Assignment
		want string
	}{
		{Assignment{DueAt: &future}, AssignmentOpen},
		{Assignment{DueAt: &past}, AssignmentOverdue},
		{Assignment{DueAt: &past, IsMissing: true}, AssignmentMissing},
		{Assignment{DueAt: &past, HasSubmitted: true}, AssignmentSubmitted},
		{Assignment{DueAt: &past, HasSubmitted: true, IsLate: true}, AssignmentLate},
	}
	for _, c := range cases {
		if got := c.a.State(now); got != c.want {
			t.Errorf("State(%+v) = %q, want %q", c.a, got, c.want)
		}
	}
	if !(Assignment{DueAt: &past}).NeedsAttention(now) {
		t.Error("overdue unsubmitted assignment should need attention")
	}
	if (Assignment{DueAt: &past, HasSubmitted: true}).NeedsAttention(now) {
		t.Error("submitted assignment should not need attention")
	}
}
