# 用户旅程全流程测试模版

> **Web 端测试是必选项，不可使用单元测试或 API 测试替代 Web 界面操作。关键验证节点必须使用 agent-browser 截屏记录作为测试证据。**
>
> API 调用仅用于辅助数据准备（如密码重置、积分充值），核心业务流程必须通过浏览器完成。

## 测试环境

- **服务器**: http://localhost:3000（worktree 构建的最新 binary）
- **数据库**: 每轮测试前清空 HackForger 数据（见 e2e-testing-guide.md 的 Database Cleanup）
- **截图保存**: `docs/tests/screenshots/journeys/full-cycle/`

## 测试账号与角色映射

| PRD 角色 | 实例用户名 | 密码 | 本次旅程中的职责 |
|---------|-----------|------|----------------|
| Admin | `hackforger` | `admin1234` | 平台管理、积分充值、兑换选项配置 |
| Organizer | `hackforger` | `admin1234` | 创建 Hackathon/Bounty/Grant、管理评审（v0.1 与 Admin 同账号） |
| Judge 1 | `judge_carol` | `admin1234` | Hackathon 评委 |
| Judge 2 | `judge_dave` | `admin1234` | Hackathon 评委 |
| Hacker 1 | `hacker_eve` | `admin1234` | 活跃参赛者、Bounty 认领者 |
| Hacker 2 | `hacker_frank` | `admin1234` | 第二参赛者、组队伙伴 |

> **密码重置**: 测试开始前，对所有非 admin 账号执行密码重置（见 e2e-testing-guide.md）。

## 多用户 Session

使用 `agent-browser --session <name>` 为每个角色创建独立浏览器会话：

```bash
# 登录所有用户（一次性完成）
for user in hackforger judge_carol judge_dave hacker_eve hacker_frank; do
  agent-browser --session $user open "http://localhost:3000/user/login"
  agent-browser --session $user snapshot -i
  agent-browser --session $user fill @e13 "$user"
  agent-browser --session $user fill @e14 "admin1234"
  agent-browser --session $user click @e17
  agent-browser --session $user wait --load networkidle
done
```

---

## 旅程 1: Hackathon 全生命周期

**角色**: Organizer + Hacker1 + Hacker2 + Judge1 + Judge2
**参考**: `docs/product/user-journeys/organizer-hackathon.md`（20 步）

### 前置条件
- 数据库已清空 HackForger 数据
- 所有用户已登录各自 session

### Step 1.1: Admin 为 Organizer 充值积分
- **角色**: hackforger (Admin)
- **操作**: 通过 API 为 organizer 充值 1000 积分（辅助操作，允许用 API）
- **验证**: 积分余额 = 1000
- **截图**: `j1-01-credits-deposit.png` — 积分管理页面显示余额

### Step 1.2: Organizer 创建 Hackathon
- **角色**: hackforger (Organizer)
- **操作**: 导航到 `/hackathons/new`，填写表单（名称、描述、团队上限、日期），提交
- **验证**: 跳转到详情页，状态 = Draft
- **截图**: `j1-02-hackathon-created.png` — 创建成功的详情页

### Step 1.3: Organizer 创建赛道（Track）
- **角色**: hackforger
- **操作**: 在管理页面添加 2 个赛道（Web Track、AI Track），设置奖金分配
- **验证**: 赛道列表显示 2 个条目，自动创建对应 Repository
- **截图**: `j1-03-tracks-created.png` — 赛道列表

### Step 1.4: Organizer 设置评审标准
- **角色**: hackforger
- **操作**: 为每个赛道添加评审标准（创新性、技术质量、展示效果）
- **验证**: 标准列表显示权重总和 = 100
- **截图**: `j1-04-criteria-set.png` — 评审标准页面

### Step 1.5: Organizer 指派评委
- **角色**: hackforger
- **操作**: 添加 judge_carol 和 judge_dave 为评委
- **验证**: 评委列表显示 2 人
- **截图**: `j1-05-judges-assigned.png` — 评委列表

