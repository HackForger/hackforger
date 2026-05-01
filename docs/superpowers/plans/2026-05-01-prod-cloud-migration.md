# Prod Cloud Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move HackForger from a developer Mac to Huawei ECS 218.91.114.178 under `https://www.synnovator.com`, deployed as native binaries managed by systemd.

**Architecture:** Single-host binary deployment. Postgres 18 via PGDG apt repo, gitea cross-compiled on Mac and scp'd, forgejo-runner host mode, Caddy auto-HTTPS via Let's Encrypt. Loopback-binding for the app stack; Caddy is the only public-facing process besides SSH.

**Tech Stack:** Bash, systemd, PostgreSQL 18, Caddy 2, forgejo-runner v0.3.0, launchd (Mac side), `make build` cross-compiled with docker for `linux/amd64`.

**Spec:** [docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md](../specs/2026-05-01-prod-cloud-migration-design.md)

---

## Plan Shape

The work splits into two phases:

- **Phase A — Scaffolding (in repo, reviewable):** Tasks 1–10. Produces every script, template, and runbook needed. All of it lives in `deploy/` (and `scripts/backup-pull.sh` on the Mac side). Single PR. Zero impact on the running Mac instance.

- **Phase B — Live execution (operational):** Tasks 11–15. Runs the scripts produced in Phase A against the real ECS, performs the cutover, verifies the backup pipeline. Gated on Phase A merge + manual prerequisites (security group + DNS + Logto callback URL).

Tasks 1–10 are commit-per-task with frequent small commits; the final commit per task is `git add … && git commit`. A single PR covers all of Phase A. Phase B tasks operate on production and are not committed (they leave artifacts on the ECS).

## Files Created / Modified

| Path | Owner | Purpose |
| ---- | ----- | ------- |
| `deploy/README.md` | Task 1 | Operator entry point: what's in this dir, in what order to run things, links to spec/runbook. |
| `deploy/build-linux.sh` | Task 2 | Mac-side: cross-compile gitea for `linux/amd64` using docker. |
| `deploy/app.ini.tmpl` | Task 3 | Production app.ini template. Placeholders for path, domain, DB password. |
| `deploy/caddy/Caddyfile.tmpl` | Task 4 | Reverse-proxy + auto-HTTPS for `www.synnovator.com`. |
| `deploy/systemd/gitea.service` | Task 5 | systemd unit for gitea. |
| `deploy/systemd/forgejo-runner.service` | Task 5 | systemd unit for the runner. |
| `deploy/systemd/hackforger-backup.service` | Task 5 | One-shot service called by the timer. |
| `deploy/systemd/hackforger-backup.timer` | Task 5 | Daily 04:00 timer. |
| `deploy/ecs-bootstrap.sh` | Task 6 | Idempotent: PG repo, apt installs, dirs, user, sudoers nothing, systemd units installed but not started. |
| `deploy/migrate-data.sh` | Task 7 | Mac-side: dump + tar + rsync. ECS-side: restore. Supports `--dry-run`. |
| `deploy/backup-ecs.sh` | Task 8 | The script the systemd timer runs nightly on the ECS. |
| `scripts/backup-pull.sh` | Task 9 | Mac-side rsync puller. |
| `deploy/launchd/com.h2os.hackforger-backup-pull.plist` | Task 9 | launchd job definition (committed for reference; install copies to `~/Library/LaunchAgents/`). |
| `deploy/cutover-runbook.md` | Task 10 | The T-1d / T-1h / T0 / T+1d operator runbook with exact commands. |
| **PR** | Task 11 | Open PR with all of the above; user reviews & merges. |
| (no repo changes Phase B) | Tasks 12–15 | Execute on ECS / Mac. |

---

## Phase A — Scaffolding (Tasks 1–10)

### Task 1: Create `deploy/` directory + README skeleton

**Files:**
- Create: `deploy/README.md`

- [ ] **Step 1: Create the directory and a placeholder file**

```bash
mkdir -p deploy/{systemd,caddy,launchd}
touch deploy/.gitkeep deploy/systemd/.gitkeep deploy/caddy/.gitkeep deploy/launchd/.gitkeep
```

- [ ] **Step 2: Write `deploy/README.md`**

```markdown
# deploy/ — HackForger production deployment

Artifacts for deploying HackForger to the Huawei Cloud ECS at `218.91.114.178`,
serving `https://www.synnovator.com`.

**Spec:** [`docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md`](../docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md)
**Runbook (cutover):** [`cutover-runbook.md`](cutover-runbook.md)

## Files

| File | Side | Purpose |
| ---- | ---- | ------- |
| `build-linux.sh` | Mac | Cross-compile `gitea` for `linux/amd64` via docker. |
| `ecs-bootstrap.sh` | ECS | Idempotent install of postgres, caddy, runner, dirs, systemd units. |
| `migrate-data.sh` | Mac+ECS | Dump-rsync-restore the post-clean-slate prod DB and data dirs. |
| `backup-ecs.sh` | ECS | Nightly `pg_dump` + 14-day rotation. Runs from systemd timer. |
| `app.ini.tmpl` | ECS | Production `app.ini` template. Bootstrap script renders it. |
| `caddy/Caddyfile.tmpl` | ECS | Caddy config for `www.synnovator.com`. |
| `systemd/*.service` `*.timer` | ECS | systemd units installed by `ecs-bootstrap.sh`. |
| `launchd/com.h2os.hackforger-backup-pull.plist` | Mac | launchd job that pulls backups from ECS. |
| `../scripts/backup-pull.sh` | Mac | What the launchd job invokes. |

## Order of operations (first-time deploy)

1. **Manual prerequisites (you):**
   - Open inbound :80 + :443 in the Huawei Cloud security group for ECS 218.91.114.178.
   - In the Aliyun DNS console: lower `synnovator.com` zone TTL to 60s, add A record `www → 218.91.114.178`.
   - In Logto admin: add `https://www.synnovator.com/user/oauth2/<source-name>/callback` to the allowed callback URLs.
2. `bash deploy/build-linux.sh` on the Mac → produces `gitea-linux-amd64`.
3. `scp gitea-linux-amd64 hackforger@218.91.114.178:/tmp/`.
4. `scp deploy/ecs-bootstrap.sh hackforger@218.91.114.178:/tmp/` and run it (interactive sudo).
5. Run `bash deploy/migrate-data.sh` from the Mac during the maintenance window.
6. Follow `cutover-runbook.md`.

## Subsequent deploys (after first cutover)

