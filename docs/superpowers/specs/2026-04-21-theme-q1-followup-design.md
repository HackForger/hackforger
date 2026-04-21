# Theme Q1 Follow-up — Logo Switching + Light Depth + Polish — Design Spec

**Date:** 2026-04-21
**Branch:** fix/theme-light-and-logo (will be created)
**Predecessor:** PR #84 (initial /quieter /bolder /harden /colorize pass — merged)
**Related:** Issue #70 (logo replacement), `/critique` Q1 report

## 1. Problem

Two user-reported defects + four open `/critique` recommendations from the Q1 polish round, addressed together as one focused theme follow-up:

1. **Logo bug.** Site currently ships a single `logo.svg` derived from the designer's `logo-2` (DARK variant: white icon bg + 5 white wordmark paths including the "Syn" letters in white). The SVG uses `@media (prefers-color-scheme: dark)` to swap the icon background rect color, but does **not** swap the wordmark text fills. On the LIGHT theme the white "Syn" letters render against the warm-paper body and disappear visually.

2. **Light theme too white.** Body at `#fbfaf7` (99% luminance) + cards at pure `#ffffff` reads as stark white from arm's-length viewing. The "warm paper & ink" design intent committed in PR #84 lives only in comments, not in pixels.

3. **Hover strobe (carry-over from /critique).** `a:hover { color: var(--color-primary) }` fires `#BBFD3B` on every link cross. Dense pages (issue lists, file trees) feel like a strobe.

4. **Focus ring uses loudest color (carry-over).** `outline: 2px solid var(--color-primary)` makes keyboard nav visually noisy.

5. **Surface lift missing on dark (carry-over).** `#131313 → #1d1d1d` step is invisible on uncalibrated displays. DESIGN.md §4 ambient-shadow rule was specified but not applied.

6. **Light primary button hover too quiet (carry-over).** `box-shadow: 0 2px 8px 0 rgba(0,0,0,0.08)` is invisible on warm paper.

7. **Cyan/Pink colorize aimed at empty markup (carry-over).** `.ui.label.cyan/.pink` selectors never fire because Forgejo labels use inline `style="background:#hex"`, not class-based colors.

## 2. Goals & Non-Goals

### Goals
- Logo correctly switches with theme class (orange-on-light / green-on-dark), no OS-preference dependency.
- Light theme commits to warm-paper depth (~93% body, lifted cards).
- All four critique items resolved without further upstream component-level changes (theme tokens + minor CSS only).
- Each change is rationalized in the file with the **why**, not just the **what**.

