# HackForger 开发环境配置

> HackForger = Fork of Forgejo + HackForger 模块（Hackathon / Bounty / Grant / Credits / Feed）
> 上游仓库: https://codeberg.org/forgejo/forgejo
> 项目仓库: https://github.com/HackForger/hackforger
> 内网实例: https://hackforger.inside.h2os.cloud
> 主力开发工具: Claude Code

---

## 一、代码 Fork 与仓库配置

### 1.1 初始 Fork（在 GitHub 上操作）

```bash
# 方式 A：通过 GitHub Web UI Fork
# 访问 https://codeberg.org/forgejo/forgejo → 在本地 clone 后推送到 GitHub

# 方式 B：通过命令行
git clone https://codeberg.org/forgejo/forgejo.git hackforger
cd hackforger

# 查看最新稳定 tag
git tag --sort=-v:refname | head -5

# 基于最新稳定版创建主分支
git checkout -b main v12.0.1  # 替换为实际最新 tag

# 配置 remote
git remote rename origin upstream
git remote add origin git@github.com:HackForger/hackforger.git

# 推送
git push -u origin main
```

### 1.2 Remote 配置

```bash
git remote -v
# origin    git@github.com:HackForger/hackforger.git (fetch)
# origin    git@github.com:HackForger/hackforger.git (push)
# upstream  https://codeberg.org/forgejo/forgejo.git (fetch)
# upstream  https://codeberg.org/forgejo/forgejo.git (push)
```

### 1.3 分支策略

```
main                        ← 稳定分支，基于上游 tag
├── develop                 ← 日常开发
│   ├── feat/bounty         ← 线 A
│   ├── feat/hackathon      ← 线 B
│   └── feat/grants-credits ← 线 C
└── upstream/forgejo        ← 跟踪上游（只读）
```

### 1.4 同步上游

```bash
git fetch upstream --tags
git checkout -b sync/v12.1.0 v12.1.0
git cherry-pick <our-commits>    # 不 rebase，cherry-pick 我们的 commit
git checkout main && git merge sync/v12.1.0
git push origin main
```

---

## 二、开发环境：本地直装

Claude Code 在终端中运行，不需要 VS Code 或 DevContainer。直接在宿主机装依赖。

### 2.1 依赖安装

```bash
# Go >= 1.24
# Node.js >= 20
# Git, Make, SQLite3
# gh CLI（GitHub CLI）

# macOS
brew install go node git sqlite3 make gh

# Ubuntu/Debian
sudo apt install golang-go nodejs npm git sqlite3 make
# gh: https://github.com/cli/cli/blob/trunk/docs/install_linux.md

# 确保 ~/go/bin 在 PATH 中
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# 验证
go version     # >= 1.24
node --version # >= 20
gh --version
```

### 2.2 gh CLI 配置

```bash
# 登录 GitHub
gh auth login

# 常用命令
gh repo list HackForger                          # Org 仓库
gh issue list -R HackForger/hackforger            # Issue 列表
gh issue create -R HackForger/hackforger --title "Bug" --body "..."
gh pr list -R HackForger/hackforger               # PR 列表
gh pr create --title "feat: bounty" --body "..."  # 在当前 repo 目录下
gh pr merge 5                                     # merge PR
gh release create v0.1.0 --title "MVP"
```

### 2.3 首次构建

```bash
cd hackforger

# 安装依赖
make deps

# 编译
make build          # 完整（backend + frontend）
make backend        # 只编译后端（~10 秒）
make frontend       # 只编译前端（~30 秒）

# 启动
./gitea web
# 访问 http://localhost:3000 → 安装向导 → 选 SQLite → 创建 admin
```

---

## 三、Claude Code 配置

### 3.1 CLAUDE.md（项目级指令）

在项目根目录创建 `CLAUDE.md`：

