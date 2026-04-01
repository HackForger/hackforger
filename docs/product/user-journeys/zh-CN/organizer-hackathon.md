## 旅程 1：Hackathon 完整生命周期

### 前置条件
- 全部 7 个用户已存在
- `admin` 已向 `organizer` 账户充值 1000 Credits 用于奖金
- 尚无 Hackathon 存在

### 步骤 1.1：Admin 向 organizer 充值 Credits

- **角色**：`admin`
- **操作**：向 organizer 账户充值 1000 Credits 作为 Hackathon 奖金
- **API**: `POST /api/v1/hackforger/credits/admin/deposit`
  ```json
  { "username": "organizer", "amount": 1000, "reason": "Hackathon prize pool" }
  ```
- **预期结果**：
  - organizer 余额 = 1000
  - 交易类型：`admin_deposit`
- **Fixture Hint**: `credit_account(organizer, balance=1000)`, `credit_transaction(admin_deposit, 1000)`

### 步骤 1.2：Organizer 创建 Hackathon（Draft 状态）

- **角色**：`organizer`
- **操作**：创建名为 "Web3 Innovation Challenge" 的新 Hackathon
- **API**: `POST /api/v1/hackforger/hackathons`
  ```json
  {
    "name": "Web3 Innovation Challenge",
    "slug": "web3-innovation",
    "description": "Build the future of decentralized web",
    "max_tracks": 3,
    "prize_pool": 800
  }
  ```
- **Git 操作**：自动在 Forgejo 上创建组织 `web3-innovation`
- **预期结果**：
  - Hackathon 已创建，状态 = `Draft(0)`
  - 组织 `web3-innovation` 已存在，`organizer` 为所有者
  - Feed 事件：`hackathon_created(30)` — 受众：全局
- **Fixture Hint**: `hackathon(id=1, status=Draft, org_id=<auto>)`

### 步骤 1.3：Organizer 创建 Track（仓库）

- **角色**：`organizer`
- **操作**：在 Hackathon 下创建 2 个 Track
- **API**: `POST /api/v1/hackforger/hackathons/1/tracks` (x2)
  ```json
  // Track 1
  { "name": "DeFi Track", "slug": "defi-track", "description": "Build DeFi protocols", "prize_allocation": 500 }
  // Track 2
  { "name": "NFT Track", "slug": "nft-track", "description": "Build NFT tooling", "prize_allocation": 300 }
  ```
- **Git 操作**：自动创建仓库 `web3-innovation/defi-track` 和 `web3-innovation/nft-track`，包含初始 README 模板
- **预期结果**：
  - 2 个 Track 已创建，每个关联到一个仓库
  - 仓库已有包含 README + 规则的初始提交
  - 奖金分配总额（800）≤ 奖金池（800）✓
- **Fixture Hint**: `hackathon_track(id=1, hackathon_id=1, repo_id=<auto>)`, `hackathon_track(id=2, ...)`

### 步骤 1.4：Organizer 设置评审标准

- **角色**：`organizer`
- **操作**：为每个 Track 定义评分标准
- **API**: `PUT /api/v1/hackforger/hackathons/1/tracks/1`（评审标准包含在 Track 更新请求体中）
  ```json
  // 针对 DeFi Track
  { "criteria": [
    { "name": "Innovation", "weight": 40, "description": "Novelty of approach" },
    { "name": "Technical Quality", "weight": 35, "description": "Code quality, architecture" },
    { "name": "Presentation", "weight": 25, "description": "Demo and documentation" }
  ]}
  ```
- **预期结果**：
  - 每个 Track 3 个评审标准，权重总和为 100
- **Fixture Hint**: `hackathon_track_criteria(track_id=1, name="Innovation", weight=40)` x3 per track

### 步骤 1.5：Organizer 指派评委

- **角色**：`organizer`
- **操作**：将 judge1 和 judge2 添加为 Hackathon 评委
- **API**: `POST /api/v1/hackforger/hackathons/1/judges` (x2)
  ```json
  { "user_id": <judge1_id> }
  { "user_id": <judge2_id> }
  ```
- **预期结果**：
  - 2 位评委已添加到 Hackathon
  - 评委不能同时注册为参赛者（强制执行）
- **Fixture Hint**: `hackathon_judge(hackathon_id=1, user_id=judge1)`, same for judge2

### 步骤 1.6：Organizer 发布 Hackathon（Draft → Open）

