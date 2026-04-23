# E2E user stories → API mapping

Audit of every step in `docs/tests/e2e/tasks/full-cycle/00-08-*.md` cross-referenced against `/api/v1/hackforger/*` (and Forgejo standard endpoints where the operation is generic).

**Coverage status:**
- ✅ — equivalent API endpoint exists and works
- ⚠️ — partial coverage, see notes
- ❌ — no API equivalent (see [gaps.md](gaps.md))
- 🤖 — operation is automatic (cron/scheduler), no API call needed
- 📦 — handled by Forgejo standard endpoints (not HackForger-specific)

| Phase | Step | Action | API call | Status |
|-------|------|--------|----------|--------|
| **0 Setup** | 0.1 | Admin creates redeem option | `POST /hackforger/credits/redeem/options` | ✅ |
| 0 | 0.2 | Admin adds API keys to pool | `POST /hackforger/credits/redeem/options/{id}/keys` | ✅ |
| 0 | 0.3 | Admin deposits credits to user | `POST /hackforger/credits/admin/deposit` | ✅ |
| **1 Hackathon setup** | 1.1 | Organizer creates hackathon (Draft) | `POST /hackforger/hackathons` | ✅ |
| 1 | 1.2 | Organizer creates 2 tracks | `POST /hackforger/hackathons/{id}/tracks` ×2 | ✅ |
| 1 | 1.3 | Organizer adds 3 review criteria per track | `POST /hackforger/hackathons/{id}/criteria` (+ track overrides) | ✅ |
| 1 | 1.4 | Organizer assigns judges | `POST /hackforger/hackathons/{id}/judges` | ✅ |
| 1 | 1.5 | Organizer publishes (Draft → Open) | `POST /hackforger/hackathons/{id}/publish` | ✅ |
| **2 Registration** | 2.1 | Hacker follows organizer | `PUT /user/following/{username}` | 📦 |
| 2 | 2.2 | Hacker registers for Web Track | `POST /hackforger/hackathons/{id}/register` | ✅ |
| 2 | 2.3 | Hacker registers for DeFi Track | `POST /hackforger/hackathons/{id}/register` | ✅ |
| 2 | 2.4 | Mutual follow (eve ↔ frank) | `PUT /user/following/{username}` | 📦 |
| 2 | 2.5 | Organizer creates org team | `POST /orgs/{org}/teams` | 📦 |
| 2 | 2.6 | Judge tries to register (rejected) | `POST /hackforger/hackathons/{id}/register` | ✅ |
| **3 Hacking** | 3.1 | Phase advance Open → Hacking | (cron `hackforger_hackathon_status` @ phase boundary) | 🤖 |
| 3 | 3.2 | Duplicate publish (rejected) | `POST /hackforger/hackathons/{id}/publish` | ✅ |
| **4 Bounty** | 4.1 | Hacker forks repo + commit | `POST /repos/{owner}/{repo}/forks`, `POST /repos/{owner}/{repo}/contents/{path}` | 📦 |
| 4 | 4.2 | Hacker creates issue | `POST /repos/{owner}/{repo}/issues` | 📦 |
| 4 | 4.3 | Hacker creates exclusive bounty | `POST /repos/{owner}/{repo}/bounties` | ✅ |
| 4 | 4.4 | Admin deposits credits (escrow support) | `POST /hackforger/credits/admin/deposit` | ✅ |
| 4 | 4.5 | Other hacker applies for bounty | `POST /repos/{owner}/{repo}/bounties/{id}/applications` | ✅ |
| 4 | 4.6 | Owner accepts application (Open → Claimed) | `PUT /repos/{owner}/{repo}/bounties/{id}/applications/{aid}` | ✅ |
| 4 | 4.7 | Hacker forks + delivers files | `POST /repos/{owner}/{repo}/contents/{path}` | 📦 |
| 4 | 4.8 | Hacker opens PR | `POST /repos/{owner}/{repo}/pulls` | 📦 |
| 4 | 4.9 | Owner reviews + completes + pays (InReview → Completed → Paid) | `POST /bounties/{id}/review`, `/complete`, `/pay` | ✅ |
| 4 | 4.10 | View balance | `GET /hackforger/credits/balance` | ✅ |
| **5 Grant** | 5.1 | Admin deposits Grant budget | `POST /hackforger/credits/admin/deposit` | ✅ |
| 5 | 5.2 | Organizer creates Grant Round (Draft) | `POST /hackforger/grants` | ✅ |
| 5 | 5.3 | Organizer opens round (Draft → Open) | `POST /hackforger/grants/{slug}/open` | ✅ |
| 5 | 5.4 | Hacker submits project A | `POST /hackforger/grants/{slug}/projects` | ✅ |
| 5 | 5.5 | Hacker submits project B | `POST /hackforger/grants/{slug}/projects` | ✅ |
| 5 | 5.6 | Organizer closes (Open → Review) | `POST /hackforger/grants/{slug}/close` | ✅ |
| 5 | 5.7 | Organizer approves project A, rejects B + sets award | `PUT /grants/{slug}/projects/{pid}` + `PUT /award` | ✅ |
| 5 | 5.8 | Organizer finalizes + distributes | `POST /finalize` + `POST /distribute` | ✅ |
| 5 | 5.9 | Hacker views balance | `GET /hackforger/credits/balance` | ✅ |
| **6 Submission** | 6.1 | Hacker continues development (commit) | `POST /repos/{owner}/{repo}/contents/{path}` | 📦 |
| 6 | 6.2 | Hacker submits via Fork+PR mode | `POST /hackforger/hackathons/{id}/submissions` | ✅ |
| 6 | 6.3 | Hacker forks DeFi track repo | `POST /repos/{owner}/{repo}/forks` | 📦 |
| 6 | 6.4 | Hacker submits via Link Repo mode | `POST /hackforger/hackathons/{id}/submissions` | ✅ |
| 6 | 6.5 | Star track repos | `PUT /user/starred/{owner}/{repo}` | 📦 |
| **7 Judging** | 7.1 | Phase advance Hacking → Judging | (cron @ phase boundary) | 🤖 |
| 7 | 7.2 | Judge1 scores all submissions | `POST /hackforger/hackathons/{id}/submissions/{sid}/score` | ✅ |
| 7 | 7.3 | Judge2 scores all submissions | `POST /hackforger/hackathons/{id}/submissions/{sid}/score` | ✅ |
| 7 | 7.4 | Organizer views finalize preview | `GET /hackforger/hackathons/{id}/finalize-preview` | ✅ |
| 7 | 7.5 | Organizer confirms settlement (Judging → Finished) | `POST /hackforger/hackathons/{id}/finalize` | ✅ |
| 7 | 7.6 | Hacker views balance change | `GET /hackforger/credits/balance` + `/transactions` | ✅ |
| 7 | 7.7 | Hacker views balance change | `GET /hackforger/credits/balance` | ✅ |
| 7 | 7.8 | View final leaderboard | `GET /hackforger/hackathons/{id}/leaderboard` | ✅ |
| **8 Credits** | 8.1 | View credits + transaction history | `GET /balance` + `/transactions` | ✅ |
| 8 | 8.2 | Browse redeem options | `GET /hackforger/credits/redeem/options` | ✅ |
| 8 | 8.3 | Redeem GPU power (200 credits) | `POST /hackforger/credits/redeem` | ✅ |
| 8 | 8.4 | Admin fulfills order (assigns key) | `POST /hackforger/credits/redeem/orders/{oid}/fulfill` | ✅ |
| 8 | 8.5 | Hacker views fulfilled order + key | `GET /hackforger/credits/redeem/orders` | ✅ |
| 8 | 8.6 | Admin manually deposits 500 | `POST /hackforger/credits/admin/deposit` | ✅ |
| 8 | 8.7 | Admin manually deducts 50 | `POST /hackforger/credits/admin/deduct` | ✅ |
| **9 Social/Feed** | 9.1 | View Following feed | `GET /hackforger/feed?scope=following` | ✅ |
| 9 | 9.2 | View Global feed | `GET /hackforger/feed?scope=global` | ✅ |
| 9 | 9.3 | Explore Hackathons | `GET /hackforger/hackathons` | ✅ |
| 9 | 9.4 | Explore Bounties | `GET /hackforger/bounties` | ✅ |
| 9 | 9.5 | Explore Grants | `GET /hackforger/grants` | ✅ |
| 9 | 9.6 | Explore Submissions | `GET /hackforger/hackathons/{id}/submissions` (per-hackathon) | ✅ |
| 9 | 9.7 | Global search (Cmd+K) | `GET /hackforger/search?q=...` | ✅ |
| 9 | 9.8 | View Reputation leaderboard | `GET /hackforger/reputation/leaderboard` | ✅ |
| 9 | 9.9 | Add reaction to PR | `POST /repos/{owner}/{repo}/issues/{index}/reactions` | 📦 |
| **10 Edge** | 10a.1 | Create bounty with past deadline | `POST /repos/{owner}/{repo}/bounties` | ✅ |
| 10 | 10a.2 | System triggers expiry | (cron `hackforger_bounty_expiry` @ deadline) | 🤖 |
| 10 | 10b.1 | Cancel bounty (refund) | `POST /repos/{owner}/{repo}/bounties/{id}/cancel` | ✅ |
| 10 | 10c.1 | Create competitive bounty (multi-tier) | `POST /bounties` + `POST /rewards` ×N | ✅ |
| 10 | 10c.2 | Multiple users apply | `POST /applications` ×N | ✅ |
| 10 | 10c.3 | Select winners + pay | `POST /winners` + `POST /pay` | ✅ |
| 10 | 10d.1 | Try to add participant as judge (rejected) | `POST /hackforger/hackathons/{id}/judges` | ✅ |
| 10 | 10d.2 | Duplicate registration (rejected) | `POST /hackforger/hackathons/{id}/register` | ✅ |
| 10 | 10d.3 | Award > grant budget (rejected) | `PUT /grants/{slug}/projects/{pid}/award` | ✅ |
| 10 | 10d.4 | Non-owner accesses manage page (403) | (any management endpoint) | ✅ |
| 10 | 10d.5 | Apply to claimed exclusive bounty (rejected) | `POST /applications` | ✅ |
| **Cross** | C.1 | Verify credits ledger consistency | `GET /transactions` for each user | ✅ |
| Cross | C.2 | Verify all Feed events present | `GET /feed?scope=global` | ✅ |

## Result

**Every E2E user-story step has an API path.** Coverage breakdown:
- ✅ HackForger-native API: 53 steps
- 📦 Forgejo standard API (git/follow/star/reactions): 10 steps
- 🤖 Automatic (cron-driven, no API call): 3 steps (3.1, 7.1, 10a.2)

There are zero E2E steps that require a web-only operation, **provided phase transitions are tested by setting short phase durations and waiting for the cron**, rather than expecting a manual "advance phase" button (see [gaps.md](gaps.md) for the dead-route situation).

## Note on E2E policy

Per `feedback_e2e_web_mandatory`: E2E tests still go through the web UI via `agent-browser`. This mapping is for skill consumers (scripts, CI, dev fixtures) — not for replacing E2E.
