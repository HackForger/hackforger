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

**Verified call-sites** (`grep` results):
| File | Markup | Rendered size |
|---|---|---|
| `templates/base/head_navbar.tmpl:10` | `<img width="30" height="30" src=".../logo.svg" alt="..." aria-hidden="true">` (no class) | 30×30 (header, every page) |
| `templates/home.tmpl:5` | `<img class="logo" width="220" height="220" src=".../logo.svg" alt="...">` | 220×220 (anonymous landing) |
| `templates/user/auth/signin_openid.tmpl:6` | `<img width="100" height="100" src=".../logo.svg" alt="...">` (no class) | 100×100 |
| `templates/status/500.tmpl:22` | `<img width="30" height="30" src=".../logo.svg" alt="..." aria-hidden="true">` (no class) | 30×30 |

A class-based selector misses 3 of 4 sites. **Selector must be an attribute selector**:

```css
/* theme-hackforger-light.css override */
img[src$="/img/logo.svg"] {
    content: url("/assets/img/logo-light.svg");
}
```

```css
/* theme-hackforger-dark.css override */
img[src$="/img/logo.svg"] {
    content: url("/assets/img/logo-dark.svg");
}
```

`src$="/img/logo.svg"` matches any `<img>` whose src ends with `/img/logo.svg` — covers both `{{AssetUrlPrefix}}` resolved with and without subpath.

**Layout safety:** both source and target SVGs share the same intrinsic dimensions (`viewBox="0 0 365 51"`). Author-set `width`/`height` attributes take precedence; layout boxes are unchanged. The header's pre-existing 30×30 squashing of a 365×51 SVG is a Forgejo design choice, not introduced by this spec.

**Accessibility:** `content: url()` on `<img>` preserves the element's accessible name from `alt`. Verified via WAI-ARIA: replaced-content does not strip `alt`. Test plan adds VoiceOver + NVDA verification at the home page logo (the one without `aria-hidden`).

**Browser support:** `content: url()` on `<img>`: Chrome 65+, Firefox 60+, Safari 9+ (caniuse "css-content"). Fallback: default `logo.svg` ships as `logo-light` content (set in §3.1), so unsupported browsers see the orange light logo regardless of theme — degraded but functional.

### 3.3 Favicon

**Out of scope for this spec.** Two reasons:
1. Favicon was not in the user-reported defects (logo white-text + light too white).
2. Per-theme favicon swap requires JS (link[rel=icon] href change) plus careful viewBox math (favicons must be effectively-square; source is 365×51 with stroke overflow). Reviewer verified the icon math in the previous draft was wrong (85×50 clips the green leaf stroke; favicons need square viewBox like 85×85 with vertical centering — non-trivial).

**Action:** Leave existing `favicon.svg` / `favicon.png` untouched. File a follow-up issue if per-theme favicon becomes important.

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

### 3.8 Cyan/Pink colorize — cleanup only

**Drop** the `.ui.label.cyan` / `.ui.label.pink` rules from PR #84. They never fire because Forgejo labels use inline `style="background:#hex"` set per-label by users, not class-based colors. Verified: zero class-based label usages exist for `.cyan` / `.pink` in the codebase. The rules are dead CSS.