1. `bash deploy/build-linux.sh` on the Mac.
2. `scp gitea-linux-amd64 hackforger@218.91.114.178:/opt/hackforger/gitea-new`.
3. `ssh hackforger@218.91.114.178 'sudo mv /opt/hackforger/gitea-new /opt/hackforger/gitea && sudo systemctl restart gitea'`.
```

- [ ] **Step 3: Branch + commit**

```bash
git checkout -b feat/prod-cloud-migration-scaffolding
git add deploy/README.md deploy/.gitkeep deploy/systemd/.gitkeep deploy/caddy/.gitkeep deploy/launchd/.gitkeep
git commit -m "feat(deploy): scaffold deploy/ directory + README"
```

Expected output: branch created, commit succeeds, working tree clean.

---

### Task 2: `deploy/build-linux.sh` (Mac-side cross-compile)

**Files:**
- Create: `deploy/build-linux.sh`
- Modify: nothing else

- [ ] **Step 1: Write the script**

Path: `deploy/build-linux.sh`. Make it idempotent and verbose (operator-facing).

```bash
#!/bin/bash
# Cross-compile gitea for linux/amd64 using docker, without polluting the Mac
# with a Linux Go toolchain.
#
# Output: ./gitea-linux-amd64 in the repo root.
# Usage:  bash deploy/build-linux.sh
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"

OUT="gitea-linux-amd64"
IMAGE="golang:1.23-bookworm"

if ! command -v docker >/dev/null 2>&1; then
  echo "FATAL: docker not found on PATH. Install Docker Desktop." >&2
  exit 1
fi

echo "[1/3] Pulling build image $IMAGE (no-op if cached)..."
docker pull "$IMAGE"

echo "[2/3] Cross-compiling gitea (this takes 3-6 minutes)..."
docker run --rm --platform linux/amd64 \
  -v "$REPO":/src -w /src \
  "$IMAGE" bash -ec '
    apt-get update -qq
    apt-get install -y -qq build-essential nodejs npm git
    git config --global --add safe.directory /src
    TAGS="bindata pam sqlite sqlite_unlock_notify" \
      GOOS=linux GOARCH=amd64 \
      make build
    mv gitea '"$OUT"'
  '

echo "[3/3] Verifying output is a Linux amd64 ELF..."
if file "$OUT" | grep -q "ELF 64-bit LSB.*x86-64"; then
  ls -lh "$OUT"
  echo "✓ Build OK"
else
  echo "FATAL: $OUT is not a Linux x86-64 ELF" >&2
  file "$OUT" >&2
  exit 2
fi
```

- [ ] **Step 2: Make it executable**

```bash
chmod +x deploy/build-linux.sh
```

- [ ] **Step 3: Verify the script runs and produces a valid binary**

Run: `bash deploy/build-linux.sh`

Expected (truncated):

```
[1/3] Pulling build image golang:1.23-bookworm (no-op if cached)...
...
[2/3] Cross-compiling gitea (this takes 3-6 minutes)...
...
[3/3] Verifying output is a Linux amd64 ELF...
-rwxr-xr-x  1 ... 130M ... gitea-linux-amd64
✓ Build OK
```

If the docker pull or build fails on the dev machine for any reason (network, disk), DO NOT MERGE — debug first. The whole migration depends on this script working.

- [ ] **Step 4: Add the build artifact to .gitignore**

Modify: `.gitignore`

Add the line `gitea-linux-amd64` after the existing `gitea` line. Confirm with:

```bash
grep -n "^gitea" .gitignore
```

Expected: `gitea` and `gitea-linux-amd64` both listed.

- [ ] **Step 5: Commit**

```bash
git add deploy/build-linux.sh .gitignore
git commit -m "feat(deploy): cross-compile gitea for linux/amd64 via docker

Produces ./gitea-linux-amd64 from the Mac without installing a Linux Go
toolchain. Used by the prod migration's binary-deploy flow."
```

---

### Task 3: `deploy/app.ini.tmpl` (production app.ini template)

**Files:**
- Create: `deploy/app.ini.tmpl`

- [ ] **Step 1: Capture the current Mac app.ini as a starting point**

```bash
cp custom/conf/app.ini /tmp/mac-app-ini-snapshot.ini
echo "Snapshot of Mac app.ini at: /tmp/mac-app-ini-snapshot.ini"
wc -l /tmp/mac-app-ini-snapshot.ini
```

This is read-only inspection; the snapshot is used to verify all the production-relevant keys are carried over.

- [ ] **Step 2: Write the template**

Path: `deploy/app.ini.tmpl`. The bootstrap script will substitute `__VAR__` placeholders.

```ini
APP_NAME = HackForger
RUN_MODE = prod
WORK_PATH = /var/lib/hackforger

[database]
DB_TYPE = postgres
HOST = 127.0.0.1:5432
NAME = hackforger
USER = hackforger
PASSWD = __PG_PASSWORD__
SSL_MODE = disable
LOG_SQL = false

[repository]
ROOT = /var/lib/hackforger/data/forgejo-repositories

[server]
DOMAIN = www.synnovator.com
HTTP_ADDR = 127.0.0.1
HTTP_PORT = 3000
ROOT_URL = https://www.synnovator.com/
DISABLE_SSH = true
LFS_START_SERVER = false

[lfs]
PATH = /var/lib/hackforger/data/lfs

[security]
INSTALL_LOCK = true
INTERNAL_TOKEN = __INTERNAL_TOKEN__

[log]
MODE = console
LEVEL = Info
ROOT_PATH = /var/lib/hackforger/log

[actions]
ENABLED = true
DEFAULT_ACTIONS_URL = https://code.forgejo.org

[oauth2]
JWT_SECRET = __OAUTH2_JWT_SECRET__

[ui]
DEFAULT_THEME = hackforger-dark

[service]
DISABLE_REGISTRATION = false
ALLOW_ONLY_EXTERNAL_REGISTRATION = true
```

The four placeholders the bootstrap script substitutes:
- `__PG_PASSWORD__` — fresh strong password generated on the ECS
- `__INTERNAL_TOKEN__` — copied from the migrated DB / Mac app.ini (must NOT change, else runner ↔ gitea breaks)
- `__OAUTH2_JWT_SECRET__` — copied (must NOT change, else all sessions invalidated)

- [ ] **Step 3: Diff against Mac snapshot to confirm coverage**

```bash
diff <(grep -oE "^\[.*\]|^[A-Z_]+\s*=" /tmp/mac-app-ini-snapshot.ini | sort -u) \
     <(grep -oE "^\[.*\]|^[A-Z_]+\s*=" deploy/app.ini.tmpl | sort -u)
```

Expected: only differences are keys that intentionally differ (e.g. paths, DOMAIN). If a section/key is missing from the template that exists on Mac AND is production-relevant, add it.

- [ ] **Step 4: Commit**

```bash
git add deploy/app.ini.tmpl
git commit -m "feat(deploy): production app.ini template

