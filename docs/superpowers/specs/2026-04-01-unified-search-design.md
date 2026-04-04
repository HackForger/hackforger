# Unified Search Design: HackForger Indexer + ⌘K Aggregation

> **Status:** Draft
> **Date:** 2026-04-01
> **Branch:** v0.1-dev/hackforger
> **Replaces:** Simple LIKE search in `services/hackforger/search.go`

---

## 1. Overview

Redesign the ⌘K search to be a unified global search across all HackForger and Forgejo entities. Three major changes:

1. **HackForger Indexer** — New bleve-based full-text indexer for 4 HackForger entity types (Hackathon, Bounty, Grant, Submission)
2. **Notifier Architecture Unification** — Extend Forgejo's `notify.Notifier` interface with HackForger events; refactor `PublishHackforgerAction` from direct-call to Notifier-dispatched; Feed, Webhook, and Indexer each become independent Notifier implementations
3. **Aggregated Search** — ⌘K queries HackForger indexer + Forgejo's existing repo/user/issue search, returns grouped results with "view all" links to Explore pages
4. **Explore Submissions** — New `/explore/submissions` tab for browsing hackathon works

---

## 2. HackForger Indexer

### 2.1 Architecture

New module at `modules/indexer/hackforger/` following Forgejo's indexer pattern:

```
modules/indexer/hackforger/
├── internal/
│   ├── indexer.go      — Indexer interface
│   └── model.go        — IndexerData, SearchOptions, SearchResult
├── bleve/
│   └── bleve.go        — Bleve backend
├── db/
│   └── db.go           — Database fallback (replaces current LIKE queries)
└── indexer.go           — Public API: Init, Update, Delete, Search
```

### 2.2 IndexerData (unified for all 4 entity types)

```go
type IndexerData struct {
    ID          int64
    EntityType  string  // "hackathon" | "bounty" | "grant" | "submission"
    Title       string  // name/title — primary search field
    Content     string  // description — secondary search field
    Status      int     // entity status value, for filtering
    OwnerID     int64   // creator user ID
    OrgID       int64   // org (hackathon/grant)
    RepoID      int64   // linked repo (bounty/submission)
    HackathonID int64   // parent hackathon (submission only)
    CreatedUnix int64
    UpdatedUnix int64
}
```

All 4 entity types share a single bleve index, differentiated by `EntityType` field. This avoids managing 4 separate indices.

### 2.3 Search Interface

```go
type SearchOptions struct {
    Keyword    string
    EntityType string           // filter by type, empty = all
    Paginator  *db.ListOptions
}

type SearchResult struct {
    Total int64
    Hits  []Match  // {ID int64, Score float64, EntityType string}
}
```

### 2.4 Public API

```go
// modules/indexer/hackforger/indexer.go
func InitHackforgerIndexer(syncReindex bool)
func UpdateHackforgerIndexer(ctx context.Context, entityType string, ids ...int64)
func DeleteHackforgerIndexer(ctx context.Context, entityType string, ids ...int64)
func SearchHackforger(ctx context.Context, opts *SearchOptions) (*SearchResult, error)
```

### 2.5 Backends

**Bleve (default):** Full-text search with unicode normalization. Index mapping: `Title` as text (boosted), `Content` as text, `EntityType`/`Status`/`OrgID` as keyword/numeric for filtering.

**DB fallback:** LIKE queries (current implementation), used when bleve unavailable or no keywords.

**DB fallback must include submission search** (new entity not present in current LIKE implementation).

**No ES/Meilisearch for v0.1.** Configuration via `app.ini`:

```ini
[indexer]
HACKFORGER_TYPE = bleve   ; bleve | db
HACKFORGER_PATH = indexers/hackforger.bleve
```

### 2.6 Initialization

Register `hackforger_indexer.InitHackforgerIndexer()` in `services/indexer/indexer.go` alongside existing `issue_indexer.InitIssueIndexer()` and `code_indexer.Init()` calls (this file is invoked from `routers/init.go` via `indexer_service.Init()`).

### 2.7 Reindex Strategy

