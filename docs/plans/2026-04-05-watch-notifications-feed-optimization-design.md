# HF-022 Watch & Notifications + HF-024 Dashboard Feed Optimization

## Problem Statement

**HF-022**: Users who participate in a Hackathon, Bounty, or Grant receive no follow-up
notifications about milestone events (phase changes, status transitions, results).
The notification infrastructure exists (`publishFeed` with 5 audience types) but two
things are missing: (a) participants are not auto-subscribed and (b) some milestone
events lack the appropriate audience flags.

**HF-024**: The Dashboard sidebar (right panel) shows raw repo/org lists. Because each
Hackathon auto-creates an org + per-track repos, and each Bounty is bound to a repo,
the sidebar is cluttered with Hackathon-binding repos that mean nothing to the user.
The community feed tab needs a purpose-built sidebar showing "参与的黑客松/悬赏/赞助".

## Design Decisions

### No new tables

Both features reuse existing infrastructure:

- **Watch**: Forgejo's `watch` table (repo-level) + `org_user` table (org membership)
- **Feed**: Existing `hackforger_action` table + `publishFeed` audience flags
- **Sidebar data**: Existing `hackathon_registration`, `bounty_application`, `grant_project` tables

### Event importance tiers

Not all events should notify all watchers. Events are classified into two tiers:

| Tier | Who sees it | Examples |
|------|------------|---------|
| **Milestone** | All participants (RepoWatchers + OrgMembers) | Phase change, finalize, winners, bounty completed, grant awarded |
| **Activity** | Actor's followers only | User registered, user submitted, user scored |

This prevents feed noise — 100 registrations don't flood every participant's feed.

---

## HF-022: Watch & Notifications

### Part A: Auto-Subscribe on Participate

When a user participates in an entity, they become a "subscriber" through existing
Forgejo mechanisms:

| Action | Auto-subscription mechanism | Already done? |
|--------|---------------------------|--------------|
| Register for Hackathon | `AddOrgUser(ctx, h.LinkedOrgID, userID)` | **Yes** (hackathon.go:378) |
| Submit to Hackathon Track | `WatchRepo(ctx, userID, track.RepoID, true)` | **No — add** |
| Apply for Bounty | `WatchRepo(ctx, userID, bounty.RepoID, true)` | **No — add** |
| Claim Bounty | `WatchRepo(ctx, userID, bounty.RepoID, true)` | **No — add** |
| Submit Grant Project | `AddOrgUser(ctx, round.OrgID, doerID)` | **No — add** |

Key insight: Hackathon registration already adds users to the org (`AddOrgUser`), so
`AudienceOrgMembers` already covers them. For Bounty, we need explicit `WatchRepo`.
For Grants, applicants are NOT currently auto-added to the org — add `AddOrgUser` in
`SubmitProject` so `AudienceOrgMembers` covers grant lifecycle updates.

**Timing**: All `WatchRepo`/`AddOrgUser` calls must happen AFTER the DB record is
successfully created, not before, to avoid subscribing on validation failure.

### Part B: Audience Flag Audit

Every `publishFeed` call must use the correct audience for its tier.

#### Hackathon Events

| Event | Current Audience | Correct Audience | Change? |
|-------|-----------------|-----------------|---------|
| `HackathonCreated` | `Global` | `Global` | No |
| `HackathonRegistered` | `Followers` | `Followers` | No (activity tier) |
| `HackathonSubmitted` | `Followers` | `Followers` | No (activity tier) |
| `HackathonScored` | `Followers` | `Followers` | No (activity tier) |
| `HackathonPhaseChanged` | `OrgMembers` | `OrgMembers` | No ✓ |
| `HackathonFinalized` | `Global` | `Global` | No ✓ |

Hackathon events are already correct. `OrgMembers` covers all registered participants
because `RegisterPost` calls `AddOrgUser`.

#### Bounty Events

| Event | Current Audience | Correct Audience | Change? |
|-------|-----------------|-----------------|---------|
| `BountyCreated` | `Global + RepoWatchers` | `Global + RepoWatchers` | No ✓ (already published in web+API handlers) |
| `BountyClaimed` | `Followers + RepoWatchers` | `Followers + RepoWatchers` | No ✓ |
| `BountyCompleted` | `Followers + RepoWatchers` | `Followers + RepoWatchers` | No ✓ |
| `BountyWinnersSelected` | `Global` | `Global` | No ✓ |
| `BountyPaid` | `0` (actor only) | `DirectUser` (to recipient) | **Fix** |
| `BountyExpired` | `RepoWatchers` | `RepoWatchers` | No ✓ |
| `BountyCancelled` | `RepoWatchers` | `RepoWatchers` | No ✓ |

Changes needed:
1. `BountyPaid`: Add `AudienceDirectUser` with `TargetUserID = bounty.ClaimerID`
   (exclusive mode). For competitive mode, `MarkPaid` applies to the entire bounty —
   use `AudienceRepoWatchers` instead since all winners watch the repo via auto-subscribe.

#### Grant Events

All four round lifecycle events currently use `AudienceGlobal` (via `publishGrantEvent`
in grants.go). With the addition of `AddOrgUser` on project submission, `AudienceOrgMembers`
now covers grant applicants.

| Event | Current Audience | Correct Audience | Change? |
|-------|-----------------|-----------------|---------|
| `GrantRoundCreated` | `Global` | `Global` | No ✓ (public announcement) |
| `GrantProjectSubmitted` | `Followers` | `Followers` | No (activity tier) |
| `GrantAwarded` | `DirectUser + OrgMembers` | `DirectUser + OrgMembers` | No ✓ |
| `GrantRoundOpened` | `Global` | `Global` | No ✓ (public invitation to apply) |
| `GrantRoundClosed` | `Global` | `OrgMembers` | **Change** (only affects existing applicants) |
| `GrantRoundFinalized` | `Global` | `Global` | No ✓ (public results) |
| `GrantRoundCancelled` | `Global` | `OrgMembers + Global` | **Change** (notify applicants + announce) |

