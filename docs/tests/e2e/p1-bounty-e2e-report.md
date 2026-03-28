# Phase 1 Bounty — E2E Test Report

> **Tester:** Claude (automated E2E)
> **Date:** 2026-03-28
> **Instance:** https://hackforger.inside.h2os.cloud
> **Branch/Commit:** Forgejo 14.0.3-44-b07705f82e+gitea-1.22.0

---

## 1. Explore Page Verification

- [x] `/explore/bounties` loads without error
- [x] Explore navbar shows Bounties tab highlighted
- [x] Empty state or bounty listing renders correctly
- [x] Status filter tabs present (All / Open / Claimed / Completed)
- [x] Filters update the list when clicked
- [ ] Pagination works (if enough bounties exist) — N/A, only 3 bounties

**Notes:** Explore Bounties 页面正常加载，navbar 中 Bounties tab 正确高亮。状态 filter tabs（All / Open / Claimed / Completed）均存在，点击可正确过滤列表。最终列表显示 3 个 bounties：Refactor auth module (Claimed/Exclusive)、Best CLI tool challenge (Completed/Competitive)、Fix database connection pooling (Open/Exclusive)。

---

## 2. Exclusive Bounty Flow

### Step 1: Create Exclusive Bounty
- [x] POST returns 201 Created
- [x] Response contains correct `mode: 0` (exclusive), `status: "open"`

### Step 2: Add Rewards
- [x] Money reward added (201)
- [x] Credits reward added (201)
- [x] GET bounty shows 2 rewards

### Step 3: Feed — bounty_created
- [ ] Global feed contains bounty_created event (action_type 34) — **FAIL: feed 为空**
- [ ] hacker_eve (repo watcher) sees event in following feed — **FAIL: feed 为空**
- [ ] agent_hunter (follows org_bob) sees event in following feed — **FAIL: feed 为空**
- [x] hacker_grace (no relationship) does NOT see event in following feed — 空（但因所有 feed 都空，无法区分是正确行为还是 bug）
- [ ] hacker_grace CAN see event in global feed — **FAIL: global feed 也为空**

### Step 4: hacker_eve applies
- [x] Application POST returns 201

### Step 5: agent_hunter applies
- [x] Application POST returns 201

### Step 6: org_bob accepts eve's application
- [x] Accept PUT returns 200（使用 `PUT /applications/{id}` + body `{"action":"accept"}`）
- [x] Bounty status = "claimed" (status=1)
- [x] Bounty claimer_id = eve's user ID (3)

### Step 7: Feed — bounty_claimed
- [ ] hacker_frank (follows eve) sees bounty_claimed in following feed — **FAIL: feed 为空**
- [ ] hacker_grace (follows eve) sees bounty_claimed in following feed — **FAIL: feed 为空**

### Step 8: eve creates PR + org_bob merges
- [x] PR created successfully（PR #3, branch eve-auth-refactor）
- [x] PR merged successfully
- [ ] Bounty status transitions to "in_review" / "delivered" — **FAIL: status 仍为 1 (claimed)**，MergePullRequest hook 未触发状态转换

### Step 9: Feed — bounty_delivered
- [ ] Entity feed contains bounty_delivered event (action_type 36) — **FAIL: feed 为空**

### Step 10: org_bob completes bounty
- [ ] Complete PUT returns 200 — **FAIL: POST /complete 返回 500**，因为 bounty status=1 (claimed) 而非 status=2 (in_review)，complete 操作被阻止
- [ ] Bounty status = "completed" — **FAIL: 无法完成**

### Step 11: Feed + Credits — bounty_completed
- [ ] hacker_eve credit balance increased by 500 — **FAIL: balance=0**
- [ ] hacker_frank sees bounty_completed in following feed — **FAIL: feed 为空**

### Step 12: org_bob marks paid
- [ ] Paid PUT returns 200 — **未测试**（被 Step 10 阻塞）
- [ ] Bounty status = "paid" — **未测试**

