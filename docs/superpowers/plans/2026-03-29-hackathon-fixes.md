# Phase 1.1 Hackathon — Org/Repo Model + View + Bug Fixes

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align hackathon model with Forgejo's Org/Repo primitives (Hackathon→Org, Track→Repo, Register→Org Member, Submit→Fork+PR), redesign detail page, fix dashboard feed bug (#1) and error handling UX.

**Architecture:** Hackathon creates a real Forgejo Organization; Track creates a real Repository in that Org. Registration adds user to Org. Submission forks the track repo and creates a PR. Extension tables (`hackathon`, `hackathon_track`, `hackathon_submission`) store HackForger-specific metadata (status machine, scores, dates, prizes) via foreign keys to the Forgejo entities.

**Tech Stack:** Go, XORM, Forgejo service APIs (`organization.CreateOrganization`, `repo_service.CreateRepository`, `repo_service.ForkRepositoryIfNotExists`, `pull_service.NewPullRequest`), Go HTML templates.

**Key Forgejo APIs:**
- `organization.CreateOrganization(ctx, org, owner)` — creates org
- `organization.AddOrgUser(ctx, orgID, userID)` — adds member
- `repo_service.CreateRepository(ctx, doer, owner, CreateRepoOptions{})` — creates repo
- `repo_service.ForkRepositoryIfNotExists(ctx, doer, owner, ForkRepoOptions{})` — forks repo
- `pull_service.NewPullRequest(ctx, repo, issue, labelIDs, uuids, pr, assigneeIDs)` — creates PR

---

## Task Ordering Rationale

```
T1: Bug #1 fix (LEFT JOIN) ──────→ no dependencies, unblocks feed testing
T2: Model migration (add fields) ─→ schema changes, no code dependencies
T3: Hackathon service (auto Org/Repo) → depends on T2 for new fields
T4: Registration service (Org member) → depends on T3 for linked_org_id
T5: Submission service (Fork+PR) ──→ depends on T3 for track repo_id
T6: View page redesign ───────────→ depends on T3-T5 for data to display
T7: Error handling UX ────────────→ independent, polish
T8: Build + E2E ───────────────────→ depends on all above
```

---

## Chunk 1: Dashboard Feed Bug Fix

### Task 1: Fix GetFeeds INNER JOIN excluding repo_id=0 events

**Root cause:** `models/activities/action.go` `GetFeeds()` uses `INNER JOIN repository` — hackathon events with `RepoID=0` are excluded.

**Files:**
- Modify: `models/activities/action.go` (~line 502)
- Modify: `models/activities/action_list.go` (`loadRepoOwner`)

- [ ] **Step 1:** In `models/activities/action.go`, find `GetFeeds` function. Change:
```go
Join("INNER", "repository", "`repository`.id = `action`.repo_id")
```
to:
```go
Join("LEFT", "repository", "`repository`.id = `action`.repo_id")
```

- [ ] **Step 2:** In `models/activities/action_list.go`, find `loadRepoOwner` function. The loop accesses `action.Repo.OwnerID` — with LEFT JOIN, `action.Repo` can be nil. Add a nil guard at the start of the loop body:
```go
if action.Repo == nil {
    continue
}
```

- [ ] **Step 3:** Verify: `go build ./models/activities/...`

- [ ] **Step 4:** Commit:
```bash
git add models/activities/action.go models/activities/action_list.go
git commit -m "fix: LEFT JOIN in GetFeeds to include repo_id=0 HackForger events"
```

---

## Chunk 2: Model Schema Changes

### Task 2: Add LinkedOrgID to Hackathon + OrgID to Registration + PRID to Submission

**Files:**
- Modify: `models/hackforger/hackathon.go` — add `LinkedOrgID int64`
- Modify: `models/hackforger/hackathon_registration.go` — add `OrgID int64`
- Modify: `models/hackforger/hackathon_submission.go` — add `PRID int64`, `PullIndex int64`
- Create: `models/forgejo_migrations/v14f_hackathon-org-repo-fields.go`

- [ ] **Step 1:** Add fields to model structs.

