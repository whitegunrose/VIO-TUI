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
	Location     string   `json:"location,omitempty"`
	MeetingTimes []string `json:"meeting_times,omitempty"`
	Notes        string   `json:"notes,omitempty"`
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
