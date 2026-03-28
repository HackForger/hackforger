# Phase 1 Bounty Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the complete Bounty module — model corrections, 7-state state machine, full API, web UI, Vue panel, Feed integration, and E2E tests.

**Architecture:** Bottom-up approach: Model → Service → Notifier → API → Web → Vue → Tests. Bounty line also owns the shared Notifier audience resolution (bitmask rewrite) used by all three parallel lines.

**Tech Stack:** Go (XORM, go-chi router), Go templates (SSR), Vue 3 Options API, Fomantic UI, SQLite for tests.

**Spec:** `docs/superpowers/specs/2026-03-28-phase1-bounty-design.md`

---

## File Map

### New Files

| File | Responsibility |
|------|---------------|
| `models/forgejo_migrations/v14c_fix-bounty-status-mode.go` | Migration: remap BountyStatus 3→6, delete Mode=2 rows (runs after v14c_add-hackforger-tables due to alphabetical ordering) |
| `models/hackforger/bounty_test.go` | Model CRUD unit tests + TestMain |
| `models/hackforger/main_test.go` | TestMain for hackforger model package |
| `models/fixtures/bounty.yml` | Test fixture data |
| `models/fixtures/bounty_reward.yml` | Test fixture data |
| `models/fixtures/bounty_application.yml` | Test fixture data |
| `models/fixtures/bounty_winner.yml` | Test fixture data |
| `services/hackforger/bounty.go` | Bounty state machine + business logic |
| `services/hackforger/bounty_test.go` | Service unit tests |
| `services/hackforger/main_test.go` | TestMain for hackforger service package |
| `routers/web/hackforger/bounty.go` | Web handlers (explore, new bounty form) |
| `templates/hackforger/bounty/explore.tmpl` | Full Bounty explore page |
| `templates/hackforger/bounty/new.tmpl` | Create Bounty form |
| `templates/hackforger/bounty/panel.tmpl` | Issue sidebar Bounty panel (SSR) |
| `templates/hackforger/bounty/badge.tmpl` | Issue list Bounty badge |
| `web_src/js/components/hackforger/BountyPanel.vue` | Interactive Bounty panel (Vue) |
| `docs/tests/e2e/p1-bounty-e2e-prompt.md` | Manual E2E test prompt |
| `docs/tests/e2e/p1-bounty-e2e-report.md` | E2E report template |

### Modified Files

| File | Change |
|------|--------|
| `models/hackforger/bounty.go` | Expand BountyStatus (7 states), BountyMode (2 modes), add DeleteBounty |
| `models/hackforger/bounty_application.go` | Add CRUD functions |
| `models/hackforger/bounty_reward.go` | Add CRUD functions + error types |
| `models/hackforger/bounty_winner.go` | Add CRUD functions |
| `models/hackforger/action_types.go` | Add ActionBountyPaid (43) |
| `services/hackforger/notifier.go` | Bitmask AudienceType + full audience resolution + MergePullRequest hook |
| `routers/api/v1/hackforger/bounty.go` | Replace skeleton with full API handlers |
| `routers/api/v1/api.go` | Register Bounty repo-level + global routes |
| `routers/web/web.go` | Register Bounty web routes |
| `routers/web/hackforger/hackathon.go` | Remove ExploreBounties (moved to bounty.go) |
| `routers/web/repo/issue.go` | Load Bounty data in ViewIssue + batch load in list |
| `templates/repo/issue/view_content/sidebar.tmpl` | Inject Bounty panel |
| `templates/shared/issuelist.tmpl` | Inject Bounty badge |
| `options/locale/locale_en-US.ini` | Add Bounty i18n keys |
| `web_src/js/features/hackforger/init.js` | Already has BountyPanel mount (verify) |

---

## Chunk 1: Model Layer

### Task 1: Create Test Infrastructure + Fixtures

**Files:**
- Create: `models/hackforger/main_test.go`
- Create: `models/fixtures/bounty.yml`
- Create: `models/fixtures/bounty_reward.yml`
- Create: `models/fixtures/bounty_application.yml`
- Create: `models/fixtures/bounty_winner.yml`

- [ ] **Step 1: Create TestMain for hackforger models**

```go
// models/hackforger/main_test.go
package hackforger

import (
	"testing"

	"forgejo.org/models/unittest"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
```

- [ ] **Step 2: Create bounty fixture with test data**

```yaml
# models/fixtures/bounty.yml
-
  id: 1
  repo_id: 1
  issue_id: 1
  publisher_id: 2
  claimer_id: 0
  title: "Fix authentication bug"
  status: 0  # Open
  mode: 0    # Exclusive
  deadline: 1735689600
  created_unix: 1672578000
  updated_unix: 1672578000

-
  id: 2
  repo_id: 1
  issue_id: 2
  publisher_id: 2
  claimer_id: 0
  title: "Best CLI tool challenge"
  status: 0  # Open (Competitive has no Claimed state)
  mode: 1    # Competitive
  deadline: 1735689600
  created_unix: 1672578100
  updated_unix: 1672578100
```

- [ ] **Step 3: Create bounty_reward fixture**

```yaml
# models/fixtures/bounty_reward.yml
-
  id: 1
  bounty_id: 1
  type: "money"
  amount: 500
  currency: "USD"
  credits: 0
  rank: 1
  note: ""
  created_unix: 1672578000

-
  id: 2
  bounty_id: 1
  type: "credits"
  amount: 0
  currency: ""
  credits: 500
  rank: 1
  note: "bonus credits"
  created_unix: 1672578000

-
  id: 3
  bounty_id: 2
  type: "credits"
  amount: 0
  currency: ""
  credits: 1000
  rank: 1
  note: "1st place"
  created_unix: 1672578100
```

- [ ] **Step 4: Create bounty_application fixture**

```yaml
# models/fixtures/bounty_application.yml
-
  id: 1
  bounty_id: 1
  user_id: 4
  status: 0  # Pending
  message: "I can fix this"
  created_unix: 1672578200
  updated_unix: 1672578200
```

- [ ] **Step 5: Create bounty_winner fixture (empty)**

```yaml
# models/fixtures/bounty_winner.yml
# empty — winners are created by service tests
```

- [ ] **Step 6: Verify fixtures load**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./models/hackforger/ -run TestMain -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: PASS (TestMain loads fixtures without error)

- [ ] **Step 7: Commit**

```bash
git add models/hackforger/main_test.go models/fixtures/bounty*.yml
git commit -m "test: add hackforger bounty test infrastructure and fixtures"
```

---

### Task 2: Expand BountyStatus and BountyMode

**Files:**
- Modify: `models/hackforger/bounty.go`
- Create: `models/forgejo_migrations/v14c_fix-bounty-status-mode.go`

- [ ] **Step 1: Update BountyStatus enum (4 → 7 states)**

In `models/hackforger/bounty.go`, replace the status and mode constants:

```go
// BountyStatus represents the status of a bounty.
type BountyStatus int

const (
	BountyStatusOpen      BountyStatus = 0
	BountyStatusClaimed   BountyStatus = 1 // was InProgress
	BountyStatusInReview  BountyStatus = 2
	BountyStatusCompleted BountyStatus = 3
	BountyStatusPaid      BountyStatus = 4
	BountyStatusExpired   BountyStatus = 5
	BountyStatusCancelled BountyStatus = 6 // was 3, moved to 6
)

// BountyMode represents how a bounty is assigned.
type BountyMode int

const (
	BountyModeExclusive   BountyMode = 0 // was FirstCome
	BountyModeCompetitive BountyMode = 1 // was Application
)
```

- [ ] **Step 2: Add DeleteBounty function**

Append to `models/hackforger/bounty.go`:

```go
// DeleteBounty deletes a bounty by its ID.
func DeleteBounty(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(new(Bounty))
	return err
}
```

- [ ] **Step 3: Create migration file**

```go
// models/forgejo_migrations/v14c_fix-bounty-status-mode.go
package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "fix bounty status and mode values for Phase 1",
		Upgrade:     fixBountyStatusMode,
	})
}

func fixBountyStatusMode(x *xorm.Engine) error {
	// Remap old Cancelled(3) → new Cancelled(6)
	if _, err := x.Exec("UPDATE bounty SET status = 6 WHERE status = 3"); err != nil {
		return err
	}
	// Remove any Invitation mode(2) rows
	if _, err := x.Exec("DELETE FROM bounty WHERE mode = 2"); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go build ./models/hackforger/... && go build ./models/forgejo_migrations/...`

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add models/hackforger/bounty.go models/forgejo_migrations/v14c_fix-bounty-status-mode.go
git commit -m "feat(bounty): expand BountyStatus to 7 states, BountyMode to 2 modes

Add migration to remap old Cancelled(3)→6 and remove Invitation mode."
```

---

### Task 3: Complete BountyApplication CRUD

**Files:**
- Modify: `models/hackforger/bounty_application.go`

- [ ] **Step 1: Write tests for BountyApplication CRUD**

In `models/hackforger/bounty_test.go`:

```go
package hackforger

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBountyApplication(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	app := &BountyApplication{
		BountyID: 1,
		UserID:   5,
		Message:  "I want to work on this",
	}
	require.NoError(t, CreateBountyApplication(db.DefaultContext, app))
	assert.Greater(t, app.ID, int64(0))
	assert.Equal(t, ApplicationStatusPending, app.Status)
}

func TestCreateBountyApplication_Duplicate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User 4 already applied to bounty 1 (from fixture)
	app := &BountyApplication{
		BountyID: 1,
		UserID:   4,
		Message:  "duplicate",
	}
	err := CreateBountyApplication(db.DefaultContext, app)
	assert.True(t, IsErrAlreadyApplied(err))
}

