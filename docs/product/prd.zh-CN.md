# HackForger v0.1 — 产品需求文档

> 本文档为 [prd.md](prd.md) 的中文翻译版本

> **状态：** Draft
> **作者：** Allen Woods
> **日期：** 2026-04-01
> **分支：** v0.1-dev/hackforger

---

## 1. 执行摘要

HackForger 是一个 Forgejo 的 Fork，将自托管的 Git 代码仓库平台转变为一个**完整的黑客松、赏金和资助平台** —— 其中每一个竞赛构件（注册、提交、评审、奖励）都映射到原生的 Git 对象（组织、仓库、Fork、PR、Review）。通过在 Forgejo 之上构建，而非创建又一个独立的黑客松工具，HackForger 消除了"提交之处"与"构建之处"之间的鸿沟，为组织者提供一站式平台，为参赛者提供从代码到奖励的无缝工作流 —— 所有这一切都由内部 Credits 经济驱动。

---

## 2. 问题陈述

### 谁面临这个问题？

三类受众：

1. **黑客松组织者**（社区、公司、DAO）—— 举办编程竞赛的人
2. **开源维护者 / 项目所有者** —— 希望通过赏金激励贡献的人
3. **资助发放方**（基金会、公司）—— 为开源项目提供资金的人

### 问题是什么？

**对于组织者：** 如今举办一场黑客松需要拼凑 3-5 个独立工具 —— 用 Devpost 或 Devfolio 处理提交，用 GitHub/GitLab 管理代码，用 Google Forms 处理报名，用 Airtable 管理评审，用 Stripe/PayPal 发放奖金。这导致了：
- **提交完整性漏洞** —— 评委审查的是 Devpost 链接而非实际代码；时间戳不可靠；抄袭检测需要人工操作
- **参与者体验碎片化** —— 参赛者在一个平台注册、在另一个平台写代码、在第三个平台提交
- **缺乏持续社区** —— 活动结束后参与者各自散去；组织者下次活动需要从零开始

**对于赏金组织者：** GitHub Issues 缺乏原生的奖励机制。附加工具（Gitcoin、Algora）要求参与者离开代码仓库平台，创建独立账号，并使用与代码库不集成的支付基础设施。

**对于资助发放方：** 资助管理通常依赖电子表格驱动，与被资助项目的实际产出没有任何关联。

### 为什么这很痛苦？

- **组织者** 将 40% 以上的时间花在后勤事务而非社区建设上
- **参赛者** 每场活动需要在 3 个以上工具之间频繁切换，丧失心流和动力
- **评委** 在评估提交时无法方便地查看代码历史、PR 讨论或提交质量
- **整个生态** 缺乏统一的激励层 —— 没有办法在黑客松、赏金和资助之间携带声誉或奖励

### 证据

- Devpost 托管了 15,000+ 场黑客松，但零 Git 集成 —— 评委看到的是链接而非仓库
- GitCoin Bounties（现已废弃）证明了代码原生奖励的需求，但因其游离于代码仓库平台之外而失败
- Forgejo 拥有 4,800+ 星标，自托管采用率持续增长，但零竞赛/激励功能
- 来自大学黑客松组织者的反馈："我们花在管理提交上的时间比指导参赛者还多"

---

## 3. 目标用户与用户画像

### 角色

HackForger 定义 4 个产品角色。**角色是产品层面的概念**，系统层面通过 capability（能访问什么、能执行哪些操作）来区分权限。一个用户可以同时拥有多个角色。

| 角色 | 产品定义 | 对应的 Capabilities |
|------|---------|-------------------|
| **Admin** | 平台管理者 | `*`（全部），拥有默认 platform-bot |
| **Organizer** | 组织者 — 创建和管理 Hackathon、Bounty、Grant | `hackathon.*`, `bounty.create`, `bounty.manage`, `grant.*`, `bot.create` |
| **Hacker** | 参与者 — 报名、提交作品、认领 Bounty、申请 Grant | `hackathon.register`, `hackathon.submit`, `bounty.claim`, `bounty.deliver`, `grant.apply`, `credits.redeem`, `bot.create` |
| **Judge** | 评审者 — 对提交物评分、Review PR | `judge.score`, `judge.review` |

