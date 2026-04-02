# 用户旅程全流程 E2E 测试任务书

> **Web 端必选规则**: 所有核心业务操作必须通过浏览器 UI 完成，不可使用 API 替代页面操作。
> API 仅用于设计上就应该通过 API 执行的操作（git clone/push、bot PAT 调用）以及辅助数据准备（密码重置）。
> 每步必须截屏作为测试证据。每步必须有明确的验证标准。

---

## 测试环境

- **服务器**: `http://localhost:3000`（worktree 构建的最新 binary）
- **数据库**: 每轮测试前清空 HackForger 数据（见 `e2e-testing-guide.md` 的 Database Cleanup）
- **截图保存**: `screenshots/full-cycle/`（相对于本文件目录）

## 测试账号与角色映射

| PRD 角色 | 实例用户名 | 密码 | Session 名 | 本次旅程中的职责 |
|---------|-----------|------|-----------|----------------|
| Admin | `hackforger` | `admin1234` | `admin` | 平台管理、积分充值、兑换选项配置 |
| Organizer | `hackforger` | `admin1234` | `admin` | 创建 Hackathon/Bounty/Grant、管理评审（v0.1 与 Admin 同账号） |
| Judge 1 | `judge_carol` | `admin1234` | `judge1` | Hackathon 评委 |
| Judge 2 | `judge_dave` | `admin1234` | `judge2` | Hackathon 评委 |
| Hacker 1 | `hacker_eve` | `admin1234` | `hacker1` | 活跃参赛者、Bounty 认领者、Grant 申请者 |
| Hacker 2 | `hacker_frank` | `admin1234` | `hacker2` | 第二参赛者、组队伙伴 |
| Platform Bot | `platform-bot` | (PAT) | `bot` | 自动化操作、Bounty 参与 |

> **密码重置**: 测试开始前，对所有非 admin 账号通过 admin API 执行密码重置（见 e2e-testing-guide.md）。

## 多用户 Session 设置

使用 `agent-browser --session <name>` 为每个角色创建独立浏览器会话：

```bash
# 登录所有用户（一次性完成）
for pair in "admin hackforger" "judge1 judge_carol" "judge2 judge_dave" "hacker1 hacker_eve" "hacker2 hacker_frank"; do
  set -- $pair
  session=$1 user=$2
  agent-browser --session $session open "http://localhost:3000/user/login"
  agent-browser --session $session snapshot -i
  agent-browser --session $session fill @e13 "$user"
  agent-browser --session $session fill @e14 "admin1234"
  agent-browser --session $session click @e17
  agent-browser --session $session wait --load networkidle
done
```

> platform-bot 通过 PAT 调用 API，无需浏览器 session。

---

## 旅程 1: Hackathon 全生命周期（20 步）

**角色**: Admin + Organizer + Hacker1 + Hacker2 + Judge1 + Judge2
**参考**: `docs/product/user-journeys/organizer-hackathon.md`

### 前置条件
- 数据库已清空 HackForger 数据
- 所有用户已登录各自 session
- 不存在任何 Hackathon

### Step 1.1: Admin 为 Organizer 充值 1000 积分
- **角色**: `admin` session (hackforger)
- **UI 路径**: 顶部导航栏 "站点管理" → 左侧菜单 "积分管理" (`/-/admin/credits`) → 积分管理首页
- **操作**: 在"手动充值"区域填写：用户名 = `hackforger`（Organizer 与 Admin 同账号），金额 = `1000`，原因 = `Hackathon prize pool`，点击"充值"按钮
- **验证**: 页面显示充值成功提示，organizer 余额 = 1000；交易类型 = `admin_deposit`
- **截图**: `screenshots/full-cycle/j1-01-credits-deposit.png`

### Step 1.2: Organizer 创建 Hackathon (Draft)
- **角色**: `admin` session (hackforger 即 Organizer)
- **UI 路径**: 顶部导航栏 "+" 菜单或直接导航 → "创建 Hackathon" (`/hackathons/new`)
- **操作**: 填写表单：
  - 名称 = `Web3 Innovation Challenge`
  - Slug = `web3-innovation`
  - 描述 = `Build the future of decentralized web`
  - 最大赛道数 = `3`
  - 奖金池 = `800`
  - 点击"创建"按钮
- **验证**:
  - 跳转到 Hackathon 详情页 (`/hackathon/web3-innovation`)，状态 = `Draft`
  - Forgejo 组织 `web3-innovation` 已自动创建，organizer 为 owner
  - Feed 事件: `hackathon_created(30)` — 全局可见
- **截图**: `screenshots/full-cycle/j1-02-hackathon-created.png`

### Step 1.3: Organizer 创建赛道 (Track)
- **角色**: `admin` session
- **UI 路径**: Hackathon 详情页 → 点击"管理" → 进入管理页面 (`/hackathon/web3-innovation/manage`)
- **操作**:
  - 在"赛道管理"区域点击"添加赛道"
  - 填写 Track 1: 名称 = `DeFi Track`, Slug = `defi-track`, 描述 = `Build DeFi protocols`, 奖金分配 = `500`，提交
  - 再次点击"添加赛道"
  - 填写 Track 2: 名称 = `NFT Track`, Slug = `nft-track`, 描述 = `Build NFT tooling`, 奖金分配 = `300`，提交
- **验证**:
  - 赛道列表显示 2 个条目
  - 自动创建 Repo `web3-innovation/defi-track` 和 `web3-innovation/nft-track`，包含初始 README
  - 奖金分配总计 (500 + 300 = 800) <= 奖金池 (800)
- **截图**: `screenshots/full-cycle/j1-03-tracks-created.png`

### Step 1.4: Organizer 设置评审标准
- **角色**: `admin` session
- **UI 路径**: 管理页面 (`/hackathon/web3-innovation/manage`) → "评审标准"区域
- **操作**:
  - 为 DeFi Track 添加评审标准:
    - 点击"添加标准"，填写: 名称 = `Innovation`, 权重 = `40`, 描述 = `Novelty of approach`，提交
    - 点击"添加标准"，填写: 名称 = `Technical Quality`, 权重 = `35`, 描述 = `Code quality, architecture`，提交
    - 点击"添加标准"，填写: 名称 = `Presentation`, 权重 = `25`, 描述 = `Demo and documentation`，提交
  - 对 NFT Track 也执行相同操作（或复用相同标准）
- **验证**: 每个赛道的评审标准列表显示 3 个条目，权重之和 = 100
- **截图**: `screenshots/full-cycle/j1-04-criteria-set.png`

### Step 1.5: Organizer 指派评委
- **角色**: `admin` session
- **UI 路径**: 管理页面 (`/hackathon/web3-innovation/manage`) → "评委管理"区域
- **操作**:
  - 在"添加评委"输入框中输入 `judge_carol`，点击"添加"
  - 再输入 `judge_dave`，点击"添加"
- **验证**:
  - 评委列表显示 `judge_carol` 和 `judge_dave`
  - 评委不能同时报名参赛（系统强制）
- **截图**: `screenshots/full-cycle/j1-05-judges-assigned.png`

### Step 1.6: Organizer 发布 Hackathon (Draft -> Open)
- **角色**: `admin` session
- **UI 路径**: 管理页面 (`/hackathon/web3-innovation/manage`) → 状态管理区域
- **操作**: 点击"发布"按钮（POST `/hackathon/web3-innovation/manage/publish`）
- **验证**:
  - 前置检查通过: 至少 1 个赛道、至少 1 个评委
  - 状态变更: `Draft(0)` -> `Open(1)`
  - Feed 事件: `hackathon_phase_changed(50)` — 全局可见
  - 在 Explore 页面 (`/explore/hackathons`) 中可搜索到该 Hackathon
- **截图**: `screenshots/full-cycle/j1-06-published.png`

