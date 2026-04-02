# 用户旅程全流程 E2E 测试

> **Web 端测试是必选项，不可使用单元测试或 API 测试替代 Web 界面操作。**
> **关键验证节点必须使用 agent-browser 截屏记录作为测试证据。**
> **API 仅用于 git 操作（Fork/PR）和设计上需通过 API 的操作。**

## 测试概述

本测试将 5 个用户旅程融合为一个连贯的叙事，模拟真实的平台使用场景：
一个 Hackathon 从创建到完结的完整生命周期中，自然地涉及 Bounty 协作、
Grant 资助、Credits 兑换和社交互动。

**测试规模**: 11 个阶段（Phase 0-10），约 80+ 步骤
**涉及角色**: Admin/Organizer, Hacker1, Hacker2, Judge1, Judge2
**v0.1 限制**: 仅使用积分（Credits）作为奖励，法币功能关闭

---

## 测试环境

- **服务器**: `http://localhost:3000`（worktree 构建的最新 binary）
- **数据库**: 每轮测试前清空 HackForger 数据（见 `e2e-testing-guide.md` 的 Database Cleanup）
- **截图保存**: `screenshots/full-cycle/`（相对于本文件目录）

## 测试账号与角色映射

| 角色 | Session 名 | 用户名 | 密码 | 本次旅程中的职责 |
|------|-----------|--------|------|----------------|
| Admin/Organizer | hackforger | hackforger | admin1234 | 平台管理、积分充值、兑换选项配置、创建 Hackathon/Bounty/Grant |
| Judge 1 | judge_carol | judge_carol | admin1234 | Hackathon 评委 |
| Judge 2 | judge_dave | judge_dave | admin1234 | Hackathon 评委 |
| Hacker 1 | hacker_eve | hacker_eve | admin1234 | 活跃参赛者、Bounty 发起者/认领者、Grant 申请者 |
| Hacker 2 | hacker_frank | hacker_frank | admin1234 | 第二参赛者、Bounty 协作者、组队伙伴 |

> **密码重置**: 测试开始前，对所有非 admin 账号通过 admin API 执行密码重置（见 e2e-testing-guide.md）。

## 多用户 Session 设置

使用 `agent-browser --session <name>` 为每个角色创建独立浏览器会话：

```bash
# 登录所有用户（一次性完成）
for pair in "hackforger hackforger" "judge_carol judge_carol" "judge_dave judge_dave" "hacker_eve hacker_eve" "hacker_frank hacker_frank"; do
  set -- $pair
  session=$1 user=$2
  agent-browser --session $session open "http://localhost:3000/user/login"
  agent-browser --session $session snapshot -i
  agent-browser --session $session fill @e13 "$user"
  agent-browser --session $session fill @e14 "admin1234"
  agent-browser --session $session click @e17
  agent-browser --session $session wait --load networkidle
done
```

---

## Phase 0: 平台准备

**目标**: Admin 配置平台基础设施，确保积分体系可用。
**角色**: hackforger (Admin)

### Step 0.1: Admin 配置兑换选项 (RedeemOption)
- **角色**: hackforger session
- **UI 路径**: 顶部导航栏 "站点管理" → 左侧菜单 "积分管理" → 点击"兑换选项" Tab → `/-/admin/credits/options`
- **操作**: 点击"创建兑换选项"，填写表单：
  - 名称 = `GPU 算力 - 100 小时`
  - 描述 = `100 小时 A100 GPU 算力`
  - 价格 = `200` Credits
  - 库存 = `10`
  - 激活 = `true`
  - 点击"创建"
- **验证**:
  - 兑换选项创建成功，出现在列表中
  - 显示价格 200、库存 10、状态为已激活
- **截图**: `screenshots/full-cycle/p0-01-redeem-option-created.png`

### Step 0.2: Admin 添加 Key Pool
- **角色**: hackforger session
- **UI 路径**: 兑换选项列表 → 点击 "GPU 算力 - 100 小时" → 进入 Key 管理页面 `/-/admin/credits/options/{id}/keys`
- **操作**: 点击"添加密钥"，输入：
  - Keys = `GPU-KEY-001`, `GPU-KEY-002`, `GPU-KEY-003`（每行一个或批量输入）
  - 点击"添加"
- **验证**:
  - 3 个密钥添加成功，显示在 Key 列表中
  - 状态均为"未使用"
- **截图**: `screenshots/full-cycle/p0-02-keys-added.png`

### Step 0.3: Admin 为 Organizer 充值积分（Hackathon 奖金池）
- **角色**: hackforger session
- **UI 路径**: 顶部导航栏 "站点管理" → 左侧菜单 "积分管理" → `/-/admin/credits`
- **操作**: 在"手动充值"区域填写：
  - 用户名 = `hackforger`（Organizer 与 Admin 同账号）
  - 金额 = `1000`
  - 原因 = `Hackathon prize pool`
  - 点击"充值"按钮
- **验证**:
  - 页面显示充值成功提示
  - hackforger 余额 = 1000
  - 交易类型 = `admin_deposit`
- **截图**: `screenshots/full-cycle/p0-03-credits-deposit-1000.png`

### Phase 0 状态快照

| 实体 | 状态 |
|------|------|
| 兑换选项 "GPU 算力 - 100 小时" | 已创建，200 Credits，库存 10 |
| Key Pool | 3 个密钥（GPU-KEY-001/002/003），未使用 |
| hackforger (Admin/Organizer) Credits | 1000 |
| 其他用户 Credits | 0 |

---

## Phase 1: Hackathon 创建

**目标**: Organizer 创建 Hackathon，配置赛道、评审标准、评委，并发布。
**角色**: hackforger (Organizer)
**来源**: J1 Steps 1.1-1.6

### Step 1.1: Organizer 创建 Hackathon (Draft)
- **角色**: hackforger session
- **UI 路径**: 顶部导航栏 "+" 菜单 → "创建 Hackathon" → `/hackathons/new`
- **操作**: 填写表单：
  - 名称 = `Web3 Innovation Challenge`
  - Slug = `web3-innovation`
  - 描述 = 使用 Markdown 格式编写（Forgejo 原生编辑器）：
    ```
    ## Web3 Innovation Challenge
    Build the future of decentralized web.
    - DeFi 协议
    - NFT 工具链
    ```
  - 最大赛道数 = `3`
  - 奖金池（积分）= `800`
  - 点击"创建"按钮
- **验证**:
  - 跳转到 Hackathon 详情页 `/hackathon/web3-innovation`，状态 = `Draft`
  - Forgejo 组织 `web3-innovation` 已自动创建，hackforger 为 owner
  - 法币奖金字段不显示（v0.1 仅积分）
  - Feed 事件: `hackathon_created(30)` -- 全局可见
- **截图**: `screenshots/full-cycle/p1-01-hackathon-created.png`

### Step 1.2: Organizer 创建赛道 (Track)
- **角色**: hackforger session
- **UI 路径**: Hackathon 详情页 → 点击"管理"按钮 → 进入管理页面 `/hackathon/web3-innovation/manage`
- **操作**:
  - 在"赛道管理"区域点击"添加赛道"
  - 填写 Track 1：名称 = `Web Track`，Slug = `web-track`，描述 = `Build web applications and tools`，奖金分配（积分）= `300`，提交
  - 再次点击"添加赛道"
  - 填写 Track 2：名称 = `DeFi Track`，Slug = `defi-track`，描述 = `Build DeFi protocols and tooling`，奖金分配（积分）= `500`，提交
- **验证**:
  - 赛道列表显示 2 个条目
  - 自动创建 Repo `web3-innovation/web-track` 和 `web3-innovation/defi-track`，包含初始 README
  - 奖金分配总计（300 + 500 = 800）<= 奖金池（800）
- **截图**: `screenshots/full-cycle/p1-02-tracks-created.png`

### Step 1.3: Organizer 设置评审标准
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage` → "评审标准"区域
- **操作**:
  - 为 Web Track 添加评审标准：
    - 点击"添加标准"，填写：名称 = `Innovation`，权重 = `40`，描述 = `Novelty of approach`，提交
    - 点击"添加标准"，填写：名称 = `Technical Quality`，权重 = `35`，描述 = `Code quality, architecture`，提交
    - 点击"添加标准"，填写：名称 = `Presentation`，权重 = `25`，描述 = `Demo and documentation`，提交
  - 为 DeFi Track 执行相同操作（或将已有标准分配给该赛道）
- **验证**:
  - 每个赛道的评审标准列表显示 3 个条目
  - 每个赛道权重之和 = 100
- **截图**: `screenshots/full-cycle/p1-03-criteria-set.png`

### Step 1.4: Organizer 指派评委
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage` → "评委管理"区域
- **操作**:
  - 在"添加评委"输入框中输入 `judge_carol`，点击"添加"
  - 再输入 `judge_dave`，点击"添加"
- **验证**:
  - 评委列表显示 `judge_carol` 和 `judge_dave`
- **截图**: `screenshots/full-cycle/p1-04-judges-assigned.png`

