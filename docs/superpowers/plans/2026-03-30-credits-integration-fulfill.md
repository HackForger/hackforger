# Credits Integration + Fulfill Enhancement — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the Credits system by wiring Hackathon prize distribution, enhancing the fulfillment workflow (delivery details, batch fulfill, auto-fulfill key pool, notifications), and finishing zh-CN i18n.

**Architecture:** Layered model→service→router changes. Migration adds columns + new table. Hackathon credits use per-track ranking (independent of global rank). Auto-fulfill uses a key pool table with two-step SELECT+UPDATE for SQLite compatibility. Order notifications use a new `AudienceDirectUser` audience type (isolated commit).

**Tech Stack:** Go, XORM, go-chi, Go HTML templates, Fomantic UI, i18n via locale INI files

**Spec:** `docs/superpowers/specs/2026-03-30-credits-integration-fulfill-design.md`

---

## ⚠ Errata (Post-Review Corrections)

The plan was reviewed by 3 subagents after initial writing. Below are corrections that MUST be applied during implementation. Implementers: read this section first and apply corrections to the relevant task code.

### E1. Chunk 1 already implemented — fixtures differ from plan

Chunk 1 (Tasks 1-7) was implemented before this review. The actual committed fixtures differ from the plan's code blocks:

- **hackathon.yml**: status=3 (Judging), not 4 as plan says (plan had wrong enum: 3=Judging, 4=Finished)
- **hackathon_track.yml**: Tiered track uses ratios `60/30/10`, not `50/30/20`
- **hackathon_submission.yml**: Users are reused across tracks: track 1 = {user 2, user 4}, track 2 = {user 2, user 4, user 5}, track 3 = {user 2, user 4}
- **redeem_option.yml**: Option 2 has cost=2000, stock=-1 (not cost=100, stock=2)
- **redeem_order.yml**: Order 2 has user_id=4/option_id=2, Order 3 has user_id=5/option_id=1
- **ClaimKey**: ForUpdate() was missing; fixed in commit 906306a162

**Action for Task 8**: Tests must use actual fixture user IDs and ratios. See corrected tests below in the errata for Task 8.

### E2. Task 8: Corrected test code matching actual fixtures

Replace the plan's Task 8 test code with:

```go
func TestDistributeHackathonCredits_WinnerTakesAll(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// Track 1: winner_takes_all, 1000 credits
	// Fixtures: user 2 (score 95.0) rank 1, user 4 (score 80.0) rank 2
	// Only user 2 should get track 1 deposit
	txns2, _, _ := ListTransactions(db.DefaultContext, 2, db.ListOptions{Page: 1, PageSize: 50})
	var track1u2 int64
	for _, tx := range txns2 {
		if tx.Reference == "hackathon:1/track:1:rank:1" {
			track1u2 += tx.Amount
		}
	}
	assert.Equal(t, int64(1000), track1u2)

	// User 4 should NOT get credits from track 1 (rank 2)
	txns4, _, _ := ListTransactions(db.DefaultContext, 4, db.ListOptions{Page: 1, PageSize: 50})
	for _, tx := range txns4 {
		assert.NotEqual(t, "hackathon:1/track:1:rank:2", tx.Reference)
	}
}

func TestDistributeHackathonCredits_Tiered(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// Track 2: tiered [60/30/10] with 1000 credits
	// Fixtures: user 2 (90.0) rank 1, user 4 (85.0) rank 2, user 5 (70.0) rank 3
	txns2, _, _ := ListTransactions(db.DefaultContext, 2, db.ListOptions{Page: 1, PageSize: 50})
	var t2u2 int64
	for _, tx := range txns2 {
		if tx.Reference == "hackathon:1/track:2:rank:1" {
			t2u2 += tx.Amount
		}
	}
	assert.Equal(t, int64(600), t2u2)

	txns4, _, _ := ListTransactions(db.DefaultContext, 4, db.ListOptions{Page: 1, PageSize: 50})
	var t2u4 int64
	for _, tx := range txns4 {
		if tx.Reference == "hackathon:1/track:2:rank:2" {
			t2u4 += tx.Amount
		}
	}
	assert.Equal(t, int64(300), t2u4)

	txns5, _, _ := ListTransactions(db.DefaultContext, 5, db.ListOptions{Page: 1, PageSize: 50})
	var t2u5 int64
	for _, tx := range txns5 {
		if tx.Reference == "hackathon:1/track:2:rank:3" {
			t2u5 += tx.Amount
		}
	}
	assert.Equal(t, int64(100), t2u5)
}

func TestDistributeHackathonCredits_Equal(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// Track 3: equal, 1000 credits, 2 users (user 2 score 88.0, user 4 score 88.0)
	txns2, _, _ := ListTransactions(db.DefaultContext, 2, db.ListOptions{Page: 1, PageSize: 50})
	var t3u2 int64
	for _, tx := range txns2 {
		if tx.Reference == "hackathon:1/track:3:rank:1" {
			t3u2 += tx.Amount
		}
	}
	assert.Equal(t, int64(500), t3u2)

	txns4, _, _ := ListTransactions(db.DefaultContext, 4, db.ListOptions{Page: 1, PageSize: 50})
	var t3u4 int64
	for _, tx := range txns4 {
		if tx.Reference == "hackathon:1/track:3:rank:2" {
			t3u4 += tx.Amount
		}
	}
	assert.Equal(t, int64(500), t3u4)
}

func TestDistributeHackathonCredits_EqualRemainder(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	track := unittest.AssertExistsAndLoadBean(t, &hackforger_model.HackathonTrack{ID: 3})
	track.PrizeCredits = 999
	require.NoError(t, hackforger_model.UpdateTrack(db.DefaultContext, track))

	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// 999 / 2 = 499, remainder 1 → rank 1 (user 2) gets 500
	txns2, _, _ := ListTransactions(db.DefaultContext, 2, db.ListOptions{Page: 1, PageSize: 50})
	var t3u2 int64
	for _, tx := range txns2 {
		if tx.Reference == "hackathon:1/track:3:rank:1" {
			t3u2 += tx.Amount
		}
	}
	assert.Equal(t, int64(500), t3u2)

	txns4, _, _ := ListTransactions(db.DefaultContext, 4, db.ListOptions{Page: 1, PageSize: 50})
	var t3u4 int64
	for _, tx := range txns4 {
		if tx.Reference == "hackathon:1/track:3:rank:2" {
			t3u4 += tx.Amount
		}
	}
	assert.Equal(t, int64(499), t3u4)
}
```

### E3. Tasks 10/14/15: `ctx.PathParamInt64` does not exist

Forgejo uses `ctx.ParamsInt64(":id")` (with colon prefix), NOT `ctx.PathParamInt64("id")`. Fix all occurrences in:
- Task 14 web handlers: `ctx.ParamsInt64(":id")` for option ID, `ctx.ParamsInt64(":oid")` for order ID
- Task 15 API handlers: same pattern

### E4. Task 10: Complete CancelOrder refactoring

The plan's note about CancelOrder is incomplete. Here is the full replacement:

```go
func CancelOrder(ctx context.Context, admin *user_model.User, orderID int64) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}

	var order *hackforger_model.RedeemOrder
	var optName string

	err := db.WithTx(ctx, func(ctx context.Context) error {
		var err error
		order, err = hackforger_model.GetRedeemOrderByID(ctx, orderID)
		if err != nil {
			return err
		}
		if order.Status != hackforger_model.OrderStatusPending {
			return ErrOrderNotPending{OrderID: orderID, Status: order.Status}
		}

		order.Status = hackforger_model.OrderStatusCancelled
		if err := hackforger_model.UpdateRedeemOrder(ctx, order); err != nil {
			return err
		}

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
			Reference: fmt.Sprintf("order_%d_refund", orderID),
			Note:      "Order cancelled and refunded",
		}
		_, err = db.GetEngine(ctx).Insert(tx)

		if opt, _ := hackforger_model.GetRedeemOptionByID(ctx, order.OptionID); opt != nil {
			optName = opt.Name
		}
		return err
	})
	if err != nil {
		return err
	}

	notifyOrderStatusChange(ctx, order, admin.ID, hackforger_model.ActionOrderCancelled, optName)
	return nil
}
```

### E5. Task 10 Step 6: Web handler must read form delivery fields

The plan incorrectly passes `""` for delivery fields. The web handler should read form values:

```go
note := ctx.Req.FormValue("note")
deliveryType := ctx.Req.FormValue("delivery_type")
deliveryValue := ctx.Req.FormValue("delivery_value")
hackforger_service.FulfillOrder(ctx, ctx.Doer, orderID, note, deliveryType, deliveryValue)
```

### E6. Task 10: Update existing TestFulfillOrder

The existing test at `services/hackforger/credits_test.go` calls FulfillOrder with 4 args. Update to 6:

```go
hackforger_service.FulfillOrder(db.DefaultContext, admin, 1, "Fulfilled by admin", "", "")
```

### E7. Task 11/13: Redeem() variable hoisting for option

Declare `var capturedOption *hackforger_model.RedeemOption` before `db.WithTx`. Inside the tx, after `db.GetEngine(ctx).Get(option)`, assign `capturedOption = option`. After the tx block, use `capturedOption.FulfillMode` for the stock sync check.

### E8. Task 14: Use base.TplName constant and http.StatusOK

Add constant: `tplAdminRedeemOptionKeys base.TplName = "hackforger/credits/admin/keys"`
Use: `ctx.HTML(http.StatusOK, tplAdminRedeemOptionKeys)` instead of `ctx.HTML(200, "...")`

### E9. Task 18: Do not show err.Error() to users

Replace `ctx.Flash.Error(err.Error())` with:
```go
if hackforger_model.IsErrInvalidDistRatios(err) {
	ctx.Flash.Error(ctx.Tr("hackforger.hackathon.invalid_ratios"))
} else {
	ctx.Flash.Error(ctx.Tr("hackforger.hackathon.invalid_ratios"))
}
```

### E10. Task 19: i18n key format — strip `hackforger.` prefix

Keys under `[hackforger]` section must NOT include the section name. The template `ctx.Locale.Tr "hackforger.credits.admin.key_pool"` resolves to section `[hackforger]` key `credits.admin.key_pool`.

**Wrong:** `hackforger.credits.admin.key_pool = Key Pool`
**Correct:** `credits.admin.key_pool = Key Pool`

Strip `hackforger.` prefix from ALL keys in Task 19 Steps 1-2.

