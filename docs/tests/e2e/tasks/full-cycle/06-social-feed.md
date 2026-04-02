# Phase 9: 社交 + Feed + 搜索

> **前置条件**: 完成 `00-setup.md` + `01-hackathon-lifecycle.md` + `02-bounty-collab.md` + `03-grant-funding.md` + `04-submission-judging.md`

本文件覆盖 Feed 系统验证、Explore 页面各 Tab、全局搜索（Cmd+K）、声誉排行榜、Reaction 互动。
环境/账号/Session 信息请参考 [00-setup.md](./00-setup.md)。

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

## 本阶段结果

| 步骤 | 状态 | 截图 |
|------|------|------|
| Step 9.1: Dashboard Feed (Following) | PASS/FAIL | screenshots/full-cycle/p9-01-dashboard-feed.png |
| Step 9.2: Global Feed | PASS/FAIL | screenshots/full-cycle/p9-02-global-feed.png |
| Step 9.3: Explore Hackathons | PASS/FAIL | screenshots/full-cycle/p9-03-explore-hackathons.png |
| Step 9.4: Explore Bounties | PASS/FAIL | screenshots/full-cycle/p9-04-explore-bounties.png |
| Step 9.5: Explore Grants | PASS/FAIL | screenshots/full-cycle/p9-05-explore-grants.png |
| Step 9.6: Explore Submissions | PASS/FAIL | screenshots/full-cycle/p9-06-explore-submissions.png |
| Step 9.7: Cmd+K 全局搜索 | PASS/FAIL | screenshots/full-cycle/p9-07-search-cmdk.png |
| Step 9.8: Reputation 排行榜 | PASS/FAIL | screenshots/full-cycle/p9-08-reputation-leaderboard.png |
| Step 9.9: Reaction 互动 | PASS/FAIL | screenshots/full-cycle/p9-09-reaction.png |
