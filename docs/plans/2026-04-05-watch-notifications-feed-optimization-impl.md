# Watch & Notifications + Dashboard Feed Optimization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Auto-subscribe participants to entity updates via Forgejo's existing watch/org mechanisms, fix audience flags on feed events, and replace the Dashboard sidebar with a HackForger participation panel when the community feed tab is active.

**Architecture:** Reuses Forgejo's `watch` table (repo-level) and `org_user` table (org membership) — no new tables. Three model query functions feed a new server-rendered sidebar template. Audience flags on `publishFeed` calls are audited and corrected.

**Tech Stack:** Go (XORM ORM), Go HTML templates, Fomantic UI CSS

---

## Group A: Auto-Subscribe on Participate (HF-022 Part A)

Tasks 1-3 are independent and can be implemented in parallel.

### Task 1: Auto-Watch Repo on Bounty Apply/Claim

**Files:**
- Modify: `services/hackforger/bounty.go:126-158` (ApplyForBounty)
- Modify: `services/hackforger/bounty.go:163-245` (AcceptApplication — exclusive claim path)

**Step 1: Add `repo_model` import**

In `services/hackforger/bounty.go`, add the import:

```go
import (
	// ... existing imports ...
	repo_model "forgejo.org/models/repo"
)
```

**Step 2: Add WatchRepo after successful apply**

In `ApplyForBounty`, after the `CreateBountyApplication` call succeeds (after line 155), add:

```go
	// Auto-watch: subscriber receives milestone events for this bounty's repo
	_ = repo_model.WatchRepo(ctx, userID, bounty.RepoID, true)
```

**Step 3: Add WatchRepo after exclusive claim**

In `AcceptApplication`, inside the `bounty.Mode == BountyModeExclusive` block, after the bounty status is updated to Claimed (after line 218), add:

```go
			// Auto-watch: claimer receives milestone events for this bounty's repo
			_ = repo_model.WatchRepo(ctx, app.UserID, bounty.RepoID, true)
```

**Step 4: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Compiles without errors.

**Step 5: Commit**

```bash
git add services/hackforger/bounty.go
git commit -m "feat(bounty): auto-watch repo on apply and claim (HF-022)"
```

---

### Task 2: Auto-Watch Track Repo on Hackathon Submission

**Files:**
- Modify: `services/hackforger/hackathon.go:461-501` (CreateSubmission)

**Step 1: Add WatchRepo after submission created**

In `CreateSubmission`, after the `CreateSubmission` DB call succeeds (after line 491), before the track-repo workflow trigger, add:

```go
	// Auto-watch: submitter receives milestone events via track repo
	if sub.TrackID > 0 {
		if track, err := hackforger_model.GetTrackByID(ctx, sub.TrackID); err == nil && track.RepoID > 0 {
			_ = repo_model.WatchRepo(ctx, doer.ID, track.RepoID, true)
		}
	}
```

Note: `repo_model` is already imported in `hackathon.go`.

Wait — the code at line 494-498 already fetches the track. To avoid a duplicate DB call, restructure to share the track lookup:

```go
	// 3. If track has a repo, auto-watch and trigger workflow
	if sub.TrackID > 0 {
		track, err := hackforger_model.GetTrackByID(ctx, sub.TrackID)
		if err == nil && track.RepoID > 0 {
			_ = repo_model.WatchRepo(ctx, doer.ID, track.RepoID, true)
			triggerSubmissionIndexUpdate(ctx, doer, h, track)
		}
	}
```

This replaces lines 493-499.

**Step 2: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Compiles without errors.

**Step 3: Commit**

```bash
git add services/hackforger/hackathon.go
git commit -m "feat(hackathon): auto-watch track repo on submission (HF-022)"
```

---

### Task 3: Auto-Add Org Member on Grant Project Submission

**Files:**
- Modify: `services/hackforger/grants.go:332-386` (SubmitProject)

**Step 1: Add `org_model` alias usage**

`org_model` is already imported in `grants.go` (line 16). No import change needed.

**Step 2: Add AddOrgUser after project created**

In `SubmitProject`, after `CreateGrantProject` succeeds (after line 368), before the notify block, add:

