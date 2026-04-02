# Theme Refinement: Unified Background + Soft Borders

**Date**: 2026-04-02
**Status**: Draft
**Branch**: `feat-theme-improve`

## Goal

Reduce visual contrast between header/footer/body regions in HackForger's UI. Create three evaluation theme variants (Flat, Whisper, Mist) for both light and dark modes. Update footer branding from Forgejo to HackForger.

## Non-Goals

- Changing the primary jade color palette
- Modifying component CSS (navbar.css, home.css, etc.)
- Touching upstream Forgejo theme files
- Creating auto-detect variants (light/dark only)

## Design Philosophy

| Variant | Background Strategy | Border Strategy | Character |
|---|---|---|---|
| **Flat** | All regions = body color | Very faint, minimum perceptible | Maximum minimalism |
| **Whisper** | 1% tonal offset from body | Faint | Subtle spatial hint |
| **Mist** | All regions = body color | Softer than current but structured | Flat + structured |

## Light Theme Token Changes

All values are CSS custom properties in `:root`. Only changed variables are listed; all other variables are inherited from the base `theme-hackforger-light.css`.

| Variable | Current | Flat | Whisper | Mist |
|---|---|---|---|---|
| `--color-nav-bg` | `#F4F2EF` | `#FAFAF8` | `#F7F5F2` | `#FAFAF8` |
| `--color-footer` | `#F4F2EF` | `#FAFAF8` | `#F7F5F2` | `#FAFAF8` |
| `--color-header-wrapper` | `#F4F2EF` | `#FAFAF8` | `#F7F5F2` | `#FAFAF8` |
| `--color-secondary` | `#DDD9D3` | `#E6E3DE` | `#E8E5E0` | `#E5E2DD` |
| `--color-box-header` | `#F4F2EF` | `#F6F4F1` | `#F7F5F2` | `#F6F4F1` |
| `--color-menu` | `#F4F2EF` | `#FAFAF8` | `#F7F5F2` | `#FAFAF8` |
| `--color-secondary-bg` | `#F4F2EF` | `#FAFAF8` | `#F7F5F2` | `#FAFAF8` |
| `--fancy-card-bg` | `#F4F2EF` | `#FAFAF8` | `#F7F5F2` | `#F6F4F1` |
| `--color-nav-hover-bg` | `var(--zinc-300)` | `#F0EDE9` | `#EDEAE6` | `#F0EDE9` |
| `--color-hover` | `#e4e4e4aa` | `#EDEDEDA0` | `#E8E8E8A8` | `#EAEAEAAA` |
| `--color-active` | `#e2e2e5` | `#E8E8EB` | `#E5E5E8` | `#E6E6E9` |
| `--color-input-border` | `#C8C0B4` | `#D0CCC5` | `#D5D1CB` | `#D8D4CE` |
| `--color-timeline` | `#C8C0B4` | `#D0CCC5` | `#D5D1CB` | `#D8D4CE` |

## Dark Theme Token Changes

| Variable | Current | Flat | Whisper | Mist |
|---|---|---|---|---|
| `--color-nav-bg` | `#141618` | `#141618` | `#161A1C` | `#141618` |
| `--color-footer` | `#141618` | `#141618` | `#161A1C` | `#141618` |
| `--color-header-wrapper` | `#141618` | `#141618` | `#161A1C` | `#141618` |
| `--color-secondary` | `#2E3545` | `#222838` | `#242A36` | `#282E3A` |
| `--color-box-header` | `#252A33` | `#1E2228` | `#202630` | `#1E2228` |
| `--color-menu` | `#1E2228` | `#141618` | `#181C20` | `#141618` |
| `--color-secondary-bg` | `#1E2228` | `#141618` | `#181C20` | `#141618` |
| `--fancy-card-bg` | `#252A33` | `#1A1E22` | `#1C2026` | `#1A1E22` |
| `--color-nav-hover-bg` | `var(--steel-600)` | `var(--steel-650)` | `var(--steel-600)` | `var(--steel-650)` |
| `--color-hover` | `#252A33` | `#1E2430` | `#202630` | `#222838` |
| `--color-active` | `#2E3545` | `#252C38` | `#283040` | `#2A3240` |
| `--color-input-border` | `#2E3545` | `#283040` | `#2A3242` | `#2E3545` |
| `--color-timeline` | `#2E3545` | `#283040` | `#2A3242` | `#2E3545` |

## Footer Template Change

**File**: `templates/base/footer_content.tmpl`

Change line 4 from:
```html
<a target="_blank" rel="noopener noreferrer" href="https://forgejo.org">{{ctx.Locale.Tr "powered_by" "Forgejo"}}</a>
```
To:
```html
<a target="_blank" rel="noopener noreferrer" href="https://github.com/HackForger/hackforger">{{ctx.Locale.Tr "powered_by" "HackForger"}}</a>
```

