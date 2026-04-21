# E2E Test: Issue #83 — Milestones Page 1970-01-01 Fix

**Related issue:** [HackForger #83](https://github.com/HackForger/hackforger/issues/83)
**Plan:** `docs/superpowers/plans/2026-04-21-issue-83-milestones-epoch-date.md`
**Tester:** Claude + agent-browser
**Environment:** Worktree instance at `http://localhost:3000` (web-only per CLAUDE.md memory `feedback_e2e_web_mandatory.md`).

## Preconditions

1. Worktree built: `make frontend && TAGS="bindata sqlite sqlite_unlock_notify" make backend`.
2. `custom/conf/app.ini` copied from the main repo (CLAUDE.md memory `feedback_worktree_config.md`).
3. `data/queues/common/LOCK` removed if server was running previously.
4. **Port 3000 available.** The main HackForger instance also uses `HTTP_PORT = 3000`. Either:
   - Stop any process on port 3000: `lsof -iTCP:3000 -sTCP:LISTEN` → if something is listening, kill it or use the fallback below.
   - Or edit the worktree's `custom/conf/app.ini` to use a free port (e.g., `HTTP_PORT = 3100`), then use that port in the agent-browser steps below.
5. Start server: `./gitea web` (worktree).
6. Login as `hackforger` / `admin1234`.

## Test Steps

### Step 1: Baseline — service-layer fix (no new bad rows)

1.1 Navigate to `/` → create a new hackathon titled `e2e-issue83`.
1.2 Link an org (required for track-with-repo creation path).
1.3 Configure two phases:
   - Phase A: type=registration, StartTime + EndTime set (any valid range).
   - Phase B: type=development, StartTime set, **leave EndTime empty** (simulates unscheduled).
1.4 Create a track `e2e-track-1` for this hackathon.
1.5 Navigate to the track repo's `/milestones` page.
1.6 **Screenshot** as `01-track-milestones-post-fix.png`.
1.7 **Assert:** exactly 1 milestone shown (for Phase A); no milestone named `development` and no `1970-01-01` visible.

### Step 2: Template hardening — legacy zero-deadline row

2.1 Stop the server (Ctrl+C).
2.2 Open the worktree's SQLite DB (path from `app.ini`: default `data/gitea.db`):
   ```bash
   sqlite3 data/gitea.db
   ```
2.3 **Verify the milestone table schema first** (NOT NULL columns differ across migrations):
   ```sql
   .schema milestone
   ```
   Expected columns (confirm before inserting): `id`, `repo_id`, `name`, `content`, `is_closed`, `num_issues`, `num_closed_issues`, `completeness`, `is_overdue`, `created_unix`, `updated_unix`, `deadline_unix`, `closed_date_unix`. If additional NOT NULL columns appear that the INSERTs below don't set, add them with sensible defaults (0 / empty string).

2.4 Get the track repo's id:
   ```sql
   SELECT id, name FROM repository WHERE name = 'e2e-track-1';
   ```
   Copy the returned id (an integer like `42`). Substitute it for `42` in 2.5 and 2.6.

2.5 Insert a legacy bad-data milestone:
   ```sql
   INSERT INTO milestone
     (repo_id, name, content, is_closed, num_issues, num_closed_issues,
      completeness, created_unix, updated_unix, deadline_unix, closed_date_unix)
   VALUES
     (42, 'legacy-bad-milestone', '', 0, 0, 0, 0,
      strftime('%s','now'), strftime('%s','now'), 0, 0);
   ```

2.6 Also insert a closed legacy milestone for the ClosedDateUnix guard:
   ```sql
   INSERT INTO milestone
     (repo_id, name, content, is_closed, num_issues, num_closed_issues,
      completeness, created_unix, updated_unix, deadline_unix, closed_date_unix)
   VALUES
     (42, 'legacy-closed-no-close-date', '', 1, 0, 0, 100,
      strftime('%s','now'), strftime('%s','now'), 0, 0);
   ```

2.7 Verify both rows inserted successfully:
   ```sql
   SELECT id, name, deadline_unix, closed_date_unix, is_closed
   FROM milestone
   WHERE name LIKE 'legacy-%';
   ```
   Expected: 2 rows returned with `deadline_unix = 0`.

2.8 Exit sqlite (`.exit`). Restart server.

### Step 3: Verify all three render paths

3.1 Visit `http://localhost:<PORT>/milestones` (user dashboard, open filter). **Screenshot** as `02-dashboard-milestones.png`.
   - **Assert:** `legacy-bad-milestone` row is present. It shows the `no_due_date` locale string (`"No due date"` / `"无截止日期"`). **No `1970-01-01` anywhere on the page.**
3.2 Switch to the closed filter on the same page. **Screenshot** as `03-dashboard-milestones-closed.png`.
   - **Assert:** `legacy-closed-no-close-date` row is present; the closed-date icon section is absent or shows no date. **No `1970-01-01`.**
3.3 Visit `http://localhost:<PORT>/{org}/e2e-track-1/milestones`. **Screenshot** as `04-repo-milestones-list.png`.
   - **Assert:** same result — bad legacy rows render without the epoch date.
3.4 Click into `legacy-bad-milestone`. The URL should be `/{org}/e2e-track-1/milestone/{id}`. **Screenshot** as `05-milestone-issues-page.png`.
   - **Assert:** milestone header does not show a `<span>` with a calendar icon; no `1970-01-01`.

### Step 4: Regression — normal Forgejo milestone path still works

4.1 Create a new repo unrelated to hackathons.
4.2 Go to the repo's milestones page, click **New Milestone**. Leave deadline blank. Save.
4.3 Visit `/milestones` and `/repo/milestones`. **Screenshot** as `06-forgejo-native-blank-deadline.png`.
   - **Assert:** the new milestone shows `no due date` (or equivalently, no date at all). Forgejo's `9999-12-31` sentinel path still works because `AfterLoad` sets `DeadlineString=""` for Year==9999.
4.4 Create another milestone with a normal deadline (say next week). **Screenshot** as `07-forgejo-native-real-deadline.png`.
   - **Assert:** renders the correct deadline.

## Success Criteria

- [ ] Screenshot 01: only scheduled phase produced a milestone (count = 1 not 2)
- [ ] Screenshot 02: dashboard shows legacy bad row without `1970-01-01`
- [ ] Screenshot 03: dashboard closed-filter shows legacy closed row without epoch
- [ ] Screenshot 04: per-repo milestones list shows legacy rows without epoch
- [ ] Screenshot 05: milestone issues page header has no epoch span
- [ ] Screenshot 06: Forgejo native blank-deadline path unaffected
- [ ] Screenshot 07: Forgejo native real-deadline path unaffected

## Failure Modes

If any screenshot shows `1970-01-01`:
- Check which template rendered it (use agent-browser `getHTML` of the element).
- If it's one of the three templates already in scope, re-verify the Edit landed correctly — the bindata cache may be stale; run `TAGS="bindata sqlite sqlite_unlock_notify" make backend` again.
- If it's a fourth template not in scope, stop and update the spec + plan.
