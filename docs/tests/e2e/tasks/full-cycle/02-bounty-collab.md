# Phase 4: 开发 + Bounty 协作

> **前置条件**: 完成 `00-setup.md` + `01-hackathon-lifecycle.md`

本文件覆盖参赛者开发过程中的 Bounty 协作流程：创建 Issue、发起 Bounty、申请认领、Fork 开发、PR 提交、审核支付。
环境/账号/Session 信息请参考 [00-setup.md](./00-setup.md)。

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

## 本阶段结果

| 步骤 | 状态 | 截图 |
|------|------|------|
| Step 4.1: Hacker1 Fork Web Track Repo | PASS/FAIL | screenshots/full-cycle/p4-01-hacker1-fork-web.png |
| Step 4.2: Hacker1 创建 Issue | PASS/FAIL | screenshots/full-cycle/p4-02-issue-created.png |
| Step 4.3: Hacker1 创建 Exclusive Bounty | PASS/FAIL | screenshots/full-cycle/p4-03-bounty-created.png |
| Step 4.4: Admin 为 Hacker1 充值 | PASS/FAIL | screenshots/full-cycle/p4-04-hacker1-deposit.png |
| Step 4.5: Hacker2 浏览 Bounty 并申请 | PASS/FAIL | screenshots/full-cycle/p4-05-bounty-application.png |
| Step 4.6: Hacker1 接受申请 | PASS/FAIL | screenshots/full-cycle/p4-06-application-accepted.png |
| Step 4.7: Hacker2 Fork 并提交 UI 设计 | PASS/FAIL | screenshots/full-cycle/p4-07-hacker2-fork-design.png |
| Step 4.8: Hacker2 创建 PR | PASS/FAIL | screenshots/full-cycle/p4-08-bounty-pr.png |
| Step 4.9: Hacker1 Complete + Pay Bounty | PASS/FAIL | screenshots/full-cycle/p4-09-bounty-completed-paid.png |
| Step 4.10: Hacker2 查看积分余额 | PASS/FAIL | screenshots/full-cycle/p4-10-hacker2-credits.png |
