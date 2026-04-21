# Logo replacement + DESIGN.md theme — design

**Date:** 2026-04-21
**Branch:** `feat/issue-70-logo-theme`
**Issue:** #70 (平台 logo)
**Author:** Allen Woods (with Claude)

## Background

Issue #70 requested a platform logo replacement. Cynthialime delivered:
- 2 SVG logos (365×51, white-bg + orange-bg variants of the same icon + wordmark)
- A full design system spec, DESIGN.md ("High-Energy Kineticism"), covering palette, typography, surfaces, and component rules

This spec covers replacing the logo **and** generating light/dark themes from DESIGN.md, set as default. All changes land in `custom/` so the main codebase is untouched and the customization can be reverted by removing the directory.

## Goals

1. Replace `custom/public/assets/img/logo.svg` and `favicon.svg` with the new brand assets.
2. Single SVG self-adapts to system color scheme (white background under light, orange under dark) using inline `<style>` + `prefers-color-scheme`.
3. Add `hackforger-light` and `hackforger-dark` themes that override the existing bundled theme files via `custom/public/assets/css/`.
4. Bundle Poppins (display) and Manrope (body) as local woff2 fonts under `custom/public/assets/fonts/`.
5. Set `hackforger-dark` as the default theme via `custom/conf/app.ini`.

## Non-goals

- Touching any file outside `custom/` (no Go, no templates, no Vue components).
- Implementing structural DESIGN.md elements that need template changes (glass navbar, overlap typography, energy chips, etc.) — those should be a separate redesign project.
- Migrating existing users away from their saved theme preference. New users + default for unauth visitors get hackforger-dark; existing logged-in users keep whatever they chose.
- Bundling Epilogue / OPPOSans — DESIGN.md mentions these as alternates; the fallback chain `'Poppins', 'Epilogue', system-ui` covers it without bundling extra fonts.

## File layout

```
custom/
├── conf/
│   └── app.ini                                     # MODIFY: add DEFAULT_THEME = hackforger-dark
├── public/assets/
│   ├── img/
│   │   ├── logo.svg                                # REPLACE: combined-variant SVG (auto light/dark)
│   │   └── favicon.svg                             # REPLACE: icon-only square crop of logo
│   ├── css/
│   │   ├── theme-hackforger-light.css              # NEW: ~350 lines, overrides bundled hackforger-light
│   │   └── theme-hackforger-dark.css               # NEW: ~400 lines, overrides bundled hackforger-dark
│   └── fonts/
│       ├── Poppins-Regular.woff2                   # NEW: 400 weight
│       ├── Poppins-Medium.woff2                    # NEW: 500
│       ├── Poppins-SemiBold.woff2                  # NEW: 600
│       ├── Poppins-Bold.woff2                      # NEW: 700
│       ├── Manrope-Regular.woff2                   # NEW: 400
│       ├── Manrope-Medium.woff2                    # NEW: 500
│       └── Manrope-SemiBold.woff2                  # NEW: 600
```

Total: 2 file modifications, 11 new files, ~1.5 MB static assets.

## Logo strategy

### Single self-adapting SVG

The original two SVGs differ only in the rounded-rect background `fill` (`#FAAD14` vs `#fff`). Combine them into one file by replacing the static `fill` with a CSS variable that flips on `prefers-color-scheme`:

```svg
<svg width="365" height="51" viewBox="0 0 365 51" xmlns="http://www.w3.org/2000/svg">
  <style>
    .icon-bg { fill: #fff; }
    @media (prefers-color-scheme: dark) {
      .icon-bg { fill: #FAAD14; }
    }
  </style>
  <rect class="icon-bg" width="85" height="50" rx="25"/>
  <!-- ...remaining paths unchanged from the original... -->
</svg>
```

This works because Forgejo embeds the logo as `<img src=".../logo.svg">` and the browser respects the inline `<style>` block when rendering the SVG. The behavior is independent of which Forgejo theme the user picked — it follows the OS color preference. This is intentional: the brand icon should match the surrounding surface luminance, regardless of whether the user override the theme.

### Favicon

Crop the logo's icon portion (the leftmost 85×50 rounded-rect) to a square and apply the same self-adapting style. Most browser tabs render favicons against the browser chrome, which itself adapts to OS color preference, so the same trick works.

## Theme strategy

### Where the new theme files live