```markdown
# HackForger 开发指南

## 项目概述
HackForger 是 Forgejo 的 Fork，新增了 Hackathon、Bounty、Grant、Credits 模块。
所有新代码集中在 `*/hackforger/` 独立目录，对 Forgejo 原文件改动最少。

## 架构规则
- Forgejo 是严格分层架构：routers → services → models → modules
- 上层只能调用下层，不能反向依赖
- 新功能的 Go 包路径统一在 `forgejo.org/models/hackforger/`、`forgejo.org/services/hackforger/` 等
- 数据库用 XORM ORM，定义 Go struct + tag 即可自动建表
- 前端是 Go template 服务端渲染 + 局部 Vue 3 组件增强，不是 SPA

## 目录结构
- `models/hackforger/` — 数据模型（15 张表）
- `services/hackforger/` — 业务逻辑
- `routers/api/v1/hackforger/` — REST API
- `routers/web/hackforger/` — Web 页面路由
- `templates/hackforger/` — Go HTML 模板
- `modules/hackforger/feed/` — Feed 事件类型定义
- `web_src/js/features/hackforger/` — Vue 组件

## 命名规范
- Go 文件: snake_case（hackathon.go, bounty_reward.go）
- Go 结构体: CamelCase（HackathonSubmission, BountyReward）
- API 路径: kebab-case（/api/v1/hackforger/grant-rounds）
- 模板文件: snake_case（judge_panel.tmpl）
- Vue 组件: PascalCase（BountyPanel.vue）

## 常用命令
- `make backend` — 编译后端
- `make frontend` — 编译前端
- `go test ./models/hackforger/... -v` — 跑 model 测试
- `go test ./services/hackforger/... -v` — 跑 service 测试
- `./gitea web` — 启动服务器（http://localhost:3000）

## 重要约束
- 使用 `gh` CLI 操作 GitHub（不要用 `tea`，那是 Codeberg/Forgejo 的 CLI）
- 不要修改 Forgejo 原有文件，除非在 implementation-plan-draft.md 第 1.2 节列出的 11 个注入点
- 所有状态变更必须调用 PublishHackforgerAction 写 Feed 事件
- Credits 的 Deposit/Redeem 必须在 db.WithTx 事务中
- 内网 HackForger 实例地址: https://hackforger.inside.h2os.cloud
- API 基础路径: https://hackforger.inside.h2os.cloud/api/v1/hackforger/
```

### 3.2 Subagent：代码地图扫描

```markdown
<!-- .claude/agents/codebase-navigator.md -->
---
name: codebase-navigator
description: Scans the HackForger/Forgejo codebase and produces a focused code map for the current task. Use this before starting any feature implementation to understand relevant modules and avoid context overload.
tools: Read, Grep, Glob
model: sonnet
---

# Codebase Navigator

You are a code architecture analyst for HackForger (a Forgejo fork). Your job is to scan the codebase and produce a **focused code map** — only the files and functions relevant to the current task.

## Process

1. Understand the task description
2. Identify which HackForger modules are involved (hackathon/bounty/grants/credits/feed/reputation)
3. Scan the relevant directories:
   - `models/hackforger/` for data models
   - `services/hackforger/` for business logic
   - `routers/api/v1/hackforger/` for API endpoints
   - `routers/web/hackforger/` for web routes
   - `templates/hackforger/` for UI templates
   - `web_src/js/features/hackforger/` for Vue components
4. If the task touches Forgejo native features, also scan:
   - `models/repo/`, `models/issues/`, `models/org/` — for understanding existing models
   - `services/repository/`, `services/issue/` — for reuse patterns
   - `routers/web/repo/`, `routers/api/v1/repo/` — for routing patterns
5. Produce a code map in this format:

\```
## Code Map: [Task Name]

### Directly Relevant Files
- `models/hackforger/bounty.go` — Bounty struct, BountyStatus enum
  - CreateBounty(), GetBountyByIssueID(), UpdateBounty()
- `services/hackforger/bounty.go` — Business logic
  - OnPullRequestMerged() — PR merge hook, triggers status change
  - OnBountyCompleted() — Credits distribution

### Forgejo Files to Reference (read-only)
- `models/issues/issue.go` — Issue struct (Bounty 1:1 binds to Issue)
- `services/pull/merge.go` — PR merge flow, need to add hook here

### Files to Create/Modify
- `services/hackforger/bounty.go` — Add new function: ...
- `templates/hackforger/bounty/issue_panel.tmpl` — Create new template

### Not Relevant (skip these)
- `models/hackforger/hackathon.go` — Not needed for this task
- `services/hackforger/grants.go` — Not needed
\```

## Rules
- NEVER include the entire Forgejo codebase. Only map what's needed.
- For large files, list only the relevant functions, not the entire file.
- Always check if a Forgejo native function already does what we need before suggesting new code.
- Flag any Forgejo files in the "11 injection points" list that this task might need to modify.
```

### 3.3 Subagent：Go/Forgejo 开发专家

