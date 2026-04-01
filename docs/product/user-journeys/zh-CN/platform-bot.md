## 旅程 6：Platform Bot 参与

### 前置条件
- `platform-bot` 是 admin 的默认 bot，持有 PAT（Personal Access Token）
- 所有 API 调用使用 `Authorization: token <PAT>` 头
- 无 Web UI 交互 — 纯 REST API

### 步骤 6.1：platform-bot 发现开放的 Bounty

- **角色**：`platform-bot`
- **API**: `GET /api/v1/hackforger/bounties?status=open`
- **认证**：`Authorization: token <platform-bot-PAT>`
- **预期结果**：
  - 返回所有仓库中开放的 Bounty 列表
  - platform-bot 解析响应以找到匹配其能力的 Bounty

### 步骤 6.2：platform-bot 申请 Bounty

- **角色**：`platform-bot`
- **场景**：假设存在一个新的开放 Bounty（或复用 J2b 中 platform-bot 已参与的 Bounty #2）
- **API**: `POST /api/v1/repos/organizer/oss-project/hackforger/bounties/<id>/applications`
  ```json
  { "proposal": "AI-generated solution. Estimated completion: 2 hours." }
  ```
- **预期结果**：申请已创建

### 步骤 6.3：platform-bot 发现开放的 Hackathon

- **角色**：`platform-bot`
- **API**: `GET /api/v1/hackforger/hackathons?status=open`
- **预期结果**：空列表（唯一的 Hackathon "Web3 Innovation" 状态为 `Finished`）
  - 验证：筛选条件正确排除了非 Open 状态的 Hackathon

> **步骤 6.4-6.5 的 Fixture 需求**：为端到端测试 platform-bot 的 Hackathon 参与流程，需创建第二个 Hackathon "AI Sprint"，状态为 `Open`，包含 1 个 Track。该 Hackathon 仅用于 J6 测试。

### 步骤 6.4：platform-bot 注册 Hackathon

- **角色**：`platform-bot`
- **Fixture 前置条件**：Hackathon "AI Sprint"（id=2）处于 `Open` 状态，包含 Track "AI Track"（id=3，仓库=`ai-sprint/ai-track`）
- **API**: `POST /api/v1/hackforger/hackathons/2/registrations`
  ```json
  { "track_id": 3 }
  ```
- **预期结果**：注册已创建，platform-bot 已添加到组织 `ai-sprint`
- **Fixture Hint**: `hackathon(id=2, slug="ai-sprint", status=Open)`, `hackathon_track(id=3, hackathon_id=2)`, `hackathon_registration(hackathon_id=2, user_id=platform-bot, status=Approved)`

### 步骤 6.5：platform-bot Fork 并提交 PR

- **角色**：`platform-bot`
- **Git 操作（通过 API）**：
  - `POST /api/v1/repos/ai-sprint/ai-track/forks` — Fork Track 仓库
  - 向 Fork 推送代码（通过 Git 协议使用 PAT 认证）
  - `POST /api/v1/repos/ai-sprint/ai-track/pulls` — 创建 PR
    ```json
    { "title": "AI-Generated Solution", "head": "platform-bot:main", "base": "main" }
    ```
- **API**: `POST /api/v1/hackforger/hackathons/2/submissions`
  ```json
  { "track_id": 3, "title": "AI-Generated Solution", "pull_request_id": <pr_id> }
  ```
- **预期结果**：提交已创建并关联到 PR
- **Fixture Hint**: `repo(owner=platform-bot, name=ai-track, fork_of=ai-sprint/ai-track)`, `hackathon_submission(hackathon_id=2, user_id=platform-bot)`

### 步骤 6.6：platform-bot 查询 Credits 余额

- **角色**：`platform-bot`
- **API**: `GET /api/v1/hackforger/credits/balance`
- **预期结果**：
  ```json
  { "balance": 50 }
  ```
  （50 Credits 来自 J2b 中 Bounty #2 第 3 名）

### 步骤 6.7：platform-bot 查看交易历史

- **角色**：`platform-bot`
- **API**: `GET /api/v1/hackforger/credits/transactions`
- **预期结果**：
  - 一笔交易：`escrow_release +50` 来自 Bounty #2

### 步骤 6.8：platform-bot 查看自身声望

- **角色**：`platform-bot`
- **API**: `GET /api/v1/hackforger/reputation/users/platform-bot`
- **预期结果**：反映 Bounty 参与情况的声望分数

### 状态快照：旅程 6 结束后

| 能力 | 已验证 |
|------|--------|
| PAT 认证 | ✓ 所有端点支持 token 认证 |
| 发现 Bounty | ✓ 按状态筛选 |
| 申请 Bounty | ✓ 申请已创建 |
| 发现 Hackathon | ✓ 按状态筛选 |
| 注册 Hackathon | ✓（如果为 Open 状态） |
| Fork + PR 工作流 | ✓ 完整的 Git 流程通过 API |
| 提交 Hackathon 作品 | ✓ 关联到 PR |
| 查询 Credits | ✓ 余额 + 历史 |
| 查询声望 | ✓ 分数检索 |

---
