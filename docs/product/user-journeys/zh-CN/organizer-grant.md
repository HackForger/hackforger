## 旅程 3：Grant 完整生命周期

### 前置条件
- `organizer` 已存在（J1+J2 剩余 400 Credits，但 Grant 预算来自 admin 充值）
- `admin` 向 `organizer` 充值 2000 Credits 用于资助轮次
- hacker1（950 Credits）、hacker2（400 Credits）已存在

### 步骤 3.0：Admin 向 organizer 充值 Credits 用于资助轮次

- **角色**：`admin`
- **API**: `POST /api/v1/hackforger/credits/admin/deposit`
  ```json
  { "username": "organizer", "amount": 2000, "reason": "Q2 Grant Round budget" }
  ```
- **预期结果**：organizer 余额 = 400（J1+J2 剩余）+ 2000 = 2400
- **Fixture Hint**: `credit_transaction(admin_deposit, organizer, 2000)`

### 步骤 3.1：Organizer 创建 Grant Round（Draft 状态）

- **角色**：`organizer`
- **API**: `POST /api/v1/hackforger/grants/rounds`
  ```json
  {
    "name": "Q2 2026 Open Source Grant",
    "slug": "q2-2026-oss-grant",
    "description": "Funding open-source infrastructure projects",
    "budget": 2000,
    "deadline": "2026-06-01T00:00:00Z"
  }
  ```
- **预期结果**：
  - Grant Round 已创建，状态 = `Draft(0)`
  - Feed 事件：`grant_round_created(39)` — 受众：全局
- **Fixture Hint**: `grant_round(id=1, status=Draft, budget=2000)`

### 步骤 3.2：Organizer 开放轮次（Draft → Open）

- **角色**：`organizer`
- **API**: `POST /api/v1/hackforger/grants/rounds/1/open`
- **预期结果**：
  - 状态：`Draft(0)` → `Open(1)`
  - 现在接受申请
  - Feed 事件：`grant_round_opened(54)` — 受众：全局
- **Fixture Hint**: `grant_round(id=1, status=Open)`

### 步骤 3.3：hacker1 提交 Grant Project 申请

- **角色**：`hacker1`
- **API**: `POST /api/v1/hackforger/grants/rounds/1/projects`
  ```json
  {
    "title": "DeFi Testing Framework",
    "description": "Open-source testing framework for DeFi smart contracts",
    "requested_amount": 1000,
    "repo_url": "https://hackforger.inside.h2os.cloud/hacker1/defi-test-framework"
  }
  ```
- **预期结果**：
  - Grant Project 已创建，状态 = `Pending`
  - Feed 事件：`grant_project_submitted(40)` — 受众：全局 + 关注者
- **Fixture Hint**: `grant_project(id=1, round_id=1, user_id=hacker1, status=Pending, requested=1000)`

### 步骤 3.4：hacker2 提交 Grant Project 申请

- **角色**：`hacker2`
- **API**: `POST /api/v1/hackforger/grants/rounds/1/projects`
  ```json
  {
    "title": "NFT Metadata Standard Library",
    "description": "Standardized metadata handling for NFTs across chains",
    "requested_amount": 1500,
    "repo_url": "https://hackforger.inside.h2os.cloud/hacker2/nft-metadata-lib"
  }
  ```
- **预期结果**：
  - Grant Project 已创建，状态 = `Pending`
  - Feed 事件：`grant_project_submitted(40)`
- **Fixture Hint**: `grant_project(id=2, round_id=1, user_id=hacker2, status=Pending, requested=1500)`

### 步骤 3.5：Organizer 关闭申请（Open → Review）

- **角色**：`organizer`
- **API**: `POST /api/v1/hackforger/grants/rounds/1/close`
- **预期结果**：
  - 状态：`Open(1)` → `Review(2)`
  - 不再接受新申请
  - Feed 事件：`grant_round_closed(55)`
- **Fixture Hint**: `grant_round(id=1, status=Review)`

### 步骤 3.6：Organizer 审核并批准/拒绝项目

- **角色**：`organizer`
- **操作**：批准 hacker1 的项目，拒绝 hacker2 的项目（超出预算）
- **API**: `PUT /api/v1/hackforger/grants/rounds/1/projects/1`
  ```json
  { "status": "approved", "awarded_amount": 1000, "comment": "Strong proposal, fully funded" }
  ```
- **API**: `PUT /api/v1/hackforger/grants/rounds/1/projects/2`
  ```json
  { "status": "rejected", "comment": "Over remaining budget, reapply next round" }
  ```
- **预期结果**：
  - 项目 1：`Pending` → `Approved`，获批 1000 Credits
  - 项目 2：`Pending` → `Rejected`
  - 总分配（1000）≤ 预算（2000）✓
  - Feed 事件：`grant_awarded(41)` 针对项目 1
- **Fixture Hint**: `grant_project(id=1, status=Approved, awarded=1000)`, `grant_project(id=2, status=Rejected)`

### 步骤 3.7：Organizer 定稿轮次（Review → Finalized）

- **角色**：`organizer`
- **操作**：锁定分配 — 承诺已做出，但 Credits 尚未分发
- **API**: `POST /api/v1/hackforger/grants/rounds/1/finalize`
- **预期结果**：
  - 状态：`Review(2)` → `Finalized(3)`
  - 分配已锁定（不再更改获批金额）
  - **Credits 尚未分发** — organizer 将先审查项目进展
  - Feed 事件：`grant_round_finalized(56)`
- **Fixture Hint**: `grant_round(id=1, status=Finalized)`

### 步骤 3.8：Organizer 审查项目进展并分发资金（Finalized → Distributed）

- **角色**：`organizer`
- **操作**：审查 hacker1 的项目活动（提交、PR、里程碑等关联仓库记录）后，organizer 决定释放资金
- **人工检查**：organizer 访问 hacker1 的仓库，审查提交历史、PR 合并、README 更新 — 平台**不**强制执行此检查，organizer 根据自己的标准做出决定
- **API**: `POST /api/v1/hackforger/grants/rounds/1/distribute`
  > **新端点** — 不在原始 prompt 的 API 列表中。PRD 的两阶段资助模型所必需。与 `/finalize` 分离以强制执行 Finalized→Distributed 门控。
- **预期结果**：
  - 状态：`Finalized(3)` → `Distributed(4)`
  - Credits 已转账：
    - hacker1：+1000（余额：950 + 1000 = 1950）
    - organizer：-1000（余额：2400 - 1000 = 1400）
  - 项目 1 状态：`Approved` → `Funded`
  - 交易类型：hacker1 为 `deposit`，organizer 为 `withdraw`
  - Feed 事件：（由轮次状态变更覆盖）
- **Fixture Hint**: `grant_round(id=1, status=Distributed)`, `grant_project(id=1, status=Funded)`, `credit_transaction(deposit, hacker1, 1000)`

### 状态快照：旅程 3 结束后

| 实体 | 状态 |
|------|------|
| Grant Round #1 | `Distributed(4)` |
| Grant Project #1（hacker1） | `Funded` |
| Grant Project #2（hacker2） | `Rejected` |
| Credits: hacker1 | 1950 (500 + 300 + 150 + 1000) |
| Credits: hacker2 | 400（未变） |
| Credits: organizer | 1400 (400 + 2000 - 1000) |

---
