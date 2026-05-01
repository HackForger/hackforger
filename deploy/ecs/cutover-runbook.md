# Cutover Runbook — Mac → ECS migration

Window: target 30 min, hard limit 60 min.
Spec: [`docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md`](../../docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md)

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
      bash deploy/ecs/build-linux.sh
      file gitea-linux-amd64   # confirm: ELF 64-bit LSB ... x86-64
      scp gitea-linux-amd64 hackforger@218.91.114.178:/opt/hackforger/gitea
      ssh hackforger@218.91.114.178 'sudo chmod +x /opt/hackforger/gitea && /opt/hackforger/gitea --version'
      ```
- [ ] Dry-run the data migration (no service stop):
      ```bash
      bash deploy/ecs/migrate-data.sh --dry-run
      ```
      Files should appear at `/tmp/hackforger-prod.pgc` and `/tmp/hackforger-data.tgz` on the ECS.

## T0 — Cutover

### (a) Stop Mac gitea (announces 503 to anyone hitting old URL)

```bash
PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t)
kill "$PID"
```

### (b–c) Run the migration script

```bash
bash deploy/ecs/migrate-data.sh --confirm
```

Wait for "Restore complete" output. Ignore the last log line about "start gitea caddy" — we do that next.

### (d) Start services on ECS

```bash
ssh hackforger@218.91.114.178 'sudo systemctl start gitea && sudo systemctl start caddy'
```

### (e) Smoke test internally on ECS

```bash
ssh hackforger@218.91.114.178 '
  curl -fsS http://127.0.0.1:3000/api/v1/version
'
```

Expected: JSON with version.

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
ADMIN_PASS=$(grep '^HACKFORGER_ADMIN_PASSWORD=' .env | cut -d= -f2)
TOKEN=$(curl -fsS -u hackforger:$ADMIN_PASS \
  https://www.synnovator.com/api/v1/admin/runners/registration-token | jq -r .token)

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
curl -fsS -u hackforger:$ADMIN_PASS https://www.synnovator.com/api/v1/admin/runners | jq
```

## Rollback (if any of d–k fails irrecoverably)

1. In Aliyun DNS, revert A record to old value (or remove it; the apex `synnovator.com` still resolves to 47.x)
2. Restart Mac gitea: `bash scripts/restart-gitea.sh`
3. Old domain `hackforger.inside.h2os.cloud` independently still works (Tailscale)
4. Diagnose ECS state offline; re-run migrate-data.sh after fix

Total rollback latency: ~3 min (60s TTL + service restart).

## T+24h — Verify backup pipeline

```bash
ls -lt ~/Backups/hackforger/db/ | head -5
tail -20 ~/Backups/hackforger/backup.log
# Expect a fresh hf-YYYYMMDD-HHMMSSZ.pgc ≤24h old
```

If no fresh dump:
```bash
launchctl list | grep hackforger-backup
launchctl print "gui/$UID/com.h2os.hackforger-backup-pull" | head -30
```

Check the ECS-side timer:
```bash
ssh hackforger@218.91.114.178 'systemctl list-timers hackforger-backup.timer'
```

## T+30d — Decommission decision

- If no problems: stop Mac gitea permanently, archive Mac's `data/` to cold storage.
- If something needs Mac for fallback: keep warm-standby until decision-day-2.
