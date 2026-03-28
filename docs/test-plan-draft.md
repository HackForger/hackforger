# HackForger 测试计划

---

## 〇、背景

HackForger 是 Forgejo 的 Fork，新增 Hackathon、Bounty、Grant、Credits、Feed 五个模块（15 张新表，~10,000 行新代码）。详细设计见 `implementation-plan-draft.md`。

### 测试策略

本项目的测试有两个特殊约束：

**1. 测试数据通过系统 API 渐进创建，不使用 mock 数据。**

原因：HackForger 的功能高度交叉耦合——Bounty 完成后积分入账，积分用于兑换，声誉基于积分/Bounty/Hackathon 综合计算。Mock 数据无法覆盖这些跨模块的真实链路。因此，E2E 测试从零开始，用 API 创建用户 → 创建活动 → 报名 → 提交 → 评审 → 完成 → 积分入账 → 兑换，每一步的产出是下一步的输入。

**2. 每个用户操作后立即验证 Feed 事件。**

原因：Feed 系统通过 `HackForgerNotifier` 在状态变更时写入 Forgejo 原有的 `action` 表。如果某个 Notifier 方法忘了写入事件，用户的 Timeline 就会缺少事件。在 E2E 中每步操作后立即检查 Feed，比最后集中验证更容易定位问题。

### 测试与开发阶段的对应关系

```
Week 1 (P0 基础设施)  → Phase 0 E2E: 用户创建 + Follow + API 可达性
Week 2-3 (三线并行)    → Phase 1 E2E: Bounty + Hackathon + Grant 基础流程
Week 4 (评审+声誉+兑换) → Phase 2 E2E: 评审 + 声誉 + Feed Dashboard + 积分兑换
Week 5 (AI 助手+收尾)  → Phase 3 E2E: 搜索 + AI + Agent 全流程 + 最终验证
```

### 核心原则

1. 测试数据通过 API 渐进创建，不 mock
2. 每个用户操作后立即验证对应的 Feed 事件
3. E2E 从用户注册开始，到兑换奖励结束
4. 最终验证 Feed 的完整性、时间顺序和受众分发正确性

---

## 一、测试角色

| 账号 | 角色 |
|------|------|
| `admin` | 平台管理员 |
| `org_alice` | 活动组织者 A（Hackathon, Grant） |
| `org_bob` | 活动组织者 B（Bounty） |
| `judge_carol` | 评委 |
| `judge_dave` | 评委 |
| `hacker_eve` | 参赛者 |
| `hacker_frank` | 参赛者 |
| `hacker_grace` | 参赛者 |
| `agent_hunter` | Hacker Agent |
| `agent_judge` | Judge Agent |

---

## 二、单元测试

### 2.1 Hackathon 模块

```
Model 层 — 文件: models/hackforger/hackathon_test.go

TestCreateHackathon                   — 正常创建，验证 ID/Slug/Status/CreatedUnix
TestCreateHackathon_DuplicateSlug     — 重复 slug 返回 ErrHackathonSlugExists
TestGetHackathonByID                  — 读取正确
TestGetHackathonBySlug                — 读取正确
TestListHackathons_FilterByStatus     — 按状态筛选
TestCreateHackathonTrack              — 赛道创建，关联正确
TestCreateHackathonSubmission         — 提交创建，初始 Status=Draft, TotalScore=0
TestCreateJudgeScore                  — 评分写入
TestCreateJudgeScore_Duplicate        — 同评委同提交重复打分 → ErrDuplicateScore

Service 层 — 文件: services/hackforger/hackathon_test.go

TestHackathonStatusTransition_Valid   — Draft→Registration 正常
TestHackathonStatusTransition_Invalid — Draft→Judging 跳步 → ErrInvalidStatusTransition
TestHackathonAutoTransition           — RegistrationEnd 已过 → 自动切 Hacking
TestRegisterForHackathon              — 创建报名记录
TestApproveRegistration               — Approved + 自动创建 Team + 成员加入
TestSubmitToHackathon                 — 创建 Submission
TestSubmitToHackathon_AfterDeadline   — 截止后提交 → ErrSubmissionClosed
TestSubmitScore                       — 评分 + TotalScore 自动更新
TestSubmitScore_NotAJudge             — 非评委 → ErrNotAJudge
TestLeaderboard                       — 按 TotalScore 降序
TestLeaderboard_ByTrack               — 赛道筛选
TestFinalizeHackathon                 — Finished + credits 发放
```