```markdown
<!-- .claude/agents/forgejo-dev.md -->
---
name: forgejo-dev
description: Expert in Forgejo's Go codebase patterns. Use for implementing HackForger features following Forgejo's architectural conventions.
tools: Read, Write, Edit, Bash, Grep, Glob
model: opus
---

# Forgejo Development Expert

You are an expert Go developer specializing in Forgejo's architecture. When implementing HackForger features:

## Architecture Rules
1. **Layer discipline**: routers → services → models → modules. Never import upward.
2. **XORM patterns**: Use `xorm:"pk autoincr"` tags. Register tables in `models/hackforger/init.go`.
3. **Error handling**: Return typed errors (e.g., `ErrBountyNotFound`), not generic errors.
4. **Context propagation**: Always pass `context.Context` as first parameter.
5. **Database transactions**: Use `db.WithTx(ctx, func(ctx context.Context) error { ... })` for multi-table operations.
6. **Feed events**: Every state change must call `PublishHackforgerAction()`.
7. **API style**: Follow Forgejo's existing patterns in `routers/api/v1/repo/`. JSON tags on all struct fields. Swagger comments on all handlers.
8. **Template style**: Use Go template syntax `{{.Field}}`, `{{range .Items}}`, `{{if .Condition}}`. CSS uses Tailwind classes.

## Common Patterns Reference
- Creating a new model: See `models/issues/issue.go` for struct + CRUD pattern
- Creating a new API endpoint: See `routers/api/v1/repo/issue.go` for handler pattern
- Creating a new web route: See `routers/web/repo/issue.go` for page handler pattern
- Service with notifications: See `services/issue/issue.go` for notification pattern
- Cron task: See `services/cron/tasks.go` for registration pattern

## Don'ts
- Don't use `tea` CLI. Use `gh` for GitHub operations.
- Don't create Forgejo Actions (`.forgejo/workflows/`) for the GitHub repo. Use GitHub Actions (`.github/workflows/`).
- Don't use React or any framework other than Vue 3 for frontend components.
- Don't add external Go dependencies without justification. Forgejo is conservative on deps.
```

### 3.4 Hooks（防止幻觉和常见错误）

```json
// .claude/settings.json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "intercept",
            "pattern": "\\btea\\s+(issue|pr|repo|release|login)",
            "message": "⚠️ 请使用 `gh` 而不是 `tea`。tea 是 Forgejo/Codeberg CLI，我们在 GitHub 上开发。等效命令：gh issue / gh pr / gh repo。API 调用请使用 curl + GITHUB_TOKEN 或 FORGEJO_TOKEN。"
          },
          {
            "type": "intercept",
            "pattern": "codeberg\\.org/(Synnovator|HackForge)",
            "message": "⚠️ 项目托管在 GitHub，不是 Codeberg。正确地址：github.com/HackForger/hackforger"
          },
          {
            "type": "intercept",
            "pattern": "hackforge\\.inside\\.h2os\\.cloud",
            "message": "⚠️ 内网实例地址已变更为 hackforger.inside.h2os.cloud（多了一个 r）。请更新 URL。"
          }
        ]
      },
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "intercept",
            "pattern": "models/(repo|issues|org|user|actions|auth)/",
            "message": "⚠️ 你正在修改 Forgejo 原有的 model 文件。HackForger 的新代码应该放在 models/hackforger/ 目录下。如果确实需要修改原有文件，请确认是 implementation-plan 中列出的 11 个注入点之一。"
          }
        ]
      }
    ]
  }
}
```

### 3.5 自定义命令

```markdown
<!-- .claude/commands/build.md -->
---
description: Build HackForger (backend, frontend, or both)
allowed-tools: Bash
---
Build the project. Arguments: "backend", "frontend", "all" (default: all)

If "$ARGUMENTS" is "backend": run `make backend`
If "$ARGUMENTS" is "frontend": run `make frontend`
Otherwise: run `make build`

Report any compilation errors clearly.
```

```markdown
<!-- .claude/commands/test.md -->
---
description: Run HackForger tests
allowed-tools: Bash, Read, Grep
---
Run tests. Arguments: module name or "all"

If "$ARGUMENTS" is "all":
  Run `go test ./models/hackforger/... ./services/hackforger/... -v -count=1`

If "$ARGUMENTS" is a module name (bounty, hackathon, grants, credits, reputation, feed):
  Run `go test ./models/hackforger/... ./services/hackforger/... -v -run "Test.*${ARGUMENTS}" -count=1`

If "$ARGUMENTS" is "e2e":
  Run `go test ./tests/integration/... -v -tags='sqlite sqlite_unlock_notify' -run TestE2E`

Report results with pass/fail summary.
```

