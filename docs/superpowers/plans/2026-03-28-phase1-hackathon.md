# Phase 1 Hackathon Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the full Hackathon lifecycle (create, register, submit, judge, finalize) with 26 API endpoints, 8 web pages, and 6 Feed events.

**Architecture:** Forgejo's strict layered architecture — models (CRUD) → services (business logic) → routers (HTTP). New `HackathonJudge` table for judge assignments. State machine in services layer. Feed events via `PublishHackforgerAction` in the existing notifier.

**Tech Stack:** Go 1.22+, XORM ORM, go-chi router, Go HTML templates, Fomantic UI CSS.

**Spec:** `docs/superpowers/specs/2026-03-28-phase1-hackathon-design.md`

---

## Chunk 0: Pre-Existing Fixes

### Task 0: Fix P0 Migration Table Names

The P0 migration (`v14c_add-hackforger-tables.go`) creates tables with bare names (`hackathon`, `hackathon_track`, etc.) but the model `TableName()` methods return prefixed names (`hackforger_hackathon`, `hackforger_hackathon_track`, etc.). This causes all CRUD to fail.

**Files:**
- Create: `models/forgejo_migrations/v14d_fix-hackforger-table-names.go`

- [ ] **Step 1: Write table rename migration**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "rename hackforger tables to use hackforger_ prefix",
		Upgrade:     renameHackforgerTables,
	})
}

func renameHackforgerTables(x *xorm.Engine) error {
	renames := [][2]string{
		{"hackathon", "hackforger_hackathon"},
		{"hackathon_track", "hackforger_hackathon_track"},
		{"hackathon_registration", "hackforger_hackathon_registration"},
		{"hackathon_submission", "hackforger_hackathon_submission"},
		{"hackathon_judge_score", "hackforger_hackathon_judge_score"},
		{"bounty", "hackforger_bounty"},
		{"bounty_reward", "hackforger_bounty_reward"},
		{"bounty_application", "hackforger_bounty_application"},
		{"bounty_winner", "hackforger_bounty_winner"},
		{"grant_round", "hackforger_grant_round"},
		{"grant_project", "hackforger_grant_project"},
		{"credit_account", "hackforger_credit_account"},
		{"credit_transaction", "hackforger_credit_transaction"},
		{"redeem_option", "hackforger_redeem_option"},
		{"redeem_order", "hackforger_redeem_order"},
		{"reputation", "hackforger_reputation"},
	}
	for _, r := range renames {
		exists, err := x.IsTableExist(r[0])
		if err != nil {
			return err
		}
		if exists {
			if _, err := x.Exec("ALTER TABLE `" + r[0] + "` RENAME TO `" + r[1] + "`"); err != nil {
				return err
			}
		}
	}
	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./models/forgejo_migrations/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add models/forgejo_migrations/v14d_fix-hackforger-table-names.go
git commit -m "fix(hackforger): rename P0 tables to use hackforger_ prefix"
```

---

## Chunk 1: Models Layer

### Task 1: HackathonJudge Model + Migration

**Files:**
- Create: `models/hackforger/hackathon_judge.go`
- Create: `models/forgejo_migrations/v14e_add-hackathon-judge-table.go`

- [ ] **Step 1: Write hackathon_judge.go model**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// HackathonJudge represents a judge assignment for a hackathon.
type HackathonJudge struct {
	ID          int64              `xorm:"pk autoincr"`
	HackathonID int64             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	UserID      int64              `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(HackathonJudge))
}

func (j *HackathonJudge) TableName() string {
	return "hackforger_hackathon_judge"
}

type ErrDuplicateJudge struct {
	HackathonID int64
	UserID      int64
}

func IsErrDuplicateJudge(err error) bool {
	_, ok := err.(ErrDuplicateJudge)
	return ok
}

func (err ErrDuplicateJudge) Error() string {
	return fmt.Sprintf("judge already assigned [hackathon_id: %d, user_id: %d]", err.HackathonID, err.UserID)
}

func (err ErrDuplicateJudge) Unwrap() error {
	return util.ErrAlreadyExist
}

// AddJudge assigns a user as judge for a hackathon.
func AddJudge(ctx context.Context, hackathonID, userID int64) error {
	exists, err := db.GetEngine(ctx).Where("hackathon_id = ? AND user_id = ?", hackathonID, userID).Exist(new(HackathonJudge))
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateJudge{HackathonID: hackathonID, UserID: userID}
	}
	return db.Insert(ctx, &HackathonJudge{HackathonID: hackathonID, UserID: userID})
}

// RemoveJudge removes a judge from a hackathon.
func RemoveJudge(ctx context.Context, hackathonID, userID int64) error {
	_, err := db.GetEngine(ctx).Where("hackathon_id = ? AND user_id = ?", hackathonID, userID).Delete(new(HackathonJudge))
	return err
}

// ListJudges returns all judges for a hackathon.
func ListJudges(ctx context.Context, hackathonID int64) ([]*HackathonJudge, error) {
	var judges []*HackathonJudge
	err := db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Find(&judges)
	return judges, err
}

// IsJudge checks if a user is a judge for a hackathon.
func IsJudge(ctx context.Context, hackathonID, userID int64) (bool, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ? AND user_id = ?", hackathonID, userID).Exist(new(HackathonJudge))
}

// CountJudges returns the number of judges for a hackathon.
func CountJudges(ctx context.Context, hackathonID int64) (int64, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonJudge))
}
```

- [ ] **Step 2: Write migration file**

File: `models/forgejo_migrations/v14e_add-hackathon-judge-table.go`

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "add hackathon judge assignment table",
		Upgrade:     addHackathonJudgeTable,
	})
}

type v14eHackathonJudge struct {
	ID          int64 `xorm:"pk autoincr"`
	HackathonID int64 `xorm:"UNIQUE(s) INDEX NOT NULL"`
	UserID      int64 `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CreatedUnix int64 `xorm:"created"`
}

func (v14eHackathonJudge) TableName() string { return "hackforger_hackathon_judge" }

func addHackathonJudgeTable(x *xorm.Engine) error {
	return x.Sync(new(v14eHackathonJudge))
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-hackathon && go build ./models/hackforger/ ./models/forgejo_migrations/`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/hackathon_judge.go models/forgejo_migrations/v14e_add-hackathon-judge-table.go
git commit -m "feat(hackforger): add HackathonJudge model and migration"
```

---

### Task 2: Enhance Hackathon Model (Pagination + Search + StatusUpdate)

**Files:**
- Modify: `models/hackforger/hackathon.go`

- [ ] **Step 1: Add Keyword to ListHackathonsOptions and switch to FindAndCount**

In `models/hackforger/hackathon.go`, modify:

```go
// ListHackathonsOptions holds options for listing hackathons.
type ListHackathonsOptions struct {
	db.ListOptions
	OrgID   int64
	Status  *HackathonStatus
	Keyword string
}

func (opts ListHackathonsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.OrgID != 0 {
		cond = cond.And(builder.Eq{"hackforger_hackathon.org_id": opts.OrgID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"hackforger_hackathon.status": *opts.Status})
	}
	if opts.Keyword != "" {
		cond = cond.And(builder.Or(
			builder.Like{"hackforger_hackathon.name", opts.Keyword},
			builder.Like{"hackforger_hackathon.description", opts.Keyword},
		))
	}
	return cond
}

// ListHackathons returns hackathons matching the given options with total count.
func ListHackathons(ctx context.Context, opts ListHackathonsOptions) ([]*Hackathon, int64, error) {
	hackathons, count, err := db.FindAndCount[Hackathon](ctx, opts)
	return hackathons, count, err
}
```

- [ ] **Step 2: Add UpdateHackathonStatus**

Append to `models/hackforger/hackathon.go`:

```go
// UpdateHackathonStatus updates only the status field of a hackathon.
func UpdateHackathonStatus(ctx context.Context, id int64, status HackathonStatus) error {
	_, err := db.GetEngine(ctx).ID(id).Cols("status").Update(&Hackathon{Status: status})
	return err
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./models/hackforger/`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/hackathon.go
git commit -m "feat(hackforger): add keyword search, pagination, and status update to Hackathon model"
```

---

### Task 3: HackathonTrack CRUD Functions

**Files:**
- Modify: `models/hackforger/hackathon_track.go`

- [ ] **Step 1: Add CRUD functions**

Append to `models/hackforger/hackathon_track.go`:

```go
import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// ErrTrackNotExist represents a "TrackNotExist" kind of error.
type ErrTrackNotExist struct {
	ID int64
}

func IsErrTrackNotExist(err error) bool {
	_, ok := err.(ErrTrackNotExist)
	return ok
}

func (err ErrTrackNotExist) Error() string {
	return fmt.Sprintf("hackathon track does not exist [id: %d]", err.ID)
}

func (err ErrTrackNotExist) Unwrap() error {
	return util.ErrNotExist
}

// GetTrackByID returns a track by its ID.
func GetTrackByID(ctx context.Context, id int64) (*HackathonTrack, error) {
	t, exists, err := db.GetByID[HackathonTrack](ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrTrackNotExist{ID: id}
	}
	return t, nil
}

// ListTracksByHackathon returns all tracks for a hackathon.
func ListTracksByHackathon(ctx context.Context, hackathonID int64) ([]*HackathonTrack, error) {
	var tracks []*HackathonTrack
	err := db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).OrderBy("id ASC").Find(&tracks)
	return tracks, err
}

// CreateTrack creates a new track.
func CreateTrack(ctx context.Context, t *HackathonTrack) error {
	return db.Insert(ctx, t)
}

// UpdateTrack updates an existing track.
func UpdateTrack(ctx context.Context, t *HackathonTrack) error {
	_, err := db.GetEngine(ctx).ID(t.ID).AllCols().Update(t)
	return err
}

// DeleteTrack deletes a track by ID.
func DeleteTrack(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(new(HackathonTrack))
	return err
}

// CountTracksByHackathon returns the number of tracks for a hackathon.
func CountTracksByHackathon(ctx context.Context, hackathonID int64) (int64, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonTrack))
}
```

Note: The existing `HackathonTrack` struct has no `SortOrder` field. The `OrderBy("sort_order")` references the spec's `SortOrder` column from the design doc. Check the struct — if `SortOrder` is missing, order by `id ASC` only. Looking at the struct (Task 3, Step 1 note): the existing struct does NOT have SortOrder. Use `OrderBy("id ASC")` instead.

- [ ] **Step 2: Verify compilation**

Run: `go build ./models/hackforger/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/hackathon_track.go
git commit -m "feat(hackforger): add HackathonTrack CRUD functions"
```

---

### Task 4: HackathonRegistration CRUD Functions

**Files:**
- Modify: `models/hackforger/hackathon_registration.go`

- [ ] **Step 1: Add CRUD functions**

Append to `models/hackforger/hackathon_registration.go`:

```go
import (
	"context"
	// existing imports...
)

// ErrDuplicateRegistration represents a duplicate registration error.
type ErrDuplicateRegistration struct {
	HackathonID int64
	UserID      int64
}

func IsErrDuplicateRegistration(err error) bool {
	_, ok := err.(ErrDuplicateRegistration)
	return ok
}

func (err ErrDuplicateRegistration) Error() string {
	return fmt.Sprintf("user already registered [hackathon_id: %d, user_id: %d]", err.HackathonID, err.UserID)
}

func (err ErrDuplicateRegistration) Unwrap() error {
	return util.ErrAlreadyExist
}

// GetRegistration returns a registration for a user in a hackathon.
func GetRegistration(ctx context.Context, hackathonID, userID int64) (*HackathonRegistration, error) {
	r := &HackathonRegistration{}
	has, err := db.GetEngine(ctx).Where("hackathon_id = ? AND user_id = ?", hackathonID, userID).Get(r)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrRegistrationNotExist{ID: 0}
	}
	return r, nil
}

// GetRegistrationByID returns a registration by its ID.
func GetRegistrationByID(ctx context.Context, id int64) (*HackathonRegistration, error) {
	r, exists, err := db.GetByID[HackathonRegistration](ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrRegistrationNotExist{ID: id}
	}
	return r, nil
}

// ListRegistrationsOptions holds options for listing registrations.
type ListRegistrationsOptions struct {
	db.ListOptions
	HackathonID int64
	Status      *RegistrationStatus
}

func (opts ListRegistrationsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.HackathonID != 0 {
		cond = cond.And(builder.Eq{"hackforger_hackathon_registration.hackathon_id": opts.HackathonID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"hackforger_hackathon_registration.status": *opts.Status})
	}
	return cond
}

// ListRegistrations returns registrations matching the given options with total count.
func ListRegistrations(ctx context.Context, opts ListRegistrationsOptions) ([]*HackathonRegistration, int64, error) {
	return db.FindAndCount[HackathonRegistration](ctx, opts)
}

