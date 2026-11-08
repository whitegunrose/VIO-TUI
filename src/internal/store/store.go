package store

import (
	"VIO/internal/model"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const appFolderName = "vio"
const dataFileName = "data.json"

// Load user's local data file or create new file
func LoadAppData() (*model.AppData, error) {
	path, err := dataFilePath()
	if err != nil {
		return nil, err
	}

	if err := ensureDataFile(path); err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read data file: %w", err)
	}

	var data model.AppData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("decode data file: %w", err)
	}

	normalized, changed := normalizeAppData(data)
	if changed {
		if err := SaveAppData(normalized); err != nil {
			return nil, err
		}
	}

	return &normalized, nil
}

// Write modified JSON data
func SaveAppData(data model.AppData) error {
	path, err := dataFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create app config directory: %w", err)
	}

	normalized, _ := normalizeAppData(data)
	jsonBytes, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("encode data file: %w", err)
	}

	if err := os.WriteFile(path, jsonBytes, 0o644); err != nil {
		return fmt.Errorf("write data file: %w", err)
	}

	return nil
}

// Build config path
func dataFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config dir: %w", err)
	}

	return filepath.Join(configDir, appFolderName, dataFileName), nil
}

// Ensure app folder and starter JSON both exist
func ensureDataFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create app config directory: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check data file: %w", err)
	}

	starter := defaultAppData()
	jsonBytes, err := json.MarshalIndent(starter, "", "  ")
	if err != nil {
		return fmt.Errorf("encode starter data: %w", err)
	}

	if err := os.WriteFile(path, jsonBytes, 0o644); err != nil {
		return fmt.Errorf("write starter data file: %w", err)
	}

	return nil
}

// Create starter data for UI
func defaultAppData() model.AppData {
	now := time.Now()
	assignmentDueSoon := now.Add(48 * time.Hour)
	assignmentDueLater := now.Add(5 * 24 * time.Hour)
	taskStart := now.Add(12 * time.Hour)
	taskDue := now.Add(24 * time.Hour)
	taskDueLater := now.Add(72 * time.Hour)

	return model.AppData{
		Student: model.Student{
			Name:   "Student Name",
			School: "School",
		},
		Courses: []model.Course{
			{
				ID:           "comp123",
				Name:         "Introduction to Computer",
				Code:         "COMP 123",
				Term:         "Spring 2032",
				CanvasID:     12345,
				Workflow:     "available",
				Location:     "JD 1234",
				MeetingTimes: []string{"Mon 12:34 PM", "Wed 12:34 PM"},
			},
			{
				ID:           "comp456",
				Name:         "Analysis of Computer",
				Code:         "COMP 456",
				Term:         "Spring 2032",
				CanvasID:     67890,
				Workflow:     "available",
				Location:     "JD 5678",
				MeetingTimes: []string{"Tue 10:23 AM", "Thu 10:23 AM"},
			},
		},
		Assignments: []model.Assignment{
			{
				ID:             "assign-1",
				CourseID:       "comp123",
				Name:           "Homework 1",
				DueAt:          &assignmentDueSoon,
				HasSubmitted:   false,
				IsMissing:      false,
				IsLate:         false,
				PointsPossible: 100,
				CanvasID:       000000,
			},
			{
				ID:             "assign-2",
				CourseID:       "comp456",
				Name:           "Homework 2",
				DueAt:          &assignmentDueLater,
				HasSubmitted:   false,
				IsMissing:      false,
				IsLate:         false,
				PointsPossible: 50,
				CanvasID:       000001,
			},
		},
		Tasks: []model.Task{
			{
				ID:          "task-1",
				Title:       "Review lecture notes",
				CourseID:    "comp123",
				Status:      "in_progress",
				StartsAt:    &taskStart,
				DueAt:       &taskDue,
				Description: "Prep before lecture.",
			},
			{
				ID:          "task-2",
				Title:       "Review lecture notes",
				CourseID:    "comp456",
				Status:      "todo",
				DueAt:       &taskDueLater,
				Description: "Prep before lecture.",
			},
		},
	}
}