```markdown
<!-- .claude/commands/codemap.md -->
---
description: Generate a focused code map for a task
allowed-tools: Read, Grep, Glob
---
Use the codebase-navigator agent to scan the codebase and produce a focused code map for: $ARGUMENTS

Output the map showing only relevant files, functions, and Forgejo injection points needed.
```

```markdown
<!-- .claude/commands/api-test.md -->
---
description: Test a HackForger API endpoint against the internal instance
allowed-tools: Bash
---
Test an API endpoint on the internal HackForger instance.

Usage: /api-test METHOD /path [JSON body]
Example: /api-test GET /hackforger/hackathons
Example: /api-test POST /hackforger/hackathons '{"name":"test"}'

Execute:
\```bash
curl -s -X $METHOD \
  -H "Authorization: token $FORGEJO_TOKEN" \
  -H "Content-Type: application/json" \
  ${BODY:+-d "$BODY"} \
  "https://hackforger.inside.h2os.cloud/api/v1$PATH" | jq .
\```

If FORGEJO_TOKEN is not set, warn the user to set it first.
```

### 3.6 MCP 服务器配置

```json
// .mcp.json（项目级）
{
  "mcpServers": {
    "forgejo": {
      "command": "forgejo-mcp",
      "args": [
        "--transport", "stdio",
        "--url", "https://hackforger.inside.h2os.cloud"
      ],
      "env": {
        "FORGEJO_ACCESS_TOKEN": "${FORGEJO_TOKEN}"
      }
    },
    "context7": {
      "command": "npx",
      "args": ["-y", "@anthropic/context7-mcp"]
    }
  }
}
```

安装 Forgejo MCP Server：

```bash
go install codeberg.org/goern/forgejo-mcp/v2@latest
```

### 3.7 环境变量

```bash
# ~/.bashrc 或 ~/.zshrc 追加

# GitHub
export GITHUB_TOKEN="your-github-token"

# HackForger 内网实例
export FORGEJO_URL="https://hackforger.inside.h2os.cloud"
export FORGEJO_TOKEN="your-hackforger-instance-token"

# Claude Code 开发
export HACKFORGER_DEV=true
```

---

## 四、GitHub Actions / Forgejo Actions Runner

### 4.1 CI 策略

HackForger 使用双 CI 体系：
- **GitHub Actions** (`.github/workflows/`) — 主仓库 CI，跑测试、lint、构建
- **Forgejo Actions** (`.forgejo/workflows/`) — 自部署的 HackForger 实例使用

开发阶段的核心逻辑在 Service 层，不依赖 CI。开发和单元测试在本地完成。

### 4.2 自部署实例：Forgejo Actions Runner

在内网服务器上部署 Runner（用于自部署的 HackForger 实例）：

```yaml
# docker-compose.runner.yml（部署在内网服务器上）
services:
  runner-dind:
    image: docker:dind
    container_name: hackforger-dind
    privileged: true
    command: ['dockerd', '-H', 'tcp://0.0.0.0:2375', '--tls=false']
    restart: unless-stopped

  runner:
    image: data.forgejo.org/forgejo/runner:6.2
    container_name: hackforger-runner
    depends_on: [runner-dind]
    environment:
      DOCKER_HOST: tcp://hackforger-dind:2375
    volumes:
      - ./runner-data:/data
    restart: unless-stopped
```

```bash
# 注册 Runner
docker exec -it hackforger-runner forgejo-runner register \
  --instance https://hackforger.inside.h2os.cloud \
  --token RUNNER_TOKEN \
  --name hackforger-runner \
  --labels docker:docker://node:20-bookworm

docker exec -it hackforger-runner forgejo-runner daemon
```

### 4.3 上游仓库的 Actions/Bot

Forgejo 上游仓库的 `.forgejo/workflows/` 是给 Forgejo 自身 CI 用的。我们的策略：

| 上游配置 | 处理 |
|---------|------|
| `.forgejo/workflows/testing.yml` | **保留**，供自部署 HackForger 实例使用 |
| `.forgejo/workflows/release.yml` | **保留但暂不用**，我们暂时不需要自动 release |
| `.github/workflows/` | **新增**，GitHub 主仓库的 CI/CD |
| Renovate/Dependabot 配置 | **关闭**，依赖更新跟随上游 |
| Issue/PR 模板 | **替换为我们自己的模板** |

