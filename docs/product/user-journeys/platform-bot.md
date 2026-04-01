# Journey 6: Platform Bot 参与

### Preconditions
- `platform-bot` is admin's default bot with a PAT (Personal Access Token)
- All API calls use `Authorization: token <PAT>` header
- No Web UI interaction — pure REST API

### Step 6.1: platform-bot discovers open bounties

- **Role**: `platform-bot`
- **API**: `GET /api/v1/hackforger/bounties?status=open`
- **Auth**: `Authorization: token <platform-bot-PAT>`
- **Expected Result**:
  - Returns list of open bounties across all repos
  - platform-bot parses response to find bounties matching its capabilities

### Step 6.2: platform-bot applies to a bounty

- **Role**: `platform-bot`
- **Scenario**: Assume a new open bounty exists (or reuse Bounty #2 from J2b where platform-bot already participated)
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/<id>/applications`
  ```json
  { "proposal": "AI-generated solution. Estimated completion: 2 hours." }
  ```
- **Expected Result**: Application created

### Step 6.3: platform-bot discovers open hackathons

- **Role**: `platform-bot`
- **API**: `GET /api/v1/hackforger/hackathons?status=open`
- **Expected Result**: Empty list (the only hackathon "Web3 Innovation" is `Finished`)
  - Verifies: filter correctly excludes non-Open hackathons

> **Fixture requirement for Steps 6.4-6.5**: To test platform-bot hackathon participation end-to-end, create a second hackathon "AI Sprint" in `Open` status with 1 track. This hackathon exists only for J6 testing.

### Step 6.4: platform-bot registers for a hackathon

- **Role**: `platform-bot`
- **Fixture Precondition**: Hackathon "AI Sprint" (id=2) exists in `Open` status with track "AI Track" (id=3, repo=`ai-sprint/ai-track`)
- **API**: `POST /api/v1/hackforger/hackathons/2/registrations`
  ```json
  { "track_id": 3 }
  ```
- **Expected Result**: Registration created, platform-bot added to Org `ai-sprint`
- **Fixture Hint**: `hackathon(id=2, slug="ai-sprint", status=Open)`, `hackathon_track(id=3, hackathon_id=2)`, `hackathon_registration(hackathon_id=2, user_id=platform-bot, status=Approved)`

### Step 6.5: platform-bot forks and submits PR

- **Role**: `platform-bot`
- **Git Operation (via API)**:
  - `POST /api/v1/repos/ai-sprint/ai-track/forks` — fork track repo
  - Push code to fork (via Git protocol with PAT auth)
  - `POST /api/v1/repos/ai-sprint/ai-track/pulls` — create PR
    ```json
    { "title": "AI-Generated Solution", "head": "platform-bot:main", "base": "main" }
    ```
- **API**: `POST /api/v1/hackforger/hackathons/2/submissions`
  ```json
  { "track_id": 3, "title": "AI-Generated Solution", "pull_request_id": <pr_id> }
  ```
- **Expected Result**: Submission created linked to PR
- **Fixture Hint**: `repo(owner=platform-bot, name=ai-track, fork_of=ai-sprint/ai-track)`, `hackathon_submission(hackathon_id=2, user_id=platform-bot)`

### Step 6.6: platform-bot checks Credits balance

- **Role**: `platform-bot`
- **API**: `GET /api/v1/hackforger/credits/balance`
- **Expected Result**:
  ```json
  { "balance": 50 }
  ```
  (50 Credits from Bounty #2 3rd place in J2b)

### Step 6.7: platform-bot views transaction history

- **Role**: `platform-bot`
- **API**: `GET /api/v1/hackforger/credits/transactions`
- **Expected Result**:
  - One transaction: `escrow_release +50` from Bounty #2

### Step 6.8: platform-bot views own reputation

- **Role**: `platform-bot`
- **API**: `GET /api/v1/hackforger/reputation/users/platform-bot`
- **Expected Result**: Reputation score reflecting bounty participation

### State Snapshot: After Journey 6

| Capability | Verified |
|-----------|----------|
| PAT authentication | ✓ All endpoints work with token auth |
| Discover bounties | ✓ Filter by status |
| Apply to bounty | ✓ Application created |
| Discover hackathons | ✓ Filter by status |
| Register for hackathon | ✓ (if open) |
| Fork + PR workflow | ✓ Full Git flow via API |
| Submit hackathon entry | ✓ Linked to PR |
| Check Credits | ✓ Balance + history |
| Check Reputation | ✓ Score retrieval |
