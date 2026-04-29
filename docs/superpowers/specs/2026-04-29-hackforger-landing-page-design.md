# HackForger Unsigned-Visitor Landing Page

**Date**: 2026-04-29
**Status**: Draft (pending user approval)
**Approach**: L0 — drop-in static HTML in `custom/public/`, behavioral hooks for slug routing only

## Problem

未登录访客访问 HackForger 首页 `/` 时，看到的是 Forgejo 上游的通用 splash 页面（logo + "HackForger / 一款极易搭建的自助 Git 服务"）。这页是 Gitea/Forgejo 的"自托管 Git 服务"营销文案，**和 HackForger 的产品定位（hackathon + bounty + grant 平台）完全脱节**。

设计师交付了一份独立的、为本届"第二届燕缘·协创者号 AI+ 国际创业大赛"专门设计的营销落地页，需要把它无缝接入 HackForger 现有的认证 / 路由 / 部署体系。

## Design Direction

**视觉**：保留落地页设计师的视觉成果，**不做任何嵌入式改造**（不嵌进 Forgejo 的 navbar/footer 模板）
**集成**：仅在最小的接合点动手——路由切换 + 关键按钮的链接
**部署**：随 git pull 走，复用 HackForger 现有的 `custom/` 覆盖机制
**未来扩展性**：本期 L0 完成；后续模板化、KPI 实时数据、Wave 状态自动同步分别由独立 issue 跟进

## Scope

**In scope**:
- 用落地页（21MB 静态资源）替换未登录访客看到的首页
- 顶栏"登录"按钮接入 Forgejo `/user/login`
- 12 张 Wave 卡片的"立即报名"按钮按 slug 映射跳转 `/hackathon/{slug}`
- slug 不存在时弹"活动还没创建"模态框
- 删除落地页内多余的"已登录用户菜单"和"登录模态框"（dead code）
- Fallback 安全网：custom 文件不存在时退回原 splash

**Out of scope**:
- KPI 统计（"算力消耗/总计发放/正在进行"）→ Issue #109
- 赛项&命题预览深度同步（Wave 名称/日期/状态从 hackathon DB 拉取）→ Issue #110
- 模板化生成器（DESIGN.md + assets → 自动生成）→ Issue #111
- Hackathon 数据模型扩展（League 概念是否需要新建）→ 由 Issue #110 引导讨论
- 修改 `templates/home.tmpl`（保留作为 fallback）
- 修改 `home_forgejo.tmpl`、navbar/footer、theme/logo 等其他 custom 资产

## Architecture

### 文件落位

```
custom/public/assets/landing/                          ← 新增目录
├── index.html                                  ← 落地页主文件（基于设计师交付的 3667 行 HTML 改造）
└── assets/
    └── images/
        ├── 小助手.webp / 飞书群.webp / 公众号二维码.jpg     ← QR 码
        ├── 轮播图1.webp / 章鱼.webp / 龙虾.webp              ← 装饰图
        ├── S1.webp / S2.webp / S3.webp                       ← 阶段封面
        ├── S1-1.webp ... S3-4.webp                           ← Wave 卡片图
        ├── Group 9213.svg / Group 9216.svg                   ← 大型矢量图（可选 svgo 优化）
        ├── 深色.svg / 浅色.svg / 深色模式.svg / 浅色模式.svg  ← Logo
        └── ...其他装饰资源
```

### URL 映射

| URL | 解析路径 | 由谁服务 |
|---|---|---|
| `/` | `routers/web/home.go::Home()` | Go handler，未登录时分流到落地页 |
| `/assets/landing/index.html` | `custom/public/assets/landing/index.html` | `public.FileHandlerFunc()` 经 `CustomAssets()` 层命中 |
| `/assets/landing/assets/images/X.webp` | `custom/public/assets/landing/assets/images/X.webp` | 同上 |

### 路由分流逻辑（伪代码）

