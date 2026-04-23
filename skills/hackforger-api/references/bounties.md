# Bounties API

Bounties are scoped to a repo. Base path: `/api/v1/repos/{owner}/{repo}/bounties`.

## Lifecycle

State machine: **Open → Claimed → InReview → Completed → Paid** (or Cancelled / Expired)

For competitive bounties: **Open → multiple applications → Winners selected → Paid**

## CRUD

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/repos/{owner}/{repo}/bounties` | List bounties on a repo |
| POST | `/repos/{owner}/{repo}/bounties` | Create bounty (escrow charged immediately) |
| GET  | `/repos/{owner}/{repo}/bounties/{id}` | Get bounty |
| PUT  | `/repos/{owner}/{repo}/bounties/{id}` | Update (Open only) |
| DELETE | `/repos/{owner}/{repo}/bounties/{id}` | Delete (Open only) |

## Rewards (competitive mode)

| Verb | Path |
|------|------|
| GET  | `/repos/{owner}/{repo}/bounties/{id}/rewards` |
| POST | `/repos/{owner}/{repo}/bounties/{id}/rewards` |
| DELETE | `/repos/{owner}/{repo}/bounties/{id}/rewards/{reward_id}` |

## Applications

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/repos/{owner}/{repo}/bounties/{id}/applications` | List applications |
| POST | `/repos/{owner}/{repo}/bounties/{id}/applications` | Hacker applies |
| PUT  | `/repos/{owner}/{repo}/bounties/{id}/applications/{application_id}` | Owner accepts/rejects (`{"action":"accept"}` → Open → Claimed) |

## State transitions (owner actions)

| Verb | Path | Transition |
|------|------|------------|
| POST | `/repos/{owner}/{repo}/bounties/{id}/review`  | Claimed → InReview (after delivery) |
| POST | `/repos/{owner}/{repo}/bounties/{id}/complete` | InReview → Completed |
| POST | `/repos/{owner}/{repo}/bounties/{id}/reject`   | InReview → Claimed (asks for redo) |
| POST | `/repos/{owner}/{repo}/bounties/{id}/pay`      | Completed → Paid (releases escrow) |
| POST | `/repos/{owner}/{repo}/bounties/{id}/cancel`   | Open/Claimed → Cancelled (refunds escrow) |
| POST | `/repos/{owner}/{repo}/bounties/{id}/expire`   | Triggered by cron at deadline |

## Competitive: select winners

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/repos/{owner}/{repo}/bounties/{id}/winners` | Select winners (`{"winners":[{"user_id":N,"reward_id":M}]}`) |
| GET  | `/repos/{owner}/{repo}/bounties/{id}/winners` | List winners |

After winners selected, call `/pay` to release multi-tier rewards.

## Cross-repo discovery

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/bounties` | All bounties across the platform (with filters) |
| GET  | `/hackforger/bounties/stats` | Aggregate stats |
| GET  | `/hackforger/bounties/leaderboard` | Hunter leaderboard |

## Common chains

**Exclusive bounty (1 winner):**
1. POST `/issues` (Forgejo stdlib — create issue first)
2. POST `/bounties` `{"issue_id":..., "mode":"exclusive", "reward":100, "deadline":"..."}`
3. POST `/applications` (hacker applies)
4. PUT `/applications/{aid}` `{"action":"accept"}` (Open → Claimed)
5. (Hacker forks, commits, opens PR)
6. POST `/review` (PR ready)
7. POST `/complete` then POST `/pay` (releases 100 credits to hacker)

**Competitive bounty (multi-tier):**
1. POST `/bounties` `{"mode":"competitive"}`
2. POST `/rewards` `{"rank":1,"amount":150}` and `{"rank":2,"amount":100}`
3. Multiple POST `/applications`
4. POST `/winners` `{"winners":[{"user_id":U1,"reward_id":R1},{"user_id":U2,"reward_id":R2}]}`
5. POST `/pay`
