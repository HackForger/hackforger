# Issue #83 Milestones 1970-01-01 Fix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop HackForger's auto-generated milestones from displaying `1970-01-01` when a phase has no `EndTime`, and harden all three milestone-rendering templates so any legacy zero-deadline row falls through to "no due date".

**Architecture:** Two-layer fix. (1) Service layer: refactor the phase→milestone conversion in `services/hackforger/hackathon.go:324-343` into a pure, unit-testable helper `phasesToMilestones`, which skips phases with `EndTime == 0`. (2) Template layer: add `{{if ne .DeadlineUnix 0}}` zero-guards in three templates, matching the project's established idiom from `templates/shared/issuelist.tmpl`.

**Tech Stack:** Go 1.22, XORM, Go html/template, testify (unit tests), agent-browser (E2E).

**Worktree:** `/Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch`
**Branch:** `fix/issue-83-milestones-epoch-date`
**Spec:** `docs/superpowers/specs/2026-04-21-issue-83-milestone-epoch-date-design.md`

---

## File Manifest

**Created:**
- `services/hackforger/hackathon_internal_test.go` — white-box test for unexported `phasesToMilestones`
- `docs/tests/e2e/tasks/2026-04-21-issue-83-milestones-epoch-date.md` — E2E task for manual agent-browser verification
- `docs/tests/e2e/reports/2026-04-21-issue-83-milestones-epoch-date-report.md` — E2E report (populated in Task 8)

**Modified:**
- `services/hackforger/hackathon.go` — extract `phasesToMilestones`; rewrite loop at line 324-343 to use it
- `templates/user/dashboard/milestones.tmpl` — guards on lines ~111 (ClosedDateUnix) and ~116 (DeadlineString + DeadlineUnix)
- `templates/repo/issue/milestones.tmpl` — guard on line ~61 (DeadlineString + DeadlineUnix)
- `templates/repo/issue/milestone_issues.tmpl` — guard on line ~38 (.Milestone.DeadlineString + .Milestone.DeadlineUnix)

No new imports expected (`timeutil` already imported in hackathon.go; no new template helpers needed).

---

## Task 1: Write Failing Test for `phasesToMilestones` Helper

**Files:**
- Create: `services/hackforger/hackathon_internal_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/hackforger/hackathon_internal_test.go` with this exact content:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// White-box test (same package) for internal helpers.

package hackforger

import (
	"testing"

	hackforger_model "forgejo.org/models/hackforger"

	"github.com/stretchr/testify/assert"
)

func TestPhasesToMilestones_SkipsPhasesWithoutEndTime(t *testing.T) {
	regType := &hackforger_model.PhaseType{Key: "registration"}
	devType := &hackforger_model.PhaseType{Key: "development"}

	phases := []*hackforger_model.Phase{
		{PhaseType: regType, StartTime: 1_700_000_000, EndTime: 1_700_100_000}, // scheduled
		{PhaseType: devType, StartTime: 1_700_100_000, EndTime: 0},             // unscheduled — must be skipped
	}

	got := phasesToMilestones(phases, 42)

	assert.Len(t, got, 1, "phases with EndTime==0 must not produce milestones")
	assert.Equal(t, int64(42), got[0].RepoID)
	assert.Equal(t, "registration", got[0].Name)
	assert.Equal(t, int64(1_700_100_000), int64(got[0].DeadlineUnix))
}

func TestPhasesToMilestones_UsesCustomNameWhenSet(t *testing.T) {
	phases := []*hackforger_model.Phase{
		{CustomName: "Final Submission", StartTime: 1, EndTime: 2},
	}

	got := phasesToMilestones(phases, 1)

	assert.Len(t, got, 1)
	assert.Equal(t, "Final Submission", got[0].Name,
		"CustomName should take precedence over PhaseType.Key")
}

func TestPhasesToMilestones_FallsBackToPhaseTypeKey(t *testing.T) {
	phases := []*hackforger_model.Phase{
		{PhaseType: &hackforger_model.PhaseType{Key: "judging"}, StartTime: 1, EndTime: 2},
	}

	got := phasesToMilestones(phases, 1)

	assert.Len(t, got, 1)
	assert.Equal(t, "judging", got[0].Name)
}

