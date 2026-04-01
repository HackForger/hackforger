# HackForger UI Modernization — Tailwind-First Design System

**Date**: 2026-04-01
**Status**: Approved
**Approach**: B — Tailwind-First Templates + Custom Components

## Problem

HackForger's 32 custom templates use Fomantic UI classes (`ui segment`, `ui celled table`) identical to Forgejo's core pages. This creates two issues:

1. **No visual identity** — hackathon, bounty, credits, and reputation pages look like repo settings pages
2. **Flat hierarchy** — every section uses the same `ui segment` pattern, giving equal visual weight to all content

## Design Direction

**Style**: Developer-focused, clean (Google Workspace-inspired)
**Theme**: Light/dark dual-theme, mapping to Forgejo's existing CSS variable system
**Icons**: SVG line icons via Forgejo's built-in Octicon library (`{{svg "octicon-*"}}`)
**Typography**: System font stack + monospace accents for data values only
**Density**: Compact, information-dense — respects developer preferences

## Scope

**In scope**: All 32 templates in `templates/hackforger/` and 2 Vue components in `web_src/js/components/hackforger/`

**Out of scope**: Forgejo upstream templates, Fomantic UI removal, new JS frameworks

## Color System

### Design Principle

Surface colors (backgrounds, text, borders) map to Forgejo's existing `--color-*` CSS variables for automatic theme switching. Badge/status colors are **new custom properties** defined in `hackforger-colors.css` (e.g., `--hf-green-bg`, `--hf-green-text`) with `@media (prefers-color-scheme)` variants — they do NOT reuse upstream Forgejo color variables.

Badge colors use higher saturation than standard Forgejo for stronger visual distinction. Light mode uses opaque tinted backgrounds; dark mode uses `rgba()` overlays.

### Light Mode Tokens

| Token | Background | Text | Usage |
|-------|-----------|------|-------|
| green | `#d1f5dc` | `#14532d` | Open, Approved, Done, positive amounts |
| blue | `#c8dffc` | `#0c3b7c` | Claimed, Active, info states |
| red | `#fcd5d0` | `#7f1d1d` | Cancelled, Rejected, negative amounts |
| yellow | `#fde68a` | `#713f12` | Pending, Warning, Gold tier |
| purple | `#e4ccf8` | `#4a1d7a` | Diamond tier, special states |
| teal | `#c4f1f9` | `#0e5e6f` | Paid, Completed |
| neutral | `#e4e7ec` | `#374151` | Silver tier, inactive |

### Dark Mode Tokens

| Token | Background | Text | Usage |
|-------|-----------|------|-------|
| green | `rgba(86,211,100,0.18)` | `#56d364` | Same as light |
| blue | `rgba(121,192,255,0.16)` | `#79c0ff` | Same as light |
| red | `rgba(255,123,114,0.16)` | `#ff7b72` | Same as light |
| yellow | `rgba(227,179,65,0.18)` | `#e3b341` | Same as light |
| purple | `rgba(188,140,255,0.16)` | `#bc8cff` | Same as light |
| teal | `rgba(86,212,221,0.16)` | `#56d4dd` | Same as light |
| neutral | `#30363d` | `#9eaab6` | Same as light |

### Surface Colors

| Surface | Light | Dark |
|---------|-------|------|
| Page background | `--color-body` (maps to `#f5f6f8`) | `--color-body` (maps to `#0e1117`) |
| Card background | `--color-box-body` | `--color-box-body` |
| Card header / subtle bg | `--color-box-header` | `--color-box-header` |
| Borders / dividers | `--color-secondary` | `--color-secondary` |
| Primary text | `--color-text` | `--color-text` |
| Secondary text | `--color-text-light` | `--color-text-light` |
| Tertiary text | `--color-text-light-2` | `--color-text-light-2` |

## Component Library

All components use Tailwind `tw-*` utilities in templates. Shared patterns are extracted into CSS classes in `web_src/css/hackforger.css` to avoid duplication.

### `.hf-badge`

Status badge with color variants. Replaces Fomantic `ui label`.

```html
<span class="hf-badge hf-badge-green">Approved</span>
```

Properties:
- `padding: 2px 8px`
- `border-radius: 4px`
- `font-size: 0.7rem`
- `font-weight: 600`
- Variants: `hf-badge-green`, `hf-badge-blue`, `hf-badge-red`, `hf-badge-yellow`, `hf-badge-purple`, `hf-badge-teal`, `hf-badge-neutral`

### `.hf-card`

Content container with optional header. Replaces Fomantic `ui segment`.

```html
<div class="hf-card">
  <div class="hf-card-head">
    {{svg "octicon-people" 16}} Participants
    <span class="hf-count">5</span>
  </div>
  <div class="hf-card-body">...</div>
</div>
```

Properties:
- `border: 1px solid var(--color-secondary)`
- `border-radius: 8px` (matches `--border-radius-medium`)
- `overflow: hidden`
- Header: `padding: 10px 14px`, subtle background (`var(--color-box-header)`)
- Body: `padding: 12px 14px`

### `.hf-timeline`

Horizontal phase indicator for hackathon stages. New component.

```html
<div class="hf-timeline">
  <div class="hf-timeline-step hf-step-done">
    <div class="hf-step-name">Registration</div>
    <div class="hf-step-date">Mar 15</div>
    <div class="hf-step-status">Done</div>
  </div>
  <div class="hf-timeline-step hf-step-active">...</div>
  <div class="hf-timeline-step hf-step-upcoming">...</div>
</div>
```

Properties:
- `display: flex`, border on container, steps separated by left border
- `hf-step-done`: green tinted background
- `hf-step-active`: blue tinted background
- `hf-step-upcoming`: neutral background, muted text
- Text centered, uppercase label, date below

