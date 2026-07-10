# HackForger 使用指南

**HackForger** 是面向协作项目的业务中立 Forgejo 发行版，提供完整的 Git 协作能力，并加入活动、悬赏、资助、积分、动态等可复用扩展模块。本页是能力清单 —— 找到你需要的功能，按链接深入。

## Git 协作基础能力

这些能力继承自 Forgejo，使用方法与 Forgejo / Gitea / GitHub 一致：

* **仓库 (Repositories)** — 创建、Fork、分支、标签、保护规则、模板仓库、镜像同步
* **Pull Request / 合并请求** — 代码评审、冲突解决、检查、自动合并
* **Issue / 议题** — 标签、里程碑、看板、自动化、锁定/置顶
* **Wiki / 项目文档** — 仓库内置 Wiki、自定义主页
* **Forgejo Actions / 工作流** — `.forgejo/workflows/*.yml` CI/CD，自托管 runner 支持
* **全文搜索** — 代码、Issue、Commit、Wiki 多维度搜索
* **Webhook / 外部集成** — 推送代码/Issue 等事件到 Slack、企微、外部 CI 等
* **SSH / HTTPS 访问** — 个人访问令牌 (PAT)、SSH key 管理
* **组织 / 团队** — 多级权限、团队仓库共享

如果你熟悉 GitHub，这里的操作习惯几乎完全一致。详细使用文档可参考上游 [Forgejo Docs](https://forgejo.org/docs/latest/)。

## HackForger 平台扩展模块

这些是 HackForger 在 Forgejo 之外新增的通用协作模块：

| 模块 | 说明 | 访问路径 |
|------|------|---------|
| 黑客松 Hackathon | 活动创建、报名、赛道、评委、评分、排行榜 | `/explore/hackathons` |
| 悬赏 Bounty | Issue 关联、申请/接单、托管、多人竞争奖金 | `/explore/bounties` |
| 资助 Grant | 资助轮次、项目申请、评审、分配 | `/explore/grants` |
| 积分 Credits | 平台统一积分账本、兑换商店、订单记录 | `/credits` |
| 提交作品 Submissions | 跨活动的作品聚合浏览 | `/explore/submissions` |
| 社区动态 Feed | 活动/悬赏/资助/积分事件流 | `/` 控制面板 |
| 声誉 Reputation | 用户参与度量、排行榜、层级徽章 | `/explore/reputation` |

## 开发者资源

* **API 参考** — `/api/swagger` 查看完整 OpenAPI 文档；HackForger 专用接口以 `/api/v1/hackforger/` 开头
* **CLI 工具** — `hackforger-cli`（仓库根目录，Go 实现，用于批量创建/管理活动）
* **AI Agent 集成** — 通过 PAT 授权让 AI 代表用户操作平台 API

## 需要帮助？

* 平台功能问题 —— 阅读本页“平台玩法”区
* Git / 仓库问题 —— 参考 [Forgejo Docs](https://forgejo.org/docs/latest/)
* 发现 bug —— 在 [HackForger GitHub 仓库](https://github.com/HackForger/hackforger/issues) 提 issue
