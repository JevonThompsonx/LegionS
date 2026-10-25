# Steam Configuration

## Account
- **Persona**: zLeafNinja
- **AccountName**: jevonxt
- **SteamID64**: 76561198161955122
- **Account ID**: 201689394
- **Auto-login**: enabled
- **Last online**: 2026-06-17 (~18:00 UTC)

## Steam Install
- **Type**: rpm-ostree (native, NOT flatpak)
- **Binary**: /usr/sbin/steam
- **Data directory**: `/var/home/jevonx/.local/share/Steam`
- **Launcher script**: /usr/bin/bazzite-steam
- **Running**: YES (PID 6115)

## Directory Structure
```
/var/home/jevonx/.local/share/Steam/
├── steamapps/
│   ├── common/              # Game installs (empty — no Steam games installed on this machine)
│   ├── compatdata/          # Proton prefixes (25 dirs)
│   ├── libraryfolders.vdf   # Library paths
│   │   # 0: /var/home/jevonx/.local/share/Steam
│   │   # 1: /run/media/system/Games_Data/SteamLibrary (offline — NFS backend)
│   ├── appmanifest_228980.acf  # CS2 (Steam game, not a shortcut)
│   └── shadercache/
├── userdata/
│   └── 201689394/           # zLeafNinja's user data
│       ├── config/
│       │   ├── shortcuts.vdf          # 45 non-Steam shortcuts
│       │   ├── shortcuts.vdf.bak.*    # backups
│       │   ├── shortcuts-manifest.json # scanner manifest
│       │   ├── loginusers.vdf
│       │   └── localconfig.vdf
│       └── grid/            # Artwork (152 files)
├── config/
│   ├── loginusers.vdf       # Steam login data
│   └── libraryfolders.vdf
└── logs/
```

## Library Folders
Two library folders configured:
1. **Library 0**: `/var/home/jevonx/.local/share/Steam` — local, has 1 Steam game (CS2, app 228980)
2. **Library 1**: `/run/media/system/Games_Data/SteamLibrary` — 20 Steam games, but this mount is currently empty (backend NFS server at 100.68.123.39 is offline)

## Compatdata Prefixes (25)
All are 308-730MB (Proton prefixes with actual game data for the 12 launched games):

| AppID | Game | Size |
|-------|------|------|
| 0 | Root prefix | 730M |
| 1493710 | (unknown) | 314M |
| 2001120 | (unknown) | 309M |
| 2180100 | (unknown) | 311M |
| 2307956191 | (unknown) | 308M |
| 2751009950 | (unknown) | 320M |
| 2762301062 | (unknown) | 309M |
| 2838846817 | (unknown) | 727M |
| 2844747063 | (unknown) | 318M |
| 291550 | (unknown) | 767M |
| 3043076606 | (unknown) | 309M |
| 3051717631 | (unknown) | 308M |
| 3139174814 | (unknown) | 309M |
| 3521963739 | (unknown) | 308M |
| 3658110 | (unknown) | 285M |
| 3730360677 | (unknown) | 309M |
| 3826476388 | (unknown) | 330M |
| 3838297451 | (unknown) | 308M |
| 3864962558 | (unknown) | 310M |
| 3887261341 | Split Fiction | 308M |
| 3930144540 | (unknown) | 309M |
| 4187498376 | (unknown) | 310M |
| 4286379091 | (unknown) | 309M |
| 999997 | Test | 584K |
| 999998 | Test | 588K |

## Config gap: hardcoded Steam user path

`gamepack/internal/proton` `SteamRunning` reads `/var/home/jevonx/.steam/steam.pid` + `steam.pipe`
literally, ignoring `cfg` — the tool is silently bound to user `jevonx`. Tracked in
MODERNIZATION-TODO.md ("Parameterize the hardcoded Steam user path"): add
`SteamUser`/`SteamHome` to `cfg.Config` and derive the pid/pipe paths from it.

## Grid Artwork (152 files)
Downloaded via SteamGridDB API (key in Add-Non-Steam-Games-Linux/.env).
Stored in `/var/home/jevonx/.local/share/Steam/userdata/201689394/config/grid/`.
Filenames: `<appid>p.png` (portrait), `<appid>_hero.png` (hero), `<appid>_logo.png` (logo), `<appid>_icon.png` (icon), `<appid>.png` (horizontal).