---

## 五、内网 HackForger 实例

### 5.1 app.ini 配置

```ini
; 部署在内网服务器上

[server]
DOMAIN = hackforger.inside.h2os.cloud
ROOT_URL = https://hackforger.inside.h2os.cloud/
HTTP_ADDR = 0.0.0.0
HTTP_PORT = 3000
SSH_DOMAIN = hackforger.inside.h2os.cloud
SSH_PORT = 22
DISABLE_SSH = false
LFS_START_SERVER = true

[database]
DB_TYPE = sqlite3
PATH = /data/hackforger/hackforger.db

[actions]
ENABLED = true
DEFAULT_ACTIONS_URL = https://data.forgejo.org

[hackforger]
ENABLED = true

[hackforger.hackathon]
MAX_TRACKS_PER_HACKATHON = 10
AUTO_CREATE_TEAM_ON_APPROVE = true

[hackforger.bounty]
ALLOWED_CURRENCIES = USD,CNY,EUR,credits
AUTO_EXPIRE_CHECK_INTERVAL = 5m

[hackforger.credits]
ENABLED = true
INITIAL_BALANCE = 0

[hackforger.reputation]
RECALC_INTERVAL = 1h
SCORE_WEIGHTS = stars:1,bounties:5,hackathon_wins:10,grants:3

[hackforger.feed]
GLOBAL_EVENTS_IN_FEED = true

[hackforger.assistant]
ENABLED = false
; 后续启用时配置：
; LLM_PROVIDER = ollama
; LLM_API_URL = http://localhost:11434/v1
; LLM_MODEL = llama3.2
```

### 5.2 HTTPS 配置

内网通过反向代理提供 HTTPS（假设已有 Caddy/Nginx）：

```
# Caddyfile
hackforger.inside.h2os.cloud {
    reverse_proxy localhost:3000
    tls internal  # 或使用内网 CA 签发的证书
}
```

---

## 六、开发工作流

### 6.1 日常流程（Claude Code）

```bash
# 1. 进入项目目录
cd hackforger

# 2. 启动 Claude Code
claude

# 3. 开始任务前先扫描代码地图
> /codemap bounty status machine implementation

# 4. 开发功能
> 实现 Bounty 的 exclusive 模式状态机，包括 Open → Claimed → InReview → Completed → Paid 的转换逻辑

# 5. 编译检查
> /build backend

# 6. 运行测试
> /test bounty

# 7. 测试 API
> /api-test GET /hackforger/bounties

# 8. 提交
> 提交当前改动，commit message: "feat(bounty): implement exclusive mode status machine"
```

### 6.2 团队协作

```bash
# 创建功能分支
git checkout -b feat/bounty develop
git push -u origin feat/bounty

# 开发完成后创建 PR（使用 gh）
gh pr create \
  --title "feat(bounty): implement exclusive mode" \
  --body "Implements the Bounty exclusive mode status machine..." \
  --base develop

# Review + Merge
gh pr merge <PR_NUMBER>
```

### 6.3 测试

```bash
# 单元测试
go test ./models/hackforger/... ./services/hackforger/... -v -count=1

# 只跑某个测试
go test ./services/hackforger/... -v -run TestExclusiveBountyFlow_Happy

# 集成测试（需要内网实例运行）
go test ./tests/integration/... -v -tags='sqlite sqlite_unlock_notify' -run TestE2E_Phase1

# 覆盖率
go test ./models/hackforger/... ./services/hackforger/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## 七、首次启动检查清单

```
□ GitHub 账号已加入 HackForger org，有 repo write 权限
□ gh auth login 配置完成
□ git clone 成功，remote 配置正确（origin=GitHub, upstream=Forgejo/Codeberg）
□ Go >= 1.24 / Node >= 20 / SQLite3 已安装
□ make deps 成功
□ make build 编译成功
□ ./gitea web 启动成功
□ 安装向导完成（SQLite, admin 账号）
□ CLAUDE.md 已放置在项目根目录
□ .claude/agents/ 目录包含 codebase-navigator.md 和 forgejo-dev.md
□ .claude/settings.json 包含 hooks 配置（拦截 tea/codeberg 调用）
□ .claude/commands/ 包含 build/test/codemap/api-test 命令
□ 环境变量 GITHUB_TOKEN / FORGEJO_TOKEN 已设置
□ GET https://hackforger.inside.h2os.cloud/api/v1/hackforger/hackathons → 200
```