func TestPhasesToMilestones_EmptyInput(t *testing.T) {
	got := phasesToMilestones(nil, 1)
	assert.Len(t, got, 0)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch && go test ./services/hackforger/ -run TestPhasesToMilestones -v 2>&1 | tail -30`

Expected: compilation FAILS with `undefined: phasesToMilestones`.

- [ ] **Step 3: Commit the failing test**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch
git add services/hackforger/hackathon_internal_test.go
git commit -m "$(cat <<'EOF'
test(hackforger): add failing tests for phasesToMilestones helper

Covers the four behaviors the fix needs to guarantee:
- Phase with EndTime==0 is skipped (the bug fix)
- CustomName takes precedence over PhaseType.Key
- PhaseType.Key is used when CustomName is empty
- Empty input produces empty output

Refs: #83

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Implement `phasesToMilestones` + Wire Into `CreateTrackWithRepo`

**Files:**
- Modify: `services/hackforger/hackathon.go` — add helper, replace inline loop

- [ ] **Step 1: Add the helper function**

In `services/hackforger/hackathon.go`, just before `func CreateTrackWithRepo` (currently line 250 — use Grep to locate), insert:

```go
// phasesToMilestones converts hackathon phases into Milestone records for the
// track repo. Phases with EndTime == 0 (unscheduled) produce no milestone —
// a milestone without a deadline has no semantic meaning in HackForger's
// phase-marker model (unlike Forgejo's user-authored milestones, which are
// meaningful as issue-grouping buckets even without deadlines).
func phasesToMilestones(phases []*hackforger_model.Phase, repoID int64) []*issues_model.Milestone {
	result := make([]*issues_model.Milestone, 0, len(phases))
	for _, p := range phases {
		if p.EndTime == 0 {
			continue
		}
		name := p.CustomName
		if name == "" && p.PhaseType != nil {
			name = p.PhaseType.Key // "registration", "development", "judging", "results"
		}
		result = append(result, &issues_model.Milestone{
			RepoID:       repoID,
			Name:         name,
			DeadlineUnix: timeutil.TimeStamp(p.EndTime),
		})
	}
	return result
}
```

- [ ] **Step 2: Rewrite the loop in `CreateTrackWithRepo`**

Replace the block at lines 324-343 (the current auto-create-phase-milestones block) with:

```go
	// Auto-create phase milestones on the track repo from Phase records.
	// Phases without EndTime are skipped; see phasesToMilestones for rationale.
	phases, phaseErr := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
	if phaseErr == nil {
		for _, m := range phasesToMilestones(phases, repo.ID) {
			_ = issues_model.NewMilestone(ctx, m)
		}
	}
	return nil
}
```

Use the Edit tool; the exact `old_string` to match (from line 324 through line 345) is:

```
	// Auto-create phase milestones on the track repo from Phase records.
	// Each phase becomes a milestone; deadline = phase EndTime.
	phases, phaseErr := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
	if phaseErr == nil {
		for _, p := range phases {
			name := p.CustomName
			if name == "" && p.PhaseType != nil {
				name = p.PhaseType.Key // "registration", "development", "judging", "results"
			}
			var deadline timeutil.TimeStamp
			if p.EndTime > 0 {
				deadline = timeutil.TimeStamp(p.EndTime)
			}
			_ = issues_model.NewMilestone(ctx, &issues_model.Milestone{
				RepoID:       repo.ID,
				Name:         name,
				DeadlineUnix: deadline,
			})
		}
	}
	return nil
}
```

- [ ] **Step 3: Run the unit tests — expect PASS**

Run: `cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch && go test ./services/hackforger/ -run TestPhasesToMilestones -v 2>&1 | tail -30`

Expected: all four test cases `PASS`.

- [ ] **Step 4: Run the full hackforger service test suite — no regressions**

Run: `cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch && go test ./services/hackforger/... 2>&1 | tail -20`

Expected: `ok  forgejo.org/services/hackforger ...` (all tests pass, including the pre-existing `TestPublishHackathon_NoCriteria`).

- [ ] **Step 5: Commit**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch
git add services/hackforger/hackathon.go
git commit -m "$(cat <<'EOF'
fix(hackforger): skip milestone creation for phases without EndTime

Root cause: the phase-to-milestone loop in CreateTrackWithRepo stored
DeadlineUnix=0 for any phase with EndTime=0 (unscheduled), which rendered
as 1970-01-01 in all three milestone templates.

Refactor the loop into pure phasesToMilestones helper that returns the
list of milestones to create, with a single-point skip-when-zero rule.
Call site becomes a clean two-liner.

Fixes: #83 (service layer)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Template Guard — User Dashboard Milestones

**Files:**
- Modify: `templates/user/dashboard/milestones.tmpl` (lines 111-119)

- [ ] **Step 1: Apply the guard on `ClosedDateUnix` (line 111-114)**

Use the Edit tool:

**old_string:**
```
										<div class="flex-text-block">
											{{if .IsClosed}}
												{{$closedDate:= DateUtils.TimeSince .ClosedDateUnix}}
												{{svg "octicon-clock" 14}}
												{{ctx.Locale.Tr "repo.milestones.closed" $closedDate}}
											{{else}}
