package saves

import (
	"os"
	"path/filepath"

	"gamepack/internal/cfg"
)

// Mapping is one save location for a game: where it lives in GameSaves
// (gamesaves-relative) and where it must appear inside a Proton prefix
// (drive_c-relative, without the drive_c/users/<user> prefix resolution).
type Mapping struct {
	AppID        uint32
	GameSavesRel string
	ProtonRel    string // e.g. "users/steamuser/AppData/Local/SplitFiction"
	NoSymlink    bool   // must NOT be symlinked (real dir inside prefix, e.g. Cuphead LocalLow)
}

// Golden mirrors GameSaves/link-saves.sh — the authoritative manual mapping.
var Golden = []Mapping{
	{AppID: 2986239671, GameSavesRel: "appdata-roaming/Cuphead", ProtonRel: "users/steamuser/AppData/Roaming/Cuphead"},
	{AppID: 4155978628, GameSavesRel: "appdata-roaming/FEZ", ProtonRel: "users/steamuser/AppData/Roaming/FEZ"},
	{AppID: 4151545067, GameSavesRel: "saved-games/Respawn/JediSurvivor", ProtonRel: "users/steamuser/Saved Games/Respawn/JediSurvivor"},
	{AppID: 3794432564, GameSavesRel: "appdata-local/Rune Factory Guardians of Azuma", ProtonRel: "users/steamuser/AppData/Local/Rune Factory Guardians of Azuma"},
	{AppID: 4076669000, GameSavesRel: "saved-games/CD Projekt Red", ProtonRel: "users/steamuser/Saved Games/CD Projekt Red"},
	{AppID: 4076669000, GameSavesRel: "appdata-roaming/CD Projekt Red", ProtonRel: "users/steamuser/AppData/Roaming/CD Projekt Red"},
	{AppID: 2508814786, GameSavesRel: "appdata-local/SplitFiction", ProtonRel: "users/steamuser/AppData/Local/SplitFiction"},
	{AppID: 2152676719, GameSavesRel: "appdata-roaming/Nippon Ichi Software, Inc/Disgaea 7", ProtonRel: "users/steamuser/AppData/Roaming/Nippon Ichi Software, Inc/Disgaea 7"},
	{AppID: 2773086359, GameSavesRel: "appdata-local/Dispatch", ProtonRel: "users/steamuser/AppData/Local/Dispatch"},
	{AppID: 2955550478, GameSavesRel: "saved-games/Hades", ProtonRel: "users/steamuser/Saved Games/Hades"},
}

// GoldenFor returns golden mappings for an appid whose GameSaves source dir exists.
func GoldenFor(c *cfg.Config, appid uint32) []Mapping {
	var out []Mapping
	for _, m := range Golden {
		if m.AppID != appid {
			continue
		}
		if fi, err := os.Stat(c.GameSavesRelPath(m.GameSavesRel)); err == nil && fi.IsDir() {
			out = append(out, m)
		}
	}
	return out
}

// DiscoverLiveSymlinks scans a prefix's steamuser tree for symlinks into
// GameSaves, yielding mappings the auto-linker added that are not in the
// golden table. Returns discovered mappings (appid-agnostic).
func DiscoverLiveSymlinks(prefixDir, gamesaves string) []Mapping {
	var out []Mapping
	root := filepath.Join(prefixDir, "pfx", "drive_c")
	gsAbs, _ := filepath.Abs(gamesaves)
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type()&os.ModeSymlink == 0 {
			return nil
		}
		tgt, err := os.Readlink(p)
		if err != nil {
			return nil
		}
		if !filepath.IsAbs(tgt) {
			tgt = filepath.Join(filepath.Dir(p), tgt)
		}
		tgtAbs, err := filepath.Abs(tgt)
		if err != nil {
			return nil
		}
		rel, ok := under(gsAbs, tgtAbs)
		if !ok {
			return nil
		}
		protonRel := filepath.ToSlash(relAfter(root, p))
		out = append(out, Mapping{GameSavesRel: filepath.ToSlash(rel), ProtonRel: protonRel})
		return nil
	})
	return out
}

func relAfter(root, p string) string {
	r := filepath.Clean(root) + string(filepath.Separator)
	pp := filepath.Clean(p)
	if len(pp) > len(r) && pp[:len(r)] == r {
		return pp[len(r):]
	}
	return pp
}

// under reports whether path is under root and returns the relative path.
