# Session Learnings: 2026-05-27 — Game Manifest + Relocation Feature

## Context
Added game manifest tracking (`shortcuts-manifest.json`), missing-game cleanup (menu [6]), and automated relocation detection when a game is moved to a new path.

## Technical Lessons

### 1. Manifest Design: JSON Beside VDF
- Manifest file `shortcuts-manifest.json` lives alongside `shortcuts.vdf` in each Steam user's config dir (`userdata/<id>/config/`)
- Per-user tracking is essential because Steam is multi-user and each user has independent shortcuts
- Format: `{"version": 1, "entries": [{app_name, exe_path, appid, icon_path, status, last_seen}, ...]}`
- Version field allows future schema migration; version mismatch triggers clean reset (with backup of old file)
- Two states: `"active"` (currently in VDF) and `"removed"` (previously added, exe gone, preserved for relocation)
- Atomic write: write to temp file in same dir, `Path.replace()` for atomic rename, `chmod 0o600` after
- Corrupt JSON detected on load → backup as `.json.bak`, reset to empty

### 2. Critical Safety: Parent Dir Check Before Removal
```python
parent = exe_path_obj.parent
if not parent.exists():
    # Drive unmounted; skip to avoid mass-deleting shortcuts
    continue
```
Without this guard, if a user has games on an unmounted external drive, all those shortcuts would be deleted. Checking `parent.exists()` before `exe_path_obj.exists()` prevents this. The log message includes "(drive unmounted?)" for debuggability.

### 3. Relocation Matching Strategy
Three-tier matching when scanning for new games with removed manifest entries:
1. **Similarity score** — `similarity_score()` strips non-alphanum, lowercases, uses `difflib.SequenceMatcher.ratio()`; threshold ≥0.90
2. **Exe stem fallback** — If similarity fails, compare `Path(old_exe).stem.lower()` == `Path(candidate.exe_path).stem.lower()`; exact stem match scores 0.91 (above threshold)
3. **User confirmation** — Show old vs. candidate with match percentage; `input("Re-add [y/N]?")` — preserves original appid via `build_shortcut_with_appid()`
4. **Dedup** — Relocated paths are removed from the new-game candidate pool to prevent double-add

Key insight: Relocation detection must run against `get_removed_entries()` only, not `get_active_entries()`. Active entries are still in shortcuts.vdf; removed entries are the ones we want to recover.