> **设计原则：** 跟随 Forgejo 的 capability-based 权限模式。Forgejo 没有 "role" 表 — 它通过 `AccessMode × UnitType` 组合隐式表达角色。HackForger 同样不建 role 表，而是通过扩展 `UnitType`（新增 `TypeHackathon`, `TypeBounty`, `TypeGrant`, `TypeCredits`）复用现有 TeamUnit 机制。产品文档中的"角色"仅作为 capability 集合的语法糖。

### Bot 模型

Bot 不是独立角色，而是附着在用户上的自动化代理，权限继承自创建者（只能是子集）。

| Bot | 创建者 | 用途 |
|-----|--------|------|
| `platform-bot` | admin（平台默认创建） | 辅助平台管理、回复用户查询 |
| organizer 的 bot | organizer（可选） | 自动化 hackathon/bounty 管理 |
| hacker 的 bot | hacker（可选） | 自动发现和参与 bounty/hackathon |

Bot 使用 PAT 认证，`User.Type = UserTypeBot(4)`（Forgejo 原生支持），权限上限 = 创建者的 capability 集合。

### 用户画像

| 画像 | 角色 | 动机 | 痛点 |
|------|------|------|------|
| **组织者 Olivia** | 社区负责人 / DevRel | 办好 hackathon、发布 bounty、分配 grant | 需要同时管理 Devpost + GitHub + Google Forms + Airtable |
| **参赛者 Hao** | 全栈开发者 | 赢得奖金、积累作品集、学习新技术 | 在提交平台和代码平台之间反复切换 |
| **评委 Julia** | 技术评审员 | 给出公正的结构化评价 | 从截图链接评审代码体验极差 |
| **管理员 Alex** | 平台运维 | 维护平台运行、管理 Credits | 需要运营面板而非直接查数据库 |

### 待完成工作（Jobs-to-Be-Done）

| 工作 | 角色 | 成功标准 |
|------|------|----------|
| 在一个平台完成 hackathon 全流程 | Organizer | 从创建到发奖零外部工具 |
| 在写代码的地方提交和被评审 | Hacker | Fork → code → PR → score，同一个 URL |
| 给 Issue 挂悬赏，完成后付款 | Organizer | Issue → Bounty → Claim → PR → Pay，5 次点击 |
| 带完整上下文评审代码质量 | Judge | 在同一视图看到 PR diff、commit history、评分标准 |
| 分配 grant 并跟踪产出 | Organizer | 申请 → 审批 → 资助 → 跟踪，无需 spreadsheet |
| 跨活动积累声誉 | Hacker | 一个 profile 展示 hackathon 成绩、bounty 完成、grant 获得 |
| Bot 自动参与平台活动 | Any (via bot) | Bot 通过 API 发现、申请、提交，权限受创建者约束 |

---

## 4. 战略背景

### 愿景

**让每一个 Git 代码仓库平台都成为黑客松平台。** 正如 Forgejo 将自托管变成了 GitHub 的可行替代方案，HackForger 将自托管 Git 变成了 Devpost + Gitcoin + Open Collective 的可行替代 —— 由社区拥有，而非被 SaaS 厂商控制。

### 为什么要 Git 原生？

核心架构洞察：**黑客松概念天然对应 Git 原语**。

| HackForger 概念 | Git 原语 | 映射理由 |
|-----------------|----------|----------|
| Hackathon（黑客松） | Organization（组织） | 作用域（成员、团队、仓库）、权限、头像 |
| Track（赛道） | Repository（仓库） | 代码模板、README 规则说明、Issues 用于问答 |
| 注册 | Org Membership（组织成员） | 已控制仓库访问权限 |
| 提交 | Fork + Pull Request | 带时间戳、可 diff、可评审、不可伪造 |
| 评审 | PR Review + 结构化评分 | 行内评论 + 数值化评分标准 |
| 团队 | Org Team（组织团队） | 已处理分组权限 |
| 结果 | Release（发布） | 永久产物，含排行榜 + 链接 |

