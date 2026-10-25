# LegionS — Overview (merged, human-readable)

This is the single narrative entry point for the merged LegionS repository. It consolidates
the retired ROG Ally repos (`rog`, `rogMigration`, `Add-Non-Steam-Games`, `Add-Non-Steam-Games-Linux`),
the proxmox host audit snapshot, and the Legion Go S migration into one readable map.

For the full file catalog see [INDEX.md](INDEX.md); for a machine-readable listing see
[MANIFEST.json](MANIFEST.json); for LLM crawlers see [llms.txt](llms.txt).

## What this repo is

A **machine reference + tooling monorepo** for a handheld gaming PC:

- **Device (current):** Lenovo Legion Go S (Z2 Go), white Windows edition, running Bazzite
  `bazzite-deck-gnome` (F45) on a 4 TB NVMe (TEAM TM8FPW004T).
- **Previous device:** ROG Ally (retired) — historical data retained under `migration/`,
  `proxmox/`, `state/`, `games/`, `saves/`, `steam/`.
- **Two code projects:**
  1. `gamepack/` — Go TUI/CLI that archives/restores games on `Games_Data` (bubbletea).
  2. `tools/Add-Non-Steam-Games-Linux/` — Python tool that adds non-Steam games to Steam
     and fetches artwork from SteamGridDB.

## Start here (reading order)

1. [README.md](README.md) — machine reference + layout table.
2. [ARCHITECTURE.md](ARCHITECTURE.md) — machines, Tailscale IPs, Steam paths.
3. [state/CURRENT.md](state/CURRENT.md) — most recent verified machine status.
4. [migration/LEGION-GO-S.md](migration/LEGION-GO-S.md) — migration runbook.
5. [gamepack/README.md](gamepack/README.md) — archiver usage.
6. [tools/Add-Non-Steam-Games-Linux/README.md](tools/Add-Non-Steam-Games-Linux/README.md) — non-Steam game adder.

## Repository map

| Area | Path | Readable index |
|------|------|----------------|
| Architecture & machines | `ARCHITECTURE.md`, `ROG_DRIVE.md` | [docs/README.md](docs/README.md) |
| Current state | `state/` | [state/README.md](state/README.md) |
| Migration history | `migration/` | [migration/README.md](migration/README.md) |
| Game shortcuts (data) | `games/` | [games/README.md](games/README.md) |
| Save data | `saves/` | [saves/README.md](saves/README.md) |
| Steam config (data) | `steam/` | [steam/README.md](steam/README.md) |
| Host audit snapshot | `proxmox/` | [proxmox/README.md](proxmox/README.md) |
| Game archiver (Go) | `gamepack/` | [gamepack/README.md](gamepack/README.md) |
| Non-Steam tool (Python) | `tools/` | [tools/README.md](tools/README.md) |
| Display troubleshooting | `docs/display-blackscreen.md` | [docs/README.md](docs/README.md) |
| Tasks | `TODO.md`, `MODERNIZATION-TODO.md` | [TASKS.md](TASKS.md) |

## Machine state (consolidated)

See [docs/machine-state.md](docs/machine-state.md) for a single-page consolidation of
shortcuts, unlaunched games, save layout, Steam config, and current status.

## Tasks (consolidated)

See [TASKS.md](TASKS.md) for a single index of both machine-ops and gamepack-audit backlogs.

## Conventions

- Dated files under `state/`, `games/`, `saves/`, `steam/`, `migration/`, `proxmox/` are
  **historical snapshots** — verify live state before acting. Do not rewrite them as current.
- `TODO.md` (machine-ops) and `MODERNIZATION-TODO.md` (gamepack audit) are disjoint; keep both.
- Known gaps are documented in [AGENTS.md](AGENTS.md) and must not be silently "fixed".

## Agent notes

See [AGENTS.md](AGENTS.md) for gates, known gaps, branches, and verification commands.