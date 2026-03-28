# Phase 1 Bounty — Design Spec

> **Branch:** `feat/phase1-bounty`
> **Approach:** Bottom-Up (Model → Service → Notifier → API → Web → Vue → E2E)
> **Parallel context:** Hackathon and Grant lines developed by two colleagues in separate worktrees. Bounty line owns Notifier audience resolution as shared infrastructure.

---

## 1. Model Corrections + Migration

### 1.1 BountyStatus Expansion (4 → 7 states)

P0 defined 4 states. Phase 1 expands to 7 to match the spec's state machine:

```go
BountyStatusOpen       BountyStatus = 0
BountyStatusClaimed    BountyStatus = 1  // was InProgress
BountyStatusInReview   BountyStatus = 2  // new
BountyStatusCompleted  BountyStatus = 3
BountyStatusPaid       BountyStatus = 4  // new
BountyStatusExpired    BountyStatus = 5  // new
BountyStatusCancelled  BountyStatus = 6  // was 3, moved to 6
```

State machine:

```
Exclusive:  Open → Claimed → InReview → Completed → Paid
                               ↓ (reject) → Claimed (redo)

Competitive: Open → InReview → Completed → Paid

Common: Open/Claimed → Expired (cron, deadline passed)
        Open/Claimed → Cancelled (publisher)
```

### 1.2 BountyMode Correction (3 → 2 modes)

```go
BountyModeExclusive   BountyMode = 0  // was FirstCome
BountyModeCompetitive BountyMode = 1  // was Application
// Invitation(2) removed
```

### 1.3 Migration

Add a migration in `models/forgejo_migrations/` to remap old status value 3 (Cancelled) → 6. P0 tables have no real data, but the migration ensures correctness for any instance that ran P0.

### 1.4 CRUD Completion

P0 left Application, Winner, and Reward models with only struct + error types. Phase 1 adds:

**BountyApplication:**
- `CreateBountyApplication(ctx, app) error`
- `GetBountyApplicationByID(ctx, id) (*BountyApplication, error)`
- `ListBountyApplications(ctx, opts) ([]*BountyApplication, int64, error)`
- `UpdateBountyApplication(ctx, app) error`

**BountyWinner:**
- `CreateBountyWinner(ctx, w) error`
- `ListBountyWinners(ctx, bountyID) ([]*BountyWinner, error)`

**BountyReward:**
- `CreateBountyReward(ctx, r) error`
- `ListBountyRewards(ctx, bountyID) ([]*BountyReward, error)`
- `DeleteBountyReward(ctx, id) error`

---

## 2. Service Layer — Bounty State Machine

### 2.1 New file: `services/hackforger/bounty.go`

**State transition matrix:**

| Current | Allowed Transition | Trigger | Condition |
|---------|-------------------|---------|-----------|
| Open | Claimed | AcceptApplication | Exclusive mode |
| Open | InReview | StartReview (manual) | Competitive mode |
| Open | Expired | Cron | `Deadline > 0 && now > Deadline` |
| Open | Cancelled | Publisher | CancelBounty |
| Claimed | InReview | MergePullRequest hook | PR author == ClaimerID |
| Claimed | Cancelled | Publisher | CancelBounty |
| Claimed | Expired | Cron | deadline passed |
| InReview | Completed | Publisher | CompleteBounty |
| InReview | Claimed | Publisher | RejectDelivery (Exclusive only) |
| Completed | Paid | Publisher | MarkPaid |

**Core functions:**