### Step 1.6: Organizer 发布 Hackathon (Draft → Open)
- **角色**: hackforger
- **操作**: 点击"发布"按钮
- **验证**: 状态变为 Open，在 Explore 页面可见
- **截图**: `j1-06-published.png` — Explore 页面显示新 Hackathon

### Step 1.7: Hacker1 关注 Organizer（社交互动）
- **角色**: hacker_eve
- **操作**: 导航到 organizer 的个人页面，点击 Follow
- **验证**: Follow 按钮变为 Unfollow
- **截图**: `j1-07-follow-organizer.png` — 关注后的页面

### Step 1.8: Hacker1 报名参赛
- **角色**: hacker_eve
- **操作**: 导航到 Hackathon 详情页，选择 Web Track，提交报名
- **验证**: 报名成功，显示报名状态
- **截图**: `j1-08-hacker1-registered.png` — 报名确认页面

### Step 1.9: Hacker2 报名并组队（社交互动）
- **角色**: hacker_frank
- **操作**: 报名 AI Track，填写团队名称
- **验证**: 报名成功
- **截图**: `j1-09-hacker2-registered.png` — 报名确认

### Step 1.10: Organizer 启动 Hacking (Open → Hacking)
- **角色**: hackforger
- **操作**: 在管理页面点击"开始 Hacking"
- **验证**: 状态变为 Hacking
- **截图**: `j1-10-hacking-started.png` — 管理页面状态

### Step 1.11: Hacker1 Fork 赛道 Repo 并开发
- **角色**: hacker_eve
- **操作**: 导航到 Web Track Repo → Fork → 在 Fork 中编辑文件
- **Git 操作**: Fork + 编辑 + Commit
- **截图**: `j1-11-hacker1-fork.png` — Fork 后的仓库页面

### Step 1.12: Hacker1 提交 PR 作为参赛作品
- **角色**: hacker_eve
- **操作**: 从 Fork 创建 Pull Request 到赛道 Repo
- **Git 操作**: Create PR
- **验证**: PR 创建成功，Submission 自动关联
- **截图**: `j1-12-hacker1-pr.png` — PR 页面

### Step 1.13: Hacker2 提交参赛作品
- **角色**: hacker_frank
- **操作**: 同 Hacker1，Fork AI Track → 开发 → PR
- **截图**: `j1-13-hacker2-pr.png` — PR 页面

### Step 1.14: 其他用户 Star 赛道 Repo（社交信号）
- **角色**: hacker_eve
- **操作**: Star 另一个赛道的 Repo
- **截图**: `j1-14-star-repo.png` — Star 后的 Repo 页面

### Step 1.15: Organizer 启动评审 (Hacking → Judging)
- **角色**: hackforger
- **操作**: 在管理页面点击"开始评审"（`/judge` endpoint）
- **验证**: 状态变为 Judging
- **截图**: `j1-15-judging-started.png` — 管理页面

### Step 1.16: Judge1 评分
- **角色**: judge_carol
- **操作**: 导航到评审页面，为每个 Submission 按 Criteria 打分
- **验证**: 分数保存成功
- **截图**: `j1-16-judge1-scores.png` — 评分界面

### Step 1.17: Judge2 评分
- **角色**: judge_dave
- **操作**: 同 Judge1，独立评分
- **截图**: `j1-17-judge2-scores.png` — 评分界面

### Step 1.18: Organizer Finalize (Judging → Finished)
- **角色**: hackforger
- **操作**: 查看排名预览 → 确认 Finalize
- **验证**: 状态变为 Finished，排名确定，积分自动发放
- **截图**: `j1-18-finalized.png` — Finalize 结果页面

### Step 1.19: 验证排行榜
- **角色**: 任意
- **操作**: 查看 Hackathon 排行榜
- **验证**: 排名与评分一致
- **截图**: `j1-19-leaderboard.png` — 排行榜页面

### Step 1.20: 验证积分发放
- **角色**: hacker_eve
- **操作**: 导航到 `/credits`，查看余额变化
- **验证**: 余额增加了对应奖金
- **截图**: `j1-20-credits-received.png` — 积分页面显示新余额

---

## 旅程 2: Bounty 全生命周期

**角色**: Organizer + Hacker1 + Hacker2
**参考**: `docs/product/user-journeys/organizer-bounty.md`（19 步）

