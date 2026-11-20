package store

import (
	"VIO/internal/model"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Fuzz tests check laws: rules that must hold for every input. See
// aidlc-docs/design-artifacts/testing_strategy.md. To search for new failing
// inputs: go test -run '^$' -fuzz FuzzNormalize -fuzztime 30s ./internal/store

func fuzzSeeds(f *testing.F) {
	for _, d := range []model.AppData{defaultAppData(), validData()} {
		b, _ := json.Marshal(d)
		f.Add(b)
	}
	for _, s := range []string{
		`{}`,
		`{"tasks": []}`,
		`{"courses": [{"id": "a"}, {"id": "a"}], "tasks": [{"id": "t", "course_id": "gone", "status": "Done"}]}`,
		`{"courses": [{"id": "c", "meeting_times": ["Mon 12:34 PM", "whenever"]}]}`,
		`{"notes": [{"id": ""}, {"id": ""}, {"title": "x", "course_id": "nope"}]}`,
		`{"assignments": [{"id": "x", "course_id": ""}, {"id": "x"}]}`,
		`{"courses": [{"id": "c"}], "assignments": [{"id": "r", "course_id": "c", "name": "Reading", "due_at": null}], "tasks": []}`,
		`{"courses": [{"id": "comp123-2"}, {"id": "comp123"}, {"id": "comp123"}]}`,
		`{"student": {"name": "x"`,
		``,
		// regression, found by fuzzing: an assignment's fallback course didn't exist
		`{"Courses":[{}],"Assignments":[{},{}],"tAsks":[]}`,
		// regression, found by fuzzing: an already-overdue task reported a change on every pass
		`{"tasks": [{"id": "t", "title": "late", "status": "overdue", "due_at": "2020-01-01T00:00:00Z"}]}`,
	} {
		f.Add([]byte(s))
	}
}

// Law S1: normalizing is idempotent. Cleaning already-clean data changes
// nothing and reports no change, so a clean file is never rewritten on load.
// Law S4: an assignment with no due date stays undated, and one with a due
// date keeps it.
// Law S2: after normalizing, IDs are unique within courses, assignments,
// tasks and notes, and every course reference points at a real course.
func FuzzNormalize(f *testing.F) {
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, raw []byte) {
		var in model.AppData
		if json.Unmarshal(raw, &in) != nil {
			return
		}
		once, _ := normalizeAppData(in)
		if len(in.Assignments) == len(once.Assignments) {
			for i := range in.Assignments {
				if (in.Assignments[i].DueAt == nil) != (once.Assignments[i].DueAt == nil) {
					t.Fatalf("assignment %d: due date presence changed (undated assignments must stay undated)", i)
				}
			}
		}
		twice, changed := normalizeAppData(once)
		if changed || !reflect.DeepEqual(once, twice) {
			t.Fatalf("second normalize changed the data (changed=%v)\nonce:  %+v\ntwice: %+v", changed, once, twice)
		}

		unique := func(kind string, ids []string) {
			seen := map[string]bool{}
			for _, id := range ids {
				if seen[id] {
					t.Fatalf("duplicate %s ID %q after normalize", kind, id)
				}
				seen[id] = true
			}
		}
		var courses, assignments, tasks, notes []string
		valid := map[string]bool{}
		for _, c := range once.Courses {
			courses = append(courses, c.ID)
			valid[c.ID] = true
		}
		for _, a := range once.Assignments {
			assignments = append(assignments, a.ID)
			if !valid[a.CourseID] {
				t.Fatalf("assignment %q points at missing course %q", a.ID, a.CourseID)
			}
		}
		for _, x := range once.Tasks {
			tasks = append(tasks, x.ID)
			if x.CourseID != "" && !valid[x.CourseID] {
				t.Fatalf("task %q points at missing course %q", x.ID, x.CourseID)
			}
		}
		for _, n := range once.Notes {
			notes = append(notes, n.ID)
		}
		unique("course", courses)
		unique("assignment", assignments)
		unique("task", tasks)
		unique("note", notes)
	})
}

// Law S3: loading any bytes never panics. If the file can't be decoded, the
// load reports a CorruptDataError and leaves the file exactly as it was. If
// it can, saving what was loaded and loading again gives the same data.
func FuzzLoad(f *testing.F) {
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, raw []byte) {
		path := filepath.Join(t.TempDir(), "vio", dataFileName)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}

		data, err := loadAppDataFrom(path)
		if err != nil {
			var corrupt *CorruptDataError
			if !errors.As(err, &corrupt) {
				t.Fatalf("load error is not a CorruptDataError: %v", err)
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(after, raw) {
				t.Fatalf("an unreadable file was modified by load")
			}
			return
		}

		if err := saveAppDataTo(path, *data); err != nil {
			t.Fatalf("save after load: %v", err)
		}
		again, err := loadAppDataFrom(path)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		a, _ := json.Marshal(data)
		b, _ := json.Marshal(again)
		if !bytes.Equal(a, b) {
			t.Fatalf("save + load changed the data\nfirst:  %s\nsecond: %s", a, b)
		}
	})
}
