# Simplify Registration Flow Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the individual/team registration dropdown with a repo-based flow where selecting a repo at registration time determines participation type.

**Architecture:** Merge the registration and submission forms into a single repo-based form. Registration phase creates both a Registration and Submission record. Development phase updates the existing Submission. OrgID/TeamName are auto-derived from the selected repo's owner.

**Tech Stack:** Go handlers, Go templates (SSR), XORM, Forgejo repo model APIs

---

### Task 1: Update view handler to load repos instead of orgs

**Files:**
- Modify: `routers/web/hackforger/hackathon.go:255-268` (ViewHackathon, user data section)

**Step 1: Replace UserOrgs with UserRepos in ViewHackathon**

In the `ViewHackathon` function, find lines 266-267:
```go
orgs, _ := organization_model.GetUserOrgsList(ctx, ctx.Doer)
ctx.Data["UserOrgs"] = orgs
```

Replace with:
```go
repos, _, _ := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
    Actor:   ctx.Doer,
    OwnerID: ctx.Doer.ID,
    Private: true,
})
ctx.Data["UserRepos"] = repos

tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
// Tracks already loaded above at line 222, reuse ctx.Data["Tracks"]
```

Also add `repo_model` to imports if not already present.

**Step 2: Verify compilation**

Run: `cd /path/to/worktree && TAGS="bindata sqlite sqlite_unlock_notify" go build -tags "bindata sqlite sqlite_unlock_notify" ./routers/web/hackforger/`

**Step 3: Commit**

```
git add routers/web/hackforger/hackathon.go
git commit -m "refactor: load user repos instead of orgs for registration view"
```

---

### Task 2: Rewrite the registration form template

**Files:**
- Modify: `templates/hackforger/hackathon/view.tmpl:80-101` (Registration CTA section)

**Step 1: Replace the registration form**

Find lines 80-101 (the `Registration CTA` section) and replace with:

```html
{{/* Registration CTA */}}
{{if and (eq .Hackathon.StatusCache 1) .SignedUserID (not .IsRegistered)}}
<div class="hf-card">
    <div class="hf-card-head">
        <div class="hf-card-head-left">{{svg "octicon-person-add" 16}} {{ctx.Locale.Tr "hackforger.hackathon.register"}}</div>
    </div>
    <div class="hf-card-body">
        <form method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/register" class="ui form">
            <div class="required field">
                <label>{{ctx.Locale.Tr "hackforger.hackathon.submit.field.repo"}}</label>
                <select name="repo_id" class="ui search selection dropdown" required>
                    <option value="">— {{ctx.Locale.Tr "hackforger.hackathon.register.select_repo"}} —</option>
                    {{range .UserRepos}}
                    <option value="{{.ID}}" {{if eq .ID $.PreselectedRepoID}}selected{{end}}>{{.FullName}}</option>
                    {{end}}
                </select>
                <a href="{{AppSubUrl}}/repo/create?redirect_to={{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}" class="tw-text-sm tw-mt-1 tw-inline-block">+ {{ctx.Locale.Tr "hackforger.hackathon.register.create_repo"}}</a>
            </div>
            {{if .Tracks}}
            <div class="required field">
                <label>{{ctx.Locale.Tr "hackforger.hackathon.submit.field.track"}}</label>
                <select name="track_id" class="ui selection dropdown" required>
                    <option value="">— {{ctx.Locale.Tr "hackforger.hackathon.register.select_track"}} —</option>
                    {{range .Tracks}}
                    <option value="{{.ID}}">{{.Name}}</option>
                    {{end}}
                </select>
            </div>
            {{end}}
            <div class="required field">
                <label>{{ctx.Locale.Tr "hackforger.hackathon.submit.field.title"}}</label>
                <input name="title" required placeholder="{{ctx.Locale.Tr "hackforger.hackathon.submit.field.title"}}">
            </div>
            <div class="field">
                <label>{{ctx.Locale.Tr "hackforger.hackathon.submit.field.description"}}</label>
                <textarea name="description" rows="3"></textarea>
            </div>
            <div class="field">
                <label>{{ctx.Locale.Tr "hackforger.hackathon.submit.field.demo_url"}}</label>
                <input name="demo_url" type="url">
            </div>
            <button class="hf-btn hf-btn-primary" type="submit">{{ctx.Locale.Tr "hackforger.hackathon.register"}}</button>
        </form>
    </div>
</div>
{{end}}
```

**Step 2: Handle preselected repo from redirect**

In `ViewHackathon` handler, add after the repos loading:
```go
preselectedRepoID := ctx.FormInt64("repo_id")
ctx.Data["PreselectedRepoID"] = preselectedRepoID
```

**Step 3: Commit**

```
git add templates/hackforger/hackathon/view.tmpl routers/web/hackforger/hackathon.go
git commit -m "feat: replace individual/team dropdown with repo-based registration form"
```

---

### Task 3: Rewrite RegisterPost handler

