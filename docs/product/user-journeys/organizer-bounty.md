# Journey 2: Bounty 全生命周期

### Preconditions
- `organizer` owns repo `organizer/oss-project` with existing issues
- `admin` has deposited 600 Credits to `organizer`'s account for bounty rewards
- hacker1, hacker2, platform-bot exist

> **v0.1 集成场景：** 在实际使用中，Bounty 常在 Hackathon Hacking 阶段内创建 — 参赛者在赛道 Repo 上发起 Issue 并挂载 Bounty 寻求协作。

### Step 2.0: Admin deposits Credits to organizer

- **Role**: `admin`
- **API**: `POST /api/v1/hackforger/credits/admin/deposit`
  ```json
  { "username": "organizer", "amount": 600, "reason": "Bounty reward pool" }
  ```
- **Expected Result**: organizer balance = 600
- **Fixture Hint**: `credit_account(organizer, balance=600)`

---

### Journey 2a: Exclusive Bounty (Happy Path)

### Step 2a.1: Organizer creates Issue

- **Role**: `organizer`
- **Operation**: Create an issue on their repo for the bounty target
- **API**: `POST /api/v1/repos/organizer/oss-project/issues` (Forgejo native)
  ```json
  { "title": "Implement OAuth2 PKCE flow", "body": "We need OAuth2 PKCE support..." }
  ```
- **Expected Result**: Issue #1 created on `organizer/oss-project`
- **Fixture Hint**: `issue(repo=organizer/oss-project, index=1)`

### Step 2a.2: Organizer creates Exclusive Bounty on Issue

- **Role**: `organizer`
- **Operation**: Attach an exclusive bounty to the issue
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties`
  ```json
  {
    "issue_index": 1,
    "mode": 0,
    "description": "Implement OAuth2 PKCE flow with tests",
    "deadline": "2026-05-01T00:00:00Z"
  }
  ```
- **Expected Result**:
  - Bounty created with status = `Open(0)`, mode = `Exclusive(0)`
  - Bounty linked to Issue #1 (1:1)
  - Feed event: `bounty_created(34)` — audience: global + repo watchers
- **Fixture Hint**: `bounty(id=1, repo_id=<oss-project>, issue_id=1, mode=Exclusive, status=Open)`

### Step 2a.3: Organizer adds Reward with escrow

- **Role**: `organizer`
- **Operation**: Set the bounty reward amount (triggers Credits escrow)
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/1/rewards`
  ```json
  { "amount": 300, "description": "Full implementation with tests" }
  ```
- **Expected Result**:
  - Reward created: 300 Credits
  - Credits escrowed from organizer: balance 600 → 300 (300 held by platform)
  - Transaction type: `escrow`
  - 验证 organizer 余额: 600 → 300（300 积分被托管）
- **Fixture Hint**: `bounty_reward(bounty_id=1, amount=300)`, `credit_transaction(escrow, organizer, -300)`

### Step 2a.4: hacker1 applies for the Bounty

- **Role**: `hacker1`
- **Operation**: Submit application to claim the bounty
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/1/applications`
  ```json
  { "proposal": "I have experience with OAuth2. Estimated 3 days." }
  ```
- **Expected Result**:
  - Application created with status = `Pending(0)`
- **Fixture Hint**: `bounty_application(bounty_id=1, user_id=hacker1, status=Pending)`

### Step 2a.5: Organizer accepts hacker1's application (Bounty → Claimed)

- **Role**: `organizer`
- **Operation**: Review and accept the application
- **API**: `PUT /api/v1/repos/organizer/oss-project/hackforger/bounties/1/applications/<aid>/review`
  ```json
  { "status": 1, "comment": "Approved, go ahead" }
  ```
- **Expected Result**:
  - Application status: `Pending(0)` → `Accepted(1)`
  - Bounty status: `Open(0)` → `Claimed(1)`
  - hacker1 is the exclusive claimant
  - Feed event: `bounty_claimed(35)` — audience: repo watchers + followers
- **Fixture Hint**: `bounty(id=1, status=Claimed)`, `bounty_application(status=Accepted)`

### Step 2a.6: hacker1 forks repo and develops

- **Role**: `hacker1`
- **Git Operation**:
  - `POST /api/v1/repos/organizer/oss-project/forks` → creates `hacker1/oss-project`
  - hacker1 pushes OAuth2 PKCE implementation to fork
- **Expected Result**: Fork `hacker1/oss-project` with implementation commits
- **Fixture Hint**: `repo(owner=hacker1, name=oss-project, fork_of=organizer/oss-project)`

### Step 2a.7: hacker1 submits PR (Claimed → InReview)

- **Role**: `hacker1`
- **Git Operation**: `POST /api/v1/repos/organizer/oss-project/pulls`
  ```json
  {
    "title": "feat: implement OAuth2 PKCE flow",
    "head": "hacker1:main",
    "base": "main",
    "body": "Closes #1\n\nImplements PKCE with full test coverage"
  }
  ```
- **Expected Result**:
  - PR created linking to Issue #1
  - Bounty status: `Claimed(1)` → `InReview(2)`
  - Feed event: `bounty_delivered(36)` — audience: repo watchers
- **Fixture Hint**: `bounty(id=1, status=InReview)`, `pull_request(repo=organizer/oss-project, head=hacker1:main)`

### Step 2a.8: Organizer reviews and completes (InReview → Completed)

- **Role**: `organizer`
- **Operation**: Review PR, approve, and mark bounty as completed
- **Git Operation**: Organizer adds PR Review (approve)
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/1/complete`
- **Expected Result**:
  - Bounty status: `InReview(2)` → `Completed(3)`
  - Feed event: `bounty_completed(37)` — audience: global
