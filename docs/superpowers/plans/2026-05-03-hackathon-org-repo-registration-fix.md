# Hackathon Org-Repo Registration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hackathon registration's repo dropdown shows org-owned repos the user can write to (currently shows nothing for org members), and the API gains a `repo_id` field with the same org-derivation behavior as the web flow.

**Architecture:** Extract a shared helper (`services/hackforger/repos.go`) with `ListUserWritableRepos` (used by web template) and `UserCanRegisterRepo` (used by web POST handler + new API repo branch). Web handler edits replace one `SearchRepository` call (2 sites) and one inline access check (1 site). API handler gains 4 optional form fields and a repo-bound branch that mirrors web behavior (organizer-self block + repo access check + submission creation + AddOrgUser).

**Tech Stack:** Go 1.25, Forgejo fork, XORM, existing `services/hackforger/` package conventions. PG18 (Mac dev shares with cloud after `sync-from-prod.sh`). Tests follow `*_test.go` + `package hackforger_test` pattern from `services/hackforger/hackathon_test.go`.

**Spec:** [`docs/superpowers/specs/2026-05-03-hackathon-org-repo-registration-fix.md`](../specs/2026-05-03-hackathon-org-repo-registration-fix.md) (commit `9793fa1ea2`)

---

## Worktree

All work is in **`.claude/worktrees/org-repo-fix`** on branch **`fix/hackathon-org-repo-registration`** (already created off `origin/v0.1-dev/hackforger`). All paths below are relative to that worktree root.

The worktree shares the Mac's PG (`hackforger` DB), so build → restart hits a live DB with real production data (synced via `sync-from-prod.sh`). No fresh-DB setup needed.

## Files Created / Modified

| Path | Owner | Purpose |
| ---- | ----- | ------- |
| `services/hackforger/repos.go` | Tasks 1-3 | NEW — helper with `ListUserWritableRepos`, `UserCanRegisterRepo`, `ErrRepoAccessDenied` |
| `services/hackforger/repos_test.go` | Task 1 | NEW — error-type test (the only piece testable without DB fixtures) |
| `routers/web/hackforger/hackathon.go` | Tasks 4-5 | Replace 2× `SearchRepository` (lines 285-291, 490-496) with helper; replace inline access check (lines 380-389) in `RegisterPost` |
| `routers/api/v1/hackforger/hackathon_registration.go` | Task 6 | Extend `RegisterForm` with 4 optional fields; rewrite `Register` handler to add organizer-self block + optional repo binding + submission/AddOrgUser parity |
| `docs/tests/e2e/reports/2026-05-03-org-repo-registration.md` | Tasks 8-10 | NEW — smoke-test report (frontmatter ready in Task 8, body filled after manual verification in Task 10) |

Estimated diff: ~280 lines (helper + test + 2 handler edits + report). No DB schema change.

---

## Task 1: Helper file scaffolding + error type

Lay down the file with the exported error type and `IsErr*` predicate. Test with a pure-Go assertion that doesn't need DB fixtures.

**Files:**
- Create: `services/hackforger/repos.go`
- Create: `services/hackforger/repos_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/hackforger/repos_test.go`:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
)

func TestErrRepoAccessDenied_Error(t *testing.T) {
	err := hackforger_service.ErrRepoAccessDenied{UserID: 42, RepoID: 7}
	assert.Equal(t, "user 42 lacks write permission to repo 7", err.Error())
}

func TestIsErrRepoAccessDenied(t *testing.T) {
	err := hackforger_service.ErrRepoAccessDenied{UserID: 1, RepoID: 2}
	assert.True(t, hackforger_service.IsErrRepoAccessDenied(err))
	assert.False(t, hackforger_service.IsErrRepoAccessDenied(nil))
	assert.False(t, hackforger_service.IsErrRepoAccessDenied(assert.AnError))
}
```

- [ ] **Step 2: Run test — should fail because file doesn't exist**

```bash
go test ./services/hackforger/... -run TestErrRepoAccessDenied -v
```

Expected: compile error — `undefined: hackforger_service.ErrRepoAccessDenied`.

- [ ] **Step 3: Create `services/hackforger/repos.go` with error type + IsErr predicate ONLY**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Package hackforger — helper for hackathon registration's "which repos can
// this user use" question. Extracted to share between web (template dropdown)
// and API (POST validation).
package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	perm_model "forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
)

// dropdownMax is the number of writable repos shown in the registration UI.
// Set conservatively for UX (long dropdown is unusable). Power users with
// more writable repos can call the API directly with any repo_id.
const dropdownMax = 10

// fetchPageSize is the SearchRepository page size BEFORE the Write+ filter.
// Larger than dropdownMax so the post-filter has enough candidates: a user
// might have read-only repos that take up early slots. 50 balances "enough
// candidates for 10 writable" against "not too many wasted permission lookups."
const fetchPageSize = 50

// ErrRepoAccessDenied is returned when a user tries to register with a repo
// they don't have write access to.
type ErrRepoAccessDenied struct {
	UserID int64
	RepoID int64
}

func (e ErrRepoAccessDenied) Error() string {
	return fmt.Sprintf("user %d lacks write permission to repo %d", e.UserID, e.RepoID)
}

// IsErrRepoAccessDenied checks if err is ErrRepoAccessDenied.
func IsErrRepoAccessDenied(err error) bool {
	_, ok := err.(ErrRepoAccessDenied)
	return ok
}

// Stub function bodies — implemented in Tasks 2 and 3.
func ListUserWritableRepos(ctx context.Context, user *user_model.User) ([]*repo_model.Repository, error) {
	_ = db.ListOptions{}
	_ = perm_model.AccessModeWrite
	_ = access_model.GetUserRepoPermission
	_ = repo_model.SearchRepository
	return nil, nil
}

func UserCanRegisterRepo(ctx context.Context, user *user_model.User, repoID int64) (*repo_model.Repository, error) {
	return nil, nil
}
```