### 2.2 Bounty 模块

```
Model 层 — 文件: models/hackforger/bounty_test.go

TestCreateBounty                      — 正常创建，Status=Open
TestCreateBounty_DuplicateIssue       — 同 Issue 重复绑定 → ErrBountyAlreadyExists
TestCreateBountyReward                — Reward 关联正确
TestCreateBountyReward_Multiple       — 同一 Bounty 多种 Reward
TestCreateBountyApplication           — 申请，Status=Pending
TestCreateBountyApplication_Duplicate — 重复申请 → ErrAlreadyApplied
TestGetBountyByIssueID                — 查询正确
TestListBounties_Filter               — 按状态/金额筛选
TestCreateBountyWinner                — competitive 获奖记录

Service 层 — 文件: services/hackforger/bounty_test.go

TestExclusiveBountyFlow_Happy         — 完整流程: create→claim→PR→merge→review→complete→积分
TestExclusiveBounty_RejectAndRedo     — 拒绝交付→退回 Claimed
TestCompetitiveBountyFlow_Happy       — 多人提交→选 Winners→按 rank 发积分
TestBountyApplication_AcceptRejectsOthers — 接受一人→自动拒绝其他
TestBountyExpiry                      — 超期自动 Expired
TestBountyCancel                      — Publisher 取消
TestBountyCancel_NotPublisher         — 非 Publisher 取消 → ErrNotPublisher
TestBountyPRMerge_WrongUser           — 非 Claimer 的 PR merge → 状态不变
TestBountyComplete_NoCreditsReward    — 无 credits Reward → 余额不变
TestBountyLeaderboard                 — Hunter 排行
```

### 2.3 Grant 模块

```
文件: models/hackforger/grants_test.go + services/hackforger/grants_test.go

TestCreateGrantRound                  — Status=Setup
TestCreateGrantProject                — Status=Pending, AwardAmount=0
TestGrantRoundStatusTransition        — Setup→Open→Reviewing→Finalized→Distributed
TestSubmitProjectToGrant              — 正常提交
TestSubmitProject_RoundNotOpen        — 非 Open 状态 → ErrRoundNotOpen
TestApproveProject                    — Status=Approved
TestAllocateAward                     — AwardAmount + AwardCredits 设置
TestAllocateAward_ExceedsBudget       — 超预算 → ErrExceedsBudget
TestFinalizeRound                     — Finalized + 积分发放
TestFinalizeRound_UnallocatedProjects — 未分配项目存在 → ErrUnallocatedProjects
TestGrantProjectStarCount             — Star 数通过 Repo API 正确读取
```

### 2.4 Credits 模块

```
文件: models/hackforger/credits_test.go + services/hackforger/credits_test.go

TestGetOrCreateCreditAccount          — 首次创建 Balance=0，重复返回同一记录
TestDeposit                           — +500, Balance 正确, Transaction 正确
TestDeposit_Multiple                  — 累加 Balance，Transaction.Balance 快照递增
TestRedeem_Success                    — 扣款 + Order 创建
TestRedeem_InsufficientBalance        — 余额不足 → ErrInsufficientCredits + 余额不变
TestRedeem_OutOfStock                 — 无库存 → ErrOutOfStock
TestRedeem_DecreasesStock             — Stock 扣减
TestRedeem_UnlimitedStock             — Stock=-1 不变
TestFulfillOrder                      — Status=fulfilled + note 正确
TestRefundOrder                       — Status=refunded + 余额退回
TestDeposit_Concurrent                — 10 goroutine 并发 Deposit → 最终 Balance 正确
```

### 2.5 Reputation 模块

```
文件: services/hackforger/reputation_test.go

TestRecalculateReputation             — 各字段正确，Score 按权重计算
TestReputationLeaderboard             — 按 Score 降序
TestRecalculateAll                    — 所有用户更新
```

