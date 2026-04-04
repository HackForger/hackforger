# Wave 2: Medium Features — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add data integrity guards, notifications, timeline events, and UI features.

**Architecture:** Split into Wave 2a (backend validation) and 2b (UI/notifications). 2a tasks are independent and run concurrently. 2b tasks are mostly independent but share some upstream injection points.

**Tech Stack:** Go (XORM migrations, services), Go templates, JavaScript, Vue 3, i18n

**Spec:** `docs/superpowers/specs/2026-04-04-wave2-medium-features.md`

---

## Concurrency Groups

```
Group A — Data Integrity (all parallel):
  Task 1: Duplicate submission prevention (HF-011)
  Task 2: Organizer self-registration block (HF-007)
  Task 3: Duplicate activity block (HF-014)
  Task 4: Judge signup validation (HF-013)
  Task 5: Bounty issue edit boundary (HF-015)
  Task 6: Grant budget read-only (HF-019)

Group B — User Search Selector (shared component, do first in 2b):
  Task 7: User search selector component (HF-017)

Group C — Notifications & Timeline (parallel after Task 7):
  Task 8: Bounty status in Issue timeline (HF-016)
  Task 9: "Join Org" button (HF-023)
  Task 10: Grant approval notification (HF-018)
  Task 11: Follow/Watch notification wiring (HF-022)
  Task 12: Judge review UI enrichment (HF-012)

Group D — Dashboard & Explore (parallel):
  Task 13: Dashboard feed optimization (HF-024)
  Task 14: Credit leaderboard on Explore (HF-021)
```

---

### Task 1: Duplicate Submission Prevention (HF-011)

**Files:**
- Create: `models/forgejo_migrations/v14b_hackforger-unique-submission.go`
- Modify: `services/hackforger/hackathon_submission.go`
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`
- Create: `services/hackforger/hackathon_submission_test.go` (or add to existing)

- [ ] **Step 1: Write migration**

```go
// models/forgejo_migrations/v14b_hackforger-unique-submission.go
package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "Add unique constraint on hackforger_submission(user_id, track_id)",
		Upgrade: func(x *xorm.Engine) error {
			_, err := x.Exec("CREATE UNIQUE INDEX IF NOT EXISTS UQE_hackforger_submission_user_track ON hackforger_submission(user_id, track_id)")
			return err
		},
	})
}
```

- [ ] **Step 2: Add service-layer check**

In `services/hackforger/hackathon_submission.go`, before insert:

```go
exists, err := models_hackforger.SubmissionExistsByUserAndTrack(ctx, userID, trackID)
if err != nil {
    return err
}
if exists {
    return ErrDuplicateSubmission
}
```

Define error:
```go
var ErrDuplicateSubmission = errors.New("duplicate submission in track")
```

- [ ] **Step 3: Add i18n keys**

```ini
; en-US
hackforger.error.duplicate_submission = You have already submitted to this track.
; zh-CN
hackforger.error.duplicate_submission = 你已经在此赛道中提交过作品。
```

- [ ] **Step 4: Map error in web route handler**

```go
if errors.Is(err, hackforger_service.ErrDuplicateSubmission) {
    ctx.Flash.Error(ctx.Tr("hackforger.error.duplicate_submission"), true)
    ctx.Redirect(/* submission page */)
    return
}
```

- [ ] **Step 5: Write test**

```go
func TestDuplicateSubmissionBlocked(t *testing.T) {
    // Setup: create hackathon, track, user
    // First submission succeeds
    err := hackforger_service.CreateSubmission(ctx, hackathonID, userID, trackID, ...)
    assert.NoError(t, err)
    // Second submission to same track fails
    err = hackforger_service.CreateSubmission(ctx, hackathonID, userID, trackID, ...)
    assert.ErrorIs(t, err, hackforger_service.ErrDuplicateSubmission)
}
```

Run: `go test ./services/hackforger/... -run TestDuplicateSubmission -v`

- [ ] **Step 6: Commit and create PR**

```bash
git checkout -b feat/hf-011-unique-submission
git add models/forgejo_migrations/ services/hackforger/ routers/ options/locale/
git commit -m "feat(hackforger): HF-011 prevent duplicate submission per track"
gh pr create --title "feat(hackforger): HF-011 duplicate submission prevention" --body "Add UNIQUE constraint on (user_id, track_id) and service-layer check to prevent duplicate submissions within the same track."
```

---

### Task 2: Organizer Self-Registration Block (HF-007)

**Files:**
- Modify: `services/hackforger/hackathon.go` (registration handler)
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`
- Create/modify: `services/hackforger/hackathon_test.go`

