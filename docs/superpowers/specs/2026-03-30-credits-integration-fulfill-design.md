# Week 4 Line-C: Credits Integration + Fulfill Enhancement

**Date:** 2026-03-30
**Status:** Draft
**Scope:** Hackathon-Credits linkage, order fulfillment flow (F1-F4), i18n completion

---

## 1. Overview

Week 3 delivered the Credits core system: account/transaction/redeem models, Deposit/Redeem service functions, web+API routes, admin management, and integration with Bounty and Grant modules. Week 4 Line-C completes the remaining work:

1. **Hackathon -> Credits integration**: Award prize credits to winners when a hackathon is finalized
2. **Fulfill flow enhancements**: Notification (F1), delivery details (F2), batch fulfill (F3), auto-fulfill key pool (F4)
3. **i18n completion**: All missing zh-CN translations for credits UI

### What Already Exists

| Component | Status |
|-----------|--------|
| Credits models (4 tables: CreditAccount, CreditTransaction, RedeemOption, RedeemOrder) | Done |
| Deposit / Redeem / AdminDeposit / AdminDeduct / FulfillOrder / CancelOrder | Done |
| Web routes + templates (user overview, redeem, orders; admin credits, options, orders) | Done |
| API routes (balance, transactions, options, orders, admin ops) | Done |
| Bounty -> Credits Deposit (on claim + winner selection) | Done |
| Grant -> Credits Deposit (on project distribute) | Done |
| Feed event for CreditsRedeemed (ActionType 42) | Done |
| Unit tests (admin deposit/deduct, fulfill, cancel+refund) | Done |

### What This Phase Adds

| Component | Description |
|-----------|-------------|
| Hackathon -> Credits | 3 distribution modes per track, auto-deposit on finalize |
| F1: Notification | Feed events for order fulfilled/cancelled |
| F2: Delivery details | Structured delivery info on orders |
| F3: Batch fulfill | Multi-order fulfillment in one action |
| F4: Auto-fulfill | Key pool table, auto-claim on redeem |
| i18n | Complete zh-CN translations for all credits UI |

---

## 2. Hackathon -> Credits Integration

### 2.1 Data Model Changes

**`HackathonTrack`** — add two columns:

| Column | Type | Default | Description |
|--------|------|---------|-------------|
| `prize_dist_mode` | VARCHAR(20) NOT NULL | `"winner_takes_all"` | Distribution mode |
| `prize_dist_ratios` | TEXT | `""` | JSON array of `{rank, pct}` objects |

**Distribution modes:**

| Mode | Behavior |
|------|----------|
| `winner_takes_all` | Rank 1 receives 100% of `PrizeCredits` |
| `tiered` | Each rank in `PrizeDistRatios` receives `PrizeCredits * pct / 100` |
| `equal` | All ranked submissions receive `PrizeCredits / N` (remainder to rank 1) |

**`PrizeDistRatios` JSON format** (only used when mode = `tiered`):

```json
[
  {"rank": 1, "pct": 50},
  {"rank": 2, "pct": 30},
  {"rank": 3, "pct": 20}
]
```

Validation rules:
- All `pct` values > 0
- `sum(pct)` must equal 100
- Ranks must be contiguous: 1, 2, ..., N
- N >= 1

### 2.2 Service Logic

Add `distributeHackathonCredits(ctx, hackathonID)` called from `FinalizeHackathon()` after `CalculateRanks()` succeeds.

**Important: per-track ranking.** The existing `CalculateRanks()` computes a single global ranking across all submissions in a hackathon. For prize distribution, we need per-track rankings. Rather than modifying `CalculateRanks()` (which is used for leaderboard display), `distributeHackathonCredits()` computes its own per-track ranking by sorting submissions within each track by `TotalScore` descending. This is independent of the global `Rank` field.

**ListSubmissions change needed:** The existing `ListSubmissionsOptions` has no `OrderBy` support. Add a `SortByScore bool` field; when true, the `ToOrders()` method returns `"total_score DESC, id ASC"`. Alternatively, the service layer can sort the returned slice in-memory (simpler, adequate for hackathon-sized datasets).

**Pseudocode:**

