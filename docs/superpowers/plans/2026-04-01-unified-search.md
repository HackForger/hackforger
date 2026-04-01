# Unified Search Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace simple LIKE search with bleve-based indexer for HackForger entities, unify Notifier architecture, aggregate search across all entity types, and add Explore Submissions tab.

**Architecture:** HackForger indexer (bleve + db fallback) for 4 entity types. Forgejo Notifier interface extended with 4 HackForger methods — Feed, Webhook, and Indexer are each independent Notifier implementations. Aggregated search queries HackForger indexer + Forgejo's existing repo/user/issue search in parallel, returns grouped results.

**Tech Stack:** Go, bleve (full-text search), XORM, Forgejo Notifier pattern, vanilla JS.

**Spec:** `docs/superpowers/specs/2026-04-01-unified-search-design.md`

---

## Phase 1: Notifier Unification

This is the foundation. All subsequent phases depend on the Notifier being correctly wired.

### Task 1: Extend Notifier interface + NullNotifier + dispatch

**Files:**
- Create: `services/notify/hackforger_event.go` (HackforgerEventOpts struct)
- Modify: `services/notify/notifier.go` (append 4 methods before closing `}`)
- Modify: `services/notify/null.go` (append 4 empty implementations)
- Modify: `services/notify/notify.go` (append 4 dispatch functions)

- [ ] **Step 1.1: Create HackforgerEventOpts struct**

Create `services/notify/hackforger_event.go`:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package notify

import activities_model "forgejo.org/models/activities"

// HackforgerEventOpts holds parameters for HackForger entity events.
type HackforgerEventOpts struct {
	OpType       activities_model.ActionType // e.g., ActionHackathonCreated
	EntityType   string                      // "hackathon" | "bounty" | "grant" | "submission"
	EntityID     int64
	EntityName   string
	EntitySlug   string
	OrgID        int64
	RepoID       int64
	AudienceType int   // bit flags for Feed distribution
	TargetUserID int64 // for direct-user audience
	Content      any   // JSON-serializable extra data
}
```

- [ ] **Step 1.2: Extend Notifier interface**

In `services/notify/notifier.go`, append before closing `}` (after `ActionRunNowDone`):

```go
	// HackForger entity events
	HackforgerEntityCreated(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts)
	HackforgerEntityUpdated(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts)
	HackforgerEntityDeleted(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts)
	HackforgerEntityStatusChanged(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts)
```

- [ ] **Step 1.3: Add NullNotifier empty implementations**

Append to `services/notify/null.go`:

```go
func (*NullNotifier) HackforgerEntityCreated(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts) {
}
func (*NullNotifier) HackforgerEntityUpdated(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts) {
}
func (*NullNotifier) HackforgerEntityDeleted(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts) {
}
func (*NullNotifier) HackforgerEntityStatusChanged(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts) {
}
```

- [ ] **Step 1.4: Add dispatch functions**

Append to `services/notify/notify.go`:

```go
func HackforgerEntityCreated(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts) {
	for _, notifier := range notifiers {
		notifier.HackforgerEntityCreated(ctx, doer, opts)
	}
}

func HackforgerEntityUpdated(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts) {
	for _, notifier := range notifiers {
		notifier.HackforgerEntityUpdated(ctx, doer, opts)
	}
}

func HackforgerEntityDeleted(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts) {
	for _, notifier := range notifiers {
		notifier.HackforgerEntityDeleted(ctx, doer, opts)
	}
}

