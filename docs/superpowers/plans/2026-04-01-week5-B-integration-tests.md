# Week 5 Sub-Plan B: Integration Tests + Feature Implementation

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Write integration tests for all HackForger API endpoints. For new features (search, assistant, webhook), write tests first (TDD), then implement until tests pass.

**Architecture:** Each test file maps to a user journey. Tests use Forgejo's integration test framework (real HTTP server + SQLite + fixtures). New features driven by failing tests.

**Tech Stack:** Go integration tests, `tests.PrepareTestEnv(t)`, `MakeRequest`, XORM.

**Spec:** `docs/superpowers/specs/2026-04-01-week5-phase5-design.md` (Section 4)

**Depends on:** Sub-Plan A (fixtures enriched, renames applied, test skeleton created)

---

## Chunk 1: Existing Module Tests (B1-B5)

These test existing functionality. Tests should pass immediately after writing (no implementation needed).

> **Important:** Each test function must use `defer tests.PrepareTestEnv(t)()` for fresh fixture state. Do NOT chain state-changing operations across separate Test functions — each function gets its own clean database snapshot from fixtures.

### Task 1: Hackathon integration tests (B1)

**Files:**
- Modify: `tests/integration/hackforger_hackathon_test.go` (started in Plan A Task 6)

Reference: `routers/api/v1/api.go:1764-1802` for all hackathon routes.

- [ ] **Step 1.1: Write CRUD tests**

```go
func TestHackForgerHackathonCreate(t *testing.T) { ... }
func TestHackForgerHackathonGet(t *testing.T) { ... }
func TestHackForgerHackathonList(t *testing.T) { ... }
func TestHackForgerHackathonUpdate(t *testing.T) { ... }
func TestHackForgerHackathonDelete(t *testing.T) { ... }
```

Pattern for each: `hackforgerLoginAs(t, 2)` → call API → assert response. Create uses `POST /api/v1/hackforger/hackathons` with `CreateHackathonForm`. Get uses fixture hackathon id=1.

- [ ] **Step 1.2: Run CRUD tests**

Run: `go test ./tests/integration/ -run "TestHackForgerHackathon(Create|Get|List|Update|Delete)" -tags="bindata sqlite sqlite_unlock_notify" -v -count=1`
Expected: PASS

- [ ] **Step 1.3: Write status flow tests**

```go
func TestHackForgerHackathonPublish(t *testing.T) {
	// Hackathon 1 (Draft) → POST /publish → status becomes Open
}
func TestHackForgerHackathonStart(t *testing.T) {
	// Hackathon 2 (Open) → POST /start → status becomes Hacking
}
func TestHackForgerHackathonJudge(t *testing.T) {
	// Already tested in Plan A Task 6, verify still passes
}
func TestHackForgerHackathonFinalize(t *testing.T) {
	// Hackathon 4 (Judging) → POST /finalize (or /finalize-confirm renamed)
}
func TestHackForgerHackathonCancel(t *testing.T) {
	// Hackathon 1 (Draft) → POST /cancel → status becomes Cancelled
}
```

- [ ] **Step 1.4: Run status flow tests — expect PASS**

- [ ] **Step 1.5: Write Track CRUD + Criteria tests**

```go
func TestHackForgerTrackCreate(t *testing.T) { ... }
func TestHackForgerTrackList(t *testing.T) { ... }
func TestHackForgerTrackUpdate(t *testing.T) { ... }
func TestHackForgerTrackDelete(t *testing.T) { ... }
func TestHackForgerCriteriaCreate(t *testing.T) { ... }
func TestHackForgerCriteriaList(t *testing.T) { ... }
```

- [ ] **Step 1.6: Run track/criteria tests — expect PASS**

- [ ] **Step 1.7: Write Registration + Submission + Judge tests**

