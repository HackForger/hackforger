# Prod Cloud Migration: Mac → Ubuntu ECS (Binary Deployment, www.synnovator.com)

**Date:** 2026-05-01
**Predecessor:** [docs/superpowers/specs/2026-04-30-prod-init-clean-slate-design.md](2026-04-30-prod-init-clean-slate-design.md) — produced the clean post-init state on the Mac that this migration moves to the cloud.

## 1. Summary

Move the live HackForger instance — currently on a developer Mac at `localhost:3000`, exposed via local Caddy + Tailscale at `https://hackforger.inside.h2os.cloud` — to a dedicated **Huawei Cloud ECS at `218.91.114.178`** and re-front it under the **public domain `https://www.synnovator.com`**. The application stack (PostgreSQL 18, gitea binary, forgejo-runner, Caddy) is installed from scratch as **native binaries managed by systemd**; no containers in production. Data is dumped from the Mac and restored on the ECS in a single maintenance window. After cutover the Mac becomes warm standby and a daily rsync pulls fresh backups from the ECS.

The other cloud box at `47.116.160.70` (Aliyun) is **unrelated to this migration** — it hosts shared synnovator infra (private docker registry, traefik, oneauth, etc.). HackForger does not run there. It does not run on it, but its existence is noted because it currently owns `synnovator.com` (the apex domain), and the DNS for `www.synnovator.com` will be flipped from Aliyun-side routing to the new ECS.

## 2. Goals & Non-Goals

**Goals**

- Stand up postgres + gitea + runner + caddy on the ECS as systemd services starting at boot.
- One-shot data migration of the post-clean-slate state from Mac to ECS (DB + repos + attachments + custom assets), preserving identifiers (user IDs, hackathon IDs, repo IDs, OAuth source).
- `https://www.synnovator.com` resolves to the ECS, terminates TLS via Caddy + Let's Encrypt, serves HackForger.
- Daily rsync from ECS back to Mac with a 14-day rotation.
- Idempotent install scripts checked into the repo (under `deploy/`), so a new operator can rebuild the box from this spec.

**Non-Goals**

- HA / multi-node failover (single-box deployment).
- Containerization of the app stack. Binary + systemd is intentional — see §11 Q1 for the reasoning.
- Decommissioning the Mac — it stays as warm standby for at least 30 days post-cutover.
- Migrating the OAuth provider (Logto). Existing `login_source` rows carry over verbatim. **Logto admin will need a new callback URL added: `https://www.synnovator.com/...`** — see §11 Q3.
- Touching the Aliyun box at 47.116.160.70 (other than read-only inspection that already happened).
- CI/CD that auto-deploys new gitea builds. Initial deploys are manual `scp` + `systemctl restart`.

## 3. Current State (Mac, the source)

```
                                                  ┌──────────────────────────────────────┐
                                                  │ Mac (h2oslabs workstation)           │
                                                  │                                      │
   tailnet: hackforger.inside.h2os.cloud  ──────► │  Caddy (launchd)  → 127.0.0.1:3000   │
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

Total payload ≈ 58 MB. rsync time over WAN is bounded by setup, not bytes.

## 4. Target State (Huawei ECS, the destination)

```
                                              ┌────────────────────────────────────────────┐
   public DNS:                                │ Huawei ECS 218.91.114.178                  │
   www.synnovator.com  ─────────────────────► │ (Ubuntu 24.04, 2c/3.6GiB, 40GB)            │
   (Aliyun DNS console flips A record)        │                                            │
                                              │  caddy.service       :80 / :443            │
                                              │     ↓ reverse_proxy 127.0.0.1:3000         │
                                              │  gitea.service       :3000 (loopback only) │
                                              │     ↕ TCP localhost                        │
                                              │  postgresql.service  :5432 (loopback only) │
                                              │     hackforger DB                          │
                                              │  forgejo-runner.service                    │
                                              │     registered against 127.0.0.1:3000      │
                                              │     host mode (jobs run as runner user)    │
                                              │                                            │
                                              │  /var/lib/hackforger/                      │
                                              │    ├── data/        (gitea WORK_PATH)      │
                                              │    ├── custom/      (app.ini + assets)     │
                                              │    └── pg-backups/  (nightly dumps)        │
                                              │  /opt/hackforger/                          │
                                              │    └── gitea        (binary, systemd's     │
                                              │                      ExecStart points here)│
                                              └────────────────────────────────────────────┘
