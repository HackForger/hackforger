# Phase 1 Grant + Credits Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Grant Round full-stack and Credits full-stack (Line C of HackForger parallel development).

**Architecture:** Model-first bottom-up. All new code in `*/hackforger/` directories. Grant service owns state machine + budget validation. Credits service handles transactional deposit/redeem. Feed events via `PublishHackforgerAction` calls from service layer. SSR-first web pages.

**Tech Stack:** Go 1.23, XORM ORM, go-chi router, Go HTML templates, Fomantic UI CSS, SQLite (dev).

**Spec:** `docs/superpowers/specs/2026-03-28-phase1-grant-design.md`

---

## File Map

### Files to Modify (existing P0 code)

| File | Change |
|------|--------|
| `models/hackforger/grant_round.go` | Fix status enum (6 states), add GetBySlug/Delete/BudgetUsage |
| `models/hackforger/grant_project.go` | Add GetByUserAndRound/Delete/Count, new error types |
| `models/hackforger/credit_account.go` | Add GetCreditAccount/ListCreditAccounts |
| `models/hackforger/credit_transaction.go` | Add ListAllTransactions |
| `models/hackforger/redeem_option.go` | Add Create/Update/GetByID |
| `models/hackforger/redeem_order.go` | Add GetByID/Update |
| `models/hackforger/action_types.go` | Add ActionGrantRoundCancelled = 57 |
| `services/hackforger/credits.go` | Append AdminDeposit/AdminDeduct/FulfillOrder/CancelOrder/CreateRedeemOption/UpdateRedeemOption |
| `routers/api/v1/hackforger/grants.go` | Replace stub with full CRUD + state operations |
| `routers/api/v1/hackforger/credits.go` | Replace stub with full API |
| `routers/api/v1/api.go:1739-1746` | Expand `/hackforger` group with sub-groups |
| `routers/web/web.go:504-506` | Keep explore route, add `/grants`, `/credits`, `/-/admin/credits` groups |
| `routers/web/hackforger/hackathon.go` | Remove ExploreGrants (moved to grants.go) |
| `templates/hackforger/explore.tmpl` | Add grants tab content |
| `options/locale/locale_en-US.ini` | Append grant + credits i18n keys |

### Files to Create

| File | Responsibility |
|------|---------------|
| `services/hackforger/grants.go` | Grant state machine, permission checks, all business logic |
| `routers/web/hackforger/grants.go` | Web handlers for grant pages |
| `routers/web/hackforger/credits.go` | Web handlers for credits pages |
| `templates/hackforger/grants/explore.tmpl` | Grant list (for explore page) |
| `templates/hackforger/grants/new.tmpl` | Create grant round form |
| `templates/hackforger/grants/detail.tmpl` | Round detail page |
| `templates/hackforger/grants/projects.tmpl` | Public project list |
| `templates/hackforger/grants/submit.tmpl` | Submit project form |
| `templates/hackforger/grants/manage.tmpl` | Management panel |
| `templates/hackforger/grants/manage_project.tmpl` | Single project review |
| `templates/hackforger/credits/overview.tmpl` | Balance + transactions + redeem |
| `templates/hackforger/credits/redeem.tmpl` | Redeem confirmation |
| `templates/hackforger/credits/orders.tmpl` | User order list |
| `templates/hackforger/credits/admin/credits.tmpl` | Admin credits management |
| `templates/hackforger/credits/admin/options.tmpl` | Admin redeem options |
| `templates/hackforger/credits/admin/orders.tmpl` | Admin orders |
| `models/hackforger/grant_round_test.go` | Grant round model tests |
| `models/hackforger/grant_project_test.go` | Grant project model tests |
| `models/hackforger/credits_test.go` | Credits model tests |
| `services/hackforger/grants_test.go` | Grant service tests |
| `services/hackforger/credits_test.go` | Credits service tests |
| `models/fixtures/grant_round.yml` | Test fixtures |
| `models/fixtures/grant_project.yml` | Test fixtures |
| `models/fixtures/credit_account.yml` | Test fixtures |
| `models/fixtures/credit_transaction.yml` | Test fixtures |
| `models/fixtures/redeem_option.yml` | Test fixtures |
| `models/fixtures/redeem_order.yml` | Test fixtures |
| `docs/tests/e2e/phase1-grant-e2e.md` | Manual E2E prompt + report template |

---

## Chunk 1: Model Layer — Grant Round

### Task 1: Fix GrantRound Status Enum

**Files:**
- Modify: `models/hackforger/grant_round.go:18-26`

- [ ] **Step 1: Update status enum**

In `models/hackforger/grant_round.go`, replace the existing status constants:

```go
// GrantRoundStatus represents the status of a grant round.
type GrantRoundStatus int

const (
	GrantRoundStatusDraft       GrantRoundStatus = iota // 0
	GrantRoundStatusOpen                                // 1
	GrantRoundStatusReview                              // 2
	GrantRoundStatusFinalized                           // 3 - renamed from Complete
	GrantRoundStatusDistributed                         // 4 - new
	GrantRoundStatusCancelled                           // 5 - re-numbered
)

// GrantRoundStatusNames maps status to display name.
var GrantRoundStatusNames = map[GrantRoundStatus]string{
	GrantRoundStatusDraft:       "draft",
	GrantRoundStatusOpen:        "open",
	GrantRoundStatusReview:      "review",
	GrantRoundStatusFinalized:   "finalized",
	GrantRoundStatusDistributed: "distributed",
	GrantRoundStatusCancelled:   "cancelled",
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go build ./models/hackforger/...`
Expected: Success (no references to old `GrantRoundStatusComplete` exist in codebase)

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/grant_round.go
git commit -m "refactor(grant): update GrantRound status enum to 6 states

Rename Complete->Finalized, add Distributed, re-number Cancelled.
No migration needed: no production data exists yet."
```

### Task 2: Add GrantRound CRUD Functions + Error Types

**Files:**
- Modify: `models/hackforger/grant_round.go`
- Create: `models/fixtures/grant_round.yml`
- Create: `models/hackforger/grant_round_test.go`

- [ ] **Step 1: Create test fixtures**

Create `models/fixtures/grant_round.yml`:

```yaml
# grant_round.yml - Test fixture data
-
  id: 1
  org_id: 3       # org from existing fixtures
  owner_id: 2     # user 2
  name: "Test Grant Round Draft"
  slug: "test-grant-draft"
  description: "A draft grant round for testing"
  status: 0       # Draft
  budget: 10000.00
  currency: "USD"
  budget_credits: 5000
  deadline: 0
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 2
  org_id: 3
  owner_id: 2
  name: "Test Grant Round Open"
  slug: "test-grant-open"
  description: "An open grant round for testing"
  status: 1       # Open
  budget: 20000.00
  currency: "USD"
  budget_credits: 10000
  deadline: 1735689600
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 3
  org_id: 3
  owner_id: 2
  name: "Test Grant Round Review"
  slug: "test-grant-review"
  description: "A reviewing grant round"
  status: 2       # Review
  budget: 15000.00
  currency: "USD"
  budget_credits: 8000
  deadline: 1735689600
  created_unix: 1609459200
  updated_unix: 1609459200
```

- [ ] **Step 2: Write failing tests**

Create `models/hackforger/grant_round_test.go`:

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

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestCreateGrantRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "New Round", Slug: "new-round",
		Description: "test", Status: hackforger_model.GrantRoundStatusDraft,
		Budget: 5000, Currency: "USD", BudgetCredits: 2000,
	}
	err := hackforger_model.CreateGrantRound(db.DefaultContext, round)
	require.NoError(t, err)
	assert.Greater(t, round.ID, int64(0))
	assert.Equal(t, hackforger_model.GrantRoundStatusDraft, round.Status)
}

func TestGetGrantRoundBySlug(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	round, err := hackforger_model.GetGrantRoundBySlug(db.DefaultContext, "test-grant-draft")
	require.NoError(t, err)
	assert.Equal(t, int64(1), round.ID)
	assert.Equal(t, "Test Grant Round Draft", round.Name)
	assert.Equal(t, hackforger_model.GrantRoundStatusDraft, round.Status)

	// non-existent slug
	_, err = hackforger_model.GetGrantRoundBySlug(db.DefaultContext, "non-existent")
	assert.True(t, hackforger_model.IsErrGrantRoundNotExist(err))
}

func TestDeleteGrantRound_OnlyDraft(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Draft round can be deleted
	err := hackforger_model.DeleteGrantRound(db.DefaultContext, 1)
	require.NoError(t, err)
	_, err = hackforger_model.GetGrantRoundByID(db.DefaultContext, 1)
	assert.True(t, hackforger_model.IsErrGrantRoundNotExist(err))

	// Open round cannot be deleted
	err = hackforger_model.DeleteGrantRound(db.DefaultContext, 2)
	assert.Error(t, err)
	assert.True(t, hackforger_model.IsErrGrantRoundNotDraft(err))
}

func TestGetGrantRoundBudgetUsage(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 3 (Review) with no projects allocated yet
	usedAmount, usedCredits, err := hackforger_model.GetGrantRoundBudgetUsage(db.DefaultContext, 3)
	require.NoError(t, err)
	assert.Equal(t, float64(0), usedAmount)
	assert.Equal(t, int64(0), usedCredits)
}

func TestListGrantRounds_FilterByStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	status := hackforger_model.GrantRoundStatusOpen
	rounds, count, err := hackforger_model.ListGrantRounds(db.DefaultContext, hackforger_model.ListGrantRoundsOptions{
		Status: &status,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, hackforger_model.GrantRoundStatusOpen, rounds[0].Status)
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./models/hackforger/... -run "TestGetGrantRoundBySlug|TestDeleteGrantRound|TestGetGrantRoundBudgetUsage|TestListGrantRounds_FilterByStatus" -v`
Expected: FAIL — `GetGrantRoundBySlug`, `DeleteGrantRound`, `GetGrantRoundBudgetUsage` undefined

- [ ] **Step 4: Implement GrantRound CRUD additions**

Append to `models/hackforger/grant_round.go`:

```go
// ErrGrantRoundNotDraft is returned when trying to delete a non-draft round.
type ErrGrantRoundNotDraft struct {
	ID     int64
	Status GrantRoundStatus
}

func IsErrGrantRoundNotDraft(err error) bool {
	_, ok := err.(ErrGrantRoundNotDraft)
	return ok
}

func (err ErrGrantRoundNotDraft) Error() string {
	return fmt.Sprintf("grant round is not in draft status [id: %d, status: %d]", err.ID, err.Status)
}

func (err ErrGrantRoundNotDraft) Unwrap() error {
	return util.ErrInvalidArgument
}

// ErrGrantRoundSlugExists is returned when a slug already exists.
type ErrGrantRoundSlugExists struct {
	Slug string
}

func IsErrGrantRoundSlugExists(err error) bool {
	_, ok := err.(ErrGrantRoundSlugExists)
	return ok
}

func (err ErrGrantRoundSlugExists) Error() string {
	return fmt.Sprintf("grant round slug already exists [slug: %s]", err.Slug)
}

func (err ErrGrantRoundSlugExists) Unwrap() error {
	return util.ErrAlreadyExist
}

// GetGrantRoundBySlug returns a grant round by its slug.
func GetGrantRoundBySlug(ctx context.Context, slug string) (*GrantRound, error) {
	r := new(GrantRound)
	has, err := db.GetEngine(ctx).Where("slug = ?", slug).Get(r)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrGrantRoundNotExist{ID: 0}
	}
	return r, nil
}

// DeleteGrantRound deletes a grant round. Only Draft rounds can be deleted.
func DeleteGrantRound(ctx context.Context, id int64) error {
	r, err := GetGrantRoundByID(ctx, id)
	if err != nil {
		return err
	}
	if r.Status != GrantRoundStatusDraft {
		return ErrGrantRoundNotDraft{ID: id, Status: r.Status}
	}
	_, err = db.GetEngine(ctx).ID(id).Delete(new(GrantRound))
	return err
}

// GetGrantRoundBudgetUsage returns the sum of allocated amounts and credits
// for approved/funded projects in a round.
func GetGrantRoundBudgetUsage(ctx context.Context, roundID int64) (float64, int64, error) {
	type result struct {
		SumAmount  float64 `xorm:"sum_amount"`
		SumCredits int64   `xorm:"sum_credits"`
	}
	var res result
	_, err := db.GetEngine(ctx).
		Table("grant_project").
		Select("COALESCE(SUM(award_amount), 0) AS sum_amount, COALESCE(SUM(award_credits), 0) AS sum_credits").
		Where("round_id = ? AND status IN (?, ?)", roundID, GrantProjectStatusApproved, GrantProjectStatusFunded).
		Get(&res)
	if err != nil {
		return 0, 0, err
	}
	return res.SumAmount, res.SumCredits, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./models/hackforger/... -run "TestGetGrantRoundBySlug|TestDeleteGrantRound|TestGetGrantRoundBudgetUsage|TestListGrantRounds_FilterByStatus" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add models/hackforger/grant_round.go models/hackforger/grant_round_test.go models/fixtures/grant_round.yml
git commit -m "feat(grant): add GrantRound GetBySlug, Delete, BudgetUsage + tests"
```

---

## Chunk 2: Model Layer — Grant Project + Credits

### Task 3: Add GrantProject CRUD + Error Types

**Files:**
- Modify: `models/hackforger/grant_project.go`
- Create: `models/fixtures/grant_project.yml`
- Create: `models/hackforger/grant_project_test.go`

- [ ] **Step 1: Create test fixtures**

Create `models/fixtures/grant_project.yml`:

```yaml
-
  id: 1
  round_id: 2       # Open round
  user_id: 4         # hacker user
  repo_id: 1
  title: "Eve's Project"
  description: "A test project submission"
  status: 0          # Pending
  award_amount: 0
  award_credits: 0
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 2
  round_id: 3       # Review round
  user_id: 4
  repo_id: 1
  title: "Eve's Approved Project"
  description: "An approved project"
  status: 1          # Approved
  award_amount: 5000.00
  award_credits: 2000
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 3
  round_id: 3       # Review round
  user_id: 5         # different user
  repo_id: 2
  title: "Frank's Approved Project"
  description: "Another approved project"
  status: 1          # Approved
  award_amount: 3000.00
  award_credits: 1500
  created_unix: 1609459200
  updated_unix: 1609459200
```

- [ ] **Step 2: Write failing tests**

Create `models/hackforger/grant_project_test.go`:

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

func TestGetGrantProjectByUserAndRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Existing project
	p, err := hackforger_model.GetGrantProjectByUserAndRound(db.DefaultContext, 4, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), p.ID)

	// Non-existent
	_, err = hackforger_model.GetGrantProjectByUserAndRound(db.DefaultContext, 999, 2)
	assert.True(t, hackforger_model.IsErrGrantProjectNotExist(err))
}

func TestDeleteGrantProject_OnlyPending(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Pending project can be deleted
	err := hackforger_model.DeleteGrantProject(db.DefaultContext, 1)
	require.NoError(t, err)

	// Approved project cannot be deleted
	err = hackforger_model.DeleteGrantProject(db.DefaultContext, 2)
	assert.Error(t, err)
}

func TestCountGrantProjectsByRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	status := hackforger_model.GrantProjectStatusApproved
	count, err := hackforger_model.CountGrantProjectsByRound(db.DefaultContext, 3, &status)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count) // projects 2 and 3
}

func TestListGrantProjectsByRound_FilterByStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	status := hackforger_model.GrantProjectStatusPending
	projects, count, err := hackforger_model.ListGrantProjectsByRound(db.DefaultContext,
		hackforger_model.ListGrantProjectsByRoundOptions{
			RoundID: 2,
			Status:  &status,
		})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, "Eve's Project", projects[0].Title)
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./models/hackforger/... -run "TestGetGrantProjectByUserAndRound|TestDeleteGrantProject|TestCountGrantProjectsByRound|TestListGrantProjectsByRound_FilterByStatus" -v`
Expected: FAIL

- [ ] **Step 4: Implement GrantProject additions**

Append to `models/hackforger/grant_project.go`:

```go
// Note: GrantProjectStatusFunded (3) is already defined in P0 code. Do NOT re-add it.

// ErrGrantProjectAlreadyExists is returned when a user already submitted to a round.
type ErrGrantProjectAlreadyExists struct {
	UserID  int64
	RoundID int64
}

func IsErrGrantProjectAlreadyExists(err error) bool {
	_, ok := err.(ErrGrantProjectAlreadyExists)
	return ok
}

func (err ErrGrantProjectAlreadyExists) Error() string {
	return fmt.Sprintf("grant project already exists [user_id: %d, round_id: %d]", err.UserID, err.RoundID)
}

func (err ErrGrantProjectAlreadyExists) Unwrap() error {
	return util.ErrAlreadyExist
}

// ErrGrantRoundNotOpen is returned when submitting to a non-open round.
type ErrGrantRoundNotOpen struct {
	RoundID int64
	Status  GrantRoundStatus
}

func IsErrGrantRoundNotOpen(err error) bool {
	_, ok := err.(ErrGrantRoundNotOpen)
	return ok
}

func (err ErrGrantRoundNotOpen) Error() string {
	return fmt.Sprintf("grant round is not open [round_id: %d, status: %d]", err.RoundID, err.Status)
}

func (err ErrGrantRoundNotOpen) Unwrap() error {
	return util.ErrInvalidArgument
}

// ErrExceedsBudget is returned when an allocation would exceed the round budget.
type ErrExceedsBudget struct {
	RoundID       int64
	BudgetField   string
	Budget        float64
	Used          float64
	Requested     float64
}

func IsErrExceedsBudget(err error) bool {
	_, ok := err.(ErrExceedsBudget)
	return ok
}

func (err ErrExceedsBudget) Error() string {
	return fmt.Sprintf("allocation exceeds budget [round_id: %d, field: %s, budget: %.2f, used: %.2f, requested: %.2f]",
		err.RoundID, err.BudgetField, err.Budget, err.Used, err.Requested)
}

func (err ErrExceedsBudget) Unwrap() error {
	return util.ErrInvalidArgument
}

// ErrUnallocatedProjects is returned when finalizing a round with unallocated approved projects.
type ErrUnallocatedProjects struct {
	RoundID int64
	Count   int64
}

func IsErrUnallocatedProjects(err error) bool {
	_, ok := err.(ErrUnallocatedProjects)
	return ok
}

func (err ErrUnallocatedProjects) Error() string {
	return fmt.Sprintf("round has unallocated approved projects [round_id: %d, count: %d]", err.RoundID, err.Count)
}

func (err ErrUnallocatedProjects) Unwrap() error {
	return util.ErrInvalidArgument
}

// GetGrantProjectByUserAndRound returns a project for a specific user in a round.
func GetGrantProjectByUserAndRound(ctx context.Context, userID, roundID int64) (*GrantProject, error) {
	p := new(GrantProject)
	has, err := db.GetEngine(ctx).Where("user_id = ? AND round_id = ?", userID, roundID).Get(p)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrGrantProjectNotExist{ID: 0}
	}
	return p, nil
}

// DeleteGrantProject deletes a project. Only Pending projects can be deleted.
func DeleteGrantProject(ctx context.Context, id int64) error {
	p, err := GetGrantProjectByID(ctx, id)
	if err != nil {
		return err
	}
	if p.Status != GrantProjectStatusPending {
		return fmt.Errorf("cannot delete grant project with status %d", p.Status)
	}
	_, err = db.GetEngine(ctx).ID(id).Delete(new(GrantProject))
	return err
}

// CountGrantProjectsByRound counts projects by round and optional status filter.
func CountGrantProjectsByRound(ctx context.Context, roundID int64, status *GrantProjectStatus) (int64, error) {
	sess := db.GetEngine(ctx).Where("round_id = ?", roundID)
	if status != nil {
		sess = sess.And("status = ?", *status)
	}
	return sess.Count(new(GrantProject))
}
```

Note: `GrantProjectStatusFunded` (3) is already defined in P0 code at `grant_project.go:24`. Do NOT add it again.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./models/hackforger/... -run "TestGetGrantProjectByUserAndRound|TestDeleteGrantProject|TestCountGrantProjectsByRound|TestListGrantProjectsByRound_FilterByStatus" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add models/hackforger/grant_project.go models/hackforger/grant_project_test.go models/fixtures/grant_project.yml
git commit -m "feat(grant): add GrantProject GetByUserAndRound, Delete, Count + error types + tests"
```