func TestListBountyApplications(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	apps, count, err := ListBountyApplications(db.DefaultContext, ListBountyApplicationsOptions{
		BountyID: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Len(t, apps, 1)
	assert.Equal(t, int64(4), apps[0].UserID)
}

func TestUpdateBountyApplication(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	app := unittest.AssertExistsAndLoadBean(t, &BountyApplication{ID: 1})
	app.Status = ApplicationStatusAccepted
	require.NoError(t, UpdateBountyApplication(db.DefaultContext, app))

	updated := unittest.AssertExistsAndLoadBean(t, &BountyApplication{ID: 1})
	assert.Equal(t, ApplicationStatusAccepted, updated.Status)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./models/hackforger/ -run "TestCreateBountyApplication|TestListBountyApplications|TestUpdateBountyApplication" -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: FAIL — functions not defined

- [ ] **Step 3: Implement CRUD in bounty_application.go**

Add to `models/hackforger/bounty_application.go`:

```go
import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

// CreateBountyApplication creates a new application. Returns ErrAlreadyApplied
// if the user has already applied to the same bounty.
func CreateBountyApplication(ctx context.Context, app *BountyApplication) error {
	has, err := db.GetEngine(ctx).
		Where("bounty_id = ? AND user_id = ?", app.BountyID, app.UserID).
		Exist(new(BountyApplication))
	if err != nil {
		return err
	}
	if has {
		return ErrAlreadyApplied{BountyID: app.BountyID, UserID: app.UserID}
	}
	_, err = db.GetEngine(ctx).Insert(app)
	return err
}

// GetBountyApplicationByID returns an application by ID.
func GetBountyApplicationByID(ctx context.Context, id int64) (*BountyApplication, error) {
	app := new(BountyApplication)
	has, err := db.GetEngine(ctx).ID(id).Get(app)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("bounty application does not exist [id: %d]", id)
	}
	return app, nil
}

// ListBountyApplicationsOptions holds options for listing applications.
type ListBountyApplicationsOptions struct {
	db.ListOptions
	BountyID int64
	UserID   int64
	Status   *ApplicationStatus
}

func (opts ListBountyApplicationsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.BountyID > 0 {
		cond = cond.And(builder.Eq{"bounty_id": opts.BountyID})
	}
	if opts.UserID > 0 {
		cond = cond.And(builder.Eq{"user_id": opts.UserID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"status": *opts.Status})
	}
	return cond
}

// ListBountyApplications returns applications matching the given options.
func ListBountyApplications(ctx context.Context, opts ListBountyApplicationsOptions) ([]*BountyApplication, int64, error) {
	return db.FindAndCount[BountyApplication](ctx, opts)
}

// UpdateBountyApplication updates an existing application.
func UpdateBountyApplication(ctx context.Context, app *BountyApplication) error {
	_, err := db.GetEngine(ctx).ID(app.ID).AllCols().Update(app)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./models/hackforger/ -run "TestCreateBountyApplication|TestListBountyApplications|TestUpdateBountyApplication" -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add models/hackforger/bounty_application.go models/hackforger/bounty_test.go
git commit -m "feat(bounty): add BountyApplication CRUD functions with tests"
```

---

### Task 4: Complete BountyReward CRUD + Error Types

**Files:**
- Modify: `models/hackforger/bounty_reward.go`

- [ ] **Step 1: Write tests for BountyReward CRUD**

Append to `models/hackforger/bounty_test.go`:

```go
func TestCreateBountyReward(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	r := &BountyReward{
		BountyID: 2,
		Type:     RewardTypeCredits,
		Credits:  500,
		Rank:     2,
	}
	require.NoError(t, CreateBountyReward(db.DefaultContext, r))
	assert.Greater(t, r.ID, int64(0))
}

func TestListBountyRewards(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	rewards, err := ListBountyRewards(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, rewards, 2) // fixture has 2 rewards for bounty 1
}

func TestDeleteBountyReward(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	require.NoError(t, DeleteBountyReward(db.DefaultContext, 1))

	rewards, err := ListBountyRewards(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, rewards, 1)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./models/hackforger/ -run "TestCreateBountyReward|TestListBountyRewards|TestDeleteBountyReward" -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: FAIL

- [ ] **Step 3: Implement in bounty_reward.go**

Add to `models/hackforger/bounty_reward.go`:

```go
import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// ErrBountyRewardNotExist represents a "BountyRewardNotExist" kind of error.
type ErrBountyRewardNotExist struct {
	ID int64
}

func IsErrBountyRewardNotExist(err error) bool {
	_, ok := err.(ErrBountyRewardNotExist)
	return ok
}

func (err ErrBountyRewardNotExist) Error() string {
	return fmt.Sprintf("bounty reward does not exist [id: %d]", err.ID)
}

func (err ErrBountyRewardNotExist) Unwrap() error {
	return util.ErrNotExist
}

// CreateBountyReward creates a new reward.
func CreateBountyReward(ctx context.Context, r *BountyReward) error {
	_, err := db.GetEngine(ctx).Insert(r)
	return err
}

// ListBountyRewards returns all rewards for a bounty, ordered by rank.
func ListBountyRewards(ctx context.Context, bountyID int64) ([]*BountyReward, error) {
	var rewards []*BountyReward
	err := db.GetEngine(ctx).Where("bounty_id = ?", bountyID).OrderBy("rank ASC").Find(&rewards)
	return rewards, err
}

// DeleteBountyReward deletes a reward by ID.
func DeleteBountyReward(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(new(BountyReward))
	return err
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./models/hackforger/ -run "TestCreateBountyReward|TestListBountyRewards|TestDeleteBountyReward" -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add models/hackforger/bounty_reward.go models/hackforger/bounty_test.go
git commit -m "feat(bounty): add BountyReward CRUD functions and error types"
```

---

### Task 5: Complete BountyWinner CRUD + More Model Tests

**Files:**
- Modify: `models/hackforger/bounty_winner.go`

- [ ] **Step 1: Write tests for BountyWinner + existing Bounty CRUD**

Append to `models/hackforger/bounty_test.go`:

```go
func TestCreateBountyWinner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	w := &BountyWinner{
		BountyID: 2,
		UserID:   4,
		Rank:     1,
	}
	require.NoError(t, CreateBountyWinner(db.DefaultContext, w))
	assert.Greater(t, w.ID, int64(0))
}

func TestListBountyWinners(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create a winner first
	w := &BountyWinner{BountyID: 2, UserID: 4, Rank: 1}
	require.NoError(t, CreateBountyWinner(db.DefaultContext, w))

	winners, err := ListBountyWinners(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Len(t, winners, 1)
	assert.Equal(t, int64(4), winners[0].UserID)
}

func TestCreateBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	b := &Bounty{
		RepoID:      1,
		IssueID:     100, // unused issue ID
		PublisherID: 2,
		Title:       "New bounty",
		Mode:        BountyModeExclusive,
	}
	require.NoError(t, CreateBounty(db.DefaultContext, b))
	assert.Greater(t, b.ID, int64(0))
	assert.Equal(t, BountyStatusOpen, b.Status)
}

func TestCreateBounty_DuplicateIssue(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	b := &Bounty{
		RepoID:      1,
		IssueID:     1, // already used by fixture bounty 1
		PublisherID: 2,
		Title:       "duplicate",
	}
	err := CreateBounty(db.DefaultContext, b)
	assert.True(t, IsErrBountyAlreadyExists(err))
}

func TestGetBountyByIssueID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	b, err := GetBountyByIssueID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), b.ID)
	assert.Equal(t, "Fix authentication bug", b.Title)
}

func TestListBounties_Filter(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	status := BountyStatusOpen
	bounties, count, err := ListBounties(db.DefaultContext, ListBountiesOptions{
		RepoID: 1,
		Status: &status,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Len(t, bounties, 1)
	assert.Equal(t, BountyStatusOpen, bounties[0].Status)
}
```

- [ ] **Step 2: Run to verify failure for winner tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./models/hackforger/ -run "TestCreateBountyWinner|TestListBountyWinners" -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: FAIL — functions not defined

- [ ] **Step 3: Implement in bounty_winner.go**

Add to `models/hackforger/bounty_winner.go`:

```go
import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

// CreateBountyWinner creates a new winner record.
func CreateBountyWinner(ctx context.Context, w *BountyWinner) error {
	_, err := db.GetEngine(ctx).Insert(w)
	return err
}

// ListBountyWinners returns all winners for a bounty, ordered by rank.
func ListBountyWinners(ctx context.Context, bountyID int64) ([]*BountyWinner, error) {
	var winners []*BountyWinner
	err := db.GetEngine(ctx).Where("bounty_id = ?", bountyID).OrderBy("rank ASC").Find(&winners)
	return winners, err
}
```

- [ ] **Step 4: Run all model tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./models/hackforger/ -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add models/hackforger/bounty_winner.go models/hackforger/bounty_test.go
git commit -m "feat(bounty): add BountyWinner CRUD and complete model test suite"
```

---

### Task 6: Add ActionBountyPaid to action_types.go

**Files:**
- Modify: `models/hackforger/action_types.go`

- [ ] **Step 1: Add ActionBountyPaid constant**

In `models/hackforger/action_types.go`, after `ActionCreditsRedeemed`:

```go
ActionCreditsRedeemed       activities_model.ActionType = 42
ActionBountyPaid            activities_model.ActionType = 43 // entity feed only
```

And add to `HackforgerActionTypeName` map:

```go
ActionBountyPaid:            "bounty_paid",
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go build ./models/hackforger/...`

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/action_types.go
git commit -m "feat(bounty): add ActionBountyPaid (43) for entity feed tracking"
```

---

## Chunk 2: Service Layer

### Task 7: Rewrite Notifier with Bitmask Audience Resolution

**Files:**
- Modify: `services/hackforger/notifier.go`

- [ ] **Step 1: Write notifier audience resolution tests**

Create `services/hackforger/main_test.go`:

```go
package hackforger

import (
	"testing"

	"forgejo.org/models/unittest"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
```

Create `services/hackforger/notifier_test.go`:

```go
package hackforger

import (
	"testing"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishHackforgerAction_Global(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := PublishHackforgerAction(db.DefaultContext, &HackforgerActionOpts{
		ActUserID:    2,
		OpType:       hackforger_model.ActionBountyCreated,
		RepoID:       1,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: 1, EntityName: "test"},
		AudienceType: AudienceGlobal,
	})
	require.NoError(t, err)

	// Check global record exists (UserID=0)
	var actions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).Where("op_type = ? AND user_id = 0", hackforger_model.ActionBountyCreated).Find(&actions)
	require.NoError(t, err)
	assert.NotEmpty(t, actions)
}

func TestPublishHackforgerAction_Followers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User 4 follows user 2 in Forgejo fixtures
	err := PublishHackforgerAction(db.DefaultContext, &HackforgerActionOpts{
		ActUserID:    2,
		OpType:       hackforger_model.ActionBountyClaimed,
		RepoID:       1,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: 1, EntityName: "test"},
		AudienceType: AudienceFollowers,
	})
	require.NoError(t, err)

	// Actor record should exist
	var actorActions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).Where("op_type = ? AND user_id = ? AND act_user_id = ?",
		hackforger_model.ActionBountyClaimed, 2, 2).Find(&actorActions)
	require.NoError(t, err)
	assert.NotEmpty(t, actorActions, "actor's own record should exist")
}