```go
// routers/web/home.go::Home()
if ctx.IsSigned {
    // ...原 Forgejo 逻辑：dashboard / 2FA / change_password / etc.
}

// 未登录分支
if setting.LandingPageURL != setting.LandingPageHome {
    ctx.Redirect(setting.AppSubURL + string(setting.LandingPageURL))
    return
}

if ctx.GetSiteCookie(setting.CookieRememberName) != "" {
    ctx.Redirect(setting.AppSubURL + "/user/login")
    return
}

// 新增：检查 hackforger landing 是否存在并通过 http.ServeContent 服务
landingPath := filepath.Join(setting.CustomPath, "public", "landing", "index.html")
if f, err := os.Open(landingPath); err == nil {
    defer f.Close()
    if fi, err := f.Stat(); err == nil && !fi.IsDir() {
        // 防止共享 cache 把匿名页污染给已登录用户
        // 注意：必须用 http.ServeContent 而非 httpcache.ServeContentWithCacheControl，
        // 后者会用 SetCacheControlInHeader 覆盖我们设置的 Cache-Control。
        ctx.Resp.Header().Set("Cache-Control", "private, no-store")
        ctx.Resp.Header().Set("Vary", "Cookie")
        http.ServeContent(ctx.Resp, ctx.Req, "index.html", fi.ModTime(), f)
        return
    }
}

// Fallback：落地页缺失时退回原 splash（不会 500）
ctx.Data["PageIsHome"] = true
ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled
ctx.Data["OpenGraphDescription"] = setting.UI.Meta.Description
ctx.HTML(http.StatusOK, tplHome)
```

**关键决策点**：
- 用 stdlib `http.ServeContent`：得到 Last-Modified/304/Range 支持，且**不**覆盖我们设置的 Cache-Control
- 显式设置 `Cache-Control: private, no-store` + `Vary: Cookie`：防止 CDN/反代把匿名访客版本错误地缓存给已登录用户（cache poisoning across cookie state）
- Fallback 路径保留 → 全新部署时（无 custom/public/assets/landing/）不会 500
- `os.Open` + `f.Stat()` 每次请求一次磁盘 IO；OS page cache 命中后 < 1µs，损耗可忽略
- **AppSubURL 假设**：本设计假定 Forgejo 部署在根路径（`AppSubURL=""`）。HackForger 当前内部实例满足此约束。如未来部署到子路径，落地页内的 `/user/login`、`/hackathon/{slug}` 硬链接需要相应调整

### HTML 中的资源路径处理（避免 404）

**问题**：落地页的 `<img src="assets/images/X.webp">` 是相对路径。当 HTML 通过 `Home()` 服务在 `/` 路径上时，浏览器解析相对路径会得到 `/assets/images/X.webp`，但实际文件在 `/assets/landing/assets/images/X.webp`。

**解决方案**：在 HTML `<head>` 最前面加一个 `<base>` 标签：
```html
<base href="/assets/landing/">
```

效果：
- ✅ 相对路径 `assets/images/X.webp` → `/assets/landing/assets/images/X.webp`（正确）
- ✅ 绝对路径 `/user/login`、`/hackathon/{slug}` 不受 `<base>` 影响（仍是绝对路径）
- ✅ 完整 URL（CDN）`https://cdn.jsdelivr.net/...` 不受影响
- ✅ Fragment-only 锚点 `href="#opc-stage-register"` 不受影响（指向当前文档）
- ✅ JS 用 `fetch('/hackathon/...')` 是绝对路径，不受 `<base>` 影响

**风险**：如果 HTML 内有任何相对路径的内部锚点跳转（极少见），`<base>` 会破坏。grep `href="[^/h#]"` 校验：

### 落地页内 HTML 改动（来自设计师原始文件）

