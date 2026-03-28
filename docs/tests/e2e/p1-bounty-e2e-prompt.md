# Phase 1 Bounty — Manual E2E Test Prompt

> **Purpose:** After completing all Phase 1 Bounty tasks, follow this guide to verify the Exclusive and Competitive bounty flows work end-to-end on the running HackForger instance.
>
> **Instance:** https://hackforger.inside.h2os.cloud (or `http://localhost:3000` for local dev)
>
> **Prerequisites:**
> - P0 infrastructure verified (all 16 tables exist, API endpoints reachable, explore pages working)
> - P0 users/orgs/repos exist: `org_bob`, `hacker_eve`, `hacker_frank`, `hacker_grace`, `agent_hunter`
> - Repo `acme-dev/backend` exists with at least 2 open issues
> - `hacker_frank` and `hacker_grace` follow `hacker_eve`
> - `agent_hunter` follows `org_bob`
> - `hacker_eve` watches `acme-dev/backend`
> - Server running, database migrated

```bash
# Setup variables used throughout this guide
BASE=http://localhost:3000/api/v1
BOB_TOKEN=<org_bob PAT>
EVE_TOKEN=<hacker_eve PAT>
FRANK_TOKEN=<hacker_frank PAT>
GRACE_TOKEN=<hacker_grace PAT>
HUNTER_TOKEN=<agent_hunter PAT>
```

---

## 1. Explore Page Verification

1. Navigate to `/explore/bounties` in the browser (no login required).

2. Verify:
   - Page loads without 500 error
   - Explore navbar shows all tabs including **Bounties** highlighted
   - Empty state or existing bounties are listed
   - Status filter tabs are present: **All / Open / Claimed / Completed**

3. If bounties already exist, verify:
   - Clicking each filter tab updates the list
   - Pagination controls appear when items exceed page size

```bash
# API equivalent
curl -s "$BASE/hackforger/bounties" | python3 -m json.tool
# Expected: [] or list of bounty objects
```

---

## 2. Exclusive Bounty Flow

This mirrors `TestE2E_Phase1_BountyExclusive` from the test plan.

### Step 1: org_bob creates Exclusive Bounty

```bash
curl -s -X POST "$BASE/repos/acme-dev/backend/bounties?token=$BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Refactor auth module",
    "issue_id": 1,
    "mode": "exclusive",
    "deadline": "2026-05-01T00:00:00Z",
    "description": "Refactor the authentication module to use JWT tokens"
  }' | python3 -m json.tool

# Save the returned bounty ID
BOUNTY_1_ID=<id from response>
```

**Verify:**
- Response status: 201 Created
- Response body contains `id`, `title`, `mode: "exclusive"`, `status: "open"`

### Step 2: Add rewards (money + credits)

```bash
# Add money reward
curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_1_ID/rewards?token=$BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "money",
    "amount": 200,
    "currency": "USD",
    "rank": 1
  }' | python3 -m json.tool

# Add credits reward
curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_1_ID/rewards?token=$BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "credits",
    "amount": 500,
    "rank": 1
  }' | python3 -m json.tool
```

**Verify:**
- Both return 201 Created
- GET the bounty and confirm rewards array has 2 entries

### Step 3: Feed check — bounty_created

```bash
# Global feed should contain the bounty_created event
curl -s "$BASE/hackforger/feed?token=$BOB_TOKEN" | python3 -m json.tool
# Look for: action_type 34 (bounty_created), content mentioning "Refactor auth module"

# hacker_eve (watches repo) following feed should contain it
curl -s "$BASE/hackforger/feed?scope=following&token=$EVE_TOKEN" | python3 -m json.tool
# Expected: contains bounty_created event

# agent_hunter (follows org_bob) following feed should contain it
curl -s "$BASE/hackforger/feed?scope=following&token=$HUNTER_TOKEN" | python3 -m json.tool
# Expected: contains bounty_created event

# hacker_grace (no watch/follow on bob) following feed should NOT contain it
curl -s "$BASE/hackforger/feed?scope=following&token=$GRACE_TOKEN" | python3 -m json.tool
# Expected: does NOT contain bounty_created event
# But grace CAN see it in global feed
```

### Step 4: hacker_eve applies for bounty

```bash
curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_1_ID/applications?token=$EVE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "I have experience with JWT auth, happy to take this on."
  }' | python3 -m json.tool

EVE_APP_ID=<id from response>
```

