## 旅程 2：Bounty 完整生命周期

### 前置条件
- `organizer` 拥有仓库 `organizer/oss-project`，其中包含已有 Issue
- `admin` 已向 `organizer` 账户充值 600 Credits 用于赏金奖励
- hacker1、hacker2、platform-bot 已存在

### 步骤 2.0：Admin 向 organizer 充值 Credits

- **角色**：`admin`
- **API**: `POST /api/v1/hackforger/credits/admin/deposit`
  ```json
  { "username": "organizer", "amount": 600, "reason": "Bounty reward pool" }
  ```
- **预期结果**：organizer 余额 = 200（J1 剩余）+ 600 = 800
- **Fixture Hint**: `credit_account(organizer, balance=800)`

---

### 旅程 2a：独占赏金（正常流程）

### 步骤 2a.1：Organizer 创建 Issue

- **角色**：`organizer`
- **操作**：在其仓库上创建一个 Issue 作为赏金目标
- **API**: `POST /api/v1/repos/organizer/oss-project/issues`（Forgejo 原生接口）
  ```json
  { "title": "Implement OAuth2 PKCE flow", "body": "We need OAuth2 PKCE support..." }
  ```
- **预期结果**：Issue #1 已在 `organizer/oss-project` 上创建
- **Fixture Hint**: `issue(repo=organizer/oss-project, index=1)`

### 步骤 2a.2：Organizer 在 Issue 上创建独占 Bounty

- **角色**：`organizer`
- **操作**：为 Issue 附加一个独占赏金
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties`
  ```json
  {
    "issue_index": 1,
    "mode": 0,
    "description": "Implement OAuth2 PKCE flow with tests",
    "deadline": "2026-05-01T00:00:00Z"
  }
  ```
- **预期结果**：
  - Bounty 已创建，状态 = `Open(0)`，模式 = `Exclusive(0)`
  - Bounty 关联到 Issue #1（1:1）
  - Feed 事件：`bounty_created(34)` — 受众：全局 + 仓库关注者
- **Fixture Hint**: `bounty(id=1, repo_id=<oss-project>, issue_id=1, mode=Exclusive, status=Open)`

### 步骤 2a.3：Organizer 添加奖励并托管资金

- **角色**：`organizer`
- **操作**：设置赏金奖励金额（触发 Credits 托管）
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/1/rewards`
  ```json
  { "amount": 300, "description": "Full implementation with tests" }
  ```
- **预期结果**：
  - 奖励已创建：300 Credits
  - 从 organizer 托管 Credits：余额 800 → 500（300 由平台持有）
  - 交易类型：`escrow`
- **Fixture Hint**: `bounty_reward(bounty_id=1, amount=300)`, `credit_transaction(escrow, organizer, -300)`

### 步骤 2a.4：hacker1 申请领取 Bounty

- **角色**：`hacker1`
- **操作**：提交申请以领取赏金
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/1/applications`
  ```json
  { "proposal": "I have experience with OAuth2. Estimated 3 days." }
  ```
- **预期结果**：
  - 申请已创建，状态 = `Pending(0)`
- **Fixture Hint**: `bounty_application(bounty_id=1, user_id=hacker1, status=Pending)`

### 步骤 2a.5：Organizer 接受 hacker1 的申请（Bounty → Claimed）

- **角色**：`organizer`
- **操作**：审核并接受申请
- **API**: `PUT /api/v1/repos/organizer/oss-project/hackforger/bounties/1/applications/<aid>/review`
  ```json
  { "status": 1, "comment": "Approved, go ahead" }
  ```
- **预期结果**：
  - 申请状态：`Pending(0)` → `Accepted(1)`
  - Bounty 状态：`Open(0)` → `Claimed(1)`
  - hacker1 成为独占领取者
  - Feed 事件：`bounty_claimed(35)` — 受众：仓库关注者 + 粉丝
- **Fixture Hint**: `bounty(id=1, status=Claimed)`, `bounty_application(status=Accepted)`

### 步骤 2a.6：hacker1 Fork 仓库并开发

- **角色**：`hacker1`
- **Git 操作**：
  - `POST /api/v1/repos/organizer/oss-project/forks` → 创建 `hacker1/oss-project`
  - hacker1 向 Fork 推送 OAuth2 PKCE 实现
- **预期结果**：Fork `hacker1/oss-project` 包含实现提交
- **Fixture Hint**: `repo(owner=hacker1, name=oss-project, fork_of=organizer/oss-project)`

### 步骤 2a.7：hacker1 提交 PR（Claimed → InReview）

- **角色**：`hacker1`
- **Git 操作**：`POST /api/v1/repos/organizer/oss-project/pulls`
  ```json
  {
    "title": "feat: implement OAuth2 PKCE flow",
    "head": "hacker1:main",
    "base": "main",
    "body": "Closes #1\n\nImplements PKCE with full test coverage"
  }
  ```
- **预期结果**：
  - PR 已创建并关联到 Issue #1
  - Bounty 状态：`Claimed(1)` → `InReview(2)`
  - Feed 事件：`bounty_delivered(36)` — 受众：仓库关注者
- **Fixture Hint**: `bounty(id=1, status=InReview)`, `pull_request(repo=organizer/oss-project, head=hacker1:main)`

### 步骤 2a.8：Organizer 评审并完成（InReview → Completed）

- **角色**：`organizer`
- **操作**：评审 PR，批准，并标记赏金为已完成
- **Git 操作**：organizer 添加 PR Review（批准）
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/1/complete`
- **预期结果**：
  - Bounty 状态：`InReview(2)` → `Completed(3)`
  - Feed 事件：`bounty_completed(37)` — 受众：全局
