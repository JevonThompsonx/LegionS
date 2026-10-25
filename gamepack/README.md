# gamepack — ROG Game Archiver

Interactive tool for the ROG Ally (Bazzite/gamescope) to archive and restore
games on `Games_Data` with maximum zstd compression, while keeping save data
matched and wired back into Steam.

Runs **on the ROG** over SSH (binary cross-compiled on the dev machine). Uses
`bubbletea` TUI + CLI.

## Deploy

```bash
# on the dev machine
cd gamepack
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o gamepack ./cmd/gamepack
scp gamepack rog:/run/media/system/Games_Data/Tools/gamepack/gamepack
ssh rog 'ln -sf /run/media/system/Games_Data/Tools/gamepack/gamepack ~/.local/share/bin/gamepack'
```

No runtime deps beyond system `zstd`, `tar`, `steam`, `flatpak` (ludusavi), and
coreutils `flock`/`pgrep`. `~/.local/bin` is root-owned on Bazzite, so the shim
goes in `~/.local/share/bin`. `make build` / `make deploy` handle build+push.

## Usage

```bash
gamepack                    # TUI
gamepack --list             # catalog
gamepack --refresh          # rebuild .index.json
gamepack --pack "Cuphead"               # pack one game (CLI)
gamepack --pack "Cuphead" --remove-source  # ...and delete the game folder after
gamepack --unpack "Cuphead"
gamepack --pack-ns "Animal Crossing New Horizons"
gamepack --unpack-ns "Animal Crossing New Horizons"
gamepack --force            # re-pack over an existing archive / unpack over an existing dir
```

TUI keys: `1-5` main menu, `1-9`/space toggle rows, `pgup/pgdn`/`ctrl-u/d`
page, `home/end` jump, `enter` run, `esc` back, `q` quit.

## How it works

- **Catalog**: parses Steam's binary `shortcuts.vdf` (userdata/<id>/config/)
  for non-Steam games; resolves each exe to its game dir (detects
  "prefix-container" installs like Split Fiction by walking up for
  `drive_c/`+`system.reg`); excludes emulators/tools/NSwitch. Cache in
  `Games_Data/Archive/.index.json`.
- **Pack**: streams `tar | zstd --ultra -22 --long=27` of the game dir into
  `Archive/PC/<slug>/game.tar.zst`, snapshots save dirs into `saves.tar.zst`,
  archives the Proton prefix (minus shadercache/locks) into `prefix.tar.zst`,
  writes `manifest.json` (sha256, sizes, save mapping). After a successful pack
  the TUI asks whether to remove the game folder to reclaim space.
- **Compression**: PC games + saves use `-22` max; NSwitch ROMs (already
  compressed) use `zstd -3`. Expected ratios: repack-style games ~85%, loose
  files ~60%.
- **Saves**: source of truth is `~/GameSaves/` (syncthing-synced). Save
  discovery = embedded golden table (mirrors `link-saves.sh`) + live symlink
  scan + ludusavi fallback. Restore is skip-newer (never clobbers a fresher
  save) and re-creates the prefix symlinks (honoring the Cuphead LocalLow
  no-symlink caveat).
- **Unpack**: verifies sha256, extracts to staging then atomic rename, restores
  prefix (from `prefix.tar.zst`, or interactively via `steam://rungameid` +
  exit polling), restores saves + links. Game-running detection uses live
  non-zombie process matching the exe name (pfx.lock flock proved unreliable).
- **NSwitch**: separate `Archive/NSwitch/` tree; packs `rom.tar.zst` (zstd -3)
  and optionally matched Citron saves (curated title-id map + TUI prompt).

## Reboot resilience

A reboot kills a running pack (transient systemd unit). The partial archive
dir is detected (no `manifest.json` = incomplete) and re-packed cleanly on the
next run; `zstd` refuses to overwrite an existing stdin-output, so the stale
file is removed first. Avoid reboots mid-pack (~10h at -22 for an 89G game).

## Disk layout (post-move)

Proton prefixes were relocated to the big drive and symlinked back, so Steam
sees the same `compatdata/<appid>` paths:

```
/run/media/system/Games_Data/
├── CompatData/<appid>/      <- real Proton prefixes (symlinked from ~/.local/share/Steam/steamapps/compatdata)
├── Citron/                  <- emulator dir (symlinked from ~/.local/share/citron)
├── Archive/{PC,NSwitch}/    <- archives + .index.json + .journal.json
└── Tools/gamepack/          <- this binary
```

## Notes

- **Coverage gate**: `make coverage` (scripts/coverage-check.sh) enforces ≥40% on
  the pure-logic packages (catalog, cfg, compress, nswitch, pack, saves, tui).
  `proton`, `state`, and `unpack` are exempt — they shell out to Steam/Proton
  and need live-ROG integration testing. 80% is the long-term target (tdd-workflow);
  the gate is the floor that must never regress. Raise `MIN_COVERAGE` as it grows.
- `steam steam://rungameid/<appid>` from an SSH session did not visibly launch
  a non-Steam game in game mode during testing — the save-attach pause screen
  tells you to launch manually from the ROG, then detects game exit.
- `~/GameSaves` and `~/ludusavi-backup` stay on the OS drive (syncthing-synced).
- The `link-game-saves` auto-linker recreates save symlinks inside prefixes;
  it was stopped briefly during the compatdata move to avoid a race.
- Archived games keep their Steam shortcut; launching one that's archived
  fails until you `gamepack --unpack <name>` it.
- `smart-save-checker` service was failing (pre-existing, unrelated).
- **Known gap — journaling is dead code**: `internal/state/journal.go`
  (`Journal`/`JournalStep`) and `cfg.JournalPath()` have zero non-test callers,
  so the TUI help line "state is journaled; safe to rerun" overpromises —
  resume safety currently rests only on "no manifest = incomplete". Tracked in
  MODERNIZATION-TODO.md ("Wire up or remove the journal").
- **Deploy**: `make deploy` builds a static binary and copies it via
  `scp gamepack rog:/tmp/gamepack.new` + `ssh rog '... mv ... Games_Data/Tools/gamepack/gamepack'`
  (requires the `rog` SSH host).
