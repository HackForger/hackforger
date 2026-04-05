# Remaining Fixes & Missing Modules Design

**Date:** 2026-04-05
**Status:** Approved
**Scope:** Phase System logic gaps, UI bug fixes, PhaseNotifier, Phase API, Admin PhaseType page, Bounty/Grant integration

---

## Group A: Phase System Logic Gaps

### A1-A2: API Router AllowsAction

**Problem:** Web routers use `AllowsAction()` but API routers still hardcode `h.StatusCache != HackathonStatusOpen/Hacking`. API callers bypass Phase time window checks.

**Fix:**
- `routers/api/v1/hackforger/hackathon_registration.go:67` — replace `h.StatusCache != HackathonStatusOpen` with `AllowsAction(ctx, "hackathon", h.ID, "register")`
- `routers/api/v1/hackforger/hackathon_submission.go:69` — replace `h.StatusCache != HackathonStatusHacking` with `AllowsAction(ctx, "hackathon", h.ID, "submit_work")`

### A3: Judge Service AllowsAction

**Problem:** `services/hackforger/hackathon_judge.go:36` hardcodes `h.StatusCache != HackathonStatusJudging`.

**Fix:** Replace with `AllowsAction(ctx, "hackathon", hackathonID, "score")`. The service function needs the hackathonID to call AllowsAction, which it already has from the function parameter.

---

## Group B: UI Bug Fixes

### B1-B3: Fomantic Dropdown Selection Class

**Problem:** Multiple `<select>` elements use `class="ui dropdown"` but are missing the `selection` modifier needed for proper Fomantic styling.

**Files:**
- `templates/hackforger/hackathon/manage.tmpl` — track select, judge track select
- `templates/hackforger/hackathon/view.tmpl` — org select for registration
- `templates/hackforger/hackathon/submit.tmpl` — repo select, track select
- `templates/hackforger/grants/submit.tmpl` — repo select

**Fix:** Add `selection` class: `class="ui dropdown selection"` or `class="ui dropdown selection search"` where search is needed.

### B4: PhaseTimeline.vue Fixed Width Overflow

**Problem:** `tw-w-32` (128px) on phase names truncates long names and doesn't adapt to screen size.

**Fix:** Replace `tw-w-32` with `tw-min-w-24 tw-max-w-48` and add `tw-truncate` for overflow handling. The rename input should use `tw-w-full` within its container.

### B5: Credits Admin Fixed Width

**Problem:** `tw-w-24` on inputs in `credits/admin/orders.tmpl` causes overflow on small screens.

**Fix:** Replace `tw-w-24` with `tw-min-w-16 tw-flex-1` to allow flexible sizing.

### B6: BountyPanel.vue Inline Style

**Problem:** `style="width: 100px"` instead of Tailwind class.

**Fix:** Replace with `class="tw-w-24"`.

---

## Group C: Missing Modules

### C1: PhaseNotifier (gocron-based)

**Architecture:**
- New file: `services/hackforger/phase_notifier.go`
- Uses existing `gocron.Scheduler` from `services/cron/cron.go`
- Phase CRUD operations call `SchedulePhaseJobs(phase)` / `UnschedulePhaseJobs(phaseID)`
- Server startup calls `RebuildPhaseSchedule()` to scan DB and register all future phase jobs

**Job registration:**
```
For each phase where end_time > now:
  if start_time > now:
    scheduler.Every(1).LimitRunsTo(1).StartAt(startTime).Tag("phase-{id}-start").Do(onPhaseStart, phaseID)
  scheduler.Every(1).LimitRunsTo(1).StartAt(endTime).Tag("phase-{id}-end").Do(onPhaseEnd, phaseID)
```

**Event handlers:**
- `onPhaseStart(phaseID)` — calls `SyncStatusCache` for the phase's activity, publishes feed event
- `onPhaseEnd(phaseID)` — same

**Lifecycle:**
- Phase created → `SchedulePhaseJobs(phase)`
- Phase time updated → `UnschedulePhaseJobs(id)` then `SchedulePhaseJobs(phase)`
- Phase deleted → `UnschedulePhaseJobs(id)`
- Server start → `RebuildPhaseSchedule()` in `initHackforgerTasks()`

