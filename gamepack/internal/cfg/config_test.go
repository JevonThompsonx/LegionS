package cfg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLibraryCompatDataRootsAndHasPrefix(t *testing.T) {
	root := t.TempDir()
	primary := filepath.Join(root, "Steam")
	secondary := filepath.Join(root, "GamesData", "SteamLibrary")

	// primary library folder + libraryfolders.vdf listing both
	if err := os.MkdirAll(filepath.Join(primary, "steamapps", "compatdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(secondary, "steamapps", "compatdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	lvf := `"libraryfolders"
{
	"0"
	{
		"path"		"` + primary + `"
	}
	"1"
	{
		"path"		"` + secondary + `"
	}
}`
	if err := os.WriteFile(filepath.Join(primary, "steamapps", "libraryfolders.vdf"), []byte(lvf), 0o644); err != nil {
		t.Fatal(err)
	}

	// prefix only in the SECONDARY library
	pfx := filepath.Join(secondary, "steamapps", "compatdata", "2955550478", "pfx")
	if err := os.MkdirAll(pfx, 0o755); err != nil {
		t.Fatal(err)
	}

	c := Default()
	c.CompatData = filepath.Join(primary, "steamapps", "compatdata")

	roots := c.LibraryCompatDataRoots()
	if len(roots) != 2 {
		t.Fatalf("expected 2 compatdata roots, got %d: %v", len(roots), roots)
	}
	if !c.HasPrefix(2955550478) {
		t.Error("HasPrefix should find prefix in secondary library")
	}
	if c.HasPrefix(999999999) {
		t.Error("HasPrefix should be false for unknown appid")
	}

	// without the libraryfolders.vdf, only primary is checked
	if err := os.Remove(filepath.Join(primary, "steamapps", "libraryfolders.vdf")); err != nil {
		t.Fatal(err)
	}
	if c.HasPrefix(2955550478) {
		t.Error("HasPrefix should not find secondary prefix without libraryfolders.vdf")
	}
}
