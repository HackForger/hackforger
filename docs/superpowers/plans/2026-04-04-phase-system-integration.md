# Phase System Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace manual hackathon phase control buttons with time-driven automatic phase transitions using the existing Phase System.

**Architecture:** Rename `Status` to `StatusCache` (computed cache from Phase table). Add `IsPublished` flag. `SyncStatusCache()` derives status from active phases with lifecycle hooks. Phase Timeline Vue component replaces manual buttons on manage page.

**Tech Stack:** Go (XORM), Vue 3, Go templates, SQLite3

**Spec:** `docs/superpowers/specs/2026-04-04-phase-system-integration-design.md`

---

## Task Dependency Graph

```
Task 1 (Migration)
    │
    ▼
Task 2 (Model rename)
    │
    ├─────────────────────┐
    ▼                     ▼
Task 3 (SyncStatusCache) Task 4 (UpdatePhaseTime guard)
    │
    ▼
Task 5 (Phase CRUD routes)
    │
    ├─────────────────┐
    ▼                 ▼
Task 6 (Vue component) Task 7 (Action gating)
    │                 │
    └────────┬────────┘
             ▼
Task 8 (Manage page)
             │
             ▼
Task 9 (View page)
             │
             ▼
Task 10 (Publish flow)
             │
             ▼
Task 11 (Cleanup old handlers)
```

**Concurrency:** Tasks 3+4 can run in parallel. Tasks 6+7 can run in parallel.

---

### Task 1: Database Migration

**Files:**
- Create: `models/forgejo_migrations/v14b_hackforger-phase-integration.go`

- [ ] **Step 1: Write the migration file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "Rename hackathon.status to status_cache, add is_published, migrate time fields to phase records",
		Upgrade:     hackforgerPhaseIntegration,
	})
}

