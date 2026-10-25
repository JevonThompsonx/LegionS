package pack

import (
	"encoding/json"
	"os"

	"gamepack/internal/catalog"
	"gamepack/internal/saves"
)

type ZstdInfo struct {
	Version       string `json:"version"`
	Level         int    `json:"level"`
	LongWindowLog int    `json:"longWindowLog"`
	Threads       int    `json:"threads"`
}

type SaveEntryManifest struct {
	GameSavesRel string   `json:"gamesavesRel"`
	ProtonRel    string   `json:"protonRel"`
	Files        []string `json:"files,omitempty"`
}

type Manifest struct {
	Name         string              `json:"name"`
	AppID        uint32              `json:"appid"`
	Exe          string              `json:"exe"`
	StartDir     string              `json:"startDir"`
	GameDir      string              `json:"gameDir"`
	GameDirType  catalog.GameDirType `json:"gameDirType"`
	OrigSize     int64               `json:"origSize"`
	ArchivedSize int64               `json:"archivedSize"`
	Ratio        float64             `json:"ratio"`
	SHA256       string              `json:"sha256"`
	SHA256Saves  string              `json:"sha256Saves,omitempty"`
	SHA256Prefix string              `json:"sha256Prefix,omitempty"`
	Zstd         ZstdInfo            `json:"zstd"`
	SavesCount   int                 `json:"savesCount"`
	SaveEntries  []SaveEntryManifest `json:"saveEntries"`
	HasPrefix    bool                `json:"hasPrefix"`
	PackDate     string              `json:"packDate"`
	ToolVersion  string              `json:"toolVersion"`
}

func FromSaveEntries(entries []saves.SaveEntry) []SaveEntryManifest {
	out := make([]SaveEntryManifest, 0, len(entries))
	for _, e := range entries {
		out = append(out, SaveEntryManifest{
			GameSavesRel: e.GameSavesRel,
			ProtonRel:    e.ProtonRel,
			Files:        e.Files,
		})
	}
	return out
}

func WriteManifest(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
