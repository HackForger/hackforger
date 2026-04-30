# Prod Init: Clean Slate Migration

**Date:** 2026-04-30
**Issue:** [#122](https://github.com/HackForger/hackforger/issues/122) (slug mapping), [#114](https://github.com/HackForger/hackforger/issues/114) (active landing tracker)

## Summary

Move the internal HackForger instance (`https://hackforger.inside.h2os.cloud`) from a polluted test state to a clean production starting point. The instance has accumulated ~30 test users, 108 repos, 30+ test hackathons, and miscellaneous test fixtures during development. We need a deterministic clean state that holds only:

- The `hackforger` admin user (with a freshly-generated strong password)
- The Logto OAuth `login_source` row (preserves sign-in flow)
- A small set of curated `system_setting` rows
- 12 newly-created hackathons matching the slug mapping in #122

All test data is preserved as a recoverable backup before destruction.

## Goals & Non-Goals

**Goals**

- Drop all test users, repos, hackathons, attachments, sessions
- Preserve admin auth (admin user re-created with strong password) and OAuth (Logto)
- Preserve i18n locale changes, custom landing assets, app.ini server config
- Create 12 production hackathons per #122 with correct slug + display name
- Wire landing page to point to the 12 new slugs (replacing `tbd` placeholders)
- Recoverable backup of everything destroyed
- Smooth user-facing transition: anonymous visitors should never see a broken page

**Non-Goals**

- Migrating to a different domain/instance
- Changing user-facing visual design (covered separately by #112 work)
- Setting up full hackathon metadata: tracks, phases, judging criteria, prize pools (ops-managed via web UI post-init)
- Building automated DB-sync (already rejected in #114; ops manages via weekly template per route 1)

## Architecture

### Phases

```
[Phase 1] Create dev worktree (branch: dev/test-data-backup-2026-04-30)
[Phase 2] Backup current state to dev worktree
[Phase 3] Capture essentials (Logto, system_setting) to SQL re-insert script
[Phase 4] Wipe & fresh init (drop forgejo.db, restart, recreate admin via CLI, re-insert essentials)
[Phase 5] Generate admin token, POST 12 hackathons via API
[Phase 6] Wire landing page to point to new slugs (PR to main branch)
[Phase 7] Verify end-to-end and close #122 + #114
```

Each phase is gated by validation before moving to the next.

### Data flow

```
Old state                              dev worktree                       New state
─────────                              ────────────                       ─────────
forgejo.db (~30 users, 108 repos) ─┐
data/forgejo-repositories/ (20MB) ─┤
data/attachments/ (11MB)           ├──► data-snapshot/ (gitignored        forgejo.db
data/avatars/ (704KB)              │    tarballs + committed              (1 admin user,
custom/conf/app.ini                │    SQL dump + RESTORE.md)            12 hackathons,
custom/public/.../landing/         │                                      Logto OAuth)
                                   ├──► essentials.sql (committed,        data/forgejo-repositories/
                                   │    Logto config + curated            (orgs from 12 hackathons)
                                   │    system_settings)
                                   │                                      .env (admin password +
                                   └──► RESTORE.md                            new admin token,
                                        (recovery instructions)               gitignored)
```

## Components

### `.claude/worktrees/dev/`

A new git worktree on branch `dev/test-data-backup-2026-04-30`. Holds the backup of the test environment. Large binary files are `.gitignore`'d; metadata + SQL + recovery docs are committed.

```
.claude/worktrees/dev/data-snapshot/
├── forgejo.db.sql              # Full SQL dump of pre-wipe DB (committed)
├── forgejo-repositories.tar.gz # All test repos (gitignored, ~20MB)
├── attachments.tar.gz          # User attachments (gitignored, ~11MB)
├── avatars.tar.gz              # User avatars (gitignored, ~700KB)
├── repo-avatars.tar.gz         # Repo avatars (gitignored)
├── jwt-private.pem.snapshot    # OAuth2 JWT signing key — preserved across wipe (gitignored, sensitive)
├── action-runners.sql          # Backup of action_runner rows (committed) — needed if runners re-register manually
├── app.ini.snapshot            # Server config snapshot (committed)
├── landing-index.html.snapshot # Landing page snapshot (committed)
├── essentials.sql              # Re-insert script for Logto + system_setting (committed)
└── RESTORE.md                  # Step-by-step rollback instructions (committed)
```

**Indexer dirs (`data/indexers/*.bleve`) are NOT backed up** — they're stale references after wipe and Forgejo rebuilds them on next startup. Delete during wipe.

**`data/jwt/private.pem` is preserved across wipe** (copied to dev worktree for backup, but the original at `data/jwt/private.pem` stays in place). This keeps any in-flight OAuth2 access tokens valid; cleaner than forcing all OAuth2 reauth.

### `.env` (in main repo, gitignored)

```
HACKFORGER_ADMIN_PASSWORD=<openssl-generated 24-char alphanumeric>
FORGEJO_TOKEN=<freshly-generated admin access token after wipe>
```

This file lives at the main repo root. The user views it manually after the wipe completes to retrieve the new credentials. Both lines are written by the script during Phase 4 / Phase 5.

### 12 Hackathon entities (created via REST API)

Each hackathon is created via `POST /api/v1/hackforger/hackathons` with the slug + name from #122.

**API contract gotcha (verified in `routers/api/v1/hackforger/hackathon.go:24` and `services/hackforger/hackathon.go:124-141`)**: The API form requires `org_id` (`binding:"Required"`), but the service layer **always creates a new org for the hackathon and overwrites `h.OrgID`**. So we MUST send a non-zero `org_id` to pass binding validation, but its value doesn't matter functionally. We'll send `org_id: 1` (the admin user's id — since users and orgs share an ID space in Forgejo's `user` table) to satisfy validation. The resulting hackathon's actual `org_id` will be the new auto-created org.

Slug mapping (per [#122](https://github.com/HackForger/hackforger/issues/122)):

| Slug                       | Name                          |
|----------------------------|-------------------------------|
| `opc-2026-shuzhi-w1`       | `【初赛W1】数智OPC加速赛`     |
| `opc-2026-shuzhi-w2`       | `【复赛W2】数智OPC加速赛`     |
| `opc-2026-shuzhi-w3`       | `【半决赛W3】数智OPC加速赛`   |
| `opc-2026-shuzhi-w4`       | `【决赛W4】数智OPC加速赛`     |
| `opc-2026-crossborder-w1`  | `【初赛W1】跨境OPC加速赛`     |
| `opc-2026-crossborder-w2`  | `【复赛W2】跨境OPC加速赛`     |
| `opc-2026-crossborder-w3`  | `【半决赛W3】跨境OPC加速赛`   |
| `opc-2026-crossborder-w4`  | `【决赛W4】跨境OPC加速赛`     |
| `opc-2026-youth-w1`        | `【初赛W1】全球青年培育赛`    |
| `opc-2026-youth-w2`        | `【复赛W2】全球青年培育赛`    |
| `opc-2026-youth-w3`        | `【半决赛W3】全球青年培育赛`  |
| `opc-2026-youth-w4`        | `【决赛W4】全球青年培育赛`    |

Description and prize_summary fields use a generic boilerplate; tracks/phases/judging are ops follow-up work.

### Landing page slug binding (`custom/public/assets/landing/index.html`)

Replace the JS config block:

```diff
window.HACKFORGER_LANDING_CONFIG = {
  stages: {
-   's1-w1': { slug: 'tbd', enabled: true  },
+   's1-w1': { slug: 'opc-2026-shuzhi-w1', enabled: true  },
-   's1-w2': { slug: 'tbd', enabled: false },
+   's1-w2': { slug: 'opc-2026-shuzhi-w2', enabled: false },
    ... (12 lines total)
  }
};
```

Today is 2026-04-30. S1-W1 registration window is 4.30–5.5, so `s1-w1.enabled = true`. All other 11 entries stay `enabled: false` (registration not yet open or for finals).

This file is updated via a regular PR to `v0.1-dev/hackforger`, admin-merged.

## Wipe Strategy (Phase 4 detail)

We chose **approach C — full reset + re-insert essentials**, after considering:
- **A · surgical SQL DELETE**: rejected because the schema has 50+ tables related to hackforger entities (hackathon, bounty, grant, credit, feed, etc.) plus many Forgejo core tables; risk of leaving orphan rows is high.
- **B · pure full reset**: rejected because we'd lose the Logto OAuth config (already documented and in production use, regenerating it requires recreating the OAuth app on Logto's side which we want to avoid).
- **C · full reset + re-insert essentials** (chosen): wipe forgejo.db entirely, let migrations recreate empty schema, then re-INSERT only:
  - The hackforger admin user (via `gitea admin user create` CLI; **new strong random password** stored in `.env`)
  - The Logto `login_source` row (via SQL INSERT from saved cfg, **with explicit `id=3`** to preserve callback URL `/user/oauth2/Logto`)
  - Curated `system_setting` rows: `revision`, `picture.disable_gravatar`, `repository.open-with.editor-apps`, `hackforger.landing.hackathon_prefix` (re-INSERT **without explicit `id`** — let AUTOINCREMENT assign; row identity is `setting_key` UNIQUE INDEX)

**Operational details:**
- Stop the instance with `pkill -TERM` and `sleep 2` before snapshotting `forgejo.db`, otherwise pending WAL writes can be lost.
- Delete with **exact filenames** (`forgejo.db`, `forgejo.db-wal`, `forgejo.db-shm`) — never glob `forgejo.db*`, since `data/forgejo.db.bak.*` rotation backups live there.
- Remove `data/queues/common/LOCK` between stop and restart (LevelDB lock — known restart pitfall, per CLAUDE.md).
- Delete `data/indexers/*.bleve` (stale references after wipe; Forgejo rebuilds on startup).
- **Keep** `data/jwt/private.pem` (preserves OAuth2 access token validity); back it up to dev worktree but don't rotate.

The SQL dump from Phase 2 lets us recover the exact pre-wipe state if anything goes wrong.

## Error Handling & Rollback

| Failure point          | Recovery action                                                              |
|------------------------|------------------------------------------------------------------------------|
| Phase 2 (backup)       | No state changed yet. Investigate disk/permissions, retry.                   |
| Phase 3 (essentials)   | No state changed yet. Re-extract or hand-edit `essentials.sql`.              |
| Phase 4 (wipe + init)  | Restore `data/forgejo.db` from `data-snapshot/forgejo.db.sql`; restore tarballs to `data/` directories; restart. |
| Phase 5 (hackathons)   | DB is clean. Re-run the failing POST(s); each `POST` is idempotent on slug uniqueness (returns 409 if duplicate, treat as success). |
| Phase 6 (landing wire) | Standard PR flow; revert PR if needed.                                       |
| Phase 7 (verify)       | If a specific hackathon fails check, re-create it; if landing routing breaks, revert HTML PR. |

`RESTORE.md` documents the exact commands.

## Testing

| Stage                          | Test                                                                          |
|--------------------------------|-------------------------------------------------------------------------------|
| After Phase 2                  | Verify backup file sizes match pre-wipe state; SQL dump is valid (`sqlite3 :memory: < forgejo.db.sql` parses without error). |
| After Phase 4                  | Instance returns HTTP 200 at `/`; `gitea admin user list \| wc -l == 2` (header + 1 user); `SELECT * FROM login_source` returns Logto row with id=3. |
| After Phase 5                  | `curl /hackathon/opc-2026-shuzhi-w1` ... × 12 all return HTTP 200 with the correct page title; `SELECT count(*) FROM hackathon == 12`. |
| After Phase 6                  | Landing page DOM contains 12 new slugs in `HACKFORGER_LANDING_CONFIG.stages`; clicking S1-W1 routes to `/hackathon/opc-2026-shuzhi-w1`. |
| Final visual                   | Open landing in browser; all 12 cards show titles per #112 (already shipped); buttons reflect today's state per #112 (S1-W1 active, others disabled with date labels). |

## Security & Operational Notes

- The `.env` file at main repo root contains the admin password and new admin token. It is `.gitignore`'d, written with `umask 077` then `chmod 600`. The user is responsible for relocating it to a proper secret store post-init and deleting the local copy.
- The Logto OAuth row is re-inserted with **explicit `id=3`** to preserve the callback URL `/user/oauth2/Logto` and any admin UI bookmarks at `/-/admin/auths/3`. The Logto app on the provider side is unchanged.
- `data/jwt/private.pem` is preserved (not deleted during wipe) so existing OAuth2 access tokens stay valid for their TTL.
- Existing FORGEJO_TOKEN values held by ops/dev tooling become invalid after wipe (rows in `access_token` table are dropped). A new token is generated and saved to `.env`.
- The instance is briefly down (~30–60s) during wipe + restart. Internal usage only; no public users affected.
- We do NOT change `INTERNAL_TOKEN` in app.ini; it stays the same so server-internal cron/auth keeps working.
- **action_runner re-registration** (Phase 7 follow-up): if any GitHub-Actions-style runner is currently registered (used by hackathon submission workflows per `services/hackforger/hackathon.go:165-201`), it must be re-registered after the wipe. `action-runners.sql` in the backup captures pre-wipe runner rows for reference.

## Open Questions

None blocking. Proceed to writing-plans for implementation.
