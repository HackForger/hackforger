# HackForger Landing Page — Stage Sync from DB + Admin League Prefix (Issue #110 + #112)

**Date**: 2026-04-29
**Status**: Draft (pending user approval)
**Approach**: L1 — DB-driven stage cards + naming-convention binding + admin opt-in via single setting + Issue #112 content fixes

## Problem

PR #108 (L0) 已经把落地页静态文件接入 HackForger，但运营痛点未解决：
1. **手编 HTML 才能绑定 hackathon**：每加 1 个 hackathon 需要编辑 `HACKFORGER_LANDING_CONFIG` JS 块的 slug
2. **状态不同步**：hackathon 阶段切换时（registration → hacking → judging）落地页按钮不会自动变灰
3. **多种文案重复硬编码**：Wave 名称、日期徽章、赛区赛道在 HTML 里写死，每届赛事都要全文搜索替换
4. **Issue #112**：测试员发现单位数量徽章 + 协办单位列表 + "颁奖典礼/活动" 等需要修改

本设计在 L0 基础上加 schema-zero 的"运营 opt-in 启用 + 自动同步"机制，并合并 Issue #112 的内容修改。

## Design Direction

**核心原则**：
- Schema-zero——不新增数据库 column / table；用 Forgejo 已有的 `system_setting` 存一个 prefix 字符串
- 命名约定——hackathon slug 按 `<prefix>-s{N}-w{M}` 格式，自动映射到 12 个落地页 slot
- Patch 语义 hydrate——API 失败 / 部分匹配时落地页降级到 HTML 硬编码内容
- Opt-in—— prefix 未配置时落地页**不显示**，访客看到 Forgejo 原始 splash

## Scope

**In scope**:
- 单条 admin setting：`hackforger.landing.league_prefix`（key-value）
- 公开 API：`GET /api/v1/hackforger/landing/stages`
- 落地页 JS hydration（patch 语义）
- 卡片高度对齐（CSS）
- `routers/web/home.go::Home()` 加 prefix gate
- Admin 设置 UI（一个文本框）
- Issue #112 三项文案修改（删除 5 处数量徽章 + 删除"上海市学生事务中心" + "颁奖典礼" → "颁奖活动"）

**Out of scope**:
- 新建 League 实体（HackForger 现有数据模型不需要）
- 在 Hackathon 表加 column（运营改 prefix 比维护 column 简单）
- 多 league 并存（一个实例只服务一届）
- KPI 统计接入（→ Issue #109）
- 模板化生成器（→ Issue #111）

## Architecture

### Schema-zero（无 migration）

存储位置：复用 Forgejo `system_setting` 表（`models/system/setting.go`，HackForger 也用此表存全局配置）。

```
key   = "hackforger.landing.league_prefix"
value = "league-2"   (空字符串 = 未配置)
```

**没有新表 / 新 column / 新 migration**——但需在 `models/system/setting.go` 加一个**单 key 读取的 helper**（现有只有 `GetAllSettings(ctx)` map 形式 + `SetSettings(ctx, map)` 写入；按已有 `GetRevision` 的模式加一个）：

```go
// 新增 in models/system/setting.go
func GetSettingByKey(ctx context.Context, key string) (string, error) {
    setting := &Setting{}
    has, err := db.GetEngine(ctx).Where("setting_key = ?", key).Get(setting)
    if err != nil { return "", err }
    if !has { return "", nil }
    return setting.SettingValue, nil
}
```

是 schema-zero，但**非 code-zero**。这一行代码加在系统通用模块，HackForger 之外也可复用。

### 路由 dispatch（home.go 修改）

```go
// routers/web/home.go::Home() 未登录分支
prefix := system_setting.GetSetting(ctx, "hackforger.landing.league_prefix")
landingPath := filepath.Join(setting.CustomPath, "public", "assets", "landing", "index.html")

if prefix != "" {
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

// fallback: Forgejo 原始 splash
ctx.HTML(http.StatusOK, tplHome)
```