```
func distributeHackathonCredits(ctx, hackathonID):
  tracks = ListTracksByHackathon(ctx, hackathonID)
  for each track where PrizeCredits > 0:
    submissions = ListSubmissions(ctx, {TrackID: track.ID, ListAll: true})
    // Sort by TotalScore descending (per-track ranking, independent of global Rank)
    sort submissions by TotalScore DESC, ID ASC
    // Assign per-track rank: 1, 2, 3, ...
    ranked = filter(submissions, sub.TotalScore > 0)  // exclude unscored
    if len(ranked) == 0: continue

    switch track.PrizeDistMode:
      case "winner_takes_all":
        Deposit(ctx, ranked[0].UserID, track.PrizeCredits,
          "hackathon:{id}/track:{tid}:rank:1", "Hackathon winner: {track.Name}")

      case "tiered":
        ratios = parseJSON(track.PrizeDistRatios)
        deposited = 0
        for i, r := range ratios:
          trackRankIdx = r.Rank - 1  // 0-indexed
          if trackRankIdx >= len(ranked): continue  // fewer submissions than ratios, skip
          sub = ranked[trackRankIdx]
          amount = track.PrizeCredits * r.Pct / 100
          deposited += amount
          Deposit(ctx, sub.UserID, amount,
            "hackathon:{id}/track:{tid}:rank:{r.Rank}", "Hackathon rank {r.Rank}: {track.Name}")
        // Rounding remainder: deposited may be < PrizeCredits due to integer division.
        // Award remainder to rank 1 (highest ranked, most deserving).
        if remainder := track.PrizeCredits - deposited; remainder > 0 && len(ranked) > 0:
          Deposit(ctx, ranked[0].UserID, remainder,
            "hackathon:{id}/track:{tid}:rank:1:remainder", "Hackathon rounding remainder: {track.Name}")

      case "equal":
        perUser = track.PrizeCredits / int64(len(ranked))
        remainder = track.PrizeCredits % int64(len(ranked))
        for i, sub := range ranked:
          amount = perUser
          if i == 0: amount += remainder  // rank 1 gets remainder
          Deposit(ctx, sub.UserID, amount,
            "hackathon:{id}/track:{tid}:rank:{i+1}", "Hackathon participant: {track.Name}")
```

**Edge cases:**
- Track has `PrizeCredits = 0`: skip entirely
- No scored submissions in track (`TotalScore == 0`): skip, no deposits
- Tiered mode with fewer submissions than ratio entries: skip missing ranks, unclaimed portion is NOT redistributed (organizer set the rules; see example below)
- Integer division remainder: always assigned to rank 1 (highest-ranked) in both tiered and equal modes for consistency

**Atomicity:** Each track's deposits are independent transactions (the existing `Deposit()` wraps each call in `db.WithTx`). If track 1 deposits succeed but track 2 fails, partial credits remain awarded. This is acceptable — the failure window is small, and wrapping all deposits in a single mega-transaction would be complex. On failure, the error is logged and returned so the admin can investigate.

**Tiered with fewer submissions example:**
Track has 1000 credits, ratios `[{1: 50%}, {2: 30%}, {3: 20%}]`, but only 2 submissions:
- Rank 1 gets `1000 * 50 / 100 = 500`
- Rank 2 gets `1000 * 30 / 100 = 300`
- Rank 3 skipped (no submission)
- `deposited = 800`, remainder `200` goes to rank 1
- Final: rank 1 = 700, rank 2 = 300, total = 1000 (no credits lost)

### 2.3 Track Create/Update Validation

In web handler and API handler for track creation/update:
- If `PrizeDistMode` is not one of `{"winner_takes_all", "tiered", "equal"}`, reject
- If `PrizeDistMode == "tiered"`:
  - Parse `PrizeDistRatios` as JSON `[]struct{Rank int; Pct int}`
  - Verify all `Pct > 0`, `sum(Pct) == 100`, ranks are `1..N` contiguous
  - Return typed error `ErrInvalidDistRatios` on failure
- If `PrizeDistMode != "tiered"`, `PrizeDistRatios` is ignored (cleared on save)

### 2.4 Transaction Reference Format

```
hackathon:{hackathonID}/track:{trackID}:rank:{rank}
```

Example: `hackathon:5/track:12:rank:1`