Carries over INTERNAL_TOKEN and OAUTH2_JWT_SECRET from the migrated DB
(verbatim, else sessions/runner break). Rotates PG password. Hard-codes
www.synnovator.com as DOMAIN/ROOT_URL."
```

---

### Task 4: `deploy/caddy/Caddyfile.tmpl`

**Files:**
- Create: `deploy/caddy/Caddyfile.tmpl`

- [ ] **Step 1: Write the Caddyfile template**

Path: `deploy/caddy/Caddyfile.tmpl`

```caddy
# HackForger reverse proxy + auto-HTTPS via Let's Encrypt HTTP-01.
# Installed by deploy/ecs-bootstrap.sh to /etc/caddy/Caddyfile.

{
    email __ACME_EMAIL__
}

www.synnovator.com {
    encode gzip zstd

    # Forward to gitea on loopback. Gitea sees the original Host and X-Forwarded-*.
    reverse_proxy 127.0.0.1:3000

    # Sensible defaults for git ops over HTTPS:
    # - Larger upload size for git push of repos with binary blobs
    # - Don't buffer streaming responses (matters for clone progress)
    request_body {
        max_size 500MB
    }

    log {
        output file /var/log/caddy/access.log {
            roll_size 100mb
            roll_keep 5
        }
        format console
    }
}
```

The `__ACME_EMAIL__` placeholder is filled by `ecs-bootstrap.sh` (env var `ACME_EMAIL`, default `ops@synnovator.com`).

- [ ] **Step 2: Commit**

```bash
git add deploy/caddy/Caddyfile.tmpl
git commit -m "feat(deploy): Caddyfile template for www.synnovator.com

Auto-HTTPS via LE HTTP-01. Reverse-proxies to gitea on 127.0.0.1:3000.
500MB request body limit for git push. Access log rotation."
```

---

### Task 5: systemd unit files (4 of them)

**Files:**
- Create: `deploy/systemd/gitea.service`
- Create: `deploy/systemd/forgejo-runner.service`
- Create: `deploy/systemd/hackforger-backup.service`
- Create: `deploy/systemd/hackforger-backup.timer`

- [ ] **Step 1: Write `deploy/systemd/gitea.service`**

```ini
[Unit]
Description=HackForger (gitea fork)
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=hackforger
Group=hackforger
WorkingDirectory=/var/lib/hackforger
ExecStart=/opt/hackforger/gitea web --custom-path /var/lib/hackforger/custom --work-path /var/lib/hackforger
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536
Environment=USER=hackforger HOME=/var/lib/hackforger

# Hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/lib/hackforger /var/log
PrivateTmp=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes

[Install]
WantedBy=multi-user.target
```

- [ ] **Step 2: Write `deploy/systemd/forgejo-runner.service`**

```ini
[Unit]
Description=Forgejo Actions runner for HackForger
After=network.target gitea.service
Wants=gitea.service

[Service]
Type=simple
User=hackforger
Group=hackforger
WorkingDirectory=/var/lib/forgejo-runner
ExecStart=/usr/local/bin/forgejo-runner daemon --config /etc/forgejo-runner/config.yml
Restart=on-failure
RestartSec=5s

# Hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/lib/forgejo-runner /tmp
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
```

- [ ] **Step 3: Write `deploy/systemd/hackforger-backup.service`**

```ini
[Unit]
Description=HackForger nightly DB dump
After=postgresql.service

[Service]
Type=oneshot
User=hackforger
Group=hackforger
ExecStart=/usr/local/bin/hackforger-backup.sh
StandardOutput=journal
StandardError=journal
```

- [ ] **Step 4: Write `deploy/systemd/hackforger-backup.timer`**

```ini
[Unit]
Description=Run HackForger backup daily at 04:00 Asia/Shanghai

[Timer]
OnCalendar=*-*-* 04:00:00 Asia/Shanghai
Persistent=true
RandomizedDelaySec=120s
Unit=hackforger-backup.service

[Install]
WantedBy=timers.target
```

`Persistent=true` means a missed run (machine off at 04:00) executes on next boot. `RandomizedDelaySec=120s` adds jitter so a fleet of timers wouldn't stampede the DB if we ever scaled out.

- [ ] **Step 5: Commit**

```bash
git add deploy/systemd/gitea.service \
        deploy/systemd/forgejo-runner.service \
        deploy/systemd/hackforger-backup.service \
        deploy/systemd/hackforger-backup.timer
git commit -m "feat(deploy): systemd units for gitea, runner, backup timer

Hardened with ProtectSystem=strict + ReadWritePaths. Backup timer runs
04:00 Asia/Shanghai with Persistent=true for missed runs."
```

---

### Task 6: `deploy/ecs-bootstrap.sh` (idempotent install)

**Files:**
- Create: `deploy/ecs-bootstrap.sh`

This is the longest script. It must be safe to run twice. Each block checks state before mutating.

- [ ] **Step 1: Write the script**

Path: `deploy/ecs-bootstrap.sh`. The script is invoked on the ECS as the `hackforger` user; sudo is used per-command.

```bash
#!/bin/bash
# Idempotent bootstrap of HackForger on the Huawei ECS.
# Run as the hackforger user on 218.91.114.178. Will prompt for sudo password.
#
# Re-running is safe: each step checks state first.
#
# Inputs (env vars, optional):
#   ACME_EMAIL  — email Caddy gives to Let's Encrypt (default: ops@synnovator.com)
#
# Outputs:
#   Postgres 18 installed and running
#   /var/lib/hackforger/ created with correct ownership
#   /opt/hackforger/ created
#   forgejo-runner binary installed
#   Caddy installed
#   systemd units installed (NOT started — gitea binary + DB import come later)

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
  rsync curl jq ufw
ok "apt installs done"

# ---------------------------------------------------------------------------
log "[4/9] Configuring postgres (loopback only, hackforger DB+role)"
PG_CONF=/etc/postgresql/18/main/postgresql.conf
PG_HBA=/etc/postgresql/18/main/pg_hba.conf

# Ensure listen on 127.0.0.1 only (default is localhost which is fine, but force explicit)
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
         /var/lib/forgejo-runner /etc/forgejo-runner /var/log/caddy; do
  if [ -d "$d" ]; then
    skip "$d exists"
  else
    sudo install -d -o hackforger -g hackforger "$d"
    ok "created $d"
  fi
done

sudo chown -R hackforger:hackforger /opt/hackforger /var/lib/hackforger /var/lib/forgejo-runner
sudo chown caddy:caddy /var/log/caddy

# ---------------------------------------------------------------------------
log "[6/9] Installing forgejo-runner v$RUNNER_VERSION"
if command -v forgejo-runner >/dev/null && forgejo-runner --version 2>/dev/null | grep -q "$RUNNER_VERSION"; then
  skip "forgejo-runner $RUNNER_VERSION already installed"