### E11. Task 19: Add feed action locale keys for types 58/59

The feed rendering uses keys from a different section (not `[hackforger]`). Add near the other `hackforger_xxx` action keys in the locale files:

**en-US** (near line 3505):
```ini
hackforger_order_fulfilled = fulfilled a redeem order
hackforger_order_cancelled = cancelled a redeem order
```

**zh-CN** (same location):
```ini
hackforger_order_fulfilled = 完成了一个兑换订单
hackforger_order_cancelled = 取消了一个兑换订单
```

Also add rendering branches in `templates/user/dashboard/feeds.tmpl` for `"hackforger_order_fulfilled"` and `"hackforger_order_cancelled"`.

### E12. Task 20: Add full Prerequisites section to E2E prompt

Follow the pattern from existing E2E prompts (e.g., `phase1-grant-e2e.md`) — include server startup sequence, app.ini copy, test accounts, and data cleanup instructions.

---

## Chunk 1: Migration + Model Layer

### Task 1: Database Migration

**Files:**
- Create: `models/forgejo_migrations/v14g_credits-fulfill-hackathon-prizes.go`

- [ ] **Step 1: Write the migration file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

func init() {
	registerMigration(&Migration{
		Description: "add credits fulfill and hackathon prize distribution fields",
		Upgrade:     addCreditsFulfillHackathonPrizes,
	})
}

// v14g structs for migration — only include columns being added/created.

type v14gHackathonTrack struct {
	PrizeDistMode   string `xorm:"VARCHAR(20) NOT NULL DEFAULT 'winner_takes_all'"`
	PrizeDistRatios string `xorm:"TEXT NOT NULL DEFAULT ''"`
}

func (v14gHackathonTrack) TableName() string { return "hackathon_track" }

type v14gRedeemOption struct {
	FulfillMode string `xorm:"VARCHAR(16) NOT NULL DEFAULT 'manual'"`
}

func (v14gRedeemOption) TableName() string { return "redeem_option" }

type v14gRedeemOrder struct {
	DeliveryType  string `xorm:"VARCHAR(32) NOT NULL DEFAULT ''"`
	DeliveryValue string `xorm:"TEXT NOT NULL DEFAULT ''"`
}

func (v14gRedeemOrder) TableName() string { return "redeem_order" }

type v14gRedeemOptionKey struct {
	ID          int64  `xorm:"pk autoincr"`
	OptionID    int64  `xorm:"INDEX NOT NULL"`
	KeyValue    string `xorm:"TEXT NOT NULL"`
	IsUsed      bool   `xorm:"NOT NULL DEFAULT false"`
	OrderID     int64  `xorm:"INDEX"`
	CreatedUnix int64  `xorm:"created"`
}

func (v14gRedeemOptionKey) TableName() string { return "redeem_option_key" }

