# E2E Testing Lessons Learned

> Accumulated from Phase 1 Bounty testing (5 rounds, 9+ bugs). These patterns apply to all HackForger modules (Hackathon, Grant, Credits).

## Migration & Schema

- **Migration struct must match Go model exactly** — P0 migration column names (e.g., `creator_id`) diverged from model fields (`publisher_id`). Unit tests pass because XORM Sync auto-corrects schema from Go structs, but the production database has wrong columns from the migration. Always verify migration struct field names match the model.
- **Issue Index ≠ Issue DB ID** — Users and API pass issue Index (#N from URL), but DB foreign keys use primary key ID. Always resolve via `issues_model.GetIssueByIndex(ctx, repoID, index)` before storing as a foreign key.

## Forgejo Notifier Hooks

- **`pr.Issue` is the PR itself, not the referenced issue** — `pr.LoadIssue()` loads the PR's own Issue row in the issue table. It does NOT load the issue referenced via "fixes #N". To find related issues on PR merge, scan repo bounties by claimer/author match instead of relying on `pr.Issue.ID`.
- **Notifier method signatures must match exactly** — The `hackforgerNotifier` embeds `NullNotifier`. If the method signature doesn't match the interface, it silently falls back to the no-op. Always verify with debug logs that the hook is actually called.

## Go Templates

- **`{{if .Field}}` on struct crashes if field doesn't exist** — `{{if .Bounty}}` on `*issues.Issue` causes a 500 error because Go templates fail hard on non-existent struct fields. Use a map lookup pattern instead: `{{if and $.BountyMap (index $.BountyMap .ID)}}` with a map populated in the handler.

## Frontend / Vue

- **Vue components must call web routes, not API routes** — API routes (`/api/v1/...`) require token auth. Browser sessions only have cookies. Vue components embedded in pages must call web POST routes (`/{owner}/{repo}/bounties/{id}/{action}`) which accept session cookies automatically. See `docs/frontend-dev-guide.md`.
- **Frontend rebuild cycle after Vue/JS changes** — Stale bindata cache causes changes to not take effect. Full cycle: `rm -rf public/assets/js public/assets/css public/assets/fonts && make frontend && rm .bindata_* && make build`. Missing any step means the old JS is still embedded.

## Feed System

- **Audience visibility in feed queries** — `AudienceFollowers | AudienceRepoWatchers` events do NOT appear in `type=global` feed. Only events with `AudienceGlobal` bit set create `UserID=0` records visible globally. This is by design but confusing for testers — document it clearly in E2E prompts.
- **New action types need 4 changes** — Adding a new HackForger event type requires: (1) `String()` case in `models/activities/action.go`, (2) icon in `modules/templates/util_misc.go` `ActionIcon()`, (3) rendering in `templates/user/dashboard/feeds.tmpl`, (4) i18n text in both `locale_en-US.ini` and `locale_zh-CN.ini` `[action]` section. Missing any of these causes blank entries in the activity feed.
- **Replace P0 skeleton endpoints** — P0 stubs return hardcoded empty data (e.g., Feed API returning `{"items":[]}`). Easy to forget replacing them when implementing the real logic.

## Testing Infrastructure

- **Worktree needs `custom/conf/app.ini`** — Git worktrees don't include the gitignored `custom/` directory. Server shows install wizard without it. Copy from main repo before starting. See `docs/tests/local-testing-guide.md`.
- **Shared SQLite database** — All worktrees point to the same DB at `/Users/h2oslabs/Workspace/hackforger/data/forgejo.db`. Clean test data between runs with targeted `DELETE FROM` statements, not by dropping the DB.
- **E2E should cover both UI and API** — API-only testing misses auth/CSRF issues that only manifest in browser context. Every state-changing flow should be verified via UI button clicks, not just curl.

## Agent-browser quirks

- **`click <button[type=submit]>` does NOT trigger form submission.** This is a Chrome DevTools Protocol / Playwright behavior: programmatic `Element.click()` fires the click event but does not perform the implicit submit that a real mouse click would trigger (per [MDN](https://developer.mozilla.org/en-US/docs/Web/API/Element/click)). Symptom: click reports `✓ Done`, but no POST hits the server, no DB row appears, page does not navigate. **Workaround**: use `agent-browser eval "var b = document.querySelector('button[type=submit]'); b.form.requestSubmit(b)"` instead. `requestSubmit()` walks the full submit event chain (jQuery AYS handlers, validation, etc.) and triggers the actual POST.
  - This was originally misdiagnosed as CSRF (`CrossOriginProtection` rejection). It is NOT a CSRF/origin issue — the request never leaves the browser.
- **Must accept dialog after click on submit (if AYS confirm fires).** `alert` and `beforeunload` are auto-accepted; `confirm` and `prompt` are not. Use `agent-browser dialog accept` after a click that may trigger AYS unsaved-changes confirmation.

## Test instance setup (PostgreSQL)

- **Cannot share `WORK_PATH` with prod instance** — queue/jwt/data subdirs cause LevelDB lock conflict (`[F] Unable to create notification-service queue`). Test instance must have its own `WORK_PATH` (e.g., `/tmp/hackforger-test-data`).
- **Stale `forgejo_migration` rows block startup** — if a no-longer-existing migration ID is recorded in `forgejo_migration`, the new binary refuses to start with "newer database than this Forgejo release". Forgejo prints the exact `DELETE FROM forgejo_migration WHERE id IN (...)` SQL needed to recover. Common when developing migrations on the test DB and then rolling back the source code.
- **Fresh PG DB has empty seed tables** — Forgejo's `Migrate(x, freshDB=true)` mark-completes all migrations without running their `Upgrade` functions, so any seed data inserted via migration `Upgrade` is missing on fresh DBs. See `docs/notes/pg-migration-pitfalls.md` and run the manual SQL in the dev backup's `RESTORE.md`.