- **Fixture Hint**: `bounty(id=1, status=Completed)`

### 步骤 2a.9：Organizer 标记为已支付（Completed → Paid）

- **角色**：`organizer`
- **操作**：触发向 hacker1 支付 Credits
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/1/pay`
- **预期结果**：
  - Bounty 状态：`Completed(3)` → `Paid(4)`
  - 托管的 Credits 释放给 hacker1：`escrow_release` 交易
  - hacker1 余额：500（来自 J1）+ 300 = 800
  - Feed 事件：`bounty_paid(43)` — 受众：全局
- **Fixture Hint**: `bounty(id=1, status=Paid)`, `credit_transaction(escrow_release, hacker1, +300)`

### 状态快照：旅程 2a 结束后

| 实体 | 状态 |
|------|------|
| Bounty #1（独占） | `Paid(4)` |
| organizer Credits | 500 (200 + 600 - 300 托管后已释放) |
| hacker1 Credits | 800 (500 + 300) |

---

### 旅程 2b：竞争赏金

### 步骤 2b.1：Organizer 创建 Issue #2

- **角色**：`organizer`
- **API**: `POST /api/v1/repos/organizer/oss-project/issues`
  ```json
  { "title": "Design new landing page", "body": "We need a fresh landing page design..." }
  ```
- **预期结果**：Issue #2 已创建
- **Fixture Hint**: `issue(repo=organizer/oss-project, index=2)`

### 步骤 2b.2：Organizer 创建带多个奖励的竞争 Bounty

- **角色**：`organizer`
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties`
  ```json
  {
    "issue_index": 2,
    "mode": 1,
    "description": "Best landing page design wins",
    "deadline": "2026-05-15T00:00:00Z"
  }
  ```
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/2/rewards` (x3)
  ```json
  { "place": 1, "amount": 150, "description": "1st place" }
  { "place": 2, "amount": 100, "description": "2nd place" }
  { "place": 3, "amount": 50,  "description": "3rd place" }
  ```
- **预期结果**：
  - Bounty #2 已创建，模式 = `Competitive(1)`，状态 = `Open(0)`
  - 3 个奖励合计 300 Credits
  - 从 organizer 托管 Credits：500 → 200（300 被持有）
  - 交易类型：`escrow`
  - Feed 事件：`bounty_created(34)`
- **Fixture Hint**: `bounty(id=2, mode=Competitive, status=Open)`, `bounty_reward(place=1, amount=150)` etc.

### 步骤 2b.3：hacker1 Watch 仓库（社交）

- **角色**：`hacker1`
- **操作**：Watch 仓库以获取赏金更新通知
- **API**: `PUT /api/v1/repos/organizer/oss-project/subscription`（Forgejo 原生接口）
- **预期结果**：hacker1 已订阅仓库通知
- **Fixture Hint**: `watch(user=hacker1, repo=organizer/oss-project)`

### 步骤 2b.4：三位参与者提交解决方案

- **角色**：`hacker1`、`hacker2`、`platform-bot`
- **操作**：每人提交包含其设计的 PR
- **Git 操作**：每人 Fork（如未 Fork 过）并创建 PR
  - hacker1：从 `hacker1/oss-project` 创建 PR（已在 2a 中 Fork）
  - hacker2：`POST /api/v1/repos/organizer/oss-project/forks`，然后创建 PR
  - platform-bot：`POST /api/v1/repos/organizer/oss-project/forks`（通过 PAT），然后创建 PR
- **API**：每人创建申请
  - `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/2/applications` (x3)
- **预期结果**：
  - 仓库上创建了 3 个 PR
  - 3 个申请已创建，状态 = `Accepted(1)`（竞争模式自动接受所有申请 — 无 Pending 门控）
  - Bounty 保持 `Open(0)`（竞争模式允许多个并发参与者）
  - 注意：`BountyApplication Pending(0)` 仅在独占模式下有意义（已在 J2a 步骤 2a.4 中覆盖）
- **Fixture Hint**: `bounty_application(bounty_id=2, user_id=hacker1, status=Accepted)`, same for hacker2 and platform-bot

### 步骤 2b.5：Organizer 选择获胜者

- **角色**：`organizer`
- **操作**：审查所有提交并选出第 1、2、3 名
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/2/winners`
  ```json
  {
    "winners": [
      { "user_id": <hacker1_id>, "place": 1 },
      { "user_id": <hacker2_id>, "place": 2 },
      { "user_id": <platform-bot_id>, "place": 3 }
    ]
  }
  ```
