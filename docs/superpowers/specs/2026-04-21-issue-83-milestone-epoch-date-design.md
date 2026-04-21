# Issue #83: Milestones Page Shows All Dates As 1970-01-01

**Issue**: [HackForger #83](https://github.com/HackForger/hackforger/issues/83)
**Priority**: P2 (non-blocking, UI polish)
**Branch**: `fix/issue-83-milestones-epoch-date`

## Problem

On `/milestones` (user dashboard), every milestone row shows `1970-01-01` where a deadline should be. The effect is purely visual but unprofessional.

Screenshot context: HackForger's internal instance at `https://hackforger.inside.h2os.cloud/milestones` after hackathons are created.

## Root Cause

**Not a data corruption bug. Not a template bug. A semantic mismatch between Forgejo and HackForger's milestone creation paths.**

### Forgejo's invariant

`routers/web/repo/milestone.go:136-138` injects a sentinel default when users leave the deadline field blank:

```go
if len(form.Deadline) == 0 {
    form.Deadline = "9999-12-31"
}
```

Result: `DeadlineUnix` is **never zero** in Forgejo-created milestones. The downstream template check `{{if .DeadlineString}}` (which compares against empty string) works because `DeadlineString` is always a real date.

### HackForger breaks the invariant

`services/hackforger/hackathon.go:333-341` auto-derives milestones from hackathon phases without the sentinel:

```go
var deadline timeutil.TimeStamp
if p.EndTime > 0 {
    deadline = timeutil.TimeStamp(p.EndTime)
}
// If p.EndTime == 0, deadline stays zero-value (0)
_ = issues_model.NewMilestone(ctx, &issues_model.Milestone{
    RepoID:       repo.ID,
    Name:         name,
    DeadlineUnix: deadline,
})
```

When `Phase.EndTime == 0` (phase not yet scheduled), `DeadlineUnix = 0` is stored. On load, `models/issues/milestone.go:93-99` runs:

```go
func (m *Milestone) AfterLoad() {
    m.NumOpenIssues = m.NumIssues - m.NumClosedIssues
    if m.DeadlineUnix.Year() == 9999 {  // Forgejo sentinel shortcut
        return
    }
    m.DeadlineString = m.DeadlineUnix.FormatDate()  // → "1970-01-01" when DeadlineUnix==0
    ...
}
```

The `Year() == 9999` shortcut is precisely what makes Forgejo's blank-deadline path render as empty `DeadlineString`. `DeadlineUnix == 0` yields `Year() == 1970` — the shortcut doesn't trigger, and the formatter produces a non-empty `"1970-01-01"` string.

Three templates then check `{{if .DeadlineString}}`:
- `templates/user/dashboard/milestones.tmpl:116` (global `/milestones` dashboard)
- `templates/repo/issue/milestones.tmpl:61` (per-repo milestone list)
- `templates/repo/issue/milestone_issues.tmpl:38` (single-milestone issue list header)

All three pass the guard for `"1970-01-01"` and render the epoch date.

### Why Forgejo upstream doesn't have this bug

In Forgejo, milestones are **user-authored grouping constructs** — a milestone like `"v1.0"` is meaningful even without a deadline (it's a bucket for issues). The `9999-12-31` sentinel is a pragmatic workaround to keep the non-null invariant.

In HackForger, milestones are **auto-derived phase-deadline markers** — a phase without an end time has no meaningful "milestone", it's a degenerate case.

Different semantics → different correct handling.

## Decision: Skip milestone creation for phases without deadlines (Plan B)

**Change in scope**: `services/hackforger/hackathon.go` only. No upstream file changes. No template changes required for the primary fix.

```go
for _, p := range phases {
    if p.EndTime == 0 {
        continue  // No deadline → no milestone (milestones are deadline markers)
    }
    name := p.CustomName
    if name == "" && p.PhaseType != nil {
        name = p.PhaseType.Key
    }
    _ = issues_model.NewMilestone(ctx, &issues_model.Milestone{
        RepoID:       repo.ID,
        Name:         name,
        DeadlineUnix: timeutil.TimeStamp(p.EndTime),
    })
}
```

### Alternatives considered

| Option | Summary | Why rejected |
|---|---|---|
| **A: Sentinel `9999-12-31`** | Match Forgejo UI convention, fill blank EndTime with max-date | UX still shows weird year 9999; creates milestones that have no semantic reality |
| **C: Upstream fix in `models/issues/milestone.go`** | `if !m.DeadlineUnix.IsZero() { m.DeadlineString = ... }` | Violates CLAUDE.md "no upstream changes except injection points"; right long-term but disproportionate for a P2 bug; would need paired upstream PR to avoid merge rot |

### Defensive hardening (in scope, secondary)

Three templates render milestone deadlines and all share the same weak guard. The project's established idiom for zero-guarding timestamps is `{{if ne .X 0}}` — prior art in `templates/shared/issuelist.tmpl:111` (`{{if ne .DeadlineUnix 0}}`) and `templates/repo/issue/view_content/sidebar/due_deadline.tmpl:7,27,31`. Use the same idiom for consistency.

**Template guard changes:**

| File | Current (line) | Fixed |
|---|---|---|
| `templates/user/dashboard/milestones.tmpl` | `{{if .DeadlineString}}` (line 116) | `{{if and .DeadlineString (ne .DeadlineUnix 0)}}` |
| `templates/user/dashboard/milestones.tmpl` | `{{$closedDate:= DateUtils.TimeSince .ClosedDateUnix}}` unguarded (line 112) | wrap in `{{if ne .ClosedDateUnix 0}}` |
| `templates/repo/issue/milestones.tmpl` | `{{if .DeadlineString}}` (line 61) | `{{if and .DeadlineString (ne .DeadlineUnix 0)}}` |
| `templates/repo/issue/milestone_issues.tmpl` | `{{if .Milestone.DeadlineString}}` (line 38) | `{{if and .Milestone.DeadlineString (ne .Milestone.DeadlineUnix 0)}}` |

All three DeadlineString templates already have an `{{else}}` branch (or absence-of-block behavior) that renders nothing or the "no due date" locale string — redirecting into that else-branch is the desired outcome.

**Rationale**: The primary fix (Plan B) removes the data path that creates zero-deadline milestones today, but any legacy rows already in the DB, or any future code path that bypasses the service layer, would still trigger the bug on these three templates. Template guards are defense-in-depth at ~4 lines of change across 3 files.

## Scope

### In scope

1. `services/hackforger/hackathon.go` — skip milestone creation when `Phase.EndTime == 0`
2. `templates/user/dashboard/milestones.tmpl` — zero guards on `ClosedDateUnix` (line 111-114) and `DeadlineString`/`DeadlineUnix` (line 116)
3. `templates/repo/issue/milestones.tmpl` — zero guard on `DeadlineString`/`DeadlineUnix` (line 61)
4. `templates/repo/issue/milestone_issues.tmpl` — zero guard on `.Milestone.DeadlineString`/`.Milestone.DeadlineUnix` (line 38)
5. Unit test for `services/hackforger/hackathon.go` covering the skip-when-zero behavior
6. E2E verification via agent-browser (see Testing)

### Out of scope

- **Backfill of existing bad milestones**. Any milestones already created with `DeadlineUnix=0` will remain in the DB. Template hardening (items 2-4) ensures all three rendering paths fall through to "no due date" / no-render instead of "1970-01-01". A dedicated cleanup migration is not required — the template guards cover every known DeadlineString consumer.
- **Upstream Forgejo patch**. Not pursued in this PR. A separate upstream PR proposing the `IsZero` guard in `models/issues/milestone.go` can be considered later.
- **Milestone creation paths beyond hackathon phases**. If other HackForger services (bounty, grant) ever create milestones directly, they are not covered here. Currently only `hackathon.go:337` calls `NewMilestone`; verified via grep `NewMilestone` across `services/hackforger/`.
- **Phase EndTime validation at hackathon publish time**. A separate concern (HF-series issue) — HackForger already requires phases be scheduled before `PublishHackathon`, so in the common path `EndTime > 0`. The skip-on-zero behavior is defensive for edge cases (draft state, admin-created hackathons, migration data).

## Files Changed

| File | Change | Lines |
|---|---|---|
| `services/hackforger/hackathon.go` | `continue` when `p.EndTime == 0` | ~3 lines added in the phase loop |
| `templates/user/dashboard/milestones.tmpl` | Zero guards on DeadlineString + ClosedDateUnix | ~4 lines modified |
| `templates/repo/issue/milestones.tmpl` | Zero guard on DeadlineString | ~1 line modified |
| `templates/repo/issue/milestone_issues.tmpl` | Zero guard on Milestone.DeadlineString | ~1 line modified |
| `services/hackforger/hackathon_test.go` (existing) | Unit test for skip-on-zero | ~25 lines |

No new files, no migrations, no i18n changes. `services/hackforger/hackathon_test.go` already exists and uses `unittest` + `testify`.

## Testing Plan

### Unit Test

In `services/hackforger/hackathon_test.go`, add a test case:

- Setup: Create a hackathon with 2 phases — one with `EndTime > 0`, one with `EndTime == 0`
- Act: Call `CreateTrackWithRepo` (the path that triggers phase-to-milestone auto-generation)
- Assert:
  - `issues_model.GetMilestones(...)` returns exactly **1** milestone for the track repo (count assertion — primary intent check)
  - The returned milestone's `Name` matches the scheduled phase's name
  - The returned milestone's `DeadlineUnix == phase1.EndTime`
  - No milestone exists with `DeadlineUnix == 0`

### E2E Verification (agent-browser, web-only)

Per CLAUDE.md memory `feedback_e2e_web_mandatory.md`: E2E must use agent-browser at `http://localhost:3000`. Test document location: `docs/tests/e2e/tasks/2026-04-21-issue-83-milestone-epoch-date.md`.

**Steps** (screenshots required at each checkpoint):

1. **Setup**: Start server in worktree (`make frontend && make backend && cp custom/conf/app.ini ...`), log in as `hackforger`/`admin1234`
2. **Reproduce baseline** (pre-fix verification):
   - Navigate to a test hackathon with at least one phase having `EndTime == 0`
   - Trigger track creation (or use existing)
   - Visit `/milestones` → screenshot: confirm at least one milestone shows `1970-01-01`
3. **Apply fix**, rebuild backend, restart server
4. **Verify fix path**:
   - Create a new hackathon with one phase scheduled (`EndTime > 0`) and one phase unscheduled (`EndTime == 0`)
   - Create a track → verify only 1 milestone created on the track repo (not 2)
   - Screenshot: `/milestones` page showing the scheduled phase's milestone with correct date
5. **Verify hardening path** (all three templates):
   - Insert a legacy bad-data milestone via SQL (run against worktree's SQLite DB):
     ```sql
     INSERT INTO milestone (repo_id, name, content, is_closed, num_issues, num_closed_issues, completeness, created_unix, updated_unix, deadline_unix, closed_date_unix)
     VALUES ({TRACK_REPO_ID}, 'legacy-bad-milestone', '', 0, 0, 0, 0, strftime('%s','now'), strftime('%s','now'), 0, 0);
     ```
   - Visit `/milestones` (user dashboard) → screenshot: legacy milestone shows "no due date" text, not "1970-01-01"
   - Visit `/{owner}/{repo}/milestones` (per-repo list) → screenshot: same outcome
   - Visit `/{owner}/{repo}/milestone/{id}` (single-milestone issue list) → screenshot: header shows no deadline span, not "1970-01-01"
6. **Regression check**:
   - Visit `/milestones` with a normal Forgejo-created milestone (deadline set) → screenshot: date renders correctly
   - Visit `/milestones` with a Forgejo-created milestone with blank deadline (becomes `9999-12-31`) → screenshot: should render "9999-12-31" (pre-existing upstream behavior, out of scope)

E2E report template at `docs/tests/e2e/templates/single-feature-test.md` (per CLAUDE.md memory).

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Existing bad milestones (`DeadlineUnix=0`) already in DB | Medium (dev data likely has them) | Low (handled by guards on all three templates) | Template guards cover the 3 known DeadlineString render paths |
| Other services create milestones with zero deadline | Low (grep confirmed only `hackathon.go:337` calls `NewMilestone` in HackForger code) | Medium | Template guards are the safety net |
| Another DeadlineString consumer added later without the guard | Medium over time | Low-Medium | Design note: any new milestone-rendering template must use `{{if ne .DeadlineUnix 0}}` pattern — matches project convention in `issuelist.tmpl` and `due_deadline.tmpl` |
| Phase EndTime gets set to 0 after milestone already created (via edit path) | Low | Low | Out of scope; separate issue if observed |
| Upstream Forgejo template merge adds a fourth DeadlineString render site | Low | Medium | Re-audit on next upstream merge; the `{{if ne .DeadlineUnix 0}}` idiom is already canonical upstream (used in `issuelist.tmpl`) |

## Success Criteria

1. After fix, `/milestones` page shows no `1970-01-01` entries for any milestone — verified by agent-browser screenshot
2. New hackathons with unscheduled phases don't produce placeholder milestones — verified by unit test + agent-browser
3. Existing bad milestones in DB render as "no due date" instead of `1970-01-01` — verified by agent-browser
4. Forgejo-native milestone creation flow is unaffected — verified by agent-browser regression check
