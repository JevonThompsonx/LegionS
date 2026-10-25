# Game Save State

## GameSaves Directory
**Path**: `/var/home/jevonx/GameSaves/`
**Total size**: 2.2G
**Source**: Merged from ROG drive (bazzite-rog install), host OS GameSaves, and rog machine save-data
**Merge strategy**: rsync --update (keep newest for conflicting files)

### Structure
```
GameSaves/
├── appdata-roaming/      → mapped to AppData/Roaming/<Game> in Proton prefix
├── appdata-local/        → mapped to AppData/Local/<Game>
├── appdata-locallow/     → mapped to AppData/LocalLow/<Studio>/<Game>
├── saved-games/          → mapped to Saved Games/<Game>
├── my-games/             → mapped to Documents/My Games/<Game>
├── steam-userdata/       → mapped to AppData/Local/Steam/userdata/<id>
│   ├── 201689394/        → Steam userdata for zLeafNinja
│   └── 22202/            → Steam userdata for another account?
├── cache-backups/        → Ludusavi backups (read-only, do not touch)
├── ludusavi-backup/      → Ludusavi configured backups
├── smart-sync/           → Smart save checker infrastructure
│   ├── lib/              → Library modules
│   ├── tests/            → 57 BATS tests
│   └── smart-sync.sh     → Core sync engine
├── link-game-saves.sh    → Auto-symlink script (systemd triggered)
├── link-saves.sh         → Manual explicit symlink script
├── link-saves.log        → Run log
├── sync-cuphead-saves.sh → Post-game sync for Cuphead (excluded from symlinks)
├── SMART-SAVE-SYNC-SPEC.md → Documentation
├── .stfolder/            → Syncthing marker
└── .stversions/          → Syncthing version backups
```

## Symlinks Created
**27,746 symlinks** created across all 25 compatdata prefixes by `link-game-saves.sh`.
Last run: 2026-06-17 22:24 UTC.

### Exclusion
Cuphead is excluded from symlinks (breaks with symlinks — causes "SAVE FAILED"). Instead:
- GameSaves copies saves to the prefix directory
- `sync-cuphead-saves.sh` syncs back after play

### How it works (link-game-saves.sh):
1. Walks each GameSaves subdirectory (appdata-roaming, appdata-local, etc.)
2. Checks if the directory contains actual save files (excludes .log, .txt, .png, .jpg, .ini, .cfg, .bak)
3. For each compatible directory, creates a symlink at:
   `compatdata/<appid>/pfx/drive_c/users/steamuser/<AppData path>/<Game>/`
4. Skips existing symlinks to avoid self-referential loops
5. Backs up existing real directories before replacing with symlinks

## Ludusavi Backup
**Path**: `/var/home/jevonx/ludusavi-backup/` (2.6G)
**Managed by**: com.github.mtkennerly.ludusavi (flatpak installed)
**Game saves backed up**: See subdirectories for game list

## Syncthing
**Status**: Running (started)
**Config**: `/var/home/jevonx/.config/syncthing/config.xml`
**State**: `/var/home/jevonx/.local/state/syncthing/`
**Shared folders**:
- GameSaves (path: `/var/home/jevonx/GameSaves/`)
- ludusavi-backup (path: `/var/home/jevonx/ludusavi-backup/`)
**GUI**: http://localhost:8384 (localhost only)
**Peers to connect**: TrueNAS (if online), other machines via syncthing IDs

## Smart Save Checker
**Location**: `/var/home/jevonx/GameSaves/smart-sync/`
**Purpose**: Protects against empty/default file overwrites during Syncthing sync
**Mechanism**: Compares current save files against last-known-good manifest; restores from .stversions if corruption detected
**Status**: Configured but timer/path services not yet enabled (needs systemctl --user enable)