这不是隐喻 —— HackForger 真正创建这些 Git 对象。一次提交**就是**一个 PR。一次评审**就是**一次代码审查。这从根本上消除了一整类完整性问题（时间戳伪造、抄袭、"我提交了但平台弄丢了"）。

### 为什么是现在？

1. Forgejo v10+ 稳定了 Fork 版本，使其成为安全的构建基础
2. 自托管 Git 的采用率持续增长（Forgejo、Gitea、GitLab CE）—— 社区追求主权
3. Devpost 已停滞不前；在 Git 集成方面无新功能
4. AI 智能体正在成为黑客松参与者 —— Git 原生平台天然支持 API 优先的智能体
5. Credits/代币经济已被验证（GitPOAP、SourceCred），但无一与代码仓库平台集成

### 竞争格局

| 平台 | Git 集成 | 奖励 | 自托管 | 弱点 |
|------|----------|------|--------|------|
| Devpost | 无（仅链接） | 外部 | 否 | 评委看不到代码 |
| Devfolio | GitHub OAuth | 外部 | 否 | 仅 SaaS，闭源 |
| Gitcoin | GitHub Issues | 加密货币 | 否 | 赏金产品已废弃 |
| Algora | GitHub Issues | 美元 | 否 | 附加式，非原生 |
| **HackForger** | **原生（本身就是代码仓库平台）** | **Credits（内部）** | **是** | **新项目，社区较小** |

---

## 5. 解决方案概览

### 模块架构

HackForger 在 Forgejo 基础上新增 5 个模块，全部位于隔离的 `*/hackforger/` 目录中：

```
┌─────────────────────────────────────────────────────────┐
│                    HackForger Platform                    │
├──────────┬──────────┬──────────┬──────────┬──────────────┤
│ Hackathon│  Bounty  │  Grant   │ Credits  │ Feed +       │
│          │          │          │          │ Reputation   │
├──────────┴──────────┴──────────┴──────────┴──────────────┤
│              Forgejo Core (Git, Users, Orgs, PRs)         │
└─────────────────────────────────────────────────────────┘
```

**模块依赖关系图：**

```
Hackathon ──→ Credits (prize distribution)
Bounty    ──→ Credits (reward payment)
Grant     ──→ Credits (grant distribution)
Feed      ←── ALL (event emission via PublishHackforgerAction)
Reputation ←── ALL (score aggregation from hackathon wins, bounty completions, grants)
Credits   ──→ (standalone, receives deposits from other modules)
```

### 模块边界

#### Hackathon（黑客松）模块
- **管辖范围：** Hackathon 生命周期（Draft → Open → Hacking → Judging → Finished/Cancelled）
- **Git 集成：** 自动创建 Org + Repos，注册 = Org 成员资格，提交 = Fork + PR
- **评审：** 多维度评分系统（按赛道设置评分标准，加权计分）
- **约束：** 每场 Hackathon 最多 10 个赛道
- **产出：** 排行榜、包含结果的 Release

#### Bounty（赏金）模块
- **管辖范围：** Bounty 生命周期（Open → Claimed → InReview → Completed → Paid / Expired / Cancelled）
- **两种模式：** Exclusive（独占式，单人认领）vs. Competitive（竞争式，多人提交，排名获奖）
- **Git 集成：** 挂载在 Issue 上（1:1 关系），通过 PR 交付，继承 Issue 的所有能力
- **产出：** Credits 支付给获胜者