- **Fixture Hint**: `bounty(id=1, status=Completed)`

### Step 2a.9: Organizer marks as Paid (Completed → Paid)

- **Role**: `organizer`
- **Operation**: Trigger Credits payment to hacker1
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/1/pay`
- **Expected Result**:
  - Bounty status: `Completed(3)` → `Paid(4)`
  - Escrowed Credits released to hacker1: `escrow_release` transaction
  - hacker1 balance: 500 (from J1) + 300 = 800
  - Feed event: `bounty_paid(43)` — audience: global
- **Fixture Hint**: `bounty(id=1, status=Paid)`, `credit_transaction(escrow_release, hacker1, +300)`

### State Snapshot: After Journey 2a

| Entity | State |
|--------|-------|
| Bounty #1 (Exclusive) | `Paid(4)` |
| organizer Credits | 300 (600 - 300 escrowed, now released) |
| hacker1 Credits | 800 (500 + 300) |

---

### Journey 2b: Competitive Bounty

### Step 2b.1: Organizer creates Issue #2

- **Role**: `organizer`
- **API**: `POST /api/v1/repos/organizer/oss-project/issues`
  ```json
  { "title": "Design new landing page", "body": "We need a fresh landing page design..." }
  ```
- **Expected Result**: Issue #2 created
- **Fixture Hint**: `issue(repo=organizer/oss-project, index=2)`

### Step 2b.2: Organizer creates Competitive Bounty with multiple rewards

- **Role**: `organizer`
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties`
  ```json
  {
    "issue_index": 2,
    "mode": 1,
    "description": "Best landing page design wins",
    "deadline": "2026-05-15T00:00:00Z"
  }
  ```
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/2/rewards` (x3)
  ```json
  { "place": 1, "amount": 150, "description": "1st place" }
  { "place": 2, "amount": 100, "description": "2nd place" }
  { "place": 3, "amount": 50,  "description": "3rd place" }
  ```
- **Expected Result**:
  - Bounty #2 created, mode = `Competitive(1)`, status = `Open(0)`
  - 3 rewards totaling 300 Credits
  - Credits escrowed from organizer: 300 → 0 (300 held)
  - Transaction type: `escrow`
  - Feed event: `bounty_created(34)`
- **Fixture Hint**: `bounty(id=2, mode=Competitive, status=Open)`, `bounty_reward(place=1, amount=150)` etc.

### Step 2b.3: hacker1 watches the repo (Social)

- **Role**: `hacker1`
- **Operation**: Watch the repo to get notifications on bounty updates
- **API**: `PUT /api/v1/repos/organizer/oss-project/subscription` (Forgejo native)
- **Expected Result**: hacker1 subscribed to repo notifications
- **Fixture Hint**: `watch(user=hacker1, repo=organizer/oss-project)`

### Step 2b.4: Three participants submit solutions

- **Role**: `hacker1`, `hacker2`, `platform-bot`
- **Operations**: Each submits a PR with their design
- **Git Operations**: Each forks (if not already forked) and creates PR
  - hacker1: PR from `hacker1/oss-project` (already forked from 2a)
  - hacker2: `POST /api/v1/repos/organizer/oss-project/forks`, then PR
  - platform-bot: `POST /api/v1/repos/organizer/oss-project/forks` (via PAT), then PR
- **API**: Each creates application
  - `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/2/applications` (x3)
- **Expected Result**:
  - 3 PRs created on the repo
  - 3 applications created with status = `Accepted(1)` (Competitive mode auto-accepts all applications — no Pending gate)
  - Bounty stays `Open(0)` (Competitive mode allows multiple concurrent participants)
  - Note: `BountyApplication Pending(0)` is only meaningful in Exclusive mode (covered by J2a Step 2a.4)
- **Fixture Hint**: `bounty_application(bounty_id=2, user_id=hacker1, status=Accepted)`, same for hacker2 and platform-bot

### Step 2b.5: Organizer selects winners

- **Role**: `organizer`
- **Operation**: Review all submissions and select 1st, 2nd, 3rd place
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/2/winners`
  ```json
  {
    "winners": [
      { "user_id": <hacker1_id>, "place": 1 },
      { "user_id": <hacker2_id>, "place": 2 },
      { "user_id": <platform-bot_id>, "place": 3 }
    ]
  }
  ```
