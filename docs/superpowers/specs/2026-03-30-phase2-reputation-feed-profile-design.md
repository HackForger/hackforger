# Phase 2: Reputation + Feed + Profile — Design Spec

> **Branch:** `feat/phase2-reputation-feed-profile`
> **Approach:** Bottom-Up (Model → Service → Notifier refactor → API → Web → Templates → E2E)
> **Context:** Week4 Line-B of implementation-plan-draft.md. Feed architecture redesigned — separate `hackforger_action` table instead of reusing Forgejo's `action` table.

---

## 0. Key Architecture Decision: Feed Separation

**Problem:** The original plan reused Forgejo's `action` table for HackForger events. This table is repo-centric — `GetFeeds()` filters by `AccessibleRepoIDsQuery`, which excludes HackForger events with `repo_id=0` (hackathons, grants, credits). Hacking around this requires modifying Forgejo's core `action.go`.

**Decision:** Separate the two feed systems entirely.

| Concern | Forgejo `action` table | New `hackforger_action` table |
|---------|----------------------|------------------------------|
| Events | Commits, PRs, issues, stars | Hackathon, bounty, grant, credits |
| Scope | Repo-centric (`repo_id` required) | Entity-centric (`entity_type` + `entity_id`) |
| ACL | Repo access control | Own audience logic (global/followers/org/watchers) |
| Query | `GetFeeds()` — untouched | New `GetHackforgerFeeds()` |
| Display | Dashboard "Code" tab, Profile "Activity" tab | Dashboard "Community" tab, Profile "Community" tab |

**Benefits:**
- Zero changes to Forgejo's `action.go` / `GetFeeds()`
- Entity metadata as first-class columns (indexed, no JSON parsing for queries)
- Independent pagination per tab (no cross-table merge needed)
- Lower upstream merge cost

