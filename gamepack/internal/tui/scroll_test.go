package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gamepack/internal/cfg"
)

func mkItems(n int) []listItem {
	out := make([]listItem, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, listItem{label: "Game " + string(rune('A'+i%26)) + " " + itoa(i), slug: itoa(i)})
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func TestMaxOffset(t *testing.T) {
	if got := maxOffset(10, 20); got != 0 {
		t.Errorf("maxOffset(10,20) = %d", got)
	}
	if got := maxOffset(58, 20); got != 38 {
		t.Errorf("maxOffset(58,20) = %d", got)
	}
}

func TestClampOffset(t *testing.T) {
	// cursor below window pulls window down
	if got := clampOffset(0, 25, 58, 20); got != 6 {
		t.Errorf("clampOffset below = %d, want 6", got)
	}
	// cursor above window pushes window up
	if got := clampOffset(20, 5, 58, 20); got != 5 {
		t.Errorf("clampOffset above = %d, want 5", got)
	}
	// list fits: always 0
	if got := clampOffset(5, 5, 10, 20); got != 0 {
		t.Errorf("clampOffset fits = %d, want 0", got)
	}
	// clamp at top
	if got := clampOffset(0, 0, 58, 20); got != 0 {
		t.Errorf("clampOffset top = %d", got)
	}
	// clamp at bottom
	if got := clampOffset(38, 57, 58, 20); got != 38 {
		t.Errorf("clampOffset bottom = %d, want 38", got)
	}
}

func TestListHeight(t *testing.T) {
	if got := listHeight(0); got != 20 {
		t.Errorf("listHeight(0) = %d", got)
	}
	if got := listHeight(10); got != 5 {
		t.Errorf("listHeight(10) = %d, want 5 (min)", got)
	}
	if got := listHeight(50); got != 26 {
		t.Errorf("listHeight(50) = %d, want 26 (max)", got)
	}
	if got := listHeight(30); got != 23 {
		t.Errorf("listHeight(30) = %d, want 23", got)
	}
}

func TestCursorAlwaysVisible(t *testing.T) {
	m := newList(cfg.Default(), taskPackPC, 30)
	m.items = mkItems(58)
	m.loading = false
	m.ready = true
	keys := []tea.KeyMsg{
		{Type: tea.KeyPgDown}, {Type: tea.KeyPgDown}, {Type: tea.KeyPgUp},
		{Type: tea.KeyDown}, {Type: tea.KeyDown}, {Type: tea.KeyPgDown},
		{Type: tea.KeyDown}, {Type: tea.KeyUp}, {Type: tea.KeyPgUp},
		{Type: tea.KeyDown}, {Type: tea.KeyPgDown}, {Type: tea.KeyEnd},
		{Type: tea.KeyHome}, {Type: tea.KeyPgDown},
	}
	for _, k := range keys {
		nm, _ := m.Update(k)
		m = nm
		total := len(m.items)
		if m.cur < 0 || m.cur >= total {
			t.Fatalf("cursor out of range: cur=%d total=%d", m.cur, total)
		}
		if m.offset < 0 || m.offset > maxOffset(total, m.height) {
			t.Fatalf("offset out of range: off=%d total=%d h=%d", m.offset, total, m.height)
		}
		if total > m.height && (m.cur < m.offset || m.cur >= m.offset+m.height) {
			t.Fatalf("cursor %d not visible in window [%d,%d)", m.cur, m.offset, m.offset+m.height)
		}
	}
}

func TestPageDownJumpsPage(t *testing.T) {
	m := newList(cfg.Default(), taskPackPC, 30)
	m.height = 20
	m.items = mkItems(58)
	m.loading = false
	m.ready = true
	// from top: pgdown moves cur to height (20) => offset clamps to 1
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = nm
	if m.cur != 20 {
		t.Errorf("pgdown cur = %d, want 20", m.cur)
	}
	if m.offset != 1 {
		t.Errorf("pgdown offset = %d, want 1", m.offset)
	}
	// repeatedly page down clamps at last
	for i := 0; i < 10; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	}
	if m.cur != 57 {
		t.Errorf("repeated pgdown cur = %d, want 57", m.cur)
	}
}