#### Grant（资助）模块
- **管辖范围：** Grant Round（资助轮次）生命周期（Draft → Open → Review → Finalized → Distributed / Cancelled）
- **模式：** 基于申请（提交项目 → 评审 → 批准 → 分配预算）
- **两阶段资助：** `Finalized` = 项目批准 + 预算分配（承诺），`Distributed` = 组织者在审查项目进展后手动释放 Credits。平台不强制要求项目更新 —— 由组织者根据自身判断决定（检查提交、PR、里程碑等）
- **约束：** 总分配额 ≤ 轮次预算
- **产出：** 在组织者手动触发后，Credits 分发给已批准的项目

#### Credits（积分）模块
- **管辖范围：** 账户余额、交易历史、兑换、托管
- **操作：** Deposit（充值，来自 Hackathon/Bounty/Grant 获奖）、Withdraw（内部提取）、Redeem（兑换奖励）、Escrow（平台托管赏金奖励）、Refund（托管资金退还给组织者）
- **交易类型：** `deposit`、`withdraw`、`redeem`、`admin_deposit`、`admin_deduct`、`escrow`、`escrow_release`、`escrow_refund`
- **管理功能：** 手动充值/扣减、兑换选项管理、密钥池、订单履行
- **不变式：** 所有余额变更使用 `db.WithTx`（ACID）

#### Feed + Reputation（动态 + 声誉）模块
- **Feed：** 独立的 `hackforger_action` 表（非 Forgejo 的 `action` 表），24 种事件类型，4 种受众类型（全局/关注者/组织/关注仓库者）
- **Reputation（声誉）：** 基于黑客松名次、赏金完成、资助记录的加权评分；分级徽章
- **发现：** 黑客松、赏金、资助的 Explore 页面；跨实体全文搜索

### 状态机

**Hackathon：**
```
Draft ──publish──→ Open ──start──→ Hacking ──start_judging──→ Judging ──finalize──→ Finished
  │                  │                │              │                                    
  └───cancel────────→└───cancel──────→└───cancel────→└───cancel──→ Cancelled
```

**Bounty（Exclusive 独占模式）：**
```
Open ──claim──→ Claimed ──submit_pr──→ InReview ──complete──→ Completed ──pay──→ Paid
  │                                                              │
  └──expire──→ Expired (Credits refunded to organizer)           └──cancel──→ Cancelled (Credits refunded)
```
> **托管（Escrow）：** 当创建带有 Credits 奖励的 Bounty 时，平台从组织者余额中托管相应金额。完成后 → 支付给认领者。过期/取消时 → 退还给组织者。

**Bounty（Competitive 竞争模式）：**
```
Open ──(multiple submissions)──→ Open ──select_winners──→ Completed ──pay──→ Paid
  │
  └──expire/cancel──→ Expired/Cancelled (Credits refunded to organizer)
```

**Grant Round：**
```
Draft ──open──→ Open ──close──→ Review ──finalize──→ Finalized ──distribute──→ Distributed
  │               │                │                                              
  └──cancel──────→└──cancel───────→└──cancel──→ Cancelled
```

**RedeemOrder（兑换订单）：**
```
pending ──fulfill──→ fulfilled
  │
  └──cancel──→ cancelled
```

---

## 6. 成功指标

### 主要指标（v0.1 目标 —— 上线后前 3 个月）

| 指标 | 定义 | 目标 |
|------|------|------|
| **Hackathon 完成率** | 达到 Finished 状态的黑客松百分比 | ≥ 70% |
| **Bounty 完成率** | 达到 Paid 状态的赏金百分比 | ≥ 50% |
| **通过 PR 提交率** | 黑客松提交中真正通过 PR 提交的百分比（非外部链接） | 100%（架构保证） |

### 次要指标

| 指标 | 定义 | 目标 |
|------|------|------|
| Credits 流转率 | Credits 充值总额 → 兑换比率 | ≥ 30% 已兑换 |
| 重复参与率 | 参加 2 场以上活动的参赛者百分比 | ≥ 25% |
| 评委响应时间 | 从 Judging 阶段开始到所有评分提交的平均时间 | < 7 天 |
| Grant 预算利用率 | 轮次预算中分配给已批准项目的百分比 | ≥ 60% |
| Feed 参与度 | 每周至少查看一次 Feed 的活跃用户百分比 | ≥ 40% |

