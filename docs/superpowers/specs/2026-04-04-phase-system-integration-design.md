# Phase System Integration Design

## Goal

Replace manual hackathon phase control buttons with time-driven automatic phase transitions powered by the existing Phase System (PhaseType/Phase/PhaseController). Organizers set phase start/end times via a Phase Timeline editor; the system determines the current phase by checking `start_time <= now < end_time`.

## Background

The Phase System core (models, service, migration with 11 seed types) was built in Wave 3 but never wired into the hackathon workflow. Currently, hackathon status transitions (Draft -> Open -> Hacking -> Judging -> Finished) are triggered by manual buttons on the manage page. This design replaces those buttons with time-based automation.

## Architecture

### Data Model Changes

**Hackathon table:**
- Rename `status` column to `status_cache` (migration). This field becomes a computed cache derived from the Phase table. Source of truth is the Phase table. Comment: `// Computed from Phase system. Do NOT set directly. Use SyncStatusCache().`
- Add `is_published` column (`bool`, default `false`). Controls whether the phase timeline is active. StatusCache remains Draft until `is_published = true` AND a phase's start_time has arrived.
- Remove `registration_start`, `registration_end`, `hacking_start`, `hacking_end`, `judging_end` columns — but **NOT in the same migration** as the Phase data migration. Old columns are dropped in a separate follow-up migration after the integration is validated.
- No public setter for `StatusCache`. Updates only via internal `SyncStatusCache()`.

**Phase table:** No schema changes. Existing schema already supports:
- `phase_type_id` (FK to PhaseType catalog)
- `activity_kind` + `activity_id` (polymorphic reference to hackathon/bounty/grant)
- `start_time`, `end_time` (unix timestamps)
- `sort_order` (sequence within an activity)
- Methods: `IsLocked()`, `IsActive()`, `IsFuture()`

### StatusCache Sync Logic

`SyncStatusCache()` is called in two contexts:
1. **After every Phase CRUD operation** (add, update, delete)
2. **Lazily on hackathon read** — when loading a hackathon for display/API, check if StatusCache matches current phase state; if not, update and fire lifecycle hooks

When StatusCache changes (old != new), `SyncStatusCache()` publishes a `hackathon_phase_changed` feed event, replacing the current `publishPhaseChange()` calls in the manual button handlers.

**StatusCache derivation rules:**

| Phase State | StatusCache |
|-------------|-------------|
| `is_published = false` (regardless of phases) | Draft (0) |
| Published, all phases are future (not yet started) | Draft (0) |
| Published, active phase type = `registration` | Open (1) |
| Published, active phase type = `development` | Hacking (2) |
| Published, active phase type = `judging` | Judging (3) |
| Published, active phase type = `results` | Finished (4) |
| Published, between phases (gap, no active phase) | **Next upcoming phase's mapped status** |
| Published, all phases locked, last = `results` | Finished (4) |
| Manually cancelled | Cancelled (5) |

The mapping from PhaseType.Key to HackathonStatus:

```
"registration" -> HackathonStatusOpen (1)
"development"  -> HackathonStatusHacking (2)
"judging"      -> HackathonStatusJudging (3)
"results"      -> HackathonStatusFinished (4)
```

**Gap handling:** Between phases, StatusCache maps to the **next upcoming (future) phase's** status. This avoids confusing users — e.g., during a registration→development gap, StatusCache shows Hacking (2) ("awaiting hacking") rather than Open (1) ("registration still open"). If no future phases remain, use the last locked phase's status.

### Phase Transition Lifecycle Hooks

When `SyncStatusCache()` detects a status change (old StatusCache != computed new status), it triggers lifecycle hooks that replace the side effects currently embedded in manual button handlers:

| Transition | Hook | Replaces |
|-----------|------|----------|
| Draft → Open | Validate: ≥1 track exists | `PublishHackathon` precondition |
| Open → Hacking | Tag track repos with `v0-kickoff` | `StartHacking` side effect |
| Hacking → Judging | Validate: ≥1 criteria, ≥1 submission. If validation fails, log warning but allow transition (organizer should have set this up before judging starts). | `StartJudging` preconditions |
| Judging → Finished | No auto-action. Finalize requires manual confirmation. | N/A (manual `ConfirmFinalize` kept) |
| Any → Cancelled | Delete all future phases | `CancelHackathon` cleanup |