- [ ] **Step 1: Add validation in registration service**

```go
func RegisterForHackathon(ctx context.Context, hackathonID, userID int64) error {
    hackathon, err := models_hackforger.GetHackathonByID(ctx, hackathonID)
    if err != nil {
        return err
    }

    // Block organizer self-registration
    if hackathon.OwnerID == userID {
        return ErrCannotRegisterOwnEvent
    }

    // Also check if user is org owner
    if hackathon.OrgID > 0 {
        isOwner, err := org_model.IsOrganizationOwner(ctx, hackathon.OrgID, userID)
        if err != nil {
            return err
        }
        if isOwner {
            return ErrCannotRegisterOwnEvent
        }
    }

    // ... proceed with registration
}
```

- [ ] **Step 2: Define error and i18n**

```go
var ErrCannotRegisterOwnEvent = errors.New("cannot register for own event")
```

```ini
; en-US
hackforger.error.cannot_register_own_event = You cannot register for an event you organize.
; zh-CN
hackforger.error.cannot_register_own_event = 你不能报名自己组织的活动。
```

- [ ] **Step 3: Write test**

```go
func TestOrganizerCannotSelfRegister(t *testing.T) {
    // Create hackathon owned by user A
    // Attempt registration as user A → expect ErrCannotRegisterOwnEvent
    // Attempt registration as user B → expect success
}
```

Run: `go test ./services/hackforger/... -run TestOrganizer -v`

- [ ] **Step 4: Commit and create PR**

```bash
git checkout -b feat/hf-007-block-self-register
git add services/hackforger/ routers/ options/locale/
git commit -m "feat(hackforger): HF-007 block organizer self-registration"
gh pr create --title "feat(hackforger): HF-007 block organizer self-registration" --body "Prevent hackathon organizers and org owners from registering for their own events."
```

---

### Task 3: Duplicate Activity Block (HF-014)

**Files:**
- Modify: `services/hackforger/hackathon.go`, `bounty.go`, `grant.go` (create handlers)
- Modify: `models/hackforger/` (add query function)
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1: Add model query function**

```go
// models/hackforger/activity.go
func ActivityExistsByNameAndKind(ctx context.Context, orgID int64, name string, kind string) (bool, error) {
    switch kind {
    case "hackathon":
        return db.GetEngine(ctx).Where("org_id = ? AND name = ?", orgID, name).Exist(&Hackathon{})
    case "bounty":
        return db.GetEngine(ctx).Where("org_id = ? AND name = ?", orgID, name).Exist(&Bounty{})
    case "grant":
        return db.GetEngine(ctx).Where("org_id = ? AND name = ?", orgID, name).Exist(&GrantRound{})
    }
    return false, fmt.Errorf("unknown activity kind: %s", kind)
}
```

- [ ] **Step 2: Add validation in each create service function**

```go
exists, err := models_hackforger.ActivityExistsByNameAndKind(ctx, orgID, name, "hackathon")
if err != nil {
    return err
}
if exists {
    return ErrDuplicateActivityName
}
```

- [ ] **Step 3: Add i18n**

```ini
; en-US
hackforger.error.duplicate_activity_name = An activity with this name already exists in the organization.
; zh-CN
hackforger.error.duplicate_activity_name = 该组织下已存在同名活动。
```

- [ ] **Step 4: Write test and commit**

```bash
git checkout -b feat/hf-014-duplicate-activity
git commit -m "feat(hackforger): HF-014 block duplicate activity creation"
gh pr create --title "feat(hackforger): HF-014 block duplicate activity names" --body "Block creating activities with duplicate names within the same org and activity kind."
```

---

### Task 4: Judge Signup Validation (HF-013)

**Files:**
- Modify: `services/hackforger/hackathon.go` (judge registration handler)
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1: Add validation logic**

