# Phase 10: 异常路径 + 约束验证

> **前置条件**: 完成 `00-setup.md`（独立，仅需基础用户存在）

本文件覆盖 Bounty 异常路径（过期、取消）、Competitive Bounty 流程、以及各种权限和约束验证。
环境/账号/Session 信息请参考 [00-setup.md](./00-setup.md)。

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

## 本阶段结果

| 步骤 | 状态 | 截图 |
|------|------|------|
| Step 10a.1: 创建即将过期的 Bounty | PASS/FAIL | screenshots/full-cycle/p10-01-bounty-expiry-setup.png |
| Step 10a.2: 触发过期 | PASS/FAIL | screenshots/full-cycle/p10-02-bounty-expired.png |
| Step 10b.1: 创建并取消 Bounty | PASS/FAIL | screenshots/full-cycle/p10-03-bounty-cancelled.png |
| Step 10c.1: 创建 Competitive Bounty | PASS/FAIL | screenshots/full-cycle/p10-04-competitive-bounty.png |
| Step 10c.2: 多人提交方案 | PASS/FAIL | screenshots/full-cycle/p10-05-competitive-applications.png |
| Step 10c.3: 选择 Winner 并支付 | PASS/FAIL | screenshots/full-cycle/p10-06-competitive-paid.png |
| Step 10d.1: [约束] 参赛者不能当评委 | PASS/FAIL | screenshots/full-cycle/p10-07-participant-as-judge-denied.png |
| Step 10d.2: [约束] 重复报名 | PASS/FAIL | screenshots/full-cycle/p10-08-duplicate-register-denied.png |
| Step 10d.3: [约束] Grant 超预算 | PASS/FAIL | screenshots/full-cycle/p10-09-grant-over-budget.png |
| Step 10d.4: [约束] 非 Owner 管理 -> 403 | PASS/FAIL | screenshots/full-cycle/p10-10-non-owner-403.png |
| Step 10d.5: [约束] Bounty 申请被拒绝 | PASS/FAIL | screenshots/full-cycle/p10-11-application-rejected.png |