else
  ARCH=$(dpkg --print-architecture)
  if [ "$ARCH" = "amd64" ]; then RUNNER_ARCH=amd64; else RUNNER_ARCH=arm64; fi
  sudo curl -fsSL \
    "https://code.forgejo.org/forgejo/runner/releases/download/$RUNNER_VERSION/forgejo-runner-$RUNNER_VERSION-linux-$RUNNER_ARCH" \
    -o /usr/local/bin/forgejo-runner
  sudo chmod +x /usr/local/bin/forgejo-runner
  ok "forgejo-runner installed"
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
if [ ! -f /etc/caddy/Caddyfile.bak ]; then
  sudo cp /etc/caddy/Caddyfile /etc/caddy/Caddyfile.bak 2>/dev/null || true
fi
sudo sed "s/__ACME_EMAIL__/$ACME_EMAIL/g" "$SCRIPT_DIR/caddy/Caddyfile.tmpl" \
  | sudo tee /etc/caddy/Caddyfile >/dev/null
ok "Caddyfile written (ACME email: $ACME_EMAIL)"

# ---------------------------------------------------------------------------
log "[8/9] Installing systemd units + backup script"
for unit in gitea.service forgejo-runner.service \
            hackforger-backup.service hackforger-backup.timer; do
  sudo cp "$SCRIPT_DIR/systemd/$unit" "/etc/systemd/system/$unit"
done
sudo cp "$SCRIPT_DIR/backup-ecs.sh" /usr/local/bin/hackforger-backup.sh
sudo chmod +x /usr/local/bin/hackforger-backup.sh
sudo systemctl daemon-reload
sudo systemctl enable hackforger-backup.timer
ok "systemd units installed (gitea + runner enabled but NOT started — start after data migration)"

# ---------------------------------------------------------------------------
log "[9/9] Summary"
echo "  Postgres:        $(systemctl is-active postgresql)"
echo "  Caddy:           $(systemctl is-active caddy)"
echo "  Gitea:           NOT STARTED (start after migrate-data.sh)"
echo "  Runner:          NOT STARTED (start after registering with gitea admin token)"
echo "  Backup timer:    $(systemctl is-active hackforger-backup.timer || echo 'inactive')"
echo
echo "  PG password:     /root/.hackforger-pg-password"
echo "  Custom path:     /var/lib/hackforger/custom"
echo "  Binary path:     /opt/hackforger/gitea (NOT YET PRESENT — scp it next)"
echo
echo "Next steps:"
echo "  1. From Mac: bash deploy/build-linux.sh"
echo "  2. From Mac: scp gitea-linux-amd64 hackforger@218.91.114.178:/opt/hackforger/gitea"
echo "  3. From Mac: bash deploy/migrate-data.sh"
echo "  4. On ECS:   sudo systemctl start gitea caddy"
ok "Bootstrap complete"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x deploy/ecs-bootstrap.sh
```

- [ ] **Step 3: Smoke-check syntax**

```bash
bash -n deploy/ecs-bootstrap.sh
echo "exit: $?"
```

Expected: exit 0 (no syntax errors).

- [ ] **Step 4: Lint with shellcheck if available**

```bash
if command -v shellcheck >/dev/null; then
  shellcheck deploy/ecs-bootstrap.sh
else
  echo "shellcheck not installed; skipping (not blocking)"
fi
```

Fix any errors shellcheck reports (warnings are advisory).

- [ ] **Step 5: Commit**

```bash
git add deploy/ecs-bootstrap.sh
git commit -m "feat(deploy): idempotent ECS bootstrap script

Installs PG18, Caddy, forgejo-runner, creates /var/lib/hackforger and
/opt/hackforger, writes Caddyfile + systemd units. Each step checks
state before mutating, so re-runs converge. PG password generated and
saved to /root/.hackforger-pg-password (chmod 600)."
```

---

### Task 7: `deploy/migrate-data.sh` (dump + rsync + restore)

**Files:**
- Create: `deploy/migrate-data.sh`

This script does both sides of the migration:
- When run on the Mac, dumps + rsyncs to the ECS, then triggers the restore via SSH.
- Supports `--dry-run` to do every step except the destructive ones (no service stop, no restore).

- [ ] **Step 1: Write the script**

Path: `deploy/migrate-data.sh`

```bash
#!/bin/bash
# Mac-side: dump the post-clean-slate prod DB and data dirs, rsync to the ECS,
# trigger restore via SSH.
#
# Idempotent for the dump+rsync side. The restore side OVERWRITES the ECS DB
# and data dirs, so guard with --confirm.
#
# Usage:
#   bash deploy/migrate-data.sh --dry-run     # dump + rsync only, no restore
#   bash deploy/migrate-data.sh --confirm     # full migration

set -euo pipefail

DRY_RUN=0
CONFIRM=0
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=1 ;;
    --confirm) CONFIRM=1 ;;
    *) echo "Usage: $0 (--dry-run|--confirm)" >&2; exit 64 ;;
  esac
done
if [ "$DRY_RUN$CONFIRM" = "00" ]; then
  echo "FATAL: pass --dry-run or --confirm" >&2
  exit 64
fi

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"

ECS=hackforger@218.91.114.178
TMP_DUMP=/tmp/hackforger-prod.pgc
TMP_TAR=/tmp/hackforger-data.tgz

log() { printf "\n\033[1;34m▶ %s\033[0m\n" "$*"; }

# ---------------------------------------------------------------------------
log "[1/6] Reading PG password from .env"
if ! grep -q "^PGPASSWORD=" .env; then
  echo "FATAL: PGPASSWORD not in .env" >&2
  exit 1
fi
PG_PASS=$(grep "^PGPASSWORD=" .env | cut -d= -f2)
export PGPASSWORD="$PG_PASS"

# ---------------------------------------------------------------------------
log "[2/6] Stop Mac gitea (if running) — only with --confirm"
if [ "$CONFIRM" = "1" ]; then
  PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t 2>/dev/null || true)
  if [ -n "$PID" ]; then
    echo "  Stopping gitea PID $PID"
    kill "$PID"
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      lsof -iTCP:3000 -sTCP:LISTEN -t >/dev/null 2>&1 || break
      sleep 1
    done
  else
    echo "  gitea not running on :3000"
  fi
else
  echo "  --dry-run: leaving Mac gitea running"
fi

# ---------------------------------------------------------------------------
log "[3/6] pg_dump → $TMP_DUMP"
pg_dump -Fc -h 127.0.0.1 -U hackforger hackforger > "$TMP_DUMP"
ls -lh "$TMP_DUMP"

# ---------------------------------------------------------------------------
log "[4/6] tar data dirs + custom assets → $TMP_TAR"
tar czf "$TMP_TAR" \
  -C "$REPO" \
  data/forgejo-repositories \
  data/attachments \
  data/avatars \
  $( [ -d data/lfs ] && echo data/lfs ) \
  custom/conf/app.ini \
  custom/public \
  custom/templates 2>/dev/null
ls -lh "$TMP_TAR"

# ---------------------------------------------------------------------------
log "[5/6] rsync to ECS"
rsync -avz --progress "$TMP_DUMP" "$TMP_TAR" "$ECS:/tmp/"

