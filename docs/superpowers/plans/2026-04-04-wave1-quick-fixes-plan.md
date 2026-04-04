# Wave 1: Quick Fixes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix silent failures, JS bugs, and small UI issues using existing Forgejo mechanisms.

**Architecture:** All changes use existing Forgejo error handling (Flash, Toast, API errors). No new tables or models. Each task is 1 PR.

**Tech Stack:** Go templates, JavaScript, CSS, i18n locale files

**Spec:** `docs/superpowers/specs/2026-04-04-wave1-quick-fixes.md`

---

## Concurrency Groups

```
Group A (all independent, run in parallel):
  Task 1: JS null error fix (HF-025)
  Task 2: Track dropdown CSS fix (HF-008)
  Task 3: Confirmation dialogs (HF-004)
  Task 4: Remove display name field (HF-026)
  Task 5: Phase Control i18n (HF-001)

Group B (all independent, run in parallel after Group A or concurrently):
  Task 6: Admin panel navigation (HF-028)
  Task 7: Credits page entry (HF-020)
  Task 8: Publish validation (HF-002)

Group C (sequential — spike investigation):
  Task 9: Scoring fix (HF-009) → then Task 10: Results preview (HF-010)

Cross-cutting (can run with any group):
  Task 11: Error handling audit — hackathon routes
  Task 12: Error handling audit — bounty routes
  Task 13: Error handling audit — grant routes
  Task 14: Error handling audit — frontend JS
```

---

### Task 1: JS Null Error Fix (HF-025)

**Files:**
- Modify: `web_src/js/features/repo-issue-pr-status.js`

- [ ] **Step 1: Add null guard**

```javascript
export function initRepoPullRequestCommitStatus() {
  for (const btn of document.querySelectorAll('.commit-status-hide-checks')) {
    const panel = btn.closest('.commit-status-panel');
    const list = panel.querySelector('.commit-status-list');
    if (!list) continue;
    btn.addEventListener('click', () => {
      list.style.maxHeight = list.style.maxHeight ? '' : '0px'; // toggle
      btn.textContent = btn.getAttribute(list.style.maxHeight ? 'data-show-all' : 'data-hide-all');
    });
  }
}
```

- [ ] **Step 2: Build frontend and verify**

Run: `make frontend`
Expected: Build succeeds, no errors

- [ ] **Step 3: Commit and create PR**

```bash
git checkout -b fix/hf-025-js-null-error
git add web_src/js/features/repo-issue-pr-status.js
git commit -m "fix(hackforger): HF-025 add null guard in repo-issue-pr-status.js"
gh pr create --title "fix(hackforger): HF-025 JS null error in commit status" --body "Add null guard for .commit-status-list element that may not exist on HackForger pages."
```

---

### Task 2: Track Dropdown CSS Fix (HF-008)

**Files:**
- Modify: Track selection templates (locate in `templates/hackforger/` or relevant submission/judge templates)
- Modify: `web_src/css/hackforger.css` (create if needed)

- [ ] **Step 1: Locate track dropdown templates**

Search for track/赛道 dropdown in HackForger templates:
```bash
grep -r "track" templates/hackforger/ --include="*.tmpl" -l
grep -r "赛道" templates/ --include="*.tmpl" -l
```

- [ ] **Step 2: Add `title` attribute to dropdown options**

In the track dropdown template, add `title="{{.Name}}"` to each `<div class="item">` so full name shows on hover.

- [ ] **Step 3: Fix CSS width**

Add to the HackForger CSS file:

```css
/* HF-008: Track dropdown should not truncate */
.hackforger-track-dropdown.ui.selection.dropdown {
  min-width: 100%;
}
.hackforger-track-dropdown.ui.selection.dropdown .menu > .item {
  white-space: normal;
  word-break: break-word;
}
```

- [ ] **Step 4: Build frontend and verify**

Run: `make frontend`
Expected: Build succeeds

- [ ] **Step 5: Commit and create PR**

```bash
git checkout -b fix/hf-008-track-dropdown
git add -A templates/hackforger/ web_src/css/
git commit -m "fix(hackforger): HF-008 fix track dropdown truncation"
gh pr create --title "fix(hackforger): HF-008 track dropdown truncation" --body "Fix track dropdown width and add title attribute for full text on hover. Resolves HackForger/hackforger#19."
```

