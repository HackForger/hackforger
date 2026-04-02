# 跨阶段验证 + 积分对账 + 报告模版

> **前置条件**: 完成所有前置文件（`00-setup.md` 至 `07-edge-cases.md`）

本文件负责跨阶段数据一致性验证、积分对账、Feed 事件完整性检查，以及最终报告的整合。
环境/账号/Session 信息请参考 [00-setup.md](./00-setup.md)。

---

## 跨阶段验证

以下检查点验证各阶段之间的数据一致性。

### 积分一致性对账

在所有阶段完成后，每个用户导航到 `/credits` 查看余额，核对交易历史。

| 用户 | 积分来源 | 积分去向 |
|------|---------|---------|
| hacker_eve | Admin 充值（Phase 4）、Hackathon 奖金（Phase 7）、Grant 资助（Phase 5）、Competitive Bounty 1st（Phase 10c） | Bounty escrow（Phase 4）、兑换算力（Phase 8） |
| hacker_frank | Bounty 奖励（Phase 4）、Hackathon 奖金（Phase 7）、Admin 充值（Phase 8）、Competitive Bounty 2nd（Phase 10c） | Admin 扣除（Phase 8） |
| hackforger (organizer) | 多次 Admin 充值（Phase 0, 5, 10） | Hackathon 奖金发放、Grant 发放、Bounty escrow |

- **验证**: 每个用户的余额与预期一致，交易记录完整无遗漏
- **截图**: `screenshots/full-cycle/cross-01-credits-reconcile.png`

### Feed 事件完整性验证

在 hackforger session 的 Dashboard Global Feed 中验证。

| 事件类型 | 来源阶段 | 预期存在 |
|---------|---------|---------|
| `hackathon_created(30)` | Phase 1 | 是 |
| `hackathon_phase_changed(50)` | Phase 1, 3, 7 | 是（publish, start, judge） |
| `hackathon_registered(31)` | Phase 2 | 是（x2） |
| `hackathon_submitted(32)` | Phase 6 | 是（x2） |
| `hackathon_scored(33)` | Phase 7 | 是（x4） |
| `hackathon_finalized(51)` | Phase 7 | 是 |
| `bounty_created(34)` | Phase 4, 10 | 是（多个）|
| `bounty_claimed(35)` | Phase 4 | 是 |
| `bounty_delivered(36)` | Phase 4 | 是 |
| `bounty_completed(37)` | Phase 4 | 是 |
| `bounty_paid(43)` | Phase 4, 10c | 是 |
| `bounty_winners_selected(38)` | Phase 10c | 是 |
| `bounty_expired(52)` | Phase 10a | 是 |
| `bounty_cancelled(53)` | Phase 10b | 是 |
| `grant_round_created(39)` | Phase 5 | 是 |
| `grant_round_opened(54)` | Phase 5 | 是 |
| `grant_project_submitted(40)` | Phase 5 | 是（x2）|
| `grant_round_closed(55)` | Phase 5 | 是 |
| `grant_awarded(41)` | Phase 5 | 是 |
| `grant_round_finalized(56)` | Phase 5 | 是 |
| `credits_redeemed(42)` | Phase 8 | 是 |
| `order_fulfilled(58)` | Phase 8 | 是 |

- **验证**: 以上事件均出现在 Global Feed 中
- **截图**: `screenshots/full-cycle/cross-02-feed-events.png`

---

## 报告模版

测试完成后，按以下格式生成报告，保存到 `docs/tests/e2e/reports/user-journey-full-cycle-report.md`：

```markdown
# 用户旅程全流程测试报告

> **日期**: YYYY-MM-DD
> **服务器**: http://localhost:3000
> **测试者**: (执行者名称)
> **代码版本**: (git commit hash)

## 概要

| 阶段 | 步骤数 | 通过 | 失败 | 跳过 | 状态 |
|------|--------|------|------|------|------|
| Phase 0: 平台准备 | 3 | ? | ? | ? | ? |
| Phase 1: Hackathon 创建 | 5 | ? | ? | ? | ? |
| Phase 2: 报名 + 社交 | 6 | ? | ? | ? | ? |
| Phase 3: Hacking 阶段 | 2 | ? | ? | ? | ? |
| Phase 4: 开发 + Bounty 协作 | 10 | ? | ? | ? | ? |
| Phase 5: Grant 资助 | 9 | ? | ? | ? | ? |
| Phase 6: 提交参赛作品 | 5 | ? | ? | ? | ? |
| Phase 7: 评审 + Finalize | 8 | ? | ? | ? | ? |
| Phase 8: Credits 兑换 | 7 | ? | ? | ? | ? |
| Phase 9: 社交 + Feed + 搜索 | 9 | ? | ? | ? | ? |
| Phase 10: 异常路径 + 约束验证 | 11 | ? | ? | ? | ? |
| 跨阶段验证 | 2 | ? | ? | ? | ? |
| **总计** | **77** | | | | |

## 逐步结果

### Phase 0: 平台准备

#### Step 0.1: Admin 配置兑换选项 -- [PASS/FAIL]
- 操作描述: ...
- 截图: `screenshots/full-cycle/p0-01-redeem-option-created.png`
- 备注: (如有异常或额外观察)

#### Step 0.2: Admin 添加 Key Pool -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p0-02-keys-added.png`

