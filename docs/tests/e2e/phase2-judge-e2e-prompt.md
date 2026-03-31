# Phase 2 Judge System — Manual E2E Test Prompt (Round 2)

**Instance:** https://hackforger.inside.h2os.cloud

## Prerequisites

### Start the server (if testing in a worktree)

See [docs/tests/local-testing-guide.md](../local-testing-guide.md) for details.

```bash
# 1. Copy config from main repo (worktrees don't have custom/)
mkdir -p custom/conf
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini

# 2. Build frontend + backend
make frontend
TAGS="bindata sqlite sqlite_unlock_notify" make build

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
- Hacker: `hacker_eve` (create if needed, must own at least 1 repo)
- Hacker 2: `hacker_frank` (create if needed, must own at least 1 repo)

### Pre-test setup

```bash
# Clean hackathon data
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db "
DELETE FROM hackathon_judge_score;
DELETE FROM hackathon_judge_criteria;
DELETE FROM hackathon_track_criteria;
DELETE FROM hackathon_judge;
DELETE FROM hackathon_submission;
DELETE FROM hackathon_registration;
DELETE FROM hackathon_track;
DELETE FROM hackathon;
"
```

Ensure `hacker_eve` and `hacker_frank` each own at least one repo (e.g., `hacker_eve/my-ai-project`, `hacker_frank/my-web-app`). Create via the web UI if needed.

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
5. Edit a criterion inline (e.g., change Innovation weight to 35, then back to 30)
6. Verify edit submits to `/manage/criteria/{id}/update` (not 404)

**Expected:**
- [ ] Hackathon created in Draft status
- [ ] 4 criteria appear in manage page with correct weights
- [ ] Inline criteria editing works (no 404)
- [ ] Criteria can be deleted (only in Draft/Open)

### TC-02: Create Tracks + Verify Track Criteria Auto-Seed + Index Files

**As:** `hackforger` (admin)

1. On manage page, create Track 1: Name="AI Track", Description="AI/ML projects"
2. Create Track 2: Name="Web Track", Description="Web applications"
3. Verify track criteria overrides section shows both tracks with all 4 criteria enabled
4. Navigate to the AI Track repo — verify `SUBMISSIONS.md` and `submissions.json` exist

**Expected:**
- [ ] Both tracks created with repos
- [ ] Track criteria auto-seeded: all 4 criteria enabled for both tracks
- [ ] Default weights shown (use hackathon-level defaults)
- [ ] Track repo contains `SUBMISSIONS.md` (header + empty table) and `submissions.json` (`[]`)

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

### TC-06: Submit Projects (New: User-Owned Repo + Track Index PR)

**As:** `hacker_eve` and `hacker_frank`

1. As `hacker_eve`: navigate to submit page, verify repo dropdown shows her repos
2. Submit to AI Track:
   - Title: "Eve's AI Bot"
   - Description: "An AI-powered assistant"
   - Demo URL: `https://example.com/eve-ai`
   - Repo: select `hacker_eve/my-ai-project` from dropdown
   - Track: AI Track
3. As `hacker_frank`: submit to Web Track:
   - Title: "Frank's Web App"
   - Repo: select `hacker_frank/my-web-app` from dropdown
   - Track: Web Track
4. As `hacker_eve`: submit to Web Track:
   - Title: "Eve's Web Tool"
   - Repo: select a repo or "No repository"
   - Track: Web Track
5. Check the AI Track repo's SUBMISSIONS.md and submissions.json

**Expected:**
- [ ] Submit form shows repo dropdown with user's repositories
- [ ] "No repository (description only)" option available
- [ ] Submissions created and visible on hackathon page
- [ ] Each submission shows linked repo name (if provided)
- [ ] AI Track repo has updated SUBMISSIONS.md with Eve's AI Bot entry
- [ ] AI Track repo has updated submissions.json with structured entry
- [ ] A PR was created in AI Track repo for the submission index update
- [ ] No fork operations occurred (no orphaned git directories)

### TC-07a: Start Judging — No Criteria Edge Case

**As:** `hackforger` (admin)

> **Setup**: Create a separate hackathon `no-criteria-test` with tracks, add a submission via API, but do NOT define any criteria.

1. Advance to Hacking phase
2. Attempt to transition to Judging phase (Hacking → Judging)

**Expected:**
- [ ] Transition blocked with criteria error (checked BEFORE submissions check)
- [ ] Hackathon remains in Hacking status
- [ ] Manage page shows warning: "Warning: No scoring criteria defined."

### TC-07b: Start Judging

**As:** `hackforger` (admin)

> **Context**: Back to the main "Phase2 Judge Test" hackathon (which has criteria).

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
8. Go back to a scored submission — verify scores are pre-filled

**Expected:**
- [ ] Judge page loads with Vue component (not bare HTML form)
- [ ] Track tabs visible (AI Track, Web Track)
- [ ] AI Track rubric shows only 3 criteria (Presentation disabled)
- [ ] Web Track rubric shows all 4 criteria
- [ ] Score inputs respect MaxScore range (0-10)
- [ ] Submit scores per submission works without page reload
- [ ] Progress bar: "1 of 1" after scoring AI Track, "2 of 2" after Web Track
- [ ] Existing scores pre-filled when revisiting a scored submission
- [ ] Re-submitting updated scores works (upsert)

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

### TC-14: API — Criteria CRUD

Using `curl` or API client. Replace `{id}` with the hackathon ID, `{cid}` with a criterion ID.