```go
	// Auto-subscribe: applicant joins the grant round's org for lifecycle updates
	if round.OrgID > 0 {
		_ = org_model.AddOrgUser(ctx, round.OrgID, doerID)
	}
```

**Step 3: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Compiles without errors.

**Step 4: Commit**

```bash
git add services/hackforger/grants.go
git commit -m "feat(grants): auto-add org member on project submission (HF-022)"
```

---

## Group B: Audience Flag Fixes (HF-022 Part B)

Tasks 4-5 are independent and can be implemented in parallel.

### Task 4: Fix BountyPaid Audience

**Files:**
- Modify: `services/hackforger/bounty.go:525-568` (MarkPaid)

**Step 1: Update BountyPaid audience flag**

In `MarkPaid`, replace line 555:

```go
			AudienceType: 0,
```

With:

```go
			AudienceType: notify_service.AudienceDirectUser | notify_service.AudienceRepoWatchers,
			TargetUserID: bounty.ClaimerID,
```

This sends the "paid" notification to:
- The claimer directly (exclusive mode — `ClaimerID` is set)
- All repo watchers (competitive mode — winners auto-watch via Task 1)

If `ClaimerID == 0` (competitive), `AudienceDirectUser` is a no-op (the notifier skips `TargetUserID <= 0`).

**Step 2: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Compiles without errors.

**Step 3: Commit**

```bash
git add services/hackforger/bounty.go
git commit -m "fix(bounty): BountyPaid now notifies claimer and repo watchers (HF-022)"
```

---

### Task 5: Fix Grant Round Audience Flags

**Files:**
- Modify: `services/hackforger/grants.go:270-275` (CloseRound)
- Modify: `services/hackforger/grants.go:324-330` (CancelRound)

**Step 1: Change CloseRound audience**

In `CloseRound`, line 275, change:

```go
	return publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundClosed, round, notify_service.AudienceGlobal)
```

To:

```go
	return publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundClosed, round, notify_service.AudienceOrgMembers)
```

**Step 2: Change CancelRound audience**

In `CancelRound`, line 329, change:

```go
	return publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundCancelled, round, notify_service.AudienceGlobal)
```

To:

```go
	return publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundCancelled, round, notify_service.AudienceOrgMembers|notify_service.AudienceGlobal)
```

**Step 3: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Compiles without errors.

**Step 4: Commit**

```bash
git add services/hackforger/grants.go
git commit -m "fix(grants): GrantRoundClosed targets org members, GrantRoundCancelled targets both (HF-022)"
```

---

## Group C: Dashboard Sidebar Queries (HF-024 Backend)

Tasks 6 is standalone model work.

### Task 6: Add Participation Query Functions

**Files:**
- Modify: `models/hackforger/hackathon_registration.go` (add GetUserHackathons)
- Modify: `models/hackforger/bounty_application.go` (add GetUserBounties)
- Modify: `models/hackforger/grant_project.go` (add GetUserGrantProjects)

**Step 1: Add GetUserHackathons**

Append to `models/hackforger/hackathon_registration.go`:

```go
// GetUserHackathons returns hackathons the user is registered for, most recent first.
func GetUserHackathons(ctx context.Context, userID int64, limit int) ([]*Hackathon, error) {
	hackathons := make([]*Hackathon, 0, limit)
	return hackathons, db.GetEngine(ctx).
		Join("INNER", "hackathon_registration", "hackathon_registration.hackathon_id = hackathon.id").
		Where("hackathon_registration.user_id = ?", userID).
		OrderBy("hackathon_registration.created_unix DESC").
		Limit(limit).
		Find(&hackathons)
}
```

**Step 2: Add GetUserBounties**

Append to `models/hackforger/bounty_application.go`:

```go
// GetUserBounties returns bounties the user has been accepted for, most recent first.
func GetUserBounties(ctx context.Context, userID int64, limit int) ([]*Bounty, error) {
	bounties := make([]*Bounty, 0, limit)
	return bounties, db.GetEngine(ctx).
		Join("INNER", "bounty_application", "bounty_application.bounty_id = bounty.id").
		Where("bounty_application.user_id = ? AND bounty_application.status = ?", userID, ApplicationStatusAccepted).
		OrderBy("bounty_application.created_unix DESC").
		Limit(limit).
		Find(&bounties)
}
```