`hackathon.go` — add after `TemplateRepoID`:
```go
LinkedOrgID int64 `xorm:"INDEX"` // the auto-created Forgejo Organization for this hackathon
```

`hackathon_registration.go` — add after `UserID`:
```go
OrgID int64 `xorm:"INDEX"` // 0=solo, >0=team via this org
```

`hackathon_submission.go` — add after `RepoID`:
```go
ForkRepoID int64 `xorm:"INDEX"` // participant's fork of the track repo
PRID       int64 `xorm:"INDEX"` // pull request ID (action table primary key)
PullIndex  int64 `xorm:""` // pull request index within the track repo
```

- [ ] **Step 2:** Write migration.

File: `models/forgejo_migrations/v14f_hackathon-org-repo-fields.go`

```go
package forgejo_migrations

import "xorm.io/xorm"

func init() {
    registerMigration(&Migration{
        Description: "add org/repo integration fields to hackathon tables",
        Upgrade:     addHackathonOrgRepoFields,
    })
}

type v14fHackathon struct {
    LinkedOrgID int64 `xorm:"INDEX DEFAULT 0"`
}
func (v14fHackathon) TableName() string { return "hackathon" }

type v14fHackathonRegistration struct {
    OrgID int64 `xorm:"INDEX DEFAULT 0"`
}
func (v14fHackathonRegistration) TableName() string { return "hackathon_registration" }

type v14fHackathonSubmission struct {
    ForkRepoID int64 `xorm:"INDEX DEFAULT 0"`
    PRID       int64 `xorm:"INDEX DEFAULT 0"`
    PullIndex  int64 `xorm:"DEFAULT 0"`
}
func (v14fHackathonSubmission) TableName() string { return "hackathon_submission" }

func addHackathonOrgRepoFields(x *xorm.Engine) error {
    if err := x.Sync(new(v14fHackathon)); err != nil {
        return err
    }
    if err := x.Sync(new(v14fHackathonRegistration)); err != nil {
        return err
    }
    return x.Sync(new(v14fHackathonSubmission))
}
```

- [ ] **Step 3:** Update `CreateRegistration` uniqueness logic for Org support.

In `hackathon_registration.go`, replace `CreateRegistration`:

```go
func CreateRegistration(ctx context.Context, r *HackathonRegistration) error {
    existing := &HackathonRegistration{}
    has, err := db.GetEngine(ctx).Where("hackathon_id = ? AND user_id = ?", r.HackathonID, r.UserID).Get(existing)
    if err != nil {
        return err
    }
    if has {
        if r.OrgID > 0 && existing.OrgID == 0 {
            // Solo → Team upgrade
            existing.OrgID = r.OrgID
            existing.TeamName = r.TeamName
            _, err := db.GetEngine(ctx).ID(existing.ID).Cols("org_id", "team_name").Update(existing)
            return err
        }
        return ErrDuplicateRegistration{HackathonID: r.HackathonID, UserID: r.UserID}
    }
    return db.Insert(ctx, r)
}
```

- [ ] **Step 4:** Verify: `go build ./models/hackforger/... ./models/forgejo_migrations/...`

- [ ] **Step 5:** Commit:
```bash
git add models/hackforger/hackathon.go models/hackforger/hackathon_registration.go models/hackforger/hackathon_submission.go models/forgejo_migrations/v14f_hackathon-org-repo-fields.go
git commit -m "feat(hackforger): add Org/Repo integration fields to hackathon models

Hackathon.LinkedOrgID → auto-created Forgejo Org
HackathonRegistration.OrgID → solo(0) or team via org
HackathonSubmission.ForkRepoID/PRID/PullIndex → Fork+PR submission"
```

---

## Chunk 3: Hackathon Service — Auto Org/Repo Creation

### Task 3: Create Org on hackathon creation, Repo on track creation

**Files:**
- Modify: `services/hackforger/hackathon.go`
- Modify: `routers/web/hackforger/hackathon.go` (import changes)

- [ ] **Step 1:** Update `CreateHackathon` service to auto-create Org.

In `services/hackforger/hackathon.go`, modify `CreateHackathon`:

```go
import (
    // add these imports
    organization_model "forgejo.org/models/organization"
    user_model "forgejo.org/models/user"
)

func CreateHackathon(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon) error {
    // 1. Create a Forgejo Organization for this hackathon
    org := &organization_model.Organization{
        Name:       h.Slug,
        FullName:   h.Name,
        Visibility: structs.VisibleTypePublic,
    }
    if err := organization_model.CreateOrganization(ctx, org, doer); err != nil {
        // If org name collision, try with suffix
        if user_model.IsErrUserAlreadyExist(err) {
            org.Name = h.Slug + "-hackathon"
            if err := organization_model.CreateOrganization(ctx, org, doer); err != nil {
                return fmt.Errorf("create hackathon org: %w", err)
            }
        } else {
            return fmt.Errorf("create hackathon org: %w", err)
        }
    }

    // 2. Create hackathon record with linked org
    h.LinkedOrgID = org.ID
    h.OrgID = org.ID // also set OrgID for feed audience
    h.OwnerID = doer.ID
    if err := hackforger_model.CreateHackathon(ctx, h); err != nil {
        return err
    }

    // 3. Publish feed event
    _ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
        ActUserID:    doer.ID,
        OpType:       hackforger_model.ActionHackathonCreated,
        AudienceType: AudienceGlobal,
        Content: hackforger_model.HackforgerActionContent{
            EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
        },
    })
    return nil
}
```

Note: The function signature changes from `(ctx context.Context, doerID int64, h *Hackathon)` to `(ctx context.Context, doer *user_model.User, h *Hackathon)` — need to update all callers to pass `ctx.Doer` instead of `ctx.Doer.ID`.

Also need `structs "forgejo.org/modules/structs"` for `VisibleTypePublic`.

- [ ] **Step 2:** Add `CreateTrackWithRepo` service function.

New function in `services/hackforger/hackathon.go`:

```go
import (
    repo_service "forgejo.org/services/repository"
)

// CreateTrackWithRepo creates a track and its associated repository in the hackathon org.
func CreateTrackWithRepo(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, track *hackforger_model.HackathonTrack) error {
    // Get the hackathon's org (as a user for repo creation)
    orgUser, err := user_model.GetUserByID(ctx, h.LinkedOrgID)
    if err != nil {
        return fmt.Errorf("get hackathon org: %w", err)
    }

    // Create repo in the hackathon org
    repoName := strings.ToLower(strings.ReplaceAll(track.Name, " ", "-"))
    repo, err := repo_service.CreateRepository(ctx, doer, orgUser, repo_service.CreateRepoOptions{
        Name:        repoName,
        Description: track.Description,
        AutoInit:    true,
        DefaultBranch: "main",
        IsPrivate:   false,
    })
    if err != nil {
        return fmt.Errorf("create track repo: %w", err)
    }

    track.RepoID = repo.ID
    return hackforger_model.CreateTrack(ctx, track)
}
```

- [ ] **Step 3:** Update web handler callers.

In `routers/web/hackforger/hackathon.go`:

Update `NewHackathonPost` — change `hackforger_service.CreateHackathon(ctx, ctx.Doer.ID, h)` to `hackforger_service.CreateHackathon(ctx, ctx.Doer, h)`.

Update `ManageTrackPost` — change direct `hackforger_model.CreateTrack(ctx, t)` to `hackforger_service.CreateTrackWithRepo(ctx, ctx.Doer, h, t)`.

Update all other callers of `CreateHackathon` (state transition functions that pass `doerID`) — they need to accept `*user_model.User` or the caller needs to resolve the user. Actually, for state transitions the doer is just for feed events — keep those as `doerID int64`. Only `CreateHackathon` needs the full User.

- [ ] **Step 4:** Update API handler callers similarly.

In `routers/api/v1/hackforger/hackathon.go`: change `hackforger_service.CreateHackathon(ctx, ctx.Doer.ID, h)` to `hackforger_service.CreateHackathon(ctx, ctx.Doer, h)`.

- [ ] **Step 5:** Verify: `go build ./services/hackforger/... ./routers/...`

