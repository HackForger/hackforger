# HackForger 用户旅程文档

> 本文档为 [user-journeys.md](user-journeys.md) 的中文翻译版本

> **目的：** 为测试夹具 (fixture)、API 集成测试和 E2E 测试脚本提供可执行的规范。
> **相关文档：** [PRD](prd.md)（战略背景）· [User Stories](user-stories.md)（开发就绪的待办列表）
> **日期：** 2026-04-01

---

## 旅程索引

| 文件 | 旅程 | 主要角色 |
|------|------|----------|
| [organizer-hackathon.md](organizer-hackathon.md) | 旅程 1：Hackathon 完整生命周期 | organizer, hacker1, hacker2, judge1, judge2 |
| [organizer-bounty.md](organizer-bounty.md) | 旅程 2：Bounty 完整生命周期（2a/2b/2c） | organizer, hacker1, hacker2, platform-bot |
| [organizer-grant.md](organizer-grant.md) | 旅程 3：Grant 完整生命周期 | organizer, hacker1, hacker2 |
| [hacker-credits.md](hacker-credits.md) | 旅程 4：Credits 流转 + 兑换 | hacker1, admin |
| [hacker-social.md](hacker-social.md) | 旅程 5：Feed + 社交 + 发现 | hacker1, hacker2 |
| [platform-bot.md](platform-bot.md) | 旅程 6：AI Agent 参与 | platform-bot |

---

## 全局前置条件

以下用户在所有旅程开始之前已存在：

| 用户名 | 角色 | 备注 |
|--------|------|------|
| `admin` | 平台管理员 | 站点管理员权限，可管理 Credits |
| `organizer` | Hackathon/Grant/Bounty 组织者 | 普通用户，拥有仓库 `organizer/oss-project` |
| `hacker1` | 参与者 | 活跃的开发者 |
| `hacker2` | 参与者 | 第二位开发者，可为队友/竞争者 |
| `judge1` | 评委 | 技术评审员 |
| `judge2` | 评委 | 第二位评审员 |
| `platform-bot` | AI 代理 | 拥有 PAT 令牌，仅通过 API 访问 |

**Credits 账本** — 跨旅程追踪运行余额：

| 用户 | J1 后 | J2 后 | J3 后 | J4 后 |
|------|-------|-------|-------|-------|
| `hacker1` | 500 | 500+300+150=950 | 950+1000=1950 | 1950-200=1750 |
| `hacker2` | 300 | 300+100=400 | 400 | 400+500-50=850 |
| `platform-bot` | 0 | 0+50=50 | 50 | 50 |
| `organizer` | 1000-800=200 | 200+100+100=400（托管资金流转：J2a/J2b 净值为 0，J2c.1/J2c.2 各 +100 admin 充值） | 400+2000-1000=1400 | 1400 |

> 注意：admin 会在需要消费的旅程开始前，向 organizer 账户预充值 Credits。
> organizer 获得 1000（J1 奖金池）+ 600（J2 赏金）+ 100（J2c.1）+ 100（J2c.2）+ 2000（J3 资助预算）。

**约定**：每个步骤列出了 **API**（REST 端点）和/或 **Web**（页面路由）。所有 HackForger 操作同时提供 Web 和 API 路由 — Web 使用表单 POST 加重定向/闪存消息，API 使用 JSON。以下步骤主要展示 API 路径以便编写测试；对应的 Web 路径遵循以下模式：
- Hackathon: `/hackforger/hackathons/{slug}/...`
- Bounty: `/{owner}/{repo}/bounties/{id}/...`
- Grants: `/hackforger/grants/{slug}/...`
- Credits: `/hackforger/credits/...`
- Admin: `/hackforger/admin/credits/...`

**API 命名偏差**：本文档使用的**目标**端点名称与 Forgejo 上游约定（单动词路径）保持一致。部分名称与当前代码库不同。这些重命名将在 fixture/E2E 阶段通过 TDD 方式实施。详见 [api-rename-plan.md](api-rename-plan.md)。

| 本文档中的端点 | 当前代码库 | 变更 |
|---------------|-----------|------|
| `POST /hackathons/{id}/judge` | `/hackathons/{id}/start-judging` | 重命名以匹配 `/publish`, `/start`, `/finalize` 模式 |
| `POST /grants/rounds/{id}/distribute` | _（尚不存在）_ | 用于两阶段资助的新端点（PRD Q7） |

---

## 跨旅程验证矩阵

### 所有实体状态覆盖情况

| 实体 | 已覆盖的状态 | 旅程 |
|------|-------------|------|
| Hackathon | Draft, Open, Hacking, Judging, Finished | J1 |
| Hackathon（Cancelled） | — | 正常流程未覆盖；添加状态为 Cancelled(5) 的测试 fixture |
| Bounty | Open, Claimed, InReview, Completed, Paid | J2a |
| Bounty | Expired | J2c.1 |
| Bounty | Cancelled | J2c.2 |
| BountyMode | Exclusive(0), Competitive(1) | J2a, J2b |
| BountyApplication | Pending, Accepted, Rejected | J2a, J2c.3 |
| HackathonRegistration | Approved | J1（自动批准，跳过 Pending） |
| HackathonRegistration（Pending） | — | 添加手动审核 Hackathon 的 fixture，注册保持 Pending 直到 organizer 批准 |
| HackathonRegistration（Rejected） | — | 添加手动拒绝场景的 fixture |
| GrantRound | Draft, Open, Review, Finalized, Distributed | J3 |
| GrantRound（Cancelled） | — | 未覆盖；添加状态为 Cancelled(5) 的测试 fixture |
| GrantProject | Pending, Approved, Funded, Rejected | J3 |
| RedeemOrder | pending, fulfilled | J4 |
| RedeemOrder（cancelled） | — | 添加测试：hacker1 在履行前取消订单 |
| CreditTransaction | deposit, withdraw, redeem, admin_deposit, admin_deduct, escrow, escrow_release, escrow_refund | J1-J4 |