# ---------------------------------------------------------------------------
log "[6/6] Trigger restore on ECS"
if [ "$CONFIRM" = "1" ]; then
  # Capture current Mac app.ini's INTERNAL_TOKEN and OAUTH2_JWT_SECRET so we
  # can carry them over verbatim — they MUST NOT change across migration.
  INTERNAL_TOKEN=$(grep "^INTERNAL_TOKEN" custom/conf/app.ini | cut -d= -f2- | xargs)
  OAUTH_JWT=$(grep "^JWT_SECRET" custom/conf/app.ini | cut -d= -f2- | xargs)

  ssh "$ECS" \
    INTERNAL_TOKEN="'$INTERNAL_TOKEN'" \
    OAUTH_JWT="'$OAUTH_JWT'" \
    bash -s <<'REMOTE_EOF'
set -euo pipefail

log() { printf "\n\033[1;34m  ▶ %s\033[0m\n" "$*"; }

log "Dropping & recreating PG database hackforger"
sudo -u postgres dropdb --if-exists hackforger
sudo -u postgres createdb -O hackforger hackforger

log "pg_restore"
PGPASSWORD=$(sudo cat /root/.hackforger-pg-password) \
  pg_restore -h 127.0.0.1 -U hackforger -d hackforger /tmp/hackforger-prod.pgc

log "Extracting data tar to /var/lib/hackforger"
sudo -u hackforger mkdir -p /var/lib/hackforger
sudo tar xzf /tmp/hackforger-data.tgz -C /var/lib/hackforger
sudo chown -R hackforger:hackforger /var/lib/hackforger

log "Rendering app.ini from template"
PG_PASS=$(sudo cat /root/.hackforger-pg-password)
sudo -u hackforger bash -c "
  sed \\
    -e 's|__PG_PASSWORD__|$PG_PASS|g' \\
    -e 's|__INTERNAL_TOKEN__|$INTERNAL_TOKEN|g' \\
    -e 's|__OAUTH2_JWT_SECRET__|$OAUTH_JWT|g' \\
    /opt/hackforger/app.ini.tmpl \\
  > /var/lib/hackforger/custom/conf/app.ini
  chmod 640 /var/lib/hackforger/custom/conf/app.ini
"
log "Smoke-test: gitea --version"
sudo -u hackforger /opt/hackforger/gitea --version || {
  echo "gitea binary missing or not executable" >&2
  exit 2
}
echo "  ✓ Restore complete. Next: sudo systemctl start gitea caddy"
REMOTE_EOF
else
  echo "  --dry-run: skipping restore step"
  echo "  Files staged on ECS at: /tmp/hackforger-prod.pgc, /tmp/hackforger-data.tgz"
fi

log "Done"
```

- [ ] **Step 2: Make executable + syntax check**

```bash
chmod +x deploy/migrate-data.sh
bash -n deploy/migrate-data.sh
echo "exit: $?"
```

Expected: exit 0.

- [ ] **Step 3: Add app.ini.tmpl to ECS via bootstrap**

The remote restore step references `/opt/hackforger/app.ini.tmpl`. Add a step to `ecs-bootstrap.sh` to copy it. Modify `deploy/ecs-bootstrap.sh`:

Find this block (Step 8, the one that copies systemd units):

```bash
log "[8/9] Installing systemd units + backup script"
for unit in gitea.service forgejo-runner.service \
            hackforger-backup.service hackforger-backup.timer; do
  sudo cp "$SCRIPT_DIR/systemd/$unit" "/etc/systemd/system/$unit"
done
```

Add right after the `for ... done`:

```bash
sudo cp "$SCRIPT_DIR/app.ini.tmpl" /opt/hackforger/app.ini.tmpl
sudo chown hackforger:hackforger /opt/hackforger/app.ini.tmpl
```

Verify by re-reading the block:

```bash
grep -n -A 3 "app.ini.tmpl" deploy/ecs-bootstrap.sh
```

Expected: shows the new lines.

- [ ] **Step 4: Commit**

```bash
git add deploy/migrate-data.sh deploy/ecs-bootstrap.sh
git commit -m "feat(deploy): one-shot data migration script

migrate-data.sh runs from the Mac. --dry-run dumps+rsyncs only;
--confirm stops Mac gitea, then triggers PG drop+restore + tar
extract + app.ini render on the ECS over SSH. INTERNAL_TOKEN and
OAUTH2_JWT_SECRET carried verbatim from Mac's app.ini to avoid
session/runner breakage. PG password rotates via the value bootstrap
saved in /root/.hackforger-pg-password.

ecs-bootstrap.sh now also installs app.ini.tmpl to /opt/hackforger/."
```

---

### Task 8: `deploy/backup-ecs.sh` (the script the systemd timer runs)

**Files:**
- Create: `deploy/backup-ecs.sh`

- [ ] **Step 1: Write the script**

Path: `deploy/backup-ecs.sh`. Bootstrap installs it as `/usr/local/bin/hackforger-backup.sh`.

```bash
#!/bin/bash
# Nightly backup of HackForger DB. Invoked by systemd timer
# /etc/systemd/system/hackforger-backup.timer.
#
# - Dumps the Postgres `hackforger` DB to /var/lib/hackforger/pg-backups/
# - Compressed custom format (-Fc)
# - Filename includes ISO date for natural sort + clarity
# - Retains 14 days, deletes older
#
# stdout/stderr go to journald via systemd.

set -euo pipefail

BACKUP_DIR=/var/lib/hackforger/pg-backups
RETAIN_DAYS=14
STAMP=$(date -u +%Y%m%d-%H%M%SZ)
OUT="$BACKUP_DIR/hf-${STAMP}.pgc"

mkdir -p "$BACKUP_DIR"

# Read PG password from the file root saved during bootstrap. Even though
# we run as hackforger, we still need to authenticate to PG (hackforger role
# requires password — md5 auth in pg_hba).
PGPASSWORD=$(sudo cat /root/.hackforger-pg-password) \
  pg_dump -Fc -h 127.0.0.1 -U hackforger hackforger \
  > "$OUT"

SIZE=$(stat -c%s "$OUT")
printf "backup-ecs OK: %s (%d bytes)\n" "$OUT" "$SIZE"

# Rotation
DELETED=$(find "$BACKUP_DIR" -name 'hf-*.pgc' -mtime "+$RETAIN_DAYS" -delete -print | wc -l)
printf "rotation: deleted %d files older than %d days\n" "$DELETED" "$RETAIN_DAYS"
```

Note the `sudo cat` in the script: this requires the operator to set up a tiny sudoers rule for `hackforger` to read just that one file. Add the rule to bootstrap:

- [ ] **Step 2: Add the sudoers rule to ecs-bootstrap.sh**

In `deploy/ecs-bootstrap.sh`, add this block right before `[8/9]` (so it's installed before the backup script that needs it):

```bash
log "[7.5/9] Allowing hackforger to read /root/.hackforger-pg-password"
SUDOERS=/etc/sudoers.d/hackforger-backup
if [ ! -f "$SUDOERS" ]; then
  echo 'hackforger ALL=(root) NOPASSWD: /usr/bin/cat /root/.hackforger-pg-password' \
    | sudo tee "$SUDOERS" >/dev/null
  sudo chmod 440 "$SUDOERS"
  sudo visudo -cf "$SUDOERS"  # validate
  ok "sudoers rule installed"