func TestPublishHackforgerAction_CombinedAudience(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Test bitmask combination: Global | RepoWatchers
	err := PublishHackforgerAction(db.DefaultContext, &HackforgerActionOpts{
		ActUserID:    2,
		OpType:       hackforger_model.ActionBountyCreated,
		RepoID:       1,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: 1, EntityName: "test"},
		AudienceType: AudienceGlobal | AudienceRepoWatchers,
	})
	require.NoError(t, err)

	// Global record
	var globalActions []*activities_model.Action
	err = db.GetEngine(db.DefaultContext).Where("op_type = ? AND user_id = 0",
		hackforger_model.ActionBountyCreated).Find(&globalActions)
	require.NoError(t, err)
	assert.NotEmpty(t, globalActions)
}

func TestPublishHackforgerAction_Dedup(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// If same user is both follower and watcher, should only get one record
	err := PublishHackforgerAction(db.DefaultContext, &HackforgerActionOpts{
		ActUserID:    2,
		OpType:       hackforger_model.ActionBountyCreated,
		RepoID:       1,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: 1, EntityName: "test"},
		AudienceType: AudienceFollowers | AudienceRepoWatchers,
	})
	require.NoError(t, err)
	// No duplicate insertion errors = pass
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./services/hackforger/ -run "TestPublishHackforgerAction" -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: FAIL — AudienceGlobal value changed, bitmask functions don't exist

- [ ] **Step 3: Rewrite notifier.go with bitmask audience**

Replace `services/hackforger/notifier.go` with:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	notify_service "forgejo.org/services/notify"
)

type hackforgerNotifier struct {
	notify_service.NullNotifier
}

var _ notify_service.Notifier = &hackforgerNotifier{}

// Init registers the HackForger notifier.
func Init() error {
	notify_service.RegisterNotifier(&hackforgerNotifier{})
	return nil
}

// AudienceType determines how feed events are distributed (bitmask).
type AudienceType int

const (
	AudienceGlobal       AudienceType = 1 << 0 // 1 — UserID=0, visible to everyone
	AudienceFollowers    AudienceType = 1 << 1 // 2 — Visible to ActUser's followers
	AudienceOrgMembers   AudienceType = 1 << 2 // 4 — Visible to org members
	AudienceRepoWatchers AudienceType = 1 << 3 // 8 — Visible to repo watchers
)

// HackforgerActionOpts holds the parameters for publishing a HackForger feed event.
type HackforgerActionOpts struct {
	ActUserID    int64
	OpType       activities_model.ActionType
	RepoID       int64
	Content      any
	AudienceType AudienceType
	OrgID        int64 // used when AudienceType includes AudienceOrgMembers
}

// PublishHackforgerAction writes HackForger events to the action table
// with full audience resolution using bitmask-based distribution.
func PublishHackforgerAction(ctx context.Context, opts *HackforgerActionOpts) error {
	contentBytes, err := json.Marshal(opts.Content)
	if err != nil {
		return err
	}
	contentStr := string(contentBytes)
	now := timeutil.TimeStampNow()

	// Always insert the actor's own action record
	actorAction := &activities_model.Action{
		ActUserID:   opts.ActUserID,
		UserID:      opts.ActUserID,
		OpType:      opts.OpType,
		RepoID:      opts.RepoID,
		Content:     contentStr,
		CreatedUnix: now,
	}
	if _, err := db.GetEngine(ctx).Insert(actorAction); err != nil {
		log.Error("PublishHackforgerAction (actor): %v", err)
		return err
	}

	// Collect target UserIDs with dedup
	targets := make(map[int64]bool)

	// Global: insert UserID=0 record
	if opts.AudienceType&AudienceGlobal != 0 {
		globalAction := &activities_model.Action{
			ActUserID:   opts.ActUserID,
			UserID:      0,
			OpType:      opts.OpType,
			RepoID:      opts.RepoID,
			Content:     contentStr,
			CreatedUnix: now,
		}
		if _, err := db.GetEngine(ctx).Insert(globalAction); err != nil {
			log.Error("PublishHackforgerAction (global): %v", err)
			return err
		}
	}

	// Followers: query user_follow table
	if opts.AudienceType&AudienceFollowers != 0 {
		var followerIDs []int64
		err := db.GetEngine(ctx).Table("follow").
			Where("follow_id = ?", opts.ActUserID).
			Cols("user_id").Find(&followerIDs)
		if err != nil {
			log.Error("PublishHackforgerAction (followers query): %v", err)
		} else {
			for _, id := range followerIDs {
				targets[id] = true
			}
		}
	}

	// Org members
	if opts.AudienceType&AudienceOrgMembers != 0 && opts.OrgID > 0 {
		var memberIDs []int64
		err := db.GetEngine(ctx).Table("org_user").
			Where("org_id = ?", opts.OrgID).
			Cols("uid").Find(&memberIDs)
		if err != nil {
			log.Error("PublishHackforgerAction (org members query): %v", err)
		} else {
			for _, id := range memberIDs {
				targets[id] = true
			}
		}
	}

	// Repo watchers
	if opts.AudienceType&AudienceRepoWatchers != 0 && opts.RepoID > 0 {
		var watcherIDs []int64
		err := db.GetEngine(ctx).Table("watch").
			Where("repo_id = ? AND mode != ?", opts.RepoID, repo_model.WatchModeDont).
			Cols("user_id").Find(&watcherIDs)
		if err != nil {
			log.Error("PublishHackforgerAction (watchers query): %v", err)
		} else {
			for _, id := range watcherIDs {
				targets[id] = true
			}
		}
	}

	// Remove actor (already inserted) and UserID=0 (handled separately)
	delete(targets, opts.ActUserID)
	delete(targets, 0)

	// Batch insert for all targets
	if len(targets) > 0 {
		actions := make([]*activities_model.Action, 0, len(targets))
		for uid := range targets {
			actions = append(actions, &activities_model.Action{
				ActUserID:   opts.ActUserID,
				UserID:      uid,
				OpType:      opts.OpType,
				RepoID:      opts.RepoID,
				Content:     contentStr,
				CreatedUnix: now,
			})
		}
		if _, err := db.GetEngine(ctx).Insert(actions); err != nil {
			log.Error("PublishHackforgerAction (batch targets): %v", err)
			return err
		}
	}

	return nil
}

// MergePullRequest is called when a PR is merged. If the PR's issue has a
// linked Bounty and the merge author is the Claimer, transition to InReview.
func (n *hackforgerNotifier) MergePullRequest(ctx context.Context, doer *user_model.User, pr *issues_model.PullRequest) {
	if err := pr.LoadIssue(ctx); err != nil {
		log.Error("hackforgerNotifier.MergePullRequest: LoadIssue: %v", err)
		return
	}

	bounty, err := hackforger_model.GetBountyByIssueID(ctx, pr.Issue.ID)
	if err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			return // no bounty linked — nothing to do
		}
		log.Error("hackforgerNotifier.MergePullRequest: GetBountyByIssueID: %v", err)
		return
	}

	if bounty.Status != hackforger_model.BountyStatusClaimed {
		return
	}
	if doer.ID != bounty.ClaimerID {
		return
	}

	bounty.Status = hackforger_model.BountyStatusInReview
	if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
		log.Error("hackforgerNotifier.MergePullRequest: UpdateBounty: %v", err)
		return
	}

	if err := PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doer.ID,
		OpType:       hackforger_model.ActionBountyDelivered,
		RepoID:       bounty.RepoID,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: bounty.ID, EntityName: bounty.Title},
		AudienceType: AudienceFollowers | AudienceRepoWatchers,
	}); err != nil {
		log.Error("hackforgerNotifier.MergePullRequest: PublishHackforgerAction: %v", err)
	}
}
```

- [ ] **Step 4: Run notifier tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./services/hackforger/ -run "TestPublishHackforgerAction" -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/hackforger/notifier.go services/hackforger/notifier_test.go services/hackforger/main_test.go
git commit -m "feat(notifier): rewrite PublishHackforgerAction with bitmask audience resolution

Supports 4 audience types via bitmask composition: Global, Followers,
OrgMembers, RepoWatchers. Includes dedup and batch insert.
Also implements MergePullRequest hook for Bounty-PR linkage."
```

---

### Task 8: Implement Bounty Service State Machine

**Files:**
- Create: `services/hackforger/bounty.go`
- Create: `services/hackforger/bounty_test.go`

