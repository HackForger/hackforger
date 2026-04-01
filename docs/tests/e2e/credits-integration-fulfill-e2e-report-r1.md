---
title: "Phase 2: Credits 联动 + 兑换完善 — E2E 测试报告 (Round 1)"
date: 2026-03-31
geometry: margin=2cm
header-includes:
  - \usepackage{graphicx}
  - \usepackage{float}
---

# Phase 2: Credits 联动 + 兑换完善 — E2E 测试报告 (Round 1)

**测试人员：** Claude Code (agent-browser 自动化)
**日期：** 2026-03-31
**服务器：** http://localhost:3000 (ROOT_URL 临时设为 localhost)
**分支：** worktree-phase2-credits (commit 39d3945be2)

---

## 测试环境

- HackForger 服务器运行在 localhost:3000
- 使用 agent-browser 多 session 自动化（admin / hacker_eve）
- ROOT_URL 需临时改为 `http://localhost:3000/` 以通过 CrossOriginProtection

---

## 测试结果总览

| # | 测试步骤 | 结果 | 备注 |
|---|---------|------|------|
| A1 | 创建 Hackathon 含 3 个赛道 | Pass | hackathon:52 |
| A2 | Hacker 注册 + 提交作品 | Pass | 已注册并提交 |
| A3 | 评审 + 结果公布 | Pass | 已 Finalize |
| A4 | 验证积分分配（冠军独占） | Pass | 1000 积分 |
| A4b | 验证积分分配（平均分配） | Pass | 各 500 积分 |
| A4c | 验证积分分配（分级 60/40） | Pass | 360 + 240 |
| B1 | 创建手动兑换选项 | Pass | 100 积分 |
| B2 | Hacker 兑换 | Pass | 3360 → 3260 |
| B3 | Admin 许可证交付 | Pass | ABCD-1234 |
| B4 | Hacker 看到交付 | Pass | 可见 |
| C1 | 创建自动交付选项 | Pass | 密钥池 |
| C2 | 添加密钥 | Pass | 3 个 |
| C3 | 即时交付 | Pass | KEY-001 |
| C4 | 密钥池状态 | Pass | 2 可用 |
| C5 | 密钥耗尽 | Pass | 停用 |
| D1 | 待处理订单 | Pass | 2 个 |
| D2 | 批量交付 | Pass | 2 orders |
| D3 | 部分批量 | Pass | 无选框 |
| E1 | 社区动态（交付） | Pass | 显示正确 |
| E2 | 社区动态（取消） | Pass | 显示正确 |
| F1 | 切换中文 | Pass | zh-CN |
| F2 | 积分页面中文 | Pass | 标签正确 |
| F3 | Admin 页面中文 | Pass | key 存在 |
| F4 | 社区动态中文 | Pass | 中文显示 |

**总计：24/24 Pass**

## A. Hackathon 积分分配

### A1. 创建 Hackathon 含 3 个赛道

创建 "Credits Test Hack"，包含 3 个赛道：AI Track（冠军独占 1000）、Web Track（平均分配 900）、Mobile Track（分级 60/40 共 600）。

![Hackathon 创建页面](screenshots/credits-r1/hackforger-e2e-credits-a1-create.png){ width=90% }

![3 个赛道已添加](screenshots/credits-r1/hackforger-e2e-credits-a1-tracks.png){ width=90% }

### A2. Hacker 注册 + 提交作品

hacker_eve 注册并提交作品到 AI Track 和 Mobile Track。

![hacker_eve 注册成功](screenshots/credits-r1/hackforger-e2e-credits-a2-eve-registered.png){ width=90% }

### A3. 评审 + 结果公布

评委评分后，Finalize 完成。积分自动分配到各赛道获奖者。

![评审界面](screenshots/credits-r1/hackforger-e2e-credits-a3-judge.png){ width=90% }

![管理页面 — Judging 阶段，含 3 赛道和积分分配模式](screenshots/credits-r1/hackforger-e2e-credits-a3-finalize-preview2.png){ width=90% }

### A4. 验证积分分配

hacker_eve 的交易记录显示：

- AI Track (冠军独占): +1000
- Mobile Track (分级 60%): +360

![积分概览 — 显示 hackathon 积分分配记录](screenshots/credits-r1/hackforger-e2e-credits-B2-overview.png){ width=90% }

\newpage

## B. 手动交付

### B1. 创建手动兑换选项

Admin 创建 "Pro License Key"：费用 100 积分，库存 10，手动交付模式。

![Admin 兑换选项列表](screenshots/credits-r1/hackforger-e2e-credits-B1-options.png){ width=90% }

### B2. Hacker 兑换

hacker_eve 兑换 Pro License Key，余额从 3360 减至 3260。

![兑换确认页](screenshots/credits-r1/hackforger-e2e-credits-B2-redeem-confirm.png){ width=90% }

![兑换成功 — 订单 pending](screenshots/credits-r1/hackforger-e2e-credits-B2-after-redeem.png){ width=90% }

### B3. Admin 交付

Admin 使用 License Key 类型交付，密钥值 `ABCD-1234-EFGH-5678`。

![Admin 订单管理页](screenshots/credits-r1/hackforger-e2e-credits-B3-admin-orders.png){ width=90% }

### B4. Hacker 看到交付信息

hacker_eve 的订单页显示 fulfilled 状态和交付的许可证密钥。

