# Add-Non-Steam-Games-Linux — Agent Notes

Python+shell tool: scan game dirs, add non-Steam games to
Steam, fetch art via SteamGridDB. Entry: `./run.sh`.

## Commands
- `make help` — list targets (test / lint / clean / install).
- `make test` — `python3 -m pytest -q tests`.
- `make lint` — `shellcheck --shell=bash --external-sources run.sh`.
- `make install` — create `.venv` + install `requirements.txt`.
- `python3 main.py --diagnose` / `--dry-run-check` /
  `--validate-flows` — read-only checks.
- No `--help`, no `argparse`: flags are string-matched in `main()`,
  and no flags at all drops into the interactive menu [0–6].

## Git
- Branch: `master` @ `ae2d1ba`, UNPUSHED (ahead of origin/master by 1;
  the docs/archive move). Do NOT push.
- Push target when clean: `master`.
- `AGENTS.md` (this file) is untracked.

## Gates (must pass before any push)
- `python3 -m pytest -q` — GREEN as of 2026-10-15:
  **78 passed + 18 subtests**, wall clock ~0.4 s.
- `shellcheck run.sh` — clean (exit 0), verified 2026-10-15.
- CI: pytest matrix 3.10–3.13 + shellcheck job.

## Docs layout
- `README.md` (188 lines): purpose, CLI/menu/`.env` tables, manifest
  design, structure, development commands. Source of truth for usage.
- `MODERNIZATION-TODO.md` (214 lines): P0/P1/P2 backlog, dated
  2026-08-24 audit — some items have since landed; read it as a
  backlog, not current state.
- `docs/README.md` — one-line index of the docs above.
- `docs/archive/` holds `POSEIDON-NOTES.md` +
  `SESSION-LEARNINGS.md` (canonical location; root stubs removed).
- `docs/archive/` is RECORD, not action items — do not
  execute items found there.

## Env
- `.python-version` pinned (3.11); runtime floor is 3.10
  (`config.MIN_PYTHON`). Local `.venv`, untracked.
- **No `.env.example`** — removed in commit `e351193`. Create
  `.env` by hand; `.gitignore` still whitelists `.env.example`
  if it ever returns. `.env` is gitignored.
- No `pyproject.toml` (Wave-2 deferred). No `requirements.lock`;
  `requests>=2.33.0` unpinned upper.
- Test fixtures use `/home/jevonx` paths (test-only,
  not prod paths).

## Conventions & gotchas
- High blast radius: anything writing `shortcuts.vdf`. Keep
  backup-before-write and the round-trip integrity tests intact.
- Real runs must have Steam closed — `ensure_ready_to_write` refuses.
- Missing-game cleanup checks `parent.exists()` first so an
  unmounted drive never mass-deletes shortcuts. Do not remove it.
- Runs write `steam-game-manager.log` in the cwd (gitignored).
- `auto-run.py` / `refresh-art.py` are untested non-interactive wrappers
  with a hardcoded `sys.path` to a ROG checkout that does not exist on
  this host. Prefer `run.sh` / `main.py`.
- This host has **no Steam installed**, so `--diagnose` prints
  `Steam path:  not found` and exits 1. Expected, not a bug.

Last verified: 2026-10-15. Suite green. No push.
