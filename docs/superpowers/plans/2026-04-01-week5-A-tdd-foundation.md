# Week 5 Sub-Plan A: TDD Foundation

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enrich test fixtures to cover all entity states, apply 4 API renames via TDD, and create integration test skeleton with shared helpers.

**Architecture:** Fixtures derived from 6 user journeys (73 steps). API renames applied TDD-style: write test with new path → get 404 → rename route → test passes. Test helpers follow Forgejo's existing `tests/integration/` patterns.

**Tech Stack:** YAML fixtures, Go integration tests, XORM, chi router.

**Spec:** `docs/superpowers/specs/2026-04-01-week5-phase5-design.md` (Sections 2, 3, 4.1-4.2)

---

## Chunk 1: Fixtures Enrichment

### Task 1: Add HackForger test users to user.yml

**Files:**
- Modify: `models/fixtures/user.yml`

The existing `user.yml` has users with IDs 1-40+. We need to ensure our HackForger roles map to specific user IDs consistently. Check existing users first, then add any missing ones.

- [ ] **Step 1.1: Audit existing user fixtures**

Read `models/fixtures/user.yml` and identify which IDs are available. Map HackForger roles:

| Role | Username | Preferred ID | Notes |
|------|----------|-------------|-------|
| admin | user1 (existing) | 1 | Already admin |
| organizer | user2 (existing) | 2 | Org owner for org 3 |
| hacker1 | user4 (existing) | 4 | Has credit_account with balance 1000 |
| hacker2 | user5 (existing) | 5 | No credit account yet |
| judge1 | user8 or new | 8 | Need to verify |
| judge2 | user9 or new | 9 | Need to verify |
| platform-bot | new | next available | Type=4 (bot) |

- [ ] **Step 1.2: Add platform-bot user if not exists**

Add to `models/fixtures/user.yml`:

```yaml
-
  id: <next_available_id>
  lower_name: "platform-bot"
  name: "platform-bot"
  full_name: "Platform Bot"
  email: "platform-bot@localhost"
  type: 4  # UserTypeBot
  is_admin: false
  created_unix: 1609459200
  updated_unix: 1609459200
```

- [ ] **Step 1.3: Verify and commit user fixtures**

Run: `go test ./models/hackforger/... -tags="bindata sqlite sqlite_unlock_notify" -v -count=1 2>&1 | tail -5`
Expected: existing tests still pass.

```bash
git add models/fixtures/user.yml
git commit -m "test: add platform-bot user to fixtures for Week 5 integration tests"
```

### Task 2: Enrich Hackathon fixtures (6 status values)

**Files:**
- Modify: `models/fixtures/hackathon.yml`
- Modify: `models/fixtures/hackathon_track.yml`
- Modify: `models/fixtures/hackathon_registration.yml`
- Modify: `models/fixtures/hackathon_submission.yml`
- Modify: `models/fixtures/hackathon_judge_score.yml`
- Create: `models/fixtures/hackathon_track_criteria.yml` (XORM table name for `HackathonTrackCriteria`)
- Create: `models/fixtures/hackathon_judge_criteria.yml` (XORM table name for `HackathonJudgeCriteria`)

- [ ] **Step 2.1: Read current hackathon fixtures**

Read all hackathon-related fixture files to understand current state.

- [ ] **Step 2.2: Write hackathon.yml with 6 records**

Each record covers one status. Use org_id=3 (existing org) and owner_id=2 (organizer):

```yaml
# hackathon.yml — 6 records, one per HackathonStatus
-
  id: 1
  org_id: 3
  owner_id: 2
  name: "Draft Hackathon"
  slug: "draft-hackathon"
  description: "A hackathon in draft status"
  status: 0  # Draft
  max_team_size: 5
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 2
  org_id: 3
  owner_id: 2
  name: "Open Hackathon"
  slug: "open-hackathon"
  description: "A hackathon accepting registrations"
  status: 1  # Open
  max_team_size: 4
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 3
  org_id: 3
  owner_id: 2
  name: "Hacking Hackathon"
  slug: "hacking-hackathon"
  description: "A hackathon in the hacking phase"
  status: 2  # Hacking
  max_team_size: 5
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 4
  org_id: 3
  owner_id: 2
  name: "Judging Hackathon"
  slug: "judging-hackathon"
  description: "A hackathon being judged"
  status: 3  # Judging
  max_team_size: 5
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 5
  org_id: 3
  owner_id: 2
  name: "Finished Hackathon"
  slug: "finished-hackathon"
  description: "A completed hackathon with results"
  status: 4  # Finished
  max_team_size: 5
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 6
  org_id: 3
  owner_id: 2
  name: "Cancelled Hackathon"
  slug: "cancelled-hackathon"
  description: "A cancelled hackathon"
  status: 5  # Cancelled
  max_team_size: 5
  created_unix: 1609459200
  updated_unix: 1609459200
```