**Verify:** Response status 201 Created

### Step 5: agent_hunter applies for bounty

```bash
curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_1_ID/applications?token=$HUNTER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "My team can deliver this in 3 days."
  }' | python3 -m json.tool
```

**Verify:** Response status 201 Created

### Step 6: org_bob accepts eve's application

```bash
curl -s -X PUT "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_1_ID/applications/$EVE_APP_ID/accept?token=$BOB_TOKEN" \
  | python3 -m json.tool
```

**Verify:**
- Response status 200
- GET the bounty: `status` = `"claimed"`, `claimer_id` = eve's user ID

### Step 7: Feed check — bounty_claimed

```bash
# hacker_frank (follows eve) should see bounty_claimed in following feed
curl -s "$BASE/hackforger/feed?scope=following&token=$FRANK_TOKEN" | python3 -m json.tool
# Look for: action_type 35 (bounty_claimed), mentioning "Refactor auth module"

# hacker_grace (follows eve) should also see it
curl -s "$BASE/hackforger/feed?scope=following&token=$GRACE_TOKEN" | python3 -m json.tool
# Look for: action_type 35 (bounty_claimed)
```

### Step 8: eve creates PR + org_bob merges

1. As hacker_eve, create a branch and PR on `acme-dev/backend` that references the bounty issue.
2. As org_bob, merge the PR.

```bash
# This step is best done via the web UI:
# 1. eve: Create PR referencing issue #1
# 2. bob: Merge the PR
# The MergePullRequest hook should trigger bounty status -> InReview
```

**Verify:**
- After merge, GET the bounty: `status` = `"in_review"` (or `"delivered"`)

### Step 9: Feed check — bounty_delivered

```bash
# Entity feed should contain bounty_delivered event
curl -s "$BASE/hackforger/feed?type=bounty&entity_id=$BOUNTY_1_ID&token=$BOB_TOKEN" \
  | python3 -m json.tool
# Look for: action_type 36 (bounty_delivered)
```

### Step 10: org_bob completes bounty

```bash
curl -s -X PUT "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_1_ID/complete?token=$BOB_TOKEN" \
  | python3 -m json.tool
```

**Verify:**
- Response status 200
- GET the bounty: `status` = `"completed"`

### Step 11: Feed + credits check — bounty_completed

```bash
# eve's credit balance should have increased by 500
curl -s "$BASE/hackforger/credits/balance?token=$EVE_TOKEN" | python3 -m json.tool
# Expected: balance >= 500

# hacker_frank (follows eve) should see bounty_completed
curl -s "$BASE/hackforger/feed?scope=following&token=$FRANK_TOKEN" | python3 -m json.tool
# Look for: action_type 37 (bounty_completed), mentioning "Refactor auth module"
```

### Step 12: org_bob marks paid

```bash
curl -s -X PUT "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_1_ID/paid?token=$BOB_TOKEN" \
  | python3 -m json.tool
```

**Verify:** Response status 200, bounty status = `"paid"`

### Step 13: Verify entity feed completeness

```bash
curl -s "$BASE/hackforger/feed?type=bounty&entity_id=$BOUNTY_1_ID&token=$BOB_TOKEN" \
  | python3 -m json.tool
```

**Verify:** The feed items, ordered chronologically, contain the full lifecycle:
1. `bounty_created` (action_type 34)
2. `bounty_claimed` (action_type 35)
3. `bounty_delivered` (action_type 36)
4. `bounty_completed` (action_type 37)
5. `bounty_paid` (action_type 38 or similar)

---

## 3. Competitive Bounty Flow

This mirrors `TestE2E_Phase1_BountyCompetitive` from the test plan.

### Step 1: org_bob creates Competitive Bounty with ranked rewards

```bash
curl -s -X POST "$BASE/repos/acme-dev/backend/bounties?token=$BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Best CLI tool challenge",
    "issue_id": 2,
    "mode": "competitive",
    "deadline": "2026-05-15T00:00:00Z",
    "description": "Build the best CLI tool for our backend"
  }' | python3 -m json.tool

BOUNTY_2_ID=<id from response>

# Add ranked rewards
curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_2_ID/rewards?token=$BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "credits", "amount": 1000, "rank": 1}' | python3 -m json.tool

curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_2_ID/rewards?token=$BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "credits", "amount": 500, "rank": 2}' | python3 -m json.tool

curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_2_ID/rewards?token=$BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "credits", "amount": 200, "rank": 3}' | python3 -m json.tool
```

