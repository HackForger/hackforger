# Journey 1: Hackathon 全生命周期

### Preconditions
- All 7 users exist
- `admin` has deposited 1000 Credits to `organizer`'s account for prizes
- No hackathon exists yet

### Step 1.1: Admin deposits Credits to organizer

- **Role**: `admin`
- **Operation**: Deposit 1000 Credits to organizer's account for hackathon prizes
- **API**: `POST /api/v1/hackforger/credits/admin/deposit`
  ```json
  { "username": "organizer", "amount": 1000, "reason": "Hackathon prize pool" }
  ```
- **Expected Result**:
  - organizer balance = 1000
  - Transaction type: `admin_deposit`
- **Fixture Hint**: `credit_account(organizer, balance=1000)`, `credit_transaction(admin_deposit, 1000)`

### Step 1.2: Organizer creates Hackathon (Draft)

- **Role**: `organizer`
- **Operation**: Create a new hackathon "Web3 Innovation Challenge"
- **API**: `POST /api/v1/hackforger/hackathons`
  ```json
  {
    "name": "Web3 Innovation Challenge",
    "slug": "web3-innovation",
    "description": "Build the future of decentralized web",
    "max_tracks": 3,
    "prize_pool": 800
  }
  ```
- **Git Operation**: Auto-creates Organization `web3-innovation` on Forgejo
- **Expected Result**:
  - Hackathon created with status = `Draft(0)`
  - Org `web3-innovation` exists with `organizer` as owner
  - Feed event: `hackathon_created(30)` — audience: global

> **编辑器：** 描述字段支持 Markdown 格式（使用 Forgejo 原生 Markdown 编辑器，支持图片上传和代码块）。

- **Fixture Hint**: `hackathon(id=1, status=Draft, org_id=<auto>)`

### Step 1.3: Organizer creates Tracks (Repos)

- **Role**: `organizer`
- **Operation**: Create 2 tracks under the hackathon
- **API**: `POST /api/v1/hackforger/hackathons/1/tracks` (x2)
  ```json
  // Track 1
  { "name": "DeFi Track", "slug": "defi-track", "description": "Build DeFi protocols", "prize_allocation": 500 }
  // Track 2
  { "name": "NFT Track", "slug": "nft-track", "description": "Build NFT tooling", "prize_allocation": 300 }
  ```
- **Git Operation**: Auto-creates repos `web3-innovation/defi-track` and `web3-innovation/nft-track` with initial README template
- **Expected Result**:
  - 2 tracks created, each linked to a repo
  - Repos have initial commit with README + rules
  - Prize allocation total (800) ≤ prize pool (800) ✓
- **Fixture Hint**: `hackathon_track(id=1, hackathon_id=1, repo_id=<auto>)`, `hackathon_track(id=2, ...)`

### Step 1.4: Organizer sets judging criteria

- **Role**: `organizer`
- **Operation**: Define scoring criteria for each track
- **API**: `PUT /api/v1/hackforger/hackathons/1/tracks/1` (criteria included in track update body)
  ```json
  // For DeFi Track
  { "criteria": [
    { "name": "Innovation", "weight": 40, "description": "Novelty of approach" },
    { "name": "Technical Quality", "weight": 35, "description": "Code quality, architecture" },
    { "name": "Presentation", "weight": 25, "description": "Demo and documentation" }
  ]}
  ```
- **Expected Result**:
  - 3 criteria per track, weights sum to 100
- **Fixture Hint**: `hackathon_track_criteria(track_id=1, name="Innovation", weight=40)` x3 per track

### Step 1.5: Organizer assigns judges

- **Role**: `organizer`
- **Operation**: Add judge1 and judge2 as hackathon judges
- **API**: `POST /api/v1/hackforger/hackathons/1/judges` (x2)
  ```json
  { "user_id": <judge1_id> }
  { "user_id": <judge2_id> }
  ```
- **Expected Result**:
  - 2 judges added to hackathon
  - Judges cannot also register as participants (enforced)
- **Fixture Hint**: `hackathon_judge(hackathon_id=1, user_id=judge1)`, same for judge2

### Step 1.6: Organizer publishes Hackathon (Draft → Open)

- **Role**: `organizer`
- **Operation**: Publish the hackathon, opening registration
- **API**: `POST /api/v1/hackforger/hackathons/1/publish`
- **Precondition Check**: At least 1 track exists, at least 1 judge assigned
- **Expected Result**:
  - Status changes: `Draft(0)` → `Open(1)`
  - Feed event: `hackathon_phase_changed(50)` — audience: global
  - Hackathon visible on Explore page
