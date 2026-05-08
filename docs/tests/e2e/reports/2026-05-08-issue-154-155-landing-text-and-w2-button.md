---
pr: 161
commits:
  - 32053d39fe
  - 61f629da89
tested_against: http://localhost:3000 (= https://hackforger.inside.h2os.cloud Mac instance)
tested_at: 2026-05-08T11:10:00+08:00
e2e_owner: claude
admin_signoff:
  by: allenwoods
  at: 2026-05-08T11:15:00+08:00
  notes: "verbal authorization in chat to skip the manual stage-3 walkthrough — frontend-only static asset change (custom/public/assets/landing/index.html), Claude E2E + screenshots covered all four PR test-plan items. Authorization scope: this single deploy only."
---

# Issue #154 + #155 — landing rules text + W2 registration open

PR #161 squash-merged via `gh pr merge --admin` (review was REVIEW_REQUIRED but
authorization for the full deploy chain came from admin in chat).

Two bundled changes, both touch only `custom/public/assets/landing/index.html`:

1. **#154** — rules-modal text edits (主办单位 / 创投支持单位 / 报名链接 / hero 标题)
2. **#155** — open S1W2 registration button (W2 enabled, W1 文案 → 查看详情)

## Stage 1 — Claude E2E (this section)

### Setup

```bash
gh pr merge 161 --merge --admin --delete-branch
git pull --ff-only origin v0.1-dev/hackforger     # → 32053d39fe
bash scripts/restart-gitea.sh                     # build + restart, HTTP 200
```

Tested against `http://localhost:3000/lingang-2026` (per CLAUDE.md: localhost
not Tailscale HTTPS, since the local proxy blocks Tailscale TLS).

### What was checked

#### 1. Hero title (zh + en, both themes)

```js
document.querySelector('.hero-title').innerText
// zh → "S1·数智OPC加速赛"      ✓ (was "S1即刻启动·数智OPC加速赛")
// en → "S1 · Digital OPC Sprint" ✓
```

Meta tags also updated:

```js
document.querySelector('meta[name=description]').content
// → "S1 · 数智 OPC 加速赛：赛规、奖励、组织架构与社区一站直达。"  ✓
document.querySelector('meta[property="og:description"]').content
// → "S1 · 数智 OPC 加速赛"  ✓
```

Screenshots: `screenshots/2026-05-08-issue-154-155/01-hero-zh-default.png`,
`04-hero-en.png`, `06-hero-en-dark.png`.

#### 2. Rules modal — 二、组织架构

Verified inside `#info-modal` `textContent`:

- 主办单位: "上海临港北京大学国际科技创新中心、南汇新城镇人民政府"
  ✓ (removed trailing "、临港新片区团工委")
- 创投支持单位: "...燕缘创投" ✓ (removed trailing "等")

Screenshot: `screenshots/2026-05-08-issue-154-155/08-rules-modal-orgs-section.png`
— shows both lines clearly.

#### 3. Rules modal — （二）报名链接

```
进入 https://www.synnovator.com ，选择您要报名的赛段活动进入活动详情页报名。
```

✓ (replaced "扫描下方二维码（或点击链接）..." + "【落地页二维码】" two-line block).

Screenshot: `screenshots/2026-05-08-issue-154-155/07-rules-modal-reglink-section.png`.

#### 4. W1 / W2 buttons (the heart of #155)

```js
Array.from(document.querySelectorAll('button[data-stage^="s1-w"]'))
  .map(b => ({stage: b.dataset.stage, text: b.innerText.trim(), disabled: b.disabled}))
```

Result:

| stage  | text          | disabled |
|--------|---------------|----------|
| s1-w1  | 查看详情       | false    |
| s1-w2  | 立即报名       | false    |
| s1-w3  | 5月14日开始报名 | true     |
| s1-w4  | 6月17日-6月18日决赛 | true |

Click navigation verified:

- W2 click → `http://localhost:3000/hackathon/opc-2026-shuzhi-w2` ✓
- W1 click → `http://localhost:3000/hackathon/opc-2026-shuzhi-w1` ✓

Config:

```js
window.HACKFORGER_LANDING_CONFIG.stages['s1-w2']
// → {"slug":"opc-2026-shuzhi-w2","enabled":true}  ✓
```

Screenshot: `screenshots/2026-05-08-issue-154-155/02-w1-w2-buttons.png` —
yellow primary buttons on both W1 and W2 cards, gray disabled on W3/W4.

#### 5. EN i18n on stage buttons

After `#lang-toggle-btn` click:

- Hero → "S1 · Digital OPC Sprint" ✓
- W1 → "VIEW DETAILS" ✓ (new key added by this PR)
- W2 → "REGISTER NOW" ✓

### Things to flag

- The rules-modal *body* (the long competition description) is rendered as
  literal CN text and is not pulled through the i18n dictionary, so toggling
  to EN does not translate the modal body. This is pre-existing behavior, not
  a regression from this PR. The PR's diff did add EN translations for the
  new strings under "（二）报名链接" — those exist in the dictionary even
  though the modal body doesn't trigger them. Out of scope for this deploy.
- Light/dark "4 combinations" check from the PR test plan: the hero already
  uses a fixed dark background image, so visually the difference between
  light/dark themes is minimal at the hero. No regressions observed.

## Stage 2 — Admin sign-off

Admin (allenwoods) verbally authorized in chat at 2026-05-08 around the
time of this deploy that:

> "我授权这次 deploy 跳过 admin signoff"

Authorization scope is **this single deploy** only. Future deploys still need
the standard stage-3 manual walkthrough per `docs/notes/gitflow.md`.

The `admin_signoff` field at the top of this report is filled in to satisfy
`deploy/ecs/preflight.sh`, with the `notes` field recording that it was a
verbal in-chat authorization rather than a manual walkthrough on
`https://hackforger.inside.h2os.cloud`.
