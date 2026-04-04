# Wave 3: Phase System

**Parent**: [Bug Fix & Phase System Design](2026-04-04-bug-fix-and-phase-system-design.md)
**Goal**: Introduce event lifecycle management as HackForger's core architectural addition to Forgejo.

## Core Concept

Every activity (Hackathon, Bounty, Grant) has a lifecycle composed of ordered phases. Each phase has a time window and controls what actions are permitted. This is the fundamental concept that distinguishes HackForger from Forgejo -- the introduction of **time-bounded events**.

## 3.1 Data Model

### Table: `hackforger_phase_type` (Admin-defined phase catalog)

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT PK AUTO | |
| activity_kind | VARCHAR(20) NOT NULL | `hackathon` / `bounty` / `grant` |
| key | VARCHAR(50) NOT NULL | Unique within activity_kind. e.g. `registration` |
| display_name_i18n | VARCHAR(100) NOT NULL | i18n key, e.g. `hackforger.phase.registration` |
| is_unique | BOOL NOT NULL DEFAULT true | If true, an activity can only have one phase of this type |
| allowed_actions | TEXT NOT NULL | JSON array of action strings |
| default_order | INT NOT NULL DEFAULT 0 | Suggested ordering when creating new activities |
| created_unix | BIGINT NOT NULL | |

**Unique constraint**: `UNIQUE(activity_kind, key)`

### Table: `hackforger_phase` (Activity instance phases)

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT PK AUTO | |
| phase_type_id | BIGINT NOT NULL FK | → `hackforger_phase_type.id` |
| activity_kind | VARCHAR(20) NOT NULL | Denormalized for query efficiency |
| activity_id | BIGINT NOT NULL | FK to hackathon/bounty/grant ID |
| sort_order | INT NOT NULL | Order within this activity |
| start_time | BIGINT NOT NULL | Unix timestamp |
| end_time | BIGINT NOT NULL | Unix timestamp, must be > start_time |
| created_unix | BIGINT NOT NULL | |
| updated_unix | BIGINT NOT NULL | |

**Index**: `IDX_phase_activity (activity_kind, activity_id, sort_order)`

### No `is_locked` column

Lock state is computed, not stored:
- `IsLocked()`: `phase.EndTime < time.Now().Unix()`
- `IsActive()`: `phase.StartTime <= time.Now().Unix() && phase.EndTime > time.Now().Unix()`
- `IsFuture()`: `phase.StartTime > time.Now().Unix()`

## 3.2 Action Vocabulary per Activity Kind

Each activity type defines its own set of action strings. Adding a new action requires code changes (register constant + implement business logic). Admin can freely assign existing actions to any phase type within the same activity_kind.

### Hackathon Actions

| Action | Description |
|--------|-------------|
| `register` | 参赛者报名 |
| `form_team` | 组队 |
| `submit_work` | 提交作品 |
| `score` | 评委打分 |
| `publish_results` | 发布结果 |

### Bounty Actions

| Action | Description |
|--------|-------------|
| `apply` | 申请认领 |
| `claim` | 确认认领 |
| `submit_pr` | 提交 PR |
| `review` | 评审 PR |
| `settle` | 结算奖励 |

### Grant Actions

| Action | Description |
|--------|-------------|
| `apply` | 提交申请 |
| `review` | 评审申请 |
| `approve` | 批准 |
| `disburse` | 拨款 |

### Default Phase Types (seed data)

**Hackathon**:

| key | display_name | unique | allowed_actions | order |
|-----|-------------|--------|----------------|-------|
| registration | hackforger.phase.hackathon.registration | Y | ["register", "form_team"] | 1 |
| development | hackforger.phase.hackathon.development | N | ["submit_work"] | 2 |
| judging | hackforger.phase.hackathon.judging | Y | ["score"] | 3 |
| results | hackforger.phase.hackathon.results | Y | ["publish_results"] | 4 |