**关键变化**：从 PR #108 的 "文件存在就 serve" → "**prefix 配置 AND 文件存在** 才 serve"。  
未配置 prefix 等价于"落地页未启用" → 退到 Forgejo splash。

### 公开 API：`GET /api/v1/hackforger/landing/stages`

**无需鉴权**，返回 12 slot 完整数据。

**Query 逻辑**（基于现有 `models/hackforger/phase.go::GetCurrentPhase` 模式）：

```go
// 1. prefix 空 → 直接返回 stages 全空（无 SQL 调用）

// 2. 构建 12 个期望 slug
expectedSlugs := []string{
    prefix + "-s1-w1", prefix + "-s1-w2", prefix + "-s1-w3", prefix + "-s1-w4",
    prefix + "-s2-w1", prefix + "-s2-w2", prefix + "-s2-w3", prefix + "-s2-w4",
    prefix + "-s3-w1", prefix + "-s3-w2", prefix + "-s3-w3", prefix + "-s3-w4",
}

// 3. 一次 SQL 拉到 12 个 hackathon
hackathons := []*Hackathon{}
db.GetEngine(ctx).In("slug", expectedSlugs).
    Where("is_published = ?", true).
    Find(&hackathons)

// 4. 一次 SQL 拉到所有相关 tracks（IN 查询，非 N+1）
hackathonIDs := /* extract IDs */
tracks := []*HackathonTrack{}
db.GetEngine(ctx).In("hackathon_id", hackathonIDs).Find(&tracks)
// 在 Go 里 group by hackathon_id

// 5. 对每个 hackathon 调用现有 GetCurrentPhase（一次性查"目前活跃 phase"）
//    GetCurrentPhase 内部用 time.Now().Unix() 比较 int64 时间戳
//    NOTE: phase.start_time/end_time 是 Unix 整数，不能用 SQL NOW()
for _, h := range hackathons {
    phase, err := GetCurrentPhase(ctx, "hackathon", h.ID)
    // phase 可能为 nil（未设 phase / 未到任何 phase 时间窗）→ 用 status_cache 兜底
}
```

**为什么不用单条 JOIN**：phase 表的 `start_time/end_time` 是 Unix int64（不是 SQL TIMESTAMP），不能用 `NOW()`；且同一 hackathon 可能有多个时间窗重叠的 phase，需要 deterministic 选一个。复用 `GetCurrentPhase`（已经处理这两个问题）比手写 JOIN 安全。

**性能**：12 hackathon × 1 phase query = 13 次 SQL（含主查询），加 1 次 tracks IN 查询。5 min 缓存后均摊到零。如发现 phase 表 `(activity_kind, activity_id)` 复合索引缺失，本 PR 加上。

**Response shape**:
```json
{
  "league_prefix": "league-2",
  "stages": {
    "s1-w1": {
      "slug": "league-2-s1-w1",
      "name": "初赛/Wave 1",
      "registration_window": {
        "start": "2026-04-29T00:00:00+08:00",
        "end":   "2026-05-05T23:59:59+08:00"
      },
      "current_phase": {
        "key": "registration",
        "display_name_i18n": "hackforger.phase.registration",
        "ends_at": "2026-05-05T23:59:59+08:00"
      },
      "tracks": ["数字文化", "数字营销", "数字产业", "智能应用", "智能网联", "X创新"],
      "enabled": true
    },
    "s1-w2": { "slug": null, "enabled": false },
    "...": "12 slot 全返回；slug=null 表示未匹配"
  },
  "fetched_at": "2026-04-29T16:30:00Z"
}
```