- [ ] **Step 2.3: Write hackathon_track.yml**

Tracks for hackathons that need them (Hacking, Judging, Finished):

```yaml
-
  id: 1
  hackathon_id: 3  # Hacking Hackathon
  name: "Web Track"
  slug: "web-track"
  repo_id: 1
  description: "Build a web app"
  prize_pool: 1000
  prize_currency: "credits"
  prize_dist_json: '[{"rank":1,"ratio":50},{"rank":2,"ratio":30},{"rank":3,"ratio":20}]'
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 2
  hackathon_id: 4  # Judging Hackathon
  name: "AI Track"
  slug: "ai-track"
  repo_id: 2
  description: "Build an AI tool"
  prize_pool: 2000
  prize_currency: "credits"
  prize_dist_json: '[{"rank":1,"ratio":60},{"rank":2,"ratio":40}]'
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 3
  hackathon_id: 5  # Finished Hackathon
  name: "CLI Track"
  slug: "cli-track"
  repo_id: 3
  description: "Build a CLI tool"
  prize_pool: 500
  prize_currency: "credits"
  prize_dist_json: '[{"rank":1,"ratio":100}]'
  created_unix: 1609459200
  updated_unix: 1609459200
```

- [ ] **Step 2.4: Write hackathon_registration.yml (4 records, 3 statuses)**

```yaml
-
  id: 1
  hackathon_id: 3
  user_id: 4  # hacker1
  status: 1   # Approved
  team_name: "Team Alpha"
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 2
  hackathon_id: 3
  user_id: 5  # hacker2
  status: 1   # Approved
  team_name: "Team Beta"
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 3
  hackathon_id: 2  # Open Hackathon
  user_id: 4  # hacker1
  status: 0   # Pending
  team_name: ""
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 4
  hackathon_id: 2  # Open Hackathon
  user_id: 5  # hacker2
  status: 2   # Rejected
  team_name: ""
  created_unix: 1609459200
  updated_unix: 1609459200
```

- [ ] **Step 2.5: Write hackathon_submission.yml, hackathon_judge_score.yml, criteria fixtures**

Submissions for Judging Hackathon (id=4), scores from judge1 (user 8) and judge2 (user 9). Read existing fixture files first to check column names and adapt accordingly.

- [ ] **Step 2.6: Run model tests and commit**

Run: `go test ./models/hackforger/... -tags="bindata sqlite sqlite_unlock_notify" -v -count=1 2>&1 | tail -10`
Expected: all existing tests pass with enriched fixtures.

```bash
git add models/fixtures/hackathon*.yml
git commit -m "test: enrich hackathon fixtures — 6 statuses, tracks, registrations, submissions, scores"
```

### Task 3: Enrich Bounty fixtures (7 status values)

**Files:**
- Modify: `models/fixtures/bounty.yml`
- Modify: `models/fixtures/bounty_reward.yml`
- Modify: `models/fixtures/bounty_application.yml`
- Modify: `models/fixtures/bounty_winner.yml`

- [ ] **Step 3.1: Read current bounty fixtures**

Read all bounty-related fixture files.

- [ ] **Step 3.2: Write bounty.yml with 7 records (one per BountyStatus)**

Include both Exclusive (mode=0) and Competitive (mode=1) bounties. Use repo_id=1 and publisher_id=2 (organizer). Each bounty needs a valid issue_id — check `models/fixtures/issue.yml` for available IDs.

```yaml
# 7 records: Open(0), Claimed(1), InReview(2), Completed(3), Paid(4), Expired(5), Cancelled(6)
# Bounties 1-2 are Exclusive, 3-7 cover remaining statuses including one Competitive
```

- [ ] **Step 3.3: Write bounty_reward.yml with tiered rewards**

At least one bounty with single reward, one with 3 tiered rewards (1st/2nd/3rd for Competitive mode).

- [ ] **Step 3.4: Write bounty_application.yml (3 statuses)**

Pending(0), Accepted(1), Rejected(2).

- [ ] **Step 3.5: Write bounty_winner.yml**

