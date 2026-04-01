# Organizer Stories

## Epic 1: Hackathon Lifecycle

> **Epic Hypothesis:** We believe that providing a Git-Native hackathon platform where submissions are PRs and judging happens on code will increase submission integrity and reduce organizer logistics overhead, because the code-to-submission gap is eliminated.

---

### Story H-001: Create Hackathon

- **Summary:** Organizer creates a hackathon that auto-provisions a Git organization

#### Use Case:
- **As a** hackathon organizer
- **I want to** create a new hackathon with name, description, and prize pool
- **so that** I have a fully provisioned competition space without manual Git setup

#### Acceptance Criteria:

- **Scenario:** Organizer creates a hackathon from scratch
- **Given:** I am logged in as `organizer`
- **When:** I submit a new hackathon form with name "Web3 Innovation", slug "web3-innovation", and prize pool 800
- **Then:** A hackathon is created in `Draft(0)` status, a Forgejo Organization `web3-innovation` is auto-created with me as owner, and a `hackathon_created(30)` feed event is emitted

**Journey ref:** J1 Steps 1.2

---

### Story H-002: Create Track (Repo)

- **Summary:** Organizer adds competition tracks that auto-create repos with code templates

#### Use Case:
- **As a** hackathon organizer
- **I want to** create tracks under my hackathon with name, description, and prize allocation
- **so that** participants have dedicated repos to fork and code in

#### Acceptance Criteria:

- **Scenario:** Organizer creates a track within a hackathon
- **Given:** A hackathon "Web3 Innovation" exists in `Draft(0)` status with fewer than 10 tracks
- **When:** I create a track "DeFi Track" with prize allocation 500
- **Then:** A repo `web3-innovation/defi-track` is created with an initial README, the track is linked to the hackathon, and prize allocation total does not exceed the prize pool

- **Scenario:** Track limit enforcement
- **Given:** A hackathon already has 10 tracks
- **When:** I attempt to create an 11th track
- **Then:** The request is rejected with an error indicating the maximum track limit

**Journey ref:** J1 Step 1.3

---

### Story H-003: Set Judging Criteria

- **Summary:** Organizer defines weighted scoring criteria per track

#### Use Case:
- **As a** hackathon organizer
- **I want to** define scoring criteria (name, weight, description) for each track
- **so that** judges evaluate submissions consistently on dimensions I care about

#### Acceptance Criteria:

- **Scenario:** Organizer sets criteria for a track
- **Given:** A track "DeFi Track" exists under the hackathon
- **When:** I set 3 criteria: Innovation (40%), Technical Quality (35%), Presentation (25%)
- **Then:** The criteria are saved with weights summing to 100, and judges will see these criteria when scoring

**Journey ref:** J1 Step 1.4

---

### Story H-004: Assign Judges

- **Summary:** Organizer adds judges who cannot also be participants

#### Use Case:
- **As a** hackathon organizer
- **I want to** assign users as judges for my hackathon
- **so that** qualified reviewers can evaluate submissions

#### Acceptance Criteria:

- **Scenario:** Organizer assigns a judge
- **Given:** A hackathon exists and user `judge1` is not registered as a participant
- **When:** I add `judge1` as a judge
- **Then:** `judge1` is added to the judge list and is prevented from registering as a participant

- **Scenario:** Conflict — user is already a participant
- **Given:** `hacker1` is already registered for the hackathon
- **When:** I attempt to add `hacker1` as a judge
- **Then:** The request is rejected with a conflict error

**Journey ref:** J1 Step 1.5

---

### Story H-005: Publish Hackathon (Draft → Open)

- **Summary:** Organizer publishes hackathon to open registration

#### Use Case:
- **As a** hackathon organizer
- **I want to** publish my hackathon when tracks and judges are ready
- **so that** participants can discover and register for the event

#### Acceptance Criteria:

- **Scenario:** Organizer publishes a ready hackathon
- **Given:** The hackathon is in `Draft(0)` status with at least 1 track and 1 judge
- **When:** I publish the hackathon
- **Then:** Status transitions to `Open(1)`, the hackathon appears on the Explore page, and a `hackathon_phase_changed(50)` feed event is emitted