```go
func TestHackForgerRegister(t *testing.T) { ... }
func TestHackForgerListRegistrations(t *testing.T) { ... }
func TestHackForgerUpdateRegistration(t *testing.T) { ... }
func TestHackForgerCreateSubmission(t *testing.T) { ... }
func TestHackForgerListSubmissions(t *testing.T) { ... }
func TestHackForgerAddJudge(t *testing.T) { ... }
func TestHackForgerSubmitScore(t *testing.T) { ... }
func TestHackForgerLeaderboard(t *testing.T) { ... }
```

- [ ] **Step 1.8: Run all hackathon tests — expect PASS**

- [ ] **Step 1.9: Write permission tests**

```go
func TestHackForgerHackathonPublishNonOwner(t *testing.T) {
	// hacker1 tries to publish organizer's hackathon → 403
}
func TestHackForgerScoreNonJudge(t *testing.T) {
	// hacker1 tries to score → 403
}
```

- [ ] **Step 1.10: Write Feed event verification**

After each state change test, add `hackforgerAssertFeedEvent(t, actionType, entityID)`.

- [ ] **Step 1.11: Run all and commit**

Run: `go test ./tests/integration/ -run TestHackForger -tags="bindata sqlite sqlite_unlock_notify" -v -count=1`

```bash
git add tests/integration/hackforger_hackathon_test.go
git commit -m "test: hackathon integration tests — CRUD, status flow, tracks, scoring, permissions"
```

### Task 2: Bounty integration tests (B2)

**Files:**
- Modify: `tests/integration/hackforger_bounty_test.go` (started in Plan A Task 7)

Reference: `routers/api/v1/api.go:1537-1558` (repo-level bounty routes), `1806-1808` (global bounty routes).

- [ ] **Step 2.1: Write Exclusive Bounty lifecycle tests**

```go
func TestHackForgerBountyExclusiveLifecycle(t *testing.T) {
	// Create → Apply → Accept → Complete → Pay
	// Verify status transitions at each step
}
```

- [ ] **Step 2.2: Write Competitive Bounty lifecycle tests**

```go
func TestHackForgerBountyCompetitiveLifecycle(t *testing.T) {
	// Create with mode=1 → Multiple applications → SelectWinners → Pay
}
```

- [ ] **Step 2.3: Write exception path tests**

```go
func TestHackForgerBountyExpire(t *testing.T) { ... }
func TestHackForgerBountyCancel(t *testing.T) { ... }
func TestHackForgerBountyRejectApplication(t *testing.T) { ... }
func TestHackForgerBountyRejectEndpoint(t *testing.T) {
	// Test renamed /reject endpoint (was /reject-delivery)
}
```

- [ ] **Step 2.4: Write Reward CRUD tests**

```go
func TestHackForgerRewardCRUD(t *testing.T) { ... }
```

- [ ] **Step 2.5: Write global bounty list + stats + leaderboard tests**

```go
func TestHackForgerBountyGlobalList(t *testing.T) { ... }
func TestHackForgerBountyStats(t *testing.T) { ... }
func TestHackForgerHunterLeaderboard(t *testing.T) { ... }
```

- [ ] **Step 2.6: Run all and commit**

```bash
git add tests/integration/hackforger_bounty_test.go
git commit -m "test: bounty integration tests — exclusive, competitive, exceptions, rewards"
```

### Task 3: Grant integration tests (B3)

**Files:**
- Modify: `tests/integration/hackforger_grant_test.go` (already exists)

Reference: `routers/api/v1/api.go:1813-1834`

- [ ] **Step 3.1: Audit existing tests**

Read current `hackforger_grant_test.go` to understand what's already covered. Add missing tests.

- [ ] **Step 3.2: Add status flow tests**

```go
func TestAPIGrantRoundOpen(t *testing.T) { ... }
func TestAPIGrantRoundClose(t *testing.T) { ... }
func TestAPIGrantRoundFinalize(t *testing.T) { ... }
func TestAPIGrantRoundDistribute(t *testing.T) {
	// Test the distribute endpoint (POST /grants/rounds/{id}/distribute)
	// Round must be in Finalized(3) status → transitions to Distributed(4)
}
func TestAPIGrantRoundCancel(t *testing.T) { ... }
```

- [ ] **Step 3.3: Add Project tests**

