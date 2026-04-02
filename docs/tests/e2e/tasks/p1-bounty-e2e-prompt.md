# Phase 1 Bounty — E2E Test Prompt

> **Instance:** https://hackforger.inside.h2os.cloud
> **Login:** hackforger / admin1234 (ID=1)
> **API Token:** `c3a899f2b4442797d96205fb50c8829c7724edf4`
> **Test users:** hacker_eve(ID=3) / hacker_frank(ID=4) / hacker_grace(ID=5), password: `password`
> **Credits page:** `/credits` (requires login)
> **Credits API:** `/api/v1/hackforger/credits/balance?token=TOKEN` (admin: add `&user_id=N`)
> **Feed API:** `/api/v1/hackforger/feed?token=TOKEN&type=global`

## Test Data

| Bounty | Issue | Title | Mode | Rewards |
|--------|-------|-------|------|---------|
| 18 | #21 | Implement rate limiting middleware | Exclusive | USD 200 + 500 credits |
| 19 | #22 | Design plugin architecture | Competitive | 1000/500/200 credits (1st/2nd/3rd) |

Issue #6: no bounty (test "Create Bounty" button)

---

## 1. Explore Page (UI)

1. Visit `/explore/bounties` → 2 bounties, titles link to Issues
2. Switch to Chinese → navbar shows "黑客松 / 悬赏 / 资助"
3. Filter tabs work (全部 / 开放 / 已认领 / 已完成)

## 2. Issue Panel (UI)

1. `/acme-dev/backend/issues/21` → Bounty panel: "开放" (green), USD 200 / 500 credits
2. `/acme-dev/backend/issues/6` (no bounty) → "创建悬赏" green button
3. **Verify: no "token is required" error on page load**

## 3. Exclusive Bounty Full Flow (Bounty 18, Issue #21)

### 3a. Apply (UI — hacker_eve)
1. Log in as hacker_eve / password
2. Visit `/acme-dev/backend/issues/21`
3. Panel shows "Apply" button → click → enter "I can implement this" → submit
4. Verify: application submitted (button disappears or shows applied state)

### 3b. Accept Application (UI — hackforger)
1. Log in as hackforger / admin1234
2. Visit `/acme-dev/backend/issues/21`
3. Panel shows application from hacker_eve → click "Accept"
4. **★ Verify via UI: no "token is required" error, panel updates to "已认领" (blue)**
5. Verify: claimer shows "hacker_eve" (not "User #3")

API verification:
```bash
T=c3a899f2b4442797d96205fb50c8829c7724edf4
curl -sk "https://hackforger.inside.h2os.cloud/api/v1/repos/acme-dev/backend/bounties/18?token=$T"
# Expect: Status=1, ClaimerID=3
```

### 3c. PR Create + Merge (UI)
1. Log in as hacker_eve
2. Create branch `eve-rate-limit` in acme-dev/backend repo
3. Commit a file (e.g., `rate_limiter.go`), message: `feat: rate limiting - fixes #21`
4. Create Pull Request
5. Log in as hackforger → merge PR

### 3d. ★ Verify InReview (UI + API)
1. Refresh `/acme-dev/backend/issues/21`
2. **Panel status should be "审核中" (orange)**
3. Should show "Complete Bounty" + "Reject Delivery" buttons
```bash
curl -sk "https://hackforger.inside.h2os.cloud/api/v1/repos/acme-dev/backend/bounties/18?token=$T"
# Expect: Status=2 (InReview)
```

### 3e. ★ Complete via UI Button (UI — hackforger)
1. On Issue #21, click **"Complete Bounty" button**
2. **★ KEY: no "token is required" error — button works via web route**
3. Panel status changes to "已完成" (purple)
4. "Mark as Paid" button appears