- [ ] **Step 1: Write service tests**

Create `services/hackforger/bounty_test.go`:

```go
package hackforger

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyForBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := ApplyForBounty(db.DefaultContext, 1, 5, "I can do this")
	require.NoError(t, err)

	apps, _, err := hackforger_model.ListBountyApplications(db.DefaultContext, hackforger_model.ListBountyApplicationsOptions{
		BountyID: 1,
		UserID:   5,
	})
	require.NoError(t, err)
	assert.Len(t, apps, 1)
}

func TestApplyForBounty_NotOpen(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Bounty 2 is Claimed (status=1)
	err := ApplyForBounty(db.DefaultContext, 2, 5, "too late")
	assert.Error(t, err)
}

func TestAcceptApplication_Exclusive(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Accept application 1 (user 4 for bounty 1, exclusive mode)
	err := AcceptApplication(db.DefaultContext, 1, 2) // doer=2 is publisher
	require.NoError(t, err)

	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, bounty.Status)
	assert.Equal(t, int64(4), bounty.ClaimerID)
}

func TestAcceptApplication_NotPublisher(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := AcceptApplication(db.DefaultContext, 1, 5) // doer=5 is NOT publisher
	assert.Error(t, err)
}

func TestCancelBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := CancelBounty(db.DefaultContext, 1, 2) // publisher cancels
	require.NoError(t, err)

	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusCancelled, bounty.Status)
}

func TestCancelBounty_NotPublisher(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := CancelBounty(db.DefaultContext, 1, 5)
	assert.Error(t, err)
}

func TestCompleteBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 to InReview state first
	b, _ := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	b.Status = hackforger_model.BountyStatusInReview
	b.ClaimerID = 4
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, b))

	err := CompleteBounty(db.DefaultContext, 1, 2) // publisher completes
	require.NoError(t, err)

	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusCompleted, bounty.Status)
}

func TestRejectDelivery(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 to InReview
	b, _ := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	b.Status = hackforger_model.BountyStatusInReview
	b.ClaimerID = 4
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, b))

	err := RejectDelivery(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, bounty.Status)
}

func TestMarkPaid(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	b, _ := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	b.Status = hackforger_model.BountyStatusCompleted
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, b))

	err := MarkPaid(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusPaid, bounty.Status)
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./services/hackforger/ -run "TestApplyForBounty|TestAcceptApplication|TestCancelBounty|TestCompleteBounty|TestRejectDelivery|TestMarkPaid" -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: FAIL

- [ ] **Step 3: Implement bounty service**

Create `services/hackforger/bounty.go`:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/timeutil"
)

// ErrInvalidBountyStatus is returned when a state transition is not allowed.
type ErrInvalidBountyStatus struct {
	BountyID int64
	Current  hackforger_model.BountyStatus
	Target   hackforger_model.BountyStatus
}

func (e ErrInvalidBountyStatus) Error() string {
	return fmt.Sprintf("invalid bounty status transition [bounty: %d, %d → %d]", e.BountyID, e.Current, e.Target)
}

// ErrNotPublisher is returned when a non-publisher tries a publisher-only action.
type ErrNotPublisher struct {
	BountyID int64
	DoerID   int64
}

func (e ErrNotPublisher) Error() string {
	return fmt.Sprintf("user %d is not the publisher of bounty %d", e.DoerID, e.BountyID)
}

func requirePublisher(bounty *hackforger_model.Bounty, doerID int64) error {
	if bounty.PublisherID != doerID {
		return ErrNotPublisher{BountyID: bounty.ID, DoerID: doerID}
	}
	return nil
}

// ApplyForBounty creates an application for a bounty.
func ApplyForBounty(ctx context.Context, bountyID, userID int64, message string) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}
	if bounty.Status != hackforger_model.BountyStatusOpen {
		return ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Target: hackforger_model.BountyStatusOpen}
	}

	return hackforger_model.CreateBountyApplication(ctx, &hackforger_model.BountyApplication{
		BountyID: bountyID,
		UserID:   userID,
		Message:  message,
	})
}

// AcceptApplication accepts an application. For Exclusive mode, rejects all
// other pending applications and sets Bounty to Claimed.
func AcceptApplication(ctx context.Context, applicationID, doerID int64) error {
	app, err := hackforger_model.GetBountyApplicationByID(ctx, applicationID)
	if err != nil {
		return err
	}

	bounty, err := hackforger_model.GetBountyByID(ctx, app.BountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		app.Status = hackforger_model.ApplicationStatusAccepted
		if err := hackforger_model.UpdateBountyApplication(ctx, app); err != nil {
			return err
		}

		if bounty.Mode == hackforger_model.BountyModeExclusive {
			// Reject all other pending applications
			pending := hackforger_model.ApplicationStatusPending
			others, _, err := hackforger_model.ListBountyApplications(ctx, hackforger_model.ListBountyApplicationsOptions{
				BountyID: bounty.ID,
				Status:   &pending,
			})
			if err != nil {
				return err
			}
			for _, other := range others {
				if other.ID == app.ID {
					continue
				}
				other.Status = hackforger_model.ApplicationStatusRejected
				if err := hackforger_model.UpdateBountyApplication(ctx, other); err != nil {
					return err
				}
			}

			bounty.Status = hackforger_model.BountyStatusClaimed
			bounty.ClaimerID = app.UserID
			if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
				return err
			}

			return PublishHackforgerAction(ctx, &HackforgerActionOpts{
				ActUserID:    app.UserID,
				OpType:       hackforger_model.ActionBountyClaimed,
				RepoID:       bounty.RepoID,
				Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: bounty.ID, EntityName: bounty.Title},
				AudienceType: AudienceFollowers | AudienceRepoWatchers,
			})
		}

		// Competitive: just accept, no status change
		return nil
	})
}

// RejectApplication rejects an application.
func RejectApplication(ctx context.Context, applicationID, doerID int64) error {
	app, err := hackforger_model.GetBountyApplicationByID(ctx, applicationID)
	if err != nil {
		return err
	}
	bounty, err := hackforger_model.GetBountyByID(ctx, app.BountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}
	app.Status = hackforger_model.ApplicationStatusRejected
	return hackforger_model.UpdateBountyApplication(ctx, app)
}

// StartReview transitions a Competitive bounty from Open to InReview.
func StartReview(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}
	if bounty.Status != hackforger_model.BountyStatusOpen || bounty.Mode != hackforger_model.BountyModeCompetitive {
		return ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Target: hackforger_model.BountyStatusInReview}
	}
	bounty.Status = hackforger_model.BountyStatusInReview
	return hackforger_model.UpdateBounty(ctx, bounty)
}

// CompleteBounty verifies delivery and transitions to Completed.
// If there are credits-type rewards, deposits them to the claimer.
func CompleteBounty(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}
	if bounty.Status != hackforger_model.BountyStatusInReview {
		return ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Target: hackforger_model.BountyStatusCompleted}
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		bounty.Status = hackforger_model.BountyStatusCompleted
		if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
			return err
		}

		// Deposit credits rewards to claimer
		if bounty.ClaimerID > 0 {
			rewards, err := hackforger_model.ListBountyRewards(ctx, bounty.ID)
			if err != nil {
				return err
			}
			for _, r := range rewards {
				if r.Type == hackforger_model.RewardTypeCredits && r.Credits > 0 {
					if err := Deposit(ctx, bounty.ClaimerID, r.Credits,
						fmt.Sprintf("bounty:%d", bounty.ID), "Bounty reward"); err != nil {
						return err
					}
				}
			}
		}

		return PublishHackforgerAction(ctx, &HackforgerActionOpts{
			ActUserID:    doerID,
			OpType:       hackforger_model.ActionBountyCompleted,
			RepoID:       bounty.RepoID,
			Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: bounty.ID, EntityName: bounty.Title},
			AudienceType: AudienceFollowers | AudienceRepoWatchers,
		})
	})
}

// RejectDelivery returns an Exclusive bounty from InReview to Claimed.
func RejectDelivery(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}
	if bounty.Status != hackforger_model.BountyStatusInReview || bounty.Mode != hackforger_model.BountyModeExclusive {
		return ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Target: hackforger_model.BountyStatusClaimed}
	}
	bounty.Status = hackforger_model.BountyStatusClaimed
	return hackforger_model.UpdateBounty(ctx, bounty)
}

// WinnerInput holds the data for selecting a winner.
type WinnerInput struct {
	UserID int64
	Rank   int
}

// SelectWinners selects winners for a Competitive bounty, deposits ranked credits.
func SelectWinners(ctx context.Context, bountyID, doerID int64, winners []WinnerInput) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}
	if bounty.Status != hackforger_model.BountyStatusInReview || bounty.Mode != hackforger_model.BountyModeCompetitive {
		return ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Target: hackforger_model.BountyStatusCompleted}
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		rewards, err := hackforger_model.ListBountyRewards(ctx, bounty.ID)
		if err != nil {
			return err
		}
		rewardByRank := make(map[int]*hackforger_model.BountyReward)
		for _, r := range rewards {
			rewardByRank[r.Rank] = r
		}

		for _, w := range winners {
			if err := hackforger_model.CreateBountyWinner(ctx, &hackforger_model.BountyWinner{
				BountyID: bountyID,
				UserID:   w.UserID,
				Rank:     w.Rank,
			}); err != nil {
				return err
			}

			if r, ok := rewardByRank[w.Rank]; ok && r.Type == hackforger_model.RewardTypeCredits && r.Credits > 0 {
				if err := Deposit(ctx, w.UserID, r.Credits,
					fmt.Sprintf("bounty:%d:rank:%d", bounty.ID, w.Rank), "Bounty winner reward"); err != nil {
					return err
				}
			}
		}

		bounty.Status = hackforger_model.BountyStatusCompleted
		if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
			return err
		}

		return PublishHackforgerAction(ctx, &HackforgerActionOpts{
			ActUserID:    doerID,
			OpType:       hackforger_model.ActionBountyWinnersSelected,
			RepoID:       bounty.RepoID,
			Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: bounty.ID, EntityName: bounty.Title},
			AudienceType: AudienceGlobal,
		})
	})
}

// MarkPaid transitions a Completed bounty to Paid.
func MarkPaid(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}
	if bounty.Status != hackforger_model.BountyStatusCompleted {
		return ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Target: hackforger_model.BountyStatusPaid}
	}

	bounty.Status = hackforger_model.BountyStatusPaid
	if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
		return err
	}

	// Entity-feed-only: AudienceType=0 means only actor record
	return PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doerID,
		OpType:       hackforger_model.ActionBountyPaid,
		RepoID:       bounty.RepoID,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: bounty.ID, EntityName: bounty.Title},
		AudienceType: 0,
	})
}

// CancelBounty cancels a bounty (allowed from Open or Claimed).
func CancelBounty(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}
	if bounty.Status != hackforger_model.BountyStatusOpen && bounty.Status != hackforger_model.BountyStatusClaimed {
		return ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Target: hackforger_model.BountyStatusCancelled}
	}

	bounty.Status = hackforger_model.BountyStatusCancelled
	if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
		return err
	}

	return PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doerID,
		OpType:       hackforger_model.ActionBountyCancelled,
		RepoID:       bounty.RepoID,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: bounty.ID, EntityName: bounty.Title},
		AudienceType: AudienceRepoWatchers,
	})
}

// CheckExpiredBounties checks for bounties that have passed their deadline
// and marks them as Expired.
func CheckExpiredBounties(ctx context.Context) error {
	now := timeutil.TimeStampNow()
	var bounties []*hackforger_model.Bounty
	err := db.GetEngine(ctx).
		Where("deadline > 0 AND deadline < ? AND status IN (?, ?)",
			now, hackforger_model.BountyStatusOpen, hackforger_model.BountyStatusClaimed).
		Find(&bounties)
	if err != nil {
		return err
	}

	for _, b := range bounties {
		b.Status = hackforger_model.BountyStatusExpired
		if err := hackforger_model.UpdateBounty(ctx, b); err != nil {
			return err
		}
		_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
			ActUserID:    b.PublisherID,
			OpType:       hackforger_model.ActionBountyExpired,
			RepoID:       b.RepoID,
			Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: b.ID, EntityName: b.Title},
			AudienceType: AudienceRepoWatchers,
		})
	}
	return nil
}

// UpdateBountyMeta updates bounty title and deadline. Only allowed when Open.
func UpdateBountyMeta(ctx context.Context, bountyID, doerID int64, title string, deadline timeutil.TimeStamp) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}
	if bounty.Status != hackforger_model.BountyStatusOpen {
		return ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Target: hackforger_model.BountyStatusOpen}
	}
	if title != "" {
		bounty.Title = title
	}
	bounty.Deadline = deadline
	return hackforger_model.UpdateBounty(ctx, bounty)
}

// DeleteBounty deletes a bounty. Only allowed when Open with no applications.
func DeleteBounty(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}
	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}
	if bounty.Status != hackforger_model.BountyStatusOpen {
		return ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Target: hackforger_model.BountyStatusOpen}
	}

	apps, count, err := hackforger_model.ListBountyApplications(ctx, hackforger_model.ListBountyApplicationsOptions{
		BountyID: bountyID,
	})
	if err != nil {
		return err
	}
	_ = apps
	if count > 0 {
		return fmt.Errorf("cannot delete bounty with existing applications [bounty_id: %d, count: %d]", bountyID, count)
	}

	return hackforger_model.DeleteBounty(ctx, bountyID)
}
```