| # | 类型 | 原状 | 改造后 |
|---|---|---|---|
| 0 | `<head>` 第一行 | `<meta charset="utf-8"/>` | **新增 `<base href="/assets/landing/">`** 让相对图片路径正确解析（必须在所有 link/script/img 之前） |
| 1 | 顶栏 "登录" 按钮（约 line 724） | `<button id="btn-open-login">登录</button>`（打开 JS 模态框） | `<a href="/user/login" class="...">登录</a>` |
| 2 | 登录模态框（约 line 749-790） | `<div id="login-modal">...` 整段 | **删除整段**（约 50 行） |
| 3 | 已登录用户菜单（约 line 726-745，以 `<!-- 已登录 -->` 注释为锚点） | 头像菜单 + 退出登录按钮 | **删除整段**（约 20 行）——这页只对未登录显示 |
| 4 | Hero "立即报名" 三个大按钮 | scroll 到 `#opc-stage-register` | **保留**原行为不变 |
| 5 | 12 张 Wave 卡片"立即报名"按钮 | 1 个 active（无 onclick）+ 11 个 disabled | 12 个按钮**统一**加 `data-stage="s{N}-w{M}"`；通过 event delegation 触发，**不**用 inline onclick |
| 6 | KPI 卡片（约 line 1346, 1355, 1363） | `<p>即将发送</p>` | **保留**——本期不接（Issue #109 跟进） |
| 7 | `<head>` 内（在 `<base>` 之后、CDN 之前）| 无 | **新增** `<script id="hackforger-landing-config">` 配置块 |
| 8 | `<head>` 内（紧跟配置块之后）| 无 | **新增** `<script id="hackforger-landing-behavior">` 行为块（含 event delegation listener） |

### 配置块设计（line 7 新增）

```html
<script id="hackforger-landing-config">
window.HACKFORGER_LANDING_CONFIG = {
  stages: {
    // 'tbd' = 弹"活动还没创建"；填实际 hackathon slug 后跳 /hackathon/{slug}
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
    's3-w4': { slug: 'tbd', enabled: false },
  }
};
</script>
```

### JS 行为脚本（head 内、紧跟 config 块之后）

放在 `<head>` 而非 `<body>` 末尾，配合 **event delegation** 模式（在 document 上挂一个 listener，匹配 `[data-stage]` 按钮点击），避免：
1. body-end 脚本与中段按钮之间的解析竞态（用户可能在脚本加载前点击）
2. 12 个按钮各加 inline `onclick=` 的 CSP 噪音

```javascript
(function() {
  function showLandingInfoModal(title, body) {
    var modal = document.getElementById('info-modal');
    if (!modal) { alert(title + '\n\n' + body); return; }
    var titleEl = modal.querySelector('[data-info-title], #info-modal-title, h3');
    var bodyEl  = modal.querySelector('[data-info-body], #info-modal-body, .info-md-body');
    if (titleEl) titleEl.textContent = title;
    if (bodyEl)  bodyEl.textContent  = body;
    modal.classList.remove('hidden');
    modal.setAttribute('aria-hidden', 'false');
  }

  function handleRegisterClick(stageId) {
    try {
      // 防御：config 解析失败（运维误改 JSON 语法）时不抛异常给用户
      var cfg = window.HACKFORGER_LANDING_CONFIG &&
                window.HACKFORGER_LANDING_CONFIG.stages &&
                window.HACKFORGER_LANDING_CONFIG.stages[stageId];
      if (!cfg || !cfg.enabled) return; // disabled stage：无反应

      if (!cfg.slug || cfg.slug === 'tbd') {
        showLandingInfoModal('活动还没创建', '本届赛事正在筹备中，请稍后再试');
        return;
      }

      var url = '/hackathon/' + encodeURIComponent(cfg.slug);
      // HEAD 校验：404 = 不存在；其他（200/302→follow→200/401/403）= 存在
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

  // Event delegation：单个 listener 截获所有 [data-stage] 按钮点击
  document.addEventListener('click', function(e) {
    var btn = e.target.closest('[data-stage]');
    if (!btn) return;
    if (btn.disabled || btn.getAttribute('aria-disabled') === 'true') return;
    var stageId = btn.getAttribute('data-stage');
    if (stageId) handleRegisterClick(stageId);
  });
})();

// showLandingInfoModal 复用页面已有的 #info-modal（line 786）：
function showLandingInfoModal(title, body) {
  const modal = document.getElementById('info-modal');
  modal.querySelector('[data-info-title]').textContent = title;
  modal.querySelector('[data-info-body]').textContent = body;
  modal.classList.remove('hidden');
}
```

