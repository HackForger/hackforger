# Phase 6-7: 提交参赛作品 + 评审 + Finalize

> **前置条件**: 完成 `00-setup.md` + `01-hackathon-lifecycle.md` + `02-bounty-collab.md` + `03-grant-funding.md`

本文件覆盖参赛作品提交（Fork+PR 和 Link Repo 两种模式）、评审打分、排名预览、结算、积分发放。
环境/账号/Session 信息请参考 [00-setup.md](./00-setup.md)。

---

## Phase 6: 提交参赛作品

**目标**: 展示两种提交模式 -- Fork+PR 和 Link Repo。Star 赛道 Repo 作为社交信号。
**角色**: hacker_eve, hacker_frank, judge_carol
**来源**: J1 Steps 1.11-1.15

### Step 6.1: Hacker1 在 Fork 中继续开发
- **角色**: hacker_eve session
- **UI 路径**: 导航到 `hacker_eve/web-track` → 编辑文件
- **操作**:
  - 点击"创建新文件"→ 文件名 = `src/app.md` → 使用 Markdown 编写应用描述：
    ```
    ## Web App Toolkit
    Core module implementing:
    - Component library
    - State management utilities
    - API integration layer
    ```
  - 提交 commit（message = `feat: add core app module`）
- **验证**:
  - 新文件已提交到 `hacker_eve/web-track`
- **截图**: `screenshots/full-cycle/p6-01-hacker1-develop.png`

### Step 6.2: Hacker1 通过 Fork+PR 模式提交作品
- **角色**: hacker_eve session
- **UI 路径**: 导航到 `hacker_eve/web-track` → 点击"新建 Pull Request"
- **操作**:
  - 选择 Base Repo = `web3-innovation/web-track`，Base Branch = `main`
  - Head Repo = `hacker_eve/web-track`，Head Branch = `main`
  - 标题 = `Web App Toolkit - hacker_eve submission`
  - 描述（Markdown）= `My web application toolkit submission for the Web Track.`
  - 点击"创建 Pull Request"
  - PR 创建成功后，导航到 Hackathon 提交页面 `/hackathon/web3-innovation/submit`
  - 选择赛道 = `Web Track`
  - 填写标题 = `Web App Toolkit`
  - 填写描述 = `A modular web application toolkit with component library and state management`
  - 关联 PR（选择刚创建的 PR）
  - 点击"提交"
- **验证**:
  - PR 创建在 `web3-innovation/web-track`
  - Submission 已创建并关联 PR（可追溯时间戳、可 diff、可 review）
  - Feed 事件: `hackathon_submitted(32)` -- 组织 + 粉丝可见
- **截图**: `screenshots/full-cycle/p6-02-hacker1-submission-pr.png`

### Step 6.3: Hacker2 Fork DeFi Track Repo 并开发
- **角色**: hacker_frank session
- **UI 路径**: 导航到 DeFi Track Repo `/web3-innovation/defi-track` → 点击"Fork"按钮
- **操作**:
  - Fork 到个人账号 (hacker_frank)，确认
  - 进入 `hacker_frank/defi-track`
  - 点击"创建新文件"→ 文件名 = `protocol/defi-core.md` → 使用 Markdown 编写项目描述
  - 提交 commit
- **验证**:
  - Fork `hacker_frank/defi-track` 存在，包含开发 commit
- **截图**: `screenshots/full-cycle/p6-03-hacker2-fork-defi.png`

### Step 6.4: Hacker2 通过 Link Repo 模式提交作品
- **角色**: hacker_frank session
- **UI 路径**: 导航到 Hackathon 提交页面 `/hackathon/web3-innovation/submit`
- **操作**:
  - 选择赛道 = `DeFi Track`
  - 填写标题 = `DeFi Yield Aggregator`
  - 填写描述 = `Cross-chain yield aggregator by Team DeFi Duo`
  - 提交模式选择"Link 已有 Repo"（如果 UI 支持），填写 Repo 链接 = `http://localhost:3000/hacker_frank/defi-track`
  - 或者：通过 Fork+PR 模式，从 `hacker_frank/defi-track` 创建 PR 到 `web3-innovation/defi-track`，然后关联 PR
  - 点击"提交"
- **验证**:
  - DeFi Track 上出现 Submission
  - Feed 事件: `hackathon_submitted(32)` for hacker_frank
- **截图**: `screenshots/full-cycle/p6-04-hacker2-submission-link.png`

