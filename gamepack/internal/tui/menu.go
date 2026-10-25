package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type menuModel struct {
	items []string
	cur   int
}

type menuSelectMsg struct{ kind taskKind }

func newMenu() menuModel {
	return menuModel{items: []string{
		"Pack games (PC)",
		"Unpack games (PC)",
		"Pack NSwitch ROMs",
		"Unpack NSwitch ROMs",
		"Quit",
	}}
}

func (m menuModel) Init() tea.Cmd { return nil }

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Number keys select directly: 1..N
		for i := 0; i < len(m.items); i++ {
			if msg.String() == fmt.Sprintf("%d", i+1) {
				m.cur = i
				return m, selectAt(m.items, i)
			}
		}
		switch msg.String() {
		case "up", "k":
			m.cur = (m.cur - 1 + len(m.items)) % len(m.items)
		case "down", "j":
			m.cur = (m.cur + 1) % len(m.items)
		case "enter", " ":
			return m, selectAt(m.items, m.cur)
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func selectAt(items []string, i int) tea.Cmd {
	if i >= len(items) || items[i] == "Quit" {
		return tea.Quit
	}
	return func() tea.Msg {
		return menuSelectMsg{kind: taskKind(i)}
	}
}

func (m menuModel) View() string {
	var b string
	b += titleStyle.Render("gamepack — ROG Game Archiver") + "\n"
	for i, item := range m.items {
		num := fmt.Sprintf("%d", i+1)
		prefix := "  "
		if i == m.cur {
			prefix = selStyle.Render("> ")
			item = selStyle.Render(item)
		}
		b += fmt.Sprintf("%s %s%s\n", num, prefix, item)
	}
	b += "\n" + helpStyle.Render("press 1-5, or ↑/↓ + Enter")
	return boxStyle.Render(b)
}