else
  skip "sudoers rule exists"
fi
```

- [ ] **Step 3: Make executable + syntax check**

```bash
chmod +x deploy/backup-ecs.sh
bash -n deploy/backup-ecs.sh
bash -n deploy/ecs-bootstrap.sh
echo "exit: $?"
```

Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add deploy/backup-ecs.sh deploy/ecs-bootstrap.sh
git commit -m "feat(deploy): nightly DB backup script + sudoers carve-out

backup-ecs.sh runs from systemd timer hackforger-backup.timer.
Dumps PG → /var/lib/hackforger/pg-backups/hf-YYYYMMDD-HHMMSSZ.pgc,
retains 14 days. Bootstrap now installs a tiny sudoers rule allowing
hackforger to cat /root/.hackforger-pg-password (read-only, single
file) so the backup script can authenticate to PG."
```

---

### Task 9: `scripts/backup-pull.sh` + launchd plist (Mac side)

**Files:**
- Create: `scripts/backup-pull.sh`
- Create: `deploy/launchd/com.h2os.hackforger-backup-pull.plist`

- [ ] **Step 1: Write the puller script**

Path: `scripts/backup-pull.sh`

```bash
#!/bin/bash
# Mac-side: pull the ECS's nightly backups to ~/Backups/hackforger/.
# Invoked by launchd job com.h2os.hackforger-backup-pull at 04:30 daily.

set -euo pipefail

DEST="${HOME}/Backups/hackforger"
mkdir -p "$DEST"/{db,data,custom}

REMOTE="hackforger@218.91.114.178"
LOG="$DEST/backup.log"

log() {
  printf "%s %s\n" "$(date -Iseconds)" "$*" | tee -a "$LOG"
}

log "===== backup-pull start ====="

# DB dumps: full mirror (last 14 days are kept on the ECS)
rsync -az --delete \
  --rsync-path="rsync" \
  "$REMOTE:/var/lib/hackforger/pg-backups/" "$DEST/db/" \
  2>&1 | tail -5 | sed 's/^/  /' | tee -a "$LOG"

# Live data: mirror current state of repos, attachments, avatars.
# This is NOT point-in-time consistent; rely on db/ for actual recovery.
rsync -az --delete \
  "$REMOTE:/var/lib/hackforger/data/" "$DEST/data/" \
  2>&1 | tail -5 | sed 's/^/  /' | tee -a "$LOG"

# Custom assets (app.ini + landing page customizations)
rsync -az --delete \
  "$REMOTE:/var/lib/hackforger/custom/" "$DEST/custom/" \
  2>&1 | tail -5 | sed 's/^/  /' | tee -a "$LOG"

# Sanity: the newest .pgc file should be ≤24h old
NEWEST=$(find "$DEST/db" -name 'hf-*.pgc' -mtime -1 | head -1 || true)
if [ -z "$NEWEST" ]; then
  log "WARN: no .pgc file <24h old in $DEST/db/"
else
  log "newest dump: $NEWEST"
fi

log "===== backup-pull OK ====="
```

- [ ] **Step 2: Make executable**

```bash
chmod +x scripts/backup-pull.sh
```

- [ ] **Step 3: Write the launchd plist**

Path: `deploy/launchd/com.h2os.hackforger-backup-pull.plist`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.h2os.hackforger-backup-pull</string>

    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>/Users/h2oslabs/Workspace/hackforger/scripts/backup-pull.sh</string>
    </array>

    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>4</integer>
        <key>Minute</key>
        <integer>30</integer>
    </dict>

    <key>StandardOutPath</key>
    <string>/Users/h2oslabs/Backups/hackforger/launchd.log</string>

    <key>StandardErrorPath</key>
    <string>/Users/h2oslabs/Backups/hackforger/launchd.log</string>

    <key>RunAtLoad</key>
    <false/>
</dict>
</plist>
```

- [ ] **Step 4: Verify the plist is valid XML**

```bash
plutil -lint deploy/launchd/com.h2os.hackforger-backup-pull.plist
```

Expected: `deploy/launchd/com.h2os.hackforger-backup-pull.plist: OK`.

- [ ] **Step 5: Commit**

```bash
git add scripts/backup-pull.sh deploy/launchd/com.h2os.hackforger-backup-pull.plist
git commit -m "feat(deploy): Mac-side backup puller + launchd plist

scripts/backup-pull.sh rsyncs the ECS's pg-backups + data + custom dirs
to ~/Backups/hackforger/, runs daily at 04:30 via launchd. --delete
mirroring (no local accumulation beyond what ECS keeps). Sanity check
warns if no .pgc dump <24h old."
```

---

### Task 10: `deploy/cutover-runbook.md` (operator runbook)

**Files:**
- Create: `deploy/cutover-runbook.md`

- [ ] **Step 1: Write the runbook**

Path: `deploy/cutover-runbook.md`. This is the operational checklist used during the maintenance window.

```markdown
# Cutover Runbook — Mac → ECS migration

Window: target 30 min, hard limit 60 min.
Last rehearsed: (fill in date)
Operator: (fill in name)
Spec: [`docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md`](../docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md)

## T-24h — Prep

- [ ] In Aliyun DNS console, lower TTL of `synnovator.com` zone to 60s
- [ ] In Huawei console, verify ECS security group has inbound 80 + 443 open from `0.0.0.0/0`
        Verify externally:
        ```bash
        nc -z 218.91.114.178 80   && echo OK
        nc -z 218.91.114.178 443  && echo OK
        ```
- [ ] In Logto admin, add allowed callback URL `https://www.synnovator.com/user/oauth2/<source>/callback`
        (find the exact source name in the Mac DB:
        ```bash
        psql -h 127.0.0.1 -U hackforger hackforger \
          -c "SELECT name FROM login_source WHERE type=6;"
        ```
        )
- [ ] Notify users in advance (Slack / status page)

## T-1h — Dry run

- [ ] On the Mac:
      ```bash
      bash deploy/build-linux.sh
      file gitea-linux-amd64   # confirm: ELF 64-bit LSB ... x86-64
      scp gitea-linux-amd64 hackforger@218.91.114.178:/opt/hackforger/gitea
      ssh hackforger@218.91.114.178 'sudo chmod +x /opt/hackforger/gitea && /opt/hackforger/gitea --version'
      ```
