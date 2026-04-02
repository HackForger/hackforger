# Theme Refinement Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create 3 evaluation theme variants (Flat, Whisper, Mist) with unified backgrounds and soft borders, plus update footer branding to HackForger.

**Architecture:** CSS-only theme changes via Forgejo's custom property system. Each variant is a self-contained theme CSS file (full copy of base with overrides). Registration in Go + locale files. One template edit for footer.

**Tech Stack:** CSS custom properties, Go (settings registration), Go templates, INI locale files

**Spec:** `docs/superpowers/specs/2026-04-02-theme-refinement-design.md`

---

## Chunk 1: Footer Branding Change

### Task 1: Update footer template

**Files:**
- Modify: `templates/base/footer_content.tmpl:4`

- [ ] **Step 1: Edit footer template**

In `templates/base/footer_content.tmpl`, change line 4 from:
```html
			<a target="_blank" rel="noopener noreferrer" href="https://forgejo.org">{{ctx.Locale.Tr "powered_by" "Forgejo"}}</a>
```
To:
```html
			<a target="_blank" rel="noopener noreferrer" href="https://github.com/HackForger/hackforger">{{ctx.Locale.Tr "powered_by" "HackForger"}}</a>
```

Only change the `href` value and the string parameter `"Forgejo"` → `"HackForger"`. The `powered_by` i18n key uses `%s` format, so this works across all 50+ locales without any locale file changes.

- [ ] **Step 2: Commit**

```bash
git add templates/base/footer_content.tmpl
git commit -m "feat(ui): update footer branding from Forgejo to HackForger"
```

---

## Chunk 2: Light Theme Variants (3 files)

Each light variant is a full copy of `web_src/css/themes/theme-hackforger-light.css` (317 lines) with specific CSS variable overrides applied. The base file structure is:
- Lines 1-3: `@import` statements (chroma, codemirror, markup — light versions)
- Lines 5-283: `:root { ... }` block with all CSS variables
- Lines 284-317: Component-specific overrides (`.ui.secondary`, `::selection`, etc.)

### Task 2: Create Flat Light theme

**Files:**
- Create: `web_src/css/themes/theme-hackforger-flat-light.css`
- Reference: `web_src/css/themes/theme-hackforger-light.css`

- [ ] **Step 1: Copy base light theme**

```bash
cp web_src/css/themes/theme-hackforger-light.css web_src/css/themes/theme-hackforger-flat-light.css
```

- [ ] **Step 2: Apply Flat light overrides**

In `theme-hackforger-flat-light.css`, replace these CSS variable values inside the `:root` block. Use exact line references from the base file:

| Line | Variable | Old Value | New Value |
|---|---|---|---|
| 68 | `--color-secondary` | `#DDD9D3` | `#E6E3DE` |
| 217 | `--color-box-header` | `#F4F2EF` | `#F6F4F1` |
| 226 | `--color-footer` | `#F4F2EF` | `#FAFAF8` |
| 231 | `--color-input-border` | `#C8C0B4` | `#D0CCC5` |
| 233 | `--color-header-wrapper` | `#F4F2EF` | `#FAFAF8` |
| 227 | `--color-timeline` | `#C8C0B4` | `#D0CCC5` |
| 238 | `--color-hover` | `#e4e4e4aa` | `#EDEDEDA0` |
| 239 | `--color-active` | `#e2e2e5` | `#E8E8EB` |
| 240 | `--color-menu` | `#F4F2EF` | `#FAFAF8` |
| 242 | `--fancy-card-bg` | `#F4F2EF` | `#FAFAF8` |
| 250 | `--color-secondary-bg` | `#F4F2EF` | `#FAFAF8` |
| 264 | `--color-nav-bg` | `#F4F2EF` | `#FAFAF8` |
| 265 | `--color-nav-hover-bg` | `var(--zinc-300)` | `#F0EDE9` |

- [ ] **Step 3: Commit**

```bash
git add web_src/css/themes/theme-hackforger-flat-light.css
git commit -m "feat(ui): add HackForger Flat light theme variant"
```

### Task 3: Create Whisper Light theme

**Files:**
- Create: `web_src/css/themes/theme-hackforger-whisper-light.css`

- [ ] **Step 1: Copy base light theme**

```bash
cp web_src/css/themes/theme-hackforger-light.css web_src/css/themes/theme-hackforger-whisper-light.css
```