- [ ] **Step 6:** Commit:
```bash
git add services/hackforger/hackathon.go routers/web/hackforger/hackathon.go routers/api/v1/hackforger/hackathon.go
git commit -m "feat(hackforger): auto-create Org on hackathon creation, Repo on track creation

CreateHackathon now creates a Forgejo Organization (slug as org name).
CreateTrackWithRepo creates a Repository in that Org for each track."
```

---

## Chunk 4: Registration = Org Membership

### Task 4: Registration adds user to hackathon Org

**Files:**
- Modify: `routers/web/hackforger/hackathon.go` (`RegisterPost`, `ViewHackathon`)
- Modify: `routers/api/v1/hackforger/hackathon_registration.go`

- [ ] **Step 1:** Update `RegisterPost` web handler.

```go
import (
    organization_model "forgejo.org/models/organization"
)

func RegisterPost(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil { return }
    if h.Status != hackforger_model.HackathonStatusOpen {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }

    orgID, _ := strconv.ParseInt(ctx.FormString("org_id"), 10, 64)
    teamName := ctx.FormString("team_name")

    // Determine team name
    if orgID > 0 {
        // Verify org membership
        isMember, err := organization_model.IsOrganizationMember(ctx, orgID, ctx.Doer.ID)
        if err != nil || !isMember {
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.not_org_member"))
            ctx.Redirect("/hackathon/" + h.Slug)
            return
        }
        if org, err := organization_model.GetOrgByID(ctx, orgID); err == nil {
            teamName = org.Name
        }
    } else if teamName == "" {
        teamName = ctx.Doer.Name
    }

    r := &hackforger_model.HackathonRegistration{
        HackathonID: h.ID,
        UserID:      ctx.Doer.ID,
        OrgID:       orgID,
        TeamName:    teamName,
    }
    if err := hackforger_model.CreateRegistration(ctx, r); err != nil {
        if hackforger_model.IsErrDuplicateRegistration(err) {
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.already_registered"))
        } else {
            ctx.Flash.Error(err.Error())
        }
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }

    // Add user to hackathon org as member
    if h.LinkedOrgID > 0 {
        _ = organization_model.AddOrgUser(ctx, h.LinkedOrgID, ctx.Doer.ID)
    }

    ctx.Flash.Success(ctx.Tr("hackforger.hackathon.register.success"))
    ctx.Redirect("/hackathon/" + h.Slug)
}
```

- [ ] **Step 2:** Update `ViewHackathon` to pass user's orgs for registration dropdown.

Add to `ViewHackathon` (before `ctx.HTML`):
```go
if ctx.Doer != nil {
    orgs, _ := organization_model.GetUserOrgsList(ctx, ctx.Doer)
    ctx.Data["UserOrgs"] = orgs
}
```

- [ ] **Step 3:** Add i18n keys.

English:
```ini
hackathon.register.mode = Participate as
hackathon.register.solo = Individual
hackathon.register.team = Team
hackathon.register.display_name = Display Name
hackathon.register.display_name_hint = Optional — defaults to your username or org name
hackathon.error.not_org_member = You are not a member of this organization.
```

Chinese:
```ini
hackathon.register.mode = 参赛方式
hackathon.register.solo = 个人参赛
hackathon.register.team = 团队参赛
hackathon.register.display_name = 显示名称
hackathon.register.display_name_hint = 可选 — 默认使用用户名或组织名
hackathon.error.not_org_member = 你不是该组织的成员。
```

- [ ] **Step 4:** Verify and commit:
```bash
git add routers/web/hackforger/hackathon.go options/locale/
git commit -m "feat(hackforger): registration adds user to hackathon Org

Solo or team registration. Solo→Team auto-upgrade. User added to
hackathon's Forgejo Organization as member on registration."
```

---

## Chunk 5: Submission = Fork + PR

### Task 5: Submission forks track repo and creates PR

**Files:**
- Modify: `routers/web/hackforger/hackathon.go` (`SubmitPost`)
- Modify: `services/hackforger/hackathon.go` (new `CreateSubmission` service)

