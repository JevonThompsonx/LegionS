# proxmox/ — Vendored rog host audit snapshot

The `proxmox` repo remains the source of truth for infrastructure/fleet data; this directory is a convenience snapshot of its `servers/rog` audit.

| Path | Purpose |
|------|---------|
| [rog/audit-2026-08-28.md](rog/audit-2026-08-28.md) | Fleet audit summary for rog (2026-08-28). |
| [rog/snapshots/2026-08-28/](rog/snapshots/2026-08-28/) | Machine-parseable dumps (blkid, df, mounts, systemd, docker, etc.). |

Do not treat as live state; reverify against the proxmox repo. Related: `../ARCHITECTURE.md`, `../state/`.