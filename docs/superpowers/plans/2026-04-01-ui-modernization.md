# UI Modernization Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Fomantic UI classes in all 32 HackForger templates with Tailwind-first `.hf-*` component classes for a distinctive, developer-focused visual identity with light/dark theme support.

**Architecture:** Two new CSS files define color tokens and component classes. A status_badge helper template centralizes status→color mapping. Templates are migrated module-by-module. Vue components update template section only.

**Tech Stack:** Tailwind CSS 3.4 (tw- prefix), Go html/template SSR, CSS custom properties, Forgejo Octicon SVGs

**Spec:** `docs/superpowers/specs/2026-04-01-ui-modernization-design.md`

---

## Chunk 1: CSS Foundation + Helper Template

### Task 1: Create color token CSS file

**Files:**
- Create: `web_src/css/hackforger-colors.css`

- [ ] **Step 1: Create hackforger-colors.css with light/dark color tokens**

```css
/* HackForger color tokens — light mode defaults, dark mode overrides */
:root {
  --hf-green-bg: #d1f5dc;
  --hf-green-text: #14532d;
  --hf-blue-bg: #c8dffc;
  --hf-blue-text: #0c3b7c;
  --hf-red-bg: #fcd5d0;
  --hf-red-text: #7f1d1d;
  --hf-yellow-bg: #fde68a;
  --hf-yellow-text: #713f12;
  --hf-purple-bg: #e4ccf8;
  --hf-purple-text: #4a1d7a;
  --hf-teal-bg: #c4f1f9;
  --hf-teal-text: #0e5e6f;
  --hf-neutral-bg: #e4e7ec;
  --hf-neutral-text: #374151;
}

[data-theme="gitea-dark"],
[data-theme="gitea-auto"][data-system-theme="dark"],
[data-theme="forgejo-dark"],
[data-theme="forgejo-auto"][data-system-theme="dark"] {
  --hf-green-bg: rgba(86,211,100,0.18);
  --hf-green-text: #56d364;
  --hf-blue-bg: rgba(121,192,255,0.16);
  --hf-blue-text: #79c0ff;
  --hf-red-bg: rgba(255,123,114,0.16);
  --hf-red-text: #ff7b72;
  --hf-yellow-bg: rgba(227,179,65,0.18);
  --hf-yellow-text: #e3b341;
  --hf-purple-bg: rgba(188,140,255,0.16);
  --hf-purple-text: #bc8cff;
  --hf-teal-bg: rgba(86,212,221,0.16);
  --hf-teal-text: #56d4dd;
  --hf-neutral-bg: #30363d;
  --hf-neutral-text: #9eaab6;
}
```

**Deviation from spec:** The spec says `@media (prefers-color-scheme)` but Forgejo actually uses `data-theme` attribute on `<html>`, not media queries. The selectors above are correct for Forgejo's theming system. Check `templates/base/head.tmpl` for the theme attribute pattern. Do NOT use `@media (prefers-color-scheme)` — it won't work with Forgejo's theme picker.

- [ ] **Step 2: Commit**

```bash
git add web_src/css/hackforger-colors.css
git commit -m "feat(ui): add HackForger color token CSS variables"
```

### Task 2: Create component class CSS file

**Files:**
- Create: `web_src/css/hackforger.css`

- [ ] **Step 1: Create hackforger.css with all component classes**

Components to define (see spec for exact properties):
- `.hf-badge` + variants `.hf-badge-green`, `.hf-badge-blue`, `.hf-badge-red`, `.hf-badge-yellow`, `.hf-badge-purple`, `.hf-badge-teal`, `.hf-badge-neutral`
- `.hf-card`, `.hf-card-head`, `.hf-card-body`
- `.hf-timeline`, `.hf-timeline-step`, `.hf-step-done`, `.hf-step-active`, `.hf-step-upcoming`, `.hf-step-name`, `.hf-step-date`, `.hf-step-status`
- `.hf-stat`, `.hf-stat-value`, `.hf-stat-label`
- `.hf-row`, `.hf-row-left`
- `.hf-btn`, `.hf-btn-primary`, `.hf-btn-secondary`
- `.hf-count`

Each badge variant uses the corresponding `--hf-{color}-bg` and `--hf-{color}-text` tokens. Surface colors use Forgejo variables (`var(--color-box-body)`, `var(--color-box-header)`, `var(--color-secondary)`).

- [ ] **Step 2: Commit**

```bash
git add web_src/css/hackforger.css
git commit -m "feat(ui): add HackForger component CSS classes"
```

### Task 3: Import CSS files in index.css

**Files:**
- Modify: `web_src/css/index.css` (add 2 import lines before `@tailwind utilities`)

- [ ] **Step 1: Add imports before the @tailwind utilities line**

```css
@import "./hackforger-colors.css";
@import "./hackforger.css";

@tailwind utilities;
```

- [ ] **Step 2: Commit**

```bash
git add web_src/css/index.css
git commit -m "feat(ui): import HackForger CSS in main stylesheet"
```

### Task 4: Create status_badge helper template

**Files:**
- Create: `templates/hackforger/helpers/status_badge.tmpl`

- [ ] **Step 1: Create the status badge helper**

This template renders a badge span given `.Status` (int) and `.Type` (string: "hackathon", "bounty", "registration", "tier").

The mapping logic:
- hackathon: 0→neutral, 1→green, 2→blue, 3→yellow, 4→neutral, 5→red
- bounty: 0→green, 1→blue, 2→yellow, 3→purple, 4→teal, 5→red, 6→neutral
- registration: 0→yellow, 1→green, 2→red
- tier (string match): "Diamond"→purple, "Gold"→yellow, "Silver"→neutral, "Bronze"→yellow

Output: `<span class="hf-badge hf-badge-{color}">{{.Label}}</span>`

Note: Go templates don't support function-style helpers easily. Instead, use a `{{define}}` block with `{{if}}` chains, invoked via `{{template "hackforger/helpers/status_badge" dict "Status" .Status "Type" "hackathon" "Label" .StatusLabel}}`.

Check if Forgejo's template system supports `dict` helper — look for `dict` usage in existing templates. If not available, use separate define blocks per type: `status_badge_hackathon`, `status_badge_bounty`, etc.

- [ ] **Step 2: Commit**

```bash
git add templates/hackforger/helpers/status_badge.tmpl
git commit -m "feat(ui): add status_badge helper template"
```

---

## Chunk 2: Hackathon Templates (7 files)

### Task 5: Migrate hackathon/view.tmpl

**Files:**
- Modify: `templates/hackforger/hackathon/view.tmpl`

- [ ] **Step 1: Rewrite the template**

Key changes:
- Header: `tw-flex` layout stays, replace `ui label` status badge → use status_badge helper
- Timeline section: replace `ui segment` + `ui three column grid` → `.hf-timeline` with `.hf-step-done`/`.hf-step-active`/`.hf-step-upcoming` based on current hackathon status
- Description/Prizes: replace `ui segment` → `.hf-card` with `.hf-card-head` + `.hf-card-body`
- Tracks: replace `ui divided list` → `.hf-card` with `.hf-row` items
- Registration form: keep `ui form` and `ui dropdown` (Fomantic JS-dependent), wrap in `.hf-card`
- Participants: replace `ui divided list` → `.hf-card` with `.hf-row` items, registration status badges via helper
- Submissions table: keep as table (truly tabular data), wrap in `.hf-card`, replace `ui celled table` → minimal table styling
- Buttons: replace `ui button` → `.hf-btn .hf-btn-primary` / `.hf-btn-secondary`

- [ ] **Step 2: Commit**

### Task 6: Migrate hackathon/explore.tmpl

**Files:**
- Modify: `templates/hackforger/hackathon/explore.tmpl`

- [ ] **Step 1: Rewrite**

- Replace `ui relaxed divided list` → `.hf-card` with `.hf-row` items
- Replace inline status color conditionals → status_badge helper
- Replace `ui placeholder segment` empty state → `.hf-card` with centered icon + text
- Keep `{{template "hackforger/explore_search" .}}` and `{{template "base/paginate" .}}` unchanged

- [ ] **Step 2: Commit**

### Task 7: Migrate hackathon/manage.tmpl

**Files:**
- Modify: `templates/hackforger/hackathon/manage.tmpl`

- [ ] **Step 1: Rewrite**

- Phase Control: replace `ui segment` → `.hf-card`, `ui label` → status_badge, `ui green/blue/orange button` → `.hf-btn` variants
- Tracks section: replace `ui segment` → `.hf-card`, keep `ui form` for add track form
- Scoring Criteria: replace `ui segment` + `ui celled table` → `.hf-card` with table inside (truly tabular — keep table but remove `ui celled`)
- Track Scoring Overrides: same pattern — `.hf-card` wrapper, keep table + `ui form` for inline edit
- Registrations: `.hf-card`, replace action buttons → `.hf-btn`
- Judges: `.hf-card` with `.hf-row` items, keep `ui form` for add judge
- Finalize Preview: `.hf-card`, keep table for ranked results

- [ ] **Step 2: Commit**

### Task 8: Migrate hackathon/new.tmpl, submit.tmpl, leaderboard.tmpl, judge.tmpl

**Files:**
- Modify: `templates/hackforger/hackathon/new.tmpl`
- Modify: `templates/hackforger/hackathon/submit.tmpl`
- Modify: `templates/hackforger/hackathon/leaderboard.tmpl`
- Modify: `templates/hackforger/hackathon/judge.tmpl`

- [ ] **Step 1: Migrate new.tmpl and submit.tmpl**

Both are form pages. Replace `ui segment` wrappers → `.hf-card`. Keep `ui form` class. Replace `ui button` → `.hf-btn`.

- [ ] **Step 2: Migrate leaderboard.tmpl**

Replace `ui celled table` → table inside `.hf-card`. Use `.hf-badge` for tier display. Highlight #1 row with green-tinted background.

- [ ] **Step 3: Migrate judge.tmpl**

Minimal template — just a mount point for Vue component. Replace `ui container` wrapper with `.hf-card` if needed. The Vue component (JudgeScoreCard) handles its own rendering.

- [ ] **Step 4: Commit all four**

```bash
git add templates/hackforger/hackathon/
git commit -m "feat(ui): migrate all hackathon templates to hf-* classes"
```

---

## Chunk 3: Bounty + Credits Templates (12 files)

### Task 9: Migrate bounty templates (4 files)

**Files:**
- Modify: `templates/hackforger/bounty/panel.tmpl`
- Modify: `templates/hackforger/bounty/badge.tmpl`
- Modify: `templates/hackforger/bounty/explore.tmpl`
- Modify: `templates/hackforger/bounty/new.tmpl`

- [ ] **Step 1: Migrate panel.tmpl**

Key changes:
- Replace `ui segment` → remove outer segment (panel is embedded in issue sidebar)
- Replace `ui small label {color}` status badge → status_badge helper with type "bounty"
- Replace `ui mini label` reward chips → `.hf-badge` with border style (reward chips, not status)
- Winner list: `.hf-row` items
- Vue mount point (`#hackforger-bounty-panel`) stays unchanged

- [ ] **Step 2: Migrate badge.tmpl, explore.tmpl, new.tmpl**

- `badge.tmpl`: simple inline badge, replace Fomantic label → `.hf-badge`
- `explore.tmpl`: listing page, similar pattern to hackathon/explore
- `new.tmpl`: form page, keep `ui form`, wrap in `.hf-card`

- [ ] **Step 3: Commit**

### Task 10: Migrate credits templates (8 files)

**Files:**
- Modify: `templates/hackforger/credits/dashboard.tmpl`
- Modify: `templates/hackforger/credits/overview.tmpl`
- Modify: `templates/hackforger/credits/redeem.tmpl`
- Modify: `templates/hackforger/credits/orders.tmpl`
- Modify: `templates/hackforger/credits/admin/credits.tmpl`
- Modify: `templates/hackforger/credits/admin/keys.tmpl`
- Modify: `templates/hackforger/credits/admin/options.tmpl`
- Modify: `templates/hackforger/credits/admin/orders.tmpl`

- [ ] **Step 1: Migrate dashboard.tmpl**

Key changes:
- Replace `ui statistic` → `.hf-stat` with `.hf-stat-value` and `.hf-stat-label`
- Replace `ui celled table` → table inside `.hf-card`
- Green/red amount colors: use `tw-text-green` / `tw-text-red` (Tailwind mapped to Forgejo CSS vars)
- Replace `ui placeholder segment` empty state → `.hf-card` with centered content

- [ ] **Step 2: Migrate overview.tmpl, redeem.tmpl, orders.tmpl**

- overview: stats + transaction list, same patterns as dashboard
- redeem: form page, keep `ui form`, wrap in `.hf-card`
- orders: table listing, `.hf-card` wrapper

- [ ] **Step 3: Migrate admin templates (4 files)**

Admin pages use `ui celled table` extensively. Replace with tables inside `.hf-card`. Replace `ui button` → `.hf-btn`. These are internal-facing so lower visual priority.

- [ ] **Step 4: Commit**

```bash
git add templates/hackforger/bounty/ templates/hackforger/credits/
git commit -m "feat(ui): migrate bounty + credits templates to hf-* classes"
```

---

## Chunk 4: Grants + Reputation + Explore + Vue (remaining)

### Task 11: Migrate grants templates (7 files)

**Files:**
- Modify: All 7 files in `templates/hackforger/grants/`

- [ ] **Step 1: Migrate detail.tmpl**

Key change: replace `ui six steps` progress bar → `.hf-timeline` (reuse same component, adapt for grant statuses: draft → open → review → finalized → distributed). Replace `ui segment` → `.hf-card`. Replace `ui label` → `.hf-badge`.

- [ ] **Step 2: Migrate remaining 6 grant templates**

- `explore.tmpl`: listing with `.hf-card` + `.hf-row`
- `manage.tmpl`: admin form, keep `ui form`, `.hf-card` wrappers
- `manage_project.tmpl`: detail view, `.hf-card`
- `new.tmpl`: form, keep `ui form`
- `projects.tmpl`: listing, `.hf-card` + `.hf-row` or table
- `submit.tmpl`: form, keep `ui form`