**Step 3: Add GetUserGrantProjects**

Append to `models/hackforger/grant_project.go`:

```go
// GetUserGrantRounds returns grant rounds the user has submitted projects to, most recent first.
func GetUserGrantRounds(ctx context.Context, userID int64, limit int) ([]*GrantRound, error) {
	rounds := make([]*GrantRound, 0, limit)
	return rounds, db.GetEngine(ctx).
		Join("INNER", "grant_project", "grant_project.round_id = grant_round.id").
		Where("grant_project.user_id = ?", userID).
		OrderBy("grant_project.created_unix DESC").
		Limit(limit).
		Find(&rounds)
}
```

**Step 4: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Compiles without errors.

**Step 5: Commit**

```bash
git add models/hackforger/hackathon_registration.go models/hackforger/bounty_application.go models/hackforger/grant_project.go
git commit -m "feat(models): add participation query functions for dashboard sidebar (HF-024)"
```

---

## Group D: Dashboard Handler + Template (HF-024 Frontend)

Task 7 depends on Task 6.

### Task 7: Dashboard Handler Changes

**Files:**
- Modify: `routers/web/user/home.go:123-138` (Dashboard community block)

**Step 1: Add sidebar data loading**

In `Dashboard()`, inside the `if feedType == "community"` block (after line 123, before the `GetHackforgerFeeds` call), add:

```go
		// Load participation lists for community sidebar (always use logged-in user,
		// not ctxUser, since the sidebar shows "my" participations)
		if ctx.Doer != nil {
			if hackathons, err := hackforger_model.GetUserHackathons(ctx, ctx.Doer.ID, 10); err != nil {
				log.Error("GetUserHackathons: %v", err)
			} else {
				ctx.Data["UserHackathons"] = hackathons
			}
			if bounties, err := hackforger_model.GetUserBounties(ctx, ctx.Doer.ID, 10); err != nil {
				log.Error("GetUserBounties: %v", err)
			} else {
				ctx.Data["UserBounties"] = bounties
			}
			if rounds, err := hackforger_model.GetUserGrantRounds(ctx, ctx.Doer.ID, 10); err != nil {
				log.Error("GetUserGrantRounds: %v", err)
			} else {
				ctx.Data["UserGrantRounds"] = rounds
			}
		}
```

**Step 2: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Compiles without errors.

**Step 3: Commit**

```bash
git add routers/web/user/home.go
git commit -m "feat(dashboard): load participation data for community sidebar (HF-024)"
```

---

### Task 8: Community Sidebar Template

**Files:**
- Create: `templates/hackforger/feed/community_sidebar.tmpl`
- Modify: `templates/user/dashboard/dashboard.tmpl:32`

**Step 1: Create the sidebar template**

Create `templates/hackforger/feed/community_sidebar.tmpl`:

```html
<div class="flex-container-sidebar">
	<div class="ui segment">
		<h4 class="ui header">{{svg "octicon-rocket" 16}} {{ctx.Locale.Tr "hackforger.feed.sidebar.my_hackathons"}}</h4>
		{{if .UserHackathons}}
		<div class="ui relaxed list">
			{{range .UserHackathons}}
			<div class="item">
				<a href="{{AppSubUrl}}/hackathon/{{.Slug | PathEscape}}">{{.Name}}</a>
				<span class="ui mini label {{if eq .StatusCache 5}}red{{else if eq .StatusCache 0}}grey{{else}}blue{{end}}">{{ctx.Locale.Tr (printf "hackforger.hackathon.status.%s" (.StatusCacheName))}}</span>
			</div>
			{{end}}
		</div>
		{{else}}
		<p class="text grey">{{ctx.Locale.Tr "hackforger.feed.sidebar.empty"}}</p>
		{{end}}
	</div>

	<div class="ui segment">
		<h4 class="ui header">{{svg "octicon-gift" 16}} {{ctx.Locale.Tr "hackforger.feed.sidebar.my_bounties"}}</h4>
		{{if .UserBounties}}
		<div class="ui relaxed list">
			{{range .UserBounties}}
			<div class="item">
				<a href="{{AppSubUrl}}/bounty/{{.ID}}">{{.Title}}</a>
				<span class="ui mini label {{if eq .Status 6}}red{{else if or (eq .Status 3) (eq .Status 4)}}green{{else}}blue{{end}}">{{ctx.Locale.Tr (printf "hackforger.bounty.status.%s" (.StatusName))}}</span>
			</div>
			{{end}}
		</div>
		{{else}}
		<p class="text grey">{{ctx.Locale.Tr "hackforger.feed.sidebar.empty"}}</p>
		{{end}}
	</div>

	<div class="ui segment">
		<h4 class="ui header">{{svg "octicon-milestone" 16}} {{ctx.Locale.Tr "hackforger.feed.sidebar.my_grants"}}</h4>
		{{if .UserGrantRounds}}
		<div class="ui relaxed list">
			{{range .UserGrantRounds}}
			<div class="item">
				<a href="{{AppSubUrl}}/grants/{{.Slug | PathEscape}}">{{.Name}}</a>
				<span class="ui mini label {{if eq .Status 5}}red{{else if eq .Status 3}}green{{else}}blue{{end}}">{{ctx.Locale.Tr (printf "hackforger.grant.status.%s" (.StatusName))}}</span>
			</div>
			{{end}}
		</div>
		{{else}}
		<p class="text grey">{{ctx.Locale.Tr "hackforger.feed.sidebar.empty"}}</p>
		{{end}}
	</div>
</div>
```

**Step 2: Modify dashboard.tmpl sidebar conditional**

In `templates/user/dashboard/dashboard.tmpl`, replace line 32:

```html
	{{template "user/dashboard/repolist" .}}
```

With:

```html
	{{if eq .FeedType "community"}}
		{{template "hackforger/feed/community_sidebar" .}}
	{{else}}
		{{template "user/dashboard/repolist" .}}
	{{end}}
```

**Step 3: Add StatusCacheName and StatusName helper methods**

The template uses `.StatusCacheName` and `.StatusName` methods. These need to exist on the model structs.

In `models/hackforger/hackathon.go`, add:

```go
// StatusCacheName returns the human-readable name for the hackathon's cached status.
func (h *Hackathon) StatusCacheName() string {
	if name, ok := HackathonStatusNames[h.StatusCache]; ok {
		return name
	}
	return "unknown"
}
```

In `models/hackforger/bounty.go`, add:

```go
// StatusName returns the human-readable name for the bounty's status.
func (b *Bounty) StatusName() string {
	if name, ok := BountyStatusNames[b.Status]; ok {
		return name
	}
	return "unknown"
}
```

In `models/hackforger/grant_round.go`, add:

```go
// StatusName returns the human-readable name for the grant round's status.
func (r *GrantRound) StatusName() string {
	if name, ok := GrantRoundStatusNames[r.Status]; ok {
		return name
	}
	return "unknown"
}
```

**Step 4: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Compiles without errors.

**Step 5: Commit**

```bash
git add templates/hackforger/feed/community_sidebar.tmpl templates/user/dashboard/dashboard.tmpl models/hackforger/hackathon.go models/hackforger/bounty.go models/hackforger/grant_round.go
git commit -m "feat(dashboard): community sidebar with participation lists (HF-024)"
```

---

## Group E: i18n Keys

### Task 9: Add i18n Entries

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

**Step 1: Add English locale keys**

In `options/locale/locale_en-US.ini`, under the `[hackforger]` section, add:

```ini
feed.sidebar.my_hackathons = My Hackathons
feed.sidebar.my_bounties = My Bounties
feed.sidebar.my_grants = My Grants
feed.sidebar.empty = No participation yet
```

Also add status translation keys if not already present:

```ini
hackathon.status.draft = Draft
hackathon.status.open = Open
hackathon.status.hacking = Hacking
hackathon.status.judging = Judging
hackathon.status.finished = Finished
hackathon.status.cancelled = Cancelled
bounty.status.open = Open
bounty.status.claimed = Claimed
bounty.status.in_review = In Review
bounty.status.completed = Completed
bounty.status.paid = Paid
bounty.status.expired = Expired
bounty.status.cancelled = Cancelled
grant.status.draft = Draft
grant.status.open = Open
grant.status.review = Review
grant.status.finalized = Finalized
grant.status.distributed = Distributed
grant.status.cancelled = Cancelled
```

