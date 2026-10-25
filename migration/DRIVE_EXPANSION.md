# 4TB Drive Expansion — Full Session Log & Handoff

**Date**: 2026-06-28  
**Status**: ✅ COMPLETE (verified 2026-07-11) — Ally boots from the 4TB drive;
Games_Data mounts 3.5T at /run/media/system/Games_Data; btrfs root (UUID 6b91a72a) healthy.
Rest of this doc is the historical session log.  
**Pickup point**: n/a — done

---

## Context

bazzite-mobile (100.119.216.62) has two external USB spinning HDDs:

| Device | Size | Was | Purpose |
|--------|------|-----|---------|
| sdd    | 1.8T | sdb (renamed when port changed mid-session) | Source: ROG Ally's 2TB drive data |
| sdc    | 3.7T | sdc | Destination: 4TB drive for Ally |

**Goal**: Expand sdc1 (Games_Data ext4) from 1.6T → 3.5T, verify Ally boots from sdc.

**Note on sdb→sdd rename**: Drive physically moved to different USB port. Linux renumbered it sdd. All commands going forward must use sdd, not sdb.

---

## What Was Completed

### Partition Rearrangement
Free space was at END of disk; sdc1 (games) is the FIRST partition. Moved sdc2/3/4 to end of disk using within-disk dd copies, then extended sdc1.

**Final partition table on sdc:**
```
sdc1  0–3614GiB    3614GiB  ext4   Games_Data (was 1663GiB)  ← EXPANDED ✓
      3614–3615G   1GiB     —      alignment gap (intentional, normal)
sdc2  3615–3616G   1GiB     fat32  EFI  boot,esp
sdc3  3616–3618G   2GiB     ext4   boot  bls_boot
sdc4  3618–3815G   197GiB   btrfs  bazzite OS
```

`df -h /var/mnt/games-data` confirmed: 3.5T total, 1.3T used, 2.1T free ✓

### First Boot Test — FAILED
Moved sdc to ROG Ally as sole drive. System entered emergency mode.

**Root cause**: btrfs on sdc4 is corrupted.
```
BTRFS error (device sdc4): bad tree block start, mirror 1 want 29093855232 have 0
BTRFS error (device sdc4): Transaction aborted (error -5)
BTRFS info  (device sdc4): forced readonly
```

Kernel + initramfs found sdc4 by UUID correctly, mounted root, but btrfs went forced-readonly immediately. Systemd failed to write /var /run /tmp → emergency mode. **Bootloader (grub/BLS) is NOT the issue.**

**Corruption likely pre-existed** in the Rescuezilla copy. The within-disk dd propagated it.

---

## Currently Running

```bash
sudo dd if=/dev/sdd4 of=/dev/sdc4 bs=4M conv=noerror,sync status=progress
```

Fresh copy of sdd4 (healthy 2TB source) → sdc4 (new position at 3618GiB on sdc).
Speed: ~20-27MB/s (both USB spinning HDDs, likely shared USB controller).
Size: 197GiB. Do not interrupt.

---

## UUID & PARTUUID Reference

### Filesystem UUIDs (preserved by dd — these are correct and stable)
| Partition | Filesystem UUID |
|-----------|-----------------|
| sdc1 ext4 Games_Data | f3d4f998-421b-49da-aa8e-87c1e2594906 ⚠ DUPLICATE with sdd1 |
| sdc2 fat32 EFI | 313B-603E |
| sdc3 ext4 boot | 84c080bf-7435-4ea9-a533-1995825ae94c |
| sdc4 btrfs OS | 6b91a72a-a9fc-4c00-acbe-cf2229a903e0 (same after fresh copy from sdd4) |

### PARTUUIDs (CHANGED — sdc2/3/4 were deleted and recreated)
| Partition | OLD PARTUUID (sdd = original source) | NEW PARTUUID (sdc after changes) |
|-----------|--------------------------------------|----------------------------------|
| sdc2 EFI  | 514948a7-555d-4ff7-8721-eb681b0429d0 | 18e801d6-a822-4628-be61-861759954a9e |
| sdc3 boot | a934eceb-bc9e-42a8-8841-56ca282b6823 | 96f9a874-608e-420f-940d-dd59624c84cd |
| sdc4 OS   | 72edb1ef-3e2e-411e-82d9-a2e6d952cda5 | 98260448-002d-4f3e-8489-bc7480f72aa7 |
| sdc1 data | 851e0e33-2347-493f-b7ea-6c372f01b2a9 | 851e0e33-... (UNCHANGED — resizepart preserves PARTUUID) |

### Boot Configuration (already verified — no changes needed)
- **BLS entries** (sdc3:/loader.1/entries/ostree-*.conf): use `root=UUID=6b91a72a` — correct, filesystem UUID preserved ✓
- **Grub** (sdc2 EFI): uses `search --fs-uuid` for sdc3 — correct, UUID 84c080bf preserved ✓
- **btrfs subvolumes**: ID 256 = root, ID 257 = home, ID 258 = var
- **Kernel cmdline**: `rootflags=subvol=root amd_iommu=off bluetooth.disable_ertm=1`

---

## Post-DD Steps (run these after dd completes)