### `.gitignore` 改动

```
# 现有
/custom/public/*
!/custom/public/assets/

# 新增
!/custom/public/assets/landing/
!/custom/public/assets/landing/**
```

## 行为对照表

| 访客状态 | 操作 | 期望结果 |
|---|---|---|
| 未登录 | 访问 `/` | 看到落地页（HTML 文件存在）或 Forgejo splash（fallback） |
| 已登录 | 访问 `/` | 走原 Dashboard 流程（不变） |
| 未登录 | 点顶栏"登录"按钮 | 跳到 `/user/login` |
| 未登录 | 在 `/user/login` 点底部"注册"链接 | 跳到 `/user/sign_up`（Forgejo 内置） |
| 未登录 | 点 Hero 大"立即报名" | 平滑滚动到 #opc-stage-register（不变） |
| 未登录 | 点 active Wave "立即报名"，slug=`tbd` | 弹"活动还没创建"模态框（不发请求） |
| 未登录 | 点 active Wave "立即报名"，slug=不存在 | HEAD 请求 404 → 弹"活动还没创建"（带网络 IO） |
| 未登录 | 点 active Wave "立即报名"，slug=已存在 | 跳到 `/hackathon/{slug}` 公开详情页 |
| 未登录 | 在赛事详情页点站内"立即报名" | reqSignIn 中间件重定向到 `/user/login?redirect_to=/hackathon/{slug}/register` |
| 未登录 | 登录成功 | 自动回到 `redirect_to` 指定的报名流程页 |
| 未登录 | 点 disabled Wave 按钮 | 无反应（不弹窗、不跳转、不报错） |
| 任意 | 部署到全新实例（custom/public/assets/landing/ 不存在） | 退回原 Forgejo splash（无 500） |

## 不做的事

- ❌ 不修改 `templates/home.tmpl`（保留作为 fallback 渲染目标）
- ❌ 不修改 `home_forgejo.tmpl`
- ❌ 不修改 `setting.LandingPageURL` 枚举
- ❌ 不引入 Go template 渲染落地页（保持纯静态 HTML）
- ❌ 不动落地页的 Tailwind CDN / Google Fonts CDN（与 Forgejo 编译版样式不共存）
- ❌ 不接 KPI 统计（"即将发送"保留占位 → Issue #109）
- ❌ 不接 Wave 详情数据（赛项预览硬编码 → Issue #110）
- ❌ 不做模板化生成器（→ Issue #111）
- ❌ 不动 Forgejo 上游文件（除 `routers/web/home.go::Home()` 内部一段未登录分支扩展）

## 风险与权衡

