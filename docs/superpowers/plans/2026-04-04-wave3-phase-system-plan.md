# Wave 3: Phase System — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce the Phase System — HackForger's core architectural addition that brings event lifecycle management to Forgejo.

**Architecture:** Two new tables (phase_type, phase), a PhaseController service, a PhaseScheduler for notifications, a reusable Phase Timeline UI component, admin management pages, and API routes. All activity types (Hackathon, Bounty, Grant) integrate through `AllowsAction()` gating.

**Tech Stack:** Go (XORM, services, time.AfterFunc), Go templates, Vue 3, CSS, REST API

**Spec:** `docs/superpowers/specs/2026-04-04-wave3-phase-system.md`

---

## Concurrency Groups

```
Group A — Foundation (sequential, build bottom-up):
  Task 1: Migration + data model
  Task 2: Action vocabulary + validation
  Task 3: PhaseController service
  Task 4: PhaseScheduler (notification timers)

Group B — API + Web (parallel after Group A):
  Task 5: Phase API routes
  Task 6: Phase Timeline UI component (template + Vue)
  Task 7: Admin Phase Type management

Group C — Integration (parallel after Group B):
  Task 8: Integrate Phase gating into Hackathon/Bounty/Grant services
```

---

### Task 1: Migration + Data Model

**Files:**
- Create: `models/hackforger/phase_type.go`
- Create: `models/hackforger/phase.go`
- Create: `models/forgejo_migrations/v14b_hackforger-phase-tables.go`

- [ ] **Step 1: Define PhaseType model**

```go
// models/hackforger/phase_type.go
package hackforger

import (
	"encoding/json"

	"forgejo.org/modules/timeutil"
)

type PhaseType struct {
	ID              int64              `xorm:"pk autoincr"`
	ActivityKind    string             `xorm:"VARCHAR(20) NOT NULL INDEX"`
	Key             string             `xorm:"VARCHAR(50) NOT NULL"`
	DisplayNameI18n string             `xorm:"VARCHAR(100) NOT NULL"`
	IsUnique        bool               `xorm:"NOT NULL DEFAULT true"`
	AllowedActions  string             `xorm:"TEXT NOT NULL"` // JSON array
	DefaultOrder    int                `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix     timeutil.TimeStamp `xorm:"created"`
}

func (pt *PhaseType) GetAllowedActions() ([]string, error) {
	var actions []string
	err := json.Unmarshal([]byte(pt.AllowedActions), &actions)
	return actions, err
}

func (pt *PhaseType) HasAction(action string) bool {
	actions, err := pt.GetAllowedActions()
	if err != nil {
		return false
	}
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func init() {
	db.RegisterModel(new(PhaseType))
}
```

- [ ] **Step 2: Define Phase model**

```go
// models/hackforger/phase.go
package hackforger

import (
	"time"

	"forgejo.org/modules/timeutil"
)

type Phase struct {
	ID          int64              `xorm:"pk autoincr"`
	PhaseTypeID int64              `xorm:"NOT NULL INDEX"`
	ActivityKind string            `xorm:"VARCHAR(20) NOT NULL"`
	ActivityID  int64              `xorm:"NOT NULL"`
	SortOrder   int                `xorm:"NOT NULL"`
	StartTime   int64              `xorm:"NOT NULL"`
	EndTime     int64              `xorm:"NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`

	// Loaded via join, not stored
	PhaseType *PhaseType `xorm:"-"`
}

func (p *Phase) IsLocked() bool {
	return p.EndTime < time.Now().Unix()
}

func (p *Phase) IsActive() bool {
	now := time.Now().Unix()
	return p.StartTime <= now && p.EndTime > now
}

func (p *Phase) IsFuture() bool {
	return p.StartTime > time.Now().Unix()
}

func init() {
	db.RegisterModel(new(Phase))
}
```

- [ ] **Step 3: Add model query functions**

```go
// models/hackforger/phase.go (continued)

func GetPhasesByActivity(ctx context.Context, activityKind string, activityID int64) ([]*Phase, error) {
	phases := make([]*Phase, 0)
	err := db.GetEngine(ctx).
		Where("activity_kind = ? AND activity_id = ?", activityKind, activityID).
		OrderBy("sort_order ASC").
		Find(&phases)
	if err != nil {
		return nil, err
	}
	// Load PhaseType for each
	for _, p := range phases {
		pt := &PhaseType{}
		if _, err := db.GetEngine(ctx).ID(p.PhaseTypeID).Get(pt); err != nil {
			return nil, err
		}
		p.PhaseType = pt
	}
	return phases, nil
}

func GetCurrentPhase(ctx context.Context, activityKind string, activityID int64) (*Phase, error) {
	now := time.Now().Unix()
	phase := &Phase{}
	has, err := db.GetEngine(ctx).
		Where("activity_kind = ? AND activity_id = ? AND start_time <= ? AND end_time > ?",
			activityKind, activityID, now, now).
		Get(phase)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil // between phases or no phases defined
	}
	// Load PhaseType
	pt := &PhaseType{}
	if _, err := db.GetEngine(ctx).ID(phase.PhaseTypeID).Get(pt); err != nil {
		return nil, err
	}
	phase.PhaseType = pt
	return phase, nil
}