The stubs reference all the imports so `go vet` and the compiler don't complain about unused imports while the file is incomplete.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./services/hackforger/... -run TestErrRepoAccessDenied -v
```

Expected: PASS for both `TestErrRepoAccessDenied_Error` and `TestIsErrRepoAccessDenied`.

- [ ] **Step 5: Commit**

```bash
cd .claude/worktrees/org-repo-fix
git add services/hackforger/repos.go services/hackforger/repos_test.go
git commit -m "feat(hackforger): scaffold repos helper with ErrRepoAccessDenied"
```

---

## Task 2: Implement `ListUserWritableRepos`

**Files:**
- Modify: `services/hackforger/repos.go` (replace stub on line ~50)

- [ ] **Step 1: Replace the `ListUserWritableRepos` stub**

Find the stub:
```go
func ListUserWritableRepos(ctx context.Context, user *user_model.User) ([]*repo_model.Repository, error) {
	_ = db.ListOptions{}
	_ = perm_model.AccessModeWrite
	_ = access_model.GetUserRepoPermission
	_ = repo_model.SearchRepository
	return nil, nil
}
```

Replace with the real implementation:

```go
// ListUserWritableRepos returns up to `dropdownMax` repositories the user has
// at least Write access to, including org-owned repos. Used to populate the
// hackathon registration repo dropdown.
//
// Note on choice of SearchRepoOptions:
//   - Actor=user + Private=true uses AccessibleRepositoryCondition, so
//     visibility includes user-owned + org-team-granted + collaborator repos.
//   - AllPublic/AllLimited=false intentionally excludes repos where the user
//     is "just a member of a public org with no team grant" — those won't
//     pass the Write+ post-filter anyway, so skipping them up front is faster.
//   - Each returned repo gets its Owner loaded so templates can render
//     "owner_name/repo_name" without N+1 lazy-load.
func ListUserWritableRepos(ctx context.Context, user *user_model.User) ([]*repo_model.Repository, error) {
	repos, _, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		Actor:              user,
		Private:            true,
		AllPublic:          false,
		AllLimited:         false,
		IncludeDescription: false,
		ListOptions:        db.ListOptions{Page: 1, PageSize: fetchPageSize},
	})
	if err != nil {
		return nil, err
	}

	out := make([]*repo_model.Repository, 0, dropdownMax)
	for _, r := range repos {
		if len(out) >= dropdownMax {
			break
		}
		perm, err := access_model.GetUserRepoPermission(ctx, r, user)
		if err != nil {
			continue
		}
		if perm.AccessMode < perm_model.AccessModeWrite {
			continue
		}
		if err := r.LoadOwner(ctx); err != nil {
			continue // skip repo whose owner can't be loaded
		}
		out = append(out, r)
	}
	return out, nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd .claude/worktrees/org-repo-fix
go build ./services/hackforger/...
echo "exit: $?"
```

Expected: exit 0, no output.

- [ ] **Step 3: Run all hackforger tests to confirm no regression**

```bash
go test ./services/hackforger/... -short
```

Expected: existing tests still PASS, new error-type tests still PASS.

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/repos.go
git commit -m "feat(hackforger): implement ListUserWritableRepos"
```

---

## Task 3: Implement `UserCanRegisterRepo`

**Files:**
- Modify: `services/hackforger/repos.go` (replace stub at the bottom)

- [ ] **Step 1: Replace the `UserCanRegisterRepo` stub**

Find:
```go
func UserCanRegisterRepo(ctx context.Context, user *user_model.User, repoID int64) (*repo_model.Repository, error) {
	return nil, nil
}
```

Replace with:

```go
// UserCanRegisterRepo loads a repo by ID and verifies the user has at least
// Write access to it. Returns the loaded repo (with Owner loaded) on success.
//
// Returns ErrRepoAccessDenied if the user lacks write permission.
// Returns repo_model.ErrRepoNotExist if the repo doesn't exist.
func UserCanRegisterRepo(ctx context.Context, user *user_model.User, repoID int64) (*repo_model.Repository, error) {
	repo, err := repo_model.GetRepositoryByID(ctx, repoID)
	if err != nil {
		return nil, err // includes repo_model.ErrRepoNotExist
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
```

- [ ] **Step 2: Verify build**

```bash
cd .claude/worktrees/org-repo-fix
go build ./services/hackforger/...
echo "exit: $?"
```

Expected: exit 0.

- [ ] **Step 3: Vet**

```bash
go vet ./services/hackforger/...
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/repos.go
git commit -m "feat(hackforger): implement UserCanRegisterRepo"
```

---

## Task 4: Web — replace 2× `SearchRepository` calls with helper

**Files:**
- Modify: `routers/web/hackforger/hackathon.go` (lines 285-291 and 490-496)