### Step 1.5: Organizer 发布 Hackathon (Draft -> Open)
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage` → 状态管理区域
- **操作**: 点击"发布"按钮（POST `/hackathon/web3-innovation/manage/publish`）
- **验证**:
  - 前置检查通过：至少 1 个赛道、至少 1 个评委
  - 状态变更：`Draft(0)` -> `Open(1)`
  - Feed 事件: `hackathon_phase_changed(50)` -- 全局可见
  - 在 Explore 页面 `/explore/hackathons` 中可搜索到该 Hackathon
- **截图**: `screenshots/full-cycle/p1-05-published.png`

### Phase 1 状态快照

| 实体 | 状态 |
|------|------|
| Hackathon "Web3 Innovation Challenge" | `Open(1)` |
| Web Track | 已创建，奖金 300 |
| DeFi Track | 已创建，奖金 500 |
| 评审标准 | 每赛道 3 项（Innovation 40%, Technical Quality 35%, Presentation 25%）|
| 评委 | judge_carol, judge_dave |
| hackforger Credits | 1000 |

---

## Phase 2: 报名 + 社交

**目标**: 参赛者报名，建立社交关系，组队协作。包含约束验证（评委不能报名）。
**角色**: hacker_eve, hacker_frank, judge_carol
**来源**: J1 Steps 1.7-1.9 + J5 社交

### Step 2.1: Hacker1 关注 Organizer
- **角色**: hacker_eve session
- **UI 路径**: 顶部导航栏搜索框输入 `hackforger` → 点击搜索结果进入 hackforger 个人页面 `/hackforger`
- **操作**: 点击页面上的 "Follow" 按钮
- **验证**:
  - 按钮变为 "Unfollow"
  - hacker_eve 的 Following Feed 中将显示 hackforger 未来的活动
- **截图**: `screenshots/full-cycle/p2-01-follow-organizer.png`

### Step 2.2: Hacker1 报名 Web Track
- **角色**: hacker_eve session
- **UI 路径**: 导航到 Hackathon 详情页 `/hackathon/web3-innovation` → 报名区域
- **操作**: 选择赛道 = `Web Track`，点击"报名"按钮（POST `/hackathon/web3-innovation/register`）
- **验证**:
  - 报名成功，状态 = `Approved(1)`（自动审批）
  - hacker_eve 被添加为组织 `web3-innovation` 的成员
  - Feed 事件: `hackathon_registered(31)` -- 粉丝 + 组织可见
- **截图**: `screenshots/full-cycle/p2-02-hacker1-registered.png`

### Step 2.3: Hacker2 报名 DeFi Track
- **角色**: hacker_frank session
- **UI 路径**: 导航到 Hackathon 详情页 `/hackathon/web3-innovation` → 报名区域
- **操作**: 选择赛道 = `DeFi Track`，点击"报名"按钮
- **验证**:
  - 报名成功，hacker_frank 成为 `web3-innovation` 成员
  - Feed 事件: `hackathon_registered(31)` for hacker_frank
- **截图**: `screenshots/full-cycle/p2-03-hacker2-registered.png`

### Step 2.4: Hacker1 与 Hacker2 互相关注
- **角色**: hacker_eve session + hacker_frank session
- **UI 路径**:
  - hacker_eve session: 导航到 `/hacker_frank` → 点击 "Follow"
  - hacker_frank session: 导航到 `/hacker_eve` → 点击 "Follow"
- **操作**: 各自点击对方个人主页的 "Follow" 按钮
- **验证**:
  - hacker_eve <-> hacker_frank 互相关注
  - 两人的 Following Feed 中将显示对方的活动
- **截图**: `screenshots/full-cycle/p2-04-mutual-follow.png`

### Step 2.5: 组队 -- 在 Hackathon Org 内创建 Team
- **角色**: hackforger session（Organizer 管理组织）
- **UI 路径**: 导航到组织页面 `/org/web3-innovation/settings` → 点击"Teams" Tab → 点击"创建 Team"
- **操作**:
  - 通过 Forgejo 原生 API 创建 Team（组织团队管理为 Forgejo 原生概念）：
    - API: `POST /api/v1/orgs/web3-innovation/teams` → `{ "name": "DeFi Duo", "permission": "write" }`
    - API: `PUT /api/v1/teams/<team_id>/members/hacker_eve`
    - API: `PUT /api/v1/teams/<team_id>/members/hacker_frank`
- **验证**:
  - Team "DeFi Duo" 创建成功
  - hacker_eve 和 hacker_frank 均为成员
- **截图**: `screenshots/full-cycle/p2-05-team-created.png`

### Step 2.6: [约束验证] Judge 尝试报名 -> 应被拒绝
- **角色**: judge_carol session
- **UI 路径**: 导航到 Hackathon 详情页 `/hackathon/web3-innovation` → 报名区域
- **操作**: 选择赛道 = `Web Track`，点击"报名"按钮
- **验证**:
  - 系统拒绝报名，显示错误提示（评委不能同时报名参赛）
  - judge_carol 的报名状态未改变
  - 报名列表中不包含 judge_carol
- **截图**: `screenshots/full-cycle/p2-06-judge-register-denied.png`

### Phase 2 状态快照

| 实体 | 状态 |
|------|------|
| Hackathon | `Open(1)` |
| hacker_eve | 已报名 Web Track，关注 hackforger + hacker_frank |
| hacker_frank | 已报名 DeFi Track，关注 hacker_eve |
| Team "DeFi Duo" | 创建于 web3-innovation，成员：hacker_eve + hacker_frank |
| judge_carol | 报名被拒绝（评委身份冲突）|

---

## Phase 3: Hacking 阶段

**目标**: Organizer 启动 Hacking 阶段，包含约束验证。
**角色**: hackforger (Organizer)
**来源**: J1 Step 1.10

### Step 3.1: Organizer 启动 Hacking (Open -> Hacking)
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage` → 状态管理区域
- **操作**: 点击"开始 Hacking"按钮（POST `/hackathon/web3-innovation/manage/start`）
- **验证**:
  - 状态变更：`Open(1)` -> `Hacking(2)`
  - Feed 事件: `hackathon_phase_changed(50)` -- 全局 + 组织可见
- **截图**: `screenshots/full-cycle/p3-01-hacking-started.png`

### Step 3.2: [约束验证] 重复发布 -> 应被拒绝
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage`
- **操作**: 尝试再次点击"发布"按钮（如果仍可见），或通过浏览器 fetch 发送 POST `/hackathon/web3-innovation/manage/publish`
- **验证**:
  - 系统拒绝操作，返回错误（已经处于 Hacking 状态，不能回退到 Open）
  - 状态保持 `Hacking(2)` 不变
- **截图**: `screenshots/full-cycle/p3-02-duplicate-publish-denied.png`

---

## Phase 4: 开发 + Bounty 协作

**目标**: 参赛者开始开发，过程中通过 Bounty 进行协作。Hacker1 在赛道 Repo 创建 Issue 并发起 Bounty，Hacker2 申请并完成，积分自动发放。
**角色**: hacker_eve, hacker_frank, hackforger
**来源**: J1 Steps 1.11-1.12 + J2 Steps 2a.1-2a.9（重组整合）

### Step 4.1: Hacker1 Fork Web Track Repo 开始开发
- **角色**: hacker_eve session
- **UI 路径**: 导航到 Web Track Repo 页面 `/web3-innovation/web-track` → 点击"Fork"按钮
- **操作**:
  - 在 Fork 确认页面选择 Fork 到个人账号 (hacker_eve)，点击确认
  - Fork 创建完成后，进入 `hacker_eve/web-track`
  - 点击 README.md → 点击编辑按钮 → 使用 Markdown 添加项目描述：
    ```
    ## My Web App Project
    A decentralized web application toolkit.
    ```
  - 提交 commit
- **验证**:
  - Fork `hacker_eve/web-track` 存在，且包含新 commit
- **截图**: `screenshots/full-cycle/p4-01-hacker1-fork-web.png`

### Step 4.2: Hacker1 在赛道 Repo 创建 Issue "需要 UI 设计帮助"
- **角色**: hacker_eve session
- **UI 路径**: 导航到赛道 Repo `/web3-innovation/web-track` → 点击 "Issues" Tab → 点击"新建 Issue"
- **操作**: 填写标题 = `Need UI design help for dashboard`，内容（Markdown）= `Looking for someone to help design the dashboard UI. Requirements: responsive layout, dark mode support.`，点击"提交 Issue"
- **验证**:
  - Issue #1 创建成功，显示在 Issue 列表
- **截图**: `screenshots/full-cycle/p4-02-issue-created.png`

### Step 4.3: Hacker1 在该 Issue 上创建 Exclusive Bounty
- **角色**: hacker_eve session
- **UI 路径**: 导航到 Bounty 创建页面 `/web3-innovation/web-track/bounties/new`
- **操作**: 填写表单：
  - 关联 Issue = Issue #1 (`Need UI design help for dashboard`)
  - 模式 = `Exclusive`
  - 描述 = `Design a responsive dashboard UI with dark mode support`
  - 奖励金额（积分）= `100`
  - 截止日期 = `2026-05-15`
  - 点击"创建 Bounty"
- **Git 操作**: 无
- **验证**:
  - Bounty 创建成功，状态 = `Open(0)`，模式 = `Exclusive(0)`
  - Bounty 与 Issue #1 关联（1:1）
  - 注意：此时积分由 Hacker1 的余额（目前为 0）支付。如果系统要求余额不足，需先由 Admin 为 hacker_eve 充值。
    - **备选**：由 Organizer (hackforger) 创建此 Bounty，从 Organizer 余额 escrow
  - Feed 事件: `bounty_created(34)` -- 全局 + Repo watcher 可见
- **截图**: `screenshots/full-cycle/p4-03-bounty-created.png`

> **注意**: 如果 hacker_eve 余额不足支付 Bounty escrow，可由 hackforger (Admin) 先为 hacker_eve 充值 100 积分，或由 hackforger 作为 Bounty 发起者。以下步骤假设 Bounty 已成功创建并 escrow。

### Step 4.4: Admin 为 Hacker1 充值（如需 Bounty escrow）
- **角色**: hackforger session
- **UI 路径**: 站点管理 → 积分管理 `/-/admin/credits`
- **操作**: 在"手动充值"区域填写：用户名 = `hacker_eve`，金额 = `100`，原因 = `Bounty escrow for UI design task`，点击"充值"
- **验证**:
  - hacker_eve 余额 = 100
  - 交易类型 = `admin_deposit`
- **截图**: `screenshots/full-cycle/p4-04-hacker1-deposit.png`

### Step 4.5: Hacker2 浏览 Bounty 列表并申请
- **角色**: hacker_frank session
- **UI 路径**: 导航到 Explore 页面 → 点击 "Bounties" Tab `/explore/bounties` → 找到 "Need UI design help for dashboard" → 点击进入 Bounty 详情页
- **操作**: 点击"申请认领"按钮，填写申请说明 = `I'm experienced in UI/UX design. Can deliver within 2 days.`，提交
- **验证**:
  - 申请创建成功，状态 = `Pending(0)`