func addCreditsFulfillHackathonPrizes(x *xorm.Engine) error {
	// Add columns to existing tables
	if err := x.Sync(new(v14gHackathonTrack)); err != nil {
		return err
	}
	if err := x.Sync(new(v14gRedeemOption)); err != nil {
		return err
	}
	if err := x.Sync(new(v14gRedeemOrder)); err != nil {
		return err
	}
	// Create new table
	return x.Sync(new(v14gRedeemOptionKey))
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/phase2-credits && go build ./models/forgejo_migrations/...`
Expected: compiles with no errors

- [ ] **Step 3: Commit**

```bash
git add models/forgejo_migrations/v14g_credits-fulfill-hackathon-prizes.go
git commit -m "migration(v14g): add credits fulfill + hackathon prize fields"
```

---

### Task 2: HackathonTrack Model — Add Distribution Fields

**Files:**
- Modify: `models/hackforger/hackathon_track.go` (struct at line 16, add after `PrizeCredits` line 24)

- [ ] **Step 1: Add fields to HackathonTrack struct**

In `models/hackforger/hackathon_track.go`, add after the `PrizeCredits` field (line 24):

```go
	PrizeDistMode   string             `xorm:"VARCHAR(20) NOT NULL DEFAULT 'winner_takes_all'"`
	PrizeDistRatios string             `xorm:"TEXT NOT NULL DEFAULT ''"`
```

- [ ] **Step 2: Add PrizeDistRatio helper type and validation**

Add at the end of `models/hackforger/hackathon_track.go`:

```go
// PrizeDistRatio represents one entry in the prize distribution ratios JSON.
type PrizeDistRatio struct {
	Rank int `json:"rank"`
	Pct  int `json:"pct"`
}

// ParsePrizeDistRatios parses the JSON string into a slice of PrizeDistRatio.
func ParsePrizeDistRatios(s string) ([]PrizeDistRatio, error) {
	if s == "" {
		return nil, nil
	}
	var ratios []PrizeDistRatio
	if err := json.Unmarshal([]byte(s), &ratios); err != nil {
		return nil, err
	}
	return ratios, nil
}

// ValidatePrizeDistRatios checks: all pct > 0, sum == 100, ranks contiguous 1..N.
func ValidatePrizeDistRatios(ratios []PrizeDistRatio) error {
	if len(ratios) == 0 {
		return ErrInvalidDistRatios{Reason: "ratios must not be empty"}
	}
	total := 0
	for i, r := range ratios {
		if r.Rank != i+1 {
			return ErrInvalidDistRatios{Reason: fmt.Sprintf("ranks must be contiguous 1..N, got rank %d at position %d", r.Rank, i+1)}
		}
		if r.Pct <= 0 {
			return ErrInvalidDistRatios{Reason: fmt.Sprintf("pct must be > 0 for rank %d", r.Rank)}
		}
		total += r.Pct
	}
	if total != 100 {
		return ErrInvalidDistRatios{Reason: fmt.Sprintf("sum of pct must be 100, got %d", total)}
	}
	return nil
}

// ErrInvalidDistRatios is returned when prize distribution ratios are invalid.
type ErrInvalidDistRatios struct {
	Reason string
}

func (err ErrInvalidDistRatios) Error() string {
	return fmt.Sprintf("invalid prize distribution ratios: %s", err.Reason)
}

func IsErrInvalidDistRatios(err error) bool {
	_, ok := err.(ErrInvalidDistRatios)
	return ok
}
```

Add `"encoding/json"` to the import block.

- [ ] **Step 3: Write tests for ratio validation**

Create or append to `models/hackforger/hackathon_track_test.go`:

```go
package hackforger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePrizeDistRatios_Valid(t *testing.T) {
	ratios := []PrizeDistRatio{
		{Rank: 1, Pct: 50},
		{Rank: 2, Pct: 30},
		{Rank: 3, Pct: 20},
	}
	assert.NoError(t, ValidatePrizeDistRatios(ratios))
}

func TestValidatePrizeDistRatios_SumNot100(t *testing.T) {
	ratios := []PrizeDistRatio{
		{Rank: 1, Pct: 50},
		{Rank: 2, Pct: 30},
	}
	err := ValidatePrizeDistRatios(ratios)
	require.Error(t, err)
	assert.True(t, IsErrInvalidDistRatios(err))
}

func TestValidatePrizeDistRatios_NonContiguous(t *testing.T) {
	ratios := []PrizeDistRatio{
		{Rank: 1, Pct: 50},
		{Rank: 3, Pct: 50},
	}
	err := ValidatePrizeDistRatios(ratios)
	require.Error(t, err)
	assert.True(t, IsErrInvalidDistRatios(err))
}

func TestValidatePrizeDistRatios_ZeroPct(t *testing.T) {
	ratios := []PrizeDistRatio{
		{Rank: 1, Pct: 100},
		{Rank: 2, Pct: 0},
	}
	err := ValidatePrizeDistRatios(ratios)
	require.Error(t, err)
	assert.True(t, IsErrInvalidDistRatios(err))
}

func TestValidatePrizeDistRatios_Empty(t *testing.T) {
	err := ValidatePrizeDistRatios(nil)
	require.Error(t, err)
}

func TestParsePrizeDistRatios(t *testing.T) {
	ratios, err := ParsePrizeDistRatios(`[{"rank":1,"pct":60},{"rank":2,"pct":40}]`)
	require.NoError(t, err)
	assert.Len(t, ratios, 2)
	assert.Equal(t, 60, ratios[0].Pct)
}
```

- [ ] **Step 4: Run tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/phase2-credits && go test ./models/hackforger/ -run TestValidatePrizeDistRatios -v && go test ./models/hackforger/ -run TestParsePrizeDistRatios -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add models/hackforger/hackathon_track.go models/hackforger/hackathon_track_test.go
git commit -m "feat(credits): add prize distribution mode + ratios to HackathonTrack"
```

---

### Task 3: RedeemOption Model — Add FulfillMode

**Files:**
- Modify: `models/hackforger/redeem_option.go` (struct at line 16, add after `IsActive` line 22)

- [ ] **Step 1: Add FulfillMode field**

In `models/hackforger/redeem_option.go`, add after the `IsActive` field (line 22):

```go
	FulfillMode string `xorm:"VARCHAR(16) NOT NULL DEFAULT 'manual'"`
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./models/hackforger/...`
Expected: compiles with no errors

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/redeem_option.go
git commit -m "feat(credits): add FulfillMode field to RedeemOption"
```

---

### Task 4: RedeemOrder Model — Add Delivery Fields

**Files:**
- Modify: `models/hackforger/redeem_order.go` (struct at line 25, add after `FulfillNote` line 31)

- [ ] **Step 1: Add delivery fields**

In `models/hackforger/redeem_order.go`, add after the `FulfillNote` field (line 31):

```go
	DeliveryType  string             `xorm:"VARCHAR(32) NOT NULL DEFAULT ''"`
	DeliveryValue string             `xorm:"TEXT NOT NULL DEFAULT ''"`
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./models/hackforger/...`
Expected: compiles with no errors

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/redeem_order.go
git commit -m "feat(credits): add delivery fields to RedeemOrder"
```

---

### Task 5: New Model — RedeemOptionKey

**Files:**
- Create: `models/hackforger/redeem_option_key.go`
- Create: `models/hackforger/redeem_option_key_test.go`

- [ ] **Step 1: Write the model file**

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

// RedeemOptionKey represents a single key/code in an auto-fulfill option's key pool.
type RedeemOptionKey struct {
	ID          int64              `xorm:"pk autoincr"`
	OptionID    int64              `xorm:"INDEX NOT NULL"`
	KeyValue    string             `xorm:"TEXT NOT NULL"`
	IsUsed      bool               `xorm:"NOT NULL DEFAULT false"`
	OrderID     int64              `xorm:"INDEX"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
}

func init() {
	db.RegisterModel(new(RedeemOptionKey))
}

// AddKeys bulk-inserts keys for an option.
func AddKeys(ctx context.Context, optionID int64, keys []string) error {
	beans := make([]any, 0, len(keys))
	for _, k := range keys {
		beans = append(beans, &RedeemOptionKey{
			OptionID: optionID,
			KeyValue: k,
		})
	}
	return db.Insert(ctx, beans...)
}

// ClaimKey atomically claims an unused key for an order.
// Uses SELECT then UPDATE for SQLite compatibility.
// Caller must wrap in db.WithTx for transactional safety.
func ClaimKey(ctx context.Context, optionID, orderID int64) (*RedeemOptionKey, error) {
	key := new(RedeemOptionKey)
	has, err := db.GetEngine(ctx).
		Where("option_id = ? AND is_used = ?", optionID, false).
		OrderBy("id ASC").
		ForUpdate().
		Limit(1).
		Get(key)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrOutOfStock{OptionID: optionID}
	}

	key.IsUsed = true
	key.OrderID = orderID
	if _, err := db.GetEngine(ctx).ID(key.ID).Cols("is_used", "order_id").Update(key); err != nil {
		return nil, err
	}
	return key, nil
}

// CountAvailableKeys returns the number of unused keys for an option.
func CountAvailableKeys(ctx context.Context, optionID int64) (int64, error) {
	return db.GetEngine(ctx).Where("option_id = ? AND is_used = ?", optionID, false).Count(new(RedeemOptionKey))
}

// CountTotalKeys returns total key count for an option.
func CountTotalKeys(ctx context.Context, optionID int64) (int64, error) {
	return db.GetEngine(ctx).Where("option_id = ?", optionID).Count(new(RedeemOptionKey))
}

// ListKeysByOption returns all keys for an option (admin use).
func ListKeysByOption(ctx context.Context, optionID int64) ([]*RedeemOptionKey, error) {
	var keys []*RedeemOptionKey
	err := db.GetEngine(ctx).Where("option_id = ?", optionID).OrderBy("id ASC").Find(&keys)
	return keys, err
}

// ErrKeyPoolEmpty is returned when no unused keys are available.
type ErrKeyPoolEmpty struct {
	OptionID int64
}

func (err ErrKeyPoolEmpty) Error() string {
	return fmt.Sprintf("key pool is empty for option [id: %d]", err.OptionID)
}

func (err ErrKeyPoolEmpty) Unwrap() error {
	return util.ErrNotExist
}

func IsErrKeyPoolEmpty(err error) bool {
	_, ok := err.(ErrKeyPoolEmpty)
	return ok
}
```

- [ ] **Step 2: Write tests**

Create `models/hackforger/redeem_option_key_test.go`:

```go
package hackforger

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddKeys(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	keys := []string{"KEY-001", "KEY-002", "KEY-003"}
	require.NoError(t, AddKeys(db.DefaultContext, 1, keys))

	total, err := CountTotalKeys(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
}

func TestClaimKey(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	require.NoError(t, AddKeys(db.DefaultContext, 1, []string{"CLAIM-A", "CLAIM-B"}))

	key, err := ClaimKey(db.DefaultContext, 1, 99)
	require.NoError(t, err)
	assert.Equal(t, "CLAIM-A", key.KeyValue)
	assert.True(t, key.IsUsed)
	assert.Equal(t, int64(99), key.OrderID)

	avail, err := CountAvailableKeys(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), avail)
}

func TestClaimKey_Empty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	_, err := ClaimKey(db.DefaultContext, 999, 1)
	require.Error(t, err)
	assert.True(t, IsErrOutOfStock(err))
}
```

- [ ] **Step 3: Add empty fixture file**

Create `models/fixtures/redeem_option_key.yml`:

```yaml
[]
```

- [ ] **Step 4: Run tests**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/phase2-credits && go test ./models/hackforger/ -run TestAddKeys -v && go test ./models/hackforger/ -run TestClaimKey -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add models/hackforger/redeem_option_key.go models/hackforger/redeem_option_key_test.go models/fixtures/redeem_option_key.yml
git commit -m "feat(credits): add RedeemOptionKey model for auto-fulfill key pool"
```

---

### Task 6: Action Types — Order Fulfilled/Cancelled

**Files:**
- Modify: `models/hackforger/action_types.go` (add after line 37 `ActionGrantRoundCancelled`)

- [ ] **Step 1: Add new action type constants**

In `models/hackforger/action_types.go`, add after `ActionGrantRoundCancelled` (line 37):

```go
	ActionOrderFulfilled    activities_model.ActionType = 58
	ActionOrderCancelled    activities_model.ActionType = 59
```

- [ ] **Step 2: Add string mappings**

In the `HackforgerActionTypeName` map (after line 66):

```go
	ActionOrderFulfilled:    "order_fulfilled",
	ActionOrderCancelled:    "order_cancelled",
```

- [ ] **Step 3: Verify compiles**

Run: `go build ./models/hackforger/...`
Expected: compiles with no errors

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/action_types.go
git commit -m "feat(credits): add order_fulfilled/cancelled action types (58/59)"
```

---

### Task 7: Test Fixtures — Hackathon Prize Distribution Data

**Files:**
- Modify: `models/fixtures/hackathon.yml`
- Modify: `models/fixtures/hackathon_track.yml`
- Modify: `models/fixtures/hackathon_submission.yml`
- Modify: `models/fixtures/hackathon_judge_score.yml`

These fixtures support the `distributeHackathonCredits` tests in Chunk 2.

- [ ] **Step 1: Add hackathon fixture**

Replace `models/fixtures/hackathon.yml` content:

```yaml
-
  id: 1
  name: "Test Hackathon"
  slug: "test-hackathon"
  status: 4
  creator_id: 1
  created_unix: 1609459200
  updated_unix: 1609459200
```

(Status 4 = Judging, so FinalizeHackathon can transition it to Finished.)

- [ ] **Step 2: Add track fixtures with distribution modes**

Replace `models/fixtures/hackathon_track.yml` content:

```yaml
-
  id: 1
  hackathon_id: 1
  name: "Winner Takes All Track"
  prize_credits: 1000
  prize_dist_mode: "winner_takes_all"
  prize_dist_ratios: ""
  created_unix: 1609459200
-
  id: 2
  hackathon_id: 1
  name: "Tiered Track"
  prize_credits: 1000
  prize_dist_mode: "tiered"
  prize_dist_ratios: '[{"rank":1,"pct":50},{"rank":2,"pct":30},{"rank":3,"pct":20}]'
  created_unix: 1609459200
-
  id: 3
  hackathon_id: 1
  name: "Equal Track"
  prize_credits: 1000
  prize_dist_mode: "equal"
  prize_dist_ratios: ""
  created_unix: 1609459200
-
  id: 4
  hackathon_id: 1
  name: "Zero Prize Track"
  prize_credits: 0
  prize_dist_mode: "winner_takes_all"
  prize_dist_ratios: ""
  created_unix: 1609459200
```

- [ ] **Step 3: Add submission fixtures with scores**

Replace `models/fixtures/hackathon_submission.yml` content:

```yaml
# Track 1 (winner_takes_all): user 2 is rank 1
-
  id: 1
  hackathon_id: 1
  registration_id: 1
  user_id: 2
  track_id: 1
  title: "Submission A"
  status: 2
  total_score: 9.5
  rank: 1
  created_unix: 1609459200
-
  id: 2
  hackathon_id: 1
  registration_id: 2
  user_id: 3
  track_id: 1
  title: "Submission B"
  status: 2
  total_score: 7.0
  rank: 2
  created_unix: 1609459200
# Track 2 (tiered): 3 users with scores
-
  id: 3
  hackathon_id: 1
  registration_id: 3
  user_id: 4
  track_id: 2
  title: "Submission C"
  status: 2
  total_score: 9.0
  rank: 1
  created_unix: 1609459200
-
  id: 4
  hackathon_id: 1
  registration_id: 4
  user_id: 5
  track_id: 2
  title: "Submission D"
  status: 2
  total_score: 8.0
  rank: 2
  created_unix: 1609459200
-
  id: 5
  hackathon_id: 1
  registration_id: 5
  user_id: 6
  track_id: 2
  title: "Submission E"
  status: 2
  total_score: 7.0
  rank: 3
  created_unix: 1609459200
# Track 3 (equal): 2 users with scores
-
  id: 6
  hackathon_id: 1
  registration_id: 6
  user_id: 7
  track_id: 3
  title: "Submission F"
  status: 2
  total_score: 8.5
  rank: 1
  created_unix: 1609459200
-
  id: 7
  hackathon_id: 1
  registration_id: 7
  user_id: 8
  track_id: 3
  title: "Submission G"
  status: 2
  total_score: 6.0
  rank: 2
  created_unix: 1609459200
```

- [ ] **Step 4: Add redeem fixtures for fulfill tests**

Append to `models/fixtures/redeem_option.yml` (file likely exists with fixture ID 1):

```yaml
-
  id: 2
  name: "Auto License Key"
  description: "Software license"
  cost: 100
  stock: 2
  is_active: true
  fulfill_mode: "auto"
  created_unix: 1609459200
```

Append to `models/fixtures/redeem_order.yml` (add pending orders for batch test):

```yaml
-
  id: 2
  user_id: 3
  option_id: 1
  cost: 500
  status: "pending"
  created_unix: 1609459300
-
  id: 3
  user_id: 5
  option_id: 1
  cost: 500
  status: "pending"
  created_unix: 1609459400
```

- [ ] **Step 5: Commit**

```bash
git add models/fixtures/
git commit -m "test: add fixtures for hackathon prize distribution + fulfill tests"
```

---

## Chunk 2: Hackathon -> Credits Service

### Task 8: distributeHackathonCredits Service Function

**Files:**
- Modify: `services/hackforger/hackathon.go` (add function, wire into FinalizeHackathon at line 193)
- Create or modify: `services/hackforger/hackathon_test.go`

- [ ] **Step 1: Write failing tests for all 3 distribution modes**

Create `services/hackforger/hackathon_credits_test.go`:

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

func TestDistributeHackathonCredits_WinnerTakesAll(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Track 1: winner_takes_all, PrizeCredits=1000, user 2 has highest score
	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// User 2 (rank 1 in track 1) should get 1000
	acct, err := GetOrCreateCreditAccount(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), acct.Balance)

	// User 3 (rank 2 in track 1) should get 0 from this track
	acct3, err := GetOrCreateCreditAccount(db.DefaultContext, 3)
	require.NoError(t, err)
	// Balance may include credits from other tracks — check transaction reference
	txns, _, err := ListTransactions(db.DefaultContext, 3, db.ListOptions{Page: 1, PageSize: 50})
	require.NoError(t, err)
	hasTrack1 := false
	for _, tx := range txns {
		if tx.Reference == "hackathon:1/track:1:rank:2" {
			hasTrack1 = true
		}
	}
	assert.False(t, hasTrack1, "user 3 should NOT receive credits from winner_takes_all track")
}

func TestDistributeHackathonCredits_Tiered(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// Track 2: tiered [50/30/20] with 1000 credits
	// User 4: rank 1 -> 500, User 5: rank 2 -> 300, User 6: rank 3 -> 200
	acct4, _ := GetOrCreateCreditAccount(db.DefaultContext, 4)
	acct5, _ := GetOrCreateCreditAccount(db.DefaultContext, 5)
	acct6, _ := GetOrCreateCreditAccount(db.DefaultContext, 6)

	// User 4 may have initial balance from fixture (1000), so check delta via transactions
	txns4, _, _ := ListTransactions(db.DefaultContext, 4, db.ListOptions{Page: 1, PageSize: 50})
	var track2Amount int64
	for _, tx := range txns4 {
		if tx.Reference == "hackathon:1/track:2:rank:1" {
			track2Amount += tx.Amount
		}
	}
	assert.Equal(t, int64(500), track2Amount, "user 4 should get 500 from tiered track")

	_ = acct4
	_ = acct5
	_ = acct6

	txns5, _, _ := ListTransactions(db.DefaultContext, 5, db.ListOptions{Page: 1, PageSize: 50})
	var t2u5 int64
	for _, tx := range txns5 {
		if tx.Reference == "hackathon:1/track:2:rank:2" {
			t2u5 += tx.Amount
		}
	}
	assert.Equal(t, int64(300), t2u5)

	txns6, _, _ := ListTransactions(db.DefaultContext, 6, db.ListOptions{Page: 1, PageSize: 50})
	var t2u6 int64
	for _, tx := range txns6 {
		if tx.Reference == "hackathon:1/track:2:rank:3" {
			t2u6 += tx.Amount
		}
	}
	assert.Equal(t, int64(200), t2u6)
}

func TestDistributeHackathonCredits_Equal(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// Track 3: equal, 1000 credits, 2 users -> 500 each
	txns7, _, _ := ListTransactions(db.DefaultContext, 7, db.ListOptions{Page: 1, PageSize: 50})
	var t3u7 int64
	for _, tx := range txns7 {
		if tx.Reference == "hackathon:1/track:3:rank:1" {
			t3u7 += tx.Amount
		}
	}
	assert.Equal(t, int64(500), t3u7)

	txns8, _, _ := ListTransactions(db.DefaultContext, 8, db.ListOptions{Page: 1, PageSize: 50})
	var t3u8 int64
	for _, tx := range txns8 {
		if tx.Reference == "hackathon:1/track:3:rank:2" {
			t3u8 += tx.Amount
		}
	}
	assert.Equal(t, int64(500), t3u8)
}

func TestDistributeHackathonCredits_EqualRemainder(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Temporarily set track 3 prize to 999 to test remainder
	track := unittest.AssertExistsAndLoadBean(t, &hackforger_model.HackathonTrack{ID: 3})
	track.PrizeCredits = 999
	require.NoError(t, hackforger_model.UpdateTrack(db.DefaultContext, track))

	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// 999 / 2 = 499 per user, remainder 1 goes to rank 1 (user 7)
	txns7, _, _ := ListTransactions(db.DefaultContext, 7, db.ListOptions{Page: 1, PageSize: 50})
	var t3u7 int64
	for _, tx := range txns7 {
		if tx.Reference == "hackathon:1/track:3:rank:1" {
			t3u7 += tx.Amount
		}
	}
	assert.Equal(t, int64(500), t3u7) // 499 + 1 remainder

	txns8, _, _ := ListTransactions(db.DefaultContext, 8, db.ListOptions{Page: 1, PageSize: 50})
	var t3u8 int64
	for _, tx := range txns8 {
		if tx.Reference == "hackathon:1/track:3:rank:2" {
			t3u8 += tx.Amount
		}
	}
	assert.Equal(t, int64(499), t3u8)
}

func TestDistributeHackathonCredits_ZeroPrize(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Track 4 has PrizeCredits=0, should be skipped
	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)
	// No error, no deposits for track 4 — test passes if no panic
}
```

- [ ] **Step 2: Run tests — verify they fail**

Run: `go test ./services/hackforger/ -run TestDistributeHackathonCredits -v`
Expected: FAIL — `distributeHackathonCredits` undefined

- [ ] **Step 3: Implement distributeHackathonCredits**

Add to `services/hackforger/hackathon.go`:

```go
// distributeHackathonCredits awards prize credits for each track
// based on the track's distribution mode and per-track score ranking.
func distributeHackathonCredits(ctx context.Context, hackathonID int64) error {
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, hackathonID)
	if err != nil {
		return err
	}

	for _, track := range tracks {
		if track.PrizeCredits <= 0 {
			continue
		}

		subs, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
			TrackID: track.ID,
			ListOptions: db.ListOptions{ListAll: true},
		})
		if err != nil {
			return err
		}

		// Per-track ranking: sort by TotalScore descending (independent of global Rank)
		sort.Slice(subs, func(i, j int) bool {
			if subs[i].TotalScore != subs[j].TotalScore {
				return subs[i].TotalScore > subs[j].TotalScore
			}
			return subs[i].ID < subs[j].ID
		})

		// Filter out unscored submissions
		var ranked []*hackforger_model.HackathonSubmission
		for _, s := range subs {
			if s.TotalScore > 0 {
				ranked = append(ranked, s)
			}
		}
		if len(ranked) == 0 {
			continue
		}

		switch track.PrizeDistMode {
		case "winner_takes_all":
			if err := Deposit(ctx, ranked[0].UserID, track.PrizeCredits,
				fmt.Sprintf("hackathon:%d/track:%d:rank:1", hackathonID, track.ID),
				fmt.Sprintf("Hackathon winner: %s", track.Name),
			); err != nil {
				return err
			}

		case "tiered":
			ratios, err := hackforger_model.ParsePrizeDistRatios(track.PrizeDistRatios)
			if err != nil {
				return err
			}
			var deposited int64
			for _, r := range ratios {
				idx := r.Rank - 1
				if idx >= len(ranked) {
					continue
				}
				amount := track.PrizeCredits * int64(r.Pct) / 100
				deposited += amount
				if err := Deposit(ctx, ranked[idx].UserID, amount,
					fmt.Sprintf("hackathon:%d/track:%d:rank:%d", hackathonID, track.ID, r.Rank),
					fmt.Sprintf("Hackathon rank %d: %s", r.Rank, track.Name),
				); err != nil {
					return err
				}
			}
			// Remainder to rank 1
			if remainder := track.PrizeCredits - deposited; remainder > 0 {
				if err := Deposit(ctx, ranked[0].UserID, remainder,
					fmt.Sprintf("hackathon:%d/track:%d:rank:1:remainder", hackathonID, track.ID),
					fmt.Sprintf("Hackathon rounding remainder: %s", track.Name),
				); err != nil {
					return err
				}
			}

		case "equal":
			n := int64(len(ranked))
			perUser := track.PrizeCredits / n
			remainder := track.PrizeCredits % n
			for i, sub := range ranked {
				amount := perUser
				if i == 0 {
					amount += remainder
				}
				if err := Deposit(ctx, sub.UserID, amount,
					fmt.Sprintf("hackathon:%d/track:%d:rank:%d", hackathonID, track.ID, i+1),
					fmt.Sprintf("Hackathon participant: %s", track.Name),
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
```

Add `"sort"` to imports.

- [ ] **Step 4: Wire into FinalizeHackathon**

In `services/hackforger/hackathon.go`, in `FinalizeHackathon()`, add after line 193 (`CalculateRanks` succeeds) and before `UpdateHackathonStatus`:

```go
	if err := distributeHackathonCredits(ctx, h.ID); err != nil {
		return err
	}
```

- [ ] **Step 5: Run tests — verify they pass**

Run: `go test ./services/hackforger/ -run TestDistributeHackathonCredits -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add services/hackforger/hackathon.go services/hackforger/hackathon_credits_test.go
git commit -m "feat(credits): hackathon prize distribution on finalize (3 modes)"
```

---

## Chunk 3: Fulfill Flow — Service Layer

### Task 9: AudienceDirectUser in Notifier (Separate Commit)

**Files:**
- Modify: `services/hackforger/notifier.go` (lines 35-40 for enum, ~line 97 for switch)

- [ ] **Step 1: Add AudienceDirectUser enum value**

In `services/hackforger/notifier.go`, add after `AudienceRepoWatchers` (line 39):

```go
	AudienceDirectUser                       // Visible to a specific user (opts.TargetUserID)
```

- [ ] **Step 2: Add TargetUserID to opts**

In `HackforgerActionOpts` struct (line 43), add:

```go
	TargetUserID int64 // used when AudienceType == AudienceDirectUser
```

- [ ] **Step 3: Add switch case in PublishHackforgerAction**

In the switch block (around line 97), add a new case before the closing `}`:

```go
	case AudienceDirectUser:
		if opts.TargetUserID > 0 && opts.TargetUserID != opts.ActUserID {
			directAction := &activities_model.Action{
				ActUserID:   opts.ActUserID,
				UserID:      opts.TargetUserID,
				OpType:      opts.OpType,
				Content:     contentStr,
				RepoID:      opts.RepoID,
				CreatedUnix: now,
			}
			if _, err := db.GetEngine(ctx).Insert(directAction); err != nil {
				return err
			}
		}
```

- [ ] **Step 4: Verify compiles**

Run: `go build ./services/hackforger/...`
Expected: compiles with no errors

- [ ] **Step 5: Commit (separate, isolated)**

```bash
git add services/hackforger/notifier.go
git commit -m "feat(feed): add AudienceDirectUser for targeted notifications

Isolated commit — Line-B feed rework can rebase independently."
```

---

### Task 10: Update FulfillOrder + Add notifyOrderStatusChange

**Files:**
- Modify: `services/hackforger/credits.go` (FulfillOrder at line 265, CancelOrder at line 285)

- [ ] **Step 1: Write test for updated FulfillOrder with delivery fields**

Add to `services/hackforger/credits_test.go`:

```go
func TestFulfillOrder_WithDelivery(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	err := FulfillOrder(db.DefaultContext, admin, 1, "Enjoy!", "license_key", "ABCD-1234-EFGH")
	require.NoError(t, err)

	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order.Status)
	assert.Equal(t, "license_key", order.DeliveryType)
	assert.Equal(t, "ABCD-1234-EFGH", order.DeliveryValue)
	assert.Equal(t, "Enjoy!", order.FulfillNote)
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `go test ./services/hackforger/ -run TestFulfillOrder_WithDelivery -v`
Expected: FAIL — too many arguments to FulfillOrder

- [ ] **Step 3: Update FulfillOrder signature and implementation**

In `services/hackforger/credits.go`, replace the `FulfillOrder` function (line 265):

```go
// FulfillOrder marks a pending order as fulfilled with optional delivery info.
func FulfillOrder(ctx context.Context, admin *user_model.User, orderID int64, note, deliveryType, deliveryValue string) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}

	order, err := hackforger_model.GetRedeemOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status != hackforger_model.OrderStatusPending {
		return ErrOrderNotPending{OrderID: orderID, Status: order.Status}
	}

	order.Status = hackforger_model.OrderStatusFulfilled
	order.FulfillNote = note
	order.DeliveryType = deliveryType
	order.DeliveryValue = deliveryValue
	if err := hackforger_model.UpdateRedeemOrder(ctx, order); err != nil {
		return err
	}

	// Load option name for notification
	opt, _ := hackforger_model.GetRedeemOptionByID(ctx, order.OptionID)
	optName := ""
	if opt != nil {
		optName = opt.Name
	}
	notifyOrderStatusChange(ctx, order, admin.ID, hackforger_model.ActionOrderFulfilled, optName)
	return nil
}
```

- [ ] **Step 4: Add notifyOrderStatusChange helper**

Add to `services/hackforger/credits.go`:

```go
// notifyOrderStatusChange publishes a feed event for order status changes.
// Isolated from feed core — targets the order owner directly.
func notifyOrderStatusChange(ctx context.Context, order *hackforger_model.RedeemOrder, adminID int64, actionType activities_model.ActionType, optionName string) {
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    adminID,
		OpType:       actionType,
		AudienceType: AudienceDirectUser,
		TargetUserID: order.UserID,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "credits",
			EntityName: optionName,
			Extra:      map[string]any{"order_id": order.ID},
		},
	})
}
```

Add `activities_model "forgejo.org/models/activities"` to imports if not present.

- [ ] **Step 5: Update CancelOrder to also notify**

In `CancelOrder`, add before the final `return err` (inside the tx, after the refund transaction insert):

```go
	// Notify after tx completes — load option name
	var optName string
	if opt, _ := hackforger_model.GetRedeemOptionByID(ctx, order.OptionID); opt != nil {
		optName = opt.Name
	}
```

And after the `db.WithTx` block, before the final `return`:

```go
	notifyOrderStatusChange(ctx, order, admin.ID, hackforger_model.ActionOrderCancelled, optName)
```

Note: `order` and `optName` need to be captured outside the transaction closure. Refactor CancelOrder to capture the order/optName from inside the tx.

- [ ] **Step 6: Fix all callers of FulfillOrder (web + API handlers)**

In `routers/web/hackforger/credits.go`, update the `AdminCreditOrdersFulfill` handler call from:
```go
FulfillOrder(ctx, admin, orderID, note)
```
to:
```go
FulfillOrder(ctx, admin, orderID, note, "", "")
```

In `routers/api/v1/hackforger/credits.go`, update similarly. Also add `DeliveryType` and `DeliveryValue` to the `FulfillOrderForm`:

```go
type FulfillOrderForm struct {
	Note          string `json:"note"`
	DeliveryType  string `json:"delivery_type"`
	DeliveryValue string `json:"delivery_value"`
}
```

And pass them through:
```go
FulfillOrder(ctx, admin, orderID, form.Note, form.DeliveryType, form.DeliveryValue)
```

- [ ] **Step 7: Run tests — verify they pass**

Run: `go test ./services/hackforger/ -run TestFulfillOrder -v`
Expected: all PASS (including old TestFulfillOrder test which may need signature update)

- [ ] **Step 8: Commit**

```bash
git add services/hackforger/credits.go services/hackforger/credits_test.go routers/web/hackforger/credits.go routers/api/v1/hackforger/credits.go
git commit -m "feat(credits): delivery details + order status notifications (F1+F2)"
```

---

### Task 11: Auto-Fulfill in Redeem()

**Files:**
- Modify: `services/hackforger/credits.go` (Redeem function at line 60)

- [ ] **Step 1: Write test for auto-fulfill**

Add to `services/hackforger/credits_test.go`:

```go
func TestRedeem_AutoFulfill(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Setup: user 4 has balance 1000, option 2 is auto-fulfill with cost 100
	// Add keys to the pool
	require.NoError(t, hackforger_model.AddKeys(db.DefaultContext, 2, []string{"AUTO-KEY-1", "AUTO-KEY-2"}))

	order, err := Redeem(db.DefaultContext, 4, 2)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order.Status)
	assert.Equal(t, "license_key", order.DeliveryType)
	assert.Equal(t, "AUTO-KEY-1", order.DeliveryValue)

	// Balance should be 1000 - 100 = 900
	acct, _ := GetOrCreateCreditAccount(db.DefaultContext, 4)
	assert.Equal(t, int64(900), acct.Balance)

	// Available keys should be 1
	avail, _ := hackforger_model.CountAvailableKeys(db.DefaultContext, 2)
	assert.Equal(t, int64(1), avail)
}

func TestRedeem_AutoFulfill_NoKeys(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Option 2 is auto-fulfill but has no keys
	_, err := Redeem(db.DefaultContext, 4, 2)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrOutOfStock(err))

	// Balance should be unchanged (tx rolled back)
	acct, _ := GetOrCreateCreditAccount(db.DefaultContext, 4)
	assert.Equal(t, int64(1000), acct.Balance)
}
```

- [ ] **Step 2: Run tests — verify they fail**

Run: `go test ./services/hackforger/ -run TestRedeem_AutoFulfill -v`
Expected: FAIL

- [ ] **Step 3: Modify Redeem() to support auto-fulfill**

In `services/hackforger/credits.go`, inside the `Redeem` function's `db.WithTx` block, after the order is inserted (around line 130), add:

```go
		// Auto-fulfill: claim a key from the pool
		if option.FulfillMode == "auto" {
			key, err := hackforger_model.ClaimKey(ctx, optionID, order.ID)
			if err != nil {
				return err // rolls back entire tx
			}
			order.Status = hackforger_model.OrderStatusFulfilled
			order.DeliveryType = "license_key"
			order.DeliveryValue = key.KeyValue
			if _, err := db.GetEngine(ctx).ID(order.ID).
				Cols("status", "delivery_type", "delivery_value").
				Update(order); err != nil {
				return err
			}
		}
```

- [ ] **Step 4: Run tests — verify they pass**

Run: `go test ./services/hackforger/ -run TestRedeem_AutoFulfill -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add services/hackforger/credits.go services/hackforger/credits_test.go
git commit -m "feat(credits): auto-fulfill with key pool on redemption (F4)"
```

---

### Task 12: BatchFulfillOrders

**Files:**
- Modify: `services/hackforger/credits.go`

- [ ] **Step 1: Write test**

Add to `services/hackforger/credits_test.go`:

```go
func TestBatchFulfillOrders(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// Orders 2 and 3 are pending (from fixtures)
	success, failed, err := BatchFulfillOrders(db.DefaultContext, admin,
		[]int64{2, 3}, "Batch done", "custom", "Shipped via batch")
	require.NoError(t, err)
	assert.Equal(t, 2, success)
	assert.Empty(t, failed)

	order2, _ := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 2)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order2.Status)
	assert.Equal(t, "custom", order2.DeliveryType)

	order3, _ := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 3)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order3.Status)
}

func TestBatchFulfillOrders_PartialFailure(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// Order 1 already fulfilled in previous test setup, 2 is pending
	// Fulfill order 2 first to make it non-pending
	require.NoError(t, FulfillOrder(db.DefaultContext, admin, 2, "done", "", ""))

	success, failed, err := BatchFulfillOrders(db.DefaultContext, admin,
		[]int64{2, 3}, "Batch", "", "")
	require.NoError(t, err)
	assert.Equal(t, 1, success)  // only order 3
	assert.Len(t, failed, 1)    // order 2 fails (not pending)
}
```

- [ ] **Step 2: Run test — verify it fails**

Run: `go test ./services/hackforger/ -run TestBatchFulfillOrders -v`
Expected: FAIL — BatchFulfillOrders undefined

- [ ] **Step 3: Implement BatchFulfillOrders**

Add to `services/hackforger/credits.go`:

```go
// BatchFulfillOrders fulfills multiple pending orders. Each order is
// processed independently so partial success is possible.
func BatchFulfillOrders(ctx context.Context, admin *user_model.User, orderIDs []int64, note, deliveryType, deliveryValue string) (int, []int64, error) {
	if !admin.IsAdmin {
		return 0, nil, ErrNotAdmin{UserID: admin.ID}
	}

	var success int
	var failed []int64
	for _, oid := range orderIDs {
		if err := FulfillOrder(ctx, admin, oid, note, deliveryType, deliveryValue); err != nil {
			failed = append(failed, oid)
		} else {
			success++
		}
	}
	return success, failed, nil
}
```

- [ ] **Step 4: Run tests — verify they pass**

Run: `go test ./services/hackforger/ -run TestBatchFulfillOrders -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add services/hackforger/credits.go services/hackforger/credits_test.go
git commit -m "feat(credits): batch fulfill orders service function (F3)"
```

---

### Task 13: Key Pool Management Service Functions

**Files:**
- Modify: `services/hackforger/credits.go`

- [ ] **Step 1: Write tests**

Add to `services/hackforger/credits_test.go`:

```go
func TestAddKeysAndSyncStock(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// Option 2 is auto-fulfill
	err := AddKeysToOption(db.DefaultContext, admin, 2, []string{"K1", "K2", "K3"})
	require.NoError(t, err)

	opt, err := hackforger_model.GetRedeemOptionByID(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, opt.Stock) // stock synced to available key count
}

func TestGetKeyPoolStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	require.NoError(t, hackforger_model.AddKeys(db.DefaultContext, 2, []string{"S1", "S2"}))
	// Claim one key
	_, err := hackforger_model.ClaimKey(db.DefaultContext, 2, 99)
	require.NoError(t, err)

	total, avail, err := GetKeyPoolStatus(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, int64(1), avail)
}
```

- [ ] **Step 2: Implement service functions**

Add to `services/hackforger/credits.go`:

```go
// AddKeysToOption adds keys to an auto-fulfill option and syncs stock.
func AddKeysToOption(ctx context.Context, admin *user_model.User, optionID int64, keys []string) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}
	if err := hackforger_model.AddKeys(ctx, optionID, keys); err != nil {
		return err
	}
	return syncOptionStock(ctx, optionID)
}

// GetKeyPoolStatus returns total and available key counts for an option.
func GetKeyPoolStatus(ctx context.Context, optionID int64) (total, available int64, err error) {
	total, err = hackforger_model.CountTotalKeys(ctx, optionID)
	if err != nil {
		return
	}
	available, err = hackforger_model.CountAvailableKeys(ctx, optionID)
	return
}

// syncOptionStock updates the Stock field to match available key count.
func syncOptionStock(ctx context.Context, optionID int64) error {
	avail, err := hackforger_model.CountAvailableKeys(ctx, optionID)
	if err != nil {
		return err
	}
	opt, err := hackforger_model.GetRedeemOptionByID(ctx, optionID)
	if err != nil {
		return err
	}
	opt.Stock = int(avail)
	if avail == 0 {
		opt.IsActive = false
	}
	return hackforger_model.UpdateRedeemOption(ctx, opt)
}
```

Also update `Redeem()`: after the auto-fulfill block, add a stock sync call after the transaction:

After the `db.WithTx` block in `Redeem()`, before the feed event publish, add:

```go
	// Sync stock for auto-fulfill options after key claim
	if option.FulfillMode == "auto" {
		_ = syncOptionStock(ctx, optionID)
	}
```

Note: `option` needs to be captured from inside the tx. Declare `var option *hackforger_model.RedeemOption` before the tx and assign inside.

- [ ] **Step 3: Run tests**

Run: `go test ./services/hackforger/ -run "TestAddKeysAndSyncStock|TestGetKeyPoolStatus" -v`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/credits.go services/hackforger/credits_test.go
git commit -m "feat(credits): key pool management + stock sync service functions"
```

---

## Chunk 4: Routes, API + Templates

### Task 14: Web Routes — Key Pool Management + Batch Fulfill

**Files:**
- Modify: `routers/web/hackforger/credits.go` (add handlers)
- Modify: `routers/web/web.go` (register routes)

- [ ] **Step 1: Add key pool web handlers**

Add to `routers/web/hackforger/credits.go`:

```go
// AdminRedeemOptionKeys shows the key pool management page for an auto-fulfill option.
func AdminRedeemOptionKeys(ctx *context.Context) {
	optionID := ctx.PathParamInt64("id")
	opt, err := hackforger_model.GetRedeemOptionByID(ctx, optionID)
	if err != nil {
		ctx.ServerError("GetRedeemOptionByID", err)
		return
	}
	keys, err := hackforger_model.ListKeysByOption(ctx, optionID)
	if err != nil {
		ctx.ServerError("ListKeysByOption", err)
		return
	}
	total, avail, _ := hackforger_service.GetKeyPoolStatus(ctx, optionID)

	ctx.Data["Option"] = opt
	ctx.Data["Keys"] = keys
	ctx.Data["TotalKeys"] = total
	ctx.Data["AvailableKeys"] = avail
	ctx.HTML(200, "hackforger/credits/admin/keys")
}

// AdminRedeemOptionKeysAdd handles bulk key upload.
func AdminRedeemOptionKeysAdd(ctx *context.Context) {
	optionID := ctx.PathParamInt64("id")
	keysRaw := ctx.FormString("keys")
	lines := strings.Split(strings.TrimSpace(keysRaw), "\n")
	var keys []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			keys = append(keys, l)
		}
	}
	if len(keys) == 0 {
		ctx.Flash.Error(ctx.Tr("hackforger.credits.admin.keys_empty"))
		ctx.Redirect(fmt.Sprintf("/-/admin/credits/options/%d/keys", optionID))
		return
	}
	if err := hackforger_service.AddKeysToOption(ctx, ctx.Doer, optionID, keys); err != nil {
		ctx.ServerError("AddKeysToOption", err)
		return
	}
	ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.keys_added", len(keys)))
	ctx.Redirect(fmt.Sprintf("/-/admin/credits/options/%d/keys", optionID))
}