func GetPhaseTypesByActivityKind(ctx context.Context, activityKind string) ([]*PhaseType, error) {
	types := make([]*PhaseType, 0)
	return types, db.GetEngine(ctx).
		Where("activity_kind = ?", activityKind).
		OrderBy("default_order ASC").
		Find(&types)
}
```

- [ ] **Step 4: Write migration with seed data**

```go
// models/forgejo_migrations/v14b_hackforger-phase-tables.go
package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "Create hackforger_phase_type and hackforger_phase tables with seed data",
		Upgrade: func(x *xorm.Engine) error {
			// Tables auto-created by XORM from registered models.
			// Seed default phase types.
			type PhaseType struct {
				ID              int64  `xorm:"pk autoincr"`
				ActivityKind    string `xorm:"VARCHAR(20) NOT NULL"`
				Key             string `xorm:"VARCHAR(50) NOT NULL"`
				DisplayNameI18n string `xorm:"VARCHAR(100) NOT NULL"`
				IsUnique        bool   `xorm:"NOT NULL DEFAULT true"`
				AllowedActions  string `xorm:"TEXT NOT NULL"`
				DefaultOrder    int    `xorm:"NOT NULL DEFAULT 0"`
				CreatedUnix     int64  `xorm:"created"`
			}

			if err := x.Sync(new(PhaseType)); err != nil {
				return err
			}

			type Phase struct {
				ID           int64  `xorm:"pk autoincr"`
				PhaseTypeID  int64  `xorm:"NOT NULL"`
				ActivityKind string `xorm:"VARCHAR(20) NOT NULL"`
				ActivityID   int64  `xorm:"NOT NULL"`
				SortOrder    int    `xorm:"NOT NULL"`
				StartTime    int64  `xorm:"NOT NULL"`
				EndTime      int64  `xorm:"NOT NULL"`
				CreatedUnix  int64  `xorm:"created"`
				UpdatedUnix  int64  `xorm:"updated"`
			}

			if err := x.Sync(new(Phase)); err != nil {
				return err
			}

			// Create index
			if _, err := x.Exec("CREATE INDEX IF NOT EXISTS IDX_phase_activity ON hackforger_phase(activity_kind, activity_id, sort_order)"); err != nil {
				return err
			}

			// Seed hackathon phase types
			seeds := []PhaseType{
				{ActivityKind: "hackathon", Key: "registration", DisplayNameI18n: "hackforger.phase.hackathon.registration", IsUnique: true, AllowedActions: `["register","form_team"]`, DefaultOrder: 1},
				{ActivityKind: "hackathon", Key: "development", DisplayNameI18n: "hackforger.phase.hackathon.development", IsUnique: false, AllowedActions: `["submit_work"]`, DefaultOrder: 2},
				{ActivityKind: "hackathon", Key: "judging", DisplayNameI18n: "hackforger.phase.hackathon.judging", IsUnique: true, AllowedActions: `["score"]`, DefaultOrder: 3},
				{ActivityKind: "hackathon", Key: "results", DisplayNameI18n: "hackforger.phase.hackathon.results", IsUnique: true, AllowedActions: `["publish_results"]`, DefaultOrder: 4},
				// Bounty
				{ActivityKind: "bounty", Key: "open", DisplayNameI18n: "hackforger.phase.bounty.open", IsUnique: true, AllowedActions: `["apply","claim"]`, DefaultOrder: 1},
				{ActivityKind: "bounty", Key: "in_progress", DisplayNameI18n: "hackforger.phase.bounty.in_progress", IsUnique: true, AllowedActions: `["submit_pr"]`, DefaultOrder: 2},
				{ActivityKind: "bounty", Key: "review", DisplayNameI18n: "hackforger.phase.bounty.review", IsUnique: true, AllowedActions: `["review"]`, DefaultOrder: 3},
				{ActivityKind: "bounty", Key: "closed", DisplayNameI18n: "hackforger.phase.bounty.closed", IsUnique: true, AllowedActions: `["settle"]`, DefaultOrder: 4},
				// Grant
				{ActivityKind: "grant", Key: "application", DisplayNameI18n: "hackforger.phase.grant.application", IsUnique: true, AllowedActions: `["apply"]`, DefaultOrder: 1},
				{ActivityKind: "grant", Key: "review", DisplayNameI18n: "hackforger.phase.grant.review", IsUnique: true, AllowedActions: `["review","approve"]`, DefaultOrder: 2},
				{ActivityKind: "grant", Key: "disbursement", DisplayNameI18n: "hackforger.phase.grant.disbursement", IsUnique: true, AllowedActions: `["disburse"]`, DefaultOrder: 3},
			}

			for i := range seeds {
				if _, err := x.Insert(&seeds[i]); err != nil {
					return err
				}
			}

			return nil
		},
	})
}
```

- [ ] **Step 5: Write model tests**

```go
// models/hackforger/phase_test.go
func TestPhaseIsLocked(t *testing.T) {
	past := &Phase{EndTime: time.Now().Unix() - 3600}
	assert.True(t, past.IsLocked())

	future := &Phase{StartTime: time.Now().Unix() + 3600, EndTime: time.Now().Unix() + 7200}
	assert.False(t, future.IsLocked())
	assert.True(t, future.IsFuture())
}

