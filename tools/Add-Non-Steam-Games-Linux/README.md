# Add Non-Steam Games Linux

Scan game directories, add non-Steam games to Steam, download artwork from SteamGridDB. Filters Wine prefix system files, overlay injectors, engine data, and Steam downloading dirs.

## Quick Start

```bash
printf 'STEAMGRIDDB_API_KEY=your_key_here\n' > .env   # optional, see note
./run.sh                    # preferred launcher (see below)
```

> There is **no `.env.example`** in the repo (removed in commit `e351193`
> "cleanup env example"; `.gitignore` still whitelists it if it ever returns).
> Create `.env` directly, or skip it — artwork actions then prompt for the key
> interactively via `getpass` and never persist it. `.env` is gitignored; never
> commit a key.

## `run.sh` launcher

`./run.sh` is the preferred entry point: it picks `.venv/bin/python3` when
present (system `python3` otherwise, 3.10+ required — enforced by
`config.MIN_PYTHON`), bootstraps `.venv` and installs `requirements.txt` on
first run, then `exec`s `main.py "$@"` — all CLI flags pass through
(e.g. `./run.sh --diagnose`). Note: it only prints the "copy `.env.example`"
reminder when `.env.example` actually exists, and that file does not exist in
this repo, so no reminder is shown. Manual alternative:
`pip install -r requirements.txt && python main.py`.

> Packaging note: there is no `pyproject.toml` yet — Wave 2 queue item in
> MODERNIZATION-TODO.md (packaging metadata + `[tool.ruff/pytest/mypy]`
> config). Deferred; `requirements.txt` + `run.sh` remain the install path.

## CLI

| Flag | Description |
|------|-------------|
| `--diagnose` | Run diagnostics |
| `--dry-run-check` | Validate setup without making changes |
| `--validate-flows` | Validate all workflows |

These three are non-destructive (they only read Steam state). There is no
`--help` and no `argparse` yet — flags are matched as strings in `main()`
(migrating to real subcommands is a P0 backlog item, see MODERNIZATION-TODO.md).
Anything else, including no flags at all, drops into the interactive menu below.

## Menu

| Option | Description |
|--------|-------------|
| `[1]` | Scan for broken shortcuts and fix them interactively |
| `[2]` | Scan for new games to add (includes relocation detection) |
| `[3]` | Download artwork for all shortcuts from SteamGridDB |
| `[4]` | Full run: fix → check missing → add new (with relocation) → artwork |
| `[5]` | List all current non-Steam shortcuts |
| `[6]` | Check for missing games and remove their shortcuts |
| `[0]` | Exit |

## .env

| Variable | Description |
|----------|-------------|
| `STEAMGRIDDB_API_KEY` | SteamGridDB API key for artwork |
| `SCAN_DIRS` | Extra dirs to scan (colon/comma/semicolon separated; falls back to `GAME_SCAN_DIRS`) |
| `STEAM_PATH` | Override Steam install path |

Process-environment values are also honored; `.env` next to `main.py` is read by
`config.load_local_env`, which refuses world-readable `.env` files.

## Game Manifest (`shortcuts-manifest.json`)

A per-user JSON manifest sits alongside `shortcuts.vdf` in each Steam user's config dir. It tracks every game that has been added, enabling two critical features: **missing-game cleanup** and **relocation detection**.

### Manifest File Location

```
~/.local/share/Steam/userdata/<user_id>/config/shortcuts-manifest.json
```

### Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| `app_name` | string | Game display name |
| `exe_path` | string | Normalized absolute path to game executable |
| `appid` | int | Steam-generated signed shortcut appid |
| `icon_path` | string | Optional icon path |
| `status` | string | `"active"` or `"removed"` |
| `last_seen` | string | ISO-8601 UTC timestamp of last add/update |

### States

- **`active`** — Game is currently in shortcuts.vdf and the exe file exists
- **`removed`** — Game was previously added but the exe file no longer existed when a cleanup pass ran; the shortcut was removed from shortcuts.vdf but the manifest entry is preserved for relocation

### Menu Option [6] — Missing Game Cleanup

Scans manifest entries with status `"active"`. For each entry:

1. Check if parent dir of `exe_path` exists → if not, assume drive unmounted and **skip** (never remove)
2. Check if `exe_path` itself exists → if yes, skip
3. If exe missing and parent exists → remove shortcut from `shortcuts.vdf`, set manifest status to `"removed"`

