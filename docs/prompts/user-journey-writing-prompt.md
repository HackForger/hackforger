# Prompt: 撰写 HackForger User Journey 文档

## 你的任务

为 HackForger 平台撰写完整的 User Journey 文档。这份文档将直接用于：
1. 编写 test fixtures（YAML 测试数据）
2. 编写全部 API integration tests
3. 指导 E2E 测试脚本编写

因此 journey 必须具体到**每一步操作对应的 API endpoint 或 Web action**，而不是抽象的产品描述。

## 项目背景

HackForger 是 Forgejo（自托管 Git Forge）的 Fork，新增 Hackathon、Bounty、Grant、Credits、Feed 模块。

### 核心设计理念：Git-Native

Hackathon 生命周期天然映射到 Git 操作：

| HackForger 概念 | Forgejo 原生实体 | 说明 |
|----------------|-----------------|------|
| Hackathon | Organization | 创建 Hackathon 自动创建 Org |
| Track（赛道） | Repository | 每个赛道是 Org 内的 Repo |
| 报名 | Org Membership | 报名 = 加入 Org |
| 提交作品 | Fork + PR | Fork 赛道 Repo，提交 PR |
| 评审 | PR Review + 数字评分 | Review 文字反馈 + 结构化评分 |
| 阶段截止 | Milestone.Deadline | 四个 Milestone: Registration/Hacking/Judging/Results |
| 阶段标记 | Tag | v0-kickoff / submission-deadline / v1-results |
| 最终成果 | Release | 包含排行榜 + 获奖者 + PR 链接 |

Bounty 挂载在 Issue 上（1:1），继承 Issue 的 Label、Assignee、Comment、PR 关联、Timeline 等能力。

### 社交原语（来自 Forgejo 原生）

这些功能已内置，journey 中应自然使用：
- **Follow** — 关注用户，Feed 中看到其动态
- **Star** — 给 Repo 点赞（Hackathon 中可用作投票信号）
- **Watch** — 订阅 Repo 通知
- **Organization + Team** — 组队机制（Hackathon 报名时自动加入 Org，参赛队伍可创建 Team）
- **Fork** — 参赛者 Fork 赛道 Repo 开始开发
- **Pull Request** — 提交作品 / Bounty 交付
- **Issue** — Bounty 载体
- **Reaction** — 对 Issue/Comment/PR 的 emoji 反馈

## 角色定义

请使用以下固定角色（与 test fixtures 对应）：

| 角色 | 用户名 | 说明 |
|------|--------|------|
| 平台管理员 | `admin` | 全局管理、Credits 手动发放、兑换选项配置 |
| 活动组织者 | `organizer` | 创建 Hackathon/Grant Round，管理评审 |
| Bounty 发布者 | `publisher` | 在 Repo Issue 上创建 Bounty |
| 参赛者 1 | `hacker1` | 活跃参与者，参加 Hackathon + 认领 Bounty |
| 参赛者 2 | `hacker2` | 第二参与者，组队、竞赛对手 |
| 评审 1 | `judge1` | Hackathon 评委 |
| 评审 2 | `judge2` | 第二评委 |
| AI Agent | `agent-bot` | 通过 API 自动操作（PAT 认证） |

## 需要覆盖的 Journey

### Journey 1: Hackathon 全生命周期

从创建到 Finalize + Credits 发放的完整流程，必须体现 Git-Native 设计：

**涉及状态流转：** Draft → Open → Hacking → Judging → Finished

**必须包含的 Git 操作：**
- organizer 创建 Hackathon（自动创建 Org）
- organizer 创建 Track（自动创建 Repo，初始化代码模板）
- organizer 设置评审标准（Criteria）和奖金分配比例
- hacker1 报名（加入 Org）→ Fork Track Repo → 开发 → 提交 PR
- hacker2 报名 → 组队（创建 Team，加入成员）→ Fork → PR
- judge1/judge2 被指派为评委 → 评分（按 Criteria 打分）
- organizer Finalize → 排名 → Credits 自动发放
- 验证 Feed 事件在各阶段正确产生