// CreateRegistration creates a new registration.
func CreateRegistration(ctx context.Context, r *HackathonRegistration) error {
	exists, err := db.GetEngine(ctx).Where("hackathon_id = ? AND user_id = ?", r.HackathonID, r.UserID).Exist(new(HackathonRegistration))
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateRegistration{HackathonID: r.HackathonID, UserID: r.UserID}
	}
	return db.Insert(ctx, r)
}

// UpdateRegistrationStatus updates the status of a registration.
func UpdateRegistrationStatus(ctx context.Context, id int64, status RegistrationStatus) error {
	_, err := db.GetEngine(ctx).ID(id).Cols("status").Update(&HackathonRegistration{Status: status})
	return err
}

// CountRegistrations returns the total number of registrations for a hackathon.
func CountRegistrations(ctx context.Context, hackathonID int64) (int64, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonRegistration))
}
```

Note: Needs `"xorm.io/builder"` and `"context"` in imports. Adjust the existing import block to include these.

- [ ] **Step 2: Verify compilation**

Run: `go build ./models/hackforger/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/hackathon_registration.go
git commit -m "feat(hackforger): add HackathonRegistration CRUD functions"
```

---

### Task 5: HackathonSubmission CRUD Functions

**Files:**
- Modify: `models/hackforger/hackathon_submission.go`

- [ ] **Step 1: Add CRUD functions**

Append to `models/hackforger/hackathon_submission.go`:

```go
import (
	"context"
	// existing imports...
)

// ListSubmissionsOptions holds options for listing submissions.
type ListSubmissionsOptions struct {
	db.ListOptions
	HackathonID int64
	TrackID     int64
	Status      *SubmissionStatus
}

func (opts ListSubmissionsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.HackathonID != 0 {
		cond = cond.And(builder.Eq{"hackforger_hackathon_submission.hackathon_id": opts.HackathonID})
	}
	if opts.TrackID != 0 {
		cond = cond.And(builder.Eq{"hackforger_hackathon_submission.track_id": opts.TrackID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"hackforger_hackathon_submission.status": *opts.Status})
	}
	return cond
}

// GetSubmissionByID returns a submission by its ID.
func GetSubmissionByID(ctx context.Context, id int64) (*HackathonSubmission, error) {
	s, exists, err := db.GetByID[HackathonSubmission](ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrSubmissionNotExist{ID: id}
	}
	return s, nil
}

// ListSubmissions returns submissions matching the given options with total count.
func ListSubmissions(ctx context.Context, opts ListSubmissionsOptions) ([]*HackathonSubmission, int64, error) {
	return db.FindAndCount[HackathonSubmission](ctx, opts)
}

// CreateSubmission creates a new submission.
func CreateSubmission(ctx context.Context, s *HackathonSubmission) error {
	return db.Insert(ctx, s)
}

// UpdateSubmission updates an existing submission.
func UpdateSubmission(ctx context.Context, s *HackathonSubmission) error {
	_, err := db.GetEngine(ctx).ID(s.ID).AllCols().Update(s)
	return err
}

// CountSubmissions returns the number of submissions for a hackathon.
func CountSubmissions(ctx context.Context, hackathonID int64) (int64, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonSubmission))
}

// SubmissionRanking holds a submission ID and its computed rank.
type SubmissionRanking struct {
	SubmissionID int64
	TotalScore   float64
	Rank         int
}

// UpdateSubmissionRanks batch-updates TotalScore and Rank for submissions.
func UpdateSubmissionRanks(ctx context.Context, rankings []SubmissionRanking) error {
	for _, r := range rankings {
		if _, err := db.GetEngine(ctx).ID(r.SubmissionID).
			Cols("total_score", "rank").
			Update(&HackathonSubmission{TotalScore: r.TotalScore, Rank: r.Rank}); err != nil {
			return err
		}
	}
	return nil
}
```

Note: Needs `"xorm.io/builder"` and `"context"` in imports.

- [ ] **Step 2: Verify compilation**

Run: `go build ./models/hackforger/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/hackathon_submission.go
git commit -m "feat(hackforger): add HackathonSubmission CRUD functions"
```

---

### Task 6: HackathonJudgeScore CRUD Functions

**Files:**
- Modify: `models/hackforger/hackathon_judge_score.go`

- [ ] **Step 1: Add CRUD functions**

Append to `models/hackforger/hackathon_judge_score.go`:

```go
import (
	"context"
	// existing imports...
)

// GetScore returns a judge's score for a submission.
func GetScore(ctx context.Context, judgeID, submissionID int64) (*HackathonJudgeScore, error) {
	s := &HackathonJudgeScore{}
	has, err := db.GetEngine(ctx).Where("judge_id = ? AND submission_id = ?", judgeID, submissionID).Get(s)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return s, nil
}

// ListScoresBySubmission returns all scores for a submission.
func ListScoresBySubmission(ctx context.Context, submissionID int64) ([]*HackathonJudgeScore, error) {
	var scores []*HackathonJudgeScore
	err := db.GetEngine(ctx).Where("submission_id = ?", submissionID).Find(&scores)
	return scores, err
}

// CreateScore creates a new score record.
func CreateScore(ctx context.Context, s *HackathonJudgeScore) error {
	exists, err := db.GetEngine(ctx).Where("judge_id = ? AND submission_id = ?", s.JudgeID, s.SubmissionID).Exist(new(HackathonJudgeScore))
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateScore{JudgeID: s.JudgeID, SubmissionID: s.SubmissionID}
	}
	return db.Insert(ctx, s)
}

// UpdateScore updates an existing score.
func UpdateScore(ctx context.Context, s *HackathonJudgeScore) error {
	_, err := db.GetEngine(ctx).ID(s.ID).Cols("score", "comment").Update(s)
	return err
}

// HasAllJudgesScored checks if every assigned judge has scored a given submission.
func HasAllJudgesScored(ctx context.Context, hackathonID, submissionID int64) (bool, error) {
	judgeCount, err := db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonJudge))
	if err != nil {
		return false, err
	}
	if judgeCount == 0 {
		return false, nil
	}
	scoreCount, err := db.GetEngine(ctx).Where("submission_id = ?", submissionID).Count(new(HackathonJudgeScore))
	if err != nil {
		return false, err
	}
	return scoreCount >= judgeCount, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./models/hackforger/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/hackathon_judge_score.go
git commit -m "feat(hackforger): add HackathonJudgeScore CRUD functions"
```

---

### Task 7: Model Unit Tests

**Files:**
- Create: `models/hackforger/main_test.go`
- Create: `models/hackforger/hackathon_test.go`

- [ ] **Step 1: Create test bootstrap**

File: `models/hackforger/main_test.go`

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/modules/testimport"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
```

- [ ] **Step 2: Write hackathon model tests**

File: `models/hackforger/hackathon_test.go`

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndGetHackathon(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	h := &hackforger_model.Hackathon{
		OrgID:   1,
		OwnerID: 1,
		Name:    "Test Hackathon",
		Slug:    "test-hackathon",
		Status:  hackforger_model.HackathonStatusDraft,
	}
	require.NoError(t, hackforger_model.CreateHackathon(db.DefaultContext, h))
	assert.Greater(t, h.ID, int64(0))

	got, err := hackforger_model.GetHackathonByID(db.DefaultContext, h.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Hackathon", got.Name)

	gotBySlug, err := hackforger_model.GetHackathonBySlug(db.DefaultContext, "test-hackathon")
	require.NoError(t, err)
	assert.Equal(t, h.ID, gotBySlug.ID)
}

func TestListHackathonsWithKeyword(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create two hackathons
	h1 := &hackforger_model.Hackathon{OrgID: 1, OwnerID: 1, Name: "Spring Hack", Slug: "spring-hack"}
	h2 := &hackforger_model.Hackathon{OrgID: 1, OwnerID: 1, Name: "Winter Jam", Slug: "winter-jam"}
	require.NoError(t, hackforger_model.CreateHackathon(db.DefaultContext, h1))
	require.NoError(t, hackforger_model.CreateHackathon(db.DefaultContext, h2))

	// Search by keyword
	results, count, err := hackforger_model.ListHackathons(db.DefaultContext, hackforger_model.ListHackathonsOptions{
		Keyword: "Spring",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, "Spring Hack", results[0].Name)
}

func TestUpdateHackathonStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	h := &hackforger_model.Hackathon{OrgID: 1, OwnerID: 1, Name: "Status Test", Slug: "status-test"}
	require.NoError(t, hackforger_model.CreateHackathon(db.DefaultContext, h))

	require.NoError(t, hackforger_model.UpdateHackathonStatus(db.DefaultContext, h.ID, hackforger_model.HackathonStatusOpen))

	got, err := hackforger_model.GetHackathonByID(db.DefaultContext, h.ID)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.HackathonStatusOpen, got.Status)
}

func TestDeleteHackathonOnlyDraft(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	h := &hackforger_model.Hackathon{OrgID: 1, OwnerID: 1, Name: "Del Test", Slug: "del-test", Status: hackforger_model.HackathonStatusOpen}
	require.NoError(t, hackforger_model.CreateHackathon(db.DefaultContext, h))

	err := hackforger_model.DeleteHackathon(db.DefaultContext, h.ID)
	assert.Error(t, err) // Should fail — not Draft
}