**Files:**
- Modify: `routers/web/hackforger/hackathon.go:291-394` (RegisterPost function)

**Step 1: Rewrite RegisterPost**

Replace the entire `RegisterPost` function body (keeping the signature). New logic:

```go
func RegisterPost(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil {
        return
    }

    // Block organizer self-registration
    if h.OwnerID == ctx.Doer.ID {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.cannot_register_own"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }
    if h.LinkedOrgID > 0 {
        if org, err := organization_model.GetOrgByID(ctx, h.LinkedOrgID); err == nil {
            if isOwner, _ := org.IsOwnedBy(ctx, ctx.Doer.ID); isOwner {
                ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.cannot_register_own"))
                ctx.Redirect("/hackathon/" + h.Slug)
                return
            }
        }
    }

    // Check duplicate registration
    if _, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID); err == nil {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.already_registered"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }

    // Phase check
    canRegister, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register")
    if !canRegister {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }

    // Judges cannot register
    judges, _ := hackforger_model.ListJudges(ctx, h.ID)
    for _, j := range judges {
        if j.UserID == ctx.Doer.ID {
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.judge_cannot_register"))
            ctx.Redirect("/hackathon/" + h.Slug)
            return
        }
    }

    // Parse repo selection
    repoID := ctx.FormInt64("repo_id")
    if repoID <= 0 {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.repo_required"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }

    // Verify repo ownership
    repo, err := repo_model.GetRepositoryByID(ctx, repoID)
    if err != nil {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.repo_required"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }

    // Check user has access: repo owner is the user, or user is member of the owning org
    hasAccess := repo.OwnerID == ctx.Doer.ID
    if !hasAccess {
        isMember, _ := organization_model.IsOrganizationMember(ctx, repo.OwnerID, ctx.Doer.ID)
        hasAccess = isMember
    }
    if !hasAccess {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.repo_required"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }

    // Derive OrgID and TeamName from repo owner
    var orgID int64
    var teamName string
    repoOwner, _ := user_model.GetUserByID(ctx, repo.OwnerID)
    if repoOwner != nil && repoOwner.IsOrganization() {
        orgID = repoOwner.ID
        teamName = repoOwner.Name
    } else {
        teamName = ctx.Doer.Name
    }

    trackID := ctx.FormInt64("track_id")

    // Create registration
    r := &hackforger_model.HackathonRegistration{
        HackathonID: h.ID,
        UserID:      ctx.Doer.ID,
        OrgID:       orgID,
        TeamName:    teamName,
        RepoID:      repoID,
        TrackID:     trackID,
        Status:      hackforger_model.RegistrationStatusApproved,
    }
    if err := hackforger_model.CreateRegistration(ctx, r); err != nil {
        if hackforger_model.IsErrDuplicateRegistration(err) {
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.already_registered"))
        } else {
            log.Error("CreateRegistration: %v", err)
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
        }
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }

    // Simultaneously create submission
    title := ctx.FormString("title")
    if title == "" {
        title = repo.Name
    }
    s := &hackforger_model.HackathonSubmission{
        HackathonID:    h.ID,
        RegistrationID: r.ID,
        UserID:         ctx.Doer.ID,
        Title:          title,
        Description:    ctx.FormString("description"),
        DemoURL:        ctx.FormString("demo_url"),
        TrackID:        trackID,
        RepoID:         repoID,
        Status:         hackforger_model.SubmissionStatusSubmitted,
    }
    if err := hackforger_service.CreateSubmission(ctx, ctx.Doer, h, s); err != nil {
        log.Error("CreateSubmission on register: %v", err)
        // Registration succeeded, submission failed — not critical, user can submit later
    }

    // Add user to hackathon org
    if h.LinkedOrgID > 0 {
        _ = organization_model.AddOrgUser(ctx, h.LinkedOrgID, ctx.Doer.ID)
    }

    // Publish feed event
    notify_service.HackforgerEntityCreated(ctx, ctx.Doer, &notify_service.HackforgerEventOpts{
        OpType:       hackforger_model.ActionHackathonRegistered,
        EntityType:   "hackathon",
        EntityID:     h.ID,
        EntityName:   h.Name,
        EntitySlug:   h.Slug,
        AudienceType: notify_service.AudienceFollowers,
        Content: hackforger_model.HackforgerActionContent{
            EntityType: "hackathon",
            EntityID:   h.ID,
            EntityName: h.Name,
            EntitySlug: h.Slug,
        },
    })

    ctx.Flash.Success(ctx.Tr("hackforger.hackathon.register.success"))
    ctx.Redirect("/hackathon/" + h.Slug)
}
```

**Step 2: Add `repo_model` import if missing**

Check imports at top of file — ensure `repo_model "forgejo.org/models/repo"` is present.

**Step 3: Verify compilation**

**Step 4: Commit**

```
git add routers/web/hackforger/hackathon.go
git commit -m "feat: rewrite RegisterPost to create registration + submission from repo selection"
```

