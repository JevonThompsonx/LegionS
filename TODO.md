# TODO — ROG Ally Gaming Machine

> Scope: machine ops tasks (OS, Steam, drives, saves). For gamepack code work see MODERNIZATION-TODO.md.

## NEW (2026-08-11)

- [x] Game-mode idle black screen FIXED + VERIFIED (2026-08-11): `systemctl unmask
      suspend.target sleep.target` — idle now does real s2idle sleep that wakes cleanly
      (power-button + SSH-triggered suspend both verified; journal `PM: suspend entry/exit`,
      `SMU resumed`). hibernate/hybrid-sleep stay masked.
- [x] Steam idle backlight dim disabled (2026-08-10): `IdleBacklightDim*Seconds` → 0 in
      config.vdf (backup config.vdf.bak-20260810)
- [ ] Rebase handheld to Bazzite 44 (F44.20260802) once deck-gnome stable tag flips from
      F43 (monitor via skopeo) — replaces the old `ujust update` plan
- [ ] After F44 rebase: re-verify suspend+wake (power button) and dock un/dock; confirm
      suspend.target/sleep.target still unmasked

## NEW (2026-07-11)

### After current ROM transfer finishes (do NOT reboot before then)
- SUPERSEDED: ~~`ujust update` on handheld (image is 3 months stale: 43.20260420, kernel
      6.17.7), reboot~~ — stable channel has no newer F43 image; replaced by F44 rebase
      item above (2026-08-10)
- [ ] Verify dock undock after Bazzite update: unplug dock, screen should switch to
      internal panel (root cause was stuck DP-11 hotplug — recovery recipe in
      `state/CURRENT.md`; hhd `gamemode.gamescope.dpms: false` also applied)
- [ ] Start syncthing on handheld (`systemctl --user start syncthing`), verify peers
      (unit ExecStart fixed to /usr/bin/syncthing on 2026-07-11)
- [ ] Investigate flaky USB ethernet (192.168.67.203) — dropped mid-rsync; wlan .151 stable
- [ ] Rename handheld hostname `bazzite-mobile` → `rog` AND update
      `~/Projects/Roms/scripts/switch-*.sh` `EXPECTED_HOSTNAME` in the same change

## DONE (2026-06-18)

### Steam Account Transfer
- Shortcuts from account 195131525 → 201689394 (zLeafNinja, active account)
- 12 junk shortcuts deleted (data files, sub-exes picked up by scanner)
- 7 names fixed (Danganronpa V3, Rune Factory, Crypt of the NecroDancer, etc.)
- Hades: exe fixed to x64Vk/Hades.exe (Vulkan 64-bit, best perf on AMD)
- Binding of Isaac: exe fixed (was pointing through Tales of Arise folder)
- 9 missing games added: Dragon Marked for Death, Elden Ring, ELDEN RING NIGHTREIGN,
  FEZ, It Takes Two, OCTOPATH TRAVELER II, Tales of Arise, The Witcher 3, Tokyo Mirage Sessions #FE Encore
- Grid artwork (152 files) copied to 201689394
- Total: 53 shortcuts in correct user account

### Save Data
- link-game-saves.sh run on ROG — saves linked into existing 14 Proton prefixes

### Artwork Download
- Add-Non-Steam-Games [3] run for all 53 shortcuts (2026-06-18)

## Priority: HIGH

### 1. Launch Each Unlaunched Game (41 remaining)
For each game not yet launched:
1. Launch from Steam
2. Let Proton create prefix
3. Quit game
4. Run `bash ~/GameSaves/link-game-saves.sh`

Games: Card-en-ciel, Crypt of the NecroDancer, Deltarune, Dispatch, Dragon Marked for Death,
Eden-windows, Eiyuden Chronicle Rising, Elden Ring, ELDEN RING NIGHTREIGN, Esoteric Ebb,
Etrian Odyssey (1/2/3), FEZ, Final Fantasy III, Final Fantasy X/X-2, Furi, Ghost Trick,
Haste, Hollow Knight Silksong, It Takes Two, MARVEL Cosmic Invasion, Nine Sols,
OCTOPATH TRAVELER 0, OCTOPATH TRAVELER II, Of Ash and Steel, Ogu and the Secret Forest,
Portal, Portal 2, Pragmata, Schedule I, Slime Rancher, Tales of Arise, Tales of Xillia,
Tactics Ogre Reborn, The Binding of Isaac Repentance, The Great Ace Attorney Chronicles,
The Last of Us, The Witcher 3, Tokyo Mirage Sessions #FE Encore, Trails Beyond the Horizon, Unbeatable

### 2. Copy Games from bazzite-rog Drive → ROG (not on ROG yet)
Games at `/run/media/system/bazzite/home/jevonx/Games/` on bazzite-mobile not in ROG Games_Data:
Absolum, Beastieball, Celeste, Constance, Dragon Quest I-II-SteamGG.NET, Flowstone Saga,
Mega Man BN Legacy Collection, Neon.Abyss.2.v2025.08.01, Possessor(s), Ra Ra BOOM,
RiftOfTheNecroDancer-SteamGG.NET, Risk of Rain 2, Rune Factory 3 Special, Shape of Dreams,
Spiritfall, Suikoden I&II HD Remaster

Already on ROG (skip): Esoteric Ebb, Furi, Haste, Ogu, Tactics Ogre Reborn

Copy command (run on bazzite-mobile):
```bash
rsync -av --progress \
  "/run/media/system/bazzite/home/jevonx/Games/<GameDir>/" \
  "root@100.119.14.48:/run/media/system/Games_Data/<GameDir>/"
```
Then run Add-Non-Steam-Games option [2] on ROG to add shortcuts.

## Priority: MEDIUM

### 3. Syncthing peers
Connect to TrueNAS, roadman (100.116.138.103), skellyshome (100.71.134.54) for GameSaves sync.

### 4. Smart save checker timers
```bash
systemctl --user enable smart-save-checker.timer smart-save-checker.path
systemctl --user start smart-save-checker.timer smart-save-checker.path
```

## Priority: LOW

### 5. Fix bazzite-mobile hostname persistence
```bash
sudo hostnamectl set-hostname bazzite-mobile
```

## Environment Reference

### ROG Ally (rog)
- **Tailscale**: 100.119.14.48
- **SSH**: `root@100.119.14.48` or `jevonx@rog`
- **Steam**: user 201689394 (zLeafNinja), 53 non-Steam shortcuts
- **Games**: `/run/media/system/Games_Data/` (1.6T ext4)
- **Add-Non-Steam-Games**: `/run/media/system/Games_Data/Add-Non-Steam-Games-Linux/`

### bazzite-mobile (this machine)
- **Tailscale**: 100.119.216.62
- **ROG drive**: `/run/media/system/bazzite/` (Micron 2400, old bazzite-rog OS)
- **bazzite-rog games**: `/run/media/system/bazzite/home/jevonx/Games/`