**字段语义**：
- `slug = null` → 该 slot 未匹配（hackathon 不存在 / 未发布 / prefix 不匹配）→ JS 不修改对应卡片
- `current_phase.key` ∈ `{registration, development, peer_review, results, ...}`，由 `PhaseType.Key` 决定
- `current_phase.display_name_i18n`：**返回 i18n key**（如 `hackforger.phase.registration`），**不返回本地化字符串**——public API 没有用户 locale，前端按已加载的 locale 解析；fallback 是 key 字符串本身。**Track 名称是 free-form 用户输入，按原文返回**（不做 i18n）
- `enabled` 当前定义：`current_phase.key == "registration"` → true；其他 phase 或 phase 缺失 → 用 `hackathon.status_cache` 兜底（`Open=true`，其他=false）
- `tracks`：来自 `HackathonTrack.Name` 列表（去重后保留原顺序）

**缓存**（spec 选 Option A：handler 内 sync.Map + timestamp，简单+无 Redis 依赖）：
```go
// 在 routers/api/v1/hackforger/landing.go
var landingCache struct {
    sync.RWMutex
    payload   []byte
    expiresAt time.Time
}

func GetLandingStagesAPI(ctx) {
    landingCache.RLock()
    if time.Now().Before(landingCache.expiresAt) {
        // serve cached bytes
    }
    landingCache.RUnlock()
    // ... compute fresh, write under Lock(), serve
}

func InvalidateLandingCache() {
    landingCache.Lock()
    landingCache.expiresAt = time.Time{}
    landingCache.Unlock()
}
```
- TTL: 5 min
- Admin 改 prefix 时调用 `InvalidateLandingCache()`
- Hackathon 发布/取消发布 / phase 切换时也调用（接入 hackforger 现有 hook 点）

### JS Hydration（patch 语义）

**触发**：`DOMContentLoaded` 后立即 fetch。

**Patch 规则**：只有 `data.slug` 非空才修改对应卡片的 DOM；否则保留 HTML 默认内容。

```javascript
async function hydrateLanding() {
  try {
    const r = await fetch('/api/v1/hackforger/landing/stages');
    if (!r.ok) return;
    const { stages } = await r.json();

    Object.entries(stages).forEach(([slot, data]) => {
      if (!data.slug) return;  // patch 语义：未绑定就保留 HTML
      hydrateCard(slot, data);
    });

    // 注：patch 语义不可逆——一旦 hydrate 过的 slot 之后 unbind（hackathon unpublished），
    // 当前刷新看到的还是旧数据。预期解：用户重新加载页面（HTML 默认值会重新生效）。
    // 不在本期实现"恢复 HTML 默认"——需要 snapshot 初始 DOM，复杂度高于收益。

    // 同步更新 HACKFORGER_LANDING_CONFIG 让点击行为也用上活数据
    Object.entries(stages).forEach(([slot, data]) => {
      if (!window.HACKFORGER_LANDING_CONFIG.stages[slot]) return;
      window.HACKFORGER_LANDING_CONFIG.stages[slot] = {
        slug: data.slug || 'tbd',
        enabled: data.enabled
      };
    });
  } catch (e) {
    if (window.console) console.warn('[hackforger-landing] hydrate failed:', e);
    // 静默失败，HTML 默认值就是 fallback
  }
}

function hydrateCard(slot, data) {
  const card = document.querySelector(`[data-hydrate-card="${slot}"]`);
  if (!card) return;
  
  if (data.name) {
    const nameEl = card.querySelector('[data-hydrate="name"]');
    if (nameEl) nameEl.textContent = data.name;
  }
  
  if (data.registration_window) {
    const win = data.registration_window;
    const winEl = card.querySelector('[data-hydrate-badge="window-registration"]');
    if (winEl) winEl.textContent = `#报名 ${formatRange(win)}`;
  }
  
  if (data.current_phase) {
    highlightCurrentPhaseBadge(card, data.current_phase.key);
  }
  
  const btn = card.querySelector('[data-stage]');
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