On first startup (or when index is empty/corrupted), `InitHackforgerIndexer` triggers an async queue-based reindex:
1. Query all Hackathon/Bounty/GrantRound/Submission records from DB
2. Push to indexer queue in batches (batch size: 50)
3. Non-blocking — server starts accepting requests immediately, search falls back to DB until reindex completes
4. Admin reindex endpoint: `POST /api/v1/admin/hackforger/reindex` for manual trigger

---

## 3. Notifier Architecture Unification

### 3.1 Extend Notifier Interface

Add to `services/notify/notifier.go` (append before closing `}`):

```go
// HackForger events
HackforgerEntityCreated(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts)
HackforgerEntityUpdated(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts)
HackforgerEntityDeleted(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts)
HackforgerEntityStatusChanged(ctx context.Context, doer *user_model.User, opts *HackforgerEventOpts)
```

### 3.2 HackforgerEventOpts

Defined in `services/notify/` (same package as the Notifier interface, matching how other Notifier methods reference parameter types):

```go
type HackforgerEventOpts struct {
    OpType       activities_model.ActionType  // specific action (e.g., ActionHackathonCreated)
    EntityType   string  // "hackathon" | "bounty" | "grant" | "submission"
    EntityID     int64
    EntityName   string
    EntitySlug   string
    OrgID        int64
    RepoID       int64
    AudienceType int     // bit flags for Feed distribution
    TargetUserID int64   // for direct-user audience
    Content      any     // JSON-serializable extra data
}
```

This replaces the current `HackforgerActionOpts` in `services/hackforger/notifier.go`.

> **Note:** `ActUserID` from the old `HackforgerActionOpts` is replaced by the `doer *user_model.User` parameter in the Notifier method signature. `OpType` is required for the Feed writer (determines which ActionType to store) and the Webhook dispatcher (maps to HookEventType via `ActionTypeToHookEvent`).

### 3.3 Notifier Implementations

Three Notifier implementations handle HackForger events:

| Notifier | File | Responsibility |
|----------|------|---------------|
| `hackforgerNotifier` | `services/hackforger/notifier.go` | Write Feed (hackforger_action table) |
| webhook `notifier` | `services/webhook/notifier.go` | Dispatch webhook to subscribers |
| `indexerNotifier` | `services/indexer/notify.go` | Update/delete hackforger indexer |

### 3.4 Refactor: PublishHackforgerAction → Notifier Dispatch

**Before (current, ~30 call sites):**
```go
// In service/router code:
hackforger_svc.PublishHackforgerAction(ctx, &HackforgerActionOpts{...})
// PublishHackforgerAction internally: writes Feed + dispatches webhook
```

**After:**
```go
// In service/router code:
notify_service.HackforgerEntityCreated(ctx, doer, &notify.HackforgerEventOpts{...})

// Dispatches to all registered Notifiers:
// 1. hackforgerNotifier → writes Feed (calls internal publishFeed())
// 2. webhookNotifier → dispatches webhook
// 3. indexerNotifier → updates search index
```

**`PublishHackforgerAction` becomes `publishFeed()`** — an internal function of `hackforgerNotifier`, only responsible for writing to `hackforger_action` table. Webhook dispatch code moves to `webhookNotifier`.

### 3.5 Files Changed

| File | Change |
|------|--------|
| `services/notify/notifier.go` | +4 methods + import HackforgerEventOpts |
| `services/notify/null.go` | +4 empty implementations |
| `services/notify/notify.go` | +4 dispatch functions |
| `services/hackforger/notifier.go` | Refactor: implement 4 Notifier methods for Feed; remove webhook dispatch; rename PublishHackforgerAction to publishFeed |
| `services/webhook/notifier.go` | +4 methods for HackForger webhook dispatch |
| `services/indexer/notify.go` | +4 methods calling hackforger_indexer |
| ~30 call sites in services/ + routers/ | Replace `PublishHackforgerAction(opts)` → `notify_service.HackforgerEntityXxx(ctx, doer, opts)` |

### 3.6 Event Mapping

| Service Operation | Notifier Method |
|------------------|----------------|
| CreateHackathon, CreateBounty, CreateGrantRound, CreateSubmission | `HackforgerEntityCreated` |
| UpdateHackathon (name/desc), UpdateBounty (title), UpdateGrantRound, UpdateSubmission | `HackforgerEntityUpdated` |
| DeleteHackathon, DeleteBounty, DeleteGrantRound, DeleteSubmission | `HackforgerEntityDeleted` |
| Publish, Start, Judge, Finalize, Cancel, Complete, Pay, Accept, Expire, etc. | `HackforgerEntityStatusChanged` |

