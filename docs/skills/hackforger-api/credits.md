# Credits API

Base path: `/api/v1/hackforger/credits`. All amounts are integers (credits).

## User-facing

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/credits/balance` | Current balance for the authenticated user |
| GET  | `/hackforger/credits/transactions` | Ledger (filter by type: deposit, bounty_pay, grant_award, hackathon_award, redeem, escrow_lock, escrow_release, manual) |

## Redeem catalog (browse + buy)

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/credits/redeem/options` | List active redeem options (with stock) |
| POST | `/hackforger/credits/redeem` | Redeem an option (creates pending order, deducts credits) |
| GET  | `/hackforger/credits/redeem/orders` | User's order history (with assigned key once fulfilled) |

## Admin: redeem option management

Requires `reqSiteAdmin()`.

| Verb | Path |
|------|------|
| POST | `/hackforger/credits/redeem/options` |
| PUT  | `/hackforger/credits/redeem/options/{id}` |
| GET  | `/hackforger/credits/redeem/options/{id}/keys` |
| POST | `/hackforger/credits/redeem/options/{id}/keys` |

## Admin: order fulfillment

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/hackforger/credits/redeem/orders/{oid}/fulfill` | Mark single order fulfilled (assigns key from pool) |
| POST | `/hackforger/credits/redeem/orders/fulfill` | Batch fulfill (`{"order_ids":[1,2,3]}`) |
| POST | `/hackforger/credits/redeem/orders/{oid}/cancel` | Refund credits to user |

## Admin: manual ledger adjustments

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/hackforger/credits/admin/deposit` | Manual deposit (`{"user":"alice","amount":100,"reason":"..."}`) |
| POST | `/hackforger/credits/admin/deduct` | Manual deduct |

## Common chains

**Hacker redeems GPU hours:**
1. GET `/hackforger/credits/balance` (verify ≥ 200)
2. GET `/hackforger/credits/redeem/options` (find option_id)
3. POST `/hackforger/credits/redeem` `{"option_id":N}` (deducts 200, creates pending order)
4. (Admin fulfills out-of-band: POST `/orders/{oid}/fulfill`)
5. GET `/hackforger/credits/redeem/orders` (read assigned key from fulfilled order)

**Admin sets up new redeem option + keys:**
1. POST `/options` `{"name":"GPU Hours","price":200,"stock":10}`
2. POST `/options/{id}/keys` `{"keys":["KEY-001","KEY-002",...]}`

## Notes on transactions

The ledger is append-only. Every state change in bounties/grants/hackathons writes a transaction with the appropriate `type` and references the originating entity. Use `GET /transactions?type=bounty_pay` etc. to filter for audit.