At least 2 winners linked to the Competitive bounty.

- [ ] **Step 3.6: Run model tests and commit**

Run: `go test ./models/hackforger/... -tags="bindata sqlite sqlite_unlock_notify" -v -count=1 2>&1 | tail -10`

```bash
git add models/fixtures/bounty*.yml
git commit -m "test: enrich bounty fixtures — 7 statuses, tiered rewards, applications, winners"
```

### Task 4: Enrich Grant fixtures (6 status values)

**Files:**
- Modify: `models/fixtures/grant_round.yml`
- Modify: `models/fixtures/grant_project.yml`

- [ ] **Step 4.1: Read current grant fixtures**

- [ ] **Step 4.2: Write grant_round.yml with 6 records**

```yaml
# 6 records: Draft(0), Open(1), Review(2), Finalized(3), Distributed(4), Cancelled(5)
# All use org_id=3, owner_id=2 (organizer)
```

- [ ] **Step 4.3: Write grant_project.yml with 4 statuses**

Pending, Approved, Funded, Rejected. Link to Open and Finalized rounds.

- [ ] **Step 4.4: Run model tests and commit**

```bash
git add models/fixtures/grant_*.yml
git commit -m "test: enrich grant fixtures — 6 round statuses, 4 project statuses"
```

### Task 5: Enrich Credits + Reputation + Feed fixtures

**Files:**
- Modify: `models/fixtures/credit_account.yml`
- Modify: `models/fixtures/credit_transaction.yml`
- Modify: `models/fixtures/redeem_option.yml`
- Modify: `models/fixtures/redeem_option_key.yml`
- Modify: `models/fixtures/redeem_order.yml`
- Modify: `models/fixtures/reputation.yml`
- Modify: `models/fixtures/hackforger_action.yml` (may need to create if table name differs)
- Modify: `models/fixtures/follow.yml`

- [ ] **Step 5.1: Read current credits/reputation fixtures**

- [ ] **Step 5.2: Write credit_account.yml**

Accounts for organizer (id=2), hacker1 (id=4), hacker2 (id=5), platform-bot. Balances must reconcile with transactions.

- [ ] **Step 5.3: Write credit_transaction.yml (all 5 types)**

deposit, withdraw, redeem, admin_deposit, admin_deduct. At least 6 records covering all types.

- [ ] **Step 5.4: Write redeem fixtures**

redeem_option.yml (2+ options with stock), redeem_option_key.yml (3+ keys), redeem_order.yml (pending, fulfilled, cancelled).

- [ ] **Step 5.5: Write reputation.yml**

Reputation scores for hacker1, hacker2, organizer.

- [ ] **Step 5.6: Write hackforger_action fixture (Feed events)**

Check actual table name first (`hackforger_action` vs another). At least 10 records covering various ActionTypes (30-59) for Feed query tests.

- [ ] **Step 5.7: Add Follow records**

Add to `models/fixtures/follow.yml`: hacker1→organizer, hacker1→hacker2, hacker2→hacker1.

- [ ] **Step 5.8: Verify cross-journey Credits consistency**

Manually verify: sum(deposits) - sum(withdrawals) = account balances for each user.

- [ ] **Step 5.9: Run ALL model tests and commit**

Run: `go test ./models/hackforger/... -tags="bindata sqlite sqlite_unlock_notify" -v -count=1`

```bash
git add models/fixtures/
git commit -m "test: enrich credits, reputation, feed, follow fixtures — full status coverage"
```

---

## Chunk 2: API Renames via TDD

### Task 6: Rename start-judging → judge (TDD)

**Files:**
- Modify: `routers/api/v1/api.go:1774`
- Modify: `routers/api/v1/hackforger/hackathon.go` (handler function name)
- Modify: `routers/web/web.go` (web route if exists)
- Modify: `routers/web/hackforger/hackathon.go` (web handler name if exists)
- Test: `tests/integration/hackforger_hackathon_test.go` (new file, first test)

- [ ] **Step 6.1: Write the failing test**

Create `tests/integration/hackforger_hackathon_test.go`:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"
)