- **截图**: `screenshots/full-cycle/p4-05-bounty-application.png`

### Step 4.6: Hacker1 接受申请 (Bounty: Open -> Claimed)
- **角色**: hacker_eve session
- **UI 路径**: 导航到 Bounty 申请列表 `/web3-innovation/web-track/bounties/{bounty_id}/applications`
- **操作**: 找到 hacker_frank 的申请，点击"接受"按钮（POST `/web3-innovation/web-track/bounties/{bounty_id}/applications/{aid}`），附加评论 = `Great, go ahead!`
- **验证**:
  - 申请状态：`Pending(0)` -> `Accepted(1)`
  - Bounty 状态：`Open(0)` -> `Claimed(1)`
  - hacker_frank 成为独占认领者
  - Feed 事件: `bounty_claimed(35)` -- Repo watcher + 粉丝可见
- **截图**: `screenshots/full-cycle/p4-06-application-accepted.png`

### Step 4.7: Hacker2 Fork Repo 并提交 UI 设计
- **角色**: hacker_frank session
- **UI 路径**: 导航到赛道 Repo `/web3-innovation/web-track` → 点击"Fork"按钮
- **操作**:
  - Fork 到个人账号 (hacker_frank)，确认
  - 进入 `hacker_frank/web-track`
  - 点击"创建新文件"→ 文件名 = `design/dashboard.md` → 使用 Markdown 编写 UI 设计文档：
    ```
    ## Dashboard UI Design
    - Responsive grid layout
    - Dark mode toggle
    - Card-based component structure
    ```
  - 提交 commit
- **验证**:
  - Fork `hacker_frank/web-track` 存在，包含设计 commit
- **截图**: `screenshots/full-cycle/p4-07-hacker2-fork-design.png`

### Step 4.8: Hacker2 创建 PR (Bounty: Claimed -> InReview)
- **角色**: hacker_frank session
- **UI 路径**: 导航到 `hacker_frank/web-track` → 点击"新建 Pull Request"
- **操作**: 填写 PR：
  - Base Repo = `web3-innovation/web-track`，Base Branch = `main`
  - Head Repo = `hacker_frank/web-track`，Head Branch = `main`
  - 标题 = `feat: dashboard UI design`
  - 描述 = `Closes #1 -- Dashboard UI design with responsive layout and dark mode`
  - 点击"创建 Pull Request"
- **验证**:
  - PR 创建成功，关联 Issue #1
  - Bounty 状态：`Claimed(1)` -> `InReview(2)`
  - Feed 事件: `bounty_delivered(36)` -- Repo watcher 可见
- **截图**: `screenshots/full-cycle/p4-08-bounty-pr.png`

### Step 4.9: Hacker1 Review + Complete Bounty (InReview -> Completed -> Paid)
- **角色**: hacker_eve session
- **UI 路径**: 导航到 PR 页面 → Review PR → 然后通过 Bounty 操作按钮完成
- **操作**:
  - 在 PR 页面添加 Review（Approve）
  - 通过 Bounty 操作按钮标记为 Complete（POST `/web3-innovation/web-track/bounties/{bounty_id}/complete`）
  - 然后点击"支付"按钮（POST `/web3-innovation/web-track/bounties/{bounty_id}/pay`）
- **验证**:
  - Bounty 状态：`InReview(2)` -> `Completed(3)` -> `Paid(4)`
  - 托管积分释放给 hacker_frank：`escrow_release` 交易
  - hacker_frank 余额：0 + 100 = 100
  - Feed 事件: `bounty_completed(37)` + `bounty_paid(43)`
- **截图**: `screenshots/full-cycle/p4-09-bounty-completed-paid.png`

### Step 4.10: Hacker2 查看积分余额变化
- **角色**: hacker_frank session
- **UI 路径**: 顶部导航栏用户菜单 → 点击"积分" → 进入积分概览页 `/credits`
- **操作**: 查看余额和交易记录
- **验证**:
  - 余额 = 100
  - 交易记录显示：`escrow_release` +100（Bounty UI 设计奖励）
- **截图**: `screenshots/full-cycle/p4-10-hacker2-credits.png`

### Phase 4 状态快照

| 实体 | 状态 |
|------|------|
| Bounty #1 (Exclusive, web-track) | `Paid(4)` |
| hacker_eve Credits | 100 - 100 (escrow) + 0 = 0（或如 Admin 充值了 100 则余额变化取决于 escrow 流程）|
| hacker_frank Credits | 100 |
| web3-innovation/web-track | 有 1 个 PR + 1 个 Issue |

---

## Phase 5: Grant 资助

**目标**: Organizer 创建 Grant Round，参赛者申请资助，Organizer 审批并发放积分。
**角色**: hackforger (Organizer), hacker_eve, hacker_frank
**来源**: J3 Steps 3.0-3.8

### Step 5.1: Admin 为 Organizer 额外充值（Grant 预算）
- **角色**: hackforger session
- **UI 路径**: 站点管理 → 积分管理 `/-/admin/credits`
- **操作**: 在"手动充值"区域填写：用户名 = `hackforger`，金额 = `500`，原因 = `DeFi Track Boost Grant budget`，点击"充值"
- **验证**:
  - hackforger 余额增加 500
- **截图**: `screenshots/full-cycle/p5-01-grant-budget-deposit.png`

### Step 5.2: Organizer 创建 Grant Round (Draft)
- **角色**: hackforger session
- **UI 路径**: 导航到 Grant 创建页面 `/grants/new`
- **操作**: 填写表单：
  - 名称 = `DeFi Track Boost`
  - Slug = `defi-track-boost`
  - 描述（Markdown）= `额外资助 DeFi 赛道的参赛者，帮助提升项目质量。`
  - 预算（积分）= `500`
  - 截止日期 = `2026-06-01`
  - 点击"创建"
- **验证**:
  - Grant Round 创建成功，状态 = `Draft(0)`
  - Feed 事件: `grant_round_created(39)` -- 全局可见
- **截图**: `screenshots/full-cycle/p5-02-grant-round-created.png`

### Step 5.3: Organizer 开放申请 (Draft -> Open)
- **角色**: hackforger session
- **UI 路径**: Grant Round 管理页面 `/grants/defi-track-boost/manage`
- **操作**: 点击"开放申请"按钮（POST `/grants/defi-track-boost/manage/open`）
- **验证**:
  - 状态变更：`Draft(0)` -> `Open(1)`
  - 现在接受项目申请
  - Feed 事件: `grant_round_opened(54)` -- 全局可见
- **截图**: `screenshots/full-cycle/p5-03-grant-opened.png`

### Step 5.4: Hacker1 提交 Grant 项目申请
- **角色**: hacker_eve session
- **UI 路径**: 导航到 Grant Round 详情页 `/grants/defi-track-boost` → 点击"提交项目" → 进入提交页面 `/grants/defi-track-boost/submit`
- **操作**: 填写表单：
  - 标题 = `DeFi Automation Toolkit`
  - 描述（Markdown）= `Open-source toolkit for automating DeFi operations. Includes smart contract templates and testing utilities.`
  - 申请金额 = `300`
  - Repo URL = `http://localhost:3000/hacker_eve/web-track`
  - 点击"提交"
- **验证**:
  - Grant Project 创建成功，状态 = `Pending`
  - Feed 事件: `grant_project_submitted(40)` -- 全局 + 粉丝可见
- **截图**: `screenshots/full-cycle/p5-04-project1-submitted.png`

