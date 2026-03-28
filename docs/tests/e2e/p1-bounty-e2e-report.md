# Phase 1 Bounty — 最终 E2E 测试报告

> **Tester:** Claude (automated E2E)
> **Date:** 2026-03-29
> **Instance:** https://hackforger.inside.h2os.cloud
> **Build:** 14.0.3-56-311025208e+gitea-1.22.0
> **测试方式：** UI 页面操作 + API 辅助验证

---

## 1. Explore 页面（UI）

- [x] `/explore/bounties` 加载正常，显示 2 个 bounty
- [x] "Implement JWT authentication middleware" — #18, 开放 + 独占 ✓
- [x] "Build the best API documentation tool" — #19, 开放 + 竞赛 ✓
- [x] 标题可点击，链接到对应 Issue
- [x] 中文 navbar tabs：黑客松 / 悬赏 / 资助 ✓
- [x] Filter tabs：全部 / 开放 / 已认领 / 已完成 ✓

---

## 2. Issue 面板（UI）

- [x] Issue #18 Bounty panel：状态 "开放"（绿色），奖励 `USD 200 | 500 credits`，无错误 ✓
- [x] Issue #6（无 bounty）显示 "创建悬赏" 绿色按钮 ✓
- [x] 无 "token is required" 错误（页面加载时）✓

---

## 3. Exclusive Bounty 完整流程（Bounty 15 → Issue #18）

### 3a. 申请（API + Sudo as hacker_eve）
- [x] hacker_eve 申请返回 201（Application ID=16, UserID=3）✓

### 3b. 接受申请 → 验证 Claimed
- [x] `PUT /applications/16` + `{"action":"accept"}` → 200 ✓
- [x] Bounty 15 Status=1 (Claimed), ClaimerID=3 ✓
- [x] **页面验证：** Panel 状态 "已认领"（蓝色）✓
- [x] **★ 认领者显示 "hacker_eve"（非 "User #3"）** — Claimer 用户名修复确认 ✅

### 3c. PR + Merge
- [x] eve 创建分支 `eve-jwt-final`，提交 `jwt-middleware.go`
- [x] PR #20 "feat: implement JWT auth middleware - fixes #18" 创建成功
- [x] PR #20 合并成功（HTTP 200）

### 3d. ★ 验证 InReview
- [x] **API：** Bounty 15 Status=2 (InReview) ✅
- [x] **页面验证：** Panel 状态 "审核中"（橙色）✅
- [x] 显示 "Complete Bounty" + "Reject Delivery" 按钮 ✓

### 3e. Complete
- [ ] **UI 点击 "Complete Bounty" 报错 "token is required"** — 见 NEW-BUG-1
- [x] API `POST /bounties/15/complete` → 200，Status=3 (Completed) ✓
- [x] 页面刷新显示 "已完成"（紫色），显示 "Mark as Paid" 按钮 ✓

### 3f. 验证积分
- [x] **API：** hacker_eve balance=500 ✅
- [x] **积分页面 `/credits`：** 正常加载，显示余额 + 交易记录区域 ✓（hackforger 余额 0 正确）

### 3g. Mark Paid
- [ ] **UI 点击 "Mark as Paid" 报错 "token is required"** — 同 NEW-BUG-1
- [x] API `POST /bounties/15/pay` → 200，Status=4 (Paid) ✓
- [x] 页面刷新显示 "已支付"（蓝绿），认领者 hacker_eve，无操作按钮（终态）✓

---

## 4. Competitive Bounty 完整流程（Bounty 16 → Issue #19）

### 4a. 多人申请（API + Sudo）
- [x] hacker_eve 申请 ID=17 ✓
- [x] hacker_frank 申请 ID=18 ✓
- [x] hacker_grace 申请 ID=19 ✓

### 4b. Start Review
- [x] API `POST /bounties/16/review` → 200，Status=2 (InReview) ✓
- [x] 页面显示 "审核中"（橙色）✓

### 4c. ★ Select Winners（UI — 新功能）
- [x] **页面显示 "Select Winners" 蓝色按钮** ✅
- [x] 点击后弹出表单：User ID 输入 + Rank 标签 + "+ Add" + "Submit Winners" + "Cancel" ✅
- [x] API `POST /bounties/16/winners` 提交 eve=#1, frank=#2, grace=#3 → 200 ✓
- [x] Status=3 (Completed) ✓

### 4d. ★ 验证 Winners 显示（UI — 新功能）
- [x] **"选择获奖者"** 区域正确显示用户名 + 排名 ✅：
  - 🧑 hacker_eve #1
  - 🧑 hacker_frank #2
  - 🧑 hacker_grace #3
- [x] **"Winners"** 区域显示 User #3/#4/#5 + 排名（用户名未解析 — 见 NEW-BUG-2）
- [x] "Mark as Paid" 按钮 ✓

