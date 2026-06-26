---
pr: 190
commits:
  - a8ae70a293
  - 04ee7891ad
tested_against: https://www.synnovator.com (production — post-deploy curl verification; direct-push fast path, no Mac stage-2 E2E run)
tested_at: 2026-06-26T21:42:00+08:00
e2e_owner: claude
admin_signoff:
  by: allenwoods
  at: 2026-06-26T21:23:00+08:00
  notes: "In-chat fast-deploy authorization (admin, allen.woods@outlook.com): provided PR #190 link with '请将这个landing page的修改推送到生产环境。鉴于只有landing page改动，可以直接推送' — same established merge+direct-to-prod fast path as PR #181 / #186 / #189 for landing-only changes. When the deploy was blocked by the preflight gate, admin was asked and explicitly chose 'Skip the gate (--skip-preflight)'. This is an authorization-to-skip the stage-3 manual walk-through, NOT a record that the admin personally walked the acceptance steps. Scope: this single deploy."
---

# PR #190 — S1/S2 决赛二级页 + 决赛/S3 按钮启用 — E2E smoke test (backfill)

## What shipped

PR #190 (`feat(landing): add S1/S2 finals showcase pages + enable finals & S3
buttons`, landing commit `a8ae70a293`, merge `04ee7891ad`). Static landing
assets only — no Go code, no `go.mod/sum`, no `web_src/`, no `Makefile`
(`git diff 477e2bb1dd..04ee7891ad -- '*.go' go.mod go.sum web_src Makefile`
is empty). The redeploy rebuilt a behavior-identical binary; only the
`custom/public/assets/landing/` files actually changed what users see.

Changes to `custom/public/assets/landing/index.html` (8 add / 7 del):

- **S1-W4 finals button**: was a disabled grey `<button>6月27日-6月28日决赛</button>`,
  now an enabled link `<a href="s1-ranking.html">查看决赛队伍</a>`.
- **S2-W4 finals button**: was disabled grey, now `<a href="s2-ranking.html">查看决赛队伍</a>`.
- **S2-W2 / S2-W3 / S3-W1 buttons**: relabelled `即刻报名` → `查看详情`.
- **S3-W2 button**: was disabled grey `7月1日开始报名`, now enabled
  `<button data-stage="s3-w2">即刻报名</button>`; matching `enabled: false` →
  `enabled: true` in the JS stage-config block.
- i18n dictionary: added `"查看决赛队伍": "View Finalist Teams"`.

Two **new** pages plus their image sets:

- `s1-ranking.html` (S1 决赛项目展示) — 22 ranked project covers under
  `assets/images/s1-covers/` + `S1-mark.webp`.
- `s2-ranking.html` (S2 决赛项目展示) — 19 covers under
  `assets/images/s2-covers/` + `S2-mark.webp` + `s2-final-banner.webp`.

## Link/path resolution (the risk surface)

The landing `index.html` is served at `/` by `routers/web/home.go` (and at
`/lingang-2026`), **not** from its on-disk path. The new buttons use
*relative* hrefs (`href="s1-ranking.html"`). These resolve correctly only
because `index.html` carries `<base href="/assets/landing/">`, so
`s1-ranking.html` → `/assets/landing/s1-ranking.html` — which is exactly where
the file is served. The two new ranking pages have **no** `<base>` tag, so
their relative `assets/images/...` refs resolve against their own URL
(`/assets/landing/`), which is also correct. Verified by checking served
status codes for the actual deployed paths (below).

## Post-deploy verification (production, curl)

Deployed SHA confirmed: `/var/lib/hackforger/.last-deploy` =
`04ee7891adbe4a9675ec75aa7451b0c0c99bd307` = `prod` HEAD.
`/api/v1/version` → `...-176-04ee7891ad+gitea-1.22.0` (new binary live).

HTTP status (all 200):

| URL | status |
| --- | --- |
| `/` | 200 |
| `/assets/landing/s1-ranking.html` | 200 |
| `/assets/landing/s2-ranking.html` | 200 |
| `/assets/landing/assets/images/S1-mark.webp` | 200 |
| `/assets/landing/assets/images/S2-mark.webp` | 200 |
| `/assets/landing/assets/images/s1-covers/rank-01.webp` | 200 |
| `/assets/landing/assets/images/s2-covers/S2_cover_01_supplychainos_3x2.webp` | 200 |

Content checks on `/`:

- ✓ contains `查看决赛队伍`, `href="s1-ranking.html"`, `href="s2-ranking.html"`, `查看详情`
- ✓ S3-W2 button now renders enabled `即刻报名` (the old disabled label
  `7月1日开始报名` survives only as an i18n dictionary key on line 3633, not
  as a button — identical position in local prod file and served file).
- ✓ `s1-ranking.html` served title `S1 决赛项目展示 | 滴水湖全球 OPC 人工智能挑战赛`, 22 `s1-covers/` refs.
- ✓ `s2-ranking.html` served title `S2 决赛项目展示 | 滴水湖全球 OPC 人工智能挑战赛`, 19 `s2-covers/` refs.

## Process notes

- PR #190's base branch (`v0.1-dev/hackforger`) has a `REVIEW_REQUIRED`
  protection policy and no CI checks; merged with `gh pr merge --admin`
  under the admin's in-session authorization.
- `prod` fast-forwarded `477e2bb1dd` → `04ee7891ad` and pushed to origin.
- Deploy via `bash deploy/ecs/redeploy.sh --skip-preflight` (admin-authorized
  gate skip — see `admin_signoff.notes`). This is the documented hotfix/
  direct-push exception in `docs/notes/gitflow.md`; this backfill report is
  the retroactive audit trail it calls for.

## Result

PASS — all landing pages, the relabelled/enabled buttons, the new finals
showcase pages, and their image assets serve correctly on
https://www.synnovator.com at the deployed SHA.