- **角色**：`organizer`
- **操作**：发布 Hackathon，开放注册
- **API**: `POST /api/v1/hackforger/hackathons/1/publish`
- **前置条件检查**：至少存在 1 个 Track，至少指派了 1 位评委
- **预期结果**：
  - 状态变更：`Draft(0)` → `Open(1)`
  - Feed 事件：`hackathon_phase_changed(50)` — 受众：全局
  - Hackathon 在探索页面可见
- **Fixture Hint**: `hackathon(id=1, status=Open)`

### 步骤 1.7：hacker1 关注 organizer（社交）

- **角色**：`hacker1`
- **操作**：关注 organizer 以在 Feed 中查看其动态
- **API**: `PUT /api/v1/user/following/organizer`（Forgejo 原生接口）
- **预期结果**：
  - hacker1 现已关注 organizer
  - organizer 的后续操作会出现在 hacker1 的关注 Feed 中
- **Fixture Hint**: `follow(follower=hacker1, followee=organizer)`

### 步骤 1.8：hacker1 注册 Hackathon

- **角色**：`hacker1`
- **操作**：注册参加 Hackathon
- **API**: `POST /api/v1/hackforger/hackathons/1/registrations`
  ```json
  { "track_id": 1 }
  ```
- **Git 操作**：hacker1 被添加为组织 `web3-innovation` 的成员
- **预期结果**：
  - 注册已创建，状态 = `Approved(1)`（未配置人工审核时自动批准）
  - hacker1 现在是 `web3-innovation` 组织的成员
  - Feed 事件：`hackathon_registered(31)` — 受众：关注者 + 组织
- **Fixture Hint**: `hackathon_registration(hackathon_id=1, user_id=hacker1, status=Approved, track_id=1)`

### 步骤 1.9：hacker2 注册并组建团队（社交）

- **角色**：`hacker2`
- **操作**：注册同一 Hackathon 的同一 Track
- **API**: `POST /api/v1/hackforger/hackathons/1/registrations`
  ```json
  { "track_id": 1 }
  ```
- **Git 操作**：hacker2 被添加为组织 `web3-innovation` 的成员
- **社交**：hacker1 和 hacker2 互相关注
  - `PUT /api/v1/user/following/hacker2`（由 hacker1 操作）
  - `PUT /api/v1/user/following/hacker1`（由 hacker2 操作）
- **组建团队**：在组织中创建团队 "DeFi Duo"
  - `POST /api/v1/orgs/web3-innovation/teams`（由 organizer 或 hacker2 操作）
    ```json
    { "name": "DeFi Duo", "permission": "write" }
    ```
  - `PUT /api/v1/teams/<team_id>/members/hacker1`
  - `PUT /api/v1/teams/<team_id>/members/hacker2`
- **预期结果**：
  - hacker2 已注册，获得组织成员资格
  - 团队 "DeFi Duo" 已创建，包含两位黑客
  - hacker1 ↔ hacker2 互相关注
  - Feed 事件：`hackathon_registered(31)` 由 hacker2 触发
- **Fixture Hint**: `hackathon_registration(user_id=hacker2, track_id=1)`, `team(org=web3-innovation, name="DeFi Duo")`, `team_member(hacker1, hacker2)`

### 步骤 1.10：Organizer 启动 Hacking 阶段（Open → Hacking）

- **角色**：`organizer`
- **操作**：将 Hackathon 转入 Hacking 阶段
- **API**: `POST /api/v1/hackforger/hackathons/1/start`
- **预期结果**：
  - 状态变更：`Open(1)` → `Hacking(2)`
  - Feed 事件：`hackathon_phase_changed(50)` — 受众：全局 + 组织
- **Fixture Hint**: `hackathon(id=1, status=Hacking)`

### 步骤 1.11：hacker1 Fork Track 仓库并开发

- **角色**：`hacker1`
- **操作**：Fork DeFi Track 仓库开始编码
- **Git 操作**：
  - `POST /api/v1/repos/web3-innovation/defi-track/forks`（Forgejo 原生接口）
    ```json
    { "organization": "" }
    ```
  - 创建 `hacker1/defi-track` 作为 Fork
  - hacker1 向其 Fork 推送提交
- **预期结果**：
  - Fork `hacker1/defi-track` 已存在
  - hacker1 可以向其 Fork 推送代码
- **Fixture Hint**: `repo(owner=hacker1, name=defi-track, fork_of=web3-innovation/defi-track)`

### 步骤 1.12：hacker2 Fork 并开发（团队提交）

- **角色**：`hacker2`
- **操作**：Fork 同一 Track 仓库用于团队提交
- **Git 操作**：`POST /api/v1/repos/web3-innovation/defi-track/forks`
  - 创建 `hacker2/defi-track`
  - hacker1 和 hacker2 均可向 hacker2 的 Fork 推送（团队仓库）