Also remove the `.ui.label.green` rule for the same reason (Forgejo doesn't emit `class="label green"` — it emits `class="ui label" style="..."`).

**Aspirational chip work is out of scope.** Targeted HackForger chip styling (bounty status, hackathon phase, credit tx direction) requires both new CSS AND new template/Vue markup with stable class names. None of those classes exist today. That work belongs in a separate "HackForger chip system" spec, not bolted onto this theme polish.

Net effect: the colorize section of both themes shrinks to one paragraph cleanup; no new selectors added.

## 4. File-by-file changes

### `custom/public/assets/img/`
- **CREATE** `logo-light.svg` (= `/tmp/issue70/logo-1.bin` verbatim — orange variant. Strip any `<style>` or `@media` block; verify with `grep -c '<style\|@media' logo-light.svg` returns 0)
- **CREATE** `logo-dark.svg` (= `/tmp/issue70/logo-2.bin` verbatim — green variant. Same strip + verify)
- **MODIFY** `logo.svg` — replace contents with the contents of `logo-light.svg` (default fallback for any non-themed caller, e.g., browsers that don't support `content: url()`)
- **DO NOT TOUCH** `favicon.svg` / `favicon.png` (out of scope per §3.3)

### `custom/public/assets/css/theme-hackforger-light.css`
- Replace the surface-ladder block (section 3.4 values).
- Replace `a:hover` color (3.5).
- Replace `:focus-visible` outline color (3.5).
- Replace primary button hover shadow (3.7).
- Append `img[src$="/img/logo.svg"] { content: url("/assets/img/logo-light.svg"); }`.
- **Remove** the `.ui.label.cyan/.pink/.green` rules from PR #84 (3.8).

### `custom/public/assets/css/theme-hackforger-dark.css`
- Append surface-lift `inset` shadow rules (3.6).
- Replace `a:hover` color (3.5).
- Replace `:focus-visible` outline color (3.5).
- Append `img[src$="/img/logo.svg"] { content: url("/assets/img/logo-dark.svg"); }`.
- **Remove** the `.ui.label.cyan/.pink/.green` rules from PR #84 (3.8).

## 5. Testing strategy

Manual E2E only — no unit/integration tests for theme CSS. After merge, run `make frontend && make backend` (per CLAUDE.md), restart `gitea web`, then in browser:

### 5.1 Light theme
1. Hard-refresh (Cmd+Shift+R).
2. **Logo at all 4 call-sites** (per §3.2 table): header navbar (every page), `/` home, `/user/login/openid` signin, 500 error page if reproducible. Each must render the orange variant; black "Syn" letters visible against warm paper.
3. **Surface depth**: body bg reads as cream (not white) from 3ft viewing distance.
4. **Card lift inversion**: cards/segments are clearly lighter than body (paper-on-desk metaphor visible).
5. **Hover quietness**: hover any link on an issue list — text shifts to `--color-primary-dark-1` (#A8E632), no strobe to fluorescent.
6. **Focus ring**: Tab through a form — outline is dark-green-1, 3px offset.
7. **Primary button hover**: green tint shadow visible.

### 5.2 Dark theme
1. Switch theme via user prefs → reload.
2. **Logo at all 4 call-sites**: green variant; white "Syn" letters visible against deep-black body.
3. **Surface lift**: side panel / repo file tree / nav menu show 1px white-4% top-edge highlight (subtle, intentional, only visible on calibrated displays).
4. Same hover/focus quieter checks.

### 5.3 Theme toggle behavior
Switch theme in user prefs and confirm: the logo changes **on the next page navigation** (not live, because the theme CSS file is loaded by `<link>` and Forgejo reloads the page on theme change anyway). If logo is stale after navigation, it indicates aggressive browser caching of `logo-light.svg` / `logo-dark.svg` — hard-refresh resolves; document if seen.

### 5.4 Accessibility
- **Screen reader announcement**: VoiceOver (Cmd+F5 on macOS) on the home page logo — should announce "logo" (or the localized translation). The header logo has `aria-hidden="true"` so should be skipped — verify it IS skipped.
- **WCAG AA contrast** between `--color-text` (resolved value: check `getComputedStyle`) and the new light surface tokens `#efece1` / `#e7e3d4` / `#ddd8c5` / `#e3decd` / `#f6f3e9`. Use a contrast checker; document the actual ratios as a CSS comment block. Minimum 4.5:1 for `--color-text` body, 3:1 for `--color-text-light` (large or non-essential text).
- **Color-only meaning**: confirm the logo swap doesn't lose information for monochrome users (the brand mark is icon + word; both variants encode "HackForger" in the wordmark).

### 5.5 Browser support smoke test
Verify on Safari (Mac), Chrome, Firefox: the `content: url()` swap takes effect. If any browser falls back to default `logo.svg`, that's acceptable degradation per §3.2 (default = light variant ships).

### 5.6 Frontend rebuild
Per CLAUDE.md: worktree must run `make frontend` before `make backend`. Confirm `public/assets/` rebuild is fresh by checking timestamps after build.

## 6. Risks & mitigations

| Risk | Mitigation |
|---|---|
| `content: url()` on `<img>` not supported on older browsers | Default `logo.svg` ships as light variant; unsupported browsers see the light logo regardless of theme — degraded but functional. Test plan §5.5 verifies on 3 browsers. |
| Forgejo caches CSS / SVG aggressively after theme swap | Test plan §5.3 calls out hard-refresh requirement; document if observed. |
| Surface depths feel too dark vs. PR #84 light | User explicitly asked "太白了" (too white) → directional risk is low. Values committed in §3.4 are reversible in a follow-up if needed. |
| Logo SVGs contain designer's `<style>` or `@media` rules that conflict with the static copy approach | §4 mandates `grep -c '<style\|@media'` returns 0 after copying. |
| Light surface tokens reduce contrast against `--color-text` below WCAG AA | Test plan §5.4 makes contrast verification explicit. If a token fails, narrow the surface step (e.g., `#efece1` → `#f0ede2`) until AA passes. |
| `content: url()` strips `<img alt>` from accessibility tree | Test plan §5.4 verifies VoiceOver behavior. If broken, fall back to dropping the swap on that single surface and accept light-variant logo there. |

## 7. Out of scope

- Per-theme favicon (browser caching makes this non-trivial; needs JS).
- Component-level redesign (we're token-only; deeper component changes would belong in a UI modernization spec).
- Dark theme surface ladder change (PR #84 settled it).
- Animation/motion changes.
- Mobile viewport adjustments.