func TestPhaseTypeHasAction(t *testing.T) {
	pt := &PhaseType{AllowedActions: `["register","form_team"]`}
	assert.True(t, pt.HasAction("register"))
	assert.True(t, pt.HasAction("form_team"))
	assert.False(t, pt.HasAction("submit_work"))
}
```

Run: `go test ./models/hackforger/... -run TestPhase -v`

- [ ] **Step 6: Commit and create PR**

```bash
git checkout -b feat/hf-phase-data-model
git add models/hackforger/ models/forgejo_migrations/
git commit -m "feat(hackforger): Phase System data model and migration"
gh pr create --title "feat(hackforger): Phase data model + migration" --body "Create hackforger_phase_type and hackforger_phase tables with seed data for Hackathon, Bounty, and Grant phase types. Resolves HF-003, HF-005, HF-006."
```

---

### Task 2: Action Vocabulary + Validation

**Files:**
- Create: `modules/hackforger/phase_actions.go`

- [ ] **Step 1: Define action vocabulary registry**

```go
// modules/hackforger/phase_actions.go
package hackforger

import "fmt"

// Action vocabularies per activity kind
var actionVocabulary = map[string]map[string]string{
	"hackathon": {
		"register":        "hackforger.action.register",
		"form_team":       "hackforger.action.form_team",
		"submit_work":     "hackforger.action.submit_work",
		"score":           "hackforger.action.score",
		"publish_results": "hackforger.action.publish_results",
	},
	"bounty": {
		"apply":     "hackforger.action.apply",
		"claim":     "hackforger.action.claim",
		"submit_pr": "hackforger.action.submit_pr",
		"review":    "hackforger.action.review",
		"settle":    "hackforger.action.settle",
	},
	"grant": {
		"apply":    "hackforger.action.apply",
		"review":   "hackforger.action.review",
		"approve":  "hackforger.action.approve",
		"disburse": "hackforger.action.disburse",
	},
}

// GetActionsForKind returns all valid action strings for an activity kind.
func GetActionsForKind(activityKind string) (map[string]string, error) {
	actions, ok := actionVocabulary[activityKind]
	if !ok {
		return nil, fmt.Errorf("unknown activity kind: %s", activityKind)
	}
	return actions, nil
}

// ValidateActions checks that all actions in the list are valid for the given activity kind.
func ValidateActions(activityKind string, actions []string) error {
	vocab, err := GetActionsForKind(activityKind)
	if err != nil {
		return err
	}
	for _, action := range actions {
		if _, ok := vocab[action]; !ok {
			return fmt.Errorf("invalid action %q for activity kind %q", action, activityKind)
		}
	}
	return nil
}
```

- [ ] **Step 2: Add i18n keys for action descriptions**

```ini
; en-US
hackforger.action.register = Register
hackforger.action.form_team = Form Team
hackforger.action.submit_work = Submit Work
hackforger.action.score = Score
hackforger.action.publish_results = Publish Results
hackforger.action.apply = Apply
hackforger.action.claim = Claim
hackforger.action.submit_pr = Submit PR
hackforger.action.review = Review
hackforger.action.settle = Settle
hackforger.action.approve = Approve
hackforger.action.disburse = Disburse

; zh-CN
hackforger.action.register = 报名
hackforger.action.form_team = 组队
hackforger.action.submit_work = 提交作品
hackforger.action.score = 打分
hackforger.action.publish_results = 发布结果
hackforger.action.apply = 申请
hackforger.action.claim = 认领
hackforger.action.submit_pr = 提交 PR
hackforger.action.review = 评审
hackforger.action.settle = 结算
hackforger.action.approve = 批准
hackforger.action.disburse = 拨款
```

- [ ] **Step 3: Write tests**

```go
func TestValidateActions(t *testing.T) {
	assert.NoError(t, ValidateActions("hackathon", []string{"register", "form_team"}))
	assert.Error(t, ValidateActions("hackathon", []string{"register", "invalid_action"}))
	assert.Error(t, ValidateActions("unknown_kind", []string{"register"}))
}
```

Run: `go test ./modules/hackforger/... -run TestValidate -v`

- [ ] **Step 4: Commit and create PR**

```bash
git checkout -b feat/hf-phase-action-vocabulary
git add modules/hackforger/ options/locale/
git commit -m "feat(hackforger): Phase action vocabulary and validation"
gh pr create --title "feat(hackforger): Phase action vocabulary" --body "Define action vocabularies per activity kind with validation. Actions: hackathon (5), bounty (5), grant (4)."
```

---

### Task 3: PhaseController Service

**Files:**
- Create: `services/hackforger/phase.go`
- Create: `services/hackforger/phase_test.go`

- [ ] **Step 1: Implement core functions**

```go
// services/hackforger/phase.go
package hackforger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	models_hackforger "forgejo.org/models/hackforger"
	modules_hackforger "forgejo.org/modules/hackforger"
	"forgejo.org/models/db"
)

var (
	ErrPhaseActionNotAllowed = errors.New("action not allowed in current phase")
	ErrPhaseLocked           = errors.New("phase is locked and cannot be modified")
	ErrPhaseOverlap          = errors.New("phase time overlaps with adjacent phase")
	ErrPhaseUniqueViolation  = errors.New("only one phase of this type is allowed")
	ErrPhaseNotFuture        = errors.New("can only remove future phases")
)

func CurrentPhase(ctx context.Context, activityKind string, activityID int64) (*models_hackforger.Phase, error) {
	return models_hackforger.GetCurrentPhase(ctx, activityKind, activityID)
}

