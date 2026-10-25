# Modernization Todo List — Add-Non-Steam-Games-Linux

Audit date: 2026-08-24
Scope: `main.py`, `vdf_manager.py`, `game_scanner.py`, `artwork_manager.py`,
`steam_paths.py`, `fixer.py`, `shortcut_builder.py`, `game_manifest.py`,
`config.py`, `logger_setup.py`, `tests/`, `requirements.txt`.

Summary: The codebase is already in good shape for a single-purpose Linux
tool — `from __future__ import annotations` everywhere, `slots=True`
dataclasses, safe VDF writes via atomic temp-file replace, Steam-running
guard, backup+restore on write failure, masking of API keys in logs, and a
passing test suite (72 passed + 18 subtests). The gaps below are
modernization/robustness wins, not correctness emergencies.

Priorities: **P0** = security/safety/correctness risk, **P1** = efficiency/robustness/UX, **P2** = polish/DX.
Effort: **S** (<1h), **M** (½–1 day), **L** (>1 day).

---

## Status (as of 2026-08-24)

- **Current branch:** `master` @ `0ad5026` (baseline; clean working tree for tracked files).
- **Audit Wave 1 — COMPLETE.** Two passes finished, **no code changes made**:
  - Phase 1 baseline audit (`/tmp/baseline-AddNonSteam.md`): git state, build/test/deps, dangerous operations, TODO-vs-code verification. Test suite green: **72 passed + 18 subtests**.
  - Phase 2 second audit (`/tmp/second-audit-AddNonSteam.md`): Security / Reliability / Performance / DX lenses. Reconciled every TODO item against source; flagged over-aggressive recommendations and one understated estimate.
- **Branches created at baseline SHA `0ad5026` for the implementation wave:**
  - `modernization/2026-08-24-safety` — VDF size/depth bounds, `libraryfolders` parse guard, artwork SSRF allow-list, `.env` hardening.
  - `modernization/2026-08-24-feature` — `pyproject.toml`, `argparse`/`typer` CLI, CI matrix, alias-dict consolidation.
  - (`master` left untouched; Wave 2 code work happens on these branches.)
- **Wave 2 (code implementation) — QUEUED, not started.** See "Wave 2 Queue" at the end of this doc.

### Corrections from the second audit (apply when implementing)
- `clean_game_name` regex estimate was **understated**: ~62 regex applications/call (16 explicit `re.sub` + 2 loops over 31 patterns), **none precompiled**. The `lru_cache` + `re.compile` fix is *more* valuable than the original TODO implies — prioritize it.
- `.env` "tighten to `0o600` / reject group-readable" is **over-aggressive**: a hard `PermissionError` breaks legit group-readable setups (flatpak sandbox gid). **Warn + opt-out (`STEAMGRIDDB_ENV_INSECURE=1`), keep the world-readable block.**
- `asyncio` + `aiohttp` rewrite is **over-aggressive**: adds a 2nd HTTP dependency and risks breaking the tuned 401/429 rate-limit logic. **Do the JSON result cache + `ThreadPoolExecutor`/`Semaphore` wins first; defer full async.**
- `mypy` "strict-ish" config is correct — **do NOT escalate to full `--strict`** on day one (large untyped-param churn).
- `test_helpers.py` dead weight: **relocate `make_executable` to `tests/_helpers.py`** (private, not collected), don't delete the helper.

### Update 2026-10-15 (documentation pass — no code changes)

Verified against `git log`, the working tree and a live `make test`:

- **Landed since this audit:** VDF size/depth bounds + artwork SSRF allow-list
  (`86444fa`), precompiled regexes + `lru_cache` hot-path cleaners (`d2bb95e`),
  TTL-bounded on-disk SGDB cache (`fdc5c43`), `Path.stat()` discovery cache
  (`883e95a`), GitHub Actions pytest matrix + shellcheck (`1cb5f66`),
  pre-commit shellcheck config (`3cca165`), Dependabot (`7be64f9`),
  Makefile (`ab70180`), `.python-version` + `.editorconfig` (`8d6daf8`).
  → the "No CI workflow" QoL item, the Wave-2 safety branch items, and the
  performance branch cheap wins are now **done**; the checkboxes below are not
  updated and should be treated as stale.
