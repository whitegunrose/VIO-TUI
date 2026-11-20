package main

import (
	"VIO/internal/model"
	"VIO/internal/store"
	"errors"
	"fmt"
	"os"

	"github.com/rivo/tview"
)

const (
	restoreButton = "Restore backup"
	quitButton    = "Quit"
)

// loadOrRecover loads data.json. If it can't be read (for example a crash
// left it empty), it offers to restore data.json.bak instead of crashing.
// Choosing Quit leaves every file untouched.
func loadOrRecover() *model.AppData {
	data, err := store.LoadAppData()
	if err == nil {
		return data
	}

	var corrupt *store.CorruptDataError
	if !errors.As(err, &corrupt) {
		fail("vio: %v\n", err)
	}

	if !corrupt.HasBackup {
		fail("vio: %s can't be read (%v)\n"+
			"There is no backup to restore. Fix the file by hand, or move it\n"+
			"aside and VIO will start with fresh sample data.\n",
			corrupt.Path, corrupt.Err)
	}

	if !askToRestore(corrupt) {
		fail("vio: left %s untouched.\nThe backup is at %s.\n", corrupt.Path, corrupt.BackupPath)
	}

	data, err = store.RestoreBackup()
	if err != nil {
		fail("vio: restoring the backup failed: %v\n", err)
	}
	return data
}

// askToRestore shows a one-off modal before the dashboard is built.
func askToRestore(corrupt *store.CorruptDataError) bool {
	restore := false
	app := tview.NewApplication()
	modal := tview.NewModal().
		SetText(fmt.Sprintf(
			"Your data file can't be read.\n\n%v\n\n"+
				"Restore the backup from %s?\n\n"+
				"The unreadable file will be kept next to it as data.json.corrupt-….",
			corrupt.Err,
			corrupt.BackupTime.Format("Mon Jan 2, 3:04 PM"),
		)).
		AddButtons([]string{restoreButton, quitButton}).
		SetDoneFunc(func(_ int, label string) {
			restore = label == restoreButton
			app.Stop()
		})

	if err := app.SetRoot(modal, true).SetFocus(modal).Run(); err != nil {
		fail("vio: %v\n", err)
	}
	return restore
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