func TestHackathonJudgeCRUD(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	hackathonID := int64(999)

	// Add judge
	require.NoError(t, hackforger_model.AddJudge(db.DefaultContext, hackathonID, 1))

	// Duplicate should error
	err := hackforger_model.AddJudge(db.DefaultContext, hackathonID, 1)
	assert.True(t, hackforger_model.IsErrDuplicateJudge(err))

	// IsJudge
	isJudge, err := hackforger_model.IsJudge(db.DefaultContext, hackathonID, 1)
	require.NoError(t, err)
	assert.True(t, isJudge)

	// List judges
	judges, err := hackforger_model.ListJudges(db.DefaultContext, hackathonID)
	require.NoError(t, err)
	assert.Len(t, judges, 1)

	// Count
	count, err := hackforger_model.CountJudges(db.DefaultContext, hackathonID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Remove
	require.NoError(t, hackforger_model.RemoveJudge(db.DefaultContext, hackathonID, 1))
	count, err = hackforger_model.CountJudges(db.DefaultContext, hackathonID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestHackathonTrackCRUD(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t1 := &hackforger_model.HackathonTrack{HackathonID: 1, Name: "AI Track"}
	require.NoError(t, hackforger_model.CreateTrack(db.DefaultContext, t1))
	assert.Greater(t, t1.ID, int64(0))

	got, err := hackforger_model.GetTrackByID(db.DefaultContext, t1.ID)
	require.NoError(t, err)
	assert.Equal(t, "AI Track", got.Name)

	tracks, err := hackforger_model.ListTracksByHackathon(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, tracks, 1)

	count, err := hackforger_model.CountTracksByHackathon(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	require.NoError(t, hackforger_model.DeleteTrack(db.DefaultContext, t1.ID))
	count, err = hackforger_model.CountTracksByHackathon(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestRegistrationCRUD(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	r := &hackforger_model.HackathonRegistration{HackathonID: 1, UserID: 2, TeamName: "Team Alpha"}
	require.NoError(t, hackforger_model.CreateRegistration(db.DefaultContext, r))

	// Duplicate
	err := hackforger_model.CreateRegistration(db.DefaultContext, &hackforger_model.HackathonRegistration{HackathonID: 1, UserID: 2, TeamName: "Dup"})
	assert.True(t, hackforger_model.IsErrDuplicateRegistration(err))

	// Get
	got, err := hackforger_model.GetRegistration(db.DefaultContext, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, "Team Alpha", got.TeamName)

	// Update status
	require.NoError(t, hackforger_model.UpdateRegistrationStatus(db.DefaultContext, r.ID, hackforger_model.RegistrationStatusApproved))
	got, err = hackforger_model.GetRegistrationByID(db.DefaultContext, r.ID)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.RegistrationStatusApproved, got.Status)
}

func TestSubmissionCRUD(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	s := &hackforger_model.HackathonSubmission{
		HackathonID:    1,
		RegistrationID: 1,
		UserID:         2,
		Title:          "My Project",
		DemoURL:        "https://example.com/demo",
	}
	require.NoError(t, hackforger_model.CreateSubmission(db.DefaultContext, s))
	assert.Greater(t, s.ID, int64(0))

	got, err := hackforger_model.GetSubmissionByID(db.DefaultContext, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "My Project", got.Title)

	count, err := hackforger_model.CountSubmissions(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestScoreCRUD(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	sc := &hackforger_model.HackathonJudgeScore{
		HackathonID:  1,
		SubmissionID: 1,
		JudgeID:      10,
		Score:        8.5,
		Comment:      "Good work",
	}
	require.NoError(t, hackforger_model.CreateScore(db.DefaultContext, sc))

	// Duplicate
	err := hackforger_model.CreateScore(db.DefaultContext, &hackforger_model.HackathonJudgeScore{
		HackathonID: 1, SubmissionID: 1, JudgeID: 10, Score: 9,
	})
	assert.True(t, hackforger_model.IsErrDuplicateScore(err))

	// Get
	got, err := hackforger_model.GetScore(db.DefaultContext, 10, 1)
	require.NoError(t, err)
	assert.Equal(t, 8.5, got.Score)

	// List
	scores, err := hackforger_model.ListScoresBySubmission(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, scores, 1)
}
```

- [ ] **Step 3: Run tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-hackathon && TAGS="bindata sqlite sqlite_unlock_notify" go test ./models/hackforger/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/main_test.go models/hackforger/hackathon_test.go
git commit -m "test(hackforger): add Hackathon model unit tests"
```

---

## Chunk 2: Services Layer

### Task 8: Hackathon State Machine Service

**Files:**
- Create: `services/hackforger/hackathon.go`

- [ ] **Step 1: Write state machine service**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
)

// PublishHackathon transitions a hackathon from Draft → Open.
// Requires: at least 1 track.
func PublishHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status != hackforger_model.HackathonStatusDraft {
		return fmt.Errorf("hackathon must be in Draft status to publish [id: %d, status: %d]", h.ID, h.Status)
	}
	trackCount, err := hackforger_model.CountTracksByHackathon(ctx, h.ID)
	if err != nil {
		return err
	}
	if trackCount == 0 {
		return fmt.Errorf("hackathon must have at least 1 track to publish [id: %d]", h.ID)
	}

	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusOpen); err != nil {
		return err
	}

	publishPhaseChange(ctx, doerID, h, hackforger_model.HackathonStatusDraft, hackforger_model.HackathonStatusOpen)
	return nil
}

// StartHacking transitions a hackathon from Open → Hacking.
func StartHacking(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status != hackforger_model.HackathonStatusOpen {
		return fmt.Errorf("hackathon must be in Open status to start hacking [id: %d, status: %d]", h.ID, h.Status)
	}

	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusHacking); err != nil {
		return err
	}

	publishPhaseChange(ctx, doerID, h, hackforger_model.HackathonStatusOpen, hackforger_model.HackathonStatusHacking)
	return nil
}

// StartJudging transitions a hackathon from Hacking → Judging.
// Requires: at least 1 submission.
func StartJudging(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status != hackforger_model.HackathonStatusHacking {
		return fmt.Errorf("hackathon must be in Hacking status to start judging [id: %d, status: %d]", h.ID, h.Status)
	}
	subCount, err := hackforger_model.CountSubmissions(ctx, h.ID)
	if err != nil {
		return err
	}
	if subCount == 0 {
		return fmt.Errorf("hackathon must have at least 1 submission to start judging [id: %d]", h.ID)
	}

	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusJudging); err != nil {
		return err
	}

	publishPhaseChange(ctx, doerID, h, hackforger_model.HackathonStatusHacking, hackforger_model.HackathonStatusJudging)
	return nil
}

// FinalizeHackathon transitions a hackathon from Judging → Finished.
// Calculates ranks and publishes finalized event.
func FinalizeHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status != hackforger_model.HackathonStatusJudging {
		return fmt.Errorf("hackathon must be in Judging status to finalize [id: %d, status: %d]", h.ID, h.Status)
	}

	if err := CalculateRanks(ctx, h.ID); err != nil {
		return err
	}

	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusFinished); err != nil {
		return err
	}

	// Finalized event (global audience)
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doerID,
		OpType:       hackforger_model.ActionHackathonFinalized,
		AudienceType: AudienceGlobal,
		Content: hackforger_model.HackforgerPhaseContent{
			HackforgerActionContent: hackforger_model.HackforgerActionContent{
				EntityType: "hackathon",
				EntityID:   h.ID,
				EntityName: h.Name,
				EntitySlug: h.Slug,
			},
			OldStatus: "judging",
			NewStatus: "finished",
		},
	})
	return nil
}

// CancelHackathon transitions a hackathon to Cancelled.
// Any non-Finished status can be cancelled.
func CancelHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status == hackforger_model.HackathonStatusFinished {
		return fmt.Errorf("cannot cancel a finished hackathon [id: %d]", h.ID)
	}
	if h.Status == hackforger_model.HackathonStatusCancelled {
		return fmt.Errorf("hackathon is already cancelled [id: %d]", h.ID)
	}

	oldStatus := h.Status
	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusCancelled); err != nil {
		return err
	}

	publishPhaseChange(ctx, doerID, h, oldStatus, hackforger_model.HackathonStatusCancelled)
	return nil
}

// CreateHackathon creates a hackathon and publishes a creation event.
func CreateHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if err := hackforger_model.CreateHackathon(ctx, h); err != nil {
		return err
	}

	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doerID,
		OpType:       hackforger_model.ActionHackathonCreated,
		AudienceType: AudienceGlobal,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon",
			EntityID:   h.ID,
			EntityName: h.Name,
			EntitySlug: h.Slug,
		},
	})
	return nil
}

// hackathonStatusName maps status to string for feed content.
var hackathonStatusName = map[hackforger_model.HackathonStatus]string{
	hackforger_model.HackathonStatusDraft:     "draft",
	hackforger_model.HackathonStatusOpen:      "open",
	hackforger_model.HackathonStatusHacking:   "hacking",
	hackforger_model.HackathonStatusJudging:   "judging",
	hackforger_model.HackathonStatusFinished:  "finished",
	hackforger_model.HackathonStatusCancelled: "cancelled",
}

// publishPhaseChange publishes a HackathonPhaseChanged feed event.
func publishPhaseChange(ctx context.Context, doerID int64, h *hackforger_model.Hackathon, oldStatus, newStatus hackforger_model.HackathonStatus) {
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doerID,
		OpType:       hackforger_model.ActionHackathonPhaseChanged,
		AudienceType: AudienceOrgMembers,
		OrgID:        h.OrgID,
		Content: hackforger_model.HackforgerPhaseContent{
			HackforgerActionContent: hackforger_model.HackforgerActionContent{
				EntityType: "hackathon",
				EntityID:   h.ID,
				EntityName: h.Name,
				EntitySlug: h.Slug,
			},
			OldStatus: hackathonStatusName[oldStatus],
			NewStatus: hackathonStatusName[newStatus],
		},
	})
}

// unused import guard
var _ = db.DefaultContext
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./services/hackforger/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add services/hackforger/hackathon.go
git commit -m "feat(hackforger): add Hackathon state machine service"
```

---

### Task 9: Hackathon Judge Service (Scoring + Ranks)

**Files:**
- Create: `services/hackforger/hackathon_judge.go`

- [ ] **Step 1: Write judge service**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"
	"sort"

	hackforger_model "forgejo.org/models/hackforger"
)

// SubmitScore validates and records a judge's score for a submission.
func SubmitScore(ctx context.Context, judgeID, submissionID int64, score float64, comment string) error {
	// Get submission to find hackathon
	sub, err := hackforger_model.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return err
	}

	// Verify hackathon is in Judging phase
	h, err := hackforger_model.GetHackathonByID(ctx, sub.HackathonID)
	if err != nil {
		return err
	}
	if h.Status != hackforger_model.HackathonStatusJudging {
		return fmt.Errorf("hackathon must be in Judging status to score [id: %d, status: %d]", h.ID, h.Status)
	}

	// Verify user is a judge
	isJudge, err := hackforger_model.IsJudge(ctx, h.ID, judgeID)
	if err != nil {
		return err
	}
	if !isJudge {
		return fmt.Errorf("user is not a judge for this hackathon [user_id: %d, hackathon_id: %d]", judgeID, h.ID)
	}

	// Validate score range (0-10)
	if score < 0 || score > 10 {
		return fmt.Errorf("score must be between 0 and 10 [got: %f]", score)
	}

	// Create or update score
	existing, err := hackforger_model.GetScore(ctx, judgeID, submissionID)
	if err != nil {
		return err
	}
	if existing != nil {
		existing.Score = score
		existing.Comment = comment
		if err := hackforger_model.UpdateScore(ctx, existing); err != nil {
			return err
		}
	} else {
		s := &hackforger_model.HackathonJudgeScore{
			HackathonID:  h.ID,
			SubmissionID: submissionID,
			JudgeID:      judgeID,
			Score:        score,
			Comment:      comment,
		}
		if err := hackforger_model.CreateScore(ctx, s); err != nil {
			return err
		}
	}

	// Publish scored event
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    judgeID,
		OpType:       hackforger_model.ActionHackathonScored,
		AudienceType: AudienceFollowers,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon",
			EntityID:   h.ID,
			EntityName: h.Name,
			EntitySlug: h.Slug,
			Extra:      map[string]any{"submission_id": submissionID},
		},
	})

	return nil
}

// CalculateRanks computes average scores and assigns ranks for all submissions in a hackathon.
func CalculateRanks(ctx context.Context, hackathonID int64) error {
	subs, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
		HackathonID: hackathonID,
	})
	if err != nil {
		return err
	}

	type scored struct {
		id    int64
		avg   float64
	}
	var scoredSubs []scored

	for _, sub := range subs {
		scores, err := hackforger_model.ListScoresBySubmission(ctx, sub.ID)
		if err != nil {
			return err
		}
		if len(scores) == 0 {
			scoredSubs = append(scoredSubs, scored{id: sub.ID, avg: 0})
			continue
		}
		var total float64
		for _, s := range scores {
			total += s.Score
		}
		scoredSubs = append(scoredSubs, scored{id: sub.ID, avg: total / float64(len(scores))})
	}

	// Sort descending by average score
	sort.Slice(scoredSubs, func(i, j int) bool {
		return scoredSubs[i].avg > scoredSubs[j].avg
	})

	// Build rankings
	rankings := make([]hackforger_model.SubmissionRanking, len(scoredSubs))
	for i, s := range scoredSubs {
		rankings[i] = hackforger_model.SubmissionRanking{
			SubmissionID: s.id,
			TotalScore:   s.avg,
			Rank:         i + 1,
		}
	}

	return hackforger_model.UpdateSubmissionRanks(ctx, rankings)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./services/hackforger/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add services/hackforger/hackathon_judge.go
git commit -m "feat(hackforger): add judge scoring and rank calculation service"
```

---

### Task 10: Extend Notifier Audience Resolution

**Files:**
- Modify: `services/hackforger/notifier.go`

- [ ] **Step 1: Implement full audience resolution in PublishHackforgerAction**

Replace the `// Phase 1 TODO` block (lines 96-99) with actual audience resolution:

```go
	// Phase 1: resolve audience based on opts.AudienceType
	switch opts.AudienceType {
	case AudienceFollowers:
		// Query followers of the actor
		var followerIDs []int64
		err := db.GetEngine(ctx).Table("user_follow").
			Where("follow_id = ?", opts.ActUserID).
			Cols("user_id").Find(&followerIDs)
		if err != nil {
			log.Error("PublishHackforgerAction (followers query): %v", err)
			break
		}
		for _, uid := range followerIDs {
			if uid == opts.ActUserID {
				continue // skip actor, already inserted
			}
			fa := &activities_model.Action{
				ActUserID:   opts.ActUserID,
				UserID:      uid,
				OpType:      opts.OpType,
				RepoID:      opts.RepoID,
				Content:     contentStr,
				CreatedUnix: now,
			}
			if _, err := db.GetEngine(ctx).Insert(fa); err != nil {
				log.Error("PublishHackforgerAction (follower %d): %v", uid, err)
			}
		}

	case AudienceOrgMembers:
		if opts.OrgID == 0 {
			break
		}
		var memberIDs []int64
		err := db.GetEngine(ctx).Table("org_user").
			Where("org_id = ?", opts.OrgID).
			Cols("uid").Find(&memberIDs)
		if err != nil {
			log.Error("PublishHackforgerAction (org members query): %v", err)
			break
		}
		for _, uid := range memberIDs {
			if uid == opts.ActUserID {
				continue
			}
			ma := &activities_model.Action{
				ActUserID:   opts.ActUserID,
				UserID:      uid,
				OpType:      opts.OpType,
				RepoID:      opts.RepoID,
				Content:     contentStr,
				CreatedUnix: now,
			}
			if _, err := db.GetEngine(ctx).Insert(ma); err != nil {
				log.Error("PublishHackforgerAction (org member %d): %v", uid, err)
			}
		}

	case AudienceRepoWatchers:
		if opts.RepoID == 0 {
			break
		}
		var watcherIDs []int64
		err := db.GetEngine(ctx).Table("watch").
			Where("repo_id = ? AND mode != 2", opts.RepoID). // mode 2 = don't watch
			Cols("user_id").Find(&watcherIDs)
		if err != nil {
			log.Error("PublishHackforgerAction (watchers query): %v", err)
			break
		}
		for _, uid := range watcherIDs {
			if uid == opts.ActUserID {
				continue
			}
			wa := &activities_model.Action{
				ActUserID:   opts.ActUserID,
				UserID:      uid,
				OpType:      opts.OpType,
				RepoID:      opts.RepoID,
				Content:     contentStr,
				CreatedUnix: now,
			}
			if _, err := db.GetEngine(ctx).Insert(wa); err != nil {
				log.Error("PublishHackforgerAction (watcher %d): %v", uid, err)
			}
		}
	}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./services/hackforger/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add services/hackforger/notifier.go
git commit -m "feat(hackforger): implement full audience resolution in PublishHackforgerAction"
```

