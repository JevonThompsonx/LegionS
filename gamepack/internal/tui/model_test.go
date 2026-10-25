package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gamepack/internal/cfg"
)

func TestModelRouting(t *testing.T) {
	m := initialModel(cfg.Default())

	// menu select -> list screen
	nm, _ := m.Update(menuSelectMsg{kind: taskPackPC})
	mm := nm.(*model)
	if mm.cur != scrList {
		t.Errorf("cur = %v, want scrList", mm.cur)
	}

	// WindowSizeMsg sets height and stays on list
	nm2, _ := mm.Update(tea.WindowSizeMsg{Height: 30, Width: 100})
	mm2 := nm2.(*model)
	if mm2.height != 30 {
		t.Errorf("model height = %d, want 30", mm2.height)
	}
	if mm2.cur != scrList {
		t.Errorf("cur = %v, want scrList after resize", mm2.cur)
	}

	// Esc from list -> back to menu
	nm3, _ := mm2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm3 := nm3.(*model)
	if mm3.cur != scrMenu {
		t.Errorf("cur = %v, want scrMenu after esc", mm3.cur)
	}
}

func TestModelCtrlCQuits(t *testing.T) {
	m := initialModel(cfg.Default())
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Ctrl-C should return a quit command")
	}
	_ = nm
}

func TestMenuViewNonEmpty(t *testing.T) {
	if v := newMenu().View(); v == "" {
		t.Error("menu view empty")
	}
}
