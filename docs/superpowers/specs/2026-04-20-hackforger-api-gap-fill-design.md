# HackForger API gap fill — design

**Date:** 2026-04-20
**Branch:** `feat/hackforger-api-skill` (PR #73)
**Author:** Allen Woods (with Claude)

## Background

The PR that introduced the `/hackforger-api` skill (PR #73) audited every web-facing operation in HackForger and produced a parity report at `docs/skills/hackforger-api/gaps.md` listing 4 web-only operations that lack `/api/v1/hackforger/*` equivalents. Per the project principle in `MEMORY.md` ([`project_api_web_parity.md`](../../../../.claude/projects/-Users-h2oslabs-Workspace-hackforger/memory/project_api_web_parity.md)), every user-facing web action must have an API equivalent. This design closes those 4 gaps in the same PR.

It also addresses two **dead routes** (`POST /hackathon/{slug}/manage/start` and `/manage/judge`) that the audit surfaced — `ManagePhasePost` does not implement these actions because hackathon phase advancement is owned by a gocron scheduler (`hackforger_hackathon_status` @ every 5min). The associated UI buttons and E2E task steps are revised to reflect the time-driven model.

## Goals

1. Add 4 new API endpoints achieving full parity with their web counterparts:
   - Platform-level attachment upload
   - Reputation algorithm settings (read/write)
   - Phase type catalog CRUD (admin)
   - Org join-request creation
2. Remove 2 dead web routes and the buttons that POST to them.
3. Update E2E tasks 3.1 / 7.1 to use cron-driven phase advancement.
4. Synchronise the `hackforger-api` skill references so the documentation reflects the new state in the same PR.

## Non-goals

- New tests beyond happy-path + auth (simplified per agreed test scope).
- Refactoring `ManagePhasePost` (it correctly handles its remaining `publish`/`cancel` cases).
- Implementing org join-request *acceptance* flow — that already exists via Forgejo team membership APIs; this PR only adds the request-side endpoint.
- Migrating any existing data; all model tables already exist.

## Architectural rules followed

- All new HackForger routes go under `/api/v1/hackforger/...` (not under Forgejo's `/orgs`, `/admin`, or `/repos` namespaces) — per `feedback_hackforger_namespace`.
- Handlers carry `// swagger:operation` annotations following Forgejo upstream convention. Swagger tags by module:
  - Attachment handler → `tags: hackforgerAttachment`
  - Reputation settings handlers → `tags: hackforgerAdmin`
  - Phase type CRUD handlers → `tags: hackforgerAdmin`
  - Org join-request handler → `tags: hackforgerOrg`
  (matches the existing `hackforgerHackathon` / `hackforgerBounty` / etc. group naming in current handlers)
- Form structs in `modules/structs/hackforger.go` (or new file if cleaner), with `binding` tags matching the project's existing pattern (`RedeemForm`, `AdminDepositForm`).
- Errors flow as typed errors from model/service → API handler maps to HTTP code + i18n key. Reuse existing keys; no new locale strings unless an existing key is unsuitable.
- Shared business logic stays in `models/hackforger/` and `services/`; API and web handlers both call into the same lower layers (no cross-handler calls).

## File layout

### New files (4 handlers)

```
routers/api/v1/hackforger/
├── attachment.go        — UploadAttachmentAPI
├── admin_settings.go    — GetReputationSettingsAPI / UpdateReputationSettingsAPI
├── admin_phase_types.go — ListPhaseTypesAPI / CreatePhaseTypeAPI / UpdatePhaseTypeAPI / DeletePhaseTypeAPI
└── org.go               — CreateOrgJoinRequestAPI
```

### Modified files

| File | Change |
|------|--------|
| `routers/api/v1/api.go` | Register 4 route groups in the `/hackforger` block (paths in §"Endpoint specifications" below) |
| `modules/structs/hackforger.go` | Add 6 form/response struct definitions |
| `routers/web/web.go` | (commit 5) Remove lines 546–547: `m.Post("/start", ...)` and `m.Post("/judge", ...)` |
| `templates/hackforger/hackathon/manage*.tmpl` | (commit 5) Remove "开始 Hacking" / "开始评审" buttons |

### New tests

```
tests/integration/
├── hackforger_attachment_test.go   — 2 t.Run (happy + 401)
├── hackforger_phase_type_test.go   — 4 t.Run (CRUD + 403)
└── hackforger_org_test.go          — 3 t.Run (happy + already-member + 401)
```

(Singular file names match the existing convention: `hackforger_bounty_test.go`, `hackforger_grant_test.go`, etc.)

`tests/integration/hackforger_reputation_test.go` gets 3 appended t.Run cases for settings.

### Skill reference updates (same PR)

| File | Change |
|------|--------|
| `.claude/commands/hackforger-api.md` | Add 2 rows to the modules table for Orgs and Attachments |
| `docs/skills/hackforger-api/hackathons.md` | New "Admin: phase type catalog" section |
| `docs/skills/hackforger-api/feed-search-reputation.md` | Add 2 rows for reputation settings GET/PUT |
| `docs/skills/hackforger-api/orgs.md` (new) | join-request endpoint + relation to Forgejo team APIs |
| `docs/skills/hackforger-api/attachments.md` (new) | platform-level upload endpoint + RepoID=-1 explanation |
| `docs/skills/hackforger-api/gaps.md` | Remove 4 closed-gap rows; remove dead-routes section in commit 5 |
| `docs/skills/hackforger-api/README.md` | Add 2 entries to Files list |

### E2E doc updates (commit 5)

- `docs/tests/e2e/tasks/full-cycle/01-hackathon-lifecycle.md` — rewrite Step 3.1 to set short registration phase + wait for cron
- `docs/tests/e2e/tasks/full-cycle/04-submission-judging.md` — rewrite Step 7.1 likewise

## Endpoint specifications

### 1. `POST /hackforger/attachments`

**Auth:** `reqToken()` (any authenticated user)

**Request:** `multipart/form-data`, field `file`

**Response 200:**
```json
{ "uuid": "<attachment uuid>" }
```

**Errors:**
- 400 `attachment.file_type_forbidden` — file extension not in `setting.Attachment.AllowedTypes`
- 404 — `setting.Attachment.Enabled == false`
- 500 — upload failed (check log for detail)

**Implementation:** mirror `routers/web/hackforger/upload.go:UploadHackforgerAttachment`. Same `attachment.UploadAttachment(...)` call with `RepoID: -1`. Returns JSON via `ctx.JSON(http.StatusOK, ...)` instead of `ctx.JSON(http.StatusOK, map[string]string{"uuid": ...})` directly so the response shape matches a struct.

### 2. Reputation settings

#### `GET /hackforger/admin/reputation/settings`

**Auth:** `reqToken() + reqSiteAdmin()`

**Response 200:**
```json
{
  "weights": "<raw JSON string from setting>",
  "tiers":   "<raw JSON string from setting>"
}
```

(Strings, not parsed objects — schema can evolve and clients parse as needed; mirrors web handler behaviour.)

#### `PUT /hackforger/admin/reputation/settings`

**Auth:** `reqToken() + reqSiteAdmin()`

**Request body:**
```go
type UpdateReputationSettingsForm struct {
    Weights string `json:"weights"` // optional; if non-empty must be valid JSON
    Tiers   string `json:"tiers"`   // optional; if non-empty must be valid JSON
}
```

Both fields optional; non-empty fields are validated with `json.Valid()` before write. Empty fields are skipped (partial update).

**Response 200:** echoes the GET shape after the write.

**Errors:**
- 422 `hackforger.admin.reputation.invalid_json` — supplied weights or tiers is not valid JSON

### 3. Phase-type catalog CRUD

All under `/hackforger/admin/phase-types`. Auth: `reqToken() + reqSiteAdmin()`.

#### `GET /hackforger/admin/phase-types`

**Response 200:** flat array (clients group by `activity_kind` if needed):
```json
[
  {
    "id": 1,
    "activity_kind": "hackathon",
    "key": "registration",
    "display_name_i18n": "hackforger.phase.registration",
    "is_unique": true,
    "allowed_actions": "[\"register\"]",
    "default_order": 1
  }
]
```

#### `POST /hackforger/admin/phase-types`

**Body:**
```go
type CreatePhaseTypeForm struct {
    ActivityKind    string `json:"activity_kind" binding:"Required;In(hackathon,bounty,grant)"`
    Key             string `json:"key" binding:"Required"`
    DisplayNameI18n string `json:"display_name_i18n" binding:"Required"`
    IsUnique        bool   `json:"is_unique"`
    AllowedActions  string `json:"allowed_actions"` // defaults to "[]" server-side
    DefaultOrder    int    `json:"default_order"`
}
```

**Response 201:** the created phase type (same shape as list element).

#### `PUT /hackforger/admin/phase-types/{id}`

Same form as Create, all fields treated as full replacement (matches web handler semantics — web does full overwrite of all fields).

**Errors:** 404 if id not found.

#### `DELETE /hackforger/admin/phase-types/{id}`

**Response 204** on success. 404 if not found.

### 4. `POST /hackforger/orgs/{org}/join-request`

(Singular path matches the existing web route `POST /-/org/{org}/join-request`. Web handler does not persist a join-request row — it publishes a notification feed event. We mirror this fire-and-forget semantic; there is no entity to GET later, so plural REST styling and a `request_id` would be misleading.)

**Auth:** `reqToken()`

**URL param:** `{org}` → resolved by `orgAssignment(true)` (the unexported helper at `routers/api/v1/api.go:638`; `true` means "must exist, allow non-members"). Existing org routes use the same pattern.

**Request body:** none.

**Response 202 Accepted:**
```json
{ "status": "notified" }
```

(202 rather than 201 because no resource is created.)

**Errors:**
- 422 `hackforger.org.already_member` — doer is already a member or owner
- 404 — org not found (handled by `orgAssignment` middleware)

**Implementation:** extract the duplicate-detection + notification-publish lines from `routers/web/hackforger/org.go:JoinOrgRequest` into a new `services/hackforger/org.go:RequestOrgJoin(ctx, org, doer)` so both web and API call the same function. Web handler updated to call the extracted service.

(No duplicate-pending detection — current web behaviour does not detect duplicates either, and there is no persisted row to deduplicate against. Consistent with the spec's "no acceptance flow" non-goal.)

## Route registration patch

Inside `routers/api/v1/api.go`, the `/hackforger` group already exists (line 1764). Add at appropriate spots:

```go
m.Group("/hackforger", func() {
    // ... existing routes ...

    // Platform-level attachment upload (new)
    m.Post("/attachments", reqToken(), hackforger_api.UploadAttachmentAPI)

    // Org join-request (new) — singular, mirrors web /-/org/{org}/join-request
    m.Group("/orgs/{org}", func() {
        m.Post("/join-request", reqToken(), hackforger_api.CreateOrgJoinRequestAPI)
    }, orgAssignment(true))

    // Admin operations — note: /hackforger/admin/reindex already exists at line 1870
    // as a top-level child of /hackforger (NOT under an /admin sub-group). The new
    // /admin sub-group is created fresh; reindex stays where it is.
    m.Group("/admin", func() {
        m.Group("/reputation/settings", func() {
            m.Get("", hackforger_api.GetReputationSettingsAPI)
            m.Put("", bind(hackforger_api.UpdateReputationSettingsForm{}), hackforger_api.UpdateReputationSettingsAPI)
        }, reqSiteAdmin())
        m.Group("/phase-types", func() {
            m.Get("", hackforger_api.ListPhaseTypesAPI)
            m.Post("", bind(hackforger_api.CreatePhaseTypeForm{}), hackforger_api.CreatePhaseTypeAPI)
            m.Put("/{id}", bind(hackforger_api.UpdatePhaseTypeForm{}), hackforger_api.UpdatePhaseTypeAPI)
            m.Delete("/{id}", hackforger_api.DeletePhaseTypeAPI)
        }, reqSiteAdmin())
    })
})
```

## Testing strategy

Per agreed scope (option B): each endpoint gets one happy-path case plus one auth-failure case. Total ~240 lines.

Existing helpers from `tests/integration/hackforger_test_helper_test.go`:
- `getUserToken(t, username)` — get a personal access token
- `MakeRequest(t, ...)` — issue HTTP request with proper auth headers

Test file template:
```go
func TestAPIHackforgerAttachments(t *testing.T) {
    defer tests.PrepareTestEnv(t)()
    t.Run("Upload happy path", func(t *testing.T) { /* ~25 lines */ })
    t.Run("Reject without token", func(t *testing.T) { /* ~10 lines */ })
}
```

**Not tested** (intentional):
- Multipart parsing edge cases (delegated to upload library)
- File-type-forbidden (delegated to attachment service)
- Reputation JSON schema content (the API treats as opaque string)
- CSRF (API uses token auth, not CSRF)

## Commit plan

Five commits on `feat/hackforger-api-skill`:

| # | Subject | Files | ~Lines |
|---|---------|-------|--------|
| 1 | docs(skills): rename api-test → hackforger-api with full endpoint reference | (already pushed in PR #73 base commit `cc308d8844`) | — |
| 2 | feat(api): add 4 hackforger API endpoints to close parity gaps | 4 new handlers + form structs + route registration | ~400 |
| 3 | test(api): integration tests for new hackforger endpoints | 3 new test files + 3 t.Run appended to reputation_test | ~240 |
| 4 | docs(skills): sync references with newly-added endpoints | hackforger-api/ doc updates | ~120 |
| 5 | chore: remove dead phase-transition routes + revise E2E phase steps | web.go + templates + 2 E2E task files + gaps.md cleanup | ~80 |

## Risks and mitigations

| Risk | Mitigation |
|------|-----------|
| Org join-request service function doesn't exist; logic only in web handler | Extract to `services/hackforger/org.go:RequestOrgJoin` as part of commit 2; web handler updated to call extracted function. |
| Phase-type DefaultOrder collisions | Web handler doesn't validate uniqueness either; rely on user not creating duplicates (matches existing behaviour). |
| Removing dead web routes breaks something hidden (e.g., a Vue component still POSTs to them) | Reviewer's grep already confirmed manage.tmpl only references `/manage/judges` (plural, legitimate add-judge endpoint), not the dead `/manage/start` or `/manage/judge`. Re-grep `templates/ web_src/` immediately before commit 5 to confirm nothing slipped in since; commit 5 may end up touching only `web.go` + the 2 E2E task files + `gaps.md` if no buttons exist. |
| E2E rewrite invalidates prior screenshots | Only the prompt for steps 3.1/7.1 changes; report templates and screenshot paths stay the same. Future runs regenerate screenshots. |
| i18n key `hackforger.org.join_request.already_pending` doesn't exist; spec earlier reused it | Resolved by removing the already-pending detection from this design (see §4). Existing keys reused: `admin.reputation.invalid_json` (line 4563), `admin.phase_types.missing_fields/not_found` (4567+), `org.already_member` (4609). No new locale strings needed. |

## Open questions

None — all decisions captured during brainstorming.
