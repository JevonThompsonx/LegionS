package saves

import (
	"os"
	"path/filepath"
)

// LinkDir makes targetPath a symlink to source. Creates parent dirs. Idempotent:
// if targetPath is already a symlink to source, it is left alone; if it is a
// real dir, it is moved aside to .bak.<ts> first (mirrors link-saves.sh).
func LinkDir(source, targetPath string) error {
	if fi, err := os.Lstat(targetPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			tgt, _ := os.Readlink(targetPath)
			if tgt == source {
				return nil
			}
			_ = os.Remove(targetPath)
		} else if fi.IsDir() {
			backup := targetPath + ".bak." + stamp()
			_ = os.Rename(targetPath, backup)
		} else {
			_ = os.Remove(targetPath)
		}
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	return os.Symlink(source, targetPath)
}