// AdminCreditOrdersBatchFulfill handles batch fulfillment of pending orders.
func AdminCreditOrdersBatchFulfill(ctx *context.Context) {
	orderIDsStr := ctx.FormStrings("order_ids")
	note := ctx.FormString("note")
	deliveryType := ctx.FormString("delivery_type")
	deliveryValue := ctx.FormString("delivery_value")

	var orderIDs []int64
	for _, s := range orderIDsStr {
		id, _ := strconv.ParseInt(s, 10, 64)
		if id > 0 {
			orderIDs = append(orderIDs, id)
		}
	}
	if len(orderIDs) == 0 {
		ctx.Flash.Error(ctx.Tr("hackforger.credits.admin.no_orders_selected"))
		ctx.Redirect("/-/admin/credits/orders")
		return
	}

	success, failed, err := hackforger_service.BatchFulfillOrders(ctx, ctx.Doer, orderIDs, note, deliveryType, deliveryValue)
	if err != nil {
		ctx.ServerError("BatchFulfillOrders", err)
		return
	}
	if len(failed) > 0 {
		ctx.Flash.Warning(ctx.Tr("hackforger.credits.admin.batch_partial", success, len(failed)))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.batch_success", success))
	}
	ctx.Redirect("/-/admin/credits/orders")
}
```

Add `"strings"` and `"strconv"` to imports.

- [ ] **Step 2: Register routes in web.go**

In `routers/web/web.go`, in the admin `/credits` group (around line 922), add:

```go
	m.Get("/options/{id}/keys", hackforger_web.AdminRedeemOptionKeys)
	m.Post("/options/{id}/keys", hackforger_web.AdminRedeemOptionKeysAdd)
	m.Post("/orders/batch-fulfill", hackforger_web.AdminCreditOrdersBatchFulfill)
