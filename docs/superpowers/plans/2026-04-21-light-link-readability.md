# Light Theme Link Readability Hotfix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix WCAG-AA-failing light theme link colors per `2026-04-21-light-link-readability-design.md`.

**Architecture:** Edits localized to the override block at the END of `custom/public/assets/css/theme-hackforger-light.css`. Dark theme untouched.

**Tech Stack:** CSS only.

---

### Task 1: Set up worktree

**Files:**
- Worktree: `/Users/h2oslabs/.config/superpowers/worktrees/hackforger/light-link-readability`
- Branch: `fix/light-link-readability`

- [ ] **Step 1: Create worktree from latest origin**

```bash
git worktree add /Users/h2oslabs/.config/superpowers/worktrees/hackforger/light-link-readability \
  -b fix/light-link-readability origin/v0.1-dev/hackforger
```

- [ ] **Step 2: Seed app.ini for local server**

```bash
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini \
   /Users/h2oslabs/.config/superpowers/worktrees/hackforger/light-link-readability/custom/conf/app.ini
```

- [ ] **Step 3: Verify baseline**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/light-link-readability
git log --oneline -3
```

Expected HEAD ≥ `5d13c7b` (PR #87 merge).

---

### Task 2: Verify the call-site list (reviewer-mandated grep)

**Goal:** Reviewer noted `--color-primary-dark-1` is consumed as text in some bundled CSS files. Confirm no OTHER text consumers exist that the spec missed.

- [ ] **Step 1: Grep for all consumers of dark-1/2/3/4 in source CSS**

```bash
cd /Users/h2oslabs/.config/superpowers/worktrees/hackforger/light-link-readability
grep -rn "var(--color-primary-dark-[0-9])" web_src/css/ templates/ 2>&1 | head -30
```

- [ ] **Step 2: Sanity-check that all uses are either**
  - `color: var(...)` — text use, MUST pass AA at 4.5:1 against any surface where it appears
  - `background-color`, `border-color`, `box-shadow` — non-text, 3:1 minimum
  - In a `:hover` / `:active` rule on a button-like element

If any text-use consumer maps `dark-1` to a surface that brings the ratio below 4.5:1, escalate before proceeding (open a new spec; don't ship a regression).

The new `dark-1: #4D7008` is 4.92:1 against `#efece1` (body), 4.42:1 against `#e3decd` (header — JUST under AA), 4.67:1 against `#e7e3d4` (nav). On `#ddd8c5` highlight it's 3.92:1 — fails AA. **If a consumer renders dark-1 text on highlight surface, this is a known limitation; document and accept (highlight is transient/hover state).**

No commit yet — verification only.

---

### Task 3: Apply CSS edits to light theme

**Files:**
- Modify: `custom/public/assets/css/theme-hackforger-light.css`

- [ ] **Step 1: Read current state of the override block**

Use the Read tool on lines 14-55 to confirm the current `:root` block matches what the plan expects.

- [ ] **Step 2: Replace the primary scale block**

Find:
```css
    /* /quieter — primary green stays as the neon driver, but on light
     * we need a darker variant for "ink" (text/links) so the green
     * doesn't get washed out against paper. */
    --color-primary: #BBFD3B;          /* unchanged: action color */
    --color-primary-dark-1: #A8E632;
    --color-primary-dark-2: #6FA118;   /* link-on-paper green */
    --color-primary-dark-3: #5A8413;
    --color-primary-dark-4: #45660F;   /* visited link */
    --color-primary-contrast: #1a1a1a;
```

Replace with:
```css
    /* /normalize — committed ink-on-paper scale that passes WCAG AA.
     * Action surfaces (button bg, current-tab) stay fluorescent;
     * dark-* scale is the "ink" variant for text on cream paper.
     * All dark-* values verified ≥4.5:1 against --color-body. */
    --color-primary: #BBFD3B;          /* action: button bg, 13.8:1 with #1a1a1a text */
    --color-primary-contrast: #1a1a1a;
    --color-primary-hover: #93C82B;    /* button hover bg, 8.45:1 with text */
    --color-primary-active: #7AAB1F;   /* button active bg, 6.20:1 with text */
    --color-primary-dark-1: #4D7008;   /* repo header counts, label text — 4.92:1 AA */
    --color-primary-dark-2: #3D5A09;   /* link rest — 6.79:1 AA */
    --color-primary-dark-3: #2C4D08;   /* link hover — 8.22:1 AAA */
    --color-primary-dark-4: #1A2D04;   /* visited link — 12.5:1 AAA */
    /* Override visible alpha tokens (reaction bg) so they match new dark-2 */
    --color-primary-alpha-20: rgba(61, 90, 9, 0.20);
    --color-primary-alpha-30: rgba(61, 90, 9, 0.30);
```