### Step 5.5: Hacker2 提交 Grant 项目申请
- **角色**: hacker_frank session
- **UI 路径**: 导航到 Grant Round 详情页 → 提交页面 `/grants/defi-track-boost/submit`
- **操作**: 填写表单：
  - 标题 = `NFT Bridge Connector`
  - 描述 = `Cross-chain NFT bridge implementation`
  - 申请金额 = `400`
  - Repo URL = `http://localhost:3000/hacker_frank/web-track`
  - 点击"提交"
- **验证**:
  - Grant Project 创建成功，状态 = `Pending`
  - Feed 事件: `grant_project_submitted(40)`
- **截图**: `screenshots/full-cycle/p5-05-project2-submitted.png`

### Step 5.6: Organizer 关闭申请 (Open -> Review)
- **角色**: hackforger session
- **UI 路径**: Grant Round 管理页面 `/grants/defi-track-boost/manage`
- **操作**: 点击"关闭申请"按钮（POST `/grants/defi-track-boost/manage/close`）
- **验证**:
  - 状态变更：`Open(1)` -> `Review(2)`
  - 不再接受新的项目申请
  - Feed 事件: `grant_round_closed(55)`
- **截图**: `screenshots/full-cycle/p5-06-grant-closed.png`

### Step 5.7: Organizer 审批项目（Approve Hacker1, Reject Hacker2）
- **角色**: hackforger session
- **UI 路径**: Grant Round 管理页面 → 项目列表 → 点击每个项目进入项目管理页 `/grants/defi-track-boost/manage/projects/{pid}`
- **操作**:
  - 项目 1 (DeFi Automation Toolkit - hacker_eve)：
    - 点击"批准"按钮（POST `.../approve`）
    - 设置 Award 金额 = `300`（POST `.../award`），附评论 = `Strong proposal, aligned with track goals`
  - 项目 2 (NFT Bridge Connector - hacker_frank)：
    - 点击"拒绝"按钮（POST `.../reject`），附评论 = `Exceeds remaining budget, please reapply next round`
- **验证**:
  - 项目 1：`Pending` -> `Approved`，拨款 300 Credits
  - 项目 2：`Pending` -> `Rejected`
  - 总分配（300）<= 预算（500）
  - Feed 事件: `grant_awarded(41)` for project 1
- **截图**: `screenshots/full-cycle/p5-07-projects-reviewed.png`

### Step 5.8: Organizer Finalize + Distribute
- **角色**: hackforger session
- **UI 路径**: Grant Round 管理页面 `/grants/defi-track-boost/manage`
- **操作**:
  - 点击"锁定拨款"按钮（POST `/grants/defi-track-boost/manage/finalize`）
  - 确认后点击"发放积分"按钮（POST `/grants/defi-track-boost/manage/distribute`）
- **验证**:
  - 状态变更：`Review(2)` -> `Finalized(3)` -> `Distributed(4)`
  - 积分转账：
    - hacker_eve: +300
    - hackforger: -300
  - 项目 1 状态：`Approved` -> `Funded`
  - Feed 事件: `grant_round_finalized(56)`
- **截图**: `screenshots/full-cycle/p5-08-grant-distributed.png`

### Step 5.9: Hacker1 查看积分余额变化
- **角色**: hacker_eve session
- **UI 路径**: 顶部导航栏用户菜单 → 点击"积分" → `/credits`
- **操作**: 查看余额和交易记录
- **验证**:
  - 余额增加了 300（Grant 资助）
  - 交易记录显示：`deposit` +300（Grant "DeFi Track Boost" 资助）
- **截图**: `screenshots/full-cycle/p5-09-hacker1-credits-grant.png`

### Phase 5 状态快照

| 实体 | 状态 |
|------|------|
| Grant Round "DeFi Track Boost" | `Distributed(4)` |
| Grant Project 1 (hacker_eve) | `Funded` |
| Grant Project 2 (hacker_frank) | `Rejected` |
| hacker_eve Credits | +300（Grant 资助）|
| hacker_frank Credits | 100（不变）|

---

## Phase 6: 提交参赛作品

**目标**: 展示两种提交模式 -- Fork+PR 和 Link Repo。Star 赛道 Repo 作为社交信号。
**角色**: hacker_eve, hacker_frank, judge_carol
**来源**: J1 Steps 1.11-1.15

### Step 6.1: Hacker1 在 Fork 中继续开发
- **角色**: hacker_eve session
- **UI 路径**: 导航到 `hacker_eve/web-track` → 编辑文件
- **操作**:
  - 点击"创建新文件"→ 文件名 = `src/app.md` → 使用 Markdown 编写应用描述：
    ```
    ## Web App Toolkit
    Core module implementing:
    - Component library
    - State management utilities
    - API integration layer
    ```
  - 提交 commit（message = `feat: add core app module`）
- **验证**:
  - 新文件已提交到 `hacker_eve/web-track`
- **截图**: `screenshots/full-cycle/p6-01-hacker1-develop.png`

### Step 6.2: Hacker1 通过 Fork+PR 模式提交作品
- **角色**: hacker_eve session
- **UI 路径**: 导航到 `hacker_eve/web-track` → 点击"新建 Pull Request"
- **操作**:
  - 选择 Base Repo = `web3-innovation/web-track`，Base Branch = `main`
  - Head Repo = `hacker_eve/web-track`，Head Branch = `main`
  - 标题 = `Web App Toolkit - hacker_eve submission`
  - 描述（Markdown）= `My web application toolkit submission for the Web Track.`
  - 点击"创建 Pull Request"
  - PR 创建成功后，导航到 Hackathon 提交页面 `/hackathon/web3-innovation/submit`
  - 选择赛道 = `Web Track`
  - 填写标题 = `Web App Toolkit`
  - 填写描述 = `A modular web application toolkit with component library and state management`
  - 关联 PR（选择刚创建的 PR）
  - 点击"提交"
- **验证**:
  - PR 创建在 `web3-innovation/web-track`
  - Submission 已创建并关联 PR（可追溯时间戳、可 diff、可 review）
  - Feed 事件: `hackathon_submitted(32)` -- 组织 + 粉丝可见
- **截图**: `screenshots/full-cycle/p6-02-hacker1-submission-pr.png`

### Step 6.3: Hacker2 Fork DeFi Track Repo 并开发
- **角色**: hacker_frank session
- **UI 路径**: 导航到 DeFi Track Repo `/web3-innovation/defi-track` → 点击"Fork"按钮
- **操作**:
  - Fork 到个人账号 (hacker_frank)，确认
  - 进入 `hacker_frank/defi-track`
  - 点击"创建新文件"→ 文件名 = `protocol/defi-core.md` → 使用 Markdown 编写项目描述
  - 提交 commit
- **验证**:
  - Fork `hacker_frank/defi-track` 存在，包含开发 commit
- **截图**: `screenshots/full-cycle/p6-03-hacker2-fork-defi.png`

### Step 6.4: Hacker2 通过 Link Repo 模式提交作品
- **角色**: hacker_frank session
- **UI 路径**: 导航到 Hackathon 提交页面 `/hackathon/web3-innovation/submit`
- **操作**:
  - 选择赛道 = `DeFi Track`
  - 填写标题 = `DeFi Yield Aggregator`
  - 填写描述 = `Cross-chain yield aggregator by Team DeFi Duo`
  - 提交模式选择"Link 已有 Repo"（如果 UI 支持），填写 Repo 链接 = `http://localhost:3000/hacker_frank/defi-track`
  - 或者：通过 Fork+PR 模式，从 `hacker_frank/defi-track` 创建 PR 到 `web3-innovation/defi-track`，然后关联 PR
  - 点击"提交"
- **验证**:
  - DeFi Track 上出现 Submission
  - Feed 事件: `hackathon_submitted(32)` for hacker_frank
- **截图**: `screenshots/full-cycle/p6-04-hacker2-submission-link.png`

### Step 6.5: 多用户 Star 赛道 Repo（社交信号）
- **角色**: hacker_eve session + judge_carol session
- **UI 路径**:
  - hacker_eve session: 导航到 `/web3-innovation/defi-track` → 点击 Star 按钮
  - judge_carol session: 导航到 `/web3-innovation/web-track` → 点击 Star 按钮
- **操作**: 各自点击 Star 按钮
- **验证**:
  - 赛道 Repo 的 Star 数增加
  - Star 在 Repo 页面可见，作为社区兴趣信号
- **截图**: `screenshots/full-cycle/p6-05-star-repos.png`

### Phase 6 状态快照

| 实体 | 状态 |
|------|------|
| Hackathon | `Hacking(2)` |
| Web Track | 1 个 Submission（hacker_eve，Fork+PR 模式）|
| DeFi Track | 1 个 Submission（hacker_frank，Link Repo 模式）|
| Stars | web3-innovation/defi-track 1 star, web3-innovation/web-track 1 star |

---

## Phase 7: 评审 + Finalize

**目标**: Organizer 启动评审，两位评委独立打分，Organizer 查看排名并结算，积分自动发放。
**角色**: hackforger (Organizer), judge_carol, judge_dave, hacker_eve, hacker_frank
**来源**: J1 Steps 1.16-1.20

