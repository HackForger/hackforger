---
pr: 186
commits:
  - 913c74c490
tested_against: https://hackforger.inside.h2os.cloud (smoke env — Mac instance, accessed via localhost:3000)
tested_at: 2026-06-02T17:40:00+08:00
e2e_owner: claude
admin_signoff:
  by: allenwoods
  at: 2026-06-02T17:40:00+08:00
  notes: "In-chat authorization (admin, allen.woods@outlook.com): '收尾建议一起做完' + '收尾的内容好像没有必要推送生产'. Verified-no-op repo-hygiene cleanup after PR #186. Admin's explicit call: NOT redeployed to production this round (no visible/behavioral change). This signed report exists only so a FUTURE landing deploy that carries this commit passes preflight without surprise — it does NOT authorize a deploy by itself."
---

# PR #186 follow-up — dead i18n keys + stray root PNGs cleanup — no-op verification

## What this is

Pure repo-hygiene follow-up to PR #186 (commit `913c74c490`). **Not deployed
to production** — admin explicitly decided it doesn't need a prod push because
it changes nothing a user sees. This report is the audit-trail entry so the
next real landing deploy (which will carry `913c74c490`) clears
`deploy/ecs/preflight.sh` without manual intervention.

## Changes

1. Removed 3 dead EN-dictionary keys for the old `6月17日-6月18日` finals dates
   (`6月17日-6月18日`, `决赛圈协创松（线下）：6月17日-6月18日`, `6月17日-6月18日决赛`).
   PR #186 moved S1W4/S2W4 to `6月27日-6月28日`, so these keys no longer match
   any visible text and never rendered.
2. Removed 4 verification screenshots (`01-..04-*.png`) that PR #186's
   `efabe689d8` committed to the **repo root** instead of `docs/`. They were
   never under `custom/public/`, so `redeploy.sh` never shipped them.

## No-op verification (smoke env, served-from-disk, no rebuild needed)

`custom/public/` is read from disk per request, so the edit was live on
`http://localhost:3000` immediately. Confirmed the cleanup is behavior-neutral:

| Check | Before | After |
| --- | --- | --- |
| page status | 200 | 200 |
| dead key `6月17日-6月18日` occurrences | 3 (all i18n) | **0** |
| visible `6月27日-6月28日` occurrences | 11 | **11** (unchanged) |
| EN `Finals Jun 27–28` | 1 | 1 |
| S2W3 green「即刻报名」button | present | present |
| language switcher in header | 1 | 1 |

No visible text relied on the removed keys → zero rendered difference.

## Risk surface

Negligible — deletions only; no new code paths. Production (`097c2ed002`)
intentionally still carries the dead keys (harmless) until the next landing
deploy naturally picks up this commit.

## Cross-references

- Parent PR: https://github.com/HackForger/hackforger/pull/186
- Deploy report: `docs/tests/e2e/reports/2026-06-02-pr-186-landing-dates-s2w3.md`