- **Scenario:** Publish without prerequisites
- **Given:** The hackathon has 0 tracks
- **When:** I attempt to publish
- **Then:** The request is rejected indicating tracks are required

**Journey ref:** J1 Step 1.6

---

### Story H-009: Finalize Hackathon + Auto-Distribute Prizes

- **Summary:** Organizer finalizes the hackathon, locking rankings and distributing Credits

#### Use Case:
- **As a** hackathon organizer
- **I want to** finalize the hackathon after all judges have scored
- **so that** rankings are locked, winners are announced, and prize Credits are automatically distributed

#### Acceptance Criteria:

- **Scenario:** Organizer finalizes a fully-judged hackathon
- **Given:** The hackathon is in `Judging(3)` status and all judges have scored all submissions
- **When:** I finalize the hackathon
- **Then:** Status transitions to `Finished(4)`, rankings are computed and frozen, Credits are deposited to winners (1st=500, 2nd=300) and withdrawn from organizer, leaderboard is published, and a `hackathon_finalized(51)` feed event is emitted

**Journey ref:** J1 Steps 1.19, 1.20

---

### Story H-010: Phase Transitions (Open → Hacking → Judging)

- **Summary:** Organizer advances hackathon through competition phases

#### Use Case:
- **As a** hackathon organizer
- **I want to** transition the hackathon from Open to Hacking, and from Hacking to Judging
- **so that** each phase has clear boundaries (registration closes, then submissions close)

#### Acceptance Criteria:

- **Scenario:** Start Hacking phase
- **Given:** The hackathon is in `Open(1)` status
- **When:** I start the hackathon
- **Then:** Status transitions to `Hacking(2)`, registered hackers can now fork and submit, and a `hackathon_phase_changed(50)` feed event is emitted

- **Scenario:** Start Judging phase
- **Given:** The hackathon is in `Hacking(2)` status
- **When:** I transition to judging via `POST /judge`
- **Then:** Status transitions to `Judging(3)`, no new submissions accepted, judges can now score, and a `hackathon_phase_changed(50)` feed event is emitted

**Journey ref:** J1 Steps 1.10, 1.16

---

## Epic 3: Grant Lifecycle

> **Epic Hypothesis:** We believe that a two-phase grant model (Finalized = committed, Distributed = funds released after review) will increase grant accountability, because organizers can verify project progress before releasing funds.

---

### Story G-001: Create and Open Grant Round

- **Summary:** Organizer creates a grant round with budget and opens it for applications

#### Use Case:
- **As a** grant organizer
- **I want to** create a grant round with a budget and deadline, then open it for applications
- **so that** developers can submit project proposals for funding

#### Acceptance Criteria:

- **Scenario:** Create and open a grant round
- **Given:** I am logged in as `organizer` with sufficient Credits
- **When:** I create a grant round "Q2 2026 Open Source Grant" with budget 2000, then open it
- **Then:** Round is created in `Draft(0)`, transitions to `Open(1)` on publish, `grant_round_created(39)` and `grant_round_opened(54)` feed events are emitted, and the round appears on the Explore page

**Journey ref:** J3 Steps 3.1, 3.2

---

### Story G-003: Review and Approve/Reject Projects

- **Summary:** Organizer reviews applications and approves or rejects with allocated amounts

#### Use Case:
- **As a** grant organizer
- **I want to** review project applications and approve them with specific funding amounts (or reject)
- **so that** I can allocate my budget to the most impactful projects

#### Acceptance Criteria:

- **Scenario:** Approve a project within budget
- **Given:** Grant Round #1 is in `Review(2)` and project "DeFi Testing Framework" is `Pending`
- **When:** I approve the project with 1000 Credits allocated
- **Then:** Project status becomes `Approved`, awarded amount is recorded, total allocated (1000) ≤ budget (2000), and a `grant_awarded(41)` feed event is emitted

- **Scenario:** Reject a project
- **Given:** Project "NFT Metadata Library" is `Pending` with a 1500 request
- **When:** I reject it with comment "Over remaining budget"
- **Then:** Project status becomes `Rejected`