### Step 6.5: 多用户 Star 赛道 Repo（社交信号）
- **角色**: hacker_eve session + judge_carol session
- **UI 路径**:
  - hacker_eve session: 导航到 `/web3-innovation/defi-track` → 点击 Star 按钮
  - judge_carol session: 导航到 `/web3-innovation/web-track` → 点击 Star 按钮
- **操作**: 各自点击 Star 按钮
- **验证**:
  - 赛道 Repo 的 Star 数增加
  - Star 在 Repo 页面可见，作为社区兴趣信号
- **截图**: `screenshots/full-cycle/p6-05-star-repos.png`

### Phase 6 状态快照

| 实体 | 状态 |
|------|------|
| Hackathon | `Hacking(2)` |
| Web Track | 1 个 Submission（hacker_eve，Fork+PR 模式）|
| DeFi Track | 1 个 Submission（hacker_frank，Link Repo 模式）|
| Stars | web3-innovation/defi-track 1 star, web3-innovation/web-track 1 star |

---

## Phase 7: 评审 + Finalize

**目标**: Organizer 启动评审，两位评委独立打分，Organizer 查看排名并结算，积分自动发放。
**角色**: hackforger (Organizer), judge_carol, judge_dave, hacker_eve, hacker_frank
**来源**: J1 Steps 1.16-1.20