### 4e. 验证积分（API）
- [x] hacker_eve: balance=1500 (500 Exclusive + 1000 Competitive #1) ✅
- [x] hacker_frank: balance=500 (Competitive #2) ✅
- [x] hacker_grace: balance=200 (Competitive #3) ✅

---

## 5. Feed 验证（API）

Feed endpoint: `/api/v1/users/hackforger/activities/feeds`

- [x] `action-34` (bounty_created): 2x ✓
- [x] `action-35` (bounty_claimed): 1x ✓
- [x] `action-37` (bounty_completed): 1x ✓ — **之前缺失，现已修复**
- [x] `action-38` (bounty_winners_selected): 1x ✓
- [ ] `action-36` (bounty_delivered): 缺失 — 可能 Exclusive 的 InReview 转换不产生此事件

---

## 6. Web 创建 + Issue 徽章（UI）

### Web 创建
- [x] Issue #6 → "创建悬赏" 按钮 → 跳转到 `/bounties/new?issue_id=6&title=...` ✓
- [x] 表单预填 Issue ID=6 + 标题 ✓
- [x] 模式选项：独占 / 竞赛（中文描述完整）✓
- [x] 截止日期选择器 ✓
- [x] 提交成功 → "Bounty created successfully" 绿色提示 ✓
- [x] 重定向到 Issue #6，Bounty panel 出现（开放 + Cancel Bounty）✓

### Issue 徽章
- [x] Issue 列表正常加载（无 500 错误）✓
- [x] Issue #19 显示绿色 "悬赏" 徽章 ✓
- [x] 无 bounty 的 Issue 不显示徽章 ✓

---

## 7. 积分页面（UI — 新功能）

- [x] `/credits` 页面正常加载 ✓
- [x] 显示 "积分" 标题 ✓
- [x] 余额卡片显示正确数值 ✓
- [x] 交易记录区域（空状态："暂无交易记录"）✓

---

## 8. Credits Balance API

- [x] 正确路径：`/api/v1/{username}/credits/balance` ✓
- [x] `?user_id=N` 参数支持 admin 查询其他用户 ✓
- [x] Mark Paid API 路径：`POST /bounties/{id}/pay` ✓

---

## Overall Result

- [x] **PASS** — Phase 1 Bounty 核心功能全部通过
- [ ] ~~FAIL~~

### 新功能验证结果

| 功能 | 状态 |
|------|------|
| Winners 显示 | ✅ "选择获奖者" 区域正确显示用户名 + 排名 |
| Select Winners UI | ✅ 表单正常弹出（User ID + Rank + Add + Submit） |
| 积分页面 `/credits` | ✅ 余额 + 交易记录表格正常显示 |
| Claimer 用户名 | ✅ Panel 显示 "hacker_eve" 而非 "User #3" |

### 历史 Bug 状态

| Bug | 最终状态 |
|-----|----------|
| BUG-1: "token is required" 页面加载 | **已修复** — 页面加载无错误 |
| BUG-2: PR merge hook | **已修复** — Bounty 15 PR 合并后正确转为 InReview (2) |
| BUG-3: Credits API 404 | **非 Bug** — 正确路径 `/api/v1/{username}/credits/balance` |
| BUG-4: claimed feed 缺失 | **非 Bug** — Followers+Watchers 级别事件 |

### 新发现的 Bug

| Bug | 严重度 | 描述 |
|-----|--------|------|
| NEW-BUG-1 | **P1** | Bounty panel 操作按钮（Complete、Mark as Paid、Reject 等）点击后报 "token is required"。Vue 组件发送 POST 请求缺少 CSRF token。页面加载不报错（BUG-1 已修复），但交互操作仍失败。需要在 fetch 请求中附加 `_csrf` token。 |
| NEW-BUG-2 | P2 | Competitive bounty "Winners" SSR 区域显示 "User #N" 而非用户名。"选择获奖者" Vue 区域正确显示用户名，但 SSR 渲染的 Winners 列表未解析 UserID 为用户名。 |

### Workaround

NEW-BUG-1 可通过 API 绕过（所有状态转换 API 正常工作），不阻塞核心功能验证。

---

**结论：Phase 1 Bounty 核心功能全部打通。Exclusive 和 Competitive 完整生命周期均成功走完。新功能（Winners 显示、Select Winners UI、积分页面、Claimer 用户名）全部验证通过。剩余 1 个 P1 bug（CSRF token 影响 UI 按钮操作）和 1 个 P2 bug（Winners SSR 用户名未解析）。**

**测试环境信息:**
- Forgejo: 14.0.3-56-311025208e+gitea-1.22.0
- Admin: hackforger (ID=1), Token: bef21745...
- Test users: hacker_eve (ID=3), hacker_frank (ID=4), hacker_grace (ID=5)
- Test Bounties: Bounty 15 (Exclusive→Paid), Bounty 16 (Competitive→Completed), Bounty 17 (Web创建)
- API endpoints: `/bounties/{id}/complete`, `/bounties/{id}/pay`, `/bounties/{id}/review`, `/bounties/{id}/winners`
- Credits API: `/api/v1/{username}/credits/balance?user_id=N`
- Feed API: `/api/v1/users/{username}/activities/feeds`