- [ ] **Step 2: Apply Whisper light overrides**

Same lines as Task 2, different values:

| Line | Variable | New Value |
|---|---|---|
| 68 | `--color-secondary` | `#E8E5E0` |
| 217 | `--color-box-header` | `#F7F5F2` |
| 226 | `--color-footer` | `#F7F5F2` |
| 231 | `--color-input-border` | `#D5D1CB` |
| 233 | `--color-header-wrapper` | `#F7F5F2` |
| 227 | `--color-timeline` | `#D5D1CB` |
| 238 | `--color-hover` | `#E8E8E8A8` |
| 239 | `--color-active` | `#E5E5E8` |
| 240 | `--color-menu` | `#F7F5F2` |
| 242 | `--fancy-card-bg` | `#F7F5F2` |
| 250 | `--color-secondary-bg` | `#F7F5F2` |
| 264 | `--color-nav-bg` | `#F7F5F2` |
| 265 | `--color-nav-hover-bg` | `#EDEAE6` |

- [ ] **Step 3: Commit**

```bash
git add web_src/css/themes/theme-hackforger-whisper-light.css
git commit -m "feat(ui): add HackForger Whisper light theme variant"
```

### Task 4: Create Mist Light theme

**Files:**
- Create: `web_src/css/themes/theme-hackforger-mist-light.css`

- [ ] **Step 1: Copy base light theme**

```bash
cp web_src/css/themes/theme-hackforger-light.css web_src/css/themes/theme-hackforger-mist-light.css
```

- [ ] **Step 2: Apply Mist light overrides**

| Line | Variable | New Value |
|---|---|---|
| 68 | `--color-secondary` | `#E5E2DD` |
| 217 | `--color-box-header` | `#F6F4F1` |
| 226 | `--color-footer` | `#FAFAF8` |
| 231 | `--color-input-border` | `#D8D4CE` |
| 233 | `--color-header-wrapper` | `#FAFAF8` |
| 227 | `--color-timeline` | `#D8D4CE` |
| 238 | `--color-hover` | `#EAEAEAAA` |
| 239 | `--color-active` | `#E6E6E9` |
| 240 | `--color-menu` | `#FAFAF8` |
| 242 | `--fancy-card-bg` | `#F6F4F1` |
| 250 | `--color-secondary-bg` | `#FAFAF8` |
| 264 | `--color-nav-bg` | `#FAFAF8` |
| 265 | `--color-nav-hover-bg` | `#F0EDE9` |

- [ ] **Step 3: Commit**

```bash
git add web_src/css/themes/theme-hackforger-mist-light.css
git commit -m "feat(ui): add HackForger Mist light theme variant"
```

---

## Chunk 3: Dark Theme Variants (3 files)

Each dark variant is a full copy of `web_src/css/themes/theme-hackforger-dark.css` (369 lines). The base file structure is:
- Lines 1-3: `@import` statements (chroma, codemirror, markup — dark versions)
- Line 4: `@import "../hackforger-colors-dark.css"` — **MUST be preserved in all dark variants**
- Lines 6-286: `:root { ... }` block with `--is-dark-theme: true` and all CSS variables
- Lines 287-369: Component-specific overrides (emoji invert, `.ui.secondary`, `::selection`, etc.)

### Task 5: Create Flat Dark theme

**Files:**
- Create: `web_src/css/themes/theme-hackforger-flat-dark.css`
- Reference: `web_src/css/themes/theme-hackforger-dark.css`

- [ ] **Step 1: Copy base dark theme**

```bash
cp web_src/css/themes/theme-hackforger-dark.css web_src/css/themes/theme-hackforger-flat-dark.css
```

- [ ] **Step 2: Apply Flat dark overrides**

In `theme-hackforger-flat-dark.css`, replace these CSS variable values inside the `:root` block:

| Line | Variable | Old Value | New Value |
|---|---|---|---|
| 52 | `--color-secondary` | `#2E3545` | `#222838` |
| 220 | `--color-box-header` | `#252A33` | `#1E2228` |
| 229 | `--color-footer` | `#141618` | `#141618` (unchanged) |
| 234 | `--color-input-border` | `#2E3545` | `#283040` |
| 236 | `--color-header-wrapper` | `#141618` | `#141618` (unchanged) |
| 230 | `--color-timeline` | `#2E3545` | `#283040` |
| 241 | `--color-hover` | `#252A33` | `#1E2430` |
| 242 | `--color-active` | `#2E3545` | `#252C38` |
| 243 | `--color-menu` | `#1E2228` | `#141618` |
| 245 | `--fancy-card-bg` | `#252A33` | `#1A1E22` |
| 253 | `--color-secondary-bg` | `#1E2228` | `#141618` |
| 267 | `--color-nav-bg` | `#141618` | `#141618` (unchanged) |
| 268 | `--color-nav-hover-bg` | `var(--steel-600)` | `var(--steel-650)` |

