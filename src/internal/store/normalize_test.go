package store

import (
	"VIO/internal/model"
	"reflect"
	"testing"
	"time"
)

func ptr(t time.Time) *time.Time { return &t }

// futureDue is fixed per test run so two validData() calls compare equal.
var futureDue = time.Now().Add(48 * time.Hour).Round(time.Second)

// validData is already normal: normalizeAppData should leave it alone.
func validData() model.AppData {
	due := futureDue
	start := due.Add(-time.Hour)
	return model.AppData{
		Student: model.Student{Name: "Ada", School: "CSUN"},
		Courses: []model.Course{
			{ID: "c1", Name: "Databases", Code: "COMP 440"},
			{ID: "c2", Name: "Multimedia", Code: "COMP 467"},
		},
		Assignments: []model.Assignment{
			{ID: "a1", CourseID: "c1", Name: "HW 1", DueAt: ptr(due)},
		},
		Tasks: []model.Task{
			{ID: "t1", Title: "Read ch. 3", CourseID: "c1", Status: "in_progress", StartsAt: &start, DueAt: &due},
		},
		Notes: []model.Note{
			{ID: "n1", Title: "Ideas", CourseID: "c2"},
		},
	}
}

func TestNormalizeLeavesValidDataAlone(t *testing.T) {
	in := validData()
	out, changed := normalizeAppData(validData())
	if changed {
		t.Fatal("changed = true for already-valid data")
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("valid data was modified:\n got %+v\nwant %+v", out, in)
	}
}

func TestNormalizeFillsStudentDefaults(t *testing.T) {
	in := validData()
	in.Student = model.Student{Name: "  ", School: ""}
	out, changed := normalizeAppData(in)
	if !changed {
		t.Fatal("changed = false")
	}
	if out.Student.Name != "Student Name" || out.Student.School != "School" {
		t.Fatalf("student = %+v", out.Student)
	}
}

func TestNormalizeCourses(t *testing.T) {
	t.Run("empty list seeds sample courses", func(t *testing.T) {
		in := validData()
		in.Courses = nil
		out, changed := normalizeAppData(in)
		if !changed || len(out.Courses) != len(defaultAppData().Courses) {
			t.Fatalf("changed=%v courses=%d", changed, len(out.Courses))
		}
	})

	t.Run("duplicate IDs are made unique", func(t *testing.T) {
		in := validData()
		in.Courses = []model.Course{
			{ID: "dup", Name: "A", Code: "A 1"},
			{ID: "dup", Name: "B", Code: "B 2"},
		}
		out, changed := normalizeAppData(in)
		if !changed {
			t.Fatal("changed = false")
		}
		if out.Courses[0].ID == out.Courses[1].ID {
			t.Fatalf("IDs still collide: %q", out.Courses[0].ID)
		}
		if out.Courses[0].ID != "dup" {
			t.Fatalf("first course ID = %q, want it kept as dup", out.Courses[0].ID)
		}
	})

	t.Run("missing fields filled", func(t *testing.T) {
		in := validData()
		in.Courses = []model.Course{{}}
		out, _ := normalizeAppData(in)
		c := out.Courses[0]
		if c.ID == "" || c.Name == "" || c.Code == "" {
			t.Fatalf("course = %+v", c)
		}
	})
}

func TestNormalizeMigratesLegacyMeetingTimes(t *testing.T) {
	in := validData()
	in.Courses[0].MeetingTimes = []string{"Mon 12:30 PM", "whenever"}
	out, changed := normalizeAppData(in)
	if !changed {
		t.Fatal("changed = false")
	}
	c := out.Courses[0]
	if len(c.Meetings) != 1 {
		t.Fatalf("meetings = %+v, want one migrated meeting", c.Meetings)
	}
	m := c.Meetings[0]
	if !reflect.DeepEqual(m.Days, []string{"Mon"}) || m.Start != "12:30" {
		t.Fatalf("migrated meeting = %+v", m)
	}
	// Unreadable strings are kept so nothing is silently lost
	if !reflect.DeepEqual(c.MeetingTimes, []string{"whenever"}) {
		t.Fatalf("MeetingTimes = %v, want [whenever]", c.MeetingTimes)
	}

	// A second pass must not migrate the same meeting again
	again, _ := normalizeAppData(out)
	if len(again.Courses[0].Meetings) != 1 {
		t.Fatalf("second pass duplicated meetings: %+v", again.Courses[0].Meetings)
	}
}