**Journey ref:** J3 Step 3.6

---

### Story G-004: Finalize Round (Lock Allocations)

- **Summary:** Organizer finalizes the round, locking all allocations without distributing funds

#### Use Case:
- **As a** grant organizer
- **I want to** finalize the round to lock allocations
- **so that** approved projects know their funding is committed, but I retain the ability to review progress before releasing funds

#### Acceptance Criteria:

- **Scenario:** Finalize locks allocations
- **Given:** Grant Round #1 is in `Review(2)` with approved projects
- **When:** I finalize the round
- **Then:** Status transitions to `Finalized(3)`, no further allocation changes allowed, Credits are NOT yet distributed, and a `grant_round_finalized(56)` feed event is emitted

**Journey ref:** J3 Step 3.7

---

### Story G-005: Distribute Funds (Two-Phase Release)

- **Summary:** Organizer reviews project progress and manually releases funds

#### Use Case:
- **As a** grant organizer
- **I want to** review a funded project's progress (commits, PRs, milestones) and then release the allocated Credits
- **so that** funds go to projects that demonstrate actual work, not just proposals

#### Acceptance Criteria:

- **Scenario:** Organizer distributes after reviewing progress
- **Given:** Grant Round #1 is in `Finalized(3)` and I have reviewed hacker1's repo activity
- **When:** I trigger distribution via `POST /grants/rounds/{id}/distribute`
- **Then:** Status transitions to `Distributed(4)`, Credits are deposited to funded project owners and withdrawn from my account, project status becomes `Funded`, and the round is complete

**Journey ref:** J3 Step 3.8

---

## Epic 2: Bounty Lifecycle

> **Epic Hypothesis:** We believe that mounting bounties directly on Git Issues with Credits escrow will increase bounty completion rates, because the entire workflow (discover → claim → code → deliver → get paid) happens in the forge without context switching.

---

### Story B-001: Create Bounty on Issue (Exclusive)

- **Summary:** Organizer attaches a bounty to an Issue with Credits escrowed from their balance

#### Use Case:
- **As an** organizer
- **I want to** create an Exclusive bounty on an existing Issue with a Credits reward
- **so that** I can incentivize a developer to solve the Issue, with funds held in escrow until completion

#### Acceptance Criteria:

- **Scenario:** Organizer creates an Exclusive bounty
- **Given:** I own `organizer/oss-project` and Issue #1 exists
- **and Given:** I have sufficient Credits balance
- **When:** I create an Exclusive bounty on Issue #1 and add a reward of 300 Credits
- **Then:** The bounty is created in `Open(0)` status with mode `Exclusive(0)`, 300 Credits are escrowed from my balance (transaction type `escrow`), the bounty is linked 1:1 to Issue #1, and a `bounty_created(34)` feed event is emitted

**Journey ref:** J2a Steps 2a.1–2a.3

---

### Story B-002: Apply and Claim (Exclusive)

- **Summary:** Hacker applies to a bounty, organizer reviews, and accepted hacker becomes the exclusive claimant

#### Use Case:
- **As a** hacker
- **I want to** apply to an Exclusive bounty with a proposal
- **so that** the organizer can evaluate my qualifications before I commit to the work

#### Acceptance Criteria:

- **Scenario:** Hacker applies and is accepted
- **Given:** Bounty #1 is in `Open(0)` status, mode `Exclusive`
- **When:** I submit an application with my proposal
- **Then:** My application is created in `Pending(0)` status

- **Scenario:** Organizer accepts application
- **Given:** An application from `hacker1` is in `Pending(0)` status
- **When:** The organizer accepts it
- **Then:** Application status becomes `Accepted(1)`, bounty status transitions to `Claimed(1)`, and a `bounty_claimed(35)` feed event is emitted

- **Scenario:** Second application on claimed exclusive bounty is rejected
- **Given:** Bounty #1 is already `Claimed(1)` by hacker1
- **When:** `hacker2` applies
- **Then:** The application is rejected because the bounty is already claimed

