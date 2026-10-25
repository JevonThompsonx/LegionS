# gamepack — Modernization Todo

> Scope: gamepack Go code only. For machine ops tasks (OS, Steam, drives) see TODO.md.

**Audit date:** 2026-08-24 · **Scope:** `/home/hermes/Projects/rog/gamepack` (Go module `gamepack`) + sibling `rog-save-sync` (README-only, no code).
**Method:** Full read of `internal/{cfg,catalog,compress,pack,unpack,saves,proton,nswitch,state,tui}`, `cmd/gamepack/main.go`, `Makefile`, `scripts/coverage-check.sh`, plus `go vet`, `go build`, `go test -cover`, `go list -deps`, repo git log.

**Verified baseline (good):**
- `go build` (CGO off, linux/amd64, trimpath) ✅ clean
- `go vet ./...` ✅ clean
- `go test ./internal/...` ✅ all pass
- Coverage gate (40% floor) ✅ passes on every gated package (catalog 65%, cfg 76%, compress 76%, nswitch 43%, pack 60%, saves 57%, tui 40%)

**Two correctness gaps found (see Security & Backup Integrity below):**
1. `internal/state/journal.go` is **dead code** — 0 non-test imports. The TUI help promises "state is journaled; safe to rerun" but journaling is never wired in.
2. On `unpack`, **only `game.tar.zst` is checksum-verified**. The recorded `SHA256Saves` / `SHA256Prefix` are never checked, so bit-rot in the saves/prefix archives is undetected.

---

## 🔒 Security