- [ ] **Step 1: Replace site #1 (lines 285-291) — view handler**

Find:
```go
		repos, _, _ := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
			Actor:       ctx.Doer,
			OwnerID:     ctx.Doer.ID,
			Private:     true,
			Collaborate: optional.Some(false),
		})
		ctx.Data["UserRepos"] = repos
```

Replace with:
```go
		// regression: must NOT re-add OwnerID/Collaborate filters here — see
		// docs/superpowers/specs/2026-05-03-hackathon-org-repo-registration-fix.md
		repos, err := hackforger_service.ListUserWritableRepos(ctx, ctx.Doer)
		if err != nil {
			log.Warn("ListUserWritableRepos: %v", err)
			repos = nil
		}
		ctx.Data["UserRepos"] = repos
```

- [ ] **Step 2: Replace site #2 (lines 490-496) — submit handler context**

Find:
```go
	repos, _, _ := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		Actor:       ctx.Doer,
		OwnerID:     ctx.Doer.ID,
		Private:     true,
		Collaborate: optional.Some(false),
	})
	ctx.Data["UserRepos"] = repos
```

Replace with:
```go
	// regression: must NOT re-add OwnerID/Collaborate filters here — see
	// docs/superpowers/specs/2026-05-03-hackathon-org-repo-registration-fix.md
	repos, err := hackforger_service.ListUserWritableRepos(ctx, ctx.Doer)
	if err != nil {
		log.Warn("ListUserWritableRepos: %v", err)
		repos = nil
	}
	ctx.Data["UserRepos"] = repos
```

Note: the variable `err` may already be declared earlier in the function — if compile fails on `err declared and not used` or `no new variables on left side of :=`, change `err` to a new name like `repoErr` or use `=` instead of `:=`.

- [ ] **Step 3: Verify the `optional` import is still used elsewhere; if not, drop it**

```bash
cd .claude/worktrees/org-repo-fix
grep -c "optional\." routers/web/hackforger/hackathon.go
```

If count is 0, remove the `"forgejo.org/modules/optional"` import. If non-zero, leave it.

- [ ] **Step 4: Build**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -10
```

Expected: build succeeds. If the second `:=` complains, adjust per Step 2 note.

- [ ] **Step 5: Commit**

```bash
git add routers/web/hackforger/hackathon.go
git commit -m "feat(hackathon): use ListUserWritableRepos in registration form

Replaces both SearchRepository sites that filtered by OwnerID = ctx.Doer.ID
+ Collaborate(false) — that combination silently excluded all org-owned
repos, which was the bug linyilun hit trying to register with H2OSLabs/
page.h2oslabs.com.