func AllowsAction(ctx context.Context, activityKind string, activityID int64, action string) (bool, error) {
	phase, err := CurrentPhase(ctx, activityKind, activityID)
	if err != nil {
		return false, err
	}
	if phase == nil {
		return false, nil // no active phase = no actions allowed
	}
	return phase.PhaseType.HasAction(action), nil
}

func UpdatePhaseTime(ctx context.Context, phaseID int64, startTime, endTime int64) error {
	phase := &models_hackforger.Phase{}
	has, err := db.GetEngine(ctx).ID(phaseID).Get(phase)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("phase not found: %d", phaseID)
	}

	if phase.IsLocked() {
		return ErrPhaseLocked
	}

	if endTime <= startTime {
		return fmt.Errorf("end_time must be after start_time")
	}

	// Check overlap with adjacent phases
	if err := checkOverlap(ctx, phase.ActivityKind, phase.ActivityID, phaseID, startTime, endTime); err != nil {
		return err
	}

	phase.StartTime = startTime
	phase.EndTime = endTime
	if _, err := db.GetEngine(ctx).ID(phaseID).Cols("start_time", "end_time", "updated_unix").Update(phase); err != nil {
		return err
	}

	// Re-register notification timer
	GetPhaseScheduler().Reschedule(phase)

	return nil
}

func AddPhase(ctx context.Context, activityKind string, activityID int64, phaseTypeID int64, sortOrder int, startTime, endTime int64) (*models_hackforger.Phase, error) {
	if endTime <= startTime {
		return nil, fmt.Errorf("end_time must be after start_time")
	}

	// Check is_unique constraint
	pt := &models_hackforger.PhaseType{}
	if _, err := db.GetEngine(ctx).ID(phaseTypeID).Get(pt); err != nil {
		return nil, err
	}
	if pt.IsUnique {
		exists, err := db.GetEngine(ctx).Where(
			"activity_kind = ? AND activity_id = ? AND phase_type_id = ?",
			activityKind, activityID, phaseTypeID).Exist(&models_hackforger.Phase{})
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrPhaseUniqueViolation
		}
	}

	// Check overlap
	if err := checkOverlap(ctx, activityKind, activityID, 0, startTime, endTime); err != nil {
		return nil, err
	}

	phase := &models_hackforger.Phase{
		PhaseTypeID:  phaseTypeID,
		ActivityKind: activityKind,
		ActivityID:   activityID,
		SortOrder:    sortOrder,
		StartTime:    startTime,
		EndTime:      endTime,
	}
	if _, err := db.GetEngine(ctx).Insert(phase); err != nil {
		return nil, err
	}

	// Register notification timer
	GetPhaseScheduler().Schedule(phase)

	return phase, nil
}

func RemovePhase(ctx context.Context, phaseID int64) error {
	phase := &models_hackforger.Phase{}
	has, err := db.GetEngine(ctx).ID(phaseID).Get(phase)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("phase not found: %d", phaseID)
	}
	if !phase.IsFuture() {
		return ErrPhaseNotFuture
	}

	GetPhaseScheduler().Cancel(phaseID)

	_, err = db.GetEngine(ctx).ID(phaseID).Delete(&models_hackforger.Phase{})
	return err
}

func GetPhases(ctx context.Context, activityKind string, activityID int64) ([]*models_hackforger.Phase, error) {
	return models_hackforger.GetPhasesByActivity(ctx, activityKind, activityID)
}

func checkOverlap(ctx context.Context, activityKind string, activityID int64, excludePhaseID int64, startTime, endTime int64) error {
	builder := db.GetEngine(ctx).
		Where("activity_kind = ? AND activity_id = ?", activityKind, activityID).
		And("((start_time < ? AND end_time > ?) OR (start_time < ? AND end_time > ?) OR (start_time >= ? AND end_time <= ?))",
			endTime, startTime, endTime, startTime, startTime, endTime)
	if excludePhaseID > 0 {
		builder = builder.And("id != ?", excludePhaseID)
	}
	count, err := builder.Count(&models_hackforger.Phase{})
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrPhaseOverlap
	}
	return nil
}
```

- [ ] **Step 2: Write tests**

```go
// services/hackforger/phase_test.go
func TestAllowsAction(t *testing.T) {
	// Setup: create hackathon with "registration" phase that is currently active
	// Test: AllowsAction("hackathon", id, "register") → true
	// Test: AllowsAction("hackathon", id, "submit_work") → false
}

func TestUpdatePhaseTimeLocked(t *testing.T) {
	// Setup: create phase with end_time in the past
	// Test: UpdatePhaseTime → ErrPhaseLocked
}

func TestAddPhaseUniqueViolation(t *testing.T) {
	// Setup: create hackathon with "registration" phase (is_unique=true)
	// Test: AddPhase with same type → ErrPhaseUniqueViolation
}

