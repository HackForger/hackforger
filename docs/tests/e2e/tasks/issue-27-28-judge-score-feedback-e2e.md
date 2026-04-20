# Issue #27 + #28 Judge Score Feedback E2E

**目标**：验证评分反馈闭环与 publish 校验同时生效。
**工具**：agent-browser via http://localhost:3000
**前置**：当前 worktree 已 `make frontend && make backend`，并已执行 `sqlite3 data/forgejo.db < scripts/cleanup-invalid-hackathons.sql`。
**账号**：hackforger / admin1234（admin+organizer+judge）

## 测试流程

### TC1: Publish 拒绝无 criteria
1. 以 hackforger 登录，新建 Draft hackathon "E2E评分回归"，添加 1 个 track，不添加 criteria
2. 管理页点发布
3. 断言：顶部 flash 显示红色 "无法发布：至少需要配置一个评分标准"
4. 截图：`tests/screenshots/issue-27-28/tc1-publish-blocked.png`

### TC2: Judging 阶段新增 criteria
1. 接 TC1，在 Draft 添加 1 个 criterion ("创新" 10/25) + 把 phase 推到 Judging（SQL 调 end_time 过期 + sync status_cache）
2. 回管理页断言 criteria 表格仍可见，顶部警告 "评分标准在评审阶段开始后不可修改"
3. 再加一个 criterion ("完成度" 10/20) — 成功
4. 尝试编辑第一个 criterion — 断言 Edit/Delete 按钮被隐藏
5. 截图：`tests/screenshots/issue-27-28/tc2-judging-add-only.png`

### TC3: SubmitScores 成功反馈
1. 在 TC2 活动中提交一条 submission（hackforger 以 hacker 身份）
2. 以 hackforger (自己也是 judge) 访问 `/hackathon/{slug}/judge`
3. 填入分数，点 "Submit Scores"
4. 断言：
   - 顶部绿色 toast "分数已保存" （`showInfoToast` 带 octicon-check）
   - 表单折叠为摘要卡，显示 "保存于 HH:MM"
   - 按钮文案变 "更新分数"
   - Sticky 进度条 1/1
5. 截图：`tests/screenshots/issue-27-28/tc3-success-feedback.png`

### TC4: 刷新后摘要回显
1. 刷新 judge 页
2. 断言：submission 卡片默认折叠为摘要、显示之前填的分数
3. 截图：`tests/screenshots/issue-27-28/tc4-summary-persist.png`

### TC5: 错误场景
1. 使用 `agent-browser network route` 拦截 `/hackathon/{slug}/judge/:sid/scores` 为 500
2. 断言：红色 banner 在顶部显示 "保存分数失败，请重试"；toast 也显示红色错误
3. 恢复路由、重试、成功
4. 截图：`tests/screenshots/issue-27-28/tc5-error-banner.png`

### TC6: Feed 跳转
1. 在 dashboard feed 或 community feed 找到 "hackforger 评审了 E2E评分回归 的提交" 的条目
2. 点击活动名
3. 断言：跳到 `/hackathon/{slug}/leaderboard`（**不是** `/hackathon/{slug}`），看得到聚合分数
4. 截图：`tests/screenshots/issue-27-28/tc6-feed-to-leaderboard.png`

### TC7: Finalize 预览 (#28)
1. 以 hackforger 进 manage 页，点 "预览结果"
2. 断言：显示每个 track 的排名表 + 加权总分（不再是空白）
3. 截图：`tests/screenshots/issue-27-28/tc7-preview-rankings.png`

## 报告模板

报告保存到 `docs/tests/e2e/reports/issue-27-28-judge-score-feedback-report.md`，按 TC 列出 PASS/FAIL + 截图相对路径 + 关键 HTTP 日志片段。

### 字段

```markdown
# Issue #27 + #28 E2E Report

**Run date**: YYYY-MM-DD HH:MM
**Commit**: <short SHA>
**Server**: http://localhost:3000 (worktree build)

| TC | Description | Result | Screenshot |
|----|-------------|--------|------------|
| TC1 | Publish blocked without criteria | PASS/FAIL | tests/screenshots/issue-27-28/tc1-publish-blocked.png |
| TC2 | Judging add-only | PASS/FAIL | tests/screenshots/issue-27-28/tc2-judging-add-only.png |
| TC3 | Submit success feedback | PASS/FAIL | tests/screenshots/issue-27-28/tc3-success-feedback.png |
| TC4 | Summary persists after refresh | PASS/FAIL | tests/screenshots/issue-27-28/tc4-summary-persist.png |
| TC5 | Error banner on failure | PASS/FAIL | tests/screenshots/issue-27-28/tc5-error-banner.png |
| TC6 | Feed → leaderboard | PASS/FAIL | tests/screenshots/issue-27-28/tc6-feed-to-leaderboard.png |
| TC7 | Preview rankings | PASS/FAIL | tests/screenshots/issue-27-28/tc7-preview-rankings.png |

## Notes
<observations per TC — network failures, console errors, DB manipulations performed>

## Regression check
<did any unrelated page regress? List pages visited.>

## Conclusion
<ready-to-merge / needs-rework + specific blockers>
```