**社交互动：**
- hacker1 Follow organizer（之后在 Feed 中看到 organizer 的活动）
- 其他用户 Star 赛道 Repo（作为投票信号）
- hacker1 和 hacker2 互相 Follow

### Journey 2: Bounty 全生命周期（两种模式）

#### 2a: Exclusive Bounty（独占模式）
- publisher 在 Repo 创建 Issue → 挂载 Exclusive Bounty + Reward
- hacker1 申请 → publisher Accept → hacker1 认领
- hacker1 Fork → 开发 → 提交 PR → publisher Review
- publisher Complete → Credits 自动发放
- 验证 Bounty 状态机: Open → Claimed → InReview → Completed → Paid

#### 2b: Competitive Bounty（竞赛模式）
- publisher 创建 Competitive Bounty，多个 Reward（1st/2nd/3rd）
- hacker1, hacker2, agent-bot 各自提交方案
- publisher 选择 Winners → Credits 发放
- 验证多人获奖 + 分级奖励

#### 2c: 异常路径
- Bounty 过期（deadline 到期无人完成）
- Bounty 取消（publisher 主动取消）
- 申请被拒绝

### Journey 3: Grant 全生命周期

**涉及状态流转：** Draft → Open → Review → Finalized → Distributed

- organizer 创建 Grant Round（设定预算、截止日期）
- organizer 发布 Round（Draft → Open）
- hacker1 提交 Grant Project 申请
- hacker2 提交另一个申请
- organizer 审批（Approve/Reject）
- organizer 分配金额（Award allocation）
- organizer Finalize Round
- Credits 自动发放给获批项目
- 验证预算约束（总分配 ≤ 预算）

### Journey 4: Credits 流转 + 兑换

- admin 配置兑换选项（RedeemOption）+ 添加 Key pool
- hacker1 通过 Bounty/Hackathon 积累 Credits
- hacker1 查看余额 → 浏览兑换选项 → 下单兑换
- admin 手动 Fulfill 订单（发放 Key）
- admin 手动 Deposit / Deduct 操作
- 验证事务一致性（余额扣减 + 订单创建原子操作）

### Journey 5: Feed + 社交 + 发现

- hacker1 登录后查看 Dashboard Feed（Following Tab）
- 验证 Feed 中包含：所关注用户的行为 + 全局事件
- 通过 Explore 发现 Hackathon / Bounty / Grant
- 搜索功能（按关键词 + scope 筛选）
- 声誉排行榜查看

### Journey 6: AI Agent 参与

- agent-bot 通过 PAT 认证调用 API
- 自动发现 Open Bounties → 申请 → 认领 → 提交 PR
- 注册 Hackathon → 提交作品
- 查看自己的 Credits 余额
- 全程通过 REST API，无 Web 交互

## 输出格式要求

每个 Journey 按以下结构编写：

```markdown
## Journey N: 标题

### 前置条件
- 需要哪些用户存在
- 需要哪些 Repo/Org 存在

### 步骤

#### Step N.1: 步骤描述
- **角色**: 谁执行
- **操作**: 具体动作
- **API**: `METHOD /path` （如果是 API 调用）
- **Web**: 页面路径 （如果是 Web 操作）
- **Git 操作**: Fork/PR/Merge 等（如适用）
- **预期结果**: 状态变化、数据变化、Feed 事件
- **Fixture 暗示**: 这一步在 fixtures 中需要什么数据状态

### 状态快照
在关键转折点标注所有实体的当前状态，便于 fixtures 编写。
```

## 重要约束