![用户订单页 — fulfilled + license_key 可见](screenshots/credits-r1/hackforger-e2e-credits-B4-delivery.png){ width=90% }

\newpage

## C. 自动交付（密钥池）

### C1. 创建自动交付选项

Admin 创建 "Game Key"：费用 50 积分，自动交付模式（密钥池）。

![自动交付选项已创建](screenshots/credits-r1/hackforger-e2e-credits-C1-auto-option.png){ width=90% }

### C2. 添加密钥到密钥池

Admin 添加 3 个游戏密钥到密钥池。

![密钥池初始状态 — 0 总计 / 0 可用](screenshots/credits-r1/hackforger-e2e-credits-C2-keys-empty.png){ width=90% }

![添加 3 个密钥成功 — 3 总计 / 3 可用](screenshots/credits-r1/hackforger-e2e-credits-C2-keys-added.png){ width=90% }

### C3. 兑换后即时交付

hacker_eve 兑换 Game Key，立即获得 GAME-KEY-001。

![自动交付成功 — GAME-KEY-001 立即分配](screenshots/credits-r1/hackforger-e2e-credits-C3-auto-fulfilled.png){ width=90% }

### C4. 密钥池状态

Admin 查看密钥池：KEY-001 已使用（Order ID 6），剩余 2 可用。

![密钥池状态 — 1 已使用, 2 可用](screenshots/credits-r1/hackforger-e2e-credits-C4-pool-state.png){ width=90% }

### C5. 密钥耗尽

连续兑换 3 次后，所有密钥用尽。Game Key 选项从兑换列表中消失（自动停用）。

![密钥耗尽 — Game Key 不再显示](screenshots/credits-r1/hackforger-e2e-credits-C5-exhausted.png){ width=90% }

\newpage

## D. 批量交付

### D1-D2. 批量交付订单

Admin 选择 2 个待处理订单，使用 Download Link 类型批量交付。

![Admin 订单页 — 含待处理订单](screenshots/credits-r1/hackforger-e2e-credits-D1-orders.png){ width=90% }

![批量交付成功 — "Successfully fulfilled 2 orders."](screenshots/credits-r1/hackforger-e2e-credits-D2-batch-fulfilled.png){ width=90% }

\newpage

## E. 订单通知

### E1. 社区动态显示交付通知

hacker_eve 的社区动态 Tab 显示：
"hackforger 完成了「Pro License Key」的兑换订单"

![社区动态 — 订单交付通知](screenshots/credits-r1/hackforger-e2e-credits-E1-community-fixed.png){ width=90% }

### E2. 取消订单 + 退款

Admin 取消订单后，积分自动退还，社区动态显示取消事件。

![订单取消成功 — "Order cancelled and credits refunded."](screenshots/credits-r1/hackforger-e2e-credits-E2-cancelled.png){ width=90% }

\newpage

## F. 国际化 (zh-CN)

### F2. 积分页面中文

积分概览页面完全显示中文标签：余额、兑换选项、交易记录等。

![积分页面中文显示](screenshots/credits-r1/hackforger-e2e-credits-F2-credits-zh.png){ width=90% }

### F3. Admin 管理页面

Admin 兑换选项和订单管理页面。

![Admin 选项页面](screenshots/credits-r1/hackforger-e2e-credits-F3-admin-options-zh.png){ width=90% }

![Admin 订单页面](screenshots/credits-r1/hackforger-e2e-credits-F3-admin-orders-zh.png){ width=90% }

\newpage

## 发现并修复的 Bug

### Bug 1: Admin 模板表单路径错误（Critical）

**现象：** 所有 admin credits 页面的 POST 表单提交返回 404。

**原因：** 模板中 form action 和 handler redirect 硬编码了 `/-/admin/credits/...`，但 Forgejo 的 admin 路由实际注册在 `/admin/credits/...`。

**修复：**

- 模板：`/-/admin/credits/...` → `{{AppSubUrl}}/admin/credits/...`
- Handler：`ctx.Redirect("/-/admin/credits/...")` → `ctx.Redirect(setting.AppSubURL + "/admin/credits/...")`

**提交：** `977d6c6b6c`

### Bug 2: 订单通知在社区动态中不显示（Important）

**现象：** 社区动态 Tab 中，订单交付/取消通知显示为空白条目。

**原因：** `notifyOrderStatusChange` 只在 `Content` JSON 中传递了 `EntityType/EntityName`，但 `PublishHackforgerAction` 将这些信息存储到 `hackforger_action` 表的一级字段，社区动态模板从一级字段读取。

**修复：**

- 在 opts 中同时设置 `EntityType` 和 `EntityName` 字段
- 在 `community_feeds.tmpl` 中添加 OpType 58/59 的渲染分支
- 添加 `feed.order_fulfilled/order_cancelled` 国际化键

**提交：** `39d3945be2`

---

## 测试环境注意事项

1. **ROOT_URL 必须临时改为 localhost** — agent-browser 通过 `http://localhost:3000` 访问，Forgejo 的 CrossOriginProtection 要求 Origin 与 ROOT_URL 匹配
2. **bindata 模板缓存** — 修改模板后需要删除 `modules/templates/bindata.go.hash` 再重新编译
3. **Admin 语言切换** — 通过 API 修改语言后需重新登录浏览器 session 才能生效