```go
func RegisterJudge(ctx context.Context, hackathonID, userID int64) error {
    // Check user exists
    user, err := user_model.GetUserByID(ctx, userID)
    if err != nil {
        return ErrUserNotFound
    }

    // Check not already a judge
    isJudge, err := models_hackforger.IsJudge(ctx, hackathonID, userID)
    if err != nil {
        return err
    }
    if isJudge {
        return ErrAlreadyJudge
    }

    // Check not a participant
    isParticipant, err := models_hackforger.IsParticipant(ctx, hackathonID, userID)
    if err != nil {
        return err
    }
    if isParticipant {
        return ErrJudgeCannotBeParticipant
    }

    // ... proceed
}
```

- [ ] **Step 2: Add i18n keys**

```ini
; en-US
hackforger.error.user_not_found = User not found.
hackforger.error.already_judge = This user is already a judge for this event.
hackforger.error.judge_cannot_be_participant = A participant cannot also be a judge.
; zh-CN
hackforger.error.user_not_found = 用户不存在。
hackforger.error.already_judge = 该用户已经是此活动的评委。
hackforger.error.judge_cannot_be_participant = 参赛者不能同时担任评委。
```

- [ ] **Step 3: Write test and commit**

```bash
git checkout -b feat/hf-013-judge-validation
git commit -m "feat(hackforger): HF-013 judge signup validation"
gh pr create --title "feat(hackforger): HF-013 judge signup validation" --body "Validate judge signup: user must exist, not already a judge, not a participant."
```

---

### Task 5: Bounty Issue Edit Boundary (HF-015)

**Files:**
- Modify: `routers/web/hackforger/bounty.go` or `routers/web/repo/issue.go` (issue edit handler)
- Modify: Templates (hide Bounty fields from Issue edit form)

- [ ] **Step 1: Identify Bounty-managed fields**

Fields controlled by Bounty: amount, status, reward recipients. These should not be editable through the Issue edit interface.

- [ ] **Step 2: Hide Bounty fields in Issue edit template**

In the Issue edit template, conditionally hide Bounty-specific fields:

```html
{{if not .Issue.HasBounty}}
  {{/* Show bounty-related edit fields only if no bounty is attached */}}
{{end}}
```

- [ ] **Step 3: Add server-side guard**

In the Issue update handler, if the Issue has an associated Bounty, strip Bounty-managed fields from the update:

```go
if issue.HasBounty() {
    // Ignore bounty-managed fields from form submission
    form.Amount = issue.Bounty.Amount  // preserve original
    form.BountyStatus = issue.Bounty.Status
}
```

- [ ] **Step 4: Commit and create PR**

```bash
git checkout -b fix/hf-015-bounty-edit-boundary
git commit -m "fix(hackforger): HF-015 restrict editing Bounty-managed fields on Issue"
gh pr create --title "fix(hackforger): HF-015 bounty issue edit boundary" --body "Prevent editing Bounty-managed fields (amount, status) through the Issue edit interface."
```

---

### Task 6: Grant Budget Read-Only (HF-019)

**Files:**
- Modify: Grant Round view template
- Modify: Admin grant management template

- [ ] **Step 1: Make "已使用" amount read-only**

In grant round view template, replace editable input with read-only text:

```html
{{/* Before: <input type="number" name="used_amount" value="{{.GrantRound.UsedAmount}}"> */}}
<span class="hackforger-budget-used">{{.GrantRound.UsedAmount}}</span>
```

- [ ] **Step 2: Make admin reward amount read-only**

Same treatment in admin panel template.

- [ ] **Step 3: Commit and create PR**

```bash
git checkout -b fix/hf-019-grant-readonly
git commit -m "fix(hackforger): HF-019 make grant budget/reward amount read-only"
gh pr create --title "fix(hackforger): HF-019 grant budget read-only" --body "Render used budget amount and admin reward amount as read-only text instead of editable inputs."
```

---

### Task 7: User Search Selector (HF-017)

**Files:**
- Modify: All HackForger templates with username text inputs
- Reference: `web_src/js/features/comp/SearchUserBox.js`

- [ ] **Step 1: Understand SearchUserBox API**

Read `web_src/js/features/comp/SearchUserBox.js` to understand its initialization and data attributes.

