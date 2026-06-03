---
pr: 189
commits:
  - 7fa146b58e
  - d21ae2f739
tested_against: https://hackforger.inside.h2os.cloud (smoke env — Mac instance on dev tip, accessed via localhost:3000)
tested_at: 2026-06-03T17:25:00+08:00
e2e_owner: claude
admin_signoff:
  by: allenwoods
  at: 2026-06-03T17:30:00+08:00
  notes: "In-chat fast-deploy authorization (admin, allen.woods@outlook.com): provided PR #189 link with '新的landing page修改' — same established merge+smoke+prod fast path as PR #181 / #186 for landing-only changes. Pure text change (3 buttons relabelled). Admin authorized merge + direct-to-prod deploy. Scope: this single deploy."
---

# PR #189 — relabel passed-stage buttons to 查看详情 (#188) — E2E smoke test

## What shipped

PR #189 (`feat(landing): relabel passed stage buttons to 查看详情`, landing
commit `7fa146b58e`, merge `d21ae2f739`). Single user-facing file:
`custom/public/assets/landing/index.html` (static asset, served from disk,
no `make` required). 3-line text-only change for issue #188.

Registration windows for S1W2 / S1W3 / S2W1 have closed (date badges
`5月10日-5月15日` / `5月17日-5月24日` / `5月27日-5月29日`; today is 6月3日), so
their buttons move Open → "Passed": label changes from 报名 to `查看详情`,
while staying green and clickable (`bg-primary`, `enabled:true`) — consistent
with S1W1 which already sits in that state. Clicking still navigates to
`/hackathon/<slug>`. No config or i18n-dictionary change needed
(`查看详情→View Details` already exists).

| Card | Badge | Before → After |
| --- | --- | --- |
| S1 复赛W2 数智OPC | 5月10日-5月15日 | `立即报名` → `查看详情` |
| S1 半决赛W3 数智OPC | 5月17日-5月24日 | `即刻报名` → `查看详情` |
| S2 初赛W1 跨境OPC | 5月27日-5月29日 | `即刻报名` → `查看详情` |

## Verification

Mac instance rebuilt + restarted on dev tip `d21ae2f739` via
`scripts/restart-gitea.sh` (HTTP 200, version `172-d21ae2f739`), tested with
agent-browser against `http://localhost:3000`.

### Changed buttons (zh)

| Stage | text | disabled | bg |
| --- | --- | --- | --- |
| s1-w2 | `查看详情` | false | `rgb(187,253,59)` ✓ |
| s1-w3 | `查看详情` | false | `rgb(187,253,59)` ✓ |
| s2-w1 | `查看详情` | false | `rgb(187,253,59)` ✓ |

### EN mode

| Stage | text |
| --- | --- |
| s1-w2 / s1-w3 / s2-w1 | `View Details` ✓ |

### Untouched cards (regression)

| Stage | text | state |
| --- | --- | --- |
| s2-w2 | `即刻报名` | enabled (registration ongoing) ✓ |
| s2-w3 | `即刻报名` | enabled (registration ongoing) ✓ |
| s1-w4 | `6月27日-6月28日决赛` | disabled finals ✓ |

Screenshots (`screenshots/2026-06-03-pr-189/`):
- `01-zh-fullpage.png` — zh landing (3 cards now 查看详情)
- `02-en-fullpage.png` — EN landing (View Details)

## Notes — this deploy also carries the #186 follow-up cleanup

The dev→prod delta includes the previously-merged but undeployed cleanup
commit `913c74c490` (dead i18n keys + stray root PNGs, already covered by
`2026-06-02-pr-186-followup-cleanup.md` with admin sign-off). Confirmed still
in effect on this tip: dead key `6月17日-6月18日` count = 0. Deploying #189
naturally ships that cleanup to production too — expected and intended.

## Risk surface

Low — landing-page-only, 3-line text change:
- No Go / template / binary / DB impact. `custom/public/` is served as static
  files and rsynced to the cloud by `deploy/ecs/redeploy.sh`.
- No config / i18n / nav structure change; only 3 button labels.
- Rollback is trivial: re-deploy the previous landing HTML.

## Cross-references

- PR: https://github.com/HackForger/hackforger/pull/189 (Closes #188)
- Merge commit: `d21ae2f739`
- Pattern precedent: `docs/tests/e2e/reports/2026-06-02-pr-186-landing-dates-s2w3.md`
