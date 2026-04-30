# Prod Init Clean-Slate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate HackForger internal instance from polluted test state to a clean production starting point — wipe DB + filesystem, re-create admin, restore Logto OAuth, seed 12 production hackathons per #122, wire landing page.

**Architecture:** Approach C from spec (full reset + re-insert essentials). Five distinct contexts:
1. **dev worktree** (`.claude/worktrees/dev` on branch `dev/test-data-backup-2026-04-30`) — backup storage
2. **main repo** (`/Users/h2oslabs/Workspace/hackforger`) — running instance + data/ dir; all wipe/re-init operations happen here
3. **login-page worktree** (current) — spec/plan documentation
4. **landing-prod-slugs worktree** — new worktree for the slug-binding PR
5. **Logto admin (https://ifhqaw.logto.app)** — out of scope; DB row is preserved verbatim

**Tech Stack:** bash, sqlite3 CLI, `./gitea` CLI subcommands (`migrate`, `admin user create`, `admin user generate-access-token`), curl + REST API, Forgejo SQLite.

**Spec:** `docs/superpowers/specs/2026-04-30-prod-init-clean-slate-design.md`

---

## Pre-flight (manual confirmation gate)

Before Task 1 runs, the implementer MUST confirm:
- The user is ready for ~30-60s instance downtime
- The user has been told the new admin password will land in `.env` at the main repo root and they will manually retrieve it
- All in-flight test data is already either committed (code changes) or expendable (test hackathons / fake users)

If any of the above is unclear, stop and ask.

---

## Task 1: Create dev worktree + .gitignore configuration

**Files:**
- Create: `.claude/worktrees/dev/` (worktree on new branch `dev/test-data-backup-2026-04-30`)
- Create: `.claude/worktrees/dev/data-snapshot/` (dir)
- Modify: `.gitignore` in main repo (add `.env`, tarballs, jwt key)

- [ ] **Step 1: Create dev worktree from current main**

```bash
cd /Users/h2oslabs/Workspace/hackforger
git fetch origin v0.1-dev/hackforger
git worktree add .claude/worktrees/dev \
  -b dev/test-data-backup-2026-04-30 origin/v0.1-dev/hackforger
```

Expected: Output ends with `HEAD is now at <commit>`. Worktree dir `.claude/worktrees/dev/` exists.

- [ ] **Step 2: Create data-snapshot subdirectory**

```bash
mkdir -p /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/dev/data-snapshot
```

- [ ] **Step 3: Update `.gitignore` in dev worktree to exclude tarballs + jwt + .env**

Append to `/Users/h2oslabs/Workspace/hackforger/.claude/worktrees/dev/.gitignore`:

```
# Backup tarballs of pre-prod test data — large binaries, do not commit
.claude/worktrees/dev/data-snapshot/*.tar.gz
.claude/worktrees/dev/data-snapshot/*.pem.snapshot
.claude/worktrees/dev/data-snapshot/forgejo.db.backup

# Local secrets — never commit
.env
```

Note: this `.gitignore` is committed to the dev branch only; it does not affect the v0.1-dev/hackforger main branch.

- [ ] **Step 4: Verify .gitignore lines are effective**

```bash
cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/dev
touch data-snapshot/test.tar.gz
git check-ignore data-snapshot/test.tar.gz && echo "OK ignored" || echo "FAIL not ignored"
rm data-snapshot/test.tar.gz
```

Expected: `OK ignored`. If `FAIL`, fix the .gitignore line.

- [ ] **Step 5: Commit the worktree skeleton**

```bash
cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/dev
git add .gitignore
git commit -m "chore(dev): create test-data-backup branch with gitignore guards"
```

---

## Task 2: Backup all test data to dev worktree

**Files:**
- Create: `.claude/worktrees/dev/data-snapshot/forgejo.db.sql` (committed)
- Create: `.claude/worktrees/dev/data-snapshot/forgejo.db.backup` (gitignored, raw SQLite copy)
- Create: `.claude/worktrees/dev/data-snapshot/forgejo-repositories.tar.gz` (gitignored)
- Create: `.claude/worktrees/dev/data-snapshot/attachments.tar.gz` (gitignored)
- Create: `.claude/worktrees/dev/data-snapshot/avatars.tar.gz` (gitignored)
- Create: `.claude/worktrees/dev/data-snapshot/repo-avatars.tar.gz` (gitignored)
- Create: `.claude/worktrees/dev/data-snapshot/jwt-private.pem.snapshot` (gitignored)
- Create: `.claude/worktrees/dev/data-snapshot/app.ini.snapshot` (committed)
- Create: `.claude/worktrees/dev/data-snapshot/landing-index.html.snapshot` (committed)

- [ ] **Step 1: Stop the running instance cleanly**

```bash
PID=$(lsof -t -i:3000 -sTCP:LISTEN 2>/dev/null)
if [ -n "$PID" ]; then
  echo "Stopping PID $PID"
  kill -TERM "$PID"
  sleep 3
  if kill -0 "$PID" 2>/dev/null; then
    echo "Forcing stop"
    kill -KILL "$PID"
    sleep 1
  fi
fi
echo "Port 3000 status:"
lsof -i :3000 -P 2>/dev/null || echo "free"
```

Expected: `free`. If still occupied, investigate before proceeding.

- [ ] **Step 2: Capture full SQL dump (committed) + raw DB copy (gitignored fallback)**

```bash
cd /Users/h2oslabs/Workspace/hackforger
sqlite3 data/forgejo.db .dump > .claude/worktrees/dev/data-snapshot/forgejo.db.sql
sqlite3 data/forgejo.db ".backup '.claude/worktrees/dev/data-snapshot/forgejo.db.backup'"
ls -la .claude/worktrees/dev/data-snapshot/forgejo.db.*
```

Expected: Both files exist with non-zero size (`forgejo.db.sql` typically 1-3MB, `forgejo.db.backup` matches `data/forgejo.db` size).

- [ ] **Step 3: Verify the SQL dump is valid (parses without error)**

```bash
sqlite3 ":memory:" < .claude/worktrees/dev/data-snapshot/forgejo.db.sql >/dev/null
echo "exit=$?"
```

Expected: `exit=0`. Non-zero means corrupt dump; redo Step 2.

- [ ] **Step 4: Tarball file storage**

```bash
cd /Users/h2oslabs/Workspace/hackforger
tar czf .claude/worktrees/dev/data-snapshot/forgejo-repositories.tar.gz -C data forgejo-repositories
tar czf .claude/worktrees/dev/data-snapshot/attachments.tar.gz -C data attachments
tar czf .claude/worktrees/dev/data-snapshot/avatars.tar.gz -C data avatars 2>/dev/null || echo "avatars dir empty or missing — skipping"
[ -d data/repo-avatars ] && tar czf .claude/worktrees/dev/data-snapshot/repo-avatars.tar.gz -C data repo-avatars || echo "repo-avatars dir missing — skipping"
ls -lh .claude/worktrees/dev/data-snapshot/*.tar.gz
```

Expected: `forgejo-repositories.tar.gz` (~5-15MB compressed), `attachments.tar.gz` (~5-10MB), avatars/repo-avatars tarballs may be small or absent.

- [ ] **Step 5: Snapshot OAuth2 JWT private key (preserved across wipe)**

```bash
[ -f data/jwt/private.pem ] && \
  cp data/jwt/private.pem .claude/worktrees/dev/data-snapshot/jwt-private.pem.snapshot && \
  echo "OK jwt key snapshotted" || \
  echo "no jwt key present (may be auto-generated on first start)"
```

Note: original `data/jwt/private.pem` is NOT deleted — only copied. This key keeps existing OAuth2 access tokens valid post-wipe.

- [ ] **Step 6: Snapshot key config files (committed for audit)**

```bash
cp custom/conf/app.ini .claude/worktrees/dev/data-snapshot/app.ini.snapshot
cp custom/public/assets/landing/index.html .claude/worktrees/dev/data-snapshot/landing-index.html.snapshot
ls -la .claude/worktrees/dev/data-snapshot/*.snapshot
```

- [ ] **Step 7: Verify backup completeness — list all expected files**

```bash
cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/dev/data-snapshot
ls -la
echo "---size summary---"
du -sh *
```

Expected output includes:
- `forgejo.db.sql` (1-3MB)
- `forgejo.db.backup` (~5MB matching live DB)
- `forgejo-repositories.tar.gz` (5-15MB)
- `attachments.tar.gz` (5-10MB)
- `app.ini.snapshot` (1-2KB)
- `landing-index.html.snapshot` (~250KB)
- (jwt-private.pem.snapshot, avatars/repo-avatars tarballs if applicable)

If any expected file is missing or empty, redo the corresponding step.

---

## Task 3: Extract essentials + write RESTORE.md, commit dev worktree

**Files:**
- Create: `.claude/worktrees/dev/data-snapshot/essentials.sql` (committed)
- Create: `.claude/worktrees/dev/data-snapshot/action-runners.sql` (committed)
- Create: `.claude/worktrees/dev/data-snapshot/RESTORE.md` (committed)

- [ ] **Step 1: Extract Logto login_source row + 4 system_setting rows into essentials.sql**

```bash
cd /Users/h2oslabs/Workspace/hackforger
ESSENTIALS=.claude/worktrees/dev/data-snapshot/essentials.sql

cat > "$ESSENTIALS" <<'EOF'
-- Re-insert script for the clean-slate prod init.
-- Run AFTER ./gitea migrate has created empty schema and BEFORE first server start.
-- Only inserts data that must persist from pre-wipe state:
--   1. Logto OAuth login source (id=3, kept explicit to preserve callback URL)
--   2. Curated system_setting rows (id auto-assigned; identity is setting_key)

BEGIN TRANSACTION;

EOF

# Append the Logto row with explicit id=3
sqlite3 data/forgejo.db <<'SQL' >> "$ESSENTIALS"
.mode insert login_source
SELECT * FROM login_source WHERE id = 3;
SQL

# Append the curated system_setting rows; SELECT only the columns we want
# (omit id; AUTOINCREMENT will assign new ones)
sqlite3 data/forgejo.db <<'SQL' >> "$ESSENTIALS"
.mode insert system_setting
SELECT setting_key, setting_value, version, created, updated
FROM system_setting
WHERE setting_key IN (
  'revision',
  'picture.disable_gravatar',
  'repository.open-with.editor-apps',
  'hackforger.landing.hackathon_prefix'
);
SQL

echo "" >> "$ESSENTIALS"
echo "COMMIT;" >> "$ESSENTIALS"
cat "$ESSENTIALS"
```

Expected: An `essentials.sql` containing `BEGIN TRANSACTION;`, `INSERT INTO login_source ... id=3 ...`, several `INSERT INTO system_setting ...`, `COMMIT;`. Inspect manually before continuing.

⚠ **Manual fix-up if `.mode insert` produces wrong column list for system_setting**: the `SELECT col_a, col_b, ...` form may emit `INSERT INTO system_setting VALUES(...)` rather than `INSERT INTO system_setting(setting_key, setting_value, ...) VALUES(...)`. If columns don't include `id` first, the INSERT will fail because it expects positional alignment. In that case, hand-edit `essentials.sql` to use named-column form:

```sql
INSERT INTO system_setting(setting_key, setting_value, version, created, updated) VALUES('revision','...',1,...,...);
```

- [ ] **Step 2: Backup action_runner rows for re-registration reference**

```bash
sqlite3 data/forgejo.db <<'SQL' > .claude/worktrees/dev/data-snapshot/action-runners.sql
.mode insert action_runner
SELECT * FROM action_runner;
SQL
echo "Runner row count: $(grep -c 'INSERT INTO action_runner' .claude/worktrees/dev/data-snapshot/action-runners.sql)"
```

Expected: A small file (likely 0-3 rows). Used for reference only — runners need re-registration via Forgejo admin UI post-wipe.

- [ ] **Step 3: Write RESTORE.md with full rollback steps**

Create `.claude/worktrees/dev/data-snapshot/RESTORE.md`:

```markdown
# Pre-prod state restore instructions

This worktree contains a full backup of the test environment captured 2026-04-30 before the clean-slate migration to production.

## To roll back to the pre-wipe state

Execute from the main repo root: `/Users/h2oslabs/Workspace/hackforger`

```bash
cd /Users/h2oslabs/Workspace/hackforger

# 1. Stop running instance
PID=$(lsof -t -i:3000 -sTCP:LISTEN); [ -n "$PID" ] && kill -TERM "$PID" && sleep 3

# 2. Wipe current data dir contents (NOT the dirs themselves)
rm -f data/forgejo.db data/forgejo.db-wal data/forgejo.db-shm
rm -rf data/forgejo-repositories/* data/attachments/* data/avatars/* data/repo-avatars/* data/sessions/*
rm -rf data/indexers/*.bleve
rm -f data/queues/common/LOCK

# 3. Restore SQLite DB
cp .claude/worktrees/dev/data-snapshot/forgejo.db.backup data/forgejo.db

# 4. Restore file storage
tar xzf .claude/worktrees/dev/data-snapshot/forgejo-repositories.tar.gz -C data
tar xzf .claude/worktrees/dev/data-snapshot/attachments.tar.gz -C data
[ -f .claude/worktrees/dev/data-snapshot/avatars.tar.gz ] && tar xzf .claude/worktrees/dev/data-snapshot/avatars.tar.gz -C data
[ -f .claude/worktrees/dev/data-snapshot/repo-avatars.tar.gz ] && tar xzf .claude/worktrees/dev/data-snapshot/repo-avatars.tar.gz -C data

# 5. Restore JWT key (already preserved if we didn't delete during wipe)
[ -f .claude/worktrees/dev/data-snapshot/jwt-private.pem.snapshot ] && \
  cp .claude/worktrees/dev/data-snapshot/jwt-private.pem.snapshot data/jwt/private.pem

# 6. Restart
bash scripts/restart-gitea.sh
```

After restore, the running instance should match its 2026-04-30 pre-wipe state. Old admin password (`admin1234` per CLAUDE.md) and old FORGEJO_TOKEN values resume working.

## Files in this snapshot

| File | Purpose | Committed? |
|---|---|---|
| `forgejo.db.sql` | Full SQL dump | yes |
| `forgejo.db.backup` | Raw SQLite copy (preferred for restore — no SQL replay needed) | no (gitignored) |
| `forgejo-repositories.tar.gz` | All git repos | no |
| `attachments.tar.gz` | User uploaded attachments | no |
| `avatars.tar.gz` | User avatars | no |
| `repo-avatars.tar.gz` | Repo avatars | no |
| `jwt-private.pem.snapshot` | OAuth2 JWT signing key (also preserved in place) | no |
| `app.ini.snapshot` | Server config | yes |
| `landing-index.html.snapshot` | Landing page HTML | yes |
| `essentials.sql` | Logto OAuth + system_setting re-insert script | yes |
| `action-runners.sql` | Action runner rows (reference for re-registration) | yes |
| `RESTORE.md` | This file | yes |
```

- [ ] **Step 4: Commit dev worktree backup metadata**

```bash
cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/dev
git add data-snapshot/
git status   # double-check large tarballs are NOT staged (gitignore working)
git commit -m "chore(dev): backup pre-prod state — DB SQL dump + essentials + RESTORE.md"
```

Expected: `git status` after `add` shows only `forgejo.db.sql`, `app.ini.snapshot`, `landing-index.html.snapshot`, `essentials.sql`, `action-runners.sql`, `RESTORE.md` staged. Tarballs MUST be ignored.

If a tarball appears in staged files, abort, fix `.gitignore`, `git reset HEAD <tarball>`, retry.

---

## Task 4: Wipe + Schema Reinit + Recreate Admin

**Files:**
- Modify (delete): `data/forgejo.db`, `data/forgejo.db-wal`, `data/forgejo.db-shm` (main repo)
- Modify (clear): `data/forgejo-repositories/*`, `data/attachments/*`, `data/avatars/*`, `data/repo-avatars/*`, `data/sessions/*`, `data/indexers/*.bleve` (main repo)
- Modify (delete): `data/queues/common/LOCK` (main repo)
- Create: `/Users/h2oslabs/Workspace/hackforger/.env` (gitignored, mode 600)

- [ ] **Step 1: Confirm instance is stopped (sanity from Task 2)**

```bash
lsof -i :3000 -P 2>/dev/null || echo "free"
```

Expected: `free`. If anything is listening on 3000, stop it before continuing.

- [ ] **Step 2: Wipe data dir using exact filenames (not glob)**

```bash
cd /Users/h2oslabs/Workspace/hackforger

# Exact filenames — do NOT glob forgejo.db* (would clobber forgejo.db.bak.* rotation backups in data/)
rm -f data/forgejo.db data/forgejo.db-wal data/forgejo.db-shm

# Clear contents of these dirs (keep dirs themselves)
rm -rf data/forgejo-repositories/* data/attachments/* data/avatars/* data/sessions/*
[ -d data/repo-avatars ] && rm -rf data/repo-avatars/*

# Stale indexes — Forgejo rebuilds on next start
rm -rf data/indexers/*.bleve

# LevelDB lock — must clear between stop and restart
rm -f data/queues/common/LOCK

# Sanity: jwt key NOT deleted
ls -la data/jwt/private.pem 2>/dev/null && echo "OK jwt key preserved"
```

Expected: jwt key still present. `data/forgejo.db` gone. Other dirs empty.

- [ ] **Step 3: Run `./gitea migrate` to recreate empty schema**

```bash
cd /Users/h2oslabs/Workspace/hackforger
./gitea migrate
```

Expected: command completes without error; `data/forgejo.db` recreated with empty tables.

```bash
sqlite3 data/forgejo.db "SELECT count(*) FROM sqlite_master WHERE type='table';"
sqlite3 data/forgejo.db "SELECT count(*) FROM \"user\";"
```

Expected: First query returns ~150+ tables. Second returns `0` (no users yet).

- [ ] **Step 4: Generate strong random password and set up `.env` securely**

```bash
cd /Users/h2oslabs/Workspace/hackforger

# umask 077 makes any new files mode 600 by default for the rest of this shell
umask 077

# 24-char alphanumeric password
PASSWORD=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24)
echo "Generated password length: ${#PASSWORD}"

# Create .env with strict perms; overwrite if exists
> .env
chmod 600 .env
echo "HACKFORGER_ADMIN_PASSWORD=$PASSWORD" >> .env
ls -la .env
```

Expected: `-rw-------` permissions, `.env` size > 0.

- [ ] **Step 5: Create hackforger admin via CLI**

```bash
cd /Users/h2oslabs/Workspace/hackforger
./gitea admin user create \
  --admin \
  --username hackforger \
  --email hackforger@h2os.cloud \
  --password "$PASSWORD" \
  --must-change-password=false
```

Expected: `New user 'hackforger' has been successfully created!` printed.

```bash
sqlite3 data/forgejo.db "SELECT id, name, email, is_admin, is_active FROM \"user\";"
```

Expected: Single row: `1|hackforger|hackforger@h2os.cloud|1|1`.

- [ ] **Step 6: Confirm `.env` is gitignored**

```bash
cd /Users/h2oslabs/Workspace/hackforger
git check-ignore -v .env
```

Expected: Output names a `.gitignore` rule. If `.env` is NOT ignored, add `.env` to `.gitignore` and commit before continuing — leaking the password into git is a serious leak.

---

## Task 5: Re-INSERT essentials and verify Logto works

**Files:**
- Modify: `data/forgejo.db` (INSERT rows)

- [ ] **Step 1: Apply essentials.sql**

```bash
cd /Users/h2oslabs/Workspace/hackforger
sqlite3 data/forgejo.db < .claude/worktrees/dev/data-snapshot/essentials.sql
echo "exit=$?"
```

Expected: `exit=0`. Non-zero means SQL syntax error or constraint violation; inspect `essentials.sql` and re-run after fix.

- [ ] **Step 2: Verify the re-inserted rows are correct**

```bash
echo "=== login_source ==="
sqlite3 data/forgejo.db "SELECT id, name, type, is_active FROM login_source;"
echo "=== system_setting ==="
sqlite3 data/forgejo.db "SELECT setting_key, substr(setting_value, 1, 60) FROM system_setting ORDER BY setting_key;"
```

Expected:
- `login_source` row with `id=3, name=Logto, type=6, is_active=1`
- `system_setting` rows for at least `revision`, `picture.disable_gravatar`, `repository.open-with.editor-apps`, `hackforger.landing.hackathon_prefix`

If `login_source` returns no rows or `id ≠ 3`, fix `essentials.sql` (must include `INSERT INTO login_source(id, ...) VALUES(3, ...)`) and re-run.

- [ ] **Step 3: Start the instance**

```bash
cd /Users/h2oslabs/Workspace/hackforger
bash scripts/restart-gitea.sh 2>&1 | tail -10
```

Expected: `✓ HTTP 200 on localhost:3000`.

- [ ] **Step 4: Verify Logto button appears on sign-in page**

```bash
curl -s https://hackforger.inside.h2os.cloud/user/login | grep -E "Logto|/user/oauth2/Logto" | head -3
```

Expected: Output includes `href="/user/oauth2/Logto"` and the i18n button text per recent locale change.

If empty, the Logto row may have wrong cfg or wrong id. Inspect `sqlite3 data/forgejo.db "SELECT cfg FROM login_source WHERE id=3;"` against snapshotted essentials and fix.

- [ ] **Step 5: Verify admin password login works**

Manual check: Visit https://hackforger.inside.h2os.cloud/user/login in a browser, enter `hackforger` + password from `.env`, confirm dashboard loads.

If login fails, the password write to `.env` may have shell-escaped a `$` or `\` character. Generate a fresh password (Step 4 of Task 4) using only `[A-Za-z0-9]` and re-run from Task 4 step 4.

---

## Task 6: Generate admin token + seed 12 hackathons

**Files:**
- Modify: `.env` (append FORGEJO_TOKEN line)
- Create: `/tmp/seed-hackathons.sh` (one-shot script, not committed)

- [ ] **Step 1: Generate admin access token via CLI**

```bash
cd /Users/h2oslabs/Workspace/hackforger
TOKEN_OUTPUT=$(./gitea admin user generate-access-token \
  --username hackforger \
  --token-name "prod-init" \
  --scopes "all")
echo "$TOKEN_OUTPUT"
TOKEN=$(echo "$TOKEN_OUTPUT" | awk '/Access token was successfully created/ {print $NF}')
echo "Token length: ${#TOKEN}"
```

Expected: Token length 40+ chars. If TOKEN is empty, the `awk` pattern is wrong; inspect `TOKEN_OUTPUT` literal and adjust the awk.

- [ ] **Step 2: Append token to `.env`**

```bash
echo "FORGEJO_TOKEN=$TOKEN" >> /Users/h2oslabs/Workspace/hackforger/.env
cat /Users/h2oslabs/Workspace/hackforger/.env  # verify both lines present, file still mode 600
```

Expected: 2 lines (HACKFORGER_ADMIN_PASSWORD + FORGEJO_TOKEN).

- [ ] **Step 3: Write the seed script**

Create `/tmp/seed-hackathons.sh`:

```bash
#!/bin/bash
set -uo pipefail

# Source env to get FORGEJO_TOKEN
source /Users/h2oslabs/Workspace/hackforger/.env

API="https://hackforger.inside.h2os.cloud/api/v1/hackforger/hackathons"
DESCRIPTION="第二届燕缘·协创者号 AI+ 国际创业大赛 — 详细赛规以官方落地页为准。"

# spec format: SLUG|NAME (pipe-separated to avoid Chinese punctuation collisions)
specs=(
  "opc-2026-shuzhi-w1|【初赛W1】数智OPC加速赛"
  "opc-2026-shuzhi-w2|【复赛W2】数智OPC加速赛"
  "opc-2026-shuzhi-w3|【半决赛W3】数智OPC加速赛"
  "opc-2026-shuzhi-w4|【决赛W4】数智OPC加速赛"
  "opc-2026-crossborder-w1|【初赛W1】跨境OPC加速赛"
  "opc-2026-crossborder-w2|【复赛W2】跨境OPC加速赛"
  "opc-2026-crossborder-w3|【半决赛W3】跨境OPC加速赛"
  "opc-2026-crossborder-w4|【决赛W4】跨境OPC加速赛"
  "opc-2026-youth-w1|【初赛W1】全球青年培育赛"
  "opc-2026-youth-w2|【复赛W2】全球青年培育赛"
  "opc-2026-youth-w3|【半决赛W3】全球青年培育赛"
  "opc-2026-youth-w4|【决赛W4】全球青年培育赛"
)

failures=0
for spec in "${specs[@]}"; do
  slug="${spec%%|*}"
  name="${spec#*|}"

  body=$(jq -n --arg slug "$slug" --arg name "$name" --arg desc "$DESCRIPTION" \
    '{slug:$slug, name:$name, description:$desc, org_id:1}')

  http_code=$(curl -s -X POST -H "Authorization: token $FORGEJO_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$body" \
    -o /tmp/seed-resp.json -w "%{http_code}" \
    "$API")

  case "$http_code" in
    200|201) echo "✓ $slug created (HTTP $http_code)" ;;
    409)     echo "⚠ $slug already exists (HTTP 409, idempotent re-run OK)" ;;
    *)       echo "✗ $slug FAILED HTTP $http_code"; cat /tmp/seed-resp.json; echo ""; failures=$((failures+1)) ;;
  esac
done

echo
echo "Done. Failures: $failures"
exit $failures
```

```bash
chmod +x /tmp/seed-hackathons.sh
```

Note: relies on `jq` for JSON encoding (handles Chinese escaping). Verify `jq` is installed: `which jq`. If not, `brew install jq`.

- [ ] **Step 4: Run the seed script**

```bash
/tmp/seed-hackathons.sh
```

Expected: 12 lines `✓ ... created (HTTP 201)`, then `Done. Failures: 0`.

If failures > 0, inspect output, fix the failing condition, re-run (idempotent on slug uniqueness via 409 path).

- [ ] **Step 5: Verify all 12 hackathons in DB**

```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT slug, name FROM hackathon ORDER BY id;"
echo "---count---"
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT count(*) FROM hackathon;"
```

Expected: 12 rows, slugs match #122 mapping, count=12.

- [ ] **Step 6: HTTP-spot-check 4 representative hackathon URLs**

```bash
for slug in opc-2026-shuzhi-w1 opc-2026-shuzhi-w4 opc-2026-crossborder-w1 opc-2026-youth-w4; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "https://hackforger.inside.h2os.cloud/hackathon/$slug")
  echo "$slug → $code"
done
```

Expected: All four return `200`.

---

## Task 7: Update landing page slug bindings (PR flow)

**Files:**
- Modify: `custom/public/assets/landing/index.html` lines 715-726 (the `HACKFORGER_LANDING_CONFIG.stages` block) — operate inside a new worktree

- [ ] **Step 1: Create worktree on a new branch from latest main**

```bash
cd /Users/h2oslabs/Workspace/hackforger
git fetch origin v0.1-dev/hackforger
git worktree add .worktrees/landing-prod-slugs \
  -b feat/landing-prod-slugs origin/v0.1-dev/hackforger
```

Note: `.worktrees/` (top-level) is gitignored; this PR worktree is separate from `.claude/worktrees/dev`.

- [ ] **Step 2: Replace the JS config block**

Edit `/Users/h2oslabs/Workspace/hackforger/.worktrees/landing-prod-slugs/custom/public/assets/landing/index.html`. Find the block:

```javascript
window.HACKFORGER_LANDING_CONFIG = {
  stages: {
    's1-w1': { slug: 'tbd', enabled: true  },
    's1-w2': { slug: 'tbd', enabled: false },
    's1-w3': { slug: 'tbd', enabled: false },
    's1-w4': { slug: 'tbd', enabled: false },
    's2-w1': { slug: 'tbd', enabled: false },
    's2-w2': { slug: 'tbd', enabled: false },
    's2-w3': { slug: 'tbd', enabled: false },
    's2-w4': { slug: 'tbd', enabled: false },
    's3-w1': { slug: 'tbd', enabled: false },
    's3-w2': { slug: 'tbd', enabled: false },
    's3-w3': { slug: 'tbd', enabled: false },
    's3-w4': { slug: 'tbd', enabled: false }
  }
};
```

Replace with:

```javascript
window.HACKFORGER_LANDING_CONFIG = {
  stages: {
    's1-w1': { slug: 'opc-2026-shuzhi-w1',      enabled: true  },
    's1-w2': { slug: 'opc-2026-shuzhi-w2',      enabled: false },
    's1-w3': { slug: 'opc-2026-shuzhi-w3',      enabled: false },
    's1-w4': { slug: 'opc-2026-shuzhi-w4',      enabled: false },
    's2-w1': { slug: 'opc-2026-crossborder-w1', enabled: false },
    's2-w2': { slug: 'opc-2026-crossborder-w2', enabled: false },
    's2-w3': { slug: 'opc-2026-crossborder-w3', enabled: false },
    's2-w4': { slug: 'opc-2026-crossborder-w4', enabled: false },
    's3-w1': { slug: 'opc-2026-youth-w1',       enabled: false },
    's3-w2': { slug: 'opc-2026-youth-w2',       enabled: false },
    's3-w3': { slug: 'opc-2026-youth-w3',       enabled: false },
    's3-w4': { slug: 'opc-2026-youth-w4',       enabled: false }
  }
};
```

Today is 2026-04-30, S1-W1 registration window 4.30-5.5 → only `s1-w1.enabled = true`. All others `false` (registration not yet open or finals).

- [ ] **Step 3: Verify the change is exactly 12 lines and slugs are correct**

```bash
cd /Users/h2oslabs/Workspace/hackforger/.worktrees/landing-prod-slugs
grep -c "slug: 'opc-2026-" custom/public/assets/landing/index.html
grep -c "slug: 'tbd'" custom/public/assets/landing/index.html
git diff --stat custom/public/assets/landing/index.html
```

Expected: First count = 12, second count = 0, diff stat shows 1 file changed with reasonable line counts.

- [ ] **Step 4: Commit + push**

```bash
cd /Users/h2oslabs/Workspace/hackforger/.worktrees/landing-prod-slugs
git add custom/public/assets/landing/index.html
git commit -m "feat(landing): wire 12 schedule cards to production slugs (closes #122)

Replaces all 12 'tbd' slug placeholders in HACKFORGER_LANDING_CONFIG.stages
with the production slugs from #122. S1-W1 stays enabled (registration
window 4.30-5.5 includes today, 2026-04-30); all other 11 entries remain
disabled until their respective registration windows open.

Closes #122. Refs #114."
git push -u origin feat/landing-prod-slugs
```

- [ ] **Step 5: Open and admin-merge the PR**

```bash
cd /Users/h2oslabs/Workspace/hackforger/.worktrees/landing-prod-slugs
gh pr create --base v0.1-dev/hackforger \
  --title "feat(landing): wire 12 schedule cards to production slugs" \
  --body "Closes #122. Wires HACKFORGER_LANDING_CONFIG.stages to the production slug mapping per Cynthialime's #122 spec. Required after the prod-init data wipe so the landing page \"立即报名\" buttons route correctly."

# Wait for the PR number to appear, then merge
PR_URL=$(gh pr view --json url --jq .url)
echo "PR: $PR_URL"
gh pr merge --admin --squash
```

Expected: PR merged. `gh pr view` should report `state: MERGED`.

- [ ] **Step 6: Clean up worktree + branch**

```bash
cd /Users/h2oslabs/Workspace/hackforger
git worktree remove .worktrees/landing-prod-slugs --force
git branch -D feat/landing-prod-slugs
```

- [ ] **Step 7: Pull merge into main repo + restart**

```bash
cd /Users/h2oslabs/Workspace/hackforger
git fetch origin v0.1-dev/hackforger
git merge --ff-only origin/v0.1-dev/hackforger
bash scripts/restart-gitea.sh 2>&1 | tail -5
```

Expected: `✓ HTTP 200 on localhost:3000`.

---

## Task 8: End-to-end verification + close issues

**Files:** None — this is verification + GitHub commenting.

- [ ] **Step 1: Verify single admin user**

```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT count(*) FROM \"user\";"
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT id, name, is_admin FROM \"user\";"
```

Expected: count = 1, single row `1|hackforger|1`. (Note: hackathon orgs created by Task 6 are stored in `user` table too with `type=1`. So count may be 13 = 1 admin + 12 hackathon orgs. Verify identity rather than just count.)

```bash
echo "Admin users only:"
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT count(*) FROM \"user\" WHERE is_admin = 1;"
```

Expected: 1.

- [ ] **Step 2: Verify Logto OAuth still works**

```bash
curl -s https://hackforger.inside.h2os.cloud/user/login | grep -E "/user/oauth2/Logto" | head -1
```

Expected: 1 line containing the Logto button anchor.

- [ ] **Step 3: Verify all 12 hackathon URLs return 200**

```bash
for slug in opc-2026-shuzhi-w1 opc-2026-shuzhi-w2 opc-2026-shuzhi-w3 opc-2026-shuzhi-w4 \
            opc-2026-crossborder-w1 opc-2026-crossborder-w2 opc-2026-crossborder-w3 opc-2026-crossborder-w4 \
            opc-2026-youth-w1 opc-2026-youth-w2 opc-2026-youth-w3 opc-2026-youth-w4; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "https://hackforger.inside.h2os.cloud/hackathon/$slug")
  echo "$slug → $code"
done
```

Expected: All 12 lines end with `200`.

- [ ] **Step 4: Verify landing JS config is the new mapping**

```bash
curl -s https://hackforger.inside.h2os.cloud/ | grep -c "slug: 'opc-2026-"
curl -s https://hackforger.inside.h2os.cloud/ | grep -c "slug: 'tbd'"
```

Expected: First count = 12, second count = 0.

- [ ] **Step 5: Verify clicking S1-W1 button routes correctly**

Manual browser check (or curl follow):

```bash
curl -s -I https://hackforger.inside.h2os.cloud/hackathon/opc-2026-shuzhi-w1 | head -5
```

Expected: `HTTP/2 200`.

- [ ] **Step 6: Comment + close #122**

```bash
gh issue comment 122 --body "$(cat <<'EOF'
## ✅ Implemented

12 hackathons created in DB matching the slug mapping from this issue:

| Wave | Slug | DB | URL |
|---|---|---|---|
| S1-W1 | opc-2026-shuzhi-w1 | ✓ | https://hackforger.inside.h2os.cloud/hackathon/opc-2026-shuzhi-w1 |
| S1-W2 | opc-2026-shuzhi-w2 | ✓ | ... |
| ... | ... | ✓ | ... |
| S3-W4 | opc-2026-youth-w4 | ✓ | https://hackforger.inside.h2os.cloud/hackathon/opc-2026-youth-w4 |

Landing page \`HACKFORGER_LANDING_CONFIG.stages\` updated via PR (above). All 12 \"立即报名\" buttons now route to the correct hackathon URL.

This was part of a larger clean-slate prod init that also wiped test data and re-created the admin account. See `docs/superpowers/specs/2026-04-30-prod-init-clean-slate-design.md` for the full spec.

Closing.
EOF
)"
gh issue close 122
```

- [ ] **Step 7: Comment + close #114**

```bash
gh issue comment 114 --body "$(cat <<'EOF'
## ✅ Closed — slug naming + landing wiring shipped

Resolution path:
- Naming convention decided in #122 (option (b) — uniform `opc-2026-{league}-w{N}` for all waves)
- 12 hackathons re-seeded fresh during prod-init wipe with the new slugs (no rename of stale W1/W2 needed; old test hackathons gone)
- Landing page slug bindings updated to point to the new URLs

This issue is closed as the architecture-level discussion is settled (route 1 confirmed, weekly ops table mechanism in place via Cynthialime). Further weekly content updates flow through the ops template, not through new issues.

Closing.
EOF
)"
gh issue close 114
```

- [ ] **Step 8: Final summary in conversation**

Report to the user:
- The new admin password + token are in `/Users/h2oslabs/Workspace/hackforger/.env` (mode 600, gitignored)
- 12 hackathons live and reachable via the listed URLs
- Landing page wired and merged via PR
- Backups stored in `.claude/worktrees/dev/data-snapshot/` (run `cat RESTORE.md` for rollback steps)
- Issues #122 and #114 closed; remaining open landing-related issues unchanged (#109 KPI, #111 templatize Backlog, #115 Logto GitHub disable, #118 perms)

If any verification step in this Task 8 failed, do NOT close issues. Halt and report.

---

## Self-review against spec

| Spec section | Implementation task |
|---|---|
| Phase 1: Create dev worktree | Task 1 |
| Phase 2: Backup current state | Task 2 |
| Phase 3: Capture essentials | Task 3 |
| Phase 4: Wipe + Fresh Init | Task 4 + Task 5 (split: wipe + recreate-admin in 4, re-INSERT essentials + verify in 5) |
| Phase 5: Generate token + create 12 hackathons | Task 6 |
| Phase 6: Wire landing page | Task 7 |
| Phase 7: Verify + close issues | Task 8 |
| `.env` mode 600 + umask 077 (N2) | Task 4 step 4 |
| login_source explicit id=3 (C2) | Task 3 step 1 |
| system_setting without explicit id (C1) | Task 3 step 1 |
| `data/queues/common/LOCK` removal (C3) | Task 4 step 2 |
| Indexer wipe + rebuild (B2) | Task 4 step 2 |
| JWT key preserved (B2) | Task 2 step 5 + Task 4 step 2 |
| Exact filenames not glob (B3) | Task 4 step 2 |
| `pkill -TERM` clean stop before snapshot (C6) | Task 2 step 1 |
| `org_id: 1` to satisfy API binding (B1) | Task 6 step 3 |
| `repo-avatars` backup (C5) | Task 2 step 4 |
| 409-as-success in seed (N1) | Task 6 step 3 |
| Admin user count assertion (N4) | Task 8 step 1 |
| Action runner re-registration note (C4) | Task 3 step 2 + RESTORE.md |

All spec sections have implementing tasks. No placeholders detected.