document.addEventListener('DOMContentLoaded', hydrateLanding);
```

**点击竞态防御**——PR #108 的 event delegation listener 必须检查 `disabled` / `aria-disabled` 状态：
```javascript
// 已在 PR #108：
document.addEventListener('click', function(e) {
  var btn = e.target.closest('[data-stage]');
  if (!btn) return;
  if (btn.disabled || btn.getAttribute('aria-disabled') === 'true') return;  // ✓ 防御
  ...
});
```
确认这一行存在。如果 hydrate 在用户点击之前完成 disabled 设置，listener 会正确忽略；如果点击发生在 hydrate 完成前的极短窗口内（< 500ms），用户看到 HTML 默认状态行为（最坏跳到 disabled hackathon page），可接受。

### HTML 改造（增加 hydrate hooks）

为每个 stage 卡片加 `data-hydrate-*` 属性：

```html
<div class="tone-card" data-hydrate-card="s1-w1">
  <h5 data-hydrate="name">初赛/Wave 1</h5>
  <div class="badge-container" data-hydrate="badges">
    <span data-hydrate-badge="static-tag">官方命题</span>
    <span data-hydrate-badge="window-registration">#报名 4.29-5.5</span>
    <span data-hydrate-badge="window-development">#开发 5.6-5.7</span>
    <span data-hydrate-badge="window-peer-review">#互评 5.8</span>
    <span data-hydrate-badge="window-results">#公布晋级 5.9</span>
  </div>
  <button data-stage="s1-w1" data-hydrate="cta">立即报名</button>
</div>
```

12 张卡片全加 hooks。HTML 默认值依然是赛事原始内容（即 PR #108 的状态）。

### CSS 卡片对齐

```css
/* tone-card 强制同高 */
.tone-card { display: flex; flex-direction: column; }

/* badge 容器最小高度（约 5 个 badge）*/
.tone-card .badge-container { min-height: 56px; align-content: flex-start; }