---

### Task 4: Update SubmitForm to filter repos for development phase

**Files:**
- Modify: `routers/web/hackforger/hackathon.go:396-428` (SubmitForm function)

**Step 1: Filter repos to registered ones during development**

In `SubmitForm`, after loading user repos (line 413-417), add filtering logic:

```go
// During development phase, only show repos already registered
reg, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID)
if err == nil && reg.RepoID > 0 {
    // Filter to just the registered repo
    filteredRepos := make(repo_model.RepositoryList, 0)
    for _, r := range repos {
        if r.ID == reg.RepoID {
            filteredRepos = append(filteredRepos, r)
        }
    }
    ctx.Data["UserRepos"] = filteredRepos
    ctx.Data["LockedRepoID"] = reg.RepoID
}
```

**Step 2: Commit**

```
git add routers/web/hackforger/hackathon.go
git commit -m "feat: lock repo selection to registered repo during development phase"
```

---

### Task 5: Add i18n keys

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

**Step 1: Add keys to en-US**

Find `hackathon.register.` section and add:

```ini
hackathon.register.select_repo = Select a repository...
hackathon.register.create_repo = Create New Repository
hackathon.register.select_track = Select a track...
hackathon.register.repo_required = Please select a repository.
```

**Step 2: Add keys to zh-CN**

```ini
hackathon.register.select_repo = 选择一个仓库...
hackathon.register.create_repo = 创建新仓库
hackathon.register.select_track = 选择赛道...
hackathon.register.repo_required = 请选择一个仓库。
```

**Step 3: Commit**

```
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "feat: add i18n keys for repo-based registration flow"
```

---

### Task 6: Update participants display

**Files:**
- Modify: `templates/hackforger/hackathon/view.tmpl:114-123` (Participants section)

**Step 1: Show repo name instead of team name**

Replace the participant row display (lines 114-123):

```html
{{range .Registrations}}
<div class="hf-row">
    <div class="hf-row-left">
        <span>{{.TeamName}}</span>
        {{if .RepoID}}
            {{$repo := index $.RegRepos .RepoID}}
            {{if $repo}} · <a href="{{AppSubUrl}}/{{$repo}}">{{$repo}}</a>{{end}}
        {{end}}
    </div>
    <span class="hf-badge {{if eq .Status 0}}hf-badge-yellow{{else if eq .Status 1}}hf-badge-green{{else}}hf-badge-red{{end}}">
        {{if eq .Status 0}}{{ctx.Locale.Tr "hackforger.hackathon.view.pending"}}
        {{else if eq .Status 1}}{{ctx.Locale.Tr "hackforger.hackathon.view.approved"}}
        {{else}}{{ctx.Locale.Tr "hackforger.hackathon.view.rejected"}}{{end}}
    </span>
</div>
{{end}}
```

**Step 2: Build repo name map in ViewHackathon handler**

In `ViewHackathon`, after loading registrations (line 234-236), add:

```go
regRepos := make(map[int64]string)
for _, r := range regs {
    if r.RepoID > 0 {
        if repo, err := repo_model.GetRepositoryByID(ctx, r.RepoID); err == nil {
            regRepos[r.RepoID] = repo.FullName()
        }
    }
}
ctx.Data["RegRepos"] = regRepos
```

**Step 3: Commit**

```
git add templates/hackforger/hackathon/view.tmpl routers/web/hackforger/hackathon.go
git commit -m "feat: show repo name in participant list"
```

---

### Task 7: Remove unused org import and clean up

**Files:**
- Modify: `routers/web/hackforger/hackathon.go`

**Step 1: Check if `organization_model.GetUserOrgsList` is still used anywhere**

If the only usage was in `ViewHackathon` (now removed), ensure the import is still needed for other functions (RegisterPost still uses `organization_model` for org member checks).

**Step 2: Build full project**

Run: `make frontend && TAGS="bindata sqlite sqlite_unlock_notify" make backend`

**Step 3: Final commit**

```
git add -A
git commit -m "chore: clean up unused code after registration simplification"
```

---

### Task 8: Push and create PR

**Step 1: Push branch**

```
git push origin fix/simplify-registration
```

**Step 2: Create PR**

```
gh pr create --base v0.1-dev/hackforger --head fix/simplify-registration \
  --title "feat: simplify registration to repo-based flow (#55)" \
  --body "..."
```

---

### E2E Test Prompt

After implementation, run E2E testing via agent-browser at `http://localhost:3000`:

1. Login as hackforger/admin1234
2. Navigate to a hackathon in Registration phase
3. Verify: repo dropdown visible (no individual/team dropdown)
4. Verify: "创建新仓库" link present
5. Select a repo, select a track, fill title → submit
6. Verify: Registration created, Submission created
7. Verify: Participant list shows username + repo name
8. Navigate to same hackathon in Development phase
9. Verify: Submit form only shows the registered repo (locked)