| 风险 | 严重度 | 缓解 |
|---|---|---|
| 落地页 21MB 资源进 git，仓库膨胀 | 中 | 一次性、不持续累加；后续可用 svgo 优化 8MB SVG |
| 内网无法访问 Tailwind CDN / Google Fonts | 中 | 部署前确认 hackforger.inside.h2os.cloud 出网正常；不通则起独立任务做"本地化 CDN" |
| `custom/public/assets/landing/index.html` 编辑失误破坏页面 | 中 | 文件不存在时 fallback 到 splash；JSON 配置错误会破坏 JS 但不破坏 HTML 渲染 |
| 落地页用了 `localStorage` 模拟登录态（page 内 JS） | 低 | 删除"已登录用户菜单"分支后该 JS 路径不再执行；遗留代码无害 |
| Hackathon slug 编辑后未即时生效（CDN 缓存） | 低 | `custom/public/` 由 `public.FileHandlerFunc()` 服务，无内存缓存；浏览器缓存最长 5 分钟 |
| `os.Open`/`Stat` 在每次未登录访问时调用 | 低 | OS page cache 命中后 < 1µs；可忽略 |
| **CDN/反代共享缓存把匿名版本污染给已登录用户** | **中** | `Cache-Control: private, no-store` + `Vary: Cookie`（路由层显式设置，见 Architecture） |
| **运维误改 `HACKFORGER_LANDING_CONFIG` JSON 语法导致 JS 抛异常** | **中** | `handleRegisterClick` 用 try/catch 包裹，配置丢失时降级为"活动还没创建"模态框 |
| **SEO：搜索引擎索引落地页内容替换原 splash** | 低 | 内部实例不暴露公网，影响有限；如需上公网建议补 `<meta name="robots">` + 更新 sitemap |
| **`fetch` HEAD 默认 follow redirect**（reqSignIn 302 → /user/login 会被自动跟随到 200） | 低 | 设计上"302 也视为赛事存在"恰好是期望行为，副作用是 fetch 多打一次后端；可接受 |

## i18n 处理

落地页**自带中英文双语切换**（line 3188 起的 i18n 字典 + lang-toggle-btn）。这套机制**独立于 Forgejo 的 i18n 系统**，不需要往 `options/locale/locale_*.ini` 加任何键。

唯一可能要加 i18n 键的地方是模态框文案——但当前设计复用页面已有的 i18n 字典（`"活动还没创建": "Event not yet created"`），不用动 Forgejo locale。

## CSRF 处理

无需处理。落地页对 Forgejo 的所有交互都是 GET 跳转（`/user/login`、`/hackathon/{slug}`）和 HEAD 校验。POST 操作发生在跳转后的 Forgejo 页面内，由 Forgejo `CrossOriginProtection` 中间件保护。

## 测试

### Phase 1：实现期（开发自测，不依赖任何 hackathon 数据）

测试目标：验证 UI/JS 行为独立于业务数据。所有 case 通过 agent-browser 在 `localhost:3000` 执行。

| # | Case | 步骤 | 期望 | 截图 |
|---|---|---|---|---|
| 1 | 未登录看落地页 | 退出登录 → GET / | 落地页（不是 splash） | `01-landing-unsigned.png` |
| 2 | 已登录看 Dashboard | 登录 hackforger/admin1234 → GET / | Dashboard | `02-dashboard-signed.png` |
| 3 | 顶栏"登录" | 点击右上"登录" | 跳 /user/login | `03-login-redirect.png` |
| 4 | Hero 大按钮 | 点 Hero "立即报名" | 滚动到 #opc-stage-register | `04-hero-scroll.png` |
| 5 | slug=tbd 弹窗 | 点 active S1 W1 "立即报名"（默认 tbd） | 弹"活动还没创建" | `05-tbd-modal.png` |
| 6 | slug=bogus 弹窗 | 改 config slug='bogus-test' → 点击 | HEAD 404 → 弹"活动还没创建" | `06-404-modal.png` |
| 7 | disabled 按钮 | 点 11 个 disabled Wave 按钮 | 无反应 | `07-disabled.png` |
| 8 | 主题切换 | 点右上主题按钮 | 深/浅色切换 | `08-theme.png` |
| 9 | Hero 轮播 | 等 10 秒或点箭头 | 轮播正常 | `09-carousel.png` |
| 10 | 移动端布局 | 缩窗到 375px | 不破版 | `10-mobile.png` |
| 11 | logo/theme 覆盖 | 检查 logo 仍是 hackforger 自定义 | 其他 custom 文件未受影响 | `11-logo.png` |
| 12 | 路由回归 | 访问 /explore、/user/login、/user/sign_up | 各页正常 | `12-regression.png` |
| 13 | Fallback 安全网 | 临时 mv landing/index.html → GET / | 退回 splash 不报错 | `13-fallback.png` |
| 14 | RememberMe cookie 优先 | 设置过期 RememberMe cookie 但未登录 → GET / | 重定向到 /user/login（**不**显示落地页） | `14-rememberme.png` |
| 15 | LandingPageURL 管理员覆盖 | app.ini 加 `[server] LANDING_PAGE = explore` → 重启 → GET / | 302 → /explore（**不**显示落地页） | `15-landing-override.png` |
| 16 | 配置 JSON 语法损坏降级 | 临时改 config 引入 JS 语法错误 → 点击 active "立即报名" | 不白屏；console 报错；按钮点击降级为"活动还没创建"模态框 | `16-malformed-config.png` |
| 17 | Cache-Control 头部 | `curl -I http://localhost:3000/`（未登录）| 响应头含 `Cache-Control: private, no-store` + `Vary: Cookie` + `ETag` | `17-headers.png`（截图终端） |

