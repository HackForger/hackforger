# Logo replacement + DESIGN.md theme — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the platform logo with Cynthialime's brand SVGs, generate `hackforger-light` / `hackforger-dark` themes from DESIGN.md ("High-Energy Kineticism"), set `hackforger-dark` as default. All changes land in `custom/`, zero modifications to main codebase.

**Architecture:** Three artifact families live under `custom/public/assets/`: SVG images (`img/`), CSS theme files that override the bundled `theme-hackforger-*.css` via Forgejo's static-asset precedence (`css/`), and locally bundled woff2 web fonts (`fonts/`). Default theme is set via `custom/conf/app.ini`. Verification is browser-based — no traditional unit tests since CSS theming has no compile step in Forgejo's stack.

**Tech Stack:** SVG (with inline `<style>` for `prefers-color-scheme` adaptation), CSS3 (custom properties + Fomantic UI selector overrides), woff2 fonts (Poppins display, Manrope body — OFL 1.1 from google-webfonts-helper), Forgejo `setting.UI.DefaultTheme`.

**Spec:** `docs/superpowers/specs/2026-04-21-logo-theme-replacement-design.md`

**Branch:** `feat/issue-70-logo-theme` (worktree at `~/.config/superpowers/worktrees/hackforger/issue70-logo-theme`). Base: `v0.1-dev/hackforger`.

---

## Spec coverage map

| Spec section | Tasks |
|--------------|-------|
| Logo strategy (single self-adapting SVG) | 1 |
| Favicon | 2 |
| Font bundling | 3 |
| theme-hackforger-dark.css | 4 |
| theme-hackforger-light.css | 5 |
| Default theme via app.ini | 6 |
| Verification (E2E with agent-browser) | 7 |

---

## Pre-flight

- [ ] **Step 0a: Confirm worktree + clean working tree**

```bash
cd ~/.config/superpowers/worktrees/hackforger/issue70-logo-theme
git status   # expect clean tree on feat/issue-70-logo-theme
git log --oneline -2   # spec commit on top of latest default
```

- [ ] **Step 0b: Stage source SVGs from Issue #70**

The two original SVGs were downloaded earlier to `/tmp/issue70/logo-1.bin` (orange-bg) and `/tmp/issue70/logo-2.bin` (white-bg). Copy them into a working scratch dir:

```bash
mkdir -p /tmp/logo-build
cp /tmp/issue70/logo-1.bin /tmp/logo-build/logo-orange.svg
cp /tmp/issue70/logo-2.bin /tmp/logo-build/logo-white.svg
ls -la /tmp/logo-build/   # both ~20KB SVGs
```

If `/tmp/issue70/*.bin` was cleaned up, redownload:
```bash
mkdir -p /tmp/issue70
curl -sL "https://github.com/user-attachments/assets/5dc7aece-880a-453c-8342-b8b39f68d0e4" -o /tmp/issue70/logo-1.bin
curl -sL "https://github.com/user-attachments/assets/2ceebd09-685e-45c0-b16c-c836a8f5c43f" -o /tmp/issue70/logo-2.bin
file /tmp/issue70/logo-{1,2}.bin   # confirm both say "SVG Scalable Vector Graphics image"
```

---

## Task 1: Combined self-adapting logo SVG

**Files:**
- Create/replace: `custom/public/assets/img/logo.svg`

The two source SVGs differ only in the leftmost rounded-rect's `fill` attribute (`#fff` for white-bg, `#FAAD14` for orange-bg). Combine them into one file by extracting the white-bg version, replacing the static `fill` with a class hook, and adding an inline `<style>` block that flips the class's fill on `prefers-color-scheme: dark`.

- [ ] **Step 1.1: Read the white-bg SVG header to confirm the rect line position**

```bash
head -3 /tmp/logo-build/logo-white.svg
```
Expected: line 1 is `<svg ...>`, line 2 is `<rect width="85" height="50" rx="25" fill="white"/>`, line 3+ is the path payload.

- [ ] **Step 1.2: Generate the combined SVG**

```bash
# Take the white-bg source, replace the rect line with a classed version,
# and inject a <style> block right after the opening <svg> tag.
awk '
  NR == 1 {
    print
    print "<style>.icon-bg{fill:#fff}@media (prefers-color-scheme:dark){.icon-bg{fill:#FAAD14}}</style>"
    next
  }
  /^<rect width="85" height="50" rx="25" fill="white"\/>$/ {
    print "<rect class=\"icon-bg\" width=\"85\" height=\"50\" rx=\"25\"/>"
    next
  }
  { print }
' /tmp/logo-build/logo-white.svg > custom/public/assets/img/logo.svg

head -4 custom/public/assets/img/logo.svg
```

Expected output of `head -4`:
```
<svg width="365" height="51" viewBox="0 0 365 51" fill="none" xmlns="http://www.w3.org/2000/svg">
<style>.icon-bg{fill:#fff}@media (prefers-color-scheme:dark){.icon-bg{fill:#FAAD14}}</style>
<rect class="icon-bg" width="85" height="50" rx="25"/>
<path d="M51.98 9.06071H51.9742C48.5083 9.25531 44.7406 11.3114 41.6224 14.8094L41.3572 15.1039L41.1521 15.3515C40.1037 16.6067 39.1801 17.9692 38.3935 19.4169L38.3246 19.5429L38.0067 20.1597C37.8239 20.5152 37.65 20.8773 37.4866 21.2467L37.4852 21.2511C37.0106 22.308 36.607 23.3985 36.278 24.5137L36.1769 24.8521L36.0788 25.2301C34.9889 29.376 35.0489 33.4827 36.2648 36.7422C37.2865 39.4809 39.0262 41.404 41.2576 42.3737C44.2208 43.6636 47.7653 43.1275 51.145 41.0596C54.5201 38.9944 57.6565 35.4378 59.7065 30.8675C62.1191 25.4898 62.5349 19.6928 60.9269 15.372C59.9057 12.6355 58.1682 10.7144 55.9385 9.74341C54.685 9.20698 53.3315 8.97379 51.98 9.06071Z" fill="white"/>
```

