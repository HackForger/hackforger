# Phase 2: Reputation + Feed + Profile — Manual E2E Test Prompt

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
- Hacker 1: `hacker_eve` (or create one)
- Hacker 2: `hacker_frank` (for leaderboard ranking)
- If users don't exist, create them via Site Administration → User Accounts

### Pre-test data check

Phase 2 introduces `hackforger_action` (separate feed table) and `hackforger_reputation` tables. Verify migrations ran:

```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  ".tables" | tr ' ' '\n' | grep hackforger
# Expected output includes:
#   hackforger_action
#   hackforger_reputation
```

If stale reputation data exists from a previous run, clean up:

```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db \
  "DELETE FROM hackforger_reputation;"
```

### Checklist

- [ ] HackForger server running (verified with `./gitea --version`)
- [ ] `custom/conf/app.ini` exists in the worktree (copied from main repo)
- [ ] Admin account accessible
- [ ] Hacker accounts accessible (`hacker_eve`, `hacker_frank`)
- [ ] `hackforger_action` table exists in the database
- [ ] `hackforger_reputation` table exists in the database
- [ ] At least one completed hackathon or grant round exists (to seed feed events)

> **Tip:** If no prior Phase 1 data exists, run through Phase 1 hackathon or grant E2E first to generate events that reputation will be calculated from.

---

## Test Scenarios

### A. Dashboard Community Tab

**A1. Verify tab navigation**

1. Log in as admin (`hackforger`).
2. Navigate to the dashboard (`/`).
3. **Verify:** Two tabs are visible at the top of the feed area: "Code Activity" and "Community".
4. **Verify:** "Code Activity" tab is selected by default.
5. **Verify:** The Code Activity tab shows standard Forgejo git events (pushes, PRs, issues).

**A2. Community tab — empty state**

1. (Use a freshly seeded database or a user with no HackForger activity.)
2. Click the "Community" tab.
3. **Verify:** If no HackForger events exist for this user's feed, an empty-state message is shown (e.g., "No community activity yet.").
4. **Verify:** The page does not crash (no 500 error, no blank page).

**A3. Community tab — events after hackathon creation**

1. Log in as admin.
2. Create a new hackathon (navigate to `/hackathons/new`, fill name/slug/description, submit).
3. Navigate back to the dashboard (`/`).
4. Click the "Community" tab.
5. **Verify:** An event for "created a new hackathon" appears in the feed.
6. **Verify:** The event entry shows actor avatar, username, action description, and a timestamp.
7. **Verify:** The event is NOT shown in the "Code Activity" tab.

**A4. Community tab — additional event types**

Generate more events, then verify each appears in the Community tab:

1. Open the hackathon (navigate to manage page, click "Publish").
2. As `hacker_eve`, register for the hackathon.
3. As admin, return to dashboard Community tab.
4. **Verify:** Phase-change and registration events appear in the feed (if admin follows `hacker_eve` or events are global-audience).

**A5. Tab state preservation on refresh**

1. While on the Community tab, note the URL (it should include `?tab=community` or similar query parameter).
2. Reload the page.
3. **Verify:** Community tab remains selected after reload (tab state is reflected in the URL).

---

### B. Feed Events (via API)

For these tests, use `curl` with the admin token. Obtain a token from User Settings → Applications → Generate Token (with `read:hackforger_feed` scope, or use the admin token).

```bash
export TOKEN="<your-admin-token>"
export BASE="https://hackforger.inside.h2os.cloud"
```

**B1. Global feed**

```bash
curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/feed?type=global" | jq '.items[] | {op_type, op_name, actor_name}'
```

1. **Verify:** Response is a JSON object with an `items` array.
2. **Verify:** At minimum, a `HackathonCreated` event (op_type 30) is present.
3. **Verify:** Each item has `op_type`, `op_name`, `actor_name`, `created_unix`, and `content` fields.
4. **Verify:** No item has a blank or `null` `op_name`.

**B2. Following feed**

