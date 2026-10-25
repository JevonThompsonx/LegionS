# ROG Drive (Micron 2400) — Data Reference

The Micron 2400 (512GB NVMe, S/N 23164004D36C) was the original OS drive in a Bazzite-ROG Ally install. It is now physically connected as a second drive on this machine (bazzite-mobile).

## Partition Layout
```
nvme0n1 (Micron 2400 512GB)
├─ nvme0n1p1 (unknown/empty)      — no filesystem detected
├─ nvme0n1p2 (ext4, 1.3G)         → /run/media/system/bazzite_xboot (boot partition)
└─ nvme0n1p3 (btrfs, 50.3G, 89% used) → /run/media/system/bazzite (OS)
    ├─ subvol /root (ID 256)      — OS root (ostree deploy)
    ├─ subvol /home (ID 257)      — /var/home
    └─ subvol /var  (ID 258)      — /var
```

## ROG Drive User Data
**User**: jevonx (uid=1000)
**Home**: `/run/media/system/bazzite/home/jevonx/`
**Size**: ~29.5G

### Key Directories
```
/run/media/system/bazzite/home/jevonx/
├── Games/                        ← 27 game directories, 61 exe files
├── GameSaves/                    ← Syncthing-managed saves
├── save-data/                    ← Duplicate save directory
├── ludusavi-backup/              ← Ludusavi backups
├── .var/app/                     ← Flatpak app data (Brave, Moonlight, etc.)
├── .config/                      ← User config (hhd, gamescope, etc.)
├── .steam/                       ← Steam symlinks
├── .local/share/Steam/           ← Steam data
│   └── userdata/201689394/config/shortcuts.vdf  ← 24 shortcuts (original)
├── .ssh/                         ← SSH keys
├── .mozilla/                     ← Firefox profile
├── .opencode/                    ← OpenCode config (~200MB)
└── ~/ (literal file named ~)     ← Leftover from the old install
```

## Games on ROG Drive (27)
Absolum, Beastieball, Celeste, Constance, Dragon Quest I-II-SteamGG.NET,
Eden-Linux-ROG.AppImage, Eden-Linux-steamdeck.AppImage, Esoteric Ebb,
Flowstone Saga, Furi, Haste.v1.7.c, LS, Mega Man BN Legacy Collection,
Neon.Abyss.2.v2025.08.01, NSwitch (Nintendo Switch ROMs),
Ogu and the Secret Forest, Possessor(s), Ra Ra BOOM,
RiftOfTheNecroDancer-SteamGG.NET, Risk of Rain 2, Rune Factory 3 Special,
Shape of Dreams, Spiritfall, Suikoden I&II HD Remaster, Tactics Ogre Reborn

Notable: No Cuphead, Hades, Dispatch, or other "recently used" games — those are on the rog machine's Games_Data.

## Original Shortcuts (24)
Located at: `/run/media/system/bazzite/home/jevonx/.local/share/Steam/userdata/201689394/config/shortcuts.vdf`
Points to: `/var/home/jevonx/Games/<Game>/<Exe>` (uses bazzite-rog home path)
Only 3/24 resolve currently (flatpak apps).
Fixes to make usable:
```bash
sudo rm /var/home/jevonx/Games
sudo ln -s /run/media/system/bazzite/home/jevonx/Games /var/home/jevonx/Games
```
After this, the ROG drive games become playable via the bazzite-rog shortcuts.

## ROG Drive Shortcuts vs Scanner Shortcuts
The ROG drive has 24 shortcuts from the bazzite-rog install (different games). The scanner on the rog machine created 45 shortcuts for different games. These are two separate sets of games on two separate storage devices. They don't overlap.
