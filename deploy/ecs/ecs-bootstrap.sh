#!/bin/bash
# Idempotent bootstrap of HackForger on the Huawei ECS.
# Run as the hackforger user on 203.119.115.130. Will prompt for sudo password.
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

# Avoid locale-not-found noise from non-interactive SSH sessions
export LANG=C.UTF-8 LC_ALL=C.UTF-8

ACME_EMAIL="${ACME_EMAIL:-ops@synnovator.com}"
RUNNER_VERSION="${RUNNER_VERSION:-v12.9.0}"
CADDY_VERSION="${CADDY_VERSION:-v2.8.4}"
PREBUILT_DIR="${PREBUILT_DIR:-/tmp/binaries-prebuilt}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { printf "\n\033[1;34m▶ %s\033[0m\n" "$*"; }
ok()  { printf "  \033[32m✓\033[0m %s\n" "$*"; }
skip() { printf "  \033[33m·\033[0m %s (skipped — already done)\n" "$*"; }

if [ "$(id -un)" != "hackforger" ]; then
  echo "FATAL: must run as user 'hackforger', got '$(id -un)'" >&2
  exit 1
fi

# Cache sudo creds once at start so the rest is non-interactive.
# Supports SUDO_PASS env var for non-TTY (SSH) invocation; falls back to
# interactive `sudo -v` when run from a TTY.
log "[0/9] Caching sudo credentials"
if [ -n "${SUDO_PASS:-}" ]; then
  echo "$SUDO_PASS" | sudo -S -v
else
  sudo -v
fi
( while true; do sudo -n true; sleep 60; kill -0 "$$" || exit; done 2>/dev/null ) &
SUDO_KEEPER=$!
trap 'kill $SUDO_KEEPER 2>/dev/null || true' EXIT

# ---------------------------------------------------------------------------
# Pre-flight fixups: silence sudo hostname warnings + ensure en_US.UTF-8 locale
# is generated. Both prevent failures later (postgresql cluster init refuses
# to bootstrap when LC_TIME is set to an ungenerated locale, e.g. Mac iTerm
# leaks LC_TIME=en_DK.UTF-8 over SSH).
log "[0.5/9] Pre-flight: hostname + locale"
if ! grep -q "^127\.0\.1\.1\s\+$(hostname)" /etc/hosts; then
  echo "127.0.1.1 $(hostname)" | sudo tee -a /etc/hosts >/dev/null
  ok "added 127.0.1.1 $(hostname) to /etc/hosts"
else
  skip "hostname entry exists in /etc/hosts"
fi
if locale -a 2>/dev/null | grep -qi "en_US.utf8"; then
  skip "en_US.UTF-8 locale already generated"
else
  sudo apt-get install -y -qq --no-install-recommends locales
  sudo locale-gen en_US.UTF-8 C.UTF-8
  sudo update-locale LANG=C.UTF-8 LC_ALL= 2>/dev/null || true
  ok "en_US.UTF-8 + C.UTF-8 locales generated"
fi

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
log "[2/9] apt install: postgresql-18 + deps (caddy installed via binary, see step 3)"
sudo apt-get install -y -qq \
  postgresql-18 postgresql-client-18 \
  rsync curl jq
ok "apt installs done"

# ---------------------------------------------------------------------------
log "[3/9] Installing Caddy binary"
if command -v caddy >/dev/null 2>&1 && caddy version 2>/dev/null | grep -qF "${CADDY_VERSION#v}"; then
  skip "caddy ${CADDY_VERSION} already installed"
elif [ -x "$PREBUILT_DIR/caddy" ]; then
  sudo install -m 755 "$PREBUILT_DIR/caddy" /usr/local/bin/caddy
  ok "caddy installed from prebuilt: $(caddy version | head -1)"
else
  ARCH=$(dpkg --print-architecture)
  TARBALL="caddy_${CADDY_VERSION#v}_linux_${ARCH}.tar.gz"
  URL="https://github.com/caddyserver/caddy/releases/download/${CADDY_VERSION}/${TARBALL}"
  TMP=$(mktemp -d)
  echo "  downloading from $URL (this may be slow from CN; consider scp'ing the binary to $PREBUILT_DIR/caddy)"
  curl -fsSL --connect-timeout 10 --max-time 600 "$URL" -o "$TMP/caddy.tgz"
  tar xzf "$TMP/caddy.tgz" -C "$TMP" caddy
  sudo install -m 755 "$TMP/caddy" /usr/local/bin/caddy
  rm -rf "$TMP"
  ok "caddy installed: $(caddy version | head -1)"
fi

if ! id caddy >/dev/null 2>&1; then
  sudo groupadd --system caddy
  sudo useradd --system --gid caddy \
    --home-dir /var/lib/caddy --create-home \
    --shell /usr/sbin/nologin caddy
  ok "caddy user created"
else
  skip "caddy user exists"
fi

if [ ! -d /etc/caddy ]; then
  sudo install -d -o root -g root -m 755 /etc/caddy
  ok "/etc/caddy created"
else
  skip "/etc/caddy exists"
fi

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
if command -v forgejo-runner >/dev/null && forgejo-runner --version 2>/dev/null | grep -qF "${RUNNER_VERSION#v}"; then
  skip "forgejo-runner $RUNNER_VERSION already installed"
elif [ -x "$PREBUILT_DIR/forgejo-runner" ]; then
  sudo install -m 755 "$PREBUILT_DIR/forgejo-runner" /usr/local/bin/forgejo-runner
  ok "forgejo-runner installed from prebuilt: $(forgejo-runner --version 2>&1 | head -1)"
else
  ARCH=$(dpkg --print-architecture)
  if [ "$ARCH" = "amd64" ]; then RUNNER_ARCH=amd64; else RUNNER_ARCH=arm64; fi
  sudo curl -fsSL --connect-timeout 10 --max-time 600 \
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
log "[8/9] Installing systemd units + backup script + app.ini template + sshd drop-in"
for unit in gitea.service forgejo-runner.service caddy.service \
            hackforger-backup.service hackforger-backup.timer; do
  sudo cp "$SCRIPT_DIR/systemd/$unit" "/etc/systemd/system/$unit"
done
sudo cp "$SCRIPT_DIR/backup-ecs.sh" /usr/local/bin/hackforger-backup.sh
sudo chmod +x /usr/local/bin/hackforger-backup.sh
sudo cp "$SCRIPT_DIR/app.ini.tmpl" /opt/hackforger/app.ini.tmpl
sudo chown hackforger:hackforger /opt/hackforger/app.ini.tmpl

# sshd drop-in: makes sshd read both /home/hackforger/.ssh/authorized_keys (operator
# shell-login keys) and /var/lib/hackforger/.ssh/authorized_keys (Forgejo-managed
# keys with command="gitea serv ..." prefix). Required because the gitea systemd
# unit overrides HOME=/var/lib/hackforger + ProtectHome=yes, so Forgejo writes
# authorized_keys outside the OS user's home where sshd looks by default.
if [ -f "$SCRIPT_DIR/sshd-99-hackforger.conf" ]; then
  sudo install -m 644 -o root -g root "$SCRIPT_DIR/sshd-99-hackforger.conf" \
       /etc/ssh/sshd_config.d/99-hackforger.conf
  if sudo sshd -t; then
    sudo systemctl reload ssh
    ok "sshd drop-in installed and ssh reloaded"
  else
    echo "  WARN: sshd -t failed after installing drop-in — removed it" >&2
    sudo rm /etc/ssh/sshd_config.d/99-hackforger.conf
  fi
else
  echo "  skip: $SCRIPT_DIR/sshd-99-hackforger.conf not present (older bootstrap?)"
fi

sudo systemctl daemon-reload
sudo systemctl enable hackforger-backup.timer >/dev/null 2>&1 || true
sudo systemctl start hackforger-backup.timer
sudo systemctl enable caddy >/dev/null 2>&1 || true
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
echo "  2. From Mac: scp gitea-linux-amd64 hackforger@203.119.115.130:/opt/hackforger/gitea"
echo "  3. From Mac: bash deploy/ecs/migrate-data.sh --confirm"
echo "  4. On ECS:   sudo systemctl start gitea caddy"
ok "Bootstrap complete"