**Step 2: Add Chinese locale keys**

In `options/locale/locale_zh-CN.ini`, under the `[hackforger]` section, add:

```ini
feed.sidebar.my_hackathons = 参与的黑客松
feed.sidebar.my_bounties = 参与的悬赏
feed.sidebar.my_grants = 参与的赞助
feed.sidebar.empty = 暂无参与记录
```

Also add status translation keys:

```ini
hackathon.status.draft = 草稿
hackathon.status.open = 报名中
hackathon.status.hacking = 进行中
hackathon.status.judging = 评审中
hackathon.status.finished = 已结束
hackathon.status.cancelled = 已取消
bounty.status.open = 开放
bounty.status.claimed = 已认领
bounty.status.in_review = 审核中
bounty.status.completed = 已完成
bounty.status.paid = 已支付
bounty.status.expired = 已过期
bounty.status.cancelled = 已取消
grant.status.draft = 草稿
grant.status.open = 开放
grant.status.review = 审核中
grant.status.finalized = 已公布
grant.status.distributed = 已发放
grant.status.cancelled = 已取消
```

**Step 3: Commit**

```bash
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "feat(i18n): add sidebar and status translation keys (HF-022/HF-024)"
```

---

## Group F: Build & Verify

### Task 10: Full Build and Smoke Test

**Step 1: Build backend**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Compiles without errors.

**Step 2: Start server and smoke test**

Run: `./gitea web`

Manual verification:
1. Open `http://localhost:3000` → login
2. Click Dashboard → community tab → verify sidebar shows participation sections
3. If no participations exist, verify empty state message appears

**Step 3: Commit any fixes**

If any build errors or template issues found, fix and commit.

---

## E2E Testing Prompt

After implementation, use agent-browser to run the following E2E scenarios
against `http://localhost:3000`. Save screenshots to `tests/screenshots/`.

```
E2E Test: Watch & Notifications + Dashboard Feed

Test 1: Auto-Watch on Bounty Apply
1. Login as hacker (e.g. hack_user)
2. Navigate to a bounty page
3. Apply for the bounty
4. Screenshot: bounty apply confirmation
5. Verify: check community feed tab shows the bounty in sidebar

Test 2: Dashboard Community Sidebar
1. Login as a user with existing hackathon registration
2. Navigate to Dashboard
3. Click "community" feed tab
4. Screenshot: community sidebar showing "My Hackathons" section
5. Verify: sidebar shows participated hackathon with status badge
6. Verify: default repo list sidebar is NOT visible

Test 3: Empty State
1. Login as a fresh user with no participations
2. Navigate to Dashboard → community tab
3. Screenshot: empty sidebar showing "No participation yet"

Test 4: Activity Isolation
1. Login as hacker1, register for hackathon
2. Login as hacker2 (also registered for same hackathon)
3. Navigate to Dashboard → community tab
4. Verify: hacker2 does NOT see "hacker1 registered" in feed
5. Screenshot: hacker2's feed without hacker1's registration event
```

Store the report in the corresponding access-controlled instance repository,
not in the public source tree.

---

## Dependency Graph

```
Task 1 (Bounty WatchRepo)  ─┐
Task 2 (Hackathon WatchRepo) ├── independent, run in parallel
Task 3 (Grant AddOrgUser)   ─┘
Task 4 (BountyPaid audience) ─┐
Task 5 (Grant audience)      ─┘── independent, run in parallel
Task 6 (Model queries)       ─── prerequisite for Task 7
Task 7 (Dashboard handler)   ─── prerequisite for Task 8
Task 8 (Sidebar template)    ─── prerequisite for Task 10
Task 9 (i18n keys)           ─── prerequisite for Task 8 (template uses them)
Task 10 (Build & verify)     ─── depends on all above
```

Maximum parallel execution:
- **Round 1**: Tasks 1, 2, 3, 4, 5, 6, 9 (all independent)
- **Round 2**: Task 7 (after 6)
- **Round 3**: Task 8 (after 7, 9)
- **Round 4**: Task 10 (after all)