---

### Task 3: Confirmation Dialogs (HF-004)

**Files:**
- Modify: Hackathon management template (cancel button)
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 1: Add i18n keys**

In `options/locale/locale_en-US.ini`, add under a `[hackforger]` section:
```ini
hackforger.confirm.cancel_activity = Are you sure you want to cancel this activity? This action cannot be undone.
```

In `options/locale/locale_zh-CN.ini`:
```ini
hackforger.confirm.cancel_activity = 确认取消活动？此操作不可撤销。
```

- [ ] **Step 2: Add data-modal-confirm to cancel button**

Locate the cancel button in the hackathon management template. Add the confirm modal trigger:

```html
<button class="ui red button"
        data-modal-confirm="{{ctx.Locale.Tr "hackforger.confirm.cancel_activity"}}"
        data-modal-confirm-button-color="red">
  {{ctx.Locale.Tr "hackforger.cancel"}}
</button>
```

Note: Forgejo's `initGlobalButtons()` in `common-global.js` automatically handles `data-modal-confirm` attributes using `ConfirmModal.js`.

- [ ] **Step 3: Build and verify**

Run: `make frontend && make backend`
Expected: Build succeeds

- [ ] **Step 4: Commit and create PR**

```bash
git checkout -b fix/hf-004-cancel-confirm
git add options/locale/ templates/hackforger/
git commit -m "fix(hackforger): HF-004 add confirmation dialog for cancel activity"
gh pr create --title "fix(hackforger): HF-004 cancel confirmation dialog" --body "Add confirmation dialog before canceling activities to prevent accidental cancellation."
```

---

### Task 4: Remove Display Name Field (HF-026)

**Files:**
- Modify: Hackathon/Bounty/Grant creation and edit templates
- Modify: Corresponding web route handlers (remove field processing)

- [ ] **Step 1: Locate display name field in templates**

```bash
grep -r "显示名称" templates/ --include="*.tmpl" -l
grep -r "display.name\|display_name\|DisplayName" templates/hackforger/ --include="*.tmpl" -l
```

- [ ] **Step 2: Remove the field from templates**

Delete the form group containing the display name input from all creation/edit templates.

- [ ] **Step 3: Remove backend handling**

In the corresponding web route handlers, remove the line that reads `ctx.Req.FormValue("display_name")` and any assignment to the model's DisplayName field. Do NOT remove the DB column — existing data stays.

- [ ] **Step 4: Build and verify**

Run: `make backend && make frontend`
Expected: Build succeeds

- [ ] **Step 5: Commit and create PR**

```bash
git checkout -b fix/hf-026-remove-display-name
git add templates/ routers/
git commit -m "fix(hackforger): HF-026 remove optional display name field"
gh pr create --title "fix(hackforger): HF-026 remove display name field" --body "Remove the confusing optional display name field from creation/edit forms. DB column retained for backward compatibility."
```

---

### Task 5: Phase Control i18n (HF-001)

**Files:**
- Modify: Phase control templates
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 1: Audit Phase Control UI for untranslated strings**

```bash
grep -r "Phase\|phase\|阶段" templates/hackforger/ --include="*.tmpl" | grep -v "ctx.Locale.Tr\|\.Locale.Tr"
```

This finds hardcoded strings that should use i18n.

- [ ] **Step 2: Add i18n keys for all Phase Control strings**

In `options/locale/locale_en-US.ini`:
```ini
hackforger.phase.control = Phase Control
hackforger.phase.current = Current Phase
hackforger.phase.start_time = Start Time
hackforger.phase.end_time = End Time
hackforger.phase.status.active = Active
hackforger.phase.status.locked = Locked
hackforger.phase.status.upcoming = Upcoming
```

In `options/locale/locale_zh-CN.ini`:
```ini
hackforger.phase.control = 阶段控制
hackforger.phase.current = 当前阶段
hackforger.phase.start_time = 开始时间
hackforger.phase.end_time = 结束时间
hackforger.phase.status.active = 进行中
hackforger.phase.status.locked = 已锁定
hackforger.phase.status.upcoming = 即将开始
```