```
ApplyForBounty(ctx, bountyID, userID, message) error
  - Validate: Bounty.Status == Open, no duplicate application
  - Create BountyApplication(Pending)

AcceptApplication(ctx, applicationID, doerID) error
  - Validate: doer == Publisher
  - Exclusive: accept one → reject all other Pending → Bounty.Status=Claimed, set ClaimerID
  - Competitive: accept (no Bounty status change, multiple accepted)

RejectApplication(ctx, applicationID, doerID) error
  - Validate: doer == Publisher
  - Set application.Status = Rejected

CompleteBounty(ctx, bountyID, doerID) error
  - Validate: doer == Publisher, Status == InReview
  - Bounty → Completed
  - If credits-type Reward exists: credits.Deposit(claimerID, amount)
  - Publish ActionBountyCompleted

RejectDelivery(ctx, bountyID, doerID) error
  - Exclusive only, InReview → Claimed

SelectWinners(ctx, bountyID, doerID, winners []WinnerInput) error
  - Competitive only, Status == InReview
  - Create BountyWinner records, match Rank to Reward for credits deposit
  - Bounty → Completed
  - Publish ActionBountyWinnersSelected

MarkPaid(ctx, bountyID, doerID) error
  - Completed → Paid

CancelBounty(ctx, bountyID, doerID) error
  - Open/Claimed → Cancelled

CheckExpiredBounties(ctx) error
  - Cron: batch process bounties past deadline
```

### 2.2 MergePullRequest Hook

In `notifier.go`, flesh out `MergePullRequest`:

1. Get Issue from PR
2. `GetBountyByIssueID` → return if none
3. Check `Bounty.Status == Claimed`
4. Check `doer.ID == Bounty.ClaimerID`
5. If match → `Bounty.Status = InReview`, `UpdateBounty`
6. `PublishHackforgerAction(ActionBountyDelivered)`

---

## 3. Notifier Audience Resolution (Shared Infrastructure)

### 3.1 Bitmask AudienceType

Replace P0's iota enum with bitmask for composability:

```go
type AudienceType int

const (
    AudienceGlobal       AudienceType = 1 << 0  // 1
    AudienceFollowers    AudienceType = 1 << 1  // 2
    AudienceOrgMembers   AudienceType = 1 << 2  // 4
    AudienceRepoWatchers AudienceType = 1 << 3  // 8
)
```

### 3.2 PublishHackforgerAction Enhancement

```
PublishHackforgerAction(ctx, opts):
  1. Serialize Content → JSON
  2. Insert actor's own action record (existing)
  3. Collect target UserIDs via map[int64]bool for dedup:

     if opts.AudienceType & AudienceGlobal:
       → Insert UserID=0 record

     if opts.AudienceType & AudienceFollowers:
       → SELECT follower_id FROM user_follow WHERE follow_id = ActUserID
       → Add all to target set

     if opts.AudienceType & AudienceOrgMembers:
       → SELECT uid FROM org_user WHERE org_id = opts.OrgID
       → Add all to target set

     if opts.AudienceType & AudienceRepoWatchers:
       → SELECT user_id FROM watch WHERE repo_id = opts.RepoID AND mode != 2
       → Add all to target set

  4. Remove ActUserID from target set (already inserted in step 2)
  5. Batch insert action records for all collected UserIDs
```

### 3.3 Bounty Event → Audience Mapping

| Event | ActionType | AudienceType |
|-------|-----------|-------------|
| Bounty created | 34 | `Global \| RepoWatchers` |
| Bounty claimed | 35 | `Followers \| RepoWatchers` |
| Bounty delivered (PR merge) | 36 | `Followers \| RepoWatchers` |
| Bounty completed | 37 | `Followers \| RepoWatchers` |
| Winners selected | 38 | `Global` |
| Bounty expired | 52 | `RepoWatchers` |
| Bounty cancelled | 53 | `RepoWatchers` |
| Mark paid | — | No feed event |

---

## 4. API Layer

### 4.1 Route Structure

**Repo-level** (`/api/v1/repos/{owner}/{repo}/bounties`):

```
POST   /                          → CreateBounty
GET    /                          → ListRepoBounties
GET    /{id}                      → GetBounty
POST   /{id}/rewards              → AddReward
GET    /{id}/rewards              → ListRewards
DELETE /{id}/rewards/{rid}        → DeleteReward
POST   /{id}/applications         → ApplyForBounty
GET    /{id}/applications         → ListApplications
PUT    /{id}/applications/{aid}   → ReviewApplication
POST   /{id}/complete             → CompleteBounty
POST   /{id}/pay                  → MarkPaid
POST   /{id}/cancel               → CancelBounty
POST   /{id}/winners              → SelectWinners
GET    /{id}/winners              → ListWinners
```

