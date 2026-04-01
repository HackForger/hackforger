# Journey 5: Feed + 社交 + 发现

### Preconditions
- All previous journeys completed
- Multiple Feed events already generated
- hacker1 follows organizer and hacker2

### Step 5.1: hacker1 views Dashboard Feed (Following tab)

- **Role**: `hacker1`
- **Web**: `/` (Dashboard, Community tab)
- **API**: `GET /api/v1/hackforger/feed?type=following&page=1&limit=20`
- **Expected Result**: Feed contains events from followed users and global events:
  - `hackathon_created(30)` — organizer created Web3 Innovation
  - `hackathon_phase_changed(50)` — organizer published / started / judged
  - `hackathon_submitted(32)` — hacker2 submitted (hacker1 follows hacker2)
  - `bounty_created(34)` — global event
  - `grant_round_opened(54)` — global event
  - Events are reverse-chronological, paginated
- **Fixture Hint**: Verify `hackforger_action` rows with correct `user_id` (audience) and `op_type`

### Step 5.2: hacker1 views Global Feed

- **Role**: `hacker1`
- **API**: `GET /api/v1/hackforger/feed?type=global&page=1&limit=20`
- **Expected Result**: All HackForger events regardless of follow relationships
  - Includes all 24 event types generated across J1-J4
  - Sorted by `created_unix` DESC

### Step 5.3: Explore Hackathons

- **Role**: `hacker1` (or anonymous)
- **Web**: `/explore/hackforger/hackathons`
- **API**: `GET /api/v1/hackforger/hackathons`
- **Expected Result**:
  - Lists "Web3 Innovation Challenge" with status `Finished`
  - Shows participant count, track count, prize pool

### Step 5.4: Explore Bounties

- **Role**: `hacker1`
- **Web**: `/explore/hackforger/bounties`
- **API**: `GET /api/v1/hackforger/bounties`
- **Expected Result**:
  - Lists all bounties across all repos
  - Filters available: status, mode
  - Shows bounty #1 (Paid), #2 (Paid), #3 (Expired), #4 (Cancelled)

### Step 5.5: Explore Grant Rounds

- **Role**: `hacker1`
- **Web**: `/explore/hackforger/grants`
- **API**: `GET /api/v1/hackforger/grants/rounds`
- **Expected Result**:
  - Lists "Q2 2026 Open Source Grant" with status `Distributed`
  - Shows budget, number of funded projects

### Step 5.6: Search across entities

- **Role**: `hacker1`
- **Operation**: Search for "DeFi" across all HackForger entities
- **API**: `GET /api/v1/hackforger/search?q=DeFi&scope=all`
- **Expected Result**:
  - Results include:
    - Hackathon Track "DeFi Track"
    - Hackathon Submission "DeFi Lending Protocol"
    - Grant Project "DeFi Testing Framework"
  - Results grouped by entity type
- **Web**: `/explore/hackforger/search?q=DeFi`

### Step 5.7: Search with scope filter

- **Role**: `hacker1`
- **API**: `GET /api/v1/hackforger/search?q=DeFi&scope=bounties`
- **Expected Result**: No results (no bounty contains "DeFi")
- **API**: `GET /api/v1/hackforger/search?q=OAuth&scope=bounties`
- **Expected Result**: Bounty #1 "Implement OAuth2 PKCE flow"

### Step 5.8: View Reputation leaderboard

- **Role**: any user
- **API**: `GET /api/v1/hackforger/reputation/leaderboard`
- **Web**: `/explore/hackforger/reputation`
- **Expected Result**:
  - hacker1 ranked highest (hackathon 1st + bounty wins + grant funded)
  - hacker2 ranked second (hackathon 2nd + bounty 2nd)
  - platform-bot ranked third (bounty 3rd place only)
  - Scores reflect weighted contributions from all modules

### Step 5.9: View individual reputation

- **Role**: `hacker1`
- **API**: `GET /api/v1/hackforger/reputation/users/hacker1`
- **Expected Result**:
  ```json
  {
    "username": "hacker1",
    "total_score": 850,
    "tier": "gold",
    "breakdown": {
      "hackathon": 400,
      "bounty": 300,
      "grant": 150
    }
  }
  ```
  (Scores are illustrative — actual weights defined in `services/hackforger/reputation.go`)

### Step 5.10: Reaction on Bounty Issue (Social)

- **Role**: `hacker2`
- **Operation**: Add a reaction to hacker1's bounty completion comment
- **API**: `POST /api/v1/repos/organizer/oss-project/issues/1/comments/<comment_id>/reactions` (Forgejo native)
  ```json
  { "content": "+1" }
  ```
- **Expected Result**: Reaction added to comment (Forgejo native, no HackForger-specific handling needed)

### State Snapshot: After Journey 5

| Feature | Verified |
|---------|----------|
| Following Feed | ✓ Shows followed users' events |
| Global Feed | ✓ Shows all events |
| Explore: Hackathons | ✓ Filterable listing |
| Explore: Bounties | ✓ Cross-repo listing |
| Explore: Grants | ✓ Round listing |
| Search | ✓ Full-text across entities, scope filter |
| Reputation | ✓ Leaderboard + individual scores |
| Social: Follow/Star/Watch/Reaction | ✓ Woven into J1-J5 |