报告位置：`docs/tests/e2e/reports/2026-04-29-landing-page-impl.md`

### Phase 2：QA 期（合并后由测试员执行，依赖 Issue #109 / #STAGES 的决议）

测试员收到的交付：
- ✅ 已合并的 PR + 本 spec
- ✅ Issue #109（KPI 数据语义文档）+ Issue #110（Wave 同步约定）
- ✅ "按届编辑 checklist"（见下）

测试员动作：
1. 按 Issue #110 的约定（slug 命名、League 关联）创建测试用 hackathon
2. 编辑 `custom/public/assets/landing/index.html` 的 `HACKFORGER_LANDING_CONFIG`，把 `s1-w1.slug` 填成新建赛事
3. 跑完整 E2E：点击 → 跳详情 → 点报名 → 登录回跳 → 完成报名

报告位置：`docs/tests/e2e/reports/2026-XX-XX-landing-page-qa.md`

## 按届编辑 Checklist（运维 / 内容更新指南）

每届新赛事开始前，编辑以下内容：

**A 类 · 文案**
- [ ] `<title>` 标签（line 8）
- [ ] meta description（line 9, 11）
- [ ] Hero 大赛标题徽章（line 1244）
- [ ] 开篇段落（line 807）— 开赛时间
- [ ] Wave 时间徽章（line 1520-1523, 1536-1539, 等共 12 处）
- [ ] 赛季时间窗（line 842, 852-853, 863）— 决赛日期等
- [ ] 赛区/赛道描述（line 1004-1031, 1042-1078）
- [ ] 主办单位（line 1802-1803）
- [ ] 承办单位（line 1817-1818）
- [ ] 协办单位（line 1832-1838）
- [ ] 创投支持（line 1851-1854）
- [ ] Footer 版权年（line 1898）

**B 类 · 配置**
- [ ] `HACKFORGER_LANDING_CONFIG.stages[*].slug` — 12 个 Wave 的 hackathon slug
- [ ] `HACKFORGER_LANDING_CONFIG.stages[*].enabled` — 当前哪些 Wave 开放报名

**C 类 · 资源（替换图片）**
- [ ] `assets/images/小助手.webp`
- [ ] `assets/images/飞书群.webp`
- [ ] `assets/images/公众号二维码.jpg`
- [ ] `assets/images/轮播图1.webp` 等 Hero 图
- [ ] `assets/images/S1.webp` / `S2.webp` / `S3.webp` 等阶段封面

## 关联 Issue

本 spec 的合并 PR 之后会创建以下 issue 跟进：

- **Issue #109** — `[landing] Define & connect KPI stats on home landing page`
- **Issue #110** — `[landing] Auto-sync 「赛项&命题预览」section from hackathon DB`
- **Issue #111** — `[landing] (Backlog) Templatize landing page generation for future events`

issue body 草稿见 `docs/landing-page/issue-*.md`（在实现 PR 中一并 commit）。