### 2a: Exclusive Bounty（独占模式）

### Step 2a.1: Organizer 创建 Issue
- **角色**: hackforger
- **操作**: 在某个 Repo 中创建 Issue，描述需要修复的问题
- **截图**: `j2-01-issue-created.png` — Issue 页面

### Step 2a.2: Organizer 在 Issue 上创建 Bounty
- **角色**: hackforger
- **操作**: 在 Issue 页面或通过 Bounty 创建表单，挂载 Exclusive Bounty + 设置 Reward
- **验证**: Bounty 状态 = Open，Issue 上显示 Bounty 徽章
- **截图**: `j2-02-bounty-created.png` — 带 Bounty 徽章的 Issue

### Step 2a.3: Hacker1 申请 Bounty
- **角色**: hacker_eve
- **操作**: 浏览 Bounty 列表（Explore 或 Issue 页面），提交申请
- **验证**: 申请状态 = Pending
- **截图**: `j2-03-application-submitted.png` — 申请确认

### Step 2a.4: Organizer 接受申请 (Open → Claimed)
- **角色**: hackforger
- **操作**: 审核申请列表，Accept hacker_eve 的申请
- **验证**: Bounty 状态变为 Claimed
- **截图**: `j2-04-application-accepted.png` — 状态变更

### Step 2a.5: Hacker1 Fork + 开发 + 提交 PR
- **角色**: hacker_eve
- **操作**: Fork Repo → 修复代码 → 提交 PR
- **Git 操作**: Fork + PR
- **截图**: `j2-05-delivery-pr.png` — PR 页面

### Step 2a.6: Organizer Review + Complete (InReview → Completed)
- **角色**: hackforger
- **操作**: Review PR → 标记 Bounty 完成
- **验证**: 状态变为 Completed，积分自动发放
- **截图**: `j2-06-bounty-completed.png` — 完成状态

### Step 2a.7: Organizer 标记支付 (Completed → Paid)
- **角色**: hackforger
- **操作**: 标记为已支付
- **验证**: 状态变为 Paid
- **截图**: `j2-07-bounty-paid.png` — 支付状态

### 2b: Competitive Bounty（竞赛模式）

### Step 2b.1: Organizer 创建 Competitive Bounty
- **角色**: hackforger
- **操作**: 创建 mode=Competitive 的 Bounty，设置 3 级奖励（1st/2nd/3rd）
- **截图**: `j2-08-competitive-bounty.png` — 竞赛 Bounty 详情

### Step 2b.2: 多人申请
- **角色**: hacker_eve + hacker_frank
- **操作**: 两人分别提交申请
- **截图**: `j2-09-multiple-applicants.png` — 申请列表

### Step 2b.3: Organizer 选择获奖者
- **角色**: hackforger
- **操作**: 选择 Winner（1st: hacker_eve, 2nd: hacker_frank）
- **验证**: Winners 列表确认，积分按级别发放
- **截图**: `j2-10-winners-selected.png` — 获奖者列表

### 2c: 异常路径

### Step 2c.1: Bounty 取消
- **角色**: hackforger
- **操作**: 创建一个 Bounty → 直接取消
- **验证**: 状态变为 Cancelled
- **截图**: `j2-11-bounty-cancelled.png` — 取消状态

---

## 旅程 3: Grant 全生命周期

**角色**: Organizer + Hacker1 + Hacker2
**参考**: `docs/product/user-journeys/organizer-grant.md`（8 步）

### Step 3.1: Organizer 创建 Grant Round
- **角色**: hackforger
- **操作**: 导航到 `/grants/new`，填写名称、预算、截止日期
- **验证**: Round 状态 = Draft
- **截图**: `j3-01-round-created.png` — Round 详情页

### Step 3.2: Organizer 开放申请 (Draft → Open)
- **角色**: hackforger
- **操作**: 点击"开放申请"
- **截图**: `j3-02-round-opened.png` — Open 状态

### Step 3.3: Hacker1 提交 Grant 项目
- **角色**: hacker_eve
- **操作**: 在 Round 页面提交项目申请（标题、描述、Repo 链接）
- **截图**: `j3-03-project-submitted.png` — 提交确认

