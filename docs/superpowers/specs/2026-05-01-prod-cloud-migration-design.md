# Prod Cloud Migration: Mac → Ubuntu ECS

**Date:** 2026-05-01
**Predecessor:** [docs/superpowers/specs/2026-04-30-prod-init-clean-slate-design.md](../specs/2026-04-30-prod-init-clean-slate-design.md) — produced the clean post-init state that this migration moves to the cloud.

> **Status: DRAFT awaiting user review.** The user requested a spec without iterative Q&A, so several decisions are baked in as defaults and listed under [§11 Open Questions](#11-open-questions). Read those first; everything else assumes those answers.

## 1. Summary

Move the live HackForger instance — currently running on a developer Mac at `localhost:3000`, exposed as `https://hackforger.inside.h2os.cloud` via local Caddy + Tailscale — to a dedicated Ubuntu 24.04 ECS at `218.91.114.178` (Tailscale IP TBD). The application stack (gitea binary, PostgreSQL 18, forgejo-runner, Caddy) is installed from scratch on the ECS using **binaries managed by systemd**, no Docker. Data is dumped from the Mac and restored on the ECS in a single maintenance window. After cutover the Mac becomes a warm standby and a daily rsync pulls fresh backups from the ECS for local-disk redundancy.

## 2. Goals & Non-Goals

**Goals**

- Stand up postgres + gitea + runner + caddy on the ECS as systemd services starting at boot.
- One-shot data migration of the post-clean-slate state from Mac to ECS (DB + repos + attachments + custom assets), preserving identifiers (user IDs, hackathon IDs, repo IDs, OAuth source).
- Same external URL `https://hackforger.inside.h2os.cloud` keeps working with no end-user-visible breaking change beyond a brief maintenance window.
- Daily rsync from ECS back to Mac with N-day rotation, so the developer always has a recent local copy.
- Idempotent install scripts checked into the repo (under `deploy/`), so a new operator can rebuild the box from this spec.

**Non-Goals**

- HA / multi-node failover (single-box deployment).
- Containerization. We use raw binaries + systemd; revisit Docker only if scale or tenant-isolation demands it.
- Decommissioning the Mac — it stays as warm standby for at least 30 days post-cutover, with the binary kept buildable but the gitea daemon stopped.
- Migrating the OAuth provider (Logto). Existing `login_source` rows carry over verbatim; if the Logto callback URL ever needs to change we treat it as a separate change.
- Setting up CI/CD that pushes new gitea builds to the ECS automatically. Initial deploys are manual `scp` + `systemctl restart`; automation can come later.
- Public-internet exposure. The ECS is reached over Tailscale; there is no plan to put `hackforger.inside.h2os.cloud` on a public DNS.

## 3. Current State (Mac, the source)

```
                                                  ┌──────────────────────────────────────┐
                                                  │ Mac (h2oslabs workstation)           │
                                                  │                                      │
   tailnet: hackforger.inside.h2os.cloud  ──────► │  Caddy launchd  → reverse_proxy 3000 │
                                                  │  ./gitea web (port 3000)             │
                                                  │     ↕                                │
                                                  │  Postgres 18.3 (Homebrew, port 5432) │
                                                  │     hackforger DB    ~19 MB          │
                                                  │  data/forgejo-repositories  8.2 MB   │
                                                  │  data/attachments          11 MB     │
                                                  │  data/avatars              64 KB     │
                                                  │  custom/public            20 MB      │
                                                  │  forgejo-runner (port 3000 target)   │
                                                  └──────────────────────────────────────┘
```

Total payload ≈ 58 MB. Tiny, so rsync transfer time is not a constraint.

## 4. Target State (ECS, the destination)

```
                                                ┌──────────────────────────────────────────┐
                                                │ ecs-synnovator (Ubuntu 24.04, 2c/3.6GiB) │
                                                │ public NAT 218.91.114.178                │
                                                │ tailnet IP <reassigned to ECS>           │
                                                │                                          │
   tailnet: hackforger.inside.h2os.cloud  ────► │  caddy.service        :443 / :80         │
                                                │     ↓ reverse_proxy 127.0.0.1:3000       │
                                                │  gitea.service        :3000 (loopback)   │
                                                │     ↕ TCP localhost                      │
                                                │  postgresql.service   :5432 (loopback)   │
                                                │     hackforger DB                        │
                                                │  forgejo-runner.service                  │
                                                │     registered against 127.0.0.1:3000    │
                                                │                                          │
                                                │  /var/lib/hackforger/                    │
                                                │    ├── data/         (gitea WORK_PATH)   │
                                                │    ├── custom/       (app.ini + assets)  │
                                                │    └── pg-backups/   (nightly dumps)     │
                                                │  /opt/hackforger/                        │
                                                │    └── gitea         (binary)            │
                                                └──────────────────────────────────────────┘
```

All four daemons run **on the same box, bound to loopback**, so internal traffic never leaves the host. Only Caddy listens on the Tailscale-facing IP (and on `:80` for HTTP→HTTPS redirect, if at all). SSH (22) remains the only other external port.

## 5. Components & Versions

| Component        | Version                  | Source                                | Reason                                                                       |
| ---------------- | ------------------------ | ------------------------------------- | ---------------------------------------------------------------------------- |
| OS               | Ubuntu 24.04.2 LTS noble | preinstalled                          | Already there. LTS through 2029.                                             |
| PostgreSQL       | **18.x**                 | PGDG apt repo (`apt.postgresql.org`)  | Mac runs 18.3; matching majors avoids `pg_dump` cross-version friction.      |
| gitea (HackForger fork) | current `v0.1-dev/hackforger` HEAD | cross-compiled `GOOS=linux GOARCH=amd64`  on the Mac, scp'd | Binary mode per user direction. Avoids needing Go toolchain on the ECS. |
| forgejo-runner   | v0.3.0 (matches local)   | downloaded from code.forgejo.org      | One process, host mode (no Docker initially — see §11 Q3).                   |
| Caddy            | latest stable            | Cloudsmith repo (the official apt source) | Auto-HTTPS via Tailscale's `tailscale cert` (no public Let's Encrypt needed). |
| Tailscale        | latest                   | tailscale.com apt repo                | Already required to make `inside.h2os.cloud` reachable.                      |

