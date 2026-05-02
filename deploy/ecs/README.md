# deploy/ecs/ — HackForger production cloud deployment

Artifacts for deploying HackForger to the Huawei Cloud ECS at `203.119.115.130`,
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
| `ecs-bootstrap.sh` | ECS | Idempotent install of postgres, caddy, runner, dirs, systemd units. Run once. |
| `migrate-data.sh` | Mac+ECS | One-shot dump-rsync-restore the post-clean-slate prod DB + data dirs. Run once at cutover. |
| `redeploy.sh` | Mac→ECS | **Every subsequent deploy.** Build → rsync custom/ → scp binary → restart → record .last-deploy. |
| `preflight.sh` | Mac | Refuses to ship commits without admin-signed e2e reports. Called by `redeploy.sh`. |
| `backup-ecs.sh` | ECS | Nightly `pg_dump` + 14-day rotation. Runs from systemd timer. |
| `app.ini.tmpl` | ECS | Production `app.ini` template. Bootstrap script renders it. |
| `caddy/Caddyfile.tmpl` | ECS | Caddy config for `www.synnovator.com`. |
| `systemd/*.service` `*.timer` | ECS | systemd units installed by `ecs-bootstrap.sh`. |
| `launchd/com.h2os.hackforger-backup-pull.plist` | Mac | launchd job that pulls backups from ECS. |
| `../../scripts/backup-pull.sh` | Mac | What the launchd job invokes. |

## Order of operations (first-time deploy)

1. **Manual prerequisites (operator):**
   - Open inbound :80 + :443 in the Huawei Cloud security group for ECS 203.119.115.130.
   - In the Aliyun DNS console: lower `synnovator.com` zone TTL to 60s, add A record `www → 203.119.115.130`.
   - In Logto admin: add `https://www.synnovator.com/user/oauth2/<source-name>/callback` to the allowed callback URLs.
2. `bash deploy/ecs/build-linux.sh` on the Mac → produces `gitea-linux-amd64`.
3. `scp -r deploy/ecs hackforger@203.119.115.130:/tmp/` and `bash /tmp/ecs/ecs-bootstrap.sh` (interactive sudo).
4. `scp gitea-linux-amd64 hackforger@203.119.115.130:/opt/hackforger/gitea`.
5. Run `bash deploy/ecs/migrate-data.sh --confirm` from the Mac during the maintenance window.
6. Follow `cutover-runbook.md`.

## Subsequent deploys (after first cutover)

```bash
bash deploy/ecs/redeploy.sh
```

That's it. The script does:
1. **Preflight** — `preflight.sh` refuses to ship commits without admin-signed
   e2e reports (see `docs/notes/gitflow.md`).
2. **Build** — `build-linux.sh` cross-compiles via docker.
3. **Rsync custom/** — `custom/templates/` and `custom/public/` to the box.
   This is **load-bearing**: locale `.ini` files are baked into the binary
   via bindata, but template overrides and landing-page assets live on the
   filesystem. **Skipping this step silently breaks any PR that only
   touches templates or assets** — we hit this on the first cloud deploy.
4. **scp + atomic install** — binary lands at `/opt/hackforger/gitea`.
5. **Restart gitea** + record HEAD SHA in `/var/lib/hackforger/.last-deploy`.
6. **Smoke tests** — public `/api/v1/version`, navbar locale string, HTTPS 200.

Hotfix override (skip the smoke-test gate):
```bash
bash deploy/ecs/redeploy.sh --skip-preflight
```
…but write the report afterward — the gate exists so we don't rely on
discipline.

### Why `custom/` isn't auto-bundled into the binary

Forgejo's `custom/` directory is the override layer. Anything you drop into
`custom/templates/<path>` overrides the bindata-embedded template at the
same path; same for `custom/public/`. The build doesn't see these files
because they're in the runtime filesystem on the target host. So every
deploy needs to push them too.

If a future PR moves all template overrides under `templates/` (and gets
them into bindata), this step can drop out of `redeploy.sh`. Until then it
stays.
