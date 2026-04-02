# Phase 1: Grant + Credits — E2E Test Report (Round 2)

**Tester:** Claude (Automated)
**Date:** 2026-03-30
**Server URL:** https://hackforger.inside.h2os.cloud
**Commit/Branch:** 14.0.3-105-85c3160a6f（上轮：14.0.3-101-cf4c171e5a，+4 commits）
**Test Method:** curl + Chrome session cookie automation
**对比上轮：** Round 1 (2026-03-29) → Round 2 (2026-03-30)

---

## Test Results

| # | Test Step | Result | Notes |
|---|-----------|--------|-------|
| A1 | Explore grants page loads | ✅ Pass | "No grant rounds found."，搜索栏、状态过滤器、排序均正常；UI 现已显示英文（i18n 修复） |
| A2 | Navbar "+" has Grant Round entry | ✅ Pass | "新建资助轮次" 链接存在于 navbar |
| B1 | Create organization | ✅ Pass | `test-grants-org` (ID=24) 已存在，跳过重建 |
| B2 | Create grant round (Draft) | ✅ Pass | Flash: `资助轮次创建成功。`（i18n 正确），Status: draft，Budget: 10000 USD + 5000 credits |
| B3 | Open round | ✅ Pass | Flash: `资助轮次已开放提交。`，Status: open，manage 页面**无 500 错误**（Bug #1 修复 ✓） |
| B4 | Explore search + filter + sort | ✅ Pass | Open 过滤 ✓，搜索 "Spring" ✓，字母排序 ✓，Sort 选项显示 `Newest/Oldest/Alphabetically`（Bug #7 修复 ✓） |
| C1 | Submit project (with repo) | ✅ Pass | Flash: `项目提交成功。`，Status: 待审核，repo 链接 `hacker_eve/main-track-hacker_eve` 正确 |
| C2 | Duplicate submission error | ✅ Pass | Flash error: `您已向该轮次提交过项目。`，303 重定向至 round 页（Bug #3 修复 ✓） |
| C3 | Second hacker submits | ✅ Pass | hacker_frank 提交 "Another Widget" with repo，Flash: `项目提交成功。` |
| D1 | Close applications (Review) | ✅ Pass | Flash: `申请已关闭，轮次进入审核阶段。`，Status: review |
| D2 | Approve projects | ✅ Pass | Flash: `项目已批准。`（两个项目） |
| D3 | Allocate awards | ✅ Pass | Flash: `奖励分配已保存。`（Eve: 3000 credits + 5000 USD） |
| D4 | Exceed budget error | ✅ Pass | 错误: `Failed to allocate award: exceeds budget [field: credits, budget: 5000.00, used: 3000.00, requested: 3000.00]` |
| D5 | Allocate within budget | ✅ Pass | Flash: `奖励分配已保存。`（Frank: 2000 credits + 4000 USD）；总计 5000/5000 credits |
| D6 | Finalize round | ✅ Pass | Flash: `分配已确定。`，Status: finalized |
| E1 | Distribute first project | ✅ Pass | Flash: `奖励已发放给项目所有者。`，Round 仍为 finalized |
| E2 | Distribute second + auto-transition | ✅ Pass | Flash: `奖励已发放给项目所有者。`，Round 自动 → distributed |
| F1 | Hacker balance correct | ✅ Pass | Balance = 3000 credits ✓，Reference: `grant_round:3/project:3`（Bug #6 修复 ✓，格式正确） |
| F2 | Create redeem option (admin) | ✅ Pass | Flash: `兑换选项已创建。`，"Sticker Pack" 出现在列表 |
| F3 | Redeem option + order created | ✅ Pass | Flash: `兑换成功！`，余额 3000→2900，订单状态 Pending |
| F4 | Insufficient credits error | ✅ Pass | Flash error: `积分不足。` |
| G1 | Admin deposit credits | ✅ Pass | Flash: `积分充值成功。`，Eve: 2900→3400 |
| G2 | Admin deduct credits | ✅ Pass | Flash: `积分扣除成功。`，Eve: 3400→3300 |
| G3 | Fulfill order | ✅ Pass | Flash: `订单已完成。` |
| G4 | Cancel order + refund | ✅ Pass | Flash: `订单已取消，积分已退还。`，Eve: 3200→3300（退款 100 ✓） |
| H | Feed events rendered (not blank) | ✅ Pass | 全部 7 个事件含正确翻译文字；Credits redeemed 事件出现（Bug #4 修复 ✓） |
| I1 | CSV export | ✅ Pass | 下载成功，Status 列显示 `funded`（Bug #5 修复 ✓），RepoID 字段正确（14, 15） |
| J | Cancel round | ✅ Pass | Status → Cancelled ✓；提交时 Flash error: `该资助轮次目前未开放提交。`（Bug #3 修复 ✓） |