---

### Task 11: Service Tests

**Files:**
- Create: `services/hackforger/main_test.go`
- Create: `services/hackforger/hackathon_test.go`

- [ ] **Step 1: Create test bootstrap**

File: `services/hackforger/main_test.go`

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/modules/testimport"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
```

- [ ] **Step 2: Write state machine tests**

File: `services/hackforger/hackathon_test.go`

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishHackathon(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := db.DefaultContext

	h := &hackforger_model.Hackathon{OrgID: 1, OwnerID: 1, Name: "Pub Test", Slug: "pub-test"}
	require.NoError(t, hackforger_model.CreateHackathon(ctx, h))

	// Should fail: no tracks
	err := hackforger_service.PublishHackathon(ctx, 1, h)
	assert.Error(t, err)

	// Add a track
	require.NoError(t, hackforger_model.CreateTrack(ctx, &hackforger_model.HackathonTrack{
		HackathonID: h.ID, Name: "Track 1",
	}))

	// Now should succeed
	require.NoError(t, hackforger_service.PublishHackathon(ctx, 1, h))

	got, err := hackforger_model.GetHackathonByID(ctx, h.ID)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.HackathonStatusOpen, got.Status)
}

func TestStartJudgingRequiresSubmission(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := db.DefaultContext

	h := &hackforger_model.Hackathon{OrgID: 1, OwnerID: 1, Name: "Judge Test", Slug: "judge-test"}
	require.NoError(t, hackforger_model.CreateHackathon(ctx, h))
	require.NoError(t, hackforger_model.CreateTrack(ctx, &hackforger_model.HackathonTrack{HackathonID: h.ID, Name: "T1"}))
	require.NoError(t, hackforger_service.PublishHackathon(ctx, 1, h))
	require.NoError(t, hackforger_service.StartHacking(ctx, 1, h))

	// No submissions — should fail
	h, _ = hackforger_model.GetHackathonByID(ctx, h.ID)
	err := hackforger_service.StartJudging(ctx, 1, h)
	assert.Error(t, err)
}

func TestFullLifecycle(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := db.DefaultContext

	// Create
	h := &hackforger_model.Hackathon{OrgID: 1, OwnerID: 1, Name: "Full LC", Slug: "full-lc"}
	require.NoError(t, hackforger_service.CreateHackathon(ctx, 1, h))

	// Add track
	require.NoError(t, hackforger_model.CreateTrack(ctx, &hackforger_model.HackathonTrack{HackathonID: h.ID, Name: "T1"}))

	// Publish
	require.NoError(t, hackforger_service.PublishHackathon(ctx, 1, h))

	// Register + approve
	reg := &hackforger_model.HackathonRegistration{HackathonID: h.ID, UserID: 2, TeamName: "Team A"}
	require.NoError(t, hackforger_model.CreateRegistration(ctx, reg))
	require.NoError(t, hackforger_model.UpdateRegistrationStatus(ctx, reg.ID, hackforger_model.RegistrationStatusApproved))

	// Start hacking
	h, _ = hackforger_model.GetHackathonByID(ctx, h.ID)
	require.NoError(t, hackforger_service.StartHacking(ctx, 1, h))

	// Submit
	sub := &hackforger_model.HackathonSubmission{
		HackathonID: h.ID, RegistrationID: reg.ID, UserID: 2,
		Title: "Project X", DemoURL: "https://example.com",
	}
	require.NoError(t, hackforger_model.CreateSubmission(ctx, sub))

	// Start judging
	h, _ = hackforger_model.GetHackathonByID(ctx, h.ID)
	require.NoError(t, hackforger_service.StartJudging(ctx, 1, h))

	// Add judge + score
	require.NoError(t, hackforger_model.AddJudge(ctx, h.ID, 10))
	require.NoError(t, hackforger_service.SubmitScore(ctx, 10, sub.ID, 8.5, "Nice"))

	// Finalize
	h, _ = hackforger_model.GetHackathonByID(ctx, h.ID)
	require.NoError(t, hackforger_service.FinalizeHackathon(ctx, 1, h))

	h, _ = hackforger_model.GetHackathonByID(ctx, h.ID)
	assert.Equal(t, hackforger_model.HackathonStatusFinished, h.Status)

	// Check rank was computed
	sub, _ = hackforger_model.GetSubmissionByID(ctx, sub.ID)
	assert.Equal(t, 1, sub.Rank)
	assert.Equal(t, 8.5, sub.TotalScore)
}

func TestCancelHackathon(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := db.DefaultContext

	h := &hackforger_model.Hackathon{OrgID: 1, OwnerID: 1, Name: "Cancel Test", Slug: "cancel-test"}
	require.NoError(t, hackforger_model.CreateHackathon(ctx, h))

	require.NoError(t, hackforger_service.CancelHackathon(ctx, 1, h))
	got, _ := hackforger_model.GetHackathonByID(ctx, h.ID)
	assert.Equal(t, hackforger_model.HackathonStatusCancelled, got.Status)

	// Can't cancel Finished
	h2 := &hackforger_model.Hackathon{OrgID: 1, OwnerID: 1, Name: "Fin", Slug: "fin", Status: hackforger_model.HackathonStatusFinished}
	require.NoError(t, hackforger_model.CreateHackathon(ctx, h2))
	err := hackforger_service.CancelHackathon(ctx, 1, h2)
	assert.Error(t, err)
}
```

- [ ] **Step 3: Run tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-hackathon && TAGS="bindata sqlite sqlite_unlock_notify" go test ./services/hackforger/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/main_test.go services/hackforger/hackathon_test.go
git commit -m "test(hackforger): add Hackathon service tests (state machine + scoring)"
```

---

## Chunk 3: API Layer

**API form binding pattern:** Forgejo does NOT use `ctx.BindJSON()`. Instead:
1. Define form structs (can be inline in the handler file or in `modules/structs/`)
2. Route registration uses `bind(FormStruct{})` middleware: `m.Post("/hackathons", reqToken(), bind(CreateHackathonForm{}), handler)`
3. Handler retrieves the form via: `form := web.GetForm(ctx).(*CreateHackathonForm)`
4. Import `"forgejo.org/services/web"` for `web.GetForm()`

The code blocks below show the form structs inline. Route registration in Task 15 must include `bind()` calls for every POST/PUT route that takes a request body.

**Path parameters:** Use `ctx.ParamsInt64(":id")` and `ctx.Params(":slug")` — note the colon prefix. Route registration uses `{id}`, `{slug}` syntax (chi router).

### Task 12: Hackathon API — CRUD + State Transitions

**Files:**
- Modify: `routers/api/v1/hackforger/hackathon.go`
- Modify: `routers/api/v1/api.go`

- [ ] **Step 1: Rewrite hackathon.go API handlers**

Replace the stub in `routers/api/v1/hackforger/hackathon.go` with full CRUD + state transition handlers. This is a large file (~400 lines). Key handlers:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"
	"strconv"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// ListHackathons returns a paginated list of hackathons.
func ListHackathons(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/hackathons hackforger hackforgerListHackathons
	// ---
	// summary: List hackathons
	// produces:
	// - application/json
	// parameters:
	// - name: status
	//   in: query
	//   description: Filter by status (0=draft,1=open,2=hacking,3=judging,4=finished,5=cancelled)
	//   type: integer
	// - name: org_id
	//   in: query
	//   description: Filter by organization ID
	//   type: integer
	// - name: q
	//   in: query
	//   description: Search keyword
	//   type: string
	// - name: page
	//   in: query
	//   description: Page number
	//   type: integer
	// - name: limit
	//   in: query
	//   description: Page size
	//   type: integer
	// responses:
	//   "200":
	//     description: "HackathonList"
	//     schema:
	//       type: array
	//       items:
	//         "$ref": "#/definitions/Hackathon"

	opts := hackforger_model.ListHackathonsOptions{
		Keyword: ctx.FormString("q"),
	}
	opts.Page = ctx.FormInt("page")
	opts.PageSize = ctx.FormInt("limit")
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 50 {
		opts.PageSize = 20
	}

	if orgID := ctx.FormInt64("org_id"); orgID > 0 {
		opts.OrgID = orgID
	}
	if statusStr := ctx.FormString("status"); statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			status := hackforger_model.HackathonStatus(s)
			opts.Status = &status
		}
	}

	hackathons, count, err := hackforger_model.ListHackathons(ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, hackathons)
}

// GetHackathon returns a single hackathon by ID.
func GetHackathon(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/hackathons/{id} hackforger hackforgerGetHackathon
	// ---
	// summary: Get a hackathon
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: Hackathon ID
	//   type: integer
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/definitions/Hackathon"
	//   "404":
	//     "$ref": "#/responses/notFound"

	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, h)
}

// CreateHackathon creates a new hackathon.
func CreateHackathon(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/hackathons hackforger hackforgerCreateHackathon
	// ---
	// summary: Create a hackathon
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// responses:
	//   "201":
	//     "$ref": "#/definitions/Hackathon"
	//   "422":
	//     "$ref": "#/responses/validationError"

	type createForm struct {
		Name        string `json:"name" binding:"Required"`
		Slug        string `json:"slug" binding:"Required;AlphaDashDot;MaxSize(100)"`
		Description string `json:"description"`
		OrgID       int64  `json:"org_id" binding:"Required"`
		MaxTeamSize int    `json:"max_team_size"`
	}

	form := &createForm{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: form := web.GetForm(ctx).(*formType)

	h := &hackforger_model.Hackathon{
		OrgID:       form.OrgID,
		OwnerID:     ctx.Doer.ID,
		Name:        form.Name,
		Slug:        form.Slug,
		Description: form.Description,
		MaxTeamSize: form.MaxTeamSize,
	}
	if h.MaxTeamSize <= 0 {
		h.MaxTeamSize = 5
	}

	if err := hackforger_service.CreateHackathon(ctx, ctx.Doer.ID, h); err != nil {
		if hackforger_model.IsErrHackathonSlugAlreadyExist(err) {
			ctx.Error(http.StatusConflict, "SlugExists", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusCreated, h)
}

// UpdateHackathon updates a hackathon.
func UpdateHackathon(ctx *context.APIContext) {
	// swagger:operation PUT /hackforger/hackathons/{id} hackforger hackforgerUpdateHackathon
	// ---
	// summary: Update a hackathon
	// parameters:
	// - name: id
	//   in: path
	//   type: integer
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/definitions/Hackathon"

	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}

	type updateForm struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		MaxTeamSize *int    `json:"max_team_size"`
		PrizeSummary *string `json:"prize_summary"`
	}

	form := &updateForm{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: form := web.GetForm(ctx).(*formType)

	if form.Name != nil {
		h.Name = *form.Name
	}
	if form.Description != nil {
		h.Description = *form.Description
	}
	if form.MaxTeamSize != nil {
		h.MaxTeamSize = *form.MaxTeamSize
	}
	if form.PrizeSummary != nil {
		h.PrizeSummary = *form.PrizeSummary
	}

	if err := hackforger_model.UpdateHackathon(ctx, h); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, h)
}

// DeleteHackathon deletes a hackathon (Draft only).
func DeleteHackathon(ctx *context.APIContext) {
	// swagger:operation DELETE /hackforger/hackathons/{id} hackforger hackforgerDeleteHackathon
	// ---
	// summary: Delete a hackathon (Draft only)
	// parameters:
	// - name: id
	//   in: path
	//   type: integer
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	if err := hackforger_model.DeleteHackathon(ctx, ctx.ParamsInt64(":id")); err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.Error(http.StatusForbidden, "DeleteHackathon", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// helper to load hackathon from path param
func getHackathonFromPath(ctx *context.APIContext) *hackforger_model.Hackathon {
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.InternalServerError(err)
		}
		return nil
	}
	return h
}

// PublishHackathon transitions Draft → Open.
func PublishHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.PublishHackathon(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "PublishHackathon", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "open"})
}

// StartHackathon transitions Open → Hacking.
func StartHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.StartHacking(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "StartHacking", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "hacking"})
}

// StartJudgingHackathon transitions Hacking → Judging.
func StartJudgingHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.StartJudging(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "StartJudging", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "judging"})
}

// FinalizeHackathon transitions Judging → Finished.
func FinalizeHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.FinalizeHackathon(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "FinalizeHackathon", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "finished"})
}

// CancelHackathon transitions to Cancelled.
func CancelHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.CancelHackathon(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "CancelHackathon", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "cancelled"})
}
```