**Global** (`/api/v1/hackforger/bounties`):

```
GET    /                          → ListAllBounties
GET    /stats                     → BountyStats
GET    /leaderboard               → HunterLeaderboard
```

### 4.2 Permission Model

| Endpoint | Auth | Permission |
|----------|------|-----------|
| GET list/detail | Optional | Public bounties readable without login |
| POST create | Required | Repo Writer or Owner |
| POST apply | Required | Any logged-in user |
| PUT review / POST complete/pay/cancel/winners | Required | `doer.ID == bounty.PublisherID` |
| DELETE reward | Required | Publisher, Bounty.Status == Open |

### 4.3 Conventions

- Pagination: `?page=1&limit=20`, response header `X-Total-Count`
- Filters: `?status=&repo_id=&q=&sort=&order=`
- Errors: `{"message": "...", "url": "..."}` + HTTP status
- Create returns 201 + entity JSON
- All endpoints have Swagger annotations

### 4.4 File

Single file `routers/api/v1/hackforger/bounty.go` (~600 lines). Replace P0 skeleton.

---

## 5. Web Routes + Templates

### 5.1 Web Routes

```
/explore/bounties                    → ExploreBounties (replace P0 placeholder)
/repos/{owner}/{repo}/bounties/new   → NewBounty (form page, GET)
/repos/{owner}/{repo}/bounties/new   → NewBountyPost (form submit, POST)
```

No standalone Bounty detail page — Bounty details live inside the Issue page via panel injection.

### 5.2 Template Files

```
templates/hackforger/bounty/
  explore.tmpl    — Full-platform Bounty list with filters
  new.tmpl        — Create Bounty form (select Issue, Mode, Deadline)
  panel.tmpl      — Issue detail page embedded panel (SSR base for Vue)
  badge.tmpl      — Issue list Bounty badge
```

### 5.3 Forgejo Template Injections (2 changes)

**① `templates/repo/issue/view_content.tmpl`** (~3 lines)

Inject Bounty Panel in Issue sidebar:
```html
{{if .BountyData}}
  {{template "hackforger/bounty/panel" .}}
{{end}}
```

Requires adding ~5 lines in `routers/web/repo/issue.go` ViewIssue handler to query `GetBountyByIssueID` and pass result to template data.

**② `templates/repo/issue/list.tmpl`** (~3 lines)

Inject Bounty Badge after Issue title:
```html
{{if .BountyBadge}}
  {{template "hackforger/bounty/badge" .}}
{{end}}
```

Requires batch query `SELECT issue_id, status FROM bounty WHERE issue_id IN (...)` in Issue list handler.

### 5.4 Explore Bounties Page

SSR rendered (no Vue needed):
- Filter bar: Status (All/Open/Claimed/InReview/Completed), Sort (newest/amount/deadline)
- Bounty card list: title, repo, status label, reward amount, deadline, publisher avatar
- Pagination

### 5.5 BountyPanel (Issue Embedded)

Two layers:

**SSR base (`panel.tmpl`):**
- Status label (color-coded)
- Reward list
- Deadline
- Current Claimer (if any)

**Vue enhancement (`BountyPanel.vue`):**
- Apply button + modal (message input)
- Application list + accept/reject actions (Publisher view)
- Complete / Reject delivery buttons
- Select Winners form (Competitive)
- Local refresh on state change (no full page reload)

Mount: `panel.tmpl` renders `<div id="hackforger-bounty-panel" data-bounty-id="...">`, JS lazy-loads Vue component onto it.

---

## 6. Feed Integration

### 6.1 Event Publishing Points

Each Service function calls `PublishHackforgerAction` at the end. See Section 3.3 for the complete event → audience mapping.

