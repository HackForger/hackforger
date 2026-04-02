# Week 5 Design Spec: AI Assistant + Testing + CLI + Finalization

> **Status:** Draft
> **Date:** 2026-04-01
> **Branch:** v0.1-dev/hackforger
> **Prerequisites:** PR #11 (PRD + User Journeys), PR #12 (UI Modernization), Phase 1-4 complete

---

## 1. Overview

Week 5 completes HackForger v0.1 with five workstreams:

1. **Test fixtures + API renames** (TDD foundation)
2. **Integration tests** (full API coverage)
3. **Search + AI Assistant modal** (unified ⌘K interface)
4. **Webhook dispatch** (HackForger events into Forgejo webhook system)
5. **hackforger-cli + Swagger + Claude Code skill**

All new features are developed TDD: write integration test first, then implement until tests pass.

---

## 2. Test Fixtures

### 2.1 Design Principle

Fixtures are derived from the 6 user journeys (73 steps) in `docs/product/user-journeys/`. Every entity status value must appear at least once. Cross-journey Credits ledger must be consistent (organizer final balance = 1400).

### 2.2 Role Mapping

Aligned with PRD (PR #11) — `publisher` merged into `organizer`, `agent-bot` → `platform-bot`:

| Fixture User | Role | Notes |
|-------------|------|-------|
| `admin` (ID=1) | Admin | Platform management, Credits admin |
| `organizer` | Normal | Creates Hackathon/Bounty/Grant (was publisher) |
| `hacker1` | Normal | Active participant |
| `hacker2` | Normal | Second participant, teammate |
| `judge1` | Normal | Hackathon judge |
| `judge2` | Normal | Second judge |
| `platform-bot` | Bot(4) | Admin-created automation bot |

### 2.3 Required Fixture Records

| Entity | Records | Status Coverage |
|--------|---------|----------------|
| Hackathon | 6 | Draft(0), Open(1), Hacking(2), Judging(3), Finished(4), Cancelled(5) — one record per status |
| HackathonTrack | 3+ | Linked to hackathons above |
| HackathonTrackCriteria | 4+ | 2+ criteria per track |
| HackathonRegistration | 4 | Pending(0), Approved(1), Rejected(2) |
| HackathonSubmission | 3+ | From hacker1, hacker2, platform-bot |
| HackathonJudge | 2 | judge1, judge2 |
| HackathonJudgeScore | 4+ | judge1x2 + judge2x2 |
| Bounty | 7 | Open(0), Claimed(1), InReview(2), Completed(3), Paid(4), Expired(5), Cancelled(6) — one per status |
| BountyReward | 4+ | Single + tiered (1st/2nd/3rd) |
| BountyApplication | 3 | Pending(0), Accepted(1), Rejected(2) |
| BountyWinner | 2+ | Linked to Competitive bounty |
| GrantRound | 6 | Draft(0), Open(1), Review(2), Finalized(3), Distributed(4), Cancelled(5) — one per status |
| GrantProject | 4 | Pending, Approved, Funded, Rejected |
| CreditAccount | 4+ | organizer, hacker1, hacker2, platform-bot |
| CreditTransaction | 6+ | deposit, withdraw, redeem, admin_deposit, admin_deduct |
| RedeemOption | 2+ | Active options with stock |
| RedeemOptionKey | 3+ | Key pool entries |
| RedeemOrder | 3 | pending, fulfilled, cancelled |
| Reputation | 3+ | hacker1, hacker2, organizer |
| HackforgerAction | 10+ | Various event types for Feed tests |
| Follow | 3+ | hacker1→organizer, hacker1↔hacker2 |

### 2.4 Cross-Journey Consistency

Credits transactions must reconcile:
- Hackathon prize → deposit to winners
- Bounty completion → deposit to claimer/winners
- Grant distribution → deposit to project owners
- Redeem → withdraw from hacker1
- Final balances match user journey expectations

---

## 3. API Renames (TDD)

Per `docs/product/api-rename-plan.md`, applied via TDD: write test with new path → 404 → rename → pass.

| # | Old Path | New Path | Scope |
|---|----------|----------|-------|
| 1 | `POST /hackathons/{id}/start-judging` | `POST /hackathons/{id}/judge` | API + Web |
| 2 | `POST /bounties/{id}/reject-delivery` | `POST /bounties/{id}/reject` | API + Web |
| 3 | `POST /hackathons/{id}/finalize-confirm` | `POST /hackathons/{id}/finalize` | API + Web |
| 4 | `POST /credits/orders/batch-fulfill` | `POST /credits/orders/fulfill` (array body) | API + Web |

> **Note:** `POST /grants/rounds/{id}/distribute` already exists (implemented in Phase 2). It needs integration test coverage but is not a rename.

Each rename updates: route registration (`api.go` + `web.go`), handler function name, Swagger annotation, i18n keys (if any).

---

## 4. Integration Tests

### 4.1 Architecture

Follow Forgejo's `tests/integration/` pattern: full HTTP test server, real requests, fixture-backed SQLite.

```
tests/integration/
├── hackforger_hackathon_test.go     — Journey 1 (~20 tests)
├── hackforger_bounty_test.go        — Journey 2 (~18 tests)
├── hackforger_grant_test.go         — Journey 3 (~14 tests)
├── hackforger_credits_test.go       — Journey 4 (~12 tests)
├── hackforger_feed_test.go          — Journey 5 (~6 tests)
├── hackforger_reputation_test.go    — (~4 tests)
├── hackforger_search_test.go        — Search (TDD, ~5 tests)
├── hackforger_webhook_test.go       — Webhook dispatch (~4 tests)
├── hackforger_assistant_test.go     — Assistant chat (~3 tests)
└── hackforger_test_helper.go        — Shared test utilities
```

### 4.2 Test Helper

```go
// hackforger_test_helper.go
func createHackathonAPI(t, session, opts) *api.Hackathon
func createBountyAPI(t, session, owner, repo, opts) *api.Bounty
func createGrantRoundAPI(t, session, opts) *api.GrantRound
func loginAs(t, username) *TestSession
func assertFeedEvent(t, actionType, entityID)
func assertWebhookDelivered(t, hookEventType, entityID)
```

### 4.3 Coverage Per File

**hackforger_hackathon_test.go (Journey 1):**
- CRUD: Create / Get / List / Update / Delete
- Status flow: Draft → Publish → Start → Judge (renamed) → Finalize
- Track CRUD + Criteria
- Registration: Create / Approve / Reject
- Submission: Create / List / Update
- Judge: Add / Score / Leaderboard
- Permission: non-owner cannot publish, non-judge cannot score
- Feed: verify hackforger_action records after each state change

**hackforger_bounty_test.go (Journey 2):**
- Exclusive: Create → Apply → Accept → Complete → Pay
- Competitive: Create → Multiple applicants → SelectWinners → Pay
- Exceptions: Expire / Cancel / Reject application
- Reward CRUD
- Renamed endpoint: `/reject` replaces `/reject-delivery`

**hackforger_grant_test.go (Journey 3):**
- Round CRUD + status flow: Draft → Open → Review → Finalize → Distribute (existing)
- Project: Submit / Approve / Reject / Award allocation
- Budget constraint: total > budget → error
- Cancel path
- New endpoint: `POST /grants/rounds/{id}/distribute`

**hackforger_credits_test.go (Journey 4):**
- Balance query, Transaction list + pagination
- RedeemOption CRUD, RedeemOptionKey pool
- Redeem: place order → balance deduction (atomic)
- Admin: Deposit / Deduct / Fulfill (new array body)
- Errors: insufficient balance, out of stock

**hackforger_feed_test.go (Journey 5):**
- Global / Following / Entity / User feed queries
- Pagination
- Event type filtering

**hackforger_reputation_test.go:**
- User reputation query, Leaderboard, Admin recalculate

**hackforger_search_test.go (TDD — test first, then implement):**
- `GET /hackforger/search?q=test&scope=all` → mixed results
- `scope=hackathons` / `scope=bounties` / `scope=grants`
- Empty results, Pagination

**hackforger_assistant_test.go (TDD):**
- `POST /hackforger/assistant/chat` with `{"query": "test"}` → returns placeholder response
- Response has `status: "placeholder"`, non-empty `message`, `disclaimer`

**hackforger_webhook_test.go (TDD):**
- Create webhook → trigger bounty_created → verify hook_task record
- Verify payload structure matches `HackforgerWebhookPayload`
- Multiple event types dispatch correctly

### 4.4 Execution

```bash
# All HackForger integration tests
go test ./tests/integration/ -run TestHackForger -tags="bindata sqlite sqlite_unlock_notify" -v

# Single module
go test ./tests/integration/ -run TestHackForgerBounty -tags="bindata sqlite sqlite_unlock_notify" -v

# Full suite (must not break upstream)
make test-sqlite
```

---

## 5. Search + AI Assistant

### 5.1 Search Backend

**`services/hackforger/search.go`:**

```go
type SearchResult struct {
    Type       string `json:"type"`        // "hackathon" | "bounty" | "grant"
    ID         int64  `json:"id"`
    Title      string `json:"title"`
    Status     string `json:"status"`
    Slug       string `json:"slug"`
    MatchField string `json:"match_field"` // which field matched
}

type SearchOptions struct {
    Keyword string
    Scope   string // "all" | "hackathons" | "bounties" | "grants"
    db.ListOptions
}

func Search(ctx context.Context, opts *SearchOptions) ([]*SearchResult, int64, error)
```

Implementation: XORM `LIKE` queries against hackathon.name/description, bounty.title, grant_round.name/description. `scope=all` queries 3 tables, merges by `created_unix DESC`.

### 5.2 Assistant Backend

**`services/hackforger/assistant.go`:**

```go
type AssistantResponse struct {
    Message    string `json:"message"`
    Status     string `json:"status"`     // "placeholder" | "ready"
    Disclaimer string `json:"disclaimer"`
}

func Chat(ctx context.Context, query string, locale translation.Locale) (*AssistantResponse, error) {
    return &AssistantResponse{
        Message:    locale.TrString("hackforger.assistant.coming_soon"),
        Status:     "placeholder",
        Disclaimer: locale.TrString("hackforger.assistant.disclaimer"),
    }, nil
}
```

Current implementation returns i18n placeholder text. Future: swap internals for LLM call without changing the interface.

### 5.3 API Routes

```
GET  /api/v1/hackforger/search?q=&scope=&page=&limit=
    → { results: [], total: int }

POST /api/v1/hackforger/assistant/chat
    Body: { "query": "..." }
    → { message: "...", status: "placeholder", disclaimer: "..." }
```

### 5.4 Web Routes

```
GET  /hackforger/search?q=&scope=&page=&limit=  → JSON (for modal fetch)
POST /hackforger/assistant/chat                  → JSON (for modal fetch)
```

### 5.5 Frontend: ⌘K Search Modal

**Trigger:**
- Search icon (magnifying glass) added to navbar right side in `templates/base/head_navbar.tmpl`
- Global `⌘K` / `Ctrl+K` keyboard shortcut

**Components:**

| File | Purpose |
|------|---------|
| `templates/base/head_navbar.tmpl` | Inject search icon button |
| `templates/hackforger/search_modal.tmpl` | Modal HTML skeleton |
| `web_src/js/features/hackforger/search-modal.js` | Interaction logic |
| `web_src/css/features/hackforger/search-modal.css` | Modal styling (hf-* classes) |

**Modal Layout:**

```
┌──────────────────────────────────────────────┐
│ [icon] Search or Ask something        [⌘K]  │ ← input
│──────────────────────────────────────────────│
│ ┌──────────────────────────────────────────┐ │
│ │ {assistant.message}                      │ │ ← from /assistant/chat
│ │                     {assistant.disclaimer}│ │
│ └──────────────────────────────────────────┘ │
│──────────────────────────────────────────────│
│ [All] [Hackathons] [Bounties] [Grants]      │ ← scope tabs
│──────────────────────────────────────────────│
│ [icon] Result Title              Status     │ ← from /search
│ [icon] Result Title              Status     │
│                                              │
│              No more results                 │
└──────────────────────────────────────────────┘
```

**Behavior:**
- Modal fetches from **web routes** (`/hackforger/search`, `/hackforger/assistant/chat`), not API routes, per session-cookie auth convention (Vue components must call web routes, NOT `/api/v1/`)
- Input debounce 300ms → parallel fetch `/hackforger/search` + `/hackforger/assistant/chat`
- Scope tab click → re-fetch search only
- Click result → navigate to detail page
- `Escape` or click outside → close modal
- Result icons by type: Hackathon (rocket), Bounty (dollar), Grant (heart)

### 5.6 i18n Keys

```ini
[hackforger]
search.placeholder = Search or Ask something
search.scope.all = All
search.scope.hackathons = Hackathons
search.scope.bounties = Bounties
search.scope.grants = Grants
search.no_results = No results found
assistant.coming_soon = AI assistant coming soon. Stay tuned!
assistant.disclaimer = AI responses may be inaccurate. Please double-check.
```

**zh-CN (`locale_zh-CN.ini`):**

```ini
[hackforger]
search.placeholder = 搜索或提问
search.scope.all = 全部
search.scope.hackathons = 黑客松
search.scope.bounties = 悬赏
search.scope.grants = 资助
search.no_results = 未找到结果
assistant.coming_soon = AI 助手功能即将上线，敬请期待！
assistant.disclaimer = AI 回答可能不准确，请自行核实。
```

---

## 6. Webhook Dispatch

### 6.1 Register HackForger Event Types

In `modules/webhook/type.go`, add ~20 new `HookEventType` constants for HackForger events (hackathon_created, bounty_created, grant_round_created, etc.). Update `Event()` method.

### 6.2 ActionType → HookEventType Mapping

In `models/hackforger/action_types.go`, add mapping table:

```go
var ActionTypeToHookEvent = map[activities_model.ActionType]webhook_module.HookEventType{
    ActionHackathonCreated:      webhook_module.HookEventHackathonCreated,
    ActionHackathonPhaseChanged: webhook_module.HookEventHackathonStatusChanged,
    ActionBountyCreated:         webhook_module.HookEventBountyCreated,
    // ... all 20+ mappings
}
```

### 6.3 Payload Structure

In `modules/structs/hackforger_webhook.go`:

```go
type HackforgerWebhookPayload struct {
    Action     string         `json:"action"`
    EntityType string         `json:"entity_type"`
    EntityID   int64          `json:"entity_id"`
    EntityName string         `json:"entity_name"`
    Sender     *User          `json:"sender"`
    Extra      map[string]any `json:"extra,omitempty"`
}
```

Implements `api.Payloader` interface for Forgejo's webhook system.

### 6.4 Integration Point

Single change in `PublishHackforgerAction()` — after existing Feed insert logic, add:

```go
if hookEvent, ok := ActionTypeToHookEvent[opts.OpType]; ok {
    payload := buildWebhookPayload(opts, hookEvent)
    source := webhook_service.EventSource{Owner: actUser}
    if opts.RepoID > 0 {
        source.Repository = loadRepo(ctx, opts.RepoID)
    }
    _ = webhook_service.PrepareWebhooks(ctx, source, hookEvent, payload)
}
```

Zero changes to the ~30 call sites. One function modification covers all events.

### 6.5 Webhook Configuration UI

Update `templates/webhook/shared-settings.tmpl` to add a "HackForger Events" checkbox group alongside existing Forgejo event checkboxes, so users can subscribe to HackForger events per-repo or per-org.

---

## 7. hackforger-cli + Swagger + Skill

### 7.1 Swagger Completion

Before CLI generation:
- Audit all `routers/api/v1/hackforger/*.go` for missing `swagger:operation` annotations
- Add annotations for new endpoints (search, assistant/chat, distribute)
- Verify: `make swagger && make swagger-check` passes

### 7.2 CLI Architecture

**Location:** `cmd/hackforger-cli/`
**Framework:** cobra (Go CLI framework)
**Generation strategy:** openapi-generator → Go client library → hand-written cobra commands on top

**Command structure:**

```
hackforger-cli [global flags] <resource> <action> [args] [flags]

Global flags:
  --url       Instance URL (env: HACKFORGER_URL)
  --token     API token (env: HACKFORGER_TOKEN)
  --output    json | table | yaml (default: table)

Resources: hackathon, track, registration, submission, judge,
           bounty, reward, grant, credits, credits-admin,
           reputation, search, feed, assistant
```

Full API coverage — every Swagger-documented endpoint has a corresponding CLI command.

> **Distribution:** `hackforger-cli` is a standalone binary, separate from the Forgejo server (`gitea`). It ships as its own release artifact and can be installed independently (e.g., on developer machines that don't run the server).

### 7.3 Build

```makefile
.PHONY: hackforger-cli
hackforger-cli:
	go build -o hackforger-cli ./cmd/hackforger-cli/
```

### 7.4 Claude Code Skill

`.claude/skills/hackforger-api.md` — wraps CLI for natural language interaction with the HackForger instance. Environment: `HACKFORGER_URL` + `HACKFORGER_TOKEN`.

Update `.mcp.json` to remove non-functional `forgejo-mcp` entry.

---

## 8. E2E Testing

### 8.1 Guide Updates

Update `docs/tests/e2e/e2e-testing-guide.md` with:

**Mandatory rules section:**
1. Web UI testing is mandatory — unit tests and API tests cannot substitute for web interface operations
2. Key verification checkpoints must include `agent-browser screenshot` — saved to `docs/tests/e2e/screenshots/` and referenced in reports as evidence
3. API calls only for auxiliary setup — core flow verification must go through the browser

**Test accounts alignment** with PRD role names.

**New feature coverage:** Search ⌘K modal, Webhook configuration page, hackforger-cli smoke test.

### 8.2 E2E Prompts

New E2E prompt files in `docs/tests/e2e/`:
- `week5-search-assistant-e2e.md` — ⌘K modal open/search/scope/close
- `week5-webhook-e2e.md` — Configure webhook, trigger event, verify delivery
- `week5-full-journey-e2e.md` — End-to-end user journey validation (all 6 journeys via Web)

Each prompt explicitly states:
> **Web 端测试是必选项，不可使用单元测试或 API 测试替代 Web 界面操作。关键验证节点必须使用 agent-browser 截屏记录作为测试证据。**

### 8.3 Execution

After all integration tests pass and features are implemented:
1. Build and start server in worktree
2. Execute E2E prompts via agent-browser on `http://localhost:3000`
3. Capture screenshots at every verification checkpoint
4. Generate E2E report with embedded screenshot evidence

---

## 9. Execution Order

```
Phase A: TDD Foundation
├─ A1. Enrich test fixtures (all entity states, cross-journey consistency)
├─ A2. API renames via TDD (4 renames)
└─ A3. Integration test skeleton (helpers + empty test files)

Phase B: Integration Tests + Feature Implementation (TDD cycle)
├─ B1. Hackathon tests ←→ fix discovered issues
├─ B2. Bounty tests ←→ fix
├─ B3. Grant tests (+ distribute endpoint) ←→ fix
├─ B4. Credits tests (+ fulfill array body) ←→ fix
├─ B5. Feed + Reputation tests ←→ fix
├─ B6. Search tests → implement search backend → pass
├─ B7. Assistant tests → implement assistant.go → pass
└─ B8. Webhook tests → implement webhook dispatch → pass

Phase C: Frontend + Swagger + CLI
├─ C1. Swagger annotation completion + make swagger-check
├─ C2. ⌘K search modal (navbar injection + modal + JS)
├─ C3. hackforger-cli (generate from swagger + manual adjustment)
└─ C4. Claude Code skill configuration

Phase D: E2E Acceptance
├─ D1. Update e2e-testing-guide.md (mandatory rules + new features)
├─ D2. Write Week 5 E2E prompts
└─ D3. Execute E2E (agent-browser web tests + screenshot evidence)
```

**Dependencies:** A → B → C1 → C3. C2 parallel with C1. B+C → D.

**Parallelizable worktrees:** B1-B5 per module, C2 with C1/C3, D1-D2 during B/C.

---

## 10. Files Changed Summary

| Category | New Files | Modified Files |
|----------|-----------|----------------|
| Fixtures (YAML) | 0 | ~20 (enrich existing) |
| Integration tests | 9 | 0 |
| Services | 2 (search.go, assistant.go) | 1 (notifier.go — webhook dispatch) |
| Routers API | 2 (search.go, assistant.go) | 3 (api.go renames, grants.go distribute) |
| Routers Web | 1 (search+assistant routes) | 2 (web.go renames) |
| Modules | 0 | 1 (webhook/type.go — event types) |
| Structs | 1 (hackforger_webhook.go) | 0 |
| Models | 0 | 1 (action_types.go — hook mapping) |
| Templates | 1 (search_modal.tmpl) | 2 (head_navbar.tmpl — search icon, webhook/shared-settings.tmpl — HackForger events) |
| Frontend JS | 1 (search-modal.js) | 1 (init.js) |
| Frontend CSS | 1 (search-modal.css) | 0 |
| i18n | 0 | 2 (en-US + zh-CN) |
| CLI | ~10 (cmd/hackforger-cli/) | 1 (Makefile) |
| Swagger | 0 | ~10 (annotation additions) |
| Docs | 3 (E2E prompts) | 1 (e2e-testing-guide.md) |
| Skill | 1 (.claude/skills/hackforger-api.md) | 1 (.mcp.json) |
