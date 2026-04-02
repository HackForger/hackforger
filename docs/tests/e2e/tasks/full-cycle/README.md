# 用户旅程全流程 E2E 测试

## 概述

本测试将 5 个用户旅程融合为一个连贯的叙事，拆分为 9 个文件以支持并行执行。

> **Web 端测试是必选项，不可使用单元测试或 API 测试替代 Web 界面操作。**
> **关键验证节点必须使用 agent-browser 截屏记录作为测试证据。**
> **API 仅用于 git 操作（Fork/PR）和设计上需通过 API 的操作。**

## 文件清单

| 文件 | 阶段 | 步骤数 | 预计时长 |
|------|------|--------|---------|
| 00-setup.md | Phase 0: 平台准备 | 3 | 5 min |
| 01-hackathon-lifecycle.md | Phase 1-3: 创建→报名→Hacking | ~15 | 15 min |
| 02-bounty-collab.md | Phase 4: 开发+Bounty | ~12 | 15 min |
| 03-grant-funding.md | Phase 5: Grant 资助 | ~10 | 10 min |
| 04-submission-judging.md | Phase 6-7: 提交+评审 | ~12 | 15 min |
| 05-credits-redeem.md | Phase 8: Credits 兑换 | ~8 | 10 min |
| 06-social-feed.md | Phase 9: 社交+搜索 | ~10 | 10 min |
| 07-edge-cases.md | Phase 10: 异常路径 | ~10 | 15 min |
| 08-cross-validation.md | 跨阶段验证+报告 | 3 | 5 min |

## 依赖关系与并行策略

```text
00-setup (必须最先)
    │
    ▼
01-hackathon-lifecycle (Phase 1-3)
    │
    ├──────────────────────┐
    ▼                      ▼
02-bounty-collab       03-grant-funding     07-edge-cases
(Phase 4)              (Phase 5)            (Phase 10, 独立)
    │                      │
    └──────┬───────────────┘
           ▼
    04-submission-judging (Phase 6-7)
           │
           ├──────────────────┐
           ▼                  ▼
    05-credits-redeem    06-social-feed
    (Phase 8)            (Phase 9)
           │                  │
           └──────┬───────────┘
                  ▼
           08-cross-validation
```

**可并行的组合：**
- **并行组 1**: `02-bounty-collab` + `03-grant-funding` + `07-edge-cases`（3 个 agent 并行）
- **并行组 2**: `05-credits-redeem` + `06-social-feed`（2 个 agent 并行）

## 执行方式

### 串行执行（单 agent）
```bash
# 按文件编号顺序执行 00 → 01 → 02 → 03 → 04 → 05 → 06 → 07 → 08
```

### 并行执行（多 agent，推荐）
```bash
# Wave 1: 基础设施
agent-0: 00-setup.md

# Wave 2: Hackathon 创建
agent-0: 01-hackathon-lifecycle.md

# Wave 3: 并行开发阶段（3 个 agent）
agent-1: 02-bounty-collab.md
agent-2: 03-grant-funding.md
agent-3: 07-edge-cases.md          # 独立，不依赖 Phase 4-5

# Wave 4: 提交评审
agent-0: 04-submission-judging.md

# Wave 5: 并行收尾（2 个 agent）
agent-1: 05-credits-redeem.md
agent-2: 06-social-feed.md

# Wave 6: 最终验证
agent-0: 08-cross-validation.md
```

## 报告整合

每个文件执行后产出独立的结果片段。`08-cross-validation.md` 负责整合所有片段为最终报告。

最终报告保存到: `docs/tests/e2e/reports/user-journey-full-cycle-report.md`
PDF 渲染到: `docs/tests/e2e/pdf/`
