# Fix Error Notification Patterns

**Date:** 2026-04-13
**Issue:** #52 (赛道相关) + systematic error handling audit
**Scope:** Option B — fix all 15 broken web handlers

## Problem

HackForger's web handlers have three types of broken error notification patterns:

1. **Hardcoded English strings** — `ctx.Flash.Error("Issue is required")` instead of `ctx.Flash.Error(ctx.Tr("hackforger.bounty.error.issue_required"))`
2. **`fmt.Sprintf(err)` leaking internal errors** — `ctx.Flash.Error(fmt.Sprintf("Failed to deposit: %v", err))` exposing raw Go errors to users
3. **JSON responses in web handlers** — `ctx.JSON(http.StatusUnprocessableEntity, ...)` where Flash messages + redirect are expected

The correct Forgejo pattern is:
```
Service layer: typed errors (ErrXxx) with IsErrXxx() checkers
Web handler: check error type → ctx.Flash.Error(ctx.Tr("i18n.key")) → redirect
```

## Affected Handlers (15 total)

### bounty.go (4 issues)
- `NewBountyPost` lines 138, 146, 165, 184 — hardcoded English strings
- `BountyAction` — JSON responses + no typed error checks (but called by Vue, so JSON is correct; needs typed error checks + i18n messages in JSON)
- `BountyApplicationAction` — same as BountyAction
- `BountySelectWinners` — generic "internal error" JSON

### credits.go (9 issues)
- `AdminCreditsDeposit` line 200 — `fmt.Sprintf`
- `AdminCreditsDeduct` line 224 — `fmt.Sprintf`
- `AdminRedeemOptionsCreate` line 275 — `fmt.Sprintf`
- `AdminRedeemOptionsUpdate` lines 287, 302 — `fmt.Sprintf`
- `AdminCreditOrdersFulfill` line 339 — `fmt.Sprintf`
- `AdminCreditOrdersCancel` line 351 — `fmt.Sprintf`
- `AdminRedeemOptionKeysAdd` line 412 — `fmt.Sprintf`
- `AdminCreditOrdersBatchFulfill` line 448 — `fmt.Sprintf`

### grants.go (8 issues)
- `ManageGrantRoundOpen` line 496 — `fmt.Sprintf`
- `ManageGrantRoundClose` line 512 — `fmt.Sprintf`
- `ManageGrantRoundFinalize` line 528 — `fmt.Sprintf`
- `ManageGrantRoundDistribute` line 544 — `fmt.Sprintf`
- `ManageGrantRoundCancel` line 560 — `fmt.Sprintf`
- `ManageGrantProjectApprove` line 613 — `fmt.Sprintf`
- `ManageGrantProjectReject` line 626 — `fmt.Sprintf`
- `ManageGrantProjectAward` line 642 — `fmt.Sprintf`
- `ManageGrantProjectDistribute` line 655 — `fmt.Sprintf`

### hackathon.go (1 issue)
- `ManagePhasePost` line 615 — hardcoded `"Unknown action"`

## Design Decisions

### 1. Vue-called JSON handlers (BountyAction, BountyApplicationAction, BountySelectWinners, JudgeScoresPost, phase.go)

These handlers are intentionally JSON because they serve Vue components. The fix is to add typed error checking with i18n messages in JSON responses (which JudgeScoresPost already does correctly). Keep `ctx.JSON()` but replace generic "internal error" with typed checks.

### 2. Form handlers (credits.go admin, grants.go manage)

Convert `fmt.Sprintf("Failed to X: %v", err)` to:
- Check typed errors (e.g., `IsErrInvalidTransition`, `IsErrAccessDenied`, `IsErrNotAdmin`, `IsErrOrderNotPending`)
- Use `ctx.Flash.Error(ctx.Tr("hackforger.xxx.error.yyy"))` for known errors
- Use `ctx.ServerError("FuncName", err)` for unknown/system errors (instead of leaking via fmt.Sprintf)

### 3. New i18n keys needed

Add to both `locale_en-US.ini` and `locale_zh-CN.ini`:

```ini
; Bounty errors
bounty.error.issue_required = Issue is required.
bounty.error.issue_not_found = Issue not found.
bounty.error.already_exists = A bounty already exists for this issue.
bounty.error.created = Bounty created successfully.
bounty.error.invalid_status = Invalid bounty status transition.
bounty.error.not_publisher = Only the bounty publisher can perform this action.
bounty.error.has_applications = Cannot modify a bounty with pending applications.
bounty.error.unknown_action = Unknown action.
bounty.error.internal = An error occurred. Please try again.

; Credits admin errors
credits.admin.error.deposit_failed = Failed to deposit credits.
credits.admin.error.deduct_failed = Failed to deduct credits.
credits.admin.error.not_admin = Only administrators can perform this action.
credits.admin.error.option_not_found = Redeem option not found.
credits.admin.error.option_create_failed = Failed to create redeem option.
credits.admin.error.option_update_failed = Failed to update redeem option.
credits.admin.error.order_fulfill_failed = Failed to fulfill order.
credits.admin.error.order_cancel_failed = Failed to cancel order.
credits.admin.error.order_not_pending = This order is not in pending status.
credits.admin.error.keys_add_failed = Failed to add keys.
credits.admin.error.batch_failed = Batch operation failed.

; Grant errors
grant.error.invalid_transition = This grant cannot transition to the requested status.
grant.error.access_denied = You do not have permission to manage this grant.
grant.error.not_finalized = Grant must be finalized before distribution.
grant.error.exceeds_budget = Award allocation exceeds the remaining budget.
grant.error.unallocated = Some approved projects have not been allocated awards.
grant.error.operation_failed = Operation failed. Please try again.
grant.error.approve_failed = Failed to approve project.
grant.error.reject_failed = Failed to reject project.
grant.error.award_failed = Failed to allocate award.
grant.error.distribute_failed = Failed to distribute award.
```

## Implementation Plan

1. Add i18n keys to both locale files
2. Fix `bounty.go` — replace hardcoded strings with `ctx.Tr()`, add typed error checks to JSON handlers
3. Fix `credits.go` — replace `fmt.Sprintf` with typed error checks + i18n
4. Fix `grants.go` — replace `fmt.Sprintf` with typed error checks + i18n
5. Fix `hackathon.go` — replace hardcoded "Unknown action"
6. Build and verify compilation
7. E2E test key pages via agent-browser
