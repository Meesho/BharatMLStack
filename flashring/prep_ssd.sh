#!/usr/bin/env bash
# Mount all non-root NVMe SSDs (/dev/nvme*n1) as ext4 under /mnt/localssd1, /mnt/localssd2, ...
# Uses hourly fstrim (systemd timer or cron fallback). Safe to re-run.
set -euo pipefail

MOUNT_BASE="/mnt/localssd"

log() { echo "[$(date +'%F %T')] $*"; }
trap 'log "ERROR: Command failed: $BASH_COMMAND (line $LINENO)"' ERR

# ---------- Helpers ----------
fs_type() { lsblk -ndo FSTYPE "$1" 2>/dev/null | tr -d ' '; }
is_mounted_anywhere() { findmnt -S "$1" >/dev/null 2>&1; }
current_mountpoint() { findmnt -S "$1" -no TARGET 2>/dev/null || true; }

root_source() { findmnt -no SOURCE / 2>/dev/null || true; }
parent_of() {
  local s="$1"
  [[ "$s" =~ ^/dev/nvme[0-9]+n[0-9]+p[0-9]+$ ]] && { echo "${s%p*}"; return; }
  [[ "$s" =~ ^/dev/sd[a-z][0-9]+$ ]] && { echo "${s%[0-9]}"; return; }
  echo "$s"
}
is_boot_dev() {
  local dev="$1"
  local rsrc; rsrc="$(root_source)"
  [[ -z "$rsrc" ]] && return 1
  local rparent; rparent="$(parent_of "$rsrc")"
  [[ "$dev" == "$rsrc" || "$dev" == "$rparent" ]]
}

next_mountpoint() {
  local n=1
  while :; do
    local mp="${MOUNT_BASE}${n}"
    if ! mountpoint -q "$mp"; then
      echo "$mp"
      return 0
    fi
    ((n+=1))
  done
}

ensure_fstab_entry() {
  local uuid="$1" mp="$2"
  local line="UUID=${uuid}  ${mp}  ext4  defaults,nofail,noatime,nodiratime  0  2"
  sed -i -E "/^UUID=${uuid}[[:space:]]/d" /etc/fstab 2>/dev/null || true
  grep -q "UUID=${uuid}  ${mp}  ext4" /etc/fstab 2>/dev/null || echo "$line" >> /etc/fstab
}

sanitize_fstab_discard() {
  if grep -Eq '/mnt/localssd[0-9]+[[:space:]]+ext4' /etc/fstab 2>/dev/null; then
    log "Sanitizing /etc/fstab to remove ',discard' on /mnt/localssd* entries"
    sed -i -E '/\/mnt\/localssd[0-9]+[[:space:]]+ext4/ s/,?discard//g' /etc/fstab
  fi
}

remount_localssd_no_discard() {
  mapfile -t MPS < <(findmnt -no TARGET | grep -E "^${MOUNT_BASE}[0-9]+$" || true)
  for mp in "${MPS[@]:-}"; do
    log "Remounting $mp without 'discard'"
    mount -o remount,noatime,nodiratime "$mp" || true
  done
}

setup_fstrim_hourly() {
  if command -v systemctl >/dev/null 2>&1 && command -v fstrim >/dev/null 2>&1; then
    log "Configuring systemd fstrim.timer to run hourly"
    mkdir -p /etc/systemd/system/fstrim.timer.d
    cat >/etc/systemd/system/fstrim.timer.d/override.conf <<'EOF'
[Timer]
OnCalendar=hourly
Persistent=true
EOF
    systemctl daemon-reload
    systemctl enable --now fstrim.timer
    systemctl status fstrim.timer --no-pager -l || true
  else
    if command -v fstrim >/dev/null 2>&1; then
      log "Configuring cron.hourly for fstrim (systemd not available)"
      mkdir -p /etc/cron.hourly
      cat >/etc/cron.hourly/fstrim-localssd <<'EOF'
#!/bin/sh
/sbin/fstrim --all --quiet || /usr/sbin/fstrim --all --quiet || true
EOF
      chmod +x /etc/cron.hourly/fstrim-localssd
    else
      log "WARN: fstrim not found; install util-linux to enable trimming."
    fi
  fi
}

