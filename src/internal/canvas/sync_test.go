package canvas

import (
	"VIO/internal/model"
	"VIO/internal/store"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestMergeCoursesKeepsUserFields(t *testing.T) {
	meetings := []model.Meeting{{Days: []string{"Tue", "Thu"}, Start: "10:00", End: "11:15"}}
	existing := []model.Course{
		{ID: "canvas-course-1", CanvasID: 1, Name: "Old name", Location: "JD 1600", Meetings: meetings, Notes: "bring laptop"},
		{ID: "canvas-course-2", CanvasID: 2, Name: "Dropped course", Location: "JD 2000"},
	}
	fresh := []model.Course{
		{ID: "canvas-course-1", CanvasID: 1, Name: "New name", Code: "COMP 440", Term: "Fall 2026"},
		{ID: "canvas-course-3", CanvasID: 3, Name: "Brand new"},
	}

	got := mergeCourses(existing, fresh)

	if len(got) != 2 {
		t.Fatalf("got %d courses, want 2 (Canvas's list decides which courses exist)", len(got))
	}
	c := got[0]
	if c.Name != "New name" || c.Code != "COMP 440" || c.Term != "Fall 2026" {
		t.Fatalf("Canvas fields not updated: %+v", c)
	}
	if c.Location != "JD 1600" || c.Notes != "bring laptop" || !reflect.DeepEqual(c.Meetings, meetings) {
		t.Fatalf("user fields lost: %+v", c)
	}
	if got[1].Location != "" || got[1].Meetings != nil {
		t.Fatalf("new course picked up someone else's fields: %+v", got[1])
	}
}

func TestMergeCoursesKeepsLegacyMeetingTimes(t *testing.T) {
	existing := []model.Course{{CanvasID: 7, MeetingTimes: []string{"whenever"}}}
	got := MergeCourses(existing, []model.Course{{CanvasID: 7}})
	if !reflect.DeepEqual(got[0].MeetingTimes, []string{"whenever"}) {
		t.Fatalf("MeetingTimes = %v", got[0].MeetingTimes)
	}
}

func TestNextLink(t *testing.T) {
	cases := map[string]string{
		"": "",
		`<https://x/api/v1/courses?page=2>; rel="next", <https://x/api/v1/courses?page=1>; rel="first"`:   "https://x/api/v1/courses?page=2",
		`<https://x/api/v1/courses?page=1>; rel="current", <https://x/api/v1/courses?page=3>; rel="last"`: "",
		`<https://x/a?page=1>; rel="current",<https://x/a?page=2>; rel="next"`:                            "https://x/a?page=2",
	}
	for header, want := range cases {
		if got := nextLink(header); got != want {
			t.Errorf("nextLink(%q) = %q, want %q", header, got, want)
		}
	}
}

// fakeCanvas serves JSON pages for a path, linking each page to the next
// with a Link header the way Canvas does.
type fakeCanvas struct {
	t      *testing.T
	pages  map[string][]string // path -> JSON body of each page
	status int                 // non-zero: reply with this status instead

	mu   sync.Mutex
	hits []string
}

func (f *fakeCanvas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.hits = append(f.hits, r.URL.RequestURI())
	f.mu.Unlock()

	if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
		f.t.Errorf("Authorization = %q", got)
	}
	if f.status != 0 {
		http.Error(w, `{"errors":[{"message":"Invalid access token."}]}`, f.status)
		return
	}

	pages, ok := f.pages[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	page := 1
	fmt.Sscanf(r.URL.Query().Get("page"), "%d", &page)
	if page < 1 || page > len(pages) {
		http.Error(w, "bad page", http.StatusBadRequest)
		return
	}
	if page < len(pages) {
		q := r.URL.Query()
		q.Set("page", fmt.Sprint(page+1))
		next := fmt.Sprintf("http://%s%s?%s", r.Host, r.URL.Path, q.Encode())
		w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="current", <%s>; rel="next"`, r.URL.String(), next))
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, pages[page-1])
}

func newFake(t *testing.T, pages map[string][]string) (*fakeCanvas, store.CanvasConfig) {
	t.Helper()
	fake := &fakeCanvas{t: t, pages: pages}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	return fake, store.CanvasConfig{Domain: srv.URL, Token: "test-token", PollMinutes: 5}
}

func TestFetchActiveCoursesFollowsPagination(t *testing.T) {
	fake, cfg := newFake(t, map[string][]string{
		"/api/v1/courses": {
			`[{"id": 1, "name": "Databases", "course_code": " COMP 440 ", "workflow_state": "available", "term": {"name": "Fall 2026"}}]`,
			`[{"id": 2, "name": "Multimedia", "course_code": "COMP 467"}]`,
			`[{"id": 3, "name": "Geology", "course_code": "GEOL 102"}]`,
		},
	})
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	courses, err := client.FetchActiveCourses()
	if err != nil {
		t.Fatal(err)
	}

	if len(courses) != 3 {
		t.Fatalf("got %d courses, want 3 across 3 pages (hits: %v)", len(courses), fake.hits)
	}
	if len(fake.hits) != 3 {
		t.Fatalf("made %d requests, want 3: %v", len(fake.hits), fake.hits)
	}
	first := courses[0]
	if first.ID != "canvas-course-1" || first.Code != "COMP 440" || first.Term != "Fall 2026" || first.Workflow != "available" {
		t.Fatalf("course not mapped: %+v", first)
	}
	if !strings.Contains(fake.hits[0], "enrollment_state=active") || !strings.Contains(fake.hits[0], "per_page=100") {
		t.Fatalf("first request missing query params: %s", fake.hits[0])
	}
}

func TestFetchAssignmentsMapsSubmissionFlags(t *testing.T) {
	_, cfg := newFake(t, map[string][]string{
		"/api/v1/courses/42/assignments": {
			`[{"id": 10, "name": "HW 1", "due_at": "2026-11-20T07:59:00Z", "points_possible": 100,
			   "submission": {"submitted_at": "2026-11-19T20:00:00Z", "late": false, "missing": false}}]`,
			`[{"id": 11, "name": "   ", "due_at": null},
			  {"id": 12, "name": "Lab 2", "due_at": null, "submission": {"submitted_at": null, "late": false, "missing": true}}]`,
		},
	})
	client, _ := NewClient(cfg)

	got, err := client.FetchAssignments(42)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d assignments, want 2 (blank names skipped): %+v", len(got), got)
	}
	hw, lab := got[0], got[1]
	if hw.ID != "canvas-assignment-10" || hw.CourseID != "canvas-course-42" || hw.CanvasID != 10 {
		t.Fatalf("IDs not mapped: %+v", hw)
	}
	if !hw.HasSubmitted || hw.IsMissing || hw.PointsPossible != 100 || hw.DueAt == nil {
		t.Fatalf("HW 1 flags wrong: %+v", hw)
	}
	if lab.HasSubmitted || !lab.IsMissing || lab.DueAt != nil {
		t.Fatalf("Lab 2 flags wrong: %+v", lab)
	}
}

func TestSyncCoursesAndAssignments(t *testing.T) {
	_, cfg := newFake(t, map[string][]string{
		"/api/v1/courses": {
			`[{"id": 1, "name": "Databases", "course_code": "COMP 440"}]`,
			`[{"id": 2, "name": "Multimedia", "course_code": "COMP 467"}]`,
		},
		"/api/v1/courses/1/assignments": {`[{"id": 10, "name": "HW 1"}]`},
		"/api/v1/courses/2/assignments": {`[{"id": 20, "name": "Project 3"}]`, `[{"id": 21, "name": "Project 4"}]`},
	})
	current := model.AppData{
		Courses:     []model.Course{{ID: "canvas-course-1", CanvasID: 1, Location: "JD 1600"}},
		Assignments: []model.Assignment{{ID: "stale", Name: "Old"}},
		Tasks:       []model.Task{{ID: "t1", Title: "Keep me"}},
	}

	got, err := SyncCoursesAndAssignments(cfg, current)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Courses) != 2 || got.Courses[0].Location != "JD 1600" {
		t.Fatalf("courses = %+v", got.Courses)
	}
	var names []string
	for _, a := range got.Assignments {
		names = append(names, a.Name)
	}
	if want := []string{"HW 1", "Project 3", "Project 4"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("assignments = %v, want %v (Canvas's list replaces the old one)", names, want)
	}
	if len(got.Tasks) != 1 || got.Tasks[0].ID != "t1" {
		t.Fatalf("sync touched tasks: %+v", got.Tasks)
	}
}

func TestSyncErrorLeavesDataUnchanged(t *testing.T) {
	fake, cfg := newFake(t, nil)
	fake.status = http.StatusUnauthorized
	current := model.AppData{Courses: []model.Course{{ID: "keep", CanvasID: 1}}}

	got, err := SyncCoursesAndAssignments(cfg, current)
	if err == nil {
		t.Fatal("expected an error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("error should mention the status: %v", err)
	}
	if !reflect.DeepEqual(got, current) {
		t.Fatalf("data changed on error: %+v", got)
	}
}

func TestNewClient(t *testing.T) {
	cases := []struct {
		domain, token, wantBase string
		wantErr                 bool
	}{
		{"myschool.instructure.com", "tok", "https://myschool.instructure.com", false},
		{"https://myschool.instructure.com/", "tok", "https://myschool.instructure.com", false},
		{"  ", "tok", "", true},
		{"myschool.instructure.com", " ", "", true},
	}
	for _, tc := range cases {
		c, err := NewClient(store.CanvasConfig{Domain: tc.domain, Token: tc.token})
		if (err != nil) != tc.wantErr {
			t.Errorf("NewClient(%q, %q) err = %v, wantErr %v", tc.domain, tc.token, err, tc.wantErr)
			continue
		}
		if err == nil && c.baseURL != tc.wantBase {
			t.Errorf("NewClient(%q) base = %q, want %q", tc.domain, c.baseURL, tc.wantBase)
		}
	}
}
