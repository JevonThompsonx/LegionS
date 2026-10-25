# Game Archiver ("gamepack") — Implementation Plan

> Status: **COMPLETED 2026-08-10 — shipped, tested end-to-end on the ROG, live in use.**
> Target: Go CLI/TUI (`bubbletea`) that runs **on the ROG Ally** (`bazzite-mobile`, gamescope game mode), cross-compiled on the dev machine and scp'd over.
> Every path, appid, size, and tool version below was **live-verified via SSH** on 2026-08-09. Corrections/refinements to the supplied recon are marked **⚠️ VERIFIED-DIFF**.
> Implementation notes and live drift from this plan: see `gamepack/README.md`.

## Locked decisions (user, 2026-08-09)
- **Compression: ALWAYS max — `zstd -T1 --ultra -22 --long=27`** for PC games and saves. (User accepted slow runtimes; near-max ratio.)
- **Prefix: YES, archive `compatdata/<appid>`** (`--with-prefix`) — packed as `prefix.tar.zst` (minus `shadercache/`, `pfx.lock`), restored on unpack. Interactive launch flow remains as fallback when a prefix archive is absent.

## Post-plan changes (shipped, live)
- **Remove-source prompt**: after a successful pack the TUI asks `Remove it to free space? [y/N]` (CLI `--remove-source`) — packing now actually reclaims space instead of leaving the game in place.
- **Running-game detection hardened**: pfx.lock flock proved UNRELIABLE (stayed unlocked while Split Fiction ran); detection is now a live non-zombie process matching the exe name, with two-consecutive-poll exit confirmation.
- **Partial-archive re-pack**: a crash/reboot leaves a dir without `manifest.json`; Pack now treats it as incomplete and re-packs (zstd's no-overwrite-of-existing-stdin-output is handled by removing the stale file first). Regression-tested.
- **Index self-healing**: `IsArchived` is derived from `manifest.json` presence, so `--refresh` no longer loses archived state.
- **Proton prefixes relocated to `Games_Data/CompatData/`** (19G) and symlinked back to `compatdata/<appid>`; Citron relocated to `Games_Data/Citron/`. Games verified working (incl. Tales of Arise).
- **NSwitch CLI parity**: `--pack-ns` / `--unpack-ns`, with index entries appended on pack (previously unpack couldn't find a ROM that was never indexed). E2E round-trip verified on a 32M ROM.
- **Testing on ROG (live)**: Deltarune round-trip (byte-identical), Split Fiction 94.7G→80.3G pack, verify, unpack, restore, launch via Steam, save loads; Cuphead save recovered from syncthing `.stversions` after it overwrote the live copy; 13 games launched 2026-08-09 all created prefixes (Eiyuden hit ENOSPC mid-build → prefix parked for rebuild).

---

## 0. Verified ground truth (live checks, 2026-08-09)

| Item | Verified value |
|---|---|
| SSH | `ssh rog` works as `jevonx`; hostname `bazzite-mobile`; x86_64 |
| Games drive | `/run/media/system/Games_Data` — ext4 `/dev/nvme0n1p1`, 3.5T, 2.3T used, **1.1T free** |
| `/var/home` | 198G, **3.8G free (99% used)** — treated as write-protected for archives/binary |
| Tools on ROG | zstd 1.5.7, GNU tar 1.35, xz, python3 3.14, `steam` (v1), `flatpak`, ludusavi 0.31.0 (flatpak) |
| **Missing on ROG** | go, pv. (verified `which` all negative) |
| Hardware | nproc=16, RAM 7.4Gi total (~4.4Gi available) |
| shortcuts.vdf | `/var/home/jevonx/.local/share/Steam/userdata/201689394/config/shortcuts.vdf`, 28299 bytes, **binary VDF**, **63 entries** |
| VDF parse | python vdf 3.4 `binary_loads()` works; keys are **capitalized** `AppName`, `Exe`, `StartDir`; `appid` is signed int32 |
| Compatdata | 69 dirs: 38 real-Steam (short ids), 31 non-Steam (uint32) |
| Shortcuts vs compatdata | **28** shortcuts have prefixes, **35** do not |
| Split Fiction (TEST) | appid **2508814786**, dir `Games_Data/Split Fictionx/` **89G**, exe `drive_c/Games/Split Fiction/Split/Binaries/Win64/SplitFiction.exe` |
| Split Fiction prefix | `compatdata/2508814786/` **371M** (recon said 308M — drift ok), Proton 11.0-100 |
| ⚠️ Split Fiction save symlink | **Currently ABSENT** from prefix `.../AppData/Local/` — recon claimed a live symlink; verified NOT present. Saves live at `GameSaves/appdata-local/SplitFiction/` (SaveData.Split, Settings.Split, LocalSettings.Split, Saved/, StartupInfo.Split). Design must NOT assume symlink exists. |
| `~/.local/bin` | **root-owned, NOT writable** by jevonx ⚠️ (recon proposed symlink there — won't work). `~/.local/share` IS writable. |
| GameSaves | `~/GameSaves/` — subdirs `appdata-local/locallow/roaming`, `saved-games`, `my-games`, `steam-userdata`, `cache-backups`, `ludusavi-backup`, `smart-sync` |
| link-saves.sh | Golden table: Cuphead 2986239671, FEZ 4155978628, JediSurvivor 4151545067, RuneFactory 3794432564, Witcher3 4076669000 (2 entries), SplitFiction 2508814786, Disgaea7 2152676719, Dispatch 2773086359. Cuphead: LocalLow must NOT be symlinked |
| auto-linker | `link-game-saves.path` (PathChanged=compatdata) → `link-game-saves.service` → `GameSaves/link-game-saves.sh` **active**. ⚠️ auto-linker will fight unpacker — see §10 |
| smart-save-checker | timer every 10min active, **service currently FAILED (exit 1)** ⚠️ — pre-existing breakage, unrelated |
| NSwitch | `Games_Data/NSwitch/`, **125 ROMs, 664G** (.xci/.nsp/.nsz), no nested dirs |
| Citron saves | `~/.local/share/citron/nand/user/save/0000000000000000/E06398B8B15F1E9E9F71D466CD63878B/<titleid>/`, **16M, 7 title dirs** |
| ⚠️ ROM title IDs | Only **1 of 125** ROM filenames contains `[0100XXXXXXXXXXXX]` — most don't. Title-ID matching from filename is NOT viable; see §5 |
| ludusavi find | `flatpak run com.github.mtkennerly.ludusavi find --api "Split Fiction"` → works, score 1.0; `~/ludusavi-backup/Split Fiction/mapping.yaml` has `C:/Users/Jevon/AppData/Local/SplitFiction/*` |
| Grid icons | `grid/2508814786_icon.png` etc confirm appids |
| ⚠️ FEZ | **Install dir missing** from Games_Data (exe path dangling). Catalog must handle missing dirs |
| ⚠️ Octopath Traveler 0 | shortcut 2224065267 points at `OCTOPATH TRAVELER 0/citron-cmd.exe` — a **Citron (Switch emu) launcher**, dir exists on drive. This is a Switch game launched via Steam → NSwitch/emu exclusion logic must handle |
| Dev machine | Go **1.26.4** at `/home/linuxbrew/.linuxbrew/bin/go` (linux/amd64) |
| Existing repo | `/home/jevonx/Projects/rog` — docs only so far; module dir to be created |

---

## 1. Architecture

### 1.1 Module layout

```
/home/jevonx/Projects/rog/gamepack/          # NEW Go module (built on dev machine)
├── go.mod                                  # module gamepack; go 1.26
├── cmd/gamepack/main.go                    # entrypoint: parse flags, start TUI
├── internal/
│   ├── catalog/
│   │   ├── vdf.go                          # parse shortcuts.vdf (andygrunwald/vdf)
│   │   ├── resolve.go                      # exe → gameDir mapping + special cases
│   │   ├── exclude.go                      # tool/emu/NSwitch exclusion rules
│   │   └── index.go                        # read/write Games_Data/Archive/.index.json
│   ├── compress/
│   │   └── zstd.go                         # level/flags selection, tar pipeline
│   ├── saves/
│   │   ├── golden.go                       # embedded link-saves.sh table (appid→paths)
│   │   ├── discover.go                     # union: golden + live symlinks + ludusavi find
│   │   ├── bundle.go                       # snapshot save files into archive
│   │   ├── restore.go                      # restore files into GameSaves (skip-newer)
│   │   └── link.go                         # recreate prefix symlinks (honoring caveats)
│   ├── pack/
│   │   ├── archive.go                      # tar|zstd streaming, progress via size poll
│   │   └── manifest.go                     # manifest.json writer/reader
│   ├── unpack/
│   │   ├── extract.go                      # streaming extract, resume, verify sha
│   │   └── preflight.go                    # disk space + running-game checks
│   ├── nswitch/
│   │   ├── catalog.go                      # scan NSwitch/ ROMs
│   │   ├── match.go                        # ROM → Citron titleid (curated map + prompt)
│   │   └── bundle.go                       # pack/unpack Citron saves
│   ├── proton/
│   │   ├── launch.go                       # steam://rungameid, prefix build polling
│   │   └── running.go                      # pfx.lock flock poll + exe fallback
│   ├── tui/
│   │   ├── main.go                         # screen router
│   │   ├── menumodel.go                    # main menu
│   │   ├── checklist.go                    # checkbox game list + confirm
│   │   ├── progress.go                     # per-game progress bar
│   │   ├── pause.go                        # interactive "wait for user" prompt
│   │   └── summary.go                      # final report
│   └── state/
│       └── journal.go                      # idempotent run journal (rerun-safe)
├── Makefile                                # build + scp + install
└── README.md                               # usage on ROG
```

### 1.2 Runtime layout on ROG

```
/run/media/system/Games_Data/
├── Tools/gamepack/
│   ├── gamepack              # static linux/amd64 binary (755)
│   └── install.sh            # (optional) creates ~/.local/share/bin/gamepack symlink
├── Archive/                  # NEW — everything lands here
│   ├── .index.json           # catalog + archive index (tool-owned)
│   ├── .journal.json         # run state for idempotent reruns
│   ├── PC/<slug>/            # per-game archive bundle
│   │   ├── game.tar.zst      # game dir
│   │   ├── saves.tar.zst     # save snapshot (if any)
│   │   └── manifest.json
│   └── NSwitch/<slug>/
│       ├── rom.tar.zst
│       ├── saves.tar.zst     # matched Citron saves (if any)
│       └── manifest.json
```

**Why binary on Games_Data, not ~/.local/bin:** ⚠️ VERIFIED-DIFF — `~/.local/bin` is root-owned on Bazzite and NOT writable by jevonx. `~/.local/share` IS writable, so the shim goes there. Binary itself lives on Games_Data (home is 99% full; binary ~10–15MB static).

### 1.3 Runtime discovery (how the tool finds everything)

All paths discovered at startup from a small `Config` struct, no hardcoding beyond defaults, overridable via flags/env:

| Setting | Default (verified) |
|---|---|
| `ShortcutsVdf` | `/var/home/jevonx/.local/share/Steam/userdata/201689394/config/shortcuts.vdf` |
| `CompatData` | `/var/home/jevonx/.local/share/Steam/steamapps/compatdata` |
| `Steam` | `/usr/bin/steam` |
| `GameSaves` | `/var/home/jevonx/GameSaves` |
| `LudusaviBackup` | `/var/home/jevonx/ludusavi-backup` (fallback save discovery) |
| `NSwitchDir` | `/run/media/system/Games_Data/NSwitch` |
| `CitronSaves` | `/var/home/jevonx/.local/share/citron/nand/user/save/0000000000000000/E06398B8B15F1E9E9F71D466CD63878B` |
| `ArchiveRoot` | `/run/media/system/Games_Data/Archive` |
| `ZstdBin` | `/usr/bin/zstd` (v1.5.7 verified; needed for `--long`/`--ultra` flags — Go's `klauspost/compress` is a fallback but we shell out to system zstd for exact flag support and streaming simplicity) |

Discovery at runtime:
1. Parse shortcuts.vdf → catalog entries `{name, appid(uint32), exe, startDir}`.
2. For each entry, resolve `gameDir` (see §2.3).
3. For each entry, check prefix presence (`compatdata/<appid>/pfx` exists) → `hasPrefix`.
4. Save all into `.index.json` so the TUI never re-parses VDF on every keystroke; re-scan on `--refresh`.

---

## 2. Game catalog

### 2.1 Parse shortcuts.vdf (binary)

- Use Go lib `github.com/andygrunwald/vdf`. API: `vdf.NewParser(strings.NewReader(...)).Parse()` for **text** VDF — but shortcuts.vdf is **binary** (verified: starts with `\x00shortcuts\x00`). andygrunwald/vdf supports binary via `vdf.NewBinaryParser`. ⚠️ VERIFIED — python's `binary_loads` succeeded; plan Go side to use the binary parser path and confirm during build phase (open question Q3).
- Keys are capitalized: `AppName`, `Exe`, `StartDir` (⚠️ not lowercase as recon implied).
- `appid` is signed int32 → **real appid = uint32(signed & 0xFFFFFFFF)**. Verify against grid icons (e.g. `grid/2508814786_icon.png`).

### 2.2 Exclusions (verified catalog)

Exclude from the PC pack list (still listed in a "Tools/Emulators/Switch" TUI section or hidden):

| Exclude | Reason | Verified |
|---|---|---|
| 2379889317 Linux-Yuzu | Switch emulator AppImage | exe is AppImage in Games_Data |
| 3836569258 Eden-Linux | Switch emulator AppImage | AppImage |
| 3640729765 Switch Citron | Switch emulator AppImage | AppImage |
| 2587493377 Ascendara.AppImage | app installer tool, `~/.local/bin` | exe on /var/home |
| 2224065267 "Octopath Traveler 0" | Citron launcher wrapper (Switch game) → route to NSwitch section | exe = `OCTOPATH TRAVELER 0/citron-cmd.exe` |
| `NSwitch/`, `Tools/`, `umu/`, `external-sources/`, `imgs/`, `qbit/`, `Roms/`, `ProdKeys.NET-v22.1.0/`, `RetroDECK/`, `SteamLibrary/`, `lost+found/`, `Add-Non-Steam-Games-Linux/` | non-game dirs on drive | verified present |

Also exclude by rule: exe is an `.AppImage`, or exe resolves to a path whose basename matches `*(yuzu|citron|eden|emu|retrodeck)*`.

### 2.3 exe → gameDir resolution

- Default: `gameDir = filepath.Dir(exe)` (the dir containing the game exe).
- **Special case — container/prefix-style installs** (Split Fiction): if the exe's ancestor chain contains a dir that looks like a wine prefix root (has `drive_c/`, `system.reg`, and `config_info` at its top), the **gameDir = that prefix root**. For Split Fiction this yields `Games_Data/Split Fictionx` (89G) instead of the shallow `.../Split/Binaries/Win64` dir. Detect by walking up from exe, max 6 levels, checking for `drive_c` + `system.reg`. ✓ verified layout.
- Archive the whole `gameDir`; when packing a prefix-root dir, **exclude `shadercache/`, `tracked_files`, `*.lock`, `dosdevices/z:` symlink, `gstreamer-1.0/`** (regenerable/derived; see §7 edge cases).

### 2.4 Prefix presence & the 35 no-prefix shortcuts

- 28 shortcuts have `compatdata/<appid>/pfx`; 35 don't (verified list in §0).
- For the 35: still pack the game dir (fully functional). Save discovery will find little/none; TUI marks `no-prefix` badge. On unpack, the prefix is (re)created via the Steam launch flow (§4.4).

### 2.5 Cache to `.index.json`

Schema (same shape as `Archive` entries in §1.2):

```json
{
  "toolVersion": "0.1.0",
  "generatedAt": "2026-08-09T15:00:00Z",
  "pcGames": [
    {
      "name": "Split Fiction",
      "appid": 2508814786,
      "exe": "/run/media/system/Games_Data/Split Fictionx/drive_c/Games/Split Fiction/Split/Binaries/Win64/SplitFiction.exe",
      "startDir": "...",
      "gameDir": "/run/media/system/Games_Data/Split Fictionx",
      "gameDirType": "prefix-container",
      "sizeBytes": 95563038900,
      "hasPrefix": true,
      "isArchived": false,
      "archivedAt": null
    }
  ],
  "nswitch": [
    { "name": "Animal Crossing New Horizons", "file": "Animal Crossing New Horizons.xci", "sizeBytes": 0, "isArchived": false }
  ]
}
```

---

## 3. Compression

### 3.1 Decision: zstd level 19 + `--long` (default), `--ultra -22` optional

- **Default: `zstd -T1 -19 --long=27 --no-progress`** — level 19 with `--long` gives near-max ratio in this RAM envelope. Verified zstd 1.5.7 supports `--long` (default window 27).
- **`-22 --ultra`**: optional slowest mode for games where size matters more than time (RAM fine at `--long=27`: ~1.2GB peak, fits 4.4G available). nproc=16 but **do not run parallel zstd** — RAM bound, and `-T16` on level 19 would blow the 7.4G budget. Default `-T1`.
- **Why not xz**: xz gives ~5–15% more on game binaries but is 10–50× slower. Games are mostly already-compressed assets (audio/video/textures), so the *effective* ratio of the whole tree is typically only ~10–25%. zstd -19 is the right speed/ratio point here. (`xz -9e` is offered as an optional per-game choice later, YAGNI for v0.1.)
- **Realistic ratio expectations** for repack-style game data: overall tree ~10–25% (uncompressed loose files: .exe/.dll/shaders compress well; .ogg/.png/.pak do not). Don't promise more in the UI.
- **Streaming**: `tar --zstd --acls --xattrs --sparse -cf <out> -C <parent> <name>` (verified GNU tar 1.35 has `--zstd`). `--sparse` keeps sparse files sparse; `--xattrs`/`--acls` preserve Proton-era metadata. **No double-copy**: tar reads → zstd writes directly to target, peak disk = archive size, not 2×.

### 3.2 Per-game artifact layout + manifest schema

```
Archive/PC/split-fiction/game.tar.zst     # gameDir content (single archive)
Archive/PC/split-fiction/saves.tar.zst    # save snapshot (if saves found)
Archive/PC/split-fiction/manifest.json
```

`manifest.json` schema:

```json
{
  "name": "Split Fiction",
  "appid": 2508814786,
  "exe": "/run/media/system/Games_Data/Split Fictionx/drive_c/Games/Split Fiction/Split/Binaries/Win64/SplitFiction.exe",
  "startDir": "/run/media/system/Games_Data/Split Fictionx/drive_c/Games/Split Fiction",
  "gameDir": "/run/media/system/Games_Data/Split Fictionx",
  "gameDirType": "prefix-container",
  "origSize": 95563038900,
  "archivedSize": 72500000000,
  "ratio": 0.76,
  "sha256": "…game.tar.zst sha256…",
  "sha256Saves": "…saves.tar.zst sha256…",
  "zstd": { "version": "1.5.7", "level": 19, "longWindowLog": 27, "threads": 1 },
  "savesCount": 4,
  "saveEntries": [
    { "gamesavesRel": "appdata-local/SplitFiction", "protonRel": "drive_c/users/steamuser/AppData/Local/SplitFiction", "ludusaviPath": "C:/Users/Jevon/AppData/Local/SplitFiction", "files": ["SaveData.Split","Settings.Split","LocalSettings.Split"] }
  ],
  "packDate": "2026-08-09T15:00:00Z",
  "toolVersion": "0.1.0"
}
```

### 3.3 Progress reporting

- No `pv` on ROG (verified missing). **Poll `stat` size of the `.tar.zst` target** every 500ms vs `origSize` → percent + MB/s + ETA. tar's `--checkpoint` is a fallback if stderr plumbing is easier, but size-poll needs no subprocess parsing.

---

## 4. Save handling (the hard part)

### 4.1 Discovery (pack) — union of three sources

At pack time, per game, compute save entries as a union of:

1. **Golden table** (embedded, mirrors `GameSaves/link-saves.sh` — verified contents):
   ```
   Cuphead       2986239671  appdata-roaming/Cuphead → AppData/Roaming/Cuphead
   FEZ           4155978628  appdata-roaming/FEZ     → AppData/Roaming/FEZ
   Jedi Survivor 4151545067  saved-games/Respawn/JediSurvivor → "Saved Games/Respawn/JediSurvivor"
   Rune Factory  3794432564  appdata-local/Rune Factory Guardians of Azuma → AppData/Local/...
   Witcher 3     4076669000  saved-games/CD Projekt Red → "Saved Games/CD Projekt Red"
                              + appdata-roaming/CD Projekt Red → AppData/Roaming/...
   Split Fiction 2508814786  appdata-local/SplitFiction → AppData/Local/SplitFiction
   Disgaea 7     2152676719  appdata-roaming/Nippon Ichi Software, Inc/Disgaea 7 → AppData/Roaming/...
   Dispatch      2773086359  appdata-local/Dispatch → AppData/Local/Dispatch
   ```
   ⚠️ Split Fiction's symlink is currently **missing** from the prefix — the golden table is the reliable source; don't depend on live symlink presence.
2. **Live symlink scan**: walk `compatdata/<appid>/pfx/drive_c/users/steamuser/**` for symlinks whose target is under `/var/home/jevonx/GameSaves/`; record the `GameSaves`-relative source and the proton-relative target. This catches auto-linker additions not in the golden table.
3. **ludusavi fallback** (only if 1+2 found nothing): `flatpak run com.github.mtkennerly.ludusavi find --api "<name>"`, then read `~/ludusavi-backup/<Game>/mapping.yaml` for logical Windows paths. ⚠️ NOTE: ludusavi uses `Users/Jevon` while current prefixes use `users/steamuser` — map `Jevon→steamuser` when writing restore targets.

**Bundle**: snapshot real files (following symlinks, **copying content, not symlink targets-as-links**) into `saves.tar.zst` under `gamesaves/<relative>` layout, and record the mapping in manifest `saveEntries`. Empty-dir games → `savesCount: 0`, TUI shows "no save mapping — game-only archive" **warning**, still pack the game (saves may live inside the install dir and are already captured in `game.tar.zst`).

### 4.2 Save file layout inside saves.tar.zst

```
saves.tar.zst
└── gamesaves/
    └── appdata-local/SplitFiction/
        ├── SaveData.Split
        ├── Settings.Split
        ├── LocalSettings.Split
        └── Saved/…
```

`manifest.json.saveEntries[*].gamesavesRel` is the anchor; restore re-derives GameSaves path from it (username-agnostic).

### 4.3 Restore (unpack) — save files + symlinks

1. Extract `saves.tar.zst` → into `GameSaves/<gamesavesRel>` (real files, not symlinks). **Skip-newer**: if a GameSaves file exists with newer mtime (or different content hash) than the archived one, keep the newer one and log.
2. Recreate prefix symlinks mirroring link-saves.sh, honoring the **Cuphead caveat** (LocalLow must NOT be symlinked — causes "SAVE FAILED"): the golden table is the whitelist; auto-linker may re-add other links — coordinate in §10.
3. ⚠️ If prefix doesn't exist yet → run the interactive flow first (§4.4), then create symlinks.

### 4.4 Interactive "wait for user to launch/close game" flow

Used when unpacking a game whose prefix is missing or incomplete. This is the exact pause TUI (see §6.4 diagram).

**Step-by-step:**

1. **Pause prompt (TUI)**: shows
   ```
   ╭─ SAVE ATTACH — Split Fiction ─────────────────────────────╮
   │  The game must create its Proton prefix before saves can  │
   │  be wired in.                                             │
   │                                                            │
   │  1) In Steam, LAUNCH "Split Fiction" (I will start it)     │
   │  2) Let it reach the main menu, then CLOSE it.             │
   │                                                            │
   │  Waiting for you to close the game… (Ctrl-C to skip)       │
   ╰────────────────────────────────────────────────────────────╯
   ```
2. **Tool launches Steam**: `steam steam://rungameid/2508814786` (verified mechanism: forwarded to running gamescope Steam instance via `~/.steam/steam.pipe`; fd-based, works from SSH session).
3. **Poll prefix build**: watch `compatdata/2508814786/pfx/creation_sync_guard` to appear, then `config_info` written → prefix built.
4. **Poll game exit**: primary signal = **`flock` on `compatdata/<appid>/pfx.lock`**. Probe: `flock -n <pfx.lock> -c true` → exit 0 = unlocked (game closed); exit 1 = locked (running). Also poll for the game exe process as fallback (`pgrep -f SplitFiction.exe`). Poll every 2s. ⚠️ pfx.lock flock reliability is an open question (Q2) — verify on a real launch during build phase.
5. On close → proceed to §4.3 restore + symlinks. On Ctrl-C → skip save attach, record `savesAttached: false` in journal, continue.

### 4.5 Prefix: archive vs rebuild decision

- **v0.1 default: do NOT archive the compatdata prefix.** Prefix is regenerable (Proton rebuilds `drive_c` on first launch), 308–371M each, contains `shadercache/` noise. Saves are the durable state and they live in GameSaves anyway.
- Unpack always: extract game dir → ensure prefix via §4.4 → restore saves → link. The `tracked_files` and `config_info` are rebuilt by Proton.
- **Open question Q5**: some games bake registry/config into prefix (`user.reg`, `system.reg`). If a game misbehaves post-unpack, add a `--with-prefix` flag to also archive `compatdata/<appid>` (minus shadercache) as `prefix.tar.zst`. Defer to post-MVP.

---

## 5. NSwitch separation

### 5.1 Layout

```
Archive/NSwitch/<slug>/
├── rom.tar.zst       # single ROM, tar | zstd
├── saves.tar.zst     # matched Citron saves (only if match found)
└── manifest.json
```

### 5.2 Compression policy for ROMs

- .nsp/.nsz/.xci are **already-compressed containers**. zstd on top yields ~0%. **Use `zstd -1` (fast) or even `-0`/store** — the value is single-file packaging + integrity, not ratio. Recommend `-3` as a cheap middle ground (few % gain, fast). Document in manifest (`zstd.level`).
- Keeps unpack instant and CPU-free.

### 5.3 Save matching (⚠️ the real constraint)

- Only 1/125 ROM filenames carries a `[0100XXXXXXXXXXXX]` title ID. **Filename-based title-ID matching is not viable**.
- Citron save dirs: `.../E06398B8B15F1E9E9F71D466CD63878B/<7 titleids>/` (16M total). Verified title IDs:
  `010015100B514000, 01004B100AF18000, 0100646009FBE000, 01007300020FA000, 010074F013262000, 01007960049A0000, 0100DCA0064A6000`
- Strategy:
  1. **Curated match table** (Go map, ROM name → titleid) seeded during build phase by inspecting the 7 save dirs' games (run each once, or cross-check `citron` logs/game titles). Populate ~7 entries. These are the only ROMs with saves today.
  2. **Name matcher**: fuzzy-match ROM filename against curated table + optionally query a public titleid DB later (out of scope).
  3. **TUI prompt fallback**: if a ROM has no entry, offer "attach Citron saves?" with a pick list of the 7 titleids; defaults to no.
- Save bundle = the matched `<titleid>/` dir, packed under `saves/<titleid>/`.

---

## 6. TUI

### 6.1 Screens

```
main menu ──► PC Pack ──► [game list w/ checkboxes] ──► confirm ──► per-game progress ──► summary
   │              │
   │              └──► (unpack: same flow, reversed)
   ├──► NSwitch Pack / Unpack (same, NSwitch section)
   └──► Quit
```

### 6.2 Text diagram of TUI flow

```
┌───────────────────────────────────────────────────────┐
│  gamepack v0.1.0  —  ROG Game Archiver                │
│                                                       │
│  [1] Pack games (PC)      [3] Pack NSwitch ROMs       │
│  [2] Unpack games (PC)    [4] Unpack NSwitch ROMs     │
│  [5] Quit                                             │
└───────────────────────────────────────────────────────┘
        │ select 1
        ▼
┌───────────────────────────────────────────────────────┐
│  Pack — PC games (63 cataloged, 58 packable)          │
│  Filter: [all ▾]   Search: [______]                   │
│                                                       │
│  ☐  Alabaster Dawn            12.4G        [ ]        │
│  ☐  Split Fiction             89.0G   [ARCHIVED 68G]  │
│  ☑  FEZ                       (dir missing!)   ⚠       │
│  …                                                       │
│  [x] Confirm pack (4 selected, 14.2G → ~11.2G est.)   │
└───────────────────────────────────────────────────────┘
        │ confirm
        ▼
┌───────────────────────────────────────────────────────┐
│  Packing Split Fiction (1/4)                          │
│  ███████████████████████░░░░░░░░░░░░░░░  62%   (58s)  │
│  42.1G / 68.0G   →  Archive/PC/split-fiction/          │
│  saves: 4 files (appdata-local/SplitFiction)  ✓        │
│  sha256: b3a2c1…                                        │
└───────────────────────────────────────────────────────┘
        │ on unpack w/ missing prefix
        ▼
┌───────────────────────────────────────────────────────┐
│  ╔ SAVE ATTACH — Split Fiction ═══════════════════╗   │
│  ║ 1) I will run: steam steam://rungameid/2508814786  ║
│  ║ 2) Let it reach the main menu                      ║
│  ║ 3) CLOSE the game — I detect exit via pfx.lock     ║
│  ║    (polling every 2s)                              ║
│  ║                                                    ║
│  ║  Waiting…  [prefix built ✓]  [game closed ✓]       ║
│  ║  Ctrl-C to skip save attach                        ║
│  ╚════════════════════════════════════════════════╝   │
└───────────────────────────────────────────────────────┘
```

### 6.3 Component models

| Model | Purpose |
|---|---|
| `menumodel` | main menu, arrow keys, 1–5 select |
| `checklist` | checkbox list w/ search + filter; shows size, ARCHIVED badge, ⚠ missing-dir, no-prefix badge; Enter confirms |
| `progress` | single-game bar (polled size), plus a small multi-game list showing done/active/queued |
| `pause` | interactive save-attach prompt w/ live status lines (prefix-built, game-closed) |
| `summary` | table: name, orig→archived, ratio, saves, sha, status; Esc quits |

### 6.4 Key handling

- Up/Down/j/k, Space toggle, `/` search, Tab switch filter, Enter confirm, Esc back, Ctrl-C abort-current (logged, journal-safe).
- Multi-select packs run **sequentially** (RAM/disk bound; no benefit parallelizing zstd -19).

---

## 7. Edge cases

| # | Case | Behavior |
|---|---|---|
| E1 | Archive already exists for game | TUI prompt: `[1] Re-pack (overwrite) [2] Keep existing [3] Skip`; journal + manifest dated |
| E2 | Game dir missing at unpack | Skip w/ error, mark `failed: missing-dir` in journal, continue others |
| E3 | Partial/corrupt extract | Verify sha256 of `game.tar.zst` first; extract to `game.tar.zst.d` staging then atomic rename; resume = re-extract (idempotent) |
| E4 | Insufficient disk preflight | Before unpack: `df -B1 Archive` vs manifest `origSize` + 2% headroom; abort that game w/ clear message |
| E5 | Steam not running | Detect via `pgrep steam`/steam.pipe; skip auto-launch, print manual `steam steam://rungameid/<id>` instructions, keep waiting loop |
| E6 | SELinux | Enforcing, but jevonx already writes Games_Data (verified probe). No extra policy needed |
| E7 | Pack while game running | Refuse pack if `pfx.lock` is flock-held (running) OR game exe process active; prompt to close first |
| E8 | Flaky SSH | Every step is idempotent + journaled (`.journal.json`); rerun resumes; archive write is streaming so partials are re-done cleanly |
| E9 | Missing-dir games (FEZ) | Catalog marks `dirMissing`; excluded from pack selection unless `--force` |
| E10 | Container/prefix game dirs | gameDir = prefix-root; exclude `shadercache/`, `dosdevices/z:`, `tracked_files`, locks from archive |
| E11 | Save newer than archive | Skip-newer on restore; never clobber a fresher GameSaves file |
| E12 | NSwitch ROM with no save match | Pack ROM only; `savesCount: 0`; noted in manifest |
| E13 | /var/home pressure | Never write archives or binary to /var/home (only shim symlink in ~/.local/share, which is ~bytes) |

---

## 8. Deployment

### 8.1 Dev machine build + scp + install

```bash
# on dev machine (/home/jevonx/Projects/rog/gamepack)
cd /home/jevonx/Projects/rog/gamepack
export PATH="$PATH:/home/linuxbrew/.linuxbrew/bin"   # go 1.26.4 lives here
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" \
    -o gamepack ./cmd/gamepack

# push binary
scp gamepack rog:/run/media/system/Games_Data/Tools/gamepack/gamepack

# install shim (on ROG) — ~/.local/bin is NOT writable (root-owned), so:
ssh rog 'mkdir -p ~/.local/share/bin && \
  ln -sf /run/media/system/Games_Data/Tools/gamepack/gamepack ~/.local/share/bin/gamepack && \
  ls -la ~/.local/share/bin/gamepack'
```

### 8.2 Runtime requirements on ROG

- **None beyond stock**: system `zstd` 1.5.7 + `tar` 1.35 (both verified present), `steam`, `flatpak` (ludusavi discovery), and `pgrep`/`flock` (coreutils). No Go runtime, no daemon, no systemd units.
- **systemd timer is out of scope** (per requirements). The existing `link-game-saves.path/.service` stays — see §10 coordination.

### 8.3 Makefile target

```make
build:    # CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gamepack ./cmd/gamepack
deploy:   # scp gamepack rog:... && ssh rog ln -sf ...
test:     # go test ./...  (unit tests on catalog/compress/saves logic)
```

---

## 9. Testing plan (Split Fiction + FEZ/small)

### 9.1 Dry-run catalog resolve

- `gamepack --refresh --dry-run` → prints 63 catalog entries, resolved gameDir (Split Fiction → `Split Fictionx` container), 58 packable + 5 excluded (3 emus + Ascendara + Octopath-0→NSwitch), 35 no-prefix flagged. **Success: Split Fiction gameDir = prefix-container; excluded set exactly matches §2.2; FEZ marked dirMissing.**

### 9.2 Small-game quick test (fast iteration) — pick a ~200–600MB title

- Candidates verified on drive: Deltarune 582M, Spiritfall 1.5G, Ogu and the Secret Forest 1.8G. FEZ is **missing** from drive (can't use as small test despite recon).
- Steps: pack → verify manifest + sha256 → move dir away → unpack → launch → save flow.
- **Success criteria**: archive ratio >1.0 (i.e. file smaller), sha matches, unpacked game launches via Steam, saves restored + symlinked.

### 9.3 Split Fiction end-to-end (89G, real run)

| Step | Command (TUI) | Success criteria |
|---|---|---|
| 1 | Pack Split Fiction | Progress bar to 100%; manifest written; `game.tar.zst` sha256 computed; `saves.tar.zst` contains 4 files (SaveData/Settings/LocalSettings/Saved); `savesCount=4` |
| 2 | Verify manifest | sha256 of on-disk archive == manifest.sha256; origSize == du of gameDir |
| 3 | Move game dir away | `mv "Games_Data/Split Fictionx" /tmp/…` (on 1.1T free; or rename) |
| 4 | Unpack game | `gameDir` restored byte-identical (spot-check sha of a big exe); staging→atomic rename |
| 5 | Launch via Steam | `steam steam://rungameid/2508814786` — **verify it actually starts the shortcut** (open question Q1; must be confirmed in build phase). Game reaches main menu |
| 6 | Close game → save attach | Tool detects prefix build + game exit via pfx.lock; restores saves into GameSaves (skip-newer no-op if same); creates `AppData/Local/SplitFiction` symlink; **game loads save** (menu shows "Continue") |
| 7 | Re-run idempotency | Re-run unpack → journal shows already-done, no duplicate work, no errors |

### 9.4 NSwitch quick test

- Pack 1 small ROM (e.g. `Battle Chef Brigade Deluxe.nsz`) → verify `rom.tar.zst`, ratio ~1.0, unpack, matches titleid flow (probably no save → `savesCount:0`). Then test save-attach against the one known Citron title if identifiable.

### 9.5 Unit test surface

- VDF parse (golden sample), signed→uint32, exe→gameDir special-case, exclusion rules, golden-table load, manifest round-trip, journal idempotency, disk preflight math. (Mocked; no ROG needed.)

---

## 10. Open questions / risks + recommendations

| # | Question | Risk | Recommendation |
|---|---|---|---|
| Q1 | Does `steam steam://rungameid/<uint32>` actually launch a **non-Steam shortcut** from SSH/gamescope? | HIGH — the whole save-attach flow depends on it | **Test in build phase, not planning.** Recon says fd-5 steam.pipe forwarding works for `rungameid`; verify with a real shortcut. Fallback: `steam -applaunch <appid>`; last resort: instruct user to launch manually (pause flow already supports this). |
| Q2 | Is `pfx.lock` flock held reliably while a game runs? | MEDIUM | Probe on a live run during build phase. Fallback = exe-process poll (`pgrep -f <exe>`). If neither reliable, poll `~/.config/...` or game window; or simply require user to press a key after closing (manual confirm in pause UI). |
| Q3 | Does `andygrunwald/vdf` binary parser handle this file? | LOW | Python `binary_loads` worked (verified). If the Go lib's binary support is weak, fallback: ship a tiny pure-Go binary-VDF reader (~150 lines, spec well known) — keep in `internal/catalog/vdf.go`. |
| Q4 | 35 no-prefix shortcuts — game-dir resolution edge cases (games that install into nested dirs, DLC) | LOW | Default = dir of exe; container rule catches prefix roots; manifest records `gameDirType` for audit. |
| Q5 | Should compatdata prefix be archived too? | MEDIUM — some games store registry/config in prefix | v0.1: no (regenerable). Add `--with-prefix` flag post-MVP if a game misbehaves. |
| Q6 | ⚠️ **Auto-linker conflict**: `link-game-saves.path` fires on compatdata changes and runs `link-game-saves.sh` — during unpack it may race us creating symlinks. | MEDIUM | Coordination: our unpack writes saves to GameSaves **first**, then symlinks — auto-linker's output converges to the same end-state (link→GameSaves). On pack, we read-only. Consider temporarily masking the service during unpack runs, or leave it (idempotent, same target). **Decision in build phase.** |
| Q7 | ⚠️ `smart-save-checker` is currently FAILING (exit 1) | LOW (pre-existing) | Out of scope; note it; don't trigger it. Our restore respects skip-newer regardless. |
| Q8 | ludusavi path divergence (`Users/Jevon` vs `users/steamuser`) | LOW | Map at restore; manifest stores both. |
| Q9 | NSwitch save matching is weak (7 titleids, no filename IDs) | MEDIUM | v0.1 = curated table + TUI prompt fallback. Acceptable because only 16M of saves exist. |
| Q10 | ~/.local/bin not writable | LOW | Shim in `~/.local/share/bin` (verified writable) or invoke by absolute path. |
| Q11 | `OCTOPATH TRAVELER 0` (Citron launcher shortcut) | LOW | Route to NSwitch section, exclude from PC pack. Note: its own `compatdata/2224065267` prefix exists — pack its Citron saves under NSwitch tree if it's the primary Citron-launcher path. |
| Q12 | 38 real-Steam games have prefixes (short appids) | LOW | Out of scope (Steam-managed installs). Catalog only non-Steam shortcuts. Document. |

---

## 11. Module / function breakdown (mapping plan→code)

| Area | Files | Key funcs | Notes |
|---|---|---|---|
| Catalog | `catalog/vdf.go` | `ParseShortcuts(path) []Entry`, `ToUint32(signed)` | binary VDF parser |
| | `catalog/resolve.go` | `ResolveGameDir(entry) (dir, type, err)` | prefix-container detection |
| | `catalog/exclude.go` | `ShouldExcludePC(entry) (bool, reason)` | emu/tool/AppImage rules |
| | `catalog/index.go` | `LoadIndex`, `SaveIndex` | .index.json |
| Compress | `compress/zstd.go` | `BuildCmd(level, long, threads)`, `ArchiveTar(dir)→stream` | shells to system zstd |
| Saves | `saves/golden.go` | `LoadGolden() []Mapping` | embedded table |
| | `saves/discover.go` | `Discover(appid, prefixDir) ([]SaveEntry, error)` | union of 3 sources |
| | `saves/bundle.go` | `Bundle(entries, outPath)` | follow symlinks, copy content |
| | `saves/restore.go` | `Restore(bundle, manifest)` | skip-newer |
| | `saves/link.go` | `CreateLinks(mappings, prefixDir)` | honors Cuphead caveat |
| Pack | `pack/archive.go` | `PackGame(entry, cfg) (PackResult, error)` | tar\|zstd, progress poll |
| | `pack/manifest.go` | `WriteManifest`, `ReadManifest` | schema §3.2 |
| Unpack | `unpack/extract.go` | `ExtractGame(manifest, dest)`, `VerifySHA256` | staging+atomic rename |
| | `unpack/preflight.go` | `CheckDisk(origSize, free)`, `GameRunning(appid)` | |
| NSwitch | `nswitch/catalog.go` | `ScanROMS(dir) []ROM` | |
| | `nswitch/match.go` | `MatchTitle(romName, titleids) (titleid, ok)` | curated + fuzzy |
| | `nswitch/bundle.go` | `PackROM`, `PackSaves`, `UnpackROM` | zstd -3 |
| Proton | `proton/launch.go` | `LaunchGame(appid)`, `WaitPrefixBuilt(appid)`, `WaitGameExit(appid)` | polling |
| | `proton/running.go` | `IsRunning(appid)`, `IsUnlocked(lockPath)` | flock probe |
| TUI | `tui/*.go` | models in §6.3 | bubbletea |
| State | `state/journal.go` | `Begin/Step/Complete/Fail`, `Resume()` | idempotent reruns |

---

## 12. Build phase order (next after plan approval)

1. Scaffold module, vendored deps (`bubbletea`, `andygrunwald/vdf`, `lipgloss` optional, `klauspost/compress` optional).
2. Unit tests for catalog (binary VDF parse + ToUint32 + exe→gameDir + exclusions) — RED/GREEN.
3. Verify `steam://rungameid` for a real shortcut + pfx.lock flock (Q1/Q2) — **these gates decide save-attach design**.
4. Compression pipeline + manifest + progress (test with Deltarune 582M).
5. Saves discovery/restore/link (test with Split Fiction).
6. TUI screens.
7. NSwitch section.
8. Split Fiction E2E (§9.3).