### 护栏指标

| 指标 | 约束 |
|------|------|
| Forgejo 上游兼容性 | 在 11 个定义的注入点之外，零修改 Forgejo 核心文件 |
| API 响应时间 | 所有 HackForger 端点 p95 < 500ms |
| Credits 交易完整性 | 零余额不一致（由 `db.WithTx` 强制保证） |

---

## 7. 功能范围概述

> **注意：** 详细用户故事将放在单独的 `docs/user-stories.md` 文档中。
> 详细的分步用户旅程及 API 端点将放在 `docs/user-journeys.md` 中。

### Epic 1：Hackathon 生命周期
创建、发布、注册、提交（Fork+PR）、评审（多维度）、定稿、发放 Credits。

### Epic 2：Bounty 生命周期
在 Issue 上创建，两种模式（Exclusive/Competitive），申请、认领、交付（PR）、完成、支付 Credits。

### Epic 3：Grant 生命周期
创建轮次、接受申请、评审、分配预算、定稿、发放 Credits。

### Epic 4：Credits 经济
账户管理、充值/提取、兑换选项 + 密钥池、订单履行、管理员操作。

### Epic 5：Feed + 社交 + 发现
24 种事件类型、4 种受众类型、Explore 页面、搜索、声誉评分 + 排行榜。

### Epic 6：AI 智能体参与
完整的 REST API 覆盖、PAT 认证、对自动化参与友好的端点。

### 横切关注点
- **i18n：** 所有用户可见文本同时存在于 `locale_en-US.ini` 和 `locale_zh-CN.ini`
- **类型化错误：** 服务层返回类型化错误，Web 层通过 `ctx.Tr()` 翻译
- **Feed 事件：** 每次状态变更都触发 `PublishHackforgerAction`
- **ACID Credits：** 所有余额变更在 `db.WithTx` 内执行

---

## 8. 不在范围内（v0.1）

| 功能 | 原因 | 未来版本 |
|------|------|----------|
| Vue SPA 前端 | v0.1 使用 Go 模板 SSR；Vue 增强是渐进式的 | v0.2 |
| Webhook 事件发送 | 架构已就绪，发送逻辑延后 | v0.2 |
| AI 辅助代码审查 | v0.1 的评审系统仅限人工 | v0.3 |
| 基于 Cron 的自动阶段转换 | 基础设施已就绪，逻辑延后 | v0.2 |
| 外部支付集成（Stripe、加密货币） | Credits 仅限内部使用；外部兑现延后 | v0.3 |
| 移动端优化 UI | 桌面端优先 | v0.2 |
| Grant 的二次方资助 | 先采用标准分配模型 | v0.3 |
| 多语言黑客松内容 | i18n 覆盖 UI 界面，不覆盖用户生成内容 | v0.3 |
| 抄袭检测 | 基于 PR 的提交提供 git blame/history 供人工检查 | v0.3 |
| 通过管理界面调整高级声誉公式 | 权重仅在代码中可配置 | v0.2 |

---

## 9. 依赖与风险

### 技术依赖

| 依赖 | 状态 | 影响 |
|------|------|------|
| Forgejo v10+ 稳定 API | ✅ 已解决 | 所有 HackForger 功能的基础 |
| XORM 自动迁移 | ✅ 可用 | 16 张新表在启动时自动创建 |
| Forgejo Actions runner | ✅ 运行中 | Git 操作（分支、提交、推送、PR）使用 Actions 工作流 |
| LevelDB 队列系统 | ✅ 可用 | 异步事件处理 |

### 风险与缓解措施

