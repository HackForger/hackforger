# Phase 1 Bounty — E2E Retest Report

> **Tester:** Claude (automated E2E)
> **Date:** 2026-03-28
> **Instance:** https://hackforger.inside.h2os.cloud
> **Branch/Commit:** Forgejo 14.0.3-49-2f8d37a788+gitea-1.22.0

---

## 1. Explore 页面

- [x] `/explore/bounties` 加载正常，显示 3 个 bounty
- [x] 标题可点击，链接到对应 Issue（显示 `acme-dev/backend #N`）
- [x] 切换简体中文后 navbar tabs 显示 "黑客松 / 悬赏 / 资助"
- [x] 状态 filter tabs 显示：全部 / 开放 / 已认领 / 已完成
- [x] Filter 切换正常更新列表（"已认领" 显示空状态 "暂无悬赏。从仓库 Issue 创建一个吧！"）
- [x] 每个 bounty 卡片显示状态标签（开放）和模式标签（独占/竞赛）

**Notes:** Explore 页面所有检查项通过。i18n 中文翻译准确。

---

## 2. Issue 面板

- [x] Issue #5（Bounty 4）sidebar 显示 Bounty panel：状态 "开放"（绿色标签）
- [x] 奖励正确显示：`USD 200 | 500 credits` — credits 金额修复确认
- [x] Publisher（hackforger）看到 "Cancel Bounty" 操作按钮
- [x] 无 bounty 的 Issue #6 显示 "创建悬赏" 按钮（绿色）
- [ ] **Bounty panel 显示 "token is required" 错误** — 见下方 BUG-1

**Notes:**
- **BUG-1 (P1):** Bounty panel Vue 组件显示 "token is required" 红色错误消息。经 DOM 分析，`#hackforger-bounty-panel` 元素的 `data-` attributes 中**未传递 CSRF token**。组件需要 token 来发起 API 请求（apply/cancel），但服务端模板未注入 `data-csrf-token` 属性。该错误为 SSR（服务端渲染）在模板中嵌入的，不是前端 XHR 返回。

---

## 3. Exclusive Bounty 流程（Bounty 4 → Issue #5）

### 申请认领
- [x] hacker_eve 申请返回 201（Application ID=7, UserID=3）
- [x] agent_hunter 申请返回 201（Application ID=8, UserID=6）

### 接受申请
- [x] `PUT /applications/7` + `{"action":"accept"}` 返回 200
- [x] Bounty status = 1 (Claimed) ✓
- [x] ClaimerID = 3 (hacker_eve) ✓

### PR 创建 & 合并
- [x] eve 创建分支 `eve-jwt-refactor` 并提交代码
- [x] PR #7 "JWT auth refactor - fixes #5" 创建成功
- [x] PR #7 合并成功
- [ ] **Bounty 状态未从 Claimed → InReview** — 见 BUG-2

### Complete / Paid
- [ ] `POST /bounties/4/complete` 返回 422："invalid bounty status [bounty_id: 4, current: 1, expected: InReview]"
- [ ] 无法手动触发 Exclusive bounty 的 InReview 转换（`/review` 仅限 Competitive mode）
- [ ] Complete 和 Paid 流程被阻塞

**Notes:**
- **BUG-2 (P0 Blocker):** PR merge hook 未触发 Exclusive bounty 状态转换。合并关联 Issue #5 的 PR 后，Bounty 4 状态仍为 1 (Claimed)，未转为 2 (InReview)。无手动 API 可绕过，导致后续 complete/paid 流程完全阻塞。
- **API 发现:** Sudo header 可用于以其他用户身份执行 API 操作（需 admin token）。

---

## 4. Competitive Bounty 流程（Bounty 5 → Issue #2）

### 多人提交
- [x] hacker_eve 申请返回 201（ID=9）
- [x] hacker_frank 申请返回 201（ID=10）
- [x] hacker_grace 申请返回 201（ID=11）

### 审核 + 选择获奖者
- [x] `POST /bounties/5/review` 返回 200，Status → 2 (InReview)
- [x] `POST /bounties/5/winners` 返回 200，传入 eve=1st, frank=2nd, grace=3rd
- [x] Bounty Status = 3 (Completed) ✓

### Winners 验证
- [x] `GET /bounties/5/winners` 返回正确的 3 个 winner 记录
- [x] Rank 正确：eve(Rank=1), frank(Rank=2), grace(Rank=3)

### 积分验证
- [ ] Credits balance API 不存在（所有尝试路径均返回 404）— 见 BUG-3

