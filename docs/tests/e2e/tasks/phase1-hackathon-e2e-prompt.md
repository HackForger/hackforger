# Phase 1 Hackathon — Manual E2E Test Prompt

**Instance:** https://hackforger.inside.h2os.cloud

## Prerequisites

### Start the server (if testing in a worktree)

See [docs/tests/local-testing-guide.md](../local-testing-guide.md) for details.

```bash
# 1. Copy config from main repo (worktrees don't have custom/)
mkdir -p custom/conf
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini

# 2. Build backend (embeds templates + assets)
TAGS="bindata sqlite sqlite_unlock_notify" make build

# 3. Remove stale LevelDB lock if needed
rm -f /Users/h2oslabs/Workspace/hackforger/data/queues/common/LOCK

# 4. Kill any existing server (IMPORTANT: old process uses old binary!)
kill $(lsof -t -i :3000) 2>/dev/null
sleep 2

# 5. Start server
./gitea web

# 6. Verify the binary is the latest build
./gitea --version
# Should show a timestamp matching your build. If it shows an old commit, re-run make build.
```

Access via https://hackforger.inside.h2os.cloud/ (Caddy must be running — see local-testing-guide).

> **Critical:** After `make build`, you MUST kill the old server process before starting the new one. The old process keeps running with the old binary in memory even after the file is overwritten. `lsof -i :3000` to find the PID if needed.

### Test accounts

- Admin: `hackforger` / `admin1234`
- At least 2 additional users: `hacker_eve`, `judge_carol`
- If users don't exist, create them via Site Administration → User Accounts

### Pre-test data check

The database is shared across worktrees. Before testing, verify no leftover hackathon data from a previous run:

```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT id, name, slug, status FROM hackforger_hackathon;"
```

If `spring-hack-2026` already exists, delete it:
```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "DELETE FROM hackforger_hackathon WHERE slug='spring-hack-2026';"
```

---

## Test Steps

### 1. Create Hackathon (as admin)

1. Log in as `hackforger` (admin)
2. Navigate to `/hackathons/new` (or use navbar "+" → 创建黑客松)
3. Fill in:
   - Name: `Spring Hack 2026`
   - Slug: `spring-hack-2026`
   - Description: `Test hackathon for Phase 1 E2E`
   - Org ID: `0`
   - Max Team Size: `5`
4. Submit the form
5. **Verify:** Redirected to `/hackathon/spring-hack-2026` with status "Draft"

### 1a. ★ Verify Auto-Created Organization

1. Navigate to `/spring-hack-2026` (the org page, NOT `/hackathon/spring-hack-2026`)
2. **Verify:** Organization page exists, owner is `hackforger`
3. **Verify:** Hackathon detail page shows "Organization" link pointing to this org

### 2. Add Tracks

1. Navigate to `/hackathon/spring-hack-2026/manage`
2. Add track: `AI Track`
3. Add track: `Web Track`
4. **Verify:** Both tracks appear in the Tracks section

### 2a. ★ Verify Auto-Created Repositories

1. Navigate to `/spring-hack-2026/ai-track` (the repo page)
2. **Verify:** Repository exists with auto-init commit on `main` branch
3. Navigate to `/spring-hack-2026/web-track`
4. **Verify:** Repository exists

### 2b. ★ Verify Auto-Created Milestones

1. Navigate to `/spring-hack-2026/ai-track/milestones`
2. **Verify:** 4 milestones exist: Registration, Hacking, Judging, Results

### 3. Add Judge

1. On the manage page, find `judge_carol`'s user ID (check via Site Administration → User Accounts)
2. Enter the user ID and click "Add Judge"
3. **Verify:** Judge appears in the Judges section

### 4. Publish Hackathon (Draft → Open)

1. On the manage page, click "Publish"
2. **Verify:** Status changes to "Open (Registration)"
3. Navigate to `/explore/hackathons`
4. **Verify:** "Spring Hack 2026" appears in the listing with "Open" badge

### 5. ★ Verify Dashboard Feed — HackathonCreated