- [ ] **Step 2: Locate all username text inputs in HackForger templates**

```bash
grep -rn "input.*username\|input.*user_name\|用户名" templates/hackforger/ --include="*.tmpl"
```

- [ ] **Step 3: Replace text inputs with search selector**

For each username input, replace with the SearchUserBox pattern:

```html
<div class="ui search selection dropdown" id="hackforger-user-search"
     data-search-url="{{AppSubUrl}}/api/v1/users/search?q={query}">
  <input type="hidden" name="username" value="">
  <i class="dropdown icon"></i>
  <input class="search" placeholder="{{ctx.Locale.Tr "hackforger.search_user"}}">
  <div class="default text">{{ctx.Locale.Tr "hackforger.search_user"}}</div>
  <div class="menu"></div>
</div>
```

Initialize with Fomantic UI dropdown's remote API.

- [ ] **Step 4: Add i18n**

```ini
; en-US
hackforger.search_user = Search users...
; zh-CN
hackforger.search_user = 搜索用户...
```

- [ ] **Step 5: Build and test**

Run: `make frontend && make backend`
Verify: dropdown shows user search results when typing.

- [ ] **Step 6: Commit and create PR**

```bash
git checkout -b feat/hf-017-user-search-selector
git add templates/hackforger/ web_src/ options/locale/
git commit -m "feat(hackforger): HF-017 replace text inputs with user search selector"
gh pr create --title "feat(hackforger): HF-017 user search selector" --body "Replace all username text inputs with search dropdown using Forgejo's SearchUserBox. Resolves HackForger/hackforger#17."
```

---

### Task 8: Bounty Status in Issue Timeline (HF-016)

**Files:**
- Modify: `models/issues/comment.go` (upstream injection: add CommentTypeHackforger)
- Create: `templates/hackforger/issue_comment.tmpl`
- Modify: `templates/repo/issue/view_content/comments.tmpl` (upstream injection: include block)
- Modify: `services/hackforger/bounty.go` (add timeline comment on status change)
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1: Add CommentTypeHackforger to upstream**

In `models/issues/comment.go`, add one constant:

```go
CommentTypeHackforger // HackForger injection point: generic HackForger timeline event
```

- [ ] **Step 2: Create HackForger comment template**

```html
{{/* templates/hackforger/issue_comment.tmpl */}}
{{$payload := .Content | JsonParse}}
{{if eq $payload.type "bounty_created"}}
  <div class="timeline-item hackforger-bounty-event">
    <span class="badge bg-green">{{ctx.Locale.Tr "hackforger.bounty.created"}}</span>
    {{.Poster.Name}} {{ctx.Locale.Tr "hackforger.bounty.event_created"}}
    <span class="time-since" title="{{.CreatedUnix.FormatLong}}">{{TimeSince .CreatedUnix ctx.Locale}}</span>
  </div>
{{else if eq $payload.type "bounty_applied"}}
  <div class="timeline-item hackforger-bounty-event">
    <span class="badge bg-blue">{{ctx.Locale.Tr "hackforger.bounty.applied"}}</span>
    {{.Poster.Name}} {{ctx.Locale.Tr "hackforger.bounty.event_applied"}}
    <span class="time-since" title="{{.CreatedUnix.FormatLong}}">{{TimeSince .CreatedUnix ctx.Locale}}</span>
  </div>
{{else if eq $payload.type "bounty_accepted"}}
  <div class="timeline-item hackforger-bounty-event">
    <span class="badge bg-purple">{{ctx.Locale.Tr "hackforger.bounty.accepted"}}</span>
    {{.Poster.Name}} {{ctx.Locale.Tr "hackforger.bounty.event_accepted"}}
    <span class="time-since" title="{{.CreatedUnix.FormatLong}}">{{TimeSince .CreatedUnix ctx.Locale}}</span>
  </div>
{{else if eq $payload.type "bounty_completed"}}
  <div class="timeline-item hackforger-bounty-event">
    <span class="badge bg-gold">{{ctx.Locale.Tr "hackforger.bounty.completed"}}</span>
    {{.Poster.Name}} {{ctx.Locale.Tr "hackforger.bounty.event_completed"}}
    <span class="time-since" title="{{.CreatedUnix.FormatLong}}">{{TimeSince .CreatedUnix ctx.Locale}}</span>
  </div>
{{else if eq $payload.type "bounty_cancelled"}}
  <div class="timeline-item hackforger-bounty-event">
    <span class="badge bg-red">{{ctx.Locale.Tr "hackforger.bounty.cancelled"}}</span>
    {{.Poster.Name}} {{ctx.Locale.Tr "hackforger.bounty.event_cancelled"}}
    <span class="time-since" title="{{.CreatedUnix.FormatLong}}">{{TimeSince .CreatedUnix ctx.Locale}}</span>
  </div>
{{end}}
```

