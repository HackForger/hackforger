# deploy/ecs/ — HackForger production cloud deployment

Artifacts for deploying HackForger to the Huawei Cloud ECS at `218.91.114.178`,
serving `https://www.synnovator.com` as native binaries managed by systemd.

> **Not to be confused with `deploy/caddy/`** (sibling dir) — that's the **Mac
> developer instance**'s Caddy + Tailscale + alidns setup for
> `hackforger.inside.h2os.cloud`. This `ecs/` dir is the **public cloud**
> deployment.

**Spec:** [`docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md`](../../docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md)
**Plan:** [`docs/superpowers/plans/2026-05-01-prod-cloud-migration.md`](../../docs/superpowers/plans/2026-05-01-prod-cloud-migration.md)
**Cutover runbook:** [`cutover-runbook.md`](cutover-runbook.md)

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
| `../../scripts/backup-pull.sh` | Mac | What the launchd job invokes. |

## Order of operations (first-time deploy)

1. **Manual prerequisites (operator):**
   - Open inbound :80 + :443 in the Huawei Cloud security group for ECS 218.91.114.178.
   - In the Aliyun DNS console: lower `synnovator.com` zone TTL to 60s, add A record `www → 218.91.114.178`.
   - In Logto admin: add `https://www.synnovator.com/user/oauth2/<source-name>/callback` to the allowed callback URLs.
2. `bash deploy/ecs/build-linux.sh` on the Mac → produces `gitea-linux-amd64`.
3. `scp -r deploy/ecs hackforger@218.91.114.178:/tmp/` and `bash /tmp/ecs/ecs-bootstrap.sh` (interactive sudo).
4. `scp gitea-linux-amd64 hackforger@218.91.114.178:/opt/hackforger/gitea`.
5. Run `bash deploy/ecs/migrate-data.sh --confirm` from the Mac during the maintenance window.
6. Follow `cutover-runbook.md`.

## Subsequent deploys (after first cutover)

```bash
bash deploy/ecs/build-linux.sh
scp gitea-linux-amd64 hackforger@218.91.114.178:/opt/hackforger/gitea-new
ssh hackforger@218.91.114.178 \
  'sudo mv /opt/hackforger/gitea-new /opt/hackforger/gitea && sudo systemctl restart gitea'
```
