# E2E 测试报告 — Phase 2: Reputation + Feed + Profile

**日期:** 2026-03-30
**测试人:** Claude (Cowork AI Agent)
**实例:** https://hackforger.inside.h2os.cloud
**Forgejo 版本:** 14.0.3-135-f15d6fbac4+gitea-1.22.0
**Admin 账号:** hackforger / admin1234
**测试账号:** hacker_eve（有活跃记录）、hacker_frank（无活跃记录）

---

## 结果总览

| # | 场景 | 状态 | 备注 |
|---|------|------|------|
| 1 | Dashboard tabs (A1–A5) | ✅ PASS | |
| 2 | Feed events via API (B1–B4) | ⚠️ PARTIAL | API 字段名与规格不符 |
| 3 | Profile Community tab (C1–C4) | ⚠️ PARTIAL | API 注册不产生 feed 事件 |
| 4 | Profile Reputation tab (D1–D3) | ⚠️ PARTIAL | D3 unranked tier 显示为空 |
| 5 | Reputation sidebar card (E1–E3) | ✅ PASS | |
| 6 | Reputation API (F1–F5) | ⚠️ PARTIAL | 字段缺失、tier 空、rank 缺失 |
| 7 | Admin reputation settings (G1–G5) | ❌ FAIL | G3 无效 JSON 未被拒绝 |
| 8 | Explore leaderboard (H1–H5) | ✅ PASS | |
| 9 | Reputation change E2E (I1–I3) | ⚠️ PARTIAL | recalculate 第二次调用返回 500 |
| 10 | Error cases (J1–J5) | ⚠️ PARTIAL | J1 无效 type 返回 200 而非 400 |

**总体评分: 3/10 完全通过，6/10 部分通过，1/10 失败**

---

## 详细结果

### A. Dashboard Community Tab

| 子项 | 状态 | 说明 |
|------|------|------|
| A1 | ✅ | "代码动态"与"社区动态"两个标签均可见，"代码动态"默认选中，显示标准 git 事件 |
| A2 | ✅ | Community 空状态正确显示："暂无社区动态。去探索 Hackathon、悬赏和资助吧！" |
| A3 | ✅ | 创建 Hackathon 后，Community tab 出现"hackforger 创建了 Hackathon"事件（含头像、用户名、描述、时间戳）；事件不在"代码动态"中 |
| A4 | ✅ | Phase-change 事件（"hackforger 更改了...的阶段"）出现在 Community tab；注：hacker_eve 通过 API 注册未产生 feed 事件 |
| A5 | ✅ | Tab 切换后 URL 参数变为 `?feed=community`，刷新后保持选中 |

> **注意事项:** 创建 Hackathon 时表单被提交两次（UI 点击判断问题），Community 出现重复事件。

---

### B. Feed Events via API

| 子项 | 状态 | 说明 |
|------|------|------|
| B1 | ⚠️ | `items` 数组存在，`op_type=30 (hackathon_created)` 存在；但字段名与规格不符：`actor_name` 应改为 `actor.username`，`created_unix` 实为 `created_at`，无 `content` 字段 |
| B2 | ⚠️ | Following feed 返回 3 条（均为 admin 的事件）；hacker_eve 注册事件（op_type=31）未出现（API 注册不触发 feed 记录）|
| B3 | ✅ | Entity timeline 正确返回 hackathon 相关事件（op_type=50, 30）；entity_id=99999 返回 HTTP 200 + 空 items（未崩溃）|
| B4 | ⚠️ | `limit=2` 时返回 1 条（总数本身为 1）；`total_count` 字段存在；因数据量少无法完全验证多页 |

**API 响应字段偏差汇总（对照规格）:**

| 规格期望字段 | 实际字段 | 问题 |
|------------|---------|------|
| `actor_name` (string) | `actor.username` (object) | 结构不同 |
| `created_unix` | `created_at` | 字段名不同 |
| `content` | 缺失 | 字段缺失 |

---

### C. Profile Community Tab

| 子项 | 状态 | 说明 |
|------|------|------|
| C1 | ⚠️ | Community tab 可见且选中；空状态正确显示；但 hacker_eve 通过 API 注册的事件未出现在其 profile feed（事件录入路径缺失）|
| C2 | ✅ | 页面正常加载，无编辑控件 |
| C3 | ✅ | hacker_frank（无活跃）Community tab 显示正确空状态，无崩溃 |
| C4 | N/A | 无用户有超过 10 条 HackForger 事件，无法验证分页 |

---

### D. Profile Reputation Tab