- [ ] **Step 3: Commit**

```bash
git add web_src/css/themes/theme-hackforger-flat-dark.css
git commit -m "feat(ui): add HackForger Flat dark theme variant"
```

### Task 6: Create Whisper Dark theme

**Files:**
- Create: `web_src/css/themes/theme-hackforger-whisper-dark.css`

- [ ] **Step 1: Copy base dark theme**

```bash
cp web_src/css/themes/theme-hackforger-dark.css web_src/css/themes/theme-hackforger-whisper-dark.css
```

- [ ] **Step 2: Apply Whisper dark overrides**

| Line | Variable | New Value |
|---|---|---|
| 52 | `--color-secondary` | `#242A36` |
| 220 | `--color-box-header` | `#202630` |
| 229 | `--color-footer` | `#161A1C` |
| 234 | `--color-input-border` | `#2A3242` |
| 236 | `--color-header-wrapper` | `#161A1C` |
| 230 | `--color-timeline` | `#2A3242` |
| 241 | `--color-hover` | `#202630` |
| 242 | `--color-active` | `#283040` |
| 243 | `--color-menu` | `#181C20` |
| 245 | `--fancy-card-bg` | `#1C2026` |
| 253 | `--color-secondary-bg` | `#181C20` |
| 267 | `--color-nav-bg` | `#161A1C` |
| 268 | `--color-nav-hover-bg` | `var(--steel-600)` (unchanged) |

- [ ] **Step 3: Commit**

```bash
git add web_src/css/themes/theme-hackforger-whisper-dark.css
git commit -m "feat(ui): add HackForger Whisper dark theme variant"
```

### Task 7: Create Mist Dark theme

**Files:**
- Create: `web_src/css/themes/theme-hackforger-mist-dark.css`

- [ ] **Step 1: Copy base dark theme**

```bash
cp web_src/css/themes/theme-hackforger-dark.css web_src/css/themes/theme-hackforger-mist-dark.css
```

- [ ] **Step 2: Apply Mist dark overrides**

| Line | Variable | New Value |
|---|---|---|
| 52 | `--color-secondary` | `#282E3A` |
| 220 | `--color-box-header` | `#1E2228` |
| 229 | `--color-footer` | `#141618` (unchanged) |
| 234 | `--color-input-border` | `#2E3545` (unchanged) |
| 236 | `--color-header-wrapper` | `#141618` (unchanged) |
| 230 | `--color-timeline` | `#2E3545` (unchanged) |
| 241 | `--color-hover` | `#222838` |
| 242 | `--color-active` | `#2A3240` |
| 243 | `--color-menu` | `#141618` |
| 245 | `--fancy-card-bg` | `#1A1E22` |
| 253 | `--color-secondary-bg` | `#141618` |
| 267 | `--color-nav-bg` | `#141618` (unchanged) |
| 268 | `--color-nav-hover-bg` | `var(--steel-650)` |

- [ ] **Step 3: Commit**

```bash
git add web_src/css/themes/theme-hackforger-mist-dark.css
git commit -m "feat(ui): add HackForger Mist dark theme variant"
```

---

## Chunk 4: Registration + Build

### Task 8: Register themes in Go settings

**Files:**
- Modify: `modules/setting/ui.go:85`

- [ ] **Step 1: Add 6 new themes to Themes array**

In `modules/setting/ui.go`, line 85, append to the end of the Themes slice (before the closing `}`):

Change:
```go
Themes: []string{`forgejo-auto`, `forgejo-light`, `forgejo-dark`, `gitea-auto`, `gitea-light`, `gitea-dark`, `forgejo-auto-deuteranopia-protanopia`, `forgejo-light-deuteranopia-protanopia`, `forgejo-dark-deuteranopia-protanopia`, `forgejo-auto-tritanopia`, `forgejo-light-tritanopia`, `forgejo-dark-tritanopia`, `hackforger-light`, `hackforger-dark`},
```