### 所有 Feed 事件覆盖情况

| 事件 | ActionType | 旅程 | 步骤 |
|------|-----------|------|------|
| hackathon_created | 30 | J1 | 1.2 |
| hackathon_registered | 31 | J1 | 1.8, 1.9 |
| hackathon_submitted | 32 | J1 | 1.14, 1.15 |
| hackathon_scored | 33 | J1 | 1.17, 1.18 |
| bounty_created | 34 | J2a | 2a.2; J2b 2b.2 |
| bounty_claimed | 35 | J2a | 2a.5 |
| bounty_delivered | 36 | J2a | 2a.7 |
| bounty_completed | 37 | J2a | 2a.8 |
| bounty_winners_selected | 38 | J2b | 2b.5 |
| grant_round_created | 39 | J3 | 3.1 |
| grant_project_submitted | 40 | J3 | 3.3, 3.4 |
| grant_awarded | 41 | J3 | 3.6 |
| credits_redeemed | 42 | J4 | 4.6 |
| bounty_paid | 43 | J2a | 2a.9; J2b 2b.6 |
| hackathon_phase_changed | 50 | J1 | 1.6, 1.10, 1.16 |
| hackathon_finalized | 51 | J1 | 1.19 |
| bounty_expired | 52 | J2c | 2c.1 |
| bounty_cancelled | 53 | J2c | 2c.2 |
| grant_round_opened | 54 | J3 | 3.2 |
| grant_round_closed | 55 | J3 | 3.5 |
| grant_round_finalized | 56 | J3 | 3.7 |
| grant_round_cancelled | 57 | — | 未覆盖；添加 fixture |
| order_fulfilled | 58 | J4 | 4.7 |
| order_cancelled | 59 | — | 未覆盖；添加 fixture |

### 未覆盖的状态和事件（仅 Fixture）

以下状态和事件不属于正常流程旅程，但必须存在于测试 fixture 中：

1. **Hackathon Cancelled(5)** — organizer 取消一个 Draft 状态的 Hackathon
2. **HackathonRegistration Pending(0)** — 配置为手动审核的 Hackathon，注册保持 Pending 直到 organizer 操作
3. **HackathonRegistration Rejected(2)** — organizer 手动拒绝一个注册
4. **GrantRound Cancelled(5)** — organizer 取消一个 Open 状态的 Grant Round
5. **RedeemOrder cancelled** — 用户在 admin 履行前取消待处理订单
6. **grant_round_cancelled(57)** — 已取消轮次的 Feed 事件
7. **order_cancelled(59)** — 已取消订单的 Feed 事件
8. **账户删除时的 Credits 处理** — 余额被没收（归零，依据 PRD Q4）
9. **Track 数量限制强制执行** — 尝试创建第 11 个 Track 返回错误（每个 Hackathon 最多 10 个，依据 PRD Q6）

### Credits 账本最终状态

| 用户 | 余额 | 来源明细 |
|------|------|----------|
| `hacker1` | 1750 | +500（J1 Hackathon）+300（J2a Bounty）+150（J2b Bounty）+1000（J3 Grant）-200（J4 兑换） |
| `hacker2` | 850 | +300（J1 Hackathon）+100（J2b Bounty）+500（J4 admin 充值）-50（J4 admin 扣款） |
| `platform-bot` | 50 | +50（J2b Bounty 第 3 名） |
| `organizer` | 1400 | +1000（J1 admin 充值）-800（J1 奖金）+600（J2 admin 充值）-300（J2a 托管→释放给 hacker1）-300（J2b 托管→释放给获胜者）+100（J2c.1 admin 充值）-100（J2c.1 托管）+100（J2c.1 退还）+100（J2c.2 admin 充值）-100（J2c.2 托管）+100（J2c.2 退还）+2000（J3 admin 充值）-1000（J3 资助） |

### 所有角色使用情况

| 角色 | 参与的旅程 |
|------|-----------|
| `admin` | J1（充值）、J2（充值）、J3（充值）、J4（配置、履行、充值、扣款） |
| `organizer` | J1（完整生命周期）、J2（完整生命周期 — 创建、审核、支付、取消）、J3（完整生命周期） |
| `hacker1` | J1（注册、提交、获胜）、J2（领取、交付）、J3（申请、获资助）、J4（兑换）、J5（探索、搜索） |
| `hacker2` | J1（注册、组队、提交）、J2（竞争）、J3（申请、被拒绝）、J5（表情回应） |
| `judge1` | J1（评分） |
| `judge2` | J1（评分） |
| `platform-bot` | J2b（竞争、获得第 3 名）、J6（完整 API 工作流） |

### 社交互动汇总

| 互动 | 旅程 | 步骤 |
|------|------|------|
| hacker1 关注 organizer | J1 | 1.7 |
| hacker1 ↔ hacker2 互相关注 | J1 | 1.9 |
| 创建团队 "DeFi Duo" | J1 | 1.9 |
| 为 Track 仓库点 Star (x3) | J1 | 1.13 |
| hacker1 Watch organizer/oss-project | J2b | 2b.3 |
| 对 Issue 评论添加表情回应 | J5 | 5.10 |