```bash
TOKEN="your-api-token"
BASE="https://hackforger.inside.h2os.cloud/api/v1/hackforger"

# List criteria
curl -s "$BASE/hackathons/{id}/criteria" | jq

# Add a criterion via API (on a Draft hackathon)
curl -s -X POST "$BASE/hackathons/{id}/criteria" \
  -H "Authorization: token $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"API-Created","description":"Test","max_score":10,"weight":20,"sort_order":5}' | jq

# Update criterion
curl -s -X PUT "$BASE/hackathons/{id}/criteria/{cid}" \
  -H "Authorization: token $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"API-Updated","weight":30}' | jq

# Delete criterion
curl -s -X DELETE "$BASE/hackathons/{id}/criteria/{cid}" \
  -H "Authorization: token $TOKEN"
```

**Expected:**
- [ ] List returns all criteria for the hackathon
- [ ] Add returns 201 Created, criterion appears in subsequent list
- [ ] Update returns 200 with updated criterion JSON
- [ ] Delete returns 204 No Content
- [ ] All write operations blocked (409 Conflict) when hackathon is in Judging/Finished status

### TC-15: API — Track Effective Rubric

```bash
# Get effective rubric for AI Track (should exclude Presentation)
curl -s "$BASE/hackathons/{id}/tracks/{ai_tid}/criteria" | jq

# Get effective rubric for Web Track (should include all 4)
curl -s "$BASE/hackathons/{id}/tracks/{web_tid}/criteria" | jq

# Set track criteria override via API
curl -s -X PUT "$BASE/hackathons/{id}/tracks/{tid}/criteria/{cid}" \
  -H "Authorization: token $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled":false,"weight":0}' | jq
```

**Expected:**
- [ ] AI Track rubric returns 3 criteria (Presentation excluded)
- [ ] Web Track rubric returns 4 criteria with custom weights
- [ ] Override PUT returns 200
- [ ] Subsequent GET reflects the override change

### TC-16: API — Finalize Preview + Confirm + Leaderboard

> **Note**: Run on a separate hackathon in Judging status with scores, or test via the main hackathon after scoring but before web finalize.

```bash
# Preview finalize (does NOT change status)
curl -s "$BASE/hackathons/{id}/finalize-preview" \
  -H "Authorization: token $TOKEN" | jq

# Confirm finalize (locks results)
curl -s -X POST "$BASE/hackathons/{id}/finalize-confirm" \
  -H "Authorization: token $TOKEN" | jq

# Verify leaderboard (grouped by track)
curl -s "$BASE/hackathons/{id}/leaderboard" | jq
```

**Expected:**
- [ ] Preview returns ranked results with weighted totals and per-criteria scores
- [ ] Preview is idempotent (calling twice returns same results)
- [ ] Preview does NOT change hackathon status (still Judging)
- [ ] Confirm returns 200, hackathon transitions to Finished
- [ ] Confirm on non-Judging hackathon returns 409 Conflict
- [ ] Leaderboard returns **grouped by track** format: `[{track_id, track_name, entries: [...]}]`

---

## Round 1 Bug Fix Verification

These were bugs found in Round 1 — verify they are fixed:

### TC-BF1: Criteria Edit No Longer 404 (was BUG-01)

1. On manage page, edit a criterion's name inline
2. Click the Edit/Save button

**Expected:**
- [ ] Form submits to `/manage/criteria/{id}/update` (not `/manage/criteria/{id}`)
- [ ] No 404 error
- [ ] Criterion updated successfully

### TC-BF2: Submission Without Fork (was BUG-02)

1. Submit a project linking to user's own repo
2. Delete the submission via API or DB
3. Re-submit with the same repo

**Expected:**
- [ ] No "repository files already exist" error
- [ ] No orphaned git directories created
- [ ] Re-submission succeeds cleanly

### TC-BF3: Criteria Check Before Submissions (was BUG-03)

1. Create a hackathon with tracks, add criteria, start hacking
2. Without any submissions, attempt start-judging via API:
   ```bash
   curl -s -X POST "$BASE/hackathons/{id}/start-judging" \
     -H "Authorization: token $TOKEN" | jq
   ```

**Expected:**
- [ ] Error message mentions "no submissions" (not "no criteria")
- [ ] Criteria check passes first (criteria exist), then submissions check fails

---

## Report Template

```markdown
# Phase 2 Judge System — E2E Test Report (Round 2)

**Date:** YYYY-MM-DD
**Tester:**
**Branch:**
**Build commit:**

## Results

| TC | Name | Pass/Fail | Notes |
|----|------|-----------|-------|
| 01 | Create Hackathon + Criteria (+ edit fix) | | |
| 02 | Tracks + Auto-Seed + Index Files | | |
| 03 | Track Criteria Overrides | | |
| 04 | Per-Track Judges | | |
| 05 | Publish + Register + Hack | | |
| 06 | Submit Projects (user-owned repo + index PR) | | |
| 07a | Start Judging — No Criteria | | |
| 07b | Start Judging | | |
| 08 | Judge Scoring (Carol) | | |
| 09 | Judge Scoring (Dave) | | |
| 10 | Permission Checks | | |
| 11 | Preview Finalize | | |
| 12 | Confirm Finalize | | |
| 13 | Leaderboard | | |
| 14 | API — Criteria CRUD | | |
| 15 | API — Track Effective Rubric | | |
| 16 | API — Finalize + Leaderboard (grouped) | | |
| BF1 | Criteria Edit No 404 | | |
| BF2 | Submission Without Fork | | |
| BF3 | Criteria Check Order | | |

**Total:** X/20 passed

## Issues Found

(List any bugs, UI issues, or unexpected behavior)

## Screenshots

(Attach key screenshots)
```
