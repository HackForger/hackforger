# Hacker Stories

### Story H-006: Register for Hackathon

- **Summary:** Hacker registers and automatically gains Org membership for Git access

#### Use Case:
- **As a** hacker
- **I want to** register for a hackathon and select a track
- **so that** I can fork the track repo and start building

#### Acceptance Criteria:

- **Scenario:** Hacker registers for an open hackathon
- **Given:** The hackathon is in `Open(1)` status
- **and Given:** I am logged in as `hacker1` and not already registered
- **When:** I register for the hackathon selecting "DeFi Track"
- **Then:** My registration is created with status `Approved(1)`, I am added as a member of the hackathon's Organization, and a `hackathon_registered(31)` feed event is emitted

**Journey ref:** J1 Steps 1.8, 1.9

---

### Story H-007: Submit via Fork + PR

- **Summary:** Hacker submits by creating a PR from their fork — the submission IS the PR

#### Use Case:
- **As a** registered hacker
- **I want to** fork the track repo, develop my project, and submit a PR as my hackathon entry
- **so that** my submission is timestamped, diffable, and reviewable — with full Git history as proof of work

#### Acceptance Criteria:

- **Scenario:** Hacker submits during Hacking phase
- **Given:** The hackathon is in `Hacking(2)` status
- **and Given:** I have forked `web3-innovation/defi-track` to `hacker1/defi-track`
- **and Given:** I have pushed commits to my fork
- **When:** I create a PR from `hacker1:main` to `web3-innovation/defi-track:main` and register it as a hackathon submission
- **Then:** The submission is created linked to the PR, a `hackathon_submitted(32)` feed event is emitted, and the submission appears on the hackathon leaderboard (unscored)

- **Scenario:** Submit after Hacking phase closes
- **Given:** The hackathon is in `Judging(3)` status
- **When:** I attempt to create a submission
- **Then:** The request is rejected — submissions are closed

**Journey ref:** J1 Steps 1.11, 1.14, 1.15

---

### Story G-002: Submit Grant Application

- **Summary:** Hacker submits a project proposal requesting funding

#### Use Case:
- **As a** developer
- **I want to** submit a project proposal with description, requested amount, and repo link
- **so that** the grant organizer can evaluate my project for funding

#### Acceptance Criteria:

- **Scenario:** Hacker submits a grant application
- **Given:** Grant Round #1 is in `Open(1)` status
- **When:** I submit a project "DeFi Testing Framework" requesting 1000 Credits with a link to my repo
- **Then:** My project is created in `Pending` status and a `grant_project_submitted(40)` feed event is emitted

- **Scenario:** Submit after round closes
- **Given:** Grant Round #1 is in `Review(2)` status
- **When:** I attempt to submit a project
- **Then:** The request is rejected — applications are closed

**Journey ref:** J3 Steps 3.3, 3.4

---

### Story C-001: View Balance and Transaction History

- **Summary:** User checks their Credits balance and reviews all transactions

#### Use Case:
- **As a** platform participant
- **I want to** view my Credits balance and a history of all deposits, withdrawals, and redemptions
- **so that** I can track my earnings and spending across hackathons, bounties, and grants

#### Acceptance Criteria:

- **Scenario:** User views balance
- **Given:** I am logged in as `hacker1` with accumulated Credits
- **When:** I view my Credits dashboard
- **Then:** I see my current balance (1950) and a reverse-chronological transaction list showing each deposit source (hackathon prize, bounty reward, grant funding) and any deductions

**Journey ref:** J4 Steps 4.3, 4.4

---

### Story C-002: Redeem Credits

- **Summary:** User exchanges Credits for a reward from the marketplace

#### Use Case:
- **As a** participant with accumulated Credits
- **I want to** browse available rewards and redeem my Credits for one
- **so that** I receive tangible value for my contributions

#### Acceptance Criteria:

- **Scenario:** Successful redemption
- **Given:** I have 1950 Credits and a "GitHub Copilot - 1 Month" option exists costing 200
- **When:** I redeem for that option
- **Then:** 200 Credits are deducted atomically, a `RedeemOrder` is created in `pending` status, a `redeem` transaction is recorded, and a `credits_redeemed(42)` feed event is emitted

- **Scenario:** Insufficient balance
- **Given:** I have 50 Credits
- **When:** I attempt to redeem a 200-Credit option
- **Then:** The request is rejected with an insufficient balance error

**Journey ref:** J4 Steps 4.5, 4.6
