# Phase 1 Grant + Credits Design Spec

## Overview

Line C of the HackForger parallel development plan. Implements Grant Round full-stack (state machine, API, web pages) and Credits full-stack (admin operations, redeem flow, web pages) following a model-first bottom-up approach.

**Branch:** `feat/phase1-grant`
**Parallel with:** Line A (Bounty), Line B (Hackathon)

## Design Decisions

| Decision | Choice |
|----------|--------|
| Scope | Line C: Grant full-stack + Credits full-stack |
| Parallel conflict strategy | Each line writes own Notifier calls; audience resolution deferred post-merge |
| Grant states | 6: Draft -> Open -> Review -> Finalized -> Distributed + Cancelled |
| Credits distribution timing | Per-project at DistributeProject, calls credits.Deposit |
| Round -> Distributed | Auto-detected when all Approved projects are Funded |
| Admin Credits permission | Site admin only (IsAdmin) |
| Frontend approach | SSR-first, Vue only when SSR cannot handle the interaction |
| Budget validation | Strict: both fiat and credits checked, reject on exceed |
| Implementation order | Model-first bottom-up: Model CRUD -> Service -> API -> Web -> Templates -> i18n |

## Section 1: Model Layer

### GrantRound Changes

Current P0 model has 5 statuses (Draft=0, Open=1, Review=2, Complete=3, Cancelled=4). Change to 6:

- **Rename** `GrantRoundStatusComplete` (3) to `GrantRoundStatusFinalized` (3) -- same numeric value, name change only
- **Insert** `GrantRoundStatusDistributed = 4` -- new state
- **Re-number** `GrantRoundStatusCancelled` from 4 to 5

No migration needed: P0 has no production data (tables are auto-synced by XORM, no rows exist yet).

```go
GrantRoundStatusDraft       = 0
GrantRoundStatusOpen        = 1
GrantRoundStatusReview      = 2
GrantRoundStatusFinalized   = 3  // renamed from Complete
GrantRoundStatusDistributed = 4  // new
GrantRoundStatusCancelled   = 5  // re-numbered from 4
```

Note: The test-plan-draft.md references `Status=Setup` for initial state. This is outdated; P0 code uses `Draft`. Implementation follows the P0 code (`Draft`).

New CRUD functions:
- `GetGrantRoundBySlug(ctx, slug)` -- for web routes `/grants/{slug}`
- `DeleteGrantRound(ctx, id)` -- Draft only
- `GetGrantRoundBudgetUsage(ctx, roundID) -> (sumAmount float64, sumCredits int64)` -- for strict budget validation

### GrantProject Changes

New CRUD functions:
- `GetGrantProjectByUserAndRound(ctx, userID, roundID)` -- prevent duplicate submission
- `DeleteGrantProject(ctx, id)` -- Pending only
- `CountGrantProjectsByRound(ctx, roundID, status)` -- for Finalize pre-check

New error types:
- `ErrGrantProjectAlreadyExists` -- duplicate submission in same round
- `ErrGrantRoundNotOpen` -- submission when round is not Open
- `ErrExceedsBudget` -- allocation exceeds round budget
- `ErrUnallocatedProjects` -- Finalize with unallocated Approved projects

### Credits Model — First CRUD Functions

P0 only defined structs + `init()` for CreditAccount, CreditTransaction, RedeemOption, RedeemOrder. No CRUD functions exist. All functions below are new.

CreditAccount:
- `GetCreditAccount(ctx, userID)` -- read-only, no auto-create
- `ListCreditAccounts(ctx, opts)` -- Admin view

CreditTransaction:
- `ListAllTransactions(ctx, opts)` -- Admin global view

RedeemOption:
- `CreateRedeemOption(ctx, option)`
- `UpdateRedeemOption(ctx, option)`
- `GetRedeemOptionByID(ctx, id)`

RedeemOrder:
- `GetRedeemOrderByID(ctx, id)`
- `UpdateRedeemOrder(ctx, order)` -- for fulfill/cancel

## Section 2: Service Layer

### Grant Service (`services/hackforger/grants.go` -- new file)

