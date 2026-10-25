package saves

import (
	"testing"
)

func TestGoldenForSplitFiction(t *testing.T) {
	// GameSaves dirs live on the ROG; golden table itself is static. Verify the
	// mapping table contains the verified entries.
	var found int
	for _, m := range Golden {
		if m.AppID == 2508814786 && m.GameSavesRel == "appdata-local/SplitFiction" {
			if m.ProtonRel != "users/steamuser/AppData/Local/SplitFiction" {
				t.Errorf("Split Fiction ProtonRel = %q", m.ProtonRel)
			}
			found++
		}
		if m.AppID == 4076669000 {
			found++ // Witcher 3 dual mapping
		}
	}
	if found < 2 {
		t.Errorf("golden table missing expected entries (found %d)", found)
	}
}

func TestUnder(t *testing.T) {
	if rel, ok := under("/a/b", "/a/b/c/d"); !ok || rel != "c/d" {
		t.Errorf("under = %q, %v", rel, ok)
	}
	if _, ok := under("/a/b", "/a/x"); ok {
		t.Errorf("under should reject sibling")
	}
	if _, ok := under("/a/b", "/a/b"); !ok {
		t.Errorf("under should accept exact root")
	}
}
