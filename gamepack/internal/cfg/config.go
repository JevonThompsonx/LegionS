package cfg

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config holds every runtime path the tool needs. Defaults are the live-verified
// paths on the ROG (bazzite-mobile). All are overridable via flags/env.
type Config struct {
	ShortcutsVdf string
	CompatData   string
	SteamBin     string
	GameSaves    string
	LudusaviPath string
	NSwitchDir   string
	CitronSaves  string
	ArchiveRoot  string
	ZstdBin      string
	TarBin       string
	FlatpakBin   string
	FlockBin     string
	PgrepBin     string
	WithPrefix   bool
	Force        bool
	RemoveSource bool // delete the game folder after a successful pack
}

func Default() *Config {
	return &Config{
		ShortcutsVdf: "/var/home/jevonx/.local/share/Steam/userdata/201689394/config/shortcuts.vdf",
		CompatData:   "/var/home/jevonx/.local/share/Steam/steamapps/compatdata",
		SteamBin:     "/usr/bin/steam",
		GameSaves:    "/var/home/jevonx/GameSaves",
		LudusaviPath: "/var/home/jevonx/ludusavi-backup",
		NSwitchDir:   "/run/media/system/Games_Data/NSwitch",
		CitronSaves:  "/var/home/jevonx/.local/share/citron/nand/user/save/0000000000000000/E06398B8B15F1E9E9F71D466CD63878B",
		ArchiveRoot:  "/run/media/system/Games_Data/Archive",
		ZstdBin:      "/usr/bin/zstd",
		TarBin:       "/usr/bin/tar",
		FlatpakBin:   "/usr/bin/flatpak",
		FlockBin:     "/usr/bin/flock",
		PgrepBin:     "/usr/bin/pgrep",
		WithPrefix:   true,
	}
}

// PCArchiveDir returns the per-game archive dir for a PC game slug.
func (c *Config) PCArchiveDir(slug string) string { return filepath.Join(c.ArchiveRoot, "PC", slug) }

// NSwitchArchiveDir returns the per-game archive dir for a Switch ROM slug.
func (c *Config) NSwitchArchiveDir(slug string) string {
	return filepath.Join(c.ArchiveRoot, "NSwitch", slug)
}

func (c *Config) IndexPath() string { return filepath.Join(c.ArchiveRoot, ".index.json") }
func (c *Config) JournalPath() string {
	if p := os.Getenv("GAMEPACK_JOURNAL"); p != "" {
		return p
	}
	return filepath.Join(c.ArchiveRoot, ".journal.json")
}

func (c *Config) PrefixDir(appid uint32) string {
	return filepath.Join(c.CompatData, uint32Str(appid))
}

// LibraryCompatDataRoots returns every Steam library's compatdata dir, found
// via libraryfolders.vdf plus the primary. Games can create prefixes in any
// library, so prefix detection must check all of them.
func (c *Config) LibraryCompatDataRoots() []string {
	seen := map[string]bool{c.CompatData: true}
	roots := []string{c.CompatData}
	steamRoot := filepath.Dir(filepath.Dir(c.CompatData))
	vdfPath := filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf")
	data, err := os.ReadFile(vdfPath)
	if err != nil {
		return roots
	}
	re := regexp.MustCompile(`"path"\s+"([^"]+)"`)
	for _, m := range re.FindAllStringSubmatch(string(data), -1) {
		p := strings.ReplaceAll(m[1], `\\`, `\`)
		cd := filepath.Join(p, "steamapps", "compatdata")
		if !seen[cd] {
			seen[cd] = true
			roots = append(roots, cd)
		}
	}
	return roots
}

// HasPrefix reports whether a Proton prefix exists for appid in any library.
func (c *Config) HasPrefix(appid uint32) bool {
	for _, root := range c.LibraryCompatDataRoots() {
		if fi, err := os.Stat(filepath.Join(root, uint32Str(appid), "pfx")); err == nil && fi.IsDir() {
			return true
		}
	}
	return false
}

func (c *Config) GameSavesRelPath(rel string) string {
	return filepath.Join(c.GameSaves, filepath.FromSlash(rel))
}

func uint32Str(v uint32) string {
	return fmtUint32(v)
}

func fmtUint32(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
