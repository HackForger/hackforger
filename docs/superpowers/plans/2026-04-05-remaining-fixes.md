# Remaining Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix Phase System logic gaps, UI bugs, and implement PhaseNotifier, Phase API, Admin PhaseType page, Bounty/Grant integration

**Architecture:** Fixes apply to existing files; new modules follow established patterns (gocron for scheduling, API handlers follow existing hackathon.go patterns)

**Tech Stack:** Go, XORM, gocron, Fomantic UI, Tailwind CSS, Vue 3

**Spec:** `docs/superpowers/specs/2026-04-05-remaining-fixes-design.md`

---

### Task 1: AllowsAction Fixes (A1-A3)

**Files:**
- Modify: `routers/api/v1/hackforger/hackathon_registration.go`
- Modify: `routers/api/v1/hackforger/hackathon_submission.go`
- Modify: `services/hackforger/hackathon_judge.go`

**Step 1: Fix API registration handler**

In `routers/api/v1/hackforger/hackathon_registration.go`, find the line:
```go
if h.StatusCache != hackforger_model.HackathonStatusOpen {
```
Replace with:
```go
canRegister, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register")
if !canRegister {
```

Add import `hackforger_service "forgejo.org/services/hackforger"` if not present.

**Step 2: Fix API submission handler**

In `routers/api/v1/hackforger/hackathon_submission.go`, find:
```go
if h.StatusCache != hackforger_model.HackathonStatusHacking {
```
Replace with:
```go
canSubmit, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "submit_work")
if !canSubmit {
```

**Step 3: Fix judge service**

In `services/hackforger/hackathon_judge.go`, find:
```go
if h.StatusCache != hackforger_model.HackathonStatusJudging {
    return hackforger_model.ErrInvalidHackathonPhase{...}
}
```
Replace with:
```go
canScore, _ := AllowsAction(ctx, "hackathon", h.ID, "score")
if !canScore {
    return hackforger_model.ErrInvalidHackathonPhase{
        HackathonID: h.ID, Current: h.StatusCache, Expected: hackforger_model.HackathonStatusJudging,
    }
}
```

**Step 4: Verify build**

```bash
go build ./...
```

---

### Task 2: UI Bug Fixes (B1-B6)

**Files:**
- Modify: `templates/hackforger/hackathon/manage.tmpl`
- Modify: `templates/hackforger/hackathon/view.tmpl`
- Modify: `templates/hackforger/hackathon/submit.tmpl`
- Modify: `templates/hackforger/grants/submit.tmpl`
- Modify: `templates/hackforger/credits/admin/orders.tmpl`
- Modify: `web_src/js/components/hackforger/PhaseTimeline.vue`
- Modify: `web_src/js/components/hackforger/BountyPanel.vue`

**Step 1: Fix Fomantic dropdown classes (B1-B3)**

Search all hackforger templates for `class="ui dropdown"` without `selection` and add it:
- `class="ui dropdown"` → `class="ui dropdown selection"`
- `class="ui search dropdown"` → `class="ui search selection dropdown"`

Files to check: manage.tmpl, view.tmpl, submit.tmpl, grants/submit.tmpl

**Step 2: Fix PhaseTimeline.vue widths (B4)**

Replace `tw-w-32` on phase name elements with `tw-min-w-24 tw-max-w-48 tw-truncate`.
Replace `tw-w-32` on rename input with `tw-min-w-24 tw-max-w-48`.

**Step 3: Fix Credits admin widths (B5)**

In `credits/admin/orders.tmpl`, replace `class="tw-w-24"` on inputs with `class="tw-min-w-16 tw-flex-1"`.

**Step 4: Fix BountyPanel.vue inline style (B6)**

Replace `style="width: 100px"` with `class="tw-w-24"`.