func TestPhaseOverlapDetection(t *testing.T) {
	// Setup: create phase 10:00-12:00
	// Test: add phase 11:00-13:00 → ErrPhaseOverlap
	// Test: add phase 12:00-14:00 → success (no overlap, contiguous)
}
```

Run: `go test ./services/hackforger/... -run TestPhase -v`

- [ ] **Step 3: Commit and create PR**

```bash
git checkout -b feat/hf-phase-controller
git add services/hackforger/
git commit -m "feat(hackforger): PhaseController service with CRUD and action gating"
gh pr create --title "feat(hackforger): PhaseController service" --body "Core Phase service: CurrentPhase, AllowsAction, AddPhase, UpdatePhaseTime, RemovePhase with overlap detection, uniqueness, and lock enforcement."
```

---

### Task 4: PhaseScheduler (Notification Timers)

**Files:**
- Create: `services/hackforger/phase_scheduler.go`
- Create: `services/hackforger/phase_scheduler_test.go`

- [ ] **Step 1: Implement PhaseScheduler singleton**

```go
// services/hackforger/phase_scheduler.go
package hackforger

import (
	"context"
	"sync"
	"time"

	models_hackforger "forgejo.org/models/hackforger"
	"forgejo.org/models/db"
	"forgejo.org/modules/log"
)

var (
	scheduler     *PhaseScheduler
	schedulerOnce sync.Once
)

type PhaseScheduler struct {
	mu     sync.Mutex
	timers map[int64]*time.Timer
}

func GetPhaseScheduler() *PhaseScheduler {
	schedulerOnce.Do(func() {
		scheduler = &PhaseScheduler{
			timers: make(map[int64]*time.Timer),
		}
	})
	return scheduler
}

func (s *PhaseScheduler) Init(ctx context.Context) error {
	// Register timers for all future phases
	phases := make([]*models_hackforger.Phase, 0)
	now := time.Now().Unix()
	err := db.GetEngine(ctx).Where("start_time > ?", now).Find(&phases)
	if err != nil {
		return err
	}
	for _, p := range phases {
		s.Schedule(p)
	}

	// Retroactively publish missed events
	s.recoverMissedEvents(ctx, now)

	log.Info("PhaseScheduler initialized: %d timers registered", len(phases))
	return nil
}

func (s *PhaseScheduler) Schedule(phase *models_hackforger.Phase) {
	s.mu.Lock()
	defer s.mu.Unlock()

	duration := time.Until(time.Unix(phase.StartTime, 0))
	if duration <= 0 {
		return // already started, skip
	}

	s.timers[phase.ID] = time.AfterFunc(duration, func() {
		s.onPhaseStart(phase)
	})
}

func (s *PhaseScheduler) Reschedule(phase *models_hackforger.Phase) {
	s.Cancel(phase.ID)
	s.Schedule(phase)
}

func (s *PhaseScheduler) Cancel(phaseID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if timer, ok := s.timers[phaseID]; ok {
		timer.Stop()
		delete(s.timers, phaseID)
	}
}

func (s *PhaseScheduler) onPhaseStart(phase *models_hackforger.Phase) {
	s.mu.Lock()
	delete(s.timers, phase.ID)
	s.mu.Unlock()

	ctx := context.Background()
	// Load phase type for display
	pt := &models_hackforger.PhaseType{}
	if _, err := db.GetEngine(ctx).ID(phase.PhaseTypeID).Get(pt); err != nil {
		log.Error("PhaseScheduler: failed to load phase type %d: %v", phase.PhaseTypeID, err)
		return
	}

	// Publish Feed event + notify watchers
	PublishHackforgerAction(ctx, &HackforgerActionOptions{
		ActionType:   ActionPhaseStarted,
		ActivityKind: phase.ActivityKind,
		ActivityID:   phase.ActivityID,
		Content:      pt.DisplayNameI18n,
	})
}

func (s *PhaseScheduler) recoverMissedEvents(ctx context.Context, now int64) {
	// Find phases that started but have no corresponding Feed event
	// This handles server restarts where timers were lost
	phases := make([]*models_hackforger.Phase, 0)
	err := db.GetEngine(ctx).
		Where("start_time <= ? AND start_time > ?", now, now-86400). // within last 24h
		Find(&phases)
	if err != nil {
		log.Error("PhaseScheduler: recovery query failed: %v", err)
		return
	}

	for _, p := range phases {
		// Check if Feed event already exists for this phase start
		// If not, publish retroactively
		if !phaseEventExists(ctx, p) {
			s.onPhaseStart(p)
		}
	}
}

func phaseEventExists(ctx context.Context, phase *models_hackforger.Phase) bool {
	// Query action table for phase start event
	// Implementation depends on how PublishHackforgerAction stores events
	// Return true if event found
	return false // placeholder — implement based on action table schema
}
```

- [ ] **Step 2: Register scheduler at server startup**

Find the server initialization code (likely in `cmd/web.go` or `routers/init.go`) and add:

```go
// Initialize Phase Scheduler
if err := hackforger_service.GetPhaseScheduler().Init(ctx); err != nil {
    log.Error("Failed to initialize PhaseScheduler: %v", err)
}
```

- [ ] **Step 3: Write tests**

```go
func TestPhaseSchedulerScheduleAndCancel(t *testing.T) {
	s := &PhaseScheduler{timers: make(map[int64]*time.Timer)}
	phase := &models_hackforger.Phase{ID: 1, StartTime: time.Now().Unix() + 10}
	s.Schedule(phase)
	assert.Len(t, s.timers, 1)
	s.Cancel(1)
	assert.Len(t, s.timers, 0)
}

