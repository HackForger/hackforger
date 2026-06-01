---
title: "Issue #182 — hackathon phase custom name not shown on public page"
date: 2026-06-01
issue: 182
pr: 183
commits: [984e7893b5fd1a6b4c663664da050a70f38b0938]
environment:
  - "localhost:3000 (main instance)"
  - "https://hackforger.inside.h2os.cloud (smoke domain, via Caddy → 3000)"
tester: Claude (Opus 4.8) + codex (gpt-5.5) E2E
result: PASS
admin_signoff:
---

# Smoke test: phase custom name on public page (#182)

## Change under test

`templates/hackforger/hackathon/view.tmpl:30` — the public hackathon timeline
now renders an admin-edited phase `custom_name` (falling back to the phase
type's default i18n name), mirroring the management UI renderer
(`PhaseTimeline.vue:59`). Pure display fix; no backend/model/migration/i18n
changes.

## Reproduction (pre-fix)

On `opc-2026-crossborder-w2`, setting a phase `custom_name` left the public page
showing the default name; the custom name appeared 0 times in the page HTML.
The management UI showed the new name — the two renderers disagreed.

## Verification

### Pre-implementation review
codex (read-only) reviewed the one-line design: correct & complete, scope
confirmed to the single template, edge cases (empty string, nil PhaseType, XSS
auto-escaping) handled, precedence (custom wins) correct.

### Stage 1 — localhost E2E (codex, via real management web route)

Full path through the session-authenticated web endpoint
`PUT /hackathon/opc-2026-crossborder-w2/manage/phases/{id}` (not /api/v1),
targeting the `results` phase (id 28, not ended):

| Step | Result | Evidence |
|------|--------|----------|
| Login as SynNovator | PASS | HTTP 303 |
| Rename PUT (custom_name set) | PASS | HTTP 200; JSON response contains `custom_name` |
| Public page renders custom name | PASS | `<div class="hf-step-name">E2E公开页改名验证_RESULTS</div>` |
| Unedited phase shows default | PASS | `<div class="hf-step-name">开发</div>` |
| Cleanup (custom_name='') | PASS | HTTP 200; custom string gone from public page |

`E2E RESULT: PASS`

### Stage 2 — smoke domain (hackforger.inside.h2os.cloud, via Caddy)

Set `custom_name='总决赛颁奖盛典FINALS'` on the `results` phase, fetched the
public page through the real HTTPS domain:

- `GET https://hackforger.inside.h2os.cloud/hackathon/opc-2026-crossborder-w2` → HTTP 200
- Timeline rendered: `Development / Judging / Registration / 总决赛颁奖盛典FINALS`
  (custom name shown; other phases show defaults)
- Browser screenshot confirms the last phase shows「总决赛颁奖盛典FINALS」
  instead of the default「结果发布」.

All test `custom_name` values were reverted to empty after testing.

## Related handler fix (included in this PR)

A latent bug surfaced during E2E: `ManagePhasesUpdate`
(`routers/web/hackforger/phase.go`) wrote `custom_name` *unconditionally before*
the `UpdatePhaseTime` ended-phase guard, so renaming an **ended** phase
persisted the name in the DB yet returned `403 该阶段已结束，无法修改` — a partial
write on a failed request.

Fixed by decoupling the name from the time window:
- The guarded `UpdatePhaseTime` is now invoked **only when start/end actually
  change**, so it can no longer leave a half-applied name behind.
- The custom name (a cosmetic label) is persisted via a focused
  `UpdatePhaseCustomName` (writes only the `custom_name` column) **after** any
  time update succeeds, and is allowed on **any** phase including ended ones.
- Net behavior: renaming works on any phase; changing the **time window** of a
  locked/active phase is still rejected as before.

## Admin sign-off

Pending admin manual verification (stage 3).
