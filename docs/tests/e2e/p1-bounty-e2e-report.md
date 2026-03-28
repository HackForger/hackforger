# Phase 1 Bounty — E2E Round 3 Test Report

> **Tester:** Claude (automated E2E)
> **Date:** 2026-03-28
> **Instance:** https://hackforger.inside.h2os.cloud
> **Branch/Commit:** Forgejo 14.0.3-50-dd3a4a05c1+gitea-1.22.0

---

## Round 3 修复验证目标

| Bug | 修复描述 | 验证结果 |
|-----|----------|----------|
| BUG-1 (P1) | Vue BountyPanel 401 错误已 silenced（使用 Forgejo fetch helpers 自动携带 session） | **已修复** |
| BUG-2 (P0) | PR merge hook: CreateBounty resolver 改用 GetIssueByIndex 查找 Issue DB ID | **未修复** |
| BUG-3 (P1) | Credits Balance API 新增 `?user_id=N` 参数 | **未验证（404）** |
| BUG-4 (P1→非 Bug) | `bounty_claimed` 为 Followers+Watchers 事件，不出现在 Global feed | **已确认修复** |

---

## 1. Issue 面板 — BUG-1 验证

- [x] Issue #1（Bounty 7）sidebar 显示 Bounty panel：状态 "开放"（绿色标签）
- [x] 奖励正确显示：`USD 200 | 500 credits`
- [x] Publisher（hackforger）看到 "Cancel Bounty" 操作按钮
- [x] **无 "token is required" 错误** — BUG-1 已修复

**Notes:** Vue BountyPanel 不再显示红色错误消息。401 错误已被 silenced，组件使用 Forgejo fetch helpers 自动携带 session 认证。

---

## 2. Exclusive Bounty 流程（Bounty 7 → Issue #1）

### 申请认领
- [x] hacker_eve（Sudo）申请返回 201（Application ID=12, UserID=3）

### 接受申请
- [x] `PUT /applications/12` + `{"action":"accept"}` 返回 200
- [x] Bounty status = 1 (Claimed) ✓
- [x] ClaimerID = 3 (hacker_eve) ✓

### PR 创建 & 合并
- [x] eve 创建分支 `eve-auth-fix-r3` 并提交代码（commit message: `fix: refactor auth module - fixes #1`）
- [x] PR #8 "fix: refactor auth module - fixes #1" 创建成功
- [x] PR #8 合并成功（merged=true, merged_at=2026-03-28T23:45:55+08:00）
- [x] Issue #1 已自动关闭（state=closed）
- [ ] **Bounty 状态仍为 Claimed (1)，未转换为 InReview (2)** — BUG-2 仍存在

### Complete / Paid
- [ ] `POST /bounties/7/complete` 返回 422："invalid bounty status [bounty_id: 7, current: 1, expected: InReview]"
- [ ] Complete 和 Paid 流程被阻塞（与 Round 2 相同）

**Notes:**
- **BUG-2 (P0 Blocker) 仍未修复。** PR #8 成功合并，Issue #1 被 Forgejo 自动关闭，但 Bounty 7 的状态仍停留在 Claimed (1)。等待 5 秒后重新检查，结果不变。PR merge hook 仍未触发 Bounty 状态从 Claimed → InReview 的转换。
- **可能原因分析：** Bounty 7 的 IssueID=1，Issue #1 的 DB ID=1 且 Index=1（两者相同）。即使 GetIssueByIndex 修复正确，hook 可能在其他环节（如事件监听注册、异步队列）未正确触发。建议检查 MergePullRequest hook 中 bounty 状态转换的完整调用链。

---

## 3. Feed 验证

- [x] `action-34` (bounty_created) — 2 个事件 ✓
- [x] **`action-35` (bounty_claimed) — 1 个事件 ✓** — Round 2 缺失，现已修复
- [ ] `action-36` (bounty_delivered) — 缺失（BUG-2 阻塞，PR merge 未触发状态转换）
- [ ] `action-37` (bounty_completed) — 缺失（Exclusive bounty 未到达 completed 状态）

