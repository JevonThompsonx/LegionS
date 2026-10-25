package pack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gamepack/internal/catalog"
	"gamepack/internal/cfg"
	"gamepack/internal/compress"
	"gamepack/internal/proton"
	"gamepack/internal/saves"
)

const ToolVersion = "0.1.0"

type PackResult struct {
	GamePath    string
	SavesPath   string
	PrefixPath  string
	Manifest    *Manifest
	Slug        string
	SaveFiles   int
	AlreadyDone bool
}

// GameExcludes returns tar exclude patterns (relative to the packed dir root)
// for a game dir based on its type.
func GameExcludes(g catalog.Game) []string {
	if g.GameDirType == catalog.DirTypePrefixContainer {
		// derived/regenerable members of a prefix-container game dir
		return []string{"shadercache", "gstreamer-1.0", "pfx.lock", "tracked_files", "dosdevices/z:"}
	}
	return nil
}

// PrefixExcludes for archiving compatdata/<appid>.
func PrefixExcludes() []string { return []string{"shadercache", "pfx.lock"} }

// Pack archives one PC game: game dir (max zstd), save snapshot, and (if
// enabled) the Proton prefix. progress receives 0..1 float as it goes.
func Pack(ctx context.Context, c *cfg.Config, g catalog.Game, progress func(phase string, frac float64)) (*PackResult, error) {
	if proton.IsRunning(c, g.AppID, filepath.Base(g.Exe)) {
		return nil, fmt.Errorf("%s is running — close the game before packing", g.Name)
	}
	if g.DirMissing {
		return nil, fmt.Errorf("%s: game dir missing (%s)", g.Name, g.GameDir)
	}

	slug := catalog.Slugify(g.Name, g.AppID)
	outDir := c.PCArchiveDir(slug)
	// Only a completed bundle (manifest present) counts as archived — a partial
	// dir left by a crash/reboot must be re-packed from scratch.
	if _, err := os.Stat(filepath.Join(outDir, "manifest.json")); err == nil && !c.Force {
		return &PackResult{Slug: slug, AlreadyDone: true}, nil
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	gamePath := filepath.Join(outDir, "game.tar.zst")
	savesPath := filepath.Join(outDir, "saves.tar.zst")
	prefixPath := filepath.Join(outDir, "prefix.tar.zst")

	if progress != nil {
		progress("archiving game files", 0.05)
	}
	origSize := g.SizeBytes
	if origSize <= 0 {
		origSize = 1
	}
	lastPct := 0.0
	if err := compress.PackDir(ctx, c, g.GameDir, gamePath, compress.LevelMax, GameExcludes(g), func(bytes int64) {
		if progress != nil {
			pct := float64(bytes) / float64(origSize)
			if pct > 1 {
				pct = 1
			}
			frac := 0.05 + 0.7*pct
			if frac-lastPct > 0.02 {
				lastPct = frac
				progress("archiving game files", frac)
			}
		}
	}); err != nil {
		return nil, fmt.Errorf("pack game: %w", err)
	}
	if progress != nil {
		progress("packing saves", 0.78)
	}
	saveFiles, err := saves.Bundle(ctx, c, g.AppID, savesPath)
	if err != nil {
		return nil, fmt.Errorf("pack saves: %w", err)
	}

	hasPrefix := false
	if c.WithPrefix && g.HasPrefix {
		if progress != nil {
			progress("packing proton prefix", 0.84)
		}
		if err := compress.PackDir(ctx, c, c.PrefixDir(g.AppID), prefixPath, compress.LevelMax, PrefixExcludes(), func(bytes int64) {
			if progress != nil && bytes%50000000 < 100 {
				progress("packing proton prefix", 0.84+0.12)
			}
		}); err != nil {
			return nil, fmt.Errorf("pack prefix: %w", err)
		}
		hasPrefix = true
	}
	if progress != nil {
		progress("hashing + manifest", 0.97)
	}

	gameSHA, err := compress.SHA256Sum(gamePath)
	if err != nil {
		return nil, err
	}
	savesSHA := ""
	if saveFiles > 0 {
		savesSHA, err = compress.SHA256Sum(savesPath)
		if err != nil {
			return nil, err
		}
	}
	prefixSHA := ""
	if hasPrefix {
		prefixSHA, err = compress.SHA256Sum(prefixPath)
		if err != nil {
			return nil, err
		}
	}
	archivedSize := statSize(gamePath)
	entries := saves.Discover(c, g.AppID)

	m := &Manifest{
		Name:         g.Name,
		AppID:        g.AppID,
		Exe:          g.Exe,
		StartDir:     g.StartDir,
		GameDir:      g.GameDir,
		GameDirType:  g.GameDirType,
		OrigSize:     g.SizeBytes,
		ArchivedSize: archivedSize,
		Ratio:        float64(archivedSize) / float64(origSize),
		SHA256:       gameSHA,
		SHA256Saves:  savesSHA,
		SHA256Prefix: prefixSHA,
		Zstd:         ZstdInfo{Version: "1.5.7", Level: 22, LongWindowLog: 27, Threads: 1},
		SavesCount:   saveFiles,
		SaveEntries:  FromSaveEntries(entries),
		HasPrefix:    hasPrefix,
		PackDate:     nowISO(),
		ToolVersion:  ToolVersion,
	}
	if err := WriteManifest(filepath.Join(outDir, "manifest.json"), m); err != nil {
		return nil, err
	}
	if progress != nil {
		progress("done", 1.0)
	}

	// remove empty saves/prefix stubs if nothing was packed
	if saveFiles == 0 {
		_ = os.Remove(savesPath)
	}
	if !hasPrefix {
		_ = os.Remove(prefixPath)
	}

	return &PackResult{
		GamePath:   gamePath,
		SavesPath:  savesPath,
		PrefixPath: prefixPath,
		Manifest:   m,
		Slug:       slug,
		SaveFiles:  saveFiles,
	}, nil
}

func statSize(p string) int64 {
	if fi, err := os.Stat(p); err == nil {
		return fi.Size()
	}
	return 0
}