- [ ] **Step 3: Replace hardcoded strings in templates**

Replace all hardcoded Phase Control strings with `{{ctx.Locale.Tr "hackforger.phase.xxx"}}` calls.

- [ ] **Step 4: Build and verify**

Run: `make frontend && make backend`

- [ ] **Step 5: Commit and create PR**

```bash
git checkout -b fix/hf-001-phase-i18n
git add options/locale/ templates/hackforger/
git commit -m "fix(hackforger): HF-001 add i18n for Phase Control UI"
gh pr create --title "fix(hackforger): HF-001 Phase Control i18n" --body "Add i18n keys for all Phase Control UI strings. Supports en-US and zh-CN."
```

---

### Task 6: Admin Panel Navigation (HF-028)

**Files:**
- Modify: `templates/admin/navbar.tmpl` (upstream injection point)
- Create: `templates/admin/hackforger/` directory + placeholder pages

- [ ] **Step 1: Add HackForger section to admin navbar**

In `templates/admin/navbar.tmpl`, add after the last existing nav item:

```html
{{/* HackForger injection point: admin navigation */}}
<a class="{{if .PageIsAdminHackforger}}active {{end}}item" href="{{AppSubUrl}}/admin/hackforger">
  {{ctx.Locale.Tr "hackforger.admin.title"}}
</a>
```

- [ ] **Step 2: Add i18n keys**

```ini
; locale_en-US.ini
hackforger.admin.title = Activity Management
hackforger.admin.credits = Credits Options
hackforger.admin.hackathons = Hackathons
hackforger.admin.grants = Grant Rounds

; locale_zh-CN.ini
hackforger.admin.title = 活动管理
hackforger.admin.credits = 积分选项
hackforger.admin.hackathons = 黑客松
hackforger.admin.grants = 赞助轮次
```

- [ ] **Step 3: Create admin hackforger route (if not exists)**

In `routers/web/hackforger/admin.go`, add a handler that renders the admin hackforger page with links to sub-sections (credits, hackathons, grants).

- [ ] **Step 4: Register route in web.go**

Add the admin route registration at the appropriate injection point in `routers/web/web.go`.

- [ ] **Step 5: Build and verify**

Run: `make backend && make frontend`

- [ ] **Step 6: Commit and create PR**

```bash
git checkout -b feat/hf-028-admin-navigation
git add templates/admin/ routers/ options/locale/
git commit -m "feat(hackforger): HF-028 add admin panel navigation for HackForger"
gh pr create --title "feat(hackforger): HF-028 admin panel HackForger navigation" --body "Add 'Activity Management' tab to admin panel with links to Credits, Hackathons, and Grant Rounds management. Resolves HackForger/hackforger#18."
```

---

### Task 7: Credits Page Entry (HF-020)

**Files:**
- Modify: User menu template or dashboard sidebar
- Modify: Route registration (if credits route exists but has no UI link)

- [ ] **Step 1: Locate credits route**

```bash
grep -r "credits\|Credits" routers/ --include="*.go" -l
grep -r "credits" templates/ --include="*.tmpl" -l
```

- [ ] **Step 2: Add navigation link**

Add a link to the credits page in the user dropdown menu or dashboard sidebar. The exact template depends on where the credits page lives (user-level or site-level):

```html
<a class="item" href="{{AppSubUrl}}/user/credits">
  {{ctx.Locale.Tr "hackforger.credits.my_credits"}}
</a>
```

- [ ] **Step 3: Add i18n keys**

```ini
; locale_en-US.ini
hackforger.credits.my_credits = My Credits

; locale_zh-CN.ini
hackforger.credits.my_credits = 我的积分
```

- [ ] **Step 4: Build and verify**

Run: `make backend && make frontend`

- [ ] **Step 5: Commit and create PR**

```bash
git checkout -b fix/hf-020-credits-nav
git add templates/ options/locale/
git commit -m "fix(hackforger): HF-020 add credits page navigation entry"
gh pr create --title "fix(hackforger): HF-020 credits page navigation" --body "Add navigation link to credits page from user menu. Resolves HackForger/hackforger#18."
```

---

### Task 8: Publish Validation (HF-002)