State machine transitions:
```
Draft      -> Open, Cancelled
Open       -> Review, Cancelled
Review     -> Finalized, Cancelled
Finalized  -> Distributed (auto), Cancelled
Distributed -> (terminal)
Cancelled  -> (terminal)
```

Functions:

| Function | Validation | Feed Event | Audience |
|----------|-----------|------------|----------|
| `CreateGrantRound(ctx, doer, opts)` | doer is Org Owner/Admin | ActionGrantRoundCreated (39) | Global |
| `UpdateGrantRound(ctx, doer, round)` | Draft or Open only | -- | -- |
| `DeleteGrantRound(ctx, doer, roundID)` | Draft only | -- | -- |
| `OpenRound(ctx, doer, roundID)` | Draft -> Open | ActionGrantRoundOpened (54) | Global |
| `CloseRound(ctx, doer, roundID)` | Open -> Review | ActionGrantRoundClosed (55) | Global |
| `SubmitProject(ctx, doer, roundID, opts)` | Round == Open, no duplicate | ActionGrantProjectSubmitted (40) | Followers |
| `ApproveProject(ctx, doer, projectID)` | Pending -> Approved | -- | -- |
| `RejectProject(ctx, doer, projectID)` | Pending -> Rejected | -- | -- |
| `AllocateAward(ctx, doer, pid, amount, credits)` | Approved, Review state, budget check | -- | -- |
| `FinalizeRound(ctx, doer, roundID)` | All Approved allocated | ActionGrantRoundFinalized (56) | Global |
| `DistributeProject(ctx, doer, projectID)` | Finalized, Approved; db.WithTx: Deposit + Funded | ActionGrantAwarded (41) | Global |
| `CancelRound(ctx, doer, roundID)` | Non-terminal -> Cancelled | ActionGrantRoundCancelled (57, new) | Global |
| `ExportRoundCSV(ctx, roundID)` | -- | -- | -- |

DistributeProject auto-checks: if all Approved projects are Funded, Round -> Distributed.

Permission helper: `checkGrantRoundAccess(ctx, doer, round)` -- doer is OrgID Owner/Admin.

### Credits Service (`services/hackforger/credits.go` -- P0 file exists with Deposit/Redeem/List functions; append new admin functions)

| Function | Validation |
|----------|-----------|
| `AdminDeposit(ctx, admin, userID, amount, ref, note)` | admin.IsAdmin |
| `AdminDeduct(ctx, admin, userID, amount, ref, note)` | admin.IsAdmin; db.WithTx: deduct + Transaction(withdraw) |
| `FulfillOrder(ctx, admin, orderID, note)` | admin.IsAdmin, Pending -> Fulfilled |
| `CancelOrder(ctx, admin, orderID)` | admin.IsAdmin, Pending -> Cancelled; db.WithTx: refund balance + Transaction(refund) |
| `CreateRedeemOption(ctx, admin, opts)` | admin.IsAdmin |
| `UpdateRedeemOption(ctx, admin, optionID, opts)` | admin.IsAdmin |

### New Action Type

Add `ActionGrantRoundCancelled = 57` to `models/hackforger/action_types.go` with string mapping.

## Section 3: API Layer

### Grant API (`routers/api/v1/hackforger/grants.go` -- new file)

```
POST   /api/v1/hackforger/grant-rounds                                  CreateGrantRound
GET    /api/v1/hackforger/grant-rounds                                  ListGrantRounds
GET    /api/v1/hackforger/grant-rounds/{id}                             GetGrantRound
PUT    /api/v1/hackforger/grant-rounds/{id}                             UpdateGrantRound
DELETE /api/v1/hackforger/grant-rounds/{id}                             DeleteGrantRound
POST   /api/v1/hackforger/grant-rounds/{id}/open                       OpenRound
POST   /api/v1/hackforger/grant-rounds/{id}/close                      CloseRound
POST   /api/v1/hackforger/grant-rounds/{id}/finalize                   FinalizeRound
POST   /api/v1/hackforger/grant-rounds/{id}/distribute                 DistributeRound (convenience: iterates all Approved+allocated projects, calls DistributeProject for each)
POST   /api/v1/hackforger/grant-rounds/{id}/cancel                     CancelRound (non-terminal -> Cancelled)
GET    /api/v1/hackforger/grant-rounds/{id}/export                     ExportCSV
POST   /api/v1/hackforger/grant-rounds/{id}/projects                   SubmitProject
GET    /api/v1/hackforger/grant-rounds/{id}/projects                   ListProjects
GET    /api/v1/hackforger/grant-rounds/{id}/projects/{pid}             GetProject
PUT    /api/v1/hackforger/grant-rounds/{id}/projects/{pid}             ApproveOrReject
PUT    /api/v1/hackforger/grant-rounds/{id}/projects/{pid}/award       AllocateAward
POST   /api/v1/hackforger/grant-rounds/{id}/projects/{pid}/distribute  DistributeProject
```