---

## 3. Fulfill Flow Enhancements

### 3.1 F2: Delivery Details

**`RedeemOrder`** — add two columns:

| Column | Type | Default | Description |
|--------|------|---------|-------------|
| `delivery_type` | VARCHAR(32) | `""` | Type of delivery info |
| `delivery_value` | TEXT | `""` | The actual content |

**Delivery types:**

| Type | Use case | Example value |
|------|----------|---------------|
| `license_key` | Software license | `XXXX-XXXX-XXXX-XXXX` |
| `download_link` | Digital download | `https://...` |
| `tracking` | Physical shipment | `SF1234567890` |
| `custom` | Anything else | Free-form text |

**UI changes:**
- Admin fulfill modal: add delivery type selector + value textarea
- User orders page: show delivery info for fulfilled orders (type label + value, with copy button for keys/links)

### 3.2 F4: Auto-fulfill Key Pool

**New table `RedeemOptionKey`:**

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGINT PK autoincr | |
| `option_id` | BIGINT INDEX NOT NULL | FK to RedeemOption |
| `key_value` | TEXT NOT NULL | The actual key/code |
| `is_used` | BOOL NOT NULL DEFAULT false | Claimed flag |
| `order_id` | BIGINT INDEX | Set when claimed, FK to RedeemOrder |
| `created_unix` | BIGINT created | |

**`RedeemOption`** — add one column:

| Column | Type | Default | Description |
|--------|------|---------|-------------|
| `fulfill_mode` | VARCHAR(16) NOT NULL | `"manual"` | `"manual"` or `"auto"` |

**Updated `FulfillOrder` signature** (F2 adds delivery fields):

```go
func FulfillOrder(ctx context.Context, admin *user_model.User, orderID int64, note, deliveryType, deliveryValue string) error
```

**Modified `Redeem()` flow** (all within existing `db.WithTx`):

```
existing: deduct balance -> deduct stock -> create transaction -> create order (pending)

after creating order, add:
  if option.FulfillMode == "auto":
    // Two-step claim for SQLite compatibility (UPDATE...ORDER BY...LIMIT
    // requires SQLITE_ENABLE_UPDATE_DELETE_LIMIT which may not be enabled).
    // Concurrency note: SQLite uses database-level write locks within
    // transactions, so this is inherently serialized. For PostgreSQL/MySQL,
    // use XORM's ForUpdate() on the SELECT session to acquire a row-level
    // lock and prevent double-claiming under concurrent redeems.
    key = SELECT id, key_value FROM redeem_option_key
          WHERE option_id = option.ID AND is_used = false
          ORDER BY id ASC LIMIT 1
          [ForUpdate on PostgreSQL/MySQL]
    if no key found:
      return ErrOutOfStock  // triggers full tx rollback
    UPDATE redeem_option_key SET is_used = true, order_id = order.ID WHERE id = key.ID
    order.Status = "fulfilled"
    order.DeliveryType = "license_key"
    order.DeliveryValue = key.KeyValue
    update order
```

**Stock management for auto-fulfill options:**
- When `FulfillMode = "auto"`, the `Stock` field is derived: `Stock = count of unused keys`
- Admin adds keys -> Stock increases automatically
- Keys claimed -> Stock decreases automatically
- The `Stock` column on `RedeemOption` is kept in sync by the service layer on key add/remove operations
- If all keys are used and Stock reaches 0, set `IsActive = false` automatically
- Validation: when `FulfillMode = "auto"`, `Stock = -1` (unlimited) is NOT allowed — stock is always derived from key count. The admin cannot manually set Stock for auto-fulfill options.
- Note: existing `RedeemOption.Stock` field is type `int` (not `int64`). This is fine — key pool sizes won't exceed int range.

**Admin UI for key pool:**
- Option create/edit page: `FulfillMode` radio (Manual / Auto-fulfill)
- If Auto: show key management section:
  - Textarea to bulk-add keys (one per line)
  - Table of existing keys (value, status, linked order ID)
  - Pool status badge: "45 available / 55 used / 100 total"
- API endpoints:
  - `POST /api/v1/credits/redeem/options/{id}/keys` — bulk add keys (body: `{"keys": ["k1","k2",...]}`)
  - `GET /api/v1/credits/redeem/options/{id}/keys` — list keys with status (admin only)
  - Web equivalents under `/-/admin/credits/options/{id}/keys`