- [ ] **Step 4: Run service tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./services/hackforger/ -run "TestApplyForBounty|TestAcceptApplication|TestCancelBounty|TestCompleteBounty|TestRejectDelivery|TestMarkPaid" -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/hackforger/bounty.go services/hackforger/bounty_test.go
git commit -m "feat(bounty): implement Bounty service state machine

Includes: ApplyForBounty, AcceptApplication, RejectApplication,
StartReview, CompleteBounty, RejectDelivery, SelectWinners,
MarkPaid, CancelBounty, CheckExpiredBounties, UpdateBountyMeta,
DeleteBounty. All state changes publish feed events."
```

---

## Chunk 3: API Layer

### Task 9: Implement Bounty API Handlers

**Files:**
- Modify: `routers/api/v1/hackforger/bounty.go` (replace skeleton)
- Modify: `routers/api/v1/api.go` (register routes)

- [ ] **Step 1: Implement full API handlers**

Replace `routers/api/v1/hackforger/bounty.go` with full implementation. This is the largest single file (~650 lines). Key handlers:

- `CreateBounty` — POST, requires repo writer, calls `hackforger_model.CreateBounty` + publishes ActionBountyCreated
- `ListRepoBounties` — GET, public, calls `hackforger_model.ListBounties` with repo filter
- `GetBounty` — GET, public, calls `hackforger_model.GetBountyByID`
- `UpdateBounty` — PUT, publisher only, calls `hackforger_svc.UpdateBountyMeta`
- `DeleteBounty` — DELETE, publisher only, calls `hackforger_svc.DeleteBounty`
- `AddReward` — POST, publisher only
- `ListRewards` — GET, public
- `DeleteReward` — DELETE, publisher + Open only
- `ApplyForBounty` — POST, any user, calls `hackforger_svc.ApplyForBounty`
- `ListApplications` — GET, publisher only
- `ReviewApplication` — PUT, publisher, calls Accept/Reject
- `StartReview` — POST, publisher, competitive only
- `CompleteBounty` — POST, publisher, calls `hackforger_svc.CompleteBounty`
- `RejectDelivery` — POST, publisher, exclusive only
- `MarkPaid` — POST, publisher
- `CancelBounty` — POST, publisher
- `ExpireBounty` — POST, publisher/admin
- `SelectWinners` — POST, publisher, competitive
- `ListWinners` — GET, public
- `ListAllBounties` — GET, global list (replace P0 skeleton)
- `BountyStats` — GET, aggregate stats
- `HunterLeaderboard` — GET, ranked hunters

Each handler follows Forgejo's pattern: Swagger annotation comment → parameter parsing → service call → error handling → JSON response.

- [ ] **Step 2: Register routes in api.go**

Add to the hackforger group in `routers/api/v1/api.go`:

```go
// HackForger API routes
m.Group("/hackforger", func() {
    m.Get("/hackathons", hackforger_api.ListHackathons)
    m.Get("/bounties", hackforger_api.ListAllBounties)
    m.Get("/bounties/stats", hackforger_api.BountyStats)
    m.Get("/bounties/leaderboard", hackforger_api.HunterLeaderboard)
    m.Get("/grants/rounds", hackforger_api.ListGrantRounds)
    m.Get("/credits/balance", hackforger_api.GetBalance)
    m.Get("/feed", hackforger_api.GetFeed)
})
```

Add repo-level bounty routes inside the existing `/{username}/{reponame}` group:

```go
// HackForger Bounty routes (repo-level)
m.Group("/bounties", func() {
    m.Get("", hackforger_api.ListRepoBounties)
    m.Post("", reqToken(), mustNotBeArchived, hackforger_api.CreateBounty)
    m.Group("/{bounty_id}", func() {
        m.Get("", hackforger_api.GetBounty)
        m.Put("", reqToken(), hackforger_api.UpdateBounty)
        m.Delete("", reqToken(), hackforger_api.DeleteBounty)
        m.Get("/rewards", hackforger_api.ListRewards)
        m.Post("/rewards", reqToken(), hackforger_api.AddReward)
        m.Delete("/rewards/{reward_id}", reqToken(), hackforger_api.DeleteReward)
        m.Get("/applications", reqToken(), hackforger_api.ListApplications)
        m.Post("/applications", reqToken(), hackforger_api.ApplyForBounty)
        m.Put("/applications/{application_id}", reqToken(), hackforger_api.ReviewApplication)
        m.Post("/start-review", reqToken(), hackforger_api.StartReview)
        m.Post("/complete", reqToken(), hackforger_api.CompleteBounty)
        m.Post("/reject-delivery", reqToken(), hackforger_api.RejectDelivery)
        m.Post("/pay", reqToken(), hackforger_api.MarkPaid)
        m.Post("/cancel", reqToken(), hackforger_api.CancelBounty)
        m.Post("/expire", reqToken(), hackforger_api.ExpireBounty)
        m.Post("/winners", reqToken(), hackforger_api.SelectWinners)
        m.Get("/winners", hackforger_api.ListWinners)
    })
})
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go build ./routers/...`

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add routers/api/v1/hackforger/bounty.go routers/api/v1/api.go
git commit -m "feat(bounty): implement full Bounty REST API

20 endpoints: repo-level CRUD + applications + state transitions +
global list/stats/leaderboard. All with Swagger annotations."
```

---

## Chunk 4: Web Routes + Templates + Frontend

### Task 10: Web Routes and Explore Page

**Files:**
- Create: `routers/web/hackforger/bounty.go`
- Create: `templates/hackforger/bounty/explore.tmpl`
- Modify: `routers/web/hackforger/hackathon.go` (remove ExploreBounties)
- Modify: `routers/web/web.go` (register new routes)
- Modify: `options/locale/locale_en-US.ini` (add i18n keys)

- [ ] **Step 1: Create bounty web handler**

Create `routers/web/hackforger/bounty.go`:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/db"
	"forgejo.org/services/context"
)

const tplBountyExplore = "hackforger/bounty/explore"
const tplBountyNew = "hackforger/bounty/new"