func TestPageUpFromBottom(t *testing.T) {
	m := newList(cfg.Default(), taskPackPC, 30)
	m.height = 20
	m.items = mkItems(58)
	m.loading = false
	m.ready = true
	m.cur = 57
	m.offset = clampOffset(0, 57, 58, m.height)
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = nm
	if m.cur != 37 {
		t.Errorf("pgup from bottom cur = %d, want 37", m.cur)
	}
}

func TestNumberToggleOverWindow(t *testing.T) {
	m := newList(cfg.Default(), taskPackPC, 30)
	m.height = 20
	m.items = mkItems(58)
	m.loading = false
	m.ready = true
	m.offset = 20
	m.cur = 20
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = nm
	// window shows items 41..; key 5 => item index 44
	if m.cur != 24 {
		t.Errorf("number 5 over window cur = %d, want 24", m.cur)
	}
	if !m.items[24].checked {
		t.Error("item 24 should be checked")
	}
	if m.offset != 20 {
		t.Errorf("window should stay at offset 20, got %d", m.offset)
	}
}

func TestViewRendersOnlyWindow(t *testing.T) {
	m := newList(cfg.Default(), taskPackPC, 30)
	m.items = mkItems(58)
	m.loading = false
	m.ready = true
	m.height = 20
	m.offset = 0
	m.cur = 0
	v := m.View()
	lines := 0
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(l, "[ ]") || strings.Contains(l, "[x]") {
			lines++
		}
	}
	if lines != 20 {
		t.Errorf("rendered %d item lines, want 20", lines)
	}
	if !strings.Contains(v, "items 1–20 of 58") {
		t.Errorf("missing indicator: %q", "items 1–20 of 58")
	}
	if !strings.Contains(v, "↓ 38 below") {
		t.Errorf("missing below-hint")
	}
	if strings.Contains(v, "Game A 37") { // row 38 (0-idx 37) not in window 0..20
		t.Errorf("window leaked beyond 20 rows")
	}
}

func TestWindowSizeMsgSetsHeight(t *testing.T) {
	m := newList(cfg.Default(), taskPackPC, 30)
	m.items = mkItems(58)
	m.loading = false
	m.ready = true
	nm, _ := m.Update(tea.WindowSizeMsg{Height: 30, Width: 80})
	m = nm
	if m.height != 23 {
		t.Errorf("height after resize = %d, want 23", m.height)
	}
	nm2, _ := m.Update(tea.WindowSizeMsg{Height: 10, Width: 80})
	m = nm2
	if m.height != 5 {
		t.Errorf("height after small resize = %d, want 5", m.height)
	}
	// offset re-clamped
	if m.offset > maxOffset(58, 5) {
		t.Errorf("offset %d exceeds max %d", m.offset, maxOffset(58, 5))
	}
}

func TestNumberHiddenBeyondWindow(t *testing.T) {
	m := newList(cfg.Default(), taskPackPC, 30)
	m.items = mkItems(58)
	m.loading = false
	m.ready = true
	m.height = 5
	m.offset = 20
	m.cur = 20
	// keys 6..9 beyond a 5-row window must do nothing
	for _, r := range []rune{'6', '7', '8', '9'} {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = nm
		if m.cur != 20 {
			t.Errorf("key %c moved cursor to %d, want 20", r, m.cur)
		}
	}
	// key 5 reaches item offset+4 = 24
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = nm
	if m.cur != 24 {
		t.Errorf("key 5 cur = %d, want 24", m.cur)
	}
}

func TestScrollbarThumb(t *testing.T) {
	// whole list fits -> no scrollbar usage, but helper still returns sane range
	s, e := scrollbarThumb(0, 20, 58)
	if e <= s {
		t.Errorf("thumb range empty: [%d,%d)", s, e)
	}
}