```

All four daemons run on the same box. **Postgres, gitea, runner bind `127.0.0.1` only.** Only Caddy listens on the public IP (port 80 for the LE challenge + HTTP→HTTPS redirect, port 443 for serving). SSH (22) is the only other external port.

## 5. Components & Versions

| Component        | Version                  | Source                                 | Reason                                                                  |
| ---------------- | ------------------------ | -------------------------------------- | ----------------------------------------------------------------------- |
| OS               | Ubuntu 24.04.2 LTS noble | preinstalled                           | Already there. LTS through 2029.                                        |
| PostgreSQL       | **18.x**                 | PGDG apt repo (`apt.postgresql.org`)   | Mac runs 18.3; matching majors avoids `pg_dump` cross-version friction. |
| gitea (HackForger fork) | current `v0.1-dev/hackforger` HEAD | cross-compiled `GOOS=linux GOARCH=amd64` on the Mac, scp'd | Binary mode per user direction. Avoids needing Go toolchain on the ECS. |
| forgejo-runner   | v0.3.0                   | downloaded from code.forgejo.org       | One process, **host mode** (no Docker initially — see §11 Q1).          |
| Caddy            | latest stable (apt)      | Cloudsmith repo (`apt.fury.io/caddy`)  | Auto-HTTPS via Let's Encrypt HTTP-01 challenge.                         |

**Cross-compile recipe** (Mac, no Go toolchain mess):

```bash
# Run on the Mac. Output: ./gitea-linux-amd64
docker run --rm --platform linux/amd64 \
  -v "$PWD":/src -w /src \
  golang:1.23-bookworm bash -c '
    apt-get update -qq && apt-get install -y -qq build-essential nodejs npm git
    TAGS="bindata pam sqlite sqlite_unlock_notify" \
      GOOS=linux GOARCH=amd64 make build
    mv gitea gitea-linux-amd64
  '
```

The output is a static binary; scp it to `/opt/hackforger/gitea` on the ECS and `systemctl restart gitea`. The docker container is purely a clean build environment — production runs no docker. This keeps the dev-vs-prod story consistent ("everywhere it's just a binary") without polluting the Mac with Linux Go toolchain.

## 6. Network & TLS

**Public exposure**:

```
Internet ──► Huawei ECS public IP 218.91.114.178
              ├─ :22  SSH (already open, key-based only)
              ├─ :80  Caddy (HTTP → HTTPS redirect + LE HTTP-01 challenge)
              └─ :443 Caddy (TLS termination, reverse_proxy to gitea)
```

**Prerequisites that REQUIRE manual ops action** (called out in §11 Q4 as well):

| Action | Where | Owner |
| ------ | ----- | ----- |
| Open inbound :80 + :443 in Huawei security group | Huawei Cloud console → ECS → security group | You |
| Add A record `www.synnovator.com → 218.91.114.178`, TTL 60s | Aliyun DNS console (`hichina.com` zone) | You |
| Drop TTL to 60s 24h before cutover (so rollback is fast) | Aliyun DNS console | You |

**TLS**: Caddy automatically requests a Let's Encrypt cert for `www.synnovator.com` on first start. Requires `:80` reachable from the LE servers. The Caddyfile is two lines:

```
www.synnovator.com {
    reverse_proxy 127.0.0.1:3000
}
```

(Caddy embeds ACMEv2 client. No `certbot` needed.)

**Loopback binding** for non-Caddy services prevents accidental exposure if firewall rules ever loosen.

**No Tailscale**: The Mac instance used Tailscale because the hostname was internal. The ECS instance is public-internet-facing under a real DNS name; Tailscale is not part of the prod path. (It can still be installed on the ECS for ops convenience, but that's optional and not part of this spec.)

## 7. Data Migration: Mac → ECS

A single-shot copy executed during the maintenance window (§8). Total payload ~58 MB.

```
[Mac]                                          [ECS 218.91.114.178]
─────                                          ─────────────────────
1. Stop ./gitea web (graceful)
2. pg_dump -Fc -h 127.0.0.1 -U hackforger \
     hackforger > /tmp/hackforger-prod.pgc
3. tar czf /tmp/hackforger-data.tgz \
     -C /Users/h2oslabs/Workspace/hackforger \
     data/forgejo-repositories \
     data/attachments \
     data/avatars \
     data/lfs \
     custom/conf/app.ini \
     custom/public \
     custom/templates
