package nswitch

import (
	"os"
	"path/filepath"
	"testing"

	"gamepack/internal/cfg"
)

func TestMatchTitleInName(t *testing.T) {
	if id := MatchTitle("Super Mario Bros. Wonder [010015100B514000][v0][US].xci"); id != "010015100B514000" {
		t.Errorf("MatchTitle from filename = %q", id)
	}
	if id := MatchTitle("Battle Chef Brigade Deluxe.nsz"); id != "" {
		t.Errorf("expected no title for plain name, got %q", id)
	}
	CuratedTitleIDMap["my game"] = "0100ABCDEF000001"
	defer delete(CuratedTitleIDMap, "my game")
	if id := MatchTitle("My Game.xci"); id != "0100ABCDEF000001" {
		t.Errorf("curated match = %q", id)
	}
}

func TestScanROMS(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.nsp", "b.xci", "c.nsz", "readme.txt", "d.xcz"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := cfg.Default()
	c.NSwitchDir = dir
	roms, err := ScanROMS(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(roms) != 4 {
		t.Fatalf("expected 4 ROMs (readme.txt excluded), got %d", len(roms))
	}
}