### Step 1.7: Hacker1 关注 Organizer (社交)
- **角色**: `hacker1` session (hacker_eve)
- **UI 路径**: 顶部导航栏搜索或直接导航到 organizer 个人页面 (`/hackforger`)
- **操作**: 点击页面上的 "Follow" 按钮
- **验证**:
  - 按钮变为 "Unfollow"
  - hacker1 的 Following Feed 中将显示 organizer 未来的活动
- **截图**: `screenshots/full-cycle/j1-07-follow-organizer.png`

### Step 1.8: Hacker1 报名参赛
- **角色**: `hacker1` session
- **UI 路径**: 导航到 Hackathon 详情页 (`/hackathon/web3-innovation`) → 报名区域
- **操作**: 选择赛道 = `DeFi Track`，点击"报名"按钮（POST `/hackathon/web3-innovation/register`）
- **验证**:
  - 报名成功，状态 = `Approved(1)`（自动审批）
  - hacker_eve 被添加为组织 `web3-innovation` 的成员
  - Feed 事件: `hackathon_registered(31)` — 粉丝 + 组织可见
- **截图**: `screenshots/full-cycle/j1-08-hacker1-registered.png`

### Step 1.9: Hacker2 报名并组队 (社交)
- **角色**: `hacker2` session (hacker_frank)
- **UI 路径**: 导航到 Hackathon 详情页 (`/hackathon/web3-innovation`) → 报名区域
- **操作**:
  - 选择赛道 = `DeFi Track`，点击"报名"按钮
  - 报名成功后，hacker1 和 hacker2 互相关注:
    - `hacker1` session: 导航到 `/hacker_frank`，点击 "Follow"
    - `hacker2` session: 导航到 `/hacker_eve`，点击 "Follow"
  - 组队 "DeFi Duo": 由 organizer 在组织管理中创建 Team，或通过 API 创建 Team 并添加成员（此为 Forgejo 原生组织管理操作）:
    - API: `POST /api/v1/orgs/web3-innovation/teams` → `{ "name": "DeFi Duo", "permission": "write" }`
    - API: `PUT /api/v1/teams/<team_id>/members/hacker_eve`
    - API: `PUT /api/v1/teams/<team_id>/members/hacker_frank`
- **验证**:
  - hacker_frank 报名成功，成为 `web3-innovation` 成员
  - Team "DeFi Duo" 创建成功，两人均为成员
  - hacker_eve <-> hacker_frank 互相关注
  - Feed 事件: `hackathon_registered(31)` for hacker_frank
- **截图**: `screenshots/full-cycle/j1-09-hacker2-registered.png`

### Step 1.10: Organizer 启动 Hacking (Open -> Hacking)
- **角色**: `admin` session
- **UI 路径**: 管理页面 (`/hackathon/web3-innovation/manage`) → 状态管理区域
- **操作**: 点击"开始 Hacking"按钮（POST `/hackathon/web3-innovation/manage/start`）
- **验证**:
  - 状态变更: `Open(1)` -> `Hacking(2)`
  - Feed 事件: `hackathon_phase_changed(50)` — 全局 + 组织可见
- **截图**: `screenshots/full-cycle/j1-10-hacking-started.png`

### Step 1.11: Hacker1 Fork 赛道 Repo 并开发
- **角色**: `hacker1` session
- **UI 路径**: 导航到 DeFi Track Repo 页面 (`/web3-innovation/defi-track`) → 点击"Fork"按钮
- **操作**:
  - 在 Fork 确认页面选择 Fork 到个人账号 (hacker_eve)，点击确认
  - Fork 创建完成后，进入 `hacker_eve/defi-track`
  - 点击某个文件（如 README.md）→ 点击编辑按钮 → 添加内容（DeFi Lending Protocol 描述）→ 提交 commit
- **验证**: Fork `hacker_eve/defi-track` 存在，且包含新 commit
- **截图**: `screenshots/full-cycle/j1-11-hacker1-fork.png`

### Step 1.12: Hacker2 Fork 赛道 Repo 并开发 (团队提交)
- **角色**: `hacker2` session
- **UI 路径**: 导航到 DeFi Track Repo 页面 (`/web3-innovation/defi-track`) → 点击"Fork"按钮
- **操作**:
  - Fork 到个人账号 (hacker_frank)，确认
  - 进入 `hacker_frank/defi-track`，编辑文件添加 DeFi Yield Aggregator 内容，提交 commit
- **验证**: Fork `hacker_frank/defi-track` 存在，Team "DeFi Duo" 可在此协作
- **截图**: `screenshots/full-cycle/j1-12-hacker2-fork.png`

### Step 1.13: 多用户 Star 赛道 Repo (社交信号)
- **角色**: `hacker1` session, `judge1` session, `bot` (API)
- **UI 路径**:
  - `hacker1`: 导航到 `/web3-innovation/defi-track` → 点击 Star 按钮
  - `judge1`: 导航到 `/web3-innovation/defi-track` → 点击 Star 按钮
  - `bot`: API `PUT /api/v1/user/starred/web3-innovation/defi-track`（PAT，设计上通过 API）
- **验证**: `web3-innovation/defi-track` 显示 3 个 Star
- **截图**: `screenshots/full-cycle/j1-13-star-repo.png`

### Step 1.14: Hacker1 提交 PR 作为参赛作品
- **角色**: `hacker1` session
- **UI 路径**: 导航到 `hacker_eve/defi-track` → 点击"新建 Pull Request"按钮（或从原 Repo 页面的"新建 PR"入口）
- **操作**:
  - 选择 Base Repo = `web3-innovation/defi-track`, Base Branch = `main`
  - Head Repo = `hacker_eve/defi-track`, Head Branch = `main`
  - 标题 = `DeFi Lending Protocol - hacker_eve submission`
  - 描述 = `My DeFi lending protocol submission`
  - 点击"创建 Pull Request"
  - PR 创建成功后，导航到 Hackathon 提交页面 (`/hackathon/web3-innovation/submit`)
  - 选择赛道 = `DeFi Track`
  - 填写标题 = `DeFi Lending Protocol`
  - 填写描述 = `A decentralized lending protocol with flash loans`
  - 关联 PR（选择刚创建的 PR）
  - 点击"提交"
- **验证**:
  - PR 创建在 `web3-innovation/defi-track`
  - Submission 已创建并关联 PR（可追溯时间戳、可 diff、可 review）
  - Feed 事件: `hackathon_submitted(32)` — 组织 + 粉丝可见
- **截图**: `screenshots/full-cycle/j1-14-hacker1-submission.png`

### Step 1.15: Hacker2 提交 PR 作为参赛作品 (团队提交)
- **角色**: `hacker2` session
- **UI 路径**: 同 Step 1.14，从 `hacker_frank/defi-track` 创建 PR
- **操作**:
  - 创建 PR: 标题 = `DeFi Yield Aggregator - Team DeFi Duo`, Head = `hacker_frank:main`, Base = `main`
  - 导航到提交页面 (`/hackathon/web3-innovation/submit`)
  - 填写: 赛道 = `DeFi Track`, 标题 = `DeFi Yield Aggregator`, 描述 = `Cross-chain yield aggregator by Team DeFi Duo`, 关联 PR
  - 提交
- **验证**:
  - DeFi Track 上出现第 2 个 Submission
  - Feed 事件: `hackathon_submitted(32)` for hacker_frank
- **截图**: `screenshots/full-cycle/j1-15-hacker2-submission.png`

### Step 1.16: Organizer 启动评审 (Hacking -> Judging)
- **角色**: `admin` session
- **UI 路径**: 管理页面 (`/hackathon/web3-innovation/manage`) → 状态管理区域
- **操作**: 点击"开始评审"按钮（POST `/hackathon/web3-innovation/manage/judge`）
- **验证**:
  - 状态变更: `Hacking(2)` -> `Judging(3)`
  - 不再接受新提交
  - Feed 事件: `hackathon_phase_changed(50)`