func TestNormalizeAssignments(t *testing.T) {
	in := validData()
	in.Assignments = []model.Assignment{
		{ID: "a1", CourseID: "gone", Name: "Orphan", DueAt: ptr(time.Now())},
		{ID: "a1", CourseID: "c1", Name: "", DueAt: nil},
	}
	out, changed := normalizeAppData(in)
	if !changed {
		t.Fatal("changed = false")
	}
	valid := map[string]bool{"c1": true, "c2": true}
	if out.Assignments[0].ID == out.Assignments[1].ID {
		t.Fatal("duplicate assignment IDs kept")
	}
	for _, a := range out.Assignments {
		if a.Name == "" || a.DueAt == nil {
			t.Fatalf("assignment not filled in: %+v", a)
		}
		if !valid[a.CourseID] {
			// The fallback is the sample data's course, which may not exist
			// in the user's list; it must at least not be the dangling ID
			if a.CourseID == "gone" {
				t.Fatalf("dangling course ref kept: %+v", a)
			}
		}
	}
}

func TestNormalizeTaskStatus(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	cases := []struct {
		status string
		due    time.Time
		want   string
	}{
		{"done", future, "complete"},
		{"Completed", future, "complete"},
		{"todo", future, "in_progress"},
		{"not started", future, "in_progress"},
		{"", future, "in_progress"},
		{"something odd", future, "in_progress"},
		{"overdue", future, "overdue"},
		{"in_progress", past, "overdue"},
		{"todo", past, "overdue"},
		{"complete", past, "complete"}, // finished work is never marked overdue
	}
	for _, tc := range cases {
		in := validData()
		due := tc.due
		in.Tasks = []model.Task{{ID: "t", Title: "x", Status: tc.status, DueAt: &due}}
		out, _ := normalizeAppData(in)
		if got := out.Tasks[0].Status; got != tc.want {
			t.Errorf("status %q due %s: got %q, want %q", tc.status, tc.due.Format(time.Kitchen), got, tc.want)
		}
	}
}

func TestNormalizeTasks(t *testing.T) {
	due := time.Now().Add(48 * time.Hour)

	t.Run("start time defaults to an hour before due", func(t *testing.T) {
		in := validData()
		in.Tasks = []model.Task{{ID: "t", Title: "x", DueAt: &due}}
		out, _ := normalizeAppData(in)
		if s := out.Tasks[0].StartsAt; s == nil || !s.Equal(due.Add(-time.Hour)) {
			t.Fatalf("StartsAt = %v", s)
		}
	})

	t.Run("dangling course ref cleared", func(t *testing.T) {
		in := validData()
		in.Tasks[0].CourseID = "gone"
		out, changed := normalizeAppData(in)
		if !changed || out.Tasks[0].CourseID != "" {
			t.Fatalf("changed=%v CourseID=%q", changed, out.Tasks[0].CourseID)
		}
	})

	t.Run("duplicate IDs made unique", func(t *testing.T) {
		in := validData()
		in.Tasks = append(in.Tasks, in.Tasks[0])
		out, _ := normalizeAppData(in)
		if out.Tasks[0].ID == out.Tasks[1].ID {
			t.Fatal("duplicate task IDs kept")
		}
	})

	t.Run("nil list seeds samples, empty list stays empty", func(t *testing.T) {
		in := validData()
		in.Tasks = nil
		out, _ := normalizeAppData(in)
		if len(out.Tasks) == 0 {
			t.Fatal("nil task list was not seeded")
		}

		in = validData()
		in.Tasks = []model.Task{}
		out, _ = normalizeAppData(in)
		if len(out.Tasks) != 0 {
			t.Fatalf("deleting every task didn't stick: %d tasks", len(out.Tasks))
		}
	})
}

func TestNormalizeNotes(t *testing.T) {
	in := validData()
	in.Notes = []model.Note{
		{ID: "n1", Title: "", CourseID: "gone"},
		{ID: "n1", Title: "Second"},
		{ID: "", Title: "Third"},
	}
	out, changed := normalizeAppData(in)
	if !changed {
		t.Fatal("changed = false")
	}
	seen := map[string]bool{}
	for _, n := range out.Notes {
		if n.ID == "" || seen[n.ID] {
			t.Fatalf("note IDs not unique and non-empty: %+v", out.Notes)
		}
		seen[n.ID] = true
	}
	if out.Notes[0].Title != "Untitled" || out.Notes[0].CourseID != "" {
		t.Fatalf("first note = %+v", out.Notes[0])
	}

	// Notes are never seeded
	in = validData()
	in.Notes = nil
	out, _ = normalizeAppData(in)
	if out.Notes != nil {
		t.Fatalf("notes were seeded: %+v", out.Notes)
	}
}
