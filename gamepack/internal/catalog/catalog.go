package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gamepack/internal/cfg"
)

type GameDirType string

const (
	DirTypePlain           GameDirType = "plain"
	DirTypePrefixContainer GameDirType = "prefix-container"
)

// Game is one cataloged non-Steam game.
type Game struct {
	Name         string      `json:"name"`
	AppID        uint32      `json:"appid"`
	Exe          string      `json:"exe"`
	StartDir     string      `json:"startDir"`
	GameDir      string      `json:"gameDir"`
	GameDirType  GameDirType `json:"gameDirType"`
	SizeBytes    int64       `json:"sizeBytes"`
	HasPrefix    bool        `json:"hasPrefix"`
	DirMissing   bool        `json:"dirMissing"`
	Exclude      string      `json:"exclude,omitempty"`
	Packable     bool        `json:"packable"`
	IsArchived   bool        `json:"isArchived"`
	ArchivedAt   string      `json:"archivedAt,omitempty"`
	ArchivedSize int64       `json:"archivedSize,omitempty"`
}

// ResolveGameDir maps a shortcut's exe path to the game directory to archive.
//
// Default: the directory containing the exe. Special case: if an ancestor of
// the exe looks like a Wine/Proton prefix root (has drive_c/ + system.reg),
// the whole prefix-root directory is the game dir (e.g. Split Fiction lives in
// "Games_Data/Split Fictionx" which is itself a prefix-style container).
func ResolveGameDir(exe string) (string, GameDirType) {
	d := filepath.Dir(exe)
	cur := d
	for i := 0; i < 8; i++ {
		if isPrefixRoot(cur) {
			return cur, DirTypePrefixContainer
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return d, DirTypePlain
}

func isPrefixRoot(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, "drive_c"))
	if err != nil || !fi.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "system.reg")); err != nil {
		return false
	}
	return true
}

// ExcludeReason decides whether a shortcut should be excluded from the PC pack
// list. Returns the reason and true if excluded.
func ExcludeReason(s Shortcut) (string, bool) {
	switch s.AppID {
	case 2379889317: // Linux-Yuzu
		return "Switch emulator", true
	case 3836569258: // Eden-Linux
		return "Switch emulator", true
	case 3640729765: // Switch Citron
		return "Switch emulator", true
	case 2587493377: // Ascendara.AppImage
		return "app installer tool", true
	case 2224065267: // Octopath Traveler 0 -> Citron launcher (Switch)
		return "Switch (Citron) launcher", true
	}
	base := strings.ToLower(filepath.Base(s.Exe))
	if strings.HasSuffix(base, ".appimage") {
		return "AppImage tool/emulator", true
	}
	if strings.HasPrefix(s.Exe, "/var/home/") {
		return "exe outside games drive", true
	}
	for _, kw := range []string{"citron", "yuzu", "eden", "retrodeck", "ryujinx"} {
		if strings.Contains(base, kw) {
			return "emulator", true
		}
	}
	return "", false
}

// BuildCatalog parses shortcuts.vdf and produces the full game catalog.
func BuildCatalog(c *cfg.Config) ([]Game, error) {
	shortcuts, err := ParseShortcutsVDF(c.ShortcutsVdf)
	if err != nil {
		return nil, err
	}
	games := make([]Game, 0, len(shortcuts))
	for _, s := range shortcuts {
		if s.Name == "" {
			continue
		}
		g := Game{
			Name:     s.Name,
			AppID:    s.AppID,
			Exe:      s.Exe,
			StartDir: s.StartDir,
		}
		if reason, excl := ExcludeReason(s); excl {
			g.Exclude = reason
			g.Packable = false
			games = append(games, g)
			continue
		}
		dir, t := ResolveGameDir(s.Exe)
		g.GameDir = dir
		g.GameDirType = t
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			g.SizeBytes = dirSize(dir)
			g.DirMissing = false
		} else {
			g.DirMissing = true
		}
		if c.HasPrefix(s.AppID) {
			g.HasPrefix = true
		}
		g.Packable = !g.DirMissing
		// self-heal archived flag: a bundle only counts if manifest.json exists
		slug := Slugify(s.Name, s.AppID)
		if _, err := os.Stat(filepath.Join(c.PCArchiveDir(slug), "manifest.json")); err == nil {
			g.IsArchived = true
		}
		games = append(games, g)
	}
	return games, nil
}

// dirSize recursively sums a directory size. Best-effort; errors are ignored.
func dirSize(dir string) int64 {
	var total int64
	filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if fi, err := d.Info(); err == nil {
				total += fi.Size()
			}
		}
		return nil
	})
	return total
}

// Slugify produces a filesystem-safe slug for archive dirs (name + appid).
func Slugify(name string, appid uint32) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '_', r == '.':
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "game"
	}
	return fmt.Sprintf("%s-%d", s, appid)
}

// --- index ---

type Index struct {
	ToolVersion string  `json:"toolVersion"`
	GeneratedAt string  `json:"generatedAt"`
	PC          []Game  `json:"pcGames"`
	NSwitch     []NSROM `json:"nswitch"`
}

// NSROM is one Switch ROM entry.
type NSROM struct {
	Name         string `json:"name"`
	File         string `json:"file"`
	SizeBytes    int64  `json:"sizeBytes"`
	IsArchived   bool   `json:"isArchived"`
	ArchivedAt   string `json:"archivedAt,omitempty"`
	ArchivedSize int64  `json:"archivedSize,omitempty"`
	TitleID      string `json:"titleId,omitempty"`
}

func LoadIndex(c *cfg.Config) (*Index, error) {
	data, err := os.ReadFile(c.IndexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{}, nil
		}
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

func (ix *Index) Save(c *cfg.Config) error {
	if err := os.MkdirAll(c.ArchiveRoot, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.IndexPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.IndexPath())
}

// HasArchive checks whether an archive bundle dir already exists for a slug.
func (ix *Index) MarkArchived(c *cfg.Config, slug string, gameDir string, size int64) {
	for i := range ix.PC {
		if ix.PC[i].GameDir == gameDir {
			ix.PC[i].IsArchived = true
			ix.PC[i].ArchivedAt = nowISO()
			ix.PC[i].ArchivedSize = size
			_ = slug
			return
		}
	}
}

func nowISO() string {
	return timeNow().UTC().Format("2006-01-02T15:04:05Z")
}
