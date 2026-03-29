# Phase 1 Hackathon — E2E Test Report (Round 6)

**Date:** 2026-03-29
**Tester:** Claude (automated via curl)
**Instance:** https://hackforger.inside.h2os.cloud
**Build:** `14.0.3-73-0f15a9e40e+gitea-1.22.0`
**Test slug:** `feed-test-r6` (targeted fix-verification — Bug #13/14/15)

---

## Fix Verification Summary (Round 6)

Targeted check of the 3 remaining low-severity issues from Round 5:

| Bug | Description | Result |
|-----|-------------|--------|
| **#13** | Follower-scoped feed events (registered/submitted/scored) | ✅ **FIXED** |
| **#14** | Registration table shows duplicate rows | ✅ **FIXED** |
| **#15** | Judge shown as "User #N" instead of username | ✅ **FIXED** |

### Bug #14 — Registration duplicate rows ✅ FIXED

Manage page for `spring-hack-r5` now shows exactly 1 row per registration:
```html
<tbody>
<tr><td>hacker_eve</td><td>Approved</td><td></td></tr>
</tbody>
```
Previously showed 2 identical rows. Now: exactly 1.

### Bug #15 — Judge shown as "User #N" ✅ FIXED

Manage page for `spring-hack-r5` now shows:
```html
<a href="/judge_carol">judge_carol</a>
```
Previously displayed "User #8" with no link.

### Bug #13 — Follower-scoped feed events ✅ FIXED

**Method:** Created fresh hackathon `feed-test-r6` in build `14.0.3-73-0f15a9e40e`. Full lifecycle: publish → register → start hacking → submit → start judging → score. Admin (`hackforger`, ID:1) follows both `hacker_eve` (ID:3) and `judge_carol` (ID:8).

**Finding:** All three event types now correctly appear in the following feed:

```
GET /api/v1/hackforger/feed?type=following&limit=20
Items: 8
  op_type=33 (hackathon_scored)      user=8  ★ judge_carol scored
  op_type=50 (hackathon_phase_changed) user=1
  op_type=32 (hackathon_submitted)   user=3  ★ hacker_eve submitted
  op_type=50 (hackathon_phase_changed) user=1
  op_type=31 (hackathon_registered)  user=3  ★ hacker_eve registered
  op_type=50 (hackathon_phase_changed) user=1
  op_type=30 (hackathon_created)     user=1
  op_type=30 (hackathon_created)     user=1
```

op_type 31/32/33 are now created in the action log and properly scoped to follower feeds.

---

## Round 5 Full Test Results (reference)

> See below — all Round 5 steps remain valid. Round 6 only re-tested #13/#14/#15.

---

# Phase 1 Hackathon — E2E Test Report (Round 5)

**Date:** 2026-03-29
**Tester:** Claude (automated via curl)
**Instance:** https://hackforger.inside.h2os.cloud
**Build:** `14.0.3-73-65ce12eca8+gitea-1.22.0`
**Test slug:** `spring-hack-r5`

---

## Fix Verification Summary (Round 5)

All 4 bugs fixed in this build:

| Bug | Fix | Verified |
|-----|-----|---------|
| **#17** | `ErrNoTracks`/`ErrNoSubmissions` + `ctx.Tr()` i18n | ✅ **FIXED** |
| **#18 CRITICAL** | `baseRepo.LoadOwner(ctx)` before fork | ✅ **FIXED** |
| **#19** | `CreateNewTag` with `repo.DefaultBranch` | ✅ **FIXED** |
| **#20** | Track Code button `href="/{org}/{repo}"` via `TrackRepoNames` map | ✅ **FIXED** |

---

## Complete Step-by-Step Results

### Step 1: Create Hackathon ✅
- POST `/hackathons/new` → 303 to `/hackathon/spring-hack-r5`
- Status: **Draft** ✅

### Step 1a: ★ Auto-Created Organization ✅
- `GET /spring-hack-r5` → 200, org profile ✅
- Detail page shows "组织: spring-hack-r5" link ✅
- API: `LinkedOrgID=17`, owner `hackforger` in org ✅

### Step 2: Add Tracks via Web Form ✅
- POST `manage/tracks` → 303, **no flash error** for both tracks ✅

### Step 2a: ★ Auto-Created Repositories ✅
- `/spring-hack-r5/ai-track` → **200** ✅
- `/spring-hack-r5/web-track` → **200** ✅
- `AI Track RepoID=7`, `Web Track RepoID=8` (non-zero) ✅

### Step 2b: ★ Auto-Created Milestones ✅
- AI Track: 4 milestones — Registration, Hacking, Judging, Results ✅
- Web Track: 4 milestones — Registration, Hacking, Judging, Results ✅

### Step 3: Add Judge ✅
- `manage/judges` with `user_id=8` → 303, "评委 (1)" ✅

### Step 4: Publish (Draft → Open) ✅
- Phase = **"Current: Open (Registration)"** ✅
- Explore page: green "报名中" badge ✅

### Step 5: ★ Dashboard Feed — HackathonCreated ✅
- Feed API: `op_type=30` (hackathon_created) ✅
- Dashboard HTML: "创建了新的黑客松" + rocket icon ✅
- Dashboard HTML: "变更了黑客松阶段" + rocket icon ✅

### Step 6: Register hacker_eve ✅
- "参赛方式" dropdown with "个人参赛" + user's org options ✅
- POST `register` → 303, flash **"报名成功！"** ✅

### Step 6a: ★ Org Membership ✅
- Org `spring-hack-r5` members: `hackforger` (ID:1) + `hacker_eve` (ID:3) ✅

### Step 7: Approve + Start Hacking ✅
- Registration: Pending → Approved ✅
- Phase = **"Current: Hacking"** ✅

### Step 7a: ★ v0-kickoff Tag ✅ (Bug #19 FIXED)
- `spring-hack-r5/ai-track` tags: **`v0-kickoff`** ✅
- `spring-hack-r5/web-track` tags: **`v0-kickoff`** ✅
- Previously: 0 tags created

### Step 8: Submit Project ✅ (Bug #18 FIXED)
- POST `submit` → **303**, flash **"项目提交成功！"** ✅
- Previously: HTTP 500 PANIC — nil pointer on `BaseRepo.Owner`

### Step 8a: ★ Fork + PR ✅ (Bug #18 FIXED)
- Fork `/hacker_eve/ai-track-hacker_eve` → **200** ✅
- `hacker_eve` repos: `ai-track-hacker_eve | fork=True` ✅
- PRs on `spring-hack-r5/ai-track`: **1 open PR** — `#1 AI Assistant Bot by hacker_eve` ✅

### Step 9: Start Judging ✅
- Phase = **"Current: Judging"** ✅

### Step 10: Score Submission ✅
- Judge page shows "AI Assistant Bot" with score form ✅
- POST `judge/7/score` with score=9.0, comment="Excellent AI project!" → 303 ✅
- Flash: **"Score submitted"** ✅

### Step 11: Finalize ✅
- Phase = **"Current: Finished"** ✅

### Step 11a: ★ submission-deadline Tag + v1-results Release ✅ (Bug #19 FIXED)
**AI Track tags:** `v0-kickoff`, `submission-deadline`, `v1-results` ✅
**Web Track tags:** `v0-kickoff`, `submission-deadline`, `v1-results` ✅

**v1-results release body (AI Track):**
```markdown
## Results

| Rank | Project | Score |
|------|---------|-------|
| #1 | AI Assistant Bot | 9.0 |
```
✅ Leaderboard markdown auto-generated in release body

### Step 12: Leaderboard ✅
- 排名 / 团队·项目 / 得分 中文表头 ✅
- `#1 AI Assistant Bot — Demo` with clickable link ✅
- Score: **9.0** ✅
- Row: `class="positive"` (green highlight) ✅
- Leaderboard API: `rank=1, total_score=9, demo_url=https://demo.example.com` ✅

### Step 13: ★ Dashboard Feed — All Events ✅
Dashboard HTML events:
- "创建了新的黑客松" ×2 ✅
- "变更了黑客松阶段" ×5 (Draft→Open→Hacking→Judging→Finished) ✅
- "完成了黑客松评审" ×1 ✅

Feed API (global): `hackathon_created` ×6, `hackathon_finalized` ×1 ✅

### Step 14: ★ Detail Page Content ✅ (Bug #20 FIXED)

| Element | Status |
|---------|--------|
| Title: "Spring Hack 2026 R5" | ✅ |
| Status badge: "Finished" | ✅ |
| Organizer link: `href="/hackforger"` | ✅ |
| Org link: `href="/spring-hack-r5"` | ✅ |
| Track "代码" button AI: `href="/spring-hack-r5/ai-track"` | ✅ **Bug #20 FIXED** |
| Track "代码" button Web: `href="/spring-hack-r5/web-track"` | ✅ **Bug #20 FIXED** |
| Participants: "hacker_eve" | ✅ |
| Submission: "AI Assistant Bot" | ✅ |
| Submission PR link: `href="/spring-hack-r5/21/pulls/1"` → "PR #1" | ✅ |
| Submission score: 9.0 | ✅ |
| Submission rank: #1 | ✅ |
| Leaderboard link | ✅ |
| Manage button | ✅ |

### Step 15: Feed API ✅
- `GET /api/v1/hackforger/hackathons` → `Status=4` (Finished), `LinkedOrgID=17` ✅
- `GET /api/v1/hackforger/hackathons/21/leaderboard` → `rank=1, total_score=9` ✅
- `GET /api/v1/hackforger/hackathons/21/tracks` → 2 tracks with non-zero `RepoID` ✅
- `GET /api/v1/hackforger/feed?type=global` → `hackathon_created`, `hackathon_finalized` ✅

### Step 16: Cancel Flow ✅
- `cancel-r5`: Draft → Open → **Cancelled** ✅

### Step 17: Error Cases ✅

| # | Test | Flash message | Result |
|---|------|---------------|--------|
| 17.1 | Double registration | `error=你已经报名了该黑客松。` | ✅ **Bug #16 confirmed fixed** |
| 17.2 | Submit in wrong phase (Finished) | `error=当前阶段不可执行此操作。` | ✅ |
| 17.3 | Judge in wrong phase (Finished) | `error=当前阶段不可执行此操作。` | ✅ |
| 17.4 | Non-existent hackathon | HTTP 404 | ✅ |
| 17.5 | Publish without tracks | `error=发布前需要添加至少一个赛道。` | ✅ **Bug #17 FIXED** |
| 17.6 | Start judging without submissions | `error=开始评审前需要至少一个提交作品。` | ✅ **Bug #17 FIXED** |

---

## All Issues Status

| # | Severity | Description | Status |
|---|----------|-------------|--------|
| 1 | HIGH | Dashboard feed doesn't render hackathon events | ✅ FIXED (prior build) |
| 2 | MEDIUM | Track selector was custom DIV | ✅ FIXED (prior build) |
| 7 | SECURITY | Forms missing CSRF tokens; server doesn't enforce CSRF | ⚠️ OPEN (security-only, not functional) |
| 8 | MEDIUM | Error responses used HTTP 405 or blank pages | ✅ FIXED (prior build) |
| 12 | CRITICAL | `GetRepoInitFile[]: file does not exist` on track creation | ✅ FIXED (prior build) |
| 13 | LOW | Follower-scoped feed events (registered/submitted/scored) not in dashboard | ✅ **FIXED (Round 6 build)** |
| 14 | LOW | Registration table shows duplicate rows | ✅ **FIXED (Round 6 build)** |
| 15 | LOW | Judge shown as "User #8" instead of username | ✅ **FIXED (Round 6 build)** |
| 16 | MEDIUM | Double-registration showed generic error | ✅ FIXED (prior build) |
| 17 | MEDIUM | `no_tracks`/`no_submissions` errors were English | ✅ **FIXED this build** |
| 18 | CRITICAL | nil pointer panic on submit (`BaseRepo.Owner` not loaded) | ✅ **FIXED this build** |
| 19 | HIGH | `v0-kickoff` tag not created on Start Hacking | ✅ **FIXED this build** |
| 20 | LOW | Track Code button `href="#"` instead of real repo URL | ✅ **FIXED this build** |

---

## Overall Assessment

**Result: ✅ PASS — Full lifecycle complete**

### Lifecycle verified end-to-end:
```
Draft → Open (Registration) → Hacking → Judging → Finished
  ↓         ↓                    ↓           ↓          ↓
 Org      Register           v0-kickoff    Score    sub-deadline
created  → Org member         tag         (9.0)    tag + v1-results
          created            created               release (markdown)
                              ↓
                           Submit
                           → Fork created
                           → PR opened
```

### Org/Repo Integration Checklist

| Feature | Status |
|---------|--------|
| Auto-create Org on hackathon create | ✅ |
| Auto-create Repo per track | ✅ |
| Auto-create 4 Milestones per repo | ✅ |
| Add user to Org on registration | ✅ |
| Create `v0-kickoff` tag on Start Hacking | ✅ |
| Create Fork on submit | ✅ |
| Open PR on submit | ✅ |
| Create `submission-deadline` tag on Finalize | ✅ |
| Create `v1-results` release with leaderboard markdown | ✅ |

### Remaining Open Issues (non-blocking)

| # | Description | Impact |
|---|-------------|--------|
| 7 | CSRF tokens missing from forms | Security risk only |
