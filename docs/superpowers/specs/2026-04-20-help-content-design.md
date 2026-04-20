# Help Center Content

**Date**: 2026-04-20
**Issue**: #50 (更新帮助页内容)
**Goal**: Replace the navbar 帮助 link (currently points to `forgejo.org/docs`) with an in-app `/help` page that has two clearly-separated bilingual sections: Synnovator platform guide + HackForger system capability list.

---

## Context

### Current state (navbar)

- `templates/base/head_navbar.tmpl:51` — unsigned-user navbar help link → `https://forgejo.org/docs/latest/`
- `templates/base/head_navbar.tmpl:215` — signed-in dropdown help link → same URL

No in-app help page exists. `head_navbar.tmpl` is upstream, but the surgical edit is acceptable (2 `href` swaps, no structural change).

### What the tester asked (issue #50, comment from @Cynthialime)

Cynthia uploaded full Chinese Synnovator onboarding content (4 chapters + FAQ). That content is authoritative for the **platform玩法** section and gets used verbatim.

### User direction (2026-04-20)

- Add new `/help` with two sections: **Synnovator 平台玩法** + **HackForger 使用指南**
- Hide Forgejo's help (navbar only — not inline links in webhook/admin pages)
- Leverage PR #73's skill knowledge (`docs/skills/hackforger-api/*`) as content source for the HackForger guide section
- Depth: concise intro + capability list (not full tutorials)
- Storage: bilingual markdown files
- English translation done in this PR; quality review deferred

---

## Non-Goals

- Not replacing inline `forgejo.org/docs` links in webhook config / admin settings / user settings — those target admin/developer audiences where Forgejo docs are correct
- Not writing Git/Issue/PR step-by-step tutorials — system guide is a "what exists + links out" style
- Not implementing in-app search over help content
- Not adding a 3rd language (only zh-CN + en-US this pass)

---

## Architecture

```
options/hackforger-help/              ← content (bilingual markdown)
├── platform.zh-CN.md                 ← Cynthia's text verbatim
├── platform.en-US.md                 ← translated
├── system.zh-CN.md                   ← new: HackForger capability list (zh)
└── system.en-US.md                   ← new: HackForger capability list (en)

routers/web/hackforger/help.go        ← handler
├── loadSectionHTML(ctx, section)
│   • Reads options/hackforger-help/{section}.{lang}.md via options.AssetFS
│   • Falls back to en-US if ctx locale file missing
│   • Renders through markdown.RenderString → safe HTML
└── HelpPage(ctx) → render tplHelp

templates/hackforger/help.tmpl        ← page
├── h1 — hackforger.help.title
├── hf-card — Synnovator platform section  (markup class wraps SynnovatorHTML)
└── hf-card — HackForger system section    (markup class wraps HackforgerHTML)

templates/base/head_navbar.tmpl       ← upstream edits (2 href swaps)
├── line 51 — href → {{AppSubUrl}}/help, drop target="_blank"
└── line 215 — same
```

### Why markdown files (not Go string constants)

- Content is long-lived product copy that marketing/docs folks may want to change without Go builds
- Forgejo already has `assetfs.Layered` (`modules/options/base.go`) which gives us both bindata embedding (no I/O cost after first load) and custom/options override path
- Rebuild picks up content changes via `go generate` bindata regeneration (already wired into `make backend`)

### Why navbar-only hide scope

- Navbar help is the primary user entry — replacing it affects 99% of reach
- Inline forgejo.org links (webhook, admin auth, admin runners, user settings packages) are admin/developer context; the Forgejo docs there are authoritative and would be pointless to rewrite
- Upstream-file edits stay small and reversible

---

## File contents

### `options/hackforger-help/platform.zh-CN.md`

Cynthia's text verbatim (source: issue #50 comment, 2026-04-17). Minor cleanup: normalize smart quotes, `"Synnovator"` brand markers kept.

### `options/hackforger-help/platform.en-US.md`

Full English translation of Cynthia's content. Key terms:
- 黑客松 → Hackathon
- 悬赏 → Bounty
- 资助 → Grant
- 积分 → Credits (keep brand term)
- 协创 → Co-creation
- 赛道 → Track
- Brand "Synnovator" unchanged

### `options/hackforger-help/system.zh-CN.md` (new, drafted below)