- [ ] **Step 2: Register routes in api.go**

In `routers/api/v1/api.go`, expand the hackforger group (around line 1740):

Replace the existing hackforger group with:
```go
// HackForger API routes
m.Group("/hackforger", func() {
	// Hackathon CRUD + state transitions
	m.Get("/hackathons", hackforger_api.ListHackathons)
	m.Post("/hackathons", reqToken(), hackforger_api.CreateHackathon)
	m.Group("/hackathons/{id}", func() {
		m.Get("", hackforger_api.GetHackathon)
		m.Put("", reqToken(), hackforger_api.UpdateHackathon)
		m.Delete("", reqToken(), hackforger_api.DeleteHackathon)
		m.Post("/publish", reqToken(), hackforger_api.PublishHackathon)
		m.Post("/start", reqToken(), hackforger_api.StartHackathon)
		m.Post("/start-judging", reqToken(), hackforger_api.StartJudgingHackathon)
		m.Post("/finalize", reqToken(), hackforger_api.FinalizeHackathon)
		m.Post("/cancel", reqToken(), hackforger_api.CancelHackathon)
		// Tracks
		m.Get("/tracks", hackforger_api.ListTracks)
		m.Post("/tracks", reqToken(), hackforger_api.CreateTrack)
		m.Put("/tracks/{tid}", reqToken(), hackforger_api.UpdateTrack)
		m.Delete("/tracks/{tid}", reqToken(), hackforger_api.DeleteTrack)
		// Registrations
		m.Post("/register", reqToken(), hackforger_api.Register)
		m.Get("/registrations", hackforger_api.ListRegistrations)
		m.Put("/registrations/{rid}", reqToken(), hackforger_api.UpdateRegistration)
		// Submissions
		m.Post("/submissions", reqToken(), hackforger_api.CreateSubmission)
		m.Get("/submissions", hackforger_api.ListSubmissions)
		m.Get("/submissions/{sid}", hackforger_api.GetSubmission)
		m.Put("/submissions/{sid}", reqToken(), hackforger_api.UpdateSubmission)
		// Judges
		m.Get("/judges", hackforger_api.ListJudges)
		m.Post("/judges", reqToken(), hackforger_api.AddJudge)
		m.Delete("/judges/{uid}", reqToken(), hackforger_api.RemoveJudge)
		// Scores
		m.Post("/submissions/{sid}/score", reqToken(), hackforger_api.SubmitScore)
		m.Get("/submissions/{sid}/scores", hackforger_api.ListScores)
		// Leaderboard
		m.Get("/leaderboard", hackforger_api.GetLeaderboard)
	})

	// Keep existing stubs for other modules
	m.Get("/bounties", hackforger_api.ListBounties)
	m.Get("/grants/rounds", hackforger_api.ListGrantRounds)
	m.Get("/credits/balance", hackforger_api.GetBalance)
	m.Get("/feed", hackforger_api.GetFeed)
})
```

Note: `reqToken()` is already defined in api.go — it enforces API token authentication.

- [ ] **Step 3: Verify compilation**

Run: `go build ./routers/api/v1/...`
Expected: Will fail because Track/Registration/Submission/Judge/Score/Leaderboard handlers don't exist yet. That's expected — commit the CRUD + state transition handlers first.

Actually, this will block compilation. We need to at minimum create stub functions for all referenced handlers. Add them as stubs in new files (Task 13-15).

- [ ] **Step 4: Commit CRUD + state handlers (but don't update api.go yet — defer to after all handlers exist)**

```bash
git add routers/api/v1/hackforger/hackathon.go
git commit -m "feat(hackforger): add Hackathon CRUD and state transition API handlers"
```

---

### Task 13: Track, Registration, Submission API Handlers

**Files:**
- Create: `routers/api/v1/hackforger/hackathon_track.go`
- Create: `routers/api/v1/hackforger/hackathon_registration.go`
- Create: `routers/api/v1/hackforger/hackathon_submission.go`

- [ ] **Step 1: Write track API handlers**

File: `routers/api/v1/hackforger/hackathon_track.go`

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
)

func ListTracks(ctx *context.APIContext) {
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, tracks)
}

func CreateTrack(ctx *context.APIContext) {
	type form struct {
		Name          string  `json:"name" binding:"Required"`
		Description   string  `json:"description"`
		PrizeAmount   float64 `json:"prize_amount"`
		PrizeCurrency string  `json:"prize_currency"`
		PrizeCredits  int64   `json:"prize_credits"`
	}
	f := &form{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: f := web.GetForm(ctx).(*formType)

	t := &hackforger_model.HackathonTrack{
		HackathonID:   ctx.ParamsInt64(":id"),
		Name:          f.Name,
		Description:   f.Description,
		PrizeAmount:   f.PrizeAmount,
		PrizeCurrency: f.PrizeCurrency,
		PrizeCredits:  f.PrizeCredits,
	}
	if err := hackforger_model.CreateTrack(ctx, t); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusCreated, t)
}

func UpdateTrack(ctx *context.APIContext) {
	t, err := hackforger_model.GetTrackByID(ctx, ctx.ParamsInt64(":tid"))
	if err != nil {
		if hackforger_model.IsErrTrackNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}

	type form struct {
		Name          *string  `json:"name"`
		Description   *string  `json:"description"`
		PrizeAmount   *float64 `json:"prize_amount"`
		PrizeCurrency *string  `json:"prize_currency"`
		PrizeCredits  *int64   `json:"prize_credits"`
	}
	f := &form{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: f := web.GetForm(ctx).(*formType)
	if f.Name != nil {
		t.Name = *f.Name
	}
	if f.Description != nil {
		t.Description = *f.Description
	}
	if f.PrizeAmount != nil {
		t.PrizeAmount = *f.PrizeAmount
	}
	if f.PrizeCurrency != nil {
		t.PrizeCurrency = *f.PrizeCurrency
	}
	if f.PrizeCredits != nil {
		t.PrizeCredits = *f.PrizeCredits
	}

	if err := hackforger_model.UpdateTrack(ctx, t); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, t)
}

func DeleteTrack(ctx *context.APIContext) {
	if err := hackforger_model.DeleteTrack(ctx, ctx.ParamsInt64(":tid")); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
```

- [ ] **Step 2: Write registration API handlers**

File: `routers/api/v1/hackforger/hackathon_registration.go`

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
)

func Register(ctx *context.APIContext) {
	type form struct {
		TeamName string `json:"team_name" binding:"Required"`
		TrackID  int64  `json:"track_id"`
	}
	f := &form{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: f := web.GetForm(ctx).(*formType)

	// Verify hackathon is in Open status
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	if h.Status != hackforger_model.HackathonStatusOpen {
		ctx.Error(http.StatusBadRequest, "InvalidPhase", "hackathon is not accepting registrations")
		return
	}

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
	ctx.JSON(http.StatusCreated, r)
}

func ListRegistrations(ctx *context.APIContext) {
	opts := hackforger_model.ListRegistrationsOptions{
		HackathonID: ctx.ParamsInt64(":id"),
	}
	opts.Page = ctx.FormInt("page")
	opts.PageSize = ctx.FormInt("limit")
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 50 {
		opts.PageSize = 20
	}

	regs, count, err := hackforger_model.ListRegistrations(ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, regs)
}

func UpdateRegistration(ctx *context.APIContext) {
	type form struct {
		Status int `json:"status" binding:"Required"`
	}
	f := &form{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: f := web.GetForm(ctx).(*formType)

	rid := ctx.ParamsInt64(":rid")
	status := hackforger_model.RegistrationStatus(f.Status)

	if err := hackforger_model.UpdateRegistrationStatus(ctx, rid, status); err != nil {
		ctx.InternalServerError(err)
		return
	}

	// Publish feed event if approved
	if status == hackforger_model.RegistrationStatusApproved {
		r, err := hackforger_model.GetRegistrationByID(ctx, rid)
		if err == nil {
			h, err := hackforger_model.GetHackathonByID(ctx, r.HackathonID)
			if err == nil {
				_ = hackforger_service.PublishHackforgerAction(ctx, &hackforger_service.HackforgerActionOpts{
					ActUserID:    r.UserID,
					OpType:       hackforger_model.ActionHackathonRegistered,
					AudienceType: hackforger_service.AudienceFollowers,
					Content: hackforger_model.HackforgerActionContent{
						EntityType: "hackathon",
						EntityID:   h.ID,
						EntityName: h.Name,
						EntitySlug: h.Slug,
					},
				})
			}
		}
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "updated"})
}
```

Note: The `UpdateRegistration` handler needs to import `hackforger_service`. Add this import:
```go
import (
	hackforger_service "forgejo.org/services/hackforger"
)
```

- [ ] **Step 3: Write submission API handlers**

File: `routers/api/v1/hackforger/hackathon_submission.go`

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

func CreateSubmission(ctx *context.APIContext) {
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	if h.Status != hackforger_model.HackathonStatusHacking {
		ctx.Error(http.StatusBadRequest, "InvalidPhase", "hackathon is not accepting submissions")
		return
	}

	// Verify user is registered
	reg, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusForbidden, "NotRegistered", "user is not registered for this hackathon")
		return
	}

	type form struct {
		Title       string `json:"title" binding:"Required"`
		Description string `json:"description"`
		DemoURL     string `json:"demo_url"`
		TrackID     int64  `json:"track_id"`
		RepoID      int64  `json:"repo_id"`
	}
	f := &form{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: f := web.GetForm(ctx).(*formType)

	s := &hackforger_model.HackathonSubmission{
		HackathonID:    h.ID,
		RegistrationID: reg.ID,
		UserID:         ctx.Doer.ID,
		Title:          f.Title,
		Description:    f.Description,
		DemoURL:        f.DemoURL,
		TrackID:        f.TrackID,
		RepoID:         f.RepoID,
		Status:         hackforger_model.SubmissionStatusSubmitted,
	}
	if err := hackforger_model.CreateSubmission(ctx, s); err != nil {
		ctx.InternalServerError(err)
		return
	}

	// Feed event
	_ = hackforger_service.PublishHackforgerAction(ctx, &hackforger_service.HackforgerActionOpts{
		ActUserID:    ctx.Doer.ID,
		OpType:       hackforger_model.ActionHackathonSubmitted,
		AudienceType: hackforger_service.AudienceFollowers,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon",
			EntityID:   h.ID,
			EntityName: h.Name,
			EntitySlug: h.Slug,
			Extra:      map[string]any{"submission_id": s.ID, "title": s.Title},
		},
	})

	ctx.JSON(http.StatusCreated, s)
}

func ListSubmissions(ctx *context.APIContext) {
	opts := hackforger_model.ListSubmissionsOptions{
		HackathonID: ctx.ParamsInt64(":id"),
	}
	opts.Page = ctx.FormInt("page")
	opts.PageSize = ctx.FormInt("limit")
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 50 {
		opts.PageSize = 20
	}

	subs, count, err := hackforger_model.ListSubmissions(ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, subs)
}

func GetSubmission(ctx *context.APIContext) {
	s, err := hackforger_model.GetSubmissionByID(ctx, ctx.ParamsInt64(":sid"))
	if err != nil {
		if hackforger_model.IsErrSubmissionNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, s)
}

func UpdateSubmission(ctx *context.APIContext) {
	s, err := hackforger_model.GetSubmissionByID(ctx, ctx.ParamsInt64(":sid"))
	if err != nil {
		if hackforger_model.IsErrSubmissionNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}

	type form struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		DemoURL     *string `json:"demo_url"`
	}
	f := &form{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: f := web.GetForm(ctx).(*formType)
	if f.Title != nil {
		s.Title = *f.Title
	}
	if f.Description != nil {
		s.Description = *f.Description
	}
	if f.DemoURL != nil {
		s.DemoURL = *f.DemoURL
	}

	if err := hackforger_model.UpdateSubmission(ctx, s); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, s)
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./routers/api/v1/hackforger/...`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add routers/api/v1/hackforger/hackathon_track.go routers/api/v1/hackforger/hackathon_registration.go routers/api/v1/hackforger/hackathon_submission.go
git commit -m "feat(hackforger): add Track, Registration, Submission API handlers"
```