func hackforgerPhaseIntegration(x *xorm.Engine) error {
	// 1. Rename status -> status_cache
	if _, err := x.Exec("ALTER TABLE hackathon RENAME COLUMN status TO status_cache"); err != nil {
		return err
	}

	// 2. Add is_published column
	if _, err := x.Exec("ALTER TABLE hackathon ADD COLUMN is_published INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}

	// 3. Set is_published = true for non-Draft hackathons
	if _, err := x.Exec("UPDATE hackathon SET is_published = 1 WHERE status_cache > 0"); err != nil {
		return err
	}

	// 4. Look up phase_type IDs for hackathon kinds
	type ptRow struct {
		ID  int64  `xorm:"id"`
		Key string `xorm:"key"`
	}
	var pts []ptRow
	if err := x.SQL("SELECT id, key FROM phase_type WHERE activity_kind = 'hackathon'").Find(&pts); err != nil {
		return err
	}
	ptMap := make(map[string]int64)
	for _, pt := range pts {
		ptMap[pt.Key] = pt.ID
	}

	// 5. For each published hackathon, generate Phase records from time fields
	type hackRow struct {
		ID                int64 `xorm:"id"`
		StatusCache       int   `xorm:"status_cache"`
		RegistrationStart int64 `xorm:"registration_start"`
		RegistrationEnd   int64 `xorm:"registration_end"`
		HackingStart      int64 `xorm:"hacking_start"`
		HackingEnd        int64 `xorm:"hacking_end"`
		JudgingEnd        int64 `xorm:"judging_end"`
		CreatedUnix       int64 `xorm:"created_unix"`
	}
	var hacks []hackRow
	if err := x.SQL("SELECT id, status_cache, registration_start, registration_end, hacking_start, hacking_end, judging_end, created_unix FROM hackathon WHERE status_cache > 0").Find(&hacks); err != nil {
		return err
	}

	now := timeNow()
	for _, h := range hacks {
		order := 1
		// Helper: use value or reasonable default
		orDefault := func(val, fallback int64) int64 {
			if val > 0 {
				return val
			}
			return fallback
		}

		// Registration phase
		regStart := orDefault(h.RegistrationStart, h.CreatedUnix)
		regEnd := orDefault(h.RegistrationEnd, regStart+7*86400) // 7 days default
		if ptID, ok := ptMap["registration"]; ok && regEnd > regStart {
			if _, err := x.Exec(
				"INSERT INTO phase (phase_type_id, activity_kind, activity_id, sort_order, start_time, end_time, created_unix, updated_unix) VALUES (?, 'hackathon', ?, ?, ?, ?, ?, ?)",
				ptID, h.ID, order, regStart, regEnd, now, now,
			); err != nil {
				return err
			}
			order++
		}

		// Development phase
		devStart := orDefault(h.HackingStart, regEnd)
		devEnd := orDefault(h.HackingEnd, devStart+14*86400) // 14 days default
		if ptID, ok := ptMap["development"]; ok && devEnd > devStart {
			if _, err := x.Exec(
				"INSERT INTO phase (phase_type_id, activity_kind, activity_id, sort_order, start_time, end_time, created_unix, updated_unix) VALUES (?, 'hackathon', ?, ?, ?, ?, ?, ?)",
				ptID, h.ID, order, devStart, devEnd, now, now,
			); err != nil {
				return err
			}
			order++
		}

		// Judging phase
		judgStart := devEnd
		judgEnd := orDefault(h.JudgingEnd, judgStart+7*86400) // 7 days default
		if ptID, ok := ptMap["judging"]; ok && judgEnd > judgStart {
			if _, err := x.Exec(
				"INSERT INTO phase (phase_type_id, activity_kind, activity_id, sort_order, start_time, end_time, created_unix, updated_unix) VALUES (?, 'hackathon', ?, ?, ?, ?, ?, ?)",
				ptID, h.ID, order, judgStart, judgEnd, now, now,
			); err != nil {
				return err
			}
			order++
		}

		// Results phase
		resStart := judgEnd
		resEnd := resStart + 30*86400 // 30 days default
		if ptID, ok := ptMap["results"]; ok {
			if _, err := x.Exec(
				"INSERT INTO phase (phase_type_id, activity_kind, activity_id, sort_order, start_time, end_time, created_unix, updated_unix) VALUES (?, 'hackathon', ?, ?, ?, ?, ?, ?)",
				ptID, h.ID, order, resStart, resEnd, now, now,
			); err != nil {
				return err
			}
		}
	}

	// NOTE: Old time columns (registration_start, etc.) are intentionally NOT dropped.
	// They will be removed in a future migration after validation.
	return nil
}

```

Remove the standalone `timeNow()` function — use `now := time.Now().Unix()` inline at the top of `hackforgerPhaseIntegration` instead.

**Important:** The `"time"` import must be in the import block. The `timeNow` helper shown above must be defined inside `hackforgerPhaseIntegration` as `now := time.Now().Unix()` (not as a package-level function, to avoid namespace collision with other migration files).

- [ ] **Step 2: Verify migration compiles**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./models/forgejo_migrations/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add models/forgejo_migrations/v14b_hackforger-phase-integration.go
git commit -m "feat(phase): add migration for StatusCache rename, IsPublished, and phase data migration"
```

---

### Task 2: Model Rename (Status → StatusCache)

**Files:**
- Modify: `models/hackforger/hackathon.go`

This is a mechanical rename across the entire codebase. Every reference to `h.Status`, `.Hackathon.Status`, `hackathon.status` must become `StatusCache`/`status_cache`.

- [ ] **Step 1: Update the Hackathon struct and model functions**

In `models/hackforger/hackathon.go`:

```go
// Hackathon struct: rename Status field, add IsPublished
type Hackathon struct {
	// ... existing fields ...
	// StatusCache is computed from the Phase system. Do NOT set directly.
	// Use SyncStatusCache() to update. Source of truth is the phase table.
	StatusCache   HackathonStatus    `xorm:"'status_cache' NOT NULL DEFAULT 0"`
	IsPublished   bool               `xorm:"NOT NULL DEFAULT false"`
	// ... rest of fields ...
}
```

Update `UpdateHackathonStatus` → `UpdateHackathonStatusCache`:

```go
// UpdateHackathonStatusCache updates the status_cache field. Internal use only.
// External callers should use SyncStatusCache() instead.
func UpdateHackathonStatusCache(ctx context.Context, id int64, status HackathonStatus) error {
	_, err := db.GetEngine(ctx).ID(id).Cols("status_cache").Update(&Hackathon{StatusCache: status})
	return err
}
```

Update `ListHackathonsOptions.ToConds()`:

```go
if opts.Status != nil {
	cond = cond.And(builder.Eq{"hackathon.status_cache": *opts.Status})
}
```

Update `DeleteHackathon`:

```go
if h.StatusCache != HackathonStatusDraft {
	return fmt.Errorf("hackathon can only be deleted in draft status [id: %d, status: %d]", id, h.StatusCache)
}
```

- [ ] **Step 2: Rename all .Status references in services**

In `services/hackforger/hackathon.go`: replace all `h.Status` with `h.StatusCache`, all `UpdateHackathonStatus` with `UpdateHackathonStatusCache`.

In `services/hackforger/hackathon_judge.go`: replace `h.Status` with `h.StatusCache`.

In `services/hackforger/hackathon_criteria.go`: replace `h.Status` with `h.StatusCache`.

In `services/hackforger/search.go`: replace `h.Status` with `h.StatusCache`.

- [ ] **Step 3: Rename all .Status references in routers**

In `routers/web/hackforger/hackathon.go`: replace all `h.Status` with `h.StatusCache`.

In `routers/web/hackforger/submission.go`: replace `h.Status` with `h.StatusCache`, rename struct field `HackathonStatus` to `HackathonStatusCache`.

In `routers/api/v1/hackforger/hackathon.go`, `hackathon_registration.go`, `hackathon_submission.go`: replace `h.Status` with `h.StatusCache`.

- [ ] **Step 4: Rename all .Hackathon.Status in templates**

In `templates/hackforger/hackathon/view.tmpl`: replace `.Hackathon.Status` with `.Hackathon.StatusCache`.

In `templates/hackforger/hackathon/manage.tmpl`: replace `.Hackathon.Status` with `.Hackathon.StatusCache`.

In `templates/hackforger/hackathon/explore.tmpl` and `templates/hackforger/explore.tmpl`: replace `.Status` with `.StatusCache` where it refers to hackathon status.

- [ ] **Step 5: Update tests**

In `tests/integration/hackforger_hackathon_test.go`: replace `h.Status` with `h.StatusCache`.

- [ ] **Step 6: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Successful compilation

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor(phase): rename hackathon.Status to StatusCache, add IsPublished field"
```

---

### Task 3: SyncStatusCache Logic + Lifecycle Hooks

**Files:**
- Create: `services/hackforger/phase_sync.go`
- Modify: `models/hackforger/hackathon.go` (add `SetIsPublished`)

- [ ] **Step 1: Add helper model functions**

In `models/hackforger/hackathon.go`, add:

```go
// SetIsPublished updates the is_published flag.
func SetIsPublished(ctx context.Context, id int64, published bool) error {
	_, err := db.GetEngine(ctx).ID(id).Cols("is_published").Update(&Hackathon{IsPublished: published})
	return err
}
```

- [ ] **Step 2: Create phase_sync.go with SyncStatusCache**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	hackforger_model "forgejo.org/models/hackforger"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	notify_service "forgejo.org/services/notify"
)

// phaseKeyToStatus maps PhaseType.Key to HackathonStatus.
var phaseKeyToStatus = map[string]hackforger_model.HackathonStatus{
	"registration": hackforger_model.HackathonStatusOpen,
	"development":  hackforger_model.HackathonStatusHacking,
	"judging":      hackforger_model.HackathonStatusJudging,
	"results":      hackforger_model.HackathonStatusFinished,
}

// SyncStatusCache recomputes a hackathon's StatusCache from its Phase records.
// If the status changes, it fires lifecycle hooks and publishes a feed event.
// doerID is the user who triggered the change (0 for system/lazy sync).
func SyncStatusCache(ctx context.Context, hackathonID int64, doerID int64) error {
	h, err := hackforger_model.GetHackathonByID(ctx, hackathonID)
	if err != nil {
		return err
	}

	newStatus := computeStatusFromPhases(ctx, h)
	if newStatus == h.StatusCache {
		return nil // no change
	}

	oldStatus := h.StatusCache
	if err := hackforger_model.UpdateHackathonStatusCache(ctx, h.ID, newStatus); err != nil {
		return err
	}

	// Fire lifecycle hooks
	runLifecycleHooks(ctx, h, oldStatus, newStatus, doerID)

	// Publish feed event
	eventDoerID := doerID
	if eventDoerID == 0 {
		eventDoerID = h.OwnerID // system-triggered: use owner as doer
	}
	publishPhaseChange(ctx, eventDoerID, h, oldStatus, newStatus)

	return nil
}

// computeStatusFromPhases derives the HackathonStatus from Phase records.
func computeStatusFromPhases(ctx context.Context, h *hackforger_model.Hackathon) hackforger_model.HackathonStatus {
	// Cancelled is a manual override — never auto-change away from it
	if h.StatusCache == hackforger_model.HackathonStatusCancelled {
		return hackforger_model.HackathonStatusCancelled
	}

	// Not published → always Draft
	if !h.IsPublished {
		return hackforger_model.HackathonStatusDraft
	}

	phases, err := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
	if err != nil || len(phases) == 0 {
		return hackforger_model.HackathonStatusDraft
	}

	// Check for active phase
	current, _ := hackforger_model.GetCurrentPhase(ctx, "hackathon", h.ID)
	if current != nil && current.PhaseType != nil {
		if status, ok := phaseKeyToStatus[current.PhaseType.Key]; ok {
			return status
		}
	}

	// No active phase — find next future phase (gap handling)
	for _, p := range phases {
		if p.IsFuture() && p.PhaseType != nil {
			if status, ok := phaseKeyToStatus[p.PhaseType.Key]; ok {
				return status
			}
		}
	}

	// All phases past or no future — use last phase's mapped status
	last := phases[len(phases)-1]
	if last.PhaseType != nil {
		if status, ok := phaseKeyToStatus[last.PhaseType.Key]; ok {
			return status
		}
	}

	return hackforger_model.HackathonStatusDraft
}

// runLifecycleHooks fires side effects when status transitions occur.
func runLifecycleHooks(ctx context.Context, h *hackforger_model.Hackathon, oldStatus, newStatus hackforger_model.HackathonStatus, doerID int64) {
	switch {
	case oldStatus == hackforger_model.HackathonStatusOpen && newStatus == hackforger_model.HackathonStatusHacking:
		// Open → Hacking: tag track repos with v0-kickoff baseline
		if doerID > 0 {
			tagTrackRepos(ctx, doerID, h.ID, "v0-kickoff", "Hackathon kickoff baseline")
		}

	case oldStatus == hackforger_model.HackathonStatusHacking && newStatus == hackforger_model.HackathonStatusJudging:
		// Hacking → Judging: warn if no criteria or submissions
		criteriaCount, _ := hackforger_model.CountCriteriaByHackathon(ctx, h.ID)
		subCount, _ := hackforger_model.CountSubmissions(ctx, h.ID)
		if criteriaCount == 0 {
			log.Warn("Hackathon %d entered judging phase with no scoring criteria", h.ID)
		}
		if subCount == 0 {
			log.Warn("Hackathon %d entered judging phase with no submissions", h.ID)
		}
	}
}

// EnsureStatusCacheFresh lazily syncs StatusCache when loading a hackathon.
// Call this in route handlers after loading a hackathon.
func EnsureStatusCacheFresh(ctx context.Context, h *hackforger_model.Hackathon) {
	if h.StatusCache == hackforger_model.HackathonStatusCancelled {
		return // cancelled is manual, never auto-sync
	}
	if !h.IsPublished {
		return // unpublished hackathons are always Draft
	}
	computed := computeStatusFromPhases(ctx, h)
	if computed != h.StatusCache {
		if err := SyncStatusCache(ctx, h.ID, 0); err != nil {
			log.Error("EnsureStatusCacheFresh: %v", err)
		}
		h.StatusCache = computed // update in-memory for this request
	}
}
```

- [ ] **Step 3: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./services/hackforger/`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/phase_sync.go models/hackforger/hackathon.go
git commit -m "feat(phase): add SyncStatusCache with lifecycle hooks and lazy sync"
```

---

### Task 4: UpdatePhaseTime Active Phase Guard

**Files:**
- Modify: `services/hackforger/phase.go`

- [ ] **Step 1: Add ErrActivePhaseStartLocked error type**

In `services/hackforger/phase.go`, add after the existing error types:

```go
// ErrActivePhaseStartLocked is returned when trying to change StartTime of an active phase.
type ErrActivePhaseStartLocked struct {
	PhaseID int64
}

func (e ErrActivePhaseStartLocked) Error() string {
	return fmt.Sprintf("cannot change start time of active phase [id: %d]", e.PhaseID)
}

func IsErrActivePhaseStartLocked(err error) bool {
	_, ok := err.(ErrActivePhaseStartLocked)
	return ok
}
```

- [ ] **Step 2: Add missing IsErr* checker functions for existing error types**

The existing `ErrPhaseOverlap`, `ErrPhaseNotFuture`, `ErrPhaseUniqueViolation` structs lack `IsErr*` checkers. Add them:

```go
func IsErrPhaseOverlap(err error) bool         { _, ok := err.(ErrPhaseOverlap); return ok }
func IsErrPhaseNotFuture(err error) bool        { _, ok := err.(ErrPhaseNotFuture); return ok }
func IsErrPhaseUniqueViolation(err error) bool  { _, ok := err.(ErrPhaseUniqueViolation); return ok }
```

- [ ] **Step 3: Add guard to UpdatePhaseTime**

In `services/hackforger/phase.go`, in `UpdatePhaseTime()`, add after the `IsLocked()` check:

```go
if phase.IsActive() && startTime != phase.StartTime {
	return ErrActivePhaseStartLocked{PhaseID: phaseID}
}
```

- [ ] **Step 3: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./services/hackforger/`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/phase.go
git commit -m "feat(phase): add active phase startTime guard in UpdatePhaseTime"
```

---

### Task 5: Phase CRUD Web Route Endpoints

**Files:**
- Create: `routers/web/hackforger/phase.go`
- Modify: `routers/web/web.go` (add routes)

- [ ] **Step 1: Create phase.go route handlers**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// ManagePhases returns all phases + phase type catalog as JSON.
func ManagePhases(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}

	phases, err := hackforger_service.GetPhases(ctx, "hackathon", h.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	phaseTypes, err := hackforger_model.GetPhaseTypesByActivityKind(ctx, "hackathon")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Resolve i18n keys to translated strings for Vue component
	type ptJSON struct {
		ID           int64  `json:"id"`
		Key          string `json:"key"`
		DisplayName  string `json:"display_name"`
		IsUnique     bool   `json:"is_unique"`
		DefaultOrder int    `json:"default_order"`
	}
	ptList := make([]ptJSON, len(phaseTypes))
	for i, pt := range phaseTypes {
		ptList[i] = ptJSON{
			ID:           pt.ID,
			Key:          pt.Key,
			DisplayName:  ctx.Locale.TrString(pt.DisplayNameI18n),
			IsUnique:     pt.IsUnique,
			DefaultOrder: pt.DefaultOrder,
		}
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"phases":      phases,
		"phase_types": ptList,
		"status_cache": h.StatusCache,
	})
}