```

- [ ] **Step 3: Verify compiles**

Run: `go build ./routers/...`
Expected: compiles with no errors

- [ ] **Step 4: Commit**

```bash
git add routers/web/hackforger/credits.go routers/web/web.go
git commit -m "feat(credits): web routes for key pool + batch fulfill (F3+F4)"
```

---

### Task 15: API Routes — Key Pool + Batch Fulfill

**Files:**
- Modify: `routers/api/v1/hackforger/credits.go` (add handlers)
- Modify: `routers/api/v1/api.go` (register routes)

- [ ] **Step 1: Add API forms and handlers**

Add to `routers/api/v1/hackforger/credits.go`:

```go
// AddKeysForm is the form for adding keys to an auto-fulfill option.
type AddKeysForm struct {
	Keys []string `json:"keys" binding:"Required"`
}

// BatchFulfillForm is the form for batch order fulfillment.
type BatchFulfillForm struct {
	OrderIDs      []int64 `json:"order_ids" binding:"Required"`
	Note          string  `json:"note"`
	DeliveryType  string  `json:"delivery_type"`
	DeliveryValue string  `json:"delivery_value"`
}

// AddKeys adds keys to an auto-fulfill option's key pool.
func AddKeys(ctx *context.APIContext) {
	form := web.GetForm(ctx).(*AddKeysForm)
	optionID := ctx.PathParamInt64("id")
	if err := hackforger_service.AddKeysToOption(ctx, ctx.Doer, optionID, form.Keys); err != nil {
		ctx.Error(http.StatusInternalServerError, "AddKeysToOption", err)
		return
	}
	total, avail, _ := hackforger_service.GetKeyPoolStatus(ctx, optionID)
	ctx.JSON(http.StatusOK, map[string]any{"total": total, "available": avail})
}

