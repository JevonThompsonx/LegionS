# Migration Status

> Historical (June 2026, written on bazzite-mobile). The rog machine is back online on the
> expanded 4TB drive as of July 2026 — see `../state/CURRENT.md` for live status.

## Completed

### Flatpak Apps (31 installed)
- com.brave.Browser
- com.github.mtkennerly.ludusavi
- com.moonlight_stream.Moonlight
- com.nextcloud.desktopclient.nextcloud
- com.rustdesk.RustDesk
- net.retrodeck.retrodeck
- org.jellyfin.JellyfinDesktop
- org.virt_manager.virt-manager
- Plus 23 pre-installed (Firefox, GNOME utilities, etc.)

### User Data Transferred (ROG drive → bazzite-mobile)
- GameSaves (2.2G, merged from 3 sources)
- save-data (2G, merged into GameSaves)
- ludusavi-backup (2.6G)
- .var/app (936M)
- .config (394M)
- .steam
- .opencode (196M)
- .mozilla
- Dotfiles and scripts

### Save Files Linked
- link-game-saves.sh ran: **27,746 symlinks** created in compatdata prefixes
- Cuphead: real copy (excluded from symlinks, breaks otherwise)
- GameSaves logged at `/var/home/jevonx/GameSaves/link-saves.log`

### Shortcuts Fixed (45 → 44)
- Deleted "Command" (Windows start.exe shim)
- Renamed "Lossless Scaling Exe" → "Lossless Scaling"
- Renamed "T Lo H" → "Trails Beyond the Horizon"
- Renamed "TGAAC" → "The Great Ace Attorney Chronicles"
- Renamed "Necro Dancer" → "Crypt of the NecroDancer"
- Fixed Hades StartDir → game root (was x64/)
- Fixed Jedi Survivor StartDir → game root (was Win64/)
- Fixed Factory - Guardians of Azuma StartDir → game root (was Win64/)
- Fixed Split Fiction StartDir → game root (was Win64/)
- Rekeyed shortcuts to contiguous 0..44 (Steam requirement)

### System Config
- Hostname: `bazzite-mobile` set (runtime only, needs /etc/hostname edit to persist across reboot with rpm-ostree)
- Machine-id: regenerated
- Syncthing: configured for GameSaves + ludusavi-backup, running

## Blocked

### Rog machine (100.119.14.48) is OFFLINE
The rog machine has the Samsung 970 EVO Plus with Games_Data partition containing the 45+ games the scanner found. Until it comes back online:
- The 33 unlaunched games cannot be accessed
- The Add-Non-Steam-Games-Linux tool's scan results are stuck on that machine
- The shortcuts.vdf there may have different content from this machine

### Game exe paths are broken
All 45 shortcuts on this machine point to `/home/jevonx/Games/<Game>/` which resolves to `/var/mnt/games-data` (empty). The actual games are in two places:
1. ROG drive: `/run/media/system/bazzite/home/jevonx/Games/` (27 games, 61 exes) — accessible now
2. Rog machine: `/run/media/system/Games_Data/` (45+ games) — inaccessible (machine offline)

## Next Steps See `~/Projects/rog/TODO.md`