- **Fixture Hint**: `hackathon(id=1, status=Open)`

### Step 1.7: hacker1 follows organizer (Social)

- **Role**: `hacker1`
- **Operation**: Follow organizer to see their activity in Feed
- **API**: `PUT /api/v1/user/following/organizer` (Forgejo native)
- **Expected Result**:
  - hacker1 now follows organizer
  - Organizer's future actions appear in hacker1's Following Feed
- **Fixture Hint**: `follow(follower=hacker1, followee=organizer)`

### Step 1.8: hacker1 registers for Hackathon

- **Role**: `hacker1`
- **Operation**: Register for the hackathon
- **API**: `POST /api/v1/hackforger/hackathons/1/registrations`
  ```json
  { "track_id": 1 }
  ```
- **Git Operation**: hacker1 added as member of Org `web3-innovation`
- **Expected Result**:
  - Registration created with status = `Approved(1)` (auto-approve if no manual review configured)
  - hacker1 is now a member of `web3-innovation` org
  - Feed event: `hackathon_registered(31)` — audience: followers + org
- **Fixture Hint**: `hackathon_registration(hackathon_id=1, user_id=hacker1, status=Approved, track_id=1)`

### Step 1.9: hacker2 registers and forms a team (Social)

- **Role**: `hacker2`
- **Operation**: Register for the same hackathon, same track
- **API**: `POST /api/v1/hackforger/hackathons/1/registrations`
  ```json
  { "track_id": 1 }
  ```
- **Git Operation**: hacker2 added as member of Org `web3-innovation`
- **Social**: hacker1 and hacker2 follow each other
  - `PUT /api/v1/user/following/hacker2` (by hacker1)
  - `PUT /api/v1/user/following/hacker1` (by hacker2)
- **Team Formation**: Create team "DeFi Duo" in the org
  - `POST /api/v1/orgs/web3-innovation/teams` (by organizer or hacker2)
    ```json
    { "name": "DeFi Duo", "permission": "write" }
    ```
  - `PUT /api/v1/teams/<team_id>/members/hacker1`
  - `PUT /api/v1/teams/<team_id>/members/hacker2`
- **Expected Result**:
  - hacker2 registered, Org membership granted
  - Team "DeFi Duo" created with both hackers as members
  - hacker1 ↔ hacker2 mutual follow
  - Feed event: `hackathon_registered(31)` for hacker2
- **Fixture Hint**: `hackathon_registration(user_id=hacker2, track_id=1)`, `team(org=web3-innovation, name="DeFi Duo")`, `team_member(hacker1, hacker2)`

### Step 1.10: Organizer starts Hacking phase (Open → Hacking)

- **Role**: `organizer`
- **Operation**: Transition hackathon to Hacking phase
- **API**: `POST /api/v1/hackforger/hackathons/1/start`
- **Expected Result**:
  - Status changes: `Open(1)` → `Hacking(2)`
  - Feed event: `hackathon_phase_changed(50)` — audience: global + org
- **Fixture Hint**: `hackathon(id=1, status=Hacking)`

### Step 1.11: hacker1 forks Track Repo and develops

- **Role**: `hacker1`
- **Operation**: Fork the DeFi Track repo to start coding
- **Git Operation**:
  - `POST /api/v1/repos/web3-innovation/defi-track/forks` (Forgejo native)
    ```json
    { "organization": "" }
    ```
  - Creates `hacker1/defi-track` as a fork
  - hacker1 pushes commits to their fork
- **Expected Result**:
  - Fork `hacker1/defi-track` exists
  - hacker1 can push code to their fork
- **Fixture Hint**: `repo(owner=hacker1, name=defi-track, fork_of=web3-innovation/defi-track)`

### Step 1.12: hacker2 forks and develops (team submission)

- **Role**: `hacker2`
- **Operation**: Fork the same track repo for team submission
- **Git Operation**: `POST /api/v1/repos/web3-innovation/defi-track/forks`
  - Creates `hacker2/defi-track`
  - Both hacker1 and hacker2 can push to hacker2's fork (team repo)
- **Expected Result**:
  - Fork `hacker2/defi-track` exists
  - Team "DeFi Duo" can collaborate on this fork
- **Fixture Hint**: `repo(owner=hacker2, name=defi-track, fork_of=web3-innovation/defi-track)`

### Step 1.13: Stars on Track Repos (Social)