### 3.3 F3: Batch Fulfill

**Service function:**

```go
func BatchFulfillOrders(ctx, admin, orderIDs []int64, note, deliveryType, deliveryValue string) (successCount int, failedIDs []int64, err error)
```

- Iterates over `orderIDs`, calls `FulfillOrder()` for each
- Each fulfillment is an independent transaction (not one big tx) so partial success is possible
- Returns count of successes and list of failed IDs with reasons
- Only applies to manual-mode options (auto-fulfill orders are already fulfilled)

**API:** `POST /api/v1/credits/redeem/orders/batch-fulfill`

```json
{
  "order_ids": [1, 2, 3],
  "note": "Batch fulfillment for March",
  "delivery_type": "license_key",
  "delivery_value": ""
}
```

Note: if `delivery_value` is empty, each order gets only the note. For batch scenarios where each order needs a different key, admin should use individual fulfill or auto-fulfill mode instead.

**Web UI:**
- Admin orders page: checkbox per pending order row
- "Batch Fulfill" button appears when >= 1 checked
- Modal: note + delivery type/value fields
- Result: flash message showing success count and any failures

### 3.4 F1: Order Status Notification

**New action types** (in lifecycle range 58-59, since 30-43 and 50-57 are taken):

```go
ActionOrderFulfilled activities_model.ActionType = 58
ActionOrderCancelled activities_model.ActionType = 59
```

Add corresponding string mappings in `action_types.go`:

```go
ActionOrderFulfilled: "order_fulfilled",
ActionOrderCancelled: "order_cancelled",
```

**New audience type — `AudienceDirectUser`:**

The existing audience types (Global, Followers, OrgMembers, RepoWatchers) don't support targeting a specific user. Order notifications need to reach the order owner directly. Add:

```go
AudienceDirectUser  // Visible to a specific user (opts.TargetUserID)
```

Add `TargetUserID int64` to `HackforgerActionOpts`, used when `AudienceType == AudienceDirectUser`.

In `PublishHackforgerAction`, the `AudienceDirectUser` case inserts a single action row with `UserID = opts.TargetUserID`.

**Implementation — standalone helper in `credits.go`:**

```go
func notifyOrderStatusChange(ctx context.Context, order *RedeemOrder, adminID int64, actionType ActionType, optionName string)
```

This calls `PublishHackforgerAction` with:
- `ActUserID = adminID`
- `OpType = actionType`
- `AudienceType = AudienceDirectUser`
- `TargetUserID = order.UserID` (the order owner — the person who should see the notification)
- `Content.EntityType = "credits"`, `Content.EntityName = optionName`
- `Content.Extra = {"order_id": order.ID}`

Called from:
- `FulfillOrder()` — with `ActionOrderFulfilled`
- `CancelOrder()` — with `ActionOrderCancelled`
- `BatchFulfillOrders()` — with `ActionOrderFulfilled` for each success

**Isolation strategy:**
- All notification code stays in `services/hackforger/credits.go`
- The `AudienceDirectUser` addition to `notifier.go` is minimal (~5 lines in the switch)
- Committed as a separate, independent commit
- When Line-B later reworks the feed system, only this isolated commit may need rebasing

---

## 4. Schema Migration

One new migration file: `models/forgejo_migrations/v14g_credits-fulfill-hackathon-prizes.go`. Covers all schema changes:

**Alter `hackathon_track`:**
- ADD `prize_dist_mode` VARCHAR(20) NOT NULL DEFAULT 'winner_takes_all'
- ADD `prize_dist_ratios` TEXT NOT NULL DEFAULT ''

**Alter `redeem_option`:**
- ADD `fulfill_mode` VARCHAR(16) NOT NULL DEFAULT 'manual'

**Alter `redeem_order`:**
- ADD `delivery_type` VARCHAR(32) NOT NULL DEFAULT ''
- ADD `delivery_value` TEXT NOT NULL DEFAULT ''

**Create `redeem_option_key`:**
- Standard table creation with indexes on `option_id` and `order_id`

---

## 5. i18n Completion