- **Still open:** `pyproject.toml`, `argparse`/subcommand CLI, `mypy`,
  upper-bound/lock for `requests`, `tests/_helpers.py` relocation,
  alias-dict consolidation, `ThreadPoolExecutor` artwork stage,
  `LOG_LEVEL` override, documenting `auto-run.py`/`refresh-art.py`.
- **Branch state:** `master` is now at `ae2d1ba` (not the `0ad5026` baseline),
  UNPUSHED by one commit.
- **Suite:** green — **78 passed + 18 subtests** (was 72 + 18).

---

## Security

- [ ] **P1 · S** — `requirements.txt` pins `vdf==3.4` but `requests>=2.33.0` is
  unpinned (floats to latest). Add an upper bound or a lockfile
  (`requirements.lock` / `pip-tools`) so CI and `run.sh` reproduce the same
  `requests`. (`.env` safety check exists in `config.load_local_env`, good.)
- [ ] **P1 · M** — `vdf.bin` parsing (`vdf_manager.load_shortcuts`) does not bound
  input size or depth. A malicious/corrupt `shortcuts.vdf` (or
  `loginusers.vdf`/`libraryfolders.vdf` in `steam_paths.py`) could be
  oversized/cyclic. Add a max-bytes guard and prefer the existing
  `round_trip_integrity_test` / structured exceptions over a bare
  `RuntimeError`. `vdf.load` is called on text files without try/except in a
  few spots (`steam_paths._load_login_user_ids` is guarded; `load_libraryfolders`
  is NOT — a corrupt `libraryfolders.vdf` raises uncaught).
- [ ] **P2 · S** — `artwork_manager._mask_api_key` logs a masked key, but the raw
  key also flows through `Authorization: Bearer …` header (fine) and into
  `validate_api_key`. Ensure `.env` world-readable check (`config.py`) covers the
  flatpak/env paths too. Consider reading the key from an OS keyring as an option.
- [ ] **P2 · S** — `resolve_api_key` uses `getpass.getpass`; fine. But
  `config.load_local_env` only blocks `S_IROTH` (world-readable), not group-read
  (`S_IRGRP`). On multi-user systems tighten to `0o600` enforcement.

---

## Efficiency

- [ ] **P1 · M** — `game_scanner.discover_games` walks every `KNOWN_GAME_DIRS`
  entry fully (`MAX_SCAN_DEPTH`/`MAX_EXTENDED_SCAN_DEPTH`) and re-stats files
  repeatedly (`_is_game_executable` + `_candidate_score` + `_select_best_executable`
  re-walk). For large libraries this is O(N²) file statting. Cache `Path.stat()`
  results per directory and reuse across the candidate/scoring passes.
- [ ] **P1 · M** — `steam_paths.list_steam_users` sorts by `last_modified` and
  `game_scanner` sorts `candidate_dirs`; both fine, but `discover_games` calls
  `_select_best_executable` (a full `os.walk` to depth 5) for *every* candidate
  dir — including dirs already known to contain only hints. Add an early-exit once
  a high-confidence exe is found.
- [ ] **P2 · S** — `artwork_manager` serializes requests with
  `ARTWORK_REQUEST_DELAY_SECONDS=0.5` but uses a *module-level* `time.sleep` per
  call; for many games this dominates runtime. Batch search by submitting multiple
  terms and de-dupe matches (already partly done). Consider `concurrent.futures`
  with a rate-limited worker.
- [ ] **P2 · M** — No caching of SteamGridDB search results. Re-running
  `--validate-flows` / full-run re-queries every game. Add a small
  on-disk/JSON cache keyed by normalized lookup text (TTL-bounded).

---

## Speed / Performance

- [ ] **P1 · M** — `shortcut_builder.build_search_aliases` builds N roman/arabic
  variant regexes via `re.sub` per candidate; fine at small scale but the
  `ROMAN_TO_ARABIC`/`ROMAN_NUMERALS` loops run for every alias. Precompile the
  variant regexes once at module load.
