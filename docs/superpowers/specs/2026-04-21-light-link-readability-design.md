# Light Theme Link Readability Hotfix — Design Spec

**Date:** 2026-04-21
**Branch:** fix/light-link-readability (will be created)
**Predecessor:** PR #87 (light header + favicon)
**Triggered by:** User report on `/explore/repos` — "fluorescent green text almost unreadable"
**Audit basis:** `/impeccable:audit` Q2 report (this conversation)

## 1. Problem

PR #84 introduced a light theme `--color-primary-dark-*` override scale designed for dark-theme symmetry. On cream paper (`#efece1`), this scale fails WCAG AA across the board:

| Token | Value | Contrast on `#efece1` | WCAG AA (4.5:1)? |
|---|---|---|---|
| `--color-primary-dark-1` | `#A8E632` | 1.85:1 | ❌ FAIL |
| `--color-primary-dark-2` (link rest) | `#6FA118` | **2.62:1** | ❌ FAIL |
| `--color-primary-dark-3` (link hover, focus) | `#5A8413` | **3.77:1** | ❌ FAIL |
| `--color-primary-dark-4` (visited) | `#45660F` | 5.63:1 | ✓ pass |

Symptom: every link on every list-heavy page (explore/repos, issue lists, file trees, commit logs) is unreadable. The `/explore/repos` page is the most acute because ~80% of its visible content is link text (repo names, owner names, language tags).

Three secondary defects compound the problem:

- **Focus ring uses the same color as hover** (both `#5A8413`) — keyboard users cannot distinguish focus from hover state.
- **Backdrop-filter fallback** still references the pre-PR-#87 body color `rgba(251, 250, 247, 0.97)` — modal background no longer matches body on browsers without `backdrop-filter`.
- **`--color-text-light` (`#595959`)** lands at exactly 4.84:1 against `--color-box-body-highlight` (`#ddd8c5`) — barely AA, no margin.

## 2. Goals & Non-Goals

### Goals
- Every link text combination on light theme passes WCAG AA (4.5:1) against every surface it appears on.
- Focus ring is visually distinct from both link rest and link hover.
- Backdrop-filter fallback matches the actual body color.
- Brand identity preserved: fluorescent `#BBFD3B` stays as the action-surface color (button fills, current-tab indicators), only the **ink scale** (text on paper) shifts to deeper greens.

### Non-Goals
- No dark theme changes (audit confirmed dark inherits a working bundled mint scale).
- No `--color-secondary` rename (MED-1, separate spec).
- No automated CI contrast checker (short-term recommendation, separate task).
- No focus-ring redesign beyond color (the 2px outline + 3px offset structure is fine).

## 3. Architecture

### 3.1 The "ink vs neon" rule

DESIGN.md positions `#BBFD3B` as the high-energy action color ("the neon pulse"). On dark theme that color works as link/text because the deep-black surface gives it 13:1 contrast. On cream paper, the same color drops to 1.85:1.

The fix is to keep two parallel scales for light theme:

- **Action scale** (button fills, current-state highlights): `--color-primary: #BBFD3B`. Bright, draws the eye to clickable surfaces. Text overlaid on these uses `--color-primary-contrast: #1a1a1a` (verified 13.8:1).
- **Ink scale** (link text, focus rings on text): `--color-primary-dark-2/3/4`, all dark forest greens that pass AA on cream. This is what the original Forgejo bundled values approximated; we make them slightly more saturated to keep the "green brand" feeling.

This is what DESIGN.md actually wanted. PR #84 conflated them, treating one variable for both meanings. This spec separates them.

### 3.2 New light primary-dark-* scale + uncoupled hover/active

**Critical correction from reviewer pass:** the bundled light theme defines `--color-primary-hover: var(--color-primary-dark-2)` and `--color-primary-active: var(--color-primary-dark-4)`. Without overriding these, changing `dark-2` would also change button hover background — and a fluorescent button suddenly turning near-black-green on hover is jarring. So we **uncouple them** with explicit overrides.

Also: reviewer found `--color-primary-dark-1` is consumed as **text color** in `web_src/css/repo/header.css:50,63` and `web_src/css/modules/label.css:141`, not just as background. So `dark-1` must also pass AA on cream paper.

**Final scale:**