- [ ] **Guard `RemoveAll(gameDir)` against mis-resolution.** `--remove-source` and the TUI "remove folder" prompt call `os.RemoveAll` on `g.GameDir`, which is derived by `ResolveGameDir` (walks up to 8 ancestors looking for a prefix root). A mis-resolve (e.g. a plain game whose dir resolves outside `Games_Data`, or a bad `exe` path) would `RemoveAll` the wrong tree. **Add an assertion that `GameDir` is lexically under an allowed root (`ArchiveRoot`'s parent / `Games_Data`) before any destructive delete.**
- [ ] **Sanitize tar extraction paths (Zip-Slip style).** `compress.Extract` runs `tar -x` with no `--no-absolute-names` and no member-path validation. Self-produced archives are safe, but a foreign/corrupted archive's members could contain `../` or absolute paths and escape the target dir. Add `--no-absolute-names` and reject any member that escapes `dest`.
- [ ] **Don't extract `prefix.tar.zst` directly into `CompatData`.** `restorePrefix` extracts straight into `c.CompatData` with no staging/atomic rename (unlike the game dir, which stages then `moveContents`). A corrupt prefix archive leaves compatdata half-written. Stage to a temp dir under `CompatData` then atomic-rename, matching the game flow.
- [ ] **Parameterize the hardcoded Steam user path.** `proton.SteamRunning` reads `/var/home/jevonx/.steam/steam.pid` + `steam.pipe` literally, ignoring `cfg`. The tool is silently bound to one user. Add `SteamUser`/`SteamHome` to `cfg.Config` and derive the pid/pipe paths from it.

## 🛡️ Backup Integrity · Sync Safety · Conflict Resolution

- [ ] **Verify ALL three archive checksums on restore.** In `unpack.Unpack`, only `game.tar.zst`'s `m.SHA256` is compared. Add verification of `saves.tar.zst` (`m.SHA256Saves`) and `prefix.tar.zst` (`m.SHA256Prefix`) before extracting them; fail fast with a clear "archive corrupted" message instead of silently restoring bad data.
- [ ] **Wire up or remove the journal.** `internal/state/journal.go` (the `Journal`/`JournalStep` types) is completely unused. Either integrate it so Ctrl-C / reboot truly produces resumable, journaled state (the README reboot-resilience story currently rests only on "no manifest = incomplete", not on the journal), **or** delete the dead package and fix the TUI help text that claims journaling. Today the advertised safety does not exist.
- [ ] **Skip syncthing conflict files when bundling saves.** `saves.Bundle` snapshots `~/GameSaves` verbatim, including any `*.conflict` files syncthing may have left. Filter those out so a conflicted copy is never archived as the good save.
- [ ] **Atomic restore of multi-file saves.** `RestoreFile` is skip-newer per file, but a restore of an N-file save isn't transactional — if it aborts mid-way you get a partial save (some files old, some new). Consider staging the whole save dir then swapping, consistent with the game-dir flow.
- [ ] **Conflict awareness vs live GameSaves.** There is no check that `~/GameSaves` is quiescent (syncthing idle) at pack time; a snapshot can be taken mid-sync. Optionally check for an active syncthing / absence of `.sync` tmp files, or document the race.

## ⚡ Efficiency

- [ ] **Stop re-walking the whole catalog on every single-game op.** `catalog.BuildCatalog` does a full recursive `dirSize` + `HasPrefix` (which reads `libraryfolders.vdf` and stats `pfx` per game) for all 53 shortcuts — and it's called by the TUI on *every* pack and by `runCLI` even for one `--pack`. For an 89 GB game this filesystem walk is pure overhead before compression even starts. Cache the index and only (re)stat the targeted game, or make `BuildCatalog` lazy/per-game.
- [ ] **Avoid repeated full prefix `WalkDir`.** `saves.Discover` → `DiscoverLiveSymlinks` does a full `WalkDir` of `drive_c` on every bundle; `Bundle` then also `listFiles` each entry. Memoize per-pack.
- [ ] **Consolidate scattered time helpers.** `timeNow()` (files.go), `stamp()` (stamp.go), `nowISO()` (pack/helpers.go + catalog) are three implementations of the same idea. Centralize in one `internal/timeutil` (or cfg) package.
- [ ] **Remove the dead `KnownTitleIDs` array.** `nswitch.KnownTitleIDs` is declared but `MatchTitle` never consults it (only `CuratedTitleIDMap` + the `[titleid]` regex). Wire it in as a fallback or delete it.

## 🚀 Speed

- [ ] **Make zstd thread count configurable instead of forced `-T1`.** `compress.zstdArgs` hardcodes single-threaded compression. The ROG Ally is 8-core Zen 2; `--long=27` window memory is modest, so more threads would cut the documented ~10 h / 89 GB pack substantially. Default to a higher thread count (e.g. `GOMAXPROCS`-aware) with an env/flag override, keeping `-T1` only as a low-memory fallback.
- [ ] **Pack multiple selected games concurrently.** The TUI `execute` loop processes `r.items` strictly sequentially in one goroutine. Independent game dirs can pack in parallel worker pools (bounded by disk/CPU) to leverage the multi-core handheld.
- [ ] **Reuse the cached index in CLI pack/NS flows.** `runCLI` rebuilds the index even when only `MarkArchived` matters; the NS pack path re-`LoadIndex`es mid-loop. Single load + in-memory update + one save.

## 🎮 QoL

- [ ] **Fix the misleading "safe to rerun" TUI copy** (and the README reboot-resilience wording) until the journal is actually wired (see Backup Integrity).
- [ ] **Make CLI `--unpack` behavior match the TUI.** CLI unpack passes `Interactive: false`, so if no prefix is in the archive it silently prints a notice and skips relinking; the TUI uses `Interactive: true` and launches Steam. Either default CLI to interactive when a TTY is present, or surface the limitation more loudly.
- [ ] **Tighten fuzzy name matching.** `findGame`/`findArchived` use substring `containsFold`, so `--pack "Cup"` matches both "Cuphead" and anything containing "cup". Prefer exact-name first, then a single unambiguous prefix match, with an explicit "multiple matches" error.
- [ ] **Add `--dry-run` for unpack** (currently only pack/list/refresh have it) so users can preview what will be restored/overwritten.
- [ ] **Surface archive health in the catalog list** — e.g. flag archives whose on-disk checksums no longer match the manifest (cheap integrity signal without a full verify).

## 🧹 Go Idioms / Error Handling

- [ ] **`copyTree` doesn't preserve symlinks.** It uses `os.ReadFile`/`os.WriteFile`, so any symlink in a save/prefix restore becomes a broken/regular file or errors. Use a copy that honors symlinks (or reuse `tar` for the move, which already preserves them).
- [ ] **Clean up staging on all error paths.** `unpack.flattenStage` failing leaks `stage`; `restoreSaves`/`restorePrefix` don't `RemoveAll` their staging dirs on early error. Wrap with deferred cleanup.
- [ ] **Pipe cleanup in `compress.PackDirArgs`.** On `tarCmd.Start()` failure zstd is killed but the pipe ends aren't closed/waited consistently; ensure both commands are `Wait()`ed and pipes closed in all branches to avoid leaked FDs/goroutines.
- [ ] **De-dupe `cfg.uint32Str` → `fmtUint32` → `uint32Str`** double-hop and the `relAfter`/`under` overlapping path logic.
- [ ] **Replace the brittle `ExcludeReason` AppID switch** (hardcoded magic IDs with inline game names) with a small data-driven table or a config file, so new emulators/tools don't require a code change.

## 🧪 Testing

- [ ] **The coverage gate exempts the riskiest packages.** `proton`, `state`, `unpack` are at 0–6% and waived "because they shell out." Those are exactly the packages that do the destructive deletes, the prefix restores, and the only checksum verify — the highest-blast-radius code. Add table-driven unit tests around their *pure* logic (path resolution, skip-newer decisions, preflight space checks, SHA-comparison branches) that don't need a live ROG.
- [ ] **Add a round-trip integration test** (Golden files / `testdata`) that packs a fixture game dir + fake saves and unpacks it, asserting checksums, symlink recreation, and skip-newer behavior. This would have caught the missing saves/prefix verification.
- [ ] **Test `ResolveGameDir` edge cases** (prefix-container vs plain, exe at depth >1, missing dir) since it gates the destructive `RemoveAll`.
- [ ] **Raise `MIN_COVERAGE`** from 40 toward the documented 80% target as the above tests land; the gate is currently only the floor.

---

## Priority-ordered master checklist

| # | Item | Category | Why |
|---|------|----------|-----|
| 1 | Wire or delete the journal; fix "safe to rerun" claim | Backup Integrity / QoL | Advertised safety is currently absent |
| 2 | Verify saves + prefix checksums on restore | Backup Integrity | Bit-rot on saves/prefix is silent today |
| 3 | Guard `RemoveAll(gameDir)` with an allowed-root check | Security | Destructive op can hit wrong tree |
| 4 | Sanitize tar extraction paths | Security | Zip-Slip escape on foreign archives |
| 5 | Unit-test `unpack`/`proton` pure logic (currently exempt) | Testing | Highest-blast-radius code is untested |
| 6 | Make zstd threads configurable (not forced `-T1`) | Speed | ~10 h packs on an 8-core device |
| 7 | Cache catalog / lazy stat on single-game ops | Efficiency | Full 53-game walk before every op |
| 8 | Stage + atomic-rename `prefix.tar.zst` restore | Security | Half-written compatdata on corruption |
| 9 | Skip `.conflict` files when bundling saves | Backup Integrity | Archive syncthing conflicts as good saves |
| 10 | Concurrent multi-game packing in TUI | Speed | Sequential loop underuses cores |

---

## 🌊 Modernization Wave Status (2026-08-24)

**Current branch / SHA:** `feature/modernization` @ `72f1a14e5292d5240de75ded5143395dca55fba9` (clean except untracked `MODERNIZATION-TODO.md` + `README.md`).
**Source docs:** `/tmp/baseline-rog.md` (Phase 1) · `/tmp/second-audit-rog.md` (Phase 2).

### ✅ Wave 1 completions
- **None committed against the audit items in this wave.** Wave 1 was the *audit + second-audit verification* pass. The repo's own recent history (`feature/modernization`) already closed earlier items: coverage gate + raised pure-package coverage (`2e78a31`), NSwitch CLI parity, game-running detection fix, partial-archive re-pack, source-folder removal after pack — these predate this TODO and are tracked in `CHANGELOG`/git log, not here.
- The second audit re-verified **14 TODO claims VERIFIED, 2 REFINED, 0 NOT MATCHED** — i.e. the TODO's correctness/security/reliability/perf/DX claims are all grounded in real code at `72f1a14`. No claim was found false.

### 🔧 Corrections found (second audit refines the TODO text)
- **TODO #7 overstated:** `BuildCatalog` full re-walk is now **cold-cache only** (`list.go:113` calls it only when `len(games)==0`; `runCLI` rebuilds index only for `refresh/dry/list`, not a plain `--pack`). Downgrade from "every single-game op" to "cold-cache only." The repeated `WalkDir` of `drive_c` in `saves.Discover` remains un-memoized.
- **"Makefile coverage gate" is a misnomer:** there is **no Makefile** in the repo. The gate actually lives in `scripts/coverage-check.sh` (`MIN_COVERAGE=40`, with 80% stated as the long-term target). Fix any downstream doc that cites a Makefile gate.
- **TODO #24 nuance confirmed:** hardcoded `/var/home/jevonx` is not just the Steam path — it's also in `cfg.Default()` for ShortcutsVdf/CompatData/GameSaves/LudusaviPath/CitronSaves. Whole tool bound to user `jevonx`; `cfg.Config` has no `SteamUser`/`SteamHome` fields.

### 📋 Wave 2 queue (remaining — all need code changes)
- **SAFETY/MODERNIZATION:** #3 lexical guard on both `RemoveAll` sites (reuse `saves/files.go under()`); #4 `--no-absolute-names` + member validation in `compress.Extract`; #8 stage+atomic-rename prefix restore; #2 verify `SHA256Saves`/`SHA256Prefix` on restore; #24 add `SteamUser`/`SteamHome` to `cfg` + derive pid/pipe.
- **RELIABILITY:** #1 wire-or-delete journal + fix `run.go:279` "safe to rerun" copy; #9 `.conflict` exclude in `Bundle`; #10 staging swap for saves restore; staging cleanup on all error paths.
- **PERFORMANCE:** #6 configurable zstd threads (GOMAXPROCS-aware + env override); memoize `DiscoverLiveSymlinks` WalkDir; #7 now low-priority (cold-cache only).
- **DX/TESTING:** raise `MIN_COVERAGE` toward 80; add pure-logic tests for `unpack`/`proton`/`state` + `ResolveGameDir` edge cases + a round-trip pack→unpack integration test.