### Step 13: Entity feed completeness
- [ ] Feed contains: created -> claimed -> delivered -> completed -> paid (in order) — **FAIL: feed 始终为空**

**Notes:**
- **BUG-1 (P0 Blocker): Feed 系统未生成任何事件。** 所有 bounty 生命周期事件（created, claimed, delivered, completed）均未写入 feed，所有用户的 global feed 和 following feed 始终为空。
- **BUG-2 (P0 Blocker): PR merge hook 未触发 bounty 状态转换。** 合并关联 bounty issue 的 PR 后，bounty status 仍停留在 claimed (1)，未转换为 in_review (2)。这阻塞了后续的 complete 和 paid 操作。
- **BUG-3 (P1): Credits 未分发。** hacker_eve 的 credit balance 始终为 0，bounty complete 操作未触发 credits 分发。
- **API 发现**: `mode` 字段为 integer（0=exclusive, 1=competitive）；`deadline` 为 unix timestamp (int64)；accept application 使用 `PUT /applications/{id}` + `{"action":"accept"}`。

---

## 3. Competitive Bounty Flow

### Step 1: Create Competitive Bounty + ranked rewards
- [x] Bounty created with `mode: 1` (competitive), `status: "open"`
- [x] 3 ranked rewards added (1000, 500, 200 credits) — 均返回 201

### Step 2: Feed — bounty_created (global)
- [ ] Global feed contains bounty_created event for competitive bounty — **FAIL: feed 为空**

### Step 3: Submissions
- [x] hacker_eve application returns 201
- [x] hacker_frank application returns 201
- [x] hacker_grace application returns 201

### Step 4: Review + select winners
- [x] Review POST returns 200（使用 `POST /start-review`）
- [x] Winners POST returns 200 (eve=1st, frank=2nd, grace=3rd)
- [x] Bounty status = "completed" (status=3)

### Step 5: Feed — winners_selected (global)
- [ ] Global feed contains winners_selected event — **FAIL: feed 为空**

### Step 6: Credits distribution
- [ ] hacker_eve balance >= 1500 (500 from exclusive + 1000 from competitive) — **FAIL: balance=0**
- [ ] hacker_frank balance >= 500 — **FAIL: balance=0**
- [ ] hacker_grace balance >= 200 — **FAIL: balance=0**

**Notes:**
- Competitive bounty 的核心 flow（创建 → 添加奖励 → 提交 → start-review → 选择 winners）可以跑通。
- **BUG-3 再确认**: Winners 选择成功后 credits 未分发到任何用户账户。
- **API 发现**: review 阶段使用 `POST /start-review`（非 `/review`）；winners 选择使用 `POST /winners` + `{"winners":[{"user_id":N,"rank":N}]}`。

---

## 4. Issue Panel Verification

- [x] Bounty panel appears in issue sidebar for issue #1
- [x] Panel shows correct status label — Issue #1 显示 "Claimed"，Issue #2 显示 "Completed"
- [ ] Rewards displayed (200 USD + 500 credits) — **部分 FAIL**: Issue #1 显示 "USD 200 | 0 credits"，credits 金额显示为 0（应为 500）
- [ ] Claimer name shown (hacker_eve) — **FAIL: 未显示 claimer 名称**
- [ ] As org_bob: action buttons appear — **部分 PASS**: Issue #4 (Open 状态) 有 "Cancel Bounty" 按钮；Issue #1 (Claimed) 和 #2 (Completed) 无 action buttons
- [ ] Buttons contextually enabled/disabled based on status — **部分 FAIL**: 仅 Open 状态有按钮，其他状态缺少应有的操作按钮（如 Complete、Mark Paid）
- [x] Issue #2 panel shows mode: competitive — 未直接显示 mode 标签，但 Explore 页面确认为 competitive
- [ ] Issue #2 panel shows winners with ranks — **FAIL: 未显示任何 winner 信息**

