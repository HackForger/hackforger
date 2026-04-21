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

When `Phase.EndTime == 0` (phase not yet scheduled), `DeadlineUnix = 0` is stored. On load, `models/issues/milestone.go:99` formats it:

```go
m.DeadlineString = m.DeadlineUnix.FormatDate()  // → "1970-01-01" for Unix epoch
```

Then `templates/user/dashboard/milestones.tmpl:116` checks `{{if .DeadlineString}}` — a non-empty `"1970-01-01"` passes the guard and renders as a deadline.

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

Add `IsZero` guards to `templates/user/dashboard/milestones.tmpl` to protect against any future code path that stores zero timestamps:

- **Line 111-112**: guard `ClosedDateUnix` before `DateUtils.TimeSince`:
  ```go-tmpl
  {{if .IsClosed}}
      {{if not .ClosedDateUnix.IsZero}}
          {{$closedDate:= DateUtils.TimeSince .ClosedDateUnix}}
          {{svg "octicon-clock" 14}}
          {{ctx.Locale.Tr "repo.milestones.closed" $closedDate}}
      {{end}}
  {{else}}
      ...
  {{end}}
  ```

- **Line 116**: tighten DeadlineString check with explicit `DeadlineUnix.IsZero` negation (belt-and-suspenders in case another service path ever stores `DeadlineUnix=0`):
  ```go-tmpl
  {{if and .DeadlineString (not .DeadlineUnix.IsZero)}}
  ```

Template lines 116-123 already handle the "no deadline" else-branch correctly (renders "no due date"); the tightened conditions redirect into that else branch instead of rendering epoch.

**Rationale for the hardening**: The primary fix (Plan B) removes the data path that produces zero timestamps today. The template guards are defense-in-depth for future code paths — low cost, high future-proofing value.

## Scope

### In scope

1. `services/hackforger/hackathon.go` — skip milestone creation when `Phase.EndTime == 0`
2. `templates/user/dashboard/milestones.tmpl` — `IsZero` guards on `ClosedDateUnix` (line 111-114) and `DeadlineString`/`DeadlineUnix` (line 116)
3. Unit test for `services/hackforger/hackathon.go` covering the skip-when-zero behavior
4. E2E verification via agent-browser (see Testing)

### Out of scope

- **Backfill of existing bad milestones**. Any milestones already created with `DeadlineUnix=0` will remain in the DB. Template hardening (item 2) ensures they render correctly as "no due date" instead of "1970-01-01", which is the user-visible fix. A dedicated cleanup migration is not required — the template guard is sufficient.
- **Upstream Forgejo patch**. Not pursued in this PR. A separate upstream PR proposing the `IsZero` guard in `models/issues/milestone.go` can be considered later.
- **Milestone creation paths beyond hackathon phases**. If other HackForger services (bounty, grant) ever create milestones directly, they are not covered here. Currently only `hackathon.go:337` calls `NewMilestone`; verified via grep `NewMilestone` across `services/hackforger/`.
- **Phase EndTime validation at hackathon publish time**. A separate concern (HF-series issue) — HackForger already requires phases be scheduled before `PublishHackathon`, so in the common path `EndTime > 0`. The skip-on-zero behavior is defensive for edge cases (draft state, admin-created hackathons, migration data).

## Files Changed

| File | Change | Lines |
|---|---|---|
| `services/hackforger/hackathon.go` | `continue` when `p.EndTime == 0` | ~3 lines added in the phase loop |
| `templates/user/dashboard/milestones.tmpl` | Add `IsZero` guards | ~4 lines modified |
| `services/hackforger/hackathon_test.go` (new or existing) | Unit test for skip-on-zero | ~20 lines |

No new files, no migrations, no i18n changes.

## Testing Plan

### Unit Test

In `services/hackforger/hackathon_test.go`, add a test case:

- Setup: Create a hackathon with 2 phases — one with `EndTime > 0`, one with `EndTime == 0`
- Act: Call the track-creation path that triggers phase-to-milestone auto-generation
- Assert: Exactly 1 milestone exists on the track repo; milestone's `DeadlineUnix == phase1.EndTime`; no milestone with `DeadlineUnix == 0`

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
5. **Verify hardening path**:
   - Manually insert a milestone with `DeadlineUnix=0` and `IsClosed=false` into DB (simulates legacy bad data)
   - Visit `/milestones` → screenshot: confirm milestone shows "no due date" text, not "1970-01-01"
6. **Regression check**:
   - Visit `/milestones` with a normal Forgejo-created milestone (deadline set) → screenshot: date renders correctly
   - Visit `/milestones` with a Forgejo-created milestone with blank deadline (becomes `9999-12-31`) → screenshot: should render "9999-12-31" (pre-existing upstream behavior, out of scope)

E2E report template at `docs/tests/e2e/templates/single-feature-test.md` (per CLAUDE.md memory).

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Existing bad milestones (`DeadlineUnix=0`) already in DB | Medium (dev data likely has them) | Low (now handled by template guard) | Template guards cover this |
| Other services create milestones with zero deadline | Low (grep confirmed only `hackathon.go` does) | Medium | Template guards are safety net |
| Phase EndTime gets set to 0 after milestone already created (via edit path) | Low | Low | Out of scope; separate issue if observed |
| Upstream Forgejo later changes template format | Low | Medium | Hardening uses `.FieldName.IsZero` which is a stable `timeutil.TimeStamp` method |

## Success Criteria

1. After fix, `/milestones` page shows no `1970-01-01` entries for any milestone — verified by agent-browser screenshot
2. New hackathons with unscheduled phases don't produce placeholder milestones — verified by unit test + agent-browser
3. Existing bad milestones in DB render as "no due date" instead of `1970-01-01` — verified by agent-browser
4. Forgejo-native milestone creation flow is unaffected — verified by agent-browser regression check