# ---------- Modes ----------
umount_mode() {
  log "Unmounting /mnt/localssd* and cleaning /etc/fstab entries"
  mapfile -t MPS < <(findmnt -no TARGET | grep -E "^${MOUNT_BASE}[0-9]+$" || true)
  for mp in "${MPS[@]:-}"; do
    log "Umount $mp"
    umount "$mp" || true
  done
  sed -i -E '/\/mnt\/localssd[0-9]+[[:space:]]+ext4/d' /etc/fstab || true
  systemctl daemon-reload || true
  log "Done. Re-run this script to mount afresh."
  exit 0
}

status_mode() {
  log "Current localssd mounts:"
  findmnt -no TARGET,SOURCE,FSTYPE | grep -E "^${MOUNT_BASE}[0-9]+" || echo "None"
  exit 0
}

usage() {
  echo "Usage: $0 [--umount|--status]"
  exit 1
}

case "${1:-}" in
  --umount) umount_mode ;;
  --status) status_mode ;;
  "") ;;
  *) usage ;;
esac

# ---------- Preconditions ----------
command -v lsblk >/dev/null || { echo "lsblk not found"; exit 1; }
command -v blkid >/dev/null || { echo "blkid not found"; exit 1; }
command -v mkfs.ext4 >/dev/null || { echo "mkfs.ext4 not found"; exit 1; }

# Sync systemd with current fstab before mounts
systemctl daemon-reload || true

# Enumerate all NVMe namespaces deterministically
mapfile -t NVME_DEVS < <(ls /dev/nvme*n1 2>/dev/null | sort || true)
if [[ ${#NVME_DEVS[@]} -eq 0 ]]; then
  log "No NVMe namespaces (/dev/nvme*n1) found."
  exit 0
fi
log "Scanning devices: ${NVME_DEVS[*]}"

processed=0

for dev in "${NVME_DEVS[@]}"; do
  [[ -e "$dev" ]] || { log "$dev not found at runtime — skipping."; continue; }

  if is_boot_dev "$dev"; then
    log "Skipping boot/root device: $dev"
    continue
  fi

  log "Found $dev"

  if is_mounted_anywhere "$dev"; then
    mp_now="$(current_mountpoint "$dev")"
    log "$dev already mounted at $mp_now — leaving as-is."
    uuid="$(blkid -s UUID -o value "$dev" || true)"
    [[ -n "$uuid" ]] && ensure_fstab_entry "$uuid" "$mp_now"
    ((processed+=1))
    continue
  fi

  fstype="$(fs_type "$dev")"
  if [[ -z "$fstype" ]]; then
    log "Formatting $dev as ext4"
    mkfs.ext4 -F -m 0 -E lazy_itable_init=1,lazy_journal_init=1 "$dev"
    fstype="ext4"
  else
    log "$dev already has filesystem: $fstype"
  fi

  if [[ "$fstype" != "ext4" ]]; then
    log "Skipping $dev (unsupported fs: $fstype)."
    continue
  fi

  mp="$(next_mountpoint)"
  mkdir -p "$mp"
  log "Mounting $dev at $mp (no 'discard')"
  mount -o noatime,nodiratime "$dev" "$mp"

  uuid="$(blkid -s UUID -o value "$dev" || true)"
  if [[ -n "$uuid" ]]; then
    ensure_fstab_entry "$uuid" "$mp"
    systemctl daemon-reload || true
  else
    log "WARN: Could not read UUID for $dev; skipping fstab entry."
  fi

  ((processed+=1))
done

sanitize_fstab_discard
remount_localssd_no_discard

systemctl daemon-reload || true
setup_fstrim_hourly

if [[ "$processed" -eq 0 ]]; then
  log "No devices processed. Existing mounts sanitized and hourly fstrim scheduled (if available)."
else
  log "Done. Processed $processed device(s). Current mounts:"
  findmnt -no TARGET,SOURCE | grep "$MOUNT_BASE" || true
fi