/* CTA 永远在底部 */
.tone-card button[data-stage] { margin-top: auto; }
```

### Admin UI

**位置**：在已有的 `/-/admin/hackforger/phase-types` 旁边新增 `/-/admin/hackforger/landing-config`，与现有 admin 模式一致。

**界面**：
- 单字段表单：`League Prefix: [____________]`
- Helper text："留空则不启用 HackForger 落地页（访客看到默认 Forgejo 首页）"
- 提交后清空 API 缓存
- 受 `adminReq` 中间件保护

**校验**（**这是关键**——slug 即 Forgejo Org 名称，需通过 Forgejo username 规则）：

调用 `modules/validation.IsValidUsername(prefix + "-s1-w1")` 在 admin 提交时验证：
- 必须以字母数字开头（不能 `-` 开头）
- 不能含连续的 `-`、`.`、`_` 或以这些字符结尾
- 不能撞 Forgejo 保留名（`admin`、`api`、`assets`、`user`、`org` 等）

例如 `landing` 这个 prefix 会失败（`landing-s1-w1` 撞保留？需测试），换成 `league-2` 安全。Helper text 应给可工作示例。

API 读取 prefix 时也再 validate 一次（防 DB 直改入侵）；非法时按"未配置"处理。

### Issue #112 内容修改

合并到本 PR：

| 改动 | 实现 |
|---|---|
| 删除 5 处单位数量徽章（"指导单位 7"、"主办 3"、"承办 3"、"协办 7+"、"创投 4+"）| 在 organization section 删 `<div class="text-[10px]...">XXX · N</div>` 5 处 |
| 协办单位删除"上海市学生事务中心" | 删除对应 `<span class="org-chip...">` |
| 全文 + 弹窗内"颁奖典礼" → "颁奖活动" | sed 全文替换 |

## 行为对照表

| Setting prefix | hackathon 状态 | 用户访问 `/` 看到 |
|---|---|---|
| 未设 | 任何 | Forgejo 原始 splash |
| 已设但 HTML 文件缺失 | 任何 | Forgejo 原始 splash（fallback safety net）|
| 已设 + HTML 存在 + API 失败 | 任何 | landing page，HTML 默认状态（同 L0）|
| 已设 + 0 个 hackathon 匹配 | 任何 | landing page，所有 12 卡 HTML 默认（patch 语义不动 DOM）|
| 已设 + 部分匹配 | 部分 hackathon 已建 | 匹配的 slot hydrate 实数据；其他保留 HTML 默认 |
| 已设 + 全匹配 | 12 个 hackathon 都建好 | 全 hydrate 实数据 |
| 已设 + 全匹配，但某 hackathon phase 切换 | 该 hackathon phase=hacking | 该 slot 按钮变灰，badge 显示 #开发 |

## 不做的事

- ❌ 不新增 League 数据表（schema-zero）
- ❌ 不在 Hackathon 表加 column
- ❌ 不做"运营 UI 显式绑定"（用 slug 命名约定即可）
- ❌ 不接 KPI 统计（→ #109）
- ❌ 不做模板化（→ #111）
- ❌ 不动 PR #108 已有逻辑（除 home.go gate 加 prefix 检查这一点）

## 风险与权衡

| 风险 | 严重度 | 缓解 |
|---|---|---|
| Slug 命名约定脆弱（typo 不绑定）| 中 | UNIQUE 约束 + admin 创建 hackathon 时校验提示 |
| API cache 5 min 延迟反映 admin 改动 | 低 | Admin 改 prefix 时显式 invalidate；hackathon CRUD 也 invalidate |
| Phase 缺失时 enabled 判断兜底 | 低 | 用 `hackathon.status_cache` 推断默认（Open=enabled, 其他=disabled）|
| API 调用增加 1 次 round-trip 影响首屏 | 低 | 异步 fetch；首屏先渲染 HTML，hydrate 在后；微闪 OK |
| JS hydration 把卡片渲坏 | 低 | patch 语义保底——错误路径不动 DOM；try/catch 包裹 |
| Issue #112 文案改动与 hydrate 冲突 | 极低 | 文案改的是 organization section（与 stages 无关）|
| 命名约定变更需要 sync Go + HTML | 低 | 12 slot 名字写在一个常量数组，注释提醒下届赛事调整 |
| Phase 表 `(activity_kind, activity_id)` 索引缺失导致 12 次 phase 查询慢 | 中 | 实施时 `EXPLAIN` 检查；缺则本 PR 加复合索引（应已存在但确认） |
| 现存 demo/staging 部署 PR #108 后落地页可见，#110 合并后 prefix 未配置导致回退 splash | 低 | 发布说明明确告知；admin 配 prefix 即恢复 |
| SEO/搜索引擎索引落地页 | 低 | PR #108 已用 `Cache-Control: private, no-store`；落地页是营销内容**应该**被索引，无需 noindex |
| Patch 语义不可逆——slot 曾 hydrate 后 hackathon unpublish 不会还原 HTML 默认 | 低 | 文档化为 known limit；用户刷新页面恢复（重新拉到 slug=null，但 patch 语义已"留旧"，需 hard refresh）；如要严格还原可未来加 snapshot DOM |
| 大量爬虫 / 恶意流量打 API（无认证） | 中 | 5min 缓存 + Forgejo 已有 rate limit middleware（如已启用）；监控量级 |

## 测试

### Phase 1：实现期（开发自测）

通过 `agent-browser` 在 `localhost:3000` 跑：

| # | Case | 期望 |
|---|---|---|
| 1 | prefix 未设，访问 `/`（未登录）| 看到 Forgejo splash（"painless self-hosted Git"），**不是** landing |
| 2 | 设 prefix=`landing-test`，无任何 hackathon | 看到 landing，12 卡保留 HTML 默认（patch 语义生效）|
| 3 | 创建 hackathon `landing-test-s1-w1`（is_published=true, phase=registration）| S1 W1 卡片 hydrate 实数据，其他 11 卡保留 HTML 默认 |
| 4 | hackathon phase 改成 hacking | 5 min 后（cache 过期）刷新，S1 W1 按钮灰，badge 显示 #开发 |
| 5 | 删除 prefix（清空）| 切回 Forgejo splash |
| 6 | 模拟 API 5xx | landing 仍显示，HTML 默认值保底 |
| 7 | API 返回 `slug=null` 给所有 slot | 12 卡保留 HTML 默认 |
| 8 | 卡片高度 | 12 卡同高，badge 数量不同时不会塌陷 |
| 9 | Issue #112 ：组织架构 5 个数量徽章 | 都已删除 |
| 10 | Issue #112 ：协办单位 | "上海市学生事务中心" 已移除 |
| 11 | Issue #112 ："颁奖典礼" | 全文 + 弹窗内都变成 "颁奖活动" |
| 12 | Admin UI：未配置时显示 helper text | 提示 "留空则不启用 HackForger 落地页" |
| 13 | Admin UI：填入非法字符（如空格）| 拒绝并提示 |
| 14 | Cache invalidation | Admin 改 prefix 后立刻刷新 landing 看到变化 |
| 15 | 路由回归（/explore 等）| 不受影响 |
| 16 | DB 内已存在非法 prefix（绕开表单校验直改）| API 读取时再 validate；非法按"未配置"处理 → splash |
| 17 | i18n 解析 | 浏览器中文语言：phase badge 显示 "报名"；切换到 EN：显示对应英文（前端解析 i18n key） |
| 18 | 升级路径：当前实例（PR #108 已部署）合并 #110 后 | prefix 未设 → 立刻退到 splash；管理员配 prefix → 恢复 landing |
| 19 | 点击竞态：在 hydrate 完成前点击 active 按钮 | 跳到对应 hackathon 详情页（HTML 默认 slug） |
| 20 | Phase 表缺失复合索引（如 EXPLAIN 显示）| 本 PR 加上索引（migration 加 CREATE INDEX） |

报告：`docs/tests/e2e/reports/2026-04-XX-issue-110-impl.md`

### Phase 2：QA（合并后）

测试员按真实运营节奏走完整 league：
1. 设 prefix
2. 依次创建 12 个 hackathon
3. 推进 phase
4. 验证落地页随时同步

## 关联 Issue

- **Issue #109**（KPI stats）：独立，不在本 PR
- **Issue #110**（本 spec 实现）
- **Issue #111**（模板化 backlog）：独立，不在本 PR
- **Issue #112**（落地页文案修改）：合并到本 PR
- **PR #108**（L0 落地页）：本 PR 在其基础上演进

## 实施顺序

**Precondition**: 必须等 PR #108 合并到 main 后再开始。**禁止**在 `feat/landing-page-l0` worktree 上启动 #110——会污染 PR #108 的 review 历史。

1. PR #108 合并到 main
2. `git worktree add /Users/h2oslabs/.claude/worktrees/issue-110 -b feat/issue-110-stage-sync main`
3. 复制 `custom/conf/app.ini` 到新 worktree
4. 在新分支按以下顺序实现（**先 gate 后业务**——保证未配置 prefix 时不向访客展示半成品）：
   - **a)** `models/system/setting.go` 加 `GetSettingByKey` helper
   - **b)** `routers/web/home.go` 加 prefix gate（commit 1：`feat(landing): gate landing page on hackforger.landing.league_prefix`）
   - **c)** Admin 设置 UI（commit 2：`feat(landing): admin UI for league prefix`）
   - **d)** API endpoint + cache + invalidation hooks（commit 3：`feat(landing): API + cache for stage hydration`）
   - **e)** JS hydration + HTML hooks（commit 4：`feat(landing): JS patch-semantic hydration`）
   - **f)** CSS 对齐（commit 5：`fix(landing): equalize stage card heights`）
   - **g)** Issue #112 三项文案修改（commit 6：`fix(landing): #112 organization section content`）
5. Phase 1 E2E（20 个 test case）
6. PR opens：title `feat(landing): hydrate stages from DB + #112 content fix`，body 引用 Issue #110 + Issue #112
7. Phase 2 QA + merge

**为何先 b 再 c-g**：home.go 加 gate 后，未配置 prefix 时落地页**完全不显示**。后续 c-g 全部在 dev box 设了 prefix 才能看到，避免 broken half-state 漏到生产。