// ListKeys returns all keys for an option (admin only).
func ListKeys(ctx *context.APIContext) {
	optionID := ctx.PathParamInt64("id")
	keys, err := hackforger_model.ListKeysByOption(ctx, optionID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListKeysByOption", err)
		return
	}
	ctx.JSON(http.StatusOK, keys)
}

// BatchFulfill fulfills multiple pending orders.
func BatchFulfill(ctx *context.APIContext) {
	form := web.GetForm(ctx).(*BatchFulfillForm)
	success, failed, err := hackforger_service.BatchFulfillOrders(ctx, ctx.Doer, form.OrderIDs, form.Note, form.DeliveryType, form.DeliveryValue)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "BatchFulfillOrders", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]any{
		"success": success,
		"failed":  failed,
	})
}
```

- [ ] **Step 2: Register API routes in api.go**

In `routers/api/v1/api.go`, in the credits `/redeem` group (around line 1842), add:

```go
	m.Group("/options/{id}", func() {
		m.Post("/keys", reqToken(), reqSiteAdmin(), bind(hackforger_api.AddKeysForm{}), hackforger_api.AddKeys)
		m.Get("/keys", reqToken(), reqSiteAdmin(), hackforger_api.ListKeys)
	})
	m.Post("/orders/batch-fulfill", reqToken(), reqSiteAdmin(), bind(hackforger_api.BatchFulfillForm{}), hackforger_api.BatchFulfill)
