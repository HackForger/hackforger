# Phase 1 Bounty — E2E Test Report

> **Tester:** ___
> **Date:** ___
> **Instance:** https://hackforger.inside.h2os.cloud
> **Branch/Commit:** ___

---

## 1. Explore Page Verification

- [ ] `/explore/bounties` loads without error
- [ ] Explore navbar shows Bounties tab highlighted
- [ ] Empty state or bounty listing renders correctly
- [ ] Status filter tabs present (All / Open / Claimed / Completed)
- [ ] Filters update the list when clicked
- [ ] Pagination works (if enough bounties exist)

**Notes:** ___

---

## 2. Exclusive Bounty Flow

### Step 1: Create Exclusive Bounty
- [ ] POST returns 201 Created
- [ ] Response contains correct `mode: "exclusive"`, `status: "open"`

### Step 2: Add Rewards
- [ ] Money reward added (201)
- [ ] Credits reward added (201)
- [ ] GET bounty shows 2 rewards

### Step 3: Feed — bounty_created
- [ ] Global feed contains bounty_created event (action_type 34)
- [ ] hacker_eve (repo watcher) sees event in following feed
- [ ] agent_hunter (follows org_bob) sees event in following feed
- [ ] hacker_grace (no relationship) does NOT see event in following feed
- [ ] hacker_grace CAN see event in global feed

### Step 4: hacker_eve applies
- [ ] Application POST returns 201

### Step 5: agent_hunter applies
- [ ] Application POST returns 201

### Step 6: org_bob accepts eve's application
- [ ] Accept PUT returns 200
- [ ] Bounty status = "claimed"
- [ ] Bounty claimer_id = eve's user ID

### Step 7: Feed — bounty_claimed
- [ ] hacker_frank (follows eve) sees bounty_claimed in following feed
- [ ] hacker_grace (follows eve) sees bounty_claimed in following feed

### Step 8: eve creates PR + org_bob merges
- [ ] PR created successfully
- [ ] PR merged successfully
- [ ] Bounty status transitions to "in_review" / "delivered"

### Step 9: Feed — bounty_delivered
- [ ] Entity feed contains bounty_delivered event (action_type 36)

### Step 10: org_bob completes bounty
- [ ] Complete PUT returns 200
- [ ] Bounty status = "completed"

### Step 11: Feed + Credits — bounty_completed
- [ ] hacker_eve credit balance increased by 500
- [ ] hacker_frank sees bounty_completed in following feed

### Step 12: org_bob marks paid
- [ ] Paid PUT returns 200
- [ ] Bounty status = "paid"

### Step 13: Entity feed completeness
- [ ] Feed contains: created -> claimed -> delivered -> completed -> paid (in order)

**Notes:** ___

---

## 3. Competitive Bounty Flow

### Step 1: Create Competitive Bounty + ranked rewards
- [ ] Bounty created with `mode: "competitive"`, `status: "open"`
- [ ] 3 ranked rewards added (1000, 500, 200 credits)

### Step 2: Feed — bounty_created (global)
- [ ] Global feed contains bounty_created event for competitive bounty

### Step 3: Submissions
- [ ] hacker_eve application returns 201
- [ ] hacker_frank application returns 201
- [ ] hacker_grace application returns 201

### Step 4: Review + select winners
- [ ] Review PUT returns 200
- [ ] Winners POST returns 200 (eve=1st, frank=2nd, grace=3rd)
- [ ] Bounty status = "completed"

### Step 5: Feed — winners_selected (global)
- [ ] Global feed contains winners_selected event

### Step 6: Credits distribution
- [ ] hacker_eve balance >= 1500 (500 from exclusive + 1000 from competitive)
- [ ] hacker_frank balance >= 500
- [ ] hacker_grace balance >= 200

**Notes:** ___

---

## 4. Issue Panel Verification

- [ ] Bounty panel appears in issue sidebar for issue #1
- [ ] Panel shows correct status label
- [ ] Rewards displayed (200 USD + 500 credits)
- [ ] Claimer name shown (hacker_eve)
- [ ] As org_bob: action buttons appear
- [ ] Buttons contextually enabled/disabled based on status
- [ ] Issue #2 panel shows mode: competitive
- [ ] Issue #2 panel shows winners with ranks

**Notes:** ___

---

## 5. Issue List Badge

- [ ] Issues with bounties show green "Bounty" badge
- [ ] Issues without bounties do NOT show the badge
- [ ] Badge visible in issue list view (no click-through needed)

**Notes:** ___

---

## 6. Web Create Flow

- [ ] `/acme-dev/backend/bounties/new` page loads
- [ ] Form fields: title, issue, mode, deadline present
- [ ] Form submits without error
- [ ] Redirects to issue page with bounty panel
- [ ] New bounty appears in `/explore/bounties`

**Notes:** ___

---

## Overall Result

- [ ] **PASS** — All checks passed, Phase 1 Bounty is ready
- [ ] **FAIL** — Issues found (see notes above)

**Blockers for next phase:** ___

**Other observations:** ___