**Notes:**
- **BUG-4 已澄清并修复：** `bounty_claimed` (action-35) 事件现已正常生成。Round 2 中在 Global feed 中找不到是因为该事件仅推送给 Followers + Watchers，不进入 Global feed。本轮通过 `/users/hackforger/activities/feeds` 确认存在。
- Feed endpoint: `/api/v1/users/{username}/activities/feeds`（非 `/feeds` 或 `/user/feeds`）。
- `bounty_delivered` 和 `bounty_completed` 缺失均因 BUG-2 阻塞，非 Feed 系统本身问题。

---

## 4. Credits Balance API 验证

- [ ] `/api/v1/user/credits/balance?token=T` — 404
- [ ] `/api/v1/user/credits/balance?token=T&user_id=3` — 404
- [ ] `/api/v1/users/hackforger/credits/balance` — 404
- [ ] `/api/v1/user/credits` — 404
- [ ] Swagger spec 中无 credits 相关路径

**Notes:**
- **BUG-3 (P1) 仍存在。** Credits Balance API 的所有尝试路径均返回 404。Swagger 规范中也未发现任何 credits 相关 endpoint。可能修复尚未部署到当前实例，或路径与预期不同。

---

## 5. 未变化项（沿用 Round 2 结果）

以下测试项在 Round 2 中已通过，Round 3 未重新测试（功能未变）：

- **Explore 页面：** 全部通过（加载、i18n、filter、bounty 卡片）
- **Web 创建流程：** 全部通过（表单、预填、提交、重定向）
- **Issue 列表徽章：** 全部通过（500 错误已修复、徽章正确显示）
- **Competitive Bounty 流程：** 核心流程通过（apply → review → winners → completed）

---

## Overall Result

- [ ] **PASS** — All checks passed, Phase 1 Bounty is ready
- [x] **FAIL** — Issues found (see notes above)

### 修复历程（三轮测试对比）

| Bug | Round 1 | Round 2 | Round 3 |
|-----|---------|---------|---------|
| Issue list 500 | P0 Blocker | **已修复** | 已修复 |
| Credits 显示 0 | P1 | **已修复** | 已修复 |
| Feed 完全不工作 | P0 Blocker | **部分修复** | 进一步修复 |
| BUG-1: "token is required" | P1 | P1 | **已修复** |
| BUG-2: PR merge hook | P0 Blocker | P0 Blocker | **P0 仍存在** |
| BUG-3: Credits API 404 | P1 | P1 | **P1 仍存在** |
| BUG-4: claimed feed 缺失 | P1 | P1 | **已澄清（非 Global 事件，已修复）** |

### 仍存在的 Bug

| Bug | 严重度 | 描述 |
|-----|--------|------|
| BUG-2 | **P0** | PR merge hook 未触发 Exclusive bounty 状态 Claimed→InReview 转换。PR #8 合并后 Issue #1 关闭但 Bounty 7 状态不变。阻塞 complete/paid 全流程。 |
| BUG-3 | P1 | Credits Balance 查询 API 不存在（所有路径 404），Swagger 中无 credits endpoint |

### 已修复的 Bug

| Bug | 修复确认 |
|-----|----------|
| BUG-1 | Vue BountyPanel 不再显示 "token is required"（401 silenced，使用 Forgejo fetch helpers） |
| BUG-4 | `bounty_claimed` (action-35) 事件已生成。之前误判为缺失，实为 Followers+Watchers 级别事件 |

**Blockers for next phase:**
- **BUG-2 是唯一的 P0 Blocker：** Exclusive bounty 无法走完 claimed → in_review → completed → paid 的完整生命周期。建议重点排查 MergePullRequest hook 中 bounty 状态转换的触发条件和调用链。

**测试环境信息:**
- Forgejo: 14.0.3-50-dd3a4a05c1+gitea-1.22.0
- Admin: hackforger (ID=1)
- Test users: hacker_eve (ID=3), hacker_frank (ID=4), hacker_grace (ID=5), agent_hunter (ID=6)
- Test Bounties: Bounty 7 (Exclusive, Issue #1), Bounty 8 (Competitive, Issue #2)
- API auth: Token + Sudo header 实现多用户测试
- Feed endpoint: `/api/v1/users/{username}/activities/feeds`