| Token | New value | Used as | Contrast on `#efece1` | WCAG |
|---|---|---|---|---|
| `--color-primary` | `#BBFD3B` | Button bg, current-tab fill | with `#1a1a1a` text: 13.8:1 | ✓ AAA |
| `--color-primary-contrast` | `#1a1a1a` | Text on primary surfaces | (paired above) | ✓ AAA |
| **`--color-primary-hover`** *(new override)* | `#93C82B` | Button hover bg | with `#1a1a1a` text: 8.45:1 | ✓ AAA |
| **`--color-primary-active`** *(new override)* | `#7AAB1F` | Button active bg | with `#1a1a1a` text: 6.20:1 | ✓ AA |
| `--color-primary-dark-1` | `#4D7008` | Repo header counts, primary label text | as text on body: **4.92:1** | ✓ AA |
| `--color-primary-dark-2` | `#3D5A09` | **Link rest** | as text on body: **6.79:1** | ✓ AA |
| `--color-primary-dark-3` | `#2C4D08` | Link hover, button-hover-deeper | as text on body: **8.22:1** | ✓ AAA |
| `--color-primary-dark-4` | `#1A2D04` | Visited link | as text on body: **12.5:1** | ✓ AAA |

The dark-1 → dark-4 progression is now a smooth darkening (all four work as text on cream), with the action color (`primary` + `primary-hover` + `primary-active`) staying as bright fluorescents on a separate axis — they're never used as text-on-paper, only as paint on action buttons.

### 3.3 Primary-alpha tokens (visible drift fix)

Bundled `--color-primary-alpha-20` and `-alpha-30` use the literal hex `#18805C` (bundled forest teal), and they drive `--color-reaction-hover-bg` / `--color-reaction-active-bg`. After our other changes, reactions would tint slightly teal while everything else is yellow-green ink — visible mismatch.

Override only the two visible ones:

```css
--color-primary-alpha-20: rgba(61, 90, 9, 0.20);   /* matches new dark-2 */
--color-primary-alpha-30: rgba(61, 90, 9, 0.30);
```

Skip alpha-10/40/50/60+ — they're either invisible at low opacity or unused in user-facing surfaces.

### 3.4 Focus ring color

Move from `--color-primary-dark-3` (same as hover) to a non-green hue: teal `#0e7b75`.

Why teal: it's already in our colorize palette as `--color-cyan` (light theme defines `--color-cyan: #0d8b86`). We use a moderate darkening (`#0e7b75`) — reviewer flagged that pure-dark teal `#0a6b67` might read as "yet another dark color" next to the new dark-3 forest green. The mid-tone teal `#0e7b75` keeps chromatic distinction (cyan ~180° vs yellow-green ~80° hue separation) while still passing 3:1 non-text minimum.

Contrast against our 5 surfaces:
- on `#efece1` body: 4.34:1
- on `#e3decd` header: 3.85:1
- on `#e7e3d4` nav: 4.07:1
- on `#ddd8c5` highlight: 3.54:1
- on `#f6f3e9` card: 4.69:1

All pass 3:1 non-text minimum. Keyboard users now see: link rest = forest ink → hover = darker forest ink → focus = visibly distinct teal outline. Three distinct states, no collision.

### 3.5 Backdrop-filter fallback

```css
/* Before */
@supports not (...) {
    .full.height > .ui.menu, .ui.modal, .ui.modals.dimmer {
        background-color: rgba(251, 250, 247, 0.97) !important;
    }
}

/* After — matches new --color-body */
@supports not (...) {
    .full.height > .ui.menu, .ui.modal, .ui.modals.dimmer {
        background-color: rgba(239, 236, 225, 0.97) !important;
    }
}
```

### 3.6 `--color-text-light` tightening

Currently `#595959`. On `--color-box-body-highlight` (`#ddd8c5`), contrast is 4.84:1 — passes AA but with only 0.34 margin. Tighten to `#4a4a4a`:

- on `#efece1` body: 7.39:1 (was 5.95:1)
- on `#e7e3d4` nav: 6.66:1 (was 5.32:1)
- on `#ddd8c5` highlight: 5.84:1 (was 4.84:1) — solid AA margin
- on `#e3decd` header: 6.36:1
- on `#f6f3e9` card: 7.83:1

All pass AA with healthy margin; some pass AAA.

## 4. File-by-file changes

