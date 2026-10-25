package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"gamepack/internal/catalog"
	"gamepack/internal/cfg"
	"gamepack/internal/nswitch"
	"gamepack/internal/pack"
)

type listItem struct {
	label    string
	slug     string
	size     int64
	archived bool
	note     string
	checked  bool
}

type listModel struct {
	cfg     *cfg.Config
	kind    taskKind
	items   []listItem
	cur     int
	height  int
	offset  int
	loading bool
	ready   bool
	msg     string
}

type loadedListMsg struct {
	items []listItem
	err   error
}

type startRunMsg struct {
	kind  taskKind
	items []listItem
}

func newList(c *cfg.Config, kind taskKind, termH int) listModel {
	return listModel{cfg: c, kind: kind, loading: true, height: listHeight(termH)}
}

func (m listModel) Init() tea.Cmd {
	return func() tea.Msg {
		items, err := loadItems(m.cfg, m.kind)
		return loadedListMsg{items: items, err: err}
	}
}

// --- windowing math (pure) ---

func maxOffset(total, height int) int {
	if total <= height {
		return 0
	}
	return total - height
}

// clampOffset keeps the cursor visible inside the window.
func clampOffset(off, cur, total, height int) int {
	max := maxOffset(total, height)
	if total <= height || height <= 0 {
		return 0
	}
	if cur < off {
		off = cur
	}
	if cur >= off+height {
		off = cur - height + 1
	}
	if off > max {
		off = max
	}
	if off < 0 {
		off = 0
	}
	return off
}

// listHeight maps a terminal height to the number of list rows to render.
func listHeight(termH int) int {
	if termH <= 0 {
		return 20
	}
	const reserved = 7 // title + indicator + blank + help + box borders + spare
	h := termH - reserved
	if h < 5 {
		return 5
	}
	if h > 26 {
		return 26
	}
	return h
}

func loadItems(c *cfg.Config, kind taskKind) ([]listItem, error) {
	switch kind {
	case taskPackPC:
		ix, err := catalog.LoadIndex(c)
		if err != nil {
			return nil, err
		}
		games := ix.PC
		if len(games) == 0 {
			games, err = catalog.BuildCatalog(c)
			if err != nil {
				return nil, err
			}
			ix.PC = games
			ix.ToolVersion = pack.ToolVersion
			_ = ix.Save(c)
		}
		var out []listItem
		for _, g := range games {
			if g.Exclude != "" {
				continue
			}
			item := listItem{
				label:    g.Name,
				slug:     catalog.Slugify(g.Name, g.AppID),
				size:     g.SizeBytes,
				archived: g.IsArchived,
			}
			hasPrefix := c.HasPrefix(g.AppID)
			if g.DirMissing {
				item.note = "dir missing"
			} else if !hasPrefix {
				item.note = "no prefix (launch once in Steam)"
			}
			out = append(out, item)
		}
		sortItems(out)
		return out, nil
	case taskUnpackPC:
		ix, err := catalog.LoadIndex(c)
		if err != nil {
			return nil, err
		}
		var out []listItem
		for _, g := range ix.PC {
			if !g.IsArchived {
				continue
			}
			out = append(out, listItem{label: g.Name, slug: catalog.Slugify(g.Name, g.AppID), size: g.ArchivedSize, archived: true})
		}
		sortItems(out)
		return out, nil
	case taskPackNS:
		roms, err := nswitch.ScanROMS(c)
		if err != nil {
			return nil, err
		}
		var out []listItem
		for _, r := range roms {
			note := ""
			if r.TitleID != "" {
				note = "titleid " + r.TitleID
			}
			out = append(out, listItem{label: r.Name, slug: r.File, size: r.SizeBytes, note: note})
		}
		sortItems(out)
		return out, nil
	case taskUnpackNS:
		ix, err := catalog.LoadIndex(c)
		if err != nil {
			return nil, err
		}
		var out []listItem
		for _, r := range ix.NSwitch {
			if !r.IsArchived {
				continue
			}
			out = append(out, listItem{label: r.Name, slug: r.File, size: r.ArchivedSize, archived: true})
		}
		sortItems(out)
		return out, nil
	}
	return nil, fmt.Errorf("unknown task kind")
}