**Files:**
- Modify: `services/hackforger/hackathon.go` (or equivalent publish handler)
- Modify: `services/hackforger/bounty.go`
- Modify: `services/hackforger/grant.go`
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1: Add i18n keys**

```ini
; locale_en-US.ini
hackforger.error.publish_requires_track = Cannot publish: please add at least one track first.
hackforger.error.publish_requires_issue = Cannot publish: please link an issue first.
hackforger.error.publish_requires_budget = Cannot publish: please set a budget first.

; locale_zh-CN.ini
hackforger.error.publish_requires_track = 无法发布：请先添加至少一个赛道。
hackforger.error.publish_requires_issue = 无法发布：请先关联一个 Issue。
hackforger.error.publish_requires_budget = 无法发布：请先设置预算。
```

- [ ] **Step 2: Add validation in hackathon publish**

In the hackathon publish service function, before changing status to published:

```go
tracks, err := models_hackforger.GetTracksByHackathonID(ctx, hackathon.ID)
if err != nil {
    return err
}
if len(tracks) == 0 {
    return ErrPublishRequiresTrack
}
```

Define the error type and map it to the i18n key in the web route handler:
```go
if errors.Is(err, hackforger_service.ErrPublishRequiresTrack) {
    ctx.Flash.Error(ctx.Tr("hackforger.error.publish_requires_track"), true)
    // render current page
    return
}
```

- [ ] **Step 3: Add validation in bounty and grant publish**

Same pattern: check linked Issue for bounty, check budget > 0 for grant.

- [ ] **Step 4: Write tests**

```go
func TestPublishHackathonWithoutTracks(t *testing.T) {
    // Create hackathon with no tracks
    // Attempt publish
    // Assert ErrPublishRequiresTrack returned
}
```

Run: `go test ./services/hackforger/... -run TestPublish -v`

- [ ] **Step 5: Commit and create PR**

```bash
git checkout -b fix/hf-002-publish-validation
git add services/hackforger/ routers/ options/locale/
git commit -m "fix(hackforger): HF-002 validate prerequisites before publishing"
gh pr create --title "fix(hackforger): HF-002 publish validation" --body "Block publishing activities without required prerequisites (tracks, linked issue, budget). Show clear error message."
```

---

### Task 9: Scoring Fix — Spike (HF-009)

**Files:**
- Investigate: Scoring route handler, frontend scoring JS, scoring template

- [ ] **Step 1: Locate scoring code**

```bash
grep -r "score\|Score\|评分" routers/web/hackforger/ services/hackforger/ --include="*.go" -l
grep -r "score\|Score" web_src/js/features/hackforger/ --include="*.js" --include="*.vue" -l
grep -r "score\|评分" templates/hackforger/ --include="*.tmpl" -l
```

- [ ] **Step 2: Trace the scoring flow**

1. Find the scoring form/button in template
2. Find the JS that handles the form submit
3. Find the web route handler for the POST
4. Find the service function that persists the score
5. Check each layer for missing error handling

- [ ] **Step 3: Add error handling at each layer**

For the JS fetch call, ensure:
```javascript
import {showErrorToast} from '../../modules/toast.js';

const response = await POST(url, {data});
if (!response.ok) {
    const data = await response.json();
    showErrorToast(data.errorMessage || ctx.locale.tr('hackforger.error.score_failed'));
    return;
}
```

For the Go route handler, ensure error paths call:
```go
ctx.Flash.Error(ctx.Tr("hackforger.error.score_failed"), true)
```

- [ ] **Step 4: Test scoring manually**

Start server: `./gitea web`
Navigate to a hackathon with submissions, attempt scoring, verify:
- Success: score persists, success feedback shown
- Failure: error toast displayed

- [ ] **Step 5: Commit and create PR**

```bash
git checkout -b fix/hf-009-scoring
git add services/ routers/ web_src/ templates/ options/locale/
git commit -m "fix(hackforger): HF-009 fix scoring submission not responding"
gh pr create --title "fix(hackforger): HF-009 fix scoring submission" --body "Add proper error handling to scoring flow. Root cause: [describe after investigation]."
```

---

### Task 10: Results Preview Fix (HF-010)

**Depends on:** Task 9 (HF-009) — scoring must work before results can display.

**Files:**
- Investigate: Results route handler, results template