### 6.2 No Feed Event for MarkPaid

Paid is an internal administrative state. No public feed event.

---

## 7. Testing

### 7.1 Unit Tests

**`models/hackforger/bounty_test.go`** — CRUD tests per test-plan-draft §2.2:
- CreateBounty, DuplicateIssue, CreateReward, Multiple Rewards
- CreateApplication, DuplicateApplication, GetByIssueID
- ListBounties filter, CreateWinner

**`services/hackforger/bounty_test.go`** — State machine tests per test-plan-draft §2.2:
- ExclusiveBountyFlow_Happy (create→claim→PR→merge→review→complete→credits)
- ExclusiveBounty_RejectAndRedo
- CompetitiveBountyFlow_Happy (multi-submit→winners→rank credits)
- Application_AcceptRejectsOthers
- BountyExpiry, BountyCancel, Cancel_NotPublisher
- BountyPRMerge_WrongUser, Complete_NoCreditsReward
- BountyLeaderboard

**`services/hackforger/notifier_test.go`** — Audience resolution tests per test-plan-draft §2.6:
- AudienceFollowers, AudienceOrgMembers, AudienceGlobal, AudienceRepoWatchers
- Bitmask combinations, dedup
- ContentSerialization, PhaseContentSerialization

### 7.2 Fixtures

Create YAML fixtures in `models/fixtures/`:
- `bounty.yml`, `bounty_reward.yml`, `bounty_application.yml`, `bounty_winner.yml`

### 7.3 E2E

Execute test-plan-draft §3.2:
- `TestE2E_Phase1_BountyExclusive` — full Exclusive flow with Feed verification at each step
- `TestE2E_Phase1_BountyCompetitive` — multi-submit, winners, ranked credits

### 7.4 Manual E2E Prompt

Produce `docs/tests/e2e/p1-bounty-e2e-prompt.md` following P0 format:
- Prerequisites (P0 infrastructure ready, users/orgs/repos from P0 E2E)
- Step-by-step operations (create Bounty → apply → claim → PR → merge → complete → credits)
- Feed verification checkpoints at each step
- Report template at `docs/tests/e2e/p1-bounty-e2e-report.md`

---

## 8. Changes to Shared/Upstream Files

| File | Change | Lines |
|------|--------|-------|
| `routers/api/v1/api.go` | Register Bounty repo-level + global routes | ~15 |
| `routers/web/web.go` | Register Bounty web routes | ~5 |
| `services/hackforger/notifier.go` | Bitmask audience + MergePullRequest hook | ~80 |
| `templates/repo/issue/view_content.tmpl` | Bounty Panel injection | ~3 |
| `templates/repo/issue/list.tmpl` | Bounty Badge injection | ~3 |
| `routers/web/repo/issue.go` | Query Bounty data in ViewIssue | ~5 |
| `options/locale/locale_en-US.ini` | Bounty i18n keys | ~30 |
| `web_src/js/features/hackforger/init.js` | BountyPanel lazy-load mount | ~10 |

### Conflict Strategy

- Route registrations: add only Bounty routes in clearly delimited blocks with comments
- `notifier.go`: Bounty line owns the full audience resolution rewrite; Hackathon/Grant lines call `PublishHackforgerAction` without modifying it
- `locale_en-US.ini`: append-only, low conflict risk
- Other two lines should NOT touch `view_content.tmpl`, `list.tmpl`, or `issue.go`

---

## 9. Implementation Order

```
Step 1: Model corrections + migration + CRUD completion
Step 2: Service layer (bounty.go state machine + credits integration)
Step 3: Notifier audience resolution (bitmask rewrite)
Step 4: MergePullRequest hook
Step 5: API handlers (replace P0 skeletons + new endpoints)
Step 6: Web routes + templates (explore, new, panel, badge)
Step 7: Vue BountyPanel component
Step 8: Unit tests + fixtures
Step 9: E2E tests
Step 10: Manual E2E prompt + report template
```
