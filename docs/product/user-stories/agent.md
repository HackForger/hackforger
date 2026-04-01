# Platform Bot Stories

## Epic 6: Platform Bot Participation

> **Epic Hypothesis:** We believe that providing full REST API coverage with PAT authentication will enable the admin's platform bot to participate in bounties and hackathons on behalf of the platform, because the bot can discover, apply, fork, code, and submit entirely through API calls.

---

### Story A-001: Platform Bot Discovers Open Bounties

- **Summary:** Platform bot finds available bounties through the API

#### Use Case:
- **As an** admin's platform bot (authenticated via PAT)
- **I want to** query the global bounty list filtered by status
- **so that** I can programmatically discover bounties matching my capabilities

#### Acceptance Criteria:

- **Scenario:** Platform bot queries open bounties
- **Given:** I am authenticated with a PAT token
- **When:** I call `GET /api/v1/hackforger/bounties?status=open`
- **Then:** I receive a JSON list of open bounties with repo, issue, reward, and deadline information

**Journey ref:** J6 Step 6.1

---

### Story A-002: Platform Bot Participates in Hackathon (Full Flow)

- **Summary:** Platform bot registers, forks, codes, and submits a hackathon entry entirely via API

#### Use Case:
- **As an** admin's platform bot
- **I want to** register for a hackathon, fork the track repo, push code, and submit a PR as my entry
- **so that** I can compete alongside human participants without any Web UI interaction

#### Acceptance Criteria:

- **Scenario:** Full hackathon participation via API
- **Given:** Hackathon "AI Sprint" is in `Open` status with track "AI Track"
- **and Given:** I am authenticated via PAT
- **When:** I register → fork `ai-sprint/ai-track` → push code → create PR → submit
- **Then:** My registration is created, fork exists, PR is created, and submission is linked to the PR

**Journey ref:** J6 Steps 6.3–6.5

---

### Story A-003: Platform Bot Checks Credits + Reputation

- **Summary:** Platform bot retrieves its own Credits balance and reputation score

#### Use Case:
- **As an** admin's platform bot
- **I want to** check my Credits balance, transaction history, and reputation score
- **so that** I can report my earnings and standing to the admin

#### Acceptance Criteria:

- **Scenario:** Platform bot checks balance
- **Given:** I have earned 50 Credits from a bounty
- **When:** I call `GET /api/v1/hackforger/credits/balance`
- **Then:** I receive `{"balance": 50}`

- **Scenario:** Platform bot checks reputation
- **When:** I call `GET /api/v1/hackforger/reputation/users/platform-bot`
- **Then:** I receive my score and breakdown

**Journey ref:** J6 Steps 6.6–6.8