// ManagePhasesAdd adds a new phase.
func ManagePhasesAdd(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}

	var req struct {
		PhaseTypeID int64 `json:"phase_type_id"`
		StartTime   int64 `json:"start_time"`
		EndTime     int64 `json:"end_time"`
		SortOrder   int   `json:"sort_order"`
	}
	if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	_, err := hackforger_service.AddPhase(ctx, "hackathon", h.ID, req.PhaseTypeID, req.SortOrder, req.StartTime, req.EndTime)
	if err != nil {
		handlePhaseError(ctx, err)
		return
	}

	if syncErr := hackforger_service.SyncStatusCache(ctx, h.ID, ctx.Doer.ID); syncErr != nil {
		log.Error("SyncStatusCache: %v", syncErr)
	}

	respondWithPhases(ctx, h.ID)
}

// ManagePhasesUpdate updates a phase's time window.
func ManagePhasesUpdate(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	phaseID := ctx.ParamsInt64("phase_id")

	var req struct {
		StartTime int64 `json:"start_time"`
		EndTime   int64 `json:"end_time"`
	}
	if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := hackforger_service.UpdatePhaseTime(ctx, phaseID, req.StartTime, req.EndTime); err != nil {
		handlePhaseError(ctx, err)
		return
	}

	if syncErr := hackforger_service.SyncStatusCache(ctx, h.ID, ctx.Doer.ID); syncErr != nil {
		log.Error("SyncStatusCache: %v", syncErr)
	}

	respondWithPhases(ctx, h.ID)
}