### 4. Path Normalization Consistency
Both `game_manifest.py` and `shortcut_builder.py` must use the **same** `normalize_posix_path()`:
- `GameManifest._normalize_exe()` calls `normalize_posix_path()` then wraps in `Path()` for absolute check
- `normalized_exe_identity()` calls `unquote_posix_path()` which calls `normalize_posix_path()`
- If they diverged (e.g., one stripped trailing slashes and the other didn't), manifest lookups by exe path would silently fail, causing duplicate entries or missed relocations
- Lesson: keep path normalization in `shortcut_builder.py`, import it everywhere. Never reimplement.

### 5. `_safe_int` Wrapper for JSON-Origin Values
```python
def _safe_int(value, default: int = 0) -> int:
    try:
        return int(value) if value is not None else default
    except (TypeError, ValueError):
        return default
```
JSON may contain non-integer types for numeric fields (e.g., float `100.0`, string `"100"`, or `null`). `int(value)` handles float and string conversion; `None` returns default; `TypeError`/`ValueError` catch garbage. Used for `appid` in manifest serialization and `_serialize_entry()`.

### 6. `chmod 0o600` on Manifest Files
```python
path.chmod(_stat.S_IRUSR | _stat.S_IWUSR)  # 0o600
```
Manifest contains game paths which reveal the user's filesystem layout and game install locations — a privacy concern. Restricted permissions prevent other users on the system from reading it. Same pattern as `.env` file permission check in `config.py`.

### 7. Manifest Entry Dedup by Normalized Exe
`add_or_update()` deduplicates by normalized exe path:
```python
for entry in self._entries:
    if self._normalize_exe(entry.get("exe_path", "")) == normalized:
        entry.update({...})  # update in place, not append
        return
```
This means re-adding a previously-removed game (during relocation) just flips status back to `"active"` and updates `last_seen`. No duplicate entries.

### 8. `_apply_shortcuts()` Dual Path (New Games + Relocated)
Works in two loops within the same function:
- **Relocated loop** — Uses `build_shortcut_with_appid()` to preserve original appid (keeps Steam artwork mapping)
- **New-game loop** — Uses `build_shortcut()` which generates a fresh appid via CRC32
Both paths update the manifest with `add_or_update()` with status `"active"`, so the manifest stays consistent regardless of which path added the game.

### 9. Test Patterns for Manifest
Key test patterns added in `test_manifest.py`:
- **Round-trip**: create manifest, add entries, reload from disk, verify count and lookup
- **Corrupt recovery**: write invalid JSON, verify manifest resets and `.json.bak` is created
- **Status transitions**: active→removed→active cycle using `set_status()` and `add_or_update()`
- **Unmounted drive**: add entry with `/nonexistent_mount/game`, verify parent doesn't exist, manifest still has active entry
- **Version mismatch**: write version 999, verify manifest resets
- **`build_shortcut_with_appid()`**: verify appid is preserved exactly (signed int, e.g., -12345)

### Files Modified
- `game_manifest.py` — Created: `GameManifest` class, `_safe_int`, `_serialize_entry`, `_now_iso`, atomic save with 0o600
- `main.py` — Added `cleanup_missing_games()`, `check_and_handle_missing_games()`, `prompt_relocation_match()`, `_apply_shortcuts()` (relocated + new dual path), `_load_manifest_for_user()`, menu option [6]; refactored `scan_and_add_games()` and `full_run()` to include manifest/relocation
- `shortcut_builder.py` — Added `build_shortcut_with_appid()`, `similarity_score()`; `normalize_posix_path()`, `normalized_exe_identity()`, `unquote_posix_path()` already existed
- `tests/test_manifest.py` — Created: 16 tests covering round-trip, corrupt recovery, find_by_exe, find_by_appid, status transitions, name similarity, add-or-update timestamps, active exe paths, remove_entry, invalid status, version mismatch, non-dict root, non-list entries, build_shortcut_with_appid, missing game detection, unmounted drive skip
- `tests/test_helpers.py` — Created: `make_executable()` utility for test exe creation

### Commands Worth Remembering
```bash
# Run manifest-specific tests
python -m pytest tests/test_manifest.py -v

# Test relocation detection manually
python -c "
from game_manifest import GameManifest
from pathlib import Path
m = GameManifest(Path('/tmp/test-manifest.json'))
m.add_or_update('Hollow Knight', '/old/path/hollow_knight', -12345, status='removed')
print('Removed entries:', len(m.get_removed_entries()))
"

# Check manifest file contents
cat ~/.local/share/Steam/userdata/*/config/shortcuts-manifest.json | python -m json.tool

# Simulate unmounted drive test
python -c "
from pathlib import Path
p = Path('/nonexistent_mount/some_game.exe')
print('Parent exists:', p.parent.exists())  # False → safe skip
print('Exe exists:', p.exists())            # False
"

# Quick manifest existence check across all users
ls -la ~/.local/share/Steam/userdata/*/config/shortcuts-manifest.json 2>/dev/null
```



# Session Learnings: 2026-05-22 — Add Non-Steam Games Linux Scanner Improvements

## Context
Improved false flag detection in Steam non-Steam game scanner (Linux). Added Wine prefix filtering, overlay/injector detection, engine data file filtering, and .env setup message.

## Technical Lessons

### 1. Python `Path.with_suffix()` Gotcha
`Path(".env").with_suffix(".env.example")` produces `.env.env.example` not `.env.example`.
- `.env` has no suffix (just a leading dot), so `with_suffix()` appends to an empty suffix
- Use `env_path.parent / ".env.example"` instead

### 2. Multi-Layer Filtering Strategy
False flag filtering needs multiple layers:
1. **Executable-level**: Filter by stem name (`WINE_SYSTEM_EXE_STEMS`, `SKIP_EXE_STEMS`)
2. **Path-level**: Filter by path patterns (Wine prefix dirs, overlay dirs, engine data)
3. **Scoring**: Low-score items auto-flagged `[needs review]` in UI
4. **Composite check**: `_is_game_executable()` combines all filters

### 3. Game Executable Detection Complexity
- `.exe` files are accepted unconditionally (no execute bit needed on Linux)
- Extensionless files require execute bits (but "unity default resources" can have them)
- `.bin` files in `Engine/Content/` paths are shader caches, not games
- Level data files (level0, level46, level220) are Unity data, not executables

### 4. Wine Prefix Structure
Wine prefixes have `drive_c/windows/system32/`, `drive_c/windows/syswow64/`, etc. Real game exes may also be inside `drive_c/Program Files/`. Only skip Windows system directories, not the entire Wine prefix. Also skip `/windows/microsoft.net/` (Framework tools) and individual system tools like `msbuild`, `hh`.

### 5. .env Security
- `.env` file must not be world-readable: check `st_mode & S_IROTH` and raise `PermissionError`
- `load_local_env()` must not use `@lru_cache` — user may edit `.env` during session
- `check_env_setup()` must validate actual non-empty key value, not just string presence

### 6. Test Strategy
- Unit tests for individual filter functions (`_is_game_executable`)
- Integration tests with `discover_games()` on real `~/Games` directory
- E2E tests verify specific false positive files are filtered
- Use `tempfile.TemporaryDirectory()` for all path tests (never hardcode `/home/user/` paths)
- Never use `os.chdir()` in tests — use `unittest.mock` instead to avoid global side effects
- Module-level imports preferred over per-function imports in tests

### 7. Profile-Specific Filtering
On this system, Crypt of the NecroDancer ships `beatdown.exe` and `beattracker.exe` in `data/custom_music/` and `data/essentia/` — these are audio tools, not games. Add `custom_music` and `essentia` to `SKIP_PATH_KEYWORDS` to filter them.

### Files Modified
- `config.py` — Added WINE_SYSTEM_EXE_STEMS, OVERLAY_DIR_NAMES, ENGINE_DATA_FILE_STEMS; expanded SKIP_PATH_KEYWORDS; removed `@lru_cache`; added `.env` permission check; removed dead `ENV_FILE_NAMES`; removed `HOME = Path.home()` inconsistency; added `GAME_HINT_FILES_LOWER` optimization
- `game_scanner.py` — Added `_is_game_executable()`, `_path_is_in_wine_prefix()`, `_path_is_in_overlay_dir()`, `_path_is_in_steam_downloading()`, `_path_is_engine_data()`; expanded engine data detection; case-insensitive system path check; moved `MAX_EXTENDED_SCAN_DEPTH` after imports
- `main.py` — Added `check_env_setup()` for .env startup message with actual key validation; removed redundant `from pathlib import Path`
- `shortcut_builder.py` — Removed dead Windows-path aliases (`normalize_windows_path`, `quote_windows_path`, `unquote_windows_path`)
- `tests/test_safety.py` — Added 12 new tests; module-level `_is_game_executable` import; removed per-function re-imports; removed `os.chdir` in env test; moved test classes before `if __name__`; fixed `import stat as st` to use module-level `stat`
- `README.md` — Created

### Commands Worth Remembering
```bash
# Run all tests
python -m pytest tests/ -v

# Test specific class
python -m pytest tests/test_safety.py::FalseFlagDetectionTests -v

# Run E2E scan against ~/Games
python -c "from game_scanner import discover_games; from pathlib import Path; import logging; results = discover_games(set(), [], logging.getLogger(), set()); print(f'Found {len(results)} candidates')"

# Test if a specific file is filtered
python -c "from game_scanner import _is_game_executable; print(_is_game_executable(Path('/path/to/file.exe')))"

# Dry-run validation (safe, no writes)
python main.py --dry-run-check

# Flow validation (safe, no writes)
python main.py --validate-flows
```