### `.hf-stat`

Large numeric display for dashboard metrics. Replaces Fomantic `ui statistic`.

```html
<div class="hf-stat">
  <div class="hf-stat-value tw-text-green">2,450</div>
  <div class="hf-stat-label">Balance</div>
</div>
```

Properties:
- Value: `font-size: 1.75rem`, `font-weight: 400`, `font-variant-numeric: tabular-nums`
- Label: `font-size: 0.65rem`, uppercase, `letter-spacing: 0.06em`
- Centered layout

### `.hf-row`

List row with left/right content alignment. Replaces Fomantic `ui divided list`.

```html
<div class="hf-row">
  <div class="hf-row-left">
    {{ctx.AvatarUtils.Avatar .User 24}}
    <span>{{.User.Name}}</span>
  </div>
  <span class="hf-badge hf-badge-green">Approved</span>
</div>
```

Properties:
- `display: flex`, `align-items: center`, `justify-content: space-between`
- `padding: 8px 14px` (when inside card with no card-body padding)
- Separated by `border-bottom: 1px solid var(--color-secondary-alpha-40)`

### `.hf-btn`

Action button. Replaces Fomantic `ui button`.

```html
<a class="hf-btn hf-btn-primary">{{svg "octicon-check" 14}} Mark Complete</a>
```

Properties:
- `padding: 6px 16px`, `border-radius: 4px`, `font-size: 0.8rem`, `font-weight: 500`
- `hf-btn-primary`: `var(--color-primary)` fill (Forgejo's brand blue), white text
- `hf-btn-secondary`: subtle background, border, default text color

### `.hf-count`

Inline counter pill for card headers.

Properties:
- `padding: 1px 6px`, `border-radius: 10px`, `font-size: 0.7rem`, `font-weight: 600`
- Neutral background

## Status Color Mapping

Centralizes the currently scattered `{{if eq .Status 0}}green{{else if...}}` logic.

### Hackathon Status → Color

| Status | Value | Color |
|--------|-------|-------|
| Draft | 0 | neutral |
| Open | 1 | green |
| Hacking | 2 | blue |
| Judging | 3 | yellow |
| Finished | 4 | neutral |
| Cancelled | 5 | red |

### Bounty Status → Color

| Status | Value | Color |
|--------|-------|-------|
| Open | 0 | green |
| Claimed | 1 | blue |
| In Review | 2 | yellow |
| Completed | 3 | purple |
| Paid | 4 | teal |
| Expired | 5 | red |
| Cancelled | 6 | neutral |

### Registration Status → Color

| Status | Value | Color |
|--------|-------|-------|
| Pending | 0 | yellow |
| Approved | 1 | green |
| Rejected | 2 | red |

### Reputation Tier → Color

| Tier | Color |
|------|-------|
| Diamond | purple |
| Gold | yellow |
| Silver | neutral |
| Bronze | yellow (reuses `hf-badge-yellow`) |

## File Structure

```
web_src/css/hackforger.css          — Component classes (.hf-badge, .hf-card, etc.)
web_src/css/hackforger-colors.css   — Color token definitions (light + dark)
templates/hackforger/               — Rewritten templates using tw-* + .hf-* classes
```

CSS files are imported via `web_src/css/index.css` (appended to existing imports).

## Template Migration Strategy

Templates are migrated module-by-module in this order:

1. **Hackathon** (7 templates) — highest-traffic, most visual components
2. **Bounty** (4 templates) — includes Vue component coordination
3. **Credits** (8 templates, incl. 4 admin) — dashboard with stat displays
4. **Grants** (7 templates) — forms and detail views
5. **Reputation** (3 templates) — leaderboard and cards
6. **Explore/Feed** (3 templates) — listing pages

### Responsive Behavior

HackForger targets desktop-first developers. No mobile-specific breakpoints are required. `hf-timeline` uses `tw-flex` and will naturally wrap on narrow viewports via `tw-flex-wrap`. `hf-stat` grids use `tw-flex` with `tw-gap` — on very small screens they stack vertically.

### Migration Rules

- Replace `class="ui segment"` → `class="hf-card"` with appropriate `hf-card-head`/`hf-card-body`
- Replace `class="ui celled table"` → semantic `hf-row` lists (tables only for truly tabular data like submissions)
- Replace `class="ui label ..."` → `class="hf-badge hf-badge-{color}"`
- Replace `class="ui button"` → `class="hf-btn hf-btn-{variant}"`
- Replace `class="ui statistic"` → `class="hf-stat"`
- Replace inline status color conditionals with a `{{template "hackforger/helpers/status_badge"}}` partial that accepts `.Status` and `.StatusType` (hackathon/bounty/registration/tier) and outputs the correct `hf-badge-{color}` class
- Keep Fomantic classes for: `ui form`, `ui dropdown`, `ui modal` (interactive JS-dependent components)
- Layout utilities remain Tailwind: `tw-flex`, `tw-gap-*`, `tw-mb-*`, etc.

## Vue Component Updates

`BountyPanel.vue` and `JudgeScoreCard.vue` templates update to use `hf-*` classes. No logic changes needed — only the `<template>` section CSS classes change.

## Visual Mockups

Approved mockups are at:
- `.superpowers/brainstorm/69200-1775014946/visual-style-v3.html`

These show light/dark versions of: hackathon detail, credits dashboard, bounty sidebar, reputation leaderboard.

## Non-Goals

- No Fomantic UI removal from Forgejo core pages
- No shadcn-vue or new component library introduction
- No SPA conversion — remains Go SSR + Vue progressive enhancement
- No new Vue components (template-level changes only)
- No changes to Go backend code (models, services, routers)