### Non-Goals
- No upstream Forgejo template forks for logo. (Reject approach C.)
- No JS swap mechanism. CSS-only.
- No new component overrides beyond what's needed for the seven defects.
- No rework of dark theme surface tokens (PR #84 settled those — only ADD lift shadow).

## 3. Architecture

### 3.1 Logo files

Three new assets in `custom/public/assets/img/`:
- `logo-light.svg` — copy of `/tmp/issue70/logo-1.bin` verbatim, **strip** any `<style>` block / `@media` rules so it's a pure designer SVG.
- `logo-dark.svg` — copy of `/tmp/issue70/logo-2.bin` verbatim, same strip.
- `favicon-light.svg`, `favicon-dark.svg` — same treatment for the small icon-only variants. (Source SVGs: extract the 85×50 icon portion from each logo, viewBox cropped.)

The existing `logo.svg` becomes a **default fallback** (= `logo-light.svg` content) so any caller that doesn't go through theme CSS still gets a working logo.

### 3.2 Logo swap mechanism (CSS only)

Forgejo's header template emits the logo as `<img class="ui logo full" src="{{AppSubUrl}}/assets/img/logo.svg" alt="...">`.

We can't change the `src` from CSS, but we CAN swap the displayed image using `content: url(...)` on the img element under a body-class selector. This is a well-supported technique in WebKit/Chromium/Firefox.

In each theme override block:

```css
/* theme-hackforger-light.css override */
img.logo, img.full.logo {
    content: url("/assets/img/logo-light.svg");
}
```

```css
/* theme-hackforger-dark.css override */
img.logo, img.full.logo {
    content: url("/assets/img/logo-dark.svg");
}
```

This works because the active body class loads exactly one theme CSS, which sets `content:` to the matching variant. No JS, no template forks, no Forgejo `Site Logo` setting changes.

### 3.3 Favicon swap

Browser caches favicons aggressively and does not re-fetch on theme change. **Decision:** ship `favicon-light.svg` as the default `<link rel="icon">` (orange icon — visible against typical browser-tab backgrounds, both light and dark Chrome/Safari themes). Do NOT attempt a runtime swap. Document this limitation in the spec.

If the user later wants per-theme favicons, the right fix is a JS one-liner that updates the link href on theme change — out of scope for this spec.

### 3.4 Light theme surface ladder

Replace the warm-cream-but-too-bright values in PR #84:

| Token | PR #84 | This spec | Δ luminance |
|---|---|---|---|
| `--color-body` | #fbfaf7 (99%) | **#efece1** (93%) | -6% |
| `--color-box-body` | #f3f1ea (96%) | **#e7e3d4** (90%) | -6% |
| `--color-box-body-highlight` | #e8e5db (91%) | **#ddd8c5** (86%) | -5% |
| `--color-box-header` | #ebe8df (93%) | **#e3decd** (88%) | -5% |
| `--color-card` | #ffffff (100%) | **#f6f3e9** (96%, ABOVE body) | -4% but inverts to be lighter than body |
| `--color-active` | #e0ddd2 (88%) | **#cfc9b3** (80%) | -8% |
| `--color-secondary` | #d4d1c5 (84%) | **#c8c2ab** (78%) | -6% |

Rationale: 6-point downward shift makes the cream tint actually visible at typical viewing distance (>3ft). Cards inverting to `#f6f3e9` (lighter than `#efece1` body) creates the "fresh paper sheet on aged desk" metaphor — fundamental to "warm paper" working as a design language.

### 3.5 Hover + focus quieter pass

Both themes:

```css
/* Hover lifts to dark-1 (still distinct, no strobe) */
a:hover, a.muted:hover, a.suppressed:hover {
    color: var(--color-primary-dark-1);   /* dark: #A8E632 ; light: also #A8E632 ish */
}

/* Focus ring is dark-1 with offset for readability */
a:focus-visible, button:focus-visible, input:focus-visible,
textarea:focus-visible, select:focus-visible,
.ui.button:focus-visible, .ui.dropdown:focus-visible {
    outline: 2px solid var(--color-primary-dark-1);
    outline-offset: 3px;
}
```

Reserves raw `--color-primary` (#BBFD3B) for `:active` and current-tab states only.

### 3.6 Dark surface lift (DESIGN.md §4)

Append to dark theme override:

```css
/* Subtle 1px highlight at top edge — ambient lift without a line */
.ui.segment, .ui.card, .ui.menu, .ui.attached.header,
.repository .header-wrapper, .ui.tabular.menu {
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}
```

Uses `inset` so it lives ON the surface (no false drop-shadow), `0 1px 0` for top-edge only, 4% white = visible on calibrated displays, invisible on bad ones (graceful degradation).

### 3.7 Light primary button hover

```css
/* Replace black-8% with green-tint shadow */
.ui.primary.button:hover {
    box-shadow: 0 2px 12px 0 rgba(111, 161, 24, 0.18);
    background-color: var(--color-primary-dark-1);
}
```

### 3.8 Cyan/Pink colorize — aim at HackForger surfaces

**Drop** the `.ui.label.cyan` / `.ui.label.pink` overrides from PR #84 (selectors don't fire — Forgejo labels use inline style).

**Add** targeted styling for HackForger-controlled chip surfaces. Verify these class names exist in the codebase before writing the rules; if a name differs, update the rule. Initial targets:

- `.hackforger-bounty-status[data-status]` — color by status (open=green, in-review=cyan, complete=pink-faded)
- `.hackforger-phase-pill[data-kind]` — color by phase kind (registration=cyan, judging=pink, finalize=green)
- `.hackforger-credit-tx[data-direction]` — credit (incoming) green-tint, debit (outgoing) pink-tint

If no such class exists today, add an `extract` step in the implementation plan to grep for the existing class names and adapt selectors. **No new template/component changes** — only CSS that targets existing markup.

If after a quick grep no HackForger chip markup is found, defer the colorize-targeted-pass to a separate spec and remove the unused `.ui.label.cyan/.pink` rules from the current themes. Don't ship dead CSS.

## 4. File-by-file changes

### `custom/public/assets/img/`
- **CREATE** `logo-light.svg` (= `/tmp/issue70/logo-1.bin`, strip `<style>` block if any)
- **CREATE** `logo-dark.svg` (= `/tmp/issue70/logo-2.bin`, strip `@media` block)
- **CREATE** `favicon-light.svg` (cropped icon-only viewBox of logo-1)
- **CREATE** `favicon-dark.svg` (cropped icon-only viewBox of logo-2)
- **MODIFY** `logo.svg` — replace with the contents of `logo-light.svg` (default fallback for any non-themed caller)
- **MODIFY** `favicon.svg` — replace with `favicon-light.svg` content (default fallback; orange icon visible on any browser tab background)

### `custom/public/assets/css/theme-hackforger-light.css`
- Replace the surface-ladder block (section 3.4 values).
- Replace `a:hover` color (3.5).
- Replace `:focus-visible` outline color (3.5).
- Replace primary button hover shadow (3.7).
- Append `img.logo, img.full.logo { content: url("/assets/img/logo-light.svg"); }`.
- Remove the `.ui.label.cyan/.pink` rules; add (or stub) the HackForger-targeted chip rules per 3.8.

### `custom/public/assets/css/theme-hackforger-dark.css`
- Append surface-lift `inset` shadow rules (3.6).
- Replace `a:hover` color (3.5).
- Replace `:focus-visible` outline color (3.5).
- Append `img.logo, img.full.logo { content: url("/assets/img/logo-dark.svg"); }`.
- Same colorize cleanup as light (3.8).

## 5. Testing strategy

Manual E2E only — no unit/integration tests for theme CSS:

1. Restart `gitea web` after merge.
2. Hard-refresh `localhost:3000` in browser (Cmd+Shift+R).
3. Verify on **light theme**:
   - Logo: orange-bg icon + black "Syn" letters visible against warm paper body
   - Body bg is visibly cream (not stark white) — should read as "paper" from 3ft
   - Cards (issue cards, repo summary) are clearly lighter than body
   - Tab through a form: focus ring is dark-green-1, not fluorescent
   - Hover any link in an issue list: text shifts to dark-1, no strobe
   - `.ui.primary.button` hover: visible green tint shadow
4. Verify on **dark theme**:
   - Logo: white-bg icon + green wordmark + white "Syn" letters visible against deep-black body
   - Side panel / repo file tree shows visible top-edge highlight (1px white at 4% opacity)
   - Hover/focus same quieter behavior
5. Toggle theme in user prefs → confirm logo swaps without page reload (or with one reload, depending on how Forgejo applies theme switch).
6. Browser tab: favicon shows the orange icon (same in both themes — accepted limitation).

## 6. Risks & mitigations

| Risk | Mitigation |
|---|---|
| `content: url()` on `<img>` not supported in older browsers | Caniuse shows >97% global; acceptable. Fallback is the default `logo.svg` (which we set to logo-light), so worst case all users see the light logo regardless of theme — degraded but functional. |
| Forgejo caches CSS aggressively, theme swap doesn't appear | Document hard-refresh requirement in test plan. |
| `.hackforger-bounty-status` etc. class names don't exist today | Spec acknowledges (3.8): grep first, defer colorize-targeted pass to separate spec if no markup found. Prevents shipping dead CSS. |
| Surface depths feel too dark for users used to PR #84 light | Subjective; user explicitly asked "太白了" so directional risk is low. Spec commits to one set of values; if user disagrees post-merge, easy revert/tweak in a follow-up. |
| Logo SVGs (logo-1/logo-2) might still contain designer's `<style>` block referencing classes that conflict | Spec mandates STRIP any `<style>` and `@media` blocks during the copy step; verify with `grep` after copying. |

## 7. Out of scope

- Per-theme favicon (browser caching makes this non-trivial; needs JS).
- Component-level redesign (we're token-only; deeper component changes would belong in a UI modernization spec).
- Dark theme surface ladder change (PR #84 settled it).
- Animation/motion changes.
- Mobile viewport adjustments.