### Step 7.1: Organizer 启动评审 (Hacking -> Judging)
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage` → 状态管理区域
- **操作**: 点击"开始评审"按钮（POST `/hackathon/web3-innovation/manage/judge`）
- **验证**:
  - 状态变更：`Hacking(2)` -> `Judging(3)`
  - 不再接受新提交
  - Feed 事件: `hackathon_phase_changed(50)`
- **截图**: `screenshots/full-cycle/p7-01-judging-started.png`

### Step 7.2: Judge1 为所有提交评分
- **角色**: judge_carol session
- **UI 路径**: 导航到评审页面 `/hackathon/web3-innovation/judge`
- **操作**:
  - 页面列出所有待评分的 Submission
  - 为 hacker_eve 的 "Web App Toolkit"（Web Track）评分：
    - Innovation = `90`，Technical Quality = `85`，Presentation = `80`
    - 点击"提交评分"（POST `/hackathon/web3-innovation/judge/{sid}/scores`）
  - 为 hacker_frank 的 "DeFi Yield Aggregator"（DeFi Track）评分：
    - Innovation = `75`，Technical Quality = `80`，Presentation = `85`
    - 点击"提交评分"
- **验证**:
  - 评分保存成功
  - hacker_eve 加权分 (judge_carol)：90x0.4 + 85x0.35 + 80x0.25 = 85.75
  - hacker_frank 加权分 (judge_carol)：75x0.4 + 80x0.35 + 85x0.25 = 79.25
  - Feed 事件: `hackathon_scored(33)` x2 -- 组织可见
- **截图**: `screenshots/full-cycle/p7-02-judge1-scores.png`

### Step 7.3: Judge2 为所有提交评分
- **角色**: judge_dave session
- **UI 路径**: 导航到评审页面 `/hackathon/web3-innovation/judge`
- **操作**:
  - 为 hacker_eve 的提交评分：
    - Innovation = `88`，Technical Quality = `90`，Presentation = `75`
    - 提交评分
  - 为 hacker_frank 的提交评分：
    - Innovation = `80`，Technical Quality = `78`，Presentation = `90`
    - 提交评分
- **验证**:
  - hacker_eve 加权分 (judge_dave)：88x0.4 + 90x0.35 + 75x0.25 = 85.45
  - hacker_frank 加权分 (judge_dave)：80x0.4 + 78x0.35 + 90x0.25 = 81.80
  - hacker_eve 平均分：(85.75 + 85.45) / 2 = **85.60**（排名 #1）
  - hacker_frank 平均分：(79.25 + 81.80) / 2 = **80.53**（排名 #2）
  - Feed 事件: `hackathon_scored(33)` x2
- **截图**: `screenshots/full-cycle/p7-03-judge2-scores.png`

### Step 7.4: Organizer 查看排名预览
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage` → 点击"结算预览" → `/hackathon/web3-innovation/manage/finalize-preview`
- **操作**: 查看排名预览页面
- **验证**:
  - 排名预览显示：
    - #1: hacker_eve, 85.60 分, 奖金（积分）500（Web Track 冠军？或按总奖金分配）
    - #2: hacker_frank, 80.53 分, 奖金（积分）300
  - 法币字段不显示
- **截图**: `screenshots/full-cycle/p7-04-finalize-preview.png`

### Step 7.5: Organizer 确认结算 (Judging -> Finished)
- **角色**: hackforger session
- **UI 路径**: 结算预览页面 → 点击"确认结算"按钮（POST `/hackathon/web3-innovation/manage/finalize`）
- **操作**: 确认结算
- **验证**:
  - 状态变更：`Judging(3)` -> `Finished(4)`
  - 排名锁定并公开
  - 积分自动发放：
    - hacker_eve: +500 Credits（第 1 名，Web Track 奖金 300 + 排名奖金分配）
    - hacker_frank: +300 Credits（第 2 名）
    - hackforger (organizer): 扣除对应积分
  - 排行榜冻结并发布
  - Feed 事件: `hackathon_finalized(51)` -- 全局可见
- **截图**: `screenshots/full-cycle/p7-05-finalized.png`

### Step 7.6: Hacker1 查看积分余额
- **角色**: hacker_eve session
- **UI 路径**: 导航到积分概览页 `/credits`
- **操作**: 查看余额变化
- **验证**:
  - 余额增加了 Hackathon 奖金（具体金额取决于奖金分配规则）
  - 交易记录显示：`deposit` +XXX（Hackathon Web3 Innovation 第 1 名）
- **截图**: `screenshots/full-cycle/p7-06-hacker1-credits-hackathon.png`

### Step 7.7: Hacker2 查看积分余额
- **角色**: hacker_frank session
- **UI 路径**: 导航到积分概览页 `/credits`
- **操作**: 查看余额变化
- **验证**:
  - 余额增加了 Hackathon 奖金
  - 交易记录显示新的 `deposit` 条目
- **截图**: `screenshots/full-cycle/p7-07-hacker2-credits-hackathon.png`

### Step 7.8: 查看排行榜
- **角色**: hacker_eve session（或任意 session）
- **UI 路径**: 导航到排行榜页面 `/hackathon/web3-innovation/leaderboard`
- **操作**: 查看最终排行榜
- **验证**:
  - #1: hacker_eve，"Web App Toolkit"，总分 85.60，奖金（积分）
  - #2: hacker_frank，"DeFi Yield Aggregator"，总分 80.53，奖金（积分）
  - 仅显示积分奖金，法币列不显示
- **截图**: `screenshots/full-cycle/p7-08-leaderboard.png`

### Phase 7 状态快照

| 实体 | 状态 |
|------|------|
| Hackathon | `Finished(4)` |
| hacker_eve | 排名 #1，获得积分奖金 |
| hacker_frank | 排名 #2，获得积分奖金 |
| 评委 | judge_carol + judge_dave 均已评分 |

---

## Phase 8: Credits 兑换

**目标**: 用户查看积分总览和交易历史，兑换奖品，Admin 发货。
**角色**: hacker_eve, hackforger (Admin)
**来源**: J4 Steps 4.3-4.9

### Step 8.1: Hacker1 查看积分总览
- **角色**: hacker_eve session
- **UI 路径**: 顶部导航栏用户菜单 → 点击"积分" → `/credits`
- **操作**: 查看余额和交易历史
- **验证**:
  - 余额显示当前累计积分
  - 交易历史完整，包含：
    - `admin_deposit`（Phase 4 充值，如有）
    - `escrow`（Bounty escrow，如有）
    - `deposit`（Grant 资助 +300）
    - `deposit`（Hackathon 奖金 +XXX）
- **截图**: `screenshots/full-cycle/p8-01-credits-overview.png`

### Step 8.2: Hacker1 浏览兑换选项
- **角色**: hacker_eve session
- **UI 路径**: 积分概览页 `/credits` → 兑换选项区域
- **操作**: 浏览可用的兑换选项列表
- **验证**:
  - 显示 "GPU 算力 - 100 小时"，价格 200 Credits，库存 10
- **截图**: `screenshots/full-cycle/p8-02-redeem-options.png`

### Step 8.3: Hacker1 兑换算力
- **角色**: hacker_eve session
- **UI 路径**: 积分概览页 → 点击 "GPU 算力 - 100 小时" 的"兑换"按钮 → 进入兑换确认页 `/credits/redeem/{id}`
- **操作**: 确认兑换信息无误，点击"确认兑换"按钮
- **验证**:
  - RedeemOrder 创建，状态 = `pending`
  - 积分扣除 200
  - 交易类型：`redeem`
  - Feed 事件: `credits_redeemed(42)` -- 仅自己可见
- **截图**: `screenshots/full-cycle/p8-03-redeem-confirmed.png`

### Step 8.4: Admin Fulfill 订单
- **角色**: hackforger session
- **UI 路径**: 站点管理 → 积分管理 → 订单管理 `/-/admin/credits/orders`
- **操作**: 找到 hacker_eve 的待处理订单，点击"发货"(Fulfill) 按钮（POST `/-/admin/credits/orders/{oid}/fulfill`）
- **验证**:
  - 订单状态：`pending` -> `fulfilled`
  - 密钥 "GPU-KEY-001" 自动分配并标记为已使用
  - hacker_eve 可在订单详情中查看密钥
  - Feed 事件: `order_fulfilled(58)`
- **截图**: `screenshots/full-cycle/p8-04-order-fulfilled.png`

### Step 8.5: Hacker1 查看订单和密钥
- **角色**: hacker_eve session
- **UI 路径**: 积分概览页 `/credits` → 点击"我的订单" → `/credits/orders`
- **操作**: 查看已兑换的订单详情
- **验证**:
  - 订单状态 = `fulfilled`
  - 显示密钥 "GPU-KEY-001"
- **截图**: `screenshots/full-cycle/p8-05-order-detail.png`

### Step 8.6: Admin 手动充值演示（给 Hacker2）
- **角色**: hackforger session
- **UI 路径**: 站点管理 → 积分管理 `/-/admin/credits`
- **操作**: 在"手动充值"区域填写：用户名 = `hacker_frank`，金额 = `500`，原因 = `Community contribution bonus`，点击"充值"
- **验证**:
  - hacker_frank 余额增加 500
  - 交易类型：`admin_deposit`
