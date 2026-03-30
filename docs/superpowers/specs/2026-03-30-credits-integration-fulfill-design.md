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

**Pseudocode:**

```
func distributeHackathonCredits(ctx, hackathonID):
  tracks = ListTracksByHackathon(ctx, hackathonID)
  for each track where PrizeCredits > 0:
    submissions = ListSubmissions(ctx, {TrackID: track.ID, OrderBy: "rank ASC"})
    ranked = filter(submissions, sub.Rank > 0)
    if len(ranked) == 0: continue

    switch track.PrizeDistMode:
      case "winner_takes_all":
        Deposit(ctx, ranked[0].UserID, track.PrizeCredits,
          "hackathon:{id}/track:{tid}:rank:1", "Hackathon winner: {track.Name}")

      case "tiered":
        ratios = parseJSON(track.PrizeDistRatios)
        deposited = 0
        for i, r := range ratios:
          sub = findByRank(ranked, r.Rank)
          if sub == nil: continue  // fewer submissions than ratios, skip
          if i == len(ratios)-1 && sub != nil:
            amount = track.PrizeCredits - deposited  // last rank gets remainder
          else:
            amount = track.PrizeCredits * r.Pct / 100
          deposited += amount
          Deposit(ctx, sub.UserID, amount,
            "hackathon:{id}/track:{tid}:rank:{r.Rank}", "Hackathon rank {r.Rank}: {track.Name}")

      case "equal":
        perUser = track.PrizeCredits / int64(len(ranked))
        remainder = track.PrizeCredits % int64(len(ranked))
        for i, sub := range ranked:
          amount = perUser
          if i == 0: amount += remainder  // rank 1 gets remainder
          Deposit(ctx, sub.UserID, amount,
            "hackathon:{id}/track:{tid}:rank:{sub.Rank}", "Hackathon participant: {track.Name}")
```

**Edge cases:**
- Track has `PrizeCredits = 0`: skip entirely
- No ranked submissions in track: skip, no deposits
- Tiered mode with fewer submissions than ratio entries: skip missing ranks, unclaimed portion is NOT redistributed (organizer set the rules)
- Integer division remainder: always assigned to the highest-ranked recipient (rank 1 for equal, last ratio entry for tiered)

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

**Modified `Redeem()` flow** (all within existing `db.WithTx`):

```
existing: deduct balance -> deduct stock -> create transaction -> create order (pending)

after creating order, add:
  if option.FulfillMode == "auto":
    key = claim one unused key:
      UPDATE redeem_option_key
      SET is_used = true, order_id = order.ID
      WHERE option_id = option.ID AND is_used = false
      ORDER BY id ASC LIMIT 1
    if no key found:
      return ErrOutOfStock  // triggers full tx rollback
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

**New action types** (extending the existing HackForger range):

```go
ActionOrderFulfilled activities_model.ActionType = 43
ActionOrderCancelled activities_model.ActionType = 44
```

Add corresponding string mappings in `action_types.go`:

```go
ActionOrderFulfilled: "order_fulfilled",
ActionOrderCancelled: "order_cancelled",
```

**Implementation — standalone helper in `credits.go`:**

```go
func notifyOrderStatusChange(ctx context.Context, order *RedeemOrder, adminID int64, actionType ActionType, optionName string)
```

This calls `PublishHackforgerAction` with:
- `ActUserID = adminID`
- `OpType = actionType`
- `AudienceType = AudienceFollowers` (of the admin — minimal audience)
- `Content.EntityType = "credits"`, `Content.EntityName = optionName`
- `Content.Extra = {"order_id": order.ID, "user_id": order.UserID}`

Called from:
- `FulfillOrder()` — with `ActionOrderFulfilled`
- `CancelOrder()` — with `ActionOrderCancelled`
- `BatchFulfillOrders()` — with `ActionOrderFulfilled` for each success

**Isolation strategy:**
- All notification code stays in `services/hackforger/credits.go`
- No changes to `feed.go`, `notifier.go`, or feed query logic
- Committed as a separate, independent commit
- When Line-B later reworks the feed system, only this isolated commit may need rebasing

---

## 4. Schema Migration

One new migration file in `models/forgejo_migrations/` covering all schema changes:

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
| `models/hackforger/action_types.go` | Add ActionOrderFulfilled (43), ActionOrderCancelled (44) |
| `models/hackforger/init.go` | Register RedeemOptionKey model |
| `services/hackforger/hackathon.go` | Add distributeHackathonCredits(), call from FinalizeHackathon() |
| `services/hackforger/credits.go` | Modify Redeem() for auto-fulfill, add BatchFulfillOrders(), notifyOrderStatusChange(), key pool management funcs |
| `routers/web/hackforger/credits.go` | Add key pool web handlers, batch fulfill handler |
| `routers/api/v1/hackforger/credits.go` | Add key pool API endpoints, batch fulfill API |
| `routers/web/web.go` | Register new key pool + batch fulfill routes |
| `routers/api/v1/api.go` | Register new API routes |
| `templates/hackforger/credits/overview.tmpl` | Minor: delivery info display |
| `templates/hackforger/credits/orders.tmpl` | Show delivery info for fulfilled orders |
| `templates/hackforger/credits/admin/orders.tmpl` | Batch fulfill UI, delivery fields in fulfill modal |
| `templates/hackforger/credits/admin/options.tmpl` | FulfillMode selector, key pool link |
| `templates/hackforger/hackathon/manage_tracks.tmpl` | Prize distribution mode + ratios UI |
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
