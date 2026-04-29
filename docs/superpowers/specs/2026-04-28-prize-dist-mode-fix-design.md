# Prize Distribution Mode Fix — Design

**Date**: 2026-04-28
**Author**: Allen Woods (with Claude assistance)
**Origin**: Issue #77 E2E report — `finalize` succeeded but participant credits did not increase
**Scope**: Code bug fix only. Skill doc update is a separate workflow (see Section 6). Bug 2 (`current-phase = null`) is parked for tester re-verification (see Section 7).

---

## 1. Background

The E2E run on hackathon `93` / track `121` reported:

- `POST /api/v1/hackforger/hackathons/93/finalize` returned `200 OK`
- Leaderboard rendered correctly with submission `71` ranked `#1`
- **But** participant credit balance stayed at `0` after finalize

Live data inspection confirmed:

```json
{"ID":121, "PrizeCredits":100, "PrizeDistMode":"", "PrizeDistRatios":""}
```

Root cause chain:

1. `routers/api/v1/hackforger/hackathon_track.go:16-31` — `CreateTrackForm` and `UpdateTrackForm` do **not declare** `PrizeDistMode` or `PrizeDistRatios` fields.
2. The web form (`routers/web/hackforger/hackathon.go:763-765, 822-824`) defaults empty mode to `winner_takes_all`. The API has no such fallback.
3. XORM `db.Insert` writes the Go zero value (`""`) explicitly, bypassing the `DEFAULT 'winner_takes_all'` SQL constraint.
4. `services/hackforger/hackathon.go:952-954` — `distributeHackathonCredits` `default` branch logs a warning and **silently skips** the track. No error is returned to the caller. `finalize` reports success.

This is an **API ↔ Web parity gap** (per `project_api_web_parity` memory) plus a **silent failure** in the distribution logic.

---

## 2. Goals

- API can create/update tracks with explicit prize distribution config.
- Empty/invalid `PrizeDistMode` is rejected at create/update time, not silently lost.
- `distributeHackathonCredits` fails loudly on invalid data instead of skipping.
- Existing test data with `PrizeDistMode=''` is backfilled to `winner_takes_all`.
- All error paths surface to the user via Forgejo's Flash mechanism (web) and proper HTTP status codes (API) — no generic "internal error" shadows the real problem.

## 3. Non-goals

- Bug 2 (`current-phase = null` / registration gating) — parked for tester re-verification.
- Adding new distribution modes beyond the existing three (`winner_takes_all`, `tiered`, `equal`).
- Migrating production data (this is a test-environment fix; production has no prize-credits flow yet).
- Updating the `hackforger-api` skill (separate PR; coupled but tracked separately).

---

## 4. Architecture: Layered Validation

Validation logic is centralized in a single helper at the model layer; service and HTTP handlers call it before persisting. This honors the project convention (CRUD in `models/`, business orchestration in `services/`) while keeping a single source of truth.

| Layer | File | Responsibility |
|---|---|---|
| Model — pure validator | `models/hackforger/hackathon_track.go` | New `ValidatePrizeDistConfig(mode, ratiosJSON) error`. Returns typed errors. |
| Model — error types | `models/hackforger/hackathon_track.go` | New `ErrInvalidPrizeDistMode{Mode}` (wraps `util.ErrInvalidArgument`). Reuse existing `ErrInvalidDistRatios` for ratios. |
| API form | `routers/api/v1/hackforger/hackathon_track.go` | Add `PrizeDistMode`, `PrizeDistRatios` fields. Empty mode → `winner_takes_all` (matches web). Call `ValidatePrizeDistConfig` before `CreateTrackWithRepo` / `UpdateTrack`. |
| Web handler | `routers/web/hackforger/hackathon.go` | Existing `prize_dist_mode` form parsing already in place. Add `ValidatePrizeDistConfig` call to the create/update handlers. Add new error branches in flash error handling. |
| Service distribute | `services/hackforger/hackathon.go` | `distributeHackathonCredits` `default` branch: replace `log.Warn` + skip with `return ErrInvalidPrizeDistMode{...}`. |

**Invariant after this change**: No DB row in `hackathon_track` can have `prize_dist_mode = ''` or any unknown value, given that all writes go through `ValidatePrizeDistConfig`. The `default` branch in `distributeHackathonCredits` exists only as a defense against unknown future modes and bypassed-API direct INSERTs.

