#!/bin/bash
# Install (or remove) the HackForger sshd drop-in on prod ECS.
#
# Background: enabling Forgejo SSH (DISABLE_SSH=false) requires sshd to also
# read /var/lib/hackforger/.ssh/authorized_keys (where Forgejo writes, because
# the gitea systemd unit overrides HOME and uses ProtectHome=yes). This script
# scp's deploy/ecs/sshd-99-hackforger.conf to /etc/ssh/sshd_config.d/, validates
# with `sshd -t`, then `systemctl reload ssh`. Reload preserves existing
# connections, so a bad config does not lock out the current ssh session.
#
# Usage:
#   bash deploy/ecs/install-sshd-config.sh             # install or update
#   bash deploy/ecs/install-sshd-config.sh --rollback  # remove + reload
#
# Inputs:  .env on Mac with ECS_SUDO_PASS=...

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO"

ECS=hackforger@203.119.115.130
CONF_LOCAL="deploy/ecs/sshd-99-hackforger.conf"
CONF_REMOTE_DEST="/etc/ssh/sshd_config.d/99-hackforger.conf"

ROLLBACK=0
case "${1:-}" in
  --rollback) ROLLBACK=1 ;;
  "") ;;
  *) echo "Usage: $0 [--rollback]" >&2; exit 64 ;;
esac

if ! grep -q "^ECS_SUDO_PASS=" .env; then
  echo "FATAL: ECS_SUDO_PASS not in .env" >&2
  exit 1
fi
SUDO_PASS=$(grep "^ECS_SUDO_PASS=" .env | cut -d= -f2-)

log() { printf "\n\033[1;34m▶ %s\033[0m\n" "$*"; }
ok()  { printf "  \033[32m✓\033[0m %s\n" "$*"; }

if [ "$ROLLBACK" = "1" ]; then
  log "Rollback: remove $CONF_REMOTE_DEST + reload sshd"
  ssh "$ECS" "SUDO_PASS='$SUDO_PASS' DEST='$CONF_REMOTE_DEST' bash -s" <<'REMOTE'
set -euo pipefail
echo "$SUDO_PASS" | sudo -S -v
( while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null ) &
KEEPER=$!; trap 'kill $KEEPER 2>/dev/null || true' EXIT

if [ -f "$DEST" ]; then
  sudo rm "$DEST"
  if sudo sshd -t; then
    sudo systemctl reload ssh
    echo "  ✓ removed and sshd reloaded"
  else
    echo "  ✗ sshd -t FAILED after removal — leaving sshd as-is" >&2
    exit 1
  fi
else
  echo "  (file did not exist; nothing to do)"
fi
REMOTE
  exit 0
fi

# Install path
test -f "$CONF_LOCAL" || { echo "FATAL: $CONF_LOCAL not found" >&2; exit 1; }

log "[1/3] scp $CONF_LOCAL → $ECS:/tmp/"
TMP_REMOTE="/tmp/sshd-99-hackforger.conf.$$"
scp -q "$CONF_LOCAL" "$ECS:$TMP_REMOTE"
ok "staged at $TMP_REMOTE"

log "[2/3] sudo install + sshd -t + reload (preserves current connections)"
ssh "$ECS" "SUDO_PASS='$SUDO_PASS' TMP='$TMP_REMOTE' DEST='$CONF_REMOTE_DEST' bash -s" <<'REMOTE'
set -euo pipefail
echo "$SUDO_PASS" | sudo -S -v
( while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null ) &
KEEPER=$!; trap 'kill $KEEPER 2>/dev/null || true' EXIT

# Backup if existing
if [ -f "$DEST" ]; then
  BAK="$DEST.bak.$(date +%Y%m%d-%H%M%S)"
  sudo cp -p "$DEST" "$BAK"
  echo "  backed up existing → $BAK"
fi

# Install (644 root:root is sshd_config.d standard)
sudo install -m 644 -o root -g root "$TMP" "$DEST"
sudo rm -f "$TMP"
echo "  installed → $DEST"

# Validate before reload
if sudo sshd -t; then
  echo "  ✓ sshd -t passed"
else
  echo "  ✗ sshd -t FAILED — removing $DEST and aborting (sshd not reloaded)" >&2
  sudo rm "$DEST"
  exit 1
fi

# Reload (preserves existing connections; bad config = reload fails but old config keeps serving)
sudo systemctl reload ssh
sleep 1
sudo systemctl status ssh --no-pager -n 0 | head -3

echo
echo "  effective AuthorizedKeysFile for hackforger user:"
sudo sshd -T -C "user=hackforger,host=,addr=" 2>/dev/null | grep -i '^authorizedkeysfile' | sed 's/^/    /'
REMOTE

log "[3/3] Verify by opening a fresh ssh connection"
if ssh -o ConnectTimeout=8 -o BatchMode=yes "$ECS" 'echo POST_RELOAD_OK' >/tmp/.sshd-install-verify.$$ 2>&1; then
  cat /tmp/.sshd-install-verify.$$
  ok "fresh ssh login still works"
else
  echo "  ⚠ fresh ssh login FAILED. Output:" >&2
  cat /tmp/.sshd-install-verify.$$ >&2
  echo "  ⚠ This may NOT be a real failure if the SSH agent didn't offer the right key." >&2
  echo "  ⚠ Try manually: ssh hackforger@203.119.115.130 echo OK" >&2
  echo "  ⚠ If it really is broken, rollback with:" >&2
  echo "    bash deploy/ecs/install-sshd-config.sh --rollback" >&2
fi
rm -f /tmp/.sshd-install-verify.$$

echo
ok "Install complete."