```

**new_string:**
```
										<div class="flex-text-block">
											{{if .IsClosed}}
												{{if ne .ClosedDateUnix 0}}
													{{$closedDate:= DateUtils.TimeSince .ClosedDateUnix}}
													{{svg "octicon-clock" 14}}
													{{ctx.Locale.Tr "repo.milestones.closed" $closedDate}}
												{{end}}
											{{else}}
```

- [ ] **Step 2: Apply the guard on `DeadlineString` (line 116)**

Use the Edit tool:

**old_string:**
```
												{{if .DeadlineString}}
													<span class="flex-text-inline {{if .IsOverdue}}text red{{end}}">
														{{svg "octicon-calendar" 14}}
														{{DateUtils.AbsoluteShort (.DeadlineString|DateUtils.ParseLegacy)}}
													</span>
												{{else}}
```

**new_string:**
```
												{{if and .DeadlineString (ne .DeadlineUnix 0)}}
													<span class="flex-text-inline {{if .IsOverdue}}text red{{end}}">
														{{svg "octicon-calendar" 14}}
														{{DateUtils.AbsoluteShort (.DeadlineString|DateUtils.ParseLegacy)}}
													</span>
												{{else}}
```

- [ ] **Step 3: Visually verify the edit by re-reading lines 104-130**

Run the Read tool on `templates/user/dashboard/milestones.tmpl` offset 104 limit 30.

Expected structure (simplified):
```
{{if .UpdatedUnix}} ... {{end}}                    ← unchanged
<div>
  {{if .IsClosed}}
    {{if ne .ClosedDateUnix 0}}  ← NEW guard
      ... closed date render ...
    {{end}}                      ← NEW
  {{else}}
    {{if and .DeadlineString (ne .DeadlineUnix 0)}}  ← TIGHTENED guard
      ... deadline render ...
    {{else}}
      {{ctx.Locale.Tr "repo.milestones.no_due_date"}}
    {{end}}
  {{end}}
</div>
```

- [ ] **Step 4: Commit**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch
git add templates/user/dashboard/milestones.tmpl
git commit -m "$(cat <<'EOF'
fix(template): guard milestones dashboard against zero timestamps

Wrap ClosedDateUnix render in {{if ne .ClosedDateUnix 0}} and tighten
DeadlineString check with (ne .DeadlineUnix 0). Matches the project's
established zero-guard idiom from templates/shared/issuelist.tmpl.

Defense-in-depth for legacy rows with DeadlineUnix=0 already in the DB.

Refs: #83

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Template Guard — Repo Milestones List

**Files:**
- Modify: `templates/repo/issue/milestones.tmpl` (line 61)

- [ ] **Step 1: Apply the guard**

Use the Edit tool:

**old_string:**
```
									{{if .DeadlineString}}
										<span class="flex-text-inline {{if .IsOverdue}}text red{{end}}">
											{{svg "octicon-calendar" 14}}
											{{DateUtils.AbsoluteShort (.DeadlineString|DateUtils.ParseLegacy)}}
										</span>