### Task 4: Add Credits Model CRUD Functions

**Files:**
- Modify: `models/hackforger/credit_account.go`
- Modify: `models/hackforger/credit_transaction.go`
- Modify: `models/hackforger/redeem_option.go`
- Modify: `models/hackforger/redeem_order.go`
- Create: `models/fixtures/credit_account.yml`
- Create: `models/fixtures/credit_transaction.yml`
- Create: `models/fixtures/redeem_option.yml`
- Create: `models/fixtures/redeem_order.yml`
- Create: `models/hackforger/credits_test.go`

- [ ] **Step 1: Create test fixtures**

Create `models/fixtures/credit_account.yml`:
```yaml
-
  id: 1
  user_id: 4
  balance: 1000
  updated_unix: 1609459200
```

Create `models/fixtures/credit_transaction.yml`:
```yaml
-
  id: 1
  user_id: 4
  type: "deposit"
  amount: 1000
  balance: 1000
  reference: "grant_award"
  note: "Grant award for project"
  created_unix: 1609459200
```

Create `models/fixtures/redeem_option.yml`:
```yaml
-
  id: 1
  name: "Cloud Credits $10"
  description: "10 USD cloud computing credits"
  cost: 500
  stock: 10
  is_active: 1
  created_unix: 1609459200
  updated_unix: 1609459200

-
  id: 2
  name: "Unlimited License"
  description: "Software license"
  cost: 2000
  stock: -1
  is_active: 1
  created_unix: 1609459200
  updated_unix: 1609459200
```

Create `models/fixtures/redeem_order.yml`:
```yaml
-
  id: 1
  user_id: 4
  option_id: 1
  cost: 500
  status: "pending"
  fulfill_note: ""
  created_unix: 1609459200
  updated_unix: 1609459200
```

- [ ] **Step 2: Write failing tests**

Create `models/hackforger/credits_test.go`:

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

func TestGetCreditAccount(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	acct, err := hackforger_model.GetCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), acct.Balance)

	// Non-existent user returns nil, no error
	acct, err = hackforger_model.GetCreditAccount(db.DefaultContext, 999)
	require.NoError(t, err)
	assert.Nil(t, acct)
}

func TestCreateRedeemOption(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	opt := &hackforger_model.RedeemOption{
		Name:        "New Option",
		Description: "Test",
		Cost:        100,
		Stock:       5,
		IsActive:    true,
	}
	err := hackforger_model.CreateRedeemOption(db.DefaultContext, opt)
	require.NoError(t, err)
	assert.Greater(t, opt.ID, int64(0))

	loaded, err := hackforger_model.GetRedeemOptionByID(db.DefaultContext, opt.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Option", loaded.Name)
}

func TestGetRedeemOrderByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(4), order.UserID)
	assert.Equal(t, hackforger_model.OrderStatusPending, order.Status)
}

func TestUpdateRedeemOrder(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)

	order.Status = hackforger_model.OrderStatusFulfilled
	order.FulfillNote = "Sent via email"
	err = hackforger_model.UpdateRedeemOrder(db.DefaultContext, order)
	require.NoError(t, err)

	updated, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, updated.Status)
	assert.Equal(t, "Sent via email", updated.FulfillNote)
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./models/hackforger/... -run "TestGetCreditAccount|TestCreateRedeemOption|TestGetRedeemOrderByID|TestUpdateRedeemOrder" -v`
Expected: FAIL

- [ ] **Step 4: Implement Credits CRUD**

Append to `models/hackforger/credit_account.go`:

```go
import (
	"context"

	"forgejo.org/models/db"

	"xorm.io/builder"
)

// GetCreditAccount returns the credit account for a user, or nil if not found.
func GetCreditAccount(ctx context.Context, userID int64) (*CreditAccount, error) {
	acct := new(CreditAccount)
	has, err := db.GetEngine(ctx).Where("user_id = ?", userID).Get(acct)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return acct, nil
}

// ListCreditAccountsOptions holds options for listing credit accounts.
type ListCreditAccountsOptions struct {
	db.ListOptions
}

func (opts ListCreditAccountsOptions) ToConds() builder.Cond {
	return builder.NewCond()
}

// ListCreditAccounts returns all credit accounts (admin).
func ListCreditAccounts(ctx context.Context, opts ListCreditAccountsOptions) ([]*CreditAccount, int64, error) {
	return db.FindAndCount[CreditAccount](ctx, opts)
}
```

Append to `models/hackforger/credit_transaction.go`:

```go
import (
	"context"

	"forgejo.org/models/db"

	"xorm.io/builder"
)

// ListAllTransactionsOptions holds options for listing all transactions.
type ListAllTransactionsOptions struct {
	db.ListOptions
	UserID int64
}

func (opts ListAllTransactionsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.UserID > 0 {
		cond = cond.And(builder.Eq{"user_id": opts.UserID})
	}
	return cond
}

// ListAllTransactions returns transactions (admin global view).
func ListAllTransactions(ctx context.Context, opts ListAllTransactionsOptions) ([]*CreditTransaction, int64, error) {
	return db.FindAndCount[CreditTransaction](ctx, opts)
}
```

Append to `models/hackforger/redeem_option.go`:

```go
import "context"

// CreateRedeemOption inserts a new redeem option.
func CreateRedeemOption(ctx context.Context, opt *RedeemOption) error {
	_, err := db.GetEngine(ctx).Insert(opt)
	return err
}

// ErrRedeemOptionNotExist represents a "RedeemOptionNotExist" kind of error.
type ErrRedeemOptionNotExist struct {
	ID int64
}

func IsErrRedeemOptionNotExist(err error) bool {
	_, ok := err.(ErrRedeemOptionNotExist)
	return ok
}

func (err ErrRedeemOptionNotExist) Error() string {
	return fmt.Sprintf("redeem option does not exist [id: %d]", err.ID)
}

func (err ErrRedeemOptionNotExist) Unwrap() error {
	return util.ErrNotExist
}

// GetRedeemOptionByID returns a redeem option by ID.
func GetRedeemOptionByID(ctx context.Context, id int64) (*RedeemOption, error) {
	opt := new(RedeemOption)
	has, err := db.GetEngine(ctx).ID(id).Get(opt)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrRedeemOptionNotExist{ID: id}
	}
	return opt, nil
}

// UpdateRedeemOption updates a redeem option.
func UpdateRedeemOption(ctx context.Context, opt *RedeemOption) error {
	_, err := db.GetEngine(ctx).ID(opt.ID).AllCols().Update(opt)
	return err
}
```

Append to `models/hackforger/redeem_order.go`:

```go
import "context"

// ErrRedeemOrderNotExist represents a "RedeemOrderNotExist" kind of error.
type ErrRedeemOrderNotExist struct {
	ID int64
}

func IsErrRedeemOrderNotExist(err error) bool {
	_, ok := err.(ErrRedeemOrderNotExist)
	return ok
}

func (err ErrRedeemOrderNotExist) Error() string {
	return fmt.Sprintf("redeem order does not exist [id: %d]", err.ID)
}

func (err ErrRedeemOrderNotExist) Unwrap() error {
	return util.ErrNotExist
}

// GetRedeemOrderByID returns a redeem order by ID.
func GetRedeemOrderByID(ctx context.Context, id int64) (*RedeemOrder, error) {
	order := new(RedeemOrder)
	has, err := db.GetEngine(ctx).ID(id).Get(order)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrRedeemOrderNotExist{ID: id}
	}
	return order, nil
}

// UpdateRedeemOrder updates a redeem order.
func UpdateRedeemOrder(ctx context.Context, order *RedeemOrder) error {
	_, err := db.GetEngine(ctx).ID(order.ID).AllCols().Update(order)
	return err
}
```

Note: Each file already has some imports. Merge the new imports with existing ones — don't duplicate. The `context` import may already exist; `xorm.io/builder` is new for some files.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./models/hackforger/... -run "TestGetCreditAccount|TestCreateRedeemOption|TestGetRedeemOrderByID|TestUpdateRedeemOrder" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add models/hackforger/credit_account.go models/hackforger/credit_transaction.go models/hackforger/redeem_option.go models/hackforger/redeem_order.go models/hackforger/credits_test.go models/fixtures/credit_account.yml models/fixtures/credit_transaction.yml models/fixtures/redeem_option.yml models/fixtures/redeem_order.yml
git commit -m "feat(credits): add CRUD functions for CreditAccount, CreditTransaction, RedeemOption, RedeemOrder + tests"
```

### Task 5: Add ActionGrantRoundCancelled to action_types.go

**Files:**
- Modify: `models/hackforger/action_types.go:35-36`

- [ ] **Step 1: Add new action type**

In `models/hackforger/action_types.go`, after line 35 (`ActionGrantRoundFinalized`), add:

```go
ActionGrantRoundCancelled activities_model.ActionType = 57
```

And in the `HackforgerActionTypeName` map, add:

```go
ActionGrantRoundCancelled: "grant_round_cancelled",
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go build ./models/hackforger/...`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/action_types.go
git commit -m "feat(grant): add ActionGrantRoundCancelled (57) to action types"
```

---

## Chunk 3: Service Layer — Grants

### Task 6: Implement Grant Service (State Machine + Business Logic)

**Files:**
- Create: `services/hackforger/grants.go`
- Create: `services/hackforger/grants_test.go`

- [ ] **Step 1: Write failing tests**

Create `services/hackforger/grants_test.go`:

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

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestGrantRoundStatusTransition_Valid(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Draft -> Open
	err := hackforger_service.OpenRound(db.DefaultContext, 2, 1) // ownerID=2, roundID=1 (Draft)
	require.NoError(t, err)
	round, _ := hackforger_model.GetGrantRoundByID(db.DefaultContext, 1)
	assert.Equal(t, hackforger_model.GrantRoundStatusOpen, round.Status)
}

func TestGrantRoundStatusTransition_Invalid(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Draft -> Finalized (invalid, must go through Open -> Review first)
	err := hackforger_service.FinalizeRound(db.DefaultContext, 2, 1) // roundID=1 is Draft
	assert.Error(t, err)
}

func TestSubmitProject_RoundNotOpen(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 1 is Draft, not Open
	_, err := hackforger_service.SubmitProject(db.DefaultContext, 4, 1, hackforger_service.SubmitProjectOpts{
		Title:       "Test",
		Description: "Test project",
	})
	assert.True(t, hackforger_model.IsErrGrantRoundNotOpen(err))
}

func TestSubmitProject_Duplicate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User 4 already has a project in round 2 (from fixtures)
	_, err := hackforger_service.SubmitProject(db.DefaultContext, 4, 2, hackforger_service.SubmitProjectOpts{
		Title:       "Duplicate",
		Description: "Should fail",
	})
	assert.True(t, hackforger_model.IsErrGrantProjectAlreadyExists(err))
}

func TestAllocateAward_ExceedsBudget(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 3 has budget 15000 USD / 8000 credits
	// Projects 2 and 3 already have 5000+3000=8000 allocated
	// Trying to add more than remaining 7000 should fail
	err := hackforger_service.AllocateAward(db.DefaultContext, 2, 2, 8000.00, 0) // exceeds by 1000
	assert.True(t, hackforger_model.IsErrExceedsBudget(err))
}

func TestFinalizeRound_UnallocatedProjects(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Need a round in Review with an Approved project that has AwardAmount=0
	// Create this scenario in the test
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Finalize Test", Slug: "finalize-test",
		Status: hackforger_model.GrantRoundStatusReview, Budget: 10000, Currency: "USD", BudgetCredits: 5000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	project := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 4, Title: "Unallocated",
		Status: hackforger_model.GrantProjectStatusApproved,
	}
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, project))

	err := hackforger_service.FinalizeRound(db.DefaultContext, 2, round.ID)
	assert.True(t, hackforger_model.IsErrUnallocatedProjects(err))
}

func TestDistributeProject_CreditsDeposit(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Setup: round in Finalized state with approved+allocated project
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Distribute Test", Slug: "distribute-test",
		Status: hackforger_model.GrantRoundStatusFinalized, Budget: 10000, Currency: "USD", BudgetCredits: 5000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	project := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 4, Title: "To Distribute",
		Status: hackforger_model.GrantProjectStatusApproved,
		AwardAmount: 1000, AwardCredits: 500,
	}
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, project))

	// Get balance before
	acctBefore, _ := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	balanceBefore := acctBefore.Balance

	// Distribute
	err := hackforger_service.DistributeProject(db.DefaultContext, 2, project.ID)
	require.NoError(t, err)

	// Verify project status -> Funded
	updated, _ := hackforger_model.GetGrantProjectByID(db.DefaultContext, project.ID)
	assert.Equal(t, hackforger_model.GrantProjectStatusFunded, updated.Status)

	// Verify credits deposited
	acctAfter, _ := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	assert.Equal(t, balanceBefore+500, acctAfter.Balance)
}

func TestDistributeProject_AutoDistributeRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Setup: Finalized round with ONE approved project
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Auto Distribute", Slug: "auto-distribute-test",
		Status: hackforger_model.GrantRoundStatusFinalized, Budget: 5000, Currency: "USD", BudgetCredits: 2000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	project := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 4, Title: "Only Project",
		Status: hackforger_model.GrantProjectStatusApproved,
		AwardAmount: 1000, AwardCredits: 500,
	}
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, project))

	err := hackforger_service.DistributeProject(db.DefaultContext, 2, project.ID)
	require.NoError(t, err)

	// Round should auto-transition to Distributed
	updatedRound, _ := hackforger_model.GetGrantRoundByID(db.DefaultContext, round.ID)
	assert.Equal(t, hackforger_model.GrantRoundStatusDistributed, updatedRound.Status)
}

func TestCancelRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Open round can be cancelled
	err := hackforger_service.CancelRound(db.DefaultContext, 2, 2) // roundID=2 is Open
	require.NoError(t, err)
	round, _ := hackforger_model.GetGrantRoundByID(db.DefaultContext, 2)
	assert.Equal(t, hackforger_model.GrantRoundStatusCancelled, round.Status)
}

func TestCancelRound_AlreadyDistributed_Rejected(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Distributed Round", Slug: "cancel-distributed-test",
		Status: hackforger_model.GrantRoundStatusDistributed, Budget: 5000, Currency: "USD",
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	err := hackforger_service.CancelRound(db.DefaultContext, 2, round.ID)
	assert.Error(t, err)
}

func TestCancelRound_AlreadyCancelled_Rejected(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Cancelled Round", Slug: "cancel-cancelled-test",
		Status: hackforger_model.GrantRoundStatusCancelled, Budget: 5000, Currency: "USD",
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	err := hackforger_service.CancelRound(db.DefaultContext, 2, round.ID)
	assert.Error(t, err)
}

func TestDeleteGrantRound_NonDraft_Rejected(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 2 is Open, should not be deletable via service
	err := hackforger_service.DeleteGrantRoundService(db.DefaultContext, 2, 2)
	assert.Error(t, err)
}

func TestDistributeRound_BatchConvenience(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Setup: Finalized round with 2 approved+allocated projects
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Batch Test", Slug: "batch-distribute-test",
		Status: hackforger_model.GrantRoundStatusFinalized, Budget: 10000, Currency: "USD", BudgetCredits: 5000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	p1 := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 4, Title: "Batch P1",
		Status: hackforger_model.GrantProjectStatusApproved, AwardCredits: 300,
	}
	p2 := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 5, Title: "Batch P2",
		Status: hackforger_model.GrantProjectStatusApproved, AwardCredits: 200,
	}
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, p1))
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, p2))

	err := hackforger_service.DistributeRound(db.DefaultContext, 2, round.ID)
	require.NoError(t, err)

	// Both projects should be Funded
	up1, _ := hackforger_model.GetGrantProjectByID(db.DefaultContext, p1.ID)
	up2, _ := hackforger_model.GetGrantProjectByID(db.DefaultContext, p2.ID)
	assert.Equal(t, hackforger_model.GrantProjectStatusFunded, up1.Status)
	assert.Equal(t, hackforger_model.GrantProjectStatusFunded, up2.Status)

	// Round should be Distributed
	updatedRound, _ := hackforger_model.GetGrantRoundByID(db.DefaultContext, round.ID)
	assert.Equal(t, hackforger_model.GrantRoundStatusDistributed, updatedRound.Status)
}

func TestCreateGrantRound_Service(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	round, err := hackforger_service.CreateGrantRound(db.DefaultContext, 2, 3, hackforger_service.CreateGrantRoundOpts{
		Name: "New Round", Slug: "new-round-test", Description: "Test",
		Budget: 5000, Currency: "USD", BudgetCredits: 2000,
	})
	require.NoError(t, err)
	assert.Greater(t, round.ID, int64(0))
	assert.Equal(t, hackforger_model.GrantRoundStatusDraft, round.Status)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./services/hackforger/... -run "TestGrantRound|TestSubmitProject|TestAllocateAward|TestFinalizeRound|TestDistributeProject|TestCancelRound" -v`
Expected: FAIL — `grants.go` does not exist yet

- [ ] **Step 3: Implement Grant Service**

Create `services/hackforger/grants.go`:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	org_model "forgejo.org/models/organization"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
)

// SubmitProjectOpts holds options for submitting a project.
type SubmitProjectOpts struct {
	Title       string
	Description string
	RepoID      int64
}

// checkGrantRoundAccess verifies the doer is an owner or admin of the round's org.
func checkGrantRoundAccess(ctx context.Context, doerID int64, round *hackforger_model.GrantRound) error {
	isOwner, err := org_model.IsOrganizationOwner(ctx, round.OrgID, doerID)
	if err != nil {
		return err
	}
	if isOwner {
		return nil
	}
	isAdmin, err := org_model.IsOrganizationAdmin(ctx, round.OrgID, doerID)
	if err != nil {
		return err
	}
	if isAdmin {
		return nil
	}
	return fmt.Errorf("user %d is not an owner or admin of org %d", doerID, round.OrgID)
}

// validTransitions maps current status to allowed next statuses.
var validTransitions = map[hackforger_model.GrantRoundStatus][]hackforger_model.GrantRoundStatus{
	hackforger_model.GrantRoundStatusDraft:     {hackforger_model.GrantRoundStatusOpen, hackforger_model.GrantRoundStatusCancelled},
	hackforger_model.GrantRoundStatusOpen:      {hackforger_model.GrantRoundStatusReview, hackforger_model.GrantRoundStatusCancelled},
	hackforger_model.GrantRoundStatusReview:    {hackforger_model.GrantRoundStatusFinalized, hackforger_model.GrantRoundStatusCancelled},
	hackforger_model.GrantRoundStatusFinalized: {hackforger_model.GrantRoundStatusDistributed, hackforger_model.GrantRoundStatusCancelled},
}

func isValidTransition(from, to hackforger_model.GrantRoundStatus) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func transitionRound(ctx context.Context, doerID, roundID int64, target hackforger_model.GrantRoundStatus) (*hackforger_model.GrantRound, error) {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return nil, err
	}
	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return nil, err
	}
	if !isValidTransition(round.Status, target) {
		return nil, fmt.Errorf("invalid transition from %d to %d for round %d", round.Status, target, roundID)
	}
	round.Status = target
	if err := hackforger_model.UpdateGrantRound(ctx, round); err != nil {
		return nil, err
	}
	return round, nil
}

