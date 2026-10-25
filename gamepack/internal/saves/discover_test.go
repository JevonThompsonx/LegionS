package saves

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverLiveSymlinks(t *testing.T) {
	root := t.TempDir()
	gs := filepath.Join(root, "GameSaves") // full GameSaves root (mirrors real cfg)
	prefix := filepath.Join(root, "compatdata", "999", "pfx", "drive_c", "users", "steamuser", "AppData", "Local")
	if err := os.MkdirAll(filepath.Join(gs, "appdata-local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gs, "appdata-local", "SomeGame"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(gs, "appdata-local", "SomeGame"), filepath.Join(prefix, "SomeGame")); err != nil {
		t.Fatal(err)
	}
	// unrelated symlink outside GameSaves must be ignored
	if err := os.MkdirAll(filepath.Join(root, "other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "other"), filepath.Join(prefix, "OtherLink")); err != nil {
		t.Fatal(err)
	}
	got := DiscoverLiveSymlinks(filepath.Join(root, "compatdata", "999"), gs)
	if len(got) != 1 {
		t.Fatalf("expected 1 mapping, got %d: %+v", len(got), got)
	}
	if got[0].GameSavesRel != "appdata-local/SomeGame" {
		t.Errorf("GameSavesRel = %q", got[0].GameSavesRel)
	}
	if got[0].ProtonRel != "users/steamuser/AppData/Local/SomeGame" {
		t.Errorf("ProtonRel = %q", got[0].ProtonRel)
	}
}

func TestGoldenForFiltersMissingDirs(t *testing.T) {
	root := t.TempDir()
	c := cfgForTest(root)
	// no GameSaves dirs exist -> no entries even though golden has Split Fiction
	if got := GoldenFor(c, 2508814786); len(got) != 0 {
		t.Errorf("expected no golden entries without dirs, got %d", len(got))
	}
	gs := c.GameSavesRelPath("appdata-local/SplitFiction")
	if err := os.MkdirAll(gs, 0o755); err != nil {
		t.Fatal(err)
	}
	got := GoldenFor(c, 2508814786)
	if len(got) != 1 || got[0].GameSavesRel != "appdata-local/SplitFiction" {
		t.Errorf("golden after dir creation: %+v", got)
	}
}