1. Navigate to the admin's dashboard (`/`)
2. **Verify:** Activity feed shows "hackforger **created a new hackathon**" (not a blank entry)
3. **Verify:** The event has a rocket icon (not a question mark)

### 6. Register (as hacker_eve)

1. Log out, log in as `hacker_eve`
2. Navigate to `/hackathon/spring-hack-2026`
3. **Verify:** Registration form shows "Participate as" dropdown with "Individual" + user's orgs
4. Select "Individual", leave display name empty (defaults to username)
5. Click "Register"
6. **Verify:** Success flash, register form replaced by "already registered" message

### 6a. ★ Verify Org Membership

1. Navigate to `/spring-hack-2026` (org page) → Members tab
2. **Verify:** `hacker_eve` appears as a member of the hackathon org

### 7. Approve Registration + Start Hacking (as admin)

1. Log out, log in as admin
2. Navigate to `/hackathon/spring-hack-2026/manage`
3. **Verify:** `hacker_eve`'s registration shows as "Pending"
4. Click "Approve"
5. **Verify:** Status changes to "Approved"
6. Click "Start Hacking"
7. **Verify:** Phase control shows "Hacking"

### 7a. ★ Verify v0-kickoff Tag

1. Navigate to `/spring-hack-2026/ai-track/releases`
2. **Verify:** Tag `v0-kickoff` exists with message "Hackathon kickoff baseline"

### 8. Submit Project (as hacker_eve)

1. Log out, log in as `hacker_eve`
2. Navigate to `/hackathon/spring-hack-2026`
3. **Verify:** "Submit Project" button is visible
4. Click "Submit Project"
5. Fill in:
   - Title: `AI Assistant Bot`
   - Description: `An AI-powered assistant`
   - Demo URL: `https://example.com/demo`
   - Track: `AI Track`
6. Submit
7. **Verify:** Success flash, project appears in submissions list

### 8a. ★ Verify Fork + PR

1. Navigate to `/hacker_eve/ai-track-hacker_eve` (or similar fork name)
2. **Verify:** Fork exists (forked from `spring-hack-2026/ai-track`)
3. Navigate to `/spring-hack-2026/ai-track/pulls`
4. **Verify:** PR "AI Assistant Bot" exists, opened by `hacker_eve`

### 9. Start Judging (as admin)

1. Log in as admin
2. Navigate to `/hackathon/spring-hack-2026/manage`
3. Click "Start Judging"
4. **Verify:** Phase control shows "Judging"

### 10. Score Submission (as judge_carol)

1. Log in as `judge_carol`
2. Navigate to `/hackathon/spring-hack-2026/judge`
3. **Verify:** "AI Assistant Bot" submission is listed with score input
4. Score: `8.5`, Comment: `Great work!`
5. Submit
6. **Verify:** Success flash message "Score submitted"

### 11. Finalize (as admin)

1. Log in as admin
2. Navigate to `/hackathon/spring-hack-2026/manage`
3. Click "Finalize"
4. **Verify:** Phase control shows "Finished"

### 11a. ★ Verify submission-deadline Tag + v1-results Release

1. Navigate to `/spring-hack-2026/ai-track/releases`
2. **Verify:** Tag `submission-deadline` exists
3. **Verify:** Release `v1-results` exists with title "Spring Hack 2026 — AI Track Results"
4. **Verify:** Release body contains a markdown table with: #1, AI Assistant Bot, 8.5

### 12. Verify Leaderboard

1. Navigate to `/hackathon/spring-hack-2026/leaderboard`
2. **Verify:** "AI Assistant Bot" is ranked #1 with score 8.5
3. **Verify:** Row has green "positive" highlight

### 13. ★ Verify Dashboard Feed — All Events

1. Log in as admin, go to dashboard (`/`)
2. **Verify the following events appear with proper text (not blank):**
   - "hackforger **created a new hackathon**" (rocket icon)
   - "hackforger **changed hackathon phase**" (rocket icon)
   - "hackforger **finalized a hackathon**" (rocket icon)