- [ ] **Step 3: Commit**

### Task 12: Migrate reputation templates (3 files)

**Files:**
- Modify: `templates/hackforger/reputation/leaderboard.tmpl`
- Modify: `templates/hackforger/reputation/card.tmpl`
- Modify: `templates/hackforger/reputation/detail.tmpl`

- [ ] **Step 1: Migrate leaderboard.tmpl**

Replace `ui celled striped table` → table inside `.hf-card`. Tier badges via status_badge helper with type "tier". Highlight #1 row.

- [ ] **Step 2: Migrate card.tmpl and detail.tmpl**

- `card.tmpl`: compact reputation display, use `.hf-badge` for tier
- `detail.tmpl`: user profile reputation, `.hf-card` sections

- [ ] **Step 3: Commit**

### Task 13: Migrate explore + feed templates (3 files)

**Files:**
- Modify: `templates/hackforger/explore.tmpl`
- Modify: `templates/hackforger/explore_search.tmpl`
- Modify: `templates/hackforger/feed/community_feeds.tmpl`

- [ ] **Step 1: Migrate all three**

- `explore.tmpl`: multi-tab explore page, replace `ui divided items` → `.hf-card` + `.hf-row`
- `explore_search.tmpl`: search form component, minimal changes
- `community_feeds.tmpl`: feed listing, `.hf-card` + `.hf-row`

- [ ] **Step 2: Commit**

### Task 14: Update Vue components

**Files:**
- Modify: `web_src/js/components/hackforger/BountyPanel.vue` (template section only)
- Modify: `web_src/js/components/hackforger/JudgeScoreCard.vue` (template section only)

- [ ] **Step 1: Update BountyPanel.vue template**

Replace in `<template>` section:
- `ui small negative message` → `hf-card` with red border or keep as-is (error messages)
- `ui small green button` → `hf-btn hf-btn-primary`
- `ui small button` → `hf-btn hf-btn-secondary`
- `ui mini green/red button` → `hf-btn hf-btn-primary` / `hf-btn-secondary` (smaller)
- `ui mini label` → `hf-badge`
- `ui form` → keep (Fomantic JS-dependent for form behavior)
- `ui mini icon button` → `hf-btn hf-btn-secondary`
- `ui mini input` → keep (Fomantic form input)
- Layout: `tw-flex` utilities stay unchanged

- [ ] **Step 2: Update JudgeScoreCard.vue template**

Replace in `<template>` section:
- `ui secondary pointing menu` → keep (tab navigation, Fomantic JS)
- `ui segment` → `hf-card` with `hf-card-body`
- `ui indicating progress` → keep (Fomantic progress bar) or replace with simple Tailwind bar
- `ui mini green label` → `hf-badge hf-badge-green`
- `ui primary button` → `hf-btn hf-btn-primary`
- `ui placeholder segment` → `hf-card` with centered empty state
- `ui form`, `ui error message` → keep (Fomantic form JS)

- [ ] **Step 3: Commit**

```bash
git add templates/hackforger/grants/ templates/hackforger/reputation/ templates/hackforger/explore*.tmpl templates/hackforger/feed/ web_src/js/components/hackforger/
git commit -m "feat(ui): migrate grants, reputation, explore, feed templates + Vue components"
```

---

## Chunk 5: Build, Verify, Screenshot

### Task 15: Build frontend and backend

- [ ] **Step 1: Build frontend**

```bash
make frontend
```

Expected: successful build, no CSS/JS errors.

- [ ] **Step 2: Build backend**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
```

Expected: successful compilation, no template parse errors.

- [ ] **Step 3: Fix any build errors**

If CSS class names are wrong or templates have syntax errors, fix and rebuild.

### Task 16: Start server and take screenshots

- [ ] **Step 1: Copy app.ini and start server**

```bash
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini
rm -f data/queues/common/LOCK
./gitea web &
```

Wait for server to start on http://localhost:3000.

- [ ] **Step 2: Use agent-browser to take screenshots**

Screenshot the following pages (login as hackforger/admin1234 first):
1. Hackathon explore page: `/explore/hackathons`
2. A hackathon detail page (if data exists)
3. Credits dashboard: `/user/credits`
4. Reputation leaderboard: `/explore/reputation`
5. Bounty panel on an issue (if data exists)

Save screenshots to `docs/tests/e2e/screenshots/ui-modernization/`.

- [ ] **Step 3: Commit screenshots**

```bash
git add docs/tests/e2e/screenshots/ui-modernization/
git commit -m "docs: add UI modernization screenshots"
```

### Task 17: Final commit

- [ ] **Step 1: Ensure all changes are committed**

```bash
git status
```

- [ ] **Step 2: Create summary commit if needed**