- **截图**: `screenshots/full-cycle/j1-16-judging-started.png`

### Step 1.17: Judge1 为所有提交评分
- **角色**: `judge1` session (judge_carol)
- **UI 路径**: 导航到评审页面 (`/hackathon/web3-innovation/judge`)
- **操作**:
  - 页面列出所有待评分的 Submission
  - 为 hacker_eve 的 "DeFi Lending Protocol" 评分:
    - Innovation = `90`, Technical Quality = `85`, Presentation = `80`
    - 点击"提交评分"（POST `/hackathon/web3-innovation/judge/{sid}/scores`）
  - 为 hacker_frank 的 "DeFi Yield Aggregator" 评分:
    - Innovation = `75`, Technical Quality = `80`, Presentation = `85`
    - 点击"提交评分"
- **验证**:
  - 评分保存成功
  - hacker_eve 加权分 (judge1): 90x0.4 + 85x0.35 + 80x0.25 = 85.75
  - hacker_frank 加权分 (judge1): 75x0.4 + 80x0.35 + 85x0.25 = 79.25
  - Feed 事件: `hackathon_scored(33)` x2 — 组织可见
- **截图**: `screenshots/full-cycle/j1-17-judge1-scores.png`

### Step 1.18: Judge2 为所有提交评分
- **角色**: `judge2` session (judge_dave)
- **UI 路径**: 导航到评审页面 (`/hackathon/web3-innovation/judge`)
- **操作**:
  - 为 hacker_eve 的提交评分:
    - Innovation = `88`, Technical Quality = `90`, Presentation = `75`
    - 提交评分
  - 为 hacker_frank 的提交评分:
    - Innovation = `80`, Technical Quality = `78`, Presentation = `90`
    - 提交评分