All credits-related UI labels need zh-CN translations. The English keys already exist in `locale_en-US.ini` under `[hackforger]`. The zh-CN file currently has only 2 credits keys.

**Scope:** Add all missing zh-CN keys matching the existing en-US keys, plus new keys for:
- Prize distribution mode labels (winner_takes_all, tiered, equal)
- Delivery type labels
- Key pool management labels
- Batch fulfill UI labels
- Order notification messages

All keys go under the `[hackforger]` section in both locale files.

---

## 6. New/Modified Files Summary

### New Files

| File | Description |
|------|-------------|
| `models/hackforger/redeem_option_key.go` | RedeemOptionKey table + CRUD |
| `templates/hackforger/credits/admin/keys.tmpl` | Admin key pool management page |
| Migration file in `models/forgejo_migrations/` | Schema changes |

### Modified Files

| File | Changes |
|------|---------|
| `models/hackforger/hackathon_track.go` | Add PrizeDistMode, PrizeDistRatios fields |
| `models/hackforger/redeem_option.go` | Add FulfillMode field |
| `models/hackforger/redeem_order.go` | Add DeliveryType, DeliveryValue fields |
| `models/hackforger/action_types.go` | Add ActionOrderFulfilled (58), ActionOrderCancelled (59) |
| `models/hackforger/init.go` | Register RedeemOptionKey model |
| `services/hackforger/hackathon.go` | Add distributeHackathonCredits(), call from FinalizeHackathon() |
| `services/hackforger/credits.go` | Modify Redeem() for auto-fulfill, update FulfillOrder() signature, add BatchFulfillOrders(), notifyOrderStatusChange(), key pool management funcs |
| `services/hackforger/notifier.go` | Add AudienceDirectUser type + TargetUserID field (~5 lines, separate commit) |
| `routers/web/hackforger/credits.go` | Add key pool web handlers, batch fulfill handler |
| `routers/api/v1/hackforger/credits.go` | Add key pool API endpoints, batch fulfill API |
| `routers/web/web.go` | Register new key pool + batch fulfill routes |
| `routers/api/v1/api.go` | Register new API routes |
| `templates/hackforger/credits/overview.tmpl` | Minor: delivery info display |
| `templates/hackforger/credits/orders.tmpl` | Show delivery info for fulfilled orders |
| `templates/hackforger/credits/admin/orders.tmpl` | Batch fulfill UI, delivery fields in fulfill modal |
| `templates/hackforger/credits/admin/options.tmpl` | FulfillMode selector, key pool link |
| `templates/hackforger/hackathon/manage.tmpl` | Prize distribution mode + ratios UI in track section |
| `options/locale/locale_en-US.ini` | New i18n keys |
| `options/locale/locale_zh-CN.ini` | Complete credits zh-CN translations |

### Test Files

| File | Changes |
|------|---------|
| `services/hackforger/credits_test.go` | Tests for auto-fulfill, batch fulfill, key pool, notification |
| `services/hackforger/hackathon_test.go` or `hackathon_judge_test.go` | Test distributeHackathonCredits() for all 3 modes |
| `models/hackforger/credits_test.go` | Tests for RedeemOptionKey CRUD |
| Test fixtures YAML | Add test data for key pool, tracks with prize modes |

---

## 7. Out of Scope

- Line-B feed system rework (separate track)
- Reputation score updates from credits (separate track)
- Credits transfer between users (explicitly out of scope per plan)
- Webhook events for credits (Week 5 scope)
- Swagger annotations (Week 5 scope)

---

## 8. E2E Testing

A manual E2E test prompt and report template will be created in `docs/tests/e2e/` as part of the implementation plan. The E2E scenarios must cover:

1. **Hackathon credits distribution**: Create hackathon with 3 tracks (one per mode), finalize, verify deposits
2. **Manual fulfill with delivery**: Redeem option, admin fulfills with license key, user sees key on orders page
3. **Auto-fulfill**: Admin creates option with key pool, user redeems, gets key immediately
4. **Batch fulfill**: Multiple pending orders, admin batch-fulfills, verify all statused
5. **Order notification**: After fulfill/cancel, verify feed event visible to order owner
6. **i18n**: Switch to zh-CN, verify all credits pages render correctly