---

## 5. Error Propagation: Full Touch-Point Coverage

The current codebase has multiple `log.Error` + `Flash.Error("internal")` fallbacks that would swallow the new typed errors. **All of them must add explicit branches**, otherwise users see "internal error" with no actionable hint.

| Touch point | File:line | Required handling |
|---|---|---|
| API `CreateTrack` | `routers/api/v1/hackforger/hackathon_track.go:104` | `IsErrInvalidPrizeDistMode(err)` / `IsErrInvalidDistRatios(err)` → `ctx.Error(400, ...)` |
| API `UpdateTrack` | same file `:169` | Same as above |
| API `FinalizeConfirm` | `routers/api/v1/hackforger/hackathon_judge.go:567` | `IsErrInvalidPrizeDistMode(err)` → `ctx.Error(422, "InvalidPrizeDistMode", err)` (semantics: stored data is invalid) |
| API `FinalizeHackathon` (dead) | `routers/api/v1/hackforger/hackathon.go:356-366` | **Delete**. Not wired in `api.go` (only `FinalizeConfirmAPI` is registered at `api.go:1799`). The dead handler flattens every error to `400 "FinalizeHackathon"` with `err.Error()` exposed, which violates the project's "never show `err.Error()` to users" rule (CLAUDE.md i18n-error-pattern). Removing it eliminates the regression risk if someone re-wires it later. |
| Web `ManageTrackPost` | `routers/web/hackforger/hackathon.go:799` | Add typed branches before falling through to generic internal flash |
| Web `ManageTrackUpdatePost` | same file `:846` | Same as above |
| Web `FinalizeConfirm` | same file `:1116` | Add `IsErrInvalidPrizeDistMode` branch before generic internal flash |
| Service `distributeHackathonCredits` | `services/hackforger/hackathon.go:952-954` | Return typed error instead of `log.Warn` |
| Service `ConfirmFinalize` | same file `:526` | Already propagates `distributeHackathonCredits` errors. No change. |

### Forgejo notification mechanism — clarification

`notify_service` / `PublishHackforgerAction` is the **feed event stream** (e.g. "X registered for hackathon Y"). It is **not** the user-facing error surface. Errors surface through:

- Web: `ctx.Flash.Error(ctx.Tr(...))` — top-of-page red banner, consumed on the next request.
- API: `ctx.Error(statusCode, label, err)` — JSON response with status + label.

This design touches Flash + API status codes only. The feed pipeline is unaffected.

### i18n keys (en + zh, both required by project policy)

```ini
# options/locale/locale_en-US.ini, [hackforger]
hackforger.hackathon.error.invalid_prize_dist_mode = Invalid prize distribution mode. Must be winner_takes_all, tiered, or equal.

# options/locale/locale_zh-CN.ini, [hackforger]
hackforger.hackathon.error.invalid_prize_dist_mode = 无效的奖金分配模式。必须是 winner_takes_all、tiered 或 equal 之一。
```

`hackforger.hackathon.invalid_ratios` already exists — reuse for tiered ratios errors.

---

## 6. Data Flow

```
Agent / User
  │
  ├─ POST /api/v1/hackforger/hackathons/{id}/tracks
  │       body: {name, prize_credits, prize_dist_mode?, prize_dist_ratios?}
  │       │
  │       ├─ form parse; empty mode → "winner_takes_all"
  │       ├─ ValidatePrizeDistConfig(mode, ratios)
  │       │       ├─ mode ∈ {winner_takes_all, tiered, equal}? else ErrInvalidPrizeDistMode → 400
  │       │       └─ if tiered: ParsePrizeDistRatios + ValidatePrizeDistRatios
  │       │           bad → ErrInvalidDistRatios → 400
  │       └─ CreateTrackWithRepo (persist)
  │
  └─ POST /api/v1/hackforger/hackathons/{id}/finalize/confirm
          │
          └─ ConfirmFinalize → distributeHackathonCredits
                  │
                  └─ for each track:
                      switch mode:
                        winner_takes_all | tiered | equal → Deposit(...)
                        default → return ErrInvalidPrizeDistMode → 422
```

---

## 7. Data Migration (A — minimal cleanup)