- [ ] **Step 1:** Create `CreateSubmission` service function.

In `services/hackforger/hackathon.go`:

```go
import (
    issues_model "forgejo.org/models/issues"
    repo_model "forgejo.org/models/repo"
    repo_service "forgejo.org/services/repository"
    pull_service "forgejo.org/services/pull"
)

// CreateSubmission forks the track repo to the user's space and creates a PR.
func CreateSubmission(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, sub *hackforger_model.HackathonSubmission) error {
    // 1. Get the track repo
    if sub.TrackID == 0 {
        // No track selected — just create submission record without fork/PR
        return hackforger_model.CreateSubmission(ctx, sub)
    }

    track, err := hackforger_model.GetTrackByID(ctx, sub.TrackID)
    if err != nil || track.RepoID == 0 {
        // Track has no associated repo — create submission without fork/PR
        return hackforger_model.CreateSubmission(ctx, sub)
    }

    baseRepo, err := repo_model.GetRepositoryByID(ctx, track.RepoID)
    if err != nil {
        return fmt.Errorf("get track repo: %w", err)
    }

    // 2. Fork the track repo to user's space
    fork, err := repo_service.ForkRepositoryIfNotExists(ctx, doer, doer, repo_service.ForkRepoOptions{
        BaseRepo:    baseRepo,
        Name:        baseRepo.Name + "-" + doer.LowerName,
        Description: sub.Title,
    })
    if err != nil {
        // If fork already exists, find it
        if repo_service.IsErrForkAlreadyExist(err) {
            fork, err = repo_model.GetUserFork(ctx, baseRepo.ID, doer.ID)
            if err != nil {
                return fmt.Errorf("get existing fork: %w", err)
            }
        } else {
            return fmt.Errorf("fork track repo: %w", err)
        }
    }
    sub.ForkRepoID = fork.ID
    sub.RepoID = fork.ID

    // 3. Create a PR from fork to track repo
    issue := &issues_model.Issue{
        RepoID:   baseRepo.ID,
        Title:    sub.Title,
        Content:  sub.Description,
        PosterID: doer.ID,
    }
    pr := &issues_model.PullRequest{
        HeadRepoID: fork.ID,
        BaseRepoID: baseRepo.ID,
        HeadBranch: fork.DefaultBranch,
        BaseBranch: baseRepo.DefaultBranch,
        Type:       issues_model.PullRequestGitea,
    }
    if err := pull_service.NewPullRequest(ctx, baseRepo, issue, nil, nil, pr, nil); err != nil {
        // PR creation may fail if no commits differ — that's ok, just save submission
        log.Warn("CreateSubmission: PR creation failed (may be no diff): %v", err)
    } else {
        sub.PRID = pr.ID
        sub.PullIndex = issue.Index
    }

    // 4. Save submission record
    if err := hackforger_model.CreateSubmission(ctx, sub); err != nil {
        return err
    }

    // 5. Feed event
    _ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
        ActUserID:    doer.ID,
        OpType:       hackforger_model.ActionHackathonSubmitted,
        AudienceType: AudienceFollowers,
        RepoID:       baseRepo.ID,
        Content: hackforger_model.HackforgerActionContent{
            EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
            Extra: map[string]any{"submission_id": sub.ID, "title": sub.Title},
        },
    })
    return nil
}
```

- [ ] **Step 2:** Update `SubmitPost` web handler to call the service.

Replace direct `hackforger_model.CreateSubmission(ctx, s)` with:
```go
if err := hackforger_service.CreateSubmission(ctx, ctx.Doer, h, s); err != nil {
    ctx.Flash.Error(err.Error())
    ctx.Redirect("/hackathon/" + h.Slug + "/submit")
    return
}
```

Remove the inline feed event publishing in the web handler (now in the service).

- [ ] **Step 3:** Also update the API handler similarly.

- [ ] **Step 4:** Verify and commit:
```bash
git add services/hackforger/hackathon.go routers/web/hackforger/hackathon.go routers/api/v1/hackforger/hackathon_submission.go
git commit -m "feat(hackforger): submission forks track repo and creates PR

CreateSubmission now forks the track repo to user's space and creates
a PR. Submission record stores fork_repo_id, pr_id, and pull_index."
```

