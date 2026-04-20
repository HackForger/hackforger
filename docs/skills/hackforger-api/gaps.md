# API ↔ Web parity gaps

Backlog of web-facing operations that have **no equivalent `/api/v1/hackforger/...` endpoint**. Per the project principle (every web user action must have an API equivalent), these should be filed and closed.

## Real gaps

| # | Web route | Module | Notes / suggested API |
|---|-----------|--------|----------------------|
| 1 | `POST /hackforger/attachments` | Platform-level rich text | Uploads attachments with `RepoID=-1` for use in hackathon descriptions, grant project pages, submissions, etc. — content not bound to any repo/issue/release. **Forgejo's stdlib API only supports repo-/issue-/release-scoped uploads** (`/issues/{index}/assets`, `/releases/{id}/assets`); none can produce a `RepoID=-1` attachment. Suggest: `POST /hackforger/attachments` (multipart `file`) returning `{"uuid":"..."}`. |
| 2 | `POST /-/admin/hackforger/reputation` | Reputation algorithm config | Sets the platform-wide `reputation.weights` and `reputation.tiers` JSON via `hackforger_model.SetSetting`. There is no API to read/write these settings. Suggest: `GET/PUT /hackforger/admin/reputation/settings` (admin only). |
| 3 | `POST /-/admin/hackforger/phase-types` (+ `/{id}/edit`, `/{id}/delete`) | Phase type catalog (admin) | CRUD on the global library of phase types (Registration, Development, Judging, etc.). Suggest: `m.Group("/admin/hackforger/phase-types")` with Get/Post/Put/Delete. |
| 4 | `POST /-/org/{org}/join-request` | Org membership | Hacker requests to join organizer's org (used in team formation). Suggest: `POST /hackforger/orgs/{org}/join-requests`. |

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
