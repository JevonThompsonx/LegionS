# Machine State — Consolidated (historical snapshots)

> These are dated snapshots; verify live state before acting. Do not rewrite as current.
> Source files: `state/CURRENT.md`, `games/SHORTCUTS.md`, `games/UNLAUNCHED.md`,
> `saves/SAVES.md`, `steam/CONFIG.md`.

## Current status
See [`../state/CURRENT.md`](../state/CURRENT.md) (2026-08-11 snapshot): game-mode idle
black screen fixed (suspend.target/sleep.target unmasked), 4TB drive expansion complete,
update situation (F43 stable, F44 rebase pending).

## Shortcuts
- [`../games/SHORTCUTS.md`](../games/SHORTCUTS.md): 53 non-Steam shortcuts in
  `shortcuts.vdf` (account zLeafNinja, 201689394), with exe paths and prefix status.
- [`../games/UNLAUNCHED.md`](../games/UNLAUNCHED.md): 33 games with no compatdata
  prefix / LastPlayTime.

## Save data
- [`../saves/SAVES.md`](../saves/SAVES.md): `~/GameSaves/` layout (2.2G), appdata
  roaming/local/locallow, steam-userdata, symlinks, ludusavi backup, link-game-saves.sh.

## Steam config
- [`../steam/CONFIG.md`](../steam/CONFIG.md): account (zLeafNinja, 201689394), install
  paths, compatdata prefixes, library folders, shortcuts.vdf location.

## Related
- Architecture: [`../ARCHITECTURE.md`](../ARCHITECTURE.md)
- Migration: [`../migration/README.md`](../migration/README.md)
- Host audit: [`../proxmox/README.md`](../proxmox/README.md)