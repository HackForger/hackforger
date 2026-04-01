# Phase 2: Credits Integration + Fulfill — Manual E2E Test Prompt

**Instance:** https://hackforger.inside.h2os.cloud

## Prerequisites

### Start the server (if testing in a worktree)

See [docs/tests/local-testing-guide.md](../local-testing-guide.md) for details.

```bash
# 1. Copy config from main repo (worktrees don't have custom/)
mkdir -p custom/conf
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini

# 2. Build frontend + backend (embeds templates + assets)
make frontend
TAGS="bindata sqlite sqlite_unlock_notify" make backend

# 3. Remove stale LevelDB lock if needed
rm -f /Users/h2oslabs/Workspace/hackforger/data/queues/common/LOCK

# 4. Kill any existing server (IMPORTANT: old process uses old binary!)
kill $(lsof -t -i :3000) 2>/dev/null
sleep 2

# 5. Start server
./gitea web

# 6. Verify the binary is the latest build
./gitea --version
# Should show a timestamp matching your build. If it shows an old commit, re-run make build.
```

Access via https://hackforger.inside.h2os.cloud/ (Caddy must be running -- see local-testing-guide).

> **Critical:** After `make build`, you MUST kill the old server process before starting the new one. The old process keeps running with the old binary in memory even after the file is overwritten. `lsof -i :3000` to find the PID if needed.

### Test accounts

- Admin: `hackforger` / `admin1234`
- Hacker 1: `hacker_eve` (or create one)
- Hacker 2: `hacker_frank` (or create one)
- Judge: `judge_carol` (for hackathon judging)
- If users don't exist, create them via Site Administration -> User Accounts

### Pre-test data check

The database is shared across worktrees. Before testing, verify tables exist and check for leftover data:

```bash
# Verify credits tables exist
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  ".tables" | grep -E "credit_account|redeem_option|redeem_order|redeem_option_key"

# Check for leftover hackathon data
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT id, name, slug, status FROM hackforger_hackathon;"

# Check existing credit balances
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT id, user_id, balance FROM credit_account;"

# Check existing redeem options
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT id, name, cost, stock, fulfill_mode, is_active FROM redeem_option;"
```

If you need a clean slate for credits testing:
```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db <<SQL
  DELETE FROM redeem_option_key;
  DELETE FROM redeem_order;
  DELETE FROM credit_transaction;
  DELETE FROM credit_account;
  DELETE FROM redeem_option;
SQL
```

### Checklist

- [ ] HackForger server running (verified with `./gitea --version`)
- [ ] `custom/conf/app.ini` exists in the worktree (copied from main repo)
- [ ] Admin account accessible
- [ ] Hacker accounts accessible (hacker_eve, hacker_frank)
- [ ] Database migrated (credit_account, redeem_option, redeem_order, redeem_option_key tables exist)
- [ ] Each hacker user owns at least one repository (needed for hackathon submissions)

---

## Test Flow

### A. Hackathon Credits Distribution (3 Tracks, 3 Modes)

This scenario tests prize distribution across hackathon tracks using three different distribution modes.

**A1. Create Hackathon with 3 Tracks**

1. Log in as admin (`hackforger`).
2. Navigate to `/hackathons/new`.
3. Create a hackathon:
   - Name: `Credits Test Hack`
   - Slug: `credits-test-hack`
   - Description: `E2E test for credits distribution`
   - Organization: (select an existing org, or create `test-credits-org` first)
4. **Verify:** Hackathon created in Draft status.
5. Navigate to `/hackathons/credits-test-hack/manage`.
6. Add 3 tracks:
   - Track 1: Name `AI Track`, Prize Credits `1000`, Distribution Mode: **Winner Takes All**
   - Track 2: Name `Web Track`, Prize Credits `900`, Distribution Mode: **Equal Split**
   - Track 3: Name `Mobile Track`, Prize Credits `600`, Distribution Mode: **Tiered**, Ratios: `[{"rank":1,"pct":60},{"rank":2,"pct":40}]`
7. **Verify:** All 3 tracks appear in the tracks list with correct distribution modes.

**A2. Register Hackers + Submit Projects**

1. Click "Publish" to open registration.
2. Log in as `hacker_eve`, register for the hackathon, submit a project to the AI Track.
3. Log in as `hacker_frank`, register, submit a project to the Web Track.
4. Create a third user or use an existing one, register and submit to the Mobile Track.
5. Submit a second project to the Mobile Track (for tiered distribution).
6. **Verify:** Each track has at least one submission.

**A3. Judge and Finalize**

1. Log in as admin.
2. Start Hacking phase, then Start Judging phase.
3. Add `judge_carol` as a judge (or score as admin if allowed).
4. Score all submissions (give different scores for the Mobile Track to test tiered).
5. Click "Finalize".
6. **Verify:** Hackathon status changes to Finished.