### 2.6 Feed 模块（新增）

```
文件: services/hackforger/notifier_test.go

=== HackForgerNotifier 事件分发测试 ===
TestNotifier_AudienceFollowers
  前置: frank follows eve
  触发: Notifier 事件分发(ActUserID:eve, AudienceType:Followers, ...)
  断言: action 表中 UserID=frank 的记录存在
  断言: action 表中 UserID=grace（未 follow）的记录不存在

TestNotifier_AudienceOrgMembers
  前置: eve 和 frank 是 org 成员，grace 不是
  触发: Notifier 事件分发(AudienceType:OrgMembers, OrgID:org.ID, ...)
  断言: eve 和 frank 各有一条记录，grace 没有

TestNotifier_AudienceGlobal
  触发: Notifier 事件分发(AudienceType:Global, ...)
  断言: action 表中存在 UserID=0 的记录

TestNotifier_AudienceRepoWatchers
  前置: eve watch 了 repo，frank 没有
  触发: Notifier 事件分发(AudienceType:RepoWatchers, RepoID:repo.ID, ...)
  断言: eve 有记录，frank 没有

TestNotifier_ContentSerialization
  触发: Notifier 事件分发，传入 HackforgerActionContent
  断言: action.Content 是合法 JSON
  断言: 反序列化后字段一致

TestNotifier_PhaseContentSerialization
  触发: Notifier 事件分发，传入 HackforgerPhaseContent（含 OldStatus, NewStatus, WinnerName）
  断言: 反序列化后字段一致

文件: services/hackforger/feed_test.go
=== Feed 查询测试 ===
TestGetFollowingFeed
  前置: frank follows eve, eve 有多条 action, 还有几条全局事件
  调用: GetFollowingFeed(ctx, frank.ID, ...)
  断言: 结果包含 eve 的 action + 全局事件
  断言: 不包含 grace 的 action（frank 没 follow grace）
  断言: 按 created_unix 降序

TestGetFollowingFeed_Empty
  前置: 新用户没 follow 任何人
  调用: GetFollowingFeed
  断言: 只返回全局事件

TestGetEntityFeed
  前置: 某 Hackathon 有 created + phase_changed + registered + submitted 事件
  调用: GetEntityFeed(ctx, "hackathon", hackathonID, ...)
  断言: 返回该 Hackathon 所有相关事件
  断言: 不包含其他 Hackathon 或 Bounty 的事件

TestGetEntityFeed_Bounty
  前置: 某 Bounty 有 created + claimed + completed 事件
  调用: GetEntityFeed(ctx, "bounty", bountyID, ...)
  断言: 返回该 Bounty 所有相关事件

TestFeedAPI_TypeFilter
  调用: GET /api/v1/hackforger/feed?type=global
  断言: 只返回全局事件
  调用: GET /api/v1/hackforger/feed?type=following
  断言: 返回关注动态 + 全局事件

TestFeedAPI_Pagination
  前置: 大量 action 记录
  调用: GET /feed?type=global&page=1&limit=5
  断言: items.length == 5, total_count > 5
```

### 2.7 Search & Assistant 模块

```
文件: services/hackforger/search_test.go

TestUnifiedSearch_Bounties            — 关键词搜索命中 Bounty
TestUnifiedSearch_Hackathons          — 关键词搜索命中 Hackathon
TestUnifiedSearch_NoResults           — 无结果
TestUnifiedSearch_ScopeFilter         — scope 限制

文件: services/hackforger/assistant_test.go

TestBuildSystemPrompt                 — Prompt 包含用户余额/声誉/平台数据
TestExtractActions                    — 搜索结果转 Action 按钮
```

---

## 三、E2E 测试

### 测试框架说明

- **Go 集成测试**：使用 `tests/integration/` 框架，复用已有的 `PrepareTestEnv()`、`loginUser()` 等基础设施
- **前端 E2E**：使用 Playwright（`tests/e2e/`）
- **Test Fixtures**：需在 `models/fixtures/` 中为 hackforger 模型创建 YAML fixture 数据
- **数据库**：当前阶段仅需保证 SQLite 通过（`make test-sqlite`）

