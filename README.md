# LegionS — Lenovo Legion Go S (Z2 Go) Machine Reference

Ultimate merged repository for the **Lenovo Legion Go S (Z2 Go, white Windows edition)**
handheld, consolidated from the retired ROG Ally (`rog`) reference, `rogMigration`,
the proxmox host audit (rog), and the `Add-Non-Steam-Games-Linux` tool.

This repo replaces the ROG-specific repos (`rog`, `rogMigration`, `Add-Non-Steam-Games`,
`Add-Non-Steam-Games-Linux`), which are archived/retired. The `proxmox` repo remains
the source of truth for infrastructure/fleet data; a snapshot of its `servers/rog`
audit is vendored under `proxmox/` for convenience.

## Hardware

- Device: Lenovo Legion Go S (Z2 Go), white Windows edition
- OS: Bazzite `bazzite-deck-gnome` (F45) on the internal NVMe
- Drive: 4 TB NVMe (TEAM TM8FPW004T) — migrated from a water-damaged ROG Ally;
  both partitions retained (OS btrfs + `Games_Data` ext4 ~3.5 TB)
- Previous device: ROG Ally (retired) — see `migration/` and `proxmox/servers/rog`

## Layout

Full file/directory catalog: [INDEX.md](INDEX.md) (human) · [MANIFEST.json](MANIFEST.json) (machine).

| Path | Purpose |
|------|---------|
| `AGENTS.md` | Agent working notes |
| `ARCHITECTURE.md` | Machines, Tailscale IPs, Steam data paths, key filesystem locations |
| `state/CURRENT.md` | Most recent verified machine status snapshot |
| `gamepack/` | Go module: TUI/CLI archiving/restoring games on `Games_Data` |
| `tools/Add-Non-Steam-Games-Linux/` | Python tool for adding non-Steam games to Steam |
| `games/` | `SHORTCUTS.md`, `UNLAUNCHED.md` |
| `saves/SAVES.md` | Save-data layout and sync tooling |
| `steam/CONFIG.md` | Steam account/config snapshot |
| `migration/` | Drive expansion, migration status, Legion Go S migration runbook |
| `proxmox/` | Vendored rog host audit snapshot (proxmox repo is source of truth) |
| `docs/` | `display-blackscreen.md`, `gamepack-plan.md` |
| `TODO.md` / `MODERNIZATION-TODO.md` | Task lists |

## Migration (ROG Ally -> Legion Go S)

See `migration/LEGION-GO-S.md` for the full runbook. Summary:
- Drive health verified (SMART PASSED).
- Legacy ROG hhd state reset; hostname set to `legion-go-s`.
- Inputplumber ships `50-legion_go_s.yaml` for controller support.
- Kernel 7.2.11-ogc supports AMD Z2 Go; gamescope auto-detects 8" 1920x1200 panel.
- `Games_Data` auto-mounts at `/run/media/system/Games_Data` (~/Games symlink valid).

## gamepack commands (run from `gamepack/`)

```bash
cd gamepack
make build      # static cross-compile
make test      # go test ./internal/...
make coverage  # coverage gate
make lint
make deploy    # build + scp to device
```

## Caveats

- IPs/hostnames in ARCHITECTURE.md are dynamic — verify against live `tailscale status`.
- `state/CURRENT.md` is a historical snapshot; recheck live state.
- Local hostname on device may be stale; some scripts expect a specific name.