// CreateGrantRound creates a new grant round after verifying org access.
func CreateGrantRound(ctx context.Context, doerID, orgID int64, opts CreateGrantRoundOpts) (*hackforger_model.GrantRound, error) {
	isOwner, err := org_model.IsOrganizationOwner(ctx, orgID, doerID)
	if err != nil {
		return nil, err
	}
	isAdmin, err := org_model.IsOrganizationAdmin(ctx, orgID, doerID)
	if err != nil {
		return nil, err
	}
	if !isOwner && !isAdmin {
		return nil, fmt.Errorf("user %d is not an owner or admin of org %d", doerID, orgID)
	}

	round := &hackforger_model.GrantRound{
		OrgID:         orgID,
		OwnerID:       doerID,
		Name:          opts.Name,
		Slug:          opts.Slug,
		Description:   opts.Description,
		Status:        hackforger_model.GrantRoundStatusDraft,
		Budget:        opts.Budget,
		Currency:      opts.Currency,
		BudgetCredits: opts.BudgetCredits,
		Deadline:      opts.Deadline,
	}
	if err := hackforger_model.CreateGrantRound(ctx, round); err != nil {
		return nil, err
	}
	publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundCreated, round, AudienceGlobal)
	return round, nil
}

// CreateGrantRoundOpts holds options for creating a grant round.
type CreateGrantRoundOpts struct {
	Name          string
	Slug          string
	Description   string
	Budget        float64
	Currency      string
	BudgetCredits int64
	Deadline      timeutil.TimeStamp
}

// UpdateGrantRoundService updates a grant round (Draft or Open only).
func UpdateGrantRoundService(ctx context.Context, doerID int64, round *hackforger_model.GrantRound) error {
	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}
	if round.Status != hackforger_model.GrantRoundStatusDraft && round.Status != hackforger_model.GrantRoundStatusOpen {
		return fmt.Errorf("round %d can only be edited in Draft or Open status (current: %d)", round.ID, round.Status)
	}
	return hackforger_model.UpdateGrantRound(ctx, round)
}

// DeleteGrantRoundService deletes a grant round (Draft only, with access check).
func DeleteGrantRoundService(ctx context.Context, doerID, roundID int64) error {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return err
	}
	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}
	return hackforger_model.DeleteGrantRound(ctx, roundID)
}

// publishGrantEvent publishes a feed event for a grant action.
func publishGrantEvent(ctx context.Context, actUserID int64, opType activities_model.ActionType, round *hackforger_model.GrantRound, audience AudienceType) {
	content := &hackforger_model.HackforgerActionContent{
		EntityType: "grant_round",
		EntityID:   round.ID,
		EntityName: round.Name,
		EntitySlug: round.Slug,
	}
	if err := PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    actUserID,
		OpType:       opType,
		Content:      content,
		AudienceType: audience,
		OrgID:        round.OrgID,
	}); err != nil {
		log.Error("publishGrantEvent: %v", err)
	}
}

// OpenRound transitions a round from Draft to Open.
func OpenRound(ctx context.Context, doerID, roundID int64) error {
	round, err := transitionRound(ctx, doerID, roundID, hackforger_model.GrantRoundStatusOpen)
	if err != nil {
		return err
	}
	publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundOpened, round, AudienceGlobal)
	return nil
}

// CloseRound transitions a round from Open to Review.
func CloseRound(ctx context.Context, doerID, roundID int64) error {
	round, err := transitionRound(ctx, doerID, roundID, hackforger_model.GrantRoundStatusReview)
	if err != nil {
		return err
	}
	publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundClosed, round, AudienceGlobal)
	return nil
}

// FinalizeRound transitions a round from Review to Finalized.
// All Approved projects must have AwardAmount > 0 or AwardCredits > 0.
func FinalizeRound(ctx context.Context, doerID, roundID int64) error {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return err
	}
	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}
	if !isValidTransition(round.Status, hackforger_model.GrantRoundStatusFinalized) {
		return fmt.Errorf("invalid transition from %d to finalized for round %d", round.Status, roundID)
	}

	// Check all approved projects are allocated
	approvedStatus := hackforger_model.GrantProjectStatusApproved
	projects, _, err := hackforger_model.ListGrantProjectsByRound(ctx, hackforger_model.ListGrantProjectsByRoundOptions{
		RoundID: roundID,
		Status:  &approvedStatus,
	})
	if err != nil {
		return err
	}
	unallocated := int64(0)
	for _, p := range projects {
		if p.AwardAmount <= 0 && p.AwardCredits <= 0 {
			unallocated++
		}
	}
	if unallocated > 0 {
		return hackforger_model.ErrUnallocatedProjects{RoundID: roundID, Count: unallocated}
	}

	round.Status = hackforger_model.GrantRoundStatusFinalized
	if err := hackforger_model.UpdateGrantRound(ctx, round); err != nil {
		return err
	}
	publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundFinalized, round, AudienceGlobal)
	return nil
}

// CancelRound cancels a round (from any non-terminal state).
func CancelRound(ctx context.Context, doerID, roundID int64) error {
	round, err := transitionRound(ctx, doerID, roundID, hackforger_model.GrantRoundStatusCancelled)
	if err != nil {
		return err
	}
	publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundCancelled, round, AudienceGlobal)
	return nil
}

// SubmitProject creates a new project submission in an Open round.
func SubmitProject(ctx context.Context, doerID, roundID int64, opts SubmitProjectOpts) (*hackforger_model.GrantProject, error) {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return nil, err
	}
	if round.Status != hackforger_model.GrantRoundStatusOpen {
		return nil, hackforger_model.ErrGrantRoundNotOpen{RoundID: roundID, Status: round.Status}
	}

	// Check duplicate
	_, err = hackforger_model.GetGrantProjectByUserAndRound(ctx, doerID, roundID)
	if err == nil {
		return nil, hackforger_model.ErrGrantProjectAlreadyExists{UserID: doerID, RoundID: roundID}
	}
	if !hackforger_model.IsErrGrantProjectNotExist(err) {
		return nil, err
	}

	project := &hackforger_model.GrantProject{
		RoundID:     roundID,
		UserID:      doerID,
		RepoID:      opts.RepoID,
		Title:       opts.Title,
		Description: opts.Description,
		Status:      hackforger_model.GrantProjectStatusPending,
	}
	if err := hackforger_model.CreateGrantProject(ctx, project); err != nil {
		return nil, err
	}

	publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantProjectSubmitted, round, AudienceFollowers)
	return project, nil
}

// ApproveProject approves a pending project.
func ApproveProject(ctx context.Context, doerID, projectID int64) error {
	project, err := hackforger_model.GetGrantProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	round, err := hackforger_model.GetGrantRoundByID(ctx, project.RoundID)
	if err != nil {
		return err
	}
	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}
	if project.Status != hackforger_model.GrantProjectStatusPending {
		return fmt.Errorf("project %d is not pending (status: %d)", projectID, project.Status)
	}
	project.Status = hackforger_model.GrantProjectStatusApproved
	return hackforger_model.UpdateGrantProject(ctx, project)
}

// RejectProject rejects a pending project.
func RejectProject(ctx context.Context, doerID, projectID int64) error {
	project, err := hackforger_model.GetGrantProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	round, err := hackforger_model.GetGrantRoundByID(ctx, project.RoundID)
	if err != nil {
		return err
	}
	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}
	if project.Status != hackforger_model.GrantProjectStatusPending {
		return fmt.Errorf("project %d is not pending (status: %d)", projectID, project.Status)
	}
	project.Status = hackforger_model.GrantProjectStatusRejected
	return hackforger_model.UpdateGrantProject(ctx, project)
}

// AllocateAward sets the award amounts for an approved project with budget validation.
func AllocateAward(ctx context.Context, doerID, projectID int64, amount float64, credits int64) error {
	project, err := hackforger_model.GetGrantProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	round, err := hackforger_model.GetGrantRoundByID(ctx, project.RoundID)
	if err != nil {
		return err
	}
	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}
	if round.Status != hackforger_model.GrantRoundStatusReview {
		return fmt.Errorf("round %d is not in review status", round.ID)
	}
	if project.Status != hackforger_model.GrantProjectStatusApproved {
		return fmt.Errorf("project %d is not approved", projectID)
	}

	// Budget check: get current usage, subtract this project's existing allocation, add new
	usedAmount, usedCredits, err := hackforger_model.GetGrantRoundBudgetUsage(ctx, round.ID)
	if err != nil {
		return err
	}
	// Subtract current project allocation (in case of re-allocation)
	usedAmount -= project.AwardAmount
	usedCredits -= project.AwardCredits

	if usedAmount+amount > round.Budget {
		return hackforger_model.ErrExceedsBudget{
			RoundID: round.ID, BudgetField: "amount",
			Budget: round.Budget, Used: usedAmount, Requested: amount,
		}
	}
	if usedCredits+credits > round.BudgetCredits {
		return hackforger_model.ErrExceedsBudget{
			RoundID: round.ID, BudgetField: "credits",
			Budget: float64(round.BudgetCredits), Used: float64(usedCredits), Requested: float64(credits),
		}
	}

	project.AwardAmount = amount
	project.AwardCredits = credits
	return hackforger_model.UpdateGrantProject(ctx, project)
}

// DistributeProject marks a project as Funded and deposits credits.
func DistributeProject(ctx context.Context, doerID, projectID int64) error {
	project, err := hackforger_model.GetGrantProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	round, err := hackforger_model.GetGrantRoundByID(ctx, project.RoundID)
	if err != nil {
		return err
	}
	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}
	if round.Status != hackforger_model.GrantRoundStatusFinalized {
		return fmt.Errorf("round %d is not finalized", round.ID)
	}
	if project.Status != hackforger_model.GrantProjectStatusApproved {
		return fmt.Errorf("project %d is not approved (status: %d)", projectID, project.Status)
	}

	err = db.WithTx(ctx, func(ctx context.Context) error {
		// Deposit credits
		if project.AwardCredits > 0 {
			ref := fmt.Sprintf("grant_round:%d/project:%d", round.ID, project.ID)
			if err := Deposit(ctx, project.UserID, project.AwardCredits, ref, fmt.Sprintf("Grant award: %s", round.Name)); err != nil {
				return err
			}
		}
		// Mark project as Funded
		project.Status = hackforger_model.GrantProjectStatusFunded
		return hackforger_model.UpdateGrantProject(ctx, project)
	})
	if err != nil {
		return err
	}

	publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantAwarded, round, AudienceGlobal)

	// Auto-check: if all Approved projects are now Funded, transition round
	approvedStatus := hackforger_model.GrantProjectStatusApproved
	remaining, err := hackforger_model.CountGrantProjectsByRound(ctx, round.ID, &approvedStatus)
	if err != nil {
		log.Error("DistributeProject auto-check: %v", err)
		return nil // don't fail the distribute
	}
	if remaining == 0 {
		round.Status = hackforger_model.GrantRoundStatusDistributed
		if err := hackforger_model.UpdateGrantRound(ctx, round); err != nil {
			log.Error("DistributeProject auto-distribute round: %v", err)
		}
	}

	return nil
}