**Step 5: Build**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
make frontend
```

---

### Task 3: PhaseNotifier (C1)

**Files:**
- Create: `services/hackforger/phase_notifier.go`
- Modify: `services/cron/cron.go` (export scheduler)
- Modify: `services/cron/tasks_hackforger.go` (fill cron body + call RebuildPhaseSchedule)
- Modify: `services/hackforger/phase.go` (call Schedule/Unschedule on CRUD)

**Step 1: Export scheduler from cron package**

In `services/cron/cron.go`, the scheduler is package-private:
```go
var scheduler = gocron.NewScheduler(time.Local)
```

Add a getter:
```go
// GetScheduler returns the global gocron scheduler for external job registration.
func GetScheduler() *gocron.Scheduler {
    return scheduler
}
```

**Step 2: Create phase_notifier.go**

```go
// services/hackforger/phase_notifier.go
package hackforger

import (
    "context"
    "fmt"
    "time"

    hackforger_model "forgejo.org/models/hackforger"
    "forgejo.org/modules/log"
    "forgejo.org/services/cron"
)

// SchedulePhaseJobs registers gocron one-shot jobs for a phase's start and end times.
func SchedulePhaseJobs(phase *hackforger_model.Phase) {
    s := cron.GetScheduler()
    now := time.Now().Unix()

    if phase.StartTime > now {
        startTag := fmt.Sprintf("phase-%d-start", phase.ID)
        startTime := time.Unix(phase.StartTime, 0)
        _, err := s.Every(1).LimitRunsTo(1).StartAt(startTime).Tag(startTag).Do(onPhaseEvent, phase.ActivityKind, phase.ActivityID, phase.ID, "start")
        if err != nil {
            log.Error("SchedulePhaseJobs: failed to schedule start for phase %d: %v", phase.ID, err)
        }
    }

    if phase.EndTime > now {
        endTag := fmt.Sprintf("phase-%d-end", phase.ID)
        endTime := time.Unix(phase.EndTime, 0)
        _, err := s.Every(1).LimitRunsTo(1).StartAt(endTime).Tag(endTag).Do(onPhaseEvent, phase.ActivityKind, phase.ActivityID, phase.ID, "end")
        if err != nil {
            log.Error("SchedulePhaseJobs: failed to schedule end for phase %d: %v", phase.ID, err)
        }
    }
}

// UnschedulePhaseJobs removes gocron jobs for a phase.
func UnschedulePhaseJobs(phaseID int64) {
    s := cron.GetScheduler()
    _ = s.RemoveByTag(fmt.Sprintf("phase-%d-start", phaseID))
    _ = s.RemoveByTag(fmt.Sprintf("phase-%d-end", phaseID))
}

// RebuildPhaseSchedule scans all future phases from DB and registers gocron jobs.
// Called at server startup.
func RebuildPhaseSchedule(ctx context.Context) error {
    now := time.Now().Unix()
    // Query all phases where end_time > now (still relevant)
    phases, err := hackforger_model.GetFuturePhases(ctx, now)
    if err != nil {
        return fmt.Errorf("RebuildPhaseSchedule: %w", err)
    }
    for _, phase := range phases {
        SchedulePhaseJobs(phase)
    }
    log.Info("RebuildPhaseSchedule: registered %d phase jobs", len(phases))
    return nil
}