**Bounty**:

| key | display_name | unique | allowed_actions | order |
|-----|-------------|--------|----------------|-------|
| open | hackforger.phase.bounty.open | Y | ["apply", "claim"] | 1 |
| in_progress | hackforger.phase.bounty.in_progress | Y | ["submit_pr"] | 2 |
| review | hackforger.phase.bounty.review | Y | ["review"] | 3 |
| closed | hackforger.phase.bounty.closed | Y | ["settle"] | 4 |

**Grant**:

| key | display_name | unique | allowed_actions | order |
|-----|-------------|--------|----------------|-------|
| application | hackforger.phase.grant.application | Y | ["apply"] | 1 |
| review | hackforger.phase.grant.review | Y | ["review", "approve"] | 2 |
| disbursement | hackforger.phase.grant.disbursement | Y | ["disburse"] | 3 |

## 3.3 Phase Controller (Service Layer)

**File**: `services/hackforger/phase.go`

### Core Functions

```
CurrentPhase(activityKind, activityID) → *Phase, error
```
Returns the phase where `start_time <= now < end_time`. Returns nil if between phases or no phases defined.

```
AllowsAction(activityKind, activityID, action string) → bool, error
```
Gets current phase, checks if `action` is in the phase type's `allowed_actions` JSON array.

```
UpdatePhaseTime(phaseID, startTime, endTime) → error
```
Validates: phase not locked (end_time >= now), new times don't overlap adjacent phases, end > start. On success: cancel old notification timer, register new one.

```
AddPhase(activityKind, activityID, phaseTypeID, sortOrder, startTime, endTime) → error
```
Validates: is_unique constraint (if phase type is unique, check no existing phase of same type for this activity), time doesn't overlap with adjacent phases.

```
RemovePhase(phaseID) → error
```
Only allowed if phase is future (not active or locked).

```
GetPhases(activityKind, activityID) → []*Phase, error
```
Returns all phases for an activity, ordered by sort_order.

### Business Logic Integration

Every service function that is phase-gated calls `AllowsAction()` first:

```go
// Example: hackathon submission
func CreateSubmission(ctx, hackathonID, userID, trackID, ...) error {
    allowed, err := phase.AllowsAction("hackathon", hackathonID, "submit_work")
    if err != nil { return err }
    if !allowed {
        return ErrPhaseActionNotAllowed  // i18n: hackforger.error.phase_action_not_allowed
    }
    // ... proceed with submission
}
```

## 3.4 Phase Transition Notifications