- [ ] **Step 3: Add injection point in upstream comments template**

In `templates/repo/issue/view_content/comments.tmpl`, add:

```html
{{else if eq .Type CommentTypeHackforger}}
  {{template "hackforger/issue_comment" .}}
```

- [ ] **Step 4: Write timeline comment on Bounty status change**

In `services/hackforger/bounty.go`, after each status transition:

```go
import issues_model "forgejo.org/models/issues"

func createBountyTimelineComment(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, eventType string, data map[string]any) error {
    payload, _ := json.Marshal(map[string]any{
        "type": eventType,
        "data": data,
    })
    _, err := issues_model.CreateComment(ctx, &issues_model.CreateCommentOptions{
        Type:    issues_model.CommentTypeHackforger,
        Doer:    doer,
        Repo:    issue.Repo,
        Issue:   issue,
        Content: string(payload),
    })
    return err
}
```

Call at each transition point:
```go
createBountyTimelineComment(ctx, issue, doer, "bounty_created", nil)
createBountyTimelineComment(ctx, issue, doer, "bounty_applied", map[string]any{"applicant": applicant.Name})
// etc.
```

- [ ] **Step 5: Add i18n keys for all bounty events**

```ini
; en-US
hackforger.bounty.created = Bounty Created
hackforger.bounty.applied = Applied
hackforger.bounty.accepted = Accepted
hackforger.bounty.completed = Completed
hackforger.bounty.cancelled = Cancelled
hackforger.bounty.event_created = created a bounty on this issue
hackforger.bounty.event_applied = applied for this bounty
hackforger.bounty.event_accepted = was accepted for this bounty
hackforger.bounty.event_completed = completed this bounty
hackforger.bounty.event_cancelled = cancelled this bounty

; zh-CN
hackforger.bounty.created = 悬赏已创建
hackforger.bounty.applied = 已申请
hackforger.bounty.accepted = 已接受
hackforger.bounty.completed = 已完成
hackforger.bounty.cancelled = 已取消
hackforger.bounty.event_created = 在此 Issue 上创建了悬赏
hackforger.bounty.event_applied = 申请了此悬赏
hackforger.bounty.event_accepted = 被接受参与此悬赏
hackforger.bounty.event_completed = 完成了此悬赏
hackforger.bounty.event_cancelled = 取消了此悬赏
```

- [ ] **Step 6: Write test and commit**

```bash
git checkout -b feat/hf-016-bounty-timeline
git add models/issues/ templates/ services/hackforger/ options/locale/
git commit -m "feat(hackforger): HF-016 bounty status events in Issue timeline"
gh pr create --title "feat(hackforger): HF-016 bounty status in Issue timeline" --body "Add bounty lifecycle events to Issue comment timeline via CommentTypeHackforger injection point."
```

---

### Task 9: "Join Org" Button (HF-023)

**Files:**
- Modify: `templates/org/header.tmpl` (upstream injection: add button)
- Create: `routers/web/hackforger/org.go`
- Modify: `templates/org/team/members.tmpl` (read ?username param)
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1: Add button to org header**

In `templates/org/header.tmpl`, add after existing buttons:

```html
{{/* HackForger injection: Join Org button */}}
{{if and .SignedUser (not .IsOrganizationMember)}}
  <form method="POST" action="{{.Org.HomeLink}}/join-request">
    {{.CsrfTokenHtml}}
    <button class="ui primary button">{{ctx.Locale.Tr "hackforger.org.join_request"}}</button>
  </form>
{{end}}
```