- [ ] **Step 1: Verify scoring works after Task 9**

With the scoring fix applied, create test scores and check if results preview now shows data.

- [ ] **Step 2: If still blank, investigate results query**

```bash
grep -r "result\|Result\|预览\|preview" routers/web/hackforger/ services/hackforger/ --include="*.go" -l
grep -r "result\|preview" templates/hackforger/ --include="*.tmpl" -l
```

Check:
- Does the route handler query scores and pass them to the template?
- Does the template bind the data correctly?
- Is the results endpoint registered in the router?

- [ ] **Step 3: Fix the root cause**

Apply fix based on investigation. Common issues:
- Template not receiving data → fix route handler to pass `ctx.Data["Results"]`
- Query returning empty → fix query joins/filters
- Route not registered → add route in `web.go`

- [ ] **Step 4: Add error handling**

Ensure results page shows a meaningful message when no scores exist:
```html
{{if .Results}}
  {{/* render results table */}}
{{else}}
  <div class="ui info message">{{ctx.Locale.Tr "hackforger.results.no_scores_yet"}}</div>
{{end}}
```

- [ ] **Step 5: Commit and create PR**

```bash
git checkout -b fix/hf-010-results-preview
git add routers/ services/ templates/ options/locale/
git commit -m "fix(hackforger): HF-010 fix results preview not showing"
gh pr create --title "fix(hackforger): HF-010 fix results preview" --body "Fix results preview page to correctly display scoring results."
```

---

### Tasks 11-14: Error Handling Audit (Cross-cutting)

These four tasks follow the same pattern. Each audits one module's routes/handlers for silent failures and adds proper error handling. They can all run in parallel.

**Pattern for each task:**

- [ ] **Step 1: List all route handlers for the module**

```bash
grep -rn "func\|ctx\.Redirect\|ctx\.JSON\|ctx\.HTML" routers/web/hackforger/<module>.go
```

- [ ] **Step 2: Find all error paths without user feedback**

Look for patterns like:
```go
if err != nil {
    log.Error(...)  // logs but no user feedback
    return          // silent failure
}
```

- [ ] **Step 3: Add Flash/Toast/API error at each error path**

Web routes:
```go
if err != nil {
    ctx.Flash.Error(ctx.Tr("hackforger.error.<specific_error>"), true)
    ctx.<RenderCurrentPage>()
    return
}
```

API routes:
```go
if err != nil {
    ctx.Error(http.StatusInternalServerError, "OperationName", err)
    return
}
```

- [ ] **Step 4: Add all new i18n keys to locale files**

- [ ] **Step 5: Build and verify**

Run: `make backend`

- [ ] **Step 6: Commit and create PR**

**Task 11** — Hackathon routes:
```bash
git checkout -b fix/hf-error-audit-hackathon
git commit -m "fix(hackforger): error handling audit for hackathon routes"
```

**Task 12** — Bounty routes:
```bash
git checkout -b fix/hf-error-audit-bounty
git commit -m "fix(hackforger): error handling audit for bounty routes"
```

**Task 13** — Grant routes:
```bash
git checkout -b fix/hf-error-audit-grant
git commit -m "fix(hackforger): error handling audit for grant routes"
```

**Task 14** — Frontend JS:
```bash
git checkout -b fix/hf-error-audit-frontend-js
git commit -m "fix(hackforger): error handling audit for frontend JS"
```

Each PR body: "Systematic error handling audit for <module>. Add Flash/Toast errors for all silent failure paths. All messages use i18n."

---

## E2E Verification After Wave 1

After all Wave 1 PRs are merged, run E2E testing via agent-browser:

- [ ] Navigate to hackathon pages — verify no JS console errors
- [ ] Test track dropdown with long names — verify no truncation
- [ ] Attempt to publish hackathon without tracks — verify error message appears
- [ ] Cancel a hackathon — verify confirmation dialog appears
- [ ] Submit an invalid form — verify error toast/flash appears
- [ ] Check admin panel — verify "Activity Management" tab exists
- [ ] Check user menu — verify credits link exists
- [ ] Test scoring — verify scores persist and results preview works

Take screenshots at each checkpoint. Save report to `docs/tests/e2e/wave1-results.md`.
