#!/bin/bash
# Nightly backup of HackForger DB. Invoked by systemd timer
# /etc/systemd/system/hackforger-backup.timer.
#
# - Dumps the Postgres `hackforger` DB to /var/lib/hackforger/pg-backups/
# - Compressed custom format (-Fc)
# - Filename includes UTC ISO timestamp for natural sort
# - Retains 14 days, deletes older
#
# stdout/stderr go to journald via systemd.

set -euo pipefail

BACKUP_DIR=/var/lib/hackforger/pg-backups
RETAIN_DAYS=14
STAMP=$(date -u +%Y%m%d-%H%M%SZ)
OUT="$BACKUP_DIR/hf-${STAMP}.pgc"

mkdir -p "$BACKUP_DIR"

# Read PG password from the file root saved during bootstrap. The hackforger
# user has a sudoers carve-out for just this single read.
PGPASSWORD=$(sudo cat /root/.hackforger-pg-password) \
  pg_dump -Fc -h 127.0.0.1 -U hackforger hackforger \
  > "$OUT"

SIZE=$(stat -c%s "$OUT")
printf "backup-ecs OK: %s (%d bytes)\n" "$OUT" "$SIZE"

DELETED=$(find "$BACKUP_DIR" -name 'hf-*.pgc' -mtime "+$RETAIN_DAYS" -delete -print | wc -l)
printf "rotation: deleted %d files older than %d days\n" "$DELETED" "$RETAIN_DAYS"