Wait — per project memory, Forgejo uses `CrossOriginProtection` not CSRF tokens. Remove `{{.CsrfTokenHtml}}` and use standard POST form.

- [ ] **Step 2: Create route handler**

```go
// routers/web/hackforger/org.go
package hackforger

func JoinOrgRequest(ctx *context.Context) {
    org := ctx.Org.Organization
    // Send notification to org owner
    owner, err := org.GetOwner(ctx)
    if err != nil {
        ctx.Flash.Error(ctx.Tr("hackforger.error.internal"), true)
        ctx.Redirect(org.HomeLink())
        return
    }

    addMemberURL := fmt.Sprintf("%s/org/%s/teams/%s/members?username=%s",
        setting.AppURL, org.Name, "owners", ctx.Doer.Name)

    // Create notification via HackForger notifier
    hackforger_service.NotifyJoinOrgRequest(ctx, org, ctx.Doer, owner, addMemberURL)

    ctx.Flash.Success(ctx.Tr("hackforger.org.join_request_sent"), true)
    ctx.Redirect(org.HomeLink())
}
```

- [ ] **Step 3: Register route**

In web.go at the org routes injection point:
```go
m.Post("/:org/join-request", hackforger.JoinOrgRequest)
```

- [ ] **Step 4: Pre-fill username on add-member page**

In the members template, read the query param:
```html
<input type="text" name="uname" value="{{.Query.username}}" placeholder="...">
```

- [ ] **Step 5: Add i18n and commit**

```ini
; en-US
hackforger.org.join_request = Request to Join
hackforger.org.join_request_sent = Join request sent to the organization owner.
; zh-CN
hackforger.org.join_request = 申请加入
hackforger.org.join_request_sent = 加入申请已发送给组织所有者。
```

```bash
git checkout -b feat/hf-023-join-org
git commit -m "feat(hackforger): HF-023 add Join Org button with notification"
gh pr create --title "feat(hackforger): HF-023 join org button" --body "Add 'Request to Join' button on org pages. Sends notification to org owner with link to add-member page."
```

---

### Task 10: Grant Approval Notification (HF-018)

**Files:**
- Modify: `services/hackforger/grant.go`
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1: Add notification call on grant approval**

In the grant approval service function:

```go
func ApproveGrantApplication(ctx context.Context, applicationID int64, approverID int64, amount int64) error {
    // ... existing approval logic ...

    // Notify applicant
    hackforger_service.PublishHackforgerAction(ctx, &HackforgerActionOptions{
        ActorID:      approverID,
        TargetUserID: application.ApplicantID,
        ActionType:   ActionGrantApproved,
        Content:      fmt.Sprintf("Grant approved: %s, amount: %d", grantRound.Name, amount),
        Link:         grantRound.Link(),
    })

    return nil
}
```

- [ ] **Step 2: Add i18n**

```ini
; en-US
hackforger.notification.grant_approved = Your grant application for "%s" has been approved (amount: %d).
; zh-CN
hackforger.notification.grant_approved = 你的赞助申请「%s」已获批准（金额：%d）。
```

- [ ] **Step 3: Commit and create PR**

```bash
git checkout -b feat/hf-018-grant-notification
git commit -m "feat(hackforger): HF-018 grant approval notification"
gh pr create --title "feat(hackforger): HF-018 grant approval notification" --body "Notify applicant when their grant application is approved."
```

---

### Task 11: Follow/Watch Notification Wiring (HF-022)

**Files:**
- Modify: `services/hackforger/` (all service files with state changes)

- [ ] **Step 1: Audit all PublishHackforgerAction calls**

```bash
grep -rn "PublishHackforgerAction" services/hackforger/ --include="*.go"
```

- [ ] **Step 2: Identify missing notification triggers**

Compare against the list of all state changes in Hackathon, Bounty, Grant services. Any state change without a `PublishHackforgerAction` call needs one.

- [ ] **Step 3: Add missing calls**

For each missing trigger, add:
```go
hackforger_service.PublishHackforgerAction(ctx, &HackforgerActionOptions{
    ActorID:    doer.ID,
    RepoID:     repo.ID,
    ActionType: ActionXxx,
    Content:    "...",
})
```

- [ ] **Step 4: Verify HackForgerNotifier covers all action types**

