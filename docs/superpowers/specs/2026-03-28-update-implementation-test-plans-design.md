# Design: Update Implementation & Test Plan Documents

**Date:** 2026-03-28
**Status:** Approved
**Scope:** Structural rewrite of `docs/implementation-plan-draft.md` and `docs/test-plan-draft.md`

---

## Background

Both plan documents were drafted before a deep analysis of the Forgejo codebase. A thorough audit (3 sub-agents analyzing codebase structure, frontend tooling, and dev infrastructure) revealed 27 issues across 4 categories:

- **8** Forgejo structure analysis outdated/inaccurate
- **6** Frontend dependency analysis incomplete
- **9** Missing critical implementation steps
- **4** Test plan specific issues

## Design Decisions

1. **Approach:** Structural rewrite (Option B) — preserve core design decisions and business logic, restructure affected sections to align with actual Forgejo architecture.
2. **Feed/Notification architecture:** Implement a `HackForgerNotifier` via `services/notify/Notifier` interface instead of directly writing to the action table.
3. **Document format:** Chinese, saved to original paths.
4. **Document positioning:** These are high-level master plans, not detailed implementation guides. Remove code examples, keep architecture descriptions and decision points.
5. **Database testing:** SQLite only for current phase.

## Changes to implementation-plan-draft.md

### Section 1.1 — Directory Structure

- Remove mention of modifying `models/db/engine.go`
- Clarify that `models/hackforger/init.go` uses `init()` + `db.RegisterModel()` pattern to auto-register all 15 tables

### Section 1.2 — Changes to Original Files

Rewrite the modification list from 10 items to 11 accurate items:

| # | File | Change |
|---|------|--------|
| 1 | `routers/web/web.go` `registerRoutes()` | Register HackForger web route Group |
| 2 | `routers/api/v1/api.go` | Register HackForger API route Group |
| 3 | `services/cron/tasks_extended.go` | Register 4 cron tasks |
| 4 | `services/notify/notifier.go` | Extend Notifier interface or register new Notifier |
| 5 | `templates/repo/issue/view_content.tmpl` | Bounty panel injection point |
| 6 | `templates/repo/issue/list.tmpl` | Bounty Badge |
| 7 | `templates/explore/navbar.tmpl` | 3 new Tabs |
| 8 | `templates/base/head_navbar.tmpl` | AI search bar |
| 9 | `web_src/js/index.js` `onDomReady()` | Import + init hackforger modules |
| 10 | `routers/web/user/home.go` | Dashboard feed query extension |
| 11 | `templates/user/dashboard/feeds.tmpl` | Following Tab + HackForger event rendering |

### New Section 1.3 — Database Migration Strategy

- 15 new tables require formal migration files in `models/forgejo_migrations/`
- Must register in the migration chain, not rely on XORM auto-sync alone

### New Section 1.4 — i18n & API Documentation

- All template text requires i18n keys in `options/locale/`
- All API endpoints require Swagger annotations to pass `make swagger-check`

### New Section — Frontend Development Conventions (after directory structure)

Key conventions to note (bullet points, no code):

- Tailwind CSS: all classes use `tw-` prefix (`prefix: "tw-"`, `important: true`)
- Vue 3 Options API: existing Vue SFCs use Options API, new components follow suit
- Component mounting: feature modules lazy-load via `await import()`, mount with `createApp().mount(el)` to DOM placeholder elements in templates
- UI framework: Fomantic UI (Semantic UI fork) for dropdown/modal/form — reuse first
- Theme support: new components must support light/dark via CSS variables

### Section 2.6 — Feed System

Rewrite from "zero new tables, directly write action table" to:

- Register `HackForgerNotifier` implementing `services/notify/Notifier` interface
- For existing events (e.g., `MergePullRequest`): check for Bounty association in the Notifier method
- For HackForger-unique events: extend Notifier interface or trigger internally via Service layer
- Audience strategy table preserved (Global / Org Members / Followers / Repo Watchers)

### Section 3 — Feed Core Implementation

Remove all Go code blocks. Replace with architecture description:

- **Publish layer:** HackForgerNotifier writes to action table on state changes, audience strategy determines UserID
- **Query layer:** `services/hackforger/feed.go` provides Following / Global / Entity query modes
- **API layer:** `GET /api/v1/hackforger/feed` with type/entity_id/page/limit params
- **Render layer:** `templates/hackforger/feed/items.tmpl` + Dashboard feeds.tmpl injection

Keep: audience strategy table, Feed API response format example.
Remove: `PublishHackforgerAction` code, `GetFollowingFeed` SQL code, per-service call-site code.

### Section 4 — Service Layer Key Logic

- Keep state machine flow diagrams (ASCII)
- Replace "Bounty-PR linkage" code with one-line description: "triggered via HackForgerNotifier's `MergePullRequest` method"
- Keep "Deposit/Redeem use `db.WithTx` for transaction safety" statement, remove function bodies
- Keep route table (URL to handler mapping), remove Go registration code
- Add note on ActionType: "must also add string mapping in `ActionType.String()` method"

### Section 6 — Development Plan

Supplement Week 1 P0:
- Create migration files in `models/forgejo_migrations/`
- Create i18n key files in `options/locale/`
- Register HackForgerNotifier skeleton in `services/notify/`

Supplement Week 2-3 per-line:
- API endpoints need Swagger annotations
- Vue components follow Options API + lazy-load mounting pattern
- Template text uses i18n keys, not hardcoded strings

Supplement Week 5:
- Ensure `make swagger-check` passes
- Ensure `make test-sqlite` covers hackforger modules
- Complete test fixture YAML files

### Section 9 — Native vs Fork

Add "HackForger Notifier registration" to the "Must Fork" list.

## Changes to test-plan-draft.md

### Section 2 — Unit Tests: Fix File Paths

- Model tests: `models/hackforger/xxx_test.go` (not `tests/hackforger/`)
- Service tests: `services/hackforger/xxx_test.go`
- Test case lists unchanged, only paths corrected

### Section 2.6 — Feed Module Tests: Adapt to Notifier

- Rename `TestPublishAction_*` to `TestNotifier_*`
- Test that Notifier correctly writes to action table on events
- Audience distribution test logic unchanged

### Section 3 — E2E Tests: Clarify Framework

Add clarification:
- Go integration tests use `tests/integration/` framework (`PrepareTestEnv()`, `loginUser()`, etc.)
- Frontend E2E uses Playwright (`tests/e2e/`)
- Test fixtures needed in `models/fixtures/` as YAML

### New Section — Frontend Component Tests

- 6 Vue components need Vitest unit tests
- Files in `web_src/js/components/` co-located or in `__tests__/`

### New Section — Database Compatibility

- Current phase: SQLite only
- MySQL/PostgreSQL compatibility deferred

## Out of Scope

- Writing actual implementation code
- Creating migration files
- Detailed per-function specs (those belong in implementation plans written at dev time)