// ManagePhasesDelete deletes a future phase.
func ManagePhasesDelete(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	phaseID := ctx.ParamsInt64("phase_id")

	if err := hackforger_service.RemovePhase(ctx, phaseID); err != nil {
		handlePhaseError(ctx, err)
		return
	}

	if syncErr := hackforger_service.SyncStatusCache(ctx, h.ID, ctx.Doer.ID); syncErr != nil {
		log.Error("SyncStatusCache: %v", syncErr)
	}

	respondWithPhases(ctx, h.ID)
}

// respondWithPhases returns the updated phases list + status_cache.
func respondWithPhases(ctx *context.Context, hackathonID int64) {
	phases, _ := hackforger_service.GetPhases(ctx, "hackathon", hackathonID)
	h, _ := hackforger_model.GetHackathonByID(ctx, hackathonID)
	ctx.JSON(http.StatusOK, map[string]any{
		"phases":       phases,
		"status_cache": h.StatusCache,
	})
}

// handlePhaseError maps typed phase errors to JSON responses.
func handlePhaseError(ctx *context.Context, err error) {
	status := http.StatusInternalServerError
	msg := "internal error"

	switch {
	case hackforger_service.IsErrPhaseOverlap(err):
		status = http.StatusConflict
		msg = ctx.Locale.TrString("hackforger.phase.error.overlap")
	case hackforger_service.IsErrPhaseLocked(err):
		status = http.StatusForbidden
		msg = ctx.Locale.TrString("hackforger.phase.error.locked")
	case hackforger_service.IsErrPhaseNotFuture(err):
		status = http.StatusForbidden
		msg = ctx.Locale.TrString("hackforger.phase.error.not_future")
	case hackforger_service.IsErrPhaseUniqueViolation(err):
		status = http.StatusConflict
		msg = ctx.Locale.TrString("hackforger.phase.error.unique_violation")
	case hackforger_service.IsErrActivePhaseStartLocked(err):
		status = http.StatusForbidden
		msg = ctx.Locale.TrString("hackforger.phase.error.active_start_locked")
	default:
		log.Error("Phase operation error: %v", err)
	}

	ctx.JSON(status, map[string]string{"error": msg})
}
```

- [ ] **Step 2: Register routes in web.go**

In `routers/web/web.go`, inside the hackathon manage route group, add:

```go
// Phase Timeline CRUD (JSON endpoints for Vue component)
m.Get("/phases", hackforger_web.ManagePhases)
m.Post("/phases", hackforger_web.ManagePhasesAdd)
m.Put("/phases/{phase_id}", hackforger_web.ManagePhasesUpdate)
m.Delete("/phases/{phase_id}", hackforger_web.ManagePhasesDelete)
```

- [ ] **Step 3: Add i18n keys**

In both `locale_en-US.ini` and `locale_zh-CN.ini`, add under `[hackforger]`:

```ini
; EN
phase.error.overlap = Phase time conflicts with an existing phase
phase.error.locked = This phase has ended and cannot be modified
phase.error.not_future = Only future phases can be deleted
phase.error.unique_violation = A phase of this type already exists
phase.error.active_start_locked = Cannot change the start time of an active phase