### Step 7.1: Organizer 启动评审 (Hacking -> Judging)
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage` → 状态管理区域
- **操作**: 点击"开始评审"按钮（POST `/hackathon/web3-innovation/manage/judge`）
- **验证**:
  - 状态变更：`Hacking(2)` -> `Judging(3)`
  - 不再接受新提交
  - Feed 事件: `hackathon_phase_changed(50)`
- **截图**: `screenshots/full-cycle/p7-01-judging-started.png`

### Step 7.2: Judge1 为所有提交评分
- **角色**: judge_carol session
- **UI 路径**: 导航到评审页面 `/hackathon/web3-innovation/judge`
- **操作**:
  - 页面列出所有待评分的 Submission
  - 为 hacker_eve 的 "Web App Toolkit"（Web Track）评分：
    - Innovation = `90`，Technical Quality = `85`，Presentation = `80`
    - 点击"提交评分"（POST `/hackathon/web3-innovation/judge/{sid}/scores`）
  - 为 hacker_frank 的 "DeFi Yield Aggregator"（DeFi Track）评分：
    - Innovation = `75`，Technical Quality = `80`，Presentation = `85`
    - 点击"提交评分"
- **验证**:
  - 评分保存成功
  - hacker_eve 加权分 (judge_carol)：90x0.4 + 85x0.35 + 80x0.25 = 85.75
  - hacker_frank 加权分 (judge_carol)：75x0.4 + 80x0.35 + 85x0.25 = 79.25
  - Feed 事件: `hackathon_scored(33)` x2 -- 组织可见
- **截图**: `screenshots/full-cycle/p7-02-judge1-scores.png`

### Step 7.3: Judge2 为所有提交评分
- **角色**: judge_dave session
- **UI 路径**: 导航到评审页面 `/hackathon/web3-innovation/judge`
- **操作**:
  - 为 hacker_eve 的提交评分：
    - Innovation = `88`，Technical Quality = `90`，Presentation = `75`
    - 提交评分
  - 为 hacker_frank 的提交评分：
    - Innovation = `80`，Technical Quality = `78`，Presentation = `90`
    - 提交评分
- **验证**:
  - hacker_eve 加权分 (judge_dave)：88x0.4 + 90x0.35 + 75x0.25 = 85.45
  - hacker_frank 加权分 (judge_dave)：80x0.4 + 78x0.35 + 90x0.25 = 81.80
  - hacker_eve 平均分：(85.75 + 85.45) / 2 = **85.60**（排名 #1）
  - hacker_frank 平均分：(79.25 + 81.80) / 2 = **80.53**（排名 #2）
  - Feed 事件: `hackathon_scored(33)` x2
- **截图**: `screenshots/full-cycle/p7-03-judge2-scores.png`

### Step 7.4: Organizer 查看排名预览
- **角色**: hackforger session
- **UI 路径**: 管理页面 `/hackathon/web3-innovation/manage` → 点击"结算预览" → `/hackathon/web3-innovation/manage/finalize-preview`
- **操作**: 查看排名预览页面
- **验证**:
  - 排名预览显示：
    - #1: hacker_eve, 85.60 分, 奖金（积分）500（Web Track 冠军？或按总奖金分配）
    - #2: hacker_frank, 80.53 分, 奖金（积分）300
  - 法币字段不显示
- **截图**: `screenshots/full-cycle/p7-04-finalize-preview.png`

### Step 7.5: Organizer 确认结算 (Judging -> Finished)
- **角色**: hackforger session
- **UI 路径**: 结算预览页面 → 点击"确认结算"按钮（POST `/hackathon/web3-innovation/manage/finalize`）
- **操作**: 确认结算
- **验证**:
  - 状态变更：`Judging(3)` -> `Finished(4)`
  - 排名锁定并公开
  - 积分自动发放：
    - hacker_eve: +500 Credits（第 1 名，Web Track 奖金 300 + 排名奖金分配）
    - hacker_frank: +300 Credits（第 2 名）
    - hackforger (organizer): 扣除对应积分
  - 排行榜冻结并发布
  - Feed 事件: `hackathon_finalized(51)` -- 全局可见
- **截图**: `screenshots/full-cycle/p7-05-finalized.png`

### Step 7.6: Hacker1 查看积分余额
- **角色**: hacker_eve session
- **UI 路径**: 导航到积分概览页 `/credits`
- **操作**: 查看余额变化
- **验证**:
  - 余额增加了 Hackathon 奖金（具体金额取决于奖金分配规则）
  - 交易记录显示：`deposit` +XXX（Hackathon Web3 Innovation 第 1 名）
- **截图**: `screenshots/full-cycle/p7-06-hacker1-credits-hackathon.png`

### Step 7.7: Hacker2 查看积分余额
- **角色**: hacker_frank session
- **UI 路径**: 导航到积分概览页 `/credits`
- **操作**: 查看余额变化
- **验证**:
  - 余额增加了 Hackathon 奖金
  - 交易记录显示新的 `deposit` 条目
- **截图**: `screenshots/full-cycle/p7-07-hacker2-credits-hackathon.png`

### Step 7.8: 查看排行榜
- **角色**: hacker_eve session（或任意 session）
- **UI 路径**: 导航到排行榜页面 `/hackathon/web3-innovation/leaderboard`
- **操作**: 查看最终排行榜
- **验证**:
  - #1: hacker_eve，"Web App Toolkit"，总分 85.60，奖金（积分）
  - #2: hacker_frank，"DeFi Yield Aggregator"，总分 80.53，奖金（积分）
  - 仅显示积分奖金，法币列不显示
- **截图**: `screenshots/full-cycle/p7-08-leaderboard.png`

### Phase 7 状态快照

| 实体 | 状态 |
|------|------|
| Hackathon | `Finished(4)` |
| hacker_eve | 排名 #1，获得积分奖金 |
| hacker_frank | 排名 #2，获得积分奖金 |
| 评委 | judge_carol + judge_dave 均已评分 |

---

## 本阶段结果

| 步骤 | 状态 | 截图 |
|------|------|------|
| Step 6.1: Hacker1 继续开发 | PASS/FAIL | screenshots/full-cycle/p6-01-hacker1-develop.png |
| Step 6.2: Hacker1 Fork+PR 模式提交 | PASS/FAIL | screenshots/full-cycle/p6-02-hacker1-submission-pr.png |
| Step 6.3: Hacker2 Fork DeFi Track 并开发 | PASS/FAIL | screenshots/full-cycle/p6-03-hacker2-fork-defi.png |
| Step 6.4: Hacker2 Link Repo 模式提交 | PASS/FAIL | screenshots/full-cycle/p6-04-hacker2-submission-link.png |
| Step 6.5: 多用户 Star 赛道 Repo | PASS/FAIL | screenshots/full-cycle/p6-05-star-repos.png |
| Step 7.1: Organizer 启动评审 | PASS/FAIL | screenshots/full-cycle/p7-01-judging-started.png |
| Step 7.2: Judge1 评分 | PASS/FAIL | screenshots/full-cycle/p7-02-judge1-scores.png |
| Step 7.3: Judge2 评分 | PASS/FAIL | screenshots/full-cycle/p7-03-judge2-scores.png |
| Step 7.4: Organizer 查看排名预览 | PASS/FAIL | screenshots/full-cycle/p7-04-finalize-preview.png |
| Step 7.5: Organizer 确认结算 | PASS/FAIL | screenshots/full-cycle/p7-05-finalized.png |
| Step 7.6: Hacker1 查看积分余额 | PASS/FAIL | screenshots/full-cycle/p7-06-hacker1-credits-hackathon.png |
| Step 7.7: Hacker2 查看积分余额 | PASS/FAIL | screenshots/full-cycle/p7-07-hacker2-credits-hackathon.png |
| Step 7.8: 查看排行榜 | PASS/FAIL | screenshots/full-cycle/p7-08-leaderboard.png |