- [ ] **P2 · S** — `clean_game_name` runs ~12 sequential `re.sub` passes plus
  regex camelCase splits on every call (called per dir + per exe + per alias).
  Memoize with `@functools.lru_cache` keyed on the input string (pure function).
- [ ] **P2 · S** — `similarity_score` (`difflib.SequenceMatcher`) is invoked in hot
  loops (`_candidate_score`, `manifest.find_by_name`, `choose_best_match`).
  For short game names a cheaper normalized-Levenshtein/Jaccard is enough;
  at minimum cache its results.
- [ ] **P2 · M** — No async anywhere (single-threaded `requests`). Artwork download
  (many games × 5 art types) is the slowest path. A `asyncio` + `aiohttp` client
  with a semaphore would cut wall-clock dramatically; keep the sync CLI shell.

---

## Quality of Life / Developer Experience

- [ ] **P0 · M** — **No `pyproject.toml` / packaging metadata.** `run.sh` bootstraps
  a `.venv` but there is no `[project]` table, no console-script entry point, and
  no `[tool.{ruff,pytest,mypy,black}]` config. Add `pyproject.toml` with
  `requires-python`, deps, dev deps, and tool configs so `pytest`/`ruff`/`mypy`
  are reproducible.
- [ ] **P0 · S** — **CLI is flag-string matched, not `argparse`.** `main()` checks
  `"--diagnose" in sys.argv[1:]` and dispatches menu options via `input()` +
  string equality (`choice == "1"`). This is brittle and untestable from the CLI.
  Migrate to `argparse` (or `click`/`typer`) with subcommands:
  `scan`, `fix`, `artwork`, `list`, `cleanup`, `diagnose`, `validate`. (Menu
  interactivity can stay as a `tui` subcommand.)
- [ ] **P1 · M** — **No type checking in CI.** `from __future__ import annotations`
  is used everywhere but nothing runs `mypy`/`pyright`. Add a strict-ish config and
  a pre-commit hook. Several functions have untyped params (`logger`, `shortcuts_map`)
  that would be caught.
- [ ] **P1 · M** — **Test coverage is uneven.** 72 tests pass but `test_helpers.py`
  contains *zero* tests (only a `make_executable` helper), and the big risk areas
  are thinly covered: `game_scanner.scan` logic, `artwork_manager` network paths
  (only `test_artwork_fixes` touches it, likely mocked), and **`main.py` CLI is
  entirely untested** (no `main()`/dispatch tests). Add unit tests for
  `game_scanner`, and CLI tests via `typer.testing.CliRunner` or `argparse`
  `capsys`.
- [ ] **P1 · S** — **No mock/network isolation guarantee.** `test_artwork_fixes.py`
  references `requests`/`responses`; confirm no test hits the real SteamGridDB API
  (a leaked key or rate-limit could break CI). Pin `pytest` + a `responses`/`respx`
  fixture.
- [ ] **P1 · S** — `test_helpers.py` is dead weight (helpers unused by any test).
  Either wire `make_executable` into `game_scanner` tests or delete the file;
  clearly label helpers vs. tests.
- [ ] **P2 · S** — Add `ruff`/`flake8` + `black`/`isort` config and a pre-commit
  hook. Currently formatting is inconsistent (e.g. mixed blank-line spacing in
  `config.py` lines 251-252, 436).
- [ ] **P2 · M** — **No CI workflow.** Add a GitHub Actions (or equivalent) matrix
  on Python 3.10/3.11/3.12/3.13 running `pytest` + `ruff` + `mypy`. The
  `.pytest_cache` shows the suite was run under 3.11/3.13/3.14 already, so support
  already spans those — codify it.
- [ ] **P2 · S** — `config.py` hardcodes CachyOS-specific dirs
  (`HOME/"Games"`, `/mnt/games`, `KNOWN_GAME_DIRS` comments "on this CachyOS
  system", and `MANUAL_TITLE_ALIASES`/`OFFICIAL_TITLE_ALIASES` with per-user game
  titles). These should be data files (YAML/JSON) or user-overridable, not baked
  into source — they bit-rot and pollute the module with user-specific state.
- [ ] **P2 · S** — `shortcut_builder.prettify_exe_stem` and `clean_game_name` share
  the manual-alias dict but `MANUAL_TITLE_ALIASES` duplicates entries that also
  exist in `OFFICIAL_TITLE_ALIASES`; consolidate to a single source of truth.