- **预期结果**：
  - Fork `hacker2/defi-track` 已存在
  - 团队 "DeFi Duo" 可在此 Fork 上协作
- **Fixture Hint**: `repo(owner=hacker2, name=defi-track, fork_of=web3-innovation/defi-track)`

### 步骤 1.13：为 Track 仓库点 Star（社交）

- **角色**：`hacker1`、`judge1`、`platform-bot`
- **操作**：为 DeFi Track 仓库点 Star（社区兴趣信号）
- **API**: `PUT /api/v1/user/starred/web3-innovation/defi-track`（x3，由不同用户操作）
- **预期结果**：
  - `web3-innovation/defi-track` 获得 3 个 Star
  - Star 在仓库页面可见，作为热度/投票信号
- **Fixture Hint**: `star(user=hacker1, repo=web3-innovation/defi-track)` x3

### 步骤 1.14：hacker1 通过 PR 提交

- **角色**：`hacker1`
- **操作**：从 Fork 向 Track 仓库创建 PR，然后注册为 Hackathon 提交
- **Git 操作**：
  - `POST /api/v1/repos/web3-innovation/defi-track/pulls`（Forgejo 原生接口）
    ```json
    {
      "title": "DeFi Lending Protocol - hacker1 submission",
      "head": "hacker1:main",
      "base": "main",
      "body": "My DeFi lending protocol submission"
    }
    ```
- **API**: `POST /api/v1/hackforger/hackathons/1/submissions`
  ```json
  {
    "track_id": 1,
    "title": "DeFi Lending Protocol",
    "description": "A decentralized lending protocol with flash loans",
    "pull_request_id": <pr_id>
  }
  ```
- **预期结果**：
  - PR 已在 `web3-innovation/defi-track` 上创建
  - 提交已关联到 PR（带时间戳、可比较差异、可评审）
  - Feed 事件：`hackathon_submitted(32)` — 受众：组织 + 关注者
- **Fixture Hint**: `hackathon_submission(id=1, hackathon_id=1, track_id=1, user_id=hacker1, pull_id=<auto>)`

### 步骤 1.15：hacker2 通过 PR 提交（团队提交）

- **角色**：`hacker2`
- **操作**：与 1.14 相同，但从 hacker2 的 Fork 提交
- **Git 操作**：`POST /api/v1/repos/web3-innovation/defi-track/pulls`
  ```json
  {
    "title": "DeFi Yield Aggregator - Team DeFi Duo",
    "head": "hacker2:main",
    "base": "main"
  }
  ```
- **API**: `POST /api/v1/hackforger/hackathons/1/submissions`
  ```json
  {
    "track_id": 1,
    "title": "DeFi Yield Aggregator",
    "description": "Cross-chain yield aggregator by Team DeFi Duo",
    "pull_request_id": <pr_id>
  }
  ```
- **预期结果**：
  - DeFi Track 上的第二个提交
  - Feed 事件：`hackathon_submitted(32)` 由 hacker2 触发
- **Fixture Hint**: `hackathon_submission(id=2, user_id=hacker2, track_id=1, pull_id=<auto>)`

### 状态快照：评审前

| 实体 | 状态 |
|------|------|
| Hackathon "Web3 Innovation" | `Hacking(2)` |
| DeFi Track | 2 个提交（hacker1、hacker2） |
| NFT Track | 0 个提交 |
| hacker1 | 已注册、已提交 PR、关注了 organizer + hacker2 |
| hacker2 | 已注册、已提交 PR、关注了 hacker1 |
| judge1、judge2 | 已指派、尚未评分 |
| 组织 web3-innovation | 成员：organizer、hacker1、hacker2；团队：DeFi Duo |
| Credits（organizer） | 1000（未使用） |

### 步骤 1.16：Organizer 启动评审阶段（Hacking → Judging）

- **角色**：`organizer`
- **操作**：关闭提交，转入评审阶段
- **API**: `POST /api/v1/hackforger/hackathons/1/judge`
- **Web**: `POST /hackforger/hackathons/web3-innovation/manage/judge`
- **预期结果**：
  - 状态变更：`Hacking(2)` → `Judging(3)`
  - 不再接受新提交
  - Feed 事件：`hackathon_phase_changed(50)`
- **Fixture Hint**: `hackathon(id=1, status=Judging)`

### 步骤 1.17：judge1 为提交评分

