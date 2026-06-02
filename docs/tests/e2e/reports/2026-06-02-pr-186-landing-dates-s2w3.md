---
pr: 186
commits:
  - 5985c42cf0
  - efabe689d8
  - 97e3b3d8ef
tested_against: https://hackforger.inside.h2os.cloud (smoke env — Mac instance on dev tip, accessed via localhost:3000)
tested_at: 2026-06-02T17:15:00+08:00
e2e_owner: claude
admin_signoff:
  by: allenwoods
  at: 2026-06-02T17:17:00+08:00
  notes: "In-chat fast-deploy authorization (admin, allen.woods@outlook.com): '根据 PR #186，合并并部署到冒烟环境和生产环境。鉴于只有落地页修改，可以走快速部署流程' + confirmed '#182/#184 这两个应该已经部署到了生产环境中'. Landing-page-only static change; admin authorized merge + direct-to-prod deploy. Verified dev→cloud delta is #186 only (cloud .last-deploy was 0eef5310f1 / #182+#184 already shipped). Scope: this single deploy. Same in-chat-authorization pattern as PR #177 / #181."
---

# PR #186 — landing date updates + S2W3 open + language switcher relocation (#185) — E2E smoke test

## What shipped

PR #186 (`fix(landing): update dates & enable S2W3 + relocate language
switcher`, landing commit `5985c42cf0`, merge `97e3b3d8ef`). Single
user-facing file: `custom/public/assets/landing/index.html` (static asset,
served from disk, no `make` required). 6 changes for issue #185:

| # | Change |
| --- | --- |
| 1 | S1W4 finals date `6月17日-6月18日` → `6月27日-6月28日` (tag + button + rule-stage-01 markdown) |
| 2 | S2W4 finals date `6月17日-6月18日` → `6月27日-6月28日` (tag + button + rule-stage-02 markdown) |
| 3 | S2W3 registration opened — config `enabled: false → true`; button grey-disabled「6月4日开始报名」→ green clickable「即刻报名」(slug `opc-2026-crossborder-w3`) |
| 4 | Language switcher moved from footer → header top-right (next to login) |
| 5 | Schedule table S1/S2 rows `决赛圈协创松（线下）：6月17日-6月18日` → `6月27日-6月28日` |
| 6 | i18n: added EN entries `6月27日-6月28日`→`Jun 27 – Jun 28`, `6月27日-6月28日决赛`→`Finals Jun 27–28`, `决赛圈协创松（线下）：6月27日-6月28日`→EN |

## Verification

Mac instance rebuilt + restarted on dev tip `97e3b3d8ef` via
`scripts/restart-gitea.sh` (HTTP 200), tested with agent-browser against
`http://localhost:3000`.

### DOM / interaction (zh)

| Check | Result |
| --- | --- |
| Language switcher in `<header>`, not `<footer>` | ✓ `inHeader:true, inFooter:false` (`02-zh-header.png`) |
| S2W3 button `即刻报名`, `disabled=false`, primary `rgb(187,253,59)` | ✓ |
| S2W3 config `{slug:'opc-2026-crossborder-w3', enabled:true}` | ✓ |
| S1W4 button → `6月27日-6月28日决赛` (stays disabled — finals) | ✓ |
| S2W4 button → `6月27日-6月28日决赛` (stays disabled — finals) | ✓ |
| Schedule table rows show `6月27日-6月28日` | ✓ |
| Old visible `决赛圈协创松（线下）：6月17日-6月18日` removed from page | ✓ (only dead i18n keys remain — see note) |
| Full page renders cleanly | ✓ (`01-zh-fullpage.png`) |

### EN mode

| Check | Result |
| --- | --- |
| S1W4 / S2W4 buttons → `Finals Jun 27–28` | ✓ |
| Schedule rows → `Jun 27 – Jun 28` | ✓ |
| Full EN page renders | ✓ (`03-en-fullpage.png`) |

Screenshots (`screenshots/2026-06-02-pr-186/`):
- `01-zh-fullpage.png` — zh landing (S2W3 now green, finals cards new dates)
- `02-zh-header.png` — header top-right with relocated language switcher
- `03-en-fullpage.png` — full landing in English with `Jun 27 – Jun 28`

## Risk surface

Low — landing-page-only, single static-file change:
- No Go / template / binary / DB impact. `custom/public/` is served as static
  files and rsynced to the cloud by `deploy/ecs/redeploy.sh`.
- dev→cloud delta is exactly #186 (cloud `.last-deploy` = `0eef5310f1`;
  #182/#184 already shipped) — this deploy ships only the landing change.
- Rollback is trivial: re-deploy the previous landing HTML.

## Notes / minor observations

- **Dead i18n keys (cosmetic):** the EN dictionary still keeps the old
  `6月17日-6月18日` / `…决赛` / `决赛圈协创松（线下）：6月17日-6月18日` entries
  (lines ~3240/3489/3636) alongside the new `6月27日-6月28日` ones. They no
  longer match any visible text, so they never render — harmless leftover,
  worth a cleanup later but not blocking.
- **Stray root PNGs:** commit `efabe689d8` added 4 verification screenshots
  (`01-..04-*.png`) to the **repo root** instead of `docs/`. They are not
  under `custom/public/`, so `redeploy.sh` does not ship them to the cloud;
  they only clutter the repo root. Recommend removing in a follow-up.

## Cross-references

- PR: https://github.com/HackForger/hackforger/pull/186 (Closes #185)
- Merge commit: `97e3b3d8ef`
- Pattern precedent: `docs/tests/e2e/reports/2026-06-01-pr-181-landing-0529-en.md`