- **截图**: `screenshots/full-cycle/p8-06-admin-deposit-hacker2.png`

### Step 8.7: Admin 手动扣除演示
- **角色**: hackforger session
- **UI 路径**: 站点管理 → 积分管理 `/-/admin/credits` → "手动扣除"区域
- **操作**: 填写：用户名 = `hacker_frank`，金额 = `50`，原因 = `Duplicate reward correction`，点击"扣除"
- **验证**:
  - hacker_frank 余额减少 50
  - 交易类型：`admin_deduct`
- **截图**: `screenshots/full-cycle/p8-07-admin-deduct.png`

### Phase 8 状态快照

| 实体 | 状态 |
|------|------|
| RedeemOrder #1 (hacker_eve) | `fulfilled` |
| Key GPU-KEY-001 | 已使用 |
| hacker_eve Credits | 余额 - 200（兑换）|
| hacker_frank Credits | 余额 + 500 - 50（Admin 充值/扣除）|

---

## Phase 9: 社交 + Feed + 搜索

**目标**: 验证 Feed 系统、Explore 页面各 Tab、全局搜索（Cmd+K）、声誉排行榜、Reaction 互动。
**角色**: hacker_eve, hacker_frank
**来源**: J5 Steps 5.1-5.10

### Step 9.1: Hacker1 查看 Dashboard Feed (Following Tab)
- **角色**: hacker_eve session
- **UI 路径**: 导航到首页 Dashboard `/` → 切换到"社区动态" / "Following" Tab
- **操作**: 查看 Feed 列表
- **验证**: Feed 包含已关注用户 + 全局事件（按时间倒序、分页）：
  - `hackathon_created(30)` -- hackforger 创建了 Web3 Innovation
  - `hackathon_phase_changed(50)` -- hackforger 发布/启动/评审/结算
  - `hackathon_submitted(32)` -- hacker_frank 提交（hacker_eve 关注了 hacker_frank）
  - `bounty_created(34)` -- Bounty 创建事件
  - `grant_round_opened(54)` -- Grant 开放事件
- **截图**: `screenshots/full-cycle/p9-01-dashboard-feed.png`

### Step 9.2: Hacker1 查看 Global Feed
- **角色**: hacker_eve session
- **UI 路径**: Dashboard → 切换到"全局动态" / "Global" Tab
- **操作**: 查看全局 Feed
- **验证**:
  - 显示所有 HackForger 事件（不限关注关系）
  - 包含 Phase 1-8 中生成的所有事件类型
  - 按 `created_unix` DESC 排序
- **截图**: `screenshots/full-cycle/p9-02-global-feed.png`

### Step 9.3: Explore Hackathons Tab
- **角色**: hacker_eve session（或匿名）
- **UI 路径**: 顶部导航栏 "Explore" → 点击 "Hackathons" Tab → `/explore/hackathons`
- **操作**: 查看 Hackathon 列表
- **验证**:
  - 列出 "Web3 Innovation Challenge"，状态 = `Finished`
  - 显示参与人数、赛道数、奖金池（仅积分）
- **截图**: `screenshots/full-cycle/p9-03-explore-hackathons.png`

### Step 9.4: Explore Bounties Tab
- **角色**: hacker_eve session
- **UI 路径**: Explore 页面 → 点击 "Bounties" Tab → `/explore/bounties`
- **操作**: 查看 Bounty 列表，尝试筛选（按状态、模式）
- **验证**:
  - 列出所有 Bounty（跨 Repo）
  - 筛选功能可用
- **截图**: `screenshots/full-cycle/p9-04-explore-bounties.png`

### Step 9.5: Explore Grants Tab
- **角色**: hacker_eve session
- **UI 路径**: Explore 页面 → 点击 "Grants" Tab → `/explore/grants`
- **操作**: 查看 Grant Round 列表
- **验证**:
  - 列出 "DeFi Track Boost"，状态 = `Distributed`
  - 显示预算、已资助项目数
- **截图**: `screenshots/full-cycle/p9-05-explore-grants.png`

### Step 9.6: Explore Submissions Tab
- **角色**: hacker_eve session
- **UI 路径**: Explore 页面 → 点击 "Submissions" Tab → `/explore/submissions`
- **操作**: 查看 Submission 列表
- **验证**:
  - 列出所有 Hackathon Submissions
  - 可看到 hacker_eve 和 hacker_frank 的提交
- **截图**: `screenshots/full-cycle/p9-06-explore-submissions.png`

### Step 9.7: Cmd+K 全局搜索（分组结果）
- **角色**: hacker_eve session
- **UI 路径**: 在任意页面按 Cmd+K (macOS) / Ctrl+K 打开搜索弹窗
- **操作**: 输入关键词 `DeFi`，查看搜索结果
- **验证**: 结果按类型分组显示：
  - Hackathon 相关："DeFi Track"，"DeFi Yield Aggregator"
  - Grant 相关："DeFi Automation Toolkit"
  - Repos 相关：defi-track 相关 Repo
  - 每组有"查看全部"链接
  - 点击结果可跳转到详情页
- **截图**: `screenshots/full-cycle/p9-07-search-cmdk.png`

### Step 9.8: Explore Reputation 排行榜
- **角色**: hacker_eve session（或任意 session）
- **UI 路径**: Explore 页面 → 点击 "Reputation" Tab → `/explore/reputation`
- **操作**: 查看声誉排行榜
- **验证**:
  - hacker_eve 排名最高（Hackathon 第 1 名 + Bounty 参与 + Grant 获资助）
  - hacker_frank 排名第二（Hackathon 第 2 名 + Bounty 完成）
  - 分数反映各模块的加权贡献
- **截图**: `screenshots/full-cycle/p9-08-reputation-leaderboard.png`

### Step 9.9: Reaction 互动
- **角色**: hacker_frank session
- **UI 路径**: 导航到 hacker_eve 的 Submission 关联的 PR 页面 → 找到 PR 描述或评论
- **操作**: 点击 Reaction 按钮 → 选择 "+1" / 点赞
- **验证**:
  - Reaction 成功添加，显示在内容下方
  - Forgejo 原生 Reaction 功能正常工作
- **截图**: `screenshots/full-cycle/p9-09-reaction.png`

### Phase 9 状态快照

| 功能 | 验证状态 |
|------|---------|
| Following Feed | 显示已关注用户的事件 |
| Global Feed | 显示所有事件 |
| Explore: Hackathons | 可筛选列表 |
| Explore: Bounties | 跨 Repo 列表 |
| Explore: Grants | Round 列表 |
| Explore: Submissions | Submission 列表 |
| Explore: Reputation | 排行榜 |
| Cmd+K 搜索 | 分组结果 + 跳转 |
| Reaction | Forgejo 原生功能正常 |

---

## Phase 10: 异常路径 + 约束验证

**目标**: 验证系统在异常输入和边界条件下的行为，覆盖 Bounty 异常路径、权限约束、重复操作等场景。
**角色**: hackforger, hacker_eve, hacker_frank, judge_carol
**来源**: J2 Steps 2b + 2c + 各旅程约束规则

### 10a: Bounty 过期（Deadline 到期无人完成）

### Step 10a.1: Organizer 创建一个即将过期的 Bounty
- **角色**: hackforger session
- **UI 路径**: 导航到 Repo `hackforger/oss-project`（需先确保 Repo 存在）→ Issues → 新建 Issue
- **操作**:
  - 创建 Issue: 标题 = `Refactor API error handling`，提交
  - 如果 Repo `hackforger/oss-project` 不存在，先通过 "+" 菜单创建 Repo
  - 导航到 Bounty 创建页面 `/hackforger/oss-project/bounties/new`
  - 创建 Bounty：关联 Issue，模式 = Exclusive，截止日期设为过去日期（测试用），奖励金额 = `100`
  - 先确保 hackforger 有足够余额（如不足需先充值）
- **验证**:
  - Bounty 创建成功，状态 = `Open(0)`
  - 积分 escrow 成功
- **截图**: `screenshots/full-cycle/p10-01-bounty-expiry-setup.png`

### Step 10a.2: 触发过期
- **角色**: hackforger session 或系统 cron
- **操作**: 触发过期逻辑（截止日期到达或手动过期操作）
- **验证**:
  - Bounty 状态：`Open(0)` -> `Expired(5)`
  - 托管积分退回 hackforger：`escrow_refund` 交易
  - Feed 事件: `bounty_expired(52)`
- **截图**: `screenshots/full-cycle/p10-02-bounty-expired.png`

### 10b: Bounty 取消

### Step 10b.1: Organizer 创建并取消 Bounty
- **角色**: hackforger session
- **UI 路径**: Repo Issues → 新建 Issue → 创建 Bounty
- **操作**:
  - 创建 Issue: 标题 = `Add dark mode support`，提交
  - 创建 Bounty：关联 Issue，模式 = Exclusive，奖励金额 = `100`
  - 在 Bounty 详情页点击"取消"按钮（POST `/hackforger/oss-project/bounties/{bounty_id}/cancel`）
- **验证**:
  - Bounty 状态：`Open(0)` -> `Cancelled(6)`
  - 托管积分退回：`escrow_refund`
  - Feed 事件: `bounty_cancelled(53)`
