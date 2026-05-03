# Hackathon Registration: Allow Org Repos (Web + API parity)

**Date:** 2026-05-03
**Issue origin:** linyilun (member of `H2OSLabs`) tried to register `opc-2026-shuzhi-w1` with the org-owned repo `H2OSLabs/page.h2oslabs.com`, but the repo did not appear in the dropdown — and in fact **no repo at all appeared**, because the dropdown query is restricted to the user's personally-owned repos.

## 1. Summary

Fix two related defects in one PR:

1. **Web defect**: the registration repo dropdown silently excludes all org-owned repos (and all collaborator repos). Users who are org members with write access cannot see those repos as registration targets.
2. **API defect**: `POST /api/v1/hackforger/hackathons/{id}/registrations` accepts only `team_name` + `track_id` — no `repo_id`. So even though the DB schema supports `OrgID` / `RepoID` and the web handler derives them from the chosen repo, **API callers cannot do team/org registration at all**.

Both stem from the same gap: the project never had a single-source-of-truth helper for "which repos can this user use to register a hackathon entry."

## 2. Goals & Non-Goals

**Goals**
- Org-owned repos with write access show up in the web registration dropdown.
- API registration accepts an optional `repo_id` and applies the same org/team derivation, submission creation, and `AddOrgUser` side-effect as the web handler.
- Both code paths share **one** helper for "list writable repos" and "validate-and-load a writable repo".
- Permission threshold: **Write or higher** (per user decision, 2026-05-03). Read-only repos do NOT appear.

**Non-Goals**
- No DB schema change (`OrgID`, `TeamID`, `RepoID` already exist on `hackathon_registration`).
- No changes to grant/bounty registration flow (separate paths, separate fix if needed later).
- No UI redesign of the registration form — only the dropdown contents change.
- No backward-incompatible changes to API: `repo_id` is added as optional; existing API callers continue to work (solo registration with a synthetic team name).

## 3. Root Cause

```go
// routers/web/hackforger/hackathon.go:285-290 and 490-495 (two identical sites)
repos, _, _ := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
    Actor:       ctx.Doer,
    OwnerID:     ctx.Doer.ID,         // ← restricts to owner=user
    Private:     true,
    Collaborate: optional.Some(false), // ← excludes collaborator/org-member access
})
ctx.Data["UserRepos"] = repos
```

The combination of `OwnerID = ctx.Doer.ID` and `Collaborate = false` returns only repos directly under the user's namespace. Org repos (where `Owner.Type == Organization`) are excluded even when the user is an org owner.

The API side `routers/api/v1/hackforger/hackathon_registration.go::RegisterForm` is even more limited:

```go
type RegisterForm struct {
    TeamName string `json:"team_name" binding:"Required"`
    TrackID  int64  `json:"track_id"`
}
```

No `RepoID` means the API cannot bind to any repo — solo-only.

## 4. Design — Approach B (extract helper)

Add a small helper module at `services/hackforger/repos.go` exposing two functions, then call them from web (2 sites) and API (1 site).

### 4.1 New file: `services/hackforger/repos.go`