### Step 3.4: Hacker2 提交另一个项目
- **角色**: hacker_frank
- **操作**: 提交不同的项目申请
- **截图**: `j3-04-project2-submitted.png` — 第二个项目

### Step 3.5: Organizer 审批项目
- **角色**: hackforger
- **操作**: Approve hacker_eve 的项目，Reject hacker_frank 的项目
- **截图**: `j3-05-projects-reviewed.png` — 审批后的项目列表

### Step 3.6: Organizer 分配金额
- **角色**: hackforger
- **操作**: 为 Approved 项目设置 Award amount
- **验证**: 总分配 ≤ 预算
- **截图**: `j3-06-awards-allocated.png` — 金额分配

### Step 3.7: Organizer Finalize + Distribute
- **角色**: hackforger
- **操作**: Finalize Round → Distribute
- **验证**: 积分自动发放给获批项目
- **截图**: `j3-07-distributed.png` — Distributed 状态

### Step 3.8: 验证 Hacker1 积分增加
- **角色**: hacker_eve
- **操作**: 查看积分余额
- **截图**: `j3-08-credits-from-grant.png` — 积分变化

---

## 旅程 4: Credits 流转 + 兑换

**角色**: Admin + Hacker1
**参考**: `docs/product/user-journeys/hacker-credits.md`（8 步）

### Step 4.1: Admin 配置兑换选项
- **角色**: hackforger (Admin)
- **操作**: 在管理后台创建兑换选项（如"云计算额度 $10"），设置价格、库存
- **截图**: `j4-01-redeem-option-created.png` — 兑换选项管理页

### Step 4.2: Admin 添加 Key Pool
- **角色**: hackforger
- **操作**: 为兑换选项添加兑换密钥（Key）
- **截图**: `j4-02-keys-added.png` — Key 列表

### Step 4.3: Hacker1 查看积分余额
- **角色**: hacker_eve
- **操作**: 导航到 `/credits`
- **验证**: 显示余额（来自 Hackathon + Bounty + Grant 奖励累计）
- **截图**: `j4-03-credits-overview.png` — 积分概览

### Step 4.4: Hacker1 查看交易历史
- **角色**: hacker_eve
- **操作**: 查看积分流水
- **验证**: 显示各次 deposit 记录
- **截图**: `j4-04-transactions.png` — 交易历史

### Step 4.5: Hacker1 浏览兑换选项并下单
- **角色**: hacker_eve
- **操作**: 浏览兑换选项列表 → 选择一个 → 确认兑换
- **验证**: 订单创建，余额扣减
- **截图**: `j4-05-redeem-confirmed.png` — 兑换确认页

### Step 4.6: Admin Fulfill 订单
- **角色**: hackforger (Admin)
- **操作**: 在订单管理页面标记 Fulfill
- **验证**: 订单状态变为 Fulfilled，Key 发放
- **截图**: `j4-06-order-fulfilled.png` — 订单完成

### Step 4.7: Hacker1 查看兑换结果
- **角色**: hacker_eve
- **操作**: 查看订单详情，确认收到 Key
- **截图**: `j4-07-key-delivered.png` — 兑换结果

### Step 4.8: Admin 手动充值/扣除
- **角色**: hackforger
- **操作**: 手动 Deposit + Deduct 操作
- **截图**: `j4-08-admin-ops.png` — 管理操作

---

## 旅程 5: 社交 + Feed + 发现 + 搜索

**角色**: Hacker1 + Hacker2
**参考**: `docs/product/user-journeys/hacker-social.md`（7 步）

### Step 5.1: Hacker1 查看 Dashboard Feed
- **角色**: hacker_eve
- **操作**: 登录后查看首页 Dashboard → 切换到"社区动态" Tab
- **验证**: Feed 显示所关注用户（organizer）的最近活动
- **截图**: `j5-01-dashboard-feed.png` — 社区动态

### Step 5.2: Hacker1 和 Hacker2 互相 Follow
- **角色**: hacker_eve → 关注 hacker_frank，hacker_frank → 关注 hacker_eve
- **操作**: 导航到对方个人页面 → 点击 Follow
- **截图**: `j5-02-mutual-follow.png` — 互相关注后的页面

