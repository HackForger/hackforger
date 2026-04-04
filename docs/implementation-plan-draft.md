# HackForger 实现方案

---

## 〇、背景

### 项目是什么

HackForger 是 Forgejo（开源自托管 Git Forge）的 Fork，在其基础上新增 Hackathon 管理、Bounty 悬赏、Grant 资助、积分体系和社区 Feed 功能，面向**开源协作社区和 AI Agent 协作场景**。

对标产品是 [DoraHacks](https://dorahacks.io)——全球最大的 Hackathon + 开源激励平台。HackForger 的目标是在自托管的 Git Forge 上复现其核心功能，同时去掉 Web3/区块链依赖，让普通开发者社区也能使用。

### 为什么选 Forgejo 做底座

| 考量 | Forgejo 的优势 |
|------|---------------|
| 代码托管 | Git 托管、PR Review、Issue 跟踪、Wiki、CI/CD（Actions）——Hackathon/Bounty 的基础设施已经内置 |
| 组织模型 | Organization + Team 天然对应活动组织者和参赛队伍 |
| 社交原语 | Star（投票）、Follow（关注）、Watch（通知）、Reaction（反馈）已经存在 |
| 自托管 | 单二进制部署、SQLite 零配置、内网运行、数据完全自控 |
| 可 Fork | Go 单体应用，分层架构清晰（routers → services → models → modules），XORM 自动 migration |
| 社区治理 | 非营利组织 Codeberg e.V. 运营，GPL 协议，不会被商业公司接管 |

核心洞察：**Hackathon 的生命周期天然映射到 Git 操作**——参赛项目 = Repo Fork，团队 = Org Team，交付物 = PR Merge，代码评审 = Review。HackForger 不是"在 Git 平台上模拟 Hackathon"，而是"Hackathon 就是 Git workflow 的自然延伸"。

### 关键设计决策

以下决策在设计阶段经过逐个讨论确认：

**1. Fork 源码，不用 SPA 覆盖层**

曾考虑三种方案：纯原生（约定 + Actions）、SPA 注入（footer.tmpl + JS）、Fork 源码。结论是 Fork 源码覆盖度最高（~90% vs SPA 的 ~60% vs 原生的 ~35%），且一旦 Fork，SPA 层能做的事 Fork 都能做得更好（服务端渲染 vs 客户端注入），SPA 变成累赘。但 Fork 内部大量复用 Forgejo 原生功能——能用原生的就不写新代码。

**2. 不引入 Web3 概念**

DoraHacks 的二次方投票（QV）、二次方资助（QF）、链上防女巫等机制来自加密货币社区的公共物品资助场景，前提是"匿名用户 + 真金白银 + 无中心化决策者"。HackForger 的用户有 Git commit 历史作为身份证明，有组织者可以直接决策，不需要算法替代人。因此：投票用 Star，资金分配由组织者决定，Star 数作为参考信号。如果将来用户确实需要 QV/QF，数据模型已预留扩展空间。

**3. Bounty 挂载 Issue 而不是独立存在**

Bounty 表和 Issue 表 1:1 绑定。这样 Issue 的全部能力——Label、Assignee、Comment、关联 PR、Timeline、通知——Bounty 零成本继承，不需要重新实现。Bounty 表只附加"金额 + 状态机 + 奖励列表"。

**4. 两种 Bounty 模式**

Exclusive（独占）：1 人认领 → 1 人交付，适合明确的 bug fix / feature 任务。Competitive（竞赛）：多人提交 → 前 N 名获奖，适合开放式挑战。统一了 DoraHacks 的 "Bounty" 和 "Hackathon Prize" 两种场景。

**5. 积分是可花费货币，声誉是只读成就分**

积分（Credits）：通过完成 Bounty / 赢得 Hackathon / 获得 Grant 赚取，可兑换算力、软件许可等资源。声誉（Reputation）：自动计算的综合评分，不可花费，用于展示贡献者资历。两者分开，不会出现"花声誉"的反直觉设计。

**6. API-First，支持 Agent 参与**

每个功能同时提供 Web 页面和 REST API。Agent 通过标准 PAT 认证调用与人类相同的 API。所有状态变更触发 Webhook 事件。API 支持 `?expand=` 减少调用次数、`/summary` 结构化摘要、batch 批量操作。

**7. Feed 通过 Notifier 接口写入 Forgejo 原有 action 表**

不新建 Feed 表。注册 `HackForgerNotifier` 实现 Forgejo 的 `services/notify/Notifier` 接口，在状态变更时将事件写入已有的 `action` 表，复用其通知和渲染基础设施。新增 Dashboard "关注动态" Tab，查询逻辑是"我 Follow 的人的行为 + 全局事件"。

**8. 对原文件改动最小化**

所有新代码集中在 `*/hackforger/` 独立目录。对 Forgejo 原有文件只改 11 处（每处 1-5 行），包括路由注册、Notifier 注册、模板 include 注入点。新 model 通过 `init()` + `db.RegisterModel()` 自动注册，无需修改 `models/db/engine.go`。这使得跟随上游版本的 cherry-pick 成本极低。

**9. Hackathon 生命周期映射 Forgejo 原生实体**

Hackathon 的核心实体直接映射到 Forgejo 原生概念：

| HackForger | Forgejo 实体 | 说明 |
|-----------|-------------|------|
| Hackathon | Organization | 创建 Hackathon 自动创建 Org，Org owner = 组织者 |
| Track | Repository | 每个赛道是 Org 内的一个 Repo（自动初始化） |
| 报名 | Org Membership | 报名 = 加入 Org 成员 |
| 提交 | Fork + PR | 参赛者 Fork 赛道 Repo，提交 PR 作为作品 |
| 评审 | PR Review + 数字评分 | PR Review 用于文字反馈，数字评分(0-10)存扩展表 |

扩展表（`hackathon`, `hackathon_track`, `hackathon_submission` 等）只存 Forgejo 没有的元数据：状态机、截止日期、奖品、数字评分、排名。这些表通过 foreign key（`LinkedOrgID`, `RepoID`, `PRID`）关联到 Forgejo 实体。

**Milestone / Tag / Release 映射（Phase 2 实现）**

| Hackathon 概念 | Forgejo 原语 | 作用 |
|---------------|-------------|------|
| 阶段截止日期 | Milestone.Deadline | 每个 Track Repo 自动创建 Registration / Hacking / Judging / Results 四个 Milestone，PR 归属当前阶段 Milestone |
| 阶段切换 | Tag | `v0-kickoff`（Hacking 开始，模板代码基线）、`submission-deadline`（Hacking 结束，锁定提交）、`v1-results`（Finalize，包含获奖信息） |
| 最终成果 | Release | Finalize 后自动创建 Release：标题=赛事名+赛道+Results，内容=排行榜+获奖者+PR 链接，可附评审报告 |
| 参赛作品 | PR → Milestone | 按阶段分组查看所有提交 |

参赛者 Fork 时基于 `v0-kickoff` Tag 开始开发，评审基于 `submission-deadline` Tag 对比 diff。

### 目标用户

| 角色 | 说明 |
|------|------|
| **平台管理员** | 部署、配置、管理兑换选项、全局运营 |
| **活动组织者** | 创建 Hackathon / Bounty / Grant Round，管理报名/评审/分配 |
| **参赛 Hacker** | 发现活动、报名、组队、提交作品、认领 Bounty、兑换积分 |
| **AI Agent** | 通过 API 自动发现任务、认领 Bounty、提交代码、评审打分 |

### 范围外（当前不做）

- QV/QF 投票算法
- 防女巫机制
- 区块链 / 链上支付
- 积分转账（用户间）
- 实时聊天 / 私信
- 全局论坛
- Activity Feed 里程碑事件（配置开关已预留）

---

### 设计原则

1. 原生功能是正确抽象就直接用，否则写新代码
2. 每个新功能同时提供 Web 页面和 REST API
3. API 是一等公民——Agent 能做到人能做的一切
4. 新代码集中在独立目录，对原文件改动最小化
5. 所有状态变更产生 Feed 事件
6. 不引入 Web3 概念

---

## 一、全局架构

### 1.1 目录结构

```
forgejo/
├── models/hackforger/
│   ├── init.go
│   ├── hackathon.go
│   ├── bounty.go
│   ├── grants.go
│   ├── credits.go
│   ├── reputation.go
│   └── action_types.go            # NEW: 事件类型常量 + Content 结构体
│
├── services/hackforger/
│   ├── hackathon.go
│   ├── hackathon_judge.go
│   ├── bounty.go
│   ├── grants.go
│   ├── credits.go
│   ├── reputation.go
│   ├── notifier.go                # NEW: HackForgerNotifier 实现 notify.Notifier 接口
│   ├── feed.go                    # NEW: Feed 查询函数
│   ├── assistant.go
│   ├── search.go
│   └── cron.go
│
├── routers/api/v1/hackforger/
│   ├── hackathon.go
│   ├── bounty.go
│   ├── grants.go
│   ├── credits.go
│   ├── reputation.go
│   ├── feed.go                    # NEW: Feed API
│   ├── search.go
│   └── assistant.go
│
├── routers/web/hackforger/
│   ├── hackathon.go
│   ├── bounty.go
│   ├── grants.go
│   ├── credits.go
│   └── assistant.go
│
├── templates/hackforger/
│   ├── hackathon/*.tmpl
│   ├── bounty/*.tmpl
│   ├── grants/*.tmpl
│   ├── credits/*.tmpl
│   ├── reputation/card.tmpl
│   ├── feed/items.tmpl            # NEW: Feed 事件渲染
│   └── assistant/search_bar.tmpl
│
├── web_src/js/features/hackforger/
│   └── init.js                    # 在 onDomReady() 中调用，懒加载挂载 Vue 组件
│
└── web_src/js/components/hackforger/
    ├── BountyPanel.vue
    ├── HackathonRegForm.vue
    ├── JudgeScoreCard.vue
    ├── GrantAllocator.vue
    ├── CreditRedeemDialog.vue
    └── PlatformAssistant.vue
```

### 1.2 对原有文件的改动（共 11 处）

注：新 model 通过 `models/hackforger/init.go` 的 `init()` 函数调用 `db.RegisterModel()` 自动注册，无需修改 `models/db/engine.go`。

```
① routers/web/web.go registerRoutes()    — 注册 HackForger web 路由 Group
② routers/api/v1/api.go                  — 注册 HackForger API 路由 Group
③ services/cron/tasks_extended.go        — 注册 4 个 cron 任务（RegisterTaskFatal 模式）
④ services/notify/notifier.go            — 注册 HackForgerNotifier（或扩展 Notifier 接口）
⑤ templates/repo/issue/view_content.tmpl — Bounty 面板注入点
⑥ templates/repo/issue/list.tmpl         — Bounty Badge
⑦ templates/explore/navbar.tmpl          — 3 个新 Tab（Hackathons / Bounties / Grants）
⑧ templates/base/head_navbar.tmpl        — AI 搜索栏
⑨ web_src/js/index.js onDomReady()       — import + 初始化 hackforger 模块
⑩ routers/web/user/home.go              — Dashboard feed 查询扩展
⑪ templates/user/dashboard/feeds.tmpl   — "关注动态" Tab + HackForger 事件渲染 include
⑫ models/activities/action.go         — GetFeeds 改 INNER JOIN 为 LEFT JOIN（支持 repo_id=0 的 HackForger 事件）
⑬ routers/web/web.go                  — CrossOriginProtection 添加 ROOT_URL 为 trusted origin
```

### 1.3 数据库 Migration

15 张新表需要在 `models/forgejo_migrations/` 中创建正式 migration 文件并注册到 migration 链。不能仅依赖 XORM auto-sync，否则无法保证生产环境升级时的 schema 正确性。

### 1.4 国际化与 API 文档

- 所有模板文本需在 `options/locale/` 添加 i18n key，使用 `ctx.Locale.Tr()` 渲染
- 所有 API 端点需添加 Swagger 注解，确保通过 CI 的 `make swagger-check`

### 1.5 前端开发约定

- **Tailwind CSS**：所有 class 使用 `tw-` 前缀（项目配置了 `prefix: "tw-"`，`important: true`）
- **Vue 3 Options API**：现有 Vue SFC 均使用 Options API，新组件保持一致
- **组件挂载**：feature module 通过 `await import()` 懒加载 Vue 组件，`createApp().mount(el)` 挂载到模板中的 DOM 占位元素
- **UI 框架**：Fomantic UI（Semantic UI fork）用于 dropdown/modal/form 等基础组件，新功能优先复用
- **主题适配**：新组件需同时支持 light/dark 主题，通过 CSS 变量实现

---

## 二、数据模型

### 2.1-2.5 与 v2 相同（15 张新表）

Hackathon 4 表、Bounty 4 表、Grant 2 表、Credits 4 表、Reputation 1 表。

**Phase 1.1 新增字段：**
- `hackathon.linked_org_id` — 自动创建的 Forgejo Organization ID
- `hackathon_registration.org_id` — 0=个人参赛，>0=组织参赛
- `hackathon_submission.fork_repo_id` — 参赛者 Fork 的 Repo ID
- `hackathon_submission.pr_id` — Pull Request ID
- `hackathon_submission.pull_index` — PR 在赛道 Repo 中的序号

### 2.6 Feed 系统（零新表）

复用 Forgejo 原生 `action` 表。在 `models/hackforger/action_types.go` 中定义：

**事件类型常量**（从 30 开始，避开 Forgejo 已有的 1-27）：

| 范围 | 类别 | 事件 |
|------|------|------|
| 30-42 | 用户行为类 | HackathonCreated(30), Registered(31), Submitted(32), Scored(33), BountyCreated(34), Claimed(35), Delivered(36), Completed(37), WinnersSelected(38), GrantRoundCreated(39), ProjectSubmitted(40), GrantAwarded(41), CreditsRedeemed(42) |
| 50-56 | 实体生命周期类 | HackathonPhaseChanged(50), Finalized(51), BountyExpired(52), Cancelled(53), GrantRoundOpened(54), Closed(55), Finalized(56) |
| 60 | 里程碑类（将来可选） | Milestone(60) |

注：需同时在 `ActionType.String()` 方法中添加新类型的字符串映射。

**Content 结构体**：`HackforgerActionContent`（通用）和 `HackforgerPhaseContent`（含状态变更信息），JSON 序列化后存入 `action.content` 字段。

### 2.7 受众策略

| 事件 | 受众 |
|------|------|
| Hackathon/Grant 发布、结果公布 | 全局（UserID=0） |
| Hackathon 阶段切换 | Org 成员 |
| Bounty 发布 | 全局 + Repo Watchers |
| Bounty 认领/完成 | 发起人 Followers + Repo Watchers |
| Bounty 过期/取消 | Repo Watchers |
| 用户报名/提交/打分/兑换 | 发起人 Followers |

---

## 三、Feed 核心实现

### 3.1 架构概览

Feed 系统分四层：

- **发布层**：`services/hackforger/notifier.go` 实现 `services/notify/Notifier` 接口。在状态变更时根据受众策略将事件写入 `action` 表。对于已有事件（如 `MergePullRequest`），在 Notifier 方法中检查是否关联 Bounty 并触发状态变更。
- **查询层**：`services/hackforger/feed.go` 提供 Following（我关注的人 + 全局事件）、Global（全局）、Entity（某个 Hackathon/Bounty/Grant 的时间线）三种查询模式。
- **API 层**：`GET /api/v1/hackforger/feed`，支持 `type`、`entity_id`、`page`、`limit` 参数。
- **渲染层**：`templates/hackforger/feed/items.tmpl` 渲染 HackForger 事件 + Dashboard `feeds.tmpl` 注入 "关注动态" Tab。

### 3.2 受众分发

每个状态变更通过 Notifier 触发对应事件（约 18 处调用点）。受众策略按事件类型决定：全局（UserID=0）、Org 成员、Followers、Repo Watchers。

### 3.3 Feed API 返回格式

```
GET /api/v1/hackforger/feed
    ?type=following|global|org|hackathon|bounty|user
    &entity_id=42&org_id=5&user_id=10
    &page=1&limit=20

返回:
{
  "items": [{
    "id": 1234,
    "op_type": 30,
    "op_name": "hackathon_created",
    "actor": {"id":2, "username":"org_alice"},
    "entity": {"type":"hackathon", "id":42, "name":"Spring Hack 2026",
               "slug":"spring-hack-2026", "url":"/hackathon/spring-hack-2026"},
    "phase": null,
    "message": "org_alice 发布了新的 Hackathon「Spring Hack 2026」",
    "created_at": "2026-04-01T10:00:00Z"
  }],
  "total_count": 156
}
```

---

## 四、Service 层关键逻辑

### 4.1 Bounty 状态机

```
Exclusive:  Open → Claimed → InReview → Completed → Paid
                                ↓ (拒绝)  → Claimed（退回重做）

Competitive: Open → InReview → Completed → Paid

通用: 超 Deadline → Expired  |  Publisher 取消 → Cancelled
```

### 4.2 Bounty-PR 联动

Bounty-PR 联动通过 `HackForgerNotifier` 的 `MergePullRequest` 方法触发：当 PR merge 时检查关联 Issue 是否有绑定的 Bounty，若有且 PR 作者是 Claimer，则自动将 Bounty 状态切换为 InReview。

### 4.3 Credits 事务安全

Deposit 和 Redeem 操作均使用 `db.WithTx` 保证事务安全。Deposit 累加余额并写入 Transaction 记录；Redeem 扣减余额、扣减库存、创建 Order，三步在同一事务内完成。

### 4.5 Hackathon-Org 生命周期联动

| 操作 | Forgejo 动作 | HackForger 动作 |
|------|-------------|----------------|
| 创建 Hackathon | `CreateOrganization(slug)` | `INSERT hackathon (linked_org_id=org.ID)` |
| 添加 Track | `CreateRepository(org, track-name)` | `INSERT hackathon_track (repo_id=repo.ID)` |
| 报名（个人）| `AddOrgUser(org, user)` | `INSERT hackathon_registration (org_id=0)` |
| 报名（团队）| `AddOrgUser(org, user)` | `INSERT hackathon_registration (org_id=team_org.ID)` |
| 提交作品 | `ForkRepository` + `NewPullRequest` | `INSERT hackathon_submission (fork_repo_id, pr_id)` |

个人→团队自动升级：如果个人已报名，再以组织身份报名同一 Hackathon，原个人记录自动升级为团队记录。

### 4.4 Web 路由表

注：路由使用 go-chi router + `web.Route` wrapper 注册。权限中间件使用 `verifyAuthWithOptions` 模式。

```
/explore/hackathons                         — 公开浏览
/explore/bounties                           — 公开浏览
/explore/grants                             — 公开浏览
/hackathons/new                             — 创建（需登录）
/hackathon/{slug}                           — 详情/报名/提交/排行
/hackathon/{slug}/manage/*                  — 管理（需组织者权限）
/hackathon/{slug}/judge/*                   — 评审（需评委权限）
/grants/new                                 — 创建 Round（需登录）
/grants/{slug}                              — 详情/项目/结果
/grants/{slug}/manage/*                     — 管理（需组织者权限）
/credits                                    — 余额/流水/兑换（需登录）
/assistant/search                           — AI 搜索
/assistant/chat                             — AI 对话（需登录）
```

---

## 五、API 设计

### 5.1 原则

RESTful + JSON，`/api/v1/hackforger/...`。分页（`?page=&limit=`，响应含 `X-Total-Count`）、筛选（`?status=&sort=&order=`）、展开（`?expand=repo,owner`）、批量操作、Webhook。

### 5.2 完整端点列表

**Hackathon**
```
POST   /hackathons                              创建
GET    /hackathons                              列表（?status=&org_id=&q=）
GET    /hackathons/{id}                         详情
PUT    /hackathons/{id}                         更新
DELETE /hackathons/{id}                         删除（仅 Draft）
POST   /hackathons/{id}/publish                 → Registration
POST   /hackathons/{id}/start                   → Hacking
POST   /hackathons/{id}/start-judging           → Judging
POST   /hackathons/{id}/finalize                → Finished
GET    /hackathons/{id}/tracks                  赛道列表
POST   /hackathons/{id}/tracks                  创建赛道
PUT    /hackathons/{id}/tracks/{tid}            更新赛道
DELETE /hackathons/{id}/tracks/{tid}            删除赛道
POST   /hackathons/{id}/register                报名
GET    /hackathons/{id}/registrations           报名列表
PUT    /hackathons/{id}/registrations/{rid}     审批
POST   /hackathons/{id}/submissions             提交参赛
GET    /hackathons/{id}/submissions             提交列表
GET    /hackathons/{id}/submissions/{sid}       提交详情
PUT    /hackathons/{id}/submissions/{sid}       更新提交
GET    /hackathons/{id}/judges                  评委列表
POST   /hackathons/{id}/judges                  添加评委
DELETE /hackathons/{id}/judges/{uid}            移除评委
POST   /hackathons/{id}/submissions/{sid}/score 打分
GET    /hackathons/{id}/submissions/{sid}/scores 评分列表
GET    /hackathons/{id}/leaderboard             排行榜
GET    /hackathons/{id}/summary                 摘要（Agent 友好）
POST   /hackathons/{id}/submissions/{sid}/review      评审（人类或 AI，由 reviewer 决定）
```

**Bounty（Repo 级）**
```
POST   /repos/{owner}/{repo}/bounties                   创建
GET    /repos/{owner}/{repo}/bounties                   列表
GET    /repos/{owner}/{repo}/bounties/{id}              详情
POST   /repos/{owner}/{repo}/bounties/{id}/rewards      添加 Reward
GET    /repos/{owner}/{repo}/bounties/{id}/rewards      Reward 列表
DELETE /repos/{owner}/{repo}/bounties/{id}/rewards/{rid} 删除 Reward
POST   /repos/{owner}/{repo}/bounties/{id}/applications 申请认领
GET    /repos/{owner}/{repo}/bounties/{id}/applications 申请列表
PUT    /repos/{owner}/{repo}/bounties/{id}/applications/{aid} 审批
POST   /repos/{owner}/{repo}/bounties/{id}/complete     验收通过
POST   /repos/{owner}/{repo}/bounties/{id}/pay          标记已支付
POST   /repos/{owner}/{repo}/bounties/{id}/cancel       取消
POST   /repos/{owner}/{repo}/bounties/{id}/expire       过期
POST   /repos/{owner}/{repo}/bounties/{id}/winners      选出获奖者
GET    /repos/{owner}/{repo}/bounties/{id}/winners      获奖者列表
GET    /repos/{owner}/{repo}/bounties/{id}/entries       参赛列表
```

**Bounty（全局）**
```
GET    /hackforger/bounties                           全平台列表
GET    /hackforger/bounties/stats                     统计
GET    /hackforger/bounties/leaderboard               Hunter 排行
```

**Grant**
```
POST   /hackforger/grants/rounds                      创建 Round
GET    /hackforger/grants/rounds                      列表
GET    /hackforger/grants/rounds/{id}                 详情
PUT    /hackforger/grants/rounds/{id}                 更新
POST   /hackforger/grants/rounds/{id}/open            → Open
POST   /hackforger/grants/rounds/{id}/close           → Reviewing
POST   /hackforger/grants/rounds/{id}/finalize        锁定分配
POST   /hackforger/grants/rounds/{id}/distribute      标记已分发
GET    /hackforger/grants/rounds/{id}/export          导出 CSV
POST   /hackforger/grants/rounds/{id}/projects        提交申请
GET    /hackforger/grants/rounds/{id}/projects        项目列表
PUT    /hackforger/grants/rounds/{id}/projects/{pid}  审批+分配
GET    /hackforger/grants/rounds/{id}/projects/{pid}  详情
```

**Credits**
```
GET    /hackforger/credits/balance                    余额
GET    /hackforger/credits/transactions               流水
GET    /hackforger/credits/redeem/options              兑换列表
POST   /hackforger/credits/redeem                     兑换
GET    /hackforger/credits/redeem/orders              订单列表
POST   /hackforger/credits/admin/deposit              手动发放（Admin）
POST   /hackforger/credits/admin/deduct               手动扣除（Admin）
PUT    /hackforger/credits/redeem/options/{id}        编辑选项（Admin）
POST   /hackforger/credits/redeem/orders/{oid}/fulfill 标记完成（Admin）
```

**Reputation / Search / Assistant / Feed**
```
GET    /hackforger/reputation/users/{username}        声誉
GET    /hackforger/reputation/leaderboard             排行
POST   /hackforger/reputation/recalculate/{username}  重算（Admin）

GET    /hackforger/search?q=&scope=all|bounties|hackathons|grants
POST   /hackforger/assistant/chat                     AI 对话（SSE）

GET    /hackforger/feed?type=following|global|org|hackathon&page=&limit=
```

### 5.3 Webhook 事件

每个 Feed 事件同时触发 Webhook，事件名对应：

```
hackathon: created|status_changed|submission_created|score_submitted|finalized
bounty: created|application_created|claimed|in_review|completed|paid|winners_selected|expired|cancelled
grant: round_created|round_opened|project_submitted|awarded|round_closed|round_finalized
credits: deposited|redeemed
```

### 5.4 Agent 完整 Bounty 流程

```
Publisher            Hunter Agent              Platform
  ├─ POST bounties ──────────────────────────→│
  │                       │  ← Webhook: created│
  │                       ├─ GET /bounties ──→│
  │                       ├─ POST /apply ────→│
  │← Webhook: application │                    │
  ├─ PUT /accept ────────────────────────────→│
  │                       │  ← Webhook: claimed│
  │                       ├─ (code + PR) ───→│
  │── Merge PR ──────────────────────────────→│
  │                       │  (auto: InReview)  │
  ├─ POST /complete ─────────────────────────→│
  │                       │  (auto: credits)   │
  │                       │  ← Webhook: done   │
```

---

## 六、开发计划（3 人并行，5 周）

```
Week 1 ───────────────────────────────────────────
  全员: P0 — 基础设施
    15 张表 + init.go（db.RegisterModel 模式）
    models/forgejo_migrations/ 下创建 migration 文件
    action_types.go（事件常量 + Content 结构体 + String() 映射）
    HackForgerNotifier 骨架注册到 services/notify/
    基础 CRUD Service
    routers 注册 + 前端目录
    options/locale/ 下创建 i18n key 文件

Week 2-3 ─────────────────────────────────────────

  线 A: Bounty 全栈
    后端: 状态机 + PR 联动 + API（含 Swagger 注解）
    Feed集成: 每个状态变更通过 Notifier 触发事件
      → created / claimed / delivered / completed / expired / cancelled / winners_selected
    前端: BountyPanel.vue（Options API + 懒加载挂载） + explore.tmpl + Issue 注入
    模板文本使用 i18n key

  线 B: Hackathon 全栈
    后端: 创建/状态/报名/提交 + API（含 Swagger 注解）
    Feed集成: 每个操作通过 Notifier 触发事件
      → created / registered / submitted / phase_changed
    前端: 展示页 + HackathonRegForm.vue + Explore Tab
    模板文本使用 i18n key

  线 C: Grants + Credits 全栈
    后端: 申请/审批/分配 + 积分全流程 + API（含 Swagger 注解）
    Feed集成: 每个操作通过 Notifier 触发事件
      → round_created / opened / closed / finalized / project_submitted / awarded / redeemed
    前端: Round 页 + GrantAllocator.vue + 积分页
    模板文本使用 i18n key

Week 4 ───────────────────────────────────────────

  线 A → 评审系统
    后端: judge score + API
    Feed: scored / hackathon_finalized 事件
    前端: JudgeScoreCard.vue + 管理页

  线 B → 声誉 + Feed + Profile
    后端:
      reputation service + cron
      feed.go 查询函数（Following/Global/Entity）
      Feed API endpoint
    前端:
      reputation/card.tmpl → Profile 注入
      feed/items.tmpl（HackForger 事件渲染模板）
      Dashboard feeds.tmpl 改造（加 Tab + include）
      home.go 查询切换
    验证:
      Week 2-3 写入的所有 feed 事件可查询和渲染

  线 C → Credits 联动 + 兑换完善
    bounty/hackathon/grant → credits.Deposit 联动
    兑换订单 fulfill 流程

Week 5 ───────────────────────────────────────────
  全员: AI 助手 + 收尾
    搜索 + LLM + PlatformAssistant.vue + ⌘K
    Webhook + Swagger 注解完善 + make swagger-check 通过
    集成测试 + make test-sqlite 覆盖 hackforger 模块
    Test fixtures YAML 文件完成
```

### 代码量

| 类别 | 行数 |
|------|------|
| Go models（含 action_types.go + migrations） | ~1,800 |
| Go services（含 notifier.go + feed.go） | ~2,750 |
| Go routers | ~1,900 |
| Go templates（含 feed/items.tmpl） | ~1,700 |
| Vue + JS | ~1,500 |
| i18n + Swagger 注解 | ~500 |
| **合计** | **~10,150** |

---

## 七、Cron 任务

| 任务 | 间隔 | 功能 | Feed 事件 |
|------|------|------|----------|
| hackathon_status | 5m | 阶段自动切换 | PhaseChanged |
| bounty_expiry | 5m | 过期检查 | BountyExpired |
| reputation_recalc | 1h | 声誉重算 | — |
| grant_deadline | 1h | Round 截止检查 | RoundClosed |

---

## 八、配置项

```ini
[hackforger]
ENABLED = true

[hackforger.hackathon]
MAX_TRACKS_PER_HACKATHON = 10
MAX_CUSTOM_FIELDS = 20
AUTO_CREATE_TEAM_ON_APPROVE = true

[hackforger.bounty]
ALLOWED_CURRENCIES = USD,CNY,EUR,credits
MAX_REWARDS_PER_BOUNTY = 10
AUTO_EXPIRE_CHECK_INTERVAL = 5m

[hackforger.credits]
ENABLED = true
INITIAL_BALANCE = 0

[hackforger.reputation]
RECALC_INTERVAL = 1h
SCORE_WEIGHTS = stars:1,bounties:5,hackathon_wins:10,grants:3

[hackforger.feed]
GLOBAL_EVENTS_IN_FEED = true
ENABLE_MILESTONES = false

[hackforger.assistant]
ENABLED = true
LLM_PROVIDER = openai       # openai | anthropic | ollama
LLM_API_URL = https://api.openai.com/v1
LLM_API_KEY =
LLM_MODEL = gpt-4o-mini
MAX_CONTEXT_TOKENS = 4000
```

---

## 九、原生 vs Fork 判定总结

**直接复用原生（零新代码）：** 代码托管、Org Team 组队、Template Repo Fork、通知、Wiki、Actions CI/CD、Project Board、Star、Reaction、Watch、Profile、搜索、Label、Issue Comment、PR 交付、Issue Template 报名表单、Follow 关系、action 表（Feed 存储）。

**必须 Fork：** Hackathon 实体/展示页、结构化评审、Bounty 金额+状态机+多奖励+多人获奖、Grant 审批+金额分配、积分账本+兑换、声誉聚合、Explore 新 Tab、AI 助手、HackForgerNotifier 注册+Feed 事件发布+Dashboard Following Tab+Feed API。

**原生 + Fork 胶水：** 组队（Organization + 自动加入成员）、交付（Fork + PR merge）、赛道（Repository in Hackathon Org）、时间线（Milestone + 状态机）、社区信号（Star 数 + Grant 管理页展示）。

---

## 十、将来可选扩展

| 功能 | 扩展方式 |
|------|---------|
| QV/QF 投票 | 加 VoteSession + VoteBallot 表 + modules/hackforger/qf/ |
| 防女巫 | 加 modules/hackforger/sybil/ |
| 链上支付 | BountyReward.Type 加 "crypto" |
| 积分转账 | CreditTransaction.Type 已预留 "transfer" |
| 自动算力分配 | credits.redeemed Webhook 接 AutoService |
| 里程碑事件 | 开启 hackforger.feed.ENABLE_MILESTONES |
| Federation | 跨实例 Hackathon/Bounty 发现 |