```go
// Package hackforger — helper for hackathon registration's "which repos can this
// user use" question. Extracted to share between web (templates dropdown) and
// API (POST validation).
package hackforger

import (
    "context"

    access_model "forgejo.org/models/perm/access"
    perm_model "forgejo.org/models/perm"
    repo_model "forgejo.org/models/repo"
    user_model "forgejo.org/models/user"
)

// ListUserWritableRepos returns every repository the user has at least Write
// access to, including org-owned repos. Used to populate the registration
// repo dropdown.
//
// Order: by recent activity (whatever SearchRepository returns naturally).
// Filtering: post-fetch by access level — SearchRepository's permission filter
// is broader than we need.
func ListUserWritableRepos(ctx context.Context, user *user_model.User) ([]*repo_model.Repository, error) {
    // Fetch all visible repos for this user (owned + org + collaborator).
    repos, _, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
        Actor:           user,
        Private:         true,
        AllPublic:       false,
        AllLimited:      false,
        IncludeDescription: false,
        ListOptions:     db.ListOptions{Page: 1, PageSize: 10},   // dropdown cap (kept short — 10 is the dev-friendly max)
    })
    if err != nil {
        return nil, err
    }

    // Post-filter: keep only repos where user has >= Write.
    out := make([]*repo_model.Repository, 0, len(repos))
    for _, r := range repos {
        perm, err := access_model.GetUserRepoPermission(ctx, r, user)
        if err != nil {
            continue
        }
        if perm.AccessMode >= perm_model.AccessModeWrite {
            out = append(out, r)
        }
    }
    return out, nil
}

// UserCanRegisterRepo loads a repo by ID and verifies the user has at least
// Write access to it. Returns the loaded repo (with Owner loaded) on success.
//
// Returns ErrRepoAccessDenied if the user lacks write permission.
// Returns repo_model.ErrRepoNotExist if the repo doesn't exist.
func UserCanRegisterRepo(ctx context.Context, user *user_model.User, repoID int64) (*repo_model.Repository, error) {
    repo, err := repo_model.GetRepositoryByID(ctx, repoID)
    if err != nil {
        return nil, err  // includes ErrRepoNotExist
    }
    if err := repo.LoadOwner(ctx); err != nil {
        return nil, err
    }
    perm, err := access_model.GetUserRepoPermission(ctx, repo, user)
    if err != nil {
        return nil, err
    }
    if perm.AccessMode < perm_model.AccessModeWrite {
        return nil, ErrRepoAccessDenied{UserID: user.ID, RepoID: repoID}
    }
    return repo, nil
}

// ErrRepoAccessDenied is returned when a user tries to register with a repo
// they don't have write access to.
type ErrRepoAccessDenied struct {
    UserID int64
    RepoID int64
}

func (e ErrRepoAccessDenied) Error() string {
    return fmt.Sprintf("user %d lacks write permission to repo %d", e.UserID, e.RepoID)
}

func IsErrRepoAccessDenied(err error) bool {
    _, ok := err.(ErrRepoAccessDenied)
    return ok
}
```

### 4.2 Web changes — `routers/web/hackforger/hackathon.go`

Two sites currently call the broken `SearchRepository`. Replace both with `ListUserWritableRepos`:

```go
// Was:
//   repos, _, _ := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
//       Actor: ctx.Doer, OwnerID: ctx.Doer.ID, Private: true,
//       Collaborate: optional.Some(false),
//   })
// Now:
repos, err := hackforger_service.ListUserWritableRepos(ctx, ctx.Doer)
if err != nil {
    log.Warn("ListUserWritableRepos: %v", err)
    repos = nil  // form still renders with empty dropdown + helpful message
}
ctx.Data["UserRepos"] = repos
```

Existing `RegisterPost` access check (lines 380–390) is **replaced** with a `UserCanRegisterRepo` call:

```go
// Was: manual `repo.OwnerID == ctx.Doer.ID || IsOrganizationMember`
// Now:
repo, err := hackforger_service.UserCanRegisterRepo(ctx, ctx.Doer, repoID)
if err != nil {
    if hackforger_service.IsErrRepoAccessDenied(err) || repo_model.IsErrRepoNotExist(err) {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.repo_required"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }
    log.Error("UserCanRegisterRepo: %v", err)
    ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
    ctx.Redirect("/hackathon/" + h.Slug)
    return
}
// repo is now loaded with Owner; existing org/team derivation continues unchanged
```

The org/team derivation, registration row creation, submission creation, and `AddOrgUser` side-effect that follow are kept as-is.

### 4.3 API changes — `routers/api/v1/hackforger/hackathon_registration.go`

Extend `RegisterForm`:

```go
type RegisterForm struct {
    TeamName    string `json:"team_name" binding:"Required"`
    TrackID     int64  `json:"track_id"`
    RepoID      int64  `json:"repo_id,omitempty"`        // NEW: optional; if 0, solo registration
    Title       string `json:"title,omitempty"`           // NEW: submission title (defaults to repo name)
    Description string `json:"description,omitempty"`     // NEW: submission description
    DemoURL     string `json:"demo_url,omitempty"`        // NEW: submission demo URL
}
```

