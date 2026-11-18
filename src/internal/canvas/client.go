package canvas

import (
	"VIO/internal/model"
	"VIO/internal/store"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(cfg store.CanvasConfig) (*Client, error) {
	base := strings.TrimSpace(cfg.Domain)
	if base == "" || strings.TrimSpace(cfg.Token) == "" {
		return nil, fmt.Errorf("canvas config is incomplete")
	}

	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	base = strings.TrimRight(base, "/")

	if _, err := url.Parse(base); err != nil {
		return nil, fmt.Errorf("invalid canvas domain: %w", err)
	}

	return &Client{
		baseURL: base,
		token:   cfg.Token,
		http:    &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func SyncCoursesAndAssignments(cfg store.CanvasConfig, current model.AppData) (model.AppData, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return current, err
	}

	courses, err := client.FetchActiveCourses()
	if err != nil {
		return current, err
	}

	assignments := make([]model.Assignment, 0)
	for _, course := range courses {
		courseAssignments, err := client.FetchAssignments(course.CanvasID)
		if err != nil {
			return current, err
		}
		assignments = append(assignments, courseAssignments...)
	}

	current.Courses = mergeCourses(current.Courses, courses)
	current.Assignments = assignments
	return current, nil
}

type apiCourse struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	CourseCode    string  `json:"course_code"`
	WorkflowState string  `json:"workflow_state"`
	Term          apiTerm `json:"term"`
}

type apiTerm struct {
	Name string `json:"name"`
}

type apiAssignment struct {
	ID             int           `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	DueAt          *time.Time    `json:"due_at"`
	UnlockAt       *time.Time    `json:"unlock_at"`
	LockAt         *time.Time    `json:"lock_at"`
	PointsPossible float64       `json:"points_possible"`
	Submission     apiSubmission `json:"submission"`
}

type apiSubmission struct {
	SubmittedAt *time.Time `json:"submitted_at"`
	Late        bool       `json:"late"`
	Missing     bool       `json:"missing"`
}

func (c *Client) FetchActiveCourses() ([]model.Course, error) {
	query := url.Values{}
	query.Set("enrollment_state", "active")
	query.Add("state[]", "available")
	query.Add("include[]", "term")
	query.Set("per_page", "100")

	payload, err := getPaginatedJSON[apiCourse](c, "/api/v1/courses", query)
	if err != nil {
		return nil, err
	}

	courses := make([]model.Course, 0, len(payload))
	for _, item := range payload {
		courses = append(courses, model.Course{
			ID:       canvasCourseKey(item.ID),
			Name:     item.Name,
			Code:     strings.TrimSpace(item.CourseCode),
			Term:     item.Term.Name,
			CanvasID: item.ID,
			Workflow: item.WorkflowState,
		})
	}

	return courses, nil
}

func (c *Client) FetchAssignments(courseCanvasID int) ([]model.Assignment, error) {
	query := url.Values{}
	query.Add("include[]", "submission")
	query.Set("per_page", "100")
	query.Set("order_by", "due_at")

	path := fmt.Sprintf("/api/v1/courses/%d/assignments", courseCanvasID)
	payload, err := getPaginatedJSON[apiAssignment](c, path, query)
	if err != nil {
		return nil, err
	}

	courseID := canvasCourseKey(courseCanvasID)
	assignments := make([]model.Assignment, 0, len(payload))
	for _, item := range payload {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}

		assignments = append(assignments, model.Assignment{
			ID:             canvasAssignmentKey(item.ID),
			CourseID:       courseID,
			Name:           name,
			Description:    item.Description,
			DueAt:          toLocalTimePtr(item.DueAt),
			UnlockAt:       toLocalTimePtr(item.UnlockAt),
			LockAt:         toLocalTimePtr(item.LockAt),
			HasSubmitted:   item.Submission.SubmittedAt != nil,
			IsMissing:      item.Submission.Missing,
			IsLate:         item.Submission.Late,
			PointsPossible: item.PointsPossible,
			CanvasID:       item.ID,
		})
	}

	return assignments, nil
}

func getPaginatedJSON[T any](client *Client, path string, query url.Values) ([]T, error) {
	urlStr := client.baseURL + path
	if len(query) > 0 {
		urlStr += "?" + query.Encode()
	}

	var all []T

	for urlStr != "" {
		req, err := http.NewRequest(http.MethodGet, urlStr, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+client.token)
		req.Header.Set("Accept", "application/json")

		resp, err := client.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("canvas request failed: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("canvas request returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}

		var page []T
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decode canvas response: %w", err)
		}

		all = append(all, page...)
		urlStr = nextLink(resp.Header.Get("Link"))
	}

	return all, nil
}

func canvasCourseKey(id int) string {
	return "canvas-course-" + strconv.Itoa(id)
}

func canvasAssignmentKey(id int) string {
	return "canvas-assignment-" + strconv.Itoa(id)
}

// MergeCourses keeps the user-entered fields (location, meetings, notes)
// from existing on matching courses in fresh. main.go uses it to reapply
// edits made while a sync was in flight.
func MergeCourses(existing []model.Course, fresh []model.Course) []model.Course {
	return mergeCourses(existing, fresh)
}

// Replace Canvas-relevent fields but not user-made ones
func mergeCourses(existing []model.Course, fresh []model.Course) []model.Course {
	existingByCanvasID := make(map[int]model.Course, len(existing))
	for _, course := range existing {
		existingByCanvasID[course.CanvasID] = course
	}

	merged := make([]model.Course, 0, len(fresh))
	for _, course := range fresh {
		if old, ok := existingByCanvasID[course.CanvasID]; ok {
			course.Location = old.Location
			course.Meetings = old.Meetings
			course.MeetingTimes = old.MeetingTimes
			course.Notes = old.Notes
		}
		merged = append(merged, course)
	}

	return merged
}

func nextLink(linkHeader string) string {
	for _, part := range strings.Split(linkHeader, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}

func toLocalTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	local := t.In(time.Local)
	return &local
}

// KeepManual returns the synced Canvas assignments plus every manual
// assignment from local, since sync replaces the assignment list.
func KeepManual(synced, local []model.Assignment) []model.Assignment {
	out := make([]model.Assignment, 0, len(synced)+len(local))
	for _, a := range synced {
		if !a.Manual {
			out = append(out, a)
		}
	}
	for _, a := range local {
		if a.Manual {
			out = append(out, a)
		}
	}
	return out
}