```

**new_string:**
```
									{{if and .DeadlineString (ne .DeadlineUnix 0)}}
										<span class="flex-text-inline {{if .IsOverdue}}text red{{end}}">
											{{svg "octicon-calendar" 14}}
											{{DateUtils.AbsoluteShort (.DeadlineString|DateUtils.ParseLegacy)}}
										</span>
```

- [ ] **Step 2: Verify by re-reading lines 55-70**

Run the Read tool on `templates/repo/issue/milestones.tmpl` offset 55 limit 20.

Expected: the guard is now `{{if and .DeadlineString (ne .DeadlineUnix 0)}}` on line 61. Else-branch unchanged.

- [ ] **Step 3: Commit**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch
git add templates/repo/issue/milestones.tmpl
git commit -m "$(cat <<'EOF'
fix(template): guard per-repo milestones list against zero timestamps

Tighten DeadlineString check with (ne .DeadlineUnix 0) so legacy rows
with DeadlineUnix=0 fall through to the no-deadline branch instead
of rendering 1970-01-01.

Refs: #83

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Template Guard — Milestone Issue List Header

**Files:**
- Modify: `templates/repo/issue/milestone_issues.tmpl` (line 38)

- [ ] **Step 1: Apply the guard**

Use the Edit tool:

**old_string:**
```
						{{if .Milestone.DeadlineString}}
							<span{{if .IsOverdue}} class="text red"{{end}}>
								{{svg "octicon-calendar"}}
								{{DateUtils.AbsoluteShort (.Milestone.DeadlineString|DateUtils.ParseLegacy)}}
							</span>
```

**new_string:**
```
						{{if and .Milestone.DeadlineString (ne .Milestone.DeadlineUnix 0)}}
							<span{{if .IsOverdue}} class="text red"{{end}}>
								{{svg "octicon-calendar"}}
								{{DateUtils.AbsoluteShort (.Milestone.DeadlineString|DateUtils.ParseLegacy)}}
							</span>
```

- [ ] **Step 2: Verify by re-reading lines 35-45**

Run the Read tool on `templates/repo/issue/milestone_issues.tmpl` offset 35 limit 12.

Expected: the guard on line 38 is `{{if and .Milestone.DeadlineString (ne .Milestone.DeadlineUnix 0)}}`.

- [ ] **Step 3: Commit**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch
git add templates/repo/issue/milestone_issues.tmpl
git commit -m "$(cat <<'EOF'
fix(template): guard milestone issue list header against zero timestamps

Same zero-guard idiom as dashboard and repo milestone list — covers
the third and final DeadlineString render site in the repo.

Refs: #83

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Build + Full Test Pass

**Files:** none modified

- [ ] **Step 1: Frontend build (per CLAUDE.md memory `feedback_worktree_frontend.md`)**

Run: `cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch && make frontend 2>&1 | tail -10`

Expected: no errors. (Frontend rebuild is required in worktrees because `public/assets/` is gitignored.)

- [ ] **Step 2: Backend build**

Run: `cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch && TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -10`

Expected: binary produced at `./gitea`. If template syntax errors are present, the bindata embed step fails here — fix the template before proceeding.

- [ ] **Step 3: Run model + service tests**

Run: `cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch && go test ./services/hackforger/... ./models/hackforger/... ./models/issues/... 2>&1 | tail -30`

Expected: all `ok`. The milestone model tests in `models/issues/milestone_test.go` remain untouched, so they should continue to pass.

- [ ] **Step 4: Commit (if any build artifacts need committing — unlikely)**

If `make frontend` produced generated files in `web_src/` or similar that should be tracked (most are gitignored), commit them. Otherwise skip.

---

## Task 7: Create E2E Task Document

**Files:**
- Create: `docs/tests/e2e/tasks/2026-04-21-issue-83-milestones-epoch-date.md`

- [ ] **Step 1: Copy the single-feature template and fill it**

First read the template to copy its structure: `docs/tests/e2e/templates/single-feature-test.md`.

Then write `docs/tests/e2e/tasks/2026-04-21-issue-83-milestones-epoch-date.md` with this content:

````markdown
# E2E Test: Issue #83 — Milestones Page 1970-01-01 Fix

**Related issue:** [HackForger #83](https://github.com/HackForger/hackforger/issues/83)
**Plan:** `docs/superpowers/plans/2026-04-21-issue-83-milestones-epoch-date.md`
**Tester:** Claude + agent-browser
**Environment:** Worktree instance at `http://localhost:3000` (web-only per CLAUDE.md memory `feedback_e2e_web_mandatory.md`).