- **验证**:
  - hacker_eve 加权分 (judge2): 88x0.4 + 90x0.35 + 75x0.25 = 85.45
  - hacker_frank 加权分 (judge2): 80x0.4 + 78x0.35 + 90x0.25 = 81.80
  - hacker_eve 平均分: (85.75 + 85.45) / 2 = **85.60** (排名 #1)
  - hacker_frank 平均分: (79.25 + 81.80) / 2 = **80.53** (排名 #2)
  - Feed 事件: `hackathon_scored(33)` x2
- **截图**: `screenshots/full-cycle/j1-18-judge2-scores.png`

### Step 1.19: Organizer 结算 Hackathon (Judging -> Finished)
- **角色**: `admin` session
- **UI 路径**: 管理页面 (`/hackathon/web3-innovation/manage`) → 点击"结算预览" (`/hackathon/web3-innovation/manage/finalize-preview`)
- **操作**:
  - 查看排名预览:
    - #1: hacker_eve, 85.60 分, 奖金 500
    - #2: hacker_frank, 80.53 分, 奖金 300
  - 确认无误后点击"确认结算"按钮（POST `/hackathon/web3-innovation/manage/finalize`）
- **验证**:
  - 状态变更: `Judging(3)` -> `Finished(4)`
  - 排名锁定并公开
  - 积分自动发放:
    - hacker_eve: +500 Credits（DeFi Track 第 1 名）
    - hacker_frank: +300 Credits（DeFi Track 第 2 名）
    - organizer: 1000 - 500 - 300 = 200 Credits 剩余
  - 排行榜冻结并发布
  - Feed 事件: `hackathon_finalized(51)` — 全局可见
- **截图**: `screenshots/full-cycle/j1-19-finalized.png`

### Step 1.20: 验证排行榜
- **角色**: 任意 session
- **UI 路径**: 导航到排行榜页面 (`/hackathon/web3-innovation/leaderboard`)
- **操作**: 查看最终排行榜
- **验证**:
  - #1: hacker_eve, "DeFi Lending Protocol", 总分 85.60, 奖金 500
  - #2: hacker_frank, "DeFi Yield Aggregator", 总分 80.53, 奖金 300
- **截图**: `screenshots/full-cycle/j1-20-leaderboard.png`

### 旅程 1 状态快照

| 实体 | 状态 |
|------|------|
| Hackathon "Web3 Innovation" | `Finished(4)` |
| Credits: hacker_eve | 500 |
| Credits: hacker_frank | 300 |
| Credits: hackforger (organizer) | 200 (1000 - 500 - 300) |
| Feed 事件 | hackathon_created, phase_changed x3, registered x2, submitted x2, scored x4, finalized |

---

## 旅程 2: Bounty 全生命周期（19 步）

**角色**: Admin + Organizer + Hacker1 + Hacker2 + Platform-bot
**参考**: `docs/product/user-journeys/organizer-bounty.md`

### 前置条件
- organizer 拥有 Repo `hackforger/oss-project`（如不存在需先创建）
- 旅程 1 已完成

### Step 2.0: Admin 为 Organizer 充值 600 积分 (Bounty 奖金池)
- **角色**: `admin` session
- **UI 路径**: 顶部导航栏 "站点管理" → 左侧菜单 "积分管理" (`/-/admin/credits`)
- **操作**: 在"手动充值"区域填写: 用户名 = `hackforger`, 金额 = `600`, 原因 = `Bounty reward pool`，点击"充值"
- **验证**: organizer 余额 = 200 (J1 剩余) + 600 = 800
- **截图**: `screenshots/full-cycle/j2-00-credits-deposit.png`

---

### 2a: Exclusive Bounty 独占模式 (9 步)

### Step 2a.1: Organizer 创建 Issue
- **角色**: `admin` session (hackforger 即 Organizer)
- **UI 路径**: 导航到 Repo 页面 (`/hackforger/oss-project`) → 点击 "Issues" Tab → 点击"新建 Issue"
- **操作**: 填写标题 = `Implement OAuth2 PKCE flow`, 内容 = `We need OAuth2 PKCE support...`，点击"提交 Issue"
- **验证**: Issue #1 创建成功，显示在 Issue 列表
- **截图**: `screenshots/full-cycle/j2-01-issue-created.png`

### Step 2a.2: Organizer 在 Issue 上创建 Exclusive Bounty
- **角色**: `admin` session
- **UI 路径**: 导航到 Repo Bounty 创建页面 (`/hackforger/oss-project/bounties/new`)
- **操作**: 填写表单:
  - 关联 Issue = Issue #1
  - 模式 = `Exclusive`
  - 描述 = `Implement OAuth2 PKCE flow with tests`
  - 截止日期 = `2026-05-01`
  - 点击"创建 Bounty"
- **验证**:
  - Bounty 创建成功，状态 = `Open(0)`, 模式 = `Exclusive(0)`
  - Bounty 与 Issue #1 关联 (1:1)
  - Feed 事件: `bounty_created(34)` — 全局 + Repo watcher 可见
- **截图**: `screenshots/full-cycle/j2-02-bounty-created.png`

### Step 2a.3: Organizer 添加 Reward (积分托管)
- **角色**: `admin` session
- **UI 路径**: Bounty 详情页 → Reward 管理区域（如在创建时未设置，则通过 Bounty 管理操作添加）
- **操作**: 设置奖励金额 = `300` Credits, 描述 = `Full implementation with tests`，提交
- **验证**:
  - Reward 创建: 300 Credits
  - 积分托管 (escrow): organizer 余额 800 -> 500 (300 被平台持有)
  - 交易类型: `escrow`
- **截图**: `screenshots/full-cycle/j2-03-reward-set.png`

### Step 2a.4: Hacker1 申请 Bounty
- **角色**: `hacker1` session (hacker_eve)
- **UI 路径**: 导航到 Explore Bounties (`/explore/bounties`) → 找到 "Implement OAuth2 PKCE flow" → 点击进入 Bounty 详情 → 或直接从 Issue 页面的 Bounty 面板进入
- **操作**: 点击"申请认领"按钮，填写申请说明 = `I have experience with OAuth2. Estimated 3 days.`，提交
- **验证**: 申请创建成功，状态 = `Pending(0)`
- **截图**: `screenshots/full-cycle/j2-04-application-submitted.png`

### Step 2a.5: Organizer 接受申请 (Bounty: Open -> Claimed)
- **角色**: `admin` session
- **UI 路径**: 导航到 Bounty 申请列表 (`/hackforger/oss-project/bounties/{bounty_id}/applications`)
- **操作**: 找到 hacker_eve 的申请，点击"接受"按钮（POST `/hackforger/oss-project/bounties/{bounty_id}/applications/{aid}`），附加评论 = `Approved, go ahead`
- **验证**:
  - 申请状态: `Pending(0)` -> `Accepted(1)`
  - Bounty 状态: `Open(0)` -> `Claimed(1)`
  - hacker_eve 成为独占认领者
  - Feed 事件: `bounty_claimed(35)` — Repo watcher + 粉丝可见
- **截图**: `screenshots/full-cycle/j2-05-application-accepted.png`

### Step 2a.6: Hacker1 Fork Repo 并开发
- **角色**: `hacker1` session
- **UI 路径**: 导航到 Repo 页面 (`/hackforger/oss-project`) → 点击"Fork"按钮
- **操作**:
  - Fork 到个人账号 (hacker_eve)，确认
  - 进入 `hacker_eve/oss-project`，编辑文件添加 OAuth2 PKCE 实现代码，提交 commit
- **验证**: Fork `hacker_eve/oss-project` 存在，包含实现 commit
- **截图**: `screenshots/full-cycle/j2-06-hacker1-fork-bounty.png`

### Step 2a.7: Hacker1 提交 PR (Bounty: Claimed -> InReview)
- **角色**: `hacker1` session
- **UI 路径**: 导航到 `hacker_eve/oss-project` → 点击"新建 Pull Request"
- **操作**: 填写 PR:
  - Base Repo = `hackforger/oss-project`, Base Branch = `main`
  - Head = `hacker_eve:main`
  - 标题 = `feat: implement OAuth2 PKCE flow`
  - 描述 = `Closes #1\n\nImplements PKCE with full test coverage`
  - 点击"创建 Pull Request"
- **验证**:
  - PR 创建成功，关联 Issue #1
  - Bounty 状态: `Claimed(1)` -> `InReview(2)`
  - Feed 事件: `bounty_delivered(36)` — Repo watcher 可见
- **截图**: `screenshots/full-cycle/j2-07-delivery-pr.png`

### Step 2a.8: Organizer Review + Complete (InReview -> Completed)
- **角色**: `admin` session
- **UI 路径**: 导航到 PR 页面 → Review PR → 然后导航到 Bounty 管理操作
- **操作**:
  - 在 PR 页面添加 Review（Approve）
  - 通过 Bounty 操作按钮标记为 Complete（POST `/hackforger/oss-project/bounties/{bounty_id}/complete`）
- **验证**:
  - Bounty 状态: `InReview(2)` -> `Completed(3)`
  - Feed 事件: `bounty_completed(37)` — 全局可见
- **截图**: `screenshots/full-cycle/j2-08-bounty-completed.png`

### Step 2a.9: Organizer 标记支付 (Completed -> Paid)
- **角色**: `admin` session
- **UI 路径**: Bounty 详情页 → 操作区域
- **操作**: 点击"支付"按钮（POST `/hackforger/oss-project/bounties/{bounty_id}/pay`）
- **验证**:
  - Bounty 状态: `Completed(3)` -> `Paid(4)`
  - 托管积分释放给 hacker_eve: `escrow_release` 交易
  - hacker_eve 余额: 500 (J1) + 300 = 800
  - Feed 事件: `bounty_paid(43)` — 全局可见
- **截图**: `screenshots/full-cycle/j2-09-bounty-paid.png`

### 2a 状态快照

| 实体 | 状态 |
|------|------|
| Bounty #1 (Exclusive) | `Paid(4)` |
| organizer Credits | 500 (800 - 300 escrowed/released) |
| hacker_eve Credits | 800 (500 + 300) |

---

### 2b: Competitive Bounty 竞赛模式 (6 步)

### Step 2b.1: Organizer 创建 Issue #2
- **角色**: `admin` session
- **UI 路径**: 导航到 Repo (`/hackforger/oss-project`) → Issues → 新建 Issue
- **操作**: 填写标题 = `Design new landing page`, 内容 = `We need a fresh landing page design...`，提交
- **验证**: Issue #2 创建成功
- **截图**: `screenshots/full-cycle/j2-10-issue2-created.png`

### Step 2b.2: Organizer 创建 Competitive Bounty + 多级奖励
- **角色**: `admin` session
- **UI 路径**: 导航到 Bounty 创建页面 (`/hackforger/oss-project/bounties/new`)
- **操作**: 填写表单:
  - 关联 Issue = Issue #2
  - 模式 = `Competitive`
  - 描述 = `Best landing page design wins`
  - 截止日期 = `2026-05-15`
  - 创建 Bounty
  - 在 Bounty 详情/管理页面添加多级 Reward:
    - 1st place: 150 Credits
    - 2nd place: 100 Credits
    - 3rd place: 50 Credits
- **验证**:
  - Bounty #2 创建成功，模式 = `Competitive(1)`, 状态 = `Open(0)`
  - 3 个 Reward 共 300 Credits
  - 积分托管: organizer 余额 500 -> 200 (300 被持有)
  - 交易类型: `escrow`
  - Feed 事件: `bounty_created(34)`
- **截图**: `screenshots/full-cycle/j2-11-competitive-bounty.png`

### Step 2b.3: Hacker1 Watch Repo (社交)
- **角色**: `hacker1` session
- **UI 路径**: 导航到 Repo 页面 (`/hackforger/oss-project`) → 点击"Watch"按钮
- **操作**: 选择 Watch 选项（接收所有通知）
- **验证**: hacker_eve 订阅了 Repo 通知
- **截图**: `screenshots/full-cycle/j2-12-watch-repo.png`

### Step 2b.4: 三人提交方案
- **角色**: `hacker1` session, `hacker2` session, `bot` (API)
- **UI 路径 (hacker1)**: hacker_eve 已有 Fork（来自 2a.6），直接编辑并创建新 PR
- **UI 路径 (hacker2)**: 导航到 Repo → Fork → 编辑 → 创建 PR
- **操作**:
  - `hacker1`: 在 Fork 中编辑 landing page 设计文件 → 创建 PR（标题 = `Landing page design - hacker_eve`）→ 在 Bounty 面板提交申请
  - `hacker2`: Fork Repo → 编辑 → 创建 PR（标题 = `Landing page design - hacker_frank`）→ 提交 Bounty 申请
  - `bot`: 通过 PAT API Fork + 创建 PR + 提交申请（设计上 Bot 通过 API 操作）:
    - API: `POST /api/v1/repos/hackforger/oss-project/forks`
    - API: `POST /api/v1/repos/hackforger/oss-project/pulls`
    - API: `POST /api/v1/repos/hackforger/oss-project/hackforger/bounties/2/applications`
- **验证**:
  - 3 个 PR 创建在 Repo
  - 3 个申请创建，状态均为 `Accepted(1)`（Competitive 模式自动接受所有申请，无 Pending 阶段）
  - Bounty 保持 `Open(0)`（Competitive 模式允许多人并行参与）
- **截图**: `screenshots/full-cycle/j2-13-three-submissions.png`

### Step 2b.5: Organizer 选择获奖者 (Open -> Completed)
- **角色**: `admin` session
- **UI 路径**: 导航到 Bounty 获奖者管理页面 (`/hackforger/oss-project/bounties/{bounty_id}/winners`)
- **操作**: 选择获奖者:
  - 1st place: hacker_eve
  - 2nd place: hacker_frank
  - 3rd place: platform-bot
  - 点击"确认获奖者"（POST `/hackforger/oss-project/bounties/{bounty_id}/winners`）
- **验证**:
  - Bounty 状态: `Open(0)` -> `Completed(3)`
  - Winners 记录已保存
  - Feed 事件: `bounty_winners_selected(38)` — 全局可见
- **截图**: `screenshots/full-cycle/j2-14-winners-selected.png`

### Step 2b.6: Organizer 支付获奖者 (Completed -> Paid)
- **角色**: `admin` session
- **UI 路径**: Bounty 详情页 → 操作区域
- **操作**: 点击"支付"按钮（POST `/hackforger/oss-project/bounties/{bounty_id}/pay`）
- **验证**:
  - Bounty 状态: `Completed(3)` -> `Paid(4)`
  - 托管积分分配给获奖者:
    - hacker_eve: +150 (余额: 800 + 150 = 950)
    - hacker_frank: +100 (余额: 300 + 100 = 400)
    - platform-bot: +50 (余额: 0 + 50 = 50)
  - 交易类型: `escrow_release` for each winner
  - Feed 事件: `bounty_paid(43)`
- **截图**: `screenshots/full-cycle/j2-15-competitive-paid.png`

### 2b 状态快照

| 实体 | 状态 |
|------|------|
| Bounty #2 (Competitive) | `Paid(4)` |
| organizer Credits | 200 (500 - 300 escrowed/released) |
| hacker_eve Credits | 950 (500 + 300 + 150) |
| hacker_frank Credits | 400 (300 + 100) |
| platform-bot Credits | 50 |

---

### 2c: 异常路径 (3 步)

### Step 2c.1: Bounty 过期（截止无提交）
- **角色**: `admin` session (Organizer) + 系统
- **UI 路径**: Repo Issues 页面 → 新建 Issue → Bounty 创建页面
- **操作**:
  - 创建 Issue #3: 标题 = `Refactor API error handling`，提交
  - 注意: organizer 当前余额 = 200，需要先充值:
    - 导航到 `/-/admin/credits` → 充值 100 给 hackforger，原因 = `Bounty #3 budget`
  - 创建 Bounty #3: 关联 Issue #3, 模式 = Exclusive, 设置过去的截止日期（测试用）
  - 添加 Reward: 100 Credits（触发 escrow）
  - 触发过期（截止日期到达，或手动过期操作）
- **验证**:
  - Bounty 状态: `Open(0)` -> `Expired(5)`
  - 托管积分退回 organizer: `escrow_refund` 交易
  - organizer 余额: 200 + 100 (充值) - 100 (escrow) + 100 (退回) = 300
  - Feed 事件: `bounty_expired(52)`
- **截图**: `screenshots/full-cycle/j2-16-bounty-expired.png`

### Step 2c.2: Bounty 取消
- **角色**: `admin` session (Organizer)
- **UI 路径**: Repo Issues → 新建 Issue → Bounty 创建
- **操作**:
  - 创建 Issue #4: 标题 = `Add dark mode support`，提交
  - 先充值 100 给 organizer（`/-/admin/credits`）
  - 创建 Bounty #4: 关联 Issue #4, 模式 = Exclusive
  - 添加 Reward: 100 Credits (escrow)
  - 在 Bounty 详情页点击"取消"按钮（POST `/hackforger/oss-project/bounties/{bounty_id}/cancel`）
- **验证**:
  - Bounty 状态: `Open(0)` -> `Cancelled(6)`
  - 托管积分退回: `escrow_refund`
  - Feed 事件: `bounty_cancelled(53)`
- **截图**: `screenshots/full-cycle/j2-17-bounty-cancelled.png`

### Step 2c.3: 申请被拒绝
- **角色**: `hacker2` session, `admin` session
- **UI 路径**:
  - `hacker2`: 导航到 Bounty #1 详情 → 提交申请（在 2a.5 之前的时间窗口内，此处用于验证拒绝逻辑）
  - `admin`: 导航到申请列表
- **操作**:
  - 注意: 此步骤验证的是 Exclusive Bounty 的申请拒绝逻辑。由于 Bounty #1 已在 2a 中完成，此处可创建新的测试 Bounty 或在 Bounty #1 完成前的时间窗口测试。
  - 模拟: hacker_frank 对一个已被认领的 Exclusive Bounty 提交申请
  - Organizer 在申请列表中找到 hacker_frank 的申请，点击"拒绝"，附评论 = `Bounty already claimed`
- **验证**:
  - 申请状态: `Pending(0)` -> `Rejected(2)`
  - Bounty 状态保持不变（仍为 `Claimed(1)`）
- **截图**: `screenshots/full-cycle/j2-18-application-rejected.png`

### 旅程 2 完整状态快照

| 实体 | 状态 |
|------|------|
| Bounty #1 (Exclusive) | `Paid(4)` |
| Bounty #2 (Competitive) | `Paid(4)` |
| Bounty #3 (Expired) | `Expired(5)` |
| Bounty #4 (Cancelled) | `Cancelled(6)` |
| BountyApplication 状态覆盖 | Pending, Accepted, Rejected |
| organizer Credits | ~400 (含 2c 充值和退回) |
| hacker_eve Credits | 950 |
| hacker_frank Credits | 400 |
| platform-bot Credits | 50 |

---

## 旅程 3: Grant 全生命周期（8 步 + 1 前置）

**角色**: Admin + Organizer + Hacker1 + Hacker2
**参考**: `docs/product/user-journeys/organizer-grant.md`

### 前置条件
- organizer 有 ~400 Credits（J1+J2 剩余），但 Grant 预算由 Admin 额外充值
- hacker_eve 有 950 Credits, hacker_frank 有 400 Credits

### Step 3.0: Admin 为 Organizer 充值 2000 积分 (Grant 预算)
- **角色**: `admin` session
- **UI 路径**: 顶部导航栏 "站点管理" → 左侧菜单 "积分管理" (`/-/admin/credits`)
- **操作**: 充值: 用户名 = `hackforger`, 金额 = `2000`, 原因 = `Q2 Grant Round budget`，点击"充值"
- **验证**: organizer 余额 = ~400 (J1+J2) + 2000 = ~2400
- **截图**: `screenshots/full-cycle/j3-00-credits-deposit.png`

### Step 3.1: Organizer 创建 Grant Round (Draft)
- **角色**: `admin` session (hackforger 即 Organizer)
- **UI 路径**: 导航到 Grant 创建页面 (`/grants/new`)
- **操作**: 填写表单:
  - 名称 = `Q2 2026 Open Source Grant`
  - Slug = `q2-2026-oss-grant`
  - 描述 = `Funding open-source infrastructure projects`
  - 预算 = `2000`
  - 截止日期 = `2026-06-01`
  - 点击"创建"
- **验证**:
  - Grant Round 创建成功，状态 = `Draft(0)`
  - Feed 事件: `grant_round_created(39)` — 全局可见
- **截图**: `screenshots/full-cycle/j3-01-round-created.png`

### Step 3.2: Organizer 开放申请 (Draft -> Open)
- **角色**: `admin` session
- **UI 路径**: Grant Round 管理页面 (`/grants/q2-2026-oss-grant/manage`)
- **操作**: 点击"开放申请"按钮（POST `/grants/q2-2026-oss-grant/manage/open`）
- **验证**:
  - 状态变更: `Draft(0)` -> `Open(1)`
  - 现在接受项目申请
  - Feed 事件: `grant_round_opened(54)` — 全局可见
- **截图**: `screenshots/full-cycle/j3-02-round-opened.png`

### Step 3.3: Hacker1 提交 Grant 项目申请
- **角色**: `hacker1` session (hacker_eve)
- **UI 路径**: 导航到 Grant Round 详情页 (`/grants/q2-2026-oss-grant`) → 点击"提交项目" → 进入提交页面 (`/grants/q2-2026-oss-grant/submit`)
- **操作**: 填写表单:
  - 标题 = `DeFi Testing Framework`
  - 描述 = `Open-source testing framework for DeFi smart contracts`
  - 申请金额 = `1000`
  - Repo URL = `http://localhost:3000/hacker_eve/defi-test-framework`（或相关 Repo 链接）
  - 点击"提交"
- **验证**:
  - Grant Project 创建成功，状态 = `Pending`
  - Feed 事件: `grant_project_submitted(40)` — 全局 + 粉丝可见
- **截图**: `screenshots/full-cycle/j3-03-project1-submitted.png`

### Step 3.4: Hacker2 提交 Grant 项目申请
- **角色**: `hacker2` session (hacker_frank)
- **UI 路径**: 导航到 Grant Round 详情页 → 提交页面 (`/grants/q2-2026-oss-grant/submit`)
- **操作**: 填写表单:
  - 标题 = `NFT Metadata Standard Library`
  - 描述 = `Standardized metadata handling for NFTs across chains`
  - 申请金额 = `1500`
  - Repo URL = `http://localhost:3000/hacker_frank/nft-metadata-lib`
  - 点击"提交"
- **验证**:
  - Grant Project 创建成功，状态 = `Pending`
  - Feed 事件: `grant_project_submitted(40)`
- **截图**: `screenshots/full-cycle/j3-04-project2-submitted.png`

### Step 3.5: Organizer 关闭申请 (Open -> Review)
- **角色**: `admin` session
- **UI 路径**: Grant Round 管理页面 (`/grants/q2-2026-oss-grant/manage`)
- **操作**: 点击"关闭申请"按钮（POST `/grants/q2-2026-oss-grant/manage/close`）
- **验证**:
  - 状态变更: `Open(1)` -> `Review(2)`
  - 不再接受新的项目申请
  - Feed 事件: `grant_round_closed(55)`
- **截图**: `screenshots/full-cycle/j3-05-round-closed.png`

### Step 3.6: Organizer 审批项目（Approve/Reject + Award）
- **角色**: `admin` session
- **UI 路径**: Grant Round 管理页面 → 项目列表 → 点击每个项目进入项目管理页 (`/grants/q2-2026-oss-grant/manage/projects/{pid}`)
- **操作**:
  - 项目 1 (DeFi Testing Framework - hacker_eve):
    - 点击"批准"按钮（POST `.../approve`）
    - 设置 Award 金额 = `1000`（POST `.../award`），附评论 = `Strong proposal, fully funded`
  - 项目 2 (NFT Metadata Standard Library - hacker_frank):
    - 点击"拒绝"按钮（POST `.../reject`），附评论 = `Over remaining budget, reapply next round`
- **验证**:
  - 项目 1: `Pending` -> `Approved`, 拨款 1000 Credits
  - 项目 2: `Pending` -> `Rejected`
  - 总分配 (1000) <= 预算 (2000)
  - Feed 事件: `grant_awarded(41)` for project 1
- **截图**: `screenshots/full-cycle/j3-06-projects-reviewed.png`

### Step 3.7: Organizer Finalize Round (Review -> Finalized)
- **角色**: `admin` session
- **UI 路径**: Grant Round 管理页面 (`/grants/q2-2026-oss-grant/manage`)
- **操作**: 点击"锁定拨款"按钮（POST `/grants/q2-2026-oss-grant/manage/finalize`）
- **验证**:
  - 状态变更: `Review(2)` -> `Finalized(3)`
  - 拨款金额锁定，不可再修改
  - 积分尚未发放（Organizer 需先审查项目进展）
  - Feed 事件: `grant_round_finalized(56)`
- **截图**: `screenshots/full-cycle/j3-07-round-finalized.png`

### Step 3.8: Organizer 审查并发放积分 (Finalized -> Distributed)
- **角色**: `admin` session
- **UI 路径**: Grant Round 管理页面 → 审查已批准的项目进展（访问关联 Repo 查看 commit 历史、PR 等）→ 返回管理页面
- **操作**:
  - 手动审查: 访问 hacker_eve 的关联 Repo，查看 commit、PR、README 更新（平台不强制，Organizer 自行判断）
  - 确认后点击"发放积分"按钮（POST `/grants/q2-2026-oss-grant/manage/distribute`）
- **验证**:
  - 状态变更: `Finalized(3)` -> `Distributed(4)`
  - 积分转账:
    - hacker_eve: +1000 (余额: 950 + 1000 = 1950)
    - organizer: -1000 (余额: ~2400 - 1000 = ~1400)
  - 项目 1 状态: `Approved` -> `Funded`
  - 交易类型: `deposit` for hacker_eve, `withdraw` from organizer
- **截图**: `screenshots/full-cycle/j3-08-distributed.png`

### 旅程 3 状态快照

| 实体 | 状态 |
|------|------|
| Grant Round #1 | `Distributed(4)` |
| Grant Project #1 (hacker_eve) | `Funded` |
| Grant Project #2 (hacker_frank) | `Rejected` |
| hacker_eve Credits | 1950 (500 + 300 + 150 + 1000) |
| hacker_frank Credits | 400 (不变) |
| organizer Credits | ~1400 |

---

## 旅程 4: Credits 流转 + 兑换（9 步）

**角色**: Admin + Hacker1 + Hacker2
**参考**: `docs/product/user-journeys/hacker-credits.md`

### 前置条件
- hacker_eve 有 1950 Credits（J1+J2+J3 累计）
- Admin 已登录

### Step 4.1: Admin 配置兑换选项 (RedeemOption)
- **角色**: `admin` session
- **UI 路径**: 顶部导航栏 "站点管理" → 左侧菜单 "积分管理" → 点击"兑换选项" Tab (`/-/admin/credits/options`)
- **操作**: 点击"创建兑换选项"，填写表单:
  - 名称 = `GitHub Copilot - 1 Month`
  - 描述 = `One month of GitHub Copilot Individual subscription`
  - 价格 = `200` Credits
  - 库存 = `10`
  - 激活 = `true`
  - 点击"创建"
- **验证**: 兑换选项创建成功，出现在列表中，显示价格 200、库存 10
- **截图**: `screenshots/full-cycle/j4-01-redeem-option-created.png`

### Step 4.2: Admin 添加 Key Pool
- **角色**: `admin` session
- **UI 路径**: 兑换选项列表 → 点击 "GitHub Copilot - 1 Month" → 进入 Key 管理页面 (`/-/admin/credits/options/{id}/keys`)
- **操作**: 点击"添加密钥"，输入:
  - Keys = `COPILOT-KEY-001`, `COPILOT-KEY-002`, `COPILOT-KEY-003`（每行一个或批量输入）
  - 点击"添加"
- **验证**: 3 个密钥添加成功，显示在 Key 列表中，状态均为"未使用"
- **截图**: `screenshots/full-cycle/j4-02-keys-added.png`

### Step 4.3: Hacker1 查看积分余额
- **角色**: `hacker1` session (hacker_eve)
- **UI 路径**: 顶部导航栏用户菜单 → 点击"积分" → 进入积分概览页 (`/credits`)
- **操作**: 查看余额显示
- **验证**: 余额 = 1950
- **截图**: `screenshots/full-cycle/j4-03-credits-balance.png`

### Step 4.4: Hacker1 查看交易历史
- **角色**: `hacker1` session
- **UI 路径**: 积分概览页 (`/credits`) → 交易历史区域（或专门的交易列表区域）
- **操作**: 查看所有交易记录
- **验证**: 显示以下交易记录:
  - deposit +500 (Hackathon DeFi Track 第 1 名)
  - escrow_release +300 (Bounty #1 Exclusive 奖励)
  - escrow_release +150 (Bounty #2 Competitive 第 1 名)
  - deposit +1000 (Grant #1 资助)
- **截图**: `screenshots/full-cycle/j4-04-transactions.png`

### Step 4.5: Hacker1 浏览兑换选项
- **角色**: `hacker1` session
- **UI 路径**: 积分概览页 (`/credits`) → 兑换选项区域（显示可用的兑换商品）
- **操作**: 浏览可用的兑换选项列表
- **验证**: 显示 "GitHub Copilot - 1 Month", 价格 200 Credits, 库存 10
- **截图**: `screenshots/full-cycle/j4-05-redeem-options.png`

### Step 4.6: Hacker1 兑换积分
- **角色**: `hacker1` session
- **UI 路径**: 积分概览页 → 点击 "GitHub Copilot - 1 Month" 的"兑换"按钮 → 进入兑换确认页 (`/credits/redeem/{id}`)
- **操作**: 确认兑换信息无误，点击"确认兑换"按钮
- **验证**:
  - RedeemOrder 创建，状态 = `pending`
  - 积分扣除: 1950 - 200 = 1750
  - 交易类型: `redeem`
  - Feed 事件: `credits_redeemed(42)` — 仅自己可见
- **截图**: `screenshots/full-cycle/j4-06-redeem-confirmed.png`

### Step 4.7: Admin Fulfill 订单
- **角色**: `admin` session
- **UI 路径**: 站点管理 → 积分管理 → 订单管理 (`/-/admin/credits/orders`)
- **操作**: 找到 hacker_eve 的待处理订单，点击"发货" (Fulfill) 按钮（POST `/-/admin/credits/orders/{oid}/fulfill`）
- **验证**:
  - 订单状态: `pending` -> `fulfilled`
  - 密钥 "COPILOT-KEY-001" 自动分配并标记为已使用
  - hacker_eve 可在订单详情中查看密钥
  - Feed 事件: `order_fulfilled(58)`
- **截图**: `screenshots/full-cycle/j4-07-order-fulfilled.png`

### Step 4.8: Admin 手动充值 (给 Hacker2)
- **角色**: `admin` session
- **UI 路径**: 站点管理 → 积分管理 (`/-/admin/credits`)
- **操作**: 在"手动充值"区域填写: 用户名 = `hacker_frank`, 金额 = `500`, 原因 = `Community contribution bonus`，点击"充值"
- **验证**:
  - hacker_frank 余额: 400 + 500 = 900
  - 交易类型: `admin_deposit`
- **截图**: `screenshots/full-cycle/j4-08-admin-deposit.png`

### Step 4.9: Admin 手动扣除
- **角色**: `admin` session
- **UI 路径**: 站点管理 → 积分管理 (`/-/admin/credits`) → "手动扣除"区域
- **操作**: 填写: 用户名 = `hacker_frank`, 金额 = `50`, 原因 = `Duplicate reward correction`，点击"扣除"
- **验证**:
  - hacker_frank 余额: 900 - 50 = 850
  - 交易类型: `admin_deduct`
- **截图**: `screenshots/full-cycle/j4-09-admin-deduct.png`

### 旅程 4 状态快照

| 实体 | 状态 |
|------|------|
| hacker_eve Credits | 1750 (1950 - 200 兑换) |
| hacker_frank Credits | 850 (400 + 500 - 50) |
| RedeemOrder #1 | `fulfilled` |
| Key COPILOT-KEY-001 | 已使用 |
| 交易类型覆盖 | deposit, withdraw, redeem, admin_deposit, admin_deduct, escrow, escrow_release, escrow_refund |

---

## 旅程 5: Feed + 社交 + 发现 + 搜索（10 步）

**角色**: Hacker1 + Hacker2 + 匿名
**参考**: `docs/product/user-journeys/hacker-social.md`

### 前置条件
- 旅程 1-4 全部完成
- 多个 Feed 事件已生成
- hacker_eve 关注 organizer (hackforger) 和 hacker_frank

### Step 5.1: Hacker1 查看 Dashboard Feed (Following Tab)
- **角色**: `hacker1` session (hacker_eve)
- **UI 路径**: 导航到首页 Dashboard (`/`) → 切换到"社区动态" / "Following" Tab
- **操作**: 查看 Feed 列表
- **验证**: Feed 包含已关注用户 + 全局事件（按时间倒序、分页）:
  - `hackathon_created(30)` — organizer 创建了 Web3 Innovation
  - `hackathon_phase_changed(50)` — organizer 发布/启动/评审
  - `hackathon_submitted(32)` — hacker_frank 提交（hacker_eve 关注了 hacker_frank）
  - `bounty_created(34)` — 全局事件
  - `grant_round_opened(54)` — 全局事件
- **截图**: `screenshots/full-cycle/j5-01-dashboard-feed.png`

### Step 5.2: Hacker1 查看 Global Feed
- **角色**: `hacker1` session
- **UI 路径**: Dashboard → 切换到"全局动态" / "Global" Tab
- **操作**: 查看全局 Feed
- **验证**: 显示所有 HackForger 事件（不限关注关系）
  - 包含旅程 1-4 中生成的所有事件类型
  - 按 `created_unix` DESC 排序
- **截图**: `screenshots/full-cycle/j5-02-global-feed.png`

### Step 5.3: Explore Hackathons
- **角色**: `hacker1` session（或匿名）
- **UI 路径**: 顶部导航栏 "Explore" → 点击 "Hackathons" Tab (`/explore/hackathons`)
- **操作**: 查看 Hackathon 列表
- **验证**:
  - 列出 "Web3 Innovation Challenge"，状态 = `Finished`
  - 显示参与人数、赛道数、奖金池
- **截图**: `screenshots/full-cycle/j5-03-explore-hackathons.png`

### Step 5.4: Explore Bounties
- **角色**: `hacker1` session
- **UI 路径**: Explore 页面 → 点击 "Bounties" Tab (`/explore/bounties`)
- **操作**: 查看 Bounty 列表，尝试筛选（按状态、模式）
- **验证**:
  - 列出所有 Bounty（跨 Repo）:
    - Bounty #1 (Paid), #2 (Paid), #3 (Expired), #4 (Cancelled)
  - 筛选功能可用
- **截图**: `screenshots/full-cycle/j5-04-explore-bounties.png`

### Step 5.5: Explore Grant Rounds
- **角色**: `hacker1` session
- **UI 路径**: Explore 页面 → 点击 "Grants" Tab (`/explore/grants`)
- **操作**: 查看 Grant Round 列表
- **验证**:
  - 列出 "Q2 2026 Open Source Grant"，状态 = `Distributed`
  - 显示预算、已资助项目数
- **截图**: `screenshots/full-cycle/j5-05-explore-grants.png`

### Step 5.6: 全局搜索（跨实体）
- **角色**: `hacker1` session
- **UI 路径**: 在任意页面按 Cmd+K (macOS) / Ctrl+K 打开搜索弹窗
- **操作**: 输入关键词 `DeFi`，查看搜索结果
- **验证**: 结果按类型分组显示:
  - Hackathon Track "DeFi Track"
  - Hackathon Submission "DeFi Lending Protocol"
  - Grant Project "DeFi Testing Framework"
  - 每组有"查看全部"链接
- **截图**: `screenshots/full-cycle/j5-06-search-defi.png`

### Step 5.7: 搜索 scope 筛选
- **角色**: `hacker1` session
- **UI 路径**: 搜索弹窗（或 Explore 搜索页 `/explore/hackforger/search`）
- **操作**:
  - 搜索 `DeFi` + scope = `bounties` → 验证无结果（没有 Bounty 包含 "DeFi"）
  - 搜索 `OAuth` + scope = `bounties` → 验证返回 Bounty #1 "Implement OAuth2 PKCE flow"
- **验证**: Scope 筛选功能正常，按类型过滤搜索结果
- **截图**: `screenshots/full-cycle/j5-07-search-scope.png`

### Step 5.8: 声誉排行榜
- **角色**: 任意 session
- **UI 路径**: Explore 页面 → 点击 "Reputation" Tab (`/explore/reputation`)
- **操作**: 查看声誉排行榜
- **验证**:
  - hacker_eve 排名最高（Hackathon 第 1 名 + Bounty 胜出 + Grant 获资助）
  - hacker_frank 排名第二（Hackathon 第 2 名 + Bounty 第 2 名）
  - platform-bot 排名第三（仅 Bounty 第 3 名）
  - 分数反映各模块的加权贡献
- **截图**: `screenshots/full-cycle/j5-08-reputation-leaderboard.png`

### Step 5.9: 查看个人声誉
- **角色**: `hacker1` session
- **UI 路径**: 声誉排行榜 → 点击 hacker_eve 的条目 → 进入个人声誉详情页（或访问 hacker_eve 的个人主页的声誉区域）
- **操作**: 查看声誉分数细分
- **验证**: 显示总分、等级 (tier)、以及各模块的分数构成:
  - Hackathon 贡献分
  - Bounty 贡献分
  - Grant 贡献分
- **截图**: `screenshots/full-cycle/j5-09-individual-reputation.png`

### Step 5.10: 在 Bounty Issue 上添加 Reaction (社交)
- **角色**: `hacker2` session (hacker_frank)
- **UI 路径**: 导航到 Bounty #1 关联的 Issue (`/hackforger/oss-project/issues/1`) → 找到相关评论
- **操作**: 在 hacker_eve 的 Bounty 完成评论上点击 Reaction 按钮 → 选择 "+1" / 点赞
- **验证**: Reaction 成功添加，显示在评论下方（Forgejo 原生功能，无需 HackForger 特殊处理）
- **截图**: `screenshots/full-cycle/j5-10-reaction.png`

### 旅程 5 状态快照

| 功能 | 验证状态 |
|------|---------|
| Following Feed | 显示已关注用户的事件 |
| Global Feed | 显示所有事件 |
| Explore: Hackathons | 可筛选列表 |
| Explore: Bounties | 跨 Repo 列表 |
| Explore: Grants | Round 列表 |
| 搜索 | 全文跨实体 + Scope 筛选 |
| 声誉 | 排行榜 + 个人详情 |
| 社交: Follow/Star/Watch/Reaction | 贯穿 J1-J5 |

---

## 跨旅程验证

以下检查点验证各旅程之间的数据一致性。

### 积分一致性验证
- **角色**: `hacker1` session + `hacker2` session + `admin` session
- **UI 路径**: 每个用户各自导航到 `/credits` 查看余额
- **操作**: 对账每个用户的最终余额与交易历史

| 用户 | 预期余额 | 构成 |
|------|---------|------|
| hacker_eve | 1750 | +500 (Hackathon 1st) +300 (Bounty #1) +150 (Bounty #2 1st) +1000 (Grant) -200 (兑换) |
| hacker_frank | 850 | +300 (Hackathon 2nd) +100 (Bounty #2 2nd) +500 (Admin 充值) -50 (Admin 扣除) |
| platform-bot | 50 | +50 (Bounty #2 3rd) |
| hackforger (organizer) | ~1400 | 1000 + 600 + 100 + 100 + 2000 (充值) -800 (Hackathon) -300 (Bounty #1) -300 (Bounty #2) -1000 (Grant) +100+100 (退回) |

- **验证**: 每个用户的余额与预期一致，交易记录完整无遗漏
- **截图**: `screenshots/full-cycle/cross-01-credits-reconcile.png`

### Feed 事件完整性验证
- **角色**: `admin` session
- **UI 路径**: Dashboard → Global Feed
- **操作**: 验证 Feed 中包含旅程 1-4 的所有关键事件类型

| 事件类型 | 来源旅程 | 预期数量 |
|---------|---------|---------|
| `hackathon_created(30)` | J1 | 1 |
| `hackathon_registered(31)` | J1 | 2 |
| `hackathon_submitted(32)` | J1 | 2 |
| `hackathon_scored(33)` | J1 | 4 |
| `hackathon_phase_changed(50)` | J1 | 3 (publish, start, judge) |
| `hackathon_finalized(51)` | J1 | 1 |
| `bounty_created(34)` | J2 | 4 (#1, #2, #3, #4) |
| `bounty_claimed(35)` | J2a | 1 |
| `bounty_delivered(36)` | J2a | 1 |
| `bounty_completed(37)` | J2a | 1 |
| `bounty_paid(43)` | J2a, J2b | 2 |
| `bounty_winners_selected(38)` | J2b | 1 |
| `bounty_expired(52)` | J2c | 1 |
| `bounty_cancelled(53)` | J2c | 1 |
| `grant_round_created(39)` | J3 | 1 |
| `grant_round_opened(54)` | J3 | 1 |
| `grant_project_submitted(40)` | J3 | 2 |
| `grant_round_closed(55)` | J3 | 1 |
| `grant_awarded(41)` | J3 | 1 |
| `grant_round_finalized(56)` | J3 | 1 |
| `credits_redeemed(42)` | J4 | 1 |
| `order_fulfilled(58)` | J4 | 1 |

- **验证**: 以上事件均出现在 Global Feed 中
- **截图**: `screenshots/full-cycle/cross-02-feed-events.png`

### Webhook 投递验证
- **角色**: `admin` session
- **前置**: 在测试开始前（或旅程 1 开始前），在某个 Repo 的 Settings → Webhooks 中配置一个 Webhook，勾选 HackForger 相关事件
- **UI 路径**: Repo Settings → Webhooks → 点击已配置的 Webhook → 查看 "Recent Deliveries"
- **操作**: 检查最近投递记录
- **验证**: 已勾选的 HackForger 事件（如 bounty_created, hackathon_finalized 等）在 Recent Deliveries 中有对应记录，且 HTTP 状态码为 2xx
- **截图**: `screenshots/full-cycle/cross-03-webhook-deliveries.png`

---

## 报告模版

测试完成后，按以下格式生成报告，保存到 `docs/tests/e2e/reports/user-journey-full-cycle-report.md`:

```markdown
# 用户旅程全流程测试报告

> **日期**: YYYY-MM-DD
> **服务器**: http://localhost:3000
> **测试者**: (执行者名称)
> **代码版本**: (git commit hash)

## 概要

| 旅程 | 角色 | 步骤数 | 通过 | 失败 | 跳过 | 状态 |
|------|------|--------|------|------|------|------|
| 1. Hackathon | Organizer + Hackers + Judges | 20 | ? | ? | ? | ? |
| 2. Bounty (2a+2b+2c) | Organizer + Hackers + Bot | 19 | ? | ? | ? | ? |
| 3. Grant | Organizer + Hackers | 9 | ? | ? | ? | ? |
| 4. Credits | Admin + Hackers | 9 | ? | ? | ? | ? |
| 5. Social | Hackers | 10 | ? | ? | ? | ? |
| 跨旅程验证 | All | 3 | ? | ? | ? | ? |
| **总计** | | **70** | | | | |

## 旅程 1: Hackathon

### Step 1.1: Admin 为 Organizer 充值 1000 积分 -- [PASS/FAIL]
(操作描述 + 截图引用)

...(每步重复)

## 旅程 2: Bounty

...(同上)

## 旅程 3: Grant

...(同上)

## 旅程 4: Credits

...(同上)

## 旅程 5: Feed + 社交 + 发现 + 搜索

...(同上)

## 跨旅程验证

...(同上)

## 发现的 Bug

| # | 旅程 | 步骤 | 描述 | 严重程度 | 修复状态 |
|---|------|------|------|---------|---------|

## 结论

(总结通过率、关键发现、需要修复的问题)
```
