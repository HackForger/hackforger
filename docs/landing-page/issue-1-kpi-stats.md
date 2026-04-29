## 背景

新落地页 (`custom/public/assets/landing/index.html`, spec 见 `docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md`) 的 "GLOBAL LEAGUE PULSE" 区块有 3 张 KPI 卡片：

| 卡片 | 字段含义 | 当前显示 | HTML 定位锚点 |
|---|---|---|---|
| 算力消耗 | 大赛累计消耗算力虾粮 | "即将发送" 占位 | `pulse-compute-card` |
| 总计发放 | 已发放龙虾数 | "即将发送" 占位 | `pulse-issued-card` |
| 正在进行 | 活跃在线龙虾数 | "即将发送" 占位 | `pulse-live-card` (id `pulse-live-count`) |

落地页设计者原始注释（HTML 文件内）：「数字暂时显示'即将发送'，停用随机更新」。本期实现保留占位文案，未对接实数据。

## 需要解决

### 1. 数据语义定义（必须先于实现）

每个 KPI 的精确定义需要明确：

**算力消耗**
- SUM 哪张表？credit redeem 全表 vs 仅本届联赛？
- 范围：所有 hackathon 累计 vs 仅"本届联赛"相关？
- 时间窗：永久累计 vs 单赛季？

**总计发放**
- SUM credit issue 交易？
- 包含主办方手动发放 + 自动比赛奖励？
- 范围同上

**正在进行**
- COUNT 什么？hackathon participant？status=hacking 的？
- 是否含已注册但未提交的？
- 是否需要"在线"判断（如最近 N 分钟有活动）？

### 2. 实现要点

- 新增 public API: `GET /api/v1/hackforger/landing/stats`，无需鉴权
- 服务端缓存 5 分钟（聚合查询，避免每次访问打 DB）
- 落地页 JS fetch 后替换 `即将发送` 文案；失败时保留占位文案（fallback）

## 验收

- [ ] 数据语义文档化（本 issue 评论或独立 spec）
- [ ] 测试员根据语义建测试 fixture 模板
- [ ] 3 个 KPI 在任意符合条件的交易后 5 分钟内更新
- [ ] 公开访问，无需登录
- [ ] 接口被速率限制保护
- [ ] E2E 验证（agent-browser，screenshots）

## 相关

- Spec: `docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md`
- Implementation PR: TBD（合并后回填）
- 跟进 issue: 「赛项&命题预览」自动同步
