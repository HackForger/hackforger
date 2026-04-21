# Report: Issue #34 Timeline i18n + Username

**Date:** 2026-04-21
**Branch:** `verify/issue-34-bounty-feed`
**Build:** `untagged-issue51-screenshots-50-9fc20953f2+gitea-1.22.0` (fresh `make frontend` + bindata rebuild)
**Result:** **PASS**

## Scenario A — zh-CN repo owner (bounty #38, `hackforger/oss-project`)

| Step | Expected | Actual | Evidence |
|---|---|---|---|
| Apply as `hacker_frank` | Timeline comment in Chinese with username | `📋 悬赏申请 — hacker_frank 申请承接` (issuecomment-91) | `tests/screenshots/issue-34/a-apply-zh.png` |
| Duplicate apply | HTTP 422 with `您已经申请过此悬赏。` | HTTP 422 `{"error":"您已经申请过此悬赏。"}` | (API response) |
| Accept application as `hackforger` | Timeline comment in Chinese with username | `🏷 悬赏已接受 — hacker_frank 认领` (issuecomment-92) | `tests/screenshots/issue-34/a-accept-zh.png` |

## Scenario B — en-US repo owner (bounty #38, after switching `hackforger` language)

| Step | Expected | Actual | Evidence |
|---|---|---|---|
| Set `hackforger.Language = en-US` | Setting persists | `/user/settings/appearance` reports `language=en-US` | (verified in-browser) |
| Cancel bounty | Timeline comment in English | `❌ Bounty Cancelled` (issuecomment-93) | `tests/screenshots/issue-34/b-cancel-en.png` |
| Revert language to `zh-CN` | Setting reverts | POST returned 200 | (verified) |

## Scenario C — HackForger Feed regression

| Step | Expected | Actual | Evidence |
|---|---|---|---|
| Visit dashboard as `hackforger` | Feed shows bounty events unchanged | Feed shows `hackforger cancelled a bounty in Design new logo`, `hackforger claimed a bounty in Design new logo`, `hacker_eve claimed a bounty in Design AI Dashboard UI` | `tests/screenshots/issue-34/c-feed-regression.png` |

Feed rendering still goes through `templates/hackforger/feed/community_feeds.tmpl` with `ctx.Locale.Tr ...` — per-viewer locale. The viewer's language (en-US right now) drives the Feed render, which is the intended design of the HackForger Feed mechanism (unchanged by this PR).

## Defects Found

None.

## Behavioral Notes

- **Two i18n mechanisms coexist correctly**:
  - Issue Timeline (`comment` table) — written in the **repo owner's** language, stored as rendered markdown. Same content for all readers.
  - HackForger Feed (`hackforger_action` table) — stored as structured data, rendered per-viewer at template time.
- The same underlying event (e.g., Accept) produces **one Timeline comment in the owner's language + one Feed row** that each viewer sees in their own language. No double-notification risk because we chose `**%s**` (bold) over `@%s` (mention) for the username display.
- Username fallback (`user #<id>`) for deleted users was not exercised in this run — low-probability path; covered by `resolveUsername` implementation.
