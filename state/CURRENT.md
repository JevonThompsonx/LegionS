# Current State — 2026-08-11

- Game-mode idle black screen **FIXED + VERIFIED**. Root cause: Steam's game-mode sleep
  sequence turns the display off at idle then requests suspend; `suspend.target`/
  `sleep.target` were masked (2026-06-25 workaround) → suspend refused → display left
  dead → blank screen until reboot.
- Real fix (2026-08-11): `systemctl unmask suspend.target sleep.target` → idle now does a
  real s2idle sleep that wakes cleanly. Verified: user power-button sleep/wake + SSH-triggered
  suspend → `PM: suspend entry (s2idle)` → `amdgpu SMU resumed` → `PM: suspend exit` (0.29s).
  hibernate/hybrid-sleep stay masked (intentional).
- Also kept: Steam `IdleBacklightDim{AC,Battery}Seconds` → 0 in config.vdf (backup
  `config.vdf.bak-20260810`) so the backlight never dims in game mode.
- Update situation: stable channel = F43 43.20260420, no newer; Bazzite 44.20260802
  (F44 rebase) released but deck-gnome stable tag not yet flipped; monitor and rebase later,
  then re-verify suspend/wake + dock.
- Recovery escape documented in docs/display-blackscreen.md (VT flip / gamescope restart).

## Previous session — 2026-07-11 (black-screen recovery + ROM transfer)

## Switch ROM archive transfer — COMPLETE ✅ (reboot embargo lifted)

`switch-archive.sh` archived to `~/Media/Games/Archive` on aphrodite, both verified
full-size and removed from handheld:
- v-bayonetta_3.xci (15.97GB)
- Xenoblade 3.xci (15.97GB)
Both transfers survived dock unplugs (dock ethernet = 192.168.67.203; script retried
via wlan .151). Handheld is now safe to update + reboot.

## Handheld (ROG Ally RC71L) — ONLINE, healthy

- Tailscale node name: `rog` (100.119.14.48)
- **Local hostname is stale: `bazzite-mobile`** — inherited from the copied OS image.
  Do NOT rename yet: `switch-archive.sh`/`switch-transfer.sh` validate
  `EXPECTED_HOSTNAME="bazzite-mobile"`. Rename + update scripts together later.
- LAN: 192.168.67.203 (USB ethernet, **flaky — drops under sustained load**, caused
  rsync `code 12` mid-transfer) and 192.168.67.151 (wlan0, stable)
- 4TB drive expansion COMPLETE — booted from the expanded drive:
  - nvme0n1p1 3.5T ext4 "Games Data" at /run/media/system/Games_Data
  - nvme0n1p4 197G btrfs bazzite OS (root UUID 6b91a72a, healthy)
- OS: bazzite-deck-gnome:stable 43.20260420, kernel 6.17.7-ba29 — ~3 months stale,
  update after transfer finishes (`ujust update`, then reboot)

## Display black-screen — ROOT CAUSE FOUND (2026-07-11)

Symptom: screen black until full restart. Real cause: **dock undock leaves amdgpu
connector DP-11 (VIZIO TV through dock) stuck "connected"** — hotplug event missed at
the hardware/typec level. Gamescope keeps outputting to the phantom external display;
internal panel (eDP-1) stays disabled. Not suspend, not PSR, not purely display-sleep.

**Recovery WITHOUT restarting** (done live 2026-07-11, transfer kept running):
```bash
# as root on the handheld
echo off > /sys/class/drm/card1-DP-11/status   # kill phantom connector
chvt 3; sleep 3; chvt 1                        # force gamescope DRM re-enumeration
echo detect > /sys/class/drm/card1-DP-11/status  # restore auto-detect for redocking
```
After this, eDP-1 re-enabled and gamescope switched back to internal panel.
Happened twice on 2026-07-11 (each dock unplug); second time the phantom DP-11 had
already vanished from sysfs and the `chvt 3; sleep 3; chvt 1` flip alone fixed it —
that VT flip is the essential step. Verified working via
`gamescopectl screenshot /tmp/shot.png` (game rendering fine mid-battle).
SSH note: known_hosts entry for 100.119.14.48 was stale (old install); replaced after
verifying the key matches the trusted LAN entries.
Fallback if VT switch ever fails: `systemctl --user restart
gamescope-session-plus@steam.service` (as jevonx) — restarts Steam UI only, does not
kill SSH/rsync transfers.

Also applied (defense in depth vs display-sleep poweroff): `/etc/hhd/state.yml`
`gamemode.gamescope.dpms: false` (backup `.bak-20260711`).
Long-term: Bazzite update + reboot should clear stuck HPD; test dock unplug after.

Also fixed: user `syncthing.service` pointed at nonexistent
`~/.local/bin/syncthing` (178-restart crash loop). ExecStart now `/usr/bin/syncthing`;
stopped for now so sync traffic doesn't fight the ROM transfer — starts on next boot.

## Machines

| Machine | Tailscale | Notes |
|---------|-----------|-------|
| aphrodite | 100.127.201.94 | This machine (repo lives here now), LAN 192.168.67.210 |
| rog (handheld) | 100.119.14.48 | ROG Ally, local hostname stale "bazzite-mobile" |
| bazzite-mobile | 100.119.216.62 | Dev machine w/ Micron 2400 — OFFLINE 3 days |
| roadman / skellyshome / templetonbak | online | — |

## After transfer completes

1. ~~`ujust update` on handheld, reboot, verify display-sleep wake~~ — SUPERSEDED 2026-08-10:
   stable channel has no newer F43 image; see F44 rebase item in TODO.md
2. Start syncthing on handheld, confirm peers
3. Resume `~/Projects/rog/TODO.md` (unlaunched games, game copies from Micron 2400 —
   needs bazzite-mobile back online)
