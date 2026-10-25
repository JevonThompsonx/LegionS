# rog — agent notes

Go gamepack for the Bazzite handheld (`rog`). Main @4189ee5 merged
`feature/modernization` (vet ok, gofmt clean, `-race` ok, coverage PASS).

## Layout

- `gamepack/` — Go module (bubbletea v1.3.4, lipgloss v1.1.0 TUI/CLI).
- `go.work` — workspace, go 1.26.4, `use ./gamepack`.
- `Makefile` — thin root wrapper delegating to `gamepack/`; CI exercises it.

## Gates (run from `gamepack/`)

- Build: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath ./cmd/gamepack`
- Test: `go test ./internal/... -race -cover`
- Coverage: `bash scripts/coverage-check.sh` — floor ≥40%, exempt: proton|state|unpack
- Lint: `go vet ./...` + `gofmt -l` must be empty (`make lint` = vet+test+coverage)
- CI: `.github/workflows/ci.yml` (test/vet/gofmt/coverage)

## Docs

- `INDEX.md` — full file/directory catalog (human). `MANIFEST.json` — machine-readable file manifest (paths, sizes, sha256).
- `README.md` (49 lines) — machine reference + layout table; start here.
- `ARCHITECTURE.md` — machines, Tailscale IPs (dynamic, reverify), Steam paths.
- `ROG_DRIVE.md` — old Micron 2400 ROG drive on bazzite-mobile (data ref).
- `docs/README.md` — index for `docs/` (gamepack-plan, display-blackscreen).
- `docs/gamepack-plan.md` — gamepack design/plan.
- `docs/display-blackscreen.md` — display troubleshooting.
- `state/CURRENT.md`, `migration/{STATUS,DRIVE_EXPANSION}.md`, `gamepack/README.md` — current-state and migration records; `README.md` layout table links them.
- Data-dir notes (dated machine-state snapshots, do not rewrite as current): `games/{SHORTCUTS,UNLAUNCHED}.md`, `saves/SAVES.md`, `steam/CONFIG.md`.
- `TODO.md` (machine-ops) vs `MODERNIZATION-TODO.md` (gamepack audit) — disjoint, keep both.

## Known gaps (do not silently "fix")

- `gamepack/internal/state/journal.go` — dead code, 0 callers; TUI claims journaling.
- Restore verifies checksum on `game.tar.zst` only.
- `gamepack/internal/proton/proton.go` hardcodes `/var/home/jevonx/.steam/`.
- `gamepack/Makefile deploy` runs `scp gamepack rog:/tmp/gamepack.new` (hostname hardcoded).

## Branches / worktrees

- `wt/rog-deps-20260829` @8d7df6f (non-breaking dep updates) preserved; merge decision: owner.
- Worktree `t_43608d1e` removed 2026-09-03; keep `git worktree list` single (main only).

Last verified: 2026-10-15 (docs organized: `README.md` + `docs/README.md` index; this file's stale line count + docs index fixed as root). No push.

## Migration

Retargeted to Lenovo Legion Go S (Z2 Go) on 2026-10-25. ROG Ally retired. See migration/LEGION-GO-S.md.
