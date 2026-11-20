package store

import (
	"VIO/internal/model"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const appFolderName = "vio"
const dataFileName = "data.json"
const backupSuffix = ".bak"

// CorruptDataError means data.json exists but can't be decoded, usually
// because a crash or a hand edit left it empty or half-written. VIO offers
// to restore the backup instead of crashing.
type CorruptDataError struct {
	Path       string    // data.json
	BackupPath string    // data.json.bak
	HasBackup  bool      // the backup exists and decodes
	BackupTime time.Time // when the backup was written, if HasBackup
	Err        error     // the decode error
}

func (e *CorruptDataError) Error() string {
	return fmt.Sprintf("decode data file %s: %v", e.Path, e.Err)
}

func (e *CorruptDataError) Unwrap() error { return e.Err }

// Load user's local data file or create new file
func LoadAppData() (*model.AppData, error) {
	path, err := dataFilePath()
	if err != nil {
		return nil, err
	}
	return loadAppDataFrom(path)
}

// Write modified JSON data
func SaveAppData(data model.AppData) error {
	path, err := dataFilePath()
	if err != nil {
		return err
	}
	return saveAppDataTo(path, data)
}

// RestoreBackup replaces an unreadable data.json with data.json.bak and
// loads it. The unreadable file is kept as data.json.corrupt-<time> rather
// than deleted, in case anything in it can be recovered by hand.
func RestoreBackup() (*model.AppData, error) {
	path, err := dataFilePath()
	if err != nil {
		return nil, err
	}
	return restoreBackupAt(path, time.Now())
}

func loadAppDataFrom(path string) (*model.AppData, error) {
	if err := ensureDataFile(path); err != nil {
		return nil, err
	}
	removeStaleTemps(path)
	removeStaleTemps(path + backupSuffix)

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read data file: %w", err)
	}

	var data model.AppData
	if err := json.Unmarshal(bytes, &data); err != nil {
		corrupt := &CorruptDataError{Path: path, BackupPath: path + backupSuffix, Err: err}
		if info, ok := readableBackup(corrupt.BackupPath); ok {
			corrupt.HasBackup = true
			corrupt.BackupTime = info.ModTime()
		}
		return nil, corrupt
	}

	normalized, changed := normalizeAppData(data)
	if changed {
		if err := saveAppDataTo(path, normalized); err != nil {
			return nil, err
		}
	}

	return &normalized, nil
}

func saveAppDataTo(path string, data model.AppData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create app config directory: %w", err)
	}

	normalized, _ := normalizeAppData(data)
	jsonBytes, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("encode data file: %w", err)
	}

	// Keep the version being replaced as data.json.bak. Only a readable
	// file is backed up, so a corrupt data.json never overwrites a good
	// backup, and an unchanged save leaves the backup one real edit behind.
	if old, err := os.ReadFile(path); err == nil && json.Valid(old) && !bytes.Equal(old, jsonBytes) {
		if err := writeFileAtomic(path+backupSuffix, old, 0o644); err != nil {
			return fmt.Errorf("write backup file: %w", err)
		}
	}

	if err := writeFileAtomic(path, jsonBytes, 0o644); err != nil {
		return fmt.Errorf("write data file: %w", err)
	}

	return nil
}

func restoreBackupAt(path string, now time.Time) (*model.AppData, error) {
	backupPath := path + backupSuffix
	if _, ok := readableBackup(backupPath); !ok {
		return nil, fmt.Errorf("no readable backup at %s", backupPath)
	}
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		return nil, fmt.Errorf("read backup file: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		aside := fmt.Sprintf("%s.corrupt-%s", path, now.Format("20060102-150405"))
		if err := os.Rename(path, aside); err != nil {
			return nil, fmt.Errorf("move unreadable data file aside: %w", err)
		}
	}

	if err := writeFileAtomic(path, backup, 0o644); err != nil {
		return nil, fmt.Errorf("restore backup: %w", err)
	}

	return loadAppDataFrom(path)
}

// readableBackup reports whether the backup exists and decodes as AppData.
func readableBackup(path string) (os.FileInfo, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var data model.AppData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, false
	}
	return info, true
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

	if err := writeFileAtomic(path, jsonBytes, 0o644); err != nil {
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
				ID:       "comp123",
				Name:     "Introduction to Computer",
				Code:     "COMP 123",
				Term:     "Spring 2032",
				CanvasID: 12345,
				Workflow: "available",
				Location: "JD 1234",
				Meetings: []model.Meeting{{Days: []string{"Mon", "Wed"}, Start: "12:30", End: "13:45", Location: "JD 1234"}},
			},
			{
				ID:       "comp456",
				Name:     "Analysis of Computer",
				Code:     "COMP 456",
				Term:     "Spring 2032",
				CanvasID: 67890,
				Workflow: "available",
				Location: "JD 5678",
				Meetings: []model.Meeting{{Days: []string{"Tue", "Thu"}, Start: "10:00", End: "11:15", Location: "JD 5678"}},
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
		// Convert old "Mon 12:34 PM" strings to structured meetings; keep any
		// string that can't be read so nothing is silently lost
		if len(course.MeetingTimes) > 0 {
			var unread []string
			for _, legacy := range course.MeetingTimes {
				if m, ok := model.MeetingFromLegacy(legacy); ok {
					course.Meetings = append(course.Meetings, m)
				} else {
					unread = append(unread, legacy)
				}
			}
			course.MeetingTimes = unread
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

	// Notes are user-written only: never seeded, so an empty list stays empty
	normalizedNotes := make([]model.Note, 0, len(data.Notes))
	noteSeen := make(map[string]bool)
	for i, note := range data.Notes {
		if strings.TrimSpace(note.ID) == "" || noteSeen[note.ID] {
			note.ID = fmt.Sprintf("note-%d-%d", note.UpdatedAt.UnixNano(), i+1)
			changed = true
		}
		noteSeen[note.ID] = true
		if strings.TrimSpace(note.Title) == "" {
			note.Title = "Untitled"
			changed = true
		}
		if strings.TrimSpace(note.CourseID) != "" && !courseValid[note.CourseID] {
			note.CourseID = ""
			changed = true
		}
		normalizedNotes = append(normalizedNotes, note)
	}
	if data.Notes != nil {
		data.Notes = normalizedNotes
	}

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
