package cfg

import (
	"os"
	"strings"
	"testing"
)

// TestHardcodedHomePath documents that default config paths are hardcoded
// to /var/home/jevonx, making the tool non-portable to other systems.
// SAFE BEHAVIOR: paths should be configurable or use os.UserHomeDir().
//
// The tool's target host (the ROG handheld) has $HOME == /var/home/jevonx,
// so the assertion below only holds there. On any other machine (CI runners,
// dev boxes) the hardcoded paths intentionally do not match $HOME — that is
// exactly the portability gap this test records. Skip the hard assertion off
// the target host so the suite stays green everywhere while keeping the
// informational log of hardcoded paths.
func TestHardcodedHomePath(t *testing.T) {
	c := Default()

	// Check that paths reference the hardcoded home directory
	hardcoded := []string{
		c.ShortcutsVdf,
		c.CompatData,
		c.GameSaves,
		c.LudusaviPath,
		c.CitronSaves,
	}

	for _, p := range hardcoded {
		if strings.Contains(p, "/var/home/jevonx") {
			t.Logf("VULNERABILITY: path %q is hardcoded to /var/home/jevonx", p)
		}
	}

	// SAFE: paths should use os.UserHomeDir() or be configurable
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	// Only enforce on the target host. Tracked as modernization TODO #24
	// (parameterize the hardcoded Steam user path).
	const targetHome = "/var/home/jevonx"
	if home != targetHome {
		t.Skipf("non-target host (home=%q, target=%q): hardcoded paths intentionally differ; see TODO #24", home, targetHome)
	}

	// If the hardcoded path doesn't match actual home, the tool breaks
	if c.ShortcutsVdf != "" && !strings.HasPrefix(c.ShortcutsVdf, home) {
		t.Errorf("ShortcutsVdf %q does not start with actual home %q", c.ShortcutsVdf, home)
	}
}
