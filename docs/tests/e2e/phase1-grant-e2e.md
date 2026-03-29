# Phase 1: Grant + Credits — Manual E2E Test Prompt

**Instance:** https://hackforger.inside.h2os.cloud

## Prerequisites

### Start the server (if testing in a worktree)

See [docs/tests/local-testing-guide.md](../local-testing-guide.md) for details.

```bash
# 1. Copy config from main repo (worktrees don't have custom/)
mkdir -p custom/conf
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini

# 2. Build backend (embeds templates + assets)
TAGS="bindata sqlite sqlite_unlock_notify" make build

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

Access via https://hackforger.inside.h2os.cloud/ (Caddy must be running — see local-testing-guide).

> **Critical:** After `make build`, you MUST kill the old server process before starting the new one. The old process keeps running with the old binary in memory even after the file is overwritten. `lsof -i :3000` to find the PID if needed.

### Test accounts

- Admin: `hackforger` / `admin1234`
- Hacker 1: `hacker_eve` (or create one)
- Hacker 2: `hacker_frank` (for budget exceed test)
- If users don't exist, create them via Site Administration → User Accounts

### Pre-test data check

The database is shared across worktrees. Before testing, verify no leftover grant data from a previous run:

```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT id, name, slug, status FROM grant_round;"
# If rows exist, clean up:
# DELETE FROM grant_project WHERE round_id IN (SELECT id FROM grant_round WHERE slug LIKE 'spring-2026%');
# DELETE FROM grant_round WHERE slug LIKE 'spring-2026%';
```

### Checklist

- [ ] HackForger server running (verified with `./gitea --version`)
- [ ] `custom/conf/app.ini` exists in the worktree (copied from main repo)
- [ ] Admin account accessible
- [ ] Hacker accounts accessible
- [ ] Database migrated (grant_round, grant_project, credit_account tables exist)

---

## Test Flow

### A. Explore + Create Entry Points

**A1. Explore Grants (Public)**

1. Without logging in, navigate to `/explore/grants`.
2. **Verify:** Page loads with "No grant rounds found." message.
3. **Verify:** Search bar and sort dropdown are present.
4. **Verify:** Status filter tabs show (All, draft, open, review, finalized, distributed, cancelled).

**A2. Create via Navbar**

1. Log in as admin.
2. Click the "+" dropdown in the top navbar.
3. **Verify:** "New Grant Round" entry is visible.
4. Click it.
5. **Verify:** Redirects to `/grants/new`.

### B. Grant Round Lifecycle (Admin)

**B1. Create Organization**

1. Create a new organization called `test-grants-org` (or use an existing org where admin is owner).
2. **Verify:** Org page loads at `/test-grants-org`.

**B2. Create Grant Round**

1. Navigate to `/grants/new`.
2. Fill in:
   - Name: `Spring 2026 Grants`
   - Slug: `spring-2026`
   - Description: `Test grant round for E2E validation.`
   - Budget: `10000`
   - Currency: `USD`
   - Budget (Credits): `5000`
   - Deadline: a date in the future
   - Organization: `test-grants-org`
3. Submit.
4. **Verify:** Round is created with status **Draft**.
5. **Verify:** Redirected to round detail page `/grants/spring-2026`.
6. **Verify:** Status progress bar shows Draft as active.

**B3. Open the Round**

1. Navigate to `/grants/spring-2026/manage`.
2. Click "Open Round" button.
3. **Verify:** Status changes to **Open**.
4. **Verify:** "Submit Project" button now appears on detail page.

**B4. Verify on Explore Page**

1. Navigate to `/explore/grants`.
2. **Verify:** "Spring 2026 Grants" appears in the list.
3. Click "Open" status tab → verify it filters correctly.
4. Search for "Spring" → verify it appears.
5. Sort by "Most funded" → verify no error.

### C. Project Submission (Hacker)

**C1. Submit a Project**

1. Log out. Log in as `hacker_eve`.
2. Navigate to `/grants/spring-2026`.
3. Click "Submit Project".
4. Fill in:
   - Title: `Open Source Widget`
   - Description: `A widget that does amazing things.`
   - Repository: (optional, leave empty or select one)
5. Submit.
6. **Verify:** Project status is **Pending**.
7. **Verify:** Success flash message.

**C2. Duplicate Submission (Error Case)**

1. While still logged in as `hacker_eve`, navigate to `/grants/spring-2026/submit` again.
2. Submit another project.
3. **Verify:** Error message about already having submitted.

**C3. Second Hacker Submits**

1. Log in as `hacker_frank`.
2. Submit a project to the same round (Title: `Another Widget`).
3. **Verify:** Submission succeeds.

### D. Review + Allocation (Admin)

**D1. Close Applications**

1. Log in as admin.
2. Navigate to `/grants/spring-2026/manage`.
3. Click "Close Applications".
4. **Verify:** Status changes to **Review**.

**D2. Approve Projects**

1. In the manage page, click on `hacker_eve`'s project.
2. Click "Approve".
3. **Verify:** Project status → **Approved**.
4. Go back, approve `hacker_frank`'s project too.

**D3. Allocate Awards**

1. On `hacker_eve`'s project, enter Award Credits: `3000`, Award Amount: `5000`.
2. Submit allocation.
3. **Verify:** Awards recorded on project.

**D4. Exceed Budget (Error Case)**

1. On `hacker_frank`'s project, enter Award Credits: `3000` (total would be 6000, exceeding 5000 budget).
2. Submit.
3. **Verify:** Error about exceeding credits budget.

**D5. Allocate Within Budget**

1. Allocate `hacker_frank`: Credits `2000`, Amount `4000`.
2. **Verify:** Allocation succeeds. Budget usage bar shows 5000/5000 credits, 9000/10000 amount.

**D6. Finalize Round**

1. Click "Finalize Allocations".
2. **Verify:** Status changes to **Finalized**.

### E. Distribution

**E1. Distribute First Project**

1. On the finalized round manage page, click "Distribute" on `hacker_eve`'s project.
2. **Verify:** Project status → **Funded**.
3. **Verify:** Round is still **Finalized** (frank's project not yet distributed).

**E2. Distribute Second Project (Auto-Transition)**

1. Click "Distribute" on `hacker_frank`'s project.
2. **Verify:** Project status → **Funded**.
3. **Verify:** Round status auto-transitions to **Distributed**.

### F. Credits Verification

**F1. Check Hacker Balance**

1. Log in as `hacker_eve`.
2. Navigate to `/credits`.
3. **Verify:** Balance shows `3000` credits.
4. **Verify:** Transaction history shows deposit with reference `grant_round:X/project:Y`.

**F2. Create Redeem Option (Admin)**

1. Log in as admin.
2. Navigate to `/-/admin/credits/options`.
3. Create a new option: Name: `Sticker Pack`, Cost: `100`, Stock: `10`.
4. **Verify:** Option appears in the list.

**F3. Redeem Option (Hacker)**

1. Log in as `hacker_eve`.
2. Navigate to `/credits`.
3. Find "Sticker Pack" in the redeem options.
4. Click "Redeem".
5. **Verify:** Confirmation page shows cost and current balance.
6. Confirm.
7. **Verify:** Balance decreased by 100 (3000 → 2900).
8. Navigate to `/credits/orders`.
9. **Verify:** New order with status **Pending**.

**F4. Insufficient Credits (Error Case)**

1. As `hacker_frank` (balance 2000), try to redeem an option costing more than 2000.
2. **Verify:** Error: insufficient credits.

### G. Credits Administration (Admin)

**G1. Deposit Credits**

1. Log in as admin.
2. Navigate to `/-/admin/credits`.
3. Deposit `500` credits to `hacker_eve` with reference "bonus", note "E2E test".
4. **Verify:** Eve's balance increases (2900 → 3400).

**G2. Deduct Credits**

1. Deduct `100` credits from `hacker_eve` with reference "correction".
2. **Verify:** Eve's balance decreases (3400 → 3300).

**G3. Fulfill Order**

1. Navigate to `/-/admin/credits/orders`.
2. Find Eve's pending order.
3. Click "Fulfill", enter note: "Shipped via mail."
4. **Verify:** Order status → **Fulfilled**.

**G4. Cancel Order + Refund**

1. As `hacker_eve`, redeem another option (Sticker Pack, 100 credits).
2. As admin, navigate to orders, find the new pending order.
3. Click "Cancel".
4. **Verify:** Order status → **Cancelled**.
5. **Verify:** Credits refunded (check Eve's balance).

### H. Feed Verification (Dashboard)

1. Log in as admin.
2. Navigate to the dashboard (home page).
3. Check the activity feed for HackForger events. **Verify** these appear with proper text (not blank):

- [ ] Grant round created (by admin)
- [ ] Grant round opened
- [ ] Project submitted (by hacker_eve)
- [ ] Grant round closed for review
- [ ] Grant round finalized
- [ ] Grant awarded / project funded (for each project)
- [ ] Credits redeemed (by hacker_eve)

> **Note:** Feed audience is currently actor + global only. Follower/org-member audience resolution is deferred post-merge.

### I. Export

**I1. CSV Export**

1. As admin, navigate to `/grants/spring-2026/export`.
2. **Verify:** CSV file downloads.
3. **Verify:** Contains columns: id, title, user_id, status, award_amount, award_credits.
4. **Verify:** Both projects are listed with correct data.

### J. Cancel Round (Separate Test)

1. Create a new round (e.g., slug `cancel-test`), open it.
2. Navigate to manage, click "Cancel Round".
3. **Verify:** Status → **Cancelled**.
4. **Verify:** Cannot submit projects to a cancelled round.

---

## Report Template

| # | Test Step | Result | Notes |
|---|-----------|--------|-------|
| A1 | Explore grants page loads | [ ] Pass / [ ] Fail | |
| A2 | Navbar "+" has Grant Round entry | [ ] Pass / [ ] Fail | |
| B1 | Create organization | [ ] Pass / [ ] Fail | |
| B2 | Create grant round (Draft) | [ ] Pass / [ ] Fail | |
| B3 | Open round | [ ] Pass / [ ] Fail | |
| B4 | Explore search + filter + sort | [ ] Pass / [ ] Fail | |
| C1 | Submit project (Pending) | [ ] Pass / [ ] Fail | |
| C2 | Duplicate submission error | [ ] Pass / [ ] Fail | |
| C3 | Second hacker submits | [ ] Pass / [ ] Fail | |
| D1 | Close applications (Review) | [ ] Pass / [ ] Fail | |
| D2 | Approve projects | [ ] Pass / [ ] Fail | |
| D3 | Allocate awards | [ ] Pass / [ ] Fail | |
| D4 | Exceed budget error | [ ] Pass / [ ] Fail | |
| D5 | Allocate within budget | [ ] Pass / [ ] Fail | |
| D6 | Finalize round | [ ] Pass / [ ] Fail | |
| E1 | Distribute first project | [ ] Pass / [ ] Fail | |
| E2 | Distribute second + auto-transition | [ ] Pass / [ ] Fail | |
| F1 | Hacker balance correct | [ ] Pass / [ ] Fail | |
| F2 | Create redeem option (admin) | [ ] Pass / [ ] Fail | |
| F3 | Redeem option + order created | [ ] Pass / [ ] Fail | |
| F4 | Insufficient credits error | [ ] Pass / [ ] Fail | |
| G1 | Admin deposit credits | [ ] Pass / [ ] Fail | |
| G2 | Admin deduct credits | [ ] Pass / [ ] Fail | |
| G3 | Fulfill order | [ ] Pass / [ ] Fail | |
| G4 | Cancel order + refund | [ ] Pass / [ ] Fail | |
| H | Feed events rendered (not blank) | [ ] Pass / [ ] Fail | |
| I1 | CSV export | [ ] Pass / [ ] Fail | |
| J | Cancel round | [ ] Pass / [ ] Fail | |

**Tester:** _______________
**Date:** _______________
**Server URL:** _______________
**Commit/Branch:** _______________