func TestHackForgerHackathonJudgeEndpoint(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 2 is org owner = hackathon organizer
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Hackathon 3 is in Hacking status — can transition to Judging via /judge
	req := NewRequest(t, "POST", "/api/v1/hackforger/hackathons/3/judge").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)
}
```

- [ ] **Step 6.2: Run test to verify it fails**

Run: `go test ./tests/integration/ -run TestHackForgerHackathonJudgeEndpoint -tags="bindata sqlite sqlite_unlock_notify" -v -count=1 2>&1 | tail -10`
Expected: 404 (route not found)

- [ ] **Step 6.3: Rename the route**

In `routers/api/v1/api.go:1774`, change:
```go
// OLD:
m.Post("/start-judging", reqToken(), hackforger_api.StartJudgingHackathon)
// NEW:
m.Post("/judge", reqToken(), hackforger_api.StartJudgingHackathon)
```

Also rename in web routes if the pattern exists. Search `routers/web/web.go` for `start-judging`.

- [ ] **Step 6.4: Run test to verify it passes**

Run: `go test ./tests/integration/ -run TestHackForgerHackathonJudgeEndpoint -tags="bindata sqlite sqlite_unlock_notify" -v -count=1`
Expected: PASS

- [ ] **Step 6.5: Verify old path returns 404**

Add a second test to confirm old path is gone:

```go
func TestHackForgerHackathonStartJudgingRemoved(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "POST", "/api/v1/hackforger/hackathons/3/start-judging").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}
```

- [ ] **Step 6.6: Commit**

```bash
git add routers/api/v1/api.go routers/web/ tests/integration/hackforger_hackathon_test.go
git commit -m "refactor(api): rename /start-judging → /judge via TDD"
```

### Task 7: Rename reject-delivery → reject (TDD)

**Files:**
- Modify: `routers/api/v1/api.go:1553`
- Test: `tests/integration/hackforger_bounty_test.go` (new file)

- [ ] **Step 7.1: Write the failing test**

Create `tests/integration/hackforger_bounty_test.go` with a test that calls `POST /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/reject`. The bounty must be in InReview(2) status for this to work.

- [ ] **Step 7.2: Run test — expect 404**

- [ ] **Step 7.3: Rename route in api.go:1553**

```go
// OLD:
m.Post("/reject-delivery", reqToken(), hackforger_api.RejectDeliveryAPI)
// NEW:
m.Post("/reject", reqToken(), hackforger_api.RejectDeliveryAPI)
```

Also rename in web routes.

- [ ] **Step 7.4: Run test — expect PASS**

- [ ] **Step 7.5: Commit**

```bash
git add routers/api/v1/api.go tests/integration/hackforger_bounty_test.go
git commit -m "refactor(api): rename /reject-delivery → /reject via TDD"
```

### Task 8: Rename finalize-confirm → finalize (TDD)

**Files:**
- Modify: `routers/api/v1/api.go:1802`
- Test: add to `tests/integration/hackforger_hackathon_test.go`

- [ ] **Step 8.1: Analyze existing handler code**

Read `routers/api/v1/hackforger/hackathon.go` — find functions `FinalizeHackathon` (line 1775's handler) and `FinalizeConfirmAPI` (line 1802's handler). Determine:
- Are they the same operation or different? (`FinalizeHackathon` may be Phase 1 simple finalize; `FinalizeConfirmAPI` may be Phase 2 judge-based finalize with preview)
- If they serve different purposes, `/finalize` (1775) should be removed and `/finalize-confirm` (1802) renamed to `/finalize`
- If they're redundant, remove one

- [ ] **Step 8.2: Write the failing test**

Write a test that calls `POST /api/v1/hackforger/hackathons/{id}/finalize` using the Phase 2 judge-based finalize logic (from `FinalizeConfirmAPI`). Also write a test that confirms `/finalize-confirm` returns 404 after rename.

- [ ] **Step 8.3: Implement rename based on handler analysis**

In `api.go`:
- Remove the old `/finalize` route (line 1775) if it's the Phase 1 version
- Rename `/finalize-confirm` (line 1802) to `/finalize`
- Update handler function name: `FinalizeConfirmAPI` → `FinalizeAPI`
- Apply same rename in `routers/web/web.go` and `routers/web/hackforger/hackathon.go`

- [ ] **Step 8.3: Run test — expect PASS**

- [ ] **Step 8.4: Commit**

```bash
git commit -m "refactor(api): rename /finalize-confirm → /finalize via TDD"
```

### Task 9: Rename batch-fulfill → fulfill with array body (TDD)

**Files:**
- Modify: `routers/api/v1/api.go:1854`
- Modify: `routers/api/v1/hackforger/credits.go` (handler to accept array body)
- Test: add to `tests/integration/hackforger_credits_test.go`

- [ ] **Step 9.1: Write the failing test**

> **Note:** Verify the exact route path by reading `api.go:1840-1860` for the credits group nesting before writing the test. The path below assumes `/api/v1/hackforger/credits/redeem/orders/fulfill`.

```go
func TestHackForgerCreditsFulfillArrayBody(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	session := loginUser(t, admin.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Call new /fulfill endpoint with array of order IDs
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/redeem/orders/fulfill",
		map[string]any{"order_ids": []int64{1, 2}},
	).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)
}
```

- [ ] **Step 9.2: Run test — expect 404**

- [ ] **Step 9.3: Rename route and modify handler**

In `api.go:1854`:
```go
// OLD:
m.Post("/orders/batch-fulfill", reqToken(), reqSiteAdmin(), bind(hackforger_api.BatchFulfillForm{}), hackforger_api.BatchFulfill)
// NEW:
m.Post("/orders/fulfill", reqToken(), reqSiteAdmin(), bind(hackforger_api.FulfillForm{}), hackforger_api.FulfillOrders)
```

Update `credits.go` handler to accept `{"order_ids": [1,2]}` body (array). If single order, also accept `{"order_ids": [1]}`.

- [ ] **Step 9.4: Run test — expect PASS**

- [ ] **Step 9.5: Commit**

```bash
git commit -m "refactor(api): rename /batch-fulfill → /fulfill with array body via TDD"
```

---

## Chunk 3: Integration Test Skeleton

### Task 10: Create test helper file

**Files:**
- Create: `tests/integration/hackforger_test_helper.go`

- [ ] **Step 10.1: Write shared helper functions**

Follow the pattern from `hackforger_credits_test.go` (loginUser, getTokenForLoggedInUser, NewRequest, etc. are already available from Forgejo's test framework).

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	auth_model "forgejo.org/models/auth"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hackforgerLoginAs logs in the user by ID and returns session + token.
func hackforgerLoginAs(t *testing.T, userID int64) (*TestSession, string) {
	t.Helper()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)
	return session, token
}

// hackforgerAssertFeedEvent checks that a hackforger_action record exists
// for the given ActionType and entity.
func hackforgerAssertFeedEvent(t *testing.T, opType int, entityID int64) {
	t.Helper()
	actions := make([]*hackforger_model.HackforgerAction, 0)
	err := unittest.GetXORMEngine(t).Where("op_type = ? AND entity_id = ?", opType, entityID).Find(&actions)
	require.NoError(t, err)
	assert.NotEmpty(t, actions, "expected feed event op_type=%d entity_id=%d", opType, entityID)
}

// hackforgerGet is a shorthand for GET requests to /api/v1/hackforger/* paths.
// Note: MakeRequest returns *httptest.ResponseRecorder in Forgejo's test framework.
func hackforgerGet(t *testing.T, token, path string, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/hackforger/%s", path)).AddTokenAuth(token)
	return MakeRequest(t, req, expectedStatus)
}

// hackforgerPost is a shorthand for POST requests with JSON body.
func hackforgerPost(t *testing.T, token, path string, body any, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/hackforger/%s", path), body).AddTokenAuth(token)
	return MakeRequest(t, req, expectedStatus)
}
```