// onPhaseEvent is the callback fired by gocron when a phase starts or ends.
func onPhaseEvent(activityKind string, activityID, phaseID int64, event string) {
    ctx := context.Background()
    log.Info("PhaseNotifier: phase %d %s (activity: %s/%d)", phaseID, event, activityKind, activityID)

    if activityKind == "hackathon" {
        if err := SyncStatusCache(ctx, activityID, 0); err != nil {
            log.Error("PhaseNotifier: SyncStatusCache failed for hackathon %d: %v", activityID, err)
        }
    }
    // Future: handle bounty/grant phase events
}
```

**Step 3: Add GetFuturePhases to model**

In `models/hackforger/phase.go`:
```go
// GetFuturePhases returns all phases where end_time > the given timestamp.
func GetFuturePhases(ctx context.Context, afterUnix int64) ([]*Phase, error) {
    phases := make([]*Phase, 0)
    return phases, db.GetEngine(ctx).
        Where("end_time > ?", afterUnix).
        Find(&phases)
}
```

**Step 4: Integrate with cron startup**

In `services/cron/tasks_hackforger.go`, update `registerHackforgerHackathonStatus`:
```go
func registerHackforgerHackathonStatus() {
    RegisterTaskFatal("hackforger_hackathon_status", &BaseConfig{
        Enabled:    true,
        RunAtStart: true,  // Run at start to rebuild schedule
        Schedule:   "@every 5m",
    }, func(ctx context.Context, _ *user_model.User, _ Config) error {
        return hackforger_service.CheckHackathonTransitions(ctx)
    })
}
```

Add `RebuildPhaseSchedule` call in `initHackforgerTasks`:
```go
func initHackforgerTasks() {
    registerHackforgerHackathonStatus()
    registerHackforgerBountyExpiry()
    registerHackforgerReputationRecalc()
    registerHackforgerGrantDeadline()

    // Rebuild phase schedule from DB after all cron tasks are registered
    go func() {
        // Small delay to ensure DB is ready
        time.Sleep(5 * time.Second)
        if err := hackforger_service.RebuildPhaseSchedule(context.Background()); err != nil {
            log.Error("initHackforgerTasks: %v", err)
        }
    }()
}
```

**Step 5: Create CheckHackathonTransitions (fallback sweep)**

In `services/hackforger/phase_notifier.go`:
```go
// CheckHackathonTransitions is the fallback cron sweep that syncs all published hackathons.
func CheckHackathonTransitions(ctx context.Context) error {
    hackathons, err := hackforger_model.ListPublishedHackathons(ctx)
    if err != nil {
        return err
    }
    for _, h := range hackathons {
        if syncErr := SyncStatusCache(ctx, h.ID, 0); syncErr != nil {
            log.Error("CheckHackathonTransitions: hackathon %d: %v", h.ID, syncErr)
        }
    }
    return nil
}
```

Add `ListPublishedHackathons` to model:
```go
func ListPublishedHackathons(ctx context.Context) ([]*Hackathon, error) {
    hackathons := make([]*Hackathon, 0)
    return hackathons, db.GetEngine(ctx).
        Where("is_published = ?", true).
        Find(&hackathons)
}
```

**Step 6: Hook into Phase CRUD**

In `services/hackforger/phase.go`, after each CRUD operation:
- `AddPhase`: after `CreatePhase` → `SchedulePhaseJobs(phase)`
- `UpdatePhaseTime`: after update → `UnschedulePhaseJobs(id)` then re-fetch and `SchedulePhaseJobs(phase)`
- `RemovePhase`: before delete → `UnschedulePhaseJobs(id)`

**Step 7: Verify build**

```bash
go build ./...
```

---

### Task 4: Phase API Routes (C2)

**Files:**
- Create: `routers/api/v1/hackforger/phase.go`
- Modify: `routers/api/v1/api.go` (add routes)

**Step 1: Create API handlers**

```go
// routers/api/v1/hackforger/phase.go
package hackforger

import (
    "net/http"

    hackforger_model "forgejo.org/models/hackforger"
    "forgejo.org/modules/json"
    hackforger_service "forgejo.org/services/hackforger"
    "forgejo.org/services/context"
)

func ListPhases(ctx *context.APIContext) {
    h := getHackathonFromPath(ctx)
    if h == nil { return }
    phases, err := hackforger_service.GetPhases(ctx, "hackathon", h.ID)
    if err != nil {
        ctx.Error(http.StatusInternalServerError, "ListPhases", err)
        return
    }
    ctx.JSON(http.StatusOK, phases)
}