### Step 5.3: 通过 Explore 发现内容
- **角色**: hacker_eve
- **操作**: 依次访问 Explore 的各个 Tab（Repos → Users → Hackathons → Bounties → Grants → Submissions）
- **验证**: 每个 Tab 显示正确的内容列表
- **截图（每 Tab 一张）**:
  - `j5-03a-explore-repos.png`
  - `j5-03b-explore-users.png`
  - `j5-03c-explore-hackathons.png`
  - `j5-03d-explore-bounties.png`
  - `j5-03e-explore-grants.png`
  - `j5-03f-explore-submissions.png`

### Step 5.4: ⌘K 全局搜索
- **角色**: hacker_eve
- **操作**: 按 ⌘K 打开搜索弹窗 → 输入关键词 → 查看分组结果
- **验证**: 结果按类型分组（Hackathons / Bounties / Repos / Users 等），每组有"查看全部"链接
- **截图**: `j5-04-cmdk-search.png` — 搜索结果弹窗

### Step 5.5: 从搜索结果导航
- **角色**: hacker_eve
- **操作**: 点击搜索结果中的某一项 → 导航到详情页
- **截图**: `j5-05-navigate-from-search.png` — 目标详情页

### Step 5.6: 声誉排行榜
- **角色**: hacker_eve
- **操作**: 导航到声誉排行榜页面
- **验证**: 显示用户列表，按分数排序
- **截图**: `j5-06-reputation-leaderboard.png` — 排行榜

### Step 5.7: AI 助手（占位）
- **角色**: hacker_eve
- **操作**: 在 ⌘K 搜索弹窗中查看 AI 助手面板
- **验证**: 显示"AI 助手功能即将上线"占位消息 + disclaimer
- **截图**: `j5-07-ai-assistant.png` — AI 助手面板

---

## 跨旅程验证

以下检查点验证各旅程之间的数据一致性：

### 积分一致性
- Hacker1 (hacker_eve) 的积分余额 = Hackathon 奖金 + Bounty 奖金 + Grant Award - 兑换扣除
- **截图**: `cross-01-credits-reconcile.png` — 最终积分余额与交易历史对账

### Feed 事件完整性
- Dashboard Feed 包含旅程 1-4 中的关键事件（hackathon_created, bounty_completed, grant_awarded, credits_redeemed）
- **截图**: `cross-02-feed-events.png` — Feed 中的事件列表

### Webhook 投递（如已配置）
- 如果在旅程前配置了 Webhook，验证 Recent Deliveries 中有对应事件
- **截图**: `cross-03-webhook-deliveries.png` — Webhook 投递记录

---

## 报告模版

测试完成后，按以下格式生成报告，保存到 `docs/tests/e2e/user-journey-full-cycle-report.md`：

```markdown
# 用户旅程全流程测试报告

> **日期**: YYYY-MM-DD
> **服务器**: http://localhost:3000
> **测试者**: (执行者名称)

## 概要

| 旅程 | 角色 | 步骤数 | 通过 | 失败 | 状态 |
|------|------|--------|------|------|------|
| 1. Hackathon | Organizer + Hackers + Judges | 20 | ? | ? | ? |
| 2. Bounty | Organizer + Hackers | 11 | ? | ? | ? |
| 3. Grant | Organizer + Hackers | 8 | ? | ? | ? |
| 4. Credits | Admin + Hacker | 8 | ? | ? | ? |
| 5. Social | Hackers | 7 | ? | ? | ? |
| 跨旅程验证 | All | 3 | ? | ? | ? |
| **总计** | | **57** | | | |

## 旅程 1: Hackathon

### Step 1.1: Admin 为 Organizer 充值积分 — [PASS/FAIL]
(操作描述 + 截图)

...（每步重复）

## 发现的 Bug

| # | 旅程 | 步骤 | 描述 | 严重程度 | 修复状态 |
|---|------|------|------|---------|---------|

## 结论

(总结通过率、关键发现、需要修复的问题)
```

将报告渲染为 PDF 保存到同目录。