### 3.0 Feed 验证辅助函数

```go
// tests/integration/hackforger_e2e_helpers.go

// assertFeedContains 验证某用户的 feed 中包含指定事件
func (e *E2EContext) assertFeedContains(t *testing.T, username string,
    opType int, entityName string, feedType string) {

    resp := e.API(username).Get("/api/v1/hackforger/feed?type=" + feedType + "&limit=50")
    assert.Equal(t, 200, resp.StatusCode)

    var result FeedResponse
    json.Unmarshal(resp.Body, &result)

    found := false
    for _, item := range result.Items {
        if item.OpType == opType && item.Entity.Name == entityName {
            found = true
            break
        }
    }
    assert.True(t, found,
        "expected feed of %s (%s) to contain op_type=%d entity=%s",
        username, feedType, opType, entityName)
}

// assertFeedNotContains 验证某用户的 feed 中不包含指定事件
func (e *E2EContext) assertFeedNotContains(t *testing.T, username string,
    opType int, entityName string, feedType string) {
    // 同上，断言 found == false
}

// assertGlobalFeedContains 验证全局 feed 中包含指定事件
func (e *E2EContext) assertGlobalFeedContains(t *testing.T, opType int, entityName string) {
    e.assertFeedContains(t, "hacker_eve", opType, entityName, "global")
    // 任何用户查全局 feed 都应该能看到
}

// assertEntityFeedContains 验证某实体的 feed 中包含指定事件
func (e *E2EContext) assertEntityFeedContains(t *testing.T,
    entityType string, entityID int64, opType int) {

    resp := e.API("admin").Get(fmt.Sprintf(
        "/api/v1/hackforger/feed?type=%s&entity_id=%d&limit=50", entityType, entityID))
    // ...断言...
}
```

### 3.1 Phase 0 — 基础设施（Week 1 后）

```
TestE2E_Phase0_UserSetup

  步骤 1: 创建全部 10 个用户 + PAT

  步骤 2: 建立 Follow 关系
    hacker_eve:   follow org_alice, org_bob
    hacker_frank: follow org_alice, hacker_eve
    hacker_grace: follow hacker_eve, hacker_frank
    agent_hunter: follow org_bob
    断言: 每个 follow 关系通过 GET /user/following/{name} 验证

  步骤 3: 创建组织
    org_alice → "spring-hack-org", "grants-org"
    org_bob   → "acme-dev"

  步骤 4: 将相关用户加入 Org
    hacker_eve, hacker_frank, hacker_grace 加入 spring-hack-org（成为成员）
    断言: GET /orgs/spring-hack-org/members 包含三人

  步骤 5: 创建仓库
    spring-hack-org/project-template (is_template:true)
    acme-dev/backend + 2 个 Issue

  步骤 6: Watch 设置
    hacker_eve:   watch acme-dev/backend
    hacker_frank: watch acme-dev/backend
    断言: watchers 列表正确

  步骤 7: 验证 HackForger API + Feed API 可用
    GET /api/v1/hackforger/hackathons → 200, 空
    GET /api/v1/hackforger/feed?type=global → 200, items 可能包含 Forgejo 原生事件
```

### 3.2 Phase 1 — Bounty + Hackathon + Grants（Week 2-3 后）

