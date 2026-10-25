# Poseidon Engineering Notes — Add-Non-Steam-Games-Linux

Concise engineering context for agent sessions working in this repository.

## What it does

Interactive Python CLI that scans game directories and adds non-Steam games to
Steam on Linux: writes `shortcuts.vdf` entries per Steam user, downloads grid
artwork from SteamGridDB, tracks added games in a per-user JSON manifest
(`shortcuts-manifest.json`) for missing-game cleanup ([6]) and relocation
detection. Filters Wine prefix system files, overlay injectors, engine data,
and Steam downloading dirs.

- Entry point: `main.py` (menu [0–6]); helpers `auto-run.py`, `refresh-art.py`; launcher `run.sh`.
- Modules: `game_scanner.py`, `shortcut_builder.py`, `vdf_manager.py`,
  `steam_paths.py`, `artwork_manager.py`, `game_manifest.py`, `fixer.py`,
  `config.py`, `logger_setup.py`.
- State written: `shortcuts.vdf` + `shortcuts-manifest.json` under each
  `~/.local/share/Steam/userdata/<id>/config/`, plus grid art in `config/grid/`.
  Backups are made before VDF rewrites.
- Config: optional `.env` next to `main.py` (`STEAMGRIDDB_API_KEY`,
  `SCAN_DIRS`, `STEAM_PATH`). Values may also come from the process environment.
  If no key is configured, artwork actions prompt interactively via getpass and
  **do not persist** the key anywhere.
- Runtime/deps: Python 3.10+; only `vdf` and `requests` (see requirements.txt).
  `run.sh` prefers `.venv/bin/python3` when present.

## Commands

```bash
pip install -r requirements.txt   # deps: vdf, requests
python main.py                    # interactive menu (run.sh wraps this)
python -m pytest tests/           # unit tests (no network / no Steam writes)
python main.py --diagnose         # diagnostics-only flags:
python main.py --dry-run-check    #   also --validate-flows
```

Safety flags exist for non-destructive checking (`--diagnose`,
`--dry-run-check`, `--validate-flows`); real runs modify Steam user config.

## Conventions & constraints

- Never commit `.env` or any API key (gitignored). SteamGridDB key lives only
  in the local `.env`/environment on machines where artwork is downloaded.
- Manifest design rules (atomic temp-file write + `Path.replace()`, chmod
  0600, corrupt-file backup-and-reset) are documented in
  `SESSION-LEARNINGS.md` along with the parent-dir guard that prevents mass
  shortcut deletion when an external drive is unmounted — read it before
  touching `game_manifest.py` or removal logic.
- High blast radius: anything writing `shortcuts.vdf`. Keep the existing
  backup-before-write behavior and round-trip integrity tests intact.
- This checkout was imported from the ROG checkout (`Games_Data`); keep
  runtime state (`__pycache__`, logs, `.env`) out of commits.

## Known doc gap

README's Quick Start says `cp .env.example .env`, but `.env.example` was
intentionally removed from the repo (commit e351193 "cleanup env example").
Create `.env` directly if you need one; the gitignore still whitelists
`.env.example` should it ever return.
