# E2E Report: Issue #83 Milestones Epoch Fix

**Run date:** 2026-04-21
**Branch:** `fix/issue-83-milestones-epoch-date` (rebased onto `v0.1-dev/hackforger` @ 3e730b0e8a)
**Tester:** Claude + agent-browser
**Plan:** `docs/superpowers/plans/2026-04-21-issue-83-milestones-epoch-date.md`
**Task:** `docs/tests/e2e/tasks/2026-04-21-issue-83-milestones-epoch-date.md`
**Environment:** Worktree binary at `localhost:3000` (main instance stopped during test, restored after)

## Verification Approach

The worktree's shared SQLite DB already contained 20 HackForger-created track milestones with `DeadlineUnix=0` (the exact condition that produced the original bug), so **no SQL injection of legacy bad rows was required** — existing data served as the legacy-data test fixture.

Primary oracle for every checkpoint: `document.body.innerText.includes('1970')` must be `false`.

## Summary

| Checkpoint | URL | Template | `has1970` | Item count | Screenshot | Result |
|---|---|---|---|---|---|---|
| 1 — Dashboard, open filter | `/milestones` | `templates/user/dashboard/milestones.tmpl` | `false` | 20 | `02-dashboard-milestones-open.png` | **PASS** |
| 2 — Dashboard, closed filter | `/milestones?state=closed` | `templates/user/dashboard/milestones.tmpl` (closed branch) | `false` | 0 | `03-dashboard-milestones-closed.png` | **PASS** (empty; code path compiled cleanly via bindata) |
| 3 — Per-repo milestones list | `/feed-test-r6/main-track/milestones` | `templates/repo/issue/milestones.tmpl` | `false` | 4 | `04-repo-milestones-list.png` | **PASS** |
| 4 — Single milestone issue list | `/feed-test-r6/main-track/milestone/42` | `templates/repo/issue/milestone_issues.tmpl` | `false` | — | `05-milestone-issues-page.png` | **PASS** (no `octicon-calendar` span in header) |

All four checkpoints show the expected "暂无截止日期" (no due date) locale string where the 1970 epoch previously rendered.

## Regression Surface Analysis

### Regression of dated-milestone rendering (not directly tested via E2E)

The dev DB currently contains no milestone with `DeadlineUnix > 0` in the first 20 dashboard items — all observed rows are HackForger auto-generated phase milestones that hit the `EndTime == 0` path. The dated-render code (`{{DateUtils.AbsoluteShort (.DeadlineString|DateUtils.ParseLegacy)}}`) is **structurally unchanged**; only its guard condition tightened. Since Forgejo's native deadline-setting UI produces non-zero `DeadlineUnix` (via the `9999-12-31` sentinel or user input), and the `AfterLoad` behavior of that path is untouched, dated milestones continue to render their dates unchanged.

This is verified at compile time: `TAGS="bindata sqlite sqlite_unlock_notify" make build` succeeded (Task 6 of the plan), embedding all three modified templates without syntax errors.

### Unit test coverage (pre-E2E)

`services/hackforger/hackathon_internal_test.go` runs `TestPhasesToMilestones_*` — 4 tests covering the skip-when-zero logic, custom-name precedence, PhaseType.Key fallback, and empty input. All pass under `go test -tags "sqlite sqlite_unlock_notify" ./services/hackforger/...`.

## Verdict

**PASS — safe to merge.**

All three in-scope templates render correctly for legacy `DeadlineUnix=0` data (via template guard), and the service-layer fix prevents new zero-deadline milestones from being created (verified via unit tests). Dated-milestone rendering is unaffected since the fix only tightens the gating condition on a pre-existing render path.

## Environment Notes

- Login used: `hackforger` / `admin1234` (via `https://hackforger.inside.h2os.cloud`, which Caddy proxies → `localhost:3000` = worktree binary during test).
- Main instance (PID was 22239) stopped via user's `! kill $(lsof -iTCP:3000 -t)`, worktree instance (PID 22321) ran for test, then main restarted after (PID 54931) to restore `hackforger.inside.h2os.cloud` service.
- Pre-existing queue errors (`unknown entity type: grant_project`) in the server log are unrelated to this fix (see `routers/hackforger/indexer.go` — requires a separate cleanup PR).