- **Expected Result**:
  - Bounty status: `Open(0)` → `Completed(3)`
  - Winners recorded with places
  - Feed event: `bounty_winners_selected(38)` — audience: global
- **Fixture Hint**: `bounty_winner(bounty_id=2, user_id=hacker1, place=1)` etc.

### Step 2b.6: Organizer pays winners (Completed → Paid)

- **Role**: `organizer`
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/2/pay`
- **Expected Result**:
  - Bounty status: `Completed(3)` → `Paid(4)`
  - Escrowed Credits distributed to winners:
    - hacker1: +150 (balance: 500+300+150 = 950)
    - hacker2: +100 (balance: 300+100 = 400)
    - platform-bot: +50 (balance: 0+50 = 50)
  - Transaction type: `escrow_release` for each winner
  - Feed event: `bounty_paid(43)`
- **Fixture Hint**: `bounty(id=2, status=Paid)`, `credit_transaction(escrow_release, hacker1, +150)`, `credit_transaction(escrow_release, hacker2, +100)`, `credit_transaction(escrow_release, platform-bot, +50)`

### State Snapshot: After Journey 2b

| Entity | State |
|--------|-------|
| Bounty #2 (Competitive) | `Paid(4)` |
| organizer Credits | 0 (600 - 300 - 300 all escrowed and released) |
| hacker1 Credits | 950 (500 + 300 + 150) |
| hacker2 Credits | 400 (300 + 100) |
| platform-bot Credits | 50 |

---

### Journey 2c: Exception Paths

### Step 2c.1: Bounty expiry (deadline passed, no completion)

- **Role**: system (cron or manual trigger by `organizer`)
- **Scenario**: Organizer creates Bounty #3 on Issue #3 with 100 Credits reward, deadline passes with no submission
- **Setup**:
  - `POST /api/v1/repos/organizer/oss-project/issues` → Issue #3
  - `POST /api/v1/repos/organizer/oss-project/hackforger/bounties` → Bounty #3 (Exclusive, deadline in past for test)
  - Note: organizer has 0 Credits at this point. For this test, admin deposits 100 more.
  - `POST /api/v1/hackforger/credits/admin/deposit` → organizer +100
  - `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/3/rewards` → 100 Credits (escrowed)
- **Trigger**: Deadline passes, or manual expire
- **Expected Result**:
  - Bounty status: `Open(0)` → `Expired(5)`
  - Escrowed Credits refunded to organizer: `escrow_refund` transaction
  - organizer balance: 0 + 100 (deposited) - 100 (escrowed) + 100 (refunded) = 100
  - Feed event: `bounty_expired(52)`
- **Fixture Hint**: `bounty(id=3, status=Expired)`, `credit_transaction(escrow_refund, organizer, +100)`

### Step 2c.2: Bounty cancellation by organizer

- **Role**: `organizer`
- **Scenario**: Organizer creates Bounty #4 on Issue #4, then cancels it
- **Setup**: Similar to 2c.1 — create issue, bounty, reward (escrow)
  - admin deposits another 100 to organizer
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/4/cancel`
- **Expected Result**:
  - Bounty status: `Open(0)` → `Cancelled(6)`
  - Escrowed Credits refunded to organizer: `escrow_refund`
  - Feed event: `bounty_cancelled(53)`
- **Fixture Hint**: `bounty(id=4, status=Cancelled)`, `credit_transaction(escrow_refund, organizer, +100)`

### Step 2c.3: Application rejected

- **Role**: `organizer`, `hacker2`
- **Scenario**: hacker2 applies to Bounty #1 (but hacker1 was already accepted — exclusive mode)
- **Note**: This step conceptually happens before 2a.5. In fixtures, test that a second application on an already-claimed exclusive bounty is rejected.
- **API**: `PUT /api/v1/repos/organizer/oss-project/hackforger/bounties/1/applications/<hacker2_aid>/review`
  ```json
  { "status": 2, "comment": "Bounty already claimed" }
  ```
- **Expected Result**:
  - Application status: `Pending(0)` → `Rejected(2)`
  - Bounty remains `Claimed(1)` (no change)
- **Fixture Hint**: `bounty_application(bounty_id=1, user_id=hacker2, status=Rejected)`

### State Snapshot: After All Journey 2

| Entity | State |
|--------|-------|
| Bounty #1 (Exclusive) | `Paid(4)` |
| Bounty #2 (Competitive) | `Paid(4)` |
| Bounty #3 (Expired) | `Expired(5)` |
| Bounty #4 (Cancelled) | `Cancelled(6)` |
| All BountyApplication statuses | Pending, Accepted, Rejected covered |
| organizer Credits | 200 (600 deposited - 600 escrowed/released in J2a+J2b + 2×100 admin deposits in J2c, net +200 from refunds) |
| hacker1 Credits | 950 |
| hacker2 Credits | 400 |
| platform-bot Credits | 50 |
