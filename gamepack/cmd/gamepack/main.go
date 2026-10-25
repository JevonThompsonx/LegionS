package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gamepack/internal/catalog"
	"gamepack/internal/cfg"
	"gamepack/internal/nswitch"
	"gamepack/internal/pack"
	"gamepack/internal/tui"
	"gamepack/internal/unpack"
)

func main() {
	var (
		refresh   = flag.Bool("refresh", false, "rebuild catalog index")
		dry       = flag.Bool("dry-run", false, "print catalog, do nothing")
		packOne   = flag.String("pack", "", "pack a game by name (CLI mode)")
		unpackOne = flag.String("unpack", "", "unpack an archived game by name (CLI mode)")
		packNS    = flag.String("pack-ns", "", "pack an NSwitch ROM by name (CLI mode)")
		unpackNS  = flag.String("unpack-ns", "", "unpack an archived NSwitch ROM by name (CLI mode)")
		force     = flag.Bool("force", false, "overwrite existing archives / unpack over existing dir")
		noPrefix  = flag.Bool("no-prefix", false, "do not archive the proton prefix")
		removeSrc = flag.Bool("remove-source", false, "delete the game folder after a successful pack")
		list      = flag.Bool("list", false, "list catalog (CLI mode)")
	)
	flag.Parse()

	c := cfg.Default()
	if *noPrefix {
		c.WithPrefix = false
	}
	c.Force = *force
	c.RemoveSource = *removeSrc

	if *refresh || *dry || *list || *packOne != "" || *unpackOne != "" || *packNS != "" || *unpackNS != "" {
		runCLI(c, *refresh, *dry, *list, *packOne, *unpackOne, *packNS, *unpackNS)
		return
	}

	if err := tui.Run(c); err != nil {
		fmt.Fprintln(os.Stderr, "gamepack:", err)
		os.Exit(1)
	}
}