| 风险 | 严重程度 | 缓解措施 |
|------|----------|----------|
| **上游 Forgejo 合并冲突** | 中 | 隔离的 `*/hackforger/` 目录；仅 11 个注入点接触上游文件 |
| **Credits 双重支付** | 高 | 所有变更在 `db.WithTx` 内；无锁不允许并发余额更新 |
| **状态机非法转换** | 中 | 服务层在转换前验证当前状态；类型化错误 |
| **Feed 表增长** | 低 | 索引复合键；强制分页；无全表扫描 |
| **Hackathon Org 污染** | 低 | 自动创建的 Org 有明确标记；考虑在 v0.2 添加 Org 类型标志 |
| **评委偏见 / 公平性** | 中 | 多维度评分 + 按维度设置权重；匿名化提交顺序（未来） |

---

## 10. 开放问题

| # | 问题 | 状态 | 决定 |
|---|------|------|------|
| 1 | Hackathon Org 是否应在全局 Org 列表中可见？ | **已决定** | 是，带"Hackathon"徽章 —— 增加可发现性 |
| 2 | 用户能否在同一场 Hackathon 中同时担任评委和参赛者？ | **已决定** | 不能 —— 在注册/指派评委时强制执行 |
| 3 | 过期的 Bounty 是否应自动将 Credits 退还给组织者？ | **已决定** | 是 —— 创建 Bounty 时平台托管 Credits；过期/取消时自动退还到组织者账户。这意味着需要 `platform_escrow` 交易类型或平台持有账户。 |
| 4 | 用户账号删除时 Credits 如何处理？ | **已决定** | 余额作废（清零）。简单干净 —— 无孤立资金。 |
| 5 | 竞争模式 Bounty 是否允许截止日期后的迟交？ | **已决定** | 否 —— 截止日期为硬性截止 |
| 6 | 每场 Hackathon 最大赛道数？ | **已决定** | 每场 Hackathon 10 个赛道（创建时强制执行）。如组织者需要更多可重新评估。 |
| 7 | 资助后是否应要求项目更新？ | **已决定** | 两阶段资助：`Finalized` = 批准 + 预算分配，`Distributed` = 资金实际释放。组织者在触发分发前手动审查项目进展（提交、PR、里程碑）。平台**不**强制要求更新 —— 由组织者根据自身标准决定。这与现有的 `Finalized → Distributed` 状态转换完美映射。 |
| 8 | Feed 保留策略 —— 事件保留多久？ | **待定** | v0.1 不做清理；考虑在 v0.2 实现归档 |

---

## 附录 A：技术架构参考

### 分层图

```
┌─────────────────────────────────┐
│  Templates (templates/hackforger/)│  ← Go HTML templates (SSR)
├─────────────────────────────────┤
│  Web Routes (routers/web/)       │  ← Session cookie auth
│  API Routes (routers/api/v1/)    │  ← Token/PAT auth
├─────────────────────────────────┤
│  Services (services/hackforger/) │  ← Business logic, state machines
├─────────────────────────────────┤
│  Models (models/hackforger/)     │  ← CRUD, XORM, 16 tables
├─────────────────────────────────┤
│  Forgejo Core                    │  ← Users, Orgs, Repos, PRs, Issues
└─────────────────────────────────┘
```

### 实体数量

- **16 张数据库表**（通过 XORM 默认规则统一以 `hackforger_` 为前缀）
- **24 种 Feed 事件类型**（ActionType 30–59）
- **每模块 26+ REST API 端点**
- **32 个 HTML 模板**
- **8 个固定测试角色**

### Forgejo 注入点

HackForger 接触上游 Forgejo 代码的 11 个文件：
1. `routers/web/web.go` — 路由注册
2. `routers/api/v1/api.go` — API 路由注册
3. `models/migrations/migrations.go` — 迁移注册
4. `modules/setting/setting.go` — 功能开关（如需要）
5. `templates/base/head_navbar.tmpl` — 导航链接
6. `templates/user/dashboard/feeds.tmpl` — 社区标签页
7. `templates/explore/navbar.tmpl` — Explore 导航
8. `options/locale/locale_en-US.ini` — i18n 键值
9. `options/locale/locale_zh-CN.ini` — i18n 键值
10. `cmd/web.go` — 模块初始化
11. `models/forgejo_migrations/migrate.go` — HackForger 表迁移
