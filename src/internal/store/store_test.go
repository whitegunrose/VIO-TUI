package store

import (
	"VIO/internal/model"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func tempDataPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "vio", dataFileName)
}

func sampleData(name string) model.AppData {
	data := defaultAppData()
	data.Student.Name = name
	return data
}

func studentName(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var data model.AppData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return data.Student.Name
}

func assertNoTemps(t *testing.T, path string) {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".*.tmp-*"))
	if len(matches) > 0 {
		t.Fatalf("temp files left behind: %v", matches)
	}
}

func TestSaveKeepsPreviousVersionAsBackup(t *testing.T) {
	path := tempDataPath(t)

	if err := saveAppDataTo(path, sampleData("First")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + backupSuffix); !os.IsNotExist(err) {
		t.Fatalf("first save should not create a backup, stat err = %v", err)
	}

	if err := saveAppDataTo(path, sampleData("Second")); err != nil {
		t.Fatal(err)
	}
	if got := studentName(t, path); got != "Second" {
		t.Fatalf("data.json name = %q, want Second", got)
	}
	if got := studentName(t, path+backupSuffix); got != "First" {
		t.Fatalf("backup name = %q, want First", got)
	}
	assertNoTemps(t, path)
}

func TestUnchangedSaveKeepsOlderBackup(t *testing.T) {
	path := tempDataPath(t)
	first, second := sampleData("First"), sampleData("Second")
	for _, d := range []model.AppData{first, second, second} {
		if err := saveAppDataTo(path, d); err != nil {
			t.Fatal(err)
		}
	}
	if got := studentName(t, path+backupSuffix); got != "First" {
		t.Fatalf("backup name = %q, want First (a no-op save must not overwrite it)", got)
	}
}

func TestCorruptFileNeverOverwritesBackup(t *testing.T) {
	path := tempDataPath(t)
	if err := saveAppDataTo(path, sampleData("Good")); err != nil {
		t.Fatal(err)
	}
	if err := saveAppDataTo(path, sampleData("Newer")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"student": {"na`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := saveAppDataTo(path, sampleData("After")); err != nil {
		t.Fatal(err)
	}
	if got := studentName(t, path+backupSuffix); got != "Good" {
		t.Fatalf("backup name = %q, want Good", got)
	}
}

func TestLoadCorruptFileReportsBackup(t *testing.T) {
	for _, tc := range []struct {
		name       string
		withBackup bool
	}{
		{"with backup", true},
		{"without backup", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := tempDataPath(t)
			if err := saveAppDataTo(path, sampleData("Good")); err != nil {
				t.Fatal(err)
			}
			if tc.withBackup {
				if err := saveAppDataTo(path, sampleData("Newer")); err != nil {
					t.Fatal(err)
				}
			}
			// An empty file is what a crash mid-write usually leaves
			if err := os.WriteFile(path, nil, 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := loadAppDataFrom(path)
			var corrupt *CorruptDataError
			if !errors.As(err, &corrupt) {
				t.Fatalf("err = %v, want *CorruptDataError", err)
			}
			if corrupt.HasBackup != tc.withBackup {
				t.Fatalf("HasBackup = %v, want %v", corrupt.HasBackup, tc.withBackup)
			}
			if tc.withBackup && corrupt.BackupTime.IsZero() {
				t.Fatal("BackupTime not set")
			}
		})
	}
}

func TestUnreadableBackupIsNotOffered(t *testing.T) {
	path := tempDataPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, []byte("not json"), 0o644)
	os.WriteFile(path+backupSuffix, []byte("also not json"), 0o644)

	_, err := loadAppDataFrom(path)
	var corrupt *CorruptDataError
	if !errors.As(err, &corrupt) {
		t.Fatalf("err = %v, want *CorruptDataError", err)
	}
	if corrupt.HasBackup {
		t.Fatal("an unreadable backup must not be offered")
	}
	if _, err := restoreBackupAt(path, time.Now()); err == nil {
		t.Fatal("restoring an unreadable backup should fail")
	}
}

func TestRestoreBackup(t *testing.T) {
	path := tempDataPath(t)
	saveAppDataTo(path, sampleData("Good"))
	saveAppDataTo(path, sampleData("Newer"))
	garbage := []byte(`{"student": {"na`)
	os.WriteFile(path, garbage, 0o644)

	now := time.Date(2026, 11, 19, 22, 30, 0, 0, time.Local)
	data, err := restoreBackupAt(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if data.Student.Name != "Good" {
		t.Fatalf("restored name = %q, want Good", data.Student.Name)
	}
	if got := studentName(t, path); got != "Good" {
		t.Fatalf("data.json after restore = %q, want Good", got)
	}

	// The unreadable file is kept, not deleted
	aside, err := os.ReadFile(path + ".corrupt-20261119-223000")
	if err != nil {
		t.Fatalf("corrupt file not kept: %v", err)
	}
	if string(aside) != string(garbage) {
		t.Fatalf("kept file = %q, want the original bytes", aside)
	}
}

func TestLoadRemovesStaleTempFiles(t *testing.T) {
	path := tempDataPath(t)
	saveAppDataTo(path, sampleData("Good"))
	stale := filepath.Join(filepath.Dir(path), ".data.json.tmp-123456")
	os.WriteFile(stale, []byte("{half"), 0o644)

	if _, err := loadAppDataFrom(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale temp file still there (err = %v)", err)
	}
}

func TestFirstLoadCreatesStarterData(t *testing.T) {
	path := tempDataPath(t)
	data, err := loadAppDataFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Courses) == 0 {
		t.Fatal("starter data has no courses")
	}
	assertNoTemps(t, path)
}

func TestWriteFileAtomicSetsPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permissions")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := writeFileAtomic(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o, want 600", perm)
	}
	assertNoTemps(t, path)
}

func TestWriteFileAtomicFailureKeepsOldFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte("old"), 0o644)

	// Renaming a file over a non-empty directory fails on every platform
	bad := filepath.Join(dir, "blocked")
	os.MkdirAll(filepath.Join(bad, "child"), 0o755)
	if err := writeFileAtomic(bad, []byte("new"), 0o644); err == nil {
		t.Fatal("expected an error writing over a directory")
	}
	if got, _ := os.ReadFile(path); string(got) != "old" {
		t.Fatalf("unrelated file changed: %q", got)
	}
	assertNoTemps(t, bad)
}