; ZH
phase.error.overlap = 阶段时间与已有阶段冲突
phase.error.locked = 该阶段已结束，无法修改
phase.error.not_future = 只能删除未来的阶段
phase.error.unique_violation = 该类型的阶段已存在
phase.error.active_start_locked = 无法修改进行中阶段的开始时间
```

- [ ] **Step 4: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add routers/web/hackforger/phase.go routers/web/web.go options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "feat(phase): add Phase CRUD web route endpoints for Vue component"
```

---

### Task 6: PhaseTimeline Vue Component

**Files:**
- Create: `web_src/js/components/hackforger/PhaseTimeline.vue`
- Modify: `web_src/js/features/hackforger/init.js` (mount component)

- [ ] **Step 1: Create PhaseTimeline.vue**

```vue
<script>
import {GET, POST, PUT, DELETE} from '../../modules/fetch.js';

export default {
  props: {
    hackathonSlug: {type: String, required: true},
    initialPhases: {type: Array, default: () => []},
    phaseTypes: {type: Array, default: () => []},
    isPublished: {type: Boolean, default: false},
  },
  data() {
    return {
      phases: [...this.initialPhases],
      statusCache: 0,
      loading: false,
      error: '',
      newPhase: {phaseTypeId: '', startTime: '', endTime: ''},
    };
  },
  computed: {
    sortedPhases() {
      return [...this.phases].sort((a, b) => a.sort_order - b.sort_order || a.start_time - b.start_time);
    },
    availableTypes() {
      return this.phaseTypes.filter((pt) => {
        if (!pt.is_unique) return true;
        return !this.phases.some((p) => p.phase_type_id === pt.id);
      });
    },
  },
  methods: {
    phaseState(phase) {
      const now = Math.floor(Date.now() / 1000);
      if (phase.end_time < now) return 'locked';
      if (phase.start_time <= now && phase.end_time > now) return 'active';
      return 'future';
    },
    phaseTypeName(phase) {
      const pt = this.phaseTypes.find((t) => t.id === phase.phase_type_id);
      return pt ? pt.display_name : `Phase #${phase.id}`;
    },
    toDatetimeLocal(unix) {
      if (!unix) return '';
      return new Date(unix * 1000).toISOString().slice(0, 16);
    },
    fromDatetimeLocal(str) {
      if (!str) return 0;
      return Math.floor(new Date(str).getTime() / 1000);
    },
    hasOverlap(startTime, endTime, excludeId) {
      return this.phases.some((p) => {
        if (p.id === excludeId) return false;
        return startTime < p.end_time && endTime > p.start_time;
      });
    },
    async addPhase() {
      const startTime = this.fromDatetimeLocal(this.newPhase.startTime);
      const endTime = this.fromDatetimeLocal(this.newPhase.endTime);
      if (!this.newPhase.phaseTypeId || !startTime || !endTime) {
        this.error = 'Please fill all fields';
        return;
      }
      if (endTime <= startTime) {
        this.error = 'End time must be after start time';
        return;
      }
      if (this.hasOverlap(startTime, endTime, 0)) {
        this.error = 'Time conflicts with existing phase';
        return;
      }
      this.loading = true;
      this.error = '';
      try {
        const resp = await POST(`/hackathon/${this.hackathonSlug}/manage/phases`, {
          data: {
            phase_type_id: Number(this.newPhase.phaseTypeId),
            start_time: startTime,
            end_time: endTime,
            sort_order: this.phases.length + 1,
          },
        });
        if (!resp.ok) {
          const data = await resp.json();
          this.error = data.error || `HTTP ${resp.status}`;
          return;
        }
        const data = await resp.json();
        this.phases = data.phases || [];
        this.statusCache = data.status_cache;
        this.newPhase = {phaseTypeId: '', startTime: '', endTime: ''};
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },
    async updatePhase(phase, field, event) {
      const val = this.fromDatetimeLocal(event.target.value);
      if (!val) return;
      const startTime = field === 'start' ? val : phase.start_time;
      const endTime = field === 'end' ? val : phase.end_time;
      if (endTime <= startTime) {
        this.error = 'End time must be after start time';
        return;
      }
      this.loading = true;
      this.error = '';
      try {
        const resp = await PUT(`/hackathon/${this.hackathonSlug}/manage/phases/${phase.id}`, {
          data: {start_time: startTime, end_time: endTime},
        });
        if (!resp.ok) {
          const data = await resp.json();
          this.error = data.error || `HTTP ${resp.status}`;
          return;
        }
        const data = await resp.json();
        this.phases = data.phases || [];
        this.statusCache = data.status_cache;
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },
    async deletePhase(phase) {
      this.loading = true;
      this.error = '';
      try {
        const resp = await DELETE(`/hackathon/${this.hackathonSlug}/manage/phases/${phase.id}`);
        if (!resp.ok) {
          const data = await resp.json();
          this.error = data.error || `HTTP ${resp.status}`;
          return;
        }
        const data = await resp.json();
        this.phases = data.phases || [];
        this.statusCache = data.status_cache;
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },
  },
};
</script>

<template>
  <div>
    <div v-if="error" class="ui small negative message">{{ error }}</div>

    <!-- Phase list -->
    <div v-for="phase in sortedPhases" :key="phase.id"
         class="tw-flex tw-items-center tw-gap-3 tw-py-2 tw-border-b"
         :class="{'tw-opacity-50': phaseState(phase) === 'locked'}">
      <span class="hf-badge"
            :class="{'hf-badge-neutral': phaseState(phase) === 'locked',
                     'hf-badge-green': phaseState(phase) === 'active',
                     'hf-badge-blue': phaseState(phase) === 'future'}">
        {{ phaseState(phase) }}
      </span>
      <strong class="tw-w-32">{{ phaseTypeName(phase) }}</strong>

      <!-- Start time -->
      <input type="datetime-local"
             :value="toDatetimeLocal(phase.start_time)"
             :disabled="phaseState(phase) !== 'future'"
             @change="updatePhase(phase, 'start', $event)"
             class="tw-text-sm">

      <span class="tw-text-gray">→</span>

      <!-- End time -->
      <input type="datetime-local"
             :value="toDatetimeLocal(phase.end_time)"
             :disabled="phaseState(phase) === 'locked'"
             @change="updatePhase(phase, 'end', $event)"
             class="tw-text-sm">

      <!-- Delete button (future only) -->
      <button v-if="phaseState(phase) === 'future'"
              class="hf-btn hf-btn-danger"
              :disabled="loading"
              @click="deletePhase(phase)">
        &times;
      </button>
    </div>

    <!-- Add phase form -->
    <div class="tw-flex tw-items-center tw-gap-2 tw-mt-3">
      <select v-model="newPhase.phaseTypeId" class="tw-text-sm">
        <option value="" disabled>Select phase type...</option>
        <option v-for="pt in availableTypes" :key="pt.id" :value="pt.id">
          {{ pt.display_name_i18n }}
        </option>
      </select>
      <input type="datetime-local" v-model="newPhase.startTime" class="tw-text-sm" placeholder="Start">
      <input type="datetime-local" v-model="newPhase.endTime" class="tw-text-sm" placeholder="End">
      <button class="hf-btn hf-btn-primary" :disabled="loading" @click="addPhase">
        + Add Phase
      </button>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Mount in init.js**

