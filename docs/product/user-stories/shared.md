# Shared Stories (Feed + Social + Discovery)

## Epic 5: Feed + Social + Discovery

> **Epic Hypothesis:** We believe that a unified feed showing hackathon, bounty, and grant activity alongside reputation scoring will increase repeat participation, because users can see the community's activity and their own standing.

---

### Story F-001: Dashboard Feed (Following + Global)

- **Summary:** User sees a community activity feed filtered by followed users or all activity

#### Use Case:
- **As a** logged-in user
- **I want to** see a feed of HackForger events from people I follow and from the platform globally
- **so that** I stay informed about hackathons, bounties, and grants without checking each separately

#### Acceptance Criteria:

- **Scenario:** Following feed
- **Given:** I follow `organizer` and `hacker2`
- **When:** I view my Dashboard Community tab (Following)
- **Then:** I see events from organizer (hackathon created, phase changes) and hacker2 (submissions, bounty activity), plus global events, in reverse chronological order with pagination

- **Scenario:** Global feed
- **When:** I switch to the Global feed tab
- **Then:** I see all HackForger events regardless of follow relationships

**Journey ref:** J5 Steps 5.1, 5.2

---

### Story F-002: Explore Pages (Hackathons, Bounties, Grants)

- **Summary:** Users discover hackathons, bounties, and grant rounds through filterable listing pages

#### Use Case:
- **As a** developer looking for opportunities
- **I want to** browse and filter hackathons, bounties, and grant rounds on Explore pages
- **so that** I can find competitions and funding opportunities that match my skills

#### Acceptance Criteria:

- **Scenario:** Explore hackathons
- **Given:** I am on the Explore page
- **When:** I navigate to Hackathons
- **Then:** I see a list of hackathons with status, participant count, track count, and prize pool

- **Scenario:** Explore bounties
- **When:** I navigate to Bounties
- **Then:** I see bounties across all repos, filterable by status and mode

- **Scenario:** Explore grants
- **When:** I navigate to Grants
- **Then:** I see grant rounds with status, budget, and funded project count

**Journey ref:** J5 Steps 5.3, 5.4, 5.5

---

### Story F-003: Cross-Entity Search

- **Summary:** User searches across hackathons, bounties, grants, and submissions with scope filtering

#### Use Case:
- **As a** user
- **I want to** search for keywords across all HackForger entities or within a specific scope
- **so that** I can quickly find relevant opportunities or past projects

#### Acceptance Criteria:

- **Scenario:** Search across all entities
- **Given:** I am on the search page
- **When:** I search for "DeFi" with scope "all"
- **Then:** Results include matching tracks, submissions, and grant projects, grouped by entity type

- **Scenario:** Scoped search
- **When:** I search for "OAuth" with scope "bounties"
- **Then:** Only bounties matching "OAuth" are returned

**Journey ref:** J5 Steps 5.6, 5.7

---

### Story F-004: Reputation Leaderboard + Profile

- **Summary:** Users view a reputation leaderboard and individual scores based on platform activity

#### Use Case:
- **As a** platform participant
- **I want to** see a reputation leaderboard and my own score breakdown
- **so that** I can gauge my standing in the community and showcase my contributions

#### Acceptance Criteria:

- **Scenario:** View leaderboard
- **When:** I view the reputation leaderboard
- **Then:** Users are ranked by weighted score (hackathon wins, bounty completions, grant funding), with tier badges

- **Scenario:** View individual reputation
- **When:** I view `hacker1`'s reputation profile
- **Then:** I see total score, tier, and breakdown by module (hackathon, bounty, grant)

**Journey ref:** J5 Steps 5.8, 5.9