3. If admin follows `hacker_eve`, also verify:
   - "hacker_eve **registered for a hackathon**"
   - "hacker_eve **submitted a project to a hackathon**"
4. If admin follows `judge_carol`, also verify:
   - "judge_carol **scored a hackathon submission**"

### 14. ★ Verify Hackathon Detail Page Content

1. Navigate to `/hackathon/spring-hack-2026`
2. **Verify page includes:**
   - Header: name + status badge + organizer link + org link
   - Timeline: dates (if set during creation)
   - Description text
   - Tracks list with "Code" repo link buttons
   - Participants list with approval status badges
   - Submissions table with PR links, scores, ranks
   - Leaderboard link (since status is Finished)
   - Manage button visible (as owner)

### 15. Verify Feed API

```bash
# Global feed — should contain HackathonCreated (30) + HackathonFinalized (51) events
curl -s https://hackforger.inside.h2os.cloud/api/v1/hackforger/feed?type=global \
  | jq '.items[] | {op_type, op_name}'

# List hackathons — should show Spring Hack 2026
curl -s https://hackforger.inside.h2os.cloud/api/v1/hackforger/hackathons \
  | jq '.[].name'

# Leaderboard — should show AI Assistant Bot at rank 1
curl -s https://hackforger.inside.h2os.cloud/api/v1/hackforger/hackathons/1/leaderboard \
  | jq '.'
```

### 16. Verify Cancel Flow

1. Create another hackathon "Cancel Test" with slug `cancel-test`
2. Add a track, publish it
3. On manage page, click "Cancel"
4. **Verify:** Status shows "Cancelled"

### 17. ★ Verify Error Cases

1. **Double registration:** As `hacker_eve`, try registering for `spring-hack-2026` again → should show "你已经报名了该黑客松" (not generic error)
2. **Submit in wrong phase:** Navigate to `/hackathon/spring-hack-2026/submit` (Finished) → should redirect with flash error
3. **Judge in wrong phase:** Navigate to `/hackathon/spring-hack-2026/judge` (Finished) → should redirect with flash error
4. **Non-existent hackathon:** Navigate to `/hackathon/does-not-exist` → should show 404
5. **Publish without tracks:** Create hackathon "No Track Test", try to publish → should show error

---

## Org/Repo Integration Checklist

| Step | Forgejo Entity | Verify At |
|------|---------------|-----------|
| Create Hackathon | Organization created | `/{slug}` org page |
| Add Track | Repository created | `/{slug}/{track-name}` repo page |
| Add Track | 4 Milestones created | `/{slug}/{track-name}/milestones` |
| Register | User added to Org | `/{slug}` org members tab |
| Start Hacking | `v0-kickoff` tag | `/{slug}/{track-name}/releases` |
| Submit | Fork created | `/hacker_eve/{track-name}-hacker_eve` |
| Submit | PR created | `/{slug}/{track-name}/pulls` |
| Finalize | `submission-deadline` tag | `/{slug}/{track-name}/releases` |
| Finalize | `v1-results` release | `/{slug}/{track-name}/releases` |

## Feed Events to Verify

| Event | ActionType | Audience | Dashboard Check |
|-------|-----------|----------|----------------|
| HackathonCreated | 30 | Global (UserID=0) | ★ Must show text, not blank |
| HackathonPhaseChanged | 50 | Org members | ★ Must show text, not blank |
| HackathonRegistered | 31 | Followers of registrant | Shows if following hacker_eve |
| HackathonSubmitted | 32 | Followers of submitter | Shows if following hacker_eve |
| HackathonScored | 33 | Followers of judge | Shows if following judge_carol |
| HackathonFinalized | 51 | Global (UserID=0) | ★ Must show text, not blank |

## Known Limitations

- Track repo link in detail page uses `href="#"` placeholder (repo slug not stored on track struct)
- `AudienceOrgMembers` events now use hackathon's LinkedOrgID (should work since org is auto-created)
- Leaderboard doesn't sort by rank (relies on DB insertion order after finalize)
- PR creation may fail silently if fork has no diff from base repo (auto-init repos are identical)