// DistributeRound distributes all approved+allocated projects in a finalized round.
func DistributeRound(ctx context.Context, doerID, roundID int64) error {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return err
	}
	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}
	if round.Status != hackforger_model.GrantRoundStatusFinalized {
		return fmt.Errorf("round %d is not finalized", roundID)
	}

	approvedStatus := hackforger_model.GrantProjectStatusApproved
	projects, _, err := hackforger_model.ListGrantProjectsByRound(ctx, hackforger_model.ListGrantProjectsByRoundOptions{
		RoundID: roundID,
		Status:  &approvedStatus,
	})
	if err != nil {
		return err
	}

	for _, p := range projects {
		if err := DistributeProject(ctx, doerID, p.ID); err != nil {
			return fmt.Errorf("failed to distribute project %d: %w", p.ID, err)
		}
	}
	return nil
}

// ExportRoundCSV generates a CSV export of the round's projects.
func ExportRoundCSV(ctx context.Context, roundID int64) ([]byte, error) {
	projects, _, err := hackforger_model.ListGrantProjectsByRound(ctx, hackforger_model.ListGrantProjectsByRoundOptions{
		RoundID: roundID,
	})
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "title", "user_id", "status", "award_amount", "award_credits"})
	for _, p := range projects {
		_ = w.Write([]string{
			fmt.Sprintf("%d", p.ID),
			p.Title,
			fmt.Sprintf("%d", p.UserID),
			fmt.Sprintf("%d", p.Status),
			fmt.Sprintf("%.2f", p.AwardAmount),
			fmt.Sprintf("%d", p.AwardCredits),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./services/hackforger/... -run "TestGrantRound|TestSubmitProject|TestAllocateAward|TestFinalizeRound|TestDistributeProject|TestCancelRound" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/hackforger/grants.go services/hackforger/grants_test.go
git commit -m "feat(grant): add Grant service — state machine, budget validation, distribute + tests"
```

---

## Chunk 4: Service Layer — Credits Admin

### Task 7: Add Credits Admin Functions

**Files:**
- Modify: `services/hackforger/credits.go`
- Create: `services/hackforger/credits_test.go`

- [ ] **Step 1: Write failing tests**

Create `services/hackforger/credits_test.go`:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminDeposit(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1}) // admin user from Forgejo fixtures

	err := hackforger_service.AdminDeposit(db.DefaultContext, admin, 4, 500, "manual", "test deposit")
	require.NoError(t, err)

	acct, err := hackforger_model.GetCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), acct.Balance) // 1000 + 500
}

func TestAdminDeposit_NotAdmin(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	nonAdmin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	err := hackforger_service.AdminDeposit(db.DefaultContext, nonAdmin, 4, 500, "manual", "should fail")
	assert.Error(t, err)
}

func TestAdminDeduct(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	err := hackforger_service.AdminDeduct(db.DefaultContext, admin, 4, 200, "manual", "test deduct")
	require.NoError(t, err)

	acct, err := hackforger_model.GetCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(800), acct.Balance) // 1000 - 200
}

func TestAdminDeduct_InsufficientBalance(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	err := hackforger_service.AdminDeduct(db.DefaultContext, admin, 4, 2000, "manual", "too much")
	assert.True(t, hackforger_model.IsErrInsufficientCredits(err))
}

func TestFulfillOrder(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	err := hackforger_service.FulfillOrder(db.DefaultContext, admin, 1, "Sent via email")
	require.NoError(t, err)

	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order.Status)
	assert.Equal(t, "Sent via email", order.FulfillNote)
}

func TestCancelOrder_RefundsBalance(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// Order 1 costs 500, user 4 has balance 1000
	err := hackforger_service.CancelOrder(db.DefaultContext, admin, 1)
	require.NoError(t, err)

	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusCancelled, order.Status)

	// Balance should be refunded: 1000 + 500 = 1500
	acct, err := hackforger_model.GetCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), acct.Balance)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./services/hackforger/... -run "TestAdmin|TestFulfillOrder|TestCancelOrder" -v`
Expected: FAIL

- [ ] **Step 3: Implement admin functions**

Append to `services/hackforger/credits.go`:

```go
import (
	user_model "forgejo.org/models/user"
)

// AdminDeposit manually adds credits to a user (admin only).
func AdminDeposit(ctx context.Context, admin *user_model.User, userID int64, amount int64, reference, note string) error {
	if !admin.IsAdmin {
		return fmt.Errorf("user %d is not an admin", admin.ID)
	}
	return Deposit(ctx, userID, amount, reference, note)
}

// AdminDeduct manually removes credits from a user (admin only).
func AdminDeduct(ctx context.Context, admin *user_model.User, userID int64, amount int64, reference, note string) error {
	if !admin.IsAdmin {
		return fmt.Errorf("user %d is not an admin", admin.ID)
	}
	return db.WithTx(ctx, func(ctx context.Context) error {
		acct, err := GetOrCreateCreditAccount(ctx, userID)
		if err != nil {
			return err
		}
		if acct.Balance < amount {
			return hackforger_model.ErrInsufficientCredits{
				UserID: userID, Balance: acct.Balance, Requested: amount,
			}
		}
		acct.Balance -= amount
		if _, err := db.GetEngine(ctx).ID(acct.ID).Cols("balance").Update(acct); err != nil {
			return err
		}
		tx := &hackforger_model.CreditTransaction{
			UserID:    userID,
			Type:      hackforger_model.TransactionTypeWithdraw,
			Amount:    -amount,
			Balance:   acct.Balance,
			Reference: reference,
			Note:      note,
		}
		_, err = db.GetEngine(ctx).Insert(tx)
		return err
	})
}

// FulfillOrder marks a redeem order as fulfilled (admin only).
func FulfillOrder(ctx context.Context, admin *user_model.User, orderID int64, note string) error {
	if !admin.IsAdmin {
		return fmt.Errorf("user %d is not an admin", admin.ID)
	}
	order, err := hackforger_model.GetRedeemOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != hackforger_model.OrderStatusPending {
		return fmt.Errorf("order %d is not pending (status: %s)", orderID, order.Status)
	}
	order.Status = hackforger_model.OrderStatusFulfilled
	order.FulfillNote = note
	return hackforger_model.UpdateRedeemOrder(ctx, order)
}

// CancelOrder cancels a redeem order and refunds the credits (admin only).
func CancelOrder(ctx context.Context, admin *user_model.User, orderID int64) error {
	if !admin.IsAdmin {
		return fmt.Errorf("user %d is not an admin", admin.ID)
	}
	order, err := hackforger_model.GetRedeemOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != hackforger_model.OrderStatusPending {
		return fmt.Errorf("order %d is not pending (status: %s)", orderID, order.Status)
	}
	return db.WithTx(ctx, func(ctx context.Context) error {
		order.Status = hackforger_model.OrderStatusCancelled
		if err := hackforger_model.UpdateRedeemOrder(ctx, order); err != nil {
			return err
		}
		// Refund balance
		acct, err := GetOrCreateCreditAccount(ctx, order.UserID)
		if err != nil {
			return err
		}
		acct.Balance += order.Cost
		if _, err := db.GetEngine(ctx).ID(acct.ID).Cols("balance").Update(acct); err != nil {
			return err
		}
		tx := &hackforger_model.CreditTransaction{
			UserID:    order.UserID,
			Type:      hackforger_model.TransactionTypeRefund,
			Amount:    order.Cost,
			Balance:   acct.Balance,
			Reference: fmt.Sprintf("order:%d", order.ID),
			Note:      "Order cancelled, credits refunded",
		}
		_, err = db.GetEngine(ctx).Insert(tx)
		return err
	})
}

// CreateRedeemOptionAsAdmin creates a new redeem option (admin only).
func CreateRedeemOptionAsAdmin(ctx context.Context, admin *user_model.User, opt *hackforger_model.RedeemOption) error {
	if !admin.IsAdmin {
		return fmt.Errorf("user %d is not an admin", admin.ID)
	}
	return hackforger_model.CreateRedeemOption(ctx, opt)
}

// UpdateRedeemOptionAsAdmin updates a redeem option (admin only).
func UpdateRedeemOptionAsAdmin(ctx context.Context, admin *user_model.User, opt *hackforger_model.RedeemOption) error {
	if !admin.IsAdmin {
		return fmt.Errorf("user %d is not an admin", admin.ID)
	}
	return hackforger_model.UpdateRedeemOption(ctx, opt)
}
```

Note: Merge new imports with existing ones in `credits.go`. The file already imports `"forgejo.org/models/db"` and `hackforger_model`. Add `user_model` and `"fmt"`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./services/hackforger/... -run "TestAdmin|TestFulfillOrder|TestCancelOrder" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/hackforger/credits.go services/hackforger/credits_test.go
git commit -m "feat(credits): add admin deposit/deduct, fulfill/cancel order + tests"
```

---

## Chunk 5: API Layer

### Task 8: Grant API Handlers

**Files:**
- Modify: `routers/api/v1/hackforger/grants.go` (replace stub)
- Modify: `routers/api/v1/api.go:1739-1746`

- [ ] **Step 1: Implement Grant API handlers**

Replace `routers/api/v1/hackforger/grants.go` entirely. The file should contain all Grant Round and Grant Project API handlers with Swagger annotations. Each handler:
- Uses `context.APIContext`
- Parses path params with `ctx.PathParamInt64(":id")`
- Uses `ctx.JSON(status, data)` for responses
- Uses `ctx.Error(status, msg, err)` for errors
- Includes swagger operation comments

Key handlers to implement:
- `CreateGrantRound` — parse JSON body, call `hackforger_service.CreateGrantRound` (note: service-level CreateGrantRound needs to be added as a thin wrapper that creates the round, publishes feed event, and returns it)
- `ListGrantRounds` — use `utils.GetListOptions(ctx)`, call model `ListGrantRounds`, set `X-Total-Count`
- `GetGrantRound` — by ID
- `UpdateGrantRound` — parse body, call service
- `DeleteGrantRound` — call model
- `OpenRound`, `CloseRound`, `FinalizeRound`, `DistributeRound`, `CancelRound` — call service functions
- `ExportCSV` — set Content-Type `text/csv`, return bytes
- `SubmitProject`, `ListProjects`, `GetProject`, `ApproveOrReject`, `AllocateAward`, `DistributeProject`

Also add a `GrantRoutes` function that registers all routes in a go-chi group:

```go
func GrantRoutes(m *macaron.Macaron) {
	// This will actually use go-chi router pattern, following existing HackForger API registration
}
```

- [ ] **Step 2: Update API route registration**

**Note:** The existing P0 code registers grants at `/grants/rounds` (line 1743). This plan changes it to `/grant-rounds` (kebab-case, matching the spec). This is an intentional path change.

In `routers/api/v1/api.go:1739-1746`, replace the entire `/hackforger` group:

```go
// HackForger API routes
m.Group("/hackforger", func() {
	m.Get("/hackathons", hackforger_api.ListHackathons)
	m.Get("/bounties", hackforger_api.ListBounties)
	m.Get("/feed", hackforger_api.GetFeed)

	// Grant Round routes
	m.Group("/grant-rounds", func() {
		m.Combo("").
			Get(hackforger_api.ListGrantRounds).
			Post(reqToken(), hackforger_api.CreateGrantRound)
		m.Group("/{id}", func() {
			m.Get("", hackforger_api.GetGrantRound)
			m.Put("", reqToken(), hackforger_api.UpdateGrantRound)
			m.Delete("", reqToken(), hackforger_api.DeleteGrantRound)
			m.Post("/open", reqToken(), hackforger_api.OpenRound)
			m.Post("/close", reqToken(), hackforger_api.CloseRound)
			m.Post("/finalize", reqToken(), hackforger_api.FinalizeRound)
			m.Post("/distribute", reqToken(), hackforger_api.DistributeRound)
			m.Post("/cancel", reqToken(), hackforger_api.CancelRound)
			m.Get("/export", hackforger_api.ExportRoundCSV)
			m.Group("/projects", func() {
				m.Combo("").
					Get(hackforger_api.ListGrantProjects).
					Post(reqToken(), hackforger_api.SubmitProject)
				m.Group("/{pid}", func() {
					m.Get("", hackforger_api.GetGrantProject)
					m.Put("", reqToken(), hackforger_api.ApproveOrRejectProject)
					m.Put("/award", reqToken(), hackforger_api.AllocateAward)
					m.Post("/distribute", reqToken(), hackforger_api.DistributeProject)
				})
			})
		})
	})

	// Credits routes (Task 9)
	m.Group("/credits", func() {
		m.Get("/balance", reqToken(), hackforger_api.GetBalance)
		m.Get("/transactions", reqToken(), hackforger_api.ListTransactions)
		m.Group("/redeem", func() {
			m.Get("/options", hackforger_api.ListRedeemOptions)
			m.Post("/options", reqToken(), reqSiteAdmin(), hackforger_api.CreateRedeemOption)
			m.Put("/options/{id}", reqToken(), reqSiteAdmin(), hackforger_api.UpdateRedeemOption)
			m.Post("", reqToken(), hackforger_api.Redeem)
			m.Get("/orders", reqToken(), hackforger_api.ListRedeemOrders)
			m.Post("/orders/{oid}/fulfill", reqToken(), reqSiteAdmin(), hackforger_api.FulfillOrder)
			m.Post("/orders/{oid}/cancel", reqToken(), reqSiteAdmin(), hackforger_api.CancelOrder)
		})
		m.Group("/admin", func() {
			m.Post("/deposit", hackforger_api.AdminDeposit)
			m.Post("/deduct", hackforger_api.AdminDeduct)
		}, reqToken(), reqSiteAdmin())
	})
})
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go build ./routers/...`
Expected: Success

- [ ] **Step 4: Commit**

```bash
git add routers/api/v1/hackforger/grants.go routers/api/v1/api.go
git commit -m "feat(grant): add Grant API handlers + route registration"
```

### Task 9: Credits API Handlers

**Files:**
- Modify: `routers/api/v1/hackforger/credits.go` (replace stub)

- [ ] **Step 1: Implement Credits API handlers**

Replace `routers/api/v1/hackforger/credits.go`. Handlers:
- `GetBalance` — call `GetOrCreateCreditAccount`, return `{"balance": N}`
- `ListTransactions` — paginated, `X-Total-Count`
- `ListRedeemOptions` — call `ListRedeemOptions`
- `Redeem` — parse `{"option_id": N}`, call `Redeem`
- `ListRedeemOrders` — paginated
- `AdminDeposit` — parse body, call `AdminDeposit`
- `AdminDeduct` — parse body, call `AdminDeduct`
- `CreateRedeemOption` — parse body
- `UpdateRedeemOption` — parse body
- `FulfillOrder` — parse body with note
- `CancelOrder` — call `CancelOrder`

All with Swagger annotations.

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go build ./routers/...`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git add routers/api/v1/hackforger/credits.go
git commit -m "feat(credits): add Credits API handlers"
```

---

## Chunk 6: Web Routes + Templates (Grants)

### Task 10: Grant Web Handlers + Route Registration

**Files:**
- Create: `routers/web/hackforger/grants.go`
- Modify: `routers/web/hackforger/hackathon.go` (remove ExploreGrants)
- Modify: `routers/web/web.go` (add grant route groups)

- [ ] **Step 1: Create grants.go web handler**

Create `routers/web/hackforger/grants.go` with handlers:
- `ExploreGrants` — moved from hackathon.go, now queries actual data
- `NewGrantRound` (GET + POST)
- `GrantRoundDetail`
- `GrantRoundProjects`
- `SubmitProject` (GET + POST)
- `ManageRound`
- `ManageProject`
- `ExportCSV`

Each handler sets `ctx.Data` variables and renders the appropriate template.

- [ ] **Step 2: Remove ExploreGrants from hackathon.go**

In `routers/web/hackforger/hackathon.go`, remove the `ExploreGrants` function (lines 25-29). Keep `ExploreHackathons` and `ExploreBounties`.

- [ ] **Step 3: Register web routes**

In `routers/web/web.go`, after the `/explore` group (around line 507), add:

```go
m.Group("/grants", func() {
	m.Combo("/new").Get(hackforger_web.NewGrantRound).Post(web.Bind(forms.CreateGrantRoundForm{}), hackforger_web.NewGrantRoundPost)
	m.Group("/{slug}", func() {
		m.Get("", hackforger_web.GrantRoundDetail)
		m.Get("/projects", hackforger_web.GrantRoundProjects)
		m.Combo("/submit").Get(hackforger_web.SubmitGrantProject).Post(web.Bind(forms.SubmitGrantProjectForm{}), hackforger_web.SubmitGrantProjectPost)
		m.Get("/export", hackforger_web.ExportGrantRoundCSV)
		m.Group("/manage", func() {
			m.Get("", hackforger_web.ManageGrantRound)
			m.Post("/open", hackforger_web.ManageGrantRoundOpen)
			m.Post("/close", hackforger_web.ManageGrantRoundClose)
			m.Post("/finalize", hackforger_web.ManageGrantRoundFinalize)
			m.Post("/distribute", hackforger_web.ManageGrantRoundDistribute)
			m.Post("/cancel", hackforger_web.ManageGrantRoundCancel)
			m.Group("/projects/{pid}", func() {
				m.Get("", hackforger_web.ManageGrantProject)
				m.Post("/approve", hackforger_web.ManageGrantProjectApprove)
				m.Post("/reject", hackforger_web.ManageGrantProjectReject)
				m.Post("/award", hackforger_web.ManageGrantProjectAward)
				m.Post("/distribute", hackforger_web.ManageGrantProjectDistribute)
			})
		})
	})
}, reqSignIn)
```

Note: Form binding structs need to be created. If Forgejo uses `services/forms/` for form definitions, create the form structs there. Otherwise define them inline or in a hackforger-specific forms file.

- [ ] **Step 4: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go build ./routers/...`
Expected: Success

- [ ] **Step 5: Commit**

```bash
git add routers/web/hackforger/grants.go routers/web/hackforger/hackathon.go routers/web/web.go
git commit -m "feat(grant): add web handlers + route registration"
```

### Task 11: Grant Templates

**Files:**
- Create: `templates/hackforger/grants/explore.tmpl`
- Create: `templates/hackforger/grants/new.tmpl`
- Create: `templates/hackforger/grants/detail.tmpl`
- Create: `templates/hackforger/grants/projects.tmpl`
- Create: `templates/hackforger/grants/submit.tmpl`
- Create: `templates/hackforger/grants/manage.tmpl`
- Create: `templates/hackforger/grants/manage_project.tmpl`
- Modify: `templates/hackforger/explore.tmpl`

- [ ] **Step 1: Create grant templates**

Each template follows the pattern:
```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content">
  <!-- content -->
</div>
{{template "base/footer" .}}
```

Key templates:

**explore.tmpl** — Grant list with status filter tabs, pagination. Uses Fomantic UI card layout.

**detail.tmpl** — Status progress bar (6 steps), round info (name, desc, budget, deadline, org), "Submit Project" button when Open, project count stats.

**manage.tmpl** — State action buttons (contextual based on current status), project table with status badges, budget usage progress bar.

**manage_project.tmpl** — Project details, approve/reject buttons, award amount/credits input fields.

- [ ] **Step 2: Update explore.tmpl to include grants tab**

Modify `templates/hackforger/explore.tmpl` to check `PageIsExploreGrants` and render the grants list content instead of "Coming soon".

- [ ] **Step 3: Verify templates render**

Start server and navigate to `/explore/grants` to verify the page renders.

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && TAGS="bindata sqlite sqlite_unlock_notify" make backend && ./gitea web`
Navigate to: `http://localhost:3000/explore/grants`

- [ ] **Step 4: Commit**

```bash
git add templates/hackforger/grants/ templates/hackforger/explore.tmpl
git commit -m "feat(grant): add grant templates — explore, detail, manage, submit"
```

---

## Chunk 7: Web Routes + Templates (Credits)

### Task 12: Credits Web Handlers

**Files:**
- Create: `routers/web/hackforger/credits.go`
- Modify: `routers/web/web.go` (add credits + admin routes)

- [ ] **Step 1: Create credits.go web handler**

Create `routers/web/hackforger/credits.go` with handlers:
- `CreditsOverview` — balance, transactions, redeem options
- `RedeemConfirm` (GET + POST)
- `OrderList`
- `AdminCredits` — user account list, deposit/deduct forms
- `AdminRedeemOptions` — CRUD for options
- `AdminOrders` — order list + fulfill/cancel

- [ ] **Step 2: Register web routes**

In `routers/web/web.go`, add credits routes:

```go
m.Group("/credits", func() {
	m.Get("", hackforger_web.CreditsOverview)
	m.Combo("/redeem/{id}").Get(hackforger_web.RedeemConfirm).Post(hackforger_web.RedeemConfirmPost)
	m.Get("/orders", hackforger_web.CreditOrders)
}, reqSignIn)
```

And admin routes in the existing `/-/admin` group:

```go
m.Group("/credits", func() {
	m.Get("", hackforger_web.AdminCredits)
	m.Post("/deposit", hackforger_web.AdminCreditsDeposit)
	m.Post("/deduct", hackforger_web.AdminCreditsDeduct)
	m.Get("/options", hackforger_web.AdminRedeemOptions)
	m.Post("/options", hackforger_web.AdminRedeemOptionsCreate)
	m.Post("/options/{id}", hackforger_web.AdminRedeemOptionsUpdate)
	m.Get("/orders", hackforger_web.AdminCreditOrders)
	m.Post("/orders/{oid}/fulfill", hackforger_web.AdminCreditOrdersFulfill)
	m.Post("/orders/{oid}/cancel", hackforger_web.AdminCreditOrdersCancel)
})
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go build ./routers/...`

- [ ] **Step 4: Commit**

```bash
git add routers/web/hackforger/credits.go routers/web/web.go
git commit -m "feat(credits): add web handlers + route registration"
```

### Task 13: Credits Templates

**Files:**
- Create: `templates/hackforger/credits/overview.tmpl`
- Create: `templates/hackforger/credits/redeem.tmpl`
- Create: `templates/hackforger/credits/orders.tmpl`
- Create: `templates/hackforger/credits/admin/credits.tmpl`
- Create: `templates/hackforger/credits/admin/options.tmpl`
- Create: `templates/hackforger/credits/admin/orders.tmpl`

- [ ] **Step 1: Create credits templates**

**overview.tmpl** — Balance card at top, transaction list table (paginated), redeem options grid with "Redeem" buttons.

**redeem.tmpl** — Confirmation page showing option name, cost, current balance, confirm button.

**orders.tmpl** — Order list with status badges (pending/fulfilled/cancelled).

**admin/credits.tmpl** — User search + account list, deposit/deduct form.

**admin/options.tmpl** — Table of redeem options, create/edit forms.

**admin/orders.tmpl** — All orders table with fulfill/cancel action buttons.

- [ ] **Step 2: Verify templates render**

Start server and navigate to `/credits` (as logged-in user) and `/-/admin/credits` (as admin).

- [ ] **Step 3: Commit**

```bash
git add templates/hackforger/credits/
git commit -m "feat(credits): add credits templates — overview, redeem, orders, admin"
```

---

## Chunk 8: i18n + Final Integration

### Task 14: Add i18n Keys

**Files:**
- Modify: `options/locale/locale_en-US.ini`

- [ ] **Step 1: Append grant + credits i18n keys**

Append to the `[hackforger]` section in `options/locale/locale_en-US.ini`:

```ini
; Grant Round
grant.round.new = New Grant Round
grant.round.edit = Edit Grant Round
grant.round.name = Name
grant.round.slug = Slug
grant.round.description = Description
grant.round.budget = Budget
grant.round.budget_credits = Credits Budget
grant.round.currency = Currency
grant.round.deadline = Deadline
grant.round.organization = Organization
grant.round.status = Status
grant.round.status.draft = Draft
grant.round.status.open = Open
grant.round.status.review = Reviewing
grant.round.status.finalized = Finalized
grant.round.status.distributed = Distributed
grant.round.status.cancelled = Cancelled
grant.round.action.open = Open Round
grant.round.action.close = Close Applications
grant.round.action.finalize = Finalize Allocations
grant.round.action.distribute = Distribute All
grant.round.action.cancel = Cancel Round
grant.round.budget_usage = Budget Usage
grant.round.no_rounds = No grant rounds found.
grant.round.create_success = Grant round created successfully.
grant.round.export = Export CSV

; Grant Project
grant.project.submit = Submit Project
grant.project.title = Project Title
grant.project.description = Description
grant.project.repo = Repository
grant.project.status.pending = Pending
grant.project.status.approved = Approved
grant.project.status.rejected = Rejected
grant.project.status.funded = Funded
grant.project.approve = Approve
grant.project.reject = Reject
grant.project.award_amount = Award Amount
grant.project.award_credits = Award Credits
grant.project.allocate = Allocate Award
grant.project.distribute = Distribute
grant.project.no_projects = No projects submitted yet.
grant.project.submit_success = Project submitted successfully.
grant.project.already_submitted = You have already submitted a project to this round.

; Credits (additional keys)
credits.overview = Credits Overview
credits.your_balance = Your Balance
credits.transaction_history = Transaction History
credits.transaction.type = Type
credits.transaction.amount = Amount
credits.transaction.balance_after = Balance After
credits.transaction.reference = Reference
credits.transaction.date = Date
credits.redeem.confirm = Confirm Redemption
credits.redeem.confirm_message = Are you sure you want to redeem "%s" for %d credits?
credits.redeem.success = Successfully redeemed!
credits.redeem.insufficient = Insufficient credits.
credits.orders.title = My Orders
credits.orders.status.pending = Pending
credits.orders.status.fulfilled = Fulfilled
credits.orders.status.cancelled = Cancelled
credits.admin.title = Credits Administration
credits.admin.deposit = Deposit Credits
credits.admin.deduct = Deduct Credits
credits.admin.user_id = User ID
credits.admin.amount = Amount
credits.admin.reference = Reference
credits.admin.note = Note
credits.admin.options = Manage Redeem Options
credits.admin.orders = Manage Orders
credits.admin.fulfill = Fulfill
credits.admin.cancel = Cancel
credits.admin.fulfill_note = Fulfillment Note
```

- [ ] **Step 2: Verify no syntax errors**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add options/locale/locale_en-US.ini
git commit -m "i18n: add Grant + Credits locale keys"
```

### Task 15: Create E2E Manual Test Prompt

**Files:**
- Create: `docs/tests/e2e/phase1-grant-e2e.md`

- [ ] **Step 1: Write E2E test prompt and report template**

Create `docs/tests/e2e/phase1-grant-e2e.md` with:
- Prerequisites (server running, admin login)
- Step-by-step manual test flow:
  1. Login as admin, create org
  2. Create grant round (verify Draft status)
  3. Open round (verify status change)
  4. Login as hacker, submit project
  5. Try duplicate submission (verify error)
  6. Login as admin, close round
  7. Approve project, allocate award
  8. Try exceeding budget (verify error)
  9. Finalize round
  10. Distribute project (verify credits deposited)
  11. Verify round auto-transitions to Distributed
  12. Test credits: check balance, redeem, verify order
  13. Admin: deposit/deduct, fulfill order
  14. Verify feed events at each step (global feed)
- Report template with pass/fail checkboxes

- [ ] **Step 2: Commit**

```bash
git add docs/tests/e2e/phase1-grant-e2e.md
git commit -m "docs: add Phase 1 Grant + Credits E2E test prompt"
```

### Task 16: API Integration Tests

**Files:**
- Create: `tests/integration/hackforger_grant_test.go`
- Create: `tests/integration/hackforger_credits_test.go`

- [ ] **Step 1: Create grant integration tests**

Create `tests/integration/hackforger_grant_test.go` using the existing integration test framework (`tests/integration/integration_test.go` for setup patterns). Tests should use `MakeRequest` and `DecodeJSON` helpers.

Tests to implement:
- `TestAPIGrantRoundCRUD` — create, get, list, update, delete via API
- `TestAPIGrantRoundStatusFlow` — open, close, finalize, distribute via API
- `TestAPIGrantProjectSubmitAndApprove` — submit project, approve via API
- `TestAPIAllocateAndFinalize` — allocate award, finalize, verify budget check
- `TestAPIDistributeProject_CreditsReceived` — distribute, verify credits balance via API
- `TestAPIDistributeRound_BatchAll` — batch distribute all projects
- `TestAPICancelRound` — cancel from various states
- `TestAPIExportCSV` — verify CSV content type and content

- [ ] **Step 2: Create credits integration tests**

Create `tests/integration/hackforger_credits_test.go`:
- `TestAPICreditsBalance` — get balance as logged-in user
- `TestAPIRedeem` — redeem option, verify balance deducted and order created
- `TestAPIAdminDeposit` — admin deposits credits, verify balance
- `TestAPIAdminFulfillOrder` — admin fulfills order

- [ ] **Step 3: Run integration tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./tests/integration/... -run "TestAPIGrant|TestAPICredits" -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add tests/integration/hackforger_grant_test.go tests/integration/hackforger_credits_test.go
git commit -m "test: add Grant + Credits API integration tests"
```

### Task 17: Run All Tests

- [ ] **Step 1: Run model tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./models/hackforger/... -v`
Expected: All tests PASS

- [ ] **Step 2: Run service tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && go test ./services/hackforger/... -v`
Expected: All tests PASS

- [ ] **Step 3: Run full backend build**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/feat+phase1-grant && TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Success

- [ ] **Step 4: Commit any fixes**

If any tests or build issues arise, fix and commit.
