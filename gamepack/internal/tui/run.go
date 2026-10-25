package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"gamepack/internal/catalog"
	"gamepack/internal/cfg"
	"gamepack/internal/nswitch"
	"gamepack/internal/pack"
	"gamepack/internal/unpack"
)

type runModel struct {
	cfg      *cfg.Config
	ch       chan tea.Msg
	resp     chan bool // answers from the remove-source prompt
	pending  *removePrompt
	kind     taskKind
	items    []listItem
	statuses map[string]string
	done     int
	total    int
}

type removePrompt struct {
	slug    string
	name    string
	gameDir string
}

type runStatusMsg struct {
	slug  string
	line  string
	final bool
}

type removePromptMsg struct {
	slug    string
	name    string
	gameDir string
}

type runCompleteMsg struct{}
type runErrMsg struct{ err error }

func (m *model) runUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.run.pending != nil {
			switch msg.String() {
			case "y", "Y", "enter":
				m.run.resp <- true
				m.run.pending = nil
				return m, func() tea.Msg { return <-m.run.ch }
			case "n", "N", "esc", "q":
				m.run.resp <- false
				m.run.pending = nil
				return m, func() tea.Msg { return <-m.run.ch }
			}
			return m, nil
		}
	case removePromptMsg:
		m.run.pending = &removePrompt{slug: msg.slug, name: msg.name, gameDir: msg.gameDir}
		return m, nil
	case startRunMsg:
		m.run = runModel{
			cfg:      m.cfg,
			ch:       make(chan tea.Msg, 32),
			resp:     make(chan bool, 1),
			kind:     msg.kind,
			items:    msg.items,
			statuses: map[string]string{},
			total:    len(msg.items),
		}
		go m.run.execute(context.Background())
		return m, func() tea.Msg { return <-m.run.ch }
	case runStatusMsg:
		if msg.final {
			m.run.done++
		}
		m.run.statuses[msg.slug] = msg.line
		if m.run.done >= m.run.total {
			return m, tea.Batch(func() tea.Msg { return runCompleteMsg{} })
		}
		return m, func() tea.Msg { return <-m.run.ch }
	case runErrMsg:
		return m, func() tea.Msg { return msg }
	}
	return m, nil
}

func (r *runModel) execute(ctx context.Context) {
	for _, it := range r.items {
		var err error
		var removeDir string
		var packed bool
		switch r.kind {
		case taskPackPC:
			removeDir, packed, err = r.packPC(ctx, it)
		case taskUnpackPC:
			err = r.unpackPC(ctx, it)
		case taskPackNS:
			removeDir, packed, err = r.packNS(ctx, it)
		case taskUnpackNS:
			err = r.unpackNS(ctx, it)
		}
		if err != nil {
			r.ch <- runStatusMsg{slug: it.slug, line: "FAILED: " + err.Error(), final: true}
			r.ch <- runErrMsg{err: err}
			return
		}
		if packed && removeDir != "" {
			r.ch <- removePromptMsg{slug: it.slug, name: it.label, gameDir: removeDir}
			if <-r.resp {
				if e := os.RemoveAll(removeDir); e != nil {
					r.ch <- runStatusMsg{slug: it.slug, line: "remove failed: " + e.Error()}
				} else {
					r.ch <- runStatusMsg{slug: it.slug, line: "game folder removed"}
				}
			} else {
				r.ch <- runStatusMsg{slug: it.slug, line: "kept game folder"}
			}
		}
		r.ch <- runStatusMsg{slug: it.slug, line: "done", final: true}
	}
	r.ch <- runCompleteMsg{}
}

func (r *runModel) status(slug, line string) {
	select {
	case r.ch <- runStatusMsg{slug: slug, line: line}:
	default:
	}
}

func (r *runModel) packPC(ctx context.Context, it listItem) (string, bool, error) {
	r.status(it.slug, "packing…")
	ix, err := catalog.LoadIndex(r.cfg)
	if err != nil {
		return "", false, err
	}
	var g *catalog.Game
	for i := range ix.PC {
		if catalog.Slugify(ix.PC[i].Name, ix.PC[i].AppID) == it.slug {
			g = &ix.PC[i]
			break
		}
	}
	if g == nil {
		return "", false, fmt.Errorf("game %s not found", it.slug)
	}
	res, err := pack.Pack(ctx, r.cfg, *g, func(phase string, frac float64) {
		r.status(it.slug, fmt.Sprintf("[%3.0f%%] %s", frac*100, phase))
	})
	if err != nil {
		return "", false, err
	}
	if res.AlreadyDone {
		return "", false, fmt.Errorf("already archived (%s); re-run with --force to re-pack", res.Slug)
	}
	ix.MarkArchived(r.cfg, res.Slug, g.GameDir, res.Manifest.ArchivedSize)
	if err := ix.Save(r.cfg); err != nil {
		return "", false, err
	}
	return g.GameDir, true, nil
}