4. rsync over public internet (SSH) ────────►  /tmp/hackforger-prod.pgc
                                                /tmp/hackforger-data.tgz
                                            5. sudo -u postgres createdb hackforger -O hackforger
                                            6. pg_restore -h 127.0.0.1 -U hackforger \
                                                  -d hackforger /tmp/hackforger-prod.pgc
                                            7. tar xzf /tmp/hackforger-data.tgz \
                                                  -C /var/lib/hackforger
                                            8. chown -R hackforger:hackforger \
                                                  /var/lib/hackforger
                                            9. Edit /var/lib/hackforger/custom/conf/app.ini:
                                                  ROOT       = /var/lib/hackforger/data/forgejo-repositories
                                                  WORK_PATH  = /var/lib/hackforger
                                                  DOMAIN     = www.synnovator.com
                                                  ROOT_URL   = https://www.synnovator.com/
                                                  HTTP_ADDR  = 127.0.0.1
                                                  HTTP_PORT  = 3000
                                                  [database]
                                                    HOST   = 127.0.0.1:5432
                                                    PASSWD = <freshly generated, written to .env>
                                           10. systemctl start postgresql gitea caddy
                                           11. Re-register forgejo-runner against
                                                  http://127.0.0.1:3000 (token from
                                                  admin UI on the new instance).
```

`pg_restore` preserves user IDs, hackathon IDs, repo IDs verbatim. The 12 prod hackathons created during clean-slate keep their slugs and IDs. Internal references (foreign keys, OAuth `login_source` row) survive intact.

**Adjustments to app.ini that matter**:

- `ROOT` and `WORK_PATH` move to absolute `/var/lib/hackforger/...`.
- `DOMAIN` and `ROOT_URL` change from `hackforger.inside.h2os.cloud` to `www.synnovator.com`.
- `[database] PASSWD` rotates (we generate a new strong PG password during bootstrap; the Mac password should not travel across networks).

**Secrets that DO carry over verbatim** (because changing them would break things):

- `INTERNAL_TOKEN` (gitea ↔ runner internal API)
- `JWT_SECRET`, `OAUTH2_JWT_SECRET` (would invalidate all sessions and JWTs)
- OAuth `login_source` rows in DB (Logto config; new callback URL must be added on the Logto side, see §11 Q3)

## 8. Cutover Plan

A maintenance window: target 30 minutes, hard limit 60 minutes. Announced in advance.

```
T-1d   Drop DNS TTL for www.synnovator.com to 60s (Aliyun console)
       Notify users