### 1. Verify btrfs health
```bash
sudo btrfs check --readonly /dev/sdc4
```
Expected: no errors. If errors remain, first check source:
```bash
sudo btrfs check --readonly /dev/sdd4
```
If sdd4 also has errors, the corruption is on the original drive — escalate.

### 2. Mount OS partition and locate fstab
```bash
sudo mkdir -p /mnt/sdc4
sudo mount /dev/sdc4 /mnt/sdc4 -o subvol=root
sudo find /mnt/sdc4 -maxdepth 10 -name fstab 2>/dev/null
```
On Bazzite (ostree), fstab is inside the deployment, not at btrfs root. Path will be under:
`/mnt/sdc4/ostree/deploy/default/deploy/<hash>/etc/fstab`

Display it:
```bash
sudo find /mnt/sdc4 -maxdepth 10 -name fstab 2>/dev/null | xargs sudo cat
```

### 3. Check for stale PARTUUID references
```bash
sudo grep -r PARTUUID /mnt/sdc4/ostree/deploy/ 2>/dev/null
```

**If fstab uses PARTUUID= for /boot or /boot/efi**, patch with:
```bash
FSTAB=$(sudo find /mnt/sdc4 -maxdepth 10 -name fstab 2>/dev/null | head -1)
sudo sed -i \
  -e 's/514948a7-555d-4ff7-8721-eb681b0429d0/18e801d6-a822-4628-be61-861759954a9e/g' \
  -e 's/a934eceb-bc9e-42a8-8841-56ca282b6823/96f9a874-608e-420f-940d-dd59624c84cd/g' \
  -e 's/72edb1ef-3e2e-411e-82d9-a2e6d952cda5/98260448-002d-4f3e-8489-bc7480f72aa7/g' \
  "$FSTAB"
sudo cat "$FSTAB"
```

**If fstab uses UUID= only** (likely — Fedora/Bazzite default), no changes needed.

### 4. Fix duplicate UUID on sdc1 (recommended)
sdc1 and sdd1 both have UUID f3d4f998. Safe on the Ally alone, but causes mount ambiguity when both drives are on bazzite-mobile.

```bash
# New UUID for sdc1
sudo tune2fs -U random /dev/sdc1
NEW_UUID=$(sudo blkid -s UUID -o value /dev/sdc1)
echo "sdc1 new UUID: $NEW_UUID"

# Update fstab on sdc4 (the Ally's OS)
FSTAB=$(sudo find /mnt/sdc4 -maxdepth 10 -name fstab 2>/dev/null | head -1)
sudo sed -i "s/f3d4f998-421b-49da-aa8e-87c1e2594906/$NEW_UUID/g" "$FSTAB"

# Update bazzite-mobile's fstab (it also references this UUID for /var/mnt/games-data)
# Note: Bazzite has immutable root but /etc is a writable overlay — this works
sudo sed -i "s/f3d4f998-421b-49da-aa8e-87c1e2594906/$NEW_UUID/g" /etc/fstab
grep games-data /etc/fstab  # verify
```

### 5. Verify NFS entries have nofail
```bash
FSTAB=$(sudo find /mnt/sdc4 -maxdepth 10 -name fstab 2>/dev/null | head -1)
sudo grep nfs "$FSTAB"
```
NFS entries should have `_netdev` and `x-systemd.automount` — lazy mounts, won't block boot. If any NFS line lacks `nofail` AND the Ally often boots without network, add `nofail` to those lines.

### 6. Unmount and test
```bash
sudo umount /mnt/sdc4
```
Move sdc to ROG Ally, boot, verify:
- Games_Data mounts at /var/mnt/games-data
- Steam accessible
- No emergency mode

### 7. If emergency mode persists on Ally
In emergency mode shell:
```bash
systemctl list-units --state=failed
journalctl -xb -p err | head -80
cat /etc/fstab
```
Paste output for diagnosis.

---

## Issues Table

| Issue | Severity | Action |
|-------|----------|--------|
| sdc4 btrfs corruption | CRITICAL | FIXING NOW via dd |
| sdc1/sdd1 duplicate UUID | MEDIUM | Fix in Step 4 |
| fstab may have stale PARTUUIDs | MEDIUM | Check in Step 3 |
| NFS mounts fail without network | LOW | Verify have nofail/automount flags |

---

## bazzite-mobile Impact
After this work, bazzite-mobile's /var/mnt/games-data currently mounts sdd1 (2TB) because UUID conflict resolved to sdd first. After Step 4 (new UUID for sdc1), bazzite-mobile's fstab points to sdc1's new UUID — update it to sdd1's UUID if sdd is the intended desktop games drive:

```bash
# If desktop should use sdd1 (2TB) for games:
sudo sed -i "s/$NEW_UUID/f3d4f998-421b-49da-aa8e-87c1e2594906/g" /etc/fstab
# (reverts bazzite-mobile fstab back to original sdd1 UUID)
```

---

## Machine Reference
- **bazzite-mobile** (here): 100.119.216.62
- **rog** (ROG Ally): 100.119.14.48 — SSH: `root@100.119.14.48`
- **sdc** = 4TB drive → going to Ally
- **sdd** = 2TB drive → stays on bazzite-mobile (was sdb)
