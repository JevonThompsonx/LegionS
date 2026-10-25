# Migration Runbook: ROG Ally (Bazzite) -> Lenovo Legion Go S (Z2 Go)

Drive: 4 TB NVMe TEAM TM8FPW004T. Both partitions retained:
  - OS: btrfs (Bazzite bazzite-deck-gnome, F45) ~197 GB
  - Data: ext4 "Games Data" ~3.5 TB (auto-mounted)

## Pre-flight (done offline on roadman, 2026-10-25)
- SMART: PASSED, 0% used, 0 media errors, 45 C.
- Legacy hhd (ROG Ally) state reset -> *.rog-ally-backup; hhd-ui cache removed.
- Hostname set to legion-go-s.
- Kernel args retained (harmless): bluetooth.disable_ertm=1, amdgpu.ppfeaturemask=0xfff7ffff.

## 1. Physical install
- Power off Legion Go S, remove stock 512 GB M.2.
- Install this 4 TB NVMe (verify 2280 single-sided fit). Reassemble.

## 2. BIOS
- Enter BIOS (Volume Up + Power / F2).
- Disable Secure Boot.
- Set boot order: internal NVMe first. Save & exit.

## 3. First boot
- greenboot health check may take a few minutes.
- On failure, grub menu offers rollback deploy (ostree-2) for recovery.

## 4. Verify core paths
  ls -l ~/Games                      # -> /run/media/system/Games_Data
  ls /run/media/system/Games_Data
  df -h /run/media/system/Games_Data
  Steam -> confirm library folders under Games_Data.

## 5. Controllers
- inputplumber auto-detects Legion Go S via 50-legion_go_s.yaml.
- If dead: `sudo systemctl restart inputplumber`; check journalctl.
- Steam Input may need re-mapping.

## 6. Display / Game Mode
- Panel 8" 1920x1200 (16:10); gamescope auto-detects.
- Set in Steam -> Settings -> Display if wrong.

## 7. Sunshine (streaming) — re-pair
- Host IP changed; old pairings invalid.
- Regenerate certs if needed; pair clients against new IP.

## 8. Syncthing (save sync)
- Device cert unchanged; peers roadman/templetonbak still valid.
- Verify `systemctl --user status syncthing` and peer connectivity.
- Folder path ~/GameSaves unchanged.

## 9. Optional: rebase to a Legion-specific Bazzite image
- Current image bazzite-deck-gnome is handheld-generic and works.
- `rpm-ostree rebase ostree-image-signed:docker://ghcr.io/ublue-os/bazzite-deck-gnome:latest`

## 10. Troubleshooting
- hhd legacy state reset; hhd-ui regenerates defaults if installed.
- Journal persistent logs: /var/log/journal (500M cap).
- Rollback: select previous deploy in grub menu.
- Restore ROG hhd state from /etc/hhd/*.rog-ally-backup.
