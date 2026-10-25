package unpack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gamepack/internal/cfg"
	"gamepack/internal/compress"
	"gamepack/internal/pack"
)

// assertWithinRoot panics if target is not within root.
// SAFE BEHAVIOR: all destructive operations must validate paths stay within allowed root.
func assertWithinRoot(target, root string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(targetAbs, rootAbs+string(filepath.Separator)) && targetAbs != rootAbs {
		return fmt.Errorf("path %q escapes allowed root %q", target, root)
	}
	return nil
}

// safeRemoveAll removes target only if it's within the allowed root.
func safeRemoveAll(target, root string) error {
	if err := assertWithinRoot(target, root); err != nil {
		return err
	}
	return os.RemoveAll(target)
}

// safeMkdirAll creates dir only if it's within the allowed root.
func safeMkdirAll(target string, perm os.FileMode, root string) error {
	if err := assertWithinRoot(target, root); err != nil {
		return err
	}
	return os.MkdirAll(target, perm)
}

// safeOpenFile opens a file only if it's within the allowed root.
func safeOpenFile(name string, flag int, perm os.FileMode, root string) (*os.File, error) {
	if err := assertWithinRoot(name, root); err != nil {
		return nil, err
	}
	return os.OpenFile(name, flag, perm)
}

// isSafeArchiveEntry reports whether a tar/zip entry path is safe to extract.
// Rejects absolute paths and any path containing ".." components.
func isSafeArchiveEntry(path string) bool {
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\`) {
		return false
	}
	parts := strings.Split(path, "/")
	for _, p := range parts {
		if p == ".." {
			return false
		}
	}
	return true
}

// sanitizeDest joins destDir with entry and verifies the result stays within destDir.
func sanitizeDest(destDir, entry string) (string, error) {
	joined := filepath.Join(destDir, entry)
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	joinedAbs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(joinedAbs, destAbs+string(filepath.Separator)) && joinedAbs != destAbs {
		return "", fmt.Errorf("archive entry %q escapes destination %q", entry, destDir)
	}
	return joined, nil
}

// restorePrefixAtomic extracts prefix to a staging dir, then atomically moves into place.
// SAFE BEHAVIOR: staging + atomic swap prevents broken partial state.
func restorePrefixAtomic(ctx context.Context, c *cfg.Config, m *pack.Manifest, slug string) error {
	prefixPath := filepath.Join(c.PCArchiveDir(slug), "prefix.tar.zst")

	// Stage into a temp dir first
	stageDir, err := os.MkdirTemp("", "prefix-stage-*")
	if err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(stageDir)

	if err := compress.Extract(ctx, c, prefixPath, stageDir); err != nil {
		return fmt.Errorf("extract prefix to staging: %w", err)
	}

	// Atomic move: rename staging dir to final location
	if err := os.Rename(stageDir, c.CompatData); err != nil {
		return fmt.Errorf("atomic move prefix into place: %w", err)
	}

	return nil
}