```
TestE2E_Phase1_BountyExclusive

  步骤 1: org_bob 创建 Exclusive Bounty
    POST /repos/acme-dev/backend/bounties
    保存 bounty_1_id

  步骤 2: 添加 Rewards（money + credits）

  步骤 3: 验证 Feed — Bounty 创建事件
    全局 feed 应包含: "org_bob 在 acme-dev/backend 发布了 Bounty"
      → assertGlobalFeedContains(t, 34, "重构认证模块")
    hacker_eve（watch 了 repo）的 following feed 应包含
      → e.assertFeedContains(t, "hacker_eve", 34, "重构认证模块", "following")
    agent_hunter（follow 了 org_bob）的 following feed 应包含
      → e.assertFeedContains(t, "agent_hunter", 34, "重构认证模块", "following")
    hacker_grace（既没 watch 也没 follow bob）的 following feed 应不包含
      → e.assertFeedNotContains(t, "hacker_grace", 34, "重构认证模块", "following")
      （但 grace 查 global feed 应该能看到，因为 Bounty 发布是全局事件）

  步骤 4: hacker_eve 申请认领

  步骤 5: agent_hunter 申请认领

  步骤 6: org_bob 接受 eve 的申请

  步骤 7: 验证 Feed — Bounty 认领事件
    hacker_frank（follow eve）的 following feed:
      → assertFeedContains(t, "hacker_frank", 35, "重构认证模块", "following")
    hacker_grace（follow eve）的 following feed:
      → assertFeedContains(t, "hacker_grace", 35, "重构认证模块", "following")

  步骤 8: eve 创建 PR + org_bob merge

  步骤 9: 验证 Feed — Bounty 进入审核
    → assertEntityFeedContains(t, "bounty", bounty_1_id, 36) // delivered

  步骤 10: org_bob 验收通过

  步骤 11: 验证积分 + Feed — Bounty 完成事件
    GET /credits/balance (eve) → 500
    assertFeedContains(t, "hacker_frank", 37, "重构认证模块", "following")
    // frank follows eve，应看到 "eve 完成了 Bounty"

  步骤 12: org_bob 标记已支付

  步骤 13: 验证 Bounty entity feed 完整性
    GET /api/v1/hackforger/feed?type=bounty&entity_id={bounty_1_id}
    断言: 按时间序包含 created → claimed → delivered → completed → paid 全链路


TestE2E_Phase1_BountyCompetitive

  步骤 1: org_bob 创建 Competitive Bounty + 分级 Rewards

  步骤 2: 验证 Feed — 创建事件（全局）

  步骤 3: eve, frank, grace 各自提交 PR

  步骤 4: org_bob 选出 Winners

  步骤 5: 验证 Feed — Winners 公布事件
    assertGlobalFeedContains(t, 38, "最佳 CLI 工具挑战") // winners_selected 是全局

  步骤 6: 验证积分
    eve: 500 + 1000 = 1500
    frank: 500
    grace: 200


TestE2E_Phase1_HackathonCreation

  步骤 1: org_alice 创建 Hackathon

  步骤 2: 创建赛道

  步骤 3: 发布 Hackathon（Draft → Registration）

  步骤 4: 验证 Feed — Hackathon 创建 + 阶段切换
    全局: "org_alice 发布了新的 Hackathon「Spring Hack 2026」"
      → assertGlobalFeedContains(t, 30, "Spring Hack 2026")
    全局: "「Spring Hack 2026」开始接受报名"
      → assertGlobalFeedContains(t, 50, "Spring Hack 2026") // phase_changed
    frank（follow alice）的 following feed:
      → assertFeedContains(t, "hacker_frank", 30, "Spring Hack 2026", "following")

  步骤 5: hacker_eve 报名

  步骤 6: 验证 Feed — 报名事件
    frank（follow eve）:
      → assertFeedContains(t, "hacker_frank", 31, "Spring Hack 2026", "following")
    grace（follow eve）:
      → assertFeedContains(t, "hacker_grace", 31, "Spring Hack 2026", "following")

  步骤 7: hacker_frank + hacker_grace 报名

  步骤 8: org_alice 审批全部报名
    验证 Team 创建 + Repo Fork

  步骤 9: 切换到 Hacking

  步骤 10: 验证 Feed — 阶段切换
    Org 成员（eve, frank, grace）的 feed:
      → assertFeedContains(t, "hacker_eve", 50, "Spring Hack 2026", "following")
      内容应包含 StatusLabel: "进入 Hacking 阶段"

  步骤 11: eve 提交参赛作品

  步骤 12: 验证 Feed — 提交事件
    frank（follow eve）:
      → assertFeedContains(t, "hacker_frank", 32, "Spring Hack 2026", "following")

  步骤 13: frank 提交参赛作品

  步骤 14: 验证 Hackathon entity feed 完整性
    GET /api/v1/hackforger/feed?type=hackathon&entity_id={hackathon_id}
    断言: 包含 created → phase(registration) → registered(eve) → registered(frank) →
          registered(grace) → phase(hacking) → submitted(eve) → submitted(frank)


TestE2E_Phase1_GrantRound

  步骤 1: org_alice 创建 Grant Round

  步骤 2: 验证 Feed — Round 创建
    assertGlobalFeedContains(t, 39, "Open Source Fund Q2")

  步骤 3: 开启 Round

  步骤 4: 验证 Feed — Round 开启
    assertGlobalFeedContains(t, 54, "Open Source Fund Q2") // round_opened

  步骤 5: eve 提交项目申请

  步骤 6: 验证 Feed — 项目申请
    frank（follow eve）:
      → assertFeedContains(t, "hacker_frank", 40, "Open Source Fund Q2", "following")

  步骤 7: frank 提交申请

  步骤 8: grace Star eve 的 Repo

  步骤 9: 关闭申请期

  步骤 10: 验证 Feed — 申请关闭
    assertGlobalFeedContains(t, 55, "Open Source Fund Q2")

  步骤 11: 审批 + 分配金额

  步骤 12: Finalize Round

  步骤 13: 验证 Feed — 结果公布
    assertGlobalFeedContains(t, 56, "Open Source Fund Q2")

  步骤 14: 验证积分
    eve: 1500 + 300 = 1800
    frank: 500 + 200 = 700

  步骤 15: 验证 Grant entity feed 完整性
    GET /api/v1/hackforger/feed?type=grant&entity_id={round_id}
    断言: created → opened → project_submitted(eve) → project_submitted(frank) →
          closed → awarded(eve) → awarded(frank) → finalized
```