```

- [ ] **Step 3: Verify compiles**

Run: `go build ./routers/...`
Expected: compiles with no errors

- [ ] **Step 4: Commit**

```bash
git add routers/api/v1/hackforger/credits.go routers/api/v1/api.go
git commit -m "feat(credits): API endpoints for key pool + batch fulfill (F3+F4)"
```

---

### Task 16: Templates — Admin Key Pool Page

**Files:**
- Create: `templates/hackforger/credits/admin/keys.tmpl`

- [ ] **Step 1: Create the template**

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content">
	<div class="ui container">
		<h2>{{ctx.Locale.Tr "hackforger.credits.admin.key_pool"}} — {{.Option.Name}}</h2>

		<div class="ui message">
			<p>{{ctx.Locale.Tr "hackforger.credits.admin.keys_status" .TotalKeys .AvailableKeys}}</p>
		</div>

		<h3>{{ctx.Locale.Tr "hackforger.credits.admin.add_keys"}}</h3>
		<form class="ui form" method="post" action="/-/admin/credits/options/{{.Option.ID}}/keys">
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.credits.admin.keys_textarea"}}</label>
				<textarea name="keys" rows="8" placeholder="KEY-001&#10;KEY-002&#10;KEY-003"></textarea>
			</div>
			<button class="ui primary button">{{ctx.Locale.Tr "hackforger.credits.admin.add_keys_button"}}</button>
		</form>

		<h3>{{ctx.Locale.Tr "hackforger.credits.admin.existing_keys"}}</h3>
		{{if .Keys}}
		<table class="ui celled table">
			<thead>
				<tr>
					<th>ID</th>
					<th>{{ctx.Locale.Tr "hackforger.credits.admin.key_value"}}</th>
					<th>{{ctx.Locale.Tr "hackforger.credits.admin.key_status"}}</th>
					<th>{{ctx.Locale.Tr "hackforger.credits.admin.key_order_id"}}</th>
				</tr>
			</thead>
			<tbody>
				{{range .Keys}}
				<tr>
					<td>{{.ID}}</td>
					<td><code>{{.KeyValue}}</code></td>
					<td>
						{{if .IsUsed}}
							<span class="ui red label">{{ctx.Locale.Tr "hackforger.credits.admin.key_used"}}</span>
						{{else}}
							<span class="ui green label">{{ctx.Locale.Tr "hackforger.credits.admin.key_available"}}</span>
						{{end}}
					</td>
					<td>{{if .OrderID}}{{.OrderID}}{{else}}-{{end}}</td>
				</tr>
				{{end}}
			</tbody>
		</table>
		{{else}}
		<div class="ui info message">{{ctx.Locale.Tr "hackforger.credits.admin.no_keys"}}</div>
		{{end}}

		<a class="ui button" href="/-/admin/credits/options">{{ctx.Locale.Tr "hackforger.credits.admin.back"}}</a>
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 2: Commit**

```bash
git add templates/hackforger/credits/admin/keys.tmpl
git commit -m "feat(credits): admin key pool management template"
```

---

### Task 17: Templates — Update Existing Pages

**Files:**
- Modify: `templates/hackforger/credits/admin/options.tmpl` (add FulfillMode + key pool link)
- Modify: `templates/hackforger/credits/admin/orders.tmpl` (batch fulfill + delivery)
- Modify: `templates/hackforger/credits/orders.tmpl` (user delivery info)

- [ ] **Step 1: Update admin options — add FulfillMode to create form**

In `templates/hackforger/credits/admin/options.tmpl`, in the create form (around line 25, before the submit button), add:

```html
		<div class="field">
			<label>{{ctx.Locale.Tr "hackforger.credits.admin.fulfill_mode"}}</label>
			<select name="fulfill_mode" class="ui dropdown">
				<option value="manual">{{ctx.Locale.Tr "hackforger.credits.admin.fulfill_manual"}}</option>
				<option value="auto">{{ctx.Locale.Tr "hackforger.credits.admin.fulfill_auto"}}</option>
			</select>
		</div>
```

In the options table, add a "Mode" column and a "Keys" link for auto options:

After the "Active" column header, add `<th>{{ctx.Locale.Tr "hackforger.credits.admin.fulfill_mode"}}</th>`.

In each row, add the mode cell and a keys link:

```html
	<td>
		{{if eq .FulfillMode "auto"}}
			<span class="ui blue label">Auto</span>
			<a href="/-/admin/credits/options/{{.ID}}/keys">{{ctx.Locale.Tr "hackforger.credits.admin.manage_keys"}}</a>
		{{else}}
			<span class="ui label">Manual</span>
		{{end}}
	</td>
```

- [ ] **Step 2: Update admin orders — add batch fulfill UI**

In `templates/hackforger/credits/admin/orders.tmpl`:

Add a form wrapper around the table with batch action buttons at the top:

Before the `<table>` tag:
```html
<form id="batch-fulfill-form" method="post" action="/-/admin/credits/orders/batch-fulfill">
	<div id="batch-actions" style="display:none" class="ui segment">
		<div class="field">
			<label>{{ctx.Locale.Tr "hackforger.credits.admin.fulfill_note"}}</label>
			<input name="note" type="text" placeholder="{{ctx.Locale.Tr "hackforger.credits.admin.fulfill_note"}}">
		</div>
		<div class="inline fields">
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.credits.admin.delivery_type"}}</label>
				<select name="delivery_type" class="ui dropdown">
					<option value="">-</option>
					<option value="license_key">{{ctx.Locale.Tr "hackforger.credits.delivery.license_key"}}</option>
					<option value="download_link">{{ctx.Locale.Tr "hackforger.credits.delivery.download_link"}}</option>
					<option value="tracking">{{ctx.Locale.Tr "hackforger.credits.delivery.tracking"}}</option>
					<option value="custom">{{ctx.Locale.Tr "hackforger.credits.delivery.custom"}}</option>
				</select>
			</div>
			<div class="field">
				<label>{{ctx.Locale.Tr "hackforger.credits.admin.delivery_value"}}</label>
				<input name="delivery_value" type="text">
			</div>
		</div>
		<button class="ui primary button" type="submit">{{ctx.Locale.Tr "hackforger.credits.admin.batch_fulfill"}}</button>
	</div>
```

Add a checkbox column to each pending row:
```html
	<td>
		{{if eq .Status "pending"}}
		<input type="checkbox" name="order_ids" value="{{.ID}}" onchange="toggleBatchActions()">
		{{end}}
	</td>
```

After the table, close the form and add JavaScript:
```html
</form>
<script>
function toggleBatchActions() {
	const checked = document.querySelectorAll('input[name="order_ids"]:checked');
	document.getElementById('batch-actions').style.display = checked.length > 0 ? '' : 'none';
}
</script>
```

Also update the individual fulfill form to include delivery fields:
```html
	<div class="field">
		<select name="delivery_type" class="ui dropdown">
			<option value="">-</option>
			<option value="license_key">{{ctx.Locale.Tr "hackforger.credits.delivery.license_key"}}</option>
			<option value="download_link">{{ctx.Locale.Tr "hackforger.credits.delivery.download_link"}}</option>
			<option value="tracking">{{ctx.Locale.Tr "hackforger.credits.delivery.tracking"}}</option>
			<option value="custom">{{ctx.Locale.Tr "hackforger.credits.delivery.custom"}}</option>
		</select>
	</div>
	<div class="field">
		<input name="delivery_value" type="text" placeholder="{{ctx.Locale.Tr "hackforger.credits.admin.delivery_value"}}">
	</div>
```

- [ ] **Step 3: Update user orders — show delivery info**

In `templates/hackforger/credits/orders.tmpl`, add a "Delivery" column after "Status":

Column header:
```html
<th>{{ctx.Locale.Tr "hackforger.credits.delivery"}}</th>
```

Cell (in each row):
```html
<td>
	{{if and (eq .Status "fulfilled") (ne .DeliveryType "")}}
		<span class="ui label">{{.DeliveryType}}</span>
		<code>{{.DeliveryValue}}</code>
	{{else}}
		-
	{{end}}
</td>
```

- [ ] **Step 4: Commit**

```bash
git add templates/hackforger/credits/admin/options.tmpl templates/hackforger/credits/admin/orders.tmpl templates/hackforger/credits/orders.tmpl
git commit -m "feat(credits): update templates for delivery, batch fulfill, key pool UI"
```

---

### Task 18: Hackathon Manage Template — Prize Distribution UI

**Files:**
- Modify: `templates/hackforger/hackathon/manage.tmpl`

- [ ] **Step 1: Add prize distribution fields to track form section**

Find the track creation/editing section in `manage.tmpl`. After the `prize_credits` input field, add:

```html
<div class="field">
	<label>{{ctx.Locale.Tr "hackforger.hackathon.prize_dist_mode"}}</label>
	<select name="prize_dist_mode" class="ui dropdown" id="prize-dist-mode">
		<option value="winner_takes_all" {{if eq .PrizeDistMode "winner_takes_all"}}selected{{end}}>
			{{ctx.Locale.Tr "hackforger.hackathon.dist_winner_takes_all"}}
		</option>
		<option value="tiered" {{if eq .PrizeDistMode "tiered"}}selected{{end}}>
			{{ctx.Locale.Tr "hackforger.hackathon.dist_tiered"}}
		</option>
		<option value="equal" {{if eq .PrizeDistMode "equal"}}selected{{end}}>
			{{ctx.Locale.Tr "hackforger.hackathon.dist_equal"}}
		</option>
	</select>
</div>

<div class="field" id="tiered-ratios" style="{{if ne .PrizeDistMode "tiered"}}display:none{{end}}">
	<label>{{ctx.Locale.Tr "hackforger.hackathon.dist_ratios"}}</label>
	<textarea name="prize_dist_ratios" rows="4" placeholder='[{"rank":1,"pct":50},{"rank":2,"pct":30},{"rank":3,"pct":20}]'>{{.PrizeDistRatios}}</textarea>
	<p class="help">{{ctx.Locale.Tr "hackforger.hackathon.dist_ratios_help"}}</p>
</div>