| 子项 | 状态 | 说明 |
|------|------|------|
| D1 | ✅ | hacker_eve 声誉 tab 显示 358 分，Gold tier badge（金色样式），无崩溃 |
| D2 | ✅ | 指标明细表包含：完成的悬赏(1)、Hackathon 获奖(0)、获得的资助(1)、收获的 Star(0)、累计获得积分(3500)；无 null 或渲染错误 |
| D3 | ⚠️ | hacker_frank (0 分) 页面不崩溃，显示 0；但 tier badge 显示为空字符串而非"Bronze"或"Unranked" |

> **Bug:** 零分/未初始化用户的 tier 显示为空，应显示最低等级（Bronze）。

---

### E. Reputation Sidebar Card

| 子项 | 状态 | 说明 |
|------|------|------|
| E1 | ✅ | hacker_eve 个人资料页（仓库列表 tab）左侧边栏显示声誉卡片：等级 Gold、358 分、2条指标 |
| E2 | ✅ | Admin (hackforger) 个人资料页侧边栏显示声誉卡片：0 分、空 tier；无崩溃 |
| E3 | ✅ | 未登录状态访问 hacker_eve 个人资料页，侧边栏声誉卡片正常显示（等级 Gold、358 分、指标、"查看排行榜"链接）|

> **注意:** 初次访问（声誉未计算时）侧边栏 tier 显示为空；调用 recalculate 后更新为正确值。侧边栏"查看排行榜"链接导航至 `?tab=reputation`（个人声誉 tab），非 `/explore/reputation`（排行榜页），与规格存在偏差。

---

### F. Reputation API

| 子项 | 状态 | 说明 |
|------|------|------|
| F1 | ⚠️ | HTTP 200；字段包含 `username`、`score`、`tier`；但缺少 `metrics` 对象（规格要求）；初次调用（recalculate 前）`tier` 为空字符串 |
| F2 | ✅ | 不存在用户返回 HTTP 404 |
| F3 | ⚠️ | 返回列表（HTTP 200）；但每个条目缺少 `rank` 字段（规格要求）；`tier` 可能为空 |
| F4 | ⚠️ | `limit=1` 返回 1 条；数据量少（1人），无法全面验证分页 |
| F5 | ✅ | Admin 重算返回 HTTP 200；非 Admin 使用 hacker_eve token 返回 HTTP 403 |

**API 字段缺失汇总（F1/F3）:**

| 规格要求字段 | 实际情况 |
|------------|---------|
| `metrics` (object/array) | 缺失 |
| `rank` (leaderboard 条目) | 缺失 |
| `tier` 初始化值 | 空字符串（应为 "bronze"）|

---

### G. Admin Reputation Settings

| 子项 | 状态 | 说明 |
|------|------|------|
| G1 | ✅ | 页面加载正常（路径 `/admin/hackforger/reputation`，非 `/-/admin`）；两个 JSON 输入框均有有效默认值 |
| G2 | ✅ | 修改 `stars` 权重后保存，Flash 消息"声誉设置已保存"出现，刷新后值持久化 |
| G3 | ❌ | 输入无效 JSON `{ invalid json here` 后点击保存，系统显示**成功**消息"声誉设置已保存"，**未做任何 JSON 校验** |
| G4 | ⚠️ | "重新计算"按钮存在；UI 点击后无新 Flash 消息；API `/api/v1/hackforger/reputation/recalculate`（全局）返回 404 |
| G5 | ✅ | 未认证访问 `/admin/hackforger/reputation` 返回 303（重定向至登录页）；非 Admin 同样被拦截 |

> **注意:** 保存按钮文本显示 "settings.save"（i18n key 未解析，UI 本地化 bug）。

---

### H. Explore Leaderboard

| 子项 | 状态 | 说明 |
|------|------|------|
| H1 | ✅ | 未登录可访问 `/explore/reputation`（HTTP 200）；显示排行榜表格（排名/用户名/声誉分/等级）|
| H2 | ✅ | hacker_eve 排名第 1，分数 358 |
| H3 | ✅ | Gold tier badge 有样式（金色背景），非纯字符串 |
| H4 | ⚠️ | 仅 1 个用户，`?page=999` 仍返回全部数据（无"空状态"或"最后一页"提示），因数据量少无法完全验证 |
| H5 | ✅ | 声誉 tab 内"声誉排行榜"按钮正确跳转至 `/explore/reputation` |

> **注意:** 侧边栏的"查看排行榜 →"链接指向 `?tab=reputation`（个人声誉页），与 H5 规格（应链接到 `/explore/reputation`）有偏差。

---

### I. 端到端 Reputation 变化验证

| 子项 | 状态 | 说明 |
|------|------|------|
| I1 | ✅ | SCORE_BEFORE = 358（hacker_eve，已有悬赏+资助历史）|
| I2 | ❌ | 第二次调用 `POST /api/v1/hackforger/reputation/recalculate/hacker_eve` 返回 **HTTP 500**（首次调用 F5 时返回 200）|
| I3 | ⚠️ | 因 I2 失败，分数未变化（SCORE_AFTER = 358）；UI 中 hacker_eve 声誉 tab 正确显示 358 与 Gold tier |