**A4. Verify Credits Distribution**

1. Log in as `hacker_eve`.
2. Navigate to `/credits`.
3. **Verify:** Balance shows credits deposited from the AI Track (winner takes all = 1000 credits).
4. **Verify:** Transaction history shows deposit with reference containing `hackathon`.
5. Log in as `hacker_frank`.
6. Navigate to `/credits`.
7. **Verify:** Balance shows credits from the Web Track (equal split).
8. Check the Mobile Track participants to verify tiered distribution (60/40 split of 600 credits).

### B. Manual Fulfill with Delivery

**B1. Create a Redeem Option (Manual)**

1. Log in as admin.
2. Navigate to `/-/admin/credits/options`.
3. Create a new option:
   - Name: `Pro License Key`
   - Cost: `100`
   - Stock: `10`
   - Fulfill Mode: **Manual**
4. **Verify:** Option appears in the list with "Manual" fulfill mode label.

**B2. Redeem as Hacker**

1. Log in as `hacker_eve` (must have >= 100 credits from A4).
2. Navigate to `/credits`.
3. Find "Pro License Key" in the redeem options.
4. Click "Redeem".
5. **Verify:** Confirmation page shows cost (100) and current balance.
6. Confirm redemption.
7. **Verify:** Balance decreased by 100.
8. Navigate to `/credits/orders`.
9. **Verify:** New order with status **Pending**.

**B3. Admin Fulfills with Delivery Details**

1. Log in as admin.
2. Navigate to `/-/admin/credits/orders`.
3. Find hacker_eve's pending order.
4. In the individual fulfill form:
   - Delivery Type: **License Key**
   - Delivery Value: `ABCD-1234-EFGH-5678`
   - Note: `E2E test fulfill`
5. Click "Fulfill".
6. **Verify:** Order status changes to **Fulfilled**.

**B4. Hacker Sees Delivery**

1. Log in as `hacker_eve`.
2. Navigate to `/credits/orders`.
3. **Verify:** Order shows status **Fulfilled**.
4. **Verify:** Delivery column shows "License Key" type and the key value `ABCD-1234-EFGH-5678`.

### C. Auto-Fulfill with Key Pool

**C1. Create Auto-Fulfill Option**

1. Log in as admin.
2. Navigate to `/-/admin/credits/options`.
3. Create a new option:
   - Name: `Game Key`
   - Cost: `50`
   - Stock: `-1` (unlimited)
   - Fulfill Mode: **Auto (Key Pool)**
4. **Verify:** Option appears with "Auto (Key Pool)" label.

**C2. Add Keys to the Pool**

1. In the options list, find "Game Key" and click "Manage Keys".
2. **Verify:** Key pool management page loads showing 0 total / 0 available.
3. In the "Add Keys" textarea, enter:
   ```
   GAME-KEY-001
   GAME-KEY-002
   GAME-KEY-003
   ```
4. Click "Add Keys".
5. **Verify:** Flash message confirms 3 keys added.
6. **Verify:** Key list shows 3 keys, all with "Available" status.

**C3. Redeem and Verify Instant Fulfillment**

1. Log in as `hacker_eve` (needs >= 50 credits).
2. Navigate to `/credits`.
3. Click "Redeem" on "Game Key".
4. Confirm redemption.
5. **Verify:** Balance decreased by 50.
6. Navigate to `/credits/orders`.
7. **Verify:** Order is immediately **Fulfilled** (not Pending).
8. **Verify:** Delivery column shows a key value (e.g., `GAME-KEY-001`).

**C4. Verify Key Pool State**

1. Log in as admin.
2. Navigate to key pool for "Game Key".
3. **Verify:** One key shows "Used" status with the order ID.
4. **Verify:** Pool status shows 3 total / 2 available.

**C5. Exhaust Key Pool**

1. Redeem "Game Key" twice more (as hacker_eve or other users) to use all 3 keys.
2. Try to redeem again.
3. **Verify:** Redemption **fails** with an "out of stock" error — the entire transaction rolls back (balance unchanged).
4. **Verify:** Option is auto-deactivated (no longer visible in redeem options list).

### D. Batch Fulfill

**D1. Create Multiple Pending Orders**

1. If not already present, create a manual redeem option and have 2-3 users redeem it.
2. Log in as admin, navigate to `/-/admin/credits/orders`.
3. **Verify:** Multiple pending orders visible.

**D2. Batch Fulfill Selected Orders**

1. In the batch fulfill section at the top:
   - Select Delivery Type: **Download Link**
   - Enter Delivery Value: `https://example.com/download/v1.0`
2. Check the checkboxes next to 2+ pending orders.
3. Click "Batch Fulfill".
4. **Verify:** Flash message shows "Successfully fulfilled N orders".
5. **Verify:** Selected orders now show **Fulfilled** status.
6. **Verify:** Delivery column shows the download link.