- **截图**: `screenshots/full-cycle/p10-03-bounty-cancelled.png`

### 10c: Competitive Bounty 流程

### Step 10c.1: Organizer 创建 Competitive Bounty
- **角色**: hackforger session
- **UI 路径**: Repo Issues → 新建 Issue（标题 = `Design new landing page`） → Bounty 创建页面
- **操作**:
  - 创建 Competitive Bounty：关联 Issue，模式 = `Competitive`，描述 = `Best landing page design wins`，截止日期 = `2026-05-15`
  - 添加多级 Reward：
    - 1st place: 150 Credits
    - 2nd place: 100 Credits
  - 确保 hackforger 有足够余额（共 250 Credits）
- **验证**:
  - Bounty 创建成功，模式 = `Competitive(1)`，状态 = `Open(0)`
  - 积分 escrow：250 Credits
  - Feed 事件: `bounty_created(34)`
- **截图**: `screenshots/full-cycle/p10-04-competitive-bounty.png`

### Step 10c.2: 多人提交方案
- **角色**: hacker_eve session + hacker_frank session
- **操作**:
  - hacker_eve: 导航到 Bounty 详情 → 点击"申请参与" → 提交申请
  - hacker_frank: 同上
  - 各自在 Fork 中编辑文件并创建 PR
- **验证**:
  - 2 个申请创建，状态均为 `Accepted(1)`（Competitive 模式自动接受，无 Pending 阶段）
  - Bounty 保持 `Open(0)`（Competitive 模式允许多人并行）
- **截图**: `screenshots/full-cycle/p10-05-competitive-applications.png`

### Step 10c.3: Organizer 选择 Winner 并支付
- **角色**: hackforger session
- **UI 路径**: Bounty 获奖者管理页面 `/hackforger/oss-project/bounties/{bounty_id}/winners`
- **操作**:
  - 选择获奖者：
    - 1st place: hacker_eve
    - 2nd place: hacker_frank
  - 点击"确认获奖者"（POST `/hackforger/oss-project/bounties/{bounty_id}/winners`）
  - 点击"支付"按钮（POST `/hackforger/oss-project/bounties/{bounty_id}/pay`）
- **验证**:
  - Bounty 状态：`Open(0)` -> `Completed(3)` -> `Paid(4)`
  - 积分分配：hacker_eve +150，hacker_frank +100
  - Feed 事件: `bounty_winners_selected(38)` + `bounty_paid(43)`
- **截图**: `screenshots/full-cycle/p10-06-competitive-paid.png`

### 10d: 权限和约束验证

### Step 10d.1: [约束验证] 已报名参赛者不能被指派为评委
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage` → "评委管理"区域
- **操作**: 尝试在"添加评委"输入框中输入 `hacker_eve`（已报名参赛者），点击"添加"
- **验证**:
  - 系统拒绝，显示错误提示（参赛者不能同时担任评委）
  - 评委列表中不包含 hacker_eve
- **截图**: `screenshots/full-cycle/p10-07-participant-as-judge-denied.png`

### Step 10d.2: [约束验证] 重复报名 -> 应报错
- **角色**: hacker_eve session
- **UI 路径**: Hackathon 详情页 `/hackathon/web3-innovation` → 报名区域
- **操作**: 尝试再次报名（hacker_eve 已在 Phase 2 报名）
- **验证**:
  - 系统拒绝重复报名，显示错误提示或按钮已变为"已报名"状态
- **截图**: `screenshots/full-cycle/p10-08-duplicate-register-denied.png`

### Step 10d.3: [约束验证] Grant 超预算 -> 应报错
- **角色**: hackforger session
- **UI 路径**: Grant Round 管理页面（如有仍在 Review 状态的 Round）
- **操作**: 尝试为某个项目设置超过剩余预算的 Award 金额
- **验证**:
  - 系统拒绝，显示超预算错误提示
  - 拨款金额不变
- **截图**: `screenshots/full-cycle/p10-09-grant-over-budget.png`

### Step 10d.4: [约束验证] 非 Owner 尝试管理 Hackathon -> 403
- **角色**: hacker_frank session
- **UI 路径**: 直接导航到 `/hackathon/web3-innovation/manage`
- **操作**: 尝试访问管理页面
- **验证**:
  - 返回 403 Forbidden 或重定向到 Hackathon 详情页
  - 非 Owner 无法看到管理操作按钮
- **截图**: `screenshots/full-cycle/p10-10-non-owner-403.png`

### Step 10d.5: [约束验证] Bounty 申请被拒绝（Exclusive 已被认领）
- **角色**: hacker_frank session + hackforger session
- **操作**:
  - 在一个已处于 `Claimed` 状态的 Exclusive Bounty 上，hacker_frank 尝试提交申请
  - hackforger 拒绝该申请，附评论 = `Bounty already claimed`
- **验证**:
  - 申请状态：`Pending(0)` -> `Rejected(2)` 或系统直接拒绝提交
  - Bounty 状态保持不变
- **截图**: `screenshots/full-cycle/p10-11-application-rejected.png`

### Phase 10 状态快照

| 场景 | 验证结果 |
|------|---------|
| Bounty 过期 | Expired(5) + escrow_refund |
| Bounty 取消 | Cancelled(6) + escrow_refund |
| Competitive Bounty | 多人参与 + Winner 选择 + 支付 |
| 评委不能报名 | 拒绝 |
| 参赛者不能当评委 | 拒绝 |
| 重复报名 | 拒绝 |
| Grant 超预算 | 拒绝 |
| 非 Owner 管理 | 403 |
| Bounty 申请被拒 | Rejected(2) |

---

## 跨阶段验证

以下检查点验证各阶段之间的数据一致性。

### 积分一致性对账

在所有阶段完成后，每个用户导航到 `/credits` 查看余额，核对交易历史。

| 用户 | 积分来源 | 积分去向 |
|------|---------|---------|
| hacker_eve | Admin 充值（Phase 4）、Hackathon 奖金（Phase 7）、Grant 资助（Phase 5）、Competitive Bounty 1st（Phase 10c） | Bounty escrow（Phase 4）、兑换算力（Phase 8） |
| hacker_frank | Bounty 奖励（Phase 4）、Hackathon 奖金（Phase 7）、Admin 充值（Phase 8）、Competitive Bounty 2nd（Phase 10c） | Admin 扣除（Phase 8） |
| hackforger (organizer) | 多次 Admin 充值（Phase 0, 5, 10） | Hackathon 奖金发放、Grant 发放、Bounty escrow |

- **验证**: 每个用户的余额与预期一致，交易记录完整无遗漏
- **截图**: `screenshots/full-cycle/cross-01-credits-reconcile.png`

### Feed 事件完整性验证

在 hackforger session 的 Dashboard Global Feed 中验证。

| 事件类型 | 来源阶段 | 预期存在 |
|---------|---------|---------|
| `hackathon_created(30)` | Phase 1 | 是 |
| `hackathon_phase_changed(50)` | Phase 1, 3, 7 | 是（publish, start, judge） |
| `hackathon_registered(31)` | Phase 2 | 是（x2） |
| `hackathon_submitted(32)` | Phase 6 | 是（x2） |
| `hackathon_scored(33)` | Phase 7 | 是（x4） |
| `hackathon_finalized(51)` | Phase 7 | 是 |
| `bounty_created(34)` | Phase 4, 10 | 是（多个）|
| `bounty_claimed(35)` | Phase 4 | 是 |
| `bounty_delivered(36)` | Phase 4 | 是 |
| `bounty_completed(37)` | Phase 4 | 是 |
| `bounty_paid(43)` | Phase 4, 10c | 是 |
| `bounty_winners_selected(38)` | Phase 10c | 是 |
| `bounty_expired(52)` | Phase 10a | 是 |
| `bounty_cancelled(53)` | Phase 10b | 是 |
| `grant_round_created(39)` | Phase 5 | 是 |
| `grant_round_opened(54)` | Phase 5 | 是 |
| `grant_project_submitted(40)` | Phase 5 | 是（x2）|
| `grant_round_closed(55)` | Phase 5 | 是 |
| `grant_awarded(41)` | Phase 5 | 是 |
| `grant_round_finalized(56)` | Phase 5 | 是 |
| `credits_redeemed(42)` | Phase 8 | 是 |
| `order_fulfilled(58)` | Phase 8 | 是 |

- **验证**: 以上事件均出现在 Global Feed 中
- **截图**: `screenshots/full-cycle/cross-02-feed-events.png`

---

## 报告模版

测试完成后，按以下格式生成报告，保存到 `docs/tests/e2e/reports/user-journey-full-cycle-report.md`：

```markdown
# 用户旅程全流程测试报告

> **日期**: YYYY-MM-DD
> **服务器**: http://localhost:3000
> **测试者**: (执行者名称)
> **代码版本**: (git commit hash)

## 概要