Forgejo loads theme files from two places, in order:
1. `public/assets/css/theme-{name}.css` (bundled, baked into the binary by `make frontend`)
2. `custom/public/assets/css/theme-{name}.css` (overrides at runtime, no rebuild needed)

By placing files in `custom/`, the bundled `theme-hackforger-{light,dark}.css` (steel/zinc palette) is shadowed at runtime without modifying or rebuilding the binary.

### theme-hackforger-dark.css (default)

Implements DESIGN.md "Void and Vibe":

```css
/* Local fonts (privacy: no Google CDN call from internal instance) */
@font-face {
  font-family: 'Poppins'; font-display: swap; font-weight: 400;
  src: url('../fonts/Poppins-Regular.woff2') format('woff2');
}
/* ...repeat for 4 Poppins + 3 Manrope weights = 7 @font-face blocks */

:root {
  /* DESIGN.md §2 — Core palette */
  --color-primary: #BBFD3B;
  --color-primary-dark-1: #A6E532;
  --color-primary-light-1: #C8FE5C;
  --color-secondary: #37f5ef;
  --color-tertiary: #ff8db4;
  --color-error: #ff7351;

  /* DESIGN.md §2.4 — Surface hierarchy (Tonal Layering) */
  --color-body: #0e0e0e;                  /* base */
  --color-box-body: #131313;              /* section */
  --color-box-header: #191919;            /* card */
  --color-box-body-highlight: #262626;    /* elevated */
  --color-secondary-bg: #191919;
  --color-text: #ababab;                  /* on-surface-variant — NOT pure white per §6 Don't */
  --color-text-light: #757575;
  --color-text-light-2: #525252;
  --color-text-light-3: #3d3d3d;

  /* DESIGN.md §2.3 — Ghost border (15% opacity, no hard 1px lines) */
  --color-border: rgba(72, 72, 72, 0.15);
  --color-light-border: rgba(72, 72, 72, 0.10);

  /* Radius scale */
  --border-radius: 4px;
  --border-radius-medium: 8px;            /* cards (§5) */
  --border-radius-large: 20px;            /* modals (§5) */
  --border-radius-full: 9999px;           /* pills */

  /* Typography (§3) */
  --fonts-regular: 'Manrope', 'OPPOSans', system-ui, sans-serif;
  --fonts-display: 'Poppins', 'Epilogue', system-ui, sans-serif;
  --fonts-monospace: ui-monospace, 'Cascadia Code', monospace;
}

/* === Component overrides per DESIGN.md §5 === */

/* Buttons — pill, no shadow */
.ui.button {
  border-radius: 9999px;
  font-family: var(--fonts-regular);
  font-weight: 500;
  letter-spacing: 0.02em;
}
.ui.primary.button {
  background: var(--color-primary);
  color: #0e0e0e;
  box-shadow: none;
}
.ui.primary.button:hover {
  /* §4 — Glow shadow simulating neon */
  box-shadow: 0 0 24px rgba(187, 253, 59, 0.35);
  background: var(--color-primary-light-1);
}
.ui.secondary.button {
  background: transparent;
  border: 1px solid rgba(72, 72, 72, 0.20);  /* Ghost border @ 20% */
  color: var(--color-text);
}

/* Cards — 8px radius, no border */
.ui.card, .ui.cards > .card {
  border-radius: 8px;
  border: none !important;
  background: var(--color-box-header);
  box-shadow: none;
  transition: background 0.15s ease, transform 0.15s ease;
}
.ui.card:hover, .ui.cards > .card:hover {
  background: var(--color-box-body-highlight);
  transform: translateY(-2px);
}

/* Inputs — bottom stroke on focus */
.ui.input input,
.ui.form input[type=text], .ui.form input[type=email], .ui.form textarea {
  background: var(--color-box-body-highlight);
  border: none;
  border-bottom: 2px solid transparent;
  border-radius: 4px 4px 0 0;
  transition: border-color 0.15s ease;
}
.ui.input input:focus,
.ui.form input[type=text]:focus, .ui.form input[type=email]:focus, .ui.form textarea:focus {
  border-bottom-color: var(--color-primary);
  outline: none;
}

/* Modals — 20px radius, glassmorphism backdrop */
.ui.modal { border-radius: 20px; }
.ui.dimmer.modals { background: rgba(14, 14, 14, 0.80); backdrop-filter: blur(20px); }

/* Headlines — display font + tightened tracking (§3) */
h1, h2, .display-lg, .ui.modal .header {
  font-family: var(--fonts-display);
  letter-spacing: -0.02em;
}

/* Labels — uppercase + spaced (§3) */
.ui.label, .label-md {
  font-family: var(--fonts-regular);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 0.75rem;
}

/* §6 Don't — kill 1px solid dividers; use background shifts */
.ui.divider {
  border: none;
  height: 12px;  /* gap instead */
}
```