```go
func TestAPIGrantProjectSubmit(t *testing.T) { ... }
func TestAPIGrantProjectApprove(t *testing.T) { ... }
func TestAPIGrantProjectReject(t *testing.T) { ... }
func TestAPIGrantProjectAllocateAward(t *testing.T) { ... }
```

- [ ] **Step 3.4: Add budget constraint test**

```go
func TestAPIGrantBudgetExceeded(t *testing.T) {
	// Allocate award > remaining budget → error
}
```

- [ ] **Step 3.5: Run all and commit**

```bash
git add tests/integration/hackforger_grant_test.go
git commit -m "test: grant integration tests — status flow, distribute, projects, budget constraint"
```

### Task 4: Credits integration tests (B4)

**Files:**
- Modify: `tests/integration/hackforger_credits_test.go` (already exists)

Reference: `routers/api/v1/api.go:1841-1860`

- [ ] **Step 4.1: Audit existing tests**

Read current file — already has ~16 tests. Identify gaps.

- [ ] **Step 4.2: Add missing tests**

```go
func TestAPICreditsTransactionPagination(t *testing.T) { ... }
func TestAPICreditsRedeemOptionKeys(t *testing.T) {
	// Add keys → list keys → verify count
}
func TestAPICreditsRedeemOutOfStock(t *testing.T) { ... }
func TestAPICreditsAdminFulfillArrayBody(t *testing.T) {
	// Test renamed /fulfill endpoint with array body (was /batch-fulfill)
}
```

- [ ] **Step 4.3: Run all credits tests — expect PASS**

- [ ] **Step 4.4: Commit**

```bash
git add tests/integration/hackforger_credits_test.go
git commit -m "test: credits integration tests — pagination, keys, out-of-stock, array fulfill"
```

### Task 5: Feed + Reputation integration tests (B5)

**Files:**
- Modify: `tests/integration/hackforger_feed_test.go`
- Modify: `tests/integration/hackforger_reputation_test.go`

Reference: `routers/api/v1/api.go:1809` (feed), `1865-1868` (reputation)

- [ ] **Step 5.1: Write Feed tests**

```go
func TestHackForgerFeedGlobal(t *testing.T) {
	// GET /api/v1/hackforger/feed?type=global → returns events with user_id=0
}
func TestHackForgerFeedFollowing(t *testing.T) {
	// Login as hacker1 → GET /feed?type=following → returns events from followed users
}
func TestHackForgerFeedPagination(t *testing.T) {
	// GET /feed?page=1&limit=2 → returns 2 results + correct total
}
```

- [ ] **Step 5.2: Run feed tests — expect PASS**

- [ ] **Step 5.3: Write Reputation tests**

```go
func TestHackForgerReputationGet(t *testing.T) {
	// GET /reputation/users/{username} → returns reputation score
}
func TestHackForgerReputationLeaderboard(t *testing.T) {
	// GET /reputation/leaderboard → returns sorted list
}
func TestHackForgerReputationRecalculate(t *testing.T) {
	// POST /reputation/recalculate/{username} (admin only)
}
func TestHackForgerReputationRecalculateNonAdmin(t *testing.T) {
	// Non-admin → 403
}
```

- [ ] **Step 5.4: Run reputation tests — expect PASS**

- [ ] **Step 5.5: Commit**

```bash
git add tests/integration/hackforger_feed_test.go tests/integration/hackforger_reputation_test.go
git commit -m "test: feed + reputation integration tests — global, following, pagination, leaderboard"
```

---

## Chunk 2: TDD New Features (B6-B8)

These tests will FAIL initially. After writing tests, implement the feature to make them pass.

### Task 6: Search — test first, then implement (B6)

**Files:**
- Test: `tests/integration/hackforger_search_test.go`
- Create: `services/hackforger/search.go`
- Create: `routers/api/v1/hackforger/search.go`
- Create: `routers/web/hackforger/search.go`
- Modify: `routers/api/v1/api.go` (register search route)
- Modify: `routers/web/web.go` (register web search route)
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 6.1: Write failing search tests**

