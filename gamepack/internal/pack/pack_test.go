package pack

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gamepack/internal/catalog"
	"gamepack/internal/cfg"
)

func TestPackPartialDirIsNotAlreadyDone(t *testing.T) {
	root := t.TempDir()
	c := cfg.Default()
	c.ArchiveRoot = filepath.Join(root, "Archive")
	c.CompatData = filepath.Join(root, "compatdata")
	c.ZstdBin = "zstd"
	c.TarBin = "tar"

	gameDir := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(gameDir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameDir, "data", "file.bin"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := catalog.Game{
		Name:        "Test Game",
		AppID:       123456789,
		Exe:         filepath.Join(gameDir, "data", "file.bin"),
		GameDir:     gameDir,
		GameDirType: catalog.DirTypePlain,
		SizeBytes:   3,
	}

	slug := catalog.Slugify(g.Name, g.AppID)
	outDir := c.PCArchiveDir(slug)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// simulate a crash mid-pack: archive file present, manifest missing
	if err := os.WriteFile(filepath.Join(outDir, "game.tar.zst"), []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Pack(context.Background(), c, g, nil)
	if err != nil {
		t.Fatalf("pack should proceed over a partial dir: %v", err)
	}
	if res.AlreadyDone {
		t.Fatal("partial dir without manifest must NOT be reported as already done")
	}
	if _, err := os.Stat(filepath.Join(outDir, "manifest.json")); err != nil {
		t.Fatalf("manifest should have been written after re-pack: %v", err)
	}

	// second pack with a complete bundle IS already done
	res2, err := Pack(context.Background(), c, g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res2.AlreadyDone {
		t.Fatal("complete bundle should be reported already done")
	}
}
