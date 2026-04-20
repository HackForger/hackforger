# API ↔ Web parity gaps

Backlog of web-facing operations that have **no equivalent `/api/v1/hackforger/...` endpoint**. Per the project principle (every web user action must have an API equivalent), these should be filed and closed.

## Real gaps

> ✅ All 4 previously-listed gaps closed in PR #73 (commit `feat(api): close 4 hackforger API parity gaps`). The endpoints are now documented in:
>
> - `attachments.md` — `POST /hackforger/attachments`
> - `feed-search-reputation.md` — `GET/PUT /hackforger/admin/reputation/settings`
> - `hackathons.md` — phase type catalog CRUD (`/hackforger/admin/phase-types`)
> - `orgs.md` — `POST /hackforger/orgs/{org}/join-request`
>
> Backlog is empty as of 2026-04-20.

## Dead routes (clean-up, not parity gaps)

These web routes exist but the handler does not implement them. They flash "unknown_action" and redirect:

| Route | Handler | Status |
|-------|---------|--------|
| `POST /hackathon/{slug}/manage/start` | `ManagePhasePost` | Switch only handles `publish` and `cancel`; `start` falls to default error. |
| `POST /hackathon/{slug}/manage/judge` | `ManagePhasePost` | Same as above. |

**Why this happens:** hackathon phase transitions (Open → Hacking → Judging → Finished) are **time-driven** by the gocron scheduler (`services/cron/tasks_hackforger.go` registers `hackforger_hackathon_status` @ every 5min, and `onPhaseEvent` fires on phase start/end). Manual transition is not a supported operation.

**Recommended cleanup:**
- Remove the `/start` and `/judge` route registrations from `routers/web/web.go` (lines 546–547).
- Remove "开始 Hacking" / "开始评审" buttons from `templates/hackforger/hackathon/manage.tmpl` (or whatever invokes them).
- Update E2E tasks `full-cycle/01-hackathon-lifecycle.md` (Step 3.1) and `04-submission-judging.md` (Step 7.1) to reflect time-driven advancement: configure short phase durations during E2E setup and wait for the scheduler.

## How to consume this list

When implementing a missing API:
1. Add the route to `routers/api/v1/api.go` under the `/hackforger` group.
2. Implement the handler in `routers/api/v1/hackforger/<module>.go` reusing the existing service-layer function (the web handler likely wraps the same service call).
3. Add the new endpoint to the relevant module reference in this directory.
4. Remove the row from this gaps file.
