# E2E Test: Full User Journey Validation — HackForger v0.1

> **Web 端测试是必选项，不可使用单元测试或 API 测试替代 Web 界面操作。关键验证节点必须使用 agent-browser 截屏记录作为测试证据。**

## Overview
Validate all 6 user journeys through the Web UI. Each journey must be tested end-to-end via browser operations.

Reference documents: `docs/product/user-journeys/`

## Prerequisites
- Clean HackForger instance on http://localhost:3000
- Test users: hackforger (admin), hacker_eve, hacker_frank, judge_carol, judge_dave — all with password admin1234
- Multi-user sessions: `--session admin`, `--session hacker1`, `--session hacker2`, `--session judge1`

## Journey 1: Hackathon Lifecycle (Organizer)
Reference: `docs/product/user-journeys/organizer-hackathon.md`

### TC-J1.1: Create Hackathon
1. Login as hackforger (admin session)
2. Navigate to /hackathons/new
3. Fill form: name, slug, description, max team size, dates
4. Submit
5. **Screenshot**: `docs/tests/e2e/screenshots/week5/j1-hackathon-created.png`
6. **Verify**: Redirected to hackathon detail page, status = Draft

### TC-J1.2: Add Track + Criteria
1. Navigate to hackathon manage page
2. Create a track with name, description, prize info
3. Add judging criteria (Innovation, Completeness, etc.)
4. **Screenshot**: `docs/tests/e2e/screenshots/week5/j1-track-criteria.png`
5. **Verify**: Track and criteria appear on manage page

### TC-J1.3: Publish Hackathon
1. Click Publish button on manage page
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j1-published.png`
3. **Verify**: Status changes to Open, hackathon visible on Explore page

### TC-J1.4: Register as Hacker
1. Switch to hacker_eve session
2. Navigate to hackathon detail → Register
3. Fill registration form
4. **Screenshot**: `docs/tests/e2e/screenshots/week5/j1-registered.png`
5. **Verify**: Registration appears, status Pending or Approved

### TC-J1.5: Start Hacking → Submit Work
1. Admin session: Start hackathon (status → Hacking)
2. Hacker session: Submit work (title, description, demo URL)
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/j1-submission.png`
4. **Verify**: Submission listed on hackathon page

### TC-J1.6: Judging → Score
1. Admin: Assign judge_carol as judge
2. Admin: Transition to Judging (POST /judge)
3. Judge session: Navigate to judge page → Score submissions
4. **Screenshot**: `docs/tests/e2e/screenshots/week5/j1-judging.png`
5. **Verify**: Scores saved, leaderboard shows rankings

### TC-J1.7: Finalize + Credits
1. Admin: Finalize hackathon
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j1-finalized.png`
3. **Verify**: Status = Finished, winners determined, credits deposited
4. Check hacker_eve's credits balance
5. **Screenshot**: `docs/tests/e2e/screenshots/week5/j1-credits.png`

## Journey 2: Bounty Lifecycle (Organizer)
Reference: `docs/product/user-journeys/organizer-bounty.md`

### TC-J2.1: Create Bounty on Issue
1. Admin: Create or navigate to an existing issue in a repo
2. Create Exclusive bounty with reward
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/j2-bounty-created.png`
4. **Verify**: Bounty badge visible on issue page

### TC-J2.2: Hacker Applies → Organizer Accepts
1. Hacker session: Apply for bounty
2. Admin session: Review and accept application
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/j2-application-accepted.png`
4. **Verify**: Bounty status → Claimed

### TC-J2.3: Complete → Pay
1. Admin: Mark bounty as complete
2. Admin: Mark as paid
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/j2-bounty-paid.png`
4. **Verify**: Status = Paid, credits deposited to hacker

### TC-J2.4: Competitive Bounty + Winners
1. Create Competitive bounty with tiered rewards
2. Multiple hackers submit
3. Select winners
4. **Screenshot**: `docs/tests/e2e/screenshots/week5/j2-winners.png`
5. **Verify**: Winners listed with ranks

### TC-J2.5: Cancel + Expiry
1. Create another bounty → Cancel it
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j2-cancelled.png`
3. **Verify**: Status = Cancelled

## Journey 3: Grant Lifecycle (Organizer)
Reference: `docs/product/user-journeys/organizer-grant.md`

### TC-J3.1: Create Grant Round
1. Admin: Navigate to /grants/new
2. Fill form: name, budget, currency, deadline
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/j3-round-created.png`
4. **Verify**: Status = Draft

### TC-J3.2: Open → Submit Project
1. Open round, hacker submits project
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j3-project-submitted.png`

### TC-J3.3: Approve + Allocate
1. Approve project, allocate award amount
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j3-approved.png`

### TC-J3.4: Finalize → Distribute
1. Finalize round, then distribute
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j3-distributed.png`
3. **Verify**: Credits deposited to project owners

## Journey 4: Credits + Redeem (Hacker)
Reference: `docs/product/user-journeys/hacker-credits.md`

### TC-J4.1: View Credits
1. Hacker session: Navigate to /credits
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j4-credits-overview.png`
3. **Verify**: Balance shown, transaction history visible

### TC-J4.2: Redeem
1. Browse redeem options → Select one → Place order
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j4-redeem-order.png`
3. **Verify**: Order created, balance deducted

### TC-J4.3: Admin Fulfill
1. Admin: Navigate to admin credits → Orders → Fulfill
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j4-fulfilled.png`
3. **Verify**: Order status = fulfilled

## Journey 5: Social + Feed (Hacker)
Reference: `docs/product/user-journeys/hacker-social.md`

### TC-J5.1: Follow + Feed
1. Hacker follows organizer user
2. Navigate to Dashboard → Following tab
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/j5-feed-following.png`
4. **Verify**: Feed shows organizer's recent actions

### TC-J5.2: Explore
1. Navigate to Explore → Hackathons/Bounties/Grants tabs
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j5-explore.png`
3. **Verify**: Lists show hackathons, bounties, grant rounds

### TC-J5.3: Search via ⌘K
1. Press ⌘K → Search for known entity
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j5-search.png`
3. **Verify**: Results appear, can navigate to detail

### TC-J5.4: Reputation
1. Navigate to reputation leaderboard
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/j5-reputation.png`
3. **Verify**: Leaderboard shows users with scores

## Journey 6: Platform Bot
Reference: `docs/product/user-journeys/platform-bot.md`

### TC-J6.1: Bot API Access
1. Use API (via curl or hackforger-cli) with bot PAT to list bounties
2. **Verify**: Bot can access public endpoints
3. **Screenshot**: CLI output or API response

### TC-J6.2: Bot Applies for Bounty
1. Bot applies for an open bounty via API
2. **Verify**: Application created successfully

### TC-J6.3: Bot Registers for Hackathon
1. Bot registers for an open hackathon via API
2. **Verify**: Registration created

## Report Template
Save report to: `docs/tests/e2e/week5-full-journey-e2e-report.md`
Include ALL screenshots as embedded images. Each TC must have Pass/Fail status.