- [ ] Dry-run the data migration (no service stop):
      ```bash
      bash deploy/migrate-data.sh --dry-run
      ```
      Files should appear at `/tmp/hackforger-prod.pgc` and `/tmp/hackforger-data.tgz` on the ECS.

## T0 — Cutover

### (a) Stop Mac gitea (announces 503 to anyone hitting old URL)

```bash
launchctl list | grep -i gitea  # if running under launchd
# else:
PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t)
kill "$PID"
```

### (b–c) Run the migration script

```bash
bash deploy/migrate-data.sh --confirm
```

Wait for "Restore complete" output. Ignore the last log line about "start gitea caddy" — we do that next.

### (d) Start services on ECS

```bash
ssh hackforger@218.91.114.178 'sudo systemctl start gitea && sudo systemctl start caddy'
```

### (e) Smoke test internally on ECS

```bash
ssh hackforger@218.91.114.178 '
  curl -fsS http://127.0.0.1:3000/api/v1/version &&
  curl -fsS http://127.0.0.1:3000/api/v1/admin/users \
       -u hackforger:$(cat ~/.env-admin | grep ADMIN_PASS | cut -d= -f2)
'
```

Expected: JSON with version + admin user object.

### (f) Verify Caddy from outside (will fail TLS hostname check, that's fine)

```bash
curl -kfsS https://218.91.114.178/api/v1/version
```

Expected: same version JSON.

### (g) Flip DNS

In Aliyun DNS console, add/update A record:
```
www.synnovator.com.   60   IN   A   218.91.114.178
```

### (h) Wait for propagation

```bash
for resolver in 8.8.8.8 1.1.1.1 114.114.114.114 223.5.5.5; do
  echo "$resolver: $(dig @$resolver www.synnovator.com +short)"
done
```

All four should return `218.91.114.178`. If not, wait 30s and retry.

### (i) Verify externally over the public hostname

```bash
curl -fsSL https://www.synnovator.com/api/v1/version
```

Expected: JSON, valid HTTPS, no cert warning.
Expected log on ECS: `journalctl -u caddy -n 30 | grep "obtained certificate"`

### (j) Verify Logto SSO login

In a browser, open `https://www.synnovator.com/`, click sign-in, complete the OAuth round-trip. Confirm landing on `/dashboard` as `hackforger`.

### (k) Re-register the runner

```bash
TOKEN=$(curl -fsS -u hackforger:PASS https://www.synnovator.com/api/v1/admin/runners/registration-token | jq -r .token)
ssh hackforger@218.91.114.178 "
  cd /var/lib/forgejo-runner
  /usr/local/bin/forgejo-runner register \
    --no-interactive \
    --instance http://127.0.0.1:3000 \
    --token $TOKEN \
    --name ecs-prod-runner \
    --labels 'ubuntu-latest:host'
  sudo systemctl start forgejo-runner
"
```

Verify:
```bash
curl -fsS -u hackforger:PASS https://www.synnovator.com/api/v1/admin/runners | jq
```

Trigger a workflow_dispatch on a track repo (any existing one) and watch:
```bash
ssh hackforger@218.91.114.178 'journalctl -u forgejo-runner -n 30 -f'
```

## Rollback (if any of d–k fails irrecoverably)

1. In Aliyun DNS, revert A record to old value (or remove it; the apex `synnovator.com` still resolves to 47.x):
2. Restart Mac gitea: `bash scripts/restart-gitea.sh`
3. Old domain `hackforger.inside.h2os.cloud` independently still works (Tailscale)
4. Diagnose ECS state offline; re-run migrate-data.sh after fix

Total rollback latency: ~3 min (60s TTL + service restart).

## T+24h — Verify backup pipeline

```bash
# On Mac
ls -lt ~/Backups/hackforger/db/ | head -5
tail -20 ~/Backups/hackforger/backup.log
# Expect a fresh hf-YYYYMMDD-HHMMSSZ.pgc ≤24h old
```

If no fresh dump:
```bash
launchctl list | grep hackforger-backup
launchctl print gui/$UID/com.h2os.hackforger-backup-pull | head -30
```

Check the ECS-side timer:
```bash
ssh hackforger@218.91.114.178 'systemctl list-timers hackforger-backup.timer'
```

## T+30d — Decommission decision

- If no problems: stop Mac gitea permanently, remove `hackforger.inside.h2os.cloud` DNS record, archive Mac's `data/` to cold storage.
- If something needs Mac for fallback: keep warm-standby until decision-day-2.
```

- [ ] **Step 2: Verify all the placeholder markers are filled in (or explicitly TBD)**

```bash
grep -nE "TBD|TODO|XXX|fill in" deploy/cutover-runbook.md | grep -v "fill in date\|fill in name"
```

Expected: empty output (only the operator-fill placeholders for date/name remain, by design).

- [ ] **Step 3: Commit**

```bash
git add deploy/cutover-runbook.md
git commit -m "feat(deploy): cutover runbook with exact operator commands

T-24h prep + T-1h dry-run + T0 cutover steps a-k + rollback +
T+24h backup verification + T+30d decommission decision. All
commands are copy-pasteable; the runbook is the operator's only
required input during the maintenance window."
```

---

### Task 11: Open PR for all of Phase A

**Files:**
- Modify: nothing more

- [ ] **Step 1: Confirm the branch + commits**

```bash
git log --oneline v0.1-dev/hackforger..HEAD
```

Expected: 9 commits on top of `v0.1-dev/hackforger` (one per Phase-A task).

- [ ] **Step 2: Push the branch**

```bash
git push -u origin feat/prod-cloud-migration-scaffolding
```

- [ ] **Step 3: Open the PR**

```bash
gh pr create --title "feat(deploy): prod cloud migration scaffolding (binary deploy on Huawei ECS)" --body "$(cat <<'EOF'
## Summary
Adds `deploy/` and `scripts/backup-pull.sh` — every artifact needed for the Mac → Huawei ECS migration described in [`docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md`](docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md). No behavior change to the running Mac instance; this PR is pure scaffolding.

## What's in here
- `deploy/build-linux.sh` — cross-compile gitea via docker (Mac stays clean)
- `deploy/ecs-bootstrap.sh` — idempotent install of PG18 + Caddy + runner + dirs + systemd
- `deploy/migrate-data.sh` — dump + rsync + restore one-shot, supports `--dry-run`
- `deploy/backup-ecs.sh` — nightly `pg_dump` (14-day rotation) called by systemd timer
- `deploy/systemd/*` — gitea, runner, backup service+timer
- `deploy/caddy/Caddyfile.tmpl` — reverse-proxy + LE auto-HTTPS
- `deploy/app.ini.tmpl` — production app.ini, placeholders for PG password + INTERNAL_TOKEN + JWT secrets
- `deploy/cutover-runbook.md` — T-24h / T-1h / T0 / T+24h checklist
- `scripts/backup-pull.sh` + `deploy/launchd/com.h2os.hackforger-backup-pull.plist` — Mac-side daily rsync