// ExploreBounties renders the bounty explore page.
func ExploreBounties(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.bounties")
	ctx.Data["PageIsExploreBounties"] = true

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	var statusFilter *hackforger_model.BountyStatus
	if s := ctx.FormString("status"); s != "" {
		switch s {
		case "open":
			st := hackforger_model.BountyStatusOpen; statusFilter = &st
		case "claimed":
			st := hackforger_model.BountyStatusClaimed; statusFilter = &st
		case "in_review":
			st := hackforger_model.BountyStatusInReview; statusFilter = &st
		case "completed":
			st := hackforger_model.BountyStatusCompleted; statusFilter = &st
		}
	}

	bounties, total, err := hackforger_model.ListBounties(ctx, hackforger_model.ListBountiesOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: 20},
		Status:      statusFilter,
	})
	if err != nil {
		ctx.ServerError("ListBounties", err)
		return
	}

	ctx.Data["Bounties"] = bounties
	ctx.Data["Total"] = total
	ctx.Data["StatusFilter"] = ctx.FormString("status")

	pager := context.NewPagination(int(total), 20, page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplBountyExplore)
}
```

- [ ] **Step 2: Move ExploreBounties out of hackathon.go**

In `routers/web/hackforger/hackathon.go`, remove the `ExploreBounties` function (it's now in `bounty.go`).

- [ ] **Step 3: Create explore template**

Create `templates/hackforger/bounty/explore.tmpl`:

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content explore">
	{{template "explore/navbar" .}}
	<div class="ui container">
		<div class="ui secondary menu">
			<a class="{{if eq .StatusFilter ""}}active {{end}}item" href="{{AppSubUrl}}/explore/bounties">{{ctx.Locale.Tr "hackforger.bounty.filter.all"}}</a>
			<a class="{{if eq .StatusFilter "open"}}active {{end}}item" href="{{AppSubUrl}}/explore/bounties?status=open">{{ctx.Locale.Tr "hackforger.bounty.status.open"}}</a>
			<a class="{{if eq .StatusFilter "claimed"}}active {{end}}item" href="{{AppSubUrl}}/explore/bounties?status=claimed">{{ctx.Locale.Tr "hackforger.bounty.status.claimed"}}</a>
			<a class="{{if eq .StatusFilter "completed"}}active {{end}}item" href="{{AppSubUrl}}/explore/bounties?status=completed">{{ctx.Locale.Tr "hackforger.bounty.status.completed"}}</a>
		</div>

		{{if .Bounties}}
		<div class="ui relaxed divided list">
			{{range .Bounties}}
			<div class="item">
				<div class="content">
					<a class="header" href="#">{{.Title}}</a>
					<div class="description">
						<span class="ui label {{if eq .Status 0}}green{{else if eq .Status 1}}blue{{else if eq .Status 3}}purple{{end}}">
							{{if eq .Status 0}}{{ctx.Locale.Tr "hackforger.bounty.status.open"}}
							{{else if eq .Status 1}}{{ctx.Locale.Tr "hackforger.bounty.status.claimed"}}
							{{else if eq .Status 2}}{{ctx.Locale.Tr "hackforger.bounty.status.in_review"}}
							{{else if eq .Status 3}}{{ctx.Locale.Tr "hackforger.bounty.status.completed"}}
							{{else if eq .Status 4}}{{ctx.Locale.Tr "hackforger.bounty.status.paid"}}
							{{else if eq .Status 5}}{{ctx.Locale.Tr "hackforger.bounty.status.expired"}}
							{{else if eq .Status 6}}{{ctx.Locale.Tr "hackforger.bounty.status.cancelled"}}
							{{end}}
						</span>
					</div>
				</div>
			</div>
			{{end}}
		</div>
		{{template "base/paginate" .}}
		{{else}}
		<div class="ui placeholder segment">
			<div class="ui icon header">
				{{svg "octicon-gift" 48}}
				<br>
				{{ctx.Locale.Tr "hackforger.bounty.explore.empty"}}
			</div>
		</div>
		{{end}}
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 4: Add i18n keys**

Append to `options/locale/locale_en-US.ini` under the `[hackforger]` section:

```ini
; Bounty explore
hackforger.bounty.filter.all = All
hackforger.bounty.explore.empty = No bounties yet. Create one from a repository issue!
hackforger.bounty.explore.title = Explore Bounties

; Bounty panel
hackforger.bounty.panel.title = Bounty
hackforger.bounty.panel.rewards = Rewards
hackforger.bounty.panel.deadline = Deadline
hackforger.bounty.panel.claimer = Claimed by
hackforger.bounty.panel.apply = Apply
hackforger.bounty.panel.apply.message = Why should you be assigned?
hackforger.bounty.panel.applications = Applications
hackforger.bounty.panel.accept = Accept
hackforger.bounty.panel.reject = Reject
hackforger.bounty.panel.complete = Complete
hackforger.bounty.panel.reject_delivery = Reject Delivery
hackforger.bounty.panel.mark_paid = Mark Paid
hackforger.bounty.panel.cancel = Cancel Bounty
hackforger.bounty.panel.start_review = Start Review
hackforger.bounty.panel.select_winners = Select Winners
hackforger.bounty.panel.no_applications = No applications yet

; Bounty create form
hackforger.bounty.new.title = Create Bounty
hackforger.bounty.new.issue = Issue
hackforger.bounty.new.mode = Mode
hackforger.bounty.new.mode.exclusive = Exclusive (one person claims and delivers)
hackforger.bounty.new.mode.competitive = Competitive (multiple entries, select winners)
hackforger.bounty.new.deadline = Deadline
hackforger.bounty.new.submit = Create Bounty

; Bounty badge
hackforger.bounty.badge.bounty = Bounty
```

- [ ] **Step 5: Register web route**

In `routers/web/web.go`, update the explore section. The P0 registration already calls `hackforger_web.ExploreBounties` from the explore group. Since we moved it to `bounty.go` in the same package, no route change needed — just verify it compiles.

- [ ] **Step 6: Verify compilation and page renders**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && TAGS="bindata sqlite sqlite_unlock_notify" make backend`

Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add routers/web/hackforger/bounty.go routers/web/hackforger/hackathon.go templates/hackforger/bounty/explore.tmpl options/locale/locale_en-US.ini
git commit -m "feat(bounty): add Bounty explore page with status filters

Replace P0 placeholder with full bounty listing page.
Add i18n keys for bounty UI."
```

---

### Task 11: Issue Template Injections (Panel + Badge)

**Files:**
- Create: `templates/hackforger/bounty/panel.tmpl`
- Create: `templates/hackforger/bounty/badge.tmpl`
- Modify: `templates/repo/issue/view_content/sidebar.tmpl`
- Modify: `templates/shared/issuelist.tmpl`
- Modify: `routers/web/repo/issue.go`

- [ ] **Step 1: Create panel template**

Create `templates/hackforger/bounty/panel.tmpl`:

```html
{{if .BountyData}}
<div class="divider"></div>
<div class="ui segment">
	<div class="ui small header">
		{{svg "octicon-gift" 16}} {{ctx.Locale.Tr "hackforger.bounty.panel.title"}}
		<span class="ui label {{if eq .BountyData.Status 0}}green{{else if eq .BountyData.Status 1}}blue{{else if eq .BountyData.Status 2}}orange{{else if eq .BountyData.Status 3}}purple{{else if eq .BountyData.Status 5}}red{{end}}">
			{{if eq .BountyData.Status 0}}{{ctx.Locale.Tr "hackforger.bounty.status.open"}}
			{{else if eq .BountyData.Status 1}}{{ctx.Locale.Tr "hackforger.bounty.status.claimed"}}
			{{else if eq .BountyData.Status 2}}{{ctx.Locale.Tr "hackforger.bounty.status.in_review"}}
			{{else if eq .BountyData.Status 3}}{{ctx.Locale.Tr "hackforger.bounty.status.completed"}}
			{{else if eq .BountyData.Status 4}}{{ctx.Locale.Tr "hackforger.bounty.status.paid"}}
			{{else if eq .BountyData.Status 5}}{{ctx.Locale.Tr "hackforger.bounty.status.expired"}}
			{{else if eq .BountyData.Status 6}}{{ctx.Locale.Tr "hackforger.bounty.status.cancelled"}}
			{{end}}
		</span>
	</div>

	{{if .BountyRewards}}
	<div class="tw-mb-2">
		<strong>{{ctx.Locale.Tr "hackforger.bounty.panel.rewards"}}:</strong>
		{{range .BountyRewards}}
			<span class="ui mini label">
				{{if eq .Type "money"}}{{.Currency}} {{.Amount}}{{end}}
				{{if eq .Type "credits"}}{{.Credits}} credits{{end}}
				{{if eq .Type "other"}}{{.Note}}{{end}}
			</span>
		{{end}}
	</div>
	{{end}}

	{{if .BountyData.Deadline}}
	<div class="tw-mb-2">
		<strong>{{ctx.Locale.Tr "hackforger.bounty.panel.deadline"}}:</strong>
		{{DateTime "long" .BountyData.Deadline}}
	</div>
	{{end}}

	<div id="hackforger-bounty-panel"
		data-bounty-id="{{.BountyData.ID}}"
		data-repo-owner="{{.Repository.OwnerName}}"
		data-repo-name="{{.Repository.Name}}"
		data-status="{{.BountyData.Status}}"
		data-mode="{{.BountyData.Mode}}"
		data-is-publisher="{{if and $.SignedUser (eq $.SignedUser.ID .BountyData.PublisherID)}}true{{else}}false{{end}}"
		data-claimer-id="{{.BountyData.ClaimerID}}">
	</div>
