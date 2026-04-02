# P0 Infrastructure — Manual E2E Test Prompt

> **Purpose:** After completing all P0 tasks, follow this guide to verify the infrastructure works end-to-end on the running HackForger instance.
>
> **Instance:** https://hackforger.inside.h2os.cloud (or `http://localhost:3000` for local dev)
>
> **Prerequisites:** P0 code merged, server started with `./gitea web`, database migrated.

---

## 1. Server Startup Verification

1. Start the server: `./gitea web`
2. Watch the startup logs for:
   - `ORM engine initialization successful!` — DB engine OK
   - No errors mentioning `hackforger` tables or migrations
   - No panic from `RegisterTaskFatal` (would indicate missing i18n keys)
3. Confirm the server is accessible at http://localhost:3000

**What to check:**
- Server starts without errors
- No migration failures in logs
- All 4 cron tasks registered (check admin panel → System → Cron Tasks)

---

## 2. Database Table Verification

1. Log in as **admin**
2. Go to **Site Administration** → **Database** (or use SQLite CLI: `sqlite3 data/gitea.db`)
3. Verify all 16 HackForger tables exist:

| # | Table Name | Expected |
|---|-----------|----------|
| 1 | `hackathon` | exists |
| 2 | `hackathon_track` | exists |
| 3 | `hackathon_registration` | exists |
| 4 | `hackathon_submission` | exists |
| 5 | `hackathon_judge_score` | exists |
| 6 | `bounty` | exists |
| 7 | `bounty_reward` | exists |
| 8 | `bounty_application` | exists |
| 9 | `bounty_winner` | exists |
| 10 | `grant_round` | exists |
| 11 | `grant_project` | exists |
| 12 | `credit_account` | exists |
| 13 | `credit_transaction` | exists |
| 14 | `redeem_option` | exists |
| 15 | `redeem_order` | exists |
| 16 | `reputation` | exists |

**SQLite quick check:**
```sql
.tables
-- Should include all 16 tables above among others
```

---

## 3. API Endpoint Reachability

Use `curl` or browser to verify each stub endpoint returns 200 with empty data.

```bash
# Set your base URL and token
BASE=http://localhost:3000/api/v1
TOKEN=<your-admin-PAT>

# 3a. Hackathons list
curl -s -o /dev/null -w "%{http_code}" "$BASE/hackforger/hackathons?token=$TOKEN"
# Expected: 200

# 3b. Bounties list
curl -s -o /dev/null -w "%{http_code}" "$BASE/hackforger/bounties?token=$TOKEN"
# Expected: 200

# 3c. Grant Rounds list
curl -s -o /dev/null -w "%{http_code}" "$BASE/hackforger/grants/rounds?token=$TOKEN"
# Expected: 200

# 3d. Credits balance
curl -s -o /dev/null -w "%{http_code}" "$BASE/hackforger/credits/balance?token=$TOKEN"
# Expected: 200

# 3e. Feed
curl -s -o /dev/null -w "%{http_code}" "$BASE/hackforger/feed?token=$TOKEN"
# Expected: 200

# 3f. Verify response body (example)
curl -s "$BASE/hackforger/feed?token=$TOKEN" | python3 -m json.tool
# Expected: {"items": [], "total_count": 0}
```

---

## 4. Explore Pages (Web UI)

1. Open browser, navigate to:
   - http://localhost:3000/explore/hackathons
   - http://localhost:3000/explore/bounties
   - http://localhost:3000/explore/grants

2. For each page, verify:
   - Page loads without 500 error
   - The explore navbar is visible (tabs: Repos, Users, Organizations, **Hackathons**, **Bounties**, **Grants**)
   - The "Coming soon..." placeholder text is displayed
   - Active tab is highlighted correctly

3. Test without login (should work — explore pages are public via `ignExploreSignIn`)

---

## 5. Cron Tasks in Admin Panel

1. Log in as **admin**
2. Go to **Site Administration** → **System** → look for Cron Tasks section
3. Verify 4 HackForger cron tasks are listed:

| Task Name | Schedule | Status |
|-----------|----------|--------|
| `hackforger_hackathon_status` | `@every 5m` | Registered |
| `hackforger_bounty_expiry` | `@every 5m` | Registered |
| `hackforger_reputation_recalc` | `@every 1h` | Registered |
| `hackforger_grant_deadline` | `@every 1h` | Registered |

4. Try manually triggering one task — should complete without error (it's a no-op skeleton)

---

## 6. i18n Key Verification

1. Ensure the UI language is set to English
2. On the explore pages, verify:
   - Tab labels show "Hackathons", "Bounties", "Grants" (not raw i18n keys like `hackforger.explore.hackathons`)
   - "Coming soon..." text is rendered (not `hackforger.explore.coming_soon`)

3. Switch language to another locale (e.g., Chinese) — the text should fall back to English (since we only added en-US keys). It should NOT show raw i18n key names.

---

## 7. Frontend Build Verification

1. Open browser DevTools → Console
2. Navigate to any page
3. Verify:
   - No JavaScript errors mentioning `hackforger` or `BountyPanel`
   - The `hackforger-bounty` chunk is NOT loaded on pages without a `#hackforger-bounty-panel` element (lazy loading works)

---

## 8. Migration Idempotency

1. Stop the server
2. Restart the server: `./gitea web`
3. Verify:
   - No migration errors (the migration should be marked as already applied)
   - Server starts cleanly

---

## Done

After completing all checks, fill in the report at `docs/tests/e2e/p0-infrastructure-e2e-report.md`.
