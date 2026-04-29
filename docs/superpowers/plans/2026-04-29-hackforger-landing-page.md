# HackForger Landing Page — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Forgejo's default unsigned-visitor splash page with a bespoke landing page (`custom/public/assets/landing/index.html`), per spec `docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md`.

**Architecture:** L0 strategy — drop-in static HTML with minimal Go/JS hooks. Files live under `custom/public/assets/landing/`, served via existing `CustomAssets()` layer. Route-level dispatch in `routers/web/home.go::Home()` reads file from disk via `httpcache.ServeContentWithCacheControl`. JS in HTML wires login button + per-Wave register-button slug routing with "活动还没创建" fallback modal.

**Tech Stack:** Go (1 file modified), HTML/JS (1 file imported & lightly edited), `.gitignore` (whitelist 3 lines), no new dependencies, no template forks, no DB changes.

---

### Task 1: Pre-flight verification

**Files:**
- Worktree dir: `/Users/h2oslabs/Workspace/hackforger/.claude/worktrees/login-page` (current)
- Branch: `v0.1-dev/hackforger` (current — will need to branch off for the PR)
- Source assets: `/tmp/league-project/` (already extracted)

- [ ] **Step 1: Confirm worktree state**

```bash
cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/login-page
git status --short
git log --oneline -3
```