---

## Chunk 6: View Page Redesign

### Task 6: Redesign hackathon detail page

**Files:**
- Modify: `templates/hackforger/hackathon/view.tmpl`
- Modify: `routers/web/hackforger/hackathon.go` (`ViewHackathon`)
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1:** Update `ViewHackathon` handler to load all data.

```go
func ViewHackathon(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil { return }
    ctx.Data["Title"] = h.Name
    ctx.Data["Hackathon"] = h
    ctx.Data["StatusLabel"] = hackforger_service.HackathonStatusLabel(h.Status)

    // Owner
    if h.OwnerID > 0 {
        if owner, err := user_model.GetUserByID(ctx, h.OwnerID); err == nil {
            ctx.Data["Owner"] = owner
        }
    }

    // Linked Org
    if h.LinkedOrgID > 0 {
        if org, err := user_model.GetUserByID(ctx, h.LinkedOrgID); err == nil {
            ctx.Data["LinkedOrg"] = org
        }
    }

    // Tracks (with repo info)
    tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
    ctx.Data["Tracks"] = tracks

    // Registrations
    regs, regCount, _ := hackforger_model.ListRegistrations(ctx, hackforger_model.ListRegistrationsOptions{HackathonID: h.ID})
    ctx.Data["Registrations"] = regs
    ctx.Data["RegistrationCount"] = regCount

    // Submissions (with PR links)
    subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{HackathonID: h.ID})
    ctx.Data["Submissions"] = subs

    // Judges
    judges, _ := hackforger_model.ListJudges(ctx, h.ID)
    ctx.Data["JudgeCount"] = len(judges)

    // Current user
    if ctx.Doer != nil {
        _, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID)
        ctx.Data["IsRegistered"] = err == nil
        ctx.Data["SignedUserID"] = ctx.Doer.ID
        ctx.Data["IsOwner"] = ctx.Doer.ID == h.OwnerID
        isJudge, _ := hackforger_model.IsJudge(ctx, h.ID, ctx.Doer.ID)
        ctx.Data["IsJudge"] = isJudge
        orgs, _ := organization_model.GetUserOrgsList(ctx, ctx.Doer)
        ctx.Data["UserOrgs"] = orgs
    }

    ctx.HTML(http.StatusOK, tplView)
}
```

- [ ] **Step 2:** Rewrite `view.tmpl`.

Full template covering: header (name + status + organizer + org link + manage/judge buttons), timeline, description, prizes, tracks (with repo links), registration form (solo/team), registered indicator, participants list, submit button, submissions table (with PR links), leaderboard link.

Key additions vs current:
- **Org link**: `<a href="{{AppSubUrl}}/{{.LinkedOrg.Name}}">`
- **Track repo links**: `{{if .RepoID}}<a href="...">{{svg "octicon-repo"}} Code</a>{{end}}`
- **Submission PR links**: `{{if .PullIndex}}<a href="...">PR #{{.PullIndex}}</a>{{end}}`
- **Timeline**: dates from `RegistrationEnd`, `HackingEnd`, `JudgingEnd`
- **Org registration dropdown**: solo or select from user's orgs

(Template content is the full redesign from the previous plan version — see the `view.tmpl` in Chunk 3 of the old plan, enhanced with org/repo links.)

- [ ] **Step 3:** Add i18n keys (both locales):

```ini
; English
hackathon.view.organizer = Organizer
hackathon.view.org = Organization
hackathon.view.timeline = Timeline
hackathon.view.registration_end = Registration Ends
hackathon.view.hacking_end = Hacking Ends
hackathon.view.judging_end = Judging Ends
hackathon.view.prizes = Prizes
hackathon.view.participants = Participants
hackathon.view.submissions = Submissions
hackathon.view.pending = Pending
hackathon.view.approved = Approved
hackathon.view.rejected = Rejected
hackathon.view.pr = Pull Request
hackathon.view.code = Code

; Chinese
hackathon.view.organizer = 组织者
hackathon.view.org = 组织
hackathon.view.timeline = 时间线
hackathon.view.registration_end = 报名截止
hackathon.view.hacking_end = 开发截止
hackathon.view.judging_end = 评审截止
hackathon.view.prizes = 奖品
hackathon.view.participants = 参与者
hackathon.view.submissions = 提交作品
hackathon.view.pending = 待审核
hackathon.view.approved = 已通过
hackathon.view.rejected = 已拒绝
hackathon.view.pr = 拉取请求
hackathon.view.code = 代码
```

