package saves

import (
	"path/filepath"

	"gamepack/internal/cfg"
)

func cfgForTest(root string) *cfg.Config {
	c := cfg.Default()
	c.GameSaves = filepath.Join(root, "GameSaves")
	c.CompatData = filepath.Join(root, "compatdata")
	return c
}