In `web_src/js/features/hackforger/init.js`, add after the JudgeScoreCard block:

```javascript
// PhaseTimeline
const phaseEl = document.getElementById('hackforger-phase-timeline');
if (phaseEl) {
  (async () => {
    const {default: PhaseTimeline} = await import(
      /* webpackChunkName: "hackforger-phase" */
      '../../components/hackforger/PhaseTimeline.vue'
    );
    const {createApp} = await import('vue');
    createApp(PhaseTimeline, {
      hackathonSlug: phaseEl.dataset.hackathonSlug,
      initialPhases: JSON.parse(phaseEl.dataset.phases || '[]'),
      phaseTypes: JSON.parse(phaseEl.dataset.phaseTypes || '[]'),
      isPublished: phaseEl.dataset.isPublished === 'true',
    }).mount(phaseEl);
  })();
}
```

- [ ] **Step 3: Build frontend**

Run: `make frontend`
Expected: webpack compiled successfully

- [ ] **Step 4: Commit**

```bash
git add web_src/js/components/hackforger/PhaseTimeline.vue web_src/js/features/hackforger/init.js
git commit -m "feat(phase): add PhaseTimeline Vue component for manage page"
```

---

### Task 7: Action Gating Integration

**Files:**
- Modify: `routers/web/hackforger/hackathon.go`
- Modify: `routers/api/v1/hackforger/hackathon_registration.go`
- Modify: `routers/api/v1/hackforger/hackathon_submission.go`
- Modify: `services/hackforger/hackathon_judge.go`

- [ ] **Step 1: Replace status checks with AllowsAction in web router**

In `routers/web/hackforger/hackathon.go`:

Line 315 (register): Replace `h.StatusCache != hackforger_model.HackathonStatusOpen` with:
```go
if allowed, err := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register"); err != nil {
	ctx.ServerError("AllowsAction", err)
	return
} else if !allowed {
```

Line 396 (submit form GET): Replace `h.StatusCache != hackforger_model.HackathonStatusHacking` with:
```go
if allowed, err := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "submit_work"); err != nil {
	ctx.ServerError("AllowsAction", err)
	return
} else if !allowed {
```

Line 429 (submit POST): Same replacement.

Line 935 (judge page): Replace `h.StatusCache != hackforger_model.HackathonStatusJudging` with:
```go
if allowed, err := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "score"); err != nil {
	ctx.ServerError("AllowsAction", err)
	return
} else if !allowed {
```

- [ ] **Step 2: Replace status checks in API routers**

In `routers/api/v1/hackforger/hackathon_registration.go` line 67:
```go
if allowed, err := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register"); err != nil {
	ctx.ServerError("AllowsAction", err)
	return
} else if !allowed {
```

In `routers/api/v1/hackforger/hackathon_submission.go` line 69:
```go
if allowed, err := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "submit_work"); err != nil {
	ctx.ServerError("AllowsAction", err)
	return
} else if !allowed {
```

- [ ] **Step 3: Replace status check in judge service**

In `services/hackforger/hackathon_judge.go` line 36:
```go
if allowed, _ := AllowsAction(ctx, "hackathon", sub.HackathonID, "score"); !allowed {
```

- [ ] **Step 4: Add EnsureStatusCacheFresh calls to route handlers**

In `routers/web/hackforger/hackathon.go`, in `ViewHackathon()`, `ManageHackathon()`, and `JudgePage()`, add after `loadHackathon`:
```go
hackforger_service.EnsureStatusCacheFresh(ctx, h)
```