This reuses the existing `powered_by` i18n key (format: `Powered by %s`) across all 50+ locales. No locale file changes needed.

## Theme Registration

### 1. CSS Files (6 new files)

Location: `web_src/css/themes/`

| File | Base |
|---|---|
| `theme-hackforger-flat-light.css` | Copy of `theme-hackforger-light.css` with token overrides |
| `theme-hackforger-flat-dark.css` | Copy of `theme-hackforger-dark.css` with token overrides |
| `theme-hackforger-whisper-light.css` | Copy of `theme-hackforger-light.css` with token overrides |
| `theme-hackforger-whisper-dark.css` | Copy of `theme-hackforger-dark.css` with token overrides |
| `theme-hackforger-mist-light.css` | Copy of `theme-hackforger-light.css` with token overrides |
| `theme-hackforger-mist-dark.css` | Copy of `theme-hackforger-dark.css` with token overrides |

Each file is a full copy of the base theme with the relevant variables overridden from the tables above. This follows Forgejo's pattern where each theme file is self-contained.

**Important**: Dark variant files must preserve the `@import "../hackforger-colors-dark.css"` import from the dark base theme (line 4 of `theme-hackforger-dark.css`). Light variants do not have this import. The `--color-body` variable is intentionally NOT overridden in any variant — all variants inherit the base body color.

### 2. Go Registration

**File**: `modules/setting/ui.go:85`

Append to the Themes array:
```go
`hackforger-flat-light`, `hackforger-flat-dark`,
`hackforger-whisper-light`, `hackforger-whisper-dark`,
`hackforger-mist-light`, `hackforger-mist-dark`,
```

### 3. Locale Display Names

**Files**: `options/locale/locale_en-US.ini`, `options/locale/locale_zh-CN.ini`

Add as top-level keys (no INI section header). These are resolved dynamically by `routers/web/user/setting/profile.go:322` via `"themes.names." + themeName`. If the key doesn't exist, the raw theme name is shown as fallback. No existing `themes.names.*` keys exist in any locale file currently.

```ini
; English (locale_en-US.ini) — top-level, no section header
themes.names.hackforger-flat-light = HackForger Flat (Light)
themes.names.hackforger-flat-dark = HackForger Flat (Dark)
themes.names.hackforger-whisper-light = HackForger Whisper (Light)
themes.names.hackforger-whisper-dark = HackForger Whisper (Dark)
themes.names.hackforger-mist-light = HackForger Mist (Light)
themes.names.hackforger-mist-dark = HackForger Mist (Dark)

; Chinese
themes.names.hackforger-flat-light = HackForger 素白 (浅色)
themes.names.hackforger-flat-dark = HackForger 素白 (深色)
themes.names.hackforger-whisper-light = HackForger 轻语 (浅色)
themes.names.hackforger-whisper-dark = HackForger 轻语 (深色)
themes.names.hackforger-mist-light = HackForger 薄雾 (浅色)
themes.names.hackforger-mist-dark = HackForger 薄雾 (深色)
```

## Evaluation Plan

These themes are evaluation-only. After visual comparison on the live instance:
1. User selects the winning variant
2. Winning variant's values replace `theme-hackforger-light.css` / `theme-hackforger-dark.css`
3. Evaluation themes and their registrations are removed
4. Theme picker returns to normal size

## Accessibility Notes

- Flat light border contrast: ~1.07:1 (minimum perceptible for structural dividers)
- Flat light input border: `#D0CCC5` kept stronger than general borders
- Flat dark border contrast: ~1.12:1 (compensates for dark mode gamma)
- Whisper and Mist variants maintain higher contrast than Flat
- Existing WCAG 1.4.11 non-compliance on input borders is inherited from upstream; not addressed in this change

## Files Changed

| File | Change Type |
|---|---|
| `web_src/css/themes/theme-hackforger-flat-light.css` | New |
| `web_src/css/themes/theme-hackforger-flat-dark.css` | New |
| `web_src/css/themes/theme-hackforger-whisper-light.css` | New |
| `web_src/css/themes/theme-hackforger-whisper-dark.css` | New |
| `web_src/css/themes/theme-hackforger-mist-light.css` | New |
| `web_src/css/themes/theme-hackforger-mist-dark.css` | New |
| `modules/setting/ui.go` | Modified (Themes array) |
| `options/locale/locale_en-US.ini` | Modified (theme display names) |
| `options/locale/locale_zh-CN.ini` | Modified (theme display names) |
| `templates/base/footer_content.tmpl` | Modified (branding) |
