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

## How to consume this list

When implementing a missing API:
1. Add the route to `routers/api/v1/api.go` under the `/hackforger` group.
2. Implement the handler in `routers/api/v1/hackforger/<module>.go` reusing the existing service-layer function (the web handler likely wraps the same service call).
3. Add the new endpoint to the relevant module reference in this directory.
4. Remove the row from this gaps file.

> All listed gaps and dead-route cleanup items are closed as of 2026-04-20 (PR #73).