### Credits API (`routers/api/v1/hackforger/credits.go` -- new file)

```
GET    /api/v1/hackforger/credits/balance                    GetBalance
GET    /api/v1/hackforger/credits/transactions               ListTransactions
GET    /api/v1/hackforger/credits/redeem/options              ListRedeemOptions
POST   /api/v1/hackforger/credits/redeem                     Redeem
GET    /api/v1/hackforger/credits/redeem/orders               ListOrders
POST   /api/v1/hackforger/credits/admin/deposit              AdminDeposit
POST   /api/v1/hackforger/credits/admin/deduct               AdminDeduct
POST   /api/v1/hackforger/credits/redeem/options              CreateRedeemOption
PUT    /api/v1/hackforger/credits/redeem/options/{id}        UpdateRedeemOption
POST   /api/v1/hackforger/credits/redeem/orders/{oid}/fulfill FulfillOrder
POST   /api/v1/hackforger/credits/redeem/orders/{oid}/cancel  CancelOrder
```

### Route Registration

In `routers/api/v1/api.go`, append within `/hackforger` group:

```go
m.Group("/grant-rounds", hackforger.GrantRoutes)
m.Group("/credits", hackforger.CreditRoutes)
```

All list endpoints return `X-Total-Count` header. All handlers include Swagger annotations.

**Deliberate additions beyond implementation plan Section 5.2:**
- `DELETE /grant-rounds/{id}` -- not in plan, but needed for Draft round cleanup
- `POST /grant-rounds/{id}/cancel` -- not in plan, but service layer defines CancelRound
- `POST /credits/redeem/options` (CreateRedeemOption) -- plan only has PUT for editing; creation endpoint is needed
- `POST /grant-rounds/{id}/projects/{pid}/distribute` -- plan only has round-level distribute; per-project endpoint matches the per-project distribution design

## Section 4: Web Routes + Templates

### Web Routes

```
/explore/grants                          ExploreGrants (fill existing stub)
/grants/new                              NewGrantRound (GET: form, POST: create)
/grants/{slug}                           GrantRoundDetail
/grants/{slug}/projects                  GrantRoundProjects
/grants/{slug}/submit                    SubmitProject (GET: form, POST: submit)
/grants/{slug}/manage                    ManageRound
/grants/{slug}/manage/projects/{pid}     ManageProject
/grants/{slug}/export                    ExportCSV
/credits                                 CreditsOverview
/credits/redeem/{id}                     RedeemConfirm (GET: confirm, POST: redeem)
/credits/orders                          OrderList
/-/admin/credits                         AdminCredits
/-/admin/credits/options                 AdminRedeemOptions
/-/admin/credits/orders                  AdminOrders
```

### Templates

```
templates/hackforger/
  grants/
    explore.tmpl            -- list page with status filter
    new.tmpl                -- create form
    detail.tmpl             -- status bar + description + stats + submit button
    projects.tmpl           -- public project list with star counts
    submit.tmpl             -- project submission form
    manage.tmpl             -- status actions + project list + approve/reject + budget bar
    manage_project.tmpl     -- single project review + award allocation
  credits/
    overview.tmpl           -- balance card + transactions + redeem options grid
    redeem.tmpl             -- confirm page
    orders.tmpl             -- user order list
    admin/
      credits.tmpl          -- admin credits management
      options.tmpl          -- admin redeem options
      orders.tmpl           -- admin orders + fulfill
```

All template text via `ctx.Locale.Tr()` with prefixes `hackforger.grant.*` and `hackforger.credits.*`.