func TestPhaseSchedulerReschedule(t *testing.T) {
	s := &PhaseScheduler{timers: make(map[int64]*time.Timer)}
	phase := &models_hackforger.Phase{ID: 1, StartTime: time.Now().Unix() + 10}
	s.Schedule(phase)
	phase.StartTime = time.Now().Unix() + 20
	s.Reschedule(phase)
	assert.Len(t, s.timers, 1)
}
```

Run: `go test ./services/hackforger/... -run TestPhaseScheduler -v`

- [ ] **Step 4: Commit and create PR**

```bash
git checkout -b feat/hf-phase-scheduler
git add services/hackforger/ cmd/ routers/
git commit -m "feat(hackforger): PhaseScheduler for phase transition notifications"
gh pr create --title "feat(hackforger): PhaseScheduler notification timers" --body "In-process Go timers for phase transition notifications. Includes startup recovery for missed events."
```

---

### Task 5: Phase API Routes

**Files:**
- Create: `routers/api/v1/hackforger/phase.go`
- Modify: `routers/api/v1/api.go` (route registration injection point)

- [ ] **Step 1: Implement API handlers**

```go
// routers/api/v1/hackforger/phase.go
package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// ListPhases returns all phases for an activity
func ListPhases(ctx *context.APIContext) {
	kind := ctx.PathParam("kind")
	id := ctx.PathParamInt64("id")

	phases, err := hackforger_service.GetPhases(ctx, kind, id)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetPhases", err)
		return
	}
	ctx.JSON(http.StatusOK, phases)
}

// GetCurrentPhase returns the currently active phase
func GetCurrentPhase(ctx *context.APIContext) {
	kind := ctx.PathParam("kind")
	id := ctx.PathParamInt64("id")

	phase, err := hackforger_service.CurrentPhase(ctx, kind, id)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CurrentPhase", err)
		return
	}
	if phase == nil {
		ctx.JSON(http.StatusOK, nil)
		return
	}
	ctx.JSON(http.StatusOK, phase)
}

// AddPhase creates a new phase for an activity
func AddPhase(ctx *context.APIContext) {
	kind := ctx.PathParam("kind")
	id := ctx.PathParamInt64("id")

	form := struct {
		PhaseTypeID int64 `json:"phase_type_id" binding:"Required"`
		SortOrder   int   `json:"sort_order"`
		StartTime   int64 `json:"start_time" binding:"Required"`
		EndTime     int64 `json:"end_time" binding:"Required"`
	}{}

	if err := ctx.Bind(&form); err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "Bind", err)
		return
	}

	phase, err := hackforger_service.AddPhase(ctx, kind, id, form.PhaseTypeID, form.SortOrder, form.StartTime, form.EndTime)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "AddPhase", err)
		return
	}
	ctx.JSON(http.StatusCreated, phase)
}