| 阶段 | 步骤数 | 通过 | 失败 | 跳过 | 状态 |
|------|--------|------|------|------|------|
| Phase 0: 平台准备 | 3 | ? | ? | ? | ? |
| Phase 1: Hackathon 创建 | 5 | ? | ? | ? | ? |
| Phase 2: 报名 + 社交 | 6 | ? | ? | ? | ? |
| Phase 3: Hacking 阶段 | 2 | ? | ? | ? | ? |
| Phase 4: 开发 + Bounty 协作 | 10 | ? | ? | ? | ? |
| Phase 5: Grant 资助 | 9 | ? | ? | ? | ? |
| Phase 6: 提交参赛作品 | 5 | ? | ? | ? | ? |
| Phase 7: 评审 + Finalize | 8 | ? | ? | ? | ? |
| Phase 8: Credits 兑换 | 7 | ? | ? | ? | ? |
| Phase 9: 社交 + Feed + 搜索 | 9 | ? | ? | ? | ? |
| Phase 10: 异常路径 + 约束验证 | 11 | ? | ? | ? | ? |
| 跨阶段验证 | 2 | ? | ? | ? | ? |
| **总计** | **77** | | | | |

## 逐步结果

### Phase 0: 平台准备

#### Step 0.1: Admin 配置兑换选项 -- [PASS/FAIL]
- 操作描述: ...
- 截图: `screenshots/full-cycle/p0-01-redeem-option-created.png`
- 备注: (如有异常或额外观察)

#### Step 0.2: Admin 添加 Key Pool -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p0-02-keys-added.png`

#### Step 0.3: Admin 为 Organizer 充值积分 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p0-03-credits-deposit-1000.png`

### Phase 1: Hackathon 创建

#### Step 1.1: Organizer 创建 Hackathon -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-01-hackathon-created.png`

#### Step 1.2: Organizer 创建赛道 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-02-tracks-created.png`

#### Step 1.3: Organizer 设置评审标准 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-03-criteria-set.png`

#### Step 1.4: Organizer 指派评委 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-04-judges-assigned.png`

#### Step 1.5: Organizer 发布 Hackathon -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-05-published.png`

### Phase 2: 报名 + 社交

#### Step 2.1: Hacker1 关注 Organizer -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-01-follow-organizer.png`

#### Step 2.2: Hacker1 报名 Web Track -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-02-hacker1-registered.png`

#### Step 2.3: Hacker2 报名 DeFi Track -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-03-hacker2-registered.png`

#### Step 2.4: Hacker1 与 Hacker2 互相关注 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-04-mutual-follow.png`

#### Step 2.5: 组队 -- 在 Hackathon Org 内创建 Team -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-05-team-created.png`

#### Step 2.6: [约束] Judge 尝试报名 -> 拒绝 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-06-judge-register-denied.png`

### Phase 3: Hacking 阶段

#### Step 3.1: Organizer 启动 Hacking -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p3-01-hacking-started.png`

#### Step 3.2: [约束] 重复发布 -> 拒绝 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p3-02-duplicate-publish-denied.png`

### Phase 4: 开发 + Bounty 协作

#### Step 4.1: Hacker1 Fork Web Track Repo -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-01-hacker1-fork-web.png`

#### Step 4.2: Hacker1 创建 Issue -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-02-issue-created.png`

#### Step 4.3: Hacker1 创建 Exclusive Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-03-bounty-created.png`

#### Step 4.4: Admin 为 Hacker1 充值 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-04-hacker1-deposit.png`

#### Step 4.5: Hacker2 浏览 Bounty 并申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-05-bounty-application.png`

#### Step 4.6: Hacker1 接受申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-06-application-accepted.png`

#### Step 4.7: Hacker2 Fork 并提交 UI 设计 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-07-hacker2-fork-design.png`

#### Step 4.8: Hacker2 创建 PR -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-08-bounty-pr.png`

#### Step 4.9: Hacker1 Complete + Pay Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-09-bounty-completed-paid.png`

#### Step 4.10: Hacker2 查看积分余额变化 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-10-hacker2-credits.png`

### Phase 5: Grant 资助

#### Step 5.1: Admin 为 Organizer 充值 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-01-grant-budget-deposit.png`

#### Step 5.2: Organizer 创建 Grant Round -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-02-grant-round-created.png`

#### Step 5.3: Organizer 开放申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-03-grant-opened.png`

#### Step 5.4: Hacker1 提交 Grant 申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-04-project1-submitted.png`

#### Step 5.5: Hacker2 提交 Grant 申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-05-project2-submitted.png`

#### Step 5.6: Organizer 关闭申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-06-grant-closed.png`

#### Step 5.7: Organizer 审批项目 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-07-projects-reviewed.png`

#### Step 5.8: Organizer Finalize + Distribute -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-08-grant-distributed.png`

#### Step 5.9: Hacker1 查看积分变化 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-09-hacker1-credits-grant.png`

### Phase 6: 提交参赛作品

#### Step 6.1: Hacker1 继续开发 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-01-hacker1-develop.png`

#### Step 6.2: Hacker1 Fork+PR 模式提交 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-02-hacker1-submission-pr.png`

#### Step 6.3: Hacker2 Fork DeFi Track 并开发 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-03-hacker2-fork-defi.png`

#### Step 6.4: Hacker2 Link Repo 模式提交 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-04-hacker2-submission-link.png`

#### Step 6.5: 多用户 Star 赛道 Repo -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-05-star-repos.png`

### Phase 7: 评审 + Finalize

#### Step 7.1: Organizer 启动评审 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-01-judging-started.png`

#### Step 7.2: Judge1 评分 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-02-judge1-scores.png`

#### Step 7.3: Judge2 评分 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-03-judge2-scores.png`

#### Step 7.4: Organizer 查看排名预览 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-04-finalize-preview.png`

#### Step 7.5: Organizer 确认结算 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-05-finalized.png`

#### Step 7.6: Hacker1 查看积分余额 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-06-hacker1-credits-hackathon.png`

#### Step 7.7: Hacker2 查看积分余额 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-07-hacker2-credits-hackathon.png`

#### Step 7.8: 查看排行榜 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-08-leaderboard.png`

### Phase 8: Credits 兑换

#### Step 8.1: Hacker1 查看积分总览 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-01-credits-overview.png`

#### Step 8.2: 浏览兑换选项 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-02-redeem-options.png`

#### Step 8.3: 兑换算力 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-03-redeem-confirmed.png`

#### Step 8.4: Admin Fulfill 订单 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-04-order-fulfilled.png`

#### Step 8.5: 查看订单和密钥 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-05-order-detail.png`

#### Step 8.6: Admin 手动充值 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-06-admin-deposit-hacker2.png`

#### Step 8.7: Admin 手动扣除 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-07-admin-deduct.png`

### Phase 9: 社交 + Feed + 搜索

#### Step 9.1: Dashboard Feed (Following) -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-01-dashboard-feed.png`

#### Step 9.2: Global Feed -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-02-global-feed.png`

#### Step 9.3: Explore Hackathons -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-03-explore-hackathons.png`

#### Step 9.4: Explore Bounties -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-04-explore-bounties.png`

#### Step 9.5: Explore Grants -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-05-explore-grants.png`

#### Step 9.6: Explore Submissions -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-06-explore-submissions.png`

#### Step 9.7: Cmd+K 全局搜索 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-07-search-cmdk.png`

#### Step 9.8: Reputation 排行榜 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-08-reputation-leaderboard.png`

#### Step 9.9: Reaction 互动 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-09-reaction.png`

### Phase 10: 异常路径 + 约束验证

#### Step 10a.1: 创建即将过期的 Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-01-bounty-expiry-setup.png`

#### Step 10a.2: 触发过期 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-02-bounty-expired.png`

#### Step 10b.1: 创建并取消 Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-03-bounty-cancelled.png`

#### Step 10c.1: 创建 Competitive Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-04-competitive-bounty.png`

#### Step 10c.2: 多人提交方案 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-05-competitive-applications.png`

#### Step 10c.3: 选择 Winner 并支付 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-06-competitive-paid.png`

#### Step 10d.1: [约束] 参赛者不能当评委 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-07-participant-as-judge-denied.png`

#### Step 10d.2: [约束] 重复报名 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-08-duplicate-register-denied.png`

#### Step 10d.3: [约束] Grant 超预算 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-09-grant-over-budget.png`

#### Step 10d.4: [约束] 非 Owner 管理 -> 403 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-10-non-owner-403.png`

#### Step 10d.5: [约束] Bounty 申请被拒绝 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-11-application-rejected.png`

### 跨阶段验证

#### 积分一致性对账 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/cross-01-credits-reconcile.png`

#### Feed 事件完整性 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/cross-02-feed-events.png`

## 积分对账表

| 用户 | 充值/收入 | 扣除/支出 | 预期余额 | 实际余额 | 一致? |
|------|---------|---------|---------|---------|------|
| hacker_eve | (逐项列出) | (逐项列出) | ? | ? | ? |
| hacker_frank | (逐项列出) | (逐项列出) | ? | ? | ? |
| hackforger | (逐项列出) | (逐项列出) | ? | ? | ? |

## 发现的 Bug

| # | 阶段 | 步骤 | 描述 | 严重程度 | 截图 | 修复状态 |
|---|------|------|------|---------|------|---------|
| 1 | | | | P0/P1/P2 | | Open/Fixed |

## 结论

(总结通过率、关键发现、需要修复的问题、对 v0.1 发布的影响评估)
```