func normalizeAppData(data model.AppData) (model.AppData, bool) {
	defaults := defaultAppData()
	changed := false
	now := time.Now()

	if strings.TrimSpace(data.Student.Name) == "" {
		data.Student.Name = defaults.Student.Name
		changed = true
	}
	if strings.TrimSpace(data.Student.School) == "" {
		data.Student.School = defaults.Student.School
		changed = true
	}

	courseSeen := make(map[string]bool)
	normalizedCourses := make([]model.Course, 0, len(data.Courses))
	for i, course := range data.Courses {
		def := defaults.Courses[i%len(defaults.Courses)]
		if strings.TrimSpace(course.ID) == "" {
			course.ID = def.ID
			changed = true
		}
		if strings.TrimSpace(course.Name) == "" {
			course.Name = def.Name
			changed = true
		}
		if strings.TrimSpace(course.Code) == "" {
			course.Code = def.Code
			changed = true
		}
		if courseSeen[course.ID] {
			course.ID = fmt.Sprintf("%s-%d", def.ID, i+1)
			changed = true
		}
		courseSeen[course.ID] = true
		normalizedCourses = append(normalizedCourses, course)
	}
	if len(normalizedCourses) == 0 {
		normalizedCourses = defaults.Courses
		changed = true
	}
	data.Courses = normalizedCourses

	courseValid := make(map[string]bool, len(data.Courses))
	for _, c := range data.Courses {
		courseValid[c.ID] = true
	}

	normalizedAssignments := make([]model.Assignment, 0, len(data.Assignments))
	assignSeen := make(map[string]bool)
	for i, assignment := range data.Assignments {
		def := defaults.Assignments[i%len(defaults.Assignments)]
		if strings.TrimSpace(assignment.ID) == "" {
			assignment.ID = def.ID
			changed = true
		}
		if strings.TrimSpace(assignment.CourseID) == "" || !courseValid[assignment.CourseID] {
			assignment.CourseID = def.CourseID
			changed = true
		}
		if strings.TrimSpace(assignment.Name) == "" {
			assignment.Name = def.Name
			changed = true
		}
		if assignment.DueAt == nil {
			assignment.DueAt = def.DueAt
			changed = true
		}
		if assignSeen[assignment.ID] {
			assignment.ID = fmt.Sprintf("%s-%d", def.ID, i+1)
			changed = true
		}
		assignSeen[assignment.ID] = true
		normalizedAssignments = append(normalizedAssignments, assignment)
	}
	if len(normalizedAssignments) == 0 {
		normalizedAssignments = defaults.Assignments
		changed = true
	}
	data.Assignments = normalizedAssignments

	normalizedTasks := make([]model.Task, 0, len(data.Tasks))
	taskSeen := make(map[string]bool)
	for i, task := range data.Tasks {
		def := defaults.Tasks[i%len(defaults.Tasks)]
		if strings.TrimSpace(task.ID) == "" {
			task.ID = def.ID
			changed = true
		}
		if taskSeen[task.ID] {
			task.ID = fmt.Sprintf("%s-%d", def.ID, i+1)
			changed = true
		}
		taskSeen[task.ID] = true
		if strings.TrimSpace(task.Title) == "" {
			task.Title = def.Title
			changed = true
		}
		if strings.TrimSpace(task.CourseID) != "" && !courseValid[task.CourseID] {
			task.CourseID = ""
			changed = true
		}
		if task.DueAt == nil {
			task.DueAt = def.DueAt
			changed = true
		}
		if task.StartsAt == nil && task.DueAt != nil {
			start := task.DueAt.Add(-1 * time.Hour)
			task.StartsAt = &start
			changed = true
		}

		status := normalizeTaskStatus(task.Status)
		if status != task.Status {
			task.Status = status
			changed = true
		}
		if task.Status != "complete" && task.DueAt != nil && task.DueAt.Before(now) {
			task.Status = "overdue"
			changed = true
		}
		normalizedTasks = append(normalizedTasks, task)
	}
	// Only seed sample tasks when the list is missing, so deleting every task sticks
	if data.Tasks == nil {
		normalizedTasks = defaults.Tasks
		changed = true
	}
	data.Tasks = normalizedTasks

	return data, changed
}

func normalizeTaskStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "complete", "completed", "done":
		return "complete"
	case "overdue":
		return "overdue"
	case "in_progress", "in progress", "todo", "to_do", "not_started", "not started", "open", "pending", "":
		return "in_progress"
	default:
		return "in_progress"
	}
}