Check that `HackForgerNotifier` handles each `ActionType` and produces a notification for repo watchers.

- [ ] **Step 5: Commit and create PR**

```bash
git checkout -b feat/hf-022-watch-notifications
git commit -m "feat(hackforger): HF-022 wire Follow/Watch notifications for all state changes"
gh pr create --title "feat(hackforger): HF-022 Follow/Watch notification wiring" --body "Ensure all HackForger state changes trigger notifications for repo watchers."
```

---

### Task 12: Judge Review UI Enrichment (HF-012)

**Files:**
- Modify: Judge review template

- [ ] **Step 1: Add submitter info to judging UI**

In the judge review template, add:

```html
<div class="hackforger-submission-info">
  <a href="{{AppSubUrl}}/{{.Submission.User.Name}}">
    {{avatar .Submission.User 20}} {{.Submission.User.Name}}
  </a>
  {{if .Submission.RepoLink}}
    | <a href="{{.Submission.RepoLink}}">{{svg "octicon-repo"}} {{.Submission.RepoName}}</a>
  {{end}}
  <p>{{.Submission.Description}}</p>
  <span class="tag">{{.Submission.TrackName}}</span>
</div>
```

- [ ] **Step 2: Ensure route handler passes full submission data**

In the judge review route handler, make sure Submission is loaded with User and Repo relations:

```go
submissions, err := models_hackforger.GetSubmissionsForJudging(ctx, hackathonID, opts)
// Ensure each submission has .User and .Repo populated
```

- [ ] **Step 3: Commit and create PR**

```bash
git checkout -b feat/hf-012-judge-ui
git commit -m "feat(hackforger): HF-012 enrich judge review UI with submitter info"
gh pr create --title "feat(hackforger): HF-012 richer judge review UI" --body "Show submitter name, profile link, repo link, description, and track name in the judging interface."
```

---

### Task 13: Dashboard Feed Optimization (HF-024)

**Files:**
- Create: `models/hackforger/activity_summary.go`
- Create: `templates/hackforger/dashboard_sidebar.tmpl`
- Modify: `templates/user/dashboard/feeds.tmpl` (upstream injection)
- Modify: `routers/web/hackforger/` (add dashboard data loader)
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1: Create model function**

```go
// models/hackforger/activity_summary.go
type UserActivitySummary struct {
    Hackathons  []*Hackathon
    Bounties    []*Bounty
    GrantRounds []*GrantRound
}

func GetUserActivitySummary(ctx context.Context, userID int64) (*UserActivitySummary, error) {
    summary := &UserActivitySummary{}
    // Query hackathons where user is participant
    // Query bounties where user is applicant/assignee
    // Query grants where user is applicant
    return summary, nil
}
```

- [ ] **Step 2: Create sidebar template**

```html
{{/* templates/hackforger/dashboard_sidebar.tmpl */}}
{{if .HackforgerSummary}}
<div class="hackforger-sidebar">
  {{if .HackforgerSummary.Hackathons}}
  <h4>{{ctx.Locale.Tr "hackforger.dashboard.my_hackathons"}}</h4>
  <ul>
    {{range .HackforgerSummary.Hackathons}}
    <li><a href="{{.Link}}">{{.Name}}</a></li>
    {{end}}
  </ul>
  {{end}}
  {{/* Same for Bounties and Grants */}}
</div>
{{end}}
```

- [ ] **Step 3: Inject into dashboard feeds template**

In `templates/user/dashboard/feeds.tmpl`:
```html
{{/* HackForger injection: activity sidebar */}}
{{template "hackforger/dashboard_sidebar" .}}
```

- [ ] **Step 4: Add i18n, test, and commit**

```bash
git checkout -b feat/hf-024-dashboard-feed
git commit -m "feat(hackforger): HF-024 optimize dashboard feed with activity sidebar"
gh pr create --title "feat(hackforger): HF-024 dashboard feed optimization" --body "Add HackForger activity sidebar to dashboard showing user's hackathons, bounties, and grants. Resolves HackForger/hackforger#16."
```

---

### Task 14: Credit Leaderboard on Explore (HF-021)

