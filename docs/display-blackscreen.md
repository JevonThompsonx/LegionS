# Display Black Screen on Idle — ROG Ally (bazzite-deck-gnome)

## Current State — 2026-08-11 (FIXED + VERIFIED)

Machine: `rog` (Tailscale 100.119.14.48), ROG Ally RC71L, Bazzite deck-gnome stable
43.20260420, kernel 6.17.7-ba29, DCN 3.1.4. Steam config lives at
`/var/home/jevonx/.local/share/Steam/`.

## 1. Symptom

User report: in game mode, if the ROG runs a while without playing a game, the screen
goes blank; only a full reboot recovers. Desktop mode not confirmed (not needed — fixed
in game mode).

## 2. Root cause

Two stacked problems, both hit when game mode goes idle:

1. Steam deckui dims the backlight to 0 after 5 min (battery) / 15 min (AC) via
   `IdleBacklightDimBatterySeconds` / `IdleBacklightDimACSeconds`, then requests system
   suspend after its idle timer (~60 min).
2. `suspend.target` + `sleep.target` were **masked** (along with hibernate/hybrid-sleep),
   a 2026-06-25 user/agent workaround. The mask made the request a dead-end: logind logged
   `The system will suspend now!` → `Unit suspend.target is masked, refusing operation.`
   → Steam got `DBus.AccessDenied`. The display had already been turned off by the
   sleep sequence and **never came back** → permanent black screen until reboot.

The mask was the actual killer: the display-off happens as part of the sleep request, so
refusing the suspend left the panel off with no wake path.

## 3. Evidence

| File / command | Finding |
|---|---|
| `/etc/systemd/system/{suspend,sleep,hibernate,hybrid-sleep}.target` | `-> /dev/null`, mtime 2026-06-25 17:13, owned by no rpm package (verified `rpm -qf` = not owned) |
| journal game-mode boots | One real `systemd-logind: The system will suspend now!` per long game-mode boot, immediately followed by `Unit suspend.target is masked, refusing operation.` and `steam: Error org.freedesktop.DBus.Error.AccessDenied`. Examples (boot indices shift as boots accumulate): Aug 9 18:11:45, Aug 10 20:04:47, Aug 10 23:42:13 (the last re-verified 2026-08-11, all three lines). **Zero** successful s2idle entries while masked |
| Steam deckui (`steamui/chunk~2dcc5aaf7.js`) | Writes `idle_backlight_dim_ac_seconds` / `idle_backlight_dim_battery_seconds` (the dim); no separate display-sleep timeout field — sleep is baked into the client's power manager |
| hhd `/etc/hhd/state.yml` | `gamemode.gamescope.dpms: false` (applied 2026-07-11 for the dock black screen) — covers hhd's own overlay DPMS only; did not prevent Steam's sleep sequence |
| dmesg | 3 amdgpu `dpp3_deferred_update` WARNINGs ~25.5s post-boot (DCN 3.1.4, dcn30_dpp.c 544/553/562). PSR NOT in use (`PSR support 0`) |

## 4. Fixes applied and verified

**Primary fix (2026-08-11): unmasked suspend + sleep.**
```bash
systemctl unmask suspend.target sleep.target
```
- `suspend.target` / `sleep.target` now `static` (not masked); `hibernate.target` /
  `hybrid-sleep.target` intentionally still masked (not part of the normal sleep path).
- Idle in game mode now does a **real s2idle suspend** instead of a refused dead-end; the
  power button / any input wakes it and the display returns.
- Verified by the user (power-button sleep/wake) and by SSH-triggered suspend: journal
  shows clean `PM: suspend entry (s2idle)` → `Freezing user space processes` →
  `amdgpu ... SMU is resumed successfully!` → `PM: resume devices took 0.292 seconds` →
  `PM: suspend exit`. No errors.
- The masks live in `/etc` (ostree config state, not image-owned), so **this change
  survives the future F44 rebase** — re-verify suspend + wake after that update anyway.