**Trade-off:** One new table (overrides the original plan's "zero new tables for feed").

---

## 1. Data Layer

### 1.1 `hackforger_action` Table (NEW)

```go
// models/hackforger/hackforger_action.go

type HackforgerAction struct {
    ID          int64              `xorm:"pk autoincr"`
    UserID      int64              `xorm:"NOT NULL INDEX(idx_user_created)"`
    ActUserID   int64              `xorm:"NOT NULL INDEX(idx_actuser_created)"`
    OpType      activities_model.ActionType `xorm:"NOT NULL"` // reuses ActionType (int alias) from models/activities
    EntityType  string             `xorm:"VARCHAR(20) NOT NULL INDEX(idx_entity)"`
    EntityID    int64              `xorm:"NOT NULL DEFAULT 0"`
    EntityName  string             `xorm:"VARCHAR(255)"`
    EntitySlug  string             `xorm:"VARCHAR(255)"`
    OrgID       int64              `xorm:"DEFAULT 0"`
    RepoID      int64              `xorm:"DEFAULT 0"`
    Content     string             `xorm:"TEXT"`
    CreatedUnix timeutil.TimeStamp `xorm:"INDEX(idx_user_created) INDEX(idx_actuser_created) INDEX(idx_entity) created"`
}
```

**Indexes** (composite, matching query patterns):
- `idx_user_created`: `(user_id, created_unix)` — Dashboard Community tab: "events visible to me"
- `idx_actuser_created`: `(act_user_id, created_unix)` — Profile Community tab: "events by this user"
- `idx_entity`: `(entity_type, entity_id, created_unix)` — Entity timeline: "all events for hackathon #42"

**Column rationale:**
- `UserID`: audience target (0 = global, same convention as Forgejo's `action.user_id`)
- `EntityType`: "hackathon" / "bounty" / "grant" / "credits" — first-class column enables indexed queries without JSON parsing
- `Content`: JSON for extended data (phase change details, winner name, etc.) — uses existing `HackforgerActionContent` / `HackforgerPhaseContent` structs
- `OrgID`: hackathon's linked org, used for org-context queries
- `RepoID`: optional, populated for bounty events (bounty → issue → repo)

**Query functions** (in same file):

```go
type GetHackforgerFeedsOptions struct {
    db.ListOptions
    UserID      int64  // filter by audience (Dashboard: current user)
    ActUserID   int64  // filter by actor (Profile: context user)
    EntityType  string // filter by entity type
    EntityID    int64  // filter by specific entity
    IncludeGlobal bool // include UserID=0 records
}

func GetHackforgerFeeds(ctx context.Context, opts GetHackforgerFeedsOptions) ([]*HackforgerAction, int64, error)
func GetEntityTimeline(ctx context.Context, entityType string, entityID int64, opts db.ListOptions) ([]*HackforgerAction, int64, error)
```

**ActUser loading:** `HackforgerAction` will have a transient `ActUser *user_model.User` field (xorm:"-"). A `LoadActUsers(ctx, actions)` function bulk-loads users for a slice of actions — same pattern used by `ActionList.LoadActUsers()` in Forgejo.

### 1.2 `hackforger_setting` Table (NEW)

```go
// models/hackforger/setting.go

type HackforgerSetting struct {
    ID    int64  `xorm:"pk autoincr"`
    Key   string `xorm:"VARCHAR(100) UNIQUE NOT NULL"`
    Value string `xorm:"TEXT NOT NULL"`
}
```

**Functions:**
```go
func GetSetting(ctx context.Context, key string) (string, error)
func SetSetting(ctx context.Context, key, value string) error
func GetSettingWithDefault(ctx context.Context, key, defaultValue string) string
```

**Default seed data** (inserted by migration if not exists):

| Key | Default Value |
|-----|---------------|
| `reputation.weights` | `{"stars":1,"bounties_completed":5,"hackathon_wins":10,"grants_received":3,"credits_earned":0.1}` |
| `reputation.tiers` | `[{"name":"Bronze","min":0},{"name":"Silver","min":50},{"name":"Gold","min":200},{"name":"Diamond","min":500}]` |

### 1.3 `Reputation` Model Update

The existing `models/hackforger/reputation.go` needs one new field:

```go
type Reputation struct {
    // ... existing fields ...
    Tier string `xorm:"VARCHAR(20) NOT NULL DEFAULT 'Bronze'"` // NEW: cached tier label
}
```

The `Tier` field is denormalized — computed from `Score` + tier thresholds during cron recalculation. This avoids loading settings on every profile page view.

### 1.4 Migration

A single migration file in `models/forgejo_migrations/`:
1. Create `hackforger_action` table
2. Create `hackforger_setting` table
3. Add `tier` column to `reputation` table (XORM default name for `Reputation` struct)
4. Seed default settings

### 1.5 Cleanup: Existing `action` Table Data

Current `PublishHackforgerAction` writes to Forgejo's `action` table. After this refactor:
- Migration does NOT move existing records (dev environment, no production data)
- `PublishHackforgerAction` switches to `hackforger_action`
- Orphaned HackForger records in `action` table are harmless (rendered as unknown type, or can be cleaned up manually)

---

## 2. Feed System

### 2.1 Notifier Refactor

`services/hackforger/notifier.go` — `PublishHackforgerAction` changes:

**Before:** Writes to `activities_model.Action` (Forgejo's `action` table)
**After:** Writes to `hackforger_model.HackforgerAction` (new `hackforger_action` table)

```go
func PublishHackforgerAction(ctx context.Context, opts *HackforgerActionOpts) error {
    // Build base record from opts
    base := &hackforger_model.HackforgerAction{
        ActUserID:   opts.ActUserID,
        OpType:      opts.OpType,
        EntityType:  opts.EntityType,   // NEW: explicit field
        EntityID:    opts.EntityID,     // NEW: explicit field
        EntityName:  opts.EntityName,   // NEW: explicit field
        EntitySlug:  opts.EntitySlug,   // NEW: explicit field
        OrgID:       opts.OrgID,
        RepoID:      opts.RepoID,
        Content:     marshalContent(opts.Content),
    }

    // Audience resolution (same 4 strategies, writes to hackforger_action)
    // ... same logic, different target table ...
}
```

**HackforgerActionOpts** update — add explicit entity fields:
```go
type HackforgerActionOpts struct {
    ActUserID    int64
    OpType       activities_model.ActionType
    EntityType   string  // NEW
    EntityID     int64   // NEW
    EntityName   string  // NEW
    EntitySlug   string  // NEW
    RepoID       int64
    Content      any
    AudienceType AudienceType
    OrgID        int64
}
```

**Fix pre-existing AudienceType bug:** The current `AudienceType` uses sequential `iota` (0,1,2,3), but callers in `bounty.go` use bitwise OR (`AudienceFollowers | AudienceRepoWatchers`). This silently collapses to a single audience. Fix during this refactor:
```go
const (
    AudienceGlobal       AudienceType = 1 << iota  // 1
    AudienceFollowers                                // 2
    AudienceOrgMembers                               // 4
    AudienceRepoWatchers                             // 8
)
```
Change the `switch` in `PublishHackforgerAction` to `if opts.AudienceType&AudienceX != 0` flag checks.

**All existing callers** in `services/hackforger/bounty.go`, `hackathon.go`, `grants.go`, `credits.go` must be updated to pass the new entity fields. The Content struct is still used for extended data (phase changes, etc.) but entity metadata is no longer solely in Content.

### 2.2 Feed Query Layer

`models/hackforger/hackforger_action.go` provides:

**Dashboard Community tab:** `GetHackforgerFeeds` with `UserID = currentUser.ID` and `IncludeGlobal = true`
- SQL: `WHERE (user_id = ? OR user_id = 0) ORDER BY created_unix DESC`

**Profile Community tab:** `GetHackforgerFeeds` with `ActUserID = profileUser.ID`
- SQL: `WHERE act_user_id = ? ORDER BY created_unix DESC`

**Entity timeline (future):** `GetEntityTimeline` with `EntityType + EntityID`
- SQL: `WHERE entity_type = ? AND entity_id = ? ORDER BY created_unix DESC`

### 2.3 Feed API Update

`routers/api/v1/hackforger/feed.go` — `GetFeed()` switches from querying `action` to `hackforger_action`:

```
GET /api/v1/hackforger/feed
    ?type=global|following|entity|user
    &entity_type=hackathon&entity_id=42
    &user_id=10
    &page=1&limit=20
```

- `global`: `UserID = 0` records only
- `following`: `UserID = doer.ID OR UserID = 0` — leverages pre-distributed audience records (the notifier already inserts rows for each follower), plus global events. This is the same query as the Dashboard Community tab.
- `entity`: `EntityType + EntityID` filter (deduplicated: `GROUP BY` or `DISTINCT` on action content, since audience distribution creates multiple rows per event)
- `user`: `ActUserID = user_id` filter (deduplicated similarly)

**Note on deduplication:** For `entity` and `user` API modes, audience distribution means the same logical event may have multiple rows (one per audience member). These modes should deduplicate by grouping on `(act_user_id, op_type, entity_type, entity_id, created_unix)`.

Response format unchanged (same `feedItem` struct).

### 2.4 Template Cleanup

**Remove from `templates/user/dashboard/feeds.tmpl`:**
All 22 HackForger `.InActions` branches (lines 83-127 of current file). These will be rendered by a separate template.

**Keep in `feeds.tmpl`:** All Forgejo-native action rendering (commits, PRs, issues, etc.).

---

## 3. Reputation System

### 3.1 Service Layer

`services/hackforger/reputation.go` (NEW file):

```go
// RecalculateReputation recomputes a single user's reputation score and tier.
func RecalculateReputation(ctx context.Context, userID int64) error {
    // 1. Load or create reputation record
    // 2. Count metrics from DB (using actual XORM table/column names):
    //    - BountiesCompleted: COUNT(*) FROM bounty WHERE claimer_id=? AND status IN (3,4)
    //      (BountyStatusCompleted=3, BountyStatusPaid=4)
    //    - HackathonWins: COUNT(*) FROM hackathon_submission WHERE user_id=? AND rank=1
    //    - GrantsReceived: COUNT(*) FROM grant_project WHERE user_id=? AND status IN (1,3)
    //      (GrantProjectStatusApproved=1, GrantProjectStatusFunded=3)
    //    - TotalStars: SUM(num_stars) FROM repository WHERE owner_id=?
    //    - TotalCreditsEarned: SUM(amount) FROM credit_transaction
    //      WHERE user_id=? AND type IN ('deposit','reward')
    // 3. Load weights from hackforger_setting
    // 4. Compute Score = sum(metric × weight)  (float64, then round to int64)
    // 5. Derive Tier from Score + tier thresholds
    // 6. Update reputation record (Score, Tier, and all metric fields)
}

// RecalculateAllReputations batch-recalculates all users with reputation records.
func RecalculateAllReputations(ctx context.Context) error

// GetReputationWeights loads weights from hackforger_setting.
func GetReputationWeights(ctx context.Context) (map[string]float64, error)

// GetReputationTiers loads tier thresholds from hackforger_setting.
func GetReputationTiers(ctx context.Context) ([]ReputationTier, error)

type ReputationTier struct {
    Name string  `json:"name"`
    Min  float64 `json:"min"`
}
```

### 3.2 Cron Task

A cron skeleton already exists in `services/cron/tasks_hackforger.go`. Fill in the existing stub with the actual call:

```go
// In services/cron/tasks_hackforger.go — existing skeleton:
RegisterTaskFatal("hackforger_reputation_recalc", &OlderThanConfig{
    BaseConfig: BaseConfig{Enabled: true, RunAtStart: false, Schedule: "@every 1h"},
    OlderThan:  0,
}, func(ctx context.Context, _ *user_model.User, _ Config) error {
    return hackforger_service.RecalculateAllReputations(ctx)
})
```

### 3.3 Admin Web Routes

```
GET  /admin/hackforger/reputation          — Config page (weights + tiers)
POST /admin/hackforger/reputation          — Save config
POST /admin/hackforger/reputation/recalc   — Trigger manual recalculation
```

**Router registration:** Add to the admin group in `web.go` (alongside existing HackForger Admin Credits routes).

**Template:** `templates/admin/hackforger/reputation.tmpl` — form with JSON editors for weights and tiers.

### 3.4 API Endpoints

Add to `/api/v1/hackforger/` group:

```
GET  /reputation/users/{username}         — User reputation + tier
GET  /reputation/leaderboard              — Top N (default 50)
POST /reputation/recalculate/{username}   — Admin: force recalc
```

**Router:** `routers/api/v1/hackforger/reputation.go` (NEW file)

---

## 4. Dashboard Integration

### 4.1 Tab Bar

Add a tab bar between heatmap and feed content in `templates/user/dashboard/dashboard.tmpl`:

```html
{{template "user/heatmap" .}}
<div class="ui secondary pointing menu">
    <a class="{{if ne .FeedType "community"}}active {{end}}item" href="?feed=code">
        {{svg "octicon-code"}} {{ctx.Locale.Tr "hackforger.feed.code"}}
    </a>
    <a class="{{if eq .FeedType "community"}}active {{end}}item" href="?feed=community">
        {{svg "octicon-people"}} {{ctx.Locale.Tr "hackforger.feed.community"}}
    </a>
</div>
{{if eq .FeedType "community"}}
    {{if .HackforgerFeeds}}
        {{template "hackforger/feed/community_feeds" .}}
    {{else}}
        <div class="tw-py-8 tw-text-center text grey">
            {{ctx.Locale.Tr "hackforger.feed.community_empty"}}
        </div>
    {{end}}
{{else}}
    {{if .Feeds}}
        {{template "user/dashboard/feeds" .}}
    {{else}}
        {{template "user/dashboard/guide" .}}
    {{end}}
{{end}}
```

### 4.2 Router Changes

`routers/web/user/home.go` — `Dashboard()` function:

```go
feedType := ctx.FormString("feed")
if feedType == "" {
    feedType = "code"
}
ctx.Data["FeedType"] = feedType

if feedType == "community" {
    // Query hackforger_action table
    feeds, count, err := hackforger_model.GetHackforgerFeeds(ctx, hackforger_model.GetHackforgerFeedsOptions{
        UserID:        ctxUser.ID,
        IncludeGlobal: true,
        ListOptions:   db.ListOptions{Page: page, PageSize: setting.UI.FeedPagingNum},
    })
    // ... LoadActUsers, set ctx.Data["HackforgerFeeds"], pagination ...
} else {
    // Existing GetFeeds call (unchanged)
}
```

---

## 5. Profile Integration

### 5.1 Reputation Sidebar Card

Inject after the avatar section in `templates/user/profile.tmpl` (within the `four wide column`):

```html
<div class="ui four wide column">
    {{template "shared/user/profile_big_avatar" .}}
    {{if .Reputation}}
        {{template "hackforger/reputation/card" .}}
    {{end}}
</div>
```

**`templates/hackforger/reputation/card.tmpl`** (NEW):
- Tier badge (colored: Bronze/Silver/Gold/Diamond)
- Numeric score
- Top 3 non-zero metrics (e.g., "5 bounties, 2 hackathon wins, 120 stars")
- Link to full Reputation tab

### 5.2 New Tabs on Profile Header

Add to `templates/user/overview/header.tmpl`:

```html
<!-- After the "activity" tab -->
<a class="{{if eq .TabName "community"}}active {{end}}item"
   href="{{.ContextUser.HomeLink}}?tab=community">
    {{svg "octicon-people"}} {{ctx.Locale.Tr "hackforger.profile.community"}}
</a>
<a class="{{if eq .TabName "reputation"}}active {{end}}item"
   href="{{.ContextUser.HomeLink}}?tab=reputation">
    {{svg "octicon-star"}} {{ctx.Locale.Tr "hackforger.profile.reputation"}}
</a>
```

### 5.3 Profile Tab Content

In `templates/user/profile.tmpl`, add:

```html
{{else if eq .TabName "community"}}
    {{template "hackforger/feed/community_feeds" .}}
{{else if eq .TabName "reputation"}}
    {{template "hackforger/reputation/detail" .}}
```

In `routers/web/user/profile.go` — `prepareUserProfileTabData()`:

**Before the `switch tab` block**, load reputation unconditionally (so sidebar card appears on all tabs):
```go
// Load reputation for sidebar card (all tabs)
if rep, err := hackforger_model.GetOrCreateReputation(ctx, ctx.ContextUser.ID); err == nil {
    ctx.Data["Reputation"] = rep
}
```

**Add tab cases:**
```go
case "community":
    feeds, count, err := hackforger_model.GetHackforgerFeeds(ctx, hackforger_model.GetHackforgerFeedsOptions{
        ActUserID: ctx.ContextUser.ID,
        ListOptions: db.ListOptions{Page: page, PageSize: setting.UI.FeedPagingNum},
    })
    // ... LoadActUsers, set ctx.Data["HackforgerFeeds"], pagination ...

case "reputation":
    // Reputation already loaded above for sidebar
    // Load additional detail: weights breakdown, rank
```

### 5.4 Reputation Detail Template

**`templates/hackforger/reputation/detail.tmpl`** (NEW):
- Large tier badge + score
- Full metrics table with values and weights
- Rank (position in leaderboard)
- Link to global leaderboard

### 5.5 Web Route: Leaderboard Page

```
GET /explore/reputation — Public leaderboard page
```

**Template:** `templates/hackforger/reputation/leaderboard.tmpl`
**Router:** Add to `web.go` explore group (inherits `ignExploreSignIn` middleware, consistent with `/explore/hackathons`, `/explore/bounties`, `/explore/grants`)

---

## 6. Community Feeds Template

### 6.1 `templates/hackforger/feed/community_feeds.tmpl` (NEW)

Renders `HackforgerAction` records. Unlike `feeds.tmpl` which uses Forgejo's `Action` model methods (`.GetRepoLink`, `.ShortRepoPath`), this template uses explicit fields from `HackforgerAction`.

**Structure:**
```html
<div id="community-feed" class="flex-list">
{{range .HackforgerFeeds}}
    <div class="flex-item">
        <div class="flex-item-leading">
            {{ctx.AvatarUtils.Avatar .ActUser}}
        </div>
        <div class="flex-item-main tw-gap-2">
            <div>
                <a href="{{AppSubUrl}}/{{.ActUser.Name}}">{{.ActUser.GetDisplayName}}</a>
                <!-- Event-specific rendering by OpType -->
                {{if eq .OpType 30}}
                    {{ctx.Locale.Tr "hackforger.feed.hackathon_created" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
                {{else if eq .OpType 31}}
                    ...
                {{end}}
                {{DateUtils.TimeSince .CreatedUnix}}
            </div>
        </div>
        <div class="flex-item-trailing">
            {{svg (HackforgerActionIcon .OpType) 32 "text grey tw-mr-1"}}
        </div>
    </div>
{{end}}
{{template "base/paginate" .}}
</div>
```

### 6.2 Template Helper Functions

Register in `modules/templates/helper.go` → `NewFuncMap()`:

```go
// HackforgerEntityURL generates the URL for a HackForger entity.
func HackforgerEntityURL(entityType, slug string) string {
    switch entityType {
    case "hackathon": return setting.AppSubURL + "/hackathon/" + url.PathEscape(slug)
    case "grant":     return setting.AppSubURL + "/grants/" + url.PathEscape(slug)
    case "bounty":    return "" // bounty URL requires repo context, handled differently
    default:          return ""
    }
}

// HackforgerActionIcon returns the octicon name for a HackForger action type.
func HackforgerActionIcon(opType int) string {
    // Map action types to icons
}
```

---

## 7. Forgejo File Modifications (Minimal)

| File | Change | Lines |
|------|--------|-------|
| `routers/web/user/home.go` | Add `?feed=` param, community query branch | ~20 |
| `routers/web/user/profile.go` | Add "community" + "reputation" tab cases | ~25 |
| `routers/web/web.go` | Add admin reputation routes + leaderboard route | ~10 |
| `routers/api/v1/api.go` | Add reputation API routes | ~5 |
| `templates/user/dashboard/dashboard.tmpl` | Add tab bar | ~10 |
| `templates/user/dashboard/feeds.tmpl` | Remove 22 HackForger branches | -45 |
| `templates/user/profile.tmpl` | Add community + reputation tab content | ~10 |
| `templates/user/overview/header.tmpl` | Add 2 tab links | ~8 |
| `services/cron/tasks_extended.go` | Register reputation cron | ~6 |
| `modules/templates/helper.go` | Register `HackforgerEntityURL`, `HackforgerActionIcon` | ~5 |

**Total Forgejo-file changes: ~11 files, ~150 lines added, ~45 lines removed.**

---

## 8. New Files Summary

| File | Purpose |
|------|---------|
| `models/hackforger/hackforger_action.go` | HackforgerAction model + query functions |
| `models/hackforger/setting.go` | HackforgerSetting model + CRUD |
| `services/hackforger/reputation.go` | Score calculation, cron logic |
| `routers/api/v1/hackforger/reputation.go` | Reputation API endpoints |
| `routers/web/hackforger/reputation.go` | Admin config + leaderboard web routes |
| `templates/hackforger/feed/community_feeds.tmpl` | Community feed rendering |
| `templates/hackforger/reputation/card.tmpl` | Sidebar card |
| `templates/hackforger/reputation/detail.tmpl` | Full reputation page |
| `templates/hackforger/reputation/leaderboard.tmpl` | Leaderboard page |
| `templates/admin/hackforger/reputation.tmpl` | Admin config page |
| `models/forgejo_migrations/v_xxx.go` | Migration file |

---

## 9. i18n Keys

Both `options/locale/locale_en-US.ini` and `options/locale/locale_zh-CN.ini` under `[hackforger]`:

**Feed tabs:**
- `hackforger.feed.code` = Code Activity / 代码动态
- `hackforger.feed.community` = Community / 社区动态

**Profile tabs:**
- `hackforger.profile.community` = Community / 社区
- `hackforger.profile.reputation` = Reputation / 声誉

**Feed event messages** (22 keys, pattern: `hackforger.feed.<event_name>`):
- `hackforger.feed.hackathon_created` = created hackathon %s / 创建了 Hackathon %s
- `hackforger.feed.hackathon_registered` = registered for %s / 报名了 %s
- ... (all 22 active event types; `milestone` (type 60) excluded — reserved for future use)

**Empty states:**
- `hackforger.feed.community_empty` = No community activity yet / 暂无社区动态

**Reputation:**
- `hackforger.reputation.score` = Score / 声誉分
- `hackforger.reputation.tier` = Tier / 等级
- `hackforger.reputation.bounties_completed` = Bounties Completed / 完成的悬赏
- `hackforger.reputation.hackathon_wins` = Hackathon Wins / Hackathon 获奖
- `hackforger.reputation.grants_received` = Grants Received / 获得的资助
- `hackforger.reputation.total_stars` = Stars / 收获的 Star
- `hackforger.reputation.credits_earned` = Credits Earned / 累计获得积分
- `hackforger.reputation.leaderboard` = Reputation Leaderboard / 声誉排行榜
- `hackforger.reputation.rank` = Rank / 排名

**Admin:**
- `hackforger.admin.reputation` = Reputation Settings / 声誉设置
- `hackforger.admin.reputation.weights` = Score Weights / 评分权重
- `hackforger.admin.reputation.tiers` = Tier Thresholds / 等级阈值
- `hackforger.admin.reputation.recalc` = Recalculate All / 重新计算
- `hackforger.admin.reputation.recalc_success` = Recalculation triggered / 已触发重新计算

---

## 10. Testing Strategy

### Unit Tests
- `models/hackforger/hackforger_action_test.go` — CRUD + query functions
- `models/hackforger/setting_test.go` — Get/Set/Default
- `services/hackforger/reputation_test.go` — Score calculation with known weights

### Integration Tests
- Notifier → hackforger_action write path
- Dashboard Community tab renders HackForger events
- Profile Community + Reputation tabs render correctly
- Admin config save + recalculate

### E2E Test (Manual)
Covered by separate E2E prompt document (see `docs/tests/e2e/` after implementation).