**Files:**
- Create: `models/hackforger/credits_leaderboard.go`
- Create: `routers/web/hackforger/explore.go`
- Create: `templates/hackforger/explore_credits.tmpl`
- Modify: `templates/explore/navbar.tmpl` (upstream injection)
- Modify: `routers/web/web.go` (route registration)
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1: Create model function**

```go
// models/hackforger/credits_leaderboard.go
type LeaderboardEntry struct {
    Rank     int
    User     *user_model.User
    Credits  int64
}

func GetLeaderboard(ctx context.Context, page, pageSize int) ([]*LeaderboardEntry, int64, error) {
    // Query users ordered by credit balance descending
    // Return entries + total count for pagination
}
```

- [ ] **Step 2: Create route handler**

```go
// routers/web/hackforger/explore.go
func ExploreCredits(ctx *context.Context) {
    page := ctx.FormInt("page")
    if page <= 0 { page = 1 }

    entries, total, err := models_hackforger.GetLeaderboard(ctx, page, 50)
    if err != nil {
        ctx.ServerError("GetLeaderboard", err)
        return
    }

    ctx.Data["Entries"] = entries
    ctx.Data["Total"] = total
    ctx.Data["Page"] = paginater.New(int(total), 50, page, 5)
    ctx.HTML(http.StatusOK, "hackforger/explore_credits")
}
```

- [ ] **Step 3: Create template**

```html
{{/* templates/hackforger/explore_credits.tmpl */}}
{{template "base/head" .}}
<div role="main" class="page-content">
  {{template "explore/navbar" .}}
  <div class="ui container">
    <h2>{{ctx.Locale.Tr "hackforger.credits.leaderboard"}}</h2>
    <table class="ui table">
      <thead>
        <tr>
          <th>#</th>
          <th>{{ctx.Locale.Tr "hackforger.credits.user"}}</th>
          <th>{{ctx.Locale.Tr "hackforger.credits.amount"}}</th>
        </tr>
      </thead>
      <tbody>
        {{range .Entries}}
        <tr>
          <td>{{.Rank}}</td>
          <td>{{avatar .User 20}} <a href="{{AppSubUrl}}/{{.User.Name}}">{{.User.Name}}</a></td>
          <td>{{.Credits}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>
    {{template "base/paginate" .}}
  </div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 4: Add tab to explore navbar**

In `templates/explore/navbar.tmpl`:
```html
{{/* HackForger injection: Credits tab */}}
<a class="{{if .PageIsExploreCredits}}active {{end}}item" href="{{AppSubUrl}}/explore/credits">
  {{ctx.Locale.Tr "hackforger.credits.leaderboard"}}
</a>
```

- [ ] **Step 5: Register route**

In `routers/web/web.go`:
```go
m.Get("/explore/credits", hackforger.ExploreCredits)
```

- [ ] **Step 6: Add i18n, test, and commit**

```ini
; en-US
hackforger.credits.leaderboard = Credit Leaderboard
hackforger.credits.user = User
hackforger.credits.amount = Credits
; zh-CN
hackforger.credits.leaderboard = 积分排行榜
hackforger.credits.user = 用户
hackforger.credits.amount = 积分
```

```bash
git checkout -b feat/hf-021-credit-leaderboard
git add models/hackforger/ routers/ templates/ options/locale/
git commit -m "feat(hackforger): HF-021 credit leaderboard on Explore page"
gh pr create --title "feat(hackforger): HF-021 credit leaderboard" --body "Add Credits leaderboard tab to Explore page showing users ranked by credit balance."
```

---

## E2E Verification After Wave 2

After all Wave 2 PRs are merged:

- [ ] Create a Bounty on an Issue → verify timeline shows "Bounty Created" event
- [ ] Apply for a Bounty → verify timeline shows "Applied" event
- [ ] Visit org page as non-member → verify "Request to Join" button appears
- [ ] Click join → verify org owner receives notification
- [ ] Approve a Grant → verify applicant receives notification
- [ ] Try duplicate submission to same track → verify error message
- [ ] Try organizer self-registration → verify blocked
- [ ] Check judge assignment → verify user search dropdown works
- [ ] Check dashboard → verify HackForger activity sidebar shows
- [ ] Visit /explore/credits → verify leaderboard displays

Take screenshots at each checkpoint. Save report to `docs/tests/e2e/wave2-results.md`.
