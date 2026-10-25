package unpack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"gamepack/internal/cfg"
	"gamepack/internal/compress"
	"gamepack/internal/pack"
	"gamepack/internal/proton"
	"gamepack/internal/saves"
)

// Options for an unpack run.
type Options struct {
	// Interactive, if true, pauses and launches Steam when the prefix needs
	// building. If false, prints instructions and proceeds without saves.
	Interactive bool
	OnStatus    func(string)
}

// Preflight checks disk space and running-game state before extraction.
func Preflight(c *cfg.Config, m *pack.Manifest) error {
	if _, err := os.Stat(m.GameDir); err == nil {
		return fmt.Errorf("%s already present at %s (use --force to overwrite)", m.Name, m.GameDir)
	}
	free, err := freeBytes(c.ArchiveRoot)
	if err != nil {
		return err
	}
	need := m.OrigSize
	if m.HasPrefix && m.SHA256Prefix != "" {
		need += prefixUnpackBudget()
	}
	if free < need {
		return fmt.Errorf("not enough disk space: need %d bytes, have %d", need, free)
	}
	if proton.IsRunning(c, m.AppID, filepath.Base(m.Exe)) {
		return fmt.Errorf("%s appears to be running — close it first", m.Name)
	}
	return nil
}

// Unpack restores a game from its archive bundle: game dir, then prefix (from
// prefix.tar.zst or via interactive Steam launch), then saves + symlinks.
func Unpack(ctx context.Context, c *cfg.Config, m *pack.Manifest, slug string, opts Options) error {
	if opts.OnStatus != nil {
		opts.OnStatus("verifying archive checksum")
	}
	sum, err := compress.SHA256Sum(filepath.Join(c.PCArchiveDir(slug), "game.tar.zst"))
	if err != nil {
		return err
	}
	if m.SHA256 != "" && sum != m.SHA256 {
		return fmt.Errorf("checksum mismatch for %s (got %s want %s)", m.Name, sum, m.SHA256)
	}
	if opts.OnStatus != nil {
		opts.OnStatus("extracting game")
	}
	stage := m.GameDir + ".unpack-staging"
	_ = safeRemoveAll(stage, m.GameDir)
	if err := compress.Extract(ctx, c, filepath.Join(c.PCArchiveDir(slug), "game.tar.zst"), stage); err != nil {
		return err
	}
	// tar stores contents of the packed dir (name/...) — move contents up.
	if err := flattenStage(stage, m.GameDir); err != nil {
		return err
	}
	_ = safeRemoveAll(stage, m.GameDir)

	// Prefix: restore from archive if present, else interactive launch flow.
	if m.SHA256Prefix != "" {
		if opts.OnStatus != nil {
			opts.OnStatus("restoring proton prefix")
		}
		if err := restorePrefix(ctx, c, m, slug); err != nil {
			return err
		}
	} else if opts.Interactive {
		if err := interactivePrefix(ctx, c, m, opts.OnStatus); err != nil {
			return err
		}
	} else {
		if opts.OnStatus != nil {
			opts.OnStatus("no prefix in archive — run the game once in Steam to build it, then re-run link-saves.sh")
		}
	}

	if err := restoreSaves(ctx, c, m, slug); err != nil {
		return err
	}
	if opts.OnStatus != nil {
		opts.OnStatus("done")
	}
	return nil
}

func restorePrefix(ctx context.Context, c *cfg.Config, m *pack.Manifest, slug string) error {
	prefixPath := filepath.Join(c.PCArchiveDir(slug), "prefix.tar.zst")
	// tar archive contains <appid>/... so extract into compatdata parent.
	return compress.Extract(ctx, c, prefixPath, c.CompatData)
}

func interactivePrefix(ctx context.Context, c *cfg.Config, m *pack.Manifest, onStatus func(string)) error {
	if !proton.SteamRunning(c) {
		if onStatus != nil {
			onStatus("Steam is not running — launch %s once in Steam, then close it; saves will link on next run")
		}
		return nil
	}
	if onStatus != nil {
		onStatus("launching " + m.Name + " in Steam…")
	}
	if err := proton.LaunchGame(ctx, c, m.AppID); err != nil {
		return err
	}
	if err := proton.WaitPrefixBuilt(ctx, c, m.AppID, 5*time.Minute, onStatus); err != nil {
		return err
	}
	if onStatus != nil {
		onStatus("prefix built — close the game when you reach the main menu")
	}
	return proton.WaitGameExit(ctx, c, m.AppID, filepath.Base(m.Exe), 30*time.Minute, onStatus)
}

func restoreSaves(ctx context.Context, c *cfg.Config, m *pack.Manifest, slug string) error {
	savesPath := filepath.Join(c.PCArchiveDir(slug), "saves.tar.zst")
	if _, err := os.Stat(savesPath); err != nil {
		return linkOnly(c, m)
	}
	stage := filepath.Join(c.PCArchiveDir(slug), ".saves-staging")
	_ = os.RemoveAll(stage)
	if err := compress.Extract(ctx, c, savesPath, stage); err != nil {
		return fmt.Errorf("extract saves: %w", err)
	}
	gsRoot := filepath.Join(stage, "gamesaves")
	_ = filepath.WalkDir(gsRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(gsRoot, p)
		dest := filepath.Join(c.GameSaves, rel)
		fi, _ := d.Info()
		if _, err := saves.RestoreFile(p, dest, fi.ModTime().Unix()); err != nil {
			return err
		}
		return nil
	})
	_ = os.RemoveAll(stage)
	return linkOnly(c, m)
}

func linkOnly(c *cfg.Config, m *pack.Manifest) error {
	if m.SHA256Prefix == "" {
		return nil // prefix not restored here; nothing to link against
	}
	prefix := filepath.Join(c.PrefixDir(m.AppID), "pfx", "drive_c")
	var linked int
	for _, e := range m.SaveEntries {
		source := c.GameSavesRelPath(e.GameSavesRel)
		if _, err := os.Stat(source); err != nil {
			continue
		}
		target := filepath.Join(prefix, filepath.FromSlash(e.ProtonRel))
		if err := saves.LinkDir(source, target); err != nil {
			return err
		}
		linked++
	}
	return nil
}

func flattenStage(stage, dest string) error {
	entries, err := os.ReadDir(stage)
	if err != nil {
		return err
	}
	// tar member is "<packedname>/..." — move its contents to dest.
	src := stage
	if len(entries) == 1 && entries[0].IsDir() {
		src = filepath.Join(stage, entries[0].Name())
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	return moveContents(src, dest)
}

func moveContents(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		from := filepath.Join(src, e.Name())
		to := filepath.Join(dest, e.Name())
		if err := os.Rename(from, to); err != nil {
			// cross-device fallback
			if err2 := copyTree(from, to); err2 != nil {
				return err2
			}
			_ = os.RemoveAll(from)
		}
	}
	return nil
}

func copyTree(src, dest string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, fi.Mode())
	}
	if err := os.MkdirAll(dest, fi.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dest, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func freeBytes(dir string) (int64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return int64(st.Bavail) * int64(st.Bsize), nil
}

func prefixUnpackBudget() int64 { return 1 << 30 } // 1 GiB conservative