- [ ] **Step 1.3: Visually verify in a browser**

```bash
open custom/public/assets/img/logo.svg
```
Open in light mode (system default): icon background should be **white**.
Toggle macOS to dark mode (System Settings → Appearance → Dark): the icon background should switch to **orange (#FAAD14)** without reloading the file.

If switching is lazy, `cmd+shift+r` reload the browser tab — should adapt immediately.

If switching does not happen, the `<style>` block was stripped or the selector failed. Inspect via dev tools and adjust the class name (try `style="..."` inline form as fallback).

---

## Task 2: Favicon SVG

**Files:**
- Create/replace: `custom/public/assets/img/favicon.svg`

The favicon is the icon-only square crop (the leftmost 85×50 rounded rect with its inner paths) of the logo. Use the same self-adapting `<style>` trick.

- [ ] **Step 2.1: Inspect the original logo to find the icon's path range**

The icon consists of the rect (line 2 in source) plus the next ~3 path elements (the white "S" curve and the green pulse). The wordmark "HackForger" starts at roughly path 4+. Easiest is to view the SVG in a browser dev tools and use "Edit as HTML" to count the icon paths.

For this file, the icon comprises:
- `<rect ... rx="25"/>` (background)
- `<path ... fill="white"/>` (large white S-curve, starting `M51.98 9.06...`)
- `<path ... fill="#BBFD3B"/>` (green pulse element, smaller)

Everything after the green pulse path is the wordmark.

- [ ] **Step 2.2: Generate the favicon**

```bash
# Extract the SVG opening tag, the icon paths, then close.
# Cropped viewBox: 0 0 85 50 (the icon's natural bounds).
cat > custom/public/assets/img/favicon.svg <<'EOF'
<svg width="85" height="50" viewBox="0 0 85 50" fill="none" xmlns="http://www.w3.org/2000/svg">
<style>.icon-bg{fill:#fff}@media (prefers-color-scheme:dark){.icon-bg{fill:#FAAD14}}</style>
<rect class="icon-bg" width="85" height="50" rx="25"/>
EOF
# Append the two icon paths from the source (lines starting with M51.98 and the green path):
sed -n '/^<path d="M51.98/,/Z" fill="#BBFD3B"\/>$/p' /tmp/logo-build/logo-white.svg >> custom/public/assets/img/favicon.svg
echo '</svg>' >> custom/public/assets/img/favicon.svg

cat custom/public/assets/img/favicon.svg | head -10
```

Expected: a valid SVG with the rect + 2 paths + closing tag, ~2-3KB.

- [ ] **Step 2.3: Verify in browser**

```bash
open custom/public/assets/img/favicon.svg
```
Should show the icon (rounded rect with inner curves), no wordmark. Should adapt to OS color preference like the logo.

---

## Task 3: Bundle Poppins + Manrope fonts

**Files:**
- Create: `custom/public/assets/fonts/Poppins-{Regular,Medium,SemiBold,Bold}.woff2`
- Create: `custom/public/assets/fonts/Manrope-{Regular,Medium,SemiBold}.woff2`
- Create: `custom/public/assets/fonts/OFL.txt`

google-webfonts-helper (gwfh) provides a JSON API and zip downloads of any Google Font subset+weights. Use it to grab Latin-only woff2.

- [ ] **Step 3.1: Create fonts directory**

```bash
mkdir -p custom/public/assets/fonts
```

- [ ] **Step 3.2: Download Poppins (4 weights) via gwfh**

```bash
cd /tmp
curl -sL "https://gwfh.mranftl.com/api/fonts/poppins?download=zip&subsets=latin&variants=regular,500,600,700&formats=woff2" -o poppins.zip
unzip -o poppins.zip -d /tmp/poppins-extracted
ls /tmp/poppins-extracted/
```

Expected: 4 woff2 files like `poppins-v22-latin-regular.woff2`, `-500.woff2`, `-600.woff2`, `-700.woff2`.

- [ ] **Step 3.3: Rename and move Poppins to the project**

```bash
TARGET=~/.config/superpowers/worktrees/hackforger/issue70-logo-theme/custom/public/assets/fonts
cp /tmp/poppins-extracted/poppins-v*-latin-regular.woff2 $TARGET/Poppins-Regular.woff2
cp /tmp/poppins-extracted/poppins-v*-latin-500.woff2     $TARGET/Poppins-Medium.woff2
cp /tmp/poppins-extracted/poppins-v*-latin-600.woff2     $TARGET/Poppins-SemiBold.woff2
cp /tmp/poppins-extracted/poppins-v*-latin-700.woff2     $TARGET/Poppins-Bold.woff2
ls -la $TARGET/Poppins-*
```

Expected: 4 files, each 25–35 KB.

- [ ] **Step 3.4: Repeat for Manrope (3 weights)**

```bash
cd /tmp
curl -sL "https://gwfh.mranftl.com/api/fonts/manrope?download=zip&subsets=latin&variants=regular,500,600&formats=woff2" -o manrope.zip
unzip -o manrope.zip -d /tmp/manrope-extracted
TARGET=~/.config/superpowers/worktrees/hackforger/issue70-logo-theme/custom/public/assets/fonts
cp /tmp/manrope-extracted/manrope-v*-latin-regular.woff2 $TARGET/Manrope-Regular.woff2
cp /tmp/manrope-extracted/manrope-v*-latin-500.woff2     $TARGET/Manrope-Medium.woff2
cp /tmp/manrope-extracted/manrope-v*-latin-600.woff2     $TARGET/Manrope-SemiBold.woff2
ls -la $TARGET/Manrope-*
```

Expected: 3 more files.

- [ ] **Step 3.5: Add OFL license**

OFL 1.1 requires the license text to ship with redistributed fonts. Use the canonical text:

```bash
cat > custom/public/assets/fonts/OFL.txt <<'EOF'
Copyright 2020 The Poppins Project Authors (https://github.com/itfoundry/Poppins)
Copyright 2019 The Manrope Project Authors (https://github.com/sharanda/manrope)

This Font Software is licensed under the SIL Open Font License, Version 1.1.
This license is copied below, and is also available with a FAQ at:
http://scripts.sil.org/OFL

-----------------------------------------------------------
SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007
-----------------------------------------------------------

PREAMBLE
The goals of the Open Font License (OFL) are to stimulate worldwide
development of collaborative font projects, to support the font creation
efforts of academic and linguistic communities, and to provide a free and
open framework in which fonts may be shared and improved in partnership
with others.

The OFL allows the licensed fonts to be used, studied, modified and
redistributed freely as long as they are not sold by themselves. The
fonts, including any derivative works, can be bundled, embedded,
redistributed and/or sold with any software provided that any reserved
names are not used by derivative works. The fonts and derivatives,
however, cannot be released under any other type of license. The
requirement for fonts to remain under this license does not apply
to any document created using the fonts or their derivatives.

DEFINITIONS
"Font Software" refers to the set of files released by the Copyright
Holder(s) under this license and clearly marked as such. This may
include source files, build scripts and documentation.

"Reserved Font Name" refers to any names specified as such after the
copyright statement(s).

"Original Version" refers to the collection of Font Software components as
distributed by the Copyright Holder(s).

"Modified Version" refers to any derivative made by adding to, deleting,
or substituting -- in part or in whole -- any of the components of the
Original Version, by changing formats or by porting the Font Software to a
new environment.

"Author" refers to any designer, engineer, programmer, technical
writer or other person who contributed to the Font Software.

PERMISSION & CONDITIONS
Permission is hereby granted, free of charge, to any person obtaining
a copy of the Font Software, to use, study, copy, merge, embed, modify,
redistribute, and sell modified and unmodified copies of the Font
Software, subject to the following conditions:

1) Neither the Font Software nor any of its individual components,
in Original or Modified Versions, may be sold by itself.

2) Original or Modified Versions of the Font Software may be bundled,
redistributed and/or sold with any software, provided that each copy
contains the above copyright notice and this license. These can be
included either as stand-alone text files, human-readable headers or
in the appropriate machine-readable metadata fields within text or
binary files as long as those fields can be easily viewed by the user.

3) No Modified Version of the Font Software may use the Reserved Font
Name(s) unless explicit written permission is granted by the corresponding
Copyright Holder. This restriction only applies to the primary font name
as presented to the users.

4) The name(s) of the Copyright Holder(s) or the Author(s) of the Font
Software shall not be used to promote, endorse or advertise any
Modified Version, except to acknowledge the contribution(s) of the
Copyright Holder(s) and the Author(s) or with their explicit written
permission.

5) The Font Software, modified or unmodified, in part or in whole,
must be distributed entirely under this license, and must not be
distributed under any other license. The requirement for fonts to
remain under this license does not apply to any document created
using the Font Software.

TERMINATION
This license becomes null and void if any of the above conditions are
not met.

DISCLAIMER
THE FONT SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO ANY WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT
OF COPYRIGHT, PATENT, TRADEMARK, OR OTHER RIGHT. IN NO EVENT SHALL THE
COPYRIGHT HOLDER BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
INCLUDING ANY GENERAL, SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL
DAMAGES, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF THE USE OR INABILITY TO USE THE FONT SOFTWARE OR FROM
OTHER DEALINGS IN THE FONT SOFTWARE.
EOF
```

---

## Task 4: theme-hackforger-dark.css (default theme)

**Files:**
- Create: `custom/public/assets/css/theme-hackforger-dark.css`

- [ ] **Step 4.1: Create the css directory**

```bash
mkdir -p custom/public/assets/css
```

- [ ] **Step 4.2: Write the dark theme**

```bash
cat > custom/public/assets/css/theme-hackforger-dark.css <<'EOF'
/* HackForger Dark Theme — based on DESIGN.md "High-Energy Kineticism / Void and Vibe"
   Overrides the bundled web_src/css/themes/theme-hackforger-dark.css at runtime
   via Forgejo's custom/public asset precedence. */

/* === Local fonts (no Google CDN call from internal instance) === */

@font-face {
  font-family: 'Poppins'; font-display: swap; font-weight: 400; font-style: normal;
  src: url('../fonts/Poppins-Regular.woff2') format('woff2');
}
@font-face {
  font-family: 'Poppins'; font-display: swap; font-weight: 500; font-style: normal;
  src: url('../fonts/Poppins-Medium.woff2') format('woff2');
}
@font-face {
  font-family: 'Poppins'; font-display: swap; font-weight: 600; font-style: normal;
  src: url('../fonts/Poppins-SemiBold.woff2') format('woff2');
}
@font-face {
  font-family: 'Poppins'; font-display: swap; font-weight: 700; font-style: normal;
  src: url('../fonts/Poppins-Bold.woff2') format('woff2');
}
@font-face {
  font-family: 'Manrope'; font-display: swap; font-weight: 400; font-style: normal;
  src: url('../fonts/Manrope-Regular.woff2') format('woff2');
}
@font-face {
  font-family: 'Manrope'; font-display: swap; font-weight: 500; font-style: normal;
  src: url('../fonts/Manrope-Medium.woff2') format('woff2');
}
@font-face {
  font-family: 'Manrope'; font-display: swap; font-weight: 600; font-style: normal;
  src: url('../fonts/Manrope-SemiBold.woff2') format('woff2');
}

/* === Design tokens (CSS variables) === */

:root {
  /* DESIGN.md §2 — Core palette */
  --color-primary: #BBFD3B;          /* Fluorescent green — primary CTA, success */
  --color-primary-dark-1: #A6E532;
  --color-primary-dark-2: #91CD2A;
  --color-primary-light-1: #C8FE5C;
  --color-primary-light-2: #D5FE7E;
  --color-primary-alpha-30: rgba(187, 253, 59, 0.30);
  --color-primary-alpha-50: rgba(187, 253, 59, 0.50);

  --color-secondary: #37f5ef;        /* Cyan — interactive secondary */
  --color-tertiary: #ff8db4;         /* Pink — community / hot streaks */
  --color-error: #ff7351;
  --color-error-bg: rgba(255, 115, 81, 0.15);

  /* DESIGN.md §2.4 — Surface hierarchy (Tonal Layering) */
  --color-body: #0e0e0e;
  --color-box-body: #131313;
  --color-box-body-highlight: #262626;
  --color-box-header: #191919;
  --color-secondary-bg: #191919;
  --color-menu: #191919;

  /* DESIGN.md §6 — body text NOT pure white */
  --color-text: #ababab;
  --color-text-light: #757575;
  --color-text-light-2: #525252;
  --color-text-light-3: #3d3d3d;
  --color-text-dark: #d4d4d8;        /* For high-contrast headlines */

  /* DESIGN.md §2.3 — "No-Line" rule: Ghost Border @ 15% opacity */
  --color-border: rgba(72, 72, 72, 0.15);
  --color-light-border: rgba(72, 72, 72, 0.10);

  /* Radius scale (DESIGN.md §5) */
  --border-radius: 4px;
  --border-radius-medium: 8px;       /* Cards */
  --border-radius-large: 20px;       /* Modals */
  --border-radius-full: 9999px;      /* Pills */

  /* Typography (DESIGN.md §3) */
  --fonts-regular: 'Manrope', 'OPPOSans', 'PingFang SC', system-ui, sans-serif;
  --fonts-display: 'Poppins', 'Epilogue', 'PingFang SC', system-ui, sans-serif;
  --fonts-monospace: ui-monospace, 'Cascadia Code', 'Source Code Pro', monospace;
}

/* === Body & headlines === */

body {
  background: var(--color-body);
  color: var(--color-text);
  font-family: var(--fonts-regular);
  line-height: 1.6;
}

h1, h2, h3, .ui.header, .display-lg, .display-md {
  font-family: var(--fonts-display);
  letter-spacing: -0.02em;
  color: var(--color-text-dark);
}

/* DESIGN.md §3 — labels uppercase + spaced */
.ui.label, .label-md {
  font-family: var(--fonts-regular);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 0.75rem;
  font-weight: 500;
}

/* === Buttons (DESIGN.md §5) — pill, no shadow === */

.ui.button {
  border-radius: var(--border-radius-full);
  font-family: var(--fonts-regular);
  font-weight: 500;
  letter-spacing: 0.02em;
  box-shadow: none;
  transition: background 0.15s ease, box-shadow 0.2s ease;
}
.ui.primary.button {
  background: var(--color-primary);
  color: #0e0e0e;
}
.ui.primary.button:hover, .ui.primary.button:focus {
  background: var(--color-primary-light-1);
  /* DESIGN.md §4 — neon glow */
  box-shadow: 0 0 24px var(--color-primary-alpha-30);
}
.ui.secondary.button {
  background: transparent;
  border: 1px solid rgba(72, 72, 72, 0.20);
  color: var(--color-text);
}
.ui.secondary.button:hover {
  background: var(--color-box-body-highlight);
}

/* Tertiary / ghost button — text only, primary underline on hover */
.ui.basic.button, .ui.tertiary.button {
  background: transparent;
  box-shadow: none;
  color: var(--color-text);
}
.ui.basic.button:hover, .ui.tertiary.button:hover {
  background: transparent !important;
  border-bottom: 2px solid var(--color-primary);
  color: var(--color-text-dark);
}

/* === Cards (DESIGN.md §5) — 8px radius, no border, hover lift === */

.ui.card, .ui.cards > .card {
  border-radius: var(--border-radius-medium);
  border: none !important;
  background: var(--color-box-header);
  box-shadow: none;
  transition: background 0.15s ease, transform 0.15s ease;
}
.ui.card:hover, .ui.cards > .card:hover {
  background: var(--color-box-body-highlight);
  transform: translateY(-2px);
}

/* === Inputs (DESIGN.md §5) — bottom stroke on focus === */

.ui.input input,
.ui.form input[type=text],
.ui.form input[type=email],
.ui.form input[type=password],
.ui.form input[type=search],
.ui.form input[type=number],
.ui.form input[type=url],
.ui.form textarea,
.ui.dropdown {
  background: var(--color-box-body-highlight);
  border: none;
  border-bottom: 2px solid transparent;
  border-radius: var(--border-radius) var(--border-radius) 0 0;
  color: var(--color-text);
  transition: border-color 0.15s ease;
}
.ui.input input:focus,
.ui.form input[type=text]:focus,
.ui.form input[type=email]:focus,
.ui.form input[type=password]:focus,
.ui.form input[type=search]:focus,
.ui.form input[type=number]:focus,
.ui.form input[type=url]:focus,
.ui.form textarea:focus {
  background: var(--color-box-body-highlight);
  border-bottom-color: var(--color-primary);
  outline: none;
}

/* Error state — error color text + glow */
.ui.form .error.field input,
.ui.form .error.field textarea {
  border-bottom-color: var(--color-error);
  background: var(--color-error-bg);
  color: var(--color-error);
}

/* === Modals (DESIGN.md §5) — 20px radius, glassmorphism backdrop === */

.ui.modal {
  border-radius: var(--border-radius-large);
  background: var(--color-box-header);
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.50);
}
.ui.modal > .header {
  background: var(--color-box-header);
  border-bottom: none;
  border-top-left-radius: var(--border-radius-large);
  border-top-right-radius: var(--border-radius-large);
  font-family: var(--fonts-display);
  letter-spacing: -0.01em;
}
.ui.modal > .actions {
  background: var(--color-box-header);
  border-top: none;
  border-bottom-left-radius: var(--border-radius-large);
  border-bottom-right-radius: var(--border-radius-large);
}
/* Foggy void backdrop */
.ui.dimmer.modals, .ui.dimmer {
  background: rgba(14, 14, 14, 0.80);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
}

/* === Chips & tags (DESIGN.md §5) — pill, energy variants === */

.ui.tag.label, .ui.basic.label {
  border-radius: var(--border-radius-full);
}
.ui.label.green   { background: rgba(187, 253, 59, 0.15); color: var(--color-primary); }
.ui.label.cyan    { background: rgba(55, 245, 239, 0.15); color: var(--color-secondary); }
.ui.label.pink    { background: rgba(255, 141, 180, 0.15); color: var(--color-tertiary); }
.ui.label.orange  { background: rgba(255, 115, 81, 0.15);  color: var(--color-error); }

/* === Dividers (DESIGN.md §6 Don't) — kill 1px solid lines === */

.ui.divider {
  border: none;
  height: 12px;
  margin: 0;
}
.ui.horizontal.divider {
  height: auto;
  color: var(--color-text-light-2);
}

/* === Scrollbar — subtle on dark === */

::-webkit-scrollbar { width: 10px; height: 10px; }
::-webkit-scrollbar-track { background: var(--color-body); }
::-webkit-scrollbar-thumb { background: var(--color-box-body-highlight); border-radius: 5px; }
::-webkit-scrollbar-thumb:hover { background: var(--color-text-light-3); }

/* === Links === */

a, a.muted { color: var(--color-primary); }
a:hover, a.muted:hover { color: var(--color-primary-light-1); text-decoration: none; }

/* === Code blocks === */

code, pre, .CodeMirror {
  font-family: var(--fonts-monospace);
  background: var(--color-box-body);
}
EOF

wc -l custom/public/assets/css/theme-hackforger-dark.css
```

Expected: ~250 lines.

---

## Task 5: theme-hackforger-light.css

**Files:**
- Create: `custom/public/assets/css/theme-hackforger-light.css`

Same structure as dark, with the surface palette inverted. DESIGN.md doesn't explicitly cover light mode, so derive from the dark spec's principles: keep the brand fluorescent green, swap dark surfaces for light, swap light text for dark text — but maintain the "no pure black/white" rule (§6 Don't).

- [ ] **Step 5.1: Write the light theme**

```bash
cat > custom/public/assets/css/theme-hackforger-light.css <<'EOF'
/* HackForger Light Theme — DESIGN.md "High-Energy Kineticism" adapted for light surfaces.
   Brand fluorescent green retained; surface ladder inverted from dark variant. */

/* === Local fonts === */

@font-face {
  font-family: 'Poppins'; font-display: swap; font-weight: 400; font-style: normal;
  src: url('../fonts/Poppins-Regular.woff2') format('woff2');
}
@font-face {
  font-family: 'Poppins'; font-display: swap; font-weight: 500; font-style: normal;
  src: url('../fonts/Poppins-Medium.woff2') format('woff2');
}
@font-face {
  font-family: 'Poppins'; font-display: swap; font-weight: 600; font-style: normal;
  src: url('../fonts/Poppins-SemiBold.woff2') format('woff2');
}
@font-face {
  font-family: 'Poppins'; font-display: swap; font-weight: 700; font-style: normal;
  src: url('../fonts/Poppins-Bold.woff2') format('woff2');
}
@font-face {
  font-family: 'Manrope'; font-display: swap; font-weight: 400; font-style: normal;
  src: url('../fonts/Manrope-Regular.woff2') format('woff2');
}
@font-face {
  font-family: 'Manrope'; font-display: swap; font-weight: 500; font-style: normal;
  src: url('../fonts/Manrope-Medium.woff2') format('woff2');
}
@font-face {
  font-family: 'Manrope'; font-display: swap; font-weight: 600; font-style: normal;
  src: url('../fonts/Manrope-SemiBold.woff2') format('woff2');
}

/* === Design tokens === */

:root {
  --color-primary: #BBFD3B;
  --color-primary-dark-1: #A6E532;
  --color-primary-dark-2: #91CD2A;
  --color-primary-light-1: #C8FE5C;
  --color-primary-light-2: #D5FE7E;
  --color-primary-alpha-30: rgba(187, 253, 59, 0.30);
  --color-primary-alpha-50: rgba(187, 253, 59, 0.50);

  --color-secondary: #37f5ef;
  --color-tertiary: #ff8db4;
  --color-error: #ff7351;
  --color-error-bg: rgba(255, 115, 81, 0.15);

  /* Inverted surface ladder */
  --color-body: #fafafa;
  --color-box-body: #f4f4f5;
  --color-box-body-highlight: #ececee;
  --color-box-header: #ffffff;
  --color-secondary-bg: #ffffff;
  --color-menu: #ffffff;

  /* Not pure black for body text */
  --color-text: #1a1a1a;
  --color-text-light: #525252;
  --color-text-light-2: #757575;
  --color-text-light-3: #ababab;
  --color-text-dark: #0e0e0e;

  /* Ghost border tightened for visibility on light surface */
  --color-border: rgba(60, 60, 60, 0.12);
  --color-light-border: rgba(60, 60, 60, 0.08);

  --border-radius: 4px;
  --border-radius-medium: 8px;
  --border-radius-large: 20px;
  --border-radius-full: 9999px;

  --fonts-regular: 'Manrope', 'OPPOSans', 'PingFang SC', system-ui, sans-serif;
  --fonts-display: 'Poppins', 'Epilogue', 'PingFang SC', system-ui, sans-serif;
  --fonts-monospace: ui-monospace, 'Cascadia Code', 'Source Code Pro', monospace;
}

/* === Body & headlines === */

body {
  background: var(--color-body);
  color: var(--color-text);
  font-family: var(--fonts-regular);
  line-height: 1.6;
}

h1, h2, h3, .ui.header, .display-lg, .display-md {
  font-family: var(--fonts-display);
  letter-spacing: -0.02em;
  color: var(--color-text-dark);
}

.ui.label, .label-md {
  font-family: var(--fonts-regular);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-size: 0.75rem;
  font-weight: 500;
}

/* === Buttons === */

.ui.button {
  border-radius: var(--border-radius-full);
  font-family: var(--fonts-regular);
  font-weight: 500;
  letter-spacing: 0.02em;
  box-shadow: none;
  transition: background 0.15s ease, box-shadow 0.2s ease;
}
.ui.primary.button {
  background: var(--color-primary);
  color: #0e0e0e;
}
.ui.primary.button:hover, .ui.primary.button:focus {
  background: var(--color-primary-dark-1);
  box-shadow: 0 0 24px var(--color-primary-alpha-30);
}
.ui.secondary.button {
  background: transparent;
  border: 1px solid rgba(60, 60, 60, 0.18);
  color: var(--color-text);
}
.ui.secondary.button:hover {
  background: var(--color-box-body-highlight);
}
.ui.basic.button, .ui.tertiary.button {
  background: transparent;
  box-shadow: none;
  color: var(--color-text);
}
.ui.basic.button:hover, .ui.tertiary.button:hover {
  background: transparent !important;
  border-bottom: 2px solid var(--color-primary-dark-1);
  color: var(--color-text-dark);
}

/* === Cards === */

.ui.card, .ui.cards > .card {
  border-radius: var(--border-radius-medium);
  border: none !important;
  background: var(--color-box-header);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: background 0.15s ease, transform 0.15s ease, box-shadow 0.15s ease;
}
.ui.card:hover, .ui.cards > .card:hover {
  background: var(--color-box-header);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

/* === Inputs === */

.ui.input input,
.ui.form input[type=text],
.ui.form input[type=email],
.ui.form input[type=password],
.ui.form input[type=search],
.ui.form input[type=number],
.ui.form input[type=url],
.ui.form textarea,
.ui.dropdown {
  background: var(--color-box-body);
  border: none;
  border-bottom: 2px solid transparent;
  border-radius: var(--border-radius) var(--border-radius) 0 0;
  color: var(--color-text);
  transition: border-color 0.15s ease, background 0.15s ease;
}
.ui.input input:focus,
.ui.form input[type=text]:focus,
.ui.form input[type=email]:focus,
.ui.form input[type=password]:focus,
.ui.form input[type=search]:focus,
.ui.form input[type=number]:focus,
.ui.form input[type=url]:focus,
.ui.form textarea:focus {
  background: var(--color-box-body-highlight);
  border-bottom-color: var(--color-primary-dark-1);
  outline: none;
}
.ui.form .error.field input,
.ui.form .error.field textarea {
  border-bottom-color: var(--color-error);
  background: var(--color-error-bg);
  color: var(--color-error);
}

/* === Modals === */

.ui.modal {
  border-radius: var(--border-radius-large);
  background: var(--color-box-header);
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.12);
}
.ui.modal > .header {
  background: var(--color-box-header);
  border-bottom: none;
  border-top-left-radius: var(--border-radius-large);
  border-top-right-radius: var(--border-radius-large);
  font-family: var(--fonts-display);
  letter-spacing: -0.01em;
}
.ui.modal > .actions {
  background: var(--color-box-header);
  border-top: none;
  border-bottom-left-radius: var(--border-radius-large);
  border-bottom-right-radius: var(--border-radius-large);
}
.ui.dimmer.modals, .ui.dimmer {
  background: rgba(14, 14, 14, 0.40);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
}

/* === Chips & tags === */

.ui.tag.label, .ui.basic.label {
  border-radius: var(--border-radius-full);
}
.ui.label.green   { background: rgba(187, 253, 59, 0.15); color: var(--color-primary-dark-2); }
.ui.label.cyan    { background: rgba(55, 245, 239, 0.15); color: #1cb2ad; }
.ui.label.pink    { background: rgba(255, 141, 180, 0.15); color: #d96b91; }
.ui.label.orange  { background: rgba(255, 115, 81, 0.15);  color: var(--color-error); }

/* === Dividers === */

.ui.divider {
  border: none;
  height: 12px;
  margin: 0;
}
.ui.horizontal.divider {
  height: auto;
  color: var(--color-text-light-2);
}

/* === Scrollbar === */

::-webkit-scrollbar { width: 10px; height: 10px; }
::-webkit-scrollbar-track { background: var(--color-body); }
::-webkit-scrollbar-thumb { background: var(--color-box-body-highlight); border-radius: 5px; }
::-webkit-scrollbar-thumb:hover { background: var(--color-text-light-3); }

/* === Links === */

a, a.muted { color: var(--color-primary-dark-2); }
a:hover, a.muted:hover { color: var(--color-primary-dark-1); text-decoration: none; }

/* === Code blocks === */

code, pre, .CodeMirror {
  font-family: var(--fonts-monospace);
  background: var(--color-box-body);
}
EOF

wc -l custom/public/assets/css/theme-hackforger-light.css
```

Expected: ~230 lines.

---

## Task 6: Set hackforger-dark as default theme

**Files:**
- Modify: `custom/conf/app.ini` — append `DEFAULT_THEME = hackforger-dark` to the `[ui]` section

- [ ] **Step 6.1: Inspect existing app.ini for [ui] section**

```bash
grep -n "^\[ui\]\|^DEFAULT_THEME\|^THEMES" custom/conf/app.ini | head -10
```

If `[ui]` section already exists with a DEFAULT_THEME line, edit it. Otherwise, append both.

- [ ] **Step 6.2a: If [ui] section exists, replace existing DEFAULT_THEME line in place**

```bash
# Use sed for in-place edit. macOS sed needs the empty '' arg.
sed -i '' 's|^DEFAULT_THEME *=.*|DEFAULT_THEME = hackforger-dark|' custom/conf/app.ini
grep -n "^DEFAULT_THEME" custom/conf/app.ini
```

- [ ] **Step 6.2b: If [ui] section doesn't exist OR DEFAULT_THEME line is missing, append**

```bash
# Append a fresh [ui] block at end of file.
cat >> custom/conf/app.ini <<'EOF'

[ui]
DEFAULT_THEME = hackforger-dark
EOF
tail -5 custom/conf/app.ini
```

(Pick **6.2a** OR **6.2b** based on what step 6.1 showed. Do not run both.)

- [ ] **Step 6.3: Verify**

```bash
grep -A 2 "^\[ui\]" custom/conf/app.ini
```
Expected: `[ui]` followed by `DEFAULT_THEME = hackforger-dark` somewhere in the section.

---

## Task 7: Stage commits

- [ ] **Step 7.1: Verify all artifacts present**

```bash
ls custom/public/assets/img/logo.svg custom/public/assets/img/favicon.svg
ls custom/public/assets/css/theme-hackforger-{light,dark}.css
ls custom/public/assets/fonts/{Poppins,Manrope}-*.woff2 custom/public/assets/fonts/OFL.txt
grep "DEFAULT_THEME" custom/conf/app.ini
```
Expected: 2 SVGs, 2 CSS files, 7 woff2 + OFL.txt, DEFAULT_THEME line present.

- [ ] **Step 7.2: Commit 1 — logo + favicon**

```bash
git add custom/public/assets/img/logo.svg custom/public/assets/img/favicon.svg
git commit -m "$(cat <<'EOF'
feat(custom): replace logo and favicon with brand assets (#70)

Single self-adapting SVG (white background under light, orange under
dark via prefers-color-scheme). Source: Cynthialime's two SVGs from
issue #70 comments combined into one via inline <style>.

Spec: docs/superpowers/specs/2026-04-21-logo-theme-replacement-design.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 7.3: Commit 2 — themes + fonts**

```bash
git add custom/public/assets/css/theme-hackforger-light.css \
        custom/public/assets/css/theme-hackforger-dark.css \
        custom/public/assets/fonts/

git commit -m "$(cat <<'EOF'
feat(custom): add hackforger themes following DESIGN.md spec (#70)

Implements DESIGN.md "High-Energy Kineticism / Void and Vibe":
- Fluorescent green primary (#BBFD3B), cyan + pink secondaries
- Tonal Layering surface hierarchy (no 1px solid borders)
- Pill buttons, no-border cards, input bottom-stroke focus
- 20px modal radius with glassmorphism backdrop
- Poppins display + Manrope body, locally bundled (OFL 1.1, no CDN call)

Theme files override the bundled web_src/css/themes/theme-hackforger-*
via Forgejo's custom/public asset precedence — no main-codebase changes,
no rebuild needed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 7.4: Commit 3 — default theme**

```bash
git add custom/conf/app.ini

git commit -m "$(cat <<'EOF'
feat(custom): set hackforger-dark as default theme (#70)

Anonymous visitors and new users get the new dark theme out of the
box. Logged-in users keep their saved preference (DEFAULT_THEME only
affects the fallback when no user preference is set).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 7.5: Inspect commit list**

```bash
git log --oneline origin/v0.1-dev/hackforger..HEAD
```
Expected: 4 commits — spec + 3 feat commits.

---

## Task 8: Deploy + browser verification

The instance running on port 3000 (`hackforger.inside.h2os.cloud`) currently uses the worktree at `~/.config/superpowers/worktrees/hackforger/test-instance`. Switch it to use the new worktree's binary + custom dir, OR signal the live instance to reload.

Forgejo serves files from `custom/public/...` at runtime — **no rebuild needed for CSS/SVG changes**. But the live instance currently has its `WORK_PATH` pointing at the old worktree's `custom/`. Two options:

- **8a (preferred):** Move the new `custom/` files onto the live instance's expected `custom/` directory (which is `/Users/h2oslabs/Workspace/hackforger/custom/`). Since the worktree's `custom/` is shared with the main repo (`custom/` is git-tracked, and we're on a feature branch), the simplest is to merge the PR first, then the main worktree picks up the change automatically.
- **8b:** Restart the live gitea pointing at this new worktree's custom dir.

For pre-merge verification, use **8b**:

- [ ] **Step 8.1: Stop the current live instance**

```bash
# Find PID of gitea on port 3000
lsof -iTCP:3000 -sTCP:LISTEN -n | grep gitea
# kill it (use the PID returned above)
```
> ⚠️ This requires user authorization — the permission system blocks killing a process the agent didn't start. Ask the user to run `! kill <pid>`.

- [ ] **Step 8.2: Start gitea from the issue70 worktree**

```bash
cd ~/.config/superpowers/worktrees/hackforger/issue70-logo-theme

# Need a built binary. The worktree doesn't have one yet — build first.
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -3

# Copy app.ini from main repo (so we share the live DB)
mkdir -p custom/conf
# (custom/conf/app.ini is git-tracked, but the live one might have local-only edits — back up first)
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini

# Append DEFAULT_THEME to this app.ini if Step 6 ran in a different file
grep "DEFAULT_THEME" custom/conf/app.ini || echo -e "\n[ui]\nDEFAULT_THEME = hackforger-dark" >> custom/conf/app.ini

# Clear stale lock
rm -f /Users/h2oslabs/Workspace/hackforger/data/queues/common/LOCK

# Start
./gitea web 2>&1 &
sleep 8
```

- [ ] **Step 8.3: Verify the new theme CSS is being served**

```bash
curl -sk https://hackforger.inside.h2os.cloud/assets/css/theme-hackforger-dark.css | head -5
```
Expected: starts with `/* HackForger Dark Theme — based on DESIGN.md ...` (the new file from Task 4), NOT the bundled steel/zinc one.

If you see the bundled one, the custom/ override isn't taking effect. Check `setting.StaticRootPath` and `setting.CustomPath` in the running config and confirm the file is at the path Forgejo expects.

- [ ] **Step 8.4: Verify logo is the new combined SVG**

```bash
curl -sk https://hackforger.inside.h2os.cloud/assets/img/logo.svg | head -3
```
Expected: starts with `<svg ...>` then the `<style>` block from Task 1.

- [ ] **Step 8.5: Browser verification — anonymous landing page**

Use agent-browser to capture the landing page in dark mode (default for anonymous):

```bash
# In a separate terminal / via the agent-browser skill
agent-browser screenshot https://hackforger.inside.h2os.cloud/ \
  --output docs/tests/e2e/screenshots/issue70/01-anonymous-dark.png \
  --color-scheme dark
```
Expected on screenshot:
- Logo's icon area is **orange** (`#FAAD14`)
- Page background is **#0e0e0e** (true dark)
- Headline uses **Poppins** (geometric, sporty)
- Body text uses **Manrope** (clean)
- No 1px solid borders separating sections

- [ ] **Step 8.6: Browser verification — logged-in dashboard (dark)**

```bash
# Login as hackforger first via agent-browser, then:
agent-browser screenshot https://hackforger.inside.h2os.cloud/ \
  --authenticated \
  --output docs/tests/e2e/screenshots/issue70/02-dashboard-dark.png
```
Expected: pill-shaped primary button on the page, no-border cards, neon-green accents.

- [ ] **Step 8.7: Browser verification — light theme**

In user settings (`/-/user/settings/appearance`), switch to `hackforger-light`. Take a screenshot:

```bash
agent-browser screenshot https://hackforger.inside.h2os.cloud/ \
  --authenticated \
  --output docs/tests/e2e/screenshots/issue70/03-dashboard-light.png
```
Expected: light surfaces (`#fafafa` body, white cards), same green primary, same component shapes.

- [ ] **Step 8.8: Write E2E report**

Create `docs/tests/e2e/reports/issue-70-logo-theme-report.md` with:
- The 3 screenshots embedded
- Bullet checklist mapping each DESIGN.md rule to ✅/❌
- Any issues to follow up

- [ ] **Step 8.9: Commit the E2E report**

```bash
git add docs/tests/e2e/reports/issue-70-logo-theme-report.md \
        docs/tests/e2e/screenshots/issue70/
git commit -m "$(cat <<'EOF'
docs(e2e): logo + theme replacement verification (#70)

Three screenshots via agent-browser confirming:
- Anonymous landing renders with new dark theme + adapted logo
- Logged-in dashboard shows pill buttons + no-border cards + Manrope text
- Light theme variant works when user switches in settings

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Push + open PR

- [ ] **Step 9.1: Push branch**

```bash
git push -u origin feat/issue-70-logo-theme
```

- [ ] **Step 9.2: Open PR**

```bash
gh pr create --base v0.1-dev/hackforger \
  --title "feat: replace logo + apply DESIGN.md theme (#70)" \
  --body "$(cat <<'EOF'
## Summary

Closes #70.

Replaces the platform logo with Cynthialime's brand SVGs and generates light/dark themes from DESIGN.md "High-Energy Kineticism" spec. All changes land in \`custom/\` — zero modifications to the main codebase, no rebuild required.

## Changes

- **Logo**: single self-adapting SVG (white-bg light, orange-bg dark via \`prefers-color-scheme\`)
- **Favicon**: icon-only square crop with same self-adapting style
- **Themes**: \`theme-hackforger-{light,dark}.css\` overriding bundled versions via Forgejo's \`custom/public\` asset precedence
- **Fonts**: Poppins + Manrope bundled locally as woff2 (OFL 1.1, no CDN call)
- **Default**: \`hackforger-dark\` for anonymous visitors and new users

## Verification

3 agent-browser screenshots in \`docs/tests/e2e/reports/issue-70-logo-theme-report.md\`:
1. Anonymous landing — dark theme + orange logo
2. Logged-in dashboard — pill buttons, no-border cards, neon green primary
3. Light theme — same components, light surfaces

## Test plan

- [x] Verify logo.svg is served from custom/ (Step 8.4)
- [x] Verify theme-hackforger-dark.css served from custom/ (Step 8.3)
- [x] Anonymous landing renders dark by default (Step 8.5)
- [x] Logged-in dashboard works (Step 8.6)
- [x] Light variant works when user switches (Step 8.7)
- [ ] Cynthialime visual sign-off

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 9.3: Comment on Issue #70 with PR link**

```bash
gh issue comment 70 --body "PR open: #<NEW_PR_NUMBER>. Logo + DESIGN.md theme integrated as custom/ overrides. Once merged + deployed, please verify on https://hackforger.inside.h2os.cloud."
```

---

## Self-review checklist (run before declaring done)

- [ ] Anonymous visitor sees dark theme + new logo on https://hackforger.inside.h2os.cloud
- [ ] No console errors related to font loading (open browser dev tools → Network tab → filter "font")
- [ ] All 7 woff2 files load with HTTP 200
- [ ] No 404s on /assets/img/logo.svg, /assets/img/favicon.svg
- [ ] User settings → appearance still lists hackforger-light + hackforger-dark, both functional
- [ ] PR has all 4 commits (1 spec + 1 logo + 1 themes/fonts + 1 default)