// UpdatePhaseTime updates the time window of a phase
func UpdatePhaseTime(ctx *context.APIContext) {
	phaseID := ctx.PathParamInt64("phaseID")

	form := struct {
		StartTime int64 `json:"start_time" binding:"Required"`
		EndTime   int64 `json:"end_time" binding:"Required"`
	}{}

	if err := ctx.Bind(&form); err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "Bind", err)
		return
	}

	if err := hackforger_service.UpdatePhaseTime(ctx, phaseID, form.StartTime, form.EndTime); err != nil {
		ctx.Error(http.StatusBadRequest, "UpdatePhaseTime", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// DeletePhase removes a future phase
func DeletePhase(ctx *context.APIContext) {
	phaseID := ctx.PathParamInt64("phaseID")

	if err := hackforger_service.RemovePhase(ctx, phaseID); err != nil {
		ctx.Error(http.StatusBadRequest, "RemovePhase", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// Admin handlers for phase types follow same pattern...
// ListPhaseTypes, CreatePhaseType, UpdatePhaseType, DeletePhaseType
```

- [ ] **Step 2: Register routes**

In `routers/api/v1/api.go` at the HackForger injection point:

```go
// HackForger Phase API
m.Group("/hackforger", func() {
    m.Group("/{kind}/{id}/phases", func() {
        m.Get("", hackforger.ListPhases)
        m.Post("", reqToken(), hackforger.AddPhase)
    })
    m.Get("/{kind}/{id}/current-phase", hackforger.GetCurrentPhase)
    m.Group("/phases/{phaseID}", func() {
        m.Put("", reqToken(), hackforger.UpdatePhaseTime)
        m.Delete("", reqToken(), hackforger.DeletePhase)
    })
    m.Group("/admin/phase-types", func() {
        m.Get("", hackforger.ListPhaseTypes)
        m.Post("", reqSiteAdmin(), hackforger.CreatePhaseType)
        m.Put("/{id}", reqSiteAdmin(), hackforger.UpdatePhaseType)
        m.Delete("/{id}", reqSiteAdmin(), hackforger.DeletePhaseType)
    })
})
```

- [ ] **Step 3: Write API tests and commit**

```bash
git checkout -b feat/hf-phase-api
git add routers/api/v1/hackforger/ routers/api/v1/api.go
git commit -m "feat(hackforger): Phase System REST API routes"
gh pr create --title "feat(hackforger): Phase API routes" --body "REST API for Phase CRUD: list/add/update/delete phases, admin phase type management."
```

---

### Task 6: Phase Timeline UI Component

**Files:**
- Create: `templates/hackforger/components/phase_timeline.tmpl`
- Create: `web_src/js/features/hackforger/PhaseTimeline.vue`
- Create: `web_src/css/features/hackforger-phase.css`

- [ ] **Step 1: Create Go template component**

```html
{{/* templates/hackforger/components/phase_timeline.tmpl */}}
<div class="hackforger-phase-timeline" data-phases="{{.Phases | JsonEncode}}" data-editable="{{.Editable}}">
  <div class="phase-nodes">
    {{range .Phases}}
    <div class="phase-node {{if .IsLocked}}locked{{else if .IsActive}}active{{else}}future{{end}}"
         data-phase-id="{{.ID}}">
      <div class="phase-indicator">
        {{if .IsLocked}}
          {{svg "octicon-lock" 16}}
        {{else if .IsActive}}
          <span class="pulse-dot"></span>
        {{else}}
          {{svg "octicon-clock" 16}}
        {{end}}
      </div>
      <div class="phase-info">
        <span class="phase-name">{{ctx.Locale.Tr .PhaseType.DisplayNameI18n}}</span>
        <span class="phase-time">
          {{DateTime "short" .StartTime}} - {{DateTime "short" .EndTime}}
        </span>
        {{if .IsActive}}
        <div class="phase-actions">
          {{range .PhaseType.GetAllowedActions}}
            <span class="ui label tiny">{{ctx.Locale.Tr (printf "hackforger.action.%s" .)}}</span>
          {{end}}
        </div>
        {{end}}
      </div>
    </div>
    {{if not (eq (add $.Index 1) (len $.Phases))}}
    <div class="phase-connector {{if .IsLocked}}locked{{end}}"></div>
    {{end}}
    {{end}}
  </div>
</div>
```

- [ ] **Step 2: Create CSS**

```css
/* web_src/css/features/hackforger-phase.css */
.hackforger-phase-timeline {
  display: flex;
  overflow-x: auto;
  padding: 1rem 0;
}
.phase-nodes {
  display: flex;
  align-items: center;
  gap: 0;
}
.phase-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 120px;
  padding: 0.5rem;
  border-radius: 8px;
  text-align: center;
}
.phase-node.locked {
  opacity: 0.6;
  background: var(--color-secondary-bg);
}
.phase-node.active {
  background: var(--color-primary-light-4);
  border: 2px solid var(--color-primary);
}
.phase-node.future {
  cursor: pointer;
}
.phase-connector {
  width: 40px;
  height: 2px;
  background: var(--color-secondary);
}
.phase-connector.locked {
  background: var(--color-secondary-light);
}
.pulse-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-primary);
  animation: pulse 1.5s infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
.phase-name {
  font-weight: 600;
  display: block;
}
.phase-time {
  font-size: 0.85em;
  color: var(--color-text-light);
}

/* Responsive: vertical on mobile */
@media (max-width: 768px) {
  .phase-nodes {
    flex-direction: column;
  }
  .phase-connector {
    width: 2px;
    height: 20px;
  }
}
```

- [ ] **Step 3: Create Vue component for inline editing**

```vue
<!-- web_src/js/features/hackforger/PhaseTimeline.vue -->
<script>
import {POST} from '../../modules/fetch.js';
import {showErrorToast, showInfoToast} from '../../modules/toast.js';

export default {
  props: {
    phases: { type: Array, required: true },
    editable: { type: Boolean, default: false },
  },
  data() {
    return {
      editingPhaseId: null,
      editStartTime: '',
      editEndTime: '',
    };
  },
  methods: {
    startEdit(phase) {
      if (!this.editable || phase.IsLocked || phase.IsActive) return;
      this.editingPhaseId = phase.ID;
      this.editStartTime = new Date(phase.StartTime * 1000).toISOString().slice(0, 16);
      this.editEndTime = new Date(phase.EndTime * 1000).toISOString().slice(0, 16);
    },
    async saveEdit(phaseId) {
      const startTime = Math.floor(new Date(this.editStartTime).getTime() / 1000);
      const endTime = Math.floor(new Date(this.editEndTime).getTime() / 1000);

      try {
        const response = await POST(`/hackforger/phases/${phaseId}/update-time`, {
          data: { start_time: startTime, end_time: endTime },
        });
        if (!response.ok) {
          const data = await response.json();
          showErrorToast(data.errorMessage || 'Failed to update phase time');
          return;
        }
        showInfoToast('Phase time updated');
        this.editingPhaseId = null;
        window.location.reload();
      } catch (e) {
        showErrorToast(`Network error: ${e}`);
      }
    },
    cancelEdit() {
      this.editingPhaseId = null;
    },
  },
};
</script>
```

- [ ] **Step 4: Build and verify**

Run: `make frontend && make backend`

- [ ] **Step 5: Commit and create PR**

```bash
git checkout -b feat/hf-phase-timeline-ui
git add templates/hackforger/ web_src/js/features/hackforger/ web_src/css/
git commit -m "feat(hackforger): Phase Timeline UI component"
gh pr create --title "feat(hackforger): Phase Timeline UI component" --body "Reusable Phase Timeline template + Vue enhancement for inline time editing. Supports locked/active/future states."
```

---

### Task 7: Admin Phase Type Management

**Files:**
- Create: `templates/admin/hackforger/phase_types.tmpl`
- Create: `routers/web/hackforger/admin_phase_types.go`
- Modify: Admin navbar (add Phase Types link, extend from Task 6 in Wave 1)

- [ ] **Step 1: Create admin page template**

Template with: list of phase types grouped by activity_kind, create/edit/delete forms.

- [ ] **Step 2: Create route handlers**

CRUD handlers for phase types:
- `GET /admin/hackforger/phase-types` → list page
- `POST /admin/hackforger/phase-types` → create
- `POST /admin/hackforger/phase-types/:id/edit` → update
- `POST /admin/hackforger/phase-types/:id/delete` → delete

Each handler validates actions against `modules/hackforger.ValidateActions()`.

- [ ] **Step 3: Register routes and commit**

```bash
git checkout -b feat/hf-admin-phase-types
git add templates/admin/hackforger/ routers/web/hackforger/
git commit -m "feat(hackforger): Admin Phase Type management page"
gh pr create --title "feat(hackforger): Admin Phase Type management" --body "Admin UI for managing phase type catalog: create, edit, delete phase types with action checkbox selection."
```

---

### Task 8: Integrate Phase Gating into Activity Services

**Files:**
- Modify: `services/hackforger/hackathon.go`
- Modify: `services/hackforger/bounty.go`
- Modify: `services/hackforger/grant.go`
- Modify: `options/locale/locale_en-US.ini`, `locale_zh-CN.ini`

- [ ] **Step 1: Add phase gating to all phase-sensitive operations**

For each service function that should be gated by phase:

```go
// Hackathon registration
func RegisterForHackathon(ctx context.Context, hackathonID, userID int64) error {
    allowed, err := AllowsAction(ctx, "hackathon", hackathonID, "register")
    if err != nil { return err }
    if !allowed {
        return ErrPhaseActionNotAllowed
    }
    // ... existing logic
}

// Hackathon submission
func CreateSubmission(ctx context.Context, hackathonID, userID, trackID int64, ...) error {
    allowed, err := AllowsAction(ctx, "hackathon", hackathonID, "submit_work")
    if err != nil { return err }
    if !allowed {
        return ErrPhaseActionNotAllowed
    }
    // ... existing logic
}

// Hackathon scoring
func SubmitScore(ctx context.Context, hackathonID, judgeID, submissionID int64, score int) error {
    allowed, err := AllowsAction(ctx, "hackathon", hackathonID, "score")
    if err != nil { return err }
    if !allowed {
        return ErrPhaseActionNotAllowed
    }
    // ... existing logic
}
```

Same pattern for Bounty (apply, claim, submit_pr, review, settle) and Grant (apply, review, approve, disburse).

- [ ] **Step 2: Add i18n for phase action errors**

```ini
; en-US
hackforger.error.phase_action_not_allowed = This action is not available during the current phase.
hackforger.error.phase_no_active_phase = No active phase. This activity is currently between phases.

; zh-CN
hackforger.error.phase_action_not_allowed = 当前阶段不允许此操作。
hackforger.error.phase_no_active_phase = 没有活跃阶段，当前活动处于阶段间隔中。
```

- [ ] **Step 3: Map errors in web route handlers**

```go
if errors.Is(err, hackforger_service.ErrPhaseActionNotAllowed) {
    ctx.Flash.Error(ctx.Tr("hackforger.error.phase_action_not_allowed"), true)
    // render current page
    return
}
```

- [ ] **Step 4: Include Phase Timeline on activity pages**

In hackathon/bounty/grant detail templates, add:

```html
{{template "hackforger/components/phase_timeline" dict "Phases" .Phases "Editable" .IsOrganizer}}
```

Route handlers must load phases:
```go
phases, _ := hackforger_service.GetPhases(ctx, "hackathon", hackathon.ID)
ctx.Data["Phases"] = phases
ctx.Data["IsOrganizer"] = isOrganizer
```

- [ ] **Step 5: Write integration tests**

```go
func TestHackathonSubmissionBlockedOutsidePhase(t *testing.T) {
    // Create hackathon with development phase in the future
    // Attempt submit_work → ErrPhaseActionNotAllowed
    // Fast-forward time to within development phase
    // Attempt submit_work → success
}
```

Run: `go test ./services/hackforger/... -run TestHackathon -v`

- [ ] **Step 6: Commit and create PR**

```bash
git checkout -b feat/hf-phase-integration
git add services/hackforger/ routers/ templates/ options/locale/
git commit -m "feat(hackforger): integrate Phase gating into all activity services"
gh pr create --title "feat(hackforger): Phase gating integration" --body "Gate all phase-sensitive operations (register, submit, score, apply, review, etc.) through AllowsAction(). Show Phase Timeline on all activity detail pages."
```

---

## E2E Verification After Wave 3

After all Wave 3 PRs are merged:

- [ ] Create hackathon with 4 phases (registration, development, judging, results) with specific times
- [ ] Verify Phase Timeline shows on hackathon detail page
- [ ] During registration phase: verify can register, cannot submit work
- [ ] During development phase: verify can submit, cannot register
- [ ] Edit future phase time → verify it updates
- [ ] Try editing past phase time → verify rejected
- [ ] Add a second "development" phase → verify it works (is_unique=false)
- [ ] Try adding second "registration" phase → verify blocked (is_unique=true)
- [ ] Check admin panel → Phase Types management works
- [ ] Check API → `/api/v1/hackforger/hackathon/{id}/current-phase` returns correct phase
- [ ] Verify phase transition notification fires when phase starts

Take screenshots at each checkpoint. Save report to `docs/tests/e2e/wave3-results.md`.
