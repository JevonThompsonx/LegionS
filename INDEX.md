# LegionS — Repository Index

Single entry point for navigating the merged LegionS repository. For machine-readable
file listing see [`MANIFEST.json`](MANIFEST.json).

## Top level

| Path | Type | Purpose |
|------|------|---------|
| [README.md](README.md) | doc | Machine reference + layout table. Start here. |
| [AGENTS.md](AGENTS.md) | doc | Agent working notes (gates, known gaps, branches). |
| [ARCHITECTURE.md](ARCHITECTURE.md) | doc | Machines, Tailscale IPs, Steam paths, key filesystem locations. |
| [ROG_DRIVE.md](ROG_DRIVE.md) | doc | Old Micron 2400 ROG drive (bazzite-mobile) data reference. |
| [TODO.md](TODO.md) | doc | Machine-ops task list. |
| [MODERNIZATION-TODO.md](MODERNIZATION-TODO.md) | doc | Gamepack audit/modernization backlog. |
| [Makefile](Makefile) | build | Thin root wrapper delegating to `gamepack/`. |
| [go.work](go.work) | build | Go workspace (`use ./gamepack`). |
| [.editorconfig](.editorconfig) | config | Editor formatting rules. |
| [.gitignore](.gitignore) | config | Git ignore rules. |
| [INDEX.md](INDEX.md) | doc | This index. |
| [MANIFEST.json](MANIFEST.json) | data | Machine-readable file manifest (paths, sizes, sha256). |
| [llms.txt](llms.txt) | doc | LLM crawler entry point (standard llms.txt). |

## Directories

| Path | Purpose | Index |
|------|---------|-------|
| [docs/](docs/) | Display troubleshooting, gamepack plan. | [docs/README.md](docs/README.md) |
| [gamepack/](gamepack/) | Go TUI/CLI game archiver/restorer. | [gamepack/README.md](gamepack/README.md) |
| [games/](games/) | Shortcut and unlaunched-game records. | [games/README.md](games/README.md) |
| [migration/](migration/) | Drive expansion + Legion Go S migration runbook. | [migration/README.md](migration/README.md) |
| [proxmox/](proxmox/) | Vendored rog host audit snapshot. | [proxmox/README.md](proxmox/README.md) |
| [saves/](saves/) | Save-data layout and sync tooling. | [saves/README.md](saves/README.md) |
| [state/](state/) | Current verified machine status. | [state/README.md](state/README.md) |
| [steam/](steam/) | Steam account/config snapshot. | [steam/README.md](steam/README.md) |
| [tools/](tools/) | Auxiliary tooling (non-Steam game adder). | [tools/README.md](tools/README.md) |
| [.github/](.github/) | CI workflows and dependabot config. | — |

## gamepack/ (Go module)

Module root: `gamepack/`. Commands run from `gamepack/`.

| Path | Purpose |
|------|---------|
| [gamepack/cmd/gamepack/main.go](gamepack/cmd/gamepack/main.go) | CLI entry point. |
| [gamepack/internal/](gamepack/internal/) | Packages: catalog, cfg, compress, nswitch, pack, proton, saves, state, tui, unpack. |
| [gamepack/scripts/coverage-check.sh](gamepack/scripts/coverage-check.sh) | Coverage gate (floor ≥40%). |
| [gamepack/Makefile](gamepack/Makefile) | Build/test/coverage/lint/deploy targets. |
| [gamepack/testdata/](gamepack/testdata/) | Test fixtures (shortcuts.vdf). |

## tools/Add-Non-Steam-Games-Linux/ (Python tool)

| Path | Purpose |
|------|---------|
| [tools/Add-Non-Steam-Games-Linux/main.py](tools/Add-Non-Steam-Games-Linux/main.py) | CLI + workflow orchestration. |
| [tools/Add-Non-Steam-Games-Linux/run.sh](tools/Add-Non-Steam-Games-Linux/run.sh) | Preferred launcher (venv bootstrap). |
| [tools/Add-Non-Steam-Games-Linux/README.md](tools/Add-Non-Steam-Games-Linux/README.md) | Usage, CLI/menu, manifest design. |
| [tools/Add-Non-Steam-Games-Linux/AGENTS.md](tools/Add-Non-Steam-Games-Linux/AGENTS.md) | Agent addendum. |
| [tools/Add-Non-Steam-Games-Linux/MODERNIZATION-TODO.md](tools/Add-Non-Steam-Games-Linux/MODERNIZATION-TODO.md) | Backlog (audit 2026-08-24). |
| [tools/Add-Non-Steam-Games-Linux/docs/](tools/Add-Non-Steam-Games-Linux/docs/) | Docs index + archived engineering notes. |
| [tools/Add-Non-Steam-Games-Linux/tests/](tools/Add-Non-Steam-Games-Linux/tests/) | pytest suite. |

## .github/

| Path | Purpose |
|------|---------|
| [.github/workflows/ci.yml](.github/workflows/ci.yml) | CI: test/vet/gofmt/coverage (gamepack) + pytest/shellcheck (tool). |
| [.github/dependabot.yml](.github/dependabot.yml) | Dependabot config. |

## proxmox snapshot files

See [proxmox/README.md](proxmox/README.md) and `proxmox/rog/snapshots/2026-08-28/`.