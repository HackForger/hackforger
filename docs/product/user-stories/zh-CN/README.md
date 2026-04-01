# HackForger v0.1 — 用户故事清单

> 本文档为 [user-stories.md](../../user-stories.md) 的中文翻译版本，按用户角色拆分。

> **关联文档:** [PRD](../../prd.zh-CN.md) · [用户旅程](../../user-journeys.md) (逐步规格说明)
> **格式:** Mike Cohn (As a / I want to / so that) + Gherkin 验收标准
> **日期:** 2026-04-01

## 角色文件

- [组织者故事](organizer.md) -- Hackathon + Bounty + Grant 组织者操作
- [黑客故事](hacker.md) -- 参与者/开发者操作
- [评委故事](judge.md) -- Hackathon 评委操作
- [管理员故事](admin.md) -- 平台管理员操作
- [AI Agent 故事](agent.md) -- Platform-bot API 操作
- [共享故事](shared.md) -- Feed、社交、发现（多角色）

---

## 故事索引

| ID | 史诗 | 概要 | 旅程参考 |
|----|------|------|----------|
| H-001 | Hackathon | 创建 Hackathon | J1.2 |
| H-002 | Hackathon | 创建 Track (仓库) | J1.3 |
| H-003 | Hackathon | 设置评审标准 | J1.4 |
| H-004 | Hackathon | 分配评委 | J1.5 |
| H-005 | Hackathon | 发布 Hackathon | J1.6 |
| H-006 | Hackathon | 注册参加 Hackathon | J1.8-9 |
| H-007 | Hackathon | 通过 Fork + PR 提交 | J1.11-15 |
| H-008 | Hackathon | 评分（多标准） | J1.17-18 |
| H-009 | Hackathon | 结束 + 分发奖金 | J1.19-20 |
| H-010 | Hackathon | 阶段转换 | J1.10, 1.16 |
| B-001 | Bounty | 创建独占模式 Bounty | J2a.1-3 |
| B-002 | Bounty | 申请与认领 | J2a.4-5, J2c.3 |
| B-003 | Bounty | 交付 + 完成 + 付款 | J2a.6-9 |
| B-004 | Bounty | 竞争模式 Bounty | J2b.1-6 |
| B-005 | Bounty | Bounty 过期 + 退款 | J2c.1 |
| B-006 | Bounty | Bounty 取消 + 退款 | J2c.2 |
| G-001 | Grant | 创建 + 开放轮次 | J3.1-2 |
| G-002 | Grant | 提交申请 | J3.3-4 |
| G-003 | Grant | 批准/拒绝项目 | J3.6 |
| G-004 | Grant | 定稿轮次 | J3.7 |
| G-005 | Grant | 分发资金 | J3.8 |
| C-001 | Credits | 查看余额 + 历史 | J4.3-4 |
| C-002 | Credits | 兑换 Credits | J4.5-6 |
| C-003 | Credits | admin 完成订单 | J4.1-2, 4.7 |
| C-004 | Credits | admin 充值/扣款 | J4.8-9 |
| F-001 | Feed | Dashboard 动态流 | J5.1-2 |
| F-002 | Feed | Explore 页面 | J5.3-5 |
| F-003 | Feed | 跨实体搜索 | J5.6-7 |
| F-004 | Feed | 声誉排行榜 | J5.8-9 |
| A-001 | Agent | 发现 Bounty | J6.1 |
| A-002 | Agent | 参与 Hackathon | J6.3-5 |
| A-003 | Agent | 查看 Credits + 声誉 | J6.6-8 |