#### Step 0.3: Admin 为 Organizer 充值积分 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p0-03-credits-deposit-1000.png`

### Phase 1: Hackathon 创建

#### Step 1.1: Organizer 创建 Hackathon -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-01-hackathon-created.png`

#### Step 1.2: Organizer 创建赛道 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-02-tracks-created.png`

#### Step 1.3: Organizer 设置评审标准 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-03-criteria-set.png`

#### Step 1.4: Organizer 指派评委 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-04-judges-assigned.png`

#### Step 1.5: Organizer 发布 Hackathon -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p1-05-published.png`

### Phase 2: 报名 + 社交

#### Step 2.1: Hacker1 关注 Organizer -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-01-follow-organizer.png`

#### Step 2.2: Hacker1 报名 Web Track -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-02-hacker1-registered.png`

#### Step 2.3: Hacker2 报名 DeFi Track -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-03-hacker2-registered.png`

#### Step 2.4: Hacker1 与 Hacker2 互相关注 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-04-mutual-follow.png`

#### Step 2.5: 组队 -- 在 Hackathon Org 内创建 Team -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-05-team-created.png`

#### Step 2.6: [约束] Judge 尝试报名 -> 拒绝 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p2-06-judge-register-denied.png`

### Phase 3: Hacking 阶段

#### Step 3.1: Organizer 启动 Hacking -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p3-01-hacking-started.png`

#### Step 3.2: [约束] 重复发布 -> 拒绝 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p3-02-duplicate-publish-denied.png`

### Phase 4: 开发 + Bounty 协作

#### Step 4.1: Hacker1 Fork Web Track Repo -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-01-hacker1-fork-web.png`

#### Step 4.2: Hacker1 创建 Issue -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-02-issue-created.png`

#### Step 4.3: Hacker1 创建 Exclusive Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-03-bounty-created.png`

#### Step 4.4: Admin 为 Hacker1 充值 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-04-hacker1-deposit.png`

#### Step 4.5: Hacker2 浏览 Bounty 并申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-05-bounty-application.png`

#### Step 4.6: Hacker1 接受申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-06-application-accepted.png`

#### Step 4.7: Hacker2 Fork 并提交 UI 设计 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-07-hacker2-fork-design.png`

#### Step 4.8: Hacker2 创建 PR -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-08-bounty-pr.png`

#### Step 4.9: Hacker1 Complete + Pay Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-09-bounty-completed-paid.png`

#### Step 4.10: Hacker2 查看积分余额变化 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p4-10-hacker2-credits.png`

### Phase 5: Grant 资助

#### Step 5.1: Admin 为 Organizer 充值 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-01-grant-budget-deposit.png`

#### Step 5.2: Organizer 创建 Grant Round -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-02-grant-round-created.png`

#### Step 5.3: Organizer 开放申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-03-grant-opened.png`

#### Step 5.4: Hacker1 提交 Grant 申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-04-project1-submitted.png`

#### Step 5.5: Hacker2 提交 Grant 申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-05-project2-submitted.png`

#### Step 5.6: Organizer 关闭申请 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-06-grant-closed.png`

#### Step 5.7: Organizer 审批项目 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-07-projects-reviewed.png`

#### Step 5.8: Organizer Finalize + Distribute -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-08-grant-distributed.png`

#### Step 5.9: Hacker1 查看积分变化 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p5-09-hacker1-credits-grant.png`

### Phase 6: 提交参赛作品

#### Step 6.1: Hacker1 继续开发 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-01-hacker1-develop.png`

#### Step 6.2: Hacker1 Fork+PR 模式提交 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-02-hacker1-submission-pr.png`

#### Step 6.3: Hacker2 Fork DeFi Track 并开发 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-03-hacker2-fork-defi.png`

#### Step 6.4: Hacker2 Link Repo 模式提交 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-04-hacker2-submission-link.png`

#### Step 6.5: 多用户 Star 赛道 Repo -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p6-05-star-repos.png`

### Phase 7: 评审 + Finalize