- **Role**: `hacker1`, `judge1`, `platform-bot`
- **Operation**: Star the DeFi Track repo (community interest signal)
- **API**: `PUT /api/v1/user/starred/web3-innovation/defi-track` (x3, by different users)
- **Expected Result**:
  - `web3-innovation/defi-track` has 3 stars
  - Stars visible on repo page as a popularity/voting signal
- **Fixture Hint**: `star(user=hacker1, repo=web3-innovation/defi-track)` x3

### Step 1.14: hacker1 submits via PR

- **Role**: `hacker1`
- **Operation**: Create PR from fork to track repo, then register as hackathon submission
- **Git Operation**:
  - `POST /api/v1/repos/web3-innovation/defi-track/pulls` (Forgejo native)
    ```json
    {
      "title": "DeFi Lending Protocol - hacker1 submission",
      "head": "hacker1:main",
      "base": "main",
      "body": "My DeFi lending protocol submission"
    }
    ```
- **API**: `POST /api/v1/hackforger/hackathons/1/submissions`
  ```json
  {
    "track_id": 1,
    "title": "DeFi Lending Protocol",
    "description": "A decentralized lending protocol with flash loans",
    "pull_request_id": <pr_id>
  }
  ```
- **Expected Result**:
  - PR created on `web3-innovation/defi-track`
  - Submission linked to PR (timestamped, diffable, reviewable)
  - Feed event: `hackathon_submitted(32)` — audience: org + followers
- **Fixture Hint**: `hackathon_submission(id=1, hackathon_id=1, track_id=1, user_id=hacker1, pull_id=<auto>)`

> **v0.1 支持两种提交模式：** (1) Fork + PR 模式（如上，推荐的 Git-Native 方式）；(2) Link Repo 模式 — 参赛者在 CreateSubmission 时直接填写已有 Repo URL，不需要 Fork。Journey 中 hacker1 使用 Fork+PR，hacker2 可使用 Link Repo。

### Step 1.15: hacker2 submits via PR (team submission)

- **Role**: `hacker2`
- **Operation**: Same as 1.14 but from hacker2's fork
- **Git Operation**: `POST /api/v1/repos/web3-innovation/defi-track/pulls`
  ```json
  {
    "title": "DeFi Yield Aggregator - Team DeFi Duo",
    "head": "hacker2:main",
    "base": "main"
  }
  ```
- **API**: `POST /api/v1/hackforger/hackathons/1/submissions`
  ```json
  {
    "track_id": 1,
    "title": "DeFi Yield Aggregator",
    "description": "Cross-chain yield aggregator by Team DeFi Duo",
    "pull_request_id": <pr_id>
  }
  ```
- **Expected Result**:
  - Second submission on DeFi Track
  - Feed event: `hackathon_submitted(32)` for hacker2
- **Fixture Hint**: `hackathon_submission(id=2, user_id=hacker2, track_id=1, pull_id=<auto>)`

### State Snapshot: Pre-Judging

| Entity | State |
|--------|-------|
| Hackathon "Web3 Innovation" | `Hacking(2)` |
| DeFi Track | 2 submissions (hacker1, hacker2) |
| NFT Track | 0 submissions |
| hacker1 | Registered, submitted PR, follows organizer + hacker2 |
| hacker2 | Registered, submitted PR, follows hacker1 |
| judge1, judge2 | Assigned, not yet scored |
| Org web3-innovation | Members: organizer, hacker1, hacker2; Team: DeFi Duo |
| Credits (organizer) | 1000 (unspent) |

### Step 1.16: Organizer starts Judging phase (Hacking → Judging)

- **Role**: `organizer`
- **Operation**: Close submissions, transition to judging
- **API**: `POST /api/v1/hackforger/hackathons/1/judge`
- **Web**: `POST /hackforger/hackathons/web3-innovation/manage/judge`
- **Expected Result**:
  - Status changes: `Hacking(2)` → `Judging(3)`
  - No new submissions accepted
  - Feed event: `hackathon_phase_changed(50)`
- **Fixture Hint**: `hackathon(id=1, status=Judging)`

### Step 1.17: judge1 scores submissions

- **Role**: `judge1`
- **Operation**: Score both submissions on DeFi Track by criteria
- **API**: `POST /api/v1/hackforger/hackathons/1/scores` (x2)
  ```json
  // Score for hacker1's submission
  {
    "submission_id": 1,
    "scores": [
      { "criteria_id": 1, "score": 90 },
      { "criteria_id": 2, "score": 85 },
      { "criteria_id": 3, "score": 80 }
    ]
  }
  // Score for hacker2's submission
  {
    "submission_id": 2,
    "scores": [
      { "criteria_id": 1, "score": 75 },
      { "criteria_id": 2, "score": 80 },
      { "criteria_id": 3, "score": 85 }
    ]
  }
  ```