### 3.3 Phase 2 — 评审 + 声誉 + 兑换 + Feed UI（Week 4 后）

```
TestE2E_Phase2_Judging

  步骤 1: 添加评委

  步骤 2: 切换到 Judging

  步骤 3: 验证 Feed — 阶段切换
    Org 成员: assertFeedContains(t, "hacker_eve", 50, "Spring Hack 2026", "following")
    内容 StatusLabel: "进入评审阶段"

  步骤 4: judge_carol 打分（eve 的提交 + fg 的提交）

  步骤 5: 验证 Feed — 评审事件
    Org 成员: assertFeedContains(t, "hacker_eve", 33, "Spring Hack 2026", "following")

  步骤 6: judge_dave 打分

  步骤 7: agent_judge 通过 API 提交 AI Review

  步骤 8: 验证排行榜

  步骤 9: Finalize Hackathon

  步骤 10: 验证 Feed — 结果公布（全局）
    assertGlobalFeedContains(t, 51, "Spring Hack 2026")
    // 内容应包含 WinnerName

  步骤 11: 验证积分
    eve: 1800 + 2000(prize) = 3800

  步骤 12: 验证 Hackathon 完整 entity feed
    GET /api/v1/hackforger/feed?type=hackathon&entity_id={hackathon_id}
    断言: 包含完整生命周期（约 12-15 条事件，从创建到结果公布）


TestE2E_Phase2_FeedDashboard

  这个测试专门验证 Dashboard Following Feed 的正确性。

  步骤 1: 验证 hacker_frank 的 following feed 内容
    frank follows: org_alice, hacker_eve
    frank 是 spring-hack-org 成员
    frank watch acme-dev/backend

    GET /api/v1/hackforger/feed?type=following (as frank)
    断言包含（按时间倒序，这些都应该出现）:
      - Hackathon 结果公布（全局）
      - judge 打分事件（Org 成员）
      - Hackathon 阶段切换（Org 成员）
      - Grant Round Finalized（全局）
      - eve 提交 Grant 申请（follow eve）
      - Grant Round Opened（全局）
      - eve 提交 Hackathon 参赛作品（follow eve）
      - Hackathon 阶段切换（Org 成员）
      - eve 报名 Hackathon（follow eve）
      - Hackathon 发布（全局）
      - eve 完成 Bounty（follow eve）
      - eve 认领 Bounty（follow eve）
      - Bounty 发布（全局 + watch repo）

    断言不包含:
      - grace 的个人行为（frank 没 follow grace）

  步骤 2: 验证 hacker_grace 的 following feed
    grace follows: hacker_eve, hacker_frank
    grace 是 spring-hack-org 成员
    grace 没 watch acme-dev/backend

    GET /api/v1/hackforger/feed?type=following (as grace)
    断言包含: eve 的行为 + frank 的行为 + Org 事件 + 全局事件
    断言包含: Bounty 认领（通过 follow eve，不通过 watch repo）

  步骤 3: 验证 agent_hunter 的 following feed
    agent_hunter follows: org_bob
    agent_hunter 不是任何 Org 成员

    GET /api/v1/hackforger/feed?type=following (as agent_hunter)
    断言包含: 全局事件 + Bounty 发布（follow bob）
    断言不包含: Hackathon Org 内部的阶段切换事件

  步骤 4: 验证 Web Dashboard 可访问
    GET / (as frank, with cookie session)
    断言: 200
    GET /?feed=following (as frank)
    断言: 200, 响应 HTML 包含 HackForger 事件的渲染文案


TestE2E_Phase2_Reputation
  （与 v1 相同）

TestE2E_Phase2_CreditRedeem

  步骤 1-10: （与 v1 相同）

  步骤 11: 验证 Feed — 兑换事件
    frank（follow eve）:
      → assertFeedContains(t, "hacker_frank", 42, "A100 GPU 1小时", "following")
      // eve 兑换了 GPU，frank 关注了 eve 所以应该看到
```

