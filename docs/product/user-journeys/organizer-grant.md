# Journey 3: Grant 全生命周期

### Preconditions
- `organizer` exists (has 400 remaining Credits from J1+J2, but Grant budget comes from admin deposit)
- `admin` deposits 2000 Credits to `organizer` for the grant round
- hacker1 (950 Credits), hacker2 (400 Credits) exist

> **v0.1 集成场景：** Grant Round 可在 Hackathon 进行期间创建，用于额外资助特定赛道的参赛者。详见 `docs/tests/e2e/tasks/user-journey-full-cycle.md` Phase 5。

### Step 3.0: Admin deposits Credits to organizer for grant round

- **Role**: `admin`
- **API**: `POST /api/v1/hackforger/credits/admin/deposit`
  ```json
  { "username": "organizer", "amount": 2000, "reason": "Q2 Grant Round budget" }
  ```
- **Expected Result**: organizer balance = 400 (J1+J2 remainder) + 2000 = 2400
- **Fixture Hint**: `credit_transaction(admin_deposit, organizer, 2000)`

### Step 3.1: Organizer creates Grant Round (Draft)

- **Role**: `organizer`
- **API**: `POST /api/v1/hackforger/grants/rounds`
  ```json
  {
    "name": "Q2 2026 Open Source Grant",
    "slug": "q2-2026-oss-grant",
    "description": "Funding open-source infrastructure projects",
    "budget": 2000,
    "deadline": "2026-06-01T00:00:00Z"
  }
  ```
- **Expected Result**:
  - Grant Round created with status = `Draft(0)`
  - Feed event: `grant_round_created(39)` — audience: global
- **Fixture Hint**: `grant_round(id=1, status=Draft, budget=2000)`

### Step 3.2: Organizer opens the round (Draft → Open)

- **Role**: `organizer`
- **API**: `POST /api/v1/hackforger/grants/rounds/1/open`
- **Expected Result**:
  - Status: `Draft(0)` → `Open(1)`
  - Applications now accepted
  - Feed event: `grant_round_opened(54)` — audience: global
- **Fixture Hint**: `grant_round(id=1, status=Open)`

### Step 3.3: hacker1 submits Grant Project application

- **Role**: `hacker1`
- **API**: `POST /api/v1/hackforger/grants/rounds/1/projects`
  ```json
  {
    "title": "DeFi Testing Framework",
    "description": "Open-source testing framework for DeFi smart contracts",
    "requested_amount": 1000,
    "repo_url": "https://hackforger.inside.h2os.cloud/hacker1/defi-test-framework"
  }
  ```
- **Expected Result**:
  - Grant Project created with status = `Pending`
  - Feed event: `grant_project_submitted(40)` — audience: global + followers
- **Fixture Hint**: `grant_project(id=1, round_id=1, user_id=hacker1, status=Pending, requested=1000)`

### Step 3.4: hacker2 submits Grant Project application

- **Role**: `hacker2`
- **API**: `POST /api/v1/hackforger/grants/rounds/1/projects`
  ```json
  {
    "title": "NFT Metadata Standard Library",
    "description": "Standardized metadata handling for NFTs across chains",
    "requested_amount": 1500,
    "repo_url": "https://hackforger.inside.h2os.cloud/hacker2/nft-metadata-lib"
  }
  ```
- **Expected Result**:
  - Grant Project created with status = `Pending`
  - Feed event: `grant_project_submitted(40)`
- **Fixture Hint**: `grant_project(id=2, round_id=1, user_id=hacker2, status=Pending, requested=1500)`

### Step 3.5: Organizer closes applications (Open → Review)

- **Role**: `organizer`
- **API**: `POST /api/v1/hackforger/grants/rounds/1/close`
- **Expected Result**:
  - Status: `Open(1)` → `Review(2)`
  - No new applications accepted
  - Feed event: `grant_round_closed(55)`
- **Fixture Hint**: `grant_round(id=1, status=Review)`

### Step 3.6: Organizer reviews and approves/rejects projects

- **Role**: `organizer`
- **Operation**: Approve hacker1's project, reject hacker2's project (over budget)
- **API**: `PUT /api/v1/hackforger/grants/rounds/1/projects/1`
  ```json
  { "status": "approved", "awarded_amount": 1000, "comment": "Strong proposal, fully funded" }
  ```
- **API**: `PUT /api/v1/hackforger/grants/rounds/1/projects/2`
  ```json
  { "status": "rejected", "comment": "Over remaining budget, reapply next round" }
  ```
- **Expected Result**:
  - Project 1: `Pending` → `Approved`, awarded 1000 Credits
  - Project 2: `Pending` → `Rejected`
  - Total allocated (1000) ≤ budget (2000) ✓
  - Feed event: `grant_awarded(41)` for project 1
- **Fixture Hint**: `grant_project(id=1, status=Approved, awarded=1000)`, `grant_project(id=2, status=Rejected)`

### Step 3.7: Organizer finalizes the round (Review → Finalized)

- **Role**: `organizer`
- **Operation**: Lock allocations — commitment made, but Credits not yet distributed
- **API**: `POST /api/v1/hackforger/grants/rounds/1/finalize`
- **Expected Result**:
  - Status: `Review(2)` → `Finalized(3)`
  - Allocations locked (no further changes to awarded amounts)
  - **Credits NOT yet distributed** — organizer will review project progress first
  - Feed event: `grant_round_finalized(56)`
- **Fixture Hint**: `grant_round(id=1, status=Finalized)`

### Step 3.8: Organizer reviews project progress and distributes (Finalized → Distributed)

- **Role**: `organizer`
- **Operation**: After reviewing hacker1's project activity (commits, PRs, milestones in the linked repo), organizer decides to release funds
- **Manual Check**: Organizer visits hacker1's repo, reviews commit history, PR merges, README updates — platform does NOT enforce this, organizer decides based on their own criteria
- **API**: `POST /api/v1/hackforger/grants/rounds/1/distribute`
  > **New endpoint** — not in original prompt's API list. Required by PRD's two-phase funding model. Separate from `/finalize` to enforce the Finalized→Distributed gate.
- **Expected Result**:
  - Status: `Finalized(3)` → `Distributed(4)`
  - Credits transferred:
    - hacker1: +1000 (balance: 950 + 1000 = 1950)
    - organizer: -1000 (balance: 2400 - 1000 = 1400)
  - Project 1 status: `Approved` → `Funded`
  - Transaction type: `deposit` for hacker1, `withdraw` from organizer
  - 验证: organizer 余额 2400 → 1400, hacker1 余额 +1000
  - Feed event: (covered by round state change)
- **Fixture Hint**: `grant_round(id=1, status=Distributed)`, `grant_project(id=1, status=Funded)`, `credit_transaction(deposit, hacker1, 1000)`

### State Snapshot: After Journey 3

| Entity | State |
|--------|-------|
| Grant Round #1 | `Distributed(4)` |
| Grant Project #1 (hacker1) | `Funded` |
| Grant Project #2 (hacker2) | `Rejected` |
| Credits: hacker1 | 1950 (500 + 300 + 150 + 1000) |
| Credits: hacker2 | 400 (unchanged) |
| Credits: organizer | 1400 (400 + 2000 - 1000) |
