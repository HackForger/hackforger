# API Rename Plan

> **Status:** Pending — apply during fixture/E2E phase via TDD
> **Principle:** Align HackForger API naming with Forgejo upstream convention (single verb/noun, no kebab-case actions)
> **Method:** Write E2E tests using new names first, then rename endpoints to make tests pass

## Naming Convention

Forgejo upstream pattern: sub-resource paths use **single words** — `/accept`, `/reject`, `/start`, `/stop`, `/finalize`.
HackForger should follow this, not introduce kebab-case action paths.

## Renames

| Current Path | New Path | Scope | Reason |
|-------------|----------|-------|--------|
| `POST /hackathons/{id}/start-judging` | `POST /hackathons/{id}/judge` | API + Web | Parallel to `/publish`, `/start`, `/finalize` — single verb per phase transition |
| `POST /bounties/{id}/reject-delivery` | `POST /bounties/{id}/reject` | API + Web | Within bounty context, "reject" is unambiguous |
| `POST /hackathons/{id}/finalize-confirm` | `POST /hackathons/{id}/finalize` | API + Web | "confirm" is a UI concern; API should be `/finalize` (currently two endpoints?) |
| `POST /credits/orders/batch-fulfill` | `POST /credits/orders/fulfill` with `{"batch": true}` or array body | API + Web | Single endpoint, body controls batch vs single |
| `POST /grants/rounds/{id}/distribute` | Keep as-is | — | New endpoint, single verb, consistent with convention |

## Web Route Renames (corresponding)

| Current Web Path | New Web Path |
|-----------------|-------------|
| `POST /hackathon/{slug}/manage/start-judging` | `POST /hackathon/{slug}/manage/judge` |
| `POST /{owner}/{repo}/bounties/{id}/reject-delivery` | `POST /{owner}/{repo}/bounties/{id}/reject` |
| `POST /hackathon/{slug}/manage/finalize-confirm` | `POST /hackathon/{slug}/manage/finalize` |
| `POST /admin/credits/orders/batch-fulfill` | `POST /admin/credits/orders/fulfill` (array body) |

## TDD Workflow

1. Write fixture YAML / E2E test using **new** endpoint names
2. Tests fail (404)
3. Rename route registration in `routers/api/v1/api.go` and `routers/web/web.go`
4. Rename handler functions if needed (e.g., `StartJudgingHackathon` → `JudgeHackathon`)
5. Tests pass
6. Verify no other callers reference old paths (grep for old path strings)

## Impact on User Journey Doc

`docs/user-journeys.md` should use the **new** names. Current references to update when applying:

| Journey Step | Current | New |
|-------------|---------|-----|
| Step 1.16 | `/start-judging` | `/judge` |

Other renamed endpoints don't appear in the journey doc (they're internal to bounty exception flows or admin operations).

## Notes

- Keep old paths as aliases (redirects) for one release cycle if external consumers exist
- Swagger annotations must be updated alongside route renames
- i18n keys referencing endpoint names (if any) need updating
