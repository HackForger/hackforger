# Phase 5: Grant 资助

> **前置条件**: 完成 `00-setup.md` + `01-hackathon-lifecycle.md`

本文件覆盖 Grant Round 的完整生命周期：创建、开放申请、项目提交、关闭申请、审批、Finalize、积分发放。
环境/账号/Session 信息请参考 [00-setup.md](./00-setup.md)。

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

## 本阶段结果

| 步骤 | 状态 | 截图 |
|------|------|------|
| Step 5.1: Admin 为 Organizer 充值 | PASS/FAIL | screenshots/full-cycle/p5-01-grant-budget-deposit.png |
| Step 5.2: Organizer 创建 Grant Round | PASS/FAIL | screenshots/full-cycle/p5-02-grant-round-created.png |
| Step 5.3: Organizer 开放申请 | PASS/FAIL | screenshots/full-cycle/p5-03-grant-opened.png |
| Step 5.4: Hacker1 提交 Grant 申请 | PASS/FAIL | screenshots/full-cycle/p5-04-project1-submitted.png |
| Step 5.5: Hacker2 提交 Grant 申请 | PASS/FAIL | screenshots/full-cycle/p5-05-project2-submitted.png |
| Step 5.6: Organizer 关闭申请 | PASS/FAIL | screenshots/full-cycle/p5-06-grant-closed.png |
| Step 5.7: Organizer 审批项目 | PASS/FAIL | screenshots/full-cycle/p5-07-projects-reviewed.png |
| Step 5.8: Organizer Finalize + Distribute | PASS/FAIL | screenshots/full-cycle/p5-08-grant-distributed.png |
| Step 5.9: Hacker1 查看积分变化 | PASS/FAIL | screenshots/full-cycle/p5-09-hacker1-credits-grant.png |