**D3. Partial Batch (Already Fulfilled)**

1. Try to batch fulfill orders that are already fulfilled.
2. **Verify:** Appropriate message (no orders selected, or partial success if mixed).

### E. Order Notifications in Feed

**E1. Fulfill Notification**

1. Log in as admin.
2. Fulfill a pending order (if any remain) for `hacker_eve`.
3. Log in as `hacker_eve`.
4. Navigate to the dashboard, click the **Community** tab (Line-B feed refactor renders HackForger events there).
5. **Verify:** Community feed shows "fulfilled a redeem order" event.

**E2. Cancel Notification**

1. As `hacker_eve`, redeem another option to create a pending order.
2. Log in as admin.
3. Navigate to `/-/admin/credits/orders`.
4. Find the pending order. Click "Cancel".
5. **Verify:** Order status changes to **Cancelled**.
6. **Verify:** hacker_eve's credits are refunded (check balance).
7. Log in as `hacker_eve`.
8. Navigate to the dashboard, click the **Community** tab.
9. **Verify:** Community feed shows "cancelled a redeem order" event.

### F. i18n Verification (zh-CN)

**F1. Switch Language**

1. Log in as admin.
2. Go to Settings -> Appearance -> Language, select "简体中文".
3. Save.

**F2. Verify Credits Pages**

1. Navigate to `/credits`.
2. **Verify:** Page title shows "积分概览".
3. **Verify:** Balance label shows "余额".
4. **Verify:** Redeem options show "兑换选项".
5. Navigate to `/credits/orders`.
6. **Verify:** Page shows "订单" header with correct column headers.
7. **Verify:** Delivery column header shows "交付信息".

**F3. Verify Admin Credits Pages**

1. Navigate to `/-/admin/credits`.
2. **Verify:** Title shows "积分管理".
3. **Verify:** "积分账户" section heading visible.
4. Navigate to `/-/admin/credits/options`.
5. **Verify:** Fulfill mode column shows "手动" or "自动（密钥池）".
6. Navigate to `/-/admin/credits/orders`.
7. **Verify:** Delivery type dropdown shows Chinese labels (许可证密钥, 下载链接, 物流单号, 自定义).
8. **Verify:** Batch fulfill button shows "批量完成".

**F4. Verify Feed in Chinese**

1. Navigate to the dashboard.
2. **Verify:** Feed entries for hackforger events show Chinese text (not blank, not raw keys).

**F5. Reset Language**

1. Switch language back to English.

---

## Report Template

| # | Test Step | Result | Notes |
|---|-----------|--------|-------|
| A1 | Create hackathon with 3 tracks (3 dist modes) | [ ] Pass / [ ] Fail | |
| A2 | Register hackers + submit projects | [ ] Pass / [ ] Fail | |
| A3 | Judge and finalize hackathon | [ ] Pass / [ ] Fail | |
| A4 | Verify credits distributed (winner-takes-all) | [ ] Pass / [ ] Fail | |
| A4b | Verify credits distributed (equal split) | [ ] Pass / [ ] Fail | |
| A4c | Verify credits distributed (tiered 60/40) | [ ] Pass / [ ] Fail | |
| B1 | Create manual redeem option | [ ] Pass / [ ] Fail | |
| B2 | Redeem as hacker (balance decreases) | [ ] Pass / [ ] Fail | |
| B3 | Admin fulfills with license key delivery | [ ] Pass / [ ] Fail | |
| B4 | Hacker sees delivery details in orders | [ ] Pass / [ ] Fail | |
| C1 | Create auto-fulfill option (key pool) | [ ] Pass / [ ] Fail | |
| C2 | Add keys to pool | [ ] Pass / [ ] Fail | |
| C3 | Redeem auto-fulfilled instantly | [ ] Pass / [ ] Fail | |
| C4 | Key pool shows used key | [ ] Pass / [ ] Fail | |
| C5 | Exhausted pool: redeem fails + option deactivated | [ ] Pass / [ ] Fail | |
| D1 | Multiple pending orders exist | [ ] Pass / [ ] Fail | |
| D2 | Batch fulfill with download link | [ ] Pass / [ ] Fail | |
| D3 | Partial batch (already fulfilled) | [ ] Pass / [ ] Fail | |
| E1 | Community tab shows "fulfilled a redeem order" | [ ] Pass / [ ] Fail | |
| E2 | Cancel order + refund + Community tab event | [ ] Pass / [ ] Fail | |
| F1 | Switch to zh-CN | [ ] Pass / [ ] Fail | |
| F2 | Credits pages in Chinese | [ ] Pass / [ ] Fail | |
| F3 | Admin credits pages in Chinese | [ ] Pass / [ ] Fail | |
| F4 | Feed events in Chinese | [ ] Pass / [ ] Fail | |

**Tester:** _______________
**Date:** _______________
**Server URL:** _______________
**Commit/Branch:** _______________