**总计：27/27 Pass**（上轮：25/27，进步 2）

---

## Bugs Fixed vs Round 1

| Bug # | 描述 | Round 1 | Round 2 |
|-------|------|---------|---------|
| Bug #1 | Template 500: float64 vs int（manage 页面） | ❌ | ✅ 已修复 |
| Bug #2 | i18n Keys 未翻译（大部分） | ❌ | ✅ 大部分已修复（见下方残留） |
| Bug #3 | 静默拒绝无错误提示（重复提交/向 cancelled 提交） | ❌ | ✅ 已修复 |
| Bug #4 | Credits 兑换事件未出现在 Feed | ❌ | ✅ 已修复 |
| Bug #5 | CSV Status 为数字（非字符串） | ❌ | ✅ 已修复 |
| Bug #6 | Credits 交易 reference 格式错误 | ❌ | ✅ 已修复 |
| Bug #7 | 排序下拉显示错误 i18n key | ❌ | ✅ 已修复 |

---

## Remaining Issues

### Bug #1b（新回归）— projects 页面分页模板 500
- **路径:** `/grants/{slug}/projects`
- **错误:** `template error: builtin(bindata):base/paginate:1:28 : can't evaluate field GetParams in type interface {}`
- **触发位置:** `{{$paginationParams := .Page.GetParams}}`
- **影响:** projects 页面顶部嵌入 500 错误块，但项目列表（title、user、repo、status）仍正确渲染于错误块下方
- **与上轮区别:** 上轮错误为 `gt .AwardAmount 0`（float64 vs int）；此为新引入的分页 nil 指针问题
- **严重度:** Medium（功能仍可用，但 UI 有错误块）
- **修复建议:** 检查 `grants/projects` handler 中传递给模板的 `Page` 对象是否正确初始化 `ListOptions`，确保 `Paginater` 接口实现了 `GetParams()` 方法

### Bug #2b（残留）— 部分 Admin Credits 页面 i18n 未完全翻译
- **路径:** `/admin/credits/options`（options 页面表单标签）
- **表现:** `hackforger.credits.admin.option_create`、`hackforger.credits.admin.option_name` 等标签仍显示为原始 key
- **注:** 其余大多数 HackForger 字符串已翻译，此问题仅限于 admin credits options 页面的部分表单标签
- **严重度:** Low

---

## Dashboard Feed Events (Section H) — Round 2

| Event | Actor | Text (中文) | Status |
|-------|-------|-----------|--------|
| Grant round created | hackforger | 创建了新的资助轮次 | ✅ |
| Grant round opened | hackforger | 开放了资助轮次 | ✅ |
| Project submitted | hacker_eve | 提交了资助申请 | ✅ |
| Round closed for review | hackforger | 关闭了资助轮次 | ✅ |
| Round finalized | hackforger | 完成了资助轮次 | ✅ |
| Project funded (×2) | hackforger | 获得了资助 | ✅ |
| Credits redeemed (×2) | hacker_eve | 兑换了积分 | ✅ **（新增，Bug #4 修复）** |

> **Note:** hacker_frank 的项目提交事件仍未出现（follower/org-member 受众解析暂缓，符合预期）。

---

## CSV Export — Round 2

```csv
ID,Title,UserID,RepoID,Status,AwardAmount,AwardCredits
3,Open Source Widget,3,14,funded,5000.00,3000
4,Another Widget,4,15,funded,4000.00,2000
```

✅ Status 列现为字符串 `funded`（Bug #5 修复）
✅ RepoID 字段正确（14=hacker_eve 的仓库，15=hacker_frank 的仓库）

---

## 与 Round 1 对比总结

| 维度 | Round 1 (101-cf4c171e) | Round 2 (105-85c3160a) |
|------|----------------------|----------------------|
| Pass/Total | 25/27 | **27/27** |
| Bugs Found | 7 | 2（1 回归 + 1 残留） |
| i18n 覆盖 | 极少 | 大部分（admin credits options 还有残留） |
| 500 错误 | manage + projects 页面均有 | 仅 projects 页面有（新回归，分页问题） |
| Feed 完整性 | 缺少 credits redeemed | 所有事件均出现 ✓ |
| CSV Status | 数字（3） | 字符串（funded）✓ |
| 错误提示 | 静默失败 | 有明确 flash 错误 ✓ |
