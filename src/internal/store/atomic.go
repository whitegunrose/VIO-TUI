package store

import (
	"os"
	"path/filepath"
)

// writeFileAtomic replaces path with data so that a crash or power loss
// leaves either the old file or the new one, never a half-written file.
//
// It writes to a temp file in the same folder (so the rename stays on one
// filesystem), fsyncs it, renames it over path, then fsyncs the folder so
// the rename itself is on disk.
func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, tempPattern(path))
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}

	syncDir(dir)
	return nil
}

// tempPattern names temp files ".data.json.tmp-*" so they're hidden and
// easy to find if a crash leaves one behind.
func tempPattern(path string) string {
	return "." + filepath.Base(path) + ".tmp-*"
}

// removeStaleTemps deletes temp files left by a save that crashed before
// its rename. The real file is untouched by such a crash.
func removeStaleTemps(path string) {
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), tempPattern(path)))
	for _, m := range matches {
		os.Remove(m)
	}
}

// syncDir flushes a folder's entries (the rename) to disk. Best effort:
// some platforms, Windows included, can't fsync a folder.
func syncDir(dir string) {
	f, err := os.Open(dir)
	if err != nil {
		return
	}
	f.Sync()
	f.Close()
}