func HackforgerEntityStatusChanged(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts) {
	for _, notifier := range notifiers {
		notifier.HackforgerEntityStatusChanged(ctx, doer, opts)
	}
}
```

- [ ] **Step 1.5: Verify compilation**

Run: `go vet -tags="sqlite sqlite_unlock_notify" ./services/notify/ 2>&1`
Expected: no errors

- [ ] **Step 1.6: Commit**

```bash
git add services/notify/
git commit -m "feat(notify): extend Notifier interface with 4 HackForger entity event methods"
```

### Task 2: Atomic refactor — Notifier implementations + call site migration

> **IMPORTANT:** Tasks 2, 3, and 4 from the original plan are merged into one atomic task. This ensures every commit compiles. The approach: keep `PublishHackforgerAction` alive as a deprecated wrapper during migration, then remove it after all call sites are updated.

**Files:**
- Modify: `services/hackforger/notifier.go` (refactor + add Notifier methods)
- Modify: `services/webhook/notifier.go` (add 4 HackForger methods)
- Modify: `services/hackforger/notifier_test.go` (update test calls)
- Modify: ~9 service/router files (replace call sites)

**Sub-step A: Move AudienceType constants to `services/notify/hackforger_event.go`**

Add to the file created in Task 1:

```go
type HackforgerAudienceType int

const (
	AudienceGlobal       HackforgerAudienceType = 1 << iota
	AudienceFollowers
	AudienceOrgMembers
	AudienceRepoWatchers
	AudienceDirectUser
)
```

Change `HackforgerEventOpts.AudienceType` field type from `int` to `HackforgerAudienceType`.

**Sub-step B: Add `publishFeed` internal function to `services/hackforger/notifier.go`**

Keep `PublishHackforgerAction` temporarily as a deprecated wrapper. Create `publishFeed` alongside it:

```go
// publishFeed writes a HackForger event to the hackforger_action table.
// doer.ID replaces the old ActUserID field.
func publishFeed(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) error {
	// Same logic as current PublishHackforgerAction, but:
	// - Uses doer.ID instead of opts.ActUserID
	// - Uses opts from notify_service.HackforgerEventOpts
	// - No webhook dispatch (moved to webhookNotifier)
}
```

Implement the 4 Notifier methods calling `publishFeed`.

**Sub-step C: Add 4 webhook methods to `services/webhook/notifier.go`**

(Same as old Task 3 — append webhook dispatch methods)

**Sub-step D: Replace all ~30 call sites**

For each `PublishHackforgerAction` call in services/ and routers/:
- Change to `notify_service.HackforgerEntityCreated(ctx, doer, &notify_service.HackforgerEventOpts{...})` or `HackforgerEntityStatusChanged` based on the OpType
- Also update `services/hackforger/notifier_test.go` (~8 calls)

**Sub-step E: Remove deprecated `PublishHackforgerAction` and old `HackforgerActionOpts`**

After all call sites are migrated, delete:
- `PublishHackforgerAction` function
- `HackforgerActionOpts` struct
- Old `AudienceType` constants from `services/hackforger/notifier.go`

- [ ] **Step 2.1: Execute sub-steps A through E**

- [ ] **Step 2.2: Verify full compilation**

Run: `go vet -tags="sqlite sqlite_unlock_notify" ./services/... ./routers/... 2>&1`
Expected: no errors

- [ ] **Step 2.3: Run all tests**

Run: `go test ./models/hackforger/... ./services/hackforger/... -tags="sqlite sqlite_unlock_notify" -count=1 2>&1 | tail -5`
Expected: 100 tests pass

- [ ] **Step 2.4: Commit (atomic — compiles and passes)**

```bash
git add services/ routers/
git commit -m "refactor(notify): unify HackForger events via Notifier — Feed, Webhook, ~30 call sites migrated"
```

### Task 3: (merged into Task 2)

**Files:**
- Modify: `services/webhook/notifier.go` (append 4 methods)

- [ ] **Step 3.1: Read current webhook notifier**

Read `services/webhook/notifier.go` to understand the pattern (e.g., how `NewIssue` dispatches webhooks).

- [ ] **Step 3.2: Implement 4 HackForger webhook methods**

Append to `services/webhook/notifier.go`:

```go
func (m *webhookNotifier) HackforgerEntityCreated(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	m.dispatchHackforgerWebhook(ctx, doer, opts)
}

