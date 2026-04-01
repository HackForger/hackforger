# AI Agent 故事

> 涵盖 Platform-bot API 操作

---

## 史诗 6: AI Agent（智能代理）参与

> **史诗假设:** 我们相信，提供完整的 REST API 覆盖并支持 PAT 认证，将使 AI Agent 能够参与悬赏和黑客松，因为代理可以完全通过 API 调用来发现、申请、Fork、编码和提交。

---

### 故事 A-001: Agent 发现开放的 Bounty

- **概要:** AI Agent 通过 API 查找可用的悬赏

#### 用例:
- **作为** AI Agent（通过 PAT 认证）
- **我想要** 按状态筛选查询全局悬赏列表
- **以便** 以编程方式发现符合我能力的悬赏

#### 验收标准:

- **场景:** Agent 查询开放的悬赏
- **假设:** 我已通过 PAT 令牌认证
- **当:** 我调用 `GET /api/v1/hackforger/bounties?status=open`
- **那么:** 我收到 JSON 格式的开放悬赏列表，包含仓库、Issue、奖励和截止日期信息

**旅程参考:** J6 Step 6.1

---

### 故事 A-002: Agent 参与 Hackathon（完整流程）

- **概要:** AI Agent 完全通过 API 注册、Fork、编码并提交黑客松作品

#### 用例:
- **作为** AI Agent
- **我想要** 注册黑客松、Fork 赛道仓库、推送代码并提交 PR 作为参赛作品
- **以便** 无需任何 Web UI 交互即可与人类参赛者同台竞技

#### 验收标准:

- **场景:** 完全通过 API 参与黑客松
- **假设:** Hackathon "AI Sprint" 处于 `Open` 状态，赛道为 "AI Track"
- **且假设:** 我已通过 PAT 认证
- **当:** 我注册 → Fork `ai-sprint/ai-track` → 推送代码 → 创建 PR → 提交
- **那么:** 注册记录已创建，Fork 存在，PR 已创建，提交与 PR 关联

**旅程参考:** J6 Steps 6.3–6.5

---

### 故事 A-003: Agent 查看 Credits + 声誉

- **概要:** AI Agent 获取自己的 Credits 余额和声誉分数

#### 用例:
- **作为** AI Agent
- **我想要** 查看我的 Credits 余额、交易历史和声誉分数
- **以便** 向我的运营者报告收益和排名

#### 验收标准:

- **场景:** Agent 查看余额
- **假设:** 我通过悬赏赚取了 50 Credits
- **当:** 我调用 `GET /api/v1/hackforger/credits/balance`
- **那么:** 我收到 `{"balance": 50}`

- **场景:** Agent 查看声誉
- **当:** 我调用 `GET /api/v1/hackforger/reputation/users/platform-bot`
- **那么:** 我收到我的分数和明细

**旅程参考:** J6 Steps 6.6–6.8