To:
```go
Themes: []string{`forgejo-auto`, `forgejo-light`, `forgejo-dark`, `gitea-auto`, `gitea-light`, `gitea-dark`, `forgejo-auto-deuteranopia-protanopia`, `forgejo-light-deuteranopia-protanopia`, `forgejo-dark-deuteranopia-protanopia`, `forgejo-auto-tritanopia`, `forgejo-light-tritanopia`, `forgejo-dark-tritanopia`, `hackforger-light`, `hackforger-dark`, `hackforger-flat-light`, `hackforger-flat-dark`, `hackforger-whisper-light`, `hackforger-whisper-dark`, `hackforger-mist-light`, `hackforger-mist-dark`},
```

- [ ] **Step 2: Commit**

```bash
git add modules/setting/ui.go
git commit -m "feat(ui): register 6 new theme variants in Go settings"
```

### Task 9: Add theme display names in locale files

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 1: Add English theme names**

In `options/locale/locale_en-US.ini`, add these lines after `powered_by = Powered by %s` (line 15) in the `[common]` section:

```ini
themes.names.hackforger-flat-light = HackForger Flat (Light)
themes.names.hackforger-flat-dark = HackForger Flat (Dark)
themes.names.hackforger-whisper-light = HackForger Whisper (Light)
themes.names.hackforger-whisper-dark = HackForger Whisper (Dark)
themes.names.hackforger-mist-light = HackForger Mist (Light)
themes.names.hackforger-mist-dark = HackForger Mist (Dark)
```

- [ ] **Step 2: Add Chinese theme names**

In `options/locale/locale_zh-CN.ini`, add these lines after `powered_by=由 %s 提供支持` (line 15) in the `[common]` section:

```ini
themes.names.hackforger-flat-light=HackForger 素白 (浅色)
themes.names.hackforger-flat-dark=HackForger 素白 (深色)
themes.names.hackforger-whisper-light=HackForger 轻语 (浅色)
themes.names.hackforger-whisper-dark=HackForger 轻语 (深色)
themes.names.hackforger-mist-light=HackForger 薄雾 (浅色)
themes.names.hackforger-mist-dark=HackForger 薄雾 (深色)
```

- [ ] **Step 3: Commit**

```bash
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "feat(i18n): add display names for theme variants (en-US, zh-CN)"
```

### Task 10: Build and verify

- [ ] **Step 1: Build frontend**

```bash
make frontend
```

Expected: completes without errors. The 6 new CSS files should be compiled into `public/assets/css/`.

- [ ] **Step 2: Verify theme CSS files are built**

```bash
ls public/assets/css/theme-hackforger-*
```

Expected: 8 files total (2 existing + 6 new):
- `theme-hackforger-light.css`
- `theme-hackforger-dark.css`
- `theme-hackforger-flat-light.css`
- `theme-hackforger-flat-dark.css`
- `theme-hackforger-whisper-light.css`
- `theme-hackforger-whisper-dark.css`
- `theme-hackforger-mist-light.css`
- `theme-hackforger-mist-dark.css`

- [ ] **Step 3: Build backend**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
```

Expected: completes without errors.

- [ ] **Step 4: Start server and verify theme picker**

```bash
# Copy app.ini from main repo if not already present
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini 2>/dev/null || true
# Remove stale lock if present
rm -f data/queues/common/LOCK
./gitea web &
```

Verify at http://localhost:3000:
1. Footer shows "Powered by HackForger" linking to GitHub repo
2. User Settings → Appearance → Theme dropdown shows all 6 new variants with display names
3. Switching to each theme applies without errors
4. Kill server when done: `kill %1`

---

## E2E Testing Prompt

After implementation, perform manual E2E testing via `agent-browser` at `http://localhost:3000`:

1. **Footer branding**: Navigate to any page, verify footer shows "Powered by HackForger" with correct link
2. **Theme switching**: Go to Settings → Appearance, switch to each of the 6 new themes, take screenshot of homepage for each
3. **Visual comparison**: For each theme, verify:
   - Navbar background matches/differs from body per variant spec
   - Border lines are visible but softer than base theme
   - Footer background matches navbar treatment
   - Hover states on navbar items provide visible feedback
   - Input field borders are visible on forms (e.g., login page, new issue)
4. **Dark mode**: Repeat checks for all 3 dark variants

Save screenshots to `docs/tests/e2e/theme-refinement/` for comparison.
