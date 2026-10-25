package compress

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gamepack/internal/cfg"
)

func testCfg() *cfg.Config {
	c := cfg.Default()
	if _, err := os.Stat(c.ZstdBin); err != nil {
		c.ZstdBin = "zstd"
	}
	if _, err := os.Stat(c.TarBin); err != nil {
		c.TarBin = "tar"
	}
	return c
}

func TestPackExtractRoundTrip(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	sub := filepath.Join(src, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "file.bin"), make([]byte, 1<<20), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "top.txt"), []byte("hello gamepack"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := testCfg()
	out := filepath.Join(root, "out", "game.tar.zst")
	var got int64
	if err := PackDir(context.Background(), c, src, out, LevelROM, nil, func(b int64) { got = b }); err != nil {
		t.Fatalf("pack: %v", err)
	}
	if got == 0 {
		t.Fatal("progress never reported bytes")
	}
	if fi, _ := os.Stat(out); fi.Size() == 0 {
		t.Fatal("archive is empty")
	}
	sum, err := SHA256Sum(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum) != 64 {
		t.Fatalf("sha256 len = %d", len(sum))
	}

	dest := filepath.Join(root, "dest")
	if err := Extract(context.Background(), c, out, dest); err != nil {
		t.Fatalf("extract: %v", err)
	}
	got2, err := SHA256Sum(filepath.Join(dest, "src", "sub", "file.bin"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := SHA256Sum(filepath.Join(src, "sub", "file.bin"))
	if got2 != want {
		t.Error("extracted file differs from source")
	}
}

func TestPackExtractMaxLevel(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 0)
	for i := 0; i < 50000; i++ {
		data = append(data, []byte("repeatable-game-asset-pattern-")...)
	}
	if err := os.WriteFile(filepath.Join(src, "big.bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	c := testCfg()
	out := filepath.Join(root, "out.tar.zst")
	if err := PackDir(context.Background(), c, src, out, LevelMax, nil, nil); err != nil {
		t.Fatalf("pack max: %v", err)
	}
	dest := filepath.Join(root, "dest")
	if err := Extract(context.Background(), c, out, dest); err != nil {
		t.Fatalf("extract max: %v", err)
	}
	a, _ := SHA256Sum(filepath.Join(src, "big.bin"))
	b, _ := SHA256Sum(filepath.Join(dest, "src", "big.bin"))
	if a != b {
		t.Error("max-level round trip mismatch")
	}
}
