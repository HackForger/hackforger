# Phase 1-3: Hackathon 创建 → 报名 → Hacking 启动

> **前置条件**: 完成 `00-setup.md`

本文件覆盖 Hackathon 的创建、配置、发布、报名、社交关系建立、以及 Hacking 阶段的启动。
环境/账号/Session 信息请参考 [00-setup.md](./00-setup.md)。

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
  - 标识符 = `web3-innovation`（系统将自动创建同名 Forgejo 组织）
  - 描述 = 使用 Markdown 格式编写（支持标题、列表、链接、图片、代码块）：
    ```markdown
    ## Web3 Innovation Challenge
    Build the future of decentralized web.

    ![banner](https://example.com/hackathon-banner.png)

    ### 主题
    - DeFi 协议
    - NFT 工具链
    - 去中心化身份
    ```
  - 奖品说明 = `冠军：500 积分，亚军：300 积分`（支持 Markdown）
  - 日程安排：
    - 报名开始 = `2026-04-10 09:00`
    - 报名截止 = `2026-04-20 23:59`
    - 开发开始 = `2026-04-21 09:00`
    - 开发截止 = `2026-05-05 23:59`
    - 评审截止 = `2026-05-12 23:59`
  - 最大团队人数 = `5`
  - 点击"创建黑客松"按钮
- **验证**:
  - 跳转到 Hackathon 详情页 `/hackathon/web3-innovation`，状态 = `Draft`
  - 描述以 Markdown 渲染（标题、列表、图片均正确显示）
  - 时间轴显示各阶段日期
  - Forgejo 组织 `web3-innovation` 已自动创建，hackforger 为 owner
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

## 本阶段结果

| 步骤 | 状态 | 截图 |
|------|------|------|
| Step 1.1: Organizer 创建 Hackathon | PASS/FAIL | screenshots/full-cycle/p1-01-hackathon-created.png |
| Step 1.2: Organizer 创建赛道 | PASS/FAIL | screenshots/full-cycle/p1-02-tracks-created.png |
| Step 1.3: Organizer 设置评审标准 | PASS/FAIL | screenshots/full-cycle/p1-03-criteria-set.png |
| Step 1.4: Organizer 指派评委 | PASS/FAIL | screenshots/full-cycle/p1-04-judges-assigned.png |
| Step 1.5: Organizer 发布 Hackathon | PASS/FAIL | screenshots/full-cycle/p1-05-published.png |
| Step 2.1: Hacker1 关注 Organizer | PASS/FAIL | screenshots/full-cycle/p2-01-follow-organizer.png |
| Step 2.2: Hacker1 报名 Web Track | PASS/FAIL | screenshots/full-cycle/p2-02-hacker1-registered.png |
| Step 2.3: Hacker2 报名 DeFi Track | PASS/FAIL | screenshots/full-cycle/p2-03-hacker2-registered.png |
| Step 2.4: Hacker1 与 Hacker2 互相关注 | PASS/FAIL | screenshots/full-cycle/p2-04-mutual-follow.png |
| Step 2.5: 组队创建 Team | PASS/FAIL | screenshots/full-cycle/p2-05-team-created.png |
| Step 2.6: [约束] Judge 报名被拒绝 | PASS/FAIL | screenshots/full-cycle/p2-06-judge-register-denied.png |
| Step 3.1: Organizer 启动 Hacking | PASS/FAIL | screenshots/full-cycle/p3-01-hacking-started.png |
| Step 3.2: [约束] 重复发布被拒绝 | PASS/FAIL | screenshots/full-cycle/p3-02-duplicate-publish-denied.png |
