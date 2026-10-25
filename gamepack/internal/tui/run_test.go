package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gamepack/internal/cfg"
)

func TestRunRemovePromptKeys(t *testing.T) {
	m := initialModel(cfg.Default())
	m.cur = scrRun
	m.run = runModel{resp: make(chan bool, 1), pending: &removePrompt{slug: "s", name: "Game", gameDir: "/x"}}

	// 'n' keeps the folder
	nm, _ := m.runUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	mm := nm.(*model)
	if mm.run.pending != nil {
		t.Error("pending should be cleared after answering")
	}
	if got := <-mm.run.resp; got {
		t.Error("n should answer false")
	}

	// 'y' removes
	m2 := initialModel(cfg.Default())
	m2.cur = scrRun
	m2.run = runModel{resp: make(chan bool, 1), pending: &removePrompt{slug: "s", name: "Game", gameDir: "/x"}}
	nm2, _ := m2.runUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	mm2 := nm2.(*model)
	if got := <-mm2.run.resp; !got {
		t.Error("y should answer true")
	}

	// pending prompt renders
	if v := m2.run.View(); v == "" {
		t.Error("pending prompt view should not be empty")
	}
}

func TestRunStatusProgress(t *testing.T) {
	m := initialModel(cfg.Default())
	m.cur = scrRun
	m.run = runModel{statuses: map[string]string{}, items: []listItem{{slug: "a", label: "A"}, {slug: "b", label: "B"}}, total: 2, done: 0}

	nm, cmd := m.runUpdate(runStatusMsg{slug: "a", line: "working"})
	mm := nm.(*model)
	if mm.run.statuses["a"] != "working" {
		t.Errorf("status not recorded: %q", mm.run.statuses["a"])
	}
	if cmd == nil {
		t.Error("expected channel-read cmd after non-final status")
	}
	if mm.run.done != 0 {
		t.Errorf("done should be 0 until final, got %d", mm.run.done)
	}

	// final for both items -> runCompleteMsg
	nm2, _ := mm.runUpdate(runStatusMsg{slug: "a", line: "done", final: true})
	mm2 := nm2.(*model)
	nm3, _ := mm2.runUpdate(runStatusMsg{slug: "b", line: "done", final: true})
	mm3 := nm3.(*model)
	if mm3.run.done != 2 {
		t.Errorf("done = %d, want 2", mm3.run.done)
	}
}