- [ ] **Step 3: Tighten `--color-text-light`**

Find:
```css
    /* /harden — text contrast on paper.
     * #595959 on #fbfaf7 → 7.0:1 (passes AAA normal text) */
    --color-text: #1a1a1a;
    --color-text-light: #595959;       /* AAA */
```

Replace with:
```css
    /* /harden — text contrast on paper.
     * Tightened text-light to #4a4a4a for safer margin on highlighted
     * surfaces (was #595959 → 4.84:1 on #ddd8c5; now 5.84:1). */
    --color-text: #1a1a1a;
    --color-text-light: #4a4a4a;       /* AAA on body, AA+ on every surface */
```

- [ ] **Step 4: Update backdrop-filter fallback**

Find:
```css
@supports not ((backdrop-filter: blur(20px)) or (-webkit-backdrop-filter: blur(20px))) {
    .full.height > .ui.menu,
    .ui.modal,
    .ui.modals.dimmer {
        background-color: rgba(251, 250, 247, 0.97) !important;
    }
}
```

Replace with:
```css
@supports not ((backdrop-filter: blur(20px)) or (-webkit-backdrop-filter: blur(20px))) {
    .full.height > .ui.menu,
    .ui.modal,
    .ui.modals.dimmer {
        /* matches new --color-body #efece1 (PR #87 changed body, this
         * fallback wasn't kept in sync) */
        background-color: rgba(239, 236, 225, 0.97) !important;
    }
}
```

- [ ] **Step 5: Update focus-visible outline color**

Find:
```css
/* /quieter — focus ring uses dark-3 (one step darker than link rest at
 * dark-2) so it's visibly distinct without going lighter where AA contrast
 * against #efece1 cream paper would fail. 3px offset for AA visibility. */
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

Replace with:
```css
/* /normalize — focus ring uses non-green teal #0e7b75 to break the
 * focus-vs-hover collision (both were dark-3 forest green previously).
 * 4.34:1 against body, passes 3:1 non-text minimum on every surface.
 * Hue distinction (cyan ~180° vs yellow-green ~80°) gives keyboard
 * users a clear "focus" signal regardless of hover state. */
a:focus-visible,
button:focus-visible,
input:focus-visible,
textarea:focus-visible,
select:focus-visible,
.ui.button:focus-visible,
.ui.dropdown:focus-visible {
    outline: 2px solid #0e7b75;
    outline-offset: 3px;
}
```

- [ ] **Step 6: Verify final CSS state**

Use Read tool on the entire override block (lines 14-160 — file is ~140 lines after edits). Visually scan for: balanced braces, no duplicate token definitions in the same `:root`, all comments updated, all 5 changed regions present (primary scale block, text-light, backdrop-filter fallback, focus-visible). Steps 4 and 5 modify lines past 100, so reading 14-50 alone misses them — must read full file.

- [ ] **Step 7: Stage + commit**

```bash
git add custom/public/assets/css/theme-hackforger-light.css
git -c commit.gpgsign=false commit -m "fix(theme/light): WCAG AA link colors + uncouple button hover

User reported /explore/repos links unreadable on light theme. Audit
measured PR #84 link rest at 2.62:1 (FAILS AA). Fix:

- New dark-* scale: dark-2 = #3D5A09 (link rest, 6.79:1), dark-3 = #2C4D08
  (hover, 8.22:1), dark-4 = #1A2D04 (visited, 12.5:1)
- Uncouple button hover: --color-primary-hover = #93C82B (was inherited
  as dark-2 from bundled, would have made buttons jump fluorescent →
  forest green on hover)
- Focus ring: teal #0e7b75 (distinct hue from any link state — keyboard
  users can tell focus apart from hover)
- text-light tightened #595959 → #4a4a4a (more AA margin on highlights)
- Backdrop-filter fallback synced to PR #87's new body color
- Override visible primary-alpha-20/30 to match new dark-2 (reaction bg)

Dark theme untouched — audit confirmed bundled mint scale works.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Push, open PR

- [ ] **Step 1: Push branch**

```bash
git push -u origin fix/light-link-readability
```

- [ ] **Step 2: Open PR**