func AddPhaseAPI(ctx *context.APIContext) {
    h := getHackathonFromPath(ctx)
    if h == nil { return }
    var req struct {
        PhaseTypeID int64 `json:"phase_type_id"`
        StartTime   int64 `json:"start_time"`
        EndTime     int64 `json:"end_time"`
        SortOrder   int   `json:"sort_order"`
    }
    if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
        ctx.Error(http.StatusBadRequest, "AddPhase", err)
        return
    }
    phase, err := hackforger_service.AddPhase(ctx, "hackathon", h.ID, req.PhaseTypeID, req.SortOrder, req.StartTime, req.EndTime)
    if err != nil {
        ctx.Error(http.StatusBadRequest, "AddPhase", err)
        return
    }
    ctx.JSON(http.StatusCreated, phase)
}

func UpdatePhaseAPI(ctx *context.APIContext) {
    h := getHackathonFromPath(ctx)
    if h == nil { return }
    phaseID := ctx.ParamsInt64("pid")
    var req struct {
        StartTime  int64  `json:"start_time"`
        EndTime    int64  `json:"end_time"`
        CustomName string `json:"custom_name"`
    }
    if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
        ctx.Error(http.StatusBadRequest, "UpdatePhase", err)
        return
    }
    if err := hackforger_service.UpdatePhaseTime(ctx, phaseID, req.StartTime, req.EndTime); err != nil {
        ctx.Error(http.StatusBadRequest, "UpdatePhase", err)
        return
    }
    if req.CustomName != "" {
        if phase, _ := hackforger_model.GetPhaseByID(ctx, phaseID); phase != nil {
            phase.CustomName = req.CustomName
            _ = hackforger_model.UpdatePhase(ctx, phase)
        }
    }
    phases, _ := hackforger_service.GetPhases(ctx, "hackathon", h.ID)
    ctx.JSON(http.StatusOK, phases)
}

func DeletePhaseAPI(ctx *context.APIContext) {
    h := getHackathonFromPath(ctx)
    if h == nil { return }
    phaseID := ctx.ParamsInt64("pid")
    if err := hackforger_service.RemovePhase(ctx, phaseID); err != nil {
        ctx.Error(http.StatusBadRequest, "DeletePhase", err)
        return
    }
    ctx.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func GetCurrentPhase(ctx *context.APIContext) {
    h := getHackathonFromPath(ctx)
    if h == nil { return }
    phase, err := hackforger_service.CurrentPhase(ctx, "hackathon", h.ID)
    if err != nil {
        ctx.Error(http.StatusInternalServerError, "GetCurrentPhase", err)
        return
    }
    if phase == nil {
        ctx.JSON(http.StatusOK, map[string]any{"phase": nil})
        return
    }
    ctx.JSON(http.StatusOK, phase)
}

func ReorderPhasesAPI(ctx *context.APIContext) {
    h := getHackathonFromPath(ctx)
    if h == nil { return }
    var req struct {
        Orders []hackforger_model.PhaseSortOrder `json:"orders"`
    }
    if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
        ctx.Error(http.StatusBadRequest, "ReorderPhases", err)
        return
    }
    if err := hackforger_model.BatchUpdatePhaseSortOrder(ctx, req.Orders); err != nil {
        ctx.Error(http.StatusInternalServerError, "ReorderPhases", err)
        return
    }
    phases, _ := hackforger_service.GetPhases(ctx, "hackathon", h.ID)
    ctx.JSON(http.StatusOK, phases)
}
```

**Step 2: Register routes in api.go**

In `routers/api/v1/api.go`, inside the hackathon `m.Group("/{id}", ...)` block, add:
```go
m.Get("/phases", hackforger_api.ListPhases)
m.Post("/phases", reqToken(), hackforger_api.AddPhaseAPI)
m.Post("/phases/reorder", reqToken(), hackforger_api.ReorderPhasesAPI)
m.Put("/phases/{pid}", reqToken(), hackforger_api.UpdatePhaseAPI)
m.Delete("/phases/{pid}", reqToken(), hackforger_api.DeletePhaseAPI)
m.Get("/current-phase", hackforger_api.GetCurrentPhase)
```

**Step 3: Verify build**

```bash
go build ./...
```

---

### Task 5: Admin Phase Type Management (C3)

**Files:**
- Create: `templates/hackforger/admin/phase_types.tmpl`
- Create: `routers/web/hackforger/admin_phase_types.go`
- Modify: `routers/web/web.go` (add admin routes)
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

**Step 1: Create admin route handlers**

```go
// routers/web/hackforger/admin_phase_types.go
package hackforger