1. As admin, follow `hacker_eve` via the Forgejo UI (navigate to `hacker_eve`'s profile, click Follow).
2. As `hacker_eve`, register for the hackathon (if not already done in A4).
3. Query the following feed as admin:

```bash
curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/feed?type=following" | jq '.items[] | {op_type, actor_name}'
```

4. **Verify:** Events from `hacker_eve` appear (e.g., `HackathonRegistered`, op_type 31).
5. **Verify:** Events from users not followed by admin do NOT appear.

**B3. Entity timeline feed**

After creating a hackathon, obtain its ID:

```bash
HACKATHON_ID=$(curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/hackathons" | jq '.[0].id')
echo "Hackathon ID: $HACKATHON_ID"
```

Query the entity timeline:

```bash
curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/feed?type=entity&entity_type=hackathon&entity_id=$HACKATHON_ID" \
  | jq '.items[] | {op_type, op_name, created_unix}'
```

1. **Verify:** Only events related to this specific hackathon are returned.
2. **Verify:** Events are ordered chronologically (oldest first or newest first — consistent direction).
3. **Verify:** At minimum, the hackathon-created event appears.
4. Try querying a non-existent entity: `entity_id=99999`. **Verify:** Returns an empty `items` array, not a 404 or 500.

**B4. Pagination**

```bash
curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/feed?type=global&page=1&limit=2" | jq '{total_count, item_count: (.items | length)}'
```

1. **Verify:** `items` contains at most 2 entries.
2. **Verify:** `total_count` (or equivalent pagination field) reflects the actual total.
3. Query page 2 and verify a different set of events is returned (if total > 2).

---

### C. Profile Community Tab

**C1. Basic profile community tab**

1. Log in as admin.
2. Navigate to `hacker_eve`'s profile: `/{username}?tab=community` (e.g., `/hacker_eve?tab=community`).
3. **Verify:** The "Community" tab is selected and visible.
4. **Verify:** If `hacker_eve` has performed HackForger actions (registered, submitted), those events are listed.
5. **Verify:** Events show action description, target entity link, and timestamp.
6. **Verify:** Only events where `hacker_eve` is the actor are shown (not events by other users).

**C2. Profile community tab — own profile**

1. Log in as `hacker_eve`.
2. Navigate to your own profile (`/hacker_eve?tab=community`).
3. **Verify:** Same events shown as when viewing as admin.
4. **Verify:** No "edit" controls appear on this tab for other users.

**C3. Profile community tab — empty state**

1. Create a new user account (e.g., `new_user_test`) via Site Administration.
2. Navigate to `/new_user_test?tab=community`.
3. **Verify:** Empty-state message shown (e.g., "No community activity yet.").
4. **Verify:** No crash or blank page.

**C4. Pagination on profile community tab**

1. Navigate to a user who has more than 10 HackForger events (generate by going through full hackathon + grant lifecycle as that user).
2. Navigate to `/{username}?tab=community`.
3. **Verify:** Pagination controls appear (next/prev page, or page numbers).
4. Click to page 2.
5. **Verify:** A different set of events is shown.

---

### D. Profile Reputation Tab

**D1. Basic reputation tab**

1. Ensure reputation has been calculated for `hacker_eve` (either by triggering the admin "Recalculate All" from Scenario G, or by the system auto-calculating after Phase 1 events).
2. Navigate to `/hacker_eve?tab=reputation`.
3. **Verify:** "Reputation" tab is selected and visible.
4. **Verify:** A reputation score is displayed (numeric value).
5. **Verify:** A tier badge is displayed (e.g., "Bronze", "Silver", "Gold", "Platinum").

**D2. Reputation detail table**

1. On the reputation tab, scroll to the metrics breakdown table.
2. **Verify:** The table includes rows for all tracked metrics, for example:
   - Hackathons participated
   - Hackathons won (or top-ranked)
   - Bounties claimed
   - Grant projects funded
   - Credits earned
3. **Verify:** Each metric row shows the metric name and the user's current value.
4. **Verify:** No row shows `null` or a rendering error.

**D3. Reputation tab — unranked user**

1. Navigate to `/new_user_test?tab=reputation` (the new user with no activity).
2. **Verify:** Page loads without crash.
3. **Verify:** Shows score of 0 or a "Not yet ranked" message.
4. **Verify:** Tier badge shows the lowest tier (e.g., "Bronze" or "Unranked").

---

### E. Reputation Sidebar Card

**E1. Sidebar card on profile page (any tab)**

1. Navigate to `/hacker_eve` (default tab, "Overview" or "Activity").
2. **Verify:** A reputation card is visible in the right sidebar.
3. **Verify:** The card displays:
   - Reputation score (number)
   - Tier badge/label
   - At least 2–3 top metrics with values

**E2. Sidebar card — admin profile**

1. Navigate to `/hackforger` (admin's profile).
2. **Verify:** Reputation sidebar card is present.
3. **Verify:** If admin has no HackForger activity, the card shows 0 / lowest tier without crashing.

**E3. Sidebar card — not logged in**

1. Log out.
2. Navigate to `/hacker_eve`.
3. **Verify:** Reputation sidebar card is still visible (it's a public display).
4. **Verify:** No authentication error or missing card.

---

### F. Reputation API

**F1. Get reputation for a user**

```bash
curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/reputation/users/hacker_eve" | jq '.'
```

1. **Verify:** Response is a JSON object with at least these fields:
   - `username` (or `user_name`)
   - `score` (numeric)
   - `tier` (string, e.g., `"bronze"`)
   - `metrics` (object or array of metric key/value pairs)
2. **Verify:** HTTP status is 200.

**F2. Get reputation for non-existent user**

```bash
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/reputation/users/does_not_exist_xyz"
```

1. **Verify:** HTTP status is 404.

**F3. Leaderboard API**

```bash
curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/reputation/leaderboard" | jq '.'
```

1. **Verify:** Response is a JSON array (or object with `users` array).
2. **Verify:** Each entry has `username`, `score`, `tier`, and `rank`.
3. **Verify:** Entries are sorted by score descending (rank 1 has the highest score).
4. **Verify:** `hacker_eve` appears if they have a non-zero score.

**F4. Leaderboard pagination**

```bash
curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/reputation/leaderboard?page=1&limit=1" | jq '{count: length}'
```

1. **Verify:** Returns at most 1 entry when `limit=1`.

**F5. Admin recalculate reputation for a user**

```bash
curl -s -X POST \
  -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/reputation/recalculate/hacker_eve" | jq '.'
```

1. **Verify:** HTTP status is 200 (or 204).
2. **Verify:** After the call, querying `GET .../reputation/users/hacker_eve` returns an updated (or unchanged) score without error.
3. **Verify:** Non-admin token receives HTTP 403.

```bash
# Test with non-admin token (obtain hacker_eve's token first)
export EVE_TOKEN="<hacker_eve-token>"
curl -s -o /dev/null -w "%{http_code}" -X POST \
  -H "Authorization: token $EVE_TOKEN" \
  "$BASE/api/v1/hackforger/reputation/recalculate/hacker_eve"
# Expected: 403
```

---

### G. Admin Reputation Settings

**G1. Access the settings page**

1. Log in as admin.
2. Navigate to `/-/admin/hackforger/reputation`.
3. **Verify:** Page loads without error.
4. **Verify:** Two JSON editor areas (or textareas) are pre-filled:
   - Weights configuration (metric name → weight value)
   - Tier thresholds configuration (tier name → minimum score)
5. **Verify:** Both fields contain valid JSON (not blank, not `{}`).

**G2. Modify weights and save**

1. On the settings page, change one weight value (e.g., increase `hackathon_participated` weight from its current value by 10).
2. Click "Save Settings" (or equivalent submit button).
3. **Verify:** Flash success message appears (e.g., "Settings saved.").
4. **Verify:** Reload the page — the modified weight value is still present (persisted).

**G3. Invalid JSON rejected**

1. Clear the weights textarea and enter invalid JSON: `{ invalid`.
2. Click "Save Settings".
3. **Verify:** An error message is shown (e.g., "Invalid JSON for weights.").
4. **Verify:** Previous valid settings are NOT overwritten (reload page to confirm).

**G4. Recalculate All**

1. Note `hacker_eve`'s current score (from the reputation tab or API).
2. On the admin reputation settings page, click "Recalculate All Reputations" (or equivalent button).
3. **Verify:** Flash success message appears (e.g., "Recalculation started." or "Recalculation complete.").
4. After recalculation completes (may be async — wait a moment or check again):
5. Navigate to `/hacker_eve?tab=reputation`.
6. **Verify:** Score is displayed (may be same or updated depending on data).
7. **Verify:** The page does not show a stale "reputation not found" error.

**G5. Access control**

1. Log out.
2. Attempt to navigate to `/-/admin/hackforger/reputation`.
3. **Verify:** Redirected to login page (not a 403 or 500).
4. Log in as `hacker_eve` (non-admin).
5. Attempt to navigate to `/-/admin/hackforger/reputation`.
6. **Verify:** Access denied (403 page or redirect to dashboard).

---

### H. Explore Leaderboard (Public Page)

**H1. Access leaderboard without login**

1. Log out completely.
2. Navigate to `/explore/reputation`.
3. **Verify:** Page loads (HTTP 200, no redirect to login).
4. **Verify:** A ranked table is displayed with columns: Rank, Username/Avatar, Score, Tier.

**H2. Leaderboard ranking order**

1. Log in as admin and navigate to `/explore/reputation`.
2. **Verify:** Users are ranked from highest to lowest score (rank 1 has the highest score).
3. If `hacker_eve` and `hacker_frank` both have scores: **Verify** the higher-scoring user appears above the other.

**H3. Leaderboard tier badges**

1. On the `/explore/reputation` page, inspect at least one user row.
2. **Verify:** Each row displays a tier badge (e.g., colored badge showing "Bronze", "Silver", etc.).
3. **Verify:** Tier badge color or style corresponds to the tier level (no raw tier string without styling).

**H4. Leaderboard pagination**

1. If more than 20 users exist: **Verify** pagination controls appear.
2. If fewer users exist: **Verify** all users are shown on a single page with no broken pagination links.
3. Append `?page=1` to the URL: **Verify** it works the same as no page parameter.
4. Append `?page=999` to the URL: **Verify** an empty state or last-page message is shown (not a crash).

**H5. Leaderboard link from profile**

1. Log in as any user, navigate to `/hacker_eve`.
2. In the reputation sidebar card: **Verify** there is a link to the full leaderboard (`/explore/reputation`).
3. Click it.
4. **Verify:** Navigates to the leaderboard page.

---

### I. End-to-End Reputation Change Verification

This scenario verifies that reputation actually changes after new HackForger activity.

**I1. Baseline score**

1. Query `hacker_eve`'s current reputation score:

```bash
curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/reputation/users/hacker_eve" | jq '.score'
```

2. Note the value as `SCORE_BEFORE`.

**I2. Trigger new activity**

1. (If not already done) Have `hacker_eve` submit a project to a hackathon or grant round, then finalize/fund it.
2. As admin, navigate to `/-/admin/hackforger/reputation` and click "Recalculate All".

**I3. Verify score updated**

```bash
curl -s -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/reputation/users/hacker_eve" | jq '.score'
```

1. **Verify:** `SCORE_AFTER` is greater than `SCORE_BEFORE` (assuming at least one positive event was added).
2. Navigate to `/hacker_eve?tab=reputation`.
3. **Verify:** Score displayed in the UI matches `SCORE_AFTER`.
4. Navigate to `/explore/reputation`.
5. **Verify:** `hacker_eve`'s row reflects the updated score.

---

### J. Error Cases and Edge Cases

**J1. Feed API — invalid type parameter**

```bash
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: token $TOKEN" \
  "$BASE/api/v1/hackforger/feed?type=invalid_type"
```

1. **Verify:** Returns HTTP 400 (Bad Request), not 200 or 500.

**J2. Feed API — unauthenticated access to following feed**

```bash
curl -s -o /dev/null -w "%{http_code}" \
  "$BASE/api/v1/hackforger/feed?type=following"
```

1. **Verify:** Returns HTTP 401 (Unauthorized), since following requires authentication.

**J3. Global feed — accessible without token**

```bash
curl -s -o /dev/null -w "%{http_code}" \
  "$BASE/api/v1/hackforger/feed?type=global"
```

1. **Verify:** Returns HTTP 200 (global feed is public).

**J4. Profile community tab — non-existent user**

1. Navigate to `/user_does_not_exist_xyz?tab=community`.
2. **Verify:** Returns a 404 page, not a 500 or blank page.

**J5. Dashboard Community tab — no followed users with activity**

1. Log in as `hacker_frank` (who follows nobody).
2. Navigate to the dashboard, click "Community" tab.
3. **Verify:** Community events from global-audience actions are still shown (or an appropriate empty state — not a crash).

---

## Checklist Summary

| # | Scenario | Verified |
|---|----------|---------|
| A1 | Dashboard Code/Community tabs visible | [ ] |
| A2 | Community tab empty state | [ ] |
| A3 | Community tab shows hackathon creation event | [ ] |
| A4 | Community tab shows additional event types | [ ] |
| A5 | Tab state preserved in URL on reload | [ ] |
| B1 | Global feed API returns events with expected fields | [ ] |
| B2 | Following feed shows only followed users' events | [ ] |
| B3 | Entity timeline feed scoped to single entity | [ ] |
| B4 | Feed API pagination works | [ ] |
| C1 | Profile community tab shows actor's events | [ ] |
| C2 | Own profile community tab works | [ ] |
| C3 | Profile community tab empty state | [ ] |
| C4 | Profile community tab pagination | [ ] |
| D1 | Reputation tab shows score and tier | [ ] |
| D2 | Reputation detail metrics table | [ ] |
| D3 | Reputation tab for unranked user | [ ] |
| E1 | Reputation sidebar card visible on profile | [ ] |
| E2 | Sidebar card on admin profile | [ ] |
| E3 | Sidebar card visible when logged out | [ ] |
| F1 | Reputation API returns score/tier/metrics | [ ] |
| F2 | Reputation API 404 for unknown user | [ ] |
| F3 | Leaderboard API returns sorted list | [ ] |
| F4 | Leaderboard API pagination | [ ] |
| F5 | Admin recalculate API (200 admin, 403 non-admin) | [ ] |
| G1 | Admin reputation settings page loads with pre-filled JSON | [ ] |
| G2 | Modify weights and save persists | [ ] |
| G3 | Invalid JSON rejected with error | [ ] |
| G4 | Recalculate All triggers recalc | [ ] |
| G5 | Non-admin access to settings is denied | [ ] |
| H1 | Explore leaderboard public (no login required) | [ ] |
| H2 | Leaderboard ranking order correct | [ ] |
| H3 | Leaderboard tier badges displayed | [ ] |
| H4 | Leaderboard pagination | [ ] |
| H5 | Leaderboard link from profile sidebar card | [ ] |
| I1-I3 | Reputation score actually updates after new activity | [ ] |
| J1 | Feed API rejects invalid type param (400) | [ ] |
| J2 | Following feed requires auth (401) | [ ] |
| J3 | Global feed accessible without auth (200) | [ ] |
| J4 | Profile community tab 404 for non-existent user | [ ] |
| J5 | Dashboard Community tab handles no followed activity | [ ] |

---

## Report Template

```
## E2E Report — Phase 2: Reputation + Feed + Profile

**Date:** YYYY-MM-DD
**Tester:**
**Instance:** https://hackforger.inside.h2os.cloud
**Branch:**
**Binary version** (from `./gitea --version`):

### Results

| # | Scenario | Status | Notes |
|---|----------|--------|-------|
| 1 | Dashboard tabs (A1–A5) | | |
| 2 | Feed events via API (B1–B4) | | |
| 3 | Profile Community tab (C1–C4) | | |
| 4 | Profile Reputation tab (D1–D3) | | |
| 5 | Reputation sidebar card (E1–E3) | | |
| 6 | Reputation API (F1–F5) | | |
| 7 | Admin reputation settings (G1–G5) | | |
| 8 | Explore leaderboard (H1–H5) | | |
| 9 | Reputation change E2E (I1–I3) | | |
| 10 | Error cases (J1–J5) | | |

**Overall:** X/10 PASS

### Issues Found

<!-- List any failures or unexpected behaviors here -->

### Feed Event Types Verified

| op_type | op_name | Appeared In Feed |
|---------|---------|-----------------|
| 30 | HackathonCreated | [ ] |
| 31 | HackathonRegistered | [ ] |
| 32 | HackathonSubmitted | [ ] |
| 33 | HackathonScored | [ ] |
| 50 | HackathonPhaseChanged | [ ] |
| 51 | HackathonFinalized | [ ] |
| (Grant events) | GrantRoundCreated / etc. | [ ] |

### Reputation Tiers Verified

| Tier | Threshold Met | Badge Displayed |
|------|--------------|----------------|
| Bronze (lowest) | [ ] | [ ] |
| Silver | [ ] | [ ] |
| Gold | [ ] | [ ] |
| Platinum (highest) | [ ] | [ ] |
```
