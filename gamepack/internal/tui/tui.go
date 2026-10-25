package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gamepack/internal/cfg"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).MarginBottom(1)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 2)
)

type screen int

const (
	scrMenu screen = iota
	scrList
	scrRun
)

type taskKind int

const (
	taskPackPC taskKind = iota
	taskUnpackPC
	taskPackNS
	taskUnpackNS
)

type model struct {
	cfg     *cfg.Config
	cur     screen
	width   int
	height  int
	menu    menuModel
	list    listModel
	run     runModel
	lastErr string
}

// Run starts the TUI.
func Run(c *cfg.Config) error {
	_, err := tea.NewProgram(initialModel(c)).Run()
	return err
}

func initialModel(c *cfg.Config) *model {
	return &model{
		cfg:  c,
		cur:  scrMenu,
		menu: newMenu(),
	}
}

func (m *model) Init() tea.Cmd { return m.menu.Init() }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.cur == scrList {
			nm, cmd := m.list.Update(msg)
			m.list = nm
			return m, cmd
		}
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if msg.Type == tea.KeyEsc && m.cur == scrList {
			m.cur = scrMenu
			m.menu = newMenu()
			return m, nil
		}
	case menuSelectMsg:
		m.cur = scrList
		m.list = newList(m.cfg, msg.kind, m.height)
		return m, m.list.Init()
	case runCompleteMsg:
		m.cur = scrMenu
		m.menu = newMenu()
		m.lastErr = ""
		return m, nil
	case runErrMsg:
		m.cur = scrMenu
		m.menu = newMenu()
		m.lastErr = msg.err.Error()
		return m, nil
	}

	switch m.cur {
	case scrMenu:
		return m.menuUpdate(msg)
	case scrList:
		return m.listUpdate(msg)
	case scrRun:
		return m.runUpdate(msg)
	}
	return m, nil
}

func (m *model) menuUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	nm, cmd := m.menu.Update(msg)
	m.menu = nm
	return m, cmd
}

func (m *model) listUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	nm, cmd := m.list.Update(msg)
	m.list = nm
	return m, cmd
}

func (m *model) View() string {
	switch m.cur {
	case scrMenu:
		return m.menu.View()
	case scrList:
		return m.list.View()
	case scrRun:
		return m.run.View()
	}
	return ""
}
