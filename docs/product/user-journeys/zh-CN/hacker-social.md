## 旅程 5：Feed + 社交 + 发现

### 前置条件
- 所有之前的旅程已完成
- 已生成多个 Feed 事件
- hacker1 关注了 organizer 和 hacker2

### 步骤 5.1：hacker1 查看仪表板 Feed（关注标签页）

- **角色**：`hacker1`
- **Web**: `/`（仪表板，社区标签页）
- **API**: `GET /api/v1/hackforger/feed?type=following&page=1&limit=20`
- **预期结果**：Feed 包含来自已关注用户和全局事件：
  - `hackathon_created(30)` — organizer 创建了 Web3 Innovation
  - `hackathon_phase_changed(50)` — organizer 发布/启动/评审
  - `hackathon_submitted(32)` — hacker2 提交了（hacker1 关注了 hacker2）
  - `bounty_created(34)` — 全局事件
  - `grant_round_opened(54)` — 全局事件
  - 事件按时间倒序排列，分页显示
- **Fixture Hint**: Verify `hackforger_action` rows with correct `user_id` (audience) and `op_type`

### 步骤 5.2：hacker1 查看全局 Feed

- **角色**：`hacker1`
- **API**: `GET /api/v1/hackforger/feed?type=global&page=1&limit=20`
- **预期结果**：所有 HackForger 事件，不考虑关注关系
  - 包含 J1-J4 生成的全部 24 种事件类型
  - 按 `created_unix` 倒序排列

### 步骤 5.3：探索 Hackathon

- **角色**：`hacker1`（或匿名用户）
- **Web**: `/explore/hackforger/hackathons`
- **API**: `GET /api/v1/hackforger/hackathons`
- **预期结果**：
  - 列出 "Web3 Innovation Challenge"，状态为 `Finished`
  - 显示参与者数量、Track 数量、奖金池

### 步骤 5.4：探索 Bounty

- **角色**：`hacker1`
- **Web**: `/explore/hackforger/bounties`
- **API**: `GET /api/v1/hackforger/bounties`
- **预期结果**：
  - 列出所有仓库的所有 Bounty
  - 可用筛选条件：状态、模式
  - 显示 Bounty #1（Paid）、#2（Paid）、#3（Expired）、#4（Cancelled）

### 步骤 5.5：探索 Grant Round

- **角色**：`hacker1`
- **Web**: `/explore/hackforger/grants`
- **API**: `GET /api/v1/hackforger/grants/rounds`
- **预期结果**：
  - 列出 "Q2 2026 Open Source Grant"，状态为 `Distributed`
  - 显示预算、已资助项目数

### 步骤 5.6：跨实体搜索

- **角色**：`hacker1`
- **操作**：在所有 HackForger 实体中搜索 "DeFi"
- **API**: `GET /api/v1/hackforger/search?q=DeFi&scope=all`
- **预期结果**：
  - 结果包括：
    - Hackathon Track "DeFi Track"
    - Hackathon Submission "DeFi Lending Protocol"
    - Grant Project "DeFi Testing Framework"
  - 结果按实体类型分组
- **Web**: `/explore/hackforger/search?q=DeFi`

### 步骤 5.7：带范围筛选的搜索

- **角色**：`hacker1`
- **API**: `GET /api/v1/hackforger/search?q=DeFi&scope=bounties`
- **预期结果**：无结果（没有 Bounty 包含 "DeFi"）
- **API**: `GET /api/v1/hackforger/search?q=OAuth&scope=bounties`
- **预期结果**：Bounty #1 "Implement OAuth2 PKCE flow"

### 步骤 5.8：查看声望排行榜

- **角色**：任意用户
- **API**: `GET /api/v1/hackforger/reputation/leaderboard`
- **Web**: `/explore/hackforger/reputation`
- **预期结果**：
  - hacker1 排名最高（Hackathon 第 1 名 + Bounty 获胜 + Grant 获资助）
  - hacker2 排名第二（Hackathon 第 2 名 + Bounty 第 2 名）
  - platform-bot 排名第三（仅 Bounty 第 3 名）
  - 分数反映来自所有模块的加权贡献

### 步骤 5.9：查看个人声望

- **角色**：`hacker1`
- **API**: `GET /api/v1/hackforger/reputation/users/hacker1`
- **预期结果**：
  ```json
  {
    "username": "hacker1",
    "total_score": 850,
    "tier": "gold",
    "breakdown": {
      "hackathon": 400,
      "bounty": 300,
      "grant": 150
    }
  }
  ```
  （分数仅为示例 — 实际权重定义在 `services/hackforger/reputation.go` 中）

### 步骤 5.10：在 Bounty Issue 上添加表情回应（社交）

- **角色**：`hacker2`
- **操作**：对 hacker1 的赏金完成评论添加表情回应
- **API**: `POST /api/v1/repos/organizer/oss-project/issues/1/comments/<comment_id>/reactions`（Forgejo 原生接口）
  ```json
  { "content": "+1" }
  ```
- **预期结果**：表情回应已添加到评论（Forgejo 原生功能，无需 HackForger 特殊处理）

### 状态快照：旅程 5 结束后

| 功能 | 已验证 |
|------|--------|
| 关注 Feed | ✓ 显示已关注用户的事件 |
| 全局 Feed | ✓ 显示所有事件 |
| 探索：Hackathon | ✓ 可筛选的列表 |
| 探索：Bounty | ✓ 跨仓库列表 |
| 探索：Grant | ✓ 轮次列表 |
| 搜索 | ✓ 跨实体全文搜索，范围筛选 |
| 声望 | ✓ 排行榜 + 个人分数 |
| 社交：关注/Star/Watch/表情回应 | ✓ 贯穿 J1-J5 |

---