`Register` handler gains a branch for `RepoID > 0`:

```go
r := &hackforger_model.HackathonRegistration{
    HackathonID: h.ID,
    UserID:      ctx.Doer.ID,
    TeamName:    f.TeamName,
    TrackID:     f.TrackID,
}

if f.RepoID > 0 {
    repo, err := hackforger_service.UserCanRegisterRepo(ctx, ctx.Doer, f.RepoID)
    if err != nil {
        if hackforger_service.IsErrRepoAccessDenied(err) {
            ctx.Error(http.StatusForbidden, "RepoAccessDenied", err)
            return
        }
        if repo_model.IsErrRepoNotExist(err) {
            ctx.Error(http.StatusNotFound, "RepoNotExist", err)
            return
        }
        ctx.InternalServerError(err)
        return
    }
    r.RepoID = repo.ID
    if repo.Owner.IsOrganization() {
        r.OrgID = repo.Owner.ID
        r.TeamName = repo.Owner.Name  // override caller-supplied TeamName for team registration
    }
    r.Status = hackforger_model.RegistrationStatusApproved  // match web behaviour
}

if err := hackforger_model.CreateRegistration(ctx, r); err != nil {
    // ... existing duplicate handling
}

// If repo was provided, also create the submission (parity with web)
if f.RepoID > 0 {
    title := f.Title
    if title == "" {
        title = repo.Name
    }
    s := &hackforger_model.HackathonSubmission{
        HackathonID:    h.ID,
        RegistrationID: r.ID,
        UserID:         ctx.Doer.ID,
        Title:          title,
        Description:    f.Description,
        DemoURL:        f.DemoURL,
        TrackID:        f.TrackID,
        RepoID:         f.RepoID,
        Status:         hackforger_model.SubmissionStatusSubmitted,
    }
    if err := hackforger_service.CreateSubmission(ctx, ctx.Doer, h, s); err != nil {
        log.Error("CreateSubmission on register: %v", err)
        // Don't fail the registration — log and continue (parity with web)
    }
}

// If linked org exists, add user to it (parity with web)
if h.LinkedOrgID > 0 {
    _ = organization_model.AddOrgUser(ctx, h.LinkedOrgID, ctx.Doer.ID)
}

// Existing notify_service.HackforgerEntityCreated call unchanged.
```

### 4.4 Backward compatibility

Existing API callers that pass `{team_name, track_id}` only continue to work — `RepoID == 0` triggers the original solo-registration path. Existing web flow with `repo_id` form param continues to work — same handler, same derivation, just access-checked through the helper.

## 5. Test Plan

### 5.1 Unit tests (`services/hackforger/repos_test.go`)
- `ListUserWritableRepos` returns user's own repos
- `ListUserWritableRepos` returns org repos where user is org owner
- `ListUserWritableRepos` excludes org repos where user is read-only member
- `ListUserWritableRepos` returns collaborator repos with Write+ access
- `UserCanRegisterRepo` returns repo for owner
- `UserCanRegisterRepo` returns `ErrRepoAccessDenied` for read-only access
- `UserCanRegisterRepo` returns `ErrRepoNotExist` for non-existent ID

### 5.2 E2E (web, agent-browser at hackforger.inside.h2os.cloud)
The existing report template at `docs/tests/e2e/reports/` will be filled in. Acceptance:

- [ ] Login as `linyilun` (member of `H2OSLabs` org)
- [ ] Navigate to a hackathon registration page
- [ ] Repo dropdown lists `H2OSLabs/page.h2oslabs.com` (and any other org repos linyilun can write)
- [ ] Select that repo, fill team name, click 报名
- [ ] Registration succeeds, lands on hackathon page
- [ ] Registration row has `OrgID = H2OSLabs.ID`, `TeamName = "H2OSLabs"`, `RepoID = page.h2oslabs.com.ID`
- [ ] Submission row exists with same `RepoID`
- [ ] linyilun is now a member of the hackathon's `LinkedOrg` (if set)

