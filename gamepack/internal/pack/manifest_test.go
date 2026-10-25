package pack

import (
	"os"
	"path/filepath"
	"testing"

	"gamepack/internal/catalog"
	"gamepack/internal/saves"
)

func TestManifestRoundTrip(t *testing.T) {
	m := &Manifest{
		Name:         "Split Fiction",
		AppID:        2508814786,
		Exe:          "/run/media/system/Games_Data/Split Fictionx/drive_c/Games/Split Fiction/Split/Binaries/Win64/SplitFiction.exe",
		GameDir:      "/run/media/system/Games_Data/Split Fictionx",
		GameDirType:  catalog.DirTypePrefixContainer,
		OrigSize:     95563038900,
		ArchivedSize: 72500000000,
		Ratio:        0.76,
		SHA256:       "abc123",
		Zstd:         ZstdInfo{Version: "1.5.7", Level: 22, LongWindowLog: 27, Threads: 1},
		SavesCount:   4,
		SaveEntries: FromSaveEntries([]saves.SaveEntry{
			{GameSavesRel: "appdata-local/SplitFiction", ProtonRel: "users/steamuser/AppData/Local/SplitFiction", Files: []string{"SaveData.Split"}},
		}),
		HasPrefix:   true,
		ToolVersion: ToolVersion,
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "manifest.json")
	if err := WriteManifest(p, m); err != nil {
		t.Fatal(err)
	}
	got, err := ReadManifest(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != m.Name || got.AppID != 2508814786 || got.SavesCount != 4 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.SaveEntries[0].GameSavesRel != "appdata-local/SplitFiction" {
		t.Errorf("save entry mismatch: %+v", got.SaveEntries[0])
	}
	if got.GameDirType != catalog.DirTypePrefixContainer {
		t.Errorf("gameDirType = %q", got.GameDirType)
	}
	_ = os.Remove(p)
}

func TestGameExcludes(t *testing.T) {
	plain := GameExcludes(catalog.Game{GameDirType: catalog.DirTypePlain})
	if len(plain) != 0 {
		t.Errorf("plain game should have no excludes, got %v", plain)
	}
	container := GameExcludes(catalog.Game{GameDirType: catalog.DirTypePrefixContainer})
	found := false
	for _, e := range container {
		if e == "shadercache" {
			found = true
		}
	}
	if !found {
		t.Errorf("container excludes should include shadercache: %v", container)
	}
}