**Secondary fix (2026-08-10): disabled Steam idle backlight dim.**
- `/var/home/jevonx/.local/share/Steam/config/config.vdf`:
  `IdleBacklightDimBatterySeconds` 300→0, `IdleBacklightDimACSeconds` 900→0 (screen
  never dims in game mode). Backup: `config.vdf.bak-20260810` (owner jevonx; backup
  retains the original 300/900 values). Note: Steam rewrites `config.vdf` on settings
  changes — at backup time (2026-08-10) a fresh diff was exactly the two dim lines, but
  Steam has since rewritten the file (dim lines now at 596-597, still 0; extra
  shader-cache/server-region churn shows in a new diff). Kept even though the unmask is
  the real fix — it removes the dim-to-black phase and saves nothing but keeps the panel
  from dimming at all.
- Rollback: `cp /var/home/jevonx/.local/share/Steam/config/config.vdf.bak-20260810
  /var/home/jevonx/.local/share/Steam/config/config.vdf` (as jevonx), then next
  game-mode session.

## 5. Update situation (checked 2026-08-10)

- Deployed image digest `sha256:76c4da71f4dad7f897fa4f94464fd12e783943eb89c321335cff317e9ee97d9a`
  == ghcr `bazzite-deck-gnome:stable` tag digest (skopeo) — identical (built 2026-04-20).
  `rpm-ostree upgrade` → "No upgrade available"; `bootc upgrade` → refused due to
  rpm-ostree layered packages (git newt syncthing zenity).
- GitHub latest Bazzite release: **44.20260802 "Stable (F44.20260802)"** (Fedora 44 major
  rebase). `testing` channel images are fresh (2026-08-10).
- Conclusion: the F43 `stable` channel has nothing newer; the F43→F44 rebase must be
  delivered by an explicit rebase, not `ujust update`. Recommended: monitor
  `skopeo inspect docker://ghcr.io/ublue-os/bazzite-deck-gnome:stable` until it flips to
  an F44 digest, then rebase via the Bazzite-documented path (e.g. `ujust rebase-fresh`
  or equivalent — do NOT run this blindly; major-version rebase = user decision + reboot
  + post-update verification).

## 6. After the future F44 rebase

- Re-verify: suspend + wake (power button), dock un/dock (the 2026-07-11 DP-11 case),
  and that `suspend.target`/`sleep.target` are still unmasked (`systemctl is-enabled`).
- hibernate/hybrid-sleep may stay masked unless hibernation (swap) is set up.
- If anything regresses, the old masks were at `/etc/systemd/system/{suspend,sleep}.target`
  -> `/dev/null`; re-applying them is a one-liner (`systemctl mask suspend.target sleep.target`).

## 7. Recovery escape (no reboot)

If a black screen ever recurs (e.g. after the future rebase):

```bash
ssh root@100.119.14.48 'chvt 3; sleep 3; chvt 1'
```

(VT flip, essential step per 2026-07-11 notes — that recovery was proven for the dock
DP-11 case, NOT for the idle case; if the display does not return, reboot or re-apply the
fixes above.)

Fallback:

```bash
su - jevonx -s /bin/bash -c "systemctl --user restart gamescope-session-plus@steam.service"
```

(restarts the Steam session — exits any running game and drops back to Steam UI.)

## 8. Verification checklist

- [x] `systemctl unmask suspend.target sleep.target` applied; `is-enabled` = `static`
- [x] config dim values 0 (`grep IdleBacklightDim .../config.vdf`)
- [x] backup `config.vdf.bak-20260810` exists (owner jevonx)
- [x] power-button sleep + wake works (user-verified)
- [x] SSH-triggered suspend + wake clean (`PM: suspend entry/exit`, `SMU resumed`, 0.29s)
- [x] screen returns after wake in game mode
- [ ] re-verify after F44 rebase (suspend/wake, dock)

## 9. Open risks / notes

- Steam rewrites `config.vdf` on settings changes (line numbers shift); the dim=0 values
  have persisted so far — re-check after any game-mode settings change.
- dim=0 is interpreted as "off" by the deckui; revisit if a future Steam build changes
  semantics.
- F44 rebase not yet available on the deck-gnome stable tag.
- The 2026-06-25 masks were user-created with no recorded rationale; unmasking was
  verified safe on this kernel (wake works), but re-test after the F44 rebase.