### 3.4 Phase 3 — AI 助手 + 全流程验证（Week 5 后）

```
TestE2E_Phase3_Search
  （与 v1 相同）

TestE2E_Phase3_Assistant
  （与 v1 相同）

TestE2E_Phase3_AgentFullFlow
  （与 v1 相同 + 以下 feed 验证）

  额外步骤: 验证 Agent 行为在 feed 中可见
    agent_hunter 完成 Bounty 后:
    org_bob（follow 关系不需要，因为 Bounty 完成是 Repo Watcher 可见）:
      GET /feed?type=bounty&entity_id={agent_bounty_id}
      断言: 包含 created → claimed → completed 全链路


TestE2E_Phase3_FullJourney

  步骤 1-3: 验证 Hackathon / Bounty / Grant 最终状态（与 v1 相同）

  步骤 4: 验证积分余额（与 v1 相同）

  步骤 5: 验证交易流水完整性（与 v1 相同）

  步骤 6: 验证声誉排行榜（与 v1 相同）

  步骤 7: 验证全局统计（与 v1 相同）

  步骤 8: 验证搜索（与 v1 相同）

  步骤 9（新增）: 验证 Feed 完整性

    9a: 全局 Feed 覆盖度
      GET /api/v1/hackforger/feed?type=global&limit=100
      断言: 包含以下全局事件（按时间倒序的子集）:
        - Hackathon 结果公布（op_type=51）
        - Grant Round Finalized（op_type=56）
        - Grant Round Opened（op_type=54）
        - Grant Round Created（op_type=39）
        - Bounty Winners Selected（op_type=38，competitive）
        - Bounty Created（op_type=34，至少 2 个）
        - Hackathon Phase Changes（op_type=50，多个）
        - Hackathon Created（op_type=30）
      断言: total_count >= 15（粗略下界）

    9b: 每个用户的 Feed 非空
      对 eve, frank, grace, agent_hunter:
        GET /api/v1/hackforger/feed?type=following
        断言: items.length > 0

    9c: Entity Feed 完整生命周期
      Hackathon entity feed:
        GET /feed?type=hackathon&entity_id={hackathon_id}
        断言: items 按时间排列，覆盖完整生命周期
        断言: 第一条是 created，最后一条是 finalized

      Exclusive Bounty entity feed:
        GET /feed?type=bounty&entity_id={bounty_1_id}
        断言: created → claimed → delivered → completed → paid

      Competitive Bounty entity feed:
        GET /feed?type=bounty&entity_id={bounty_2_id}
        断言: created → winners_selected → completed

      Grant Round entity feed:
        GET /feed?type=grant&entity_id={round_id}
        断言: created → opened → project_submitted(×2) → closed → awarded(×2) → finalized

    9d: Feed 时间顺序一致性
      GET /api/v1/hackforger/feed?type=global&limit=100
      遍历 items:
        断言: items[i].created_at >= items[i+1].created_at（降序）

    9e: Feed 事件与业务数据一致性
      取 Hackathon Finalized 事件的 content:
        解析 HackforgerPhaseContent JSON
        断言: WinnerName == 排行榜第一名的 Team Name
      取 Bounty Completed 事件的 content:
        解析 HackforgerActionContent JSON
        断言: Extra 中的积分数 == 实际发放积分数
```