- [ ] **Step 5: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add routers/web/hackforger/hackathon.go routers/api/v1/hackforger/hackathon_registration.go routers/api/v1/hackforger/hackathon_submission.go services/hackforger/hackathon_judge.go
git commit -m "feat(phase): replace hardcoded status checks with AllowsAction gating"
```

---

### Task 8: Manage Page — Replace Phase Buttons with Timeline

**Files:**
- Modify: `templates/hackforger/hackathon/manage.tmpl`
- Modify: `routers/web/hackforger/hackathon.go` (pass phase data to template)

- [ ] **Step 1: Pass phase data to manage template**

In `routers/web/hackforger/hackathon.go`, in `ManageHackathon()`, add after existing data loading:

```go
// Phase Timeline data
phases, _ := hackforger_service.GetPhases(ctx, "hackathon", h.ID)
phasesJSON, _ := json.Marshal(phases)
phaseTypes, _ := hackforger_model.GetPhaseTypesByActivityKind(ctx, "hackathon")
phaseTypesJSON, _ := json.Marshal(phaseTypes)
ctx.Data["PhasesJSON"] = string(phasesJSON)
ctx.Data["PhaseTypesJSON"] = string(phaseTypesJSON)
```

- [ ] **Step 2: Replace Phase Control section in manage.tmpl**

Replace the Phase Control card (lines 117-133 in manage.tmpl) with:

```html
{{/* 1. Phase Timeline */}}
<div class="hf-card">
    <div class="hf-card-head">
        <div class="hf-card-head-left">{{ctx.Locale.Tr "hackforger.hackathon.manage.phase"}}</div>
        {{template "hackforger/helpers/status_badge" dict "Type" "hackathon" "Status" .Hackathon.StatusCache "Label" .StatusLabel}}
    </div>
    <div class="hf-card-body">
        <div id="hackforger-phase-timeline"
            data-hackathon-slug="{{.Hackathon.Slug}}"
            data-phases="{{.PhasesJSON}}"
            data-phase-types="{{.PhaseTypesJSON}}"
            data-is-published="{{if .Hackathon.IsPublished}}true{{else}}false{{end}}">
        </div>
        <noscript>
            <div class="ui warning message">JavaScript is required for the phase timeline.</div>
        </noscript>

        <div class="tw-flex tw-gap-2 tw-mt-3">
            {{if not .Hackathon.IsPublished}}
            <form method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/publish" class="tw-inline">
                <button class="hf-btn hf-btn-primary">{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.publish"}}</button>
            </form>
            {{end}}
            {{if and (ne .Hackathon.StatusCache 4) (ne .Hackathon.StatusCache 5)}}
            <form method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/cancel" class="tw-inline" onsubmit="return confirm('{{ctx.Locale.Tr "hackforger.confirm.cancel_activity"}}');">
                <button class="hf-btn hf-btn-danger">{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.cancel"}}</button>
            </form>
            {{end}}
            {{if eq .Hackathon.StatusCache 3}}
            <a class="hf-btn hf-btn-primary" href="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/finalize-preview">
                {{ctx.Locale.Tr "hackforger.hackathon.manage.finalize_preview"}}
            </a>
            {{end}}
        </div>
    </div>
</div>
```

- [ ] **Step 3: Build frontend + backend**

Run: `make frontend && TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add templates/hackforger/hackathon/manage.tmpl routers/web/hackforger/hackathon.go
git commit -m "feat(phase): replace manual phase buttons with PhaseTimeline component on manage page"
```

---

### Task 9: View Page — Dynamic Phase Timeline

**Files:**
- Modify: `templates/hackforger/hackathon/view.tmpl`
- Modify: `routers/web/hackforger/hackathon.go` (pass phases to view)

- [ ] **Step 1: Pass phase data to view template**

In `ViewHackathon()`, add:

```go
phases, _ := hackforger_service.GetPhases(ctx, "hackathon", h.ID)
ctx.Data["Phases"] = phases

// Determine if registration is currently allowed
canRegister, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register")
ctx.Data["CanRegister"] = canRegister

canSubmit, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "submit_work")
ctx.Data["CanSubmit"] = canSubmit
```

- [ ] **Step 2: Replace hardcoded timeline in view.tmpl**

Replace lines 26-50 (the hardcoded 3-phase timeline) with:

```html
{{/* Dynamic Phase Timeline */}}
{{if .Phases}}
<div class="hf-timeline">
    {{range .Phases}}
    <div class="hf-timeline-step {{if .IsLocked}}hf-step-done{{else if .IsActive}}hf-step-active{{else}}hf-step-upcoming{{end}}">
        <div class="hf-step-name">{{if .PhaseType}}{{ctx.Locale.Tr .PhaseType.DisplayNameI18n}}{{end}}</div>
        <div class="hf-step-date">
            {{DateUtils.AbsoluteShort .StartTime}} — {{DateUtils.AbsoluteShort .EndTime}}
        </div>
        <div class="hf-step-status">
            {{if .IsLocked}}{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.step.done"}}
            {{else if .IsActive}}{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.step.active"}}
            {{else}}{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.step.upcoming"}}
            {{end}}
        </div>
    </div>
    {{end}}
</div>
{{end}}
```

- [ ] **Step 3: Update Registration CTA**

Replace line 90 (`{{if and (eq .Hackathon.Status 1) .SignedUserID (not .IsRegistered)}}`) with:

```html
{{if and .CanRegister .SignedUserID (not .IsRegistered)}}
```

- [ ] **Step 4: Update Submit CTA**

Replace line 137 (`{{if and (eq .Hackathon.Status 2) .IsRegistered}}`) with:

```html
{{if and .CanSubmit .IsRegistered}}
```

- [ ] **Step 5: Build backend**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add templates/hackforger/hackathon/view.tmpl routers/web/hackforger/hackathon.go
git commit -m "feat(phase): dynamic phase timeline on view page, action-gated CTAs"
```

---

### Task 10: Publish Flow Changes

**Files:**
- Modify: `services/hackforger/hackathon.go` (rewrite `PublishHackathon`)

- [ ] **Step 1: Rewrite PublishHackathon**

First, add typed errors (near the existing `ErrNoTracks`):

```go
type ErrAlreadyPublished struct{ HackathonID int64 }
func (e ErrAlreadyPublished) Error() string { return fmt.Sprintf("hackathon already published [id: %d]", e.HackathonID) }
func IsErrAlreadyPublished(err error) bool { _, ok := err.(ErrAlreadyPublished); return ok }

type ErrNoRegistrationPhase struct{ HackathonID int64 }
func (e ErrNoRegistrationPhase) Error() string { return fmt.Sprintf("no registration phase [id: %d]", e.HackathonID) }
func IsErrNoRegistrationPhase(err error) bool { _, ok := err.(ErrNoRegistrationPhase); return ok }

type ErrNoDevelopmentPhase struct{ HackathonID int64 }
func (e ErrNoDevelopmentPhase) Error() string { return fmt.Sprintf("no development phase [id: %d]", e.HackathonID) }
func IsErrNoDevelopmentPhase(err error) bool { _, ok := err.(ErrNoDevelopmentPhase); return ok }
```

Then replace the existing `PublishHackathon` function:

```go
// PublishHackathon sets IsPublished=true after validating phases and tracks.
// StatusCache will be updated lazily when the first phase's StartTime arrives.
func PublishHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.IsPublished {
		return ErrAlreadyPublished{HackathonID: h.ID}
	}

	// Validate tracks
	trackCount, err := hackforger_model.CountTracksByHackathon(ctx, h.ID)
	if err != nil {
		return err
	}
	if trackCount == 0 {
		return ErrNoTracks{HackathonID: h.ID}
	}

	// Validate phases
	phases, err := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
	if err != nil {
		return err
	}
	hasRegistration := false
	hasDevelopment := false
	for _, p := range phases {
		if p.PhaseType == nil {
			continue
		}
		switch p.PhaseType.Key {
		case "registration":
			hasRegistration = true
		case "development":
			hasDevelopment = true
		}
	}
	if !hasRegistration {
		return ErrNoRegistrationPhase{HackathonID: h.ID}
	}
	if !hasDevelopment {
		return ErrNoDevelopmentPhase{HackathonID: h.ID}
	}

	// Set published
	if err := hackforger_model.SetIsPublished(ctx, h.ID, true); err != nil {
		return err
	}
	h.IsPublished = true

	// Sync status (may transition to Open if first phase already started)
	if err := SyncStatusCache(ctx, h.ID, doerID); err != nil {
		return err
	}

	return nil
}
```

- [ ] **Step 2: Update CancelHackathon to delete future phases**

In `CancelHackathon`, add after setting status to Cancelled:

```go
// Delete all future phases
phases, _ := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
for _, p := range phases {
	if p.IsFuture() {
		_ = hackforger_model.DeletePhase(ctx, p.ID)
	}
}
```

- [ ] **Step 3: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/hackathon.go
git commit -m "feat(phase): rewrite PublishHackathon for phase-driven flow, cancel deletes future phases"
```

---

### Task 11: Cleanup — Remove Old Manual Handlers

**Files:**
- Modify: `services/hackforger/hackathon.go` (remove `StartHacking`, `StartJudging`)
- Modify: `routers/web/hackforger/hackathon.go` (remove `ManagePhasePost` cases for "start" and "judge")
- Modify: `routers/web/web.go` (clean up routes)

- [ ] **Step 1: Remove StartHacking and StartJudging functions**

In `services/hackforger/hackathon.go`, delete the `StartHacking()` and `StartJudging()` functions. Their logic has been moved to lifecycle hooks in `SyncStatusCache`.

- [ ] **Step 2: Update ManagePhasePost**

In `routers/web/hackforger/hackathon.go`, remove the "start" and "judge" cases from `ManagePhasePost`. Keep only "publish" and "cancel":

```go
func ManagePhasePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	action := ctx.Params("action")

	var err error
	switch action {
	case "publish":
		err = hackforger_service.PublishHackathon(ctx, ctx.Doer.ID, h)
	case "cancel":
		err = hackforger_service.CancelHackathon(ctx, ctx.Doer.ID, h)
	default:
		ctx.NotFound("ManagePhasePost", nil)
		return
	}

	// ... error handling unchanged ...
}
```

- [ ] **Step 3: Remove API callers of StartHacking/StartJudging**

In `routers/api/v1/hackforger/hackathon.go`, remove the `StartHackathon` and `StartJudgingHackathon` handler functions.

In `routers/api/v1/api.go`, remove the route registrations for `POST /hackforger/hackathons/{id}/start` and `POST /hackforger/hackathons/{id}/judge`.

These are breaking API changes — manual phase transitions are removed. Document in commit message.

- [ ] **Step 4: Verify compilation and no dead code**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: No errors, no unused function warnings

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/hackathon.go routers/web/hackforger/hackathon.go routers/web/web.go
git commit -m "refactor(phase): remove StartHacking/StartJudging manual handlers, clean up routes"
```

---

## Post-Implementation Checklist

After all tasks are complete:

- [ ] Run full backend compilation: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
- [ ] Run full frontend build: `make frontend`
- [ ] Start server and verify manage page loads with Phase Timeline
- [ ] Verify view page shows dynamic timeline from Phase records
- [ ] Verify publish flow validates phases before enabling
- [ ] Verify action gating (register, submit, judge) works based on active phases
- [ ] Verify StatusCache syncs correctly when phases activate/expire
- [ ] Take E2E screenshots for regression testing
