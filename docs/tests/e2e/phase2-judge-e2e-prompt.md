# Phase 2 Judge System — Manual E2E Test Prompt

**Instance:** https://hackforger.inside.h2os.cloud

## Prerequisites

### Start the server (if testing in a worktree)

See [docs/tests/local-testing-guide.md](../local-testing-guide.md) for details.

```bash
# 1. Copy config from main repo (worktrees don't have custom/)
mkdir -p custom/conf
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini

# 2. Build backend + frontend
TAGS="bindata sqlite sqlite_unlock_notify" make build
make frontend

# 3. Remove stale LevelDB lock if needed
rm -f /Users/h2oslabs/Workspace/hackforger/data/queues/common/LOCK

# 4. Kill any existing server
kill $(lsof -t -i :3000) 2>/dev/null
sleep 2

# 5. Start server
./gitea web
```

Access via https://hackforger.inside.h2os.cloud/

### Test accounts

- Admin/Organizer: `hackforger` / `admin1234`
- Judge 1: `judge_carol` (create if needed)
- Judge 2: `judge_dave` (create if needed)
- Hacker: `hacker_eve` (create if needed)
- Hacker 2: `hacker_frank` (create if needed)

### Pre-test data check

```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "SELECT id, name, slug, status FROM hackforger_hackathon;"
```

Clean up any leftover data if needed.

---

## Test Cases

### TC-01: Create Hackathon + Define Scoring Criteria

**As:** `hackforger` (admin)

1. Navigate to `/hackathons/new`
2. Create hackathon:
   - Name: "Phase2 Judge Test"
   - Slug: `p2-judge-test`
   - Description: "Testing multi-criteria scoring"
   - Max team size: 3
3. Navigate to `/hackathon/p2-judge-test/manage`
4. Add scoring criteria:
   - Criterion 1: Name="Innovation", Description="Novelty of the idea", MaxScore=10, Weight=30, SortOrder=1
   - Criterion 2: Name="Technical Execution", Description="Code quality and completeness", MaxScore=10, Weight=40, SortOrder=2
   - Criterion 3: Name="Presentation", Description="Demo quality and clarity", MaxScore=10, Weight=15, SortOrder=3
   - Criterion 4: Name="Impact", Description="Potential real-world impact", MaxScore=10, Weight=15, SortOrder=4

**Expected:**
- [ ] Hackathon created in Draft status
- [ ] 4 criteria appear in manage page with correct weights
- [ ] Criteria can be edited and reordered
- [ ] Criteria can be deleted (only in Draft/Open)

### TC-02: Create Tracks + Verify Track Criteria Auto-Seed

**As:** `hackforger` (admin)

1. On manage page, create Track 1: Name="AI Track", Description="AI/ML projects"
2. Create Track 2: Name="Web Track", Description="Web applications"
3. Verify track criteria overrides section shows both tracks with all 4 criteria enabled

**Expected:**
- [ ] Both tracks created with repos
- [ ] Track criteria auto-seeded: all 4 criteria enabled for both tracks
- [ ] Default weights shown (use hackathon-level defaults)

### TC-03: Configure Per-Track Criteria Overrides

**As:** `hackforger` (admin)

1. For AI Track: disable "Presentation" criterion, set weight of "Technical Execution" to 50
2. For Web Track: keep all criteria enabled, set weight of "Presentation" to 30

**Expected:**
- [ ] AI Track shows 3 enabled criteria (Innovation, Technical, Impact)
- [ ] AI Track: Technical Execution weight=50, Innovation weight=30, Impact weight=15
- [ ] Web Track shows 4 enabled criteria with custom Presentation weight=30
- [ ] Changes persist after page refresh

### TC-04: Add Judges Per Track

**As:** `hackforger` (admin)

1. Add `judge_carol` as judge for AI Track (by username, not user ID)
2. Add `judge_dave` as judge for Web Track (by username)
3. Add `judge_carol` also as judge for Web Track (multi-track assignment)

**Expected:**
- [ ] Judge list shows carol on AI Track and Web Track, dave on Web Track only
- [ ] Adding judge by username works (not raw user ID)
- [ ] Duplicate assignment to same track blocked with error
- [ ] Removing a judge from one track doesn't affect their other track assignment

### TC-05: Publish + Register + Start Hacking

**As:** `hackforger` → publish, then switch users

1. Publish hackathon (Draft → Open)
2. As `hacker_eve`: register for hackathon
3. As `hacker_frank`: register for hackathon
4. As `hackforger`: approve both registrations
5. As `hackforger`: start hacking (Open → Hacking)

**Expected:**
- [ ] Publish requires >= 1 track (existing validation)
- [ ] Registrations appear in manage page
- [ ] Hacking phase starts successfully

### TC-06: Submit Projects

**As:** `hacker_eve` and `hacker_frank`

1. As `hacker_eve`: submit to AI Track (title="Eve's AI Bot", demo URL, description)
2. As `hacker_frank`: submit to Web Track (title="Frank's Web App", demo URL)
3. As `hacker_eve`: also submit to Web Track (title="Eve's Web Tool")

**Expected:**
- [ ] Submissions created and visible on hackathon page
- [ ] Each submission linked to correct track

### TC-07: Start Judging

**As:** `hackforger` (admin)

1. Transition to Judging phase (Hacking → Judging)