---

### Task 14: Judge, Score, Leaderboard API Handlers

**Files:**
- Create: `routers/api/v1/hackforger/hackathon_judge.go`

- [ ] **Step 1: Write judge + score + leaderboard handlers**

File: `routers/api/v1/hackforger/hackathon_judge.go`

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

func ListJudges(ctx *context.APIContext) {
	judges, err := hackforger_model.ListJudges(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, judges)
}

func AddJudge(ctx *context.APIContext) {
	type form struct {
		UserID int64 `json:"user_id" binding:"Required"`
	}
	f := &form{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: f := web.GetForm(ctx).(*formType)

	if err := hackforger_model.AddJudge(ctx, ctx.ParamsInt64(":id"), f.UserID); err != nil {
		if hackforger_model.IsErrDuplicateJudge(err) {
			ctx.Error(http.StatusConflict, "DuplicateJudge", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusCreated)
}

func RemoveJudge(ctx *context.APIContext) {
	if err := hackforger_model.RemoveJudge(ctx, ctx.ParamsInt64(":id"), ctx.ParamsInt64(":uid")); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func SubmitScore(ctx *context.APIContext) {
	type form struct {
		Score   float64 `json:"score"`
		Comment string  `json:"comment"`
	}
	f := &form{}
	// NOTE: Form binding is handled by bind() middleware in route registration.
	// Handler receives the form via: f := web.GetForm(ctx).(*formType)

	if err := hackforger_service.SubmitScore(ctx, ctx.Doer.ID, ctx.ParamsInt64(":sid"), f.Score, f.Comment); err != nil {
		ctx.Error(http.StatusBadRequest, "SubmitScore", err)
		return
	}
	ctx.Status(http.StatusCreated)
}

func ListScores(ctx *context.APIContext) {
	scores, err := hackforger_model.ListScoresBySubmission(ctx, ctx.ParamsInt64(":sid"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, scores)
}

func GetLeaderboard(ctx *context.APIContext) {
	opts := hackforger_model.ListSubmissionsOptions{
		HackathonID: ctx.ParamsInt64(":id"),
	}
	// Get all submissions, no pagination for leaderboard
	opts.Page = 1
	opts.PageSize = 100

	subs, _, err := hackforger_model.ListSubmissions(ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	// Sort by rank (already set during finalize)
	// Return as-is — they should already be ranked
	type entry struct {
		Rank       int     `json:"rank"`
		Title      string  `json:"title"`
		UserID     int64   `json:"user_id"`
		TotalScore float64 `json:"total_score"`
		DemoURL    string  `json:"demo_url"`
	}

	result := make([]entry, 0, len(subs))
	for _, s := range subs {
		result = append(result, entry{
			Rank:       s.Rank,
			Title:      s.Title,
			UserID:     s.UserID,
			TotalScore: s.TotalScore,
			DemoURL:    s.DemoURL,
		})
	}
	ctx.JSON(http.StatusOK, result)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./routers/api/v1/hackforger/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add routers/api/v1/hackforger/hackathon_judge.go
git commit -m "feat(hackforger): add Judge, Score, Leaderboard API handlers"
```

---

### Task 15: Register All API Routes in api.go

**Files:**
- Modify: `routers/api/v1/api.go`

- [ ] **Step 1: Update the hackforger route group**

Find the existing hackforger group (around line 1739-1746) and replace with the full route tree from Task 12 Step 2.

- [ ] **Step 2: Verify compilation**

Run: `go build ./routers/api/v1/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add routers/api/v1/api.go
git commit -m "feat(hackforger): register all Hackathon API routes"
```

---

## Chunk 4: Web Layer (Templates + Routes)

### Task 16: i18n Keys

**Files:**
- Modify: `options/locale/locale_en-US.ini`

- [ ] **Step 1: Add hackathon-specific i18n keys**

Append to the `[hackforger]` section:

```ini
hackathon.create = Create Hackathon
hackathon.edit = Edit Hackathon
hackathon.delete_confirm = Are you sure you want to delete this hackathon?
hackathon.field.name = Name
hackathon.field.slug = Slug
hackathon.field.description = Description
hackathon.field.org = Organization
hackathon.field.max_team_size = Max Team Size
hackathon.field.prize_summary = Prize Summary
hackathon.field.registration_start = Registration Start
hackathon.field.registration_end = Registration End
hackathon.field.hacking_start = Hacking Start
hackathon.field.hacking_end = Hacking End
hackathon.field.judging_end = Judging End
hackathon.register = Register
hackathon.register.success = Successfully registered!
hackathon.register.already_registered = You are already registered for this hackathon.
hackathon.submit = Submit Project
hackathon.submit.success = Project submitted successfully!
hackathon.submit.field.title = Project Title
hackathon.submit.field.demo_url = Demo URL
hackathon.submit.field.description = Description
hackathon.submit.field.track = Track
hackathon.manage = Manage Hackathon
hackathon.manage.tracks = Tracks
hackathon.manage.tracks.add = Add Track
hackathon.manage.registrations = Registrations
hackathon.manage.judges = Judges
hackathon.manage.judges.add = Add Judge
hackathon.manage.phase.publish = Publish
hackathon.manage.phase.start = Start Hacking
hackathon.manage.phase.start_judging = Start Judging
hackathon.manage.phase.finalize = Finalize
hackathon.manage.phase.cancel = Cancel
hackathon.judge = Judge
hackathon.judge.score = Score (0-10)
hackathon.judge.comment = Comment
hackathon.judge.submit_score = Submit Score
hackathon.leaderboard = Leaderboard
hackathon.leaderboard.rank = Rank
hackathon.leaderboard.team = Team / Project
hackathon.leaderboard.score = Score
hackathon.leaderboard.track = Track
hackathon.error.not_found = Hackathon not found.
hackathon.error.invalid_phase = This action is not available in the current phase.
hackathon.error.no_permission = You do not have permission to perform this action.
hackathon.no_hackathons = No hackathons found.
hackathon.status.cancelled = Cancelled
```

- [ ] **Step 2: Commit**

```bash
git add options/locale/locale_en-US.ini
git commit -m "i18n(hackforger): add hackathon web page i18n keys"
```

---

### Task 17: Web Templates

**Files:**
- Create: `templates/hackforger/hackathon/new.tmpl`
- Create: `templates/hackforger/hackathon/view.tmpl`
- Create: `templates/hackforger/hackathon/submit.tmpl`
- Create: `templates/hackforger/hackathon/manage.tmpl`
- Create: `templates/hackforger/hackathon/judge.tmpl`
- Create: `templates/hackforger/hackathon/leaderboard.tmpl`
- Modify: `templates/hackforger/explore.tmpl`

- [ ] **Step 1: Create hackathon/new.tmpl**

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content hackathon">
	<div class="ui container">
		<h2>{{ctx.Locale.Tr "hackforger.hackathon.create"}}</h2>
		<form class="ui form" method="post" action="{{AppSubUrl}}/hackathons/new">
			{{.CsrfTokenHtml}}
			<div class="required field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.field.name"}}</label>
				<input name="name" required>
			</div>
			<div class="required field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.field.slug"}}</label>
				<input name="slug" required pattern="[a-z0-9\-]+">
			</div>
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.field.description"}}</label>
				<textarea name="description" rows="5"></textarea>
			</div>
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.field.org"}}</label>
				<input name="org_id" type="number" value="0">
			</div>
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.field.max_team_size"}}</label>
				<input name="max_team_size" type="number" value="5" min="1">
			</div>
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.field.prize_summary"}}</label>
				<textarea name="prize_summary" rows="3"></textarea>
			</div>
			<button class="ui primary button" type="submit">{{ctx.Locale.Tr "hackforger.hackathon.create"}}</button>
		</form>
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 2: Create hackathon/view.tmpl**

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Hackathon.Name}}" class="page-content hackathon">
	<div class="ui container">
		<h1>{{.Hackathon.Name}}</h1>
		<div class="ui label {{if eq .Hackathon.Status 1}}green{{else if eq .Hackathon.Status 2}}blue{{else if eq .Hackathon.Status 3}}orange{{else if eq .Hackathon.Status 4}}grey{{end}}">
			{{if eq .Hackathon.Status 0}}{{ctx.Locale.Tr "hackforger.hackathon.status.draft"}}
			{{else if eq .Hackathon.Status 1}}{{ctx.Locale.Tr "hackforger.hackathon.status.registration"}}
			{{else if eq .Hackathon.Status 2}}{{ctx.Locale.Tr "hackforger.hackathon.status.hacking"}}
			{{else if eq .Hackathon.Status 3}}{{ctx.Locale.Tr "hackforger.hackathon.status.judging"}}
			{{else if eq .Hackathon.Status 4}}{{ctx.Locale.Tr "hackforger.hackathon.status.finished"}}
			{{else}}{{ctx.Locale.Tr "hackforger.hackathon.status.cancelled"}}{{end}}
		</div>

		{{if .Hackathon.Description}}
		<div class="ui segment">
			<p>{{.Hackathon.Description}}</p>
		</div>
		{{end}}

		{{if .Hackathon.PrizeSummary}}
		<div class="ui segment">
			<h3>Prizes</h3>
			<p>{{.Hackathon.PrizeSummary}}</p>
		</div>
		{{end}}

		{{/* Tracks */}}
		{{if .Tracks}}
		<h3>{{ctx.Locale.Tr "hackforger.hackathon.manage.tracks"}}</h3>
		<div class="ui divided list">
			{{range .Tracks}}
			<div class="item">
				<div class="content">
					<div class="header">{{.Name}}</div>
					{{if .Description}}<div class="description">{{.Description}}</div>{{end}}
				</div>
			</div>
			{{end}}
		</div>
		{{end}}

		{{/* Registration button (Open phase only) */}}
		{{if and (eq .Hackathon.Status 1) .SignedUserID (not .IsRegistered)}}
		<form method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/register">
			{{.CsrfTokenHtml}}
			<div class="ui form">
				<div class="required field">
					<label>Team Name</label>
					<input name="team_name" required>
				</div>
				<button class="ui green button" type="submit">{{ctx.Locale.Tr "hackforger.hackathon.register"}}</button>
			</div>
		</form>
		{{end}}

		{{/* Submissions list */}}
		{{if .Submissions}}
		<h3>Submissions ({{len .Submissions}})</h3>
		<table class="ui table">
			<thead><tr><th>Project</th><th>Demo</th><th>Score</th><th>Rank</th></tr></thead>
			<tbody>
			{{range .Submissions}}
			<tr>
				<td>{{.Title}}</td>
				<td>{{if .DemoURL}}<a href="{{.DemoURL}}">Demo</a>{{end}}</td>
				<td>{{if gt .TotalScore 0.0}}{{printf "%.1f" .TotalScore}}{{else}}-{{end}}</td>
				<td>{{if gt .Rank 0}}#{{.Rank}}{{else}}-{{end}}</td>
			</tr>
			{{end}}
			</tbody>
		</table>
		{{end}}

		{{/* Submit link (Hacking phase, registered users) */}}
		{{if and (eq .Hackathon.Status 2) .IsRegistered}}
		<a class="ui primary button" href="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/submit">
			{{ctx.Locale.Tr "hackforger.hackathon.submit"}}
		</a>
		{{end}}

		{{/* Leaderboard link */}}
		{{if eq .Hackathon.Status 4}}
		<a class="ui button" href="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/leaderboard">
			{{ctx.Locale.Tr "hackforger.hackathon.leaderboard"}}
		</a>
		{{end}}
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 3: Create hackathon/submit.tmpl**

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content hackathon">
	<div class="ui container">
		<h2>{{ctx.Locale.Tr "hackforger.hackathon.submit"}} — {{.Hackathon.Name}}</h2>
		<form class="ui form" method="post">
			{{.CsrfTokenHtml}}
			<div class="required field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.submit.field.title"}}</label>
				<input name="title" required>
			</div>
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.submit.field.description"}}</label>
				<textarea name="description" rows="5"></textarea>
			</div>
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.submit.field.demo_url"}}</label>
				<input name="demo_url" type="url">
			</div>
			{{if .Tracks}}
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.hackathon.submit.field.track"}}</label>
				<select name="track_id" class="ui dropdown">
					<option value="0">— None —</option>
					{{range .Tracks}}
					<option value="{{.ID}}">{{.Name}}</option>
					{{end}}
				</select>
			</div>
			{{end}}
			<button class="ui primary button" type="submit">{{ctx.Locale.Tr "hackforger.hackathon.submit"}}</button>
		</form>
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 4: Create hackathon/manage.tmpl**

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content hackathon">
	<div class="ui container">
		<h2>{{ctx.Locale.Tr "hackforger.hackathon.manage"}} — {{.Hackathon.Name}}</h2>

		{{/* Phase control buttons */}}
		<div class="ui segment">
			<h3>Phase Control</h3>
			<span class="ui label">Current: {{.StatusLabel}}</span>
			<div class="tw-mt-2">
			{{if eq .Hackathon.Status 0}}
				<form method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/publish" class="tw-inline">
					{{.CsrfTokenHtml}}
					<button class="ui green button">{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.publish"}}</button>
				</form>
			{{else if eq .Hackathon.Status 1}}
				<form method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/start" class="tw-inline">
					{{.CsrfTokenHtml}}
					<button class="ui blue button">{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.start"}}</button>
				</form>
			{{else if eq .Hackathon.Status 2}}
				<form method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/start-judging" class="tw-inline">
					{{.CsrfTokenHtml}}
					<button class="ui orange button">{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.start_judging"}}</button>
				</form>
			{{else if eq .Hackathon.Status 3}}
				<form method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/finalize" class="tw-inline">
					{{.CsrfTokenHtml}}
					<button class="ui purple button">{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.finalize"}}</button>
				</form>
			{{end}}
			{{if and (ne .Hackathon.Status 4) (ne .Hackathon.Status 5)}}
				<form method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/cancel" class="tw-inline">
					{{.CsrfTokenHtml}}
					<button class="ui red button">{{ctx.Locale.Tr "hackforger.hackathon.manage.phase.cancel"}}</button>
				</form>
			{{end}}
			</div>
		</div>

		{{/* Tracks */}}
		<div class="ui segment">
			<h3>{{ctx.Locale.Tr "hackforger.hackathon.manage.tracks"}}</h3>
			{{range .Tracks}}
			<div class="ui item">{{.Name}}</div>
			{{end}}
			{{if eq .Hackathon.Status 0}}
			<form class="ui form tw-mt-2" method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/tracks">
				{{.CsrfTokenHtml}}
				<div class="inline fields">
					<div class="field"><input name="name" placeholder="Track name" required></div>
					<button class="ui small button" type="submit">{{ctx.Locale.Tr "hackforger.hackathon.manage.tracks.add"}}</button>
				</div>
			</form>
			{{end}}
		</div>

		{{/* Registrations */}}
		<div class="ui segment">
			<h3>{{ctx.Locale.Tr "hackforger.hackathon.manage.registrations"}} ({{len .Registrations}})</h3>
			<table class="ui table">
				<thead><tr><th>Team</th><th>Status</th><th>Action</th></tr></thead>
				<tbody>
				{{range .Registrations}}
				<tr>
					<td>{{.TeamName}}</td>
					<td>{{if eq .Status 0}}Pending{{else if eq .Status 1}}Approved{{else}}Rejected{{end}}</td>
					<td>
						{{if eq .Status 0}}
						<form method="post" action="{{AppSubUrl}}/hackathon/{{$.Hackathon.Slug}}/manage/registrations/{{.ID}}" class="tw-inline">
							{{$.CsrfTokenHtml}}
							<input type="hidden" name="status" value="1">
							<button class="ui mini green button">Approve</button>
						</form>
						<form method="post" action="{{AppSubUrl}}/hackathon/{{$.Hackathon.Slug}}/manage/registrations/{{.ID}}" class="tw-inline">
							{{$.CsrfTokenHtml}}
							<input type="hidden" name="status" value="2">
							<button class="ui mini red button">Reject</button>
						</form>
						{{end}}
					</td>
				</tr>
				{{end}}
				</tbody>
			</table>
		</div>

		{{/* Judges */}}
		<div class="ui segment">
			<h3>{{ctx.Locale.Tr "hackforger.hackathon.manage.judges"}} ({{len .Judges}})</h3>
			{{range .Judges}}
			<div class="ui item">User #{{.UserID}}
				<form method="post" action="{{AppSubUrl}}/hackathon/{{$.Hackathon.Slug}}/manage/judges/{{.UserID}}/remove" class="tw-inline">
					{{$.CsrfTokenHtml}}
					<button class="ui mini red button">Remove</button>
				</form>
			</div>
			{{end}}
			<form class="ui form tw-mt-2" method="post" action="{{AppSubUrl}}/hackathon/{{.Hackathon.Slug}}/manage/judges">
				{{.CsrfTokenHtml}}
				<div class="inline fields">
					<div class="field"><input name="user_id" type="number" placeholder="User ID" required></div>
					<button class="ui small button" type="submit">{{ctx.Locale.Tr "hackforger.hackathon.manage.judges.add"}}</button>
				</div>
			</form>
		</div>
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 5: Create hackathon/judge.tmpl**

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content hackathon">
	<div class="ui container">
		<h2>{{ctx.Locale.Tr "hackforger.hackathon.judge"}} — {{.Hackathon.Name}}</h2>
		{{range .Submissions}}
		<div class="ui segment">
			<h4>{{.Title}}</h4>
			{{if .DemoURL}}<p><a href="{{.DemoURL}}">Demo</a></p>{{end}}
			{{if .Description}}<p>{{.Description}}</p>{{end}}
			<form class="ui form" method="post" action="{{AppSubUrl}}/hackathon/{{$.Hackathon.Slug}}/judge/{{.ID}}/score">
				{{$.CsrfTokenHtml}}
				<div class="inline fields">
					<div class="field">
						<label>{{ctx.Locale.Tr "hackforger.hackathon.judge.score"}}</label>
						<input name="score" type="number" step="0.1" min="0" max="10" required>
					</div>
					<div class="field">
						<label>{{ctx.Locale.Tr "hackforger.hackathon.judge.comment"}}</label>
						<input name="comment">
					</div>
					<button class="ui primary button" type="submit">{{ctx.Locale.Tr "hackforger.hackathon.judge.submit_score"}}</button>
				</div>
			</form>
		</div>
		{{end}}
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 6: Create hackathon/leaderboard.tmpl**

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content hackathon">
	<div class="ui container">
		<h2>{{ctx.Locale.Tr "hackforger.hackathon.leaderboard"}} — {{.Hackathon.Name}}</h2>
		<table class="ui celled table">
			<thead>
				<tr>
					<th>{{ctx.Locale.Tr "hackforger.hackathon.leaderboard.rank"}}</th>
					<th>{{ctx.Locale.Tr "hackforger.hackathon.leaderboard.team"}}</th>
					<th>{{ctx.Locale.Tr "hackforger.hackathon.leaderboard.score"}}</th>
				</tr>
			</thead>
			<tbody>
			{{range .Submissions}}
			<tr {{if eq .Rank 1}}class="positive"{{end}}>
				<td>#{{.Rank}}</td>
				<td>{{.Title}}{{if .DemoURL}} — <a href="{{.DemoURL}}">Demo</a>{{end}}</td>
				<td>{{printf "%.1f" .TotalScore}}</td>
			</tr>
			{{end}}
			</tbody>
		</table>
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 7: Update explore.tmpl to show hackathon listing**

Replace `templates/hackforger/explore.tmpl` content:

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content explore">
	{{template "explore/navbar" .}}
	<div class="ui container">
		{{if .PageIsExploreHackathons}}
		<h2>{{ctx.Locale.Tr "hackforger.explore.hackathons"}}</h2>
		{{if .Hackathons}}
		<div class="ui divided items">
			{{range .Hackathons}}
			<div class="item">
				<div class="content">
					<a class="header" href="{{AppSubUrl}}/hackathon/{{.Slug}}">{{.Name}}</a>
					<div class="meta">
						<span class="ui small label">{{if eq .Status 0}}Draft{{else if eq .Status 1}}Open{{else if eq .Status 2}}Hacking{{else if eq .Status 3}}Judging{{else if eq .Status 4}}Finished{{else}}Cancelled{{end}}</span>
					</div>
					{{if .Description}}<div class="description">{{.Description}}</div>{{end}}
				</div>
			</div>
			{{end}}
		</div>
		{{else}}
		<p>{{ctx.Locale.Tr "hackforger.hackathon.no_hackathons"}}</p>
		{{end}}
		{{else if .PageIsExploreBounties}}
		<h2>{{ctx.Locale.Tr "hackforger.explore.bounties"}}</h2>
		<p>{{ctx.Locale.Tr "hackforger.explore.coming_soon"}}</p>
		{{else if .PageIsExploreGrants}}
		<h2>{{ctx.Locale.Tr "hackforger.explore.grants"}}</h2>
		<p>{{ctx.Locale.Tr "hackforger.explore.coming_soon"}}</p>
		{{end}}
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 8: Commit templates**

```bash
git add templates/hackforger/
git commit -m "feat(hackforger): add Hackathon web templates"
```

---

### Task 18: Web Route Handlers

**Files:**
- Modify: `routers/web/hackforger/hackathon.go`
- Modify: `routers/web/web.go`

- [ ] **Step 1: Rewrite web hackathon handlers**

Replace `routers/web/hackforger/hackathon.go` with complete handlers for all web routes:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"
	"strconv"
	"strings"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

const (
	tplExplore     = "hackforger/explore"
	tplNew         = "hackforger/hackathon/new"
	tplView        = "hackforger/hackathon/view"
	tplSubmit      = "hackforger/hackathon/submit"
	tplManage      = "hackforger/hackathon/manage"
	tplJudge       = "hackforger/hackathon/judge"
	tplLeaderboard = "hackforger/hackathon/leaderboard"
)

// ExploreHackathons renders the hackathon explore page.
func ExploreHackathons(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.hackathons")
	ctx.Data["PageIsExploreHackathons"] = true

	hackathons, _, err := hackforger_model.ListHackathons(ctx, hackforger_model.ListHackathonsOptions{
		Keyword: ctx.FormString("q"),
	})
	if err != nil {
		ctx.ServerError("ListHackathons", err)
		return
	}
	ctx.Data["Hackathons"] = hackathons
	ctx.HTML(http.StatusOK, tplExplore)
}

// ExploreBounties renders the bounty explore page.
func ExploreBounties(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.bounties")
	ctx.Data["PageIsExploreBounties"] = true
	ctx.HTML(http.StatusOK, tplExplore)
}

// ExploreGrants renders the grants explore page.
func ExploreGrants(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.grants")
	ctx.Data["PageIsExploreGrants"] = true
	ctx.HTML(http.StatusOK, tplExplore)
}

// NewHackathon renders the create hackathon form.
func NewHackathon(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.create")
	ctx.HTML(http.StatusOK, tplNew)
}

// NewHackathonPost handles hackathon creation form submission.
func NewHackathonPost(ctx *context.Context) {
	maxTeamSize, _ := strconv.Atoi(ctx.FormString("max_team_size"))
	if maxTeamSize <= 0 {
		maxTeamSize = 5
	}
	orgID, _ := strconv.ParseInt(ctx.FormString("org_id"), 10, 64)

	h := &hackforger_model.Hackathon{
		OrgID:        orgID,
		OwnerID:      ctx.Doer.ID,
		Name:         ctx.FormString("name"),
		Slug:         ctx.FormString("slug"),
		Description:  ctx.FormString("description"),
		PrizeSummary: ctx.FormString("prize_summary"),
		MaxTeamSize:  maxTeamSize,
	}
	if err := hackforger_service.CreateHackathon(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.create")
		ctx.RenderWithErr(err.Error(), tplNew, nil)
		return
	}
	ctx.Redirect("/hackathon/" + h.Slug)
}

// loadHackathon is a helper to load a hackathon by slug from the URL.
func loadHackathon(ctx *context.Context) *hackforger_model.Hackathon {
	h, err := hackforger_model.GetHackathonBySlug(ctx, ctx.Params(":slug"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound(ctx.Tr("hackforger.hackathon.error.not_found"), nil)
		} else {
			ctx.ServerError("GetHackathonBySlug", err)
		}
		return nil
	}
	return h
}

// ViewHackathon renders the hackathon detail page.
func ViewHackathon(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	ctx.Data["Title"] = h.Name
	ctx.Data["Hackathon"] = h

	tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	ctx.Data["Tracks"] = tracks

	subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{HackathonID: h.ID})
	ctx.Data["Submissions"] = subs

	if ctx.Doer != nil {
		_, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID)
		ctx.Data["IsRegistered"] = err == nil
		ctx.Data["SignedUserID"] = ctx.Doer.ID
	}

	ctx.HTML(http.StatusOK, tplView)
}

// RegisterPost handles hackathon registration.
func RegisterPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	if h.Status != hackforger_model.HackathonStatusOpen {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}

	r := &hackforger_model.HackathonRegistration{
		HackathonID: h.ID,
		UserID:      ctx.Doer.ID,
		TeamName:    ctx.FormString("team_name"),
	}
	if err := hackforger_model.CreateRegistration(ctx, r); err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}
	ctx.Flash.Success(ctx.Tr("hackforger.hackathon.register.success"))
	ctx.Redirect("/hackathon/" + h.Slug)
}

// SubmitForm renders the submission form.
func SubmitForm(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.submit")
	ctx.Data["Hackathon"] = h

	tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	ctx.Data["Tracks"] = tracks

	ctx.HTML(http.StatusOK, tplSubmit)
}

// SubmitPost handles project submission.
func SubmitPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	if h.Status != hackforger_model.HackathonStatusHacking {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}

	reg, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID)
	if err != nil {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.no_permission"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}

	trackID, _ := strconv.ParseInt(ctx.FormString("track_id"), 10, 64)
	s := &hackforger_model.HackathonSubmission{
		HackathonID:    h.ID,
		RegistrationID: reg.ID,
		UserID:         ctx.Doer.ID,
		Title:          ctx.FormString("title"),
		Description:    ctx.FormString("description"),
		DemoURL:        ctx.FormString("demo_url"),
		TrackID:        trackID,
		Status:         hackforger_model.SubmissionStatusSubmitted,
	}
	if err := hackforger_model.CreateSubmission(ctx, s); err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect("/hackathon/" + h.Slug + "/submit")
		return
	}
	ctx.Flash.Success(ctx.Tr("hackforger.hackathon.submit.success"))
	ctx.Redirect("/hackathon/" + h.Slug)
}

// ManageHackathon renders the organizer management page.
func ManageHackathon(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.manage")
	ctx.Data["Hackathon"] = h
	ctx.Data["StatusLabel"] = hackforger_service.HackathonStatusLabel(h.Status)

	tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	ctx.Data["Tracks"] = tracks

	regs, _, _ := hackforger_model.ListRegistrations(ctx, hackforger_model.ListRegistrationsOptions{HackathonID: h.ID})
	ctx.Data["Registrations"] = regs

	judges, _ := hackforger_model.ListJudges(ctx, h.ID)
	ctx.Data["Judges"] = judges

	ctx.HTML(http.StatusOK, tplManage)
}

// ManagePhasePost handles phase transition POST requests from the manage page.
func ManagePhasePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}

	// Extract the last path segment as the action (e.g., /manage/publish → "publish")
	path := ctx.Req.URL.Path
	segments := strings.Split(strings.TrimRight(path, "/"), "/")
	action := segments[len(segments)-1]
	var err error
	switch action {
	case "publish":
		err = hackforger_service.PublishHackathon(ctx, ctx.Doer.ID, h)
	case "start":
		err = hackforger_service.StartHacking(ctx, ctx.Doer.ID, h)
	case "start-judging":
		err = hackforger_service.StartJudging(ctx, ctx.Doer.ID, h)
	case "finalize":
		err = hackforger_service.FinalizeHackathon(ctx, ctx.Doer.ID, h)
	case "cancel":
		err = hackforger_service.CancelHackathon(ctx, ctx.Doer.ID, h)
	default:
		ctx.NotFound("Unknown action", nil)
		return
	}
	if err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

// ManageTrackPost handles track creation from the manage page.
func ManageTrackPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	t := &hackforger_model.HackathonTrack{
		HackathonID: h.ID,
		Name:        ctx.FormString("name"),
	}
	if err := hackforger_model.CreateTrack(ctx, t); err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

// ManageRegistrationPost handles registration approval/rejection.
func ManageRegistrationPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	rid := ctx.ParamsInt64(":rid")
	status, _ := strconv.Atoi(ctx.FormString("status"))
	if err := hackforger_model.UpdateRegistrationStatus(ctx, rid, hackforger_model.RegistrationStatus(status)); err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

// ManageJudgePost handles adding a judge.
func ManageJudgePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	userID, _ := strconv.ParseInt(ctx.FormString("user_id"), 10, 64)
	if err := hackforger_model.AddJudge(ctx, h.ID, userID); err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

// ManageJudgeRemovePost handles removing a judge.
func ManageJudgeRemovePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	userID := ctx.ParamsInt64(":uid")
	if err := hackforger_model.RemoveJudge(ctx, h.ID, userID); err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

// JudgePage renders the judge scoring page.
func JudgePage(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.judge")
	ctx.Data["Hackathon"] = h

	subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{HackathonID: h.ID})
	ctx.Data["Submissions"] = subs

	ctx.HTML(http.StatusOK, tplJudge)
}

// JudgeScorePost handles score submission from the judge page.
func JudgeScorePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	sid := ctx.ParamsInt64(":sid")
	score, _ := strconv.ParseFloat(ctx.FormString("score"), 64)
	comment := ctx.FormString("comment")

	if err := hackforger_service.SubmitScore(ctx, ctx.Doer.ID, sid, score, comment); err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success("Score submitted")
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/judge")
}

// Leaderboard renders the public leaderboard.
func Leaderboard(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.leaderboard")
	ctx.Data["Hackathon"] = h

	subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{HackathonID: h.ID})
	ctx.Data["Submissions"] = subs

	ctx.HTML(http.StatusOK, tplLeaderboard)
}
```

- [ ] **Step 2: Add HackathonStatusLabel helper to services/hackforger/hackathon.go**

Append to `services/hackforger/hackathon.go`:

```go
// HackathonStatusLabel returns a human-readable status label.
func HackathonStatusLabel(status hackforger_model.HackathonStatus) string {
	labels := map[hackforger_model.HackathonStatus]string{
		hackforger_model.HackathonStatusDraft:     "Draft",
		hackforger_model.HackathonStatusOpen:      "Open (Registration)",
		hackforger_model.HackathonStatusHacking:   "Hacking",
		hackforger_model.HackathonStatusJudging:   "Judging",
		hackforger_model.HackathonStatusFinished:  "Finished",
		hackforger_model.HackathonStatusCancelled: "Cancelled",
	}
	if l, ok := labels[status]; ok {
		return l
	}
	return "Unknown"
}
```

- [ ] **Step 3: Register web routes in web.go**

In `routers/web/web.go`, after the existing explore group (line ~507), add the hackathon route group:

```go
// HackForger: Hackathon web routes
m.Group("", func() {
	m.Get("/hackathons/new", hackforger_web.NewHackathon)
	m.Post("/hackathons/new", web.Bind(forms.EmptyForm{}), hackforger_web.NewHackathonPost)
	m.Get("/hackathon/{slug}", hackforger_web.ViewHackathon)
	m.Post("/hackathon/{slug}/register", hackforger_web.RegisterPost)
	m.Get("/hackathon/{slug}/submit", hackforger_web.SubmitForm)
	m.Post("/hackathon/{slug}/submit", hackforger_web.SubmitPost)
	m.Get("/hackathon/{slug}/leaderboard", hackforger_web.Leaderboard)
	m.Group("/hackathon/{slug}/manage", func() {
		m.Get("", hackforger_web.ManageHackathon)
		m.Post("/publish", hackforger_web.ManagePhasePost)
		m.Post("/start", hackforger_web.ManagePhasePost)
		m.Post("/start-judging", hackforger_web.ManagePhasePost)
		m.Post("/finalize", hackforger_web.ManagePhasePost)
		m.Post("/cancel", hackforger_web.ManagePhasePost)
		m.Post("/tracks", hackforger_web.ManageTrackPost)
		m.Post("/registrations/{rid}", hackforger_web.ManageRegistrationPost)
		m.Post("/judges", hackforger_web.ManageJudgePost)
		m.Post("/judges/{uid}/remove", hackforger_web.ManageJudgeRemovePost)
	})
	m.Group("/hackathon/{slug}/judge", func() {
		m.Get("", hackforger_web.JudgePage)
		m.Post("/{sid}/score", hackforger_web.JudgeScorePost)
	})
}, reqSignIn)
```

Note: `reqSignIn` applies to the entire group. The public routes (`/hackathon/{slug}` view and `/hackathon/{slug}/leaderboard`) should NOT require sign-in. Move them outside:

```go
// HackForger: public hackathon routes
m.Get("/hackathon/{slug}", hackforger_web.ViewHackathon)
m.Get("/hackathon/{slug}/leaderboard", hackforger_web.Leaderboard)

// HackForger: authenticated hackathon routes
m.Group("", func() {
	m.Get("/hackathons/new", hackforger_web.NewHackathon)
	m.Post("/hackathons/new", hackforger_web.NewHackathonPost)
	m.Post("/hackathon/{slug}/register", hackforger_web.RegisterPost)
	m.Get("/hackathon/{slug}/submit", hackforger_web.SubmitForm)
	m.Post("/hackathon/{slug}/submit", hackforger_web.SubmitPost)
	m.Group("/hackathon/{slug}/manage", func() {
		m.Get("", hackforger_web.ManageHackathon)
		m.Post("/publish", hackforger_web.ManagePhasePost)
		m.Post("/start", hackforger_web.ManagePhasePost)
		m.Post("/start-judging", hackforger_web.ManagePhasePost)
		m.Post("/finalize", hackforger_web.ManagePhasePost)
		m.Post("/cancel", hackforger_web.ManagePhasePost)
		m.Post("/tracks", hackforger_web.ManageTrackPost)
		m.Post("/registrations/{rid}", hackforger_web.ManageRegistrationPost)
		m.Post("/judges", hackforger_web.ManageJudgePost)
		m.Post("/judges/{uid}/remove", hackforger_web.ManageJudgeRemovePost)
	})
	m.Group("/hackathon/{slug}/judge", func() {
		m.Get("", hackforger_web.JudgePage)
		m.Post("/{sid}/score", hackforger_web.JudgeScorePost)
	})
}, reqSignIn)
```

- [ ] **Step 4: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./...`
Expected: No errors (full project build)

- [ ] **Step 5: Commit**

```bash
git add routers/web/hackforger/hackathon.go routers/web/web.go services/hackforger/hackathon.go
git commit -m "feat(hackforger): add Hackathon web routes, handlers, and templates"
```

---

## Chunk 5: Integration + Verification

### Task 19: Full Build + Test Verification

- [ ] **Step 1: Run full backend build**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-hackathon && TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Build succeeds

- [ ] **Step 2: Run all hackforger tests**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go test ./models/hackforger/... ./services/hackforger/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 3: Run linting**

Run: `make lint-go-vet 2>&1 | head -50`
Expected: No errors in hackforger files

- [ ] **Step 4: Verify templates compile (start server briefly)**

Run: `./gitea web --port 3333 &` then `sleep 3 && curl -s http://localhost:3333/explore/hackathons | head -5` then kill the process.
Expected: HTML response with hackathon explore page

---

### Task 20: Manual E2E Test Prompt

After all implementation is complete, execute this manual E2E test on the internal instance.

**Instance:** https://hackforger.inside.h2os.cloud
**Credentials:** hackforger / admin1234

- [ ] **Step 1:** Log in as admin, create org `test-org` if not exists
- [ ] **Step 2:** Create hackathon "Spring Hack 2026" with slug `spring-hack-2026`
- [ ] **Step 3:** Add 2 tracks: "AI Track", "Web3 Track"
- [ ] **Step 4:** Publish hackathon → verify status becomes Open on explore page
- [ ] **Step 5:** Create test user `hacker_eve`, register for hackathon
- [ ] **Step 6:** As admin, approve registration, advance to Hacking
- [ ] **Step 7:** As `hacker_eve`, submit project with demo URL
- [ ] **Step 8:** As admin, advance to Judging
- [ ] **Step 9:** As admin (or judge user), score the submission
- [ ] **Step 10:** As admin, finalize → verify leaderboard shows rank #1
- [ ] **Step 11:** Check Feed API: `GET /api/v1/hackforger/feed?type=global` — verify events exist
- [ ] **Step 12:** Write results to `docs/tests/e2e/phase1-hackathon-e2e-report.md`

Report template:
```markdown
# Phase 1 Hackathon E2E Report

**Date:** YYYY-MM-DD
**Tester:**
**Instance:** https://hackforger.inside.h2os.cloud

| Step | Description | Result | Notes |
|------|-------------|--------|-------|
| 1 | Create org | PASS/FAIL | |
| 2 | Create hackathon | PASS/FAIL | |
| ... | ... | ... | ... |

## Feed Events Verified
- [ ] HackathonCreated (30)
- [ ] HackathonRegistered (31)
- [ ] HackathonSubmitted (32)
- [ ] HackathonScored (33)
- [ ] HackathonPhaseChanged (50)
- [ ] HackathonFinalized (51)
```