- [ ] **Step 4:** Verify and commit:
```bash
git add templates/hackforger/hackathon/view.tmpl routers/web/hackforger/hackathon.go options/locale/
git commit -m "feat(hackforger): redesign hackathon detail page with Org/Repo links

Timeline, organizer info, org link, track repo links, org-based
registration form, participants list, submission PR links,
phase-appropriate CTAs."
```

---

## Chunk 7: Error Handling UX

### Task 7: Flash messages for all error paths

**Files:**
- Modify: `routers/web/hackforger/hackathon.go`

- [ ] **Step 1:** Fix `ManagePhasePost` — always redirect with flash, never return raw HTTP error.

Re-fetch hackathon by ID before state transition (in case concurrent update changed status):
```go
h, err := hackforger_model.GetHackathonByID(ctx, h.ID)
```

Replace `ctx.NotFound(...)` with `ctx.Flash.Error(...) + ctx.Redirect(...)`.

- [ ] **Step 2:** Fix `SubmitForm` — check phase on GET and redirect with flash if wrong phase.

- [ ] **Step 3:** Verify and commit:
```bash
git add routers/web/hackforger/hackathon.go
git commit -m "fix(hackforger): flash messages for all error responses"
```

---

## Chunk 8: Build + E2E

### Task 8: Full rebuild, test data cleanup, E2E verification

- [ ] **Step 1:** Kill server, rebuild, restart:
```bash
kill $(lsof -t -i :3000) 2>/dev/null
TAGS="bindata sqlite sqlite_unlock_notify" make build
rm -f /Users/h2oslabs/Workspace/hackforger/data/queues/common/LOCK
./gitea web
```

- [ ] **Step 2:** Clean test data:
```sql
DELETE FROM hackathon_judge_score;
DELETE FROM hackathon_judge;
DELETE FROM hackathon_submission;
DELETE FROM hackathon_registration;
DELETE FROM hackathon_track;
DELETE FROM hackathon;
DELETE FROM action WHERE op_type >= 30 AND op_type <= 60;
```

- [ ] **Step 3:** E2E test — verify:
1. Creating hackathon also creates an Org (check `/spring-hack-2026` org page)
2. Adding track creates a Repo in the org (check `/spring-hack-2026/ai-track`)
3. Registration adds user to org (check org member list)
4. Registration form shows solo/team dropdown
5. Submission forks track repo and creates PR
6. Detail page shows timeline, participants, tracks with repo links, submissions with PR links
7. Dashboard feed shows hackathon events with rocket icon and Chinese text
8. All manage buttons work in browser (CrossOriginProtection fix)
9. Error cases show flash messages

- [ ] **Step 4:** Write E2E report, commit and push.

---

## Manual E2E Prompt Addendum

After all tasks are implemented, add these verification steps to the E2E prompt:

```
### Org/Repo Integration Verification

1. After creating hackathon "Spring Hack 2026":
   - Navigate to /spring-hack-2026 → should show the Org page
   - Org owner should be the admin user

2. After adding "AI Track":
   - Navigate to /spring-hack-2026/ai-track → should show a Repo page
   - Repo should have auto-init commit on main branch

3. After registering as hacker_eve:
   - Navigate to /spring-hack-2026 (org page) → Members should include hacker_eve

4. After submitting project:
   - hacker_eve should have a fork: /hacker_eve/ai-track-hacker_eve
   - A PR should exist on /spring-hack-2026/ai-track/pulls

5. On hackathon detail page:
   - "Organization" link goes to org page
   - Each track has a "Code" link to repo
   - Each submission has a "PR #N" link
```
