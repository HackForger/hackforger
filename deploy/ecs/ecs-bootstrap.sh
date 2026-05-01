#!/bin/bash
# Idempotent bootstrap of HackForger on the Huawei ECS.
# Run as the hackforger user on 218.91.114.178. Will prompt for sudo password.
#
# Re-running is safe: each step checks state first.
#
# Inputs (env vars, optional):
#   ACME_EMAIL      — email Caddy gives to Let's Encrypt (default: ops@synnovator.com)
#   RUNNER_VERSION  — forgejo-runner version (default: v0.3.0)
#
# Outputs:
#   Postgres 18 installed and running, hackforger DB+role created
#   /var/lib/hackforger/ directory tree created
#   /opt/hackforger/ ready for the gitea binary (binary itself is scp'd separately)
#   forgejo-runner binary at /usr/local/bin/forgejo-runner
#   Caddy installed (apt) and Caddyfile in place
#   systemd units installed (NOT all started — gitea + runner wait for data + token)
#   /usr/local/bin/hackforger-backup.sh installed
#   sudoers carve-out for hackforger to read /root/.hackforger-pg-password

set -euo pipefail

ACME_EMAIL="${ACME_EMAIL:-ops@synnovator.com}"
RUNNER_VERSION="${RUNNER_VERSION:-v0.3.0}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { printf "\n\033[1;34m▶ %s\033[0m\n" "$*"; }
ok()  { printf "  \033[32m✓\033[0m %s\n" "$*"; }
skip() { printf "  \033[33m·\033[0m %s (skipped — already done)\n" "$*"; }

if [ "$(id -un)" != "hackforger" ]; then
  echo "FATAL: must run as user 'hackforger', got '$(id -un)'" >&2
  exit 1
fi

# Cache sudo creds once at start so the rest is non-interactive
log "[0/9] Caching sudo credentials"
sudo -v
( while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null ) &
SUDO_KEEPER=$!
trap 'kill $SUDO_KEEPER 2>/dev/null || true' EXIT

# ---------------------------------------------------------------------------
log "[1/9] Adding PostgreSQL 18 (PGDG) apt repository"
if [ -f /etc/apt/sources.list.d/pgdg.list ]; then
  skip "pgdg.list present"
else
  sudo install -d /usr/share/postgresql-common/pgdg
  sudo curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc \
    -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc
  echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
    | sudo tee /etc/apt/sources.list.d/pgdg.list >/dev/null
  sudo apt-get update -qq
  ok "pgdg repo added"
fi

# ---------------------------------------------------------------------------
log "[2/9] Adding Caddy apt repository"
if [ -f /etc/apt/sources.list.d/caddy-stable.list ]; then
  skip "caddy-stable.list present"
else
  sudo apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
  curl -fsSL https://dl.cloudsmith.io/public/caddy/stable/gpg.key \
    | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -fsSL https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt \
    | sudo tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
  sudo apt-get update -qq
  ok "caddy repo added"
fi

# ---------------------------------------------------------------------------
log "[3/9] apt install: postgresql-18, caddy, deps"
sudo apt-get install -y -qq \
  postgresql-18 postgresql-client-18 \
  caddy \
  rsync curl jq
ok "apt installs done"

# ---------------------------------------------------------------------------
log "[4/9] Configuring postgres (loopback only, hackforger DB+role)"
PG_CONF=/etc/postgresql/18/main/postgresql.conf

if ! sudo grep -qE "^listen_addresses\s*=\s*'127\.0\.0\.1'" "$PG_CONF"; then
  sudo sed -i "s/^#*listen_addresses\s*=.*/listen_addresses = '127.0.0.1'/" "$PG_CONF"
  sudo systemctl restart postgresql
  ok "PG bound to 127.0.0.1"
else
  skip "listen_addresses already 127.0.0.1"
fi

if sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='hackforger'" | grep -q 1; then
  skip "PG role hackforger exists"
else
  PG_PASS=$(openssl rand -base64 24 | tr -d '+/=' | cut -c1-24)
  sudo -u postgres psql -c "CREATE ROLE hackforger LOGIN PASSWORD '$PG_PASS';"
  sudo -u postgres psql -c "ALTER ROLE hackforger CREATEDB;"
  echo "$PG_PASS" | sudo tee /root/.hackforger-pg-password >/dev/null
  sudo chmod 600 /root/.hackforger-pg-password
  ok "PG role hackforger created (password saved to /root/.hackforger-pg-password)"
fi

if sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='hackforger'" | grep -q 1; then
  skip "PG database hackforger exists"
else
  sudo -u postgres createdb -O hackforger hackforger
  ok "PG database hackforger created"
fi

# ---------------------------------------------------------------------------
log "[5/9] Creating directories /opt/hackforger, /var/lib/hackforger, /var/lib/forgejo-runner"
for d in /opt/hackforger /var/lib/hackforger /var/lib/hackforger/data \
         /var/lib/hackforger/custom /var/lib/hackforger/custom/conf \
         /var/lib/hackforger/log /var/lib/hackforger/pg-backups \
         /var/lib/forgejo-runner /etc/forgejo-runner; do
  if [ -d "$d" ]; then
    skip "$d exists"
  else
    sudo install -d -o hackforger -g hackforger "$d"
    ok "created $d"
  fi
done

if [ ! -d /var/log/caddy ]; then
  sudo install -d -o caddy -g caddy /var/log/caddy
  ok "created /var/log/caddy"
else
  skip "/var/log/caddy exists"
fi

sudo chown -R hackforger:hackforger /opt/hackforger /var/lib/hackforger /var/lib/forgejo-runner

# ---------------------------------------------------------------------------
log "[6/9] Installing forgejo-runner $RUNNER_VERSION"
if command -v forgejo-runner >/dev/null && forgejo-runner --version 2>/dev/null | grep -qF "$RUNNER_VERSION"; then
  skip "forgejo-runner $RUNNER_VERSION already installed"
else
  ARCH=$(dpkg --print-architecture)
  if [ "$ARCH" = "amd64" ]; then RUNNER_ARCH=amd64; else RUNNER_ARCH=arm64; fi
  sudo curl -fsSL \
    "https://code.forgejo.org/forgejo/runner/releases/download/$RUNNER_VERSION/forgejo-runner-${RUNNER_VERSION#v}-linux-$RUNNER_ARCH" \
    -o /usr/local/bin/forgejo-runner
  sudo chmod +x /usr/local/bin/forgejo-runner
  ok "forgejo-runner installed: $(forgejo-runner --version 2>&1 | head -1)"
fi

if [ ! -f /etc/forgejo-runner/config.yml ]; then
  sudo tee /etc/forgejo-runner/config.yml >/dev/null <<'EOF'
log:
  level: info
runner:
  file: /var/lib/forgejo-runner/.runner
  capacity: 2
  timeout: 1h
  fetch_timeout: 5s
  fetch_interval: 2s
  labels:
    - "ubuntu-latest:host"
cache:
  enabled: true
  dir: /var/lib/forgejo-runner/cache
EOF
  sudo chown hackforger:hackforger /etc/forgejo-runner/config.yml
  ok "runner config.yml installed"
else
  skip "runner config.yml exists"
fi

# ---------------------------------------------------------------------------
log "[7/9] Installing Caddyfile"
if [ -f /etc/caddy/Caddyfile ] && [ ! -f /etc/caddy/Caddyfile.bak ]; then
  sudo cp /etc/caddy/Caddyfile /etc/caddy/Caddyfile.bak
fi
sudo sed "s/__ACME_EMAIL__/$ACME_EMAIL/g" "$SCRIPT_DIR/caddy/Caddyfile.tmpl" \
  | sudo tee /etc/caddy/Caddyfile >/dev/null
ok "Caddyfile written (ACME email: $ACME_EMAIL)"

# ---------------------------------------------------------------------------
log "[7.5/9] sudoers carve-out for hackforger to read PG password file"
SUDOERS=/etc/sudoers.d/hackforger-backup
if [ ! -f "$SUDOERS" ]; then
  echo 'hackforger ALL=(root) NOPASSWD: /usr/bin/cat /root/.hackforger-pg-password' \
    | sudo tee "$SUDOERS" >/dev/null
  sudo chmod 440 "$SUDOERS"
  sudo visudo -cf "$SUDOERS"
  ok "sudoers rule installed"
else
  skip "sudoers rule exists"
fi

# ---------------------------------------------------------------------------
log "[8/9] Installing systemd units + backup script + app.ini template"
for unit in gitea.service forgejo-runner.service \
            hackforger-backup.service hackforger-backup.timer; do
  sudo cp "$SCRIPT_DIR/systemd/$unit" "/etc/systemd/system/$unit"
done
sudo cp "$SCRIPT_DIR/backup-ecs.sh" /usr/local/bin/hackforger-backup.sh
sudo chmod +x /usr/local/bin/hackforger-backup.sh
sudo cp "$SCRIPT_DIR/app.ini.tmpl" /opt/hackforger/app.ini.tmpl
sudo chown hackforger:hackforger /opt/hackforger/app.ini.tmpl
sudo systemctl daemon-reload
sudo systemctl enable hackforger-backup.timer >/dev/null 2>&1 || true
sudo systemctl start hackforger-backup.timer
ok "systemd units installed (gitea + runner enabled but NOT started — start after data migration)"

# ---------------------------------------------------------------------------
log "[9/9] Summary"
echo "  Postgres:        $(systemctl is-active postgresql)"
echo "  Caddy:           $(systemctl is-active caddy)"
echo "  Gitea:           NOT STARTED (start after migrate-data.sh)"
echo "  Runner:          NOT STARTED (start after registering with gitea admin token)"
echo "  Backup timer:    $(systemctl is-active hackforger-backup.timer)"
echo
echo "  PG password:     /root/.hackforger-pg-password"
echo "  Custom path:     /var/lib/hackforger/custom"
echo "  Binary path:     /opt/hackforger/gitea (NOT YET PRESENT — scp it next)"
echo
echo "Next steps:"
echo "  1. From Mac: bash deploy/ecs/build-linux.sh"
echo "  2. From Mac: scp gitea-linux-amd64 hackforger@218.91.114.178:/opt/hackforger/gitea"
echo "  3. From Mac: bash deploy/ecs/migrate-data.sh --confirm"
echo "  4. On ECS:   sudo systemctl start gitea caddy"
ok "Bootstrap complete"