`models/forgejo_migrations/v14k_clean-empty-prize-dist-mode.go`:

```go
package forgejo_migrations

import "xorm.io/xorm"

func init() {
    registerMigration(&Migration{
        Description: "Backfill empty prize_dist_mode to 'winner_takes_all'",
        Upgrade:     cleanEmptyPrizeDistMode,
    })
}

func cleanEmptyPrizeDistMode(x *xorm.Engine) error {
    _, err := x.Exec(
        "UPDATE hackathon_track SET prize_dist_mode = 'winner_takes_all' " +
        "WHERE prize_dist_mode = '' OR prize_dist_mode IS NULL",
    )
    return err
}
```

Idempotent. Safe even if no empty rows exist. Test environment is the primary target; production deployment is also safe (no harm if all rows already valid).

**API conformance** (verified against `models/forgejo_migrations/migrate.go`):
- Function: `registerMigration(*Migration)` — not `register`
- Field: `Upgrade func(*xorm.Engine) error` — not `Migrate`
- Filename: must match `migrationFilenameRegex = /(?P<group>v[0-9]+[a-z])_(?P<id>[^/]+)\.go$` — `v14k_clean-empty-prize-dist-mode.go` matches.

---

## 8. Test Matrix

| Layer | Tests | File |
|---|---|---|
| Model unit | `ValidatePrizeDistConfig`: each of three valid modes; unknown mode → `ErrInvalidPrizeDistMode`; tiered + missing ratios → `ErrInvalidDistRatios`; tiered + malformed JSON → error; tiered + ratios sum ≠ 100 → error | `models/hackforger/hackathon_track_test.go` (extend) |
| Service unit | `distributeHackathonCredits`: happy path winner_takes_all distributes correctly; **construct an in-memory `HackathonTrack` struct directly with `PrizeDistMode=""` and persist it via raw `db.GetEngine(ctx).Insert` to bypass `ValidatePrizeDistConfig`**; assert `distributeHackathonCredits` returns `ErrInvalidPrizeDistMode`. This is necessary because, post-fix, the validator prevents this state from being created via the public API surface — but the strict default branch in `distributeHackathonCredits` is the safety net for direct INSERTs / future modes, and it must be tested. | `services/hackforger/hackathon_credits_test.go` (extend) |
| Integration — single layer | API CreateTrack: omitting mode → DB stores `winner_takes_all`; explicit invalid mode → 400; tiered + valid ratios → 201; tiered + bad ratios → 400 | `tests/integration/api_hackforger_track_test.go` (extend or new) |
| Integration — full regression trip | Reproduce the original E2E failure path: (1) create hackathon, (2) create track via API **without** `prize_dist_mode` field, (3) register a participant, (4) submit, (5) score, (6) finalize, (7) assert participant's credit balance increased by `PrizeCredits`. This is the single test that would have caught the original bug. | `tests/integration/api_hackforger_finalize_credits_test.go` (new) |

**No migration unit test** — there is no precedent for per-migration tests in `models/forgejo_migrations/v14*_test.go`. The migration is a single idempotent SQL UPDATE; correctness is verified by the integration trip above (which runs against a fresh test DB where the migration has been applied) plus manual smoke on the dev instance.

E2E on the live instance is **not** part of this PR. The tester reverifies after deploy (see Section 10).

---

## 9. Skill Doc Update (separate workflow)

`skills/hackforger-api/references/hackathons.md` needs:

- Document `prize_dist_mode` and `prize_dist_ratios` in the track create/update example body.
- Add a "common pitfall" entry: omitting mode now defaults to `winner_takes_all` (was: silently broken finalize).
- Document that `tiered` requires a non-empty, sum-to-100 `prize_dist_ratios` JSON array.

This is **not** in the code-bug PR. It is tracked as a follow-up because:

1. The skill repo path is portable (`skills/hackforger-api/`) and may be vendored to multiple clients (Codex, Cursor, etc.) with different release cadence.
2. The user asked specifically to distinguish skill issues from code bugs.

The follow-up skill PR happens after this code PR merges and the API contract is stable.

---

## 10. Bug 2 — Tester Re-verification Request

`current-phase = null` + registration gating issue is **not** fixed in this PR. Reasoning:

- `GetCurrentPhase` SQL (`start_time <= now AND end_time > now`) is correct.
- Live phase 103 has a 16-minute window already in the past — natural `null`.
- Most likely the tester's "normal phase setup" (a) used a too-narrow window that elapsed during the test, or (b) only created a `judging` phase without `registration` (which by design rejects `register` action).

Issue #77 follow-up comment will request the tester to:

1. Re-run the phase-setup → registration flow with explicit unix-second timestamps spanning hours, not minutes.
2. Create all three required phase types: `registration`, `development`, `judging` (in order, contiguous).
3. Capture full curl response for each step.
4. Report: does `current-phase` return the expected phase? Does registration succeed?
5. If still failing → attach screenshots + curl traces; we open a fresh code investigation issue.

If verification reveals a real backend bug, we cut a separate spec for it.

---

## 11. Files Touched

**Production code:**

- `models/hackforger/hackathon_track.go` — `ValidatePrizeDistConfig`, `ErrInvalidPrizeDistMode`
- `routers/api/v1/hackforger/hackathon_track.go` — form fields, validation call, error branches
- `routers/api/v1/hackforger/hackathon.go` — **delete** dead `FinalizeHackathon` handler (lines 339-366 incl. swagger block)
- `routers/web/hackforger/hackathon.go` — validation call in track create/update; error branches in finalize
- `services/hackforger/hackathon.go` — `distributeHackathonCredits` strict `default`
- `models/forgejo_migrations/v14k_clean-empty-prize-dist-mode.go` — migration
- `options/locale/locale_en-US.ini`, `options/locale/locale_zh-CN.ini` — i18n keys

**Tests:**

- `models/hackforger/hackathon_track_test.go` — extend
- `services/hackforger/hackathon_credits_test.go` — extend
- `tests/integration/api_hackforger_track_test.go` — extend or new
- `tests/integration/api_hackforger_finalize_credits_test.go` — new (full regression trip)

---

## 12. Acceptance Criteria

1. API track create without `prize_dist_mode` succeeds; DB row has `prize_dist_mode = 'winner_takes_all'`.
2. API track create with explicit invalid mode returns `400` with the new i18n message.
3. API track create with `tiered` mode and bad ratios returns `400`.
4. After running `v14k` migration, no row in `hackathon_track` has `prize_dist_mode IN ('', NULL)`.
5. `distributeHackathonCredits` on a track with empty mode returns `ErrInvalidPrizeDistMode`; finalize fails with `422` rather than `200`.
6. All four touch points (API/web × create/update) and finalize handler render the typed error rather than a generic "internal error" flash.
7. All new i18n strings present in both `locale_en-US.ini` and `locale_zh-CN.ini` under `[hackforger]`.
8. Dead `FinalizeHackathon` handler at `routers/api/v1/hackforger/hackathon.go:339-366` is removed (verified: no `api.go` route registers it; only `FinalizeConfirmAPI` is on `/finalize` per `api.go:1799`).
9. Tester re-runs hackathon `93` finalize after migration (or on a fresh hackathon with explicit modes) and confirms participant credits balance increases.

---

## 13. Design Notes (review responses)

- **Error type field naming**: `ErrInvalidPrizeDistMode{Mode string}` follows the typed-identifier convention (matches `ErrTrackNotExist{ID int64}`, `ErrPhaseNotFound{PhaseID int64}` in the same package). The other convention in the file is `ErrInvalidDistRatios{Reason string}` — used because ratios validation has many distinct failure modes that need to be communicated. `ErrInvalidPrizeDistMode` has only one failure mode (unknown value), so the value itself is the most useful field.
- **i18n key namespace**: New key `hackforger.hackathon.error.invalid_prize_dist_mode` uses the `.error.` segment. This matches the dominant convention in the file (`error.internal`, `error.invalid_phase`, `error.no_tracks`, `error.no_registration_phase`, etc.). The bare-form `hackforger.hackathon.invalid_ratios` is the older outlier from v14g; we don't normalize it in this PR (out of scope).
- **Web flash + redirect after error**: `ManageTrackPost` / `ManageTrackUpdatePost` already follow Forgejo's standard pattern of `Flash.Error + Redirect` to the manage page. The user sees the typed flash on next render. No change to the redirect path; only the flash content becomes specific.