func (r *runModel) unpackPC(ctx context.Context, it listItem) error {
	r.status(it.slug, "unpacking…")
	ix, err := catalog.LoadIndex(r.cfg)
	if err != nil {
		return err
	}
	for i := range ix.PC {
		g := ix.PC[i]
		if catalog.Slugify(g.Name, g.AppID) != it.slug || !g.IsArchived {
			continue
		}
		m, err := pack.ReadManifest(r.cfg.PCArchiveDir(it.slug) + "/manifest.json")
		if err != nil {
			return err
		}
		if err := unpack.Preflight(r.cfg, m); err != nil {
			return err
		}
		return unpack.Unpack(ctx, r.cfg, m, it.slug, unpack.Options{
			Interactive: true,
			OnStatus: func(s string) {
				r.status(it.slug, s)
			},
		})
	}
	return fmt.Errorf("no archive for %s", it.slug)
}

func (r *runModel) packNS(ctx context.Context, it listItem) (string, bool, error) {
	r.status(it.slug, "packing ROM…")
	roms, err := nswitch.ScanROMS(r.cfg)
	if err != nil {
		return "", false, err
	}
	for _, rom := range roms {
		if rom.File != it.slug {
			continue
		}
		_, _, _, err := nswitch.Pack(ctx, r.cfg, rom, func(phase string, frac float64) {
			r.status(it.slug, fmt.Sprintf("[%3.0f%%] %s", frac*100, phase))
		})
		if err != nil {
			return "", false, err
		}
		ix, _ := catalog.LoadIndex(r.cfg)
		found := false
		for i := range ix.NSwitch {
			if ix.NSwitch[i].File == it.slug {
				ix.NSwitch[i].IsArchived = true
				ix.NSwitch[i].ArchivedAt = time.Now().UTC().Format("2006-01-02T15:04:05Z")
				ix.NSwitch[i].ArchivedSize = rom.SizeBytes
				found = true
			}
		}
		if !found {
			ix.NSwitch = append(ix.NSwitch, catalog.NSROM{
				Name:         rom.Name,
				File:         rom.File,
				SizeBytes:    rom.SizeBytes,
				TitleID:      rom.TitleID,
				IsArchived:   true,
				ArchivedAt:   time.Now().UTC().Format("2006-01-02T15:04:05Z"),
				ArchivedSize: rom.SizeBytes,
			})
		}
		if err := ix.Save(r.cfg); err != nil {
			return "", false, err
		}
		return r.cfg.NSwitchDir + "/" + rom.File, true, nil
	}
	return "", false, fmt.Errorf("ROM %s not found", it.slug)
}

func (r *runModel) unpackNS(ctx context.Context, it listItem) error {
	r.status(it.slug, "unpacking ROM…")
	ix, err := catalog.LoadIndex(r.cfg)
	if err != nil {
		return err
	}
	for _, rom := range ix.NSwitch {
		if rom.File != it.slug || !rom.IsArchived {
			continue
		}
		return nswitch.Unpack(ctx, r.cfg, nswitch.ROM{Name: rom.Name, File: rom.File, TitleID: rom.TitleID}, r.cfg.NSwitchDir, catalog.Slugify(rom.Name, 0))
	}
	return fmt.Errorf("no archive for %s", it.slug)
}

func (m *runModel) View() string {
	if m.pending != nil {
		return boxStyle.Render(titleStyle.Render("Archived: "+m.pending.name+"\n") +
			fmt.Sprintf("  Game folder: %s\n\n", m.pending.gameDir) +
			selStyle.Render("  Remove it to free space?  [y]es / [N]o"))
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Running (%d/%d)", m.done, m.total)) + "\n")
	for _, it := range m.items {
		line := "  " + it.label
		if s, ok := m.statuses[it.slug]; ok {
			line += "  " + dimStyle.Render(s)
		} else {
			line += "  …"
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("Ctrl-C aborts current run (state is journaled; safe to rerun)"))
	return boxStyle.Render(b.String())
}