#### Step 7.1: Organizer 启动评审 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-01-judging-started.png`

#### Step 7.2: Judge1 评分 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-02-judge1-scores.png`

#### Step 7.3: Judge2 评分 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-03-judge2-scores.png`

#### Step 7.4: Organizer 查看排名预览 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-04-finalize-preview.png`

#### Step 7.5: Organizer 确认结算 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-05-finalized.png`

#### Step 7.6: Hacker1 查看积分余额 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-06-hacker1-credits-hackathon.png`

#### Step 7.7: Hacker2 查看积分余额 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-07-hacker2-credits-hackathon.png`

#### Step 7.8: 查看排行榜 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p7-08-leaderboard.png`

### Phase 8: Credits 兑换

#### Step 8.1: Hacker1 查看积分总览 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-01-credits-overview.png`

#### Step 8.2: 浏览兑换选项 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-02-redeem-options.png`

#### Step 8.3: 兑换算力 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-03-redeem-confirmed.png`

#### Step 8.4: Admin Fulfill 订单 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-04-order-fulfilled.png`

#### Step 8.5: 查看订单和密钥 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-05-order-detail.png`

#### Step 8.6: Admin 手动充值 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-06-admin-deposit-hacker2.png`

#### Step 8.7: Admin 手动扣除 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p8-07-admin-deduct.png`

### Phase 9: 社交 + Feed + 搜索

#### Step 9.1: Dashboard Feed (Following) -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-01-dashboard-feed.png`

#### Step 9.2: Global Feed -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-02-global-feed.png`

#### Step 9.3: Explore Hackathons -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-03-explore-hackathons.png`

#### Step 9.4: Explore Bounties -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-04-explore-bounties.png`

#### Step 9.5: Explore Grants -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-05-explore-grants.png`

#### Step 9.6: Explore Submissions -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-06-explore-submissions.png`

#### Step 9.7: Cmd+K 全局搜索 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-07-search-cmdk.png`

#### Step 9.8: Reputation 排行榜 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-08-reputation-leaderboard.png`

#### Step 9.9: Reaction 互动 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p9-09-reaction.png`

### Phase 10: 异常路径 + 约束验证

#### Step 10a.1: 创建即将过期的 Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-01-bounty-expiry-setup.png`

#### Step 10a.2: 触发过期 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-02-bounty-expired.png`

#### Step 10b.1: 创建并取消 Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-03-bounty-cancelled.png`

#### Step 10c.1: 创建 Competitive Bounty -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-04-competitive-bounty.png`

#### Step 10c.2: 多人提交方案 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-05-competitive-applications.png`

#### Step 10c.3: 选择 Winner 并支付 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-06-competitive-paid.png`

#### Step 10d.1: [约束] 参赛者不能当评委 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-07-participant-as-judge-denied.png`

#### Step 10d.2: [约束] 重复报名 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-08-duplicate-register-denied.png`

#### Step 10d.3: [约束] Grant 超预算 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-09-grant-over-budget.png`

#### Step 10d.4: [约束] 非 Owner 管理 -> 403 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-10-non-owner-403.png`

#### Step 10d.5: [约束] Bounty 申请被拒绝 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/p10-11-application-rejected.png`

### 跨阶段验证

#### 积分一致性对账 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/cross-01-credits-reconcile.png`

#### Feed 事件完整性 -- [PASS/FAIL]
- 截图: `screenshots/full-cycle/cross-02-feed-events.png`

## 积分对账表

| 用户 | 充值/收入 | 扣除/支出 | 预期余额 | 实际余额 | 一致? |
|------|---------|---------|---------|---------|------|
| hacker_eve | (逐项列出) | (逐项列出) | ? | ? | ? |
| hacker_frank | (逐项列出) | (逐项列出) | ? | ? | ? |
| hackforger | (逐项列出) | (逐项列出) | ? | ? | ? |

## 发现的 Bug

| # | 阶段 | 步骤 | 描述 | 严重程度 | 截图 | 修复状态 |
|---|------|------|------|---------|------|---------|
| 1 | | | | P0/P1/P2 | | Open/Fixed |

## 结论

(总结通过率、关键发现、需要修复的问题、对 v0.1 发布的影响评估)
```

---

## 本阶段结果

| 步骤 | 状态 | 截图 |
|------|------|------|
| 积分一致性对账 | PASS/FAIL | screenshots/full-cycle/cross-01-credits-reconcile.png |
| Feed 事件完整性 | PASS/FAIL | screenshots/full-cycle/cross-02-feed-events.png |
| 最终报告生成 | PASS/FAIL | docs/tests/e2e/reports/user-journey-full-cycle-report.md |
