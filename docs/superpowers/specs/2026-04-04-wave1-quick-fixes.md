# Wave 1: Quick Fixes

**Parent**: [Bug Fix & Phase System Design](2026-04-04-bug-fix-and-phase-system-design.md)
**Goal**: Fix silent failures, JS bugs, and small UI issues. All changes use existing Forgejo mechanisms.

## 1.1 JS Null Error Fix (HF-025)

**File**: `web_src/js/features/repo-issue-pr-status.js:6`
**Root cause**: `panel.querySelector('.commit-status-list')` returns null on HackForger pages where `.commit-status-list` doesn't exist. `list.style` then throws.
**Fix**: Add null guard `if (!list) return;` before `list.style` access.

## 1.2 Error Handling Audit (Cross-cutting)

Systematic audit of all HackForger routes and frontend code:

**Web POST routes** -- every error path must:
1. Call `ctx.Flash.Error(ctx.Tr("hackforger.error.xxx"), true)`
2. Render current page (not silent redirect)

**API routes** -- every error path must:
1. Call `ctx.Error(http.StatusXxx, "OperationName", err)`

**Vue/JS fetch calls** -- every `!resp.ok` or `.catch` must:
1. Call `showErrorToast(msg)` from `web_src/js/modules/toast.js`

**i18n**: All error messages defined in locale files under `hackforger.error.*` namespace.

**Deliverable**: i18n locale file additions + route/JS fixes. One PR per module (hackathon routes, bounty routes, grant routes, frontend JS).

## 1.3 Publish Validation (HF-002)

Before publishing any activity, validate prerequisites:
- **Hackathon**: At least 1 track exists → else `hackforger.error.publish_requires_track`
- **Bounty**: Must have linked Issue → else `hackforger.error.publish_requires_issue`
- **Grant Round**: Must have budget set → else `hackforger.error.publish_requires_budget`

Validation in service layer, error displayed via Flash.

## 1.4 Track Dropdown Truncation (HF-008, GH #19)

**Root cause**: Fomantic UI `.ui.selection.dropdown` default width or `text-overflow: ellipsis` truncates long track names.
**Fix**:
- CSS: Set `min-width` or `width: 100%` on dropdown container
- Add `title` attribute to each option for full text on hover
- Affects: track selection in submission form + judge assignment form

## 1.5 Scoring and Results (HF-009, HF-010)

**Type**: Spike tickets -- require investigation before fix can be defined.

**HF-009 (Scoring not working)**: Scoring submit button produces no response. Likely causes (to investigate in order):
1. Frontend: missing `showErrorToast()` on failed POST, making errors silent
2. Backend: scoring POST route not returning proper response/error
3. Backend: validation rejecting score but not surfacing error

**HF-010 (Results preview blank)**: Results page shows no content. Likely causes:
1. Scores not persisted (consequence of HF-009)
2. Results template not binding data from context
3. Results route/handler missing or returning empty query

**Approach**: Fix HF-009 first (add error handling to scoring flow). If scoring then works, verify HF-010 resolves. If not, investigate results query and template independently. Each fix produces its own PR.

## 1.6 Confirmation Dialogs (HF-004)

Use existing `web_src/js/features/comp/ConfirmModal.js`:
- Cancel Hackathon: `ctx.Tr("hackforger.confirm.cancel_activity")`
- Add `data-modal-confirm` attributes to cancel/delete buttons

## 1.7 Remove "Display Name" Field (HF-026)

- Remove the optional "显示名称" field from creation/edit forms
- Remove corresponding backend handling
- Keep existing display_name data in DB (no migration needed), just stop showing the field

## 1.8 Admin Panel Navigation (HF-028, GH #18)

- Add "活动管理" tab to site admin panel (`/admin`)
- Link to: Credits options, Hackathon management, Grant Round management
- Inject menu item into admin sidebar template

## 1.9 Credits Page Entry (HF-020)

- Add navigation link to credits page in user menu and/or dashboard sidebar
- Ensure `/user/credits` or equivalent route is accessible from UI

## 1.10 Phase Control i18n (HF-001)

- Audit Phase Control UI for untranslated strings
- Add all missing i18n keys to locale files