T-1h   Final dry-run rsync (no service stop) to warm caches and verify scripts
T0     Begin maintenance window:
       (a) Stop Mac ./gitea web → 503 from Mac Caddy
       (b) Run §7 steps 1–4 (dump + tar + rsync)
       (c) Run §7 steps 5–10 on ECS
       (d) Smoke test inside the ECS (still pre-DNS-flip):
             curl -fsS http://127.0.0.1:3000/api/v1/version
             curl -fsS http://127.0.0.1:3000/api/v1/admin/users \
                  -u hackforger:$NEW_PASS
       (e) From outside, hit the bare IP to confirm Caddy is up:
             curl -kfsS https://218.91.114.178/
             (will fail TLS hostname check — that's fine, just confirming serve)
       (f) Add/flip A record for www.synnovator.com → 218.91.114.178
             (Aliyun DNS console)
       (g) Wait for DNS propagation (TTL ≤60s):
             dig @8.8.8.8 www.synnovator.com +short
             dig @114.114.114.114 www.synnovator.com +short
       (h) Verify externally (browser + curl):
             curl -fsSL https://www.synnovator.com/
             curl -fsSL https://www.synnovator.com/api/v1/version
       (i) Caddy gets its first LE cert in this step; check journalctl:
             journalctl -u caddy -n 50 | grep -i "obtained certificate"
       (j) Add new callback URL in Logto admin, test login
       (k) Re-register runner; verify a workflow_dispatch picks up
T+30m  Window ends. If anything is wrong → rollback (next paragraph).
T+1d   Confirm overnight backup ran (§9), no anomalous error logs
T+30d  Decommission decision: keep Mac warm or wipe.
```

**Rollback** (within the maintenance window):

Mac data dir is untouched. To undo:
1. Revert A record `www.synnovator.com` to its old value (or remove)
2. Restart `./gitea web` on Mac
3. (Old domain `hackforger.inside.h2os.cloud` was independent and still works for the dev's local Tailscale access)

DNS rollback latency is bounded by the 60s TTL we lowered in T-1d. Total rollback: ~3 minutes.

## 9. Backup Pipeline (ECS → Mac)

Daily rsync from ECS to a Mac directory, with a 14-day retention rotation.

**On the ECS** — systemd timer + service runs nightly:

```
/etc/systemd/system/hackforger-backup.service
  Type=oneshot
  User=hackforger
  ExecStart=/usr/local/bin/hackforger-backup.sh

/etc/systemd/system/hackforger-backup.timer
  OnCalendar=*-*-* 04:00:00 Asia/Shanghai
  Persistent=true

/usr/local/bin/hackforger-backup.sh:
  #!/bin/bash
  set -euo pipefail
  STAMP=$(date +%Y%m%d)
  pg_dump -Fc -h 127.0.0.1 -U hackforger hackforger \
      > /var/lib/hackforger/pg-backups/hf-${STAMP}.pgc
  find /var/lib/hackforger/pg-backups -name 'hf-*.pgc' \
      -mtime +14 -delete
```

ECS keeps 14 days of compressed dumps locally (~15 MB × 14 ≈ 210 MB).

**On the Mac** — launchd job pulls 30 minutes after the ECS dump finishes:

```
~/Library/LaunchAgents/com.h2os.hackforger-backup-pull.plist
  StartCalendarInterval: hour=4, minute=30 (system local time)

scripts/backup-pull.sh:
  #!/bin/bash
  set -euo pipefail
  DEST="$HOME/Backups/hackforger"
  mkdir -p "$DEST"/{db,data,custom}

  REMOTE=hackforger@218.91.114.178

  rsync -az --delete \
      "$REMOTE:/var/lib/hackforger/pg-backups/" "$DEST/db/"
  rsync -az --delete \
      "$REMOTE:/var/lib/hackforger/data/"        "$DEST/data/"
  rsync -az --delete \
      "$REMOTE:/var/lib/hackforger/custom/"      "$DEST/custom/"

  echo "$(date -Iseconds) backup-pull OK" >> "$DEST/backup.log"
```

What this gives the Mac, every morning:
- `~/Backups/hackforger/db/` — last 14 daily PG dumps
- `~/Backups/hackforger/data/` — current state of repos, attachments, avatars (mirror)
- `~/Backups/hackforger/custom/` — current app.ini and custom assets

Disk impact on Mac: <300 MB at current scale.

**Auth**: SSH key-based, the `id_ed25519` key already authorized on `hackforger@218.91.114.178`.

**Network**: Mac → ECS over public internet (DNS resolves to the public IP). Backup runs at off-peak China time, so ISP throttling is unlikely to bite.

**Failure handling**: launchd logs each run; missed runs reschedule on next trigger. `backup.log` is the audit trail. Alerting (email/IM on consecutive failures) is a follow-up task.

## 10. Repo Layout for the Migration

New top-level `deploy/` directory in the project repo:

```
deploy/
├── README.md                       # how to use this dir
├── ecs-bootstrap.sh                # idempotent install: PG, runner, caddy, dirs
├── build-linux.sh                  # docker-based cross-compile recipe (§5)
├── migrate-data.sh                 # the §7 dump+rsync+restore one-shot
├── systemd/
│   ├── gitea.service
│   ├── forgejo-runner.service
│   ├── hackforger-backup.service
│   └── hackforger-backup.timer
├── caddy/
│   └── Caddyfile.tmpl              # www.synnovator.com → :3000
└── app.ini.tmpl                    # production app.ini template
```

Plus on the Mac side:

```
scripts/backup-pull.sh                        # runs on Mac via launchd
deploy/launchd/com.h2os.hackforger-backup-pull.plist  # checked in for reference
```

`ecs-bootstrap.sh` is **idempotent**: re-running it on a partially-installed box should converge to the desired state without harm. This is the safety net for when something fails mid-install.

## 11. Open Questions

| # | Topic | Default I assumed | Notes |
| - | ----- | ----------------- | ----- |
| 1 | Runner mode on Linux | **Host mode** (matches "binary deploy" intent), Docker mode deferred | No Docker dependency, simplest. Tradeoff: jobs run as the runner user, no kernel-level isolation. Acceptable while only HackForger-authored workflows exist. |
| 2 | Postgres major version | **18.x** via PGDG repo (matches Mac 18.3) | Cross-version `pg_dump`/`restore` is risky; matching avoids it. |
| 3 | Logto callback URL | **Add `https://www.synnovator.com/...` to Logto admin's allowed callbacks before cutover** | The DB row carries over but Logto won't accept the new origin until you whitelist it. **You confirm timing.** |
| 4 | Manual ops actions | (see §6 prereqs table) | You do these. I'll have the spec / plan call them out as gates. |
| 5 | Backup retention | 14 days on ECS + same 14 days on Mac (via `--delete`) | 14 × ~15 MB ≈ 210 MB. Cheap. Adjust if you want monthly archives kept longer. |
| 6 | Backup time | 04:00 ECS dump, 04:30 Mac pull | Off-peak. Mac local time vs ECS Asia/Shanghai is the same time zone. |
| 7 | Sudo on the ECS | Operator runs `ecs-bootstrap.sh` interactively, types sudo password when prompted | Avoids putting NOPASSWD in scope. One-time cost. |
| 8 | Test instance | Mac test instance (port 3001) untouched. Backup pipeline only covers prod. | Test instance is local-only ephemeral state. |
| 9 | First-deploy automation | Manual `build-linux.sh` + `scp` + `systemctl restart gitea` | CI-driven deploy is a future PR. |
| 10 | What happens to `hackforger.inside.h2os.cloud` | Stays pointing to the Mac as long as the Mac instance is up (warm standby). After T+30d decommission, the DNS A record can be removed. | Old internal name still works as a fallback during the warm-standby window. |

## 12. Acceptance Criteria

The migration is "done" when **all** of these hold:

- `https://www.synnovator.com/` returns HTTP 200 and renders the HackForger landing page, served from the ECS.
- TLS cert is a valid Let's Encrypt cert for `www.synnovator.com` (`openssl s_client` shows correct CN/SAN).
- Login as `hackforger` admin succeeds (via Logto, with the new callback URL registered).
- All 12 prod hackathons are present with correct slugs/IDs (matching the post-clean-slate state).
- A test workflow_dispatch picks up on the ECS runner and pushes to its track repo.
- Mac `./gitea web` is stopped (no port 3000 listener for >24h).
- A nightly backup file appears in `~/Backups/hackforger/db/` on the Mac, ≤24h old, restorable via `pg_restore --list`.
- Repository contains `deploy/` with all listed files; `ecs-bootstrap.sh` is idempotent.

## 13. Risks

| Risk                                          | Likelihood | Impact | Mitigation                                                          |
| --------------------------------------------- | ---------- | ------ | ------------------------------------------------------------------- |
| Cross-arch build mismatch (arm64 → amd64 oversight) | Medium | High | `file ./gitea-linux-amd64` check before scp; smoke `./gitea --version` on ECS |
| Huawei security group not opened in time      | Medium | Critical (blocks LE + users) | Verify with `nc -z 218.91.114.178 80` from outside before T0 |
| LE rate-limit hit during repeated test cert pulls | Low | Medium | Use Caddy staging endpoint for rehearsal, switch to prod endpoint for the real cutover |
| PG `pg_dump 18 → restore 18` syntax issue | Low | Medium | Matching majors. If any error, fall back to `--inserts` mode.       |
| DNS propagation slower than 60s TTL implies   | Medium | Medium | Lower TTL 24h before. Verify with multiple resolvers (`8.8.8.8`, `114.114.114.114`). |
| Logto callback not whitelisted before cutover | Medium | High (login broken) | Q3 — do this in T-1h step.                                       |
| Mac launchd backup silently stops working     | Medium | Low | `backup.log` checked weekly; no entry in 26h → manual probe        |
| 3.6 GiB RAM tight under load                  | Low (initial) | Medium | Monitor; PG `shared_buffers` tuned conservatively (256 MB).         |
| OAuth session-cookie domain mismatch (cookies set for old hostname) | Low | Low (users re-login) | Cutover will sign everyone out; this is acceptable. |

---

**Next step after approval:** invoke `superpowers:writing-plans` to produce the step-by-step implementation plan covering `deploy/` scaffolding, ECS bootstrap, dump+rsync+restore migration, Caddy config, systemd units, and the Mac backup-pull launchd job.