import (
    "net/http"
    "strconv"

    hackforger_model "forgejo.org/models/hackforger"
    "forgejo.org/modules/log"
    "forgejo.org/services/context"
)

const tplAdminPhaseTypes = "hackforger/admin/phase_types"

func AdminPhaseTypes(ctx *context.Context) {
    ctx.Data["Title"] = ctx.Tr("hackforger.admin.phase_types")
    phaseTypes, _ := hackforger_model.GetAllPhaseTypes(ctx)
    // Group by activity_kind
    grouped := make(map[string][]*hackforger_model.PhaseType)
    for _, pt := range phaseTypes {
        grouped[pt.ActivityKind] = append(grouped[pt.ActivityKind], pt)
    }
    ctx.Data["GroupedPhaseTypes"] = grouped
    ctx.Data["ActivityKinds"] = []string{"hackathon", "bounty", "grant"}
    ctx.HTML(http.StatusOK, tplAdminPhaseTypes)
}

func AdminPhaseTypeCreate(ctx *context.Context) {
    pt := &hackforger_model.PhaseType{
        ActivityKind:    ctx.FormString("activity_kind"),
        Key:             ctx.FormString("key"),
        DisplayNameI18n: ctx.FormString("display_name_i18n"),
        IsUnique:        ctx.FormString("is_unique") == "on",
        AllowedActions:  ctx.FormString("allowed_actions"),
        DefaultOrder:    ctx.FormInt("default_order"),
    }
    if err := hackforger_model.CreatePhaseType(ctx, pt); err != nil {
        log.Error("AdminPhaseTypeCreate: %v", err)
        ctx.Flash.Error("Failed to create phase type")
    } else {
        ctx.Flash.Success("Phase type created")
    }
    ctx.Redirect("/admin/hackforger/phase-types")
}

func AdminPhaseTypeUpdate(ctx *context.Context) {
    id := ctx.ParamsInt64("id")
    pt, _ := hackforger_model.GetPhaseTypeByID(ctx, id)
    if pt == nil {
        ctx.NotFound("PhaseType not found", nil)
        return
    }
    pt.Key = ctx.FormString("key")
    pt.DisplayNameI18n = ctx.FormString("display_name_i18n")
    pt.IsUnique = ctx.FormString("is_unique") == "on"
    pt.AllowedActions = ctx.FormString("allowed_actions")
    pt.DefaultOrder = ctx.FormInt("default_order")
    if err := hackforger_model.UpdatePhaseType(ctx, pt); err != nil {
        log.Error("AdminPhaseTypeUpdate: %v", err)
        ctx.Flash.Error("Failed to update phase type")
    } else {
        ctx.Flash.Success("Phase type updated")
    }
    ctx.Redirect("/admin/hackforger/phase-types")
}