### 3f. Verify Credits (UI + API)
1. Log in as hacker_eve → visit `/credits`
2. **Balance shows 500**, transaction history shows 1 deposit entry
```bash
curl -sk "https://hackforger.inside.h2os.cloud/api/v1/hackforger/credits/balance?token=$T&user_id=3"
# Expect: balance=500
```

### 3g. ★ Mark Paid via UI Button (UI — hackforger)
1. On Issue #21, click **"Mark as Paid" button**
2. **★ KEY: no error — button works**
3. Panel status changes to "已支付" (teal)
4. No action buttons remain (terminal state)

### 3h. Verify Auto-Close
1. Check Issue #21 status — **should be Closed** (auto-closed by CompleteBounty)

## 4. Competitive Bounty Full Flow (Bounty 19, Issue #22)

### 4a. Multiple Applications (UI or API)
Log in as each user and apply on Issue #22, or use API:
```bash
T=c3a899f2b4442797d96205fb50c8829c7724edf4
curl -sk -X POST "https://hackforger.inside.h2os.cloud/api/v1/repos/acme-dev/backend/bounties/19/applications?token=$T" \
  -H 'Sudo: hacker_eve' -H 'Content-Type: application/json' -d '{"message":"entry 1"}'
curl -sk -X POST "https://hackforger.inside.h2os.cloud/api/v1/repos/acme-dev/backend/bounties/19/applications?token=$T" \
  -H 'Sudo: hacker_frank' -H 'Content-Type: application/json' -d '{"message":"entry 2"}'
curl -sk -X POST "https://hackforger.inside.h2os.cloud/api/v1/repos/acme-dev/backend/bounties/19/applications?token=$T" \
  -H 'Sudo: hacker_grace' -H 'Content-Type: application/json' -d '{"message":"entry 3"}'
```

### 4b. Start Review (UI — hackforger)
1. Issue #22 → click "Start Review" → status "审核中"

### 4c. ★ Select Winners (UI — hackforger)
1. Panel shows "Select Winners" button → click
2. Form: enter User 3 (Rank 1), click "+ Add", User 4 (Rank 2), click "+ Add", User 5 (Rank 3)
3. Click "Submit Winners"
4. **★ Verify: no error, status changes to "已完成"**

### 4d. ★ Verify Winners Display (UI)
1. Refresh Issue #22
2. **SSR "选择获奖者": hacker_eve #1, hacker_frank #2, hacker_grace #3**
3. **Vue "Winners": shows usernames (not "User #N")**

### 4e. Verify Auto-Close
1. Issue #22 status — **should be Closed**

### 4f. Verify Credits (UI + API)
```bash
curl -sk "https://hackforger.inside.h2os.cloud/api/v1/hackforger/credits/balance?token=$T&user_id=3"  # eve: 1500
curl -sk "https://hackforger.inside.h2os.cloud/api/v1/hackforger/credits/balance?token=$T&user_id=4"  # frank: 500
curl -sk "https://hackforger.inside.h2os.cloud/api/v1/hackforger/credits/balance?token=$T&user_id=5"  # grace: 200
```
Also verify via UI: each user → `/credits` → correct balance + transactions

## 5. Feed Verification (API)
```bash
curl -sk "https://hackforger.inside.h2os.cloud/api/v1/hackforger/feed?token=$T&type=global"
# Expect: bounty_created(34) x2, bounty_winners_selected(38) x1
# Note: claimed/delivered/completed are Followers+Watchers events, not Global
```

## 6. Web Create Flow (UI)

1. hackforger → `/acme-dev/backend/issues/6` → click "创建悬赏"
2. Form pre-fills Issue ID + title → select mode → set deadline → submit
3. Redirect to Issue #6 → Panel shows new bounty
4. `/explore/bounties` → new bounty appears

## 7. Issue List Badge (UI)

1. `/acme-dev/backend/issues` → bounty issues show green "悬赏" badge
2. Non-bounty issues don't show badge

---

After testing, update `docs/tests/e2e/p1-bounty-e2e-report.md`.