- **Git Operation**: judge1 can add PR Review comments on each submission's PR
- **Expected Result**:
  - judge1's scores recorded for both submissions
  - Weighted score for hacker1 (judge1): 90×0.4 + 85×0.35 + 80×0.25 = 36+29.75+20 = 85.75
  - Weighted score for hacker2 (judge1): 75×0.4 + 80×0.35 + 85×0.25 = 30+28+21.25 = 79.25
  - Feed event: `hackathon_scored(33)` (x2) — audience: org
- **Fixture Hint**: `hackathon_judge_score(judge_id=judge1, submission_id=1, criteria_id=1, score=90)` etc.

### Step 1.18: judge2 scores submissions

- **Role**: `judge2`
- **Operation**: Score both submissions independently
- **API**: `POST /api/v1/hackforger/hackathons/1/scores` (x2)
  ```json
  // Score for hacker1
  { "submission_id": 1, "scores": [
    { "criteria_id": 1, "score": 88 }, { "criteria_id": 2, "score": 90 }, { "criteria_id": 3, "score": 75 }
  ]}
  // Score for hacker2
  { "submission_id": 2, "scores": [
    { "criteria_id": 1, "score": 80 }, { "criteria_id": 2, "score": 78 }, { "criteria_id": 3, "score": 90 }
  ]}
  ```
- **Expected Result**:
  - judge2's scores recorded
  - Weighted score for hacker1 (judge2): 88×0.4 + 90×0.35 + 75×0.25 = 35.2+31.5+18.75 = 85.45
  - Weighted score for hacker2 (judge2): 80×0.4 + 78×0.35 + 90×0.25 = 32+27.3+22.5 = 81.8
  - hacker1 avg: (85.75 + 85.45) / 2 = **85.60**
  - hacker2 avg: (79.25 + 81.80) / 2 = **80.53**
  - hacker1 ranked #1, hacker2 ranked #2
  - Feed event: `hackathon_scored(33)` (x2)
- **Fixture Hint**: `hackathon_judge_score(judge_id=judge2, ...)` etc.

### Step 1.19: Organizer finalizes Hackathon (Judging → Finished)

- **Role**: `organizer`
- **Operation**: Finalize hackathon, confirm rankings, distribute prizes
- **API**: `POST /api/v1/hackforger/hackathons/1/finalize`
- **Expected Result**:
  - Status changes: `Judging(3)` → `Finished(4)`
  - Rankings confirmed: hacker1 = 1st, hacker2 = 2nd
  - Credits auto-distributed from organizer's pool:
    - hacker1 receives 500 Credits (1st place, DeFi Track)
    - hacker2 receives 300 Credits (2nd place, DeFi Track)
    - Transaction type: `deposit` for winners, `withdraw` from organizer
  - Leaderboard frozen and published
  - Feed event: `hackathon_finalized(51)` — audience: global
- **Fixture Hint**: `hackathon(id=1, status=Finished)`, `credit_transaction(deposit, hacker1, 500)`, `credit_transaction(deposit, hacker2, 300)`

### Step 1.19a: 验证积分发放

- **Role**: `hacker1`, `hacker2`, `organizer`
- **Operation**: 各角色查看积分余额确认变化
- **Web**: `/credits` (积分概览页面)
- **Expected Result**:
  - hacker1 余额: +500（1st place DeFi Track）
  - hacker2 余额: +300（2nd place DeFi Track）
  - organizer 余额: 1000 - 500 - 300 = 200（剩余）

### Step 1.20: Verify leaderboard

- **Role**: any user
- **Operation**: View final leaderboard
- **API**: `GET /api/v1/hackforger/hackathons/1/leaderboard`
- **Web**: `/hackforger/hackathons/web3-innovation/leaderboard`
- **Expected Result**:
  ```json
  {
    "rankings": [
      { "rank": 1, "user": "hacker1", "submission": "DeFi Lending Protocol", "total_score": 85.60, "prize": 500 },
      { "rank": 2, "user": "hacker2", "submission": "DeFi Yield Aggregator", "total_score": 80.53, "prize": 300 }
    ]
  }
  ```

### State Snapshot: Post-Hackathon

| Entity | State |
|--------|-------|
| Hackathon | `Finished(4)` |
| Credits: hacker1 | 500 |
| Credits: hacker2 | 300 |
| Credits: organizer | 200 (1000 - 500 - 300) |
| Feed events generated | hackathon_created, phase_changed x3, registered x2, submitted x2, scored x4, finalized |