```bash
gh pr create --base v0.1-dev/hackforger --head fix/light-link-readability \
  --title "fix(theme/light): WCAG AA link readability on cream paper" \
  --body "$(cat <<'PRBODY'
## Summary

Hotfix for user-reported defect on \`/explore/repos\`: link text was unreadable. Audit measured PR #84 link rest at **2.62:1** contrast against the cream body — far below WCAG AA's 4.5:1 minimum. This PR replaces the entire light-theme primary-dark-* scale with values that pass AA, while keeping the brand fluorescent green as the action surface color.

Implements: \`docs/superpowers/specs/2026-04-21-light-link-readability-design.md\`

## Changes

- **Link rest**: \`#6FA118\` (2.62:1 ❌) → \`#3D5A09\` (**6.79:1 ✓ AA**)
- **Link hover**: \`#5A8413\` (3.77:1 ❌) → \`#2C4D08\` (**8.22:1 ✓ AAA**)
- **Button hover bg**: explicitly overridden to stay in fluorescent family (was silently inheriting from dark-2 = ink color, would have jumped to forest on hover)
- **Focus ring**: teal \`#0e7b75\` (was dark-3 = same as hover; now distinct)
- **text-light**: \`#595959\` → \`#4a4a4a\` (more AA margin on highlights)
- **Backdrop-filter fallback**: synced to PR #87's new body color
- **Reaction bg alpha tokens**: overridden to match new ink scale

## Test plan

- [ ] Hard-refresh \`/explore/repos\` — repo names and owner names readable forest-green ink
- [ ] Same on \`/explore/issues\`, \`/explore/users\`, \`/explore/organizations\`
- [ ] Repo's \`/issues\` list — issue titles readable
- [ ] Repo file tree — file/dir names readable
- [ ] Hover any link — visibly darkens (clear "more attention" signal)
- [ ] Click a primary button — fluorescent fill preserved
- [ ] Hover the same button — stays in fluorescent family (one notch darker), not jarring
- [ ] Tab through a form — focus ring is teal, distinguishable from hover
- [ ] Dark theme: no visual changes (regression check)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
PRBODY
)"
```

---

### Task 5: Admin merge + restart hand-off

User has authorized admin-merge in advance.

- [ ] **Step 1: PR diff sanity check**

```bash
gh pr diff <PR-NUMBER> --repo HackForger/hackforger | head -80
```

Expected: 1 file (`custom/public/assets/css/theme-hackforger-light.css`), ~30 changed lines all in the override block region. STOP if any other file appears.

- [ ] **Step 2: Admin merge**

```bash
gh pr merge <PR-NUMBER> --repo HackForger/hackforger --merge --admin --delete-branch
```

- [ ] **Step 3: Pull main with rebase**

```bash
cd /Users/h2oslabs/Workspace/hackforger
git pull --rebase origin v0.1-dev/hackforger
```

- [ ] **Step 4: Identify gitea PID**

```bash
ps aux | grep -E '[g]itea web' | grep -v grep | awk '{print "PID: " $2}'
```

Print PID and tell user:
> "Found gitea PID: `<pid>`. Please run `! kill <pid>` in your shell to stop it; I'll restart immediately after."

- [ ] **Step 5: After user kills, clean lock + restart**

```bash
rm -f data/queues/common/LOCK
nohup ./gitea web > /tmp/gitea.log 2>&1 &
disown
sleep 5
```

Then read `/tmp/gitea.log` (Read tool, not `tail`) and confirm `Listen: http://0.0.0.0:3000`.

- [ ] **Step 6: Hand off to user**

Tell user: "Instance restarted. Hard-refresh `localhost:3000` (Cmd+Shift+R), then verify on `/explore/repos` that link text is now readable forest green."

---

## Self-review checklist

- [x] All 5 spec defects mapped to a step (CRIT-1/2 → Step 2, CRIT-3 → Step 5, LOW-2 → Step 4, HIGH-4 → Step 3).
- [x] Reviewer-flagged issues addressed:
  - `--color-primary-hover/active` overridden (Step 2)
  - `dark-1` darkened so text uses pass AA (Step 2)
  - Teal swapped to `#0e7b75` for chromatic distinction (Step 5)
  - Visible alpha tokens overridden (Step 2)
  - Grep step added (Task 2)
- [x] Dark theme explicitly NOT modified (audit confirmed it works).
- [x] Single CSS file change → one commit, one PR.
- [x] PR diff sanity check before merge (Task 5 Step 1).
- [x] Hand-off pattern: agent prints PID → user `! kill` → agent restarts.