## Preconditions

1. Worktree built: `make frontend && TAGS="bindata sqlite sqlite_unlock_notify" make backend`.
2. `custom/conf/app.ini` copied from the main repo (CLAUDE.md memory `feedback_worktree_config.md`).
3. `data/queues/common/LOCK` removed if server was running previously.
4. Start server: `./gitea web` (worktree, port 3000 per app.ini).
5. Login as `hackforger` / `admin1234`.

## Test Steps

### Step 1: Baseline — service-layer fix (no new bad rows)

1.1 Navigate to `/` → create a new hackathon titled `e2e-issue83`.
1.2 Link an org (required for track-with-repo creation path).
1.3 Configure two phases:
   - Phase A: type=registration, StartTime + EndTime set (any valid range).
   - Phase B: type=development, StartTime set, **leave EndTime empty** (simulates unscheduled).
1.4 Create a track `e2e-track-1` for this hackathon.
1.5 Navigate to the track repo's `/milestones` page.
1.6 **Screenshot** as `screenshots/01-track-milestones-post-fix.png`.
1.7 **Assert:** exactly 1 milestone shown (for Phase A); no milestone named `development` and no `1970-01-01` visible.

### Step 2: Template hardening — legacy zero-deadline row

2.1 Stop the server (Ctrl+C).
2.2 Open the worktree's SQLite DB (path from `app.ini`: default `data/gitea.db`):
   ```bash
   sqlite3 data/gitea.db
   ```
2.3 Get the track repo's id:
   ```sql
   SELECT id, name FROM repository WHERE name = 'e2e-track-1';
   ```
   Note the id (call it `TRACK_REPO_ID`).
2.4 Insert a legacy bad-data milestone:
   ```sql
   INSERT INTO milestone
     (repo_id, name, content, is_closed, num_issues, num_closed_issues,
      completeness, created_unix, updated_unix, deadline_unix, closed_date_unix)
   VALUES
     (TRACK_REPO_ID, 'legacy-bad-milestone', '', 0, 0, 0, 0,
      strftime('%s','now'), strftime('%s','now'), 0, 0);
   ```
2.5 Also insert a closed legacy milestone for the ClosedDateUnix guard:
   ```sql
   INSERT INTO milestone
     (repo_id, name, content, is_closed, num_issues, num_closed_issues,
      completeness, created_unix, updated_unix, deadline_unix, closed_date_unix)
   VALUES
     (TRACK_REPO_ID, 'legacy-closed-no-close-date', '', 1, 0, 0, 100,
      strftime('%s','now'), strftime('%s','now'), 0, 0);
   ```
2.6 Exit sqlite (`.exit`). Restart server.

### Step 3: Verify all three render paths

3.1 Visit `http://localhost:3000/milestones` (user dashboard, open filter). **Screenshot** as `02-dashboard-milestones.png`.
   - **Assert:** `legacy-bad-milestone` row is present. It shows the `no_due_date` locale string (`"No due date"` / `"无截止日期"`). **No `1970-01-01` anywhere on the page.**
3.2 Switch to the closed filter on the same page. **Screenshot** as `03-dashboard-milestones-closed.png`.
   - **Assert:** `legacy-closed-no-close-date` row is present; the closed-date icon section is absent or shows no date. **No `1970-01-01`.**
3.3 Visit `http://localhost:3000/{org}/e2e-track-1/milestones`. **Screenshot** as `04-repo-milestones-list.png`.
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
````