Comment includes a regression hint pointing back to the spec so a future
maintainer's diff doesn't quietly revert the filter."
```

---

## Task 5: Web — replace inline access check with `UserCanRegisterRepo`

**Files:**
- Modify: `routers/web/hackforger/hackathon.go` (lines 371-389 in `RegisterPost`)

- [ ] **Step 1: Replace the verify+access block**

Find this block (starts around line 371):
```go
	// Verify repo exists
	repo, err := repo_model.GetRepositoryByID(ctx, repoID)
	if err != nil {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.repo_required"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}

	// Check user has access
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
```

Replace with:
```go
	// Load repo + verify user has Write+ access (covers owner, org-team grant,
	// collaborator) — replaces the older inline `IsOrganizationMember` (read-
	// level) check, which was too loose: a read-only org member could
	// previously register but couldn't actually push to the repo. See
	// docs/superpowers/specs/2026-05-03-hackathon-org-repo-registration-fix.md
	// §4.4 + §7 for the intentional behavior tightening.
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
```

The org/team derivation block immediately below (`var orgID int64; ...`) keeps working unchanged because `repo.LoadOwner` was already called inside `UserCanRegisterRepo`, so `repo.OwnerID` is still valid and a follow-up `user_model.GetUserByID(ctx, repo.OwnerID)` still works.

- [ ] **Step 2: Build**

```bash
cd .claude/worktrees/org-repo-fix
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -10
```

Expected: build succeeds.

- [ ] **Step 3: Quick check the import for `organization_model.IsOrganizationMember` is still used elsewhere in the file**

```bash
grep -c "organization_model\." routers/web/hackforger/hackathon.go
```

If 0, remove the `"forgejo.org/models/organization"` import (likely still used for `AddOrgUser`/`GetOrgByID`, so probably keep). If non-zero, leave it.

- [ ] **Step 4: Commit**

```bash
git add routers/web/hackforger/hackathon.go
git commit -m "fix(hackathon): tighten repo access check on register POST

Replaces the inline GetRepositoryByID + IsOrganizationMember check with
hackforger_service.UserCanRegisterRepo, which requires Write+ instead of
just org membership. A read-only org member who could previously register
(but then couldn't actually push to the repo for the hackathon) now gets
the same repo_required error message — same place in the flow, more
honest about what's wrong.

Behavior change is intentional, see spec §4.4 + §7 risks."
```

---

## Task 6: API — extend `RegisterForm` + handler with repo branch

**Files:**
- Modify: `routers/api/v1/hackforger/hackathon_registration.go`

- [ ] **Step 1: Extend `RegisterForm`**

Find:
```go
// RegisterForm is the form for registering to a hackathon.
type RegisterForm struct {
	TeamName string `json:"team_name" binding:"Required"`
	TrackID  int64  `json:"track_id"`
}
```

Replace with:
```go
// RegisterForm is the form for registering to a hackathon.
type RegisterForm struct {
	TeamName    string `json:"team_name" binding:"Required"`
	TrackID     int64  `json:"track_id"`
	RepoID      int64  `json:"repo_id,omitempty"`      // NEW: optional; if 0, solo registration (no repo binding)
	Title       string `json:"title,omitempty"`         // NEW: submission title (defaults to repo name)
	Description string `json:"description,omitempty"`   // NEW: submission description
	DemoURL     string `json:"demo_url,omitempty"`      // NEW: submission demo URL
}
```

- [ ] **Step 2: Add organizer self-registration block to `Register` handler**

Find this block (after the `IsPublished` check, before `canRegister`):
```go
	if !h.IsPublished {
		ctx.Error(http.StatusBadRequest, "InvalidPhase", "hackathon is not accepting registrations")
		return
	}
	canRegister, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register")
```

Insert immediately after the `if !h.IsPublished` block, before `canRegister`:
```go
	// Block organizer self-registration (parity with web RegisterPost)
	if h.OwnerID == ctx.Doer.ID {
		ctx.Error(http.StatusForbidden, "CannotRegisterOwn", "organizer cannot register for own hackathon")
		return
	}
	if h.LinkedOrgID > 0 {
		if org, err := organization_model.GetOrgByID(ctx, h.LinkedOrgID); err == nil {
			if isOwner, _ := org.IsOwnedBy(ctx, ctx.Doer.ID); isOwner {
				ctx.Error(http.StatusForbidden, "CannotRegisterOwn", "organizer cannot register for own hackathon")
				return
			}
		}
	}

```

- [ ] **Step 3: Replace the registration build + create with repo-aware version**

Find:
```go
	f := web.GetForm(ctx).(*RegisterForm)
	r := &hackforger_model.HackathonRegistration{
		HackathonID: h.ID,
		UserID:      ctx.Doer.ID,
		TeamName:    f.TeamName,
		TrackID:     f.TrackID,
	}
	if err := hackforger_model.CreateRegistration(ctx, r); err != nil {
		if hackforger_model.IsErrDuplicateRegistration(err) {
			ctx.Error(http.StatusConflict, "AlreadyRegistered", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}
```

Replace with:
```go
	f := web.GetForm(ctx).(*RegisterForm)

	// Optional repo binding — hoisted so submission block below can use `repo`.
	var repo *repo_model.Repository
	if f.RepoID > 0 {
		var rerr error
		repo, rerr = hackforger_service.UserCanRegisterRepo(ctx, ctx.Doer, f.RepoID)
		if rerr != nil {
			if hackforger_service.IsErrRepoAccessDenied(rerr) {
				ctx.Error(http.StatusForbidden, "RepoAccessDenied", rerr)
				return
			}
			if repo_model.IsErrRepoNotExist(rerr) {
				ctx.Error(http.StatusNotFound, "RepoNotExist", rerr)
				return
			}
			ctx.InternalServerError(rerr)
			return
		}
	}

	r := &hackforger_model.HackathonRegistration{
		HackathonID: h.ID,
		UserID:      ctx.Doer.ID,
		TeamName:    f.TeamName,
		TrackID:     f.TrackID,
		Status:      hackforger_model.RegistrationStatusApproved, // ALWAYS — match web; was previously zero-value
	}
	if repo != nil {
		r.RepoID = repo.ID
		if repo.Owner.IsOrganization() {
			r.OrgID = repo.Owner.ID
			r.TeamName = repo.Owner.Name // override caller's TeamName for team/org registration
		}
	}
	if err := hackforger_model.CreateRegistration(ctx, r); err != nil {
		if hackforger_model.IsErrDuplicateRegistration(err) {
			ctx.Error(http.StatusConflict, "AlreadyRegistered", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}

	// Repo-bound side-effects (parity with web): submission + AddOrgUser.
	if repo != nil {
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
			RepoID:         repo.ID,
			Status:         hackforger_model.SubmissionStatusSubmitted,
		}
		if err := hackforger_service.CreateSubmission(ctx, ctx.Doer, h, s); err != nil {
			log.Error("CreateSubmission on register: %v", err)
			// Don't fail registration — log and continue (parity with web)
		}
	}

	// AddOrgUser side-effect runs whether or not a repo was bound (matches web).
	if h.LinkedOrgID > 0 {
		_ = organization_model.AddOrgUser(ctx, h.LinkedOrgID, ctx.Doer.ID)
	}
```

- [ ] **Step 4: Add the new imports**

At the top of the file, ensure these imports are present (some may already be there):

```go
import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	organization_model "forgejo.org/models/organization"   // NEW (if not already imported)
	repo_model "forgejo.org/models/repo"                    // NEW
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"                               // NEW (if not already imported)
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
	notify_service "forgejo.org/services/notify"
)
```

Run `go build` and let goimports / the compiler tell you which to add or remove.

- [ ] **Step 5: Build**

```bash
cd .claude/worktrees/org-repo-fix
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -10
```

Expected: build succeeds.

- [ ] **Step 6: Run hackforger tests for any regression**

```bash
go test ./services/hackforger/... ./routers/api/v1/hackforger/... -short 2>&1 | tail -20
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add routers/api/v1/hackforger/hackathon_registration.go
git commit -m "feat(api): hackathon registration accepts repo_id (org parity with web)

POST /api/v1/hackforger/hackathons/{id}/registrations now accepts
optional fields: repo_id, title, description, demo_url. When repo_id
is provided:
  - validate Write+ access via hackforger_service.UserCanRegisterRepo
  - if repo owner is an org, override caller's team_name with the org name
  - set OrgID = org.ID, RepoID = repo.ID
  - create the submission row with the same fields the web flow uses
  - run AddOrgUser if the hackathon has a LinkedOrg (parity with web)

Also adds the organizer-self-registration block that web has had since
day 1 but API was missing (HTTP 403 CannotRegisterOwn).

Status is now always set to Approved (was zero-value/Pending for the
legacy solo path — see spec §4.4 — minor behavior fix, not a breaking
change).

Backward-compatible: existing {team_name, track_id} callers still work."
```

---

## Task 7: Build + Mac dev smoke test (loopback API)

Verify the new binary actually runs and the new fields work end-to-end against real data.

**Files:** none modified

- [ ] **Step 1: Build the binary in the worktree**

```bash
cd .claude/worktrees/org-repo-fix
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -5
ls -lh gitea
```

Expected: a fresh `./gitea` binary in the worktree root.

- [ ] **Step 2: Restart Mac instance with the new binary**

```bash
# scripts/restart-gitea.sh expects to be run from main repo, but the binary
# is in the worktree. Easiest: cp the binary then run restart.
cp gitea /Users/h2oslabs/Workspace/hackforger/gitea
cd /Users/h2oslabs/Workspace/hackforger
bash scripts/restart-gitea.sh
```

Expected: `✓ HTTP 200 on localhost:3000`.

- [ ] **Step 3: Smoke API — solo registration still works (regression)**

```bash
ADMIN_PASS=$(grep '^HACKFORGER_ADMIN_PASSWORD=' .env | cut -d= -f2)
# Use a hackathon you can register for. Pick the first one not owned by SynNovator.
curl -s -u "SynNovator:$ADMIN_PASS" \
  "http://127.0.0.1:3000/api/v1/hackforger/hackathons?limit=20" \
  | jq -r '.[] | "\(.ID)  \(.Slug)  owner=\(.OwnerID)"' | head
```

Pick a hackathon `<HID>` you (as SynNovator, admin) own — actually since SynNovator IS the owner of the 12 prod hackathons, you'll need to test with a different user. Use `linyilun` / `key853211` instead.

Test legacy solo (should still work — no `repo_id`):
```bash
HID=<pick a non-owned hackathon ID, e.g. opc-2026-shuzhi-w1's ID>
curl -sX POST -u "linyilun:key853211" \
  -H "Content-Type: application/json" \
  -d '{"team_name":"linyilun-solo-test","track_id":0}' \
  "http://127.0.0.1:3000/api/v1/hackforger/hackathons/${HID}/registrations" \
  -w "\nHTTP %{http_code}\n"
```

Expected: `HTTP 201` with a JSON body. If you get `409 AlreadyRegistered`, that's also fine for this regression test — it means the validation succeeded.

- [ ] **Step 4: Smoke API — org repo binding (the new behavior)**

If linyilun has any prior test registration on this hackathon, delete it first:
```bash
PGPASSWORD=$(grep ^PGPASSWORD .env | cut -d= -f2) psql -h 127.0.0.1 -U hackforger -d hackforger \
  -c "DELETE FROM hackathon_registration WHERE user_id = (SELECT id FROM \"user\" WHERE name='linyilun') AND hackathon_id = ${HID};"
```

Then register with the org repo:
```bash
REPO_ID=$(curl -s -u "linyilun:key853211" \
  http://127.0.0.1:3000/api/v1/repos/H2OSLabs/page.h2oslabs.com | jq -r '.id')
echo "repo id: $REPO_ID"

curl -sX POST -u "linyilun:key853211" \
  -H "Content-Type: application/json" \
  -d "{\"team_name\":\"will-be-overridden\",\"track_id\":0,\"repo_id\":${REPO_ID},\"title\":\"linyilun's H2OSLabs entry\"}" \
  "http://127.0.0.1:3000/api/v1/hackforger/hackathons/${HID}/registrations" \
  -w "\nHTTP %{http_code}\n"
```

Expected: `HTTP 201` with a body that has `OrgID` set to H2OSLabs' user ID and `TeamName: "H2OSLabs"` (NOT "will-be-overridden").

- [ ] **Step 5: Verify DB rows**

```bash
PGPASSWORD=$(grep ^PGPASSWORD .env | cut -d= -f2) psql -h 127.0.0.1 -U hackforger -d hackforger -c "
SELECT r.id, r.team_name, r.org_id, r.repo_id, r.status,
       s.title AS submission_title
FROM hackathon_registration r
LEFT JOIN hackathon_submission s ON s.registration_id = r.id
WHERE r.user_id = (SELECT id FROM \"user\" WHERE name='linyilun')
  AND r.hackathon_id = ${HID};
"
```

Expected: one row with `team_name=H2OSLabs`, `org_id` matching H2OSLabs' user.id, `repo_id` set, `status=1` (Approved), and a `submission_title` of "linyilun's H2OSLabs entry".

- [ ] **Step 6: Smoke API — read-only repo gets 403**

Pick a public repo linyilun has only read access to. The simplest is one owned by an unrelated user — try one of the other hackathon track repos.

```bash
# Find a repo owned by someone else linyilun has no team grant on.
# (Use any submission's repo from the list at /hackathon/opc-2026-shuzhi-w1)
RO_REPO_ID=$(curl -s -u "linyilun:key853211" \
  http://127.0.0.1:3000/api/v1/repos/JerryC/OneSyn | jq -r '.id')
echo "read-only repo id: $RO_REPO_ID"

# Pick a different hackathon to avoid duplicate registration error
HID2=<another hackathon ID>

curl -sX POST -u "linyilun:key853211" \
  -H "Content-Type: application/json" \
  -d "{\"team_name\":\"x\",\"track_id\":0,\"repo_id\":${RO_REPO_ID}}" \
  "http://127.0.0.1:3000/api/v1/hackforger/hackathons/${HID2}/registrations" \
  -w "\nHTTP %{http_code}\n"
```

Expected: `HTTP 403` with `RepoAccessDenied` in the body.

- [ ] **Step 7: Smoke API — non-existent repo gets 404**

```bash
curl -sX POST -u "linyilun:key853211" \
  -H "Content-Type: application/json" \
  -d '{"team_name":"x","track_id":0,"repo_id":999999999}' \
  "http://127.0.0.1:3000/api/v1/hackforger/hackathons/${HID2}/registrations" \
  -w "\nHTTP %{http_code}\n"
```

Expected: `HTTP 404` with `RepoNotExist` in the body.

- [ ] **Step 8: Cleanup test rows**

```bash
PGPASSWORD=$(grep ^PGPASSWORD .env | cut -d= -f2) psql -h 127.0.0.1 -U hackforger -d hackforger -c "
DELETE FROM hackathon_submission
  WHERE user_id = (SELECT id FROM \"user\" WHERE name='linyilun')
    AND hackathon_id IN (${HID}, ${HID2});
DELETE FROM hackathon_registration
  WHERE user_id = (SELECT id FROM \"user\" WHERE name='linyilun')
    AND hackathon_id IN (${HID}, ${HID2});
"
```

(No commit — these were smoke tests, not code changes.)

---

## Task 8: Web E2E — agent-browser

Replicate the user's original repro: log in as linyilun, navigate to the hackathon page, verify H2OSLabs/page.h2oslabs.com appears in the dropdown, complete registration.

**Files:**
- Create: `docs/tests/e2e/reports/2026-05-03-org-repo-registration.md` (frontmatter + Stage 1 body)

- [ ] **Step 1: Open the hackathon page as anonymous, then log in**

```bash
agent-browser open http://localhost:3000/hackathon/opc-2026-shuzhi-w1
agent-browser snapshot -i 2>&1 | grep -E "登录/注册|Sign in" | head -3
agent-browser click @e6   # the "登录/注册" link — verify ref number from snapshot first
agent-browser snapshot -i 2>&1 | grep -E "用户名|Username" | head -5
```

Find the username and password input refs from the snapshot, then:
```bash
agent-browser fill @<username-ref> "linyilun"
agent-browser fill @<password-ref> "key853211"
agent-browser eval "document.querySelector('button.ui.primary.button').form.requestSubmit(document.querySelector('button.ui.primary.button'))"
sleep 3
agent-browser screenshot /tmp/e2e-step-1-logged-in.png
```

- [ ] **Step 2: Verify dropdown lists H2OSLabs/page.h2oslabs.com**

```bash
agent-browser snapshot -i 2>&1 | grep -B 1 -A 20 "选择仓库\|repo_id" | head -40
agent-browser eval "
  var sel = document.querySelector('select[name=\"repo_id\"]');
  JSON.stringify(Array.from(sel.options).map(o => ({value: o.value, label: o.textContent.trim()})));
"
```

Expected: `H2OSLabs/page.h2oslabs.com` appears in the option list with a non-empty `value`. Capture screenshot:
```bash
agent-browser screenshot /tmp/e2e-step-2-dropdown-has-org-repo.png
```

- [ ] **Step 3: Complete registration with the org repo**

```bash
# Select the H2OSLabs repo
agent-browser eval "
  var sel = document.querySelector('select[name=\"repo_id\"]');
  var opt = Array.from(sel.options).find(o => o.textContent.includes('H2OSLabs/page.h2oslabs.com'));
  sel.value = opt.value;
  sel.dispatchEvent(new Event('change'));
  opt.value;
"
agent-browser snapshot -i 2>&1 | grep -E "项目名称|track_id|track" | head -5
agent-browser fill @<title-ref> "page.h2oslabs.com hackathon entry"
# Click the 报名 button via requestSubmit (avoid the click-doesn't-submit quirk):
agent-browser eval "
  var btn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.trim() === '报名');
  btn.form.requestSubmit(btn);
"
sleep 3
agent-browser screenshot /tmp/e2e-step-3-registered.png
agent-browser snapshot -i 2>&1 | grep -E "已报名|registered|success" | head -3
```

Expected: success flash message visible OR the page state changes to show the user as registered.

- [ ] **Step 4: Verify the DB row**

```bash
PGPASSWORD=$(grep ^PGPASSWORD .env | cut -d= -f2) psql -h 127.0.0.1 -U hackforger -d hackforger -c "
SELECT r.id, r.team_name, r.org_id, r.repo_id, r.status,
       u.name AS user_name, o.name AS org_name, repo.name AS repo_name
FROM hackathon_registration r
JOIN \"user\" u ON u.id = r.user_id
LEFT JOIN \"user\" o ON o.id = r.org_id
LEFT JOIN repository repo ON repo.id = r.repo_id
WHERE u.name = 'linyilun'
  AND r.hackathon_id = (SELECT id FROM hackathon WHERE slug='opc-2026-shuzhi-w1');
"
```

Expected: row with `team_name=H2OSLabs`, `org_name=H2OSLabs`, `repo_name=page.h2oslabs.com`, `status=1`.

- [ ] **Step 5: Close browser**

```bash
agent-browser close
```

- [ ] **Step 6: Write the smoke-test report scaffold (Stage 1 — Claude E2E)**

Create `docs/tests/e2e/reports/2026-05-03-org-repo-registration.md` (in the worktree):

```markdown
---
pr: <fill-in-after-PR-opened>
commits:
  - <fill-in-with-actual-SHAs-from-this-branch>
tested_against: https://hackforger.inside.h2os.cloud
tested_at: <fill-in-ISO-timestamp>
e2e_owner: claude
admin_signoff: null
---

# Hackathon Registration: org-owned repos now appear in dropdown + API parity

## Stage 1 — Claude E2E (web, agent-browser)

Repro of the original bug: linyilun (member of H2OSLabs) tried to register
opc-2026-shuzhi-w1 with H2OSLabs/page.h2oslabs.com but the dropdown was empty.

### Verified

- [x] Logged in as `linyilun` at `http://localhost:3000/hackathon/opc-2026-shuzhi-w1`
- [x] Repo dropdown now lists `H2OSLabs/page.h2oslabs.com`
- [x] Selected the repo, filled title, submitted form
- [x] DB shows: `team_name=H2OSLabs`, `org_id=<H2OSLabs.id>`, `repo_id=<page.h2oslabs.com.id>`, `status=1` (Approved)
- [x] Submission row exists with same `repo_id`

Screenshots:
- `e2e-step-1-logged-in.png` — landing post-login
- `e2e-step-2-dropdown-has-org-repo.png` — H2OSLabs/page.h2oslabs.com visible in dropdown
- `e2e-step-3-registered.png` — registration success state

### API path verified separately (Task 9 commands):
- [x] Legacy solo `{team_name, track_id}` → 201
- [x] With `repo_id` of org repo (writable) → 201, OrgID set, TeamName overridden, submission created
- [x] With `repo_id` of read-only repo → 403 RepoAccessDenied
- [x] With non-existent `repo_id` → 404 RepoNotExist

## Stage 2 — Admin manual sign-off (pending)

Admin will walk through the same flow at https://hackforger.inside.h2os.cloud
and edit `admin_signoff` below.
```

- [ ] **Step 7: Commit the report scaffold**

```bash
cd .claude/worktrees/org-repo-fix
git add docs/tests/e2e/reports/2026-05-03-org-repo-registration.md
git commit -m "test(report): hackathon org-repo registration e2e (Stage 1)

Claude-side e2e verification: linyilun can now select
H2OSLabs/page.h2oslabs.com from the registration dropdown and
the resulting DB row has the expected OrgID/TeamName/RepoID.

API path also verified (Task 9 commands).

admin_signoff: pending — to be filled by admin after manual
walkthrough at https://hackforger.inside.h2os.cloud."
```

---

## Task 9: API E2E — curl-based verification (already covered in Task 7)

Task 7 steps 3-7 cover the four API cases (solo, org-repo, read-only-repo→403, nonexistent-repo→404). No additional task needed; results referenced from the report file in Task 8.

If you want to formalize this as a script for future runs, optionally create `docs/tests/e2e/scripts/2026-05-03-org-repo-registration-api.sh` with the curl commands — but this is YAGNI for the first run.

---

## Task 10: Open PR + admin sign-off + cloud deploy

**Files:** none modified beyond the existing branch

- [ ] **Step 1: Push the branch**

```bash
cd .claude/worktrees/org-repo-fix
git log --oneline origin/v0.1-dev/hackforger..HEAD
```

Expected: spec commits + 6 implementation commits (Tasks 1, 2, 3, 4, 5, 6) + report commit (Task 8). Roughly 9 commits.

```bash
git push -u origin fix/hackathon-org-repo-registration 2>&1 | tail -3
```

- [ ] **Step 2: Open PR**

```bash
gh pr create --title "fix(hackathon): registration accepts org repos (web + API parity)" --body "$(cat <<'EOF'
linyilun (member of H2OSLabs) couldn't register \`opc-2026-shuzhi-w1\` with the org-owned repo \`H2OSLabs/page.h2oslabs.com\` — and in fact the entire repo dropdown was empty for any user, because the SearchRepository call was filtering to \`OwnerID = ctx.Doer.ID + Collaborate(false)\`. Same root cause meant the API couldn't bind a registration to any repo at all (RegisterForm had no \`repo_id\`).

Spec: \`docs/superpowers/specs/2026-05-03-hackathon-org-repo-registration-fix.md\`
Plan: \`docs/superpowers/plans/2026-05-03-hackathon-org-repo-registration-fix.md\`

## What changes
- New \`services/hackforger/repos.go\` — single source of truth for "which repos can this user use to register" (\`ListUserWritableRepos\` + \`UserCanRegisterRepo\`)
- Web handler: 2× SearchRepository sites use the helper; \`RegisterPost\` access check uses the helper (tightens from Read to Write+ — see spec §4.4 + §7)
- API handler: \`RegisterForm\` gains \`repo_id, title, description, demo_url\` (all omitempty); handler mirrors web's organizer-self-block, repo binding, submission creation, AddOrgUser

## Backward compatibility
- API \`{team_name, track_id}\` legacy callers: still work; status now Approved instead of zero-value (latent fix, not break)
- Web flow with \`repo_id\` form param: works the same, slightly tighter access requirement

## Test plan
- [x] Helper unit test: \`go test ./services/hackforger/... -run TestErrRepoAccessDenied\`
- [x] Build clean: \`TAGS="bindata sqlite sqlite_unlock_notify" make backend\`
- [x] Mac dev API smoke: solo→201, org-repo→201 with OrgID set, read-only→403, nonexistent→404
- [x] Mac dev Web E2E (agent-browser): linyilun sees H2OSLabs/page.h2oslabs.com in dropdown and can register
- [ ] Admin manual sign-off at https://hackforger.inside.h2os.cloud (Stage 2 — fills in \`admin_signoff\` in the report)

## Companion smoke-test report
\`docs/tests/e2e/reports/2026-05-03-org-repo-registration.md\` — Stage 1 done by Claude in this PR; Stage 2 admin signoff is what unblocks cloud deploy.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)" 2>&1 | tail -3
```

- [ ] **Step 3: Hand back to admin for Stage 2 sign-off**

PR is now open. STOP here — do not admin-merge yet. The admin needs to:

1. Pull the branch locally on Mac
2. Restart Mac instance with the branch's binary (`bash scripts/restart-gitea.sh` after copying binary)
3. Manually walk through the registration flow at https://hackforger.inside.h2os.cloud
4. If satisfied, edit `docs/tests/e2e/reports/2026-05-03-org-repo-registration.md` to fill in `admin_signoff` (with handle, ISO timestamp, and any notes), commit, push the report update to the branch
5. Then it's safe to admin-merge

Wait for the admin's signal that Stage 2 is complete before continuing to Step 4.

- [ ] **Step 4: After admin sign-off — admin-squash-merge**

```bash
PR=$(gh pr list --head fix/hackathon-org-repo-registration --json number --jq '.[0].number')
gh pr merge "$PR" --squash --admin --delete-branch 2>&1 | tail -3
```

- [ ] **Step 5: Sync local default + run preflight**

```bash
cd /Users/h2oslabs/Workspace/hackforger
git checkout v0.1-dev/hackforger
git pull --ff-only
git branch -f prod origin/v0.1-dev/hackforger    # advance prod branch (admin signed off)
bash deploy/ecs/preflight.sh
```

Expected: preflight reports the new commit as ✓ pass (the report file admin signed covers it).

- [ ] **Step 6: Deploy to cloud**

```bash
bash deploy/ecs/redeploy.sh
```

Expected: build → custom rsync → binary swap → restart → public smoke tests show the new version + navbar locale check still passes.

- [ ] **Step 7: Verify on prod**

```bash
ADMIN_PASS=$(grep '^HACKFORGER_ADMIN_PASSWORD=' .env | cut -d= -f2)
curl -fsS https://www.synnovator.com/api/v1/version
echo
# As linyilun on prod: same flow but against the cloud
# (skipped here — admin verifies during/after redeploy.sh)
```

- [ ] **Step 8: Clean up worktree**

```bash
git worktree remove .claude/worktrees/org-repo-fix
git worktree list
```

Expected: the org-repo-fix worktree is gone; other worktrees (dev backup, enable-remember) are untouched.

---

## Self-Review Notes

**Spec coverage:**
- §3 root cause → Tasks 4 + 5 (web) + Task 6 (API) all cite the same spec sections in their commit messages
- §4.1 helper code → Tasks 1 + 2 + 3
- §4.2 web edits → Tasks 4 + 5
- §4.3 API edits → Task 6
- §4.4 backward compatibility (Status: Approved on solo) → Task 6 step 3
- §5.1 unit tests → Task 1 (the only piece testable without DB fixtures); rest covered by Tasks 7-9 integration tests
- §5.1a regression tests → Task 4 + 5 commit messages include the regression-hint comments; Task 7 covers the API integration cases
- §5.2 web E2E → Task 8
- §5.3 API E2E → Task 7 (steps 3-7) + referenced from report in Task 8
- §5.4 manual gate → Task 10 step 3 explicitly halts for admin

**Placeholder scan:** No "TBD/TODO/fill in details" patterns. The report frontmatter has `<fill-in-after-PR-opened>` and `<fill-in-ISO-timestamp>` markers — these are intentional, the operator fills them at run time. No code-level placeholders.

**Type/name consistency:** All references use `hackforger_service.ListUserWritableRepos`, `hackforger_service.UserCanRegisterRepo`, `hackforger_service.IsErrRepoAccessDenied`. Helper file's package is `hackforger`, imported by callers as `hackforger_service` — matches existing pattern in `services/hackforger/`. `repo_model.IsErrRepoNotExist` is the upstream Forgejo predicate (not invented).

**Risks not yet mitigated by tasks:** None — every spec §7 risk has a corresponding task action or is documented in commit messages.