func sortItems(items []listItem) {
	sort.Slice(items, func(i, j int) bool { return items[i].label < items[j].label })
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = listHeight(msg.Height)
		m.offset = clampOffset(m.offset, m.cur, len(m.items), m.height)
		return m, nil
	case loadedListMsg:
		m.loading = false
		if msg.err != nil {
			m.msg = "load error: " + msg.err.Error()
			return m, nil
		}
		m.items = msg.items
		m.ready = true
		m.msg = "number/space to toggle, enter to run"
		return m, nil
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		total := len(m.items)
		switch msg.String() {
		case "up", "k":
			if m.cur > 0 {
				m.cur--
			}
		case "down", "j":
			if m.cur < total-1 {
				m.cur++
			}
		case "pgup", "ctrl+u":
			m.cur -= m.height
			if m.cur < 0 {
				m.cur = 0
			}
		case "pgdown", "ctrl+d":
			m.cur += m.height
			if m.cur > total-1 {
				m.cur = total - 1
			}
		case "home":
			m.cur = 0
		case "end":
			m.cur = total - 1
		case " ":
			m.toggle(m.cur)
		case "a":
			all := true
			for _, it := range m.items {
				if !it.checked {
					all = false
					break
				}
			}
			for i := range m.items {
				m.items[i].checked = !all
			}
		case "enter":
			var sel []listItem
			for _, it := range m.items {
				if it.checked {
					sel = append(sel, it)
				}
			}
			if len(sel) == 0 {
				m.msg = "nothing selected — press a number or space to check"
				return m, nil
			}
			return m, func() tea.Msg { return startRunMsg{kind: m.kind, items: sel} }
		case "esc":
			// handled by parent model (back to menu)
		default:
			// number keys toggle the item at that position within the window
			if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= 9 && n <= m.height {
				idx := m.offset + n - 1
				if idx < total {
					m.cur = idx
					m.toggle(idx)
				}
			}
		}
		m.offset = clampOffset(m.offset, m.cur, total, m.height)
	}
	return m, nil
}

func (m *listModel) toggle(i int) {
	if i < 0 || i >= len(m.items) {
		return
	}
	m.items[i].checked = !m.items[i].checked
}

func (m listModel) View() string {
	if m.loading {
		return boxStyle.Render(titleStyle.Render("loading catalog…"))
	}
	if m.msg != "" && !m.ready {
		return boxStyle.Render(errStyle.Render(m.msg))
	}
	total := len(m.items)
	if total == 0 {
		return boxStyle.Render(titleStyle.Render("nothing to show") + "\n" + helpStyle.Render("esc back"))
	}
	m.offset = clampOffset(m.offset, m.cur, total, m.height)
	end := m.offset + m.height
	if end > total {
		end = total
	}
	window := m.items[m.offset:end]

	var b strings.Builder
	b.WriteString(titleStyle.Render("Select games — " + m.msg + "\n"))
	if total > m.height {
		b.WriteString(dimStyle.Render(fmt.Sprintf("items %d–%d of %d · cursor row %d\n", m.offset+1, end, total, m.cur+1)))
	}

	thumbStart, thumbEnd := scrollbarThumb(m.offset, m.height, total)

	for i, it := range window {
		absIdx := m.offset + i
		mark := "[ ]"
		if it.checked {
			mark = "[x]"
		}
		rowNum := pad2(i + 1)
		line := fmt.Sprintf("%s %s %s", mark, rowNum, truncate(it.label, 40))
		if it.size > 0 {
			line += fmt.Sprintf(" %7.1fG", float64(it.size)/1e9)
		}
		if it.archived {
			line += "  archived"
		}
		if it.note != "" {
			line += "  ⚠ " + it.note
		}
		bar := " "
		if total > m.height {
			if absIdx >= thumbStart && absIdx < thumbEnd {
				bar = "█"
			} else {
				bar = "│"
			}
		}
		if absIdx == m.cur {
			line = selStyle.Render("> ") + line + " " + bar
		} else {
			line = "  " + line + " " + bar
		}
		b.WriteString(truncate(line, 68) + "\n")
	}

	if total > m.height {
		var hints []string
		if m.offset > 0 {
			hints = append(hints, fmt.Sprintf("↑ %d above", m.offset))
		}
		if end < total {
			hints = append(hints, fmt.Sprintf("↓ %d below", total-end))
		}
		b.WriteString(dimStyle.Render(strings.Join(hints, "  ")) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("number/space toggle · a=all · enter run · pgup/pgdn page · esc back · q quit"))
	return boxStyle.Render(b.String())
}

// scrollbarThumb maps [0,total) onto the height rows [0,height).
func scrollbarThumb(offset, height, total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	start := offset * height / total
	end := (offset + height) * height / total
	if end <= start {
		end = start + 1
	}
	if end > height {
		end = height
	}
	return start, end
}

func pad2(n int) string {
	if n < 10 {
		return fmt.Sprintf(" %d", n)
	}
	return fmt.Sprintf("%d", n)
}

// truncate shortens s to max runes, adding an ellipsis if cut.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return string(r[:max-1]) + "…"
}