```go
func TestHackForgerSearchAll(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	_, token := hackforgerLoginAs(t, 4) // hacker1

	resp := hackforgerGet(t, token, "search?q=Hackathon&scope=all", http.StatusOK)
	var result struct {
		Results []map[string]any `json:"results"`
		Total   int64            `json:"total"`
	}
	DecodeJSON(t, resp, &result)
	assert.Greater(t, result.Total, int64(0))
}

func TestHackForgerSearchByScope(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	_, token := hackforgerLoginAs(t, 4)

	// Search only hackathons
	resp := hackforgerGet(t, token, "search?q=Hackathon&scope=hackathons", http.StatusOK)
	var result struct {
		Results []map[string]any `json:"results"`
		Total   int64            `json:"total"`
	}
	DecodeJSON(t, resp, &result)
	for _, r := range result.Results {
		assert.Equal(t, "hackathon", r["type"])
	}
}

func TestHackForgerSearchEmpty(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	_, token := hackforgerLoginAs(t, 4)

	resp := hackforgerGet(t, token, "search?q=xyznonexistent&scope=all", http.StatusOK)
	var result struct {
		Results []map[string]any `json:"results"`
		Total   int64            `json:"total"`
	}
	DecodeJSON(t, resp, &result)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.Results)
}

func TestHackForgerSearchPagination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	_, token := hackforgerLoginAs(t, 4)

	resp := hackforgerGet(t, token, "search?q=Hackathon&scope=all&page=1&limit=2", http.StatusOK)
	var result struct {
		Results []map[string]any `json:"results"`
		Total   int64            `json:"total"`
	}
	DecodeJSON(t, resp, &result)
	assert.LessOrEqual(t, len(result.Results), 2)
}

func TestHackForgerSearchNoAuth(t *testing.T) {
	// Search should work without auth (public data)
	defer tests.PrepareTestEnv(t)()
	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=Hackathon&scope=all")
	MakeRequest(t, req, http.StatusOK)
}
```

- [ ] **Step 6.2: Run search tests — expect FAIL (404)**

Run: `go test ./tests/integration/ -run TestHackForgerSearch -tags="bindata sqlite sqlite_unlock_notify" -v -count=1`

- [ ] **Step 6.3: Implement search service**

Create `services/hackforger/search.go`:
- `SearchOptions` struct with Keyword, Scope, ListOptions
- `SearchResult` struct with Type, ID, Title, Status, Slug, MatchField
- `Search()` function: XORM LIKE queries on hackathon.name/description, bounty.title, grant_round.name/description
- `scope=all` merges results from 3 tables sorted by created_unix DESC
- Respect pagination (Page, PageSize)

- [ ] **Step 6.4: Implement search API router**

Create `routers/api/v1/hackforger/search.go`:
- `SearchAPI` handler: parse query params → call `Search()` → JSON response
- Register route in `api.go`: `m.Get("/search", hackforger_api.SearchAPI)`

- [ ] **Step 6.5: Implement search web router**

Create `routers/web/hackforger/search.go`:
- `SearchWeb` handler: same logic, returns JSON (for modal fetch)
- Register in `web.go`: `m.Get("/hackforger/search", web_hackforger.SearchWeb)`

- [ ] **Step 6.6: Run search tests — expect PASS**

- [ ] **Step 6.7: Commit**

```bash
git add services/hackforger/search.go routers/api/v1/hackforger/search.go \
  routers/web/hackforger/search.go routers/api/v1/api.go routers/web/web.go \
  tests/integration/hackforger_search_test.go
git commit -m "feat: search service + API/web routes — TDD, cross-entity LIKE queries"
```

### Task 7: Assistant — test first, then implement (B7)

**Files:**
- Test: `tests/integration/hackforger_assistant_test.go`
- Create: `services/hackforger/assistant.go`
- Create: `routers/api/v1/hackforger/assistant.go`
- Modify: `routers/web/hackforger/search.go` (add assistant web route)
- Modify: `routers/api/v1/api.go`
- Modify: `routers/web/web.go`
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 7.1: Write failing assistant tests**