---

## 4. Aggregated Search Service

### 4.1 Rewritten `services/hackforger/search.go`

```go
func UnifiedSearch(ctx context.Context, keyword string, doer *user_model.User) (*GroupedSearchResult, error)
```

Calls 4 search sources in parallel:

| Source | Function | Returns |
|--------|----------|---------|
| HackForger indexer | `hackforger_indexer.SearchHackforger()` | Hackathon/Bounty/Grant/Submission hits |
| Repo search | `repo_model.SearchRepository()` | Repos (respects visibility) |
| User search | `user_model.SearchUsers()` | Users + Orgs |
| Issue search | `issue_indexer.SearchIssues()` | Issue IDs → load details |

### 4.2 Result Structure

> **Note:** The existing `SearchResult` struct is renamed to `SearchItem`; a new `GroupedSearchResult` wrapper groups items by entity type.

```go
type GroupedSearchResult struct {
    Hackathons  []*SearchItem `json:"hackathons,omitempty"`
    Bounties    []*SearchItem `json:"bounties,omitempty"`
    Grants      []*SearchItem `json:"grants,omitempty"`
    Submissions []*SearchItem `json:"submissions,omitempty"`
    Repos       []*SearchItem `json:"repos,omitempty"`
    Users       []*SearchItem `json:"users,omitempty"`
    Issues      []*SearchItem `json:"issues,omitempty"`
}

type SearchItem struct {
    Type   string `json:"type"`
    ID     int64  `json:"id"`
    Title  string `json:"title"`
    Desc   string `json:"desc,omitempty"`
    Status string `json:"status,omitempty"`
    URL    string `json:"url"`
    Icon   string `json:"icon"`
}
```

### 4.3 Grouping & Sorting

- Each group: top 5 results, sorted by score (bleve) or updated_unix (DB) within group
- Groups displayed in fixed order: Hackathons → Bounties → Grants → Submissions → Repos → Users → Issues
- Empty groups omitted from response

### 4.4 Permissions

- HackForger entities: all public, no permission check needed
- Repos: `repo_model.SearchRepository()` with `doer` for visibility filtering
- Issues: filtered by accessible repos
- Users: all public
- Anonymous users: see public content only; logged-in users additionally see authorized private repos/issues

---

## 5. ⌘K Modal Redesign

### 5.1 Layout Change

Remove scope tabs. Replace with grouped results, each group has header + "View all" link:

```
┌──────────────────────────────────────────────┐
│ 🔍 Search or Ask something            [⌘K]  │
│──────────────────────────────────────────────│
│ ┌──────────────────────────────────────────┐ │
│ │ {assistant.message}        {disclaimer}  │ │
│ └──────────────────────────────────────────┘ │
│──────────────────────────────────────────────│
│ Hackathons                    View all →     │
│   🚀 Result Title                 Status    │
│   🚀 Result Title                 Status    │
│──────────────────────────────────────────────│
│ Bounties                      View all →     │
│   💰 Result Title                 Status    │
│──────────────────────────────────────────────│
│ Repos                         View all →     │
│   📁 owner/repo                             │
│──────────────────────────────────────────────│
│ Users                         View all →     │
│   👤 username                               │
└──────────────────────────────────────────────┘
```

### 5.2 "View All" Links

| Group | Link |
|-------|------|
| Hackathons | `/explore/hackathons?q={keyword}` |
| Bounties | `/explore/bounties?q={keyword}` |
| Grants | `/explore/grants?q={keyword}` |
| Submissions | `/explore/submissions?q={keyword}` |
| Repos | `/explore/repos?q={keyword}` |
| Users | `/explore/users?q={keyword}` |
| Issues | `/issues?q={keyword}&type=your_repositories` |

### 5.3 JS Changes

- `search-modal.js`: remove tab switching logic, replace `renderResults()` with `renderGroupedResults(data)` that iterates over each non-empty group
- Single fetch to `/hackforger/search?q=...` returns `GroupedSearchResult`
- Assistant fetch unchanged

---

