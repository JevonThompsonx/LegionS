package saves

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func timeNow() time.Time { return time.Now() }

// under reports whether path is under root and returns the relative path.
func under(root, path string) (string, bool) {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if path == root {
		return "", true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}
