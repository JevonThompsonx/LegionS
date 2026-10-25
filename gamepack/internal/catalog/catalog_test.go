package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"gamepack/internal/cfg"
)

func TestParseShortcutsVDFGolden(t *testing.T) {
	sc, err := ParseShortcutsVDF("../../testdata/shortcuts.vdf")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(sc) != 63 {
		t.Fatalf("expected 63 shortcuts, got %d", len(sc))
	}
	byName := map[string]Shortcut{}
	for _, s := range sc {
		byName[s.Name] = s
	}
	sf := byName["Split Fiction"]
	if sf.AppID != 2508814786 {
		t.Errorf("Split Fiction appid = %d, want 2508814786", sf.AppID)
	}
	if sf.Exe != "/run/media/system/Games_Data/Split Fictionx/drive_c/Games/Split Fiction/Split/Binaries/Win64/SplitFiction.exe" {
		t.Errorf("Split Fiction exe wrong: %q", sf.Exe)
	}
	if got := byName["Cuphead"].AppID; got != 2986239671 {
		t.Errorf("Cuphead appid = %d, want 2986239671", got)
	}
	if got := byName["Hades"].AppID; got != 2955550478 {
		t.Errorf("Hades appid = %d, want 2955550478", got)
	}
}

func TestResolveGameDirPrefixContainer(t *testing.T) {
	root := t.TempDir()
	container := filepath.Join(root, "Split Fictionx")
	win64 := filepath.Join(container, "drive_c", "Games", "Split Fiction", "Split", "Binaries", "Win64")
	if err := os.MkdirAll(win64, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(container, "system.reg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(win64, "SplitFiction.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, typ := ResolveGameDir(exe)
	if dir != container {
		t.Errorf("prefix container resolution = %q, want %q", dir, container)
	}
	if typ != DirTypePrefixContainer {
		t.Errorf("type = %v, want prefix-container", typ)
	}
}

func TestResolveGameDirPlain(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "Cuphead")
	if err := os.MkdirAll(game, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(game, "Cuphead.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, typ := ResolveGameDir(exe)
	if dir != game {
		t.Errorf("plain resolution = %q, want %q", dir, game)
	}
	if typ != DirTypePlain {
		t.Errorf("type = %v, want plain", typ)
	}
}

func TestExcludeReason(t *testing.T) {
	cases := []struct {
		s      Shortcut
		excl   bool
		reason string
	}{
		{Shortcut{AppID: 2379889317, Name: "Linux-Yuzu", Exe: "/run/media/system/Games_Data/Linux-Yuzu-EA-4176.AppImage"}, true, "Switch emulator"},
		{Shortcut{AppID: 2587493377, Name: "Ascendara", Exe: "/var/home/jevonx/.local/bin/Ascendara.AppImage"}, true, "app installer tool"},
		{Shortcut{AppID: 2224065267, Name: "Octopath Traveler 0", Exe: "/run/media/system/Games_Data/OCTOPATH TRAVELER 0/citron-cmd.exe"}, true, "Switch (Citron) launcher"},
		{Shortcut{AppID: 2508814786, Name: "Split Fiction", Exe: "/run/media/system/Games_Data/Split Fictionx/drive_c/Games/Split Fiction/Split/Binaries/Win64/SplitFiction.exe"}, false, ""},
	}
	for _, c := range cases {
		reason, ok := ExcludeReason(c.s)
		if ok != c.excl {
			t.Errorf("%s: exclude=%v want %v", c.s.Name, ok, c.excl)
		}
		if ok && reason != c.reason {
			t.Errorf("%s: reason=%q want %q", c.s.Name, reason, c.reason)
		}
	}
}

func TestSlugify(t *testing.T) {
	if got := Slugify("Split Fiction", 2508814786); got != "split-fiction-2508814786" {
		t.Errorf("slug = %q", got)
	}
	if got := Slugify("Cuphead", 2986239671); got != "cuphead-2986239671" {
		t.Errorf("slug = %q", got)
	}
}

func TestBuildCatalogSelfHealsArchived(t *testing.T) {
	c := cfg.Default()
	c.ShortcutsVdf = "../../testdata/shortcuts.vdf"
	root := t.TempDir()
	c.ArchiveRoot = root

	slug := Slugify("Split Fiction", 2508814786)
	adir := c.PCArchiveDir(slug)
	if err := os.MkdirAll(adir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adir, "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	games, err := BuildCatalog(c)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, g := range games {
		if g.Name == "Split Fiction" {
			found = true
			if !g.IsArchived {
				t.Error("Split Fiction should be IsArchived when manifest.json exists")
			}
		}
		if g.Name == "Deltarune" && g.IsArchived {
			t.Error("Deltarune (no manifest) should not be IsArchived")
		}
	}
	if !found {
		t.Error("Split Fiction missing from catalog")
	}
}