**Notes:**
- **BUG-4**: Bounty panel 中 credits 类型的 reward 金额始终显示为 0（应显示实际金额如 500、1000 等）。Money 类型的 reward (USD 200) 显示正确。
- **BUG-5**: Bounty panel 未显示 claimer 名称。
- **BUG-6**: Competitive bounty 的 panel 未显示 winners 及其排名。
- **BUG-7**: Bounty panel 在非 Open 状态下缺少 publisher action buttons（Complete、Mark Paid 等）。
- **BUG-8**: Issue #4 的 bounty panel 显示 "token is required" 错误信息（红色背景）。

---

## 5. Issue List Badge

- [ ] Issues with bounties show green "Bounty" badge — **FAIL: 500 Internal Server Error**
- [ ] Issues without bounties do NOT show the badge — **FAIL: 页面完全崩溃**
- [ ] Badge visible in issue list view (no click-through needed) — **FAIL: 页面不可访问**

**Notes:**
- **BUG-9 (P0 Blocker): Issue list 页面 500 错误。** 访问 `/acme-dev/backend/issues` 时返回 Internal Server Error。错误信息：`Render failed, failed to render template: repo/issue/list, error: template error: builtin(bindata):shared/issuelist:17:11 : executing "shared/issuelist" at <.Bounty>: can't evaluate field Bounty in type *issues.Issue`。模板在 `shared/issuelist` 第17行尝试访问 `{{if .Bounty}}{{template "hackforger/bounty/badge" .}}{{end}}`，但 `*issues.Issue` 结构体缺少 `Bounty` 字段。这是一个 **P0 Blocker**，因为它导致所有含 bounty 的 repo 的 issue 列表页面完全不可访问。

---

## 6. Web Create Flow

- [x] `/acme-dev/backend/bounties/new` page loads
- [x] Form fields: title, issue, mode, deadline present
- [x] Form submits without error
- [x] Redirects to issue page with bounty panel
- [x] New bounty appears in `/explore/bounties`

**Notes:**
- Web Create Flow 整体流程通畅。表单包含 Title、Issue ID、Mode（Exclusive/Competitive 单选按钮）、Deadline（日期选择器）。提交后显示 "Bounty created successfully" 提示并重定向到对应 issue 页面。
- **BUG-8 再确认**: 新创建的 bounty 在 issue panel 中显示 "token is required" 错误（可能是 CSRF 或 API token 问题）。
- 新 bounty 正确出现在 `/explore/bounties` 列表中（Open / Exclusive）。

---

## Overall Result

- [ ] **PASS** — All checks passed, Phase 1 Bounty is ready
- [x] **FAIL** — Issues found (see notes above)

**Blockers for next phase:**

1. **BUG-9 (P0)**: Issue list 页面 500 错误 — `*issues.Issue` 缺少 `Bounty` 字段，导致模板渲染崩溃。所有含 bounty 的 repo 的 issue 列表不可访问。
2. **BUG-1 (P0)**: Feed 系统完全不工作 — 所有 bounty 生命周期事件均未写入 feed。
3. **BUG-2 (P0)**: PR merge hook 未触发 bounty 状态转换 — exclusive bounty 在 PR 合并后无法进入 in_review 状态，阻塞 complete/paid 流程。

**Other observations:**

- **BUG-3**: Credits 分发不工作 — bounty complete 和 winners 选择后均未给用户发放 credits。
- **BUG-4**: Bounty panel 中 credits 类型 reward 金额始终显示为 0。
- **BUG-5**: Bounty panel 未显示 claimer 名称。
- **BUG-6**: Competitive bounty panel 未显示 winners 信息。
- **BUG-7**: Bounty panel 在非 Open 状态缺少 publisher action buttons。
- **BUG-8**: Bounty panel 显示 "token is required" 错误。
- **API 接口与文档差异**: `mode` 为 int（非 string），`deadline` 为 unix timestamp（非 ISO 格式），accept 和 review 的 endpoint pattern 与 prompt 中记录的不一致。建议更新 API 文档。
- Forgejo 版本已更新为 **14.0.3-44-b07705f82e+gitea-1.22.0**（较 P0 测试时有新提交）。