</div>
{{end}}
```

- [ ] **Step 2: Create badge template**

Create `templates/hackforger/bounty/badge.tmpl`:

```html
{{if .BountyBadge}}
<span class="ui mini label tw-ml-1 {{if eq .BountyBadge.Status 0}}green{{else if eq .BountyBadge.Status 1}}blue{{else if eq .BountyBadge.Status 3}}purple{{end}}" title="{{ctx.Locale.Tr "hackforger.bounty.badge.bounty"}}">
	{{svg "octicon-gift" 12}} {{ctx.Locale.Tr "hackforger.bounty.badge.bounty"}}
</span>
{{end}}
```

- [ ] **Step 3: Inject panel in issue sidebar**

In `templates/repo/issue/view_content/sidebar.tmpl`, after the assignees section and before the participants section, add:

```html
{{template "hackforger/bounty/panel" .}}
```

- [ ] **Step 4: Inject badge in issue list**

In `templates/shared/issuelist.tmpl`, after the issue title `</a>` tag and before the commit statuses, add:

```html
{{if .BountyBadge}}{{template "hackforger/bounty/badge" dict "BountyBadge" .BountyBadge}}{{end}}
```

- [ ] **Step 5: Load bounty data in issue handler**

In `routers/web/repo/issue.go` ViewIssue function, after loading the issue and before rendering, add:

```go
// Load HackForger Bounty data if linked
bountyData, err := hackforger_model.GetBountyByIssueID(ctx, issue.ID)
if err != nil && !hackforger_model.IsErrBountyNotExist(err) {
    ctx.ServerError("GetBountyByIssueID", err)
    return
}
if bountyData != nil {
    ctx.Data["BountyData"] = bountyData
    rewards, err := hackforger_model.ListBountyRewards(ctx, bountyData.ID)
    if err != nil {
        ctx.ServerError("ListBountyRewards", err)
        return
    }
    ctx.Data["BountyRewards"] = rewards
}
```

For the issue list handler, add batch bounty loading (query bounties for all displayed issue IDs and attach as `.BountyBadge` on each issue).

- [ ] **Step 6: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && TAGS="bindata sqlite sqlite_unlock_notify" make backend`

Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add templates/hackforger/bounty/panel.tmpl templates/hackforger/bounty/badge.tmpl templates/repo/issue/view_content/sidebar.tmpl templates/shared/issuelist.tmpl routers/web/repo/issue.go
git commit -m "feat(bounty): inject Bounty panel in Issue sidebar and badge in Issue list

Two upstream template changes (~3 lines each) plus data loading in
ViewIssue handler."
```

---

### Task 12: Vue BountyPanel Component

**Files:**
- Create/Replace: `web_src/js/components/hackforger/BountyPanel.vue`
- Verify: `web_src/js/features/hackforger/init.js` (already has mount code)

- [ ] **Step 1: Implement BountyPanel.vue**

This Vue component handles interactive actions (apply, accept/reject, complete, etc.) using Forgejo's fetch API. It reads initial state from `data-*` attributes passed as props, calls Bounty API endpoints, and updates the panel locally without full page reload.

Key features:
- Shows application form (for non-publisher users on Open bounties)
- Shows application list with accept/reject buttons (for publisher)
- Shows complete/reject-delivery buttons (for publisher on InReview)
- Shows select-winners form (for publisher on Competitive InReview)
- Shows mark-paid button (for publisher on Completed)
- Shows cancel button (for publisher on Open/Claimed)

Uses Vue 3 Options API, Fomantic UI classes, `tw-` Tailwind prefix.

- [ ] **Step 2: Update init.js to pass all required props**

The P0 `init.js` only passes `bountyId`. Update to pass all data attributes:

```javascript
export function initHackforger() {
  const bountyEl = document.getElementById('hackforger-bounty-panel');
  if (bountyEl) {
    (async () => {
      const {default: BountyPanel} = await import(
        /* webpackChunkName: "hackforger-bounty" */
        '../../components/hackforger/BountyPanel.vue'
      );
      const {createApp} = await import('vue');
      createApp(BountyPanel, {
        bountyId: bountyEl.getAttribute('data-bounty-id'),
        repoOwner: bountyEl.getAttribute('data-repo-owner'),
        repoName: bountyEl.getAttribute('data-repo-name'),
        status: Number(bountyEl.getAttribute('data-status')),
        mode: Number(bountyEl.getAttribute('data-mode')),
        isPublisher: bountyEl.getAttribute('data-is-publisher') === 'true',
        claimerId: Number(bountyEl.getAttribute('data-claimer-id')),
      }).mount(bountyEl);
    })();
  }
}
```

- [ ] **Step 3: Build frontend**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && make frontend`

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add web_src/js/components/hackforger/BountyPanel.vue web_src/js/features/hackforger/init.js
git commit -m "feat(bounty): implement interactive BountyPanel Vue component

Handles: apply, accept/reject applications, complete, reject delivery,
select winners, mark paid, cancel. Uses Options API + Fomantic UI."
```

---

## Chunk 5: Testing + E2E

### Task 13: Run Full Test Suite

- [ ] **Step 1: Run all hackforger model tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./models/hackforger/ -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: ALL PASS

- [ ] **Step 2: Run all hackforger service tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./services/hackforger/ -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: ALL PASS

- [ ] **Step 3: Run full backend build**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && TAGS="bindata sqlite sqlite_unlock_notify" make backend`

Expected: no errors

- [ ] **Step 4: Run frontend build**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && make frontend`

Expected: no errors

---

### Task 14: Create Manual E2E Test Prompt

**Files:**
- Create: `docs/tests/e2e/p1-bounty-e2e-prompt.md`
- Create: `docs/tests/e2e/p1-bounty-e2e-report.md`

- [ ] **Step 1: Write E2E prompt**

Following P0 format, create step-by-step manual test guide covering:

1. **Prerequisites**: P0 infrastructure verified, users/orgs/repos from P0 E2E exist
2. **Exclusive Bounty Flow**: Create bounty on `acme-dev/backend` issue → add rewards → apply (eve + agent_hunter) → accept eve → create PR + merge → verify InReview → complete → verify credits → mark paid → verify entity feed lifecycle
3. **Competitive Bounty Flow**: Create competitive bounty → multiple entries → select winners → verify ranked credits
4. **Feed Verification Checkpoints**: At each step, verify feed events appear for correct audiences
5. **Explore Page**: Verify `/explore/bounties` shows created bounties with filters
6. **Issue Panel**: Verify Bounty panel appears in Issue sidebar with correct status
7. **Issue Badge**: Verify Bounty badge appears in Issue list

- [ ] **Step 2: Write empty report template**

Following P0 format, create report template with checklist sections matching the prompt.

- [ ] **Step 3: Commit**

```bash
git add docs/tests/e2e/p1-bounty-e2e-prompt.md docs/tests/e2e/p1-bounty-e2e-report.md
git commit -m "docs: add Phase 1 Bounty manual E2E test prompt and report template"
```

---

### Task 15: Add Missing Service Functions (Leaderboard, Stats)

**Files:**
- Modify: `services/hackforger/bounty.go`

- [ ] **Step 1: Implement GetBountyLeaderboard**

Append to `services/hackforger/bounty.go`:

```go
// HunterStats holds aggregated bounty stats for a user.
type HunterStats struct {
	UserID          int64 `json:"user_id"`
	BountiesCompleted int  `json:"bounties_completed"`
	TotalCredits    int64 `json:"total_credits"`
}

// GetBountyLeaderboard returns hunters ranked by completed bounties.
func GetBountyLeaderboard(ctx context.Context, limit int) ([]*HunterStats, error) {
	var stats []*HunterStats
	err := db.GetEngine(ctx).SQL(
		"SELECT claimer_id AS user_id, COUNT(*) AS bounties_completed "+
			"FROM bounty WHERE status IN (?, ?) AND claimer_id > 0 "+
			"GROUP BY claimer_id ORDER BY bounties_completed DESC LIMIT ?",
		hackforger_model.BountyStatusCompleted, hackforger_model.BountyStatusPaid, limit,
	).Find(&stats)
	return stats, err
}

// BountyStatsResult holds platform-wide bounty statistics.
type BountyStatsResult struct {
	Total     int64 `json:"total"`
	Open      int64 `json:"open"`
	Completed int64 `json:"completed"`
	Paid      int64 `json:"paid"`
}

// GetBountyStats returns platform-wide bounty statistics.
func GetBountyStats(ctx context.Context) (*BountyStatsResult, error) {
	total, err := db.GetEngine(ctx).Count(new(hackforger_model.Bounty))
	if err != nil {
		return nil, err
	}
	open, err := db.GetEngine(ctx).Where("status = ?", hackforger_model.BountyStatusOpen).Count(new(hackforger_model.Bounty))
	if err != nil {
		return nil, err
	}
	completed, err := db.GetEngine(ctx).Where("status = ?", hackforger_model.BountyStatusCompleted).Count(new(hackforger_model.Bounty))
	if err != nil {
		return nil, err
	}
	paid, err := db.GetEngine(ctx).Where("status = ?", hackforger_model.BountyStatusPaid).Count(new(hackforger_model.Bounty))
	if err != nil {
		return nil, err
	}
	return &BountyStatsResult{Total: total, Open: open, Completed: completed, Paid: paid}, nil
}
```

- [ ] **Step 2: Add test for leaderboard**

Append to `services/hackforger/bounty_test.go`:

```go
func TestGetBountyLeaderboard(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Complete a bounty first so leaderboard has data
	b, _ := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	b.Status = hackforger_model.BountyStatusCompleted
	b.ClaimerID = 4
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, b))

	stats, err := GetBountyLeaderboard(db.DefaultContext, 10)
	require.NoError(t, err)
	assert.NotEmpty(t, stats)
	assert.Equal(t, int64(4), stats[0].UserID)
}
```

- [ ] **Step 3: Run test, commit**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./services/hackforger/ -run TestGetBountyLeaderboard -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

```bash
git add services/hackforger/bounty.go services/hackforger/bounty_test.go
git commit -m "feat(bounty): add GetBountyLeaderboard and GetBountyStats functions"
```

---

### Task 16: Add Missing Service Tests

**Files:**
- Modify: `services/hackforger/bounty_test.go`

- [ ] **Step 1: Add comprehensive flow tests**

Append to `services/hackforger/bounty_test.go`:

```go
func TestExclusiveBountyFlow_Happy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Apply
	require.NoError(t, ApplyForBounty(db.DefaultContext, 1, 5, "I can do it"))

	// Accept → Claimed
	pending := hackforger_model.ApplicationStatusPending
	apps, _, _ := hackforger_model.ListBountyApplications(db.DefaultContext, hackforger_model.ListBountyApplicationsOptions{
		BountyID: 1, UserID: 5, Status: &pending,
	})
	require.Len(t, apps, 1)
	require.NoError(t, AcceptApplication(db.DefaultContext, apps[0].ID, 2))

	b, _ := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, b.Status)
	assert.Equal(t, int64(5), b.ClaimerID)

	// Simulate PR merge → InReview
	b.Status = hackforger_model.BountyStatusInReview
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, b))

	// Complete → credits deposited
	require.NoError(t, CompleteBounty(db.DefaultContext, 1, 2))
	b, _ = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	assert.Equal(t, hackforger_model.BountyStatusCompleted, b.Status)

	// Mark paid
	require.NoError(t, MarkPaid(db.DefaultContext, 1, 2))
	b, _ = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	assert.Equal(t, hackforger_model.BountyStatusPaid, b.Status)
}

