## 背景

落地页 (`custom/public/assets/landing/index.html`) "赛项&命题预览" 区块 (section `#schedule`) 包含 12 张 Wave 卡片：

- S1 数智OPC加速赛 × 4 Waves
- S2 跨境OPC加速赛 × 4 Waves
- S3 全球青年培育赛 × 4 Waves

**本期已做的连接 (L0)**：
- ✅ 每张 Wave "立即报名" 按钮通过 `HACKFORGER_LANDING_CONFIG.stages` 映射到 `/hackathon/{slug}`
- ✅ slug 为 `tbd` 或 hackathon 不存在时弹"活动还没创建"
- ✅ active/disabled 状态由 config 驱动（event delegation）

**本期未做**：
- ❌ Wave 名称（"初赛/Wave 1" 等）硬编码 HTML
- ❌ 日期徽章（"#报名 4.29-5.5" 等）硬编码
- ❌ 赛区/赛道文案（"数字文化赛道" 等）硬编码
- ❌ active/disabled 状态需要手工编辑 config

## 痛点

- Hackathon 阶段切换（registration_open → hacking）时，落地页按钮仍显示"立即报名"，用户点了才发现报名已截止
- 修改任何日期/状态需要 commit HTML，部署链路重，运营无法自助

## 提议

1. 新增 API: `GET /api/v1/hackforger/landing/stages?league=<slug>`，返回数组：
   ```json
   [
     {
       "stage_id": "s1-w1",
       "hackathon_slug": "league-2-s1-w1",
       "name": "初赛/Wave 1",
       "registration_window": { "start": "2026-04-29", "end": "2026-05-05" },
       "current_phase": "registration_open",
       "tracks": ["数字文化", "数字营销", ...]
     },
     ...
   ]
   ```
2. 落地页加载时 JS 调此 API，hydrate 12 张卡片（替换文本 + 状态徽章 + active/disabled）
3. 失败时退回 HTML 内的硬编码内容（resilience；不破坏页面渲染）
4. 状态徽章自动随 hackathon phase 变化（#报名 → #开发 → #互评 → #公布晋级）

## 待回答的产品问题

- HackForger 当前数据模型有 "League" 概念吗？还是每个 hackathon 独立？
- 如果没有 League，怎么把 "S1 Wave 2" 关联到某届联赛？
  - 选项 A：命名约定（slug 必须是 `league-{N}-s{X}-w{Y}` 格式）
  - 选项 B：新加 `league_id` 外键到 hackathon 表
  - 选项 C：新加独立 `landing_binding` 表（参考用户提的"运营人员系统内绑定落地页"想法）
- 一届 league 是否固定 12 个 Wave？还是可变？

## 关联：运营人员"绑定落地页"功能

用户在 brainstorming 阶段提到希望"创建 hackathon 后，点击'绑定落地页'，slug 批量填入 placeholder"。
这是同一问题的产品视角——本 issue 的 schema 决策直接影响该 UI 的实现路径。
建议两个一起 spec。

## 验收

- [ ] League 数据模型决策落定（issue 评论或独立 spec）
- [ ] 12 张卡片实时反映 hackathon DB 状态
- [ ] 状态徽章自动切换
- [ ] Fallback 不破坏渲染
- [ ] E2E 验证

## 相关

- Spec: `docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md`
- Implementation PR: TBD（合并后回填）
- 落地页 stages 区段位置: `custom/public/assets/landing/index.html` 中 `<section id="schedule">`