**Verify:**
- Bounty created with `mode: "competitive"`, `status: "open"`
- 3 rewards with ranks 1, 2, 3

### Step 2: Feed check — bounty_created (global)

```bash
curl -s "$BASE/hackforger/feed?token=$BOB_TOKEN" | python3 -m json.tool
# Look for: action_type 34, "Best CLI tool challenge"
```

### Step 3: eve, frank, grace each submit applications

```bash
curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_2_ID/applications?token=$EVE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "My CLI tool submission"}' | python3 -m json.tool

curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_2_ID/applications?token=$FRANK_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "Frank CLI submission"}' | python3 -m json.tool

curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_2_ID/applications?token=$GRACE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "Grace CLI submission"}' | python3 -m json.tool
```

**Verify:** All 3 return 201 Created

### Step 4: org_bob starts review and selects winners

```bash
# Start review phase
curl -s -X PUT "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_2_ID/review?token=$BOB_TOKEN" \
  | python3 -m json.tool

# Select winners (eve=1st, frank=2nd, grace=3rd)
curl -s -X POST "$BASE/repos/acme-dev/backend/bounties/$BOUNTY_2_ID/winners?token=$BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "winners": [
      {"user_id": <eve_user_id>, "rank": 1},
      {"user_id": <frank_user_id>, "rank": 2},
      {"user_id": <grace_user_id>, "rank": 3}
    ]
  }' | python3 -m json.tool
```

**Verify:** Response status 200, bounty status = `"completed"`

### Step 5: Feed check — winners_selected (global)

```bash
curl -s "$BASE/hackforger/feed?token=$BOB_TOKEN" | python3 -m json.tool
# Look for: action_type 38 (winners_selected), "Best CLI tool challenge"
```

### Step 6: Verify credits distribution

```bash
# eve: should have gained 1000 credits (plus 500 from exclusive bounty = 1500 total)
curl -s "$BASE/hackforger/credits/balance?token=$EVE_TOKEN" | python3 -m json.tool
# Expected: balance >= 1500

# frank: should have gained 500 credits
curl -s "$BASE/hackforger/credits/balance?token=$FRANK_TOKEN" | python3 -m json.tool
# Expected: balance >= 500

# grace: should have gained 200 credits
curl -s "$BASE/hackforger/credits/balance?token=$GRACE_TOKEN" | python3 -m json.tool
# Expected: balance >= 200
```

---

## 4. Issue Panel Verification

1. Navigate to `acme-dev/backend` issue #1 (the one with the exclusive bounty) in the browser.

2. **Verify (logged out or any user):**
   - A Bounty panel appears in the issue sidebar
   - Panel shows the correct status label (e.g., "Paid" or "Completed")
   - Rewards are displayed (200 USD + 500 credits)
   - Claimer name (hacker_eve) is shown

3. **Verify (logged in as org_bob — the publisher):**
   - Action buttons appear in the bounty panel (e.g., "Mark Paid", "Complete")
   - Buttons are contextually disabled/enabled based on current bounty status

4. Navigate to issue #2 (competitive bounty) and verify:
   - Bounty panel shows `mode: competitive`
   - Winners are listed with their ranks

---

## 5. Issue List Badge

1. Navigate to `acme-dev/backend` issue list (`/acme-dev/backend/issues`).

2. **Verify:**
   - Issues #1 and #2 show a green "Bounty" badge next to the issue title
   - Issues without bounties do NOT show the badge
   - The badge is visible without clicking into the issue

---

## 6. Web Create Flow

1. Log in as `org_bob`.
2. Navigate to `/acme-dev/backend/bounties/new` (or click "New Bounty" from the repo page).

3. **Fill in the form:**
   - Title: "Fix database connection pooling"
   - Issue: select or enter an issue ID
   - Mode: Exclusive
   - Deadline: any future date

4. Submit the form.

5. **Verify:**
   - Form submits without error
   - Redirects to the issue page with the new bounty panel visible
   - Bounty appears in `/explore/bounties` listing

---

## Done

After completing all checks, fill in the report at `docs/tests/e2e/p1-bounty-e2e-report.md`.