- [ ] **P2 · S** — `logger_setup.setup_logging` creates a `RotatingFileHandler` with
  `backupCount=3` but never exposes log-level/format config; allow `LOG_LEVEL` env
  override for debugging.
- [ ] **P2 · S** — `auto-run.py` / `refresh-art.py` are separate entry scripts not
  referenced by `run.sh` or tested; document or fold into `main.py` subcommands.

---

## Notable strengths (keep)
- Atomic VDF writes via `tempfile.NamedTemporaryFile` + `Path.replace`
  (`vdf_manager.write_shortcuts`, `game_manifest._save`).
- Backup-then-write with restore-on-failure (`main.write_user_shortcuts`).
- `ensure_ready_to_write` refuses to touch `shortcuts.vdf` while Steam runs.
- API key masked in logs; `.env` world-readable guard.
- Round-trip VDF integrity + field-quoting verification tests.
- `GameManifest` versioned, corrupt-file `.bak` fallback, atomic save.

## Wave 2 Queue (code implementation — not yet started)

Ordered per the second audit's safe-phase plan. Work lands on the two
`modernization/2026-08-24-*` branches; `master` stays at baseline `0ad5026`
until merged. Re-run `pytest` after each branch and keep the suite green
(72 passed + 18 subtests).

### Branch `modernization/2026-08-24-safety` (no regressions)
1. Bound VDF loads: `MAX_VDF_BYTES` guard in `vdf_manager.load_shortcuts` /
   `round_trip_integrity_test` / `steam_paths.load_libraryfolders`; guard
   `load_libraryfolders` parse with a structured exception. — P1/M
2. Artwork SSRF allow-list: validate SGDB download URL is `https` + host ends
   with `steamgriddb.com` before `download_to_file`. — P2/S
3. `.env`: keep the world-readable block; convert group-readable to
   **warn + opt-out (`STEAMGRIDDB_ENV_INSECURE=1`)**, not a hard fail. — P2/S

### Branch `modernization/2026-08-24-feature`
4. Add `pyproject.toml` (deps + `[tool.ruff]`/`[tool.pytest]`/`[tool.mypy]`
   strict-ish); pin `requests` upper bound. — P0/M
5. Migrate CLI to `argparse`/`typer` subcommands; add `capsys` dispatch tests
   (`main.py` currently 0 tests). — P0/M
6. Add GitHub Actions matrix (3.10–3.13) + `respx`/`responses` network-isolation
   fixture. — P2/M
7. Consolidate `MANUAL_TITLE_ALIASES`/`OFFICIAL_TITLE_ALIASES` to a single
   source of truth + collision test; relocate `test_helpers` → `tests/_helpers.py`. — P2/S

### Performance branch (later — cheap-wins-first)
8. `@lru_cache` on `clean_game_name`/`normalize_lookup_text`/`similarity_score`;
   `re.compile` the 31 module-scope patterns + ROMAN/ARABIC variant regexes. — P1/M / P2/S
9. On-disk SGDB result cache (TTL-bounded) — **before** any concurrency change. — P2/M
10. `ThreadPoolExecutor` + `Semaphore` over the existing `requests.Session`
    (respects the 0.5 s throttle); **defer the `aiohttp` rewrite.** — P2/M
11. `discover_games` `stat()` cache + early-exit on a high-confidence exe. — P1/M

### P1/S quick wins folded into the above branches
- Pin `requests` (upper bound / lockfile). — P1/S
- `mypy` strict-ish config + pre-commit (`do NOT` go full `--strict`). — P1/M
- Network-isolation guarantee for artwork tests. — P1/S
- `LOG_LEVEL` env override in `setup_logging`. — P2/S
- Document or fold `auto-run.py` / `refresh-art.py` into subcommands. — P2/S

> Original "Suggested first sprint" (pyproject → CLI → pin requests → mypy/CI →
> VDF guards) is superseded by the Wave 2 Queue above, which also folds in the
> second-audit corrections (skip hard `0o600`, defer `aiohttp`, preserve
> `make_executable` helper).