> **Bug:** 单用户 recalculate 端点在短时间内连续调用时返回 500。

---

### J. Error Cases & Edge Cases

| 子项 | 状态 | 说明 |
|------|------|------|
| J1 | ❌ | `?type=invalid_type` 返回 **HTTP 200**（规格要求 400）|
| J2 | ✅ | 未认证访问 following feed 返回 HTTP 401 |
| J3 | ✅ | 无 token 访问 global feed 返回 HTTP 200 |
| J4 | ✅ | 访问不存在用户的 community tab 返回 HTTP 404 |
| J5 | ✅ | hacker_frank（无关注）Community tab 显示全局受众事件（或正确空状态），无崩溃 |

---

## 发现的 Bug 汇总

### 🔴 Critical（需立即修复）

**BUG-1: G3 — 无效 JSON 被接受且保存成功**
路径：`/admin/hackforger/reputation` → 权重/阈值输入框
现象：输入 `{ invalid json here` 后点击保存，系统返回成功消息"声誉设置已保存"，无任何校验错误
影响：恶意或误操作输入会破坏 reputation 计算配置，导致系统功能异常

---

### 🟠 Major（功能缺陷）

**BUG-2: J1 — Feed API 对无效 `type` 参数未返回 400**
`/api/v1/hackforger/feed?type=invalid_type` 返回 HTTP 200（期望 400 Bad Request）
影响：客户端无法通过错误码区分无效请求

**BUG-3: I2 — 单用户 recalculate 端点连续调用返回 HTTP 500**
`POST /api/v1/hackforger/reputation/recalculate/{username}` 短时间内第二次调用失败
影响：声誉重算不可靠，存在幂等性问题

**BUG-4: B 系列 / F 系列 — API 字段名与规格不符**
- Feed API: `actor_name` → `actor.username`；`created_unix` → `created_at`；缺少 `content` 字段
- Reputation API: 缺少 `metrics` 对象；Leaderboard 缺少 `rank` 字段
影响：前端/客户端集成需额外适配

**BUG-5: D3 / F3 — 未计算声誉的用户 `tier` 为空字符串**
期望值：`"bronze"` 或 `"unranked"`；实际值：`""`
影响：UI 中 tier badge 显示空白（Bronze 等级应作为默认值）

---

### 🟡 Minor（体验问题）

**BUG-6: G1 — 保存按钮文本为 "settings.save"（i18n key 未解析）**

**BUG-7: H5/E1 侧边栏 — "查看排行榜"链接指向个人声誉 tab 而非 `/explore/reputation`**
规格：侧边栏链接应直接跳转排行榜页；实际：跳转至 `?tab=reputation`（个人页）

**BUG-8: G4 — 管理后台"重新计算"按钮点击后无 Flash 反馈**
全局 recalculate API（`/api/v1/hackforger/reputation/recalculate`）不存在（返回 404）

**BUG-9: B2 — 通过 API 注册 Hackathon 不产生 feed 事件**
hacker_eve 通过 `/api/v1/hackforger/hackathons/{id}/register` 注册后，following feed 中无 `HackathonRegistered (op_type=31)` 事件
影响：API 注册与 UI 注册行为不一致

**BUG-10: A3 — 创建 Hackathon 产生重复 feed 事件**
Community feed 中出现两条相同的 "创建了 Hackathon" 事件

---

## Feed Event Types 验证汇总

| op_type | op_name | 出现于 Feed |
|---------|---------|------------|
| 30 | hackathon_created | ✅ |
| 31 | HackathonRegistered | ❌ 未出现（API 注册未触发）|
| 32 | HackathonSubmitted | N/A（未测试）|
| 33 | HackathonScored | N/A（未测试）|
| 50 | hackathon_phase_changed | ✅ |
| 51 | HackathonFinalized | N/A（未测试）|

---

## Reputation Tiers 验证汇总

| Tier | 阈值达到 | Badge 显示 |
|------|---------|-----------|
| Bronze (最低, min=0) | ⚠️ 有 score=0 用户但 badge 显示为空 | ❌ 空字符串 |
| Silver (min=50) | N/A | N/A |
| Gold (min=200) | ✅ hacker_eve score=358 | ✅ 金色 badge |
| Diamond (min=500) | N/A | N/A |

---

## 测试环境说明

- Admin 路径为 `/admin`（非标准 `/-/admin`），G5 规格描述需更新
- Hackathon 发布需要先添加赛道（track），规格中未明确此前置条件
- 声誉计算依赖已有的 Phase 1 数据（bounties、grants），测试开始前 hacker_eve 已有历史记录（score=358/Gold 为预存数据）
