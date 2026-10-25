package saves

import (
	"context"
	"os"
	"path/filepath"

	"gamepack/internal/cfg"
	"gamepack/internal/compress"
)

// SaveEntry is one save source captured in the bundle.
type SaveEntry struct {
	GameSavesRel string   `json:"gamesavesRel"`
	ProtonRel    string   `json:"protonRel"`
	Files        []string `json:"files,omitempty"`
}

// Discover returns the save entries for a game: golden table mappings with
// existing GameSaves dirs, plus any live symlink-discovered entries.
func Discover(c *cfg.Config, appid uint32) []SaveEntry {
	var entries []SaveEntry
	seen := map[string]bool{}
	add := func(m Mapping) {
		if seen[m.GameSavesRel] {
			return
		}
		seen[m.GameSavesRel] = true
		dir := c.GameSavesRelPath(m.GameSavesRel)
		entries = append(entries, SaveEntry{
			GameSavesRel: filepath.ToSlash(m.GameSavesRel),
			ProtonRel:    filepath.ToSlash(m.ProtonRel),
			Files:        listFiles(dir),
		})
	}
	for _, m := range GoldenFor(c, appid) {
		add(m)
	}
	if fi, err := os.Stat(c.PrefixDir(appid)); err == nil && fi.IsDir() {
		for _, m := range DiscoverLiveSymlinks(c.PrefixDir(appid), c.GameSaves) {
			if fi, err := os.Stat(c.GameSavesRelPath(m.GameSavesRel)); err == nil && fi.IsDir() {
				add(m)
			}
		}
	}
	return entries
}

func listFiles(dir string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		out = append(out, filepath.ToSlash(p))
		return nil
	})
	return out
}

// Bundle packs the discovered save dirs into saves.tar.zst under gamesaves/<rel>.
// Returns the number of files archived (0 if no saves found).
func Bundle(ctx context.Context, c *cfg.Config, appid uint32, outPath string) (int, error) {
	entries := Discover(c, appid)
	if len(entries) == 0 {
		return 0, nil
	}
	args := []string{"-c", "--acls", "--xattrs", "--sparse",
		"-C", c.GameSaves,
		"--transform=" + "s,^,gamesaves/,"}
	for _, e := range entries {
		args = append(args, filepath.FromSlash(e.GameSavesRel))
	}
	if err := compress.PackDirArgs(ctx, c, args, outPath, compress.LevelMax, nil); err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		n += len(e.Files)
	}
	return n, nil
}

// RestoreFile copies a file into dest, skipping if dest exists and is newer
// than the archived copy's mtime (skip-newer, never clobber fresher saves).
func RestoreFile(src, dest string, archivedMtime int64) (restored bool, err error) {
	if fi, err := os.Stat(dest); err == nil && fi.ModTime().Unix() >= archivedMtime {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return false, err
	}
	if err := copyFile(src, dest); err != nil {
		return false, err
	}
	_ = os.Chtimes(dest, timeNow(), timeNow())
	return true, nil
}
