# E2E: Issue #34 — Bounty Timeline i18n + Username

**Goal:** Confirm Timeline comments on a bounty's linked Issue (a) show the applicant's username (not `user #<id>`), (b) render in the repo owner's language, and (c) the HackForger Feed continues to show the parallel event unchanged.

## Prereqs

- Server built on this branch running on `http://localhost:3000` with fresh frontend + bindata:

  ```bash
  make frontend
  TAGS="bindata sqlite sqlite_unlock_notify" make backend
  ./gitea web
  ```

- Test accounts (all with password `admin1234`):
  - `hackforger` (admin) — publisher of bounty #38 (`hackforger/oss-project` Issue #9)
  - `hacker_eve` — publisher of bounty #39 (`e2e-test-2026/ai-innovation` Issue #1)
  - `hacker_frank` — applicant

**Report:** write findings to `docs/tests/e2e/reports/issue-34-timeline-i18n-report.md`.

## Scenario A — zh-CN repo owner

1. Confirm `hackforger` user Language is `zh-CN` (or empty, instance default). Go to `/user/settings/appearance`.
2. Login as `hacker_frank`.
3. Visit `/hackforger/oss-project/issues/9` (bounty #38). Apply via the Vue panel (or POST to `/hackforger/oss-project/bounties/38/apply`).
4. Reload the Issue. **Expected:** new Timeline comment `📋 悬赏申请 — **hacker_frank** 申请承接`. Screenshot as `tests/screenshots/issue-34/a-apply-zh.png`.
5. Try to apply again. **Expected:** HTTP 422 `{"error":"您已经申请过此悬赏。"}` (no new comment).
6. Login as `hackforger`. Accept hacker_frank's application.
7. Reload. **Expected:** Timeline comment `🏷 悬赏已接受 — **hacker_frank** 认领`. Screenshot as `a-accept-zh.png`.

## Scenario B — en-US repo owner

1. As `hackforger`, change Language to `English (US)` via POST to `/user/settings/appearance/language` with `language=en-US`.
2. Cancel bounty #38 (POST to `/hackforger/oss-project/bounties/38/cancel`).
3. Reload the Issue. **Expected:** Timeline comment `❌ Bounty Cancelled`. Screenshot as `b-cancel-en.png`.
4. Revert language: POST `language=zh-CN` to the same endpoint.

## Scenario C — HackForger Feed regression

1. Visit the dashboard (`/`) as `hackforger`.
2. **Expected:** Feed shows entries like `hackforger cancelled a bounty in Design new logo` (en-US, because the dashboard viewer's language drives Feed rendering — that is the HackForger Feed's i18n design).
3. Screenshot as `c-feed-regression.png`.

## PASS / FAIL criteria

- **PASS** when:
  - Scenario A shows Chinese Apply + Accept comments with username
  - Scenario B shows English Cancel comment
  - Scenario C shows Feed unchanged (renders via per-viewer locale, as before)
- **FAIL** if any Timeline comment still shows `user #<id>`, renders in the wrong language for the repo owner's setting, or if the Feed entry is missing.

## Notes

- The Complete path (`✅ 悬赏已完成 — 交付已接受并发放` / `✅ Bounty Completed — delivery accepted and bounty fulfilled`) was not exercised end-to-end because completion requires a merged PR. Its code path is identical to the other three (same `postBountyIssueComment` helper, same locale mechanism), so confidence transfers.
- All three exercised paths (Apply / Accept / Cancel) exercise both the with-args (`%s` username substitution) and no-args flows.