**Publish-time pre-validation** catches most issues early:
- At least 1 registration phase, 1 development phase
- At least 1 track exists
- All phase times valid (no overlaps, endTime > startTime)
- Phases in logical order (registration before development before judging)
- At least 1 scoring criterion exists (warn if not, don't block)

This means the Hacking→Judging hook's validation failure is a rare edge case (organizer deleted criteria after publishing), handled gracefully with a warning rather than blocking the time-based transition.

### UpdatePhaseTime Active Phase Guard

The existing `UpdatePhaseTime()` service function must add an explicit guard for active phases:

```
if phase.IsActive() && startTime != phase.StartTime:
    return ErrActivePhaseStartLocked{PhaseID: phase.ID}
```

Active phases can only have their `EndTime` modified (extend or shorten). `StartTime` changes are rejected because the phase has already begun. This is enforced server-side, not just in the Vue UI.

New error type: `ErrActivePhaseStartLocked`.

### Phase Timeline Vue Component

A new `PhaseTimeline.vue` component on the manage page:

**Props** (via data attributes on mount element):
- `hackathonSlug` — for API calls
- `phases` — current Phase records (JSON)
- `phaseTypes` — PhaseType catalog for hackathon kind (JSON)
- `isPublished` — whether hackathon has been published

**Features:**
- List of phases ordered by `sortOrder`, each showing: type name (i18n), start datetime, end datetime, status badge (locked/active/future)
- "Add Phase" button with PhaseType dropdown selector
- Datetime-local inputs for start/end times
- Client-side validation: `endTime > startTime`, no time overlaps between phases
- **Locked phases** (endTime < now): read-only, grayed out, no edit/delete
- **Active phase** (startTime <= now < endTime): endTime editable, startTime read-only, not deletable
- **Future phases**: fully editable, deletable
- Each change triggers a POST/PUT/DELETE to the backend; response includes updated phases list + synced StatusCache for UI refresh

**API endpoints** (web routes under manage group, session-cookie auth, return JSON):
- `GET /hackathon/{slug}/manage/phases` — list all phases + phase type catalog
- `POST /hackathon/{slug}/manage/phases` — add phase (phaseTypeID, startTime, endTime)
- `PUT /hackathon/{slug}/manage/phases/{id}` — update phase times
- `DELETE /hackathon/{slug}/manage/phases/{id}` — delete future phase

Note: These are web routes (session-cookie authenticated), not `/api/v1/` routes. Consistent with the existing `JudgeScoreCard.vue` pattern per CLAUDE.md convention.

All mutation endpoints call `SyncStatusCache()` after the operation.

### Publish Flow Changes

**Before (manual):** Click "Publish" button -> status = Open.

**After (time-driven):** Click "Publish" button -> validates phases + tracks -> sets `is_published = true` -> StatusCache updates lazily when the first phase's start_time arrives.

Publish validation:
- At least 1 registration phase
- At least 1 development phase
- At least 1 track exists
- All phase times valid (no overlaps, endTime > startTime)
- Phases in logical order (registration before development before judging)

The "Publish" button semantics change from "push to Open status" to "confirm configuration, enable the timeline". The button is disabled once published.

### Action Gating Integration

Replace hardcoded `if h.Status != HackathonStatusXxx` checks with `AllowsAction()`:

| Operation | Action String | Location | Current Check | New Check |
|-----------|--------------|----------|---------------|-----------|
| Register for hackathon | `"register"` | web router, API | `h.Status == Open` | `AllowsAction("hackathon", id, "register")` |
| Form team | `"form_team"` | web router | `h.Status == Open` | `AllowsAction("hackathon", id, "form_team")` |
| Submit work | `"submit_work"` | web router, API | `h.Status == Hacking` | `AllowsAction("hackathon", id, "submit_work")` |
| Submit form display | `"submit_work"` | web router | `h.Status == Hacking` | `AllowsAction("hackathon", id, "submit_work")` |
| Judge scoring | `"score"` | service, web router | `h.Status == Judging` | `AllowsAction("hackathon", id, "score")` |
| Judge page access | `"score"` | web router | `h.Status == Judging` | `AllowsAction("hackathon", id, "score")` |
| Finalize results | `"publish_results"` | web router | `h.Status == Judging` | `AllowsAction("hackathon", id, "publish_results")` |

**Operations that keep using StatusCache directly** (not phase-type actions):
| Operation | Current Check | Kept As-Is |
|-----------|---------------|------------|
| Delete hackathon | `h.StatusCache == Draft` | Yes — structural guard, not a phase action |
| Leaderboard score breakdown | `h.StatusCache == Finished` | Yes — display logic, not gated by phase |
| API registration | `h.StatusCache == Open` | Migrate to `AllowsAction` for consistency |
| API submission | `h.StatusCache == Hacking` | Migrate to `AllowsAction` for consistency |

`AllowsAction()` already exists in the Phase service — it checks if the currently active phase's PhaseType allows the given action. Returns `false` when no active phase exists (between phases, no actions allowed).

### Manage Page Button Changes

**Remove:**
- "Start Hacking" button (replaced by phase auto-transition)
- "Start Judging" button (replaced by phase auto-transition)

**Keep (modified):**
- "Publish" button — now means "confirm and enable timeline". Sets `is_published = true`. Disabled once published.
- "Cancel" button — sets StatusCache to Cancelled, deletes all future phases.
- "Finalize Confirm" button — kept in results phase. Finalize (ranking + credit distribution) is an irreversible human decision, not auto-triggered.

**New:**
- Phase Timeline editor (Vue component, replaces manual phase control buttons)

### View Page Changes

- Replace hardcoded 3-phase timeline with dynamic timeline rendered from Phase records
- Each phase shows: type name (i18n via PhaseType.DisplayNameI18n), time range, status (done/active/upcoming)
- Active phase highlighted with `hf-step-active` CSS class
- Registration CTA shown when `AllowsAction("register")` returns true (instead of `StatusCache == Open`)
- Status label derived from PhaseType.DisplayNameI18n of the current active phase (replaces `HackathonStatusLabel()` hardcoded strings)

### Migration Strategy

Two-phase migration for safety:

**Migration 1 (this project):**
1. Rename `hackathon.status` -> `hackathon.status_cache`
2. Add `hackathon.is_published` column (bool, default false)
3. For each existing hackathon with status > 0 (not Draft):
   - Set `is_published = true`
   - Generate Phase records from existing time fields:
     - `registration`: start=`registration_start`, end=`registration_end`
     - `development`: start=`hacking_start`, end=`hacking_end`
     - `judging`: start=`hacking_end` (if `judging_start` is missing), end=`judging_end`
     - `results`: start=`judging_end`, end=`judging_end + 30 days` (reasonable default)
   - For missing time fields (zero values), use reasonable defaults based on `created_unix` and status
4. Old time columns (`registration_start`, `registration_end`, `hacking_start`, `hacking_end`, `judging_end`) are **kept** in this migration (not dropped)

**Migration 2 (future, after validation):**
- Drop old time columns after confirming Phase integration works correctly in production

### Error Handling

All Phase mutations return typed errors:
- `ErrPhaseOverlap` — time window conflicts with existing phase
- `ErrPhaseLocked` — attempt to modify a completed phase
- `ErrPhaseNotFuture` — attempt to delete non-future phase
- `ErrPhaseUniqueViolation` — duplicate unique phase type
- `ErrPhaseActionNotAllowed` — action not permitted in current phase
- `ErrActivePhaseStartLocked` (NEW) — attempt to change startTime of an active phase

Vue component displays these as inline error messages via JSON response. Web route handlers translate typed errors to i18n flash messages or JSON error responses (depending on whether the caller is a form submit or Vue fetch).

### Code References to Update

After renaming `Status` to `StatusCache`, all references must be updated:
- `models/hackforger/hackathon.go`: struct field, `HackathonStatusNames`, `ListHackathonsOptions.ToConds()`, `UpdateHackathonStatus()`
- `services/hackforger/hackathon.go`: all status checks in `PublishHackathon`, `StartHacking`, `StartJudging`, `ConfirmFinalize`, `CancelHackathon`, `HackathonStatusLabel`
- `routers/web/hackforger/hackathon.go`: all `h.Status` references
- `routers/api/v1/hackforger/`: all status checks
- `templates/hackforger/hackathon/`: `.Hackathon.Status` -> `.Hackathon.StatusCache`

`HackathonStatusLabel()` should be updated to return the active PhaseType's `DisplayNameI18n` key when a phase is active, falling back to the StatusCache-based label.

## Scope Boundaries

**In scope:**
- StatusCache rename + sync logic + lifecycle hooks
- `IsPublished` field on Hackathon
- `ErrActivePhaseStartLocked` guard in `UpdatePhaseTime`
- Phase Timeline Vue component (`PhaseTimeline.vue`)
- Phase CRUD web route endpoints (JSON)
- Action gating integration (replace hardcoded status checks)
- Manage page: remove manual phase buttons, add Phase Timeline
- View page: dynamic phase timeline from Phase records
- Publish flow changes (`is_published` flag)
- Data migration for existing hackathons (two-phase strategy)
- Feed event publishing on status transitions (via `SyncStatusCache`)

**Out of scope (future work):**
- PhaseScheduler for proactive notifications ("phase X starts in 1 hour")
- Bounty/Grant phase integration (same architecture, separate project)
- Admin PhaseType management page (seed data sufficient for now)
- Drag-to-reorder phases in UI (use sort_order input for now)
- Dropping old time columns (Migration 2, after validation)