**Integration with existing cron:**
- The `hackforger_hackathon_status` cron task body is filled as a **fallback sweep** (catches any missed jobs due to race conditions). Runs every 5 min, scans published hackathons where StatusCache might be stale.
- PhaseNotifier is the **primary** mechanism (minute-level precision).

### C2: Phase API Routes

**New file:** `routers/api/v1/hackforger/phase.go`

**Endpoints:**
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/hackforger/hackathons/{id}/phases` | `ListPhases` | List all phases for a hackathon |
| POST | `/hackforger/hackathons/{id}/phases` | `AddPhase` | Add a phase (requires token) |
| PUT | `/hackforger/hackathons/{id}/phases/{pid}` | `UpdatePhase` | Update phase time/name |
| DELETE | `/hackforger/hackathons/{id}/phases/{pid}` | `DeletePhase` | Delete a future phase |
| GET | `/hackforger/hackathons/{id}/current-phase` | `GetCurrentPhase` | Get the currently active phase |
| POST | `/hackforger/hackathons/{id}/phases/reorder` | `ReorderPhases` | Batch update sort_order |

All handlers call existing service/model functions (no new business logic).

### C3: Admin Phase Type Management

**New files:**
- `templates/hackforger/admin/phase_types.tmpl`
- `routers/web/hackforger/admin_phase_types.go`

**Routes (under `/admin/hackforger/phase-types`):**
| Method | Path | Handler |
|--------|------|---------|
| GET | `/` | `AdminPhaseTypes` — list grouped by activity_kind |
| POST | `/` | `AdminPhaseTypeCreate` |
| POST | `/{id}/edit` | `AdminPhaseTypeUpdate` |
| POST | `/{id}/delete` | `AdminPhaseTypeDelete` |

**Template:** Table grouped by activity_kind (hackathon/bounty/grant), each row shows key, display name, is_unique, allowed_actions (as checkboxes). Create form at bottom.

### C4: Bounty/Grant Phase Integration

**Bounty:** Replace hardcoded status checks in `services/hackforger/bounty.go` with `AllowsAction` calls:
- `ApplyForBounty` → `AllowsAction(ctx, "bounty", bountyID, "apply")`
- `ClaimBounty` → `AllowsAction(ctx, "bounty", bountyID, "claim")`
- `SubmitDelivery` → `AllowsAction(ctx, "bounty", bountyID, "submit_pr")`
- `ReviewDelivery` → `AllowsAction(ctx, "bounty", bountyID, "review")`

**Grant:** Replace in `services/hackforger/grant.go`:
- `SubmitProject` → `AllowsAction(ctx, "grant", roundID, "apply")`
- `ApproveProject` → `AllowsAction(ctx, "grant", roundID, "approve")`
- `DisburseProject` → `AllowsAction(ctx, "grant", roundID, "disburse")`

**Note:** Bounty/Grant currently don't have Phase records created automatically. The integration adds AllowsAction calls but they will pass-through (return true) when no phases are configured, preserving backward compatibility. Phase records for bounties/grants are created when organizers configure them via the Phase Timeline UI.

**Fill cron bodies:**
- `hackforger_bounty_expiry` → call `ExpireOldBounties(ctx)` (already exists)
- `hackforger_grant_deadline` → call `CheckGrantDeadlines(ctx)` (needs implementation or stub)

---

## Execution Order

```
Parallel Group 1 (no dependencies):
  A1-A3: AllowsAction fixes (3 files, 10 min)
  B1-B6: UI fixes (6 files, 15 min)

Sequential Group 2 (after Group 1):
  C1: PhaseNotifier (new file + cron integration, 30 min)
  C2: Phase API routes (new file + route registration, 30 min)

Parallel Group 3 (after C1):
  C3: Admin PhaseType page (template + router, 30 min)
  C4: Bounty/Grant integration (service modifications, 20 min)

Final: Build + restart + smoke test
```