- **预期结果**：
  - Bounty 状态：`Open(0)` → `Completed(3)`
  - 获胜者及名次已记录
  - Feed 事件：`bounty_winners_selected(38)` — 受众：全局
- **Fixture Hint**: `bounty_winner(bounty_id=2, user_id=hacker1, place=1)` etc.

### 步骤 2b.6：Organizer 向获胜者支付（Completed → Paid）

- **角色**：`organizer`
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/2/pay`
- **预期结果**：
  - Bounty 状态：`Completed(3)` → `Paid(4)`
  - 托管的 Credits 分发给获胜者：
    - hacker1：+150（余额：500+300+150 = 950）
    - hacker2：+100（余额：300+100 = 400）
    - platform-bot：+50（余额：0+50 = 50）
  - 交易类型：每位获胜者为 `escrow_release`
  - Feed 事件：`bounty_paid(43)`
- **Fixture Hint**: `bounty(id=2, status=Paid)`, `credit_transaction(escrow_release, hacker1, +150)`, `credit_transaction(escrow_release, hacker2, +100)`, `credit_transaction(escrow_release, platform-bot, +50)`

### 状态快照：旅程 2b 结束后

| 实体 | 状态 |
|------|------|
| Bounty #2（竞争） | `Paid(4)` |
| organizer Credits | 200 (200 + 600 - 300 - 300 全部托管后已释放) |
| hacker1 Credits | 950 (500 + 300 + 150) |
| hacker2 Credits | 400 (300 + 100) |
| platform-bot Credits | 50 |

---

### 旅程 2c：异常路径

### 步骤 2c.1：Bounty 过期（截止日期已过，无完成提交）

- **角色**：系统（定时任务或 `organizer` 手动触发）
- **场景**：Organizer 在 Issue #3 上创建 Bounty #3 并设置 100 Credits 奖励，截止日期已过但无人提交
- **准备**：
  - `POST /api/v1/repos/organizer/oss-project/issues` → Issue #3
  - `POST /api/v1/repos/organizer/oss-project/hackforger/bounties` → Bounty #3（独占模式，截止日期设为过去时间用于测试）
  - 注意：此时 organizer 余额为 200 Credits。为此测试，admin 额外充值 100。
  - `POST /api/v1/hackforger/credits/admin/deposit` → organizer +100
  - `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/3/rewards` → 100 Credits（托管）
- **触发**：截止日期到达，或手动触发过期
- **预期结果**：
  - Bounty 状态：`Open(0)` → `Expired(5)`
  - 托管的 Credits 退还给 organizer：`escrow_refund` 交易
  - organizer 余额：200 + 100（充值）- 100（托管）+ 100（退还）= 300
  - Feed 事件：`bounty_expired(52)`
- **Fixture Hint**: `bounty(id=3, status=Expired)`, `credit_transaction(escrow_refund, organizer, +100)`

### 步骤 2c.2：Organizer 取消 Bounty

- **角色**：`organizer`
- **场景**：Organizer 在 Issue #4 上创建 Bounty #4，然后取消
- **准备**：类似 2c.1 — 创建 Issue、Bounty、奖励（托管）
  - admin 再向 organizer 充值 100
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/4/cancel`
- **预期结果**：
  - Bounty 状态：`Open(0)` → `Cancelled(6)`
  - 托管的 Credits 退还给 organizer：`escrow_refund`
  - Feed 事件：`bounty_cancelled(53)`
- **Fixture Hint**: `bounty(id=4, status=Cancelled)`, `credit_transaction(escrow_refund, organizer, +100)`

### 步骤 2c.3：申请被拒绝

- **角色**：`organizer`、`hacker2`
- **场景**：hacker2 申请 Bounty #1（但 hacker1 已被接受 — 独占模式）
- **注意**：此步骤在概念上发生在 2a.5 之前。在 fixture 中，测试对已被领取的独占赏金的第二个申请会被拒绝。
- **API**: `PUT /api/v1/repos/organizer/oss-project/hackforger/bounties/1/applications/<hacker2_aid>/review`
  ```json
  { "status": 2, "comment": "Bounty already claimed" }
  ```
- **预期结果**：
  - 申请状态：`Pending(0)` → `Rejected(2)`
  - Bounty 保持 `Claimed(1)`（无变更）
- **Fixture Hint**: `bounty_application(bounty_id=1, user_id=hacker2, status=Rejected)`

### 状态快照：旅程 2 全部结束后

| 实体 | 状态 |
|------|------|
| Bounty #1（独占） | `Paid(4)` |
| Bounty #2（竞争） | `Paid(4)` |
| Bounty #3（过期） | `Expired(5)` |
| Bounty #4（已取消） | `Cancelled(6)` |
| 所有 BountyApplication 状态 | Pending、Accepted、Rejected 均已覆盖 |
| organizer Credits | 400（200 J1 剩余 + 600 充值 - 600 托管/释放于 J2a+J2b + 2×100 admin 充值于 J2c，退还后净值 +200） |
| hacker1 Credits | 950 |
| hacker2 Credits | 400 |
| platform-bot Credits | 50 |

---