### `custom/public/assets/css/theme-hackforger-light.css`

Find the `:root { ... --color-primary: #BBFD3B; ...}` block (lines ~31-39):
- Update dark-1/2/3/4 to new values per §3.2.
- ADD new lines: `--color-primary-hover: #93C82B;` and `--color-primary-active: #7AAB1F;` (uncouple button hover from link ink scale).
- ADD: `--color-primary-alpha-20: rgba(61, 90, 9, 0.20);` and `--color-primary-alpha-30: rgba(61, 90, 9, 0.30);` per §3.3.

Find the `:root` block containing `--color-text-light: #595959;` (line ~44) and change to `#4a4a4a`. Update the inline contrast comment.

Find the focus-visible rule (line ~115 area) and change `outline: 2px solid var(--color-primary-dark-3)` to `outline: 2px solid #0e7b75`. Update the rationale comment per §3.4.

Find the `@supports not (... backdrop-filter ...)` block (line ~98 area) and change `rgba(251, 250, 247, 0.97)` → `rgba(239, 236, 225, 0.97)`.

### `custom/public/assets/css/theme-hackforger-dark.css`

**No changes.** Audit confirmed dark theme inherits working values.

## 5. Testing strategy

Manual E2E only. After merge → restart `gitea web` → hard-refresh browser:

### 5.1 Light theme (the bug under test)
1. Navigate to `/explore/repos` (the user-reported page). Repo names and owner names should be readable dark-forest-green ink. Hover any link — visibly darkens.
2. Navigate to `/explore/issues`, `/explore/users`, `/explore/organizations`. Same readability.
3. Navigate to a repo's `/issues` list — issue titles readable.
4. Navigate to a repo's file tree — file/dir names readable.
5. Navigate to commit log — commit message links readable.
6. Tab through a form (issue create) — focus ring is teal, visibly distinct from hover.
7. Click a primary button — fluorescent fill preserved (brand identity intact).
8. Open a modal on a browser without backdrop-filter (Safari < 9, hard to test) — modal bg matches body color (no white seam).

### 5.2 Dark theme regression check
1. Switch to dark theme. Repo names, issue titles, etc. — should look identical to before this PR. Mint-green scale untouched.
2. Tab through form — focus ring still uses dark theme's color (which is `--color-primary-dark-1` per PR #87, not the new teal). **Important: the teal focus override is light-theme-scoped only.**

### 5.3 Cross-browser smoke
- Chrome: link colors render as expected.
- Safari: same.
- Firefox: same.

## 6. Risks & mitigations

| Risk | Mitigation |
|---|---|
| `--color-primary-dark-1: #4D7008` used somewhere I haven't enumerated as text on cream — could fail | Reviewer pass identified `repo/header.css:50,63` and `label.css:141`. New dark-1 at 4.92:1 passes AA. Implementation plan adds a grep step to enumerate any other text consumers before merge. |
| `--color-primary-alpha-10/40+` still drift to bundled forest teal — minor visual mismatch | Spec only fixes the visible alpha-20/30 (reaction backgrounds). Other alphas at 10% / 40%+ opacity are functionally invisible. Accepted. |
| Teal focus ring breaks "green brand" consistency | Focus rings are functional UI not brand identity. Cyan is in DESIGN.md §2 secondary palette. If user objects, easy revert to `--color-primary-dark-4` (#1A2D04 = near-black green, 12.5:1 — also distinct from hover but stays in green family). |
| `--color-text-light: #4a4a4a` darker than expected for "muted text" semantics | Subjective; if too dark, intermediate `#535353` (6.5:1 on body) is a fallback. |
| Button hover/active going from fluorescent → bright-yellow-green-darker still feels "different" | This is the intentional fix. With explicit `--color-primary-hover: #93C82B` (one notch darker fluorescent) the transition stays in the same yellow-green family — user sees a darken, not a hue jump. |

## 7. Out of scope (future work)

- **CI contrast checker script.** Audit short-term recommendation. Would prevent this exact regression.
- **`--color-secondary` rename to `--hf-surface-secondary`** (MED-1).
- **Dark theme adopting DESIGN.md fluorescent as primary** (MED-2). Currently inherits mint.
- **Visited link hue change** to non-green family (HIGH-1 from audit). With the new scale, visited at 12.5:1 already differentiates from rest at 5.11:1, so deferred.