func runCLI(c *cfg.Config, refresh, dry, list bool, packName, unpackName, packNSName, unpackNSName string) {
	ctx := context.Background()

	buildIndex := func() error {
		games, err := catalog.BuildCatalog(c)
		if err != nil {
			return err
		}
		ix, _ := catalog.LoadIndex(c)
		ix.PC = games
		ix.ToolVersion = pack.ToolVersion
		return ix.Save(c)
	}

	if refresh || dry || list {
		if err := buildIndex(); err != nil {
			fmt.Fprintln(os.Stderr, "catalog:", err)
			os.Exit(1)
		}
	}
	ix, err := catalog.LoadIndex(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "index:", err)
		os.Exit(1)
	}

	if list {
		for _, g := range sortedGames(ix.PC) {
			badge := ""
			if g.IsArchived {
				badge = " [ARCHIVED]"
			}
			if g.Exclude != "" {
				badge = " [excluded: " + g.Exclude + "]"
			} else if g.DirMissing {
				badge = " [dir missing]"
			}
			fmt.Printf("%-30s appid=%-11d %8.1fG%s\n", g.Name, g.AppID, float64(g.SizeBytes)/1e9, badge)
		}
		return
	}

	if dry {
		return
	}

	if packName != "" {
		g := findGame(ix.PC, packName)
		if g == nil || g.Exclude != "" {
			fmt.Fprintln(os.Stderr, "no packable game:", packName)
			os.Exit(1)
		}
		res, err := pack.Pack(ctx, c, *g, func(phase string, frac float64) {
			fmt.Printf("\r[%3.0f%%] %-28s", frac*100, phase)
		})
		fmt.Println()
		if err != nil {
			fmt.Fprintln(os.Stderr, "pack:", err)
			os.Exit(1)
		}
		if res.AlreadyDone {
			fmt.Println("already archived:", res.Slug)
			return
		}
		ix.MarkArchived(c, res.Slug, g.GameDir, res.Manifest.ArchivedSize)
		if err := ix.Save(c); err != nil {
			fmt.Fprintln(os.Stderr, "index:", err)
		}
		fmt.Printf("packed %s -> %s (%.1fG -> %.1fG, %.0f%%)\n",
			g.Name, res.Slug, float64(res.Manifest.OrigSize)/1e9, float64(res.Manifest.ArchivedSize)/1e9,
			res.Manifest.Ratio*100)
		if c.RemoveSource {
			fmt.Printf("removing game folder: %s\n", g.GameDir)
			if err := os.RemoveAll(g.GameDir); err != nil {
				fmt.Fprintln(os.Stderr, "remove-source:", err)
			} else {
				fmt.Println("game folder removed — unpack any time to restore")
			}
		}
		return
	}

	if unpackName != "" {
		m, slug := findArchived(ix.PC, unpackName)
		if m == nil {
			fmt.Fprintln(os.Stderr, "no archive for:", unpackName)
			os.Exit(1)
		}
		if err := unpack.Unpack(ctx, c, m, slug, unpack.Options{Interactive: false, OnStatus: func(s string) {
			fmt.Println(" *", s)
		}}); err != nil {
			fmt.Fprintln(os.Stderr, "unpack:", err)
			os.Exit(1)
		}
		fmt.Println("unpacked", m.Name)
		return
	}

	if packNSName != "" {
		roms, err := nswitch.ScanROMS(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, "scan:", err)
			os.Exit(1)
		}
		target := strings.TrimSuffix(packNSName, filepath.Ext(packNSName))
		for _, rom := range roms {
			if rom.Name == target || containsFold(rom.Name, target) {
				gamePath, savesPath, n, err := nswitch.Pack(ctx, c, rom, func(phase string, frac float64) {
					fmt.Printf("\r[%3.0f%%] %-24s", frac*100, phase)
				})
				fmt.Println()
				if err != nil {
					fmt.Fprintln(os.Stderr, "pack-ns:", err)
					os.Exit(1)
				}
				ix, _ := catalog.LoadIndex(c)
				found := false
				for i := range ix.NSwitch {
					if ix.NSwitch[i].File == rom.File {
						ix.NSwitch[i].IsArchived = true
						ix.NSwitch[i].ArchivedSize = rom.SizeBytes
						ix.NSwitch[i].ArchivedAt = time.Now().UTC().Format("2006-01-02T15:04:05Z")
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
				_ = ix.Save(c)
				fmt.Printf("packed ROM %s (%s), saves: %d, sha bundle at %s\n", rom.Name, gamePath, n, savesPath)
				return
			}
		}
		fmt.Fprintln(os.Stderr, "no ROM found:", packNSName)
		os.Exit(1)
	}

	if unpackNSName != "" {
		roms, err := nswitch.ScanROMS(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, "scan:", err)
			os.Exit(1)
		}
		// match against archive dirs (ROM may still be present or not)
		ix, _ := catalog.LoadIndex(c)
		target := strings.TrimSuffix(unpackNSName, filepath.Ext(unpackNSName))
		var match *catalog.NSROM
		for i := range ix.NSwitch {
			r := ix.NSwitch[i]
			if r.IsArchived && (r.Name == target || containsFold(r.Name, target)) {
				match = &ix.NSwitch[i]
				break
			}
		}
		if match == nil {
			fmt.Fprintln(os.Stderr, "no archived ROM:", unpackNSName)
			os.Exit(1)
		}
		_ = roms
		if err := nswitch.Unpack(ctx, c, nswitch.ROM{Name: match.Name, File: match.File, TitleID: match.TitleID}, c.NSwitchDir, catalog.Slugify(match.Name, 0)); err != nil {
			fmt.Fprintln(os.Stderr, "unpack-ns:", err)
			os.Exit(1)
		}
		fmt.Println("unpacked", match.Name)
		return
	}
}

func sortedGames(games []catalog.Game) []catalog.Game {
	out := append([]catalog.Game(nil), games...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func findGame(games []catalog.Game, name string) *catalog.Game {
	for i := range games {
		if games[i].Name == name || containsFold(games[i].Name, name) {
			return &games[i]
		}
	}
	return nil
}

func findArchived(games []catalog.Game, name string) (*pack.Manifest, string) {
	for _, g := range games {
		if !g.IsArchived {
			continue
		}
		if g.Name == name || containsFold(g.Name, name) {
			slug := catalog.Slugify(g.Name, g.AppID)
			m, err := pack.ReadManifest(cfg.Default().PCArchiveDir(slug) + "/manifest.json")
			if err != nil {
				continue
			}
			return m, slug
		}
	}
	return nil, ""
}

func containsFold(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	s = lower(s)
	sub = lower(sub)
	return len(sub) == 0 || indexOf(s, sub) >= 0
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