1. **必须具体** — 每步标注 API endpoint 或 Web route，不要抽象描述
2. **必须覆盖所有状态** — 每个实体的每个状态值至少出现一次
3. **社交行为自然穿插** — Follow、Star、Watch、Team 不是独立 journey，而是穿插在 Hackathon/Bounty 流程中
4. **Git 操作显式标注** — Fork、PR、Merge、Review 是核心，不是可选项
5. **Feed 事件逐步标注** — 每个状态变更对应的 ActionType 编号和 audience
6. **异常路径必须覆盖** — 过期、取消、拒绝、权限不足
7. **跨 Journey 引用** — Journey 4 的 Credits 来自 Journey 1/2/3 的奖励，数据必须连贯

## 参考：当前已实现的数据模型

### 实体 + 状态

| 实体 | 状态枚举 |
|------|---------|
| Hackathon | Draft(0) → Open(1) → Hacking(2) → Judging(3) → Finished(4) / Cancelled(5) |
| Bounty | Open(0) → Claimed(1) → InReview(2) → Completed(3) → Paid(4) / Expired(5) / Cancelled(6) |
| BountyMode | Exclusive(0) / Competitive(1) |
| BountyApplication | Pending(0) → Accepted(1) / Rejected(2) |
| HackathonRegistration | Pending(0) → Approved(1) / Rejected(2) |
| GrantRound | Draft(0) → Open(1) → Review(2) → Finalized(3) → Distributed(4) / Cancelled(5) |
| GrantProject | Pending → Approved → Funded / Rejected |
| RedeemOrder | pending → fulfilled / cancelled |
| CreditTransaction | deposit / withdraw / redeem / admin_deposit / admin_deduct |

### Feed 事件类型（ActionType 编号）

| 事件 | 编号 | 触发时机 |
|------|------|---------|
| hackathon_created | 30 | organizer 创建 |
| hackathon_registered | 31 | hacker 报名 |
| hackathon_submitted | 32 | hacker 提交作品 |
| hackathon_scored | 33 | judge 评分 |
| bounty_created | 34 | publisher 创建 |
| bounty_claimed | 35 | hacker 认领 |
| bounty_delivered | 36 | PR 提交 |
| bounty_completed | 37 | publisher 确认完成 |
| bounty_winners_selected | 38 | 选出获奖者 |
| grant_round_created | 39 | 创建 Round |
| grant_project_submitted | 40 | 提交申请 |
| grant_awarded | 41 | 分配金额 |
| credits_redeemed | 42 | 积分兑换 |
| bounty_paid | 43 | 标记支付 |
| hackathon_phase_changed | 50 | 阶段切换 |
| hackathon_finalized | 51 | 完成评审 |
| bounty_expired | 52 | 过期 |
| bounty_cancelled | 53 | 取消 |
| grant_round_opened | 54 | 开放申请 |
| grant_round_closed | 55 | 关闭申请 |
| grant_round_finalized | 56 | 完成分配 |
| grant_round_cancelled | 57 | 取消 |
| order_fulfilled | 58 | 订单完成 |
| order_cancelled | 59 | 订单取消 |

### Webhook 事件（与 Feed 对应）

```
hackathon: created | status_changed | submission_created | score_submitted | finalized
bounty: created | application_created | claimed | in_review | completed | paid | winners_selected | expired | cancelled
grant: round_created | round_opened | project_submitted | awarded | round_closed | round_finalized
credits: deposited | redeemed
```

### API 端点列表