<script>
document.getElementById('prize-dist-mode').addEventListener('change', function() {
	document.getElementById('tiered-ratios').style.display = this.value === 'tiered' ? '' : 'none';
});
</script>
```

Note: The exact location depends on the existing template structure. Find the track form section by searching for `prize_credits` in the template.

- [ ] **Step 2: Update track create/update handler to parse new fields**

In the web handler for track creation/update (likely in `routers/web/hackforger/hackathon.go`), read the new form fields:

```go
track.PrizeDistMode = ctx.FormString("prize_dist_mode")
if track.PrizeDistMode == "" {
	track.PrizeDistMode = "winner_takes_all"
}
if track.PrizeDistMode == "tiered" {
	ratiosJSON := ctx.FormString("prize_dist_ratios")
	ratios, err := hackforger_model.ParsePrizeDistRatios(ratiosJSON)
	if err != nil {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.invalid_ratios"))
		ctx.Redirect(...)
		return
	}
	if err := hackforger_model.ValidatePrizeDistRatios(ratios); err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect(...)
		return
	}
	track.PrizeDistRatios = ratiosJSON
} else {
	track.PrizeDistRatios = ""
}
```

- [ ] **Step 3: Commit**

```bash
git add templates/hackforger/hackathon/manage.tmpl routers/web/hackforger/hackathon.go
git commit -m "feat(credits): hackathon track prize distribution mode UI"
```

---

## Chunk 5: i18n + E2E

### Task 19: Complete i18n — en-US + zh-CN

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 1: Add new English keys**

Append to the `[hackforger]` section of `locale_en-US.ini`:

```ini
; Prize distribution
hackforger.hackathon.prize_dist_mode = Prize Distribution Mode
hackforger.hackathon.dist_winner_takes_all = Winner Takes All
hackforger.hackathon.dist_tiered = Tiered (Custom Ratios)
hackforger.hackathon.dist_equal = Equal Split
hackforger.hackathon.dist_ratios = Distribution Ratios (JSON)
hackforger.hackathon.dist_ratios_help = JSON array of {rank, pct} objects. Sum of pct must equal 100.
hackforger.hackathon.invalid_ratios = Invalid distribution ratios

; Delivery
hackforger.credits.delivery = Delivery
hackforger.credits.delivery.license_key = License Key
hackforger.credits.delivery.download_link = Download Link
hackforger.credits.delivery.tracking = Tracking Number
hackforger.credits.delivery.custom = Custom

; Key pool admin
hackforger.credits.admin.key_pool = Key Pool
hackforger.credits.admin.keys_status = %d total / %d available
hackforger.credits.admin.add_keys = Add Keys
hackforger.credits.admin.keys_textarea = One key per line
hackforger.credits.admin.add_keys_button = Add Keys
hackforger.credits.admin.existing_keys = Existing Keys
hackforger.credits.admin.key_value = Key Value
hackforger.credits.admin.key_status = Status
hackforger.credits.admin.key_order_id = Order ID
hackforger.credits.admin.key_used = Used
hackforger.credits.admin.key_available = Available
hackforger.credits.admin.no_keys = No keys in pool
hackforger.credits.admin.keys_empty = Please enter at least one key
hackforger.credits.admin.keys_added = Added %d keys to pool
hackforger.credits.admin.manage_keys = Manage Keys
hackforger.credits.admin.fulfill_mode = Fulfill Mode
hackforger.credits.admin.fulfill_manual = Manual
hackforger.credits.admin.fulfill_auto = Auto (Key Pool)

; Batch fulfill
hackforger.credits.admin.batch_fulfill = Batch Fulfill
hackforger.credits.admin.batch_success = Successfully fulfilled %d orders
hackforger.credits.admin.batch_partial = Fulfilled %d orders, %d failed
hackforger.credits.admin.no_orders_selected = No orders selected
hackforger.credits.admin.delivery_type = Delivery Type
hackforger.credits.admin.delivery_value = Delivery Value

; Order notifications
hackforger.credits.order_fulfilled = Your order for "%s" has been fulfilled
hackforger.credits.order_cancelled = Your order for "%s" has been cancelled
```

- [ ] **Step 2: Add corresponding Chinese translations**

Append to the `[hackforger]` section of `locale_zh-CN.ini`:

```ini
; 奖品分配
hackforger.hackathon.prize_dist_mode = 奖品分配方式
hackforger.hackathon.dist_winner_takes_all = 冠军独占
hackforger.hackathon.dist_tiered = 分级分配（自定义比例）
hackforger.hackathon.dist_equal = 平均分配
hackforger.hackathon.dist_ratios = 分配比例 (JSON)
hackforger.hackathon.dist_ratios_help = JSON 数组，格式 {rank, pct}，pct 之和必须等于 100。
hackforger.hackathon.invalid_ratios = 无效的分配比例

; 交付信息
hackforger.credits.delivery = 交付信息
hackforger.credits.delivery.license_key = 许可证密钥
hackforger.credits.delivery.download_link = 下载链接
hackforger.credits.delivery.tracking = 物流单号
hackforger.credits.delivery.custom = 自定义

; 密钥池管理
hackforger.credits.admin.key_pool = 密钥池
hackforger.credits.admin.keys_status = 共 %d 个 / %d 个可用
hackforger.credits.admin.add_keys = 添加密钥
hackforger.credits.admin.keys_textarea = 每行一个密钥
hackforger.credits.admin.add_keys_button = 添加密钥
hackforger.credits.admin.existing_keys = 已有密钥
hackforger.credits.admin.key_value = 密钥值
hackforger.credits.admin.key_status = 状态
hackforger.credits.admin.key_order_id = 订单 ID
hackforger.credits.admin.key_used = 已使用
hackforger.credits.admin.key_available = 可用
hackforger.credits.admin.no_keys = 密钥池为空
hackforger.credits.admin.keys_empty = 请输入至少一个密钥
hackforger.credits.admin.keys_added = 已添加 %d 个密钥到密钥池
hackforger.credits.admin.manage_keys = 管理密钥
hackforger.credits.admin.fulfill_mode = 交付方式
hackforger.credits.admin.fulfill_manual = 手动交付
hackforger.credits.admin.fulfill_auto = 自动交付（密钥池）

; 批量交付
hackforger.credits.admin.batch_fulfill = 批量交付
hackforger.credits.admin.batch_success = 成功交付 %d 个订单
hackforger.credits.admin.batch_partial = 已交付 %d 个订单，%d 个失败
hackforger.credits.admin.no_orders_selected = 未选择订单
hackforger.credits.admin.delivery_type = 交付类型
hackforger.credits.admin.delivery_value = 交付内容

; 订单通知
hackforger.credits.order_fulfilled = 您的「%s」订单已交付
hackforger.credits.order_cancelled = 您的「%s」订单已取消
```

- [ ] **Step 3: Also check and fill any missing existing zh-CN keys**

Compare the existing en-US credits keys against zh-CN. Fill in any missing translations for the keys identified during exploration (overview, redeem confirmation, admin pages, etc.).

- [ ] **Step 4: Commit**

```bash
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "i18n: complete credits en-US + zh-CN translations"
```

---

### Task 20: E2E Test Prompt + Report Template

**Files:**
- Create: `docs/tests/e2e/credits-integration-fulfill-e2e.md`

- [ ] **Step 1: Write the E2E test prompt**

```markdown
# E2E Test: Credits Integration + Fulfill Enhancement

**Instance:** https://hackforger.inside.h2os.cloud
**Login:** hackforger / admin1234
**Date:** ___________

## Pre-requisites
- Server rebuilt and running with latest code
- Database migrated (v14g applied)

## Test Scenarios

### 1. Hackathon Credits Distribution
- [ ] Create a hackathon with 3 tracks:
  - Track A: prize_credits=1000, mode=winner_takes_all
  - Track B: prize_credits=1000, mode=tiered, ratios=[{1:50},{2:30},{3:20}]
  - Track C: prize_credits=100, mode=equal
- [ ] Register 3 users, submit to each track, score submissions
- [ ] Finalize hackathon
- [ ] Check each user's /credits page:
  - Track A: rank 1 user has +1000
  - Track B: rank 1 has +500, rank 2 has +300, rank 3 has +200
  - Track C: each user has equal share
- [ ] Verify transaction references show correct hackathon/track/rank format

### 2. Manual Fulfill with Delivery (F2)
- [ ] Create a manual redeem option (cost=50)
- [ ] As user, redeem the option -> order shows "pending"
- [ ] As admin, go to /-/admin/credits/orders, find the order
- [ ] Click Fulfill, enter delivery_type="license_key", delivery_value="TEST-KEY-123"
- [ ] As user, check /credits/orders -> order shows "fulfilled" with key visible

### 3. Auto-Fulfill Key Pool (F4)
- [ ] As admin, create redeem option with fulfill_mode="auto", cost=50
- [ ] Go to /-/admin/credits/options/{id}/keys
- [ ] Add 3 keys via textarea (one per line)
- [ ] Verify pool shows "3 total / 3 available"
- [ ] As user, redeem the option -> order immediately "fulfilled" with first key
- [ ] As admin, pool shows "3 total / 2 available"
- [ ] Redeem again -> second key assigned
- [ ] Redeem until keys exhausted -> option auto-deactivated, error on next attempt

### 4. Batch Fulfill (F3)
- [ ] Create manual option, have 3 users redeem -> 3 pending orders
- [ ] As admin, go to /-/admin/credits/orders
- [ ] Check 2 orders, click "Batch Fulfill" with note="March batch"
- [ ] Verify both orders show "fulfilled"
- [ ] Third order still "pending"

### 5. Order Notification (F1)
- [ ] After fulfilling an order, check user's feed
- [ ] "order_fulfilled" event should appear targeting the order owner
- [ ] Cancel an order -> "order_cancelled" event in user's feed

### 6. i18n (zh-CN)
- [ ] Switch language to 简体中文
- [ ] Visit /credits -> all labels in Chinese
- [ ] Visit /-/admin/credits/options -> Chinese labels
- [ ] Visit /-/admin/credits/orders -> Chinese labels
- [ ] Visit /-/admin/credits/options/{id}/keys -> Chinese labels

## Results

| # | Scenario | Pass/Fail | Notes |
|---|----------|-----------|-------|
| 1 | Hackathon Credits | | |
| 2 | Manual Fulfill | | |
| 3 | Auto-Fulfill | | |
| 4 | Batch Fulfill | | |
| 5 | Order Notification | | |
| 6 | i18n zh-CN | | |

**Overall:** __ / 6 passed
**Tester:** ___________
```

- [ ] **Step 2: Commit**

```bash
git add docs/tests/e2e/credits-integration-fulfill-e2e.md
git commit -m "docs: add E2E test prompt for credits integration + fulfill"
```

---

## Build Verification

### Task 21: Full Build + Test Sweep

- [ ] **Step 1: Build backend**

Run: `cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/phase2-credits && TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: compiles with no errors

- [ ] **Step 2: Run all hackforger tests**

Run: `go test ./models/hackforger/... -v && go test ./services/hackforger/... -v`
Expected: all PASS

- [ ] **Step 3: Build frontend (if templates changed)**

Run: `make frontend`
Expected: compiles with no errors

- [ ] **Step 4: Final commit if any fixups needed**

```bash
git add -A && git commit -m "fix: build and test fixups for credits integration"
```