### theme-hackforger-light.css

Same structure, lighter palette:

```css
:root {
  --color-primary: #BBFD3B;          /* keep brand fluorescent green */
  /* §2.4 inverted surface ladder */
  --color-body: #fafafa;
  --color-box-body: #f4f4f5;
  --color-box-header: #ffffff;
  --color-box-body-highlight: #ececee;
  --color-text: #1a1a1a;             /* not pure black */
  --color-text-light: #525252;
  --color-text-light-2: #757575;
  --color-text-light-3: #ababab;
  /* Ghost border, tightened to be visible on light surface */
  --color-border: rgba(60, 60, 60, 0.12);
  /* radius / typography identical to dark variant */
}

/* Component overrides identical to dark — only colors differ */
```

### Default theme

```ini
[ui]
DEFAULT_THEME = hackforger-dark
```

This applies to:
- All unauthenticated visitors (login page, public repo views)
- Any new user who hasn't set a preference

Existing logged-in users keep their saved choice. To reset everyone, an admin would have to truncate the `user_setting` table — out of scope for this PR.

## Font strategy

Two families × 4+3 weights = 7 woff2 files. Each ~25–35 KB compressed → total ~200 KB. Acceptable for a self-hosted internal tool, and avoids the privacy issue of pulling from `fonts.gstatic.com` on every page load.

Source: download from [google-webfonts-helper](https://gwfh.mranftl.com/fonts) (OFL 1.1 licensed, redistribution allowed). Latin charset only — Forgejo UI is bilingual zh/en but Chinese text falls back to system fonts (PingFang on macOS, Noto Sans CJK on Linux), which is what Cynthialime's design comp seems to assume.

## Commit plan

| # | Subject | Files | Lines |
|---|---------|-------|-------|
| 1 | feat(custom): replace logo and favicon with brand assets | 2 SVG files (logo.svg + favicon.svg, both rewritten) | ~700 lines (SVG paths) |
| 2 | feat(custom): add hackforger themes following DESIGN.md spec | 2 CSS files + 7 woff2 fonts | ~750 CSS + binary fonts |
| 3 | feat(custom): set hackforger-dark as default theme | 1 line in app.ini | 1 |

## Risks and mitigations

| Risk | Mitigation |
|------|-----------|
| `custom/public/assets/css/theme-*.css` doesn't actually override bundled theme | Verify with `curl https://hackforger.inside.h2os.cloud/assets/css/theme-hackforger-dark.css` after deploy — should serve the new file. If not, fallback is to commit to `web_src/css/themes/` instead. |
| Self-adapting SVG `<style>` not respected by Forgejo's image rendering pipeline | Test by opening the SVG file in browser standalone first. If `<style>` is stripped (rare), fall back to L2 (CSS background-image swap on a wrapper element with theme-class selector). |
| Fomantic UI selector specificity beats theme override | Use `!important` sparingly only where needed; verify by inspecting the rendered button/card/input in the running instance. |
| OFL license attribution requirement | Add `OFL.txt` next to fonts (downloaded along with the woff2 from gwfh). |
| Existing users see broken UI if they manually saved a now-renamed theme | We're not renaming themes (`hackforger-light`/`-dark` already exist as bundled steel/zinc). Just shadowing CSS. Selectors / theme names unchanged. |
| Default theme application to anonymous visitors only — logged-in users keep old preference | This is intended. If a global migration is desired, separate admin script. |

## Verification (E2E via agent-browser)

After deploy, capture 3 screenshots from `https://hackforger.inside.h2os.cloud`:

1. **Anonymous landing** (`/`) — should show new logo + dark theme background `#0e0e0e` + Poppins headlines
2. **Logged-in dashboard** (`/`) — should show pill primary button, no-border cards on hover lift, Manrope body text
3. **Switch to hackforger-light** (user settings → theme) — should show light palette with same component shapes

Save reports under `docs/tests/e2e/reports/issue-70-logo-theme-report.md`.

## Open questions

None — all decisions captured.