### 5.3 E2E (API, curl)
- [ ] `POST /api/v1/hackforger/hackathons/{id}/registrations` with `{team_name, track_id}` only → solo registration succeeds (no regression)
- [ ] Same endpoint with `{team_name, track_id, repo_id: <org-repo-id>}` as a writer → registration created with `OrgID/RepoID` set, submission created
- [ ] Same with `repo_id` of a repo the user can only read → 403 RepoAccessDenied
- [ ] Same with non-existent `repo_id` → 404 RepoNotExist

### 5.4 Manual verification gates
- Test on **Mac dev** instance (port 3000) first: build, restart, run web + API tests
- Test on **prod cloud** (`https://www.synnovator.com`) only after admin sign-off in the smoke-test report (per `docs/notes/gitflow.md`)
- Admin will personally walk through at `https://hackforger.inside.h2os.cloud` (Mac instance) before promoting to cloud

## 6. Files Touched

| File | Change |
| --- | --- |
| `services/hackforger/repos.go` | NEW — `ListUserWritableRepos`, `UserCanRegisterRepo`, `ErrRepoAccessDenied` |
| `services/hackforger/repos_test.go` | NEW — unit tests for above |
| `routers/web/hackforger/hackathon.go` | Replace 2× `SearchRepository` calls with `ListUserWritableRepos`; replace inline access check in `RegisterPost` with `UserCanRegisterRepo` |
| `routers/api/v1/hackforger/hackathon_registration.go` | Extend `RegisterForm` with 4 new optional fields; extend `Register` handler with repo-bound branch (mirrors web) |
| `docs/tests/e2e/reports/2026-05-03-org-repo-registration.md` | NEW — smoke-test report (frontmatter ready, body filled after manual verification) |

Estimated diff size: ~250 lines (helper + tests + 2 handler edits + report scaffold).

## 7. Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| Permission helper returns too few repos (overly strict) | Medium | Medium | Unit tests cover owner / org-owner / org-write / org-read / collaborator-write cases |
| Dropdown cap of 10 hides repos when user has many writable | Low-Medium | Low | The 10-item cap is for UX (long dropdown = unusable). Users with >10 writable repos can: (a) use API which takes any `repo_id` directly, or (b) accept that the dropdown shows their most recent — order is by `SearchRepository`'s natural recency. If this proves to be a real bottleneck we can add a search box to the dropdown later. |
| API caller passes `team_name` AND `repo_id` with a personal repo (no org) | Low | Low | `team_name` honored as-is for personal repos; only overridden when repo owner is an org. Documented. |
| Existing API callers break | Very low | High | All new fields are `omitempty`; default behavior unchanged |
| Permission check is async-stale (user just lost write access) | Very low | Low | `UserCanRegisterRepo` is called inside the request; window is microseconds |

## 8. Open Questions

| # | Topic | Default I assumed | Notes |
| --- | --- | --- | --- |
| 1 | Permission threshold | **Write+** (user confirmed 2026-05-03) | Read-only org members do not see org repos in dropdown |
| 2 | Pagination cap on dropdown | **10** (user choice 2026-05-03) | Keeps the dropdown short and scannable. Power users with >10 writable repos can call the API directly with any `repo_id`. |
| 3 | Should `team_name` be ignored when repo is an org? | **Yes — overridden by org name** for consistency with web behavior | Documented in handler comment; API caller's `team_name` is best-effort if repo is personal |

## 9. Rollback

- Revert the 1 commit; binary rebuild + redeploy. No DB schema change to roll back.
- Old code path (only personal repos) was the broken state — rolling back returns to the bug, not to a different broken state.

---

**Next step after approval:** invoke `superpowers:writing-plans` to produce the implementation plan, then execute in `.claude/worktrees/org-repo-fix` (worktree already created on branch `fix/hackathon-org-repo-registration`).