func (m *webhookNotifier) HackforgerEntityUpdated(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	// No webhook for field updates (name/description changes)
}

func (m *webhookNotifier) HackforgerEntityDeleted(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	// No webhook for deletion
}

func (m *webhookNotifier) HackforgerEntityStatusChanged(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	m.dispatchHackforgerWebhook(ctx, doer, opts)
}

func (m *webhookNotifier) dispatchHackforgerWebhook(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	hookEvent, ok := hackforger_model.ActionTypeToHookEvent[opts.OpType]
	if !ok {
		return
	}
	payload := &structs.HackforgerWebhookPayload{
		Action:     string(hookEvent),
		EntityType: opts.EntityType,
		EntityID:   opts.EntityID,
		EntityName: opts.EntityName,
		Sender:     convert.ToUser(ctx, doer, nil),
	}
	source := EventSource{Owner: doer}
	if opts.RepoID > 0 {
		if repo, err := repo_model.GetRepositoryByID(ctx, opts.RepoID); err == nil {
			source.Repository = repo
		}
	}
	if err := PrepareWebhooks(ctx, source, hookEvent, payload); err != nil {
		log.Error("PrepareWebhooks for HackForger event %s: %v", hookEvent, err)
	}
}
```

Add necessary imports. Read the file first to check what's already imported.

- [ ] **Step 3.3: Verify compilation**

Run: `go vet -tags="sqlite sqlite_unlock_notify" ./services/webhook/ 2>&1`

- [ ] **Step 3.4: Commit**

```bash
git add services/webhook/notifier.go
git commit -m "feat(webhook): add HackForger event methods to webhookNotifier"
```

### Task 4: Replace ~30 PublishHackforgerAction call sites

**Files:**
- Modify: `services/hackforger/bounty.go`
- Modify: `services/hackforger/hackathon.go`
- Modify: `services/hackforger/hackathon_judge.go`
- Modify: `services/hackforger/grants.go`
- Modify: `services/hackforger/credits.go`
- Modify: `routers/api/v1/hackforger/bounty.go`
- Modify: `routers/api/v1/hackforger/hackathon_registration.go`
- Modify: `routers/web/hackforger/bounty.go`
- Modify: `routers/web/hackforger/hackathon.go`

- [ ] **Step 4.1: Find all call sites**

Run: `grep -rn "PublishHackforgerAction" services/ routers/ --include="*.go" | grep -v "_test.go" | grep -v "func publishFeed"`

This gives the exact list of files and lines to change.

- [ ] **Step 4.2: Replace each call site**

For each call site, change from:

```go
// OLD:
hackforger_svc.PublishHackforgerAction(ctx, &hackforger_svc.HackforgerActionOpts{
	ActUserID:    doer.ID,
	OpType:       hackforger_model.ActionBountyCreated,
	EntityType:   "bounty",
	EntityID:     bounty.ID,
	EntityName:   bounty.Title,
	RepoID:       bounty.RepoID,
	AudienceType: hackforger_svc.AudienceGlobal | hackforger_svc.AudienceFollowers,
	Content:      content,
})
```

To:

```go
// NEW:
notify_service.HackforgerEntityCreated(ctx, doer, &notify_service.HackforgerEventOpts{
	OpType:       hackforger_model.ActionBountyCreated,
	EntityType:   "bounty",
	EntityID:     bounty.ID,
	EntityName:   bounty.Title,
	RepoID:       bounty.RepoID,
	AudienceType: int(hackforger_svc.AudienceGlobal | hackforger_svc.AudienceFollowers),
	Content:      content,
})
```

Key changes per call site:
- `hackforger_svc.PublishHackforgerAction(ctx, &hackforger_svc.HackforgerActionOpts{` → `notify_service.HackforgerEntityXxx(ctx, doer, &notify_service.HackforgerEventOpts{`
- Remove `ActUserID` field (now `doer` param)
- `AudienceType` field value may need int cast
- Choose correct Notifier method: `Created` / `StatusChanged` based on the `OpType`
- Add `import notify_service "forgejo.org/services/notify"` where missing

**Mapping OpType → Notifier method:**
- `ActionHackathonCreated`, `ActionBountyCreated`, `ActionGrantRoundCreated`, `ActionHackathonRegistered`, `ActionHackathonSubmitted`, `ActionGrantProjectSubmitted` → `HackforgerEntityCreated`
- `ActionHackathonPhaseChanged`, `ActionHackathonFinalized`, `ActionBountyCompleted`, `ActionBountyPaid`, `ActionBountyClaimed`, `ActionBountyExpired`, `ActionBountyCancelled`, `ActionBountyWinnersSelected`, `ActionBountyDelivered`, `ActionGrantRoundOpened`, `ActionGrantRoundClosed`, `ActionGrantRoundFinalized`, `ActionGrantRoundCancelled`, `ActionGrantAwarded`, `ActionCreditsRedeemed`, `ActionOrderFulfilled`, `ActionOrderCancelled`, `ActionHackathonScored` → `HackforgerEntityStatusChanged`

- [ ] **Step 4.3: Verify full compilation**

Run: `go vet -tags="sqlite sqlite_unlock_notify" ./services/hackforger/ ./routers/api/v1/hackforger/ ./routers/web/hackforger/ 2>&1`
Expected: no errors

- [ ] **Step 4.4: Run all tests**

Run: `go test ./models/hackforger/... ./services/hackforger/... -tags="sqlite sqlite_unlock_notify" -count=1 2>&1 | tail -5`
Expected: 100 tests pass

- [ ] **Step 4.5: Commit**

```bash
git add services/ routers/
git commit -m "refactor(notify): replace ~30 PublishHackforgerAction calls with Notifier dispatch"
```

---

## Phase 2: HackForger Indexer

### Task 5: Create indexer internal package

**Files:**
- Create: `modules/indexer/hackforger/internal/indexer.go`
- Create: `modules/indexer/hackforger/internal/model.go`

- [ ] **Step 5.1: Create model.go**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package internal

import "forgejo.org/models/db"

// IndexerData represents a HackForger entity in the search index.
type IndexerData struct {
	ID          int64  `json:"id"`
	EntityType  string `json:"entity_type"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Status      int    `json:"status"`
	OwnerID     int64  `json:"owner_id"`
	OrgID       int64  `json:"org_id"`
	RepoID      int64  `json:"repo_id"`
	HackathonID int64  `json:"hackathon_id"`
	CreatedUnix int64  `json:"created_unix"`
	UpdatedUnix int64  `json:"updated_unix"`
}

// SearchOptions holds search parameters.
type SearchOptions struct {
	Keyword    string
	EntityType string // filter by type, empty = all
	Paginator  *db.ListOptions
}

// Match represents a single search hit.
type Match struct {
	ID         int64   `json:"id"`
	EntityType string  `json:"entity_type"`
	Score      float64 `json:"score"`
}

// SearchResult holds search results.
type SearchResult struct {
	Total int64
	Hits  []Match
}
```

- [ ] **Step 5.2: Create indexer.go interface**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package internal

import (
	"context"

	inner "forgejo.org/modules/indexer/internal"
)

// Indexer defines the HackForger search indexer interface.
type Indexer interface {
	inner.Indexer
	Index(ctx context.Context, data ...*IndexerData) error
	Delete(ctx context.Context, entityType string, ids ...int64) error
	Search(ctx context.Context, opts *SearchOptions) (*SearchResult, error)
}
```

Read `modules/indexer/internal/indexer.go` first to confirm the base `Indexer` interface (Init, Ping, Close).

- [ ] **Step 5.3: Commit**

```bash
git add modules/indexer/hackforger/
git commit -m "feat(indexer): hackforger internal package — Indexer interface + data models"
```

### Task 6: Implement DB fallback backend

**Files:**
- Create: `modules/indexer/hackforger/db/db.go`

- [ ] **Step 6.1: Implement DB indexer**

This replaces the current LIKE queries in `services/hackforger/search.go`. It's the fallback when bleve is unavailable.

```go
package db

import (
	"context"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/indexer/hackforger/internal"
)

type Indexer struct{}

func NewIndexer() *Indexer { return &Indexer{} }

func (i *Indexer) Init(ctx context.Context) (bool, error) { return true, nil }
func (i *Indexer) Ping(ctx context.Context) error         { return nil }
func (i *Indexer) Close()                                  {}

// Index is a no-op for DB backend (data already in DB).
func (i *Indexer) Index(ctx context.Context, data ...*internal.IndexerData) error { return nil }

// Delete is a no-op for DB backend.
func (i *Indexer) Delete(ctx context.Context, entityType string, ids ...int64) error { return nil }

// Search performs LIKE queries against the database.
func (i *Indexer) Search(ctx context.Context, opts *internal.SearchOptions) (*internal.SearchResult, error) {
	// Query 4 tables with LIKE, merge results
	// Follow the pattern from the current services/hackforger/search.go
	// but add submission search
	var hits []internal.Match
	keyword := "%" + opts.Keyword + "%"
	e := db.GetEngine(ctx)

	if opts.EntityType == "" || opts.EntityType == "hackathon" {
		var hackathons []*hackforger_model.Hackathon
		if err := e.Where("name LIKE ? OR description LIKE ?", keyword, keyword).
			OrderBy("created_unix DESC").Limit(5).Find(&hackathons); err != nil {
			return nil, err
		}
		for _, h := range hackathons {
			hits = append(hits, internal.Match{ID: h.ID, EntityType: "hackathon"})
		}
	}
	if opts.EntityType == "" || opts.EntityType == "bounty" {
		var bounties []*hackforger_model.Bounty
		if err := e.Where("title LIKE ?", keyword).
			OrderBy("created_unix DESC").Limit(5).Find(&bounties); err != nil {
			return nil, err
		}
		for _, b := range bounties {
			hits = append(hits, internal.Match{ID: b.ID, EntityType: "bounty"})
		}
	}
	if opts.EntityType == "" || opts.EntityType == "grant" {
		var rounds []*hackforger_model.GrantRound
		if err := e.Where("name LIKE ? OR description LIKE ?", keyword, keyword).
			OrderBy("created_unix DESC").Limit(5).Find(&rounds); err != nil {
			return nil, err
		}
		for _, r := range rounds {
			hits = append(hits, internal.Match{ID: r.ID, EntityType: "grant"})
		}
	}
	if opts.EntityType == "" || opts.EntityType == "submission" {
		var submissions []*hackforger_model.HackathonSubmission
		if err := e.Where("title LIKE ? OR description LIKE ?", keyword, keyword).
			OrderBy("created_unix DESC").Limit(5).Find(&submissions); err != nil {
			return nil, err
		}
		for _, s := range submissions {
			hits = append(hits, internal.Match{ID: s.ID, EntityType: "submission"})
		}
	}

	return &internal.SearchResult{Total: int64(len(hits)), Hits: hits}, nil
}
```

- [ ] **Step 6.2: Write unit test for DB search**

Create `modules/indexer/hackforger/db/db_test.go`:
- Test `Search()` returns results matching keyword across 4 entity types
- Test `Search()` with EntityType filter returns only that type
- Test `Search()` with non-matching keyword returns empty
- Uses `unittest.PrepareTestDatabase()` for fixture data

- [ ] **Step 6.3: Run test to verify it passes**

Run: `go test -tags="sqlite sqlite_unlock_notify" ./modules/indexer/hackforger/db/ -v -count=1`

- [ ] **Step 6.4: Commit**

```bash
git add modules/indexer/hackforger/db/
git commit -m "feat(indexer): hackforger DB fallback backend — LIKE queries for 4 entity types"
```

### Task 7: Implement bleve backend

**Files:**
- Create: `modules/indexer/hackforger/bleve/bleve.go`

- [ ] **Step 7.1: Read the issues bleve indexer for reference**

Read `modules/indexer/issues/bleve/bleve.go` to understand:
- How `inner_bleve.NewIndexer()` is used
- How index mappings are defined
- How batch indexing works
- How search queries are built

- [ ] **Step 7.2: Implement bleve indexer**

Follow the issues bleve pattern. Key differences:
- Single index for 4 entity types (filtered by `EntityType` keyword field)
- `Title` field boosted (weight 2.0)
- `Content` field normal weight
- `EntityType`, `Status`, `OrgID` as keyword/numeric fields for filtering

The implementation will be ~150-200 lines. Use `inner_bleve.NewFlushingBatch` for batch indexing. Build search queries with bleve's `NewBooleanQuery` for combining keyword search with entity type filter.

- [ ] **Step 7.3: Verify compilation**

Run: `go vet -tags="sqlite sqlite_unlock_notify" ./modules/indexer/hackforger/... 2>&1`

- [ ] **Step 7.4: Commit**

```bash
git add modules/indexer/hackforger/bleve/
git commit -m "feat(indexer): hackforger bleve backend — full-text search with unicode normalization"
```

### Task 8: Create public indexer API + init

**Files:**
- Create: `modules/indexer/hackforger/indexer.go`
- Modify: `services/indexer/indexer.go` (register init)
- Modify: `services/indexer/notify.go` (add 4 HackForger indexer methods)

- [ ] **Step 8.1: Create public API**

`modules/indexer/hackforger/indexer.go` — follows `modules/indexer/issues/indexer.go` pattern:
- `InitHackforgerIndexer(syncReindex bool)` — switch on setting, create backend, start queue
- `UpdateHackforgerIndexer(ctx, entityType, ids...)` — push to queue
- `DeleteHackforgerIndexer(ctx, entityType, ids...)` — push delete to queue
- `SearchHackforger(ctx, opts)` — call global indexer
- Queue handler: fetch `IndexerData` from DB by entityType+ID, call `indexer.Index()`

Data loading helper: given entityType + ID, query the appropriate model table and return `*IndexerData`. This needs a switch on entityType to call the right model function.

- [ ] **Step 8.2: Add configuration**

Read `modules/setting/indexer.go` to understand how indexer settings are loaded. Add:
```go
Indexer.HackforgerType // "bleve" | "db", default "bleve"
Indexer.HackforgerPath // default "indexers/hackforger.bleve"
```

- [ ] **Step 8.3: Register in services/indexer/indexer.go**

Add to `Init()`:
```go
hackforger_indexer.InitHackforgerIndexer(false)
```

- [ ] **Step 8.4: Add indexerNotifier HackForger methods**

Append to `services/indexer/notify.go`:
```go
func (r *indexerNotifier) HackforgerEntityCreated(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	hackforger_indexer.UpdateHackforgerIndexer(ctx, opts.EntityType, opts.EntityID)
}
func (r *indexerNotifier) HackforgerEntityUpdated(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	hackforger_indexer.UpdateHackforgerIndexer(ctx, opts.EntityType, opts.EntityID)
}
func (r *indexerNotifier) HackforgerEntityDeleted(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	hackforger_indexer.DeleteHackforgerIndexer(ctx, opts.EntityType, opts.EntityID)
}
func (r *indexerNotifier) HackforgerEntityStatusChanged(ctx context.Context, doer *user_model.User, opts *notify_service.HackforgerEventOpts) {
	hackforger_indexer.UpdateHackforgerIndexer(ctx, opts.EntityType, opts.EntityID)
}
```

- [ ] **Step 8.5: Add admin reindex endpoint**

Register `POST /api/v1/admin/hackforger/reindex` in `routers/api/v1/api.go` (admin group). Handler calls `hackforger_indexer.InitHackforgerIndexer(true)` to force full reindex. Requires admin auth.

- [ ] **Step 8.6: Verify compilation + tests**

Run: `go vet -tags="sqlite sqlite_unlock_notify" ./modules/indexer/hackforger/ ./services/indexer/ 2>&1`
Run: `go test ./models/hackforger/... ./services/hackforger/... -tags="sqlite sqlite_unlock_notify" -count=1 2>&1 | tail -5`

- [ ] **Step 8.6: Commit**

```bash
git add modules/indexer/hackforger/indexer.go modules/setting/ services/indexer/
git commit -m "feat(indexer): hackforger public API, init registration, indexerNotifier methods"
```

---

## Phase 3: Aggregated Search + Modal

### Task 9: Rewrite search service

**Files:**
- Modify: `services/hackforger/search.go` (complete rewrite)

- [ ] **Step 9.1: Rewrite with UnifiedSearch**

Replace the current `Search()` function with `UnifiedSearch()` that:
1. Calls HackForger indexer for hackathon/bounty/grant/submission
2. Calls `repo_model.SearchRepository()` for repos
3. Calls `user_model.SearchUsers()` for users
4. Calls `issue_indexer.SearchIssues()` for issues
5. Returns `GroupedSearchResult`

Read `models/repo/search.go` for `SearchRepository` signature and `models/user/search.go` for `SearchUsers` signature before implementing.

Each source called in a goroutine for parallel execution, results collected via channels.

- [ ] **Step 9.2: Verify compilation**

- [ ] **Step 9.3: Commit**

```bash
git add services/hackforger/search.go
git commit -m "feat(search): rewrite with UnifiedSearch — parallel query across 7 entity types"
```

### Task 10: Update search routes + modal

**Files:**
- Modify: `routers/api/v1/hackforger/search.go`
- Modify: `routers/web/hackforger/search.go`
- Modify: `web_src/js/features/hackforger/search-modal.js`
- Modify: `templates/hackforger/search_modal.tmpl` (or `templates/base/head_navbar.tmpl`)

- [ ] **Step 10.1: Update API route handler**

Change `SearchAPI` to call `UnifiedSearch()` and return `GroupedSearchResult`.

- [ ] **Step 10.2: Update web route handler**

Same change for `SearchWeb`.

- [ ] **Step 10.3: Redesign modal JS**

In `search-modal.js`:
- Remove scope tab switching logic
- Replace `renderResults()` with `renderGroupedResults(data)` that:
  - Iterates over each group key (hackathons, bounties, grants, submissions, repos, users, issues)
  - Skips empty groups
  - Renders group header with i18n title + "View all →" link
  - Renders items within each group

- [ ] **Step 10.4: Update modal template**

Remove scope tab HTML. The results container now renders groups dynamically from JS.

- [ ] **Step 10.5: Add i18n keys for group headers + view all**

```ini
; en-US
hackforger.search.group.hackathons = Hackathons
hackforger.search.group.bounties = Bounties
hackforger.search.group.grants = Grants
hackforger.search.group.submissions = Submissions
hackforger.search.group.repos = Repositories
hackforger.search.group.users = Users
hackforger.search.group.issues = Issues
hackforger.search.view_all = View all

; zh-CN
hackforger.search.group.hackathons = 黑客松
hackforger.search.group.bounties = 悬赏
hackforger.search.group.grants = 资助
hackforger.search.group.submissions = 提交作品
hackforger.search.group.repos = 仓库
hackforger.search.group.users = 用户
hackforger.search.group.issues = 议题
hackforger.search.view_all = 查看全部
```

- [ ] **Step 10.6: Build frontend**

Run: `make frontend`
Expected: webpack compiles successfully

- [ ] **Step 10.7: Commit**

```bash
git add routers/ web_src/ templates/ options/locale/
git commit -m "feat(search): grouped results in ⌘K modal with view-all links"
```

---

## Phase 4: Explore Submissions

### Task 11: Create Explore Submissions page

**Files:**
- Create: `routers/web/hackforger/submission.go`
- Create: `templates/hackforger/submission/explore.tmpl`
- Modify: `routers/web/web.go` (register route)
- Modify: `templates/explore/navbar.tmpl` (add tab)
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 11.1: Create handler**

`routers/web/hackforger/submission.go`:
- `ExploreSubmissions(ctx *context.Context)` — query submissions with JOIN on hackathon for status
- Support keyword filter (`q` param), hackathon filter, sorting, pagination
- Set `ctx.Data["PageIsExploreSubmissions"] = true` for navbar active state

- [ ] **Step 11.2: Create template**

`templates/hackforger/submission/explore.tmpl` — follow the pattern of `templates/hackforger/explore.tmpl`:
- Uses `explore/navbar` template
- Lists submissions with title, author, hackathon, track, status badge
- Score/Rank only shown when hackathon status = Finished(4)
- DemoURL as external link icon

- [ ] **Step 11.3: Register route**

In `routers/web/web.go`, inside the explore group (where hackathons/bounties/grants are):
```go
m.Get("/submissions", hackforger_web.ExploreSubmissions)
```

- [ ] **Step 11.4: Add navbar tab**

In `templates/explore/navbar.tmpl`, after Grants tab:
```html
<a class="{{if .PageIsExploreSubmissions}}active {{end}}item" href="{{AppSubUrl}}/explore/submissions">
	{{svg "octicon-file-code"}} {{ctx.Locale.Tr "hackforger.explore.submissions"}}
</a>
```

- [ ] **Step 11.5: Add i18n keys**

Append submission-related keys to both locale files.

- [ ] **Step 11.6: Build and verify**

```bash
make frontend && TAGS="bindata sqlite sqlite_unlock_notify" make backend
```

- [ ] **Step 11.7: Commit**

```bash
git add routers/web/hackforger/submission.go templates/hackforger/submission/ \
  routers/web/web.go templates/explore/navbar.tmpl options/locale/
git commit -m "feat: Explore Submissions tab — browse hackathon works with score/rank"
```

---

## Phase 5: E2E + Verification

### Task 12: Write E2E prompt + execute

**Files:**
- Create: `docs/tests/e2e/unified-search-e2e-prompt.md`

- [ ] **Step 12.1: Write E2E prompt**

Cover:
- TC-US1: ⌘K search returns grouped results (hackathons + repos + users)
- TC-US2: Each group has "View all" link that navigates correctly
- TC-US3: Empty groups are not displayed
- TC-US4: Explore Submissions page loads and shows submissions
- TC-US5: Submission score/rank hidden for non-Finished hackathons
- TC-US6: AI assistant panel still works in new modal layout

All test cases must include `agent-browser screenshot` at verification points.

- [ ] **Step 12.2: Rebuild server**

```bash
make frontend && TAGS="bindata sqlite sqlite_unlock_notify" make backend
```

- [ ] **Step 12.3: Execute E2E tests via agent-browser**

- [ ] **Step 12.4: Write E2E report with screenshots**

- [ ] **Step 12.5: Final test verification**

```bash
go test ./models/hackforger/... ./services/hackforger/... -tags="sqlite sqlite_unlock_notify" -count=1
go vet -tags="sqlite sqlite_unlock_notify" ./...
```

- [ ] **Step 12.6: Commit**

```bash
git add docs/tests/e2e/
git commit -m "docs: unified search E2E prompt + report with screenshots"
```