## 6. Explore Submissions Tab

### 6.1 Route

`GET /explore/submissions` — registered in `routers/web/web.go` inside the explore group (inherits `ignExploreSignIn` middleware).

### 6.2 Handler

`routers/web/hackforger/submission.go` — new file:

```go
func ExploreSubmissions(ctx *context.Context) {
    // Query hackathon_submission JOIN hackathon for status
    // Filter: keyword, hackathon_id (optional)
    // Sort: newest / highest score (only for Finished hackathons)
    // Pagination
    // Pass to template
}
```

### 6.3 Template

`templates/hackforger/submission/explore.tmpl`:

List item fields:
- Title (link to submission detail or hackathon page)
- Author (username + avatar)
- Hackathon name (link)
- Track name
- Status badge
- Score + Rank — **only displayed when hackathon status = Finished(4)**
- DemoURL (external link icon, if present)

### 6.4 Navbar

`templates/explore/navbar.tmpl` — add after Grants tab:

```html
<a class="{{if .PageIsExploreSubmissions}}active {{end}}item" href="{{AppSubUrl}}/explore/submissions">
    {{svg "octicon-file-code"}} {{ctx.Locale.Tr "hackforger.explore.submissions"}}
</a>
```

### 6.5 i18n Keys

```ini
; en-US
hackforger.explore.submissions = Submissions
hackforger.submission.no_submissions = No submissions found
hackforger.submission.score = Score
hackforger.submission.rank = Rank
hackforger.submission.demo = Demo

; zh-CN
hackforger.explore.submissions = 提交作品
hackforger.submission.no_submissions = 暂无提交作品
hackforger.submission.score = 得分
hackforger.submission.rank = 排名
hackforger.submission.demo = 演示
```

---

## 7. Files Changed Summary

| Category | New Files | Modified Files |
|----------|-----------|----------------|
| Indexer | 5 (`modules/indexer/hackforger/`) | 0 |
| Notifier interface | 0 | 3 (`notifier.go`, `null.go`, `notify.go`) |
| Notifier implementations | 0 | 3 (`hackforger/notifier.go`, `webhook/notifier.go`, `indexer/notify.go`) |
| Search service | 0 | 1 (`services/hackforger/search.go` rewrite) |
| Search routes | 0 | 2 (`routers/api/v1/hackforger/search.go`, `routers/web/hackforger/search.go`) |
| Explore Submissions | 2 (`submission.go` handler, `explore.tmpl`) | 2 (`web.go`, `navbar.tmpl`) |
| ⌘K modal | 0 | 2 (`templates/base/head_navbar.tmpl`, `search-modal.js`) |
| Service call sites | 0 | ~9 files (replace ~30 PublishHackforgerAction calls → notify_service) |
| i18n | 0 | 2 (`locale_en-US.ini`, `locale_zh-CN.ini`) |
| Config | 0 | 1 (`app.ini` template) |
| Init | 0 | 1 (startup registration) |
| **Total** | **~7** | **~30** |

---

## 8. Execution Order

```
Phase 1: Notifier Unification
├── Extend notify.Notifier interface (+4 methods)
├── Add NullNotifier + dispatch implementations
├── Refactor hackforgerNotifier (Feed only, remove webhook)
├── Move webhook dispatch to webhookNotifier
├── Replace ~30 PublishHackforgerAction call sites
└── Verify: all existing tests pass

Phase 2: HackForger Indexer
├── Create modules/indexer/hackforger/ (internal + bleve + db)
├── Add indexerNotifier HackForger methods
├── Register indexer initialization at startup
├── Populate index with existing data
└── Verify: indexer returns correct results

Phase 3: Aggregated Search + Modal
├── Rewrite services/hackforger/search.go (unified search)
├── Update API/web search routes (GroupedSearchResult)
├── Redesign ⌘K modal (grouped, view-all links)
└── Verify: search returns mixed results

Phase 4: Explore Submissions
├── Handler + template + navbar tab
├── i18n keys
└── Verify: submissions browseable

Phase 5: E2E
├── Write E2E prompt: `docs/tests/e2e/unified-search-e2e-prompt.md`
├── Rebuild server
├── Execute search E2E with new grouped results
├── Test Explore Submissions page
└── Update E2E report with screenshots
```