func AdminPhaseTypeDelete(ctx *context.Context) {
    id := ctx.ParamsInt64("id")
    if err := hackforger_model.DeletePhaseType(ctx, id); err != nil {
        log.Error("AdminPhaseTypeDelete: %v", err)
        ctx.Flash.Error("Failed to delete phase type")
    } else {
        ctx.Flash.Success("Phase type deleted")
    }
    ctx.Redirect("/admin/hackforger/phase-types")
}
```

Also add `GetAllPhaseTypes` to the model:
```go
func GetAllPhaseTypes(ctx context.Context) ([]*PhaseType, error) {
    types := make([]*PhaseType, 0)
    return types, db.GetEngine(ctx).OrderBy("activity_kind ASC, default_order ASC").Find(&types)
}
```

**Step 2: Create template**

Standard Forgejo admin page with table per activity_kind, create form, edit/delete buttons.

**Step 3: Register routes**

In `web.go`, add under admin group:
```go
m.Group("/hackforger/phase-types", func() {
    m.Get("", hackforger_web.AdminPhaseTypes)
    m.Post("", hackforger_web.AdminPhaseTypeCreate)
    m.Post("/{id}/edit", hackforger_web.AdminPhaseTypeUpdate)
    m.Post("/{id}/delete", hackforger_web.AdminPhaseTypeDelete)
})
```

**Step 4: Add i18n keys**

```ini
; locale_en-US.ini
admin.phase_types = Phase Types
admin.phase_types.title = Phase Type Management
admin.phase_types.activity_kind = Activity Kind
admin.phase_types.key = Key
admin.phase_types.display_name = Display Name (i18n key)
admin.phase_types.is_unique = Unique
admin.phase_types.allowed_actions = Allowed Actions (JSON)
admin.phase_types.default_order = Order
admin.phase_types.create = Create Phase Type
admin.phase_types.edit = Edit
admin.phase_types.delete = Delete
admin.phase_types.confirm_delete = Are you sure you want to delete this phase type?
```

---

### Task 6: Bounty/Grant Phase Integration (C4)

**Files:**
- Modify: `services/hackforger/bounty.go`
- Modify: `services/hackforger/grant.go`
- Modify: `services/cron/tasks_hackforger.go`

**Step 1: Add AllowsAction to bounty operations**

For each bounty service function that checks status, add a phase-aware check:
```go
// Before the existing status check, add:
if allowed, _ := AllowsAction(ctx, "bounty", bountyID, "apply"); !allowed {
    // If phases are configured and don't allow this action, block it
    // Note: if no phases exist for this bounty, AllowsAction returns false (no active phase)
    // We need to handle backward compatibility
}
```

**Important: Backward compatibility.** Bounties/Grants don't have Phase records yet. `AllowsAction` returns false when no phase is active (including when no phases exist). We need a helper:

```go
// HasPhases checks if any phases are configured for an activity.
func HasPhases(ctx context.Context, activityKind string, activityID int64) (bool, error) {
    count, err := hackforger_model.CountPhasesByActivity(ctx, activityKind, activityID)
    return count > 0, err
}
```

Then the gating pattern becomes:
```go
hasPhases, _ := HasPhases(ctx, "bounty", bountyID)
if hasPhases {
    allowed, _ := AllowsAction(ctx, "bounty", bountyID, "apply")
    if !allowed {
        return ErrPhaseActionBlocked{Action: "apply"}
    }
}
// ... existing status check continues (backward compatible)
```

**Step 2: Same pattern for grant operations**

Apply to: SubmitProject, ApproveProject, DisburseProject

**Step 3: Fill cron bodies**

In `tasks_hackforger.go`:
- `registerHackforgerBountyExpiry` → call `hackforger_service.ExpireOldBounties(ctx)`
- `registerHackforgerGrantDeadline` → call `hackforger_service.CheckGrantDeadlines(ctx)` (create stub if doesn't exist)

**Step 4: Add model helper**

```go
func CountPhasesByActivity(ctx context.Context, activityKind string, activityID int64) (int64, error) {
    return db.GetEngine(ctx).Where("activity_kind = ? AND activity_id = ?", activityKind, activityID).Count(&Phase{})
}
```

**Step 5: Verify build**

```bash
go build ./...
```

---

### Task 7: Final Build + Smoke Test

**Step 1:** Build everything
```bash
make frontend
TAGS="bindata sqlite sqlite_unlock_notify" make backend
```

**Step 2:** Restart server, verify:
- Manage page loads with Phase Timeline (date pickers, disabled dropdown, rename, drag)
- Phase API endpoints respond (`/api/v1/hackforger/hackathons/{id}/phases`)
- Admin phase types page loads (`/admin/hackforger/phase-types`)
- PhaseNotifier logs on startup ("registered N phase jobs")
- AllowsAction blocks registration/submission/scoring outside phase windows