```go
func TestHackForgerAssistantChat(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	_, token := hackforgerLoginAs(t, 4)

	resp := hackforgerPost(t, token, "assistant/chat",
		map[string]string{"query": "What bounties are available?"},
		http.StatusOK)

	var result struct {
		Message    string `json:"message"`
		Status     string `json:"status"`
		Disclaimer string `json:"disclaimer"`
	}
	DecodeJSON(t, resp, &result)
	assert.Equal(t, "placeholder", result.Status)
	assert.NotEmpty(t, result.Message)
	assert.NotEmpty(t, result.Disclaimer)
}

func TestHackForgerAssistantChatNoAuth(t *testing.T) {
	// Assistant requires auth
	defer tests.PrepareTestEnv(t)()
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/assistant/chat",
		map[string]string{"query": "test"})
	MakeRequest(t, req, http.StatusUnauthorized)
}

func TestHackForgerAssistantChatEmptyQuery(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	_, token := hackforgerLoginAs(t, 4)

	resp := hackforgerPost(t, token, "assistant/chat",
		map[string]string{"query": ""},
		http.StatusOK) // still returns placeholder even for empty query

	var result struct {
		Status string `json:"status"`
	}
	DecodeJSON(t, resp, &result)
	assert.Equal(t, "placeholder", result.Status)
}
```

- [ ] **Step 7.2: Run assistant tests — expect FAIL (404)**

- [ ] **Step 7.3: Implement assistant service**

Create `services/hackforger/assistant.go`:
- `AssistantResponse` struct with Message, Status, Disclaimer
- `Chat()` function: returns i18n placeholder text via `locale.TrString()`

- [ ] **Step 7.4: Implement assistant API + web routers**

Create `routers/api/v1/hackforger/assistant.go`:
- `ChatAPI` handler: parse JSON body → call `Chat()` → JSON response
- Register: `m.Post("/assistant/chat", reqToken(), hackforger_api.ChatAPI)`

Add web route in `routers/web/hackforger/search.go` (or separate file):
- `ChatWeb` handler: same logic with session auth
- Register: `m.Post("/hackforger/assistant/chat", reqSignIn, web_hackforger.ChatWeb)`

- [ ] **Step 7.5: Add i18n keys**

In `options/locale/locale_en-US.ini` under `[hackforger]`:
```ini
assistant.coming_soon = AI assistant coming soon. Stay tuned!
assistant.disclaimer = AI responses may be inaccurate. Please double-check.
```

In `options/locale/locale_zh-CN.ini` under `[hackforger]`:
```ini
assistant.coming_soon = AI 助手功能即将上线，敬请期待！
assistant.disclaimer = AI 回答可能不准确，请自行核实。
```

- [ ] **Step 7.6: Run assistant tests — expect PASS**

- [ ] **Step 7.7: Commit**

```bash
git add services/hackforger/assistant.go routers/api/v1/hackforger/assistant.go \
  routers/web/hackforger/ routers/api/v1/api.go routers/web/web.go \
  options/locale/ tests/integration/hackforger_assistant_test.go
git commit -m "feat: assistant chat service — TDD, placeholder response via i18n"
```

### Task 8: Webhook dispatch — test first, then implement (B8)

**Files:**
- Test: `tests/integration/hackforger_webhook_test.go`
- Modify: `modules/webhook/type.go` (add HackForger event types)
- Modify: `models/hackforger/action_types.go` (add ActionType → HookEventType mapping)
- Create: `modules/structs/hackforger_webhook.go` (payload struct)
- Modify: `services/hackforger/notifier.go` (add webhook dispatch to PublishHackforgerAction)
- Modify: `templates/webhook/shared-settings.tmpl` (add HackForger event checkboxes)

- [ ] **Step 8.1: Write failing webhook tests**