**Expected:**
- [ ] Phase transition succeeds (criteria exist, submissions exist)
- [ ] Criteria modification now blocked ("criteria locked" error)
- [ ] Attempting to add new criterion shows error

### TC-08: Judge Scoring — JudgeScoreCard.vue

**As:** `judge_carol`

1. Navigate to `/hackathon/p2-judge-test` — should see "Judge" button
2. Click "Judge" → `/hackathon/p2-judge-test/judge`
3. See AI Track tab with Eve's submission + Web Track tab with Eve's and Frank's submissions
4. On AI Track, score Eve's submission:
   - Innovation: 9.0, comment "Highly original concept"
   - Technical Execution: 7.5, comment "Good but could be more robust"
   - Impact: 8.0, comment "Strong potential"
   (Note: Presentation NOT shown — disabled for AI Track)
5. Switch to Web Track, score Eve's submission with all 4 criteria
6. Score Frank's submission with all 4 criteria
7. Verify progress bar updates as submissions are scored

**Expected:**
- [ ] Judge page loads with Vue component (not bare HTML form)
- [ ] Track tabs visible (AI Track, Web Track)
- [ ] AI Track rubric shows only 3 criteria (Presentation disabled)
- [ ] Web Track rubric shows all 4 criteria
- [ ] Score inputs respect MaxScore range (0-10)
- [ ] Submit scores per submission works without page reload
- [ ] Progress bar: "1 of 1" after scoring AI Track, "2 of 2" after Web Track
- [ ] Existing scores pre-filled when revisiting a scored submission

### TC-09: Judge Scoring — Second Judge

**As:** `judge_dave`

1. Navigate to judge page — should only see Web Track (not assigned to AI)
2. Score both Web Track submissions with all 4 criteria

**Expected:**
- [ ] Only Web Track visible (dave not assigned to AI)
- [ ] Scores submitted successfully
- [ ] No access to AI Track submissions

### TC-10: Permission Checks

1. As `hacker_eve`: navigate to `/hackathon/p2-judge-test/judge`
2. As non-logged-in user: navigate to the same URL

**Expected:**
- [ ] `hacker_eve` redirected with "not a judge" error
- [ ] Non-logged-in user redirected to login

### TC-11: Preview Finalize

**As:** `hackforger` (admin)

1. On manage page, click "Preview Results"
2. Review per-track ranking tables

**Expected:**
- [ ] AI Track: Eve's submission ranked #1 (only entry), shows weighted score based on 3 criteria with AI Track weights
- [ ] Web Track: two submissions ranked by weighted score using Web Track weights
- [ ] Per-criteria average scores shown in breakdown
- [ ] Preview is idempotent (can click again, same results)

### TC-12: Confirm Finalize

**As:** `hackforger` (admin)

1. Click "Confirm & Finalize"
2. Accept confirmation dialog

**Expected:**
- [ ] Status changes to Finished
- [ ] Rankings persisted to submissions
- [ ] Track repos tagged with "submission-deadline"
- [ ] v1-results releases created on track repos
- [ ] Feed event published (check activity feed)

### TC-13: Leaderboard

1. Navigate to `/hackathon/p2-judge-test/leaderboard`
2. Check per-track tabs

**Expected:**
- [ ] Tab navigation works (AI Track, Web Track)
- [ ] Rankings match preview results
- [ ] Per-criteria score breakdown visible (hackathon is Finished)
- [ ] Weighted total shown
- [ ] Top entry highlighted

### TC-14: API Endpoints

Using `curl` or API client:

```bash
TOKEN="your-api-token"
BASE="https://hackforger.inside.h2os.cloud/api/v1/hackforger"

# List criteria
curl -s "$BASE/hackathons/{id}/criteria" -H "Authorization: token $TOKEN" | jq

# Get effective rubric for a track
curl -s "$BASE/hackathons/{id}/tracks/{tid}/criteria" -H "Authorization: token $TOKEN" | jq

# Get leaderboard (grouped by track)
curl -s "$BASE/hackathons/{id}/leaderboard" -H "Authorization: token $TOKEN" | jq

# Preview finalize
curl -s "$BASE/hackathons/{id}/finalize-preview" -H "Authorization: token $TOKEN" | jq
```

**Expected:**
- [ ] Criteria API returns all 4 criteria
- [ ] Effective rubric for AI Track returns 3 (Presentation excluded)
- [ ] Leaderboard grouped by track with criteria breakdown
- [ ] Finalize preview returns ranked results

---

## Report Template

```markdown
# Phase 2 Judge System — E2E Test Report

**Date:** YYYY-MM-DD
**Tester:**
**Branch:**
**Build commit:**

## Results

| TC | Name | Pass/Fail | Notes |
|----|------|-----------|-------|
| 01 | Create Hackathon + Criteria | | |
| 02 | Tracks + Auto-Seed | | |
| 03 | Track Criteria Overrides | | |
| 04 | Per-Track Judges | | |
| 05 | Publish + Register + Hack | | |
| 06 | Submit Projects | | |
| 07 | Start Judging | | |
| 08 | Judge Scoring (Carol) | | |
| 09 | Judge Scoring (Dave) | | |
| 10 | Permission Checks | | |
| 11 | Preview Finalize | | |
| 12 | Confirm Finalize | | |
| 13 | Leaderboard | | |
| 14 | API Endpoints | | |

**Total:** X/14 passed

## Issues Found

(List any bugs, UI issues, or unexpected behavior)

## Screenshots

(Attach key screenshots)
```
