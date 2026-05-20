---
pr: 177
commits:
  - 8100db9738
  - a6f53b079f
  - 5496489c29
tested_against: https://hackforger.inside.h2os.cloud (smoke env — Mac instance, localhost:3000)
tested_at: 2026-05-20T20:00:00+08:00
e2e_owner: claude
admin_signoff:
  by: allenwoods
  at: 2026-05-20T11:45:00+08:00
  notes: "In-chat authorization: '由于只改动了landing page，我授权你直接推送生产环境'. After the smoke test caught a blocking bug, admin explicitly chose '我修复并完成全流程部署' — authorizing the one-line structural fix (commit 5496489c29) and full direct-to-prod deploy. Scope: this single deploy, landing-page-only change."
---

# PR #177 — auto-rotating hero carousel with S2 banner — E2E smoke test

## What shipped

PR #177 (`feat(landing): auto-rotating hero carousel with S2 banner`,
feat commit `8100db9738`, merge `a6f53b079f`) adds a second hero slide to
the landing page carousel:

- New asset `custom/public/assets/landing/assets/images/轮播图2.webp`
  (377 KB webp, 2400×1093) — the S2 cross-border banner.
- `custom/public/assets/landing/index.html`: un-hides Slide 2 (cloned
  from S1's overlay/buttons), un-hides the second dot indicator, adds the
  `S2·跨境OPC加速赛` en-US i18n entry.

The existing carousel JS (`index.html` lines ~1977–2031) auto-enables
horizontal autoplay (5.5 s) + prev/next + dot controls once it sees more
than one non-hidden slide.

## Blocking bug found — and fixed

The first smoke run **failed**. On the live page the S2 banner never
appeared: the hero stayed on Slide 1, no autoplay, no controls.

**Root cause:** a pre-existing stray `</div>` in `index.html` closed the
`#hero-carousel` track immediately after Slide 1. The browser parser
therefore left Slides 2 and 3 as *siblings* of the track, not children.

DOM evidence (smoke env, before fix):
- `#hero-carousel` had **1** child element (expected 3).
- Slide 2 / Slide 3 had `parentElement !== #hero-carousel`.
- Carousel JS computed `total = 1` → autoplay off, `.hero-carousel-ui`
  (arrows + dots) stayed `hidden`.
- Orphaned Slide 2 rendered at document-y ≈ 652 (a 600 px block below the
  hero) and was clipped invisible by the parent's `overflow-hidden`.

The stray tag was **not introduced by PR #177** — it is an unchanged
context line in the PR diff. It was harmless while Slides 2 & 3 were
`hidden` (`total` was always 1); PR #177 un-hiding Slide 2 exposed it.

**Fix** — commit `5496489c29` (`fix(landing): remove stray </div>
closing hero carousel early`): deletes the one stray `</div>`. Nesting
re-verified line-by-line; all three `.carousel-item` divs are now
children of `#hero-carousel`.

## Verification (smoke env, after fix)

Tested with agent-browser against `http://localhost:3000` after applying
the fix and reloading.

| Check | Result |
| --- | --- |
| `#hero-carousel` child count | **3** ✓ |
| All `.carousel-item` are children of the track | **true** ✓ |
| Slide 1 renders `S1·数智OPC加速赛` | ✓ (`pr177fix-A-slide1.png`) |
| Slide 2 renders `S2·跨境OPC加速赛` + 轮播图2.webp | ✓ (`pr177fix-A-slide2.png`) |
| Autoplay rotates after 5.5 s | ✓ track `translateX(0%)` → `translateX(-100%)` |
| Active dot follows the slide | ✓ `activeDotIdx` 0 → 1 |
| Prev/next arrow controls visible | ✓ `.hero-carousel-ui` un-hidden |
| Dot indicators visible (2 dots, 3rd hidden) | ✓ `dotsVisible [true,true,false]` |
| `轮播图2.webp` asset served | ✓ HTTP 200, 385712 bytes |
| `moveCarousel()` manual nav | ✓ |

Screenshots:
- `screenshots/2026-05-20-pr-177-slide1.png` — Slide 1 (S1·数智OPC加速赛)
- `screenshots/2026-05-20-pr-177-slide2.png` — Slide 2 (S2·跨境OPC加速赛)

## Risk surface

Low — landing-page-only, static-asset change:
- No Go / template / binary / DB impact. `custom/public/` is served as
  static files and rsynced to the cloud by `deploy/ecs/redeploy.sh`.
- Slide 3 remains `hidden`; only Slides 1 + 2 are live (`total = 2`).
- The fix is a pure structural correction — one deleted tag, no behavior
  added.
- Rollback is trivial: re-deploy the previous landing HTML.

## Cross-references

- PR: https://github.com/HackForger/hackforger/pull/177
- PR #177 merge commit: `a6f53b079f`
- Carousel structure fix: `5496489c29`
- Pattern precedent: `docs/tests/e2e/reports/2026-05-14-pr-175-s1w3-landing.md`
