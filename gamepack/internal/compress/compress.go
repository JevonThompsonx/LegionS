package compress

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gamepack/internal/cfg"
)

type Level int

const (
	LevelMax Level = iota // zstd --ultra -22 --long=27 (PC games + saves)
	LevelROM              // zstd -3 (already-compressed ROM containers)
)

func zstdArgs(level Level, outPath string) []string {
	switch level {
	case LevelROM:
		return []string{"-3", "-T1", "--no-progress", "-o", outPath}
	default:
		return []string{"-T1", "--ultra", "-22", "--long=27", "--no-progress", "-o", outPath}
	}
}

// PackDir streams `tar -C parent name [excludes] | zstd <level>` to outPath.
// progress is called periodically with the number of bytes written so far.
func PackDir(ctx context.Context, c *cfg.Config, srcDir, outPath string, level Level, excludes []string, progress func(bytes int64)) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	parent := filepath.Dir(srcDir)
	name := filepath.Base(srcDir)
	tarArgs := []string{"-c", "--acls", "--xattrs", "--sparse", "-C", parent}
	for _, e := range excludes {
		tarArgs = append(tarArgs, "--exclude="+name+"/"+e)
	}
	tarArgs = append(tarArgs, name)
	return PackDirArgs(ctx, c, tarArgs, outPath, level, progress)
}

// PackDirArgs pipes an arbitrary `tar -c` arg list through zstd to outPath.
// Used for multi-source packs (save bundles). tarArgs must be a complete tar
// command line starting with "-c".
func PackDirArgs(ctx context.Context, c *cfg.Config, tarArgs []string, outPath string, level Level, progress func(bytes int64)) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	// zstd refuses to overwrite an existing file when reading from stdin
	// (safety), so drop any stale/partial target first.
	_ = os.Remove(outPath)
	tarCmd := exec.CommandContext(ctx, c.TarBin, tarArgs...)
	zstdCmd := exec.CommandContext(ctx, c.ZstdBin, zstdArgs(level, outPath)...)

	pr, pw := io.Pipe()
	tarCmd.Stdout = pw
	zstdCmd.Stdin = pr
	zstdCmd.Stdout = io.Discard

	if err := zstdCmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return fmt.Errorf("start zstd: %w", err)
	}
	if err := tarCmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		_ = zstdCmd.Process.Kill()
		return fmt.Errorf("start tar: %w", err)
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		if progress == nil {
			return
		}
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if fi, err := os.Stat(outPath); err == nil {
					progress(fi.Size())
				}
			}
		}
	}()

	tarErr := tarCmd.Wait()
	pw.Close()
	zstdErr := zstdCmd.Wait()
	close(stop)
	<-done
	if progress != nil {
		if fi, err := os.Stat(outPath); err == nil {
			progress(fi.Size())
		}
	}
	if tarErr != nil {
		return fmt.Errorf("tar failed: %w", tarErr)
	}
	if zstdErr != nil {
		return fmt.Errorf("zstd failed: %w", zstdErr)
	}
	return nil
}

// Extract streams `tar --zstd -xf archive -C dest`.
func Extract(ctx context.Context, c *cfg.Config, archive, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	args := []string{"-x", "--zstd", "-f", archive, "-C", dest}
	cmd := exec.CommandContext(ctx, c.TarBin, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract %s: %w", archive, err)
	}
	return nil
}

// SHA256Sum returns the hex sha256 of a file.
func SHA256Sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return sha256Hex(f)
}