- [ ] **Step 10.2: Verify it compiles**

Run: `go build -tags="bindata sqlite sqlite_unlock_notify" ./tests/integration/ 2>&1 | tail -5`

Note: `httpResponseData` type name may differ — check Forgejo's test framework. The actual type returned by `MakeRequest` is `*httptest.ResponseRecorder`. Adjust accordingly.

- [ ] **Step 10.3: Commit**

```bash
git add tests/integration/hackforger_test_helper.go
git commit -m "test: add HackForger integration test helper functions"
```

### Task 11: Create empty test file stubs

**Files:**
- Create: `tests/integration/hackforger_feed_test.go`
- Create: `tests/integration/hackforger_reputation_test.go`
- Create: `tests/integration/hackforger_search_test.go`
- Create: `tests/integration/hackforger_assistant_test.go`
- Create: `tests/integration/hackforger_webhook_test.go`

Note: `hackforger_hackathon_test.go` and `hackforger_bounty_test.go` were already created in Tasks 6-7. `hackforger_credits_test.go` and `hackforger_grant_test.go` already exist.

- [ ] **Step 11.1: Create stub files with package declaration**

Each file:
```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration
```

- [ ] **Step 11.2: Commit**

```bash
git add tests/integration/hackforger_*.go
git commit -m "test: add integration test file stubs for feed, reputation, search, assistant, webhook"
```
