package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gamepack/internal/cfg"
)

func TestMenuViewShowsNumbers(t *testing.T) {
	v := newMenu().View()
	for _, n := range []string{"1", "2", "3", "4", "5"} {
		if !strings.Contains(v, n) {
			t.Errorf("menu view missing number %s:\n%s", n, v)
		}
	}
	for _, s := range []string{"Pack games (PC)", "Unpack games (PC)", "Pack NSwitch ROMs", "Unpack NSwitch ROMs", "Quit"} {
		if !strings.Contains(v, s) {
			t.Errorf("menu view missing %q", s)
		}
	}
}

func TestMenuNumberSelect(t *testing.T) {
	m := newMenu()
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if nm.cur != 2 {
		t.Fatalf("cursor should be on item 3, got %d", nm.cur)
	}
	got := cmd()
	ms, ok := got.(menuSelectMsg)
	if !ok || ms.kind != taskPackNS {
		t.Errorf("expected menuSelectMsg taskPackNS, got %#v", got)
	}
}

func TestListNumberToggleAndRender(t *testing.T) {
	m := newList(cfg.Default(), taskPackPC, 24)
	m.items = []listItem{
		{label: "Alpha", slug: "alpha", size: 1000},
		{label: "Beta", slug: "beta", size: 2000},
		{label: "Gamma", slug: "gamma", size: 3000, note: "no prefix"},
	}
	m.loading = false
	m.ready = true

	// press "2" -> toggles Beta, cursor moves to it
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if !nm.items[1].checked {
		t.Error("item 2 (Beta) should be checked after pressing 2")
	}
	if nm.cur != 1 {
		t.Errorf("cursor should be on item 2, got %d", nm.cur)
	}

	// press "2" again -> untoggles
	nm2, _ := nm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if nm2.items[1].checked {
		t.Error("item 2 should be unchecked after second press")
	}

	// space toggles current
	nm3, _ := nm.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !nm3.items[1].checked {
		t.Error("space should toggle current row")
	}

	v := nm3.View()
	for _, want := range []string{"1", "2", "3", "Alpha", "Beta", "Gamma", "no prefix", "[x]"} {
		if !strings.Contains(v, want) {
			t.Errorf("list view missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "[x] 1") {
		t.Errorf("row numbers should be 2-char padded; saw %q", "[x] 1")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("a b c d e f g h i j k l m n o p q r s t u v", 10); len([]rune(got)) != 10 {
		t.Errorf("truncate len = %d, want 10 (%q)", len([]rune(got)), got)
	}
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate should not cut short strings, got %q", got)
	}
}