### Part C: Reverse Operations (Unwatch on Leave/Cancel)

Intentional design: **No automatic unwatch on entity cancellation.** Rationale:
- Hackathon org and bounty repo remain useful independently of the entity lifecycle
- There is no "leave hackathon" / "withdraw application" flow in the current codebase
- If such flows are added in the future, they should remove the subscription

### Part D: Sensitivity Note — HackathonScored

`HackathonScored` is classified as Activity tier (Followers only). This means a judge's
followers can see "Judge X scored submission Y" before results are officially announced.
This is acceptable for now since scores are not displayed in the feed content — only the
event type. If score values are later added to feed content, this should be reconsidered.

### Part E: Files to Modify

| File | Change |
|------|--------|
| `routers/web/hackforger/hackathon.go` | Add `WatchRepo` in submission handler |
| `services/hackforger/bounty.go` | Add `WatchRepo` after successful apply/claim; fix `BountyPaid` audience |
| `services/hackforger/grants.go` | Add `AddOrgUser` in `SubmitProject`; change `GrantRoundClosed` to `OrgMembers` |

---

## HF-024: Dashboard Feed Optimization

### Sidebar Replacement

When the community feed tab is active (`FeedType == "community"`), replace the default
`user/dashboard/repolist` template with a HackForger-specific sidebar.

**Default sidebar** (`repolist.tmpl`): Vue-powered repo search + org list. Rendered as
a `<script type="module">` block. Not a Go template partial — it's a client-side component.

**Community sidebar** (new `hackforger/feed/community_sidebar.tmpl`): Server-rendered
Go template showing categorized participation lists.

#### Sidebar Sections

```
┌─────────────────────────┐
│ 🏆 参与的黑客松           │
│ ├ HackFest 2026  [进行中] │
│ └ AI Challenge   [已结束] │
│                         │
│ 💰 参与的悬赏             │
│ ├ Fix auth bug   [已认领] │
│ └ Add dark mode  [已完成] │
│                         │
│ 🎯 参与的赞助             │
│ └ DeFi Round Q2  [已提交] │
└─────────────────────────┘
```

Each item: entity name (linked to entity page) + status badge.

#### Data Queries

Add three query functions to `models/hackforger/` (CRUD layer, matching Forgejo convention).
Each query is bounded to `LIMIT 10` ordered by `created_unix DESC` to keep sidebar compact.

1. **`GetUserHackathons(ctx, userID, limit)`** — JOIN `hackathon_registration` + `hackathon`
   WHERE `registration.user_id = ?`, return `[]Hackathon` with status_cache
2. **`GetUserBounties(ctx, userID, limit)`** — JOIN `bounty_application` + `bounty`
   WHERE `application.user_id = ?` AND `application.status = accepted`, return `[]Bounty`
3. **`GetUserGrantProjects(ctx, userID, limit)`** — JOIN `grant_project` + `grant_round`
   WHERE `application.user_id = ?`, return `[]GrantRound`

#### Template Conditional

In `dashboard.tmpl`, the sidebar switches based on feed type:

```go-template
{{if eq .FeedType "community"}}
    {{template "hackforger/feed/community_sidebar" .}}
{{else}}
    {{template "user/dashboard/repolist" .}}
{{end}}
```

#### Backend Changes

In `routers/web/user/home.go` `Dashboard()`, when `feedType == "community"`, load
the three participation lists into `ctx.Data`:

```go
ctx.Data["UserHackathons"] = hackathons
ctx.Data["UserBounties"] = bounties  
ctx.Data["UserGrantProjects"] = grantProjects
```

---

## i18n Keys

All new strings need entries in both `locale_en-US.ini` and `locale_zh-CN.ini`
(separate values per file, not combined):

**locale_en-US.ini**:
```ini
[hackforger]
feed.sidebar.my_hackathons = My Hackathons
feed.sidebar.my_bounties = My Bounties
feed.sidebar.my_grants = My Grants
feed.sidebar.empty = No participation yet
```

**locale_zh-CN.ini**:
```ini
[hackforger]
feed.sidebar.my_hackathons = 参与的黑客松
feed.sidebar.my_bounties = 参与的悬赏
feed.sidebar.my_grants = 参与的赞助
feed.sidebar.empty = 暂无参与记录
```

## Testing

### E2E Test Scenarios

1. **Auto-watch on register**: Register for hackathon → verify user appears in org member list
2. **Auto-watch on bounty apply**: Apply for bounty → verify user watches the bounty repo
3. **Milestone notification**: Change hackathon phase → verify registered user sees feed event
4. **Activity isolation**: User A registers → verify User B (also registered) does NOT see "A registered" in their feed
5. **Dashboard sidebar**: Switch to community tab → verify sidebar shows participated entities with correct status badges
6. **Empty state**: New user with no participations → verify sidebar shows empty state message

### Manual E2E Prompt

```
1. Login as hacker1, register for a hackathon
2. Login as organizer, change hackathon phase
3. Login as hacker1, check Dashboard community tab
   → Should see phase change event in feed
   → Sidebar should show the hackathon with current status
4. Login as hacker2 (also registered), check feed
   → Should see phase change event
   → Should NOT see "hacker1 registered" event
5. Login as hacker1, apply for a bounty
6. Login as bounty publisher, complete the bounty
7. Login as hacker1, check community tab
   → Should see bounty completed event
   → Sidebar should show the bounty
```
