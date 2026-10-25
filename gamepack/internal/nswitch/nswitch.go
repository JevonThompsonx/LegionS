package nswitch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gamepack/internal/catalog"
	"gamepack/internal/cfg"
	"gamepack/internal/compress"
)

// CitronSavesRoot pattern: <root>/<titleid>/ — title IDs known to exist.
var KnownTitleIDs = []string{
	"010015100B514000",
	"01004B100AF18000",
	"0100646009FBE000",
	"01007300020FA000",
	"010074F013262000",
	"01007960049A0000",
	"0100DCA0064A6000",
}

// CuratedTitleIDMap maps ROM display names (normalized) to Citron title IDs.
// Seeded with entries verified on the ROG; extend via the TUI prompt.
var CuratedTitleIDMap = map[string]string{}

// ROM is one Switch ROM file.
type ROM struct {
	Name      string
	File      string
	SizeBytes int64
	TitleID   string // matched or empty
}

// ScanROMS lists ROM files in the NSwitch dir.
func ScanROMS(c *cfg.Config) ([]ROM, error) {
	entries, err := os.ReadDir(c.NSwitchDir)
	if err != nil {
		return nil, err
	}
	var out []ROM
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".xci" && ext != ".nsp" && ext != ".nsz" && ext != ".xcz" {
			continue
		}
		fi, _ := e.Info()
		base := baseName(e.Name())
		out = append(out, ROM{
			Name:      base,
			File:      e.Name(),
			SizeBytes: fi.Size(),
			TitleID:   MatchTitle(base),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func baseName(file string) string {
	s := filepath.Base(file)
	if i := strings.LastIndex(s, "."); i >= 0 {
		s = s[:i]
	}
	return s
}

// MatchTitle tries curated map, then a [titleid] in the name, else empty.
func MatchTitle(name string) string {
	base := baseName(name)
	norm := strings.ToLower(strings.TrimSpace(base))
	if id, ok := CuratedTitleIDMap[norm]; ok {
		return id
	}
	re := regexp.MustCompile(`\[?(0100[0-9a-fA-F]{12})\]?`)
	if m := re.FindStringSubmatch(name); len(m) == 2 {
		return strings.ToUpper(m[1])
	}
	return ""
}

// Pack packs one ROM (tar | zstd -3) and optionally the matching Citron save dir.
func Pack(ctx context.Context, c *cfg.Config, r ROM, progress func(string, float64)) (gamePath, savesPath string, saveFiles int, err error) {
	slug := catalog.Slugify(r.Name, 0)
	outDir := c.NSwitchArchiveDir(slug)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", "", 0, err
	}
	gamePath = filepath.Join(outDir, "rom.tar.zst")
	if progress != nil {
		progress("archiving ROM", 0.1)
	}
	// Single-file pack: tar -C <dir> -f - <file> | zstd -3
	args := []string{"-c", "-C", c.NSwitchDir, r.File}
	if err := compress.PackDirArgs(ctx, c, args, gamePath, compress.LevelROM, nil); err != nil {
		return "", "", 0, fmt.Errorf("pack ROM: %w", err)
	}
	if progress != nil {
		progress("packing Citron saves", 0.85)
	}
	if r.TitleID != "" {
		src := filepath.Join(c.CitronSaves, r.TitleID)
		if fi, err := os.Stat(src); err == nil && fi.IsDir() {
			savesPath = filepath.Join(outDir, "saves.tar.zst")
			// pack the <titleid> dir so extraction preserves the name
			par := filepath.Dir(src)
			base := filepath.Base(src)
			args := []string{"-c", "-C", par, base}
			if err := compress.PackDirArgs(ctx, c, args, savesPath, compress.LevelMax, nil); err != nil {
				return "", "", 0, err
			}
			saveFiles = 1
		}
	}
	if progress != nil {
		progress("done", 1.0)
	}
	return gamePath, savesPath, saveFiles, nil
}

// Unpack restores a ROM and its saves.
func Unpack(ctx context.Context, c *cfg.Config, r ROM, outPath, slug string) error {
	if err := compress.Extract(ctx, c, filepath.Join(c.NSwitchArchiveDir(slug), "rom.tar.zst"), c.NSwitchDir); err != nil {
		return err
	}
	savesPath := filepath.Join(c.NSwitchArchiveDir(slug), "saves.tar.zst")
	if _, err := os.Stat(savesPath); err == nil {
		// archive contains <titleid>/... ; extract into citron save root parent
		if err := compress.Extract(ctx, c, savesPath, filepath.Dir(c.CitronSaves)); err != nil {
			return err
		}
	}
	return nil
}
