> **Migration note (2026-10-25):** This repo now targets the Lenovo Legion Go S (Z2 Go).
> Hostname is `legion-go-s`; ROG Ally (`rog`) is retired. ROG-era IPs/hostnames below are historical.

# Rog Architecture Document

> **Updated 2026-07-11.** Repo now lives on `aphrodite` (100.127.201.94, LAN 192.168.67.210).
> See `state/CURRENT.md` for live status.

## Machine Overview

### Machine 0: aphrodite (current dev machine)
- **Tailscale IP**: 100.127.201.94
- **LAN**: 192.168.67.210 (wlp3s0)
- **Purpose**: Current dev machine; holds `~/Projects/rog` and `~/Projects/Roms`; ROM archive destination `~/Media/Games/Archive`

### Machine 1: bazzite-mobile (opencode sandbox) — OFFLINE (3+ days as of 2026-07-11)
- **Tailscale IP**: 100.119.216.62
- **DNS**: bazzite-mobile.tail958f87.ts.net
- **Internal hostname**: bazzite
- **LAN**: 192.168.67.80/24 (wlan0)
- **Purpose**: Development machine, has the ROG drive (Micron 2400) connected

Storage:
- **nvme1n1** (LUKS-encrypted, 464G) — OS drive, /var/home/jevonx, /var
- **nvme0n1** (Micron 2400 512GB) — ROG drive (old bazzite-rog install), mounted at:
  - `/run/media/system/bazzite` — btrfs volume (3 subvols: root, home, var)
  - `/run/media/system/bazzite_xboot` — boot partition

### Machine 2: rog (100.119.14.48) — ONLINE (ROG Ally RC71L handheld)
- **Tailscale IP**: 100.119.14.48 (tailnet node name `rog`)
- **Local hostname**: `bazzite-mobile` (STALE — inherited from copied OS image; scripts in
  `~/Projects/Roms` expect this hostname, rename both together)
- **LAN**: 192.168.67.203 (USB ethernet, flaky under load) + 192.168.67.151 (wlan0, stable)
- **SSH**: `root@100.119.14.48`, `root@192.168.67.151`, or `jevonx@rog`
- **Purpose**: Gaming machine, ROG Ally original
- **OS**: bazzite-deck-gnome:stable 43.20260420, kernel 6.17.7-ba29
- **hhd**: `/etc/hhd/state.yml` — `gamemode.gamescope.dpms: false` set 2026-07-11
  (screen stayed black after display-sleep poweroff; backup `.bak-20260711`)

Storage (4TB drive, expanded 2026-06/07 — see migration/DRIVE_EXPANSION.md):
- **nvme0n1** (4TB) — single drive, partitioned:
  - p1: ext4 "Games Data" (3.5T) — game library, /run/media/system/Games_Data
  - p2: vfat EFI
  - p3: ext4 bazzite_xboot
  - p4: btrfs bazzite (197G) — OS, root UUID 6b91a72a

### Machine History
Migrated FROM bazzite-rog (old install on Micron 2400) TO the rog machine (100.119.14.48, Samsung 970 EVO Plus). The Micron 2400 (old ROG drive) is now physically connected to bazzite-mobile as a second drive.

**Steam Account History on ROG**: The Add-Non-Steam-Games script was run while account `195131525` was logged in. User then switched to primary account `201689394` (zLeafNinja). Shortcuts were transferred from 195131525 → 201689394 on 2026-06-18.

## Network Topology (via Tailscale, 2026-07-11)

| Hostname | Tailscale IP | Status |
|----------|-------------|--------|
| aphrodite | 100.127.201.94 | **ONLINE** (this machine) |
| rog | 100.119.14.48 | **ONLINE** (ROG Ally gaming machine) |
| bazzite-mobile | 100.119.216.62 | offline 3+ days |
| roadman | 100.116.138.103 | **ONLINE** |
| skellyshome | 100.71.134.54 | **ONLINE** |
| templetonbak | 100.95.134.16 | **ONLINE** |
| iphone-15-pro | 100.83.187.120 | offline ~4h |

## Key Paths

### Steam Data (on rog machine — primary gaming Steam)
- **Steam install**: `/var/home/jevonx/.local/share/Steam/`
- **Active account**: zLeafNinja (SteamID 76561198161955122, account_id 201689394)
- **Previous account**: 195131525 (SteamID 76561198155397253) — shortcuts transferred, no longer active
- **Shortcuts.vdf**: `/var/home/jevonx/.local/share/Steam/userdata/201689394/config/shortcuts.vdf` (53 shortcuts)
- **Grid artwork**: `/var/home/jevonx/.local/share/Steam/userdata/201689394/config/grid/` (152+ files)
- **Compatdata**: `/var/home/jevonx/.local/share/Steam/steamapps/compatdata/` (14 prefixes)
- **Library folder**: `/var/home/jevonx/.local/share/Steam`

### Steam Data (on bazzite-mobile — dev machine, secondary)
- **Shortcuts.vdf**: `/var/home/jevonx/.local/share/Steam/userdata/201689394/config/shortcuts.vdf`
- **Grid artwork**: `/var/home/jevonx/.local/share/Steam/userdata/201689394/config/grid/` (152 files)
- **Compatdata**: `/var/home/jevonx/.local/share/Steam/steamapps/compatdata/` (25 prefixes)
- **Library folder 1**: `/run/media/system/Games_Data/SteamLibrary` (NFS backend, offline)

### Game Data (on ROG drive, accessible from bazzite-mobile)
- **Path**: `/run/media/system/bazzite/home/jevonx/Games/`
- **27 game directories**, 61 exe files
- **Original shortcuts.vdf**: `/run/media/system/bazzite/home/jevonx/.local/share/Steam/userdata/201689394/config/shortcuts.vdf` (24 shortcuts)

### GameSaves
- **Path**: `/var/home/jevonx/GameSaves/` (2.2G)
- **App data subdirs**: appdata-roaming, appdata-local, appdata-locallow, saved-games, my-games, steam-userdata
- **Symlinks created**: 27,746 in compatdata prefixes
- **Ludusavi backup**: `/var/home/jevonx/ludusavi-backup/` (2.6G)
- **link-game-saves.sh**: `/var/home/jevonx/GameSaves/link-game-saves.sh`

### Game Data (on rog machine)
- **Path**: `/run/media/system/Games_Data/` (ext4, 1.6T, 80% used)
- **Contains**: SteamLibrary, T/ (games), NSwitch/, Eden-Windows/, Add-Non-Steam-Games-Linux tool
- **Game directories**: 53 games as non-Steam shortcuts
- **Add-Non-Steam-Games tool**: `/run/media/system/Games_Data/Add-Non-Steam-Games-Linux/`
  - Run as jevonx: `cd /run/media/system/Games_Data/Add-Non-Steam-Games-Linux && .venv/bin/python3 main.py`

## Games Symlink (bazzite-mobile only — ROG Ally is not affected)
On bazzite-mobile: `/home/jevonx/Games` → `/var/mnt/games-data` (empty directory). The 24 old bazzite-rog shortcuts reference this path. This is only relevant if you want to run the old bazzite-rog games on bazzite-mobile.

On the **rog machine**: shortcuts use absolute paths under `/run/media/system/Games_Data/` — no symlink problem. All 53 shortcuts point directly to game exes on the mounted Games_Data partition.