- **角色**：`judge1`
- **操作**：按评审标准为 DeFi Track 的两个提交评分
- **API**: `POST /api/v1/hackforger/hackathons/1/scores` (x2)
  ```json
  // hacker1 提交的评分
  {
    "submission_id": 1,
    "scores": [
      { "criteria_id": 1, "score": 90 },
      { "criteria_id": 2, "score": 85 },
      { "criteria_id": 3, "score": 80 }
    ]
  }
  // hacker2 提交的评分
  {
    "submission_id": 2,
    "scores": [
      { "criteria_id": 1, "score": 75 },
      { "criteria_id": 2, "score": 80 },
      { "criteria_id": 3, "score": 85 }
    ]
  }
  ```
- **Git 操作**：judge1 可以在每个提交的 PR 上添加 PR Review 评论
- **预期结果**：
  - judge1 对两个提交的评分已记录
  - hacker1 的加权得分（judge1）：90×0.4 + 85×0.35 + 80×0.25 = 36+29.75+20 = 85.75
  - hacker2 的加权得分（judge1）：75×0.4 + 80×0.35 + 85×0.25 = 30+28+21.25 = 79.25
  - Feed 事件：`hackathon_scored(33)` (x2) — 受众：组织
- **Fixture Hint**: `hackathon_judge_score(judge_id=judge1, submission_id=1, criteria_id=1, score=90)` etc.

### 步骤 1.18：judge2 为提交评分

- **角色**：`judge2`
- **操作**：独立为两个提交评分
- **API**: `POST /api/v1/hackforger/hackathons/1/scores` (x2)
  ```json
  // hacker1 的评分
  { "submission_id": 1, "scores": [
    { "criteria_id": 1, "score": 88 }, { "criteria_id": 2, "score": 90 }, { "criteria_id": 3, "score": 75 }
  ]}
  // hacker2 的评分
  { "submission_id": 2, "scores": [
    { "criteria_id": 1, "score": 80 }, { "criteria_id": 2, "score": 78 }, { "criteria_id": 3, "score": 90 }
  ]}
  ```
- **预期结果**：
  - judge2 的评分已记录
  - hacker1 的加权得分（judge2）：88×0.4 + 90×0.35 + 75×0.25 = 35.2+31.5+18.75 = 85.45
  - hacker2 的加权得分（judge2）：80×0.4 + 78×0.35 + 90×0.25 = 32+27.3+22.5 = 81.8
  - hacker1 平均分：(85.75 + 85.45) / 2 = **85.60**
  - hacker2 平均分：(79.25 + 81.80) / 2 = **80.53**
  - hacker1 排名第 1，hacker2 排名第 2
  - Feed 事件：`hackathon_scored(33)` (x2)
- **Fixture Hint**: `hackathon_judge_score(judge_id=judge2, ...)` etc.

### 步骤 1.19：Organizer 结束 Hackathon（Judging → Finished）

- **角色**：`organizer`
- **操作**：结束 Hackathon，确认排名，分发奖金
- **API**: `POST /api/v1/hackforger/hackathons/1/finalize`
- **预期结果**：
  - 状态变更：`Judging(3)` → `Finished(4)`
  - 排名确认：hacker1 = 第 1 名，hacker2 = 第 2 名
  - Credits 从 organizer 奖金池自动分发：
    - hacker1 获得 500 Credits（DeFi Track 第 1 名）
    - hacker2 获得 300 Credits（DeFi Track 第 2 名）
    - 交易类型：获奖者为 `deposit`，organizer 为 `withdraw`
  - 排行榜冻结并发布
  - Feed 事件：`hackathon_finalized(51)` — 受众：全局
- **Fixture Hint**: `hackathon(id=1, status=Finished)`, `credit_transaction(deposit, hacker1, 500)`, `credit_transaction(deposit, hacker2, 300)`

### 步骤 1.20：验证排行榜

- **角色**：任意用户
- **操作**：查看最终排行榜
- **API**: `GET /api/v1/hackforger/hackathons/1/leaderboard`
- **Web**: `/hackforger/hackathons/web3-innovation/leaderboard`
- **预期结果**：
  ```json
  {
    "rankings": [
      { "rank": 1, "user": "hacker1", "submission": "DeFi Lending Protocol", "total_score": 85.60, "prize": 500 },
      { "rank": 2, "user": "hacker2", "submission": "DeFi Yield Aggregator", "total_score": 80.53, "prize": 300 }
    ]
  }
  ```

### 状态快照：Hackathon 结束后

| 实体 | 状态 |
|------|------|
| Hackathon | `Finished(4)` |
| Credits: hacker1 | 500 |
| Credits: hacker2 | 300 |
| Credits: organizer | 200 (1000 - 500 - 300) |
| 已生成的 Feed 事件 | hackathon_created, phase_changed x3, registered x2, submitted x2, scored x4, finalized |

---