func TestCompetitiveBountyFlow_Happy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Bounty 2 is competitive + open
	require.NoError(t, ApplyForBounty(db.DefaultContext, 2, 4, "entry 1"))
	require.NoError(t, ApplyForBounty(db.DefaultContext, 2, 5, "entry 2"))

	// Start review
	require.NoError(t, StartReview(db.DefaultContext, 2, 2))
	b, _ := hackforger_model.GetBountyByID(db.DefaultContext, 2)
	assert.Equal(t, hackforger_model.BountyStatusInReview, b.Status)

	// Select winners
	require.NoError(t, SelectWinners(db.DefaultContext, 2, 2, []WinnerInput{
		{UserID: 4, Rank: 1},
		{UserID: 5, Rank: 2},
	}))
	b, _ = hackforger_model.GetBountyByID(db.DefaultContext, 2)
	assert.Equal(t, hackforger_model.BountyStatusCompleted, b.Status)

	winners, _ := hackforger_model.ListBountyWinners(db.DefaultContext, 2)
	assert.Len(t, winners, 2)
}

func TestBountyApplication_AcceptRejectsOthers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Two applications for bounty 1
	require.NoError(t, ApplyForBounty(db.DefaultContext, 1, 5, "me"))
	require.NoError(t, ApplyForBounty(db.DefaultContext, 1, 6, "me too"))

	// Accept user 5's application
	pending := hackforger_model.ApplicationStatusPending
	apps, _, _ := hackforger_model.ListBountyApplications(db.DefaultContext, hackforger_model.ListBountyApplicationsOptions{
		BountyID: 1, UserID: 5, Status: &pending,
	})
	require.NoError(t, AcceptApplication(db.DefaultContext, apps[0].ID, 2))

	// User 4's original fixture application + user 6 should be rejected
	rejected := hackforger_model.ApplicationStatusRejected
	rejectedApps, count, _ := hackforger_model.ListBountyApplications(db.DefaultContext, hackforger_model.ListBountyApplicationsOptions{
		BountyID: 1, Status: &rejected,
	})
	assert.Equal(t, int64(2), count)
	_ = rejectedApps
}

func TestCheckExpiredBounties(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 deadline to past
	b, _ := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	b.Deadline = 1 // Unix epoch + 1 second (far in the past)
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, b))

	require.NoError(t, CheckExpiredBounties(db.DefaultContext))

	b, _ = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	assert.Equal(t, hackforger_model.BountyStatusExpired, b.Status)
}

func TestCompleteBounty_NoCreditsReward(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create a bounty with only money reward (no credits)
	bounty := &hackforger_model.Bounty{
		RepoID: 1, IssueID: 200, PublisherID: 2, ClaimerID: 4,
		Title: "money only", Status: hackforger_model.BountyStatusInReview,
		Mode: hackforger_model.BountyModeExclusive,
	}
	require.NoError(t, hackforger_model.CreateBounty(db.DefaultContext, bounty))
	require.NoError(t, hackforger_model.CreateBountyReward(db.DefaultContext, &hackforger_model.BountyReward{
		BountyID: bounty.ID, Type: hackforger_model.RewardTypeMoney, Amount: 100, Currency: "USD",
	}))

	// Get initial credit balance
	acct, _ := GetOrCreateCreditAccount(db.DefaultContext, 4)
	initialBalance := acct.Balance

	require.NoError(t, CompleteBounty(db.DefaultContext, bounty.ID, 2))

	acct, _ = GetOrCreateCreditAccount(db.DefaultContext, 4)
	assert.Equal(t, initialBalance, acct.Balance, "balance should not change with money-only reward")
}
```

- [ ] **Step 2: Run all service tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-bounty && go test ./services/hackforger/ -v -count=1 -tags "bindata sqlite sqlite_unlock_notify"`

Expected: ALL PASS

- [ ] **Step 3: Commit**

```bash
git add services/hackforger/bounty_test.go
git commit -m "test(bounty): add comprehensive flow tests per test plan

Covers: ExclusiveBountyFlow_Happy, CompetitiveBountyFlow_Happy,
AcceptRejectsOthers, CheckExpiredBounties, NoCreditsReward."
```

---

### Task 17: Add NewBounty Web Handler and Template

**Files:**
- Modify: `routers/web/hackforger/bounty.go`
- Create: `templates/hackforger/bounty/new.tmpl`
- Modify: `routers/web/web.go`

- [ ] **Step 1: Add NewBounty and NewBountyPost handlers**

Append to `routers/web/hackforger/bounty.go`:

```go
// NewBounty renders the create bounty form.
func NewBounty(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.bounty.new.title")
	ctx.HTML(http.StatusOK, tplBountyNew)
}

// NewBountyPost handles the create bounty form submission.
func NewBountyPost(ctx *context.Context) {
	issueID := ctx.FormInt64("issue_id")
	if issueID <= 0 {
		ctx.Flash.Error("Issue is required")
		ctx.Redirect(ctx.Repo.RepoLink + "/bounties/new")
		return
	}

	mode := hackforger_model.BountyMode(ctx.FormInt("mode"))
	deadline := timeutil.TimeStamp(ctx.FormInt64("deadline"))

	bounty := &hackforger_model.Bounty{
		RepoID:      ctx.Repo.Repository.ID,
		IssueID:     issueID,
		PublisherID: ctx.Doer.ID,
		Title:       ctx.FormString("title"),
		Mode:        mode,
		Deadline:    deadline,
	}

	if err := hackforger_model.CreateBounty(ctx, bounty); err != nil {
		if hackforger_model.IsErrBountyAlreadyExists(err) {
			ctx.Flash.Error("A bounty already exists for this issue")
			ctx.Redirect(ctx.Repo.RepoLink + "/bounties/new")
			return
		}
		ctx.ServerError("CreateBounty", err)
		return
	}

	// Publish feed event
	_ = hackforger_svc.PublishHackforgerAction(ctx, &hackforger_svc.HackforgerActionOpts{
		ActUserID:    ctx.Doer.ID,
		OpType:       hackforger_model.ActionBountyCreated,
		RepoID:       bounty.RepoID,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: bounty.ID, EntityName: bounty.Title},
		AudienceType: hackforger_svc.AudienceGlobal | hackforger_svc.AudienceRepoWatchers,
	})

	ctx.Flash.Success("Bounty created successfully")
	ctx.Redirect(fmt.Sprintf("%s/issues/%d", ctx.Repo.RepoLink, bounty.IssueID))
}
```

- [ ] **Step 2: Create new.tmpl**

Create `templates/hackforger/bounty/new.tmpl` with a form containing: issue selector dropdown, mode radio buttons (Exclusive/Competitive), title input, deadline date picker, submit button. Uses Fomantic UI form classes and i18n keys.

- [ ] **Step 3: Register web routes**

In `routers/web/web.go`, add inside the repo group:

```go
// HackForger Bounty web routes
m.Group("/bounties", func() {
    m.Combo("/new").Get(hackforger_web.NewBounty).
        Post(web.Bind(forms.CreateIssueForm{}), hackforger_web.NewBountyPost)
}, reqSignIn, context.RepoMustNotBeArchived(), reqRepoIssueWriter)
```

- [ ] **Step 4: Verify compilation, commit**

```bash
git add routers/web/hackforger/bounty.go templates/hackforger/bounty/new.tmpl routers/web/web.go
git commit -m "feat(bounty): add NewBounty web form for creating bounties from repo UI"
```

---

## Summary

| Chunk | Tasks | Key Deliverables |
|-------|-------|-----------------|
| 1: Model | Tasks 1-6 | Fixtures, BountyStatus 7-state, BountyMode 2-mode, migration, CRUD for Application/Reward/Winner, ActionBountyPaid |
| 2: Service | Tasks 7-8, 15-16 | Notifier bitmask rewrite (shared infra), MergePullRequest hook, Bounty state machine (14 functions), Leaderboard, Stats, comprehensive flow tests |
| 3: API | Task 9 | 20 REST endpoints with Swagger annotations |
| 4: Web+Frontend | Tasks 10-12, 17 | Explore page, NewBounty form, Issue panel injection, Issue badge injection, BountyPanel.vue |
| 5: Testing+E2E | Tasks 13-14 | Full test suite run, manual E2E prompt/report |