**Notes:**
- Competitive 核心流程（创建 → 申请 → review → winners）完全通过。
- **BUG-3 (P1):** 无法验证 credits 是否分发。尝试了 `/user/credits`, `/users/{user}/credits`, `/{user}/credits/balance` 等路径均返回 404。缺少 Credits Balance 查询 API。
- **API 发现:** `/review` endpoint（不是 `/start-review`，可能在本次构建中变更）；`/winners` endpoint 不变。

---

## 5. Feed 验证

- [x] 全局 feed 包含 `bounty_created` 事件 (op_type=34) — 3 个事件 ✓
- [x] 全局 feed 包含 `bounty_winners_selected` 事件 (op_type=38) — 1 个事件 ✓
- [ ] **缺失 `bounty_claimed` (35) 事件** — accept application 未生成 feed 事件
- [ ] **缺失 `bounty_delivered` (36) 事件** — PR merge hook 未触发（与 BUG-2 关联）
- [ ] **缺失 `bounty_completed` (37) 事件** — Exclusive bounty 未到达 completed 状态

**Notes:**
- **BUG-4 (P1):** Feed 系统部分工作。`bounty_created` 和 `bounty_winners_selected` 事件正常生成，但 `bounty_claimed` (35) 事件缺失。accept application 操作应触发 claimed 事件但没有。
- Following feed endpoint 返回 404（`/{user}/feed?type=following`），无法验证用户关注 feed。

---

## 6. Web 创建流程

- [x] `/acme-dev/backend/bounties/new` 页面正常加载
- [x] 表单字段完整：标题（创建悬赏）、Issue ID、模式（独占/竞赛 radio）、截止日期
- [x] 从 Issue 页面点击 "创建悬赏" 自动预填 Issue ID 和标题 ✓
- [x] 表单提交无错误，重定向到 Issue 页面并显示 Bounty panel
- [x] 新 bounty 出现在 `/explore/bounties` 列表中
- [x] 完整中文化：创建悬赏、模式、独占（一人认领并交付）、竞赛（多人参与，选择获奖者）、截止日期

**Notes:** Web Create Flow 所有检查项通过。Issue 页面预填功能工作良好。

---

## 7. Issue 列表徽章

- [x] Issue 列表页面 `/acme-dev/backend/issues` 正常加载（**之前的 500 错误已修复**）
- [x] 有 bounty 的 Issue #4、#2 显示绿色 "悬赏" 徽章（带礼物图标）
- [x] 无 bounty 的 Issue #6 不显示徽章
- [x] 徽章在列表视图直接可见，无需点击

**Notes:** Issue 列表 500 错误（`*issues.Issue` 缺少 `Bounty` 字段）已修复。徽章显示正确且中文化。

---

## Overall Result

- [ ] **PASS** — All checks passed, Phase 1 Bounty is ready
- [x] **FAIL** — Issues found (see notes above)

### 已修复（对比上次测试）

| Bug | 描述 | 状态 |
|-----|------|------|
| Issue list 500 | `*issues.Issue` 缺少 `Bounty` 字段 | ✅ 已修复 |
| Credits 显示 0 | Bounty panel credits 金额显示为 0 | ✅ 已修复（现显示 500 credits） |
| Feed 完全不工作 | 所有 feed 为空 | ✅ 部分修复（created + winners_selected 正常） |

### 仍存在的 Bug

| Bug | 严重度 | 描述 |
|-----|--------|------|
| BUG-1 | P1 | Bounty panel 显示 "token is required" — Vue 组件缺少 CSRF token（`#hackforger-bounty-panel` data attributes 未注入 `data-csrf-token`） |
| BUG-2 | **P0** | PR merge hook 未触发 Exclusive bounty 状态 Claimed→InReview 转换，阻塞 complete/paid 流程 |
| BUG-3 | P1 | Credits Balance 查询 API 不存在（404），无法验证积分分发 |
| BUG-4 | P1 | `bounty_claimed` (op_type=35) feed 事件缺失，accept application 未生成 |

**Blockers for next phase:**
- BUG-2 是唯一的 P0 Blocker：Exclusive bounty 无法走完 claimed → in_review → completed → paid 的完整生命周期。

**测试环境信息:**
- Forgejo: 14.0.3-49-2f8d37a788+gitea-1.22.0
- Admin: hackforger (ID=1)
- Test users: org_bob (ID=2), hacker_eve (ID=3), hacker_frank (ID=4), hacker_grace (ID=5), agent_hunter (ID=6)
- API auth: Token + Sudo header 实现多用户测试
