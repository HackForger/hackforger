# Theme Q1 Follow-up — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the seven defects fixed by `2026-04-21-theme-q1-followup-design.md` — proper logo theme switching, deeper warm-paper light surfaces, hover/focus quieter pass, dark surface lift, light primary button hover, and dead colorize cleanup.

**Architecture:** Two new SVGs (designer's logo-1 = light, logo-2 = dark) + CSS `content: url()` swap via attribute selector under each theme override block. All other changes are token replacements + a few new rules in the existing override blocks at the tail of the bundled built CSS.

**Tech Stack:** CSS only. No Go, no JS, no template forks.

---

### Task 1: Set up worktree

**Files:**
- Worktree dir: `/Users/h2oslabs/.config/superpowers/worktrees/hackforger/theme-q1-followup`
- Branch: `fix/theme-q1-followup`

- [ ] **Step 1: Create worktree**

```bash
git worktree add /Users/h2oslabs/.config/superpowers/worktrees/hackforger/theme-q1-followup -b fix/theme-q1-followup origin/v0.1-dev/hackforger
```

- [ ] **Step 2: Copy app.ini for local server testing**

```bash
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini \
   /Users/h2oslabs/.config/superpowers/worktrees/hackforger/theme-q1-followup/custom/conf/app.ini
```

(per CLAUDE.md: worktree needs app.ini before any `gitea web`)

- [ ] **Step 3: Verify clean baseline**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/theme-q1-followup
git log --oneline -3
```

Expected: HEAD = `43e6b08` (PR #84 merge) or later.

---

### Task 2: Create logo SVG files

**Files:**
- Source: `/tmp/issue70/logo-1.bin` (orange = LIGHT variant)
- Source: `/tmp/issue70/logo-2.bin` (green = DARK variant)
- Create: `custom/public/assets/img/logo-light.svg`
- Create: `custom/public/assets/img/logo-dark.svg`
- Modify: `custom/public/assets/img/logo.svg` (default fallback = light variant content)

- [ ] **Step 1: Verify source SVGs exist + copy to target paths**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/theme-q1-followup
test -s /tmp/issue70/logo-1.bin || { echo "FATAL: /tmp/issue70/logo-1.bin missing or empty"; exit 1; }
test -s /tmp/issue70/logo-2.bin || { echo "FATAL: /tmp/issue70/logo-2.bin missing or empty"; exit 1; }
cp /tmp/issue70/logo-1.bin custom/public/assets/img/logo-light.svg
cp /tmp/issue70/logo-2.bin custom/public/assets/img/logo-dark.svg
```

If `/tmp/issue70/` is gone (volatile across sessions), STOP and ask user to re-place the source SVGs.

- [ ] **Step 2: Verify no `<style>` or `@media` blocks remain**

```bash
grep -c '<style\|@media' custom/public/assets/img/logo-light.svg
grep -c '<style\|@media' custom/public/assets/img/logo-dark.svg
```

Expected output: `0` for both.

If non-zero: edit the file to remove the `<style>...</style>` block (it's typically right after the opening `<svg>` tag), and any `@media` rules inside it. Re-run the grep until both return `0`.

- [ ] **Step 3: Sanity-check the fill colors match the spec**

```bash
grep -oE 'fill="(white|#[0-9A-Fa-f]+)"' custom/public/assets/img/logo-light.svg | sort | uniq -c
```
Expected: presence of `#FAAD14` (orange) and `#00000E` (dark text). Black "Syn" letters visible.

```bash
grep -oE 'fill="(white|#[0-9A-Fa-f]+)"' custom/public/assets/img/logo-dark.svg | sort | uniq -c
```
Expected: presence of `#BBFD3B` (green) and `white` (visible "Syn" letters on dark bg).

- [ ] **Step 4: Replace logo.svg with logo-light.svg (default fallback)**

```bash
cp custom/public/assets/img/logo-light.svg custom/public/assets/img/logo.svg
```

- [ ] **Step 5: Stage + commit**

```bash
git add custom/public/assets/img/logo.svg \
        custom/public/assets/img/logo-light.svg \
        custom/public/assets/img/logo-dark.svg
git -c commit.gpgsign=false commit -m "feat(theme): add per-theme logo SVGs (light/dark variants)

- logo-light.svg: orange icon + dark wordmark (designer's logo-1)
- logo-dark.svg: white icon + green wordmark + white 'Syn' (designer's logo-2)
- logo.svg: default fallback set to light variant for browsers without
  CSS content:url() support

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Update light theme overrides

**Files:**
- Modify: `custom/public/assets/css/theme-hackforger-light.css` (the override block at the END of the file, lines after `/* HackForger Light — DESIGN.md Polish Overrides (Q1 audit pass) */`)

- [ ] **Step 1: Read current override block to identify exact replacement targets**

```bash
tail -135 custom/public/assets/css/theme-hackforger-light.css
```

Confirm the section markers: surface ladder block, link/hover block, primary button block, label color block.

- [ ] **Step 2: Replace surface ladder values**

Find the block in `theme-hackforger-light.css`:
```css
:root {
    --color-body: #fbfaf7;            /* warm paper white */
    --color-box-body: #f3f1ea;        /* tonal step 1 */
    --color-box-body-highlight: #e8e5db;
    --color-box-header: #ebe8df;
    --color-menu: #f3f1ea;
    --color-card: #ffffff;            /* cards lift above body */
    --color-markup-table-row: #f3f1ea;
    --color-markup-code-block: #ebe8df;
    --color-active: #e0ddd2;
    --color-hover: #ebe8df;
    --color-secondary: #d4d1c5;
```

Replace with:
```css
:root {
    /* /bolder pass 2 — committed warm-paper depth (PR #84 was too white).
     * Body drops to ~93% luminance with visible cream tint; cards INVERT
     * lighter than body to land the "fresh paper sheet on aged desk" metaphor.
     * WCAG: verify --color-text contrast against new tokens (test plan §5.4). */
    --color-body: #efece1;            /* aged paper, ~93% */
    --color-box-body: #e7e3d4;        /* tonal step 1, ~90% */
    --color-box-body-highlight: #ddd8c5;
    --color-box-header: #e3decd;
    --color-menu: #e7e3d4;
    --color-card: #f6f3e9;            /* fresh sheet on desk (lighter than body) */
    --color-markup-table-row: #e7e3d4;
    --color-markup-code-block: #ddd8c5;
    --color-active: #cfc9b3;
    --color-hover: #ddd8c5;
    --color-secondary: #c8c2ab;
```

- [ ] **Step 3: KEEP light `a:hover` at dark-3** (no change needed)

The light theme's existing PR #84 hover at `--color-primary-dark-3` is in the correct direction (DARKER on cream paper = more attention; "ink-on-paper" semantic). Per revised spec §3.5: light hover stays at `dark-3`, only dark theme hover changes.

**No edit.** Proceed to Step 4.

- [ ] **Step 4: Replace focus-visible outline (light: dark-2 → dark-3)**

Find:
```css
a:focus-visible,
button:focus-visible,
input:focus-visible,
textarea:focus-visible,
select:focus-visible,
.ui.button:focus-visible,
.ui.dropdown:focus-visible {
    outline: 2px solid var(--color-primary-dark-2);
    outline-offset: 2px;
}
```

Replace with:
```css
/* /quieter — focus ring uses dark-3 (one step darker than link rest at
 * dark-2) so it's visibly distinct without going into the lighter zone
 * where AA contrast against #efece1 cream paper would fail. 3px offset
 * for AA visibility. */
a:focus-visible,
button:focus-visible,
input:focus-visible,
textarea:focus-visible,
select:focus-visible,
.ui.button:focus-visible,
.ui.dropdown:focus-visible {
    outline: 2px solid var(--color-primary-dark-3);
    outline-offset: 3px;
}
```

- [ ] **Step 5: Replace primary button hover shadow**

Find:
```css
.ui.primary.button:hover {
    box-shadow: 0 2px 8px 0 rgba(0, 0, 0, 0.08);
    background-color: var(--color-primary-dark-1);
}
```

Replace with:
```css
/* /polish — soft green-tint hover shadow on paper.
 * Black-8% was invisible against warm paper; brand tint at 18% opacity
 * provides visible affordance while keeping the "ink on paper" feel. */
.ui.primary.button:hover {
    box-shadow: 0 2px 12px 0 rgba(111, 161, 24, 0.18);
    background-color: var(--color-primary-dark-1);
}
```

- [ ] **Step 6: Remove dead `.ui.label.cyan/.pink/.green` rules**

Find and DELETE these blocks entirely (they don't fire — Forgejo labels use inline `style=`, not class names):
```css
/* /colorize — paper-tone chips. Lower opacity, darker text. */
.ui.label.cyan, .ui.labels .label.cyan {
    background: var(--color-cyan-alpha-15);
    color: var(--color-cyan);
    border: none;
    border-radius: 999px;
}
.ui.label.pink, .ui.labels .label.pink {
    background: var(--color-pink-alpha-15);
    color: var(--color-pink);
    border: none;
    border-radius: 999px;
}
.ui.label.green, .ui.labels .label.green {
    background: rgba(111, 161, 24, 0.12);
    color: var(--color-primary-dark-3);
    border: none;
    border-radius: 999px;
}
```

Replace with a single comment marker:
```css
/* /colorize cleanup — removed .ui.label.cyan/.pink/.green rules from PR #84.
 * Forgejo labels use inline style="background:#hex" set per-label, not class
 * names, so these selectors never fired. Targeted HackForger chip styling
 * (bounty/phase/credit) deferred to a separate spec. */
```

- [ ] **Step 7: Append logo swap rule**

Append at the very end of the file:
```css

/* Logo theme swap — orange variant on light theme.
 * Attribute selector covers all 4 call-sites (header, home, signin, 500),
 * since 3 of 4 templates emit the img with no class. Both source and
 * target SVGs share viewBox="0 0 365 51" — layout boxes preserved. */
img[src$="/img/logo.svg"] {
    content: url("/assets/img/logo-light.svg");
}
```

- [ ] **Step 8: Stage + commit**

```bash
git add custom/public/assets/css/theme-hackforger-light.css
git -c commit.gpgsign=false commit -m "fix(theme/light): deeper warm-paper surfaces + quieter hover/focus + logo swap

- Surface ladder dropped 5-6pt (#efece1 body) — commits to warm-paper
  metaphor that PR #84 only declared in comments.
- Cards invert to lighter-than-body (#f6f3e9) — fresh-paper-on-desk.
- Hover and focus use --color-primary-dark-1 (less strobe).
- Primary button hover gets soft green-tint shadow.
- Logo swap to logo-light.svg via img[src\$=...] attribute selector.
- Removed dead .ui.label.cyan/.pink/.green rules (Forgejo labels use
  inline style, not class names).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Update dark theme overrides

**Files:**
- Modify: `custom/public/assets/css/theme-hackforger-dark.css` (the override block at the END of the file)

- [ ] **Step 1a: Replace generic `a:hover` color (first occurrence in dark theme)**

The two hover rules in the dark theme are NOT contiguous — they're separated by `a:visited` and `.ui.menu a, .ui.dropdown .menu > .item` blocks. Edit each separately.

Find:
```css
a:hover, a.muted:hover, a.suppressed:hover {
    color: var(--color-primary);
}
```

Replace with:
```css
/* /quieter pass 2 — hover lifts to dark-1, not raw fluorescent.
 * Stops the dense-page strobe on issue lists / file trees. */
a:hover, a.muted:hover, a.suppressed:hover {
    color: var(--color-primary-dark-1);
}
```

- [ ] **Step 1b: Replace `.ui.menu a:hover` color (second occurrence)**

Find:
```css
.ui.menu a:hover, .ui.dropdown .menu > .item:hover {
    color: var(--color-primary);
}
```

Replace with:
```css
.ui.menu a:hover, .ui.dropdown .menu > .item:hover {
    color: var(--color-primary-dark-1);
}
```

- [ ] **Step 2: Replace focus-visible outline**

Find:
```css
a:focus-visible,
button:focus-visible,
input:focus-visible,
textarea:focus-visible,
select:focus-visible,
.ui.button:focus-visible,
.ui.dropdown:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
}
```

Replace with:
```css
/* /quieter — focus ring uses dark-1 with 3px offset for AA visibility
 * without competing with hover/active for attention. */
a:focus-visible,
button:focus-visible,
input:focus-visible,
textarea:focus-visible,
select:focus-visible,
.ui.button:focus-visible,
.ui.dropdown:focus-visible {
    outline: 2px solid var(--color-primary-dark-1);
    outline-offset: 3px;
}
```

- [ ] **Step 3: Remove dead `.ui.label.cyan/.pink/.green` rules**

Same as light Step 6 — find and delete the three blocks, replace with the comment marker.

- [ ] **Step 4: Append surface lift shadow**

Append before the logo swap rule:
```css

/* /polish — DESIGN.md §4 ambient lift via inset highlight.
 * 1px white at 4% opacity at the top edge of elevated surfaces.
 * Visible on calibrated displays as a subtle "lift" without a line;
 * gracefully invisible on uncalibrated displays. */
.ui.segment, .ui.card, .ui.menu, .ui.attached.header,
.repository .header-wrapper, .ui.tabular.menu {
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}
```

- [ ] **Step 5: Append logo swap rule**

```css

/* Logo theme swap — green variant on dark theme.
 * Attribute selector covers all 4 call-sites. Both SVGs share viewBox. */
img[src$="/img/logo.svg"] {
    content: url("/assets/img/logo-dark.svg");
}
```

- [ ] **Step 6: Stage + commit**

```bash
git add custom/public/assets/css/theme-hackforger-dark.css
git -c commit.gpgsign=false commit -m "fix(theme/dark): quieter hover/focus + surface lift + logo swap

- Hover and focus use --color-primary-dark-1 instead of raw fluorescent
  (stops dense-page strobe).
- 1px white-4% inset shadow on elevated surfaces (DESIGN.md §4 lift).
- Logo swap to logo-dark.svg via img[src\$=...] attribute selector.
- Removed dead .ui.label.cyan/.pink/.green rules (per spec §3.8).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Build + verify locally

**Files:**
- No file changes — verification only.

- [ ] **Step 1: Run frontend build (so any other consumers of the CSS pipeline are fresh)**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/theme-q1-followup
make frontend 2>&1 | tail -20
```

Expected: completes without error. CSS in `public/assets/css/` updated.

(Note: our theme overrides live in `custom/public/assets/css/`, which Forgejo serves AHEAD of `public/assets/css/`. So technically `make frontend` is not strictly required for our changes — but per CLAUDE.md and reviewer feedback, run it to be safe.)

- [ ] **Step 2: Verify no syntax errors in the CSS**

Use the Read tool on the last ~200 lines of each CSS file (read with offset = total-200). Visually scan: balanced `{` and `}`, no duplicate `:root` openings without proper close, attribute selector `img[src$="/img/logo.svg"]` quoted correctly, expected new comment markers present (`/* /quieter pass 2 */`, `/* /polish — DESIGN.md §4 */`, `/* Logo theme swap */`).

- [ ] **Step 3: Push branch + open PR**

```bash
git push -u origin fix/theme-q1-followup
gh pr create --base v0.1-dev/hackforger --head fix/theme-q1-followup \
  --title "fix(theme): proper logo theme switching + light depth + critique fixes" \
  --body "$(cat <<'EOF'
## Summary

Implements the spec at \`docs/superpowers/specs/2026-04-21-theme-q1-followup-design.md\`. Addresses two user-reported defects + four Q1 \`/critique\` carry-overs:

1. **Logo white-text bug fix** — designer's logo-1 (orange + black "Syn") now serves the light theme; logo-2 (green + white "Syn") serves dark. CSS \`content: url()\` swap via \`img[src$="/img/logo.svg"]\` attribute selector covers all 4 call-sites.
2. **Light "too white" fix** — surface ladder dropped 5-6pt; cards invert lighter than body for "paper on desk" metaphor.
3. **Hover/focus quieter** — \`--color-primary-dark-1\` instead of raw fluorescent.
4. **Dark surface lift** — DESIGN.md §4 1px white-4% inset shadow.
5. **Light primary hover** — green-tint shadow replaces invisible black-8%.
6. **Colorize cleanup** — removed dead \`.ui.label.cyan/.pink/.green\` rules.

## Test plan

- [ ] Hard-refresh / and / + 4 logo call-sites on light: orange icon + black "Syn" visible
- [ ] Switch to dark: green icon + white "Syn" visible at same call-sites
- [ ] Body cream visible from 3ft on light
- [ ] Cards lighter than body on light (paper-on-desk inversion)
- [ ] Hover any link in dense list — no fluorescent strobe
- [ ] Tab through form — focus ring is dark-1 with 3px offset
- [ ] Light primary button hover — green tint shadow visible
- [ ] Dark side panel — subtle 1px top-edge highlight (calibrated displays)
- [ ] WCAG AA contrast on light: \`--color-text\` against #efece1 / #e7e3d4 / #f6f3e9
- [ ] VoiceOver on home logo — "logo" announced after swap
- [ ] Cross-browser: Chrome / Safari / Firefox swap takes effect

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

### Task 6: Hand off to user for merge + restart

**Files:**
- No file changes.

The user explicitly authorized admin-merge + instance restart in advance for THIS plan. Execute, but with care: confirm the PR diff is what we expect before merging, and tell the user immediately after restart so they can browser-verify.

- [ ] **Step 1: Print PR diff summary for sanity check**

```bash
gh pr diff <PR-NUMBER> --repo HackForger/hackforger | head -100
```

Expected to show: 2 new SVG files, 1 modified SVG (logo.svg), changes in 2 CSS files. If anything else appears, STOP.

- [ ] **Step 2: Admin merge** (user authorized)

```bash
gh pr merge <PR-NUMBER> --repo HackForger/hackforger --merge --admin --delete-branch
```

If hook denies: tell user, ask for explicit re-authorization before retry.

- [ ] **Step 3: Pull main**

```bash
cd /Users/h2oslabs/Workspace/hackforger
git pull origin v0.1-dev/hackforger
```

- [ ] **Step 4: Identify running gitea PID** (do not kill — system blocks agent kills)

```bash
ps aux | grep -E '[g]itea web' | grep -v grep | awk '{print "PID: " $2}'
```

Print the PID and tell user:
> "Found gitea PID: `<pid>`. Please run `! kill <pid>` in your shell to stop it; I'll restart immediately after."

- [ ] **Step 5: After user kills, clean lock + restart**

```bash
rm -f data/queues/common/LOCK
nohup ./gitea web > /tmp/gitea.log 2>&1 &
disown
sleep 5
```

Then read `/tmp/gitea.log` (use Read tool, not `tail`) and confirm `Listen: http://0.0.0.0:3000`.

- [ ] **Step 6: Hand off to user**

Tell user: "Instance restarted on `localhost:3000`. Hard-refresh (Cmd+Shift+R) in browser, then run through PR test plan checkboxes."

---

## Self-review checklist

- [x] All 7 spec defects mapped to a task (logo file create + 2 theme override updates).
- [x] Each step is self-contained — no "see step X above" references.
- [x] Concrete CSS values everywhere — no TBD, no "appropriate".
- [x] Verification steps after each file change.
- [x] Frequent commits (per task, not at the end).
- [x] Build verification before PR.
- [x] Reviewer-flagged issues all addressed:
  - `img[src$=...]` selector (covers all 4 call-sites)
  - Dead colorize removal
  - No favicon work
  - Source SVG precondition check (Task 2 Step 1)
  - Light hover stays at `dark-3` (correct ink-on-paper direction); only dark hover changes
  - Light focus moves `dark-2` → `dark-3` (distinct from link rest, AA-safe on cream)
  - Dark `a:hover` find-block split into two contiguous sub-rules (Task 4 Step 1a/1b)
  - WCAG verification kept as PR test checkbox (no auto-injected CSS comment block — overreach removed from spec)
  - Task 6 hand-off explicit (PID print → user `! kill` → restart, not unattended)
  - Use Read not `tail | head` for CSS verification