Expected: clean working tree (besides spec file `docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md` and `league-project.zip` untracked); HEAD ≈ `5bb7715125` (PR #105 merge) or later.

- [ ] **Step 2: Create feature branch**

```bash
git checkout -b feat/issue-XX-landing-page
```

(replace `issue-XX` with the actual issue number once #KPI/#STAGES/#TMPL are filed; or use `feat/landing-page-l0` if no issue ref needed for the parent PR)

- [ ] **Step 3: Copy app.ini for local server testing**

```bash
test -f custom/conf/app.ini || cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini
ls -la custom/conf/app.ini
```

(per CLAUDE.md: worktree needs app.ini before any `gitea web`)

- [ ] **Step 4: Verify source assets are present**

```bash
ls /tmp/league-project/public/league/ | head -5
test -f /tmp/league-project/public/league/index.html && echo "OK: index.html present" || echo "FAIL"
ls /tmp/league-project/public/league/assets/images/ | wc -l
```

Expected: `index.html` + `assets/` directory; ~50 images in `assets/images/`.

---

### Task 2: Copy landing page assets into custom/public/assets/landing/

**Files:**
- Source: `/tmp/league-project/public/league/`
- Target: `custom/public/assets/landing/`

- [ ] **Step 1: Create target directory and copy files**

```bash
mkdir -p custom/public/assets/landing
cp -R /tmp/league-project/public/league/. custom/public/assets/landing/
ls custom/public/assets/landing/
```

Expected: `index.html` + `assets/` in target.

- [ ] **Step 2: Verify file count and total size**

```bash
find custom/public/assets/landing -type f | wc -l
du -sh custom/public/assets/landing
```

Expected: ~50-60 files, ~21MB total.

- [ ] **Step 3: Sanity-check encoding (Chinese filenames may have garbled in zip extraction)**

```bash
ls custom/public/assets/landing/assets/images/ | grep -E "^\?|webp$|svg$" | head -10
```

If filenames show literal `?`, the zip extraction lost UTF-8 — re-extract with `unzip -O UTF-8`. Should match references inside `index.html` (run `grep -oE 'assets/images/[^"]+' custom/public/assets/landing/index.html | sort -u` to compare).

---

### Task 3: Edit landing HTML — base href, login button, dead modal removal

**File:** `custom/public/assets/landing/index.html`

> **Important:** Each Edit call below uses **unique substring anchors** in `old_string`, not line numbers. Line numbers are approximate hints only — the file shifts as edits progress. Always grep before each edit to confirm the anchor is unique.

- [ ] **Step 0: Add `<base href>` for relative asset path resolution**

The HTML uses relative paths like `<img src="assets/images/X.webp">`. When served at `/` they resolve to `/assets/images/X.webp` (404). Adding `<base href="/assets/landing/">` makes them resolve to `/assets/landing/assets/images/X.webp` (correct).

Find (the very first line of `<head>`, look for `<meta charset` to anchor):
```html
<head>
<meta charset="utf-8"/>
```

Replace with:
```html
<head>
<base href="/assets/landing/">
<meta charset="utf-8"/>
```

Verify:
```bash
grep -n '<base href=' custom/public/assets/landing/index.html
# Expected: exactly 1 match in <head>
grep -nE 'href="[^/h#"]' custom/public/assets/landing/index.html | head -10
# Expected: empty (no internal relative-link anchors that <base> would break)
```

- [ ] **Step 1: Replace top-right "登录" button (around line 724) with anchor link**

Find:
```html
<button type="button" id="btn-open-login" class="inline-flex items-center justify-center rounded-full bg-primary text-black font-bold font-oppo hover:brightness-110 transition-all leading-tight px-3 sm:px-4 h-9 md:h-8" style="font-size:clamp(0.8rem,0.9vw + 0.55rem,0.95rem);">登录</button>
```

Replace with:
```html
<a href="/user/login" class="inline-flex items-center justify-center rounded-full bg-primary text-black font-bold font-oppo hover:brightness-110 transition-all leading-tight px-3 sm:px-4 h-9 md:h-8" style="font-size:clamp(0.8rem,0.9vw + 0.55rem,0.95rem);">登录</a>
```

- [ ] **Step 2: Delete login modal block (line 749 to its closing `</div>`)**

Locate the `<!-- 登录弹层 -->` comment near line 749 and delete the entire `<div id="login-modal" ...>...</div>` element including the `<script>` that operates on `#login-modal`, `#login-form`, etc.

After deletion, search to confirm no orphans:
```bash
grep -n "login-modal\|btn-open-login\|login-form\|login-cancel\|login-submit" custom/public/assets/landing/index.html
```

Expected: no matches.

- [ ] **Step 3: Delete already-logged-in user menu block (around line 726-745)**

Locate `<!-- 已登录 -->` comment (anchor: comment string is unique). Delete the entire enclosing block: `<button id="auth-avatar-btn" ...>` + its sibling menu div + the related `<script>` that handles avatar/logout. Use the `<!-- 已登录 -->` comment and the next `<!-- ... -->` comment (or section boundary) as start/end anchors.

Confirm:
```bash
grep -n "auth-avatar-btn\|auth-logout-btn\|auth-menu-tip" custom/public/assets/landing/index.html
```

Expected: no matches.

- [ ] **Step 4: Verify Hero "立即报名" buttons unchanged**

```bash
grep -n "opc-stage-register" custom/public/assets/landing/index.html | head -5
```

Expected: 3 onclick handlers + 1 div id (4 lines), all referencing `#opc-stage-register` for scroll behavior. Don't modify these.

---

### Task 4: Edit landing HTML — add HACKFORGER_LANDING_CONFIG and handleRegisterClick

**File:** `custom/public/assets/landing/index.html`

- [ ] **Step 1: Add config script block in `<head>`**

Locate the closing `</head>` tag. Just before it, insert:

```html
<script id="hackforger-landing-config">
window.HACKFORGER_LANDING_CONFIG = {
  stages: {
    's1-w1': { slug: 'tbd', enabled: true  },
    's1-w2': { slug: 'tbd', enabled: false },
    's1-w3': { slug: 'tbd', enabled: false },
    's1-w4': { slug: 'tbd', enabled: false },
    's2-w1': { slug: 'tbd', enabled: false },
    's2-w2': { slug: 'tbd', enabled: false },
    's2-w3': { slug: 'tbd', enabled: false },
    's2-w4': { slug: 'tbd', enabled: false },
    's3-w1': { slug: 'tbd', enabled: false },
    's3-w2': { slug: 'tbd', enabled: false },
    's3-w3': { slug: 'tbd', enabled: false },
    's3-w4': { slug: 'tbd', enabled: false }
  }
};
</script>
```

- [ ] **Step 2: Add behavior script with event delegation in `<head>`**

Place the behavior script in `<head>` (immediately after the config block from step 1) so it's available before any button is parsed. Use **event delegation** — single listener on `document` matching `[data-stage]` clicks. This eliminates inline `onclick=` (cleaner CSP, no race window between parse and script load).

Insert just after the `</script>` closing tag of `hackforger-landing-config`:

```html
<script id="hackforger-landing-behavior">
(function() {
  function showLandingInfoModal(title, body) {
    var modal = document.getElementById('info-modal');
    if (!modal) {
      alert(title + '\n\n' + body);
      return;
    }
    var titleEl = modal.querySelector('[data-info-title], #info-modal-title, h3');
    var bodyEl  = modal.querySelector('[data-info-body], #info-modal-body, .info-md-body');
    if (titleEl) titleEl.textContent = title;
    if (bodyEl)  bodyEl.textContent  = body;
    modal.classList.remove('hidden');
    modal.setAttribute('aria-hidden', 'false');
  }

  function handleRegisterClick(stageId) {
    try {
      var cfg = window.HACKFORGER_LANDING_CONFIG &&
                window.HACKFORGER_LANDING_CONFIG.stages &&
                window.HACKFORGER_LANDING_CONFIG.stages[stageId];
      if (!cfg || !cfg.enabled) return;

      if (!cfg.slug || cfg.slug === 'tbd') {
        showLandingInfoModal('活动还没创建', '本届赛事正在筹备中，请稍后再试');
        return;
      }

      var url = '/hackathon/' + encodeURIComponent(cfg.slug);
      // HEAD 校验 slug 真实性。仅 404 视为"不存在"；
      // 其他状态（200/302→follow→200/401/403）一律视为"赛事存在"并跳转。
      fetch(url, { method: 'HEAD' })
        .then(function(r) {
          if (r.status === 404) {
            showLandingInfoModal('活动还没创建', '本届赛事正在筹备中，请稍后再试');
          } else if (r.status >= 500) {
            showLandingInfoModal('服务暂不可用', '请稍后重试');
          } else {
            window.location.href = url;
          }
        })
        .catch(function() {
          showLandingInfoModal('网络错误', '无法连接到服务器，请稍后重试');
        });
    } catch (e) {
      showLandingInfoModal('活动还没创建', '本届赛事正在筹备中，请稍后再试');
      if (window.console) console.error('[hackforger-landing] handleRegisterClick error:', e);
    }
  }

  // Event delegation：统一捕获 [data-stage] 按钮点击
  document.addEventListener('click', function(e) {
    var btn = e.target.closest('[data-stage]');
    if (!btn) return;
    if (btn.disabled || btn.getAttribute('aria-disabled') === 'true') return;
    var stageId = btn.getAttribute('data-stage');
    if (stageId) handleRegisterClick(stageId);
  });
})();
</script>
```

**Note**: 由于使用 event delegation，按钮**不再需要 inline `onclick`**——只需 `data-stage` 属性。Task 5 据此简化。

- [ ] **Step 3: Inspect `#info-modal` structure to confirm selector compatibility**

```bash
grep -n "info-modal" custom/public/assets/landing/index.html | head -5
```

If the modal uses different attribute names than `[data-info-title]`/`[data-info-body]`, adjust the selectors in `showLandingInfoModal` to match. The fallback selectors (`#info-modal-title`, `h3`, `.info-md-body`) cover common patterns; confirm at least one matches.

---

### Task 5: Wire 12 Wave "立即报名" buttons with data-stage attributes

**File:** `custom/public/assets/landing/index.html`

> **Important:** Line numbers shift after every previous Edit. Use grep-anchors (image filename references like `S1-1.webp` are uniquely associated with each Wave card and don't change).

- [ ] **Step 1: Locate all 12 Wave buttons via image-filename anchors**

Each Wave card has a unique image src reference. Use them as anchors:

```bash
for stage in S1-1 S1-2 S1-3 S1-4 S2-1 S2-2 S2-3 S2-4 S3-1 S3-2 S3-3 S3-4; do
  grep -n "${stage}.webp" custom/public/assets/landing/index.html | head -1
done
```

Expected: 12 lines, each pointing to one Wave card. The button is ~10-15 lines after each img reference.

- [ ] **Step 2: Add `data-stage` to S1 W1 active button (anchor: just after S1-1.webp img)**

Find (use `S1-1.webp` to locate the surrounding card; the button is the active-style one):
```html
<button class="w-full py-3 bg-primary text-black text-xs md:text-sm font-black rounded-full uppercase tracking-widest hover:scale-[0.98] transition-transform font-oppo mt-auto">立即报名</button>
```

Replace with:
```html
<button data-stage="s1-w1" class="w-full py-3 bg-primary text-black text-xs md:text-sm font-black rounded-full uppercase tracking-widest hover:scale-[0.98] transition-transform font-oppo mt-auto">立即报名</button>
```

(No `onclick` needed — handled by event delegation in Task 4.)

- [ ] **Step 3: Add `data-stage` to 11 disabled buttons**

For each Wave below, locate the button via the `*.webp` filename anchor in the surrounding card. Each button looks like:
```html
<button disabled aria-disabled="true" class="w-full py-3 bg-[#9CA3AF] text-[#4B5563] ...">5.7-5.9报名</button>
```

Add `data-stage="<id>"` immediately after the `disabled` attribute:

| Stage ID | Wave | Filename anchor | Button text label (varies) |
|---|---|---|---|
| s1-w2 | S1 复赛/Wave 2 | `S1-2.webp` | `5.7-5.9报名` |
| s1-w3 | S1 半决赛/Wave 3 | `S1-3.webp` | `5.14-5.16报名` |
| s1-w4 | S1 决赛/Wave 4 | `S1-4.webp` | `6.17-6.18决赛` |
| s2-w1 | S2 初赛/Wave 1 | `S2-1.webp` | (similar pattern) |
| s2-w2 | S2 复赛/Wave 2 | `S2-2.webp` | (similar pattern) |
| s2-w3 | S2 半决赛/Wave 3 | `S2-3.webp` | (similar pattern) |
| s2-w4 | S2 决赛/Wave 4 | `S2-4.webp` | (similar pattern) |
| s3-w1 | S3 初赛/Wave 1 | `S3-1.webp` | (similar pattern) |
| s3-w2 | S3 复赛/Wave 2 | `S3-2.webp` | (similar pattern) |
| s3-w3 | S3 半决赛/Wave 3 | `S3-3.webp` | (similar pattern) |
| s3-w4 | S3 决赛/Wave 4 | `S3-4.webp` | (similar pattern) |

For each, the Edit `old_string` should include enough context (the button + its parent card div) to be unique within the file.

**Trade-off note**: This per-Wave id mapping must stay synchronized between (a) HTML `data-stage` attrs, (b) `HACKFORGER_LANDING_CONFIG.stages` keys, (c) the "按届编辑 Checklist" in spec. If new Waves are added to the design, all three locations need updates.

- [ ] **Step 4: Verify final state**

```bash
grep -c 'data-stage=' custom/public/assets/landing/index.html
```

Expected: `data-stage=` appears exactly 12 times.

```bash
grep -c 'onclick="handleRegisterClick' custom/public/assets/landing/index.html
```

Expected: 0 (we use event delegation, no inline onclick).

---

### Task 6: Update .gitignore to whitelist landing/

**File:** `.gitignore`

- [ ] **Step 1: Add whitelist for landing dir**

Find the existing block (lines ~73-79):
```
!/custom/public/
/custom/public/*
!/custom/public/assets/
/custom/public/assets/*
!/custom/public/assets/img/
!/custom/public/assets/css/
!/custom/public/assets/fonts/
```

Append (still inside the `/custom/public/` block):
```
!/custom/public/assets/landing/
!/custom/public/assets/landing/**
```

- [ ] **Step 2: Verify files become trackable**

```bash
git check-ignore -v custom/public/assets/landing/index.html
```

Expected: NO output (i.e., file is NOT ignored). If git check-ignore prints a rule, the whitelist isn't working — debug ordering.

```bash
git status --short custom/public/assets/landing/ | head -5
```

Expected: untracked (`??`) entries — files are visible to git.

---

### Task 7: Modify routers/web/home.go to serve landing for unsigned visitors

**File:** `routers/web/home.go`

- [ ] **Step 1: Add imports**

In the existing import block (lines 6-26), add to the stdlib group (`os`, `path/filepath`). Note: `net/http` is already imported. We do **not** need to import `httpcache` because we use stdlib `http.ServeContent` directly (the Forgejo wrapper would overwrite our Cache-Control header).

```go
"os"
"path/filepath"
```

Insert in sorted position within the existing stdlib import group (between `net/http` and `strconv`).

- [ ] **Step 2: Insert landing-page serve logic before tplHome render**

Find (around line 83-88):
```go
ctx.Data["PageIsHome"] = true
ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

ctx.Data["OpenGraphDescription"] = setting.UI.Meta.Description

ctx.HTML(http.StatusOK, tplHome)
```

Replace with:
```go
// HackForger: serve custom landing page for unsigned visitors when present.
// Falls back to default Forgejo splash if file is missing.
// We use stdlib http.ServeContent (not httpcache.ServeContentWithCacheControl)
// because the latter calls SetCacheControlInHeader which would overwrite the
// "private, no-store" directive we explicitly set below.
landingPath := filepath.Join(setting.CustomPath, "public", "landing", "index.html")
if f, err := os.Open(landingPath); err == nil {
    defer f.Close()
    if fi, err := f.Stat(); err == nil && !fi.IsDir() {
        ctx.Resp.Header().Set("Cache-Control", "private, no-store")
        ctx.Resp.Header().Set("Vary", "Cookie")
        http.ServeContent(ctx.Resp, ctx.Req, "index.html", fi.ModTime(), f)
        return
    }
}

ctx.Data["PageIsHome"] = true
ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

ctx.Data["OpenGraphDescription"] = setting.UI.Meta.Description

ctx.HTML(http.StatusOK, tplHome)
```

- [ ] **Step 3: Verify build compiles**

```bash
go vet ./routers/web/...
go build ./routers/web/
```

Expected: no errors.

- [ ] **Step 4: Verify build compiles**

```bash
go vet ./routers/web/...
```

Expected: no output (or pre-existing warnings only).

---

### Task 8: Build backend + frontend

- [ ] **Step 1: Build frontend (required first in worktree per CLAUDE.md)**

```bash
make frontend 2>&1 | tail -20
```

Expected: completes without errors. Generates `public/assets/` JS/CSS bundles.

> **Why required**: even though `custom/public/assets/landing/` is runtime-loaded and not bundled, the **backend bindata** tag embeds `public/assets/*` (built outputs, gitignored). Worktrees don't carry these — must rebuild before `make backend` or the binary will be incomplete.

- [ ] **Step 2: Build backend with bindata**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -10
```

Expected: produces `./gitea` binary.

- [ ] **Step 3: Verify binary**

```bash
ls -la gitea
./gitea --version
```

Expected: file exists, version output.

---

### Task 9: Run dev server + Phase 1 manual E2E

**Server**: `localhost:3000` via worktree's own gitea binary (not via `scripts/restart-gitea.sh` — that script is for the main repo).

- [ ] **Step 1: Start server**

```bash
rm -f data/queues/common/LOCK
./gitea web > /tmp/gitea-landing.log 2>&1 &
disown
sleep 3
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:3000/
```

Expected: `200`. Logs at `/tmp/gitea-landing.log`.

- [ ] **Step 2: Run E2E tests via agent-browser**

Invoke an agent (or do it manually) per the Phase 1 test plan in spec section "测试 → Phase 1". Tests 1-17:

| # | Description | Pass criteria |
|---|---|---|
| 1 | Unsigned visit `/` shows landing | Hero text visible, NOT "easy-to-set-up" |
| 2 | Signed visit `/` shows Dashboard | User dashboard visible |
| 3 | "登录" button navigation | URL becomes `/user/login` |
| 4 | Hero CTA scroll | Page scrolls to #opc-stage-register |
| 5 | slug=tbd modal | Modal title "活动还没创建" |
| 6 | slug=bogus 404 | After config edit + reload, modal shows |
| 7 | Disabled buttons | No reaction on click |
| 8 | Theme toggle | CSS class flip on `<html>` |
| 9 | Hero carousel | Slides advance |
| 10 | Mobile 375px | No horizontal overflow |
| 11 | Logo intact | hackforger logo (not Forgejo default) |
| 12 | Routes regression | /explore, /user/login, /user/sign_up all 200 |
| 13 | Fallback (file missing) | Splash shown after temp move-aside |
| 14 | RememberMe redirect | Stale cookie → /user/login (not landing) |
| 15 | LANDING_PAGE override | app.ini override → /explore (not landing) |
| 16 | Malformed config | JSON syntax error → modal still works (try/catch) |
| 17 | Cache headers | `curl -I /` shows `Cache-Control: private, no-store` + `Vary: Cookie` (ETag/Last-Modified set by stdlib http.ServeContent) |

Save screenshots to `docs/tests/e2e/reports/2026-04-29-landing-page-impl/` and write report at `docs/tests/e2e/reports/2026-04-29-landing-page-impl.md`.

- [ ] **Step 3: Test cleanup — restore any mutations made during testing**

Several tests required temporary mutations. Revert all of them:

```bash
# Test #6 (slug=bogus): revert HACKFORGER_LANDING_CONFIG s1-w1.slug back to 'tbd'
grep -n "slug:" custom/public/assets/landing/index.html | head -3   # confirm only 'tbd' values

# Test #13 (file missing): restore index.html if it was moved aside
test -f custom/public/assets/landing/index.html || mv /tmp/landing-index-backup.html custom/public/assets/landing/index.html

# Test #15 (LANDING_PAGE override): revert app.ini
grep -n "LANDING_PAGE" custom/conf/app.ini   # should be absent or commented out

# Test #16 (malformed config): revert any deliberate JSON syntax errors
node -e "var s=require('fs').readFileSync('custom/public/assets/landing/index.html','utf8');var m=s.match(/HACKFORGER_LANDING_CONFIG = (\{[\s\S]*?\});/);eval('(' + m[1] + ')');console.log('config parses OK')"

# Confirm git working tree only has expected changes
git diff --stat
```

Expected `git diff --stat`: only `routers/web/home.go`, `.gitignore`, `custom/public/assets/landing/*`, and docs. No app.ini changes.

- [ ] **Step 4: Stop server**

```bash
pkill -f "$(pwd)/gitea web" || true
```

(or `kill <PID>` from log)

---

### Task 9.5: Rollback recipe (if Phase 1 testing fails)

If E2E tests reveal blockers and you need a clean rollback to retry:

```bash
# Discard all working changes (DESTRUCTIVE — confirms first)
git status
# Review the list, then:
git checkout -- routers/web/home.go .gitignore custom/conf/app.ini
rm -rf custom/public/assets/landing/

# Verify restoration
git status   # only docs/ and untracked files should remain
```

Spec / plan / issue-body docs in `docs/` are kept (they're the design artifacts).

After rollback, re-read the failing test's logs (`/tmp/gitea-landing.log` and the agent-browser screenshots), update the spec/plan with the discovered constraint, and re-run from the failing task.

---

### Task 10: Prepare issue body files

**Files:**
- `docs/landing-page/issue-1-kpi-stats.md`
- `docs/landing-page/issue-2-stages-sync.md`
- `docs/landing-page/issue-3-templatization.md`

- [ ] **Step 1: Create directory and 3 issue body markdown files**

Use the bodies prepared in the brainstorming session (see chat transcript). These will be referenced from the spec and used by `gh issue create -F <file>` after PR merge.

Each file is the raw issue body (no title — title goes on `gh issue create` flag). Once PR is merged and issue numbers are assigned, the spec's "关联 Issue" section can be updated to reference real issue numbers.

- [ ] **Step 2: Verify files**

```bash
ls -la docs/landing-page/
wc -l docs/landing-page/*.md
```

Expected: 3 files, each 30-60 lines.

---

### Task 11: Commit and push

- [ ] **Step 1: Stage all changes**

```bash
git add custom/public/assets/landing/ \
        routers/web/home.go \
        .gitignore \
        docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md \
        docs/superpowers/plans/2026-04-29-hackforger-landing-page.md \
        docs/landing-page/
git status
```

Expected: ~55-60 files staged (most under `custom/public/assets/landing/assets/images/`).

- [ ] **Step 2: Verify no upstream Forgejo files leaked**

```bash
git diff --cached --name-only | grep -v -E "^(custom/|docs/|\.gitignore|routers/web/home\.go)" | head -20
```

Expected: empty. If anything else appears (e.g., accidental edits to upstream templates), unstage and revert before committing.

- [ ] **Step 3: Create commit**

```bash
git commit -m "$(cat <<'EOF'
feat(landing): replace splash with HackForger marketing landing page

Adds a bespoke unsigned-visitor landing page at custom/public/assets/landing/
served by routers/web/home.go::Home() when no LandingPageURL override
or RememberMe cookie redirects first. Falls back to the original
Forgejo splash template if the custom file is missing.

The page is fully self-contained (Tailwind CDN, Google Fonts, marked,
qrcode), with two integration points wired into HackForger:
- Top-right "登录" button → /user/login (anchor link, no JS modal)
- 12 per-Wave "立即报名" buttons → /hackathon/{slug} via
  HACKFORGER_LANDING_CONFIG, with "活动还没创建" fallback modal
  when slug is unset (tbd) or hackathon doesn't exist (HEAD 404)

Cache-Control: private, no-store + Vary: Cookie set explicitly to
prevent shared-cache poisoning of authenticated users with the
anonymous landing variant.

Spec:  docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md
Plan:  docs/superpowers/plans/2026-04-29-hackforger-landing-page.md

Follow-up issues (drafts in docs/landing-page/):
- issue-1: KPI stats data-source definition + connect
- issue-2: Auto-sync 赛项&命题预览 from hackathon DB
- issue-3: (Backlog) Template-driven landing page generator

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 4: Push and open PR**

```bash
git push -u origin feat/issue-XX-landing-page
gh pr create --title "feat(landing): replace splash with HackForger marketing landing page" --body "$(cat <<'EOF'
## Summary
- Replaces Forgejo's "easy-to-set-up Git service" splash with HackForger's bespoke marketing landing page for unsigned visitors
- L0 strategy: drop-in static HTML in custom/public/assets/landing/, served via existing CustomAssets() layer + a small route hook in Home()
- Falls back to original splash if the custom file is missing (deploy safety net)
- 12 per-Wave "立即报名" buttons wired with slug routing + "活动还没创建" fallback modal

## Design + Plan
- Spec: docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md
- Plan: docs/superpowers/plans/2026-04-29-hackforger-landing-page.md

## Follow-up (post-merge)
File these issues (draft bodies committed under docs/landing-page/):
- KPI stats data semantics + API
- Auto-sync 赛项&命题预览 from hackathon DB
- (Backlog) Template-driven landing generator

## Test plan
- [ ] Phase 1 (impl, no hackathon dependency): 17 cases at docs/tests/e2e/reports/2026-04-29-landing-page-impl.md
- [ ] Phase 2 (QA, post-merge): tester creates real hackathon per Issue #STAGES, runs full register flow

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

### Task 12: After merge — create follow-up issues

(Out of scope for this plan; handled separately after PR merge once issue numbers can reference the merged commit.)

```bash
gh issue create --repo HackForger/hackforger \
  --title "[landing] Define & connect KPI stats on home landing page (算力消耗/总计发放/正在进行)" \
  --label "enhancement,documentation" \
  -F docs/landing-page/issue-1-kpi-stats.md

gh issue create --repo HackForger/hackforger \
  --title "[landing] Auto-sync 「赛项&命题预览」section from hackathon DB" \
  --label "enhancement" \
  -F docs/landing-page/issue-2-stages-sync.md

gh issue create --repo HackForger/hackforger \
  --title "[landing] (Backlog) Templatize landing page generation for future events" \
  --label "enhancement" \
  -F docs/landing-page/issue-3-templatization.md
```

After issues are filed, update spec's "关联 Issue" section with real numbers (small follow-up commit).

---

## Critical Risk Checkpoints

These are points where the plan may need to deviate based on real-world discovery:

1. **After Task 2 step 3** (encoding check): if filenames are mojibake, you must re-extract the zip with `unzip -O UTF-8` AND verify HTML's image references match. If they don't match, edit either the HTML refs or rename files.

2. **After Task 4 step 3** (info-modal selectors): if the modal structure doesn't expose `[data-info-title]`/`#info-modal-title`, fall back to the `alert()` path or hand-pick the right selectors. Don't ship without confirming the modal actually shows the right text.

3. **After Task 7 step 3** (httpcache signature): the signature was verified via spec review against `modules/httpcache/httpcache.go:39-42` but build will be the ground truth. If `make backend` fails, adjust the call site.

4. **After Task 9 step 2** (E2E results): if more than 1 test fails, STOP — debug root cause, do not push. The Cache-Control test (#17) and the malformed-config test (#16) are the most likely to surface implementation bugs.

5. **Hackathon slug HEAD redirect behavior**: the slug-existence check assumes 404 is the response for missing slugs. If `ViewHackathon` evolves (e.g., to redirect to `/explore/hackathons` when slug missing), the check would need to switch from `r.status === 404` to `r.status === 404 || (r.redirected && r.url.includes('/explore'))`. Document if discovered during testing.