---

## 四、测试执行时序

```
Week 1 完成后:
  ✅ 所有 Model 层单元测试
  ✅ TestE2E_Phase0_UserSetup（含 Follow 关系建立 + Watch 设置）

Week 2-3 完成后:
  ✅ Bounty / Hackathon / Grant / Credits Service 单元测试
  ✅ Feed 单元测试: HackForgerNotifier 事件分发 + Content 序列化
  ✅ TestE2E_Phase1_BountyExclusive（含每步 feed 验证）
  ✅ TestE2E_Phase1_BountyCompetitive（含 feed 验证）
  ✅ TestE2E_Phase1_HackathonCreation（含 feed 验证）
  ✅ TestE2E_Phase1_GrantRound（含 feed 验证）
  注: 此时 Feed 事件已写入 action 表但 Dashboard UI 尚未改造，通过 API 验证

Week 4 完成后:
  ✅ Judging / Reputation 单元测试
  ✅ Feed 查询单元测试: GetFollowingFeed / GetEntityFeed / API
  ✅ TestE2E_Phase2_Judging（含 feed 验证）
  ✅ TestE2E_Phase2_FeedDashboard（完整 feed 正确性验证）
  ✅ TestE2E_Phase2_Reputation
  ✅ TestE2E_Phase2_CreditRedeem（含 feed 验证）

Week 5 完成后:
  ✅ Search / Assistant 单元测试
  ✅ TestE2E_Phase3_Search
  ✅ TestE2E_Phase3_Assistant
  ✅ TestE2E_Phase3_AgentFullFlow（含 feed 验证）
  ✅ TestE2E_Phase3_FullJourney（最终验证，含 feed 完整性 9a-9e）
```

---

## 五、测试数据最终快照

| 实体 | 数量 | 状态 |
|------|------|------|
| Users | 10 | 活跃，有 Follow 关系网 |
| Follow 关系 | 6 | eve←frank, eve←grace, alice←eve, alice←frank, bob←eve, bob←agent_hunter |
| Watch 关系 | 2 | acme-dev/backend ← eve, frank |
| Organizations | 3 | spring-hack-org(成员:eve,frank,grace), acme-dev, grants-org |
| Hackathons | 1 | Finished, 完整生命周期 |
| Bounties | 3+ | paid + completed + agent 完成的 |
| Grant Rounds | 1 | Finalized |
| Credit Accounts | 5+ | eve 最高 |
| Redeem Orders | 3+ | fulfilled + pending |
| Reputations | 5+ | eve 排第一 |
| **action 表 HackForger 事件** | **40+** | 覆盖全部 18 种事件类型 |

---

## 六、前端组件测试

6 个 Vue 组件需要 Vitest 单元测试，文件与组件同级放置在 `web_src/js/components/hackforger/` 下或使用 `*.test.js` 后缀。测试环境使用 Happy DOM（已在 `vitest.config.ts` 中配置）。

```
BountyPanel.test.js         — 状态渲染、认领按钮交互
HackathonRegForm.test.js    — 表单验证、提交行为
JudgeScoreCard.test.js      — 评分输入、分数计算
GrantAllocator.test.js      — 金额分配、预算校验
CreditRedeemDialog.test.js  — 余额检查、兑换确认
PlatformAssistant.test.js   — 搜索输入、结果渲染
```

---

## 七、数据库兼容性

当前阶段仅需保证 SQLite 通过（`make test-sqlite`）。MySQL/PostgreSQL 兼容性留到后续处理。