### Key Page Designs

**Grant Round Detail:** status progress bar (6 states), basic info (name, description, budget, deadline, org), "Submit Project" button when Open, project list at bottom.

**Manage Panel:** state action buttons (contextual), project table with approve/reject + amount input, budget usage progress bar. All SSR forms, no Vue needed.

**Credits Overview:** balance card, paginated transaction list, redeem option grid.

## Section 5: Conflict Management

### Line C Exclusive Files (zero conflict)

All model files, service files, API router files, web router files, templates, and test files listed above are owned solely by Line C.

### Append-Only Conflicts (trivial resolve)

| File | Line C Change |
|------|--------------|
| `routers/api/v1/api.go` | Append grant-rounds + credits groups in `/hackforger` |
| `routers/web/web.go` | Append `/grants`, `/credits`, `/-/admin/credits` route groups |
| `options/locale/locale_en-US.ini` | Append `hackforger.grant.*` + `hackforger.credits.*` keys |
| `templates/hackforger/explore.tmpl` | Fill grants tab content |
| `models/hackforger/action_types.go` | Add `ActionGrantRoundCancelled = 57` |

### Special Attention

`routers/web/hackforger/hackathon.go` currently contains `ExploreGrants`. Line C moves it to `grants.go` and removes from `hackathon.go`. Convention: Line C only touches `ExploreGrants`, Line B only touches `ExploreHackathons`.

`services/hackforger/notifier.go`: Line C does NOT modify `PublishHackforgerAction`. Calls are made from `grants.go` service only.

## Section 6: Testing

### Unit Tests

```
models/hackforger/grant_round_test.go
  TestCreateGrantRound
  TestGetGrantRoundBySlug
  TestListGrantRounds_FilterByStatus
  TestGetGrantRoundBudgetUsage
  TestDeleteGrantRound_OnlyDraft

models/hackforger/grant_project_test.go
  TestCreateGrantProject
  TestGetGrantProjectByUserAndRound_Duplicate
  TestListGrantProjectsByRound_FilterByStatus
  TestCountGrantProjectsByRound

models/hackforger/credits_test.go
  TestGetCreditAccount
  TestCreateRedeemOption
  TestUpdateRedeemOrder

models/hackforger/grant_project_test.go (continued)
  TestDeleteGrantProject_OnlyPending

services/hackforger/grants_test.go
  TestGrantRoundStatusTransition_Valid
  TestGrantRoundStatusTransition_Invalid
  TestDeleteGrantRound_NonDraft_Rejected
  TestSubmitProject_RoundNotOpen
  TestSubmitProject_Duplicate
  TestAllocateAward_ExceedsBudget
  TestFinalizeRound_UnallocatedProjects
  TestDistributeProject_CreditsDeposit
  TestDistributeProject_AutoDistributeRound
  TestDistributeRound_BatchConvenience
  TestCancelRound
  TestCancelRound_AlreadyDistributed_Rejected
  TestCancelRound_AlreadyCancelled_Rejected

services/hackforger/credits_test.go
  TestAdminDeposit
  TestAdminDeduct
  TestAdminDeduct_InsufficientBalance
  TestFulfillOrder
  TestCancelOrder_RefundsBalance
```

### API Integration Tests

```
tests/integration/hackforger_grant_test.go
  TestAPIGrantRoundCRUD
  TestAPIGrantRoundStatusFlow
  TestAPIGrantProjectSubmitAndApprove
  TestAPIAllocateAndFinalize
  TestAPIDistributeProject_CreditsReceived
  TestAPIDistributeRound_BatchAll
  TestAPICancelRound
  TestAPIExportCSV

tests/integration/hackforger_credits_test.go
  TestAPICreditsBalance
  TestAPIRedeem
  TestAPIAdminDeposit
  TestAPIAdminFulfillOrder
```

### Feed Tests (Limited Scope)

Phase 1 audience is actor + global only. Feed tests verify:
- Events written to action table
- Content JSON deserializes correctly
- Global feed query returns events

Full audience distribution tests deferred to post-merge audience resolution task.

### E2E Manual Verification

After implementation, execute the manual E2E flow stored in the corresponding
private instance repository and fill its report.