```markdown
# HackForger 使用指南

**HackForger** 是 Synnovator 平台的底层系统引擎，基于 Forgejo 构建，提供完整的 Git 协作能力，并加入了面向协创活动的扩展模块。本页是能力清单 —— 找到你需要的功能，按链接深入。

## Git 协作基础能力

这些能力继承自 Forgejo，使用方法与 Forgejo / Gitea / GitHub 一致：

* **仓库 (Repositories)** — 创建、Fork、分支、标签、保护规则、模板仓库、镜像同步
* **Pull Request / 合并请求** — 代码评审、冲突解决、检查、自动合并
* **Issue / 议题** — 标签、里程碑、看板、自动化、锁定/置顶
* **Wiki / 项目文档** — 仓库内置 Wiki、自定义主页
* **Forgejo Actions / 工作流** — `.forgejo/workflows/*.yml` CI/CD，自托管 runner 支持
* **全文搜索** — 代码、Issue、Commit、Wiki 多维度搜索
* **Webhook / 外部集成** — 推送代码/Issue 等事件到 Slack、企微、外部 CI 等
* **SSH / HTTPS 访问** — 个人访问令牌 (PAT)、SSH key 管理
* **组织 / 团队** — 多级权限、团队仓库共享

如果你熟悉 GitHub，这里的操作习惯几乎完全一致。详细使用文档可参考上游 [Forgejo Docs](https://forgejo.org/docs/latest/)。

## HackForger 平台扩展模块

这些是 HackForger 在 Forgejo 之外新增的模块，用于支撑 Synnovator 的协创活动：

| 模块 | 说明 | 访问路径 |
|------|------|---------|
| 黑客松 Hackathon | 活动创建、报名、赛道、评委、评分、排行榜 | `/explore/hackathons` |
| 悬赏 Bounty | Issue 关联、申请/接单、托管、多人竞争奖金 | `/explore/bounties` |
| 资助 Grant | 资助轮次、项目申请、评审、分配 | `/explore/grants` |
| 积分 Credits | 平台统一积分账本、兑换商店、订单记录 | `/credits` |
| 提交作品 Submissions | 跨活动的作品聚合浏览 | `/explore/submissions` |
| 社区动态 Feed | 活动/悬赏/资助/积分事件流 | `/` 控制面板 |
| 声誉 Reputation | 用户参与度量、排行榜、层级徽章 | `/explore/reputation` |

## 开发者资源

* **API 参考** — `/api/swagger` 查看完整 OpenAPI 文档；HackForger 专用接口以 `/api/v1/hackforger/` 开头
* **CLI 工具** — `hackforger-cli`（仓库根目录，Go 实现，用于批量创建/管理活动）
* **AI Agent 集成** — 通过 PAT 授权让 AI 代表用户操作平台 API

## 需要帮助？

* 平台功能问题 —— 阅读本页"Synnovator 平台玩法"区
* Git / 仓库问题 —— 参考 [Forgejo Docs](https://forgejo.org/docs/latest/)
* 发现 bug —— 在 [HackForger GitHub 仓库](https://github.com/HackForger/hackforger/issues) 提 issue
```

### `options/hackforger-help/system.en-US.md` (new, translated)

Full English translation mirroring the Chinese structure. Table module names unchanged (Hackathon / Bounty / Grant / etc.), explanation text translated.

### `routers/web/hackforger/help.go` (new)

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"html/template"
	"net/http"
	"strings"

	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	"forgejo.org/modules/options"
	"forgejo.org/services/context"
)

const tplHelp = "hackforger/help"

// helpLangFile resolves a markdown filename for the given section using the
// requester's locale. Falls back to en-US when the locale file is missing.
func helpLangFile(ctx *context.Context, section string) (string, []byte, error) {
	lang := ctx.Locale.Language() // e.g. "zh-CN", "en-US"
	tryFiles := []string{
		section + "." + lang + ".md",
		section + ".en-US.md",
	}
	for _, name := range tryFiles {
		content, err := options.AssetFS().ReadFile("hackforger-help", name)
		if err == nil {
			return name, content, nil
		}
	}
	// Both failed — return the last error.
	_, content, err := "", []byte(nil), options.AssetFS().ReadFile("hackforger-help", tryFiles[len(tryFiles)-1])
	return "", content, err
}

func renderHelpSection(ctx *context.Context, section string) (template.HTML, error) {
	_, md, err := helpLangFile(ctx, section)
	if err != nil {
		return "", err
	}
	rendered, err := markdown.RenderString(&markup.RenderContext{Ctx: ctx}, strings.TrimSpace(string(md)))
	if err != nil {
		return "", err
	}
	return rendered, nil
}

// HelpPage renders the in-app help center — platform (Synnovator) + system (HackForger).
func HelpPage(ctx *context.Context) {
	platformHTML, err := renderHelpSection(ctx, "platform")
	if err != nil {
		ctx.ServerError("HelpPlatform", err)
		return
	}
	systemHTML, err := renderHelpSection(ctx, "system")
	if err != nil {
		ctx.ServerError("HelpSystem", err)
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.help.title")
	ctx.Data["SynnovatorHTML"] = platformHTML
	ctx.Data["HackforgerHTML"] = systemHTML
	ctx.HTML(http.StatusOK, tplHelp)
}
```

### `templates/hackforger/help.tmpl` (new)

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content">
	<div class="ui container">
		<h1 class="tw-mb-4">{{ctx.Locale.Tr "hackforger.help.title"}}</h1>

		<div class="hf-card tw-mb-4">
			<div class="hf-card-head">
				<div class="hf-card-head-left">
					{{svg "octicon-rocket" 16}}
					<span class="tw-ml-2 tw-font-medium">{{ctx.Locale.Tr "hackforger.help.platform.title"}}</span>
				</div>
			</div>
			<div class="hf-card-body">
				<article class="markup">{{.SynnovatorHTML}}</article>
			</div>
		</div>

		<div class="hf-card tw-mb-4">
			<div class="hf-card-head">
				<div class="hf-card-head-left">
					{{svg "octicon-tools" 16}}
					<span class="tw-ml-2 tw-font-medium">{{ctx.Locale.Tr "hackforger.help.system.title"}}</span>
				</div>
			</div>
			<div class="hf-card-body">
				<article class="markup">{{.HackforgerHTML}}</article>
			</div>
		</div>
	</div>
</div>
{{template "base/footer" .}}
```

### `routers/web/web.go` — route registration

Insert one line in the public route group (near `/explore/*`):

```go
m.Get("/help", hackforger_web.HelpPage)
```

### `templates/base/head_navbar.tmpl` — 2 edits

Line 51 (unsigned navbar):
```diff
- <a class="item" target="_blank" rel="noopener noreferrer" href="https://forgejo.org/docs/latest/">{{ctx.Locale.Tr "help"}}</a>
+ <a class="item" href="{{AppSubUrl}}/help">{{ctx.Locale.Tr "help"}}</a>
```

Line 215 (signed-in dropdown):
```diff
- <a target="_blank" rel="noopener noreferrer" href="https://forgejo.org/docs/latest/">
+ <a href="{{AppSubUrl}}/help">
    {{svg "octicon-question"}}
    {{ctx.Locale.Tr "help"}}
  </a>
```

### `options/locale/locale_en-US.ini` — new keys (3)

In the `[hackforger]` section:
```ini
help.title = Help Center
help.platform.title = Platform Guide (Synnovator)
help.system.title = System Guide (HackForger)
```

### `options/locale/locale_zh-CN.ini` — new keys (3)

```ini
help.title = 帮助中心
help.platform.title = 平台玩法 (Synnovator)
help.system.title = 系统使用 (HackForger)
```

---

## Testing strategy

### Unit tests

Skip — it's a content-only page with trivial handler logic. Template compile errors surface at build time; markdown rendering is tested elsewhere.

### E2E (agent-browser)

Execute after full rebuild:

1. **TC1**: unsigned user, click navbar 帮助 → lands on `/help`, both sections render, Synnovator + HackForger cards visible, no console errors.
2. **TC2**: signed-in user (hackforger), click avatar dropdown → 帮助 → same `/help` page.
3. **TC3**: switch locale to en-US via language menu → reload `/help` → English content renders (both sections).
4. **TC4**: visit `/help` directly without auth → 200 OK, content renders.

Screenshots for each TC saved to `tests/screenshots/issue-50/`.

### Verification

```bash
# Ensure new keys in both locales
grep -c 'hackforger\.help\.' options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
# Expect 3 each.

# Ensure help md files readable by binary (after rebuild)
grep -c 'Synnovator' public/assets/... # N/A, content is bindata'd into binary

# No dangling forgejo.org help links in navbar
grep 'forgejo.org/docs' templates/base/head_navbar.tmpl
# Expect: 0 matches
```

---

## Rollout

1. Create `options/hackforger-help/` with 4 markdown files
2. Run `make backend` — regenerates `modules/options/bindata.go`, embeds new content
3. Run `make frontend` — no change needed but keeps artifacts consistent
4. Verify locally: `/help` renders in both languages
5. Open PR, merge, internal instance auto-deploys on next restart
6. Close issue #50 with a comment linking to the new page

---

## Risks + mitigations

- **Bindata regeneration fails silently**: `generate-bindata.go` walks `options/` directory; if our new subdir has bad permissions or unreadable files, regeneration errors would surface at `make backend` time → caught by CI. **Mitigation**: Verify by grepping `modules/options/bindata.go` for `"hackforger-help"` post-regen.
- **Locale fallback loop**: If a user's locale is e.g. `fr-FR` (neither supported), fallback to en-US runs. If en-US file is also missing, we return 500. **Mitigation**: Both en-US files ship in the same PR; en-US presence is a hard invariant tested by the grep above.
- **Upstream navbar file diff churn**: If Forgejo changes `head_navbar.tmpl` structure upstream, our 2-line edit may conflict on merge. **Mitigation**: Low frequency of such changes; diff is tiny and easy to re-apply.