```go
func TestHackForgerWebhookBountyCreated(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Create a system webhook that listens for bounty_created events
	// Then create a bounty via API
	// Then verify hook_task table has a delivery record

	_, token := hackforgerLoginAs(t, 1) // admin

	// Step 1: Create system webhook (using Forgejo's admin API)
	// NOTE: Read `routers/api/v1/admin/hooks.go` and `modules/structs/hook.go`
	// to determine the correct webhook creation payload format. Forgejo uses
	// an "events" object with named boolean fields, NOT a string array.
	// The code below is a placeholder — adjust the event field structure
	// to match what Forgejo's CreateHookOption expects.
	webhookReq := NewRequestWithJSON(t, "POST", "/api/v1/admin/hooks", map[string]any{
		"type": "forgejo",
		"config": map[string]string{
			"url":          "http://localhost:1234/webhook",
			"content_type": "json",
		},
		"events": []string{"bounty_created"}, // TODO: verify format from CreateHookOption struct
		"active": true,
	}).AddTokenAuth(token)
	MakeRequest(t, webhookReq, http.StatusCreated)

	// Step 2: Create a bounty (triggers PublishHackforgerAction → webhook dispatch)
	_, orgToken := hackforgerLoginAs(t, 2) // organizer
	bountyReq := NewRequestWithJSON(t, "POST",
		"/api/v1/repos/user2/repo1/hackforger/bounties",
		map[string]any{
			"issue_id": 1,
			"title":    "Webhook test bounty",
			"mode":     0,
		}).AddTokenAuth(orgToken)
	MakeRequest(t, bountyReq, http.StatusCreated)

	// Step 3: Verify hook_task was created
	// Check hook_task table for event_type = "bounty_created"
	// This will fail until webhook dispatch is implemented
}

func TestHackForgerWebhookPayloadStructure(t *testing.T) {
	// Verify payload JSON structure matches HackforgerWebhookPayload
}
```

- [ ] **Step 8.2: Run webhook tests — expect FAIL**

The test should fail because `bounty_created` is not a recognized webhook event type yet.

- [ ] **Step 8.3: Register HackForger event types**

In `modules/webhook/type.go`, add constants:

```go
// HackForger events
HookEventHackathonCreated       HookEventType = "hackathon_created"
HookEventHackathonStatusChanged HookEventType = "hackathon_status_changed"
HookEventHackathonSubmission    HookEventType = "hackathon_submission"
HookEventHackathonScored        HookEventType = "hackathon_scored"
HookEventHackathonFinalized     HookEventType = "hackathon_finalized"
HookEventBountyCreated          HookEventType = "bounty_created"
HookEventBountyApplication      HookEventType = "bounty_application"
HookEventBountyClaimed          HookEventType = "bounty_claimed"
HookEventBountyCompleted        HookEventType = "bounty_completed"
HookEventBountyPaid             HookEventType = "bounty_paid"
HookEventBountyWinners          HookEventType = "bounty_winners"
HookEventBountyExpired          HookEventType = "bounty_expired"
HookEventBountyCancelled        HookEventType = "bounty_cancelled"
HookEventGrantRoundCreated      HookEventType = "grant_round_created"
HookEventGrantRoundOpened       HookEventType = "grant_round_opened"
HookEventGrantProjectSubmitted  HookEventType = "grant_project_submitted"
HookEventGrantAwarded           HookEventType = "grant_awarded"
HookEventGrantRoundFinalized    HookEventType = "grant_round_finalized"
HookEventCreditsDeposited       HookEventType = "credits_deposited"
HookEventCreditsRedeemed        HookEventType = "credits_redeemed"
```

Update `Event()` switch to return the correct string for each.

- [ ] **Step 8.4: Add ActionType → HookEventType mapping**

In `models/hackforger/action_types.go`:

```go
import webhook_module "forgejo.org/modules/webhook"

var ActionTypeToHookEvent = map[activities_model.ActionType]webhook_module.HookEventType{
	ActionHackathonCreated:      webhook_module.HookEventHackathonCreated,
	ActionHackathonPhaseChanged: webhook_module.HookEventHackathonStatusChanged,
	ActionHackathonSubmitted:    webhook_module.HookEventHackathonSubmission,
	ActionHackathonScored:       webhook_module.HookEventHackathonScored,
	ActionHackathonFinalized:    webhook_module.HookEventHackathonFinalized,
	ActionBountyCreated:         webhook_module.HookEventBountyCreated,
	ActionBountyClaimed:         webhook_module.HookEventBountyClaimed,
	ActionBountyDelivered:       webhook_module.HookEventBountyApplication,
	ActionBountyCompleted:       webhook_module.HookEventBountyCompleted,
	ActionBountyWinnersSelected: webhook_module.HookEventBountyWinners,
	ActionBountyPaid:            webhook_module.HookEventBountyPaid,
	ActionBountyExpired:         webhook_module.HookEventBountyExpired,
	ActionBountyCancelled:       webhook_module.HookEventBountyCancelled,
	ActionGrantRoundCreated:     webhook_module.HookEventGrantRoundCreated,
	ActionGrantProjectSubmitted: webhook_module.HookEventGrantProjectSubmitted,
	ActionGrantAwarded:          webhook_module.HookEventGrantAwarded,
	ActionGrantRoundOpened:      webhook_module.HookEventGrantRoundOpened,
	ActionGrantRoundFinalized:   webhook_module.HookEventGrantRoundFinalized,
	ActionCreditsRedeemed:       webhook_module.HookEventCreditsRedeemed,
}
```

- [ ] **Step 8.5: Create webhook payload struct**

Create `modules/structs/hackforger_webhook.go`:

```go
package structs

import (
	"encoding/json"
	"fmt"
)

// HackforgerWebhookPayload represents a HackForger webhook event payload.
type HackforgerWebhookPayload struct {
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   int64          `json:"entity_id"`
	EntityName string         `json:"entity_name"`
	Sender     *User          `json:"sender"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// JSONPayload implements api.Payloader.
func (p *HackforgerWebhookPayload) JSONPayload() ([]byte, error) {
	// Use standard json marshal
	return json.Marshal(p)
}
```

Check what interface `api.Payloader` requires — read `modules/structs/hook.go` for the exact method signature.

- [ ] **Step 8.6: Add webhook dispatch to PublishHackforgerAction**

In `services/hackforger/notifier.go`, at the end of `PublishHackforgerAction()`:

```go
// Webhook dispatch — after Feed insert
if hookEvent, ok := hackforger_model.ActionTypeToHookEvent[opts.OpType]; ok {
	actUser, err := user_model.GetUserByID(ctx, opts.ActUserID)
	if err == nil {
		payload := &structs.HackforgerWebhookPayload{
			Action:     string(hookEvent),
			EntityType: opts.EntityType,
			EntityID:   opts.EntityID,
			EntityName: opts.EntityName,
			Sender:     convert.ToUser(ctx, actUser, nil),
		}
		source := webhook_service.EventSource{Owner: actUser}
		if opts.RepoID > 0 {
			if repo, err := repo_model.GetRepositoryByID(ctx, opts.RepoID); err == nil {
				source.Repository = repo
			}
		}
		if err := webhook_service.PrepareWebhooks(ctx, source, hookEvent, payload); err != nil {
			log.Error("PrepareWebhooks for HackForger event %s: %v", hookEvent, err)
		}
	}
}
```

Add necessary imports.

- [ ] **Step 8.7: Update webhook settings template**

In `templates/webhook/shared-settings.tmpl`, add a HackForger Events section with checkboxes for each event type. Follow the existing pattern for Forgejo events in that file.

- [ ] **Step 8.8: Run webhook tests — expect PASS**

- [ ] **Step 8.9: Run ALL integration tests to verify nothing is broken**

Run: `go test ./tests/integration/ -run TestHackForger -tags="bindata sqlite sqlite_unlock_notify" -v -count=1`

- [ ] **Step 8.10: Commit**

```bash
git add modules/webhook/type.go models/hackforger/action_types.go \
  modules/structs/hackforger_webhook.go services/hackforger/notifier.go \
  templates/webhook/shared-settings.tmpl \
  tests/integration/hackforger_webhook_test.go
git commit -m "feat: webhook dispatch for HackForger events — TDD, single integration point"
```