### Relocation Workflow

Triggered inside `[2] Scan` and `[4] Full run`. When previously-removed manifest entries exist, the tool checks newly discovered games for matches:

1. **Similarity match** — `similarity_score()` compares normalized app names (regex-stripped alphanumeric `SequenceMatcher` ratio, threshold ≥0.90)
2. **Exe stem match** — If similarity fails, compare `Path(exe).stem` (lowercased) between removed entry and candidate; exact stem match scores 0.91
3. **User confirmation** — Top candidate(s) shown with score; user confirms `y` to re-add with original appid
4. **appid preservation** — `build_shortcut_with_appid()` re-creates the shortcut using the original appid, preserving Steam artwork associations
5. **Dedup** — Relocated paths are excluded from the new-game candidate list to avoid double-add

### Safe Unmounted-Drive Handling

Before checking whether a game exe exists, `check_and_handle_missing_games()` checks `parent.exists()`:

```python
parent = exe_path_obj.parent
if not parent.exists():
    # Drive unmounted; skip to avoid removing games on removable media
    continue
```

This prevents mass-deletion of shortcuts when an external drive or secondary mount is temporarily unavailable.

## Structure

```
main.py             CLI + workflow orchestration
config.py           Scan patterns + skip lists + .env loading
game_scanner.py     Game discovery + false flag detection
artwork_manager.py  SteamGridDB artwork downloads
shortcut_builder.py Shortcut VDF construction + path normalization
vdf_manager.py      VDF read/write with validation
steam_paths.py      Steam installation detection
fixer.py            Interactive shortcut fixing
game_manifest.py    Manifest tracking (active/removed states, relocation)
logger_setup.py     Logging config
run.sh              Preferred launcher (venv bootstrap + exec main.py)
auto-run.py         Non-interactive wrapper: add ALL games, no prompts
refresh-art.py      Non-interactive wrapper: force-refresh artwork for all shortcuts
requirements.txt    Runtime deps: vdf, requests
Makefile            test / lint / clean / install targets
tests/
  test_safety.py    Safety and false-flag tests
  test_manifest.py  Manifest round-trip, relocation, unmounted-drive tests
  test_artwork_fixes.py  Artwork path/normalization tests
  test_helpers.py   Test utilities (make_executable, etc.)
docs/archive/       Historical notes (record only — not action items)
```

## Development

```bash
make test     # python3 -m pytest -q tests
make lint     # shellcheck --shell=bash --external-sources run.sh
make install  # create .venv + install requirements.txt
make clean    # drop caches, venv, coverage artifacts
```

`make help` lists the same targets. CI (`.github/workflows/ci.yml`) runs the
pytest matrix on Python 3.10–3.13 plus the shellcheck job; `pre-commit` and
Dependabot are also configured. There is no `pyproject.toml` yet — Wave 2 queue
item in MODERNIZATION-TODO.md; `requirements.txt` + `run.sh` remain the install
path.

Real runs must **not** have Steam open (`ensure_ready_to_write` refuses), and
any write to `shortcuts.vdf` is preceded by a backup.

`auto-run.py` / `refresh-art.py` are non-interactive wrappers, not part of
`run.sh`; each does
`sys.path.insert(0, "/run/media/system/Games_Data/Add-Non-Steam-Games-Linux")`
— a hardcoded ROG-checkout path that does **not** exist on this host, so imports
currently resolve to this checkout. If that path ever exists, they would import
the wrong copy. They are untested — see the P2 backlog item in
MODERNIZATION-TODO.md.

## Further documentation

| Document | Contents |
|----------|----------|
| [AGENTS.md](AGENTS.md) | Project addendum for agents: commands, conventions, gotchas |
| [MODERNIZATION-TODO.md](MODERNIZATION-TODO.md) | P0/P1/P2 modernization backlog (audit 2026-08-24) |
| [docs/archive/POSEIDON-NOTES.md](docs/archive/POSEIDON-NOTES.md) | Engineering notes: architecture, constraints, the `.env.example` gap |
| [docs/archive/SESSION-LEARNINGS.md](docs/archive/SESSION-LEARNINGS.md) | Session learnings: manifest design, relocation, safety guards |

`docs/archive/` is a historical record, not a TODO list — do not execute items
found there.
