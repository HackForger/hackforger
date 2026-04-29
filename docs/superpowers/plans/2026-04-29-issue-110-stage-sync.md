# Issue #110 + #112 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement DB-driven hydration for the HackForger landing page stage cards (Issue #110) plus organization-section content fixes (Issue #112). Per spec `docs/superpowers/specs/2026-04-29-issue-110-stage-sync-design.md`.

**Architecture:** Schema-zero — store `hackforger.landing.league_prefix` in existing `system_setting` table; admin opt-in via single text field; public API `GET /api/v1/hackforger/landing/stages` returns 12-slot map; landing JS fetches and patches DOM (only mutates slots with real `slug`); home.go gates landing page rendering on prefix being set.

**Tech Stack:** Go (model helper + API + admin route + home.go gate + i18n keys), HTML/JS (hydrate hooks + fetch logic + CSS card alignment), no new dependencies, no migration.

**Worktree:** `/Users/h2oslabs/Workspace/hackforger/.claude/worktrees/issue-110` on branch `feat/issue-110-stage-sync` (already created off `ef53e90495` = PR #108 merge).

---

### Task 1: Pre-flight verification

**Files:**
- Worktree: `/Users/h2oslabs/Workspace/hackforger/.claude/worktrees/issue-110` (current)
- Branch: `feat/issue-110-stage-sync` (current)
- Spec: `docs/superpowers/specs/2026-04-29-issue-110-stage-sync-design.md` (already in this worktree)

- [ ] **Step 1: Confirm worktree state**

```bash
cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/issue-110
git status --short
git log --oneline -3
```

Expected: HEAD = `ef53e90495` (PR #108 merge); only spec file untracked.

- [ ] **Step 2: Verify app.ini present**

```bash
ls -la custom/conf/app.ini
```

If missing: `cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini`

- [ ] **Step 3: Verify reviewer's recommendations from spec are addressed**

```bash
grep -c "GetSettingByKey" docs/superpowers/specs/2026-04-29-issue-110-stage-sync-design.md
grep -c "GetCurrentPhase" docs/superpowers/specs/2026-04-29-issue-110-stage-sync-design.md
grep -c "IsValidUsername" docs/superpowers/specs/2026-04-29-issue-110-stage-sync-design.md
grep -c "sync.Map\|landingCache" docs/superpowers/specs/2026-04-29-issue-110-stage-sync-design.md
```

Expected: each ≥ 1 (confirms reviewer's blockers are documented in spec).

---

### Task 2: Add `GetSettingByKey` helper to system_setting

**File:** `models/system/setting.go`

- [ ] **Step 1: Read existing patterns**

```bash
grep -n "func Get\|func Set\|func GetRevision" /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/issue-110/models/system/setting.go
```

Expected: see `GetAllSettings`, `SetSettings`, `GetRevision` patterns. Locate where to insert new function (after `GetRevision`).

- [ ] **Step 2: Add `GetSettingByKey` function**

Insert after `GetRevision` (which uses similar single-row pattern):

```go
// GetSettingByKey returns the value for a single setting key, or empty string if not present.
func GetSettingByKey(ctx context.Context, key string) (string, error) {
    setting := &Setting{}
    has, err := db.GetEngine(ctx).Where("setting_key = ?", key).Get(setting)
    if err != nil {
        return "", err
    }
    if !has {
        return "", nil
    }
    return setting.SettingValue, nil
}
```

- [ ] **Step 3: Verify compiles**

```bash
go vet ./models/system/...
```

Expected: no errors.

---

### Task 3: Modify home.go to gate landing on prefix setting

**File:** `routers/web/home.go`

- [ ] **Step 1: Add import**

Add to existing import block:
```go
system_model "forgejo.org/models/system"
```

- [ ] **Step 2: Modify Home() landing logic**

Locate the existing PR #108 block:
```go
landingPath := filepath.Join(setting.CustomPath, "public", "assets", "landing", "index.html")
if f, err := os.Open(landingPath); err == nil {
    ...
}
```

Wrap with prefix check:
```go
// HackForger: serve custom landing page only if admin has configured league_prefix.
// Without prefix, fall through to Forgejo's default splash.
prefix, _ := system_model.GetSettingByKey(ctx, "hackforger.landing.league_prefix")
if prefix != "" {
    landingPath := filepath.Join(setting.CustomPath, "public", "assets", "landing", "index.html")
    if f, err := os.Open(landingPath); err == nil {
        defer f.Close()
        if fi, err := f.Stat(); err == nil && !fi.IsDir() {
            ctx.Resp.Header().Set("Cache-Control", "private, no-store")
            ctx.Resp.Header().Set("Vary", "Cookie")
            http.ServeContent(ctx.Resp, ctx.Req, "index.html", fi.ModTime(), f)
            return
        }
    }
}
```

- [ ] **Step 3: Verify build**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -3
```

Expected: builds successfully; binary `gitea` updated.

- [ ] **Step 4: Smoke test gate via curl (no full E2E yet)**

```bash
# In separate terminal, start dev server on :3001 with isolated config (similar to PR #108 testing)
# OR use main's app.ini with local override
# For now: just verify build success and code compiles correctly
go vet ./routers/web/...
```

Full smoke test deferred to Task 9.

---

### Task 4: Add public API + cache (services-layer cache)

**Files:**
- `services/hackforger/landing_cache.go` (new) ← **cache lives in services to avoid routers→services→routers circular dep**
- `routers/api/v1/hackforger/landing.go` (new)
- `models/hackforger/hackathon_landing.go` (new helper functions)
- `routers/api/v1/api.go` (register route)

**Architecture note (per reviewer)**: cache must NOT live in routers — services layer needs to call invalidation, and CLAUDE.md mandates routers → services → models direction. So:
- `services/hackforger/landing_cache.go` exports `GetCachedPayload(ctx) ([]byte, error)`, `InvalidateCache()`, holds the sync.RWMutex + struct
- `routers/api/v1/hackforger/landing.go` = thin handler that calls `services/hackforger.GetCachedPayload`
- `services/hackforger/hackathon.go` and `services/hackforger/phase_sync.go` call `InvalidateCache()` directly (same package — no import needed)
- `routers/web/hackforger/admin_landing.go` (Task 5) imports `services/hackforger` for `InvalidateCache()` (routers→services is legal)

- [ ] **Step 1: Create services-layer cache**

New file `services/hackforger/landing_cache.go`:

```go
package hackforger

import (
    "sync"
    "time"

    "context"
    hackforger_model "forgejo.org/models/hackforger"
    system_model "forgejo.org/models/system"
    "forgejo.org/modules/log"
    "forgejo.org/modules/validation"
    "forgejo.org/modules/json"
)

const landingCacheTTL = 5 * time.Minute

// 12 stage slot identifiers — must match landing HTML data-stage attributes.
var landingSlotOrder = []string{
    "s1-w1", "s1-w2", "s1-w3", "s1-w4",
    "s2-w1", "s2-w2", "s2-w3", "s2-w4",
    "s3-w1", "s3-w2", "s3-w3", "s3-w4",
}

var landingCache struct {
    sync.RWMutex
    payload   []byte
    expiresAt time.Time
}

// InvalidateLandingCache clears the in-memory cache.
func InvalidateLandingCache() {
    landingCache.Lock()
    landingCache.expiresAt = time.Time{}
    landingCache.payload = nil
    landingCache.Unlock()
}

// GetLandingPayload returns cached or freshly-built JSON bytes.
func GetLandingPayload(ctx context.Context) ([]byte, error) {
    landingCache.RLock()
    if time.Now().Before(landingCache.expiresAt) && len(landingCache.payload) > 0 {
        b := landingCache.payload
        landingCache.RUnlock()
        return b, nil
    }
    landingCache.RUnlock()

    payload, err := buildLandingPayload(ctx)
    if err != nil {
        return nil, err
    }
    landingCache.Lock()
    landingCache.payload = payload
    landingCache.expiresAt = time.Now().Add(landingCacheTTL)
    landingCache.Unlock()
    return payload, nil
}

func buildLandingPayload(ctx context.Context) ([]byte, error) {
    prefix, _ := system_model.GetSettingByKey(ctx, "hackforger.landing.league_prefix")
    response := map[string]interface{}{
        "league_prefix": prefix,
        "stages":        map[string]interface{}{},
        "fetched_at":    time.Now().UTC().Format(time.RFC3339),
    }
    stages := response["stages"].(map[string]interface{})
    for _, slot := range landingSlotOrder {
        stages[slot] = map[string]interface{}{"slug": nil, "enabled": false}
    }

    if prefix == "" || !validation.IsValidUsername(prefix+"-s1-w1") {
        if prefix != "" {
            log.Warn("Invalid landing prefix in DB: %q", prefix)
        }
        return json.Marshal(response)
    }

    expectedSlugs := make([]string, 0, len(landingSlotOrder))
    slugToSlot := map[string]string{}
    for _, slot := range landingSlotOrder {
        s := prefix + "-" + slot
        expectedSlugs = append(expectedSlugs, s)
        slugToSlot[s] = slot
    }

    hackathons, err := hackforger_model.ListPublishedHackathonsBySlugs(ctx, expectedSlugs)
    if err != nil {
        return nil, err
    }

    ids := make([]int64, 0, len(hackathons))
    for _, h := range hackathons {
        ids = append(ids, h.ID)
    }
    tracksByHackathon, err := hackforger_model.ListTracksByHackathonIDs(ctx, ids)
    if err != nil {
        return nil, err
    }

    for _, h := range hackathons {
        slot := slugToSlot[h.Slug]
        if slot == "" {
            continue
        }
        // GetCurrentPhase already eagerly loads phase.PhaseType — no separate GetPhaseTypeByID call needed
        phase, _ := hackforger_model.GetCurrentPhase(ctx, "hackathon", h.ID)

        slotData := map[string]interface{}{
            "slug":    h.Slug,
            "name":    h.Name,
            "enabled": determineEnabled(h, phase),
        }
        if phase != nil && phase.PhaseType != nil {
            slotData["current_phase"] = map[string]interface{}{
                "key":               phase.PhaseType.Key,
                "display_name_i18n": phase.PhaseType.DisplayNameI18n,
                "ends_at":           time.Unix(phase.EndTime, 0).UTC().Format(time.RFC3339),
            }
        }
        // registration_window: query the registration phase specifically (separate from current_phase)
        regPhase := findPhaseByKey(ctx, h.ID, "registration")
        if regPhase != nil {
            slotData["registration_window"] = map[string]string{
                "start": time.Unix(regPhase.StartTime, 0).UTC().Format(time.RFC3339),
                "end":   time.Unix(regPhase.EndTime, 0).UTC().Format(time.RFC3339),
            }
        }

        names := []string{}
        if tracks, ok := tracksByHackathon[h.ID]; ok {
            for _, t := range tracks {
                names = append(names, t.Name)
            }
        }
        slotData["tracks"] = names

        stages[slot] = slotData
    }

    return json.Marshal(response)
}

func determineEnabled(h *hackforger_model.Hackathon, phase *hackforger_model.Phase) bool {
    if phase != nil && phase.PhaseType != nil && phase.PhaseType.Key == "registration" {
        return true
    }
    return h.StatusCache == hackforger_model.HackathonStatusOpen
}

func findPhaseByKey(ctx context.Context, hackathonID int64, key string) *hackforger_model.Phase {
    phases, err := hackforger_model.GetPhasesByActivity(ctx, "hackathon", hackathonID)
    if err != nil {
        return nil
    }
    for _, p := range phases {
        if p.PhaseType != nil && p.PhaseType.Key == key {
            return p
        }
    }
    return nil
}
```

- [ ] **Step 2: Add helper model functions**

New file `models/hackforger/hackathon_landing.go`:

```go
package hackforger

import (
    "context"
    "forgejo.org/models/db"
)

// ListPublishedHackathonsBySlugs returns hackathons matching the given slug list (only published).
func ListPublishedHackathonsBySlugs(ctx context.Context, slugs []string) ([]*Hackathon, error) {
    if len(slugs) == 0 {
        return []*Hackathon{}, nil
    }
    hackathons := []*Hackathon{}
    err := db.GetEngine(ctx).In("slug", slugs).Where("is_published = ?", true).Find(&hackathons)
    return hackathons, err
}

// ListTracksByHackathonIDs returns tracks grouped by hackathon ID.
func ListTracksByHackathonIDs(ctx context.Context, ids []int64) (map[int64][]*HackathonTrack, error) {
    if len(ids) == 0 {
        return map[int64][]*HackathonTrack{}, nil
    }
    tracks := []*HackathonTrack{}
    if err := db.GetEngine(ctx).In("hackathon_id", ids).Find(&tracks); err != nil {
        return nil, err
    }
    result := make(map[int64][]*HackathonTrack)
    for _, t := range tracks {
        result[t.HackathonID] = append(result[t.HackathonID], t)
    }
    return result, nil
}
```

- [ ] **Step 3: Create thin API handler**

New file `routers/api/v1/hackforger/landing.go`:

```go
package hackforger

import (
    "net/http"

    hackforger_service "forgejo.org/services/hackforger"
    "forgejo.org/services/context"
)

// GetLandingStagesAPI serves the landing page stage data (public, no auth).
// GET /api/v1/hackforger/landing/stages
func GetLandingStagesAPI(ctx *context.APIContext) {
    payload, err := hackforger_service.GetLandingPayload(ctx)
    if err != nil {
        ctx.Error(http.StatusInternalServerError, "GetLandingPayload", err)
        return
    }
    ctx.Resp.Header().Set("Content-Type", "application/json")
    ctx.Resp.Header().Set("Cache-Control", "public, max-age=60")
    ctx.Resp.Write(payload)
}
```

- [ ] **Step 4: Register API route**

In `routers/api/v1/api.go`, find the existing public hackforger routes (search `m.Get("/hackathons", hackforger_api.ListHackathons)`) and add the new endpoint **in the same un-auth public group**:

```go
m.Get("/landing/stages", hackforger_api.GetLandingStagesAPI)
```

- [ ] **Step 5: Wire cache invalidation hooks at specific functions**

Verified function locations (from reviewer's investigation):
- `services/hackforger/hackathon.go` `PublishHackathon` (around line 420)
- `services/hackforger/hackathon.go` `CancelHackathon` (around line 570)  
- `services/hackforger/hackathon.go` `SetIsPublished` callers (search for `SetIsPublished(false)` — unpublish path)
- `services/hackforger/phase_sync.go::SyncStatusCache` after the `if newStatus == oldStatus` early-return (around line 31)

Add a single line after the success path of each:
```go
InvalidateLandingCache()  // same package, no import needed
```

If a `DeleteHackathon` service function exists, hook there too. If hackathon delete is models-only, document that delete-during-active-binding leaves the slot empty until cache TTL expires (max 5 min staleness — acceptable).

- [ ] **Step 6: Verify compiles**

```bash
go vet ./services/hackforger/...
go vet ./routers/api/v1/hackforger/...
go vet ./models/hackforger/...
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -3
```

---

### Task 5: Add admin UI for league_prefix setting

**Files:**
- `routers/web/hackforger/admin_landing.go` (new)
- `templates/hackforger/admin/landing_config.tmpl` (new)
- `routers/web/web.go` (add routes)
- `options/locale/locale_en-US.ini` + `locale_zh-CN.ini` (i18n keys)

**Note**: Task 5 placed AFTER Task 4 (was reordered) so admin UI can import `hackforger_service.InvalidateLandingCache` (already exists by Task 5 start).

- [ ] **Step 1: Create admin handler**

New file `routers/web/hackforger/admin_landing.go`:

```go
package hackforger

import (
    "net/http"
    "strings"

    system_model "forgejo.org/models/system"
    "forgejo.org/modules/base"
    "forgejo.org/modules/setting"
    "forgejo.org/modules/validation"
    hackforger_service "forgejo.org/services/hackforger"
    "forgejo.org/services/context"
)

const tplAdminLandingConfig base.TplName = "hackforger/admin/landing_config"

// AdminLandingConfig renders the admin form.
func AdminLandingConfig(ctx *context.Context) {
    prefix, _ := system_model.GetSettingByKey(ctx, "hackforger.landing.league_prefix")
    ctx.Data["LandingLeaguePrefix"] = prefix
    ctx.Data["PageIsAdminHackforger"] = true
    ctx.HTML(http.StatusOK, tplAdminLandingConfig)
}

// AdminLandingConfigPost handles the form submission.
func AdminLandingConfigPost(ctx *context.Context) {
    prefix := strings.TrimSpace(ctx.FormString("league_prefix"))
    
    // Empty = unset (allowed; disables landing page)
    if prefix != "" {
        // Validate by attempting Forgejo's username check on synthesized slug
        if !validation.IsValidUsername(prefix + "-s1-w1") {
            ctx.Flash.Error(ctx.Tr("hackforger.landing.invalid_prefix"))
            ctx.Redirect(setting.AppSubURL + "/-/admin/hackforger/landing-config")
            return
        }
    }
    
    if err := system_model.SetSettings(ctx, map[string]string{
        "hackforger.landing.league_prefix": prefix,
    }); err != nil {
        ctx.ServerError("SetSettings", err)
        return
    }
    
    hackforger_service.InvalidateLandingCache()  // services package, imported above
    ctx.Flash.Success(ctx.Tr("hackforger.landing.saved"))
    ctx.Redirect(setting.AppSubURL + "/-/admin/hackforger/landing-config")
}
```

- [ ] **Step 2: Create template**

New file `templates/hackforger/admin/landing_config.tmpl`:

```html
{{template "base/head" .}}
<div class="page-content admin">
    {{template "admin/navbar" .}}
    <div class="ui container">
        <h4 class="ui top attached header">
            {{ctx.Locale.Tr "hackforger.landing.config.title"}}
        </h4>
        <div class="ui attached segment">
            <p>{{ctx.Locale.Tr "hackforger.landing.config.description"}}</p>
            
            {{template "base/alert" .}}
            
            <form method="POST" action="{{AppSubUrl}}/-/admin/hackforger/landing-config">
                <div class="field">
                    <label for="league_prefix">{{ctx.Locale.Tr "hackforger.landing.config.prefix_label"}}</label>
                    <input id="league_prefix" name="league_prefix" type="text" 
                           value="{{.LandingLeaguePrefix}}"
                           placeholder="league-2"
                           pattern="^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^$">
                    <p class="help">{{ctx.Locale.Tr "hackforger.landing.config.prefix_help"}}</p>
                </div>
                <button type="submit" class="ui primary button">
                    {{ctx.Locale.Tr "save"}}
                </button>
            </form>
        </div>
    </div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 3: Register routes**

In `routers/web/web.go`, find the existing HackForger admin block (around `m.Group("/hackforger/phase-types", ...)`) and add:

```go
// ***** START: HackForger Admin Landing Config *****
m.Group("/hackforger/landing-config", func() {
    m.Get("", hackforger_web.AdminLandingConfig)
    m.Post("", hackforger_web.AdminLandingConfigPost)
})
// ***** END: HackForger Admin Landing Config *****
```

Insert inside the existing `m.Group("/admin", func() { ... }, adminReq, ...)` block.

- [ ] **Step 4: Add i18n keys to both locale files**

In `options/locale/locale_en-US.ini` under `[hackforger]` section, add:
```ini
landing.config.title = Landing Page Configuration
landing.config.description = Configure the active league prefix. Hackathons matching <prefix>-s{1-3}-w{1-4} will be auto-bound to the landing page stage cards.
landing.config.prefix_label = Active League Prefix
landing.config.prefix_help = Leave empty to disable HackForger landing page (visitors see the default Forgejo home page). Example: league-2
landing.invalid_prefix = Invalid prefix. Must form valid Forgejo organization names when combined with stage suffixes.
landing.saved = Landing config saved.
```

In `options/locale/locale_zh-CN.ini` under `[hackforger]` section:
```ini
landing.config.title = 落地页配置
landing.config.description = 配置当前赛事的 league prefix。slug 符合 <prefix>-s{1-3}-w{1-4} 模式的 hackathon 将自动绑定到落地页的赛段卡片。
landing.config.prefix_label = 当前赛事 prefix
landing.config.prefix_help = 留空则不启用 HackForger 落地页（访客看到默认 Forgejo 首页）。示例：league-2
landing.invalid_prefix = 无效的 prefix。与赛段后缀组合后必须能形成合法的 Forgejo 组织名。
landing.saved = 落地页配置已保存。
```

- [ ] **Step 5: Verify build**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -3
```

---


---

### Task 6: HTML hydrate hooks + JS hydration

**File:** `custom/public/assets/landing/index.html`

- [ ] **Step 1: Add `data-hydrate-card` attribute to each of 12 stage cards**

Find each `<div class="...tone-card...">` containing a `data-stage="X"` button, and add `data-hydrate-card="X"` to the parent card div.

Use grep to locate:
```bash
grep -n 'data-stage=' custom/public/assets/landing/index.html
```

For each match, navigate up to the enclosing card div and add the attribute.

- [ ] **Step 2: Add `data-hydrate="name"` to Wave name `<h5>`**

Each card has a `<h5 class="text-lg font-bold ...">初赛/Wave 1</h5>` (or similar). Add `data-hydrate="name"`.

- [ ] **Step 3: Add `data-hydrate-badge="<key>"` to each badge in each card**

For each badge in `<div class="...flex...gap-1.5">` containing `<span>...</span>`, add a `data-hydrate-badge` attribute identifying its purpose:
- `data-hydrate-badge="static-tag"` for "官方命题"
- `data-hydrate-badge="window-registration"` for "#报名 ..."
- `data-hydrate-badge="window-development"` for "#开发 ..."
- `data-hydrate-badge="window-peer-review"` for "#互评 ..."
- `data-hydrate-badge="window-results"` for "#公布晋级 ..."

- [ ] **Step 4: Add hydration script**

Append a new `<script id="hackforger-landing-hydrate">` after the existing `hackforger-landing-behavior` script in `<head>`:

```html
<script id="hackforger-landing-hydrate">
(function() {
  // i18n: API returns phase i18n keys (e.g. "hackforger.phase.registration");
  // we map them to display strings via this client-side dictionary.
  // Intentionally simple — no Forgejo locale loader on this static page.
  // To support more locales, extend this map; default is Chinese.
  var PHASE_LABELS = {
    'hackforger.phase.registration':  '#报名',
    'hackforger.phase.development':   '#开发',
    'hackforger.phase.peer_review':   '#互评',
    'hackforger.phase.results':       '#公布晋级',
    'hackforger.phase.judging':       '#评审',
    'hackforger.phase.finished':      '#已结束'
  };
  
  function phaseLabel(i18nKey) {
    return PHASE_LABELS[i18nKey] || ('#' + i18nKey.split('.').pop());
  }
  
  function formatRange(window) {
    if (!window || !window.start || !window.end) return '';
    var s = new Date(window.start), e = new Date(window.end);
    return (s.getMonth()+1) + '.' + s.getDate() + '-' + (e.getMonth()+1) + '.' + e.getDate();
  }
  
  function hydrateCard(slot, data) {
    var card = document.querySelector('[data-hydrate-card="' + slot + '"]');
    if (!card) return;
    
    if (data.name) {
      var nameEl = card.querySelector('[data-hydrate="name"]');
      if (nameEl) nameEl.textContent = data.name;
    }
    
    if (data.registration_window) {
      var winEl = card.querySelector('[data-hydrate-badge="window-registration"]');
      if (winEl) winEl.textContent = '#报名 ' + formatRange(data.registration_window);
    }
    
    // current phase highlight
    if (data.current_phase && data.current_phase.display_name_i18n) {
      var phaseEl = card.querySelector('[data-hydrate-badge="current-phase"]');
      if (phaseEl) phaseEl.textContent = phaseLabel(data.current_phase.display_name_i18n);
    }
    
    var btn = card.querySelector('[data-stage]');
    if (btn) {
      if (data.enabled) {
        btn.removeAttribute('disabled');
        btn.removeAttribute('aria-disabled');
        btn.classList.remove('cursor-not-allowed', 'bg-[#9CA3AF]', 'text-[#4B5563]');
        btn.classList.add('bg-primary', 'text-black');
      } else {
        btn.setAttribute('disabled', '');
        btn.setAttribute('aria-disabled', 'true');
        btn.classList.add('cursor-not-allowed', 'bg-[#9CA3AF]', 'text-[#4B5563]');
        btn.classList.remove('bg-primary', 'text-black');
      }
    }
  }
  
  async function hydrateLanding() {
    try {
      var r = await fetch('/api/v1/hackforger/landing/stages');
      if (!r.ok) return;
      var json = await r.json();
      var stages = json.stages || {};
      
      Object.keys(stages).forEach(function(slot) {
        var data = stages[slot];
        if (!data.slug) return;  // patch semantic: skip unbound slots
        hydrateCard(slot, data);
        
        // Update HACKFORGER_LANDING_CONFIG so click handler uses live data
        if (window.HACKFORGER_LANDING_CONFIG && window.HACKFORGER_LANDING_CONFIG.stages[slot]) {
          window.HACKFORGER_LANDING_CONFIG.stages[slot] = {
            slug: data.slug,
            enabled: data.enabled
          };
        }
      });
    } catch (e) {
      if (window.console) console.warn('[hackforger-landing] hydrate failed:', e);
    }
  }
  
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', hydrateLanding);
  } else {
    hydrateLanding();
  }
})();
</script>
```

---

### Task 7: CSS card height alignment

**File:** `custom/public/assets/landing/index.html`

- [ ] **Step 1: Find the existing `<style>` block in `<head>`**

```bash
grep -n '<style>' custom/public/assets/landing/index.html | head -3
```

- [ ] **Step 2: Append CSS rules at end of style block**

```css
/* HackForger landing — equalize stage card heights */
.tone-card {
    display: flex;
    flex-direction: column;
}
.tone-card .badge-container,
.tone-card div[class*="flex items-center flex-wrap gap-1.5"] {
    min-height: 56px;
    align-content: flex-start;
}
.tone-card button[data-stage] {
    margin-top: auto;
}
```

(The `[class*="flex"]` selector catches the existing badge container which may not have `.badge-container` class explicitly; use both for resilience.)

---

### Task 8: Issue #112 organization-section fixes

**File:** `custom/public/assets/landing/index.html`

- [ ] **Step 1: Delete 5 unit count badges**

Find and delete each:
```bash
grep -n "Hosts · 3\|Organizers · 2\|Co-organizers · 7+\|Venture Capital · 4+\|7$" custom/public/assets/landing/index.html | head
```

For each match, locate the enclosing `<div class="text-[10px]...">XXX · N</div>` and delete that line.

Specific anchors (from PR #108 line numbers, may have shifted slightly):
- 指导单位 数字 "7" — find by surrounding "01 / GUIDANCE" or similar org section header
- 主办单位 "Hosts · 3"
- 承办单位 "Organizers · 2"  
- 协办单位 "Co-organizers · 7+"
- 创投支持 "Venture Capital · 4+"

Use unique substring anchors via Edit tool.

- [ ] **Step 2: Delete "上海市学生事务中心" from co-organizers list**

```bash
grep -n "上海市学生事务中心" custom/public/assets/landing/index.html
```

Find and delete the entire `<span class="org-chip ...">` element containing it.

- [ ] **Step 3: Replace all "颁奖典礼" with "颁奖活动"**

```bash
sed -i '' 's/颁奖典礼/颁奖活动/g' custom/public/assets/landing/index.html
grep -c "颁奖典礼" custom/public/assets/landing/index.html
```

Expected: 0 occurrences after replacement.

---

### Task 9: Build, run, and Phase 1 E2E

- [ ] **Step 1: Build (no `make frontend` needed — `custom/public/` is not bundled by webpack)**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -3
```

> Note: `make frontend` writes to `public/assets/`, not `custom/public/assets/`. Our edits in `custom/public/assets/landing/` don't go through Vite/webpack — direct served by `public.FileHandlerFunc`. Skip frontend build.

- [ ] **Step 2: Stop main gitea, start worktree binary**

```bash
# Verify port 3000 holder is main repo binary
PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t 2>/dev/null | head -1)
RUNNING_BIN=$(lsof -p "$PID" 2>/dev/null | awk '/txt.*REG.*\/gitea$/{print $NF; exit}')
[[ "$RUNNING_BIN" = "/Users/h2oslabs/Workspace/hackforger/gitea" ]] || { echo "REFUSE"; exit 1; }
kill "$PID"
sleep 2

cd /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/issue-110
rm -f /Users/h2oslabs/Workspace/hackforger/data/queues/common/LOCK
./gitea web --custom-path /Users/h2oslabs/Workspace/hackforger/.claude/worktrees/issue-110/custom > /tmp/gitea-issue-110.log 2>&1 &
disown
```

- [ ] **Step 3: Run 20 Phase 1 test cases per spec**

Per spec section "Phase 1：实现期", execute via agent-browser or curl + manual browser verification:

| # | Test | Tool |
|---|---|---|
| 1 | prefix unset, GET / → splash | curl |
| 2 | prefix=`landing-test`, no hackathon → landing with HTML defaults | curl |
| 3 | Create hackathon `landing-test-s1-w1` (registration phase) → hydrate | curl + browser |
| 4 | Phase change → hacking, after cache TTL refresh → button greys | browser |
| 5 | Clear prefix → splash | curl |
| 6 | Mock API 5xx → fallback HTML defaults | browser console |
| 7-15 | (per spec) | mixed |
| 16 | Tamper DB with malformed prefix → API treats as unset | curl + sqlite3 |
| 17 | i18n key resolution: zh-CN vs en | browser |
| 18 | Upgrade scenario: PR #108 deployed, #110 merged, prefix unset | already tested in Step 1 |
| 19 | Click during hydration window | manual timing |
| 20 | Phase index check via EXPLAIN | sqlite3 + EXPLAIN QUERY PLAN |

**Test #20 EXPLAIN pass criterion**:
```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/gitea.db \
  "EXPLAIN QUERY PLAN
   SELECT * FROM phase 
   WHERE activity_kind='hackathon' AND activity_id=1 
     AND start_time<=1714000000 AND end_time>1714000000;"
```
- **PASS**: output contains `USING INDEX` or `USING COVERING INDEX`
- **FAIL**: output contains `SCAN phase` (full table scan) → add migration `models/forgejo_migrations/v14l_phase_activity_composite_index.go` creating `(activity_kind, activity_id, end_time)` composite index

**Note**: Forgejo Phase model already declares `xorm:"NOT NULL INDEX"` on `phase_type_id` and `xorm:"VARCHAR(20) NOT NULL"` on `activity_kind` (without explicit composite index). Single-column index on `activity_kind` may suffice for the small data volume. Only add migration if EXPLAIN actually fails.

Save screenshots to `docs/tests/e2e/reports/2026-04-29-issue-110-impl/`.

- [ ] **Step 4: Write report**

Create `docs/tests/e2e/reports/2026-04-29-issue-110-impl.md` with results.

- [ ] **Step 5: Test cleanup — undo DB changes made during testing**

Tests #2-#4 inserted a `system_setting` row and a test hackathon. Without cleanup, main gitea (post-restart) would see leftover state.

```bash
# Stop worktree gitea
pkill -f "/Users/h2oslabs/Workspace/hackforger/.claude/worktrees/issue-110/gitea web"
sleep 2

# Clean DB residue (worktree shares main's DB at /Users/h2oslabs/Workspace/hackforger/data/gitea.db)
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/gitea.db <<EOF
DELETE FROM system_setting WHERE setting_key='hackforger.landing.league_prefix';
DELETE FROM hackathon WHERE slug LIKE 'landing-test-%';
EOF

# Restart main
cd /Users/h2oslabs/Workspace/hackforger
bash scripts/restart-gitea.sh
```

> **Note on data dir**: The worktree's `gitea` binary uses `--custom-path` to override `custom/`, but `WORK_PATH` (and therefore `data/`) still points at `/Users/h2oslabs/Workspace/hackforger/data/`. **Worktree and main SHARE the same sqlite DB** — clean up afterwards.

---

### Task 10: Commit + push + open PR

- [ ] **Step 1: Stage and commit (5 commits — 4-5-6 squashed per reviewer)**

```bash
# Commit 1: model helper
git add models/system/setting.go
git commit -m "feat(system): add GetSettingByKey helper"

# Commit 2: gate (depends on commit 1)
git add routers/web/home.go
git commit -m "feat(landing): gate landing on hackforger.landing.league_prefix setting"

# Commit 3: API + cache (services-layer to avoid layer violation)
git add services/hackforger/landing_cache.go \
        models/hackforger/hackathon_landing.go \
        routers/api/v1/hackforger/landing.go \
        routers/api/v1/api.go \
        services/hackforger/hackathon.go \
        services/hackforger/phase_sync.go
git commit -m "feat(landing): public API + 5min cache + invalidation hooks for stage sync"

# Commit 4: admin UI (depends on commit 3 for InvalidateLandingCache)
git add routers/web/hackforger/admin_landing.go \
        templates/hackforger/admin/landing_config.tmpl \
        routers/web/web.go \
        options/locale/locale_en-US.ini \
        options/locale/locale_zh-CN.ini
git commit -m "feat(landing): admin UI for setting league prefix"

# Commit 5: HTML changes (hooks + JS + CSS + #112) — single file, single commit
# Per reviewer: 3 logical changes touch one file in disjoint regions; squashing is cleaner than git add -p risk
git add custom/public/assets/landing/index.html
git commit -m "feat(landing): hydrate hooks + JS + card alignment + #112 content fixes"

# Commit 6: docs
git add docs/superpowers/specs/2026-04-29-issue-110-stage-sync-design.md \
        docs/superpowers/plans/2026-04-29-issue-110-stage-sync.md \
        docs/tests/e2e/reports/2026-04-29-issue-110-impl.md
git commit -m "docs(landing): spec + plan + e2e report for issue #110"
```

**Bisectability check**: each commit must independently compile. Run `go build` after each `git commit` to verify.

- [ ] **Step 2: Push**

```bash
git push -u origin feat/issue-110-stage-sync
```

- [ ] **Step 3: Open PR**

```bash
gh pr create --title "feat(landing): hydrate stages from DB + #112 content fix" --body "$(cat <<'EOF'
## Summary
- Closes #110: DB-driven hydration of 12 stage cards on landing page; admin opt-in via league_prefix setting
- Closes #112: organization-section content fixes (count badges, sponsor list, "颁奖典礼"→"颁奖活动")

## Design + Plan
- Spec: docs/superpowers/specs/2026-04-29-issue-110-stage-sync-design.md
- Plan: docs/superpowers/plans/2026-04-29-issue-110-stage-sync.md
- Phase 1 report: docs/tests/e2e/reports/2026-04-29-issue-110-impl.md

## Notable design decisions
- Schema-zero: prefix stored in existing `system_setting` table; new `GetSettingByKey` helper added
- Naming convention binding: hackathon slug `<prefix>-s{N}-w{M}` auto-maps to slot
- Patch semantics: JS hydrate only mutates slots with real `slug`; unbound slots keep HTML defaults
- Opt-in gate: home.go falls back to Forgejo splash if prefix unset

## Test plan
- [x] Phase 1 (20 cases): see report
- [ ] Phase 2 (QA, post-merge): tester sets prefix + creates real hackathons

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Critical Risk Checkpoints

1. **After Task 2** (GetSettingByKey): if `system_setting` API differs from spec assumption, switch to `GetAllSettings` + map lookup.

2. **After Task 3** (home.go gate): test that splash returns when prefix unset BEFORE proceeding to API work — early validation.

3. **After Task 4 Step 5** (cache invalidation hooks): verify the 4 specific hook points exist in `services/hackforger/hackathon.go` (`PublishHackathon`, `CancelHackathon`, `SetIsPublished` callers) and `services/hackforger/phase_sync.go::SyncStatusCache`. If function signatures differ, adjust. If a function doesn't exist (e.g., DeleteHackathon is models-only), document deferral.

4. **Layer architecture (per CLAUDE.md)**: cache MUST live in `services/hackforger/`, NOT `routers/`. The reviewer flagged this; Task 4 enforces. If during implementation you find a reason to put cache in routers, STOP — re-think the design (probably a circular import bug).

5. **Bisectability**: after EACH commit in Task 10, run `go build ./...` to verify it compiles independently. Commit 2 (home.go gate) requires commit 1 (helper); commit 4 (admin UI) requires commit 3 (cache function). If a commit fails to build standalone, fold it into the previous.

6. **After Task 6 Step 1-3** (HTML hooks): grep for `data-hydrate-card=` count = 12; mismatch means a card was missed.

7. **Task 8 (Issue #112)**: keep changes confined to organization section and "颁奖典礼" replacements only. Don't touch any stage cards (Task 6's territory). Note: all three (Task 6/7/8) land in the same commit per Task 10 Step 1 — but logically distinct.

8. **`make frontend` is NOT needed** for `custom/public/assets/landing/` edits — webpack only writes to `public/assets/`. Skip frontend build to save time.

9. **Phase 1 Test #20 (DB index)**: only add migration `models/forgejo_migrations/v14l_phase_activity_index.go` if EXPLAIN actually shows `SCAN phase` (full table scan). If `USING INDEX` already, don't add redundant index. The test #20 in Task 9 spells out the exact criterion.

10. **Test cleanup is mandatory** (Task 9 Step 5): worktree shares main's sqlite DB. Leftover `system_setting` row + test hackathon must be deleted before main restart, or main inherits unintended config.