- [ ] **Step 2: Commit**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch
git add docs/tests/e2e/tasks/2026-04-21-issue-83-milestones-epoch-date.md
git commit -m "$(cat <<'EOF'
docs(e2e): add E2E task for issue #83 milestones epoch fix

Web-only via agent-browser at localhost:3000. Covers service-layer
fix (new hackathons) + template hardening (legacy DB rows) +
Forgejo-native regression check.

Refs: #83

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Execute E2E + Write Report

**Files:**
- Create: `docs/tests/e2e/reports/2026-04-21-issue-83-milestones-epoch-date-report.md`

- [ ] **Step 1: Start the server**

Ensure app.ini copied (per CLAUDE.md memory). Run:
```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch
rm -f data/queues/common/LOCK
./gitea web > /tmp/gitea-e2e-83.log 2>&1 &
```

Wait ~3 seconds; verify: `curl -s http://localhost:3000/ | head -c 200` returns HTML.

- [ ] **Step 2: Execute the E2E steps using agent-browser**

Use the `agent-browser:agent-browser` skill. Take all 7 screenshots listed in Task 7. Store them under `docs/tests/e2e/reports/screenshots/2026-04-21-issue-83/`.

For each step, agent-browser reports the HTML of the relevant row/element — explicitly grep it for `1970-01-01` and record pass/fail.

- [ ] **Step 3: Write the E2E report**

Create `docs/tests/e2e/reports/2026-04-21-issue-83-milestones-epoch-date-report.md` with this structure:

```markdown
# E2E Report: Issue #83 Milestones Epoch Fix

**Run date:** 2026-04-21
**Branch:** fix/issue-83-milestones-epoch-date
**Plan:** docs/superpowers/plans/2026-04-21-issue-83-milestones-epoch-date.md
**Task:** docs/tests/e2e/tasks/2026-04-21-issue-83-milestones-epoch-date.md

## Summary

| Checkpoint | Screenshot | Result |
|---|---|---|
| 1 — Service skip-on-zero | 01-track-milestones-post-fix.png | PASS/FAIL |
| 2 — Dashboard legacy open | 02-dashboard-milestones.png | PASS/FAIL |
| 3 — Dashboard legacy closed | 03-dashboard-milestones-closed.png | PASS/FAIL |
| 4 — Repo milestones list | 04-repo-milestones-list.png | PASS/FAIL |
| 5 — Milestone issues header | 05-milestone-issues-page.png | PASS/FAIL |
| 6 — Forgejo blank-deadline regression | 06-forgejo-native-blank-deadline.png | PASS/FAIL |
| 7 — Forgejo real-deadline regression | 07-forgejo-native-real-deadline.png | PASS/FAIL |

## Detailed Findings

[For each failing checkpoint, one paragraph: what was observed, HTML snippet showing the offending render, root cause hypothesis, next step.]

## Verdict

[PASS — all screenshots match expectations, safe to merge]
OR
[FAIL — N checkpoints failed; fixes required before merge. Details above.]
```

- [ ] **Step 4: Stop the server**

```bash
pkill -f 'gitea web'
```

- [ ] **Step 5: Commit**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch
git add docs/tests/e2e/reports/
git commit -m "$(cat <<'EOF'
docs(e2e): report for issue #83 milestones epoch fix

Screenshots + per-checkpoint PASS/FAIL. See report for verdict.

Refs: #83

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Post-Implementation

After all tasks complete and E2E passes:

1. Summarize all commits: `git log v0.1-dev/hackforger..HEAD --oneline`
2. Present merge options to user via `superpowers:finishing-a-development-branch` skill (PR vs. direct merge).

## Rollback

If any task fails irrecoverably:

1. All work is on branch `fix/issue-83-milestones-epoch-date` — no changes to `v0.1-dev/hackforger`.
2. `git checkout v0.1-dev/hackforger` in the main repo returns to untouched baseline.
3. Delete the worktree: `git worktree remove /Users/h2oslabs/.config/superpowers/worktrees/hackforger/fix-83-milestones-epoch` and `git branch -D fix/issue-83-milestones-epoch-date`.