**Hackathon:**
```
POST   /api/v1/hackforger/hackathons
GET    /api/v1/hackforger/hackathons
GET    /api/v1/hackforger/hackathons/{id}
PUT    /api/v1/hackforger/hackathons/{id}
DELETE /api/v1/hackforger/hackathons/{id}
POST   /api/v1/hackforger/hackathons/{id}/publish
POST   /api/v1/hackforger/hackathons/{id}/start
POST   /api/v1/hackforger/hackathons/{id}/start-judging
POST   /api/v1/hackforger/hackathons/{id}/finalize
POST   /api/v1/hackforger/hackathons/{id}/tracks
GET    /api/v1/hackforger/hackathons/{id}/tracks
PUT    /api/v1/hackforger/hackathons/{id}/tracks/{trackId}
DELETE /api/v1/hackforger/hackathons/{id}/tracks/{trackId}
POST   /api/v1/hackforger/hackathons/{id}/registrations
GET    /api/v1/hackforger/hackathons/{id}/registrations
PUT    /api/v1/hackforger/hackathons/{id}/registrations/{regId}
POST   /api/v1/hackforger/hackathons/{id}/submissions
GET    /api/v1/hackforger/hackathons/{id}/submissions
GET    /api/v1/hackforger/hackathons/{id}/submissions/{subId}
PUT    /api/v1/hackforger/hackathons/{id}/submissions/{subId}
POST   /api/v1/hackforger/hackathons/{id}/judges
GET    /api/v1/hackforger/hackathons/{id}/judges
DELETE /api/v1/hackforger/hackathons/{id}/judges/{judgeId}
POST   /api/v1/hackforger/hackathons/{id}/scores
GET    /api/v1/hackforger/hackathons/{id}/leaderboard
```

**Bounty:**
```
POST   /api/v1/repos/{owner}/{repo}/hackforger/bounties
GET    /api/v1/repos/{owner}/{repo}/hackforger/bounties
GET    /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}
PUT    /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}
DELETE /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}
POST   /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/rewards
GET    /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/rewards
PUT    /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/rewards/{rid}
DELETE /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/rewards/{rid}
POST   /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/applications
GET    /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/applications
PUT    /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/applications/{aid}/review
POST   /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/complete
POST   /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/cancel
POST   /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/winners
POST   /api/v1/repos/{owner}/{repo}/hackforger/bounties/{id}/pay
GET    /api/v1/hackforger/bounties (global list)
```

**Grants:**
```
POST   /api/v1/hackforger/grants/rounds
GET    /api/v1/hackforger/grants/rounds
GET    /api/v1/hackforger/grants/rounds/{id}
PUT    /api/v1/hackforger/grants/rounds/{id}
DELETE /api/v1/hackforger/grants/rounds/{id}
POST   /api/v1/hackforger/grants/rounds/{id}/open
POST   /api/v1/hackforger/grants/rounds/{id}/close
POST   /api/v1/hackforger/grants/rounds/{id}/finalize
POST   /api/v1/hackforger/grants/rounds/{id}/distribute
POST   /api/v1/hackforger/grants/rounds/{id}/cancel
POST   /api/v1/hackforger/grants/rounds/{id}/projects
GET    /api/v1/hackforger/grants/rounds/{id}/projects
GET    /api/v1/hackforger/grants/rounds/{id}/projects/{pid}
PUT    /api/v1/hackforger/grants/rounds/{id}/projects/{pid}
```

**Credits:**
```
GET    /api/v1/hackforger/credits/balance
GET    /api/v1/hackforger/credits/transactions
GET    /api/v1/hackforger/credits/redeem/options
POST   /api/v1/hackforger/credits/redeem
GET    /api/v1/hackforger/credits/redeem/orders
POST   /api/v1/hackforger/credits/admin/deposit
POST   /api/v1/hackforger/credits/admin/deduct
PUT    /api/v1/hackforger/credits/redeem/options/{id}
POST   /api/v1/hackforger/credits/redeem/orders/{oid}/fulfill
```

**Reputation / Feed / Search:**
```
GET    /api/v1/hackforger/reputation/users/{username}
GET    /api/v1/hackforger/reputation/leaderboard
POST   /api/v1/hackforger/reputation/recalculate/{username}
GET    /api/v1/hackforger/search?q=&scope=all|bounties|hackathons|grants
GET    /api/v1/hackforger/feed?type=following|global|org|hackathon&page=&limit=
```

## 输出位置

将文档保存到 `docs/user-journeys.md`。