**Mechanism**: In-process Go timers (`time.AfterFunc`), not Forgejo queue (which doesn't support delayed scheduling).

**Implementation**: A `PhaseScheduler` singleton initialized at server startup:
- Maintains a `map[int64]*time.Timer` keyed by phase ID
- On phase create/update: `time.AfterFunc(duration, callback)` where `duration = startTime - now`
- Callback: calls `PublishHackforgerAction()` → writes Feed event + sends notification to watchers
- On phase time update: `timer.Stop()` + register new timer
- On phase delete: `timer.Stop()` + remove from map
- Notification content: `ctx.Tr("hackforger.notification.phase_entered", activityName, phaseName)` + i18n description of now-allowed actions

**Startup recovery**: On server start, `PhaseScheduler.Init()` queries all phases where `start_time > now` and registers timers. For phases where `start_time` has passed but no Feed event exists (detected by querying action table), retroactively publish the event.

**Memory**: Timer map is lightweight. Even 10,000 active phases use negligible memory.

## 3.5 Phase Timeline UI Component

### Go Template: `templates/hackforger/components/phase_timeline.tmpl`

Reusable component included anywhere phase display/editing is needed:

```html
{{template "hackforger/components/phase_timeline" dict "Phases" .Phases "Editable" .CanEdit}}
```

**Visual design**:
- Horizontal timeline with phase nodes connected by lines
- Each node shows: phase name (i18n), start/end time, status indicator
- **Locked** (past): Gray background, lock icon, no interaction
- **Active** (current): Highlighted/colored, pulse indicator, shows allowed actions
- **Future**: Default style, click to edit times (if Editable=true)
- Responsive: collapses to vertical on mobile

### Vue Enhancement: `web_src/js/features/hackforger/PhaseTimeline.vue`

- Inline time editing via date-time picker for future phases
- Calls `POST` web route to save time changes (session auth, not API)
- Optimistic UI update with error rollback
- Import `showErrorToast` for failure feedback

### Usage Locations

- Hackathon detail page (organizer view + participant view)
- Bounty detail page
- Grant Round detail page
- Admin activity management pages

## 3.6 Admin Phase Type Management

**Route**: `/admin/hackforger/phase-types`
**Template**: `templates/admin/hackforger/phase_types.tmpl`

**Features**:
- List all phase types grouped by activity_kind
- Create new phase type: select activity_kind → enter key, i18n display name → check action checkboxes (dynamically filtered by activity_kind) → set is_unique
- Edit existing: modify display name, actions, is_unique, default_order
- Delete: only if no active phases reference this type

## 3.7 API Routes

Phase data is needed by external consumers (Agent API, mobile). API endpoints in `routers/api/v1/hackforger/phase.go`:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/hackforger/{kind}/{id}/phases` | List all phases for an activity |
| GET | `/api/v1/hackforger/{kind}/{id}/current-phase` | Get current active phase |
| POST | `/api/v1/hackforger/{kind}/{id}/phases` | Add a phase (organizer) |
| PUT | `/api/v1/hackforger/phases/{phaseID}` | Update phase time (organizer) |
| DELETE | `/api/v1/hackforger/phases/{phaseID}` | Remove future phase (organizer) |
| GET | `/api/v1/hackforger/admin/phase-types` | List phase type catalog (admin) |
| POST | `/api/v1/hackforger/admin/phase-types` | Create phase type (admin) |
| PUT | `/api/v1/hackforger/admin/phase-types/{id}` | Update phase type (admin) |
| DELETE | `/api/v1/hackforger/admin/phase-types/{id}` | Delete phase type (admin) |

## 3.8 Migration

**File**: `models/forgejo_migrations/v_xxx.go`

**Wrapped in `db.WithTx` for atomic rollback on failure.**

1. Create `hackforger_phase_type` table
2. Create `hackforger_phase` table
3. Seed default phase types (Section 3.2 tables)
4. **Data migration for existing activities**: For each existing hackathon/bounty/grant that has no phases, generate default phase records using the activity's existing time fields (start_date, end_date) distributed across default phase types

If step 4 fails, the entire migration rolls back (no partial state).

## 3.9 Phase Gap Policy

Phases within an activity must be **non-overlapping** but **gaps are allowed**. During a gap (no active phase), `CurrentPhase()` returns nil and `AllowsAction()` returns false for all actions. This is a valid state -- it means the activity is "between phases" and no operations are permitted. The Phase Timeline UI shows gaps as a dimmed connector between nodes.

## 3.10 Action Validation

`allowed_actions` JSON is validated at two points:
1. **Admin write time**: When creating/updating a phase type, the API/web handler validates that all action strings in the JSON array exist in the registered action vocabulary for the given `activity_kind`. Invalid actions are rejected with `hackforger.error.invalid_phase_action`.
2. **Service read time**: `AllowsAction()` does a simple string match. If an action string in the DB doesn't match any registered action, it is silently ignored (no match = not allowed). This is safe -- the write-time validation prevents invalid data from entering.

## 3.11 Resolves

This wave resolves the following tickets:
- HF-003: Past phases locked (computed lock)
- HF-005: Auto phase advance (computed current phase + notification timers)
- HF-006: Custom phase stages (organizer adds/removes phases from type catalog)