**Journey ref:** J2a Steps 2a.4, 2a.5; J2c Step 2c.3

---

### Story B-003: Deliver via PR + Complete + Pay (Exclusive)

- **Summary:** Hacker delivers work via PR, organizer completes and pays, Credits released from escrow

#### Use Case:
- **As a** hacker who claimed a bounty
- **I want to** submit my work as a PR and get paid upon organizer approval
- **so that** I receive Credits for completed work through a transparent code review process

#### Acceptance Criteria:

- **Scenario:** Full delivery-to-payment flow
- **Given:** Bounty #1 is `Claimed(1)` by me
- **When:** I create a PR from my fork to `organizer/oss-project`
- **Then:** Bounty transitions to `InReview(2)` and a `bounty_delivered(36)` feed event is emitted

- **Scenario:** Organizer completes and pays
- **Given:** Bounty #1 is in `InReview(2)`
- **When:** Organizer marks bounty as complete, then triggers payment
- **Then:** Bounty transitions `InReview(2)` → `Completed(3)` → `Paid(4)`, escrowed Credits are released to me via `escrow_release` transaction, and `bounty_completed(37)` + `bounty_paid(43)` feed events are emitted

**Journey ref:** J2a Steps 2a.6–2a.9

---

### Story B-004: Competitive Bounty with Multiple Winners

- **Summary:** Organizer creates a bounty where multiple participants compete and top entries win ranked prizes

#### Use Case:
- **As an** organizer
- **I want to** create a Competitive bounty with 1st/2nd/3rd place rewards
- **so that** I get multiple solutions to choose from and reward the best ones

#### Acceptance Criteria:

- **Scenario:** Competitive bounty lifecycle
- **Given:** I have created a Competitive bounty on Issue #2 with rewards: 1st=150, 2nd=100, 3rd=50 Credits (all escrowed)
- **and Given:** 3 participants (`hacker1`, `hacker2`, `platform-bot`) have submitted applications (auto-accepted in Competitive mode) and PRs
- **When:** I select winners: hacker1=1st, hacker2=2nd, platform-bot=3rd
- **Then:** Bounty transitions to `Completed(3)`, a `bounty_winners_selected(38)` feed event is emitted

- **Scenario:** Pay winners
- **Given:** Winners are selected
- **When:** I trigger payment
- **Then:** Escrowed Credits are distributed: 150 to hacker1, 100 to hacker2, 50 to platform-bot (via `escrow_release` transactions), bounty transitions to `Paid(4)`, and `bounty_paid(43)` is emitted

**Journey ref:** J2b Steps 2b.1–2b.6

---

### Story B-005: Bounty Expiry + Refund

- **Summary:** Uncompleted bounties expire and escrowed Credits are refunded to organizer

#### Use Case:
- **As a** bounty organizer
- **I want** expired bounties to automatically refund my escrowed Credits
- **so that** my funds aren't locked indefinitely when no one delivers

#### Acceptance Criteria:

- **Scenario:** Bounty expires with no completion
- **Given:** Bounty #3 is in `Open(0)` with a deadline that has passed and no accepted claimant
- **When:** The expiry is triggered (cron or manual)
- **Then:** Bounty transitions to `Expired(5)`, escrowed Credits are refunded to my account via `escrow_refund` transaction, and a `bounty_expired(52)` feed event is emitted

**Journey ref:** J2c Step 2c.1

---

### Story B-006: Bounty Cancellation + Refund

- **Summary:** Organizer cancels a bounty and gets escrowed Credits refunded

#### Use Case:
- **As a** bounty organizer
- **I want to** cancel a bounty I no longer need
- **so that** my escrowed Credits are returned to my balance

#### Acceptance Criteria:

- **Scenario:** Organizer cancels an open bounty
- **Given:** Bounty #4 is in `Open(0)` status
- **When:** I cancel the bounty
- **Then:** Bounty transitions to `Cancelled(6)`, escrowed Credits are refunded via `escrow_refund`, and a `bounty_cancelled(53)` feed event is emitted

**Journey ref:** J2c Step 2c.2