Out of scope: anti-virus, fail2ban, monitoring beyond the existing telegraf/harvest agent that's already running.

## 6. Network & TLS

**DNS / hostname**:
- `hackforger.inside.h2os.cloud` currently resolves to `100.64.0.27` (a Tailscale CGNAT IP, almost certainly the Mac's tailnet IP).
- After cutover the same hostname must resolve to the ECS's tailnet IP.
- **Mechanism (assumption):** Tailscale MagicDNS resolves machine-name → tailnet IP automatically; the `inside.h2os.cloud` zone is presumably a CNAME/manual A record managed in your DNS provider. We need to flip that one record at cutover time. (See §11 Q1.)

**TLS termination**: Caddy on the ECS, using `tailscale cert` to obtain a certificate for `hackforger.inside.h2os.cloud`. No public Let's Encrypt challenge required since the host is not publicly addressable. This mirrors how the Mac Caddy works today.

**Firewall**: Tailscale ACLs already restrict who can reach this hostname. On the OS we keep the default (no ufw), since SSH and Caddy ports already need to be open and there's nothing else listening externally. If you want defense-in-depth, ufw with `allow 22, 80, 443` is a one-liner and can be added.

**Loopback binding**: gitea, postgres, runner all bind `127.0.0.1` only. Caddy is the only externally-bound process besides SSH.

## 7. Data Migration: Mac → ECS

A single-shot copy executed during the maintenance window (§8). Total payload ~58 MB, transfer time over Tailscale ≈ seconds.

```
[Mac]                                          [ECS]
─────                                          ─────
1. Stop ./gitea web (graceful)
2. pg_dumpall -h 127.0.0.1 -U hackforger \
     > /tmp/hackforger-prod.sql
3. tar czf /tmp/hackforger-data.tgz \
     -C /Users/h2oslabs/Workspace/hackforger \
     data/forgejo-repositories \
     data/attachments \
     data/avatars \
     data/lfs \
     custom/conf/app.ini \
     custom/public \
     custom/templates
4. rsync over Tailscale ───────────────────►  /tmp/hackforger-prod.sql
                                              /tmp/hackforger-data.tgz
                                          5. sudo -u postgres psql \
                                                < /tmp/hackforger-prod.sql
                                          6. tar xzf /tmp/hackforger-data.tgz \
                                                -C /var/lib/hackforger
                                          7. chown -R hackforger:hackforger \
                                                /var/lib/hackforger
                                          8. Adjust /var/lib/hackforger/custom/conf/app.ini:
                                                  ROOT       = /var/lib/hackforger/data/forgejo-repositories
                                                  WORK_PATH  = /var/lib/hackforger
                                                  DOMAIN, ROOT_URL ← unchanged (same hostname)
                                                  HTTP_ADDR  = 127.0.0.1
                                                  HTTP_PORT  = 3000
                                                  PG password ← regenerated, stored in .env
                                          9. systemctl start postgresql gitea
                                         10. Re-register forgejo-runner against
                                                 http://127.0.0.1:3000 (token from
                                                 admin UI; old runner record can be
                                                 deleted via API).
```

Identifiers (user IDs, hackathon IDs, repo IDs) are preserved verbatim by `pg_dumpall`. The 12 prod hackathons created during clean-slate keep the same slugs and numeric IDs, so any external links that already reference them stay valid.

**Adjustments to app.ini that matter**: `ROOT` (repo storage path) and `WORK_PATH` change from the Mac's repo-relative paths to absolute `/var/lib/hackforger/...`. Postgres password rotates because the Mac one ends up in plain `.env` and shouldn't travel; we generate a fresh one on the ECS and write it both into `pg_hba`/`ALTER USER` and into the new `.env`.

**Secrets that DO carry over**: `INTERNAL_TOKEN`, `JWT_SECRET`, `OAUTH2_JWT_SECRET`, OAuth `login_source` config in DB. Changing these would invalidate sessions, JWTs, and break Logto sign-in.

## 8. Cutover Plan

A maintenance window (target: 30 minutes, hard limit: 60 minutes). Announced in advance.

```
T-1d   Notify users, freeze any planned hackathon edits
T-1h   Final dry-run rsync (no service stop) to warm caches and verify scripts
T0     Begin window:
       (a) Stop Mac ./gitea web → 503 from Caddy on Mac
       (b) Run §7 steps 1-4 (dump + tar + rsync)
       (c) Run §7 steps 5-9 on ECS
       (d) Smoke test inside the ECS:
             curl -fsS http://127.0.0.1:3000/api/v1/version
             curl -fsS http://127.0.0.1:3000/api/v1/admin/users -u hackforger:$PASS
       (e) Flip DNS A-record for inside.h2os.cloud → ECS tailnet IP
       (f) Wait for DNS propagation (TTL-dependent, target <2 min)
       (g) Verify externally: curl -fsS https://hackforger.inside.h2os.cloud/
       (h) Re-register runner + verify a test workflow_dispatch picks up
T+30m  Window ends. If anything is wrong → rollback by flipping DNS back
       and starting Mac gitea (data on Mac is unchanged).
T+1d   Confirm no unexpected error logs on ECS, runner heartbeat steady,
       backup rsync ran overnight (§9).
T+30d  Decommission decision: keep Mac warm or wipe.
```

**Rollback**: The Mac's data dir is untouched during cutover. If ECS bring-up fails for any reason, flipping DNS back + `./gitea web` on Mac restores the previous state in <2 minutes. The dump file on the Mac (`/tmp/hackforger-prod.sql`) is also retained as a tertiary fallback.

## 9. Backup Pipeline (ECS → Mac)

Daily rsync from ECS to a Mac directory, with a small retention rotation. This gives the developer a recent local copy in addition to whatever cloud snapshots Huawei ECS provides.

**On the ECS** — a systemd timer + service that runs `pg_dump` every night at 04:00 China time:

```
/etc/systemd/system/hackforger-backup.service
  ExecStart=/usr/local/bin/hackforger-backup.sh
  User=hackforger

/etc/systemd/system/hackforger-backup.timer
  OnCalendar=*-*-* 04:00:00 Asia/Shanghai
  Persistent=true

/usr/local/bin/hackforger-backup.sh:
  pg_dump -Fc -h 127.0.0.1 -U hackforger hackforger \
      > /var/lib/hackforger/pg-backups/hackforger-$(date +%Y%m%d).pgc
  find /var/lib/hackforger/pg-backups -name 'hackforger-*.pgc' \
      -mtime +14 -delete
```

So the ECS itself keeps 14 days of compressed dumps locally.

**On the Mac** — a launchd job (matching the existing `com.h2os.*` pattern) that pulls from the ECS at 04:30 China time, half an hour after the dump finishes:

```
~/Library/LaunchAgents/com.h2os.hackforger-backup-pull.plist
  StartCalendarInterval: hour=4, minute=30

scripts/backup-pull.sh:
  set -euo pipefail
  DEST="$HOME/Backups/hackforger"
  mkdir -p "$DEST"/{db,data}
  rsync -az --delete \
      hackforger@hackforger.inside.h2os.cloud:/var/lib/hackforger/pg-backups/ \
      "$DEST/db/"
  rsync -az --delete \
      hackforger@hackforger.inside.h2os.cloud:/var/lib/hackforger/data/ \
      "$DEST/data/"
  rsync -az --delete \
      hackforger@hackforger.inside.h2os.cloud:/var/lib/hackforger/custom/ \
      "$DEST/custom/"
  echo "$(date -Iseconds) backup-pull OK" >> "$DEST/backup.log"
```

What this gives us on the Mac, every morning:
- `~/Backups/hackforger/db/` — last 14 daily PG dumps
- `~/Backups/hackforger/data/` — current state of all repos, attachments, avatars (mirror)
- `~/Backups/hackforger/custom/` — current app.ini and custom assets

Disk impact on Mac: <100 MB at current scale, trivially small.

**Auth**: SSH key-based, same key already in `~/.ssh/` on the Mac that today reaches the ECS as `hackforger@218.91.114.178`. Tailscale guarantees the connection.

**Failure handling**: launchd restarts a failed run on the next scheduled trigger; `backup.log` records each attempt. If the user wants alerting, that's a follow-up.

## 10. Repository Layout for the Migration

New top-level `deploy/` directory in the project repo, checked in:

```
deploy/
├── README.md                  # how to use this dir
├── ecs-bootstrap.sh           # idempotent install: PG, runner, caddy, dirs, systemd units
├── systemd/
│   ├── gitea.service
│   ├── forgejo-runner.service
│   ├── caddy.service          (only if not from caddy package)
│   ├── hackforger-backup.service
│   └── hackforger-backup.timer
├── caddy/
│   └── Caddyfile.tmpl
├── app.ini.tmpl               # production app.ini template, env-substituted
└── migrate-data.sh            # the §7 dump+rsync+restore one-shot
```

Plus a Mac-side script for the backup puller:
```
scripts/backup-pull.sh         # runs on Mac via launchd
```

And a launchd plist:
```
deploy/launchd/com.h2os.hackforger-backup-pull.plist
```
(checked in for reference; the install copies it to `~/Library/LaunchAgents/`).

## 11. Open Questions

These are baked-in defaults you should confirm or override **before** writing the implementation plan.

| # | Topic                                | Default I assumed                                                                                       | Why I picked it                                                                                                            |
| - | ------------------------------------ | ------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| 1 | DNS flip mechanism                   | Manual A-record update at cutover (your DNS provider for `inside.h2os.cloud`)                           | Tailscale MagicDNS would auto-route by machine name, but `inside.h2os.cloud` is presumably a real DNS zone, so a flip is needed. **You confirm the zone owner / TTL.** |
| 2 | Maintenance window length            | 30 min target, 60 min hard limit                                                                        | Data is 58 MB, so the bottleneck is humans not bytes. Confirm acceptable.                                                  |
| 3 | Runner mode on Linux                 | **Host mode** initially (matches "binary deploy" intent), Docker mode deferred                          | No Docker dependency, simplest. Tradeoff: jobs run as the runner user, no isolation. Acceptable while only HackForger-authored workflows exist. Switch to docker when user-defined CI becomes a thing. |
| 4 | Postgres major version on Linux      | **18.x** via PGDG repo (matches Mac 18.3)                                                               | Cross-version `pg_dump` is awkward; matching avoids it.                                                                    |
| 5 | Backup retention                     | 14 days on ECS + same 14 days on Mac (delete older)                                                     | 14 daily dumps × ~5 MB ≈ 70 MB. Cheap. Adjust if you want monthly archives kept longer.                                    |
| 6 | Backup time                          | 04:00 ECS dump, 04:30 Mac pull (Asia/Shanghai)                                                          | Off-peak, after midnight cron-y workloads. Pull lags dump by 30 min so the file exists.                                    |
| 7 | Sudo password handling during install | Operator runs `ecs-bootstrap.sh` interactively, types sudo password when prompted                       | Avoids putting sudoers NOPASSWD in scope. Install is one-time.                                                             |
| 8 | What to do with the test instance on Mac | Leave it (port 3001) untouched. Backup pipeline only covers prod, not test.                          | Test instance is local-only, ephemeral state.                                                                              |
| 9 | OAuth callback URL                   | Unchanged — Logto already calls back to `https://hackforger.inside.h2os.cloud/...`                      | Hostname stays the same, so callback works. **Confirm Logto admin doesn't pin to an IP.**                                  |
| 10 | First-deploy automation              | Manual `make build-linux` + `scp` + `systemctl restart gitea` for now                                  | Automation (CI build + deploy hook) is a future PR.                                                                        |

## 12. Acceptance Criteria

The migration is "done" when **all** of these hold:

- `https://hackforger.inside.h2os.cloud/` returns HTTP 200 and renders the landing page, served by the ECS.
- Login as `hackforger` admin succeeds (via Logto, on ECS).
- All 12 prod hackathons are present with correct slugs/IDs (matching the post-clean-slate state).
- A test workflow_dispatch picks up on the ECS runner and pushes to its track repo.
- Mac `./gitea web` is stopped (no port 3000 listener for >24h).
- A nightly backup file appears in `~/Backups/hackforger/db/` on the Mac, ≤24h old, restorable via `pg_restore --list`.
- Repository contains `deploy/` with all listed files; `ecs-bootstrap.sh` is idempotent (running twice is a no-op).

## 13. Risks

| Risk                                            | Likelihood | Impact | Mitigation                                                          |
| ----------------------------------------------- | ---------- | ------ | ------------------------------------------------------------------- |
| Cross-arch binary corruption during scp          | Low        | High   | `sha256sum` check after transfer; `gitea --version` smoke test       |
| Postgres `pg_dumpall` from 18.3 → restore on 18.x has a syntax incompatibility | Low | Medium | We're matching majors; if any error, fall back to `--inserts` mode |
| DNS flip slower than expected                    | Medium     | Medium | Lower TTL to 60 s 24h before cutover; have rollback ready           |
| Tailscale auth on ECS expires/never set up       | Medium     | High   | Step 0 of `ecs-bootstrap.sh` is `tailscale up` + auth-key prompt    |
| Mac launchd backup job silently fails            | Medium     | Low    | `backup.log` + a sanity check task: "no entry in last 26h → alert"  |
| Runner can't reach gitea (loopback misconfigured) | Low        | Medium | Smoke test in cutover step (h)                                      |
| 3.6 GiB RAM tight under load                    | Low (initial) | Medium | Monitor; PG `shared_buffers` tuned conservatively (256 MB).         |

---

**Next step after approval:** invoke `superpowers:writing-plans` to produce the step-by-step implementation plan covering `deploy/` scaffolding, ECS bootstrap, dump+rsync+restore migration, Caddy config, systemd units, and the Mac backup-pull launchd job.