## Test plan
- [x] `bash -n` syntax-checks all scripts
- [x] `plutil -lint` validates the launchd plist
- [x] `bash deploy/build-linux.sh` produces a Linux ELF binary
- [ ] Reviewer: confirm 4 manual prerequisites in the runbook are reasonable
      (Huawei security group, Aliyun DNS A record + TTL, Logto callback URL)
- [ ] Reviewer: spot-check `ecs-bootstrap.sh` for idempotency claims (each block guards on state)

## Out of scope (future PRs)
- Live cutover execution (separate operational task, not committed)
- CI-driven auto-deploy
- Docker mode for the runner
- Public-internet exposure of `hackforger.inside.h2os.cloud` (stays Tailscale-only)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 4: Wait for review + merge**

The PR review is the gate before any Phase-B work touches the ECS. Do not proceed to Task 12 until merged.

After merge:

```bash
git checkout v0.1-dev/hackforger
git pull --ff-only
```

---

## Phase B — Live execution (Tasks 12–15)

> ⚠️ Phase B operates on production. **Do not start Task 12 until:**
> - Phase A PR is merged
> - Huawei security group has 80+443 open (verify with `nc -z`)
> - Aliyun DNS TTL for `synnovator.com` zone is 60s
> - Logto callback URL is whitelisted

### Task 12: ECS Bootstrap

**Files:** none in repo (artifacts produced on ECS only)

- [ ] **Step 1: Copy bootstrap files to ECS**

```bash
ssh hackforger@218.91.114.178 'mkdir -p /tmp/deploy'
scp -r deploy/ hackforger@218.91.114.178:/tmp/
```

- [ ] **Step 2: Run bootstrap (interactive sudo prompt)**

```bash
ssh -t hackforger@218.91.114.178 'cd /tmp/deploy && bash ecs-bootstrap.sh'
```

Expected (excerpt):

```
▶ [0/9] Caching sudo credentials
[sudo] password for hackforger: ●●●●●●●●
▶ [1/9] Adding PostgreSQL 18 (PGDG) apt repository
  ✓ pgdg repo added
▶ [2/9] Adding Caddy apt repository
  ✓ caddy repo added
▶ [3/9] apt install: postgresql-18, caddy, deps
...
▶ [9/9] Summary
  Postgres:        active
  Caddy:           active
  Gitea:           NOT STARTED (start after migrate-data.sh)
  Runner:          NOT STARTED (start after registering with gitea admin token)
  Backup timer:    active
  ✓ Bootstrap complete
```

- [ ] **Step 3: Re-run to verify idempotency**

```bash
ssh -t hackforger@218.91.114.178 'cd /tmp/deploy && bash ecs-bootstrap.sh'
```

Expected: every step prints `· skipped — already done`. Total runtime <30 s.

- [ ] **Step 4: Manual sanity checks**

```bash
ssh hackforger@218.91.114.178 '
  systemctl is-active postgresql
  systemctl is-active caddy
  systemctl is-active hackforger-backup.timer
  sudo -u postgres psql -c "\\l hackforger"
  ls -la /opt/hackforger /var/lib/hackforger
  /usr/local/bin/forgejo-runner --version
'
```

Expected: all `active`, DB exists, dirs present, runner version printed.

### Task 13: Cross-Compile + Smoke

- [ ] **Step 1: Cross-compile**

```bash
cd /Users/h2oslabs/Workspace/hackforger
bash deploy/build-linux.sh
```

- [ ] **Step 2: scp to ECS**

```bash
scp gitea-linux-amd64 hackforger@218.91.114.178:/opt/hackforger/gitea
ssh hackforger@218.91.114.178 'sudo chmod +x /opt/hackforger/gitea && /opt/hackforger/gitea --version'
```

Expected: version string matching the local build.

### Task 14: Cutover

Follow `deploy/cutover-runbook.md` step by step. Check off each box in the runbook as completed.

The runbook is the contract. Don't summarize it here.

### Task 15: Backup verification (T+24h)

- [ ] **Step 1: Install the launchd job (was committed but not loaded)**

```bash
cp deploy/launchd/com.h2os.hackforger-backup-pull.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.h2os.hackforger-backup-pull.plist
launchctl list | grep hackforger-backup
```

- [ ] **Step 2: Trigger manually once (don't wait 24h to find a bug)**

```bash
launchctl start com.h2os.hackforger-backup-pull
sleep 30
ls -lh ~/Backups/hackforger/db/
tail -20 ~/Backups/hackforger/backup.log
```

Expected: a `hf-*.pgc` file in `~/Backups/hackforger/db/`, log shows `===== backup-pull OK =====`.

- [ ] **Step 3: 24h later, confirm the timer-driven run worked**

```bash
launchctl print gui/$UID/com.h2os.hackforger-backup-pull | grep -E "last_exit_status|state|run_count"
```

Expected: `last_exit_status = 0`, `run_count` incremented, `state = not running`.

- [ ] **Step 4: Restorability spot check**

```bash
NEWEST=$(ls -t ~/Backups/hackforger/db/hf-*.pgc | head -1)
pg_restore --list "$NEWEST" | head -20
```

Expected: TOC listing with `TABLE DATA` entries for HackForger tables.

- [ ] **Step 5: Mark migration complete**

Update the spec's predecessor-archive line and close any related issues.

---

## Self-Review Notes

**Spec coverage:**
- §4 Target State → Tasks 5, 6, 12 (systemd units installed by bootstrap)
- §5 Cross-compile → Task 2
- §6 Network/TLS → Task 4 (Caddyfile) + runbook prereqs
- §7 Migration → Task 7
- §8 Cutover → Tasks 10 + 14 (runbook is the implementation)
- §9 Backup pipeline → Tasks 8, 9, 15
- §10 Repo layout → Tasks 1–10 produce every file listed
- §11 Open questions: Q3 (Logto callback) → runbook T-24h. Q4 (security group) → runbook T-24h. All others have defaults baked into bootstrap or runbook.
- §12 Acceptance criteria → Task 14 (cutover) + Task 15 (backup pipeline) + idempotency check in Task 12 step 3
- §13 Risks → mitigations distributed across Tasks 13 (arch check), 14 (DNS verification), 10 (runbook structure)

**Placeholder scan:** None left. The runbook has two intentional `(fill in date/name)` markers for the operator at run time — explicitly excluded from the placeholder grep.

**Type/name consistency:** all paths checked: `/var/lib/hackforger/`, `/opt/hackforger/gitea`, `/var/lib/forgejo-runner/`, `/etc/forgejo-runner/config.yml`, `/usr/local/bin/forgejo-runner`, `/usr/local/bin/hackforger-backup.sh`, `/var/lib/hackforger/pg-backups/hf-*.pgc`. All four systemd units (gitea, forgejo-runner, hackforger-backup.service, hackforger-backup.timer) referenced consistently in bootstrap and runbook.
