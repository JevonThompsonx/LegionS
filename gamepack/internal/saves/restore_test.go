package saves

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRestoreFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.sav")
	dst := filepath.Join(dir, "dst", "src.sav")
	if err := os.WriteFile(src, []byte("progress"), 0o644); err != nil {
		t.Fatal(err)
	}

	// archived mtime in the future vs fresh dest -> restore
	restored, err := RestoreFile(src, dst, time.Now().Add(time.Hour).Unix())
	if err != nil {
		t.Fatal(err)
	}
	if !restored {
		t.Error("expected restore when dest missing")
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "progress" {
		t.Errorf("restored content = %q", data)
	}

	// dest now exists with mtime ~now; archived mtime in the past -> skip-newer
	restored2, err := RestoreFile(src, dst, time.Now().Add(-time.Hour).Unix())
	if err != nil {
		t.Fatal(err)
	}
	if restored2 {
		t.Error("expected skip when dest is newer than archived copy")
	}
}

func TestLinkDir(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "nested", "target")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := LinkDir(source, target); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected symlink")
	}
	if got, _ := os.Readlink(target); got != source {
		t.Errorf("link target = %q, want %q", got, source)
	}

	// idempotent: re-link to same source -> no error, still symlink
	if err := LinkDir(source, target); err != nil {
		t.Fatalf("idempotent relink failed: %v", err)
	}

	// real dir at target gets moved aside
	other := filepath.Join(dir, "other")
	real := filepath.Join(dir, "real-target")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LinkDir(other, real); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(real); fi.Mode()&os.ModeSymlink == 0 {
		t.Error("real dir should have been replaced by symlink")
	}
	// old dir backed up
	baks, _ := filepath.Glob(real + ".bak.*")
	if len(baks) == 0 {
		t.Error("expected a .bak backup of the replaced real dir")
	}
}
