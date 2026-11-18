package model

import "time"

// Student info
type Student struct {
	Name   string `json:"name"`
	School string `json:"school"`
}

// Course data
type Course struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Term     string `json:"term,omitempty"`
	CanvasID int    `json:"canvas_id,omitempty"`
	Workflow string `json:"workflow_state,omitempty"`

	// from user input
	Location     string    `json:"location,omitempty"`
	Meetings     []Meeting `json:"meetings,omitempty"`
	MeetingTimes []string  `json:"meeting_times,omitempty"` // legacy "Mon 12:34 PM" strings, migrated to Meetings on load
	Notes        string    `json:"notes,omitempty"`
}

// Assignment data
type Assignment struct {
	ID             string     `json:"id"`
	CourseID       string     `json:"course_id"`
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	DueAt          *time.Time `json:"due_at,omitempty"`
	UnlockAt       *time.Time `json:"unlock_at,omitempty"`
	LockAt         *time.Time `json:"lock_at,omitempty"`
	HasSubmitted   bool       `json:"has_submitted"`
	IsMissing      bool       `json:"is_missing"`
	IsLate         bool       `json:"is_late"`
	PointsPossible float64    `json:"points_possible,omitempty"`
	CanvasID       int        `json:"canvas_id,omitempty"`

	// Manual assignments are added in VIO (not from Canvas). They can be
	// edited and deleted, and Canvas sync keeps them.
	Manual bool `json:"manual,omitempty"`
}

// Assignment states, from most to least urgent.
const (
	AssignmentMissing   = "missing"
	AssignmentLate      = "late"
	AssignmentOverdue   = "overdue"
	AssignmentOpen      = "open"
	AssignmentSubmitted = "submitted"
)

// State sums up Canvas's submission flags (and the due date) into one
// label. Canvas's own missing/late flags win; an unsubmitted assignment
// past its due date is "overdue" even when Canvas hasn't flagged it.
func (a Assignment) State(now time.Time) string {
	switch {
	case a.IsMissing:
		return AssignmentMissing
	case a.HasSubmitted && a.IsLate:
		return AssignmentLate
	case a.HasSubmitted:
		return AssignmentSubmitted
	case a.IsLate:
		return AssignmentLate
	case a.DueAt != nil && a.DueAt.Before(now):
		return AssignmentOverdue
	default:
		return AssignmentOpen
	}
}

// Done reports whether the assignment needs no more work.
func (a Assignment) Done() bool {
	return a.HasSubmitted
}

// NeedsAttention reports whether the assignment is missing or past due
// without a submission.
func (a Assignment) NeedsAttention(now time.Time) bool {
	switch a.State(now) {
	case AssignmentMissing, AssignmentOverdue:
		return true
	}
	return false
}

// Task data
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	CourseID    string     `json:"course_id,omitempty"`
	Status      string     `json:"status"`
	StartsAt    *time.Time `json:"starts_at,omitempty"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	Description string     `json:"description,omitempty"`
}

// Note data: free-form text written in the notes page. The first line
// of Body becomes Title. CourseID optionally links the note to a course.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CourseID  string    `json:"course_id,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Main source of info for pages to pull from
type AppData struct {
	Student     Student      `json:"student"`
	Courses     []Course     `json:"courses"`
	Assignments []Assignment `json:"assignments"`
	Tasks       []Task       `json:"tasks"`
	Notes       []Note       `json:"notes"`
}
