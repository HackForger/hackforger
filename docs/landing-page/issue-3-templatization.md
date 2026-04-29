## 背景

HackForger 落地页第一版 (PR TBD, spec 见 `docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md`) 作为单届赛事的定制页面手工编写，文件在 `custom/public/assets/landing/`。每办新一届赛事时，文案、资产、slug 等都需要改动。

讨论过 4 种模板化方案：

| 等级 | 输入 | 输出 | 成本 | 灵活度 |
|---|---|---|---|---|
| L0 · 复制改文案 | 复制目录手动改 | `landing-event-N/` | ⏱ 30min/届 | ⭐⭐⭐⭐⭐（HTML 任意可改） |
| L1 · 字段替换模板 | YAML config + `template.html` | `landing/index.html` | 🛠 2-3 天初次 | ⭐⭐（结构必须固定） |
| L2 · 结构化模板 | DESIGN.md + schema.yaml + assets | 整个站 | 🛠 1-2 周 | ⭐⭐⭐⭐（section 可启用/变体） |
| L3 · CMS 化 | Web 后台编辑 | 实时渲染 | 🛠 4-6 周 | ⭐⭐⭐⭐⭐（运营自助） |

**当前选择 L0**——直到累积至少 2 届真实数据后，再决定 L1+ schema。

## 触发抽象的条件（满足时再启动开发）

- [ ] 已经办了 ≥ 2 届 league，能 diff 出真正的 variable / constant 字段
- [ ] 频率 ≥ 4 届/年（否则 L0 完全够用）
- [ ] 编辑者从开发/设计师扩展到产品/运营（"绑定落地页" UI 由 Issue #STAGES 跟进）

## 反模式提醒

- ❌ **不要凭空设计模板 schema**——第一届做完你以为知道什么会变，第二届才发现真正会变的是没考虑的字段（比如 sponsors 突然加"特别支持"层级）
- ❌ **不要在没真实数据前讨论 CMS**——HackForger 当前没有专职运营团队迭代落地页内容
- ✅ **YAGNI**：用过 2 次再抽象

## 决策建议

- **第 2 届来临时**：手工复制 + diff，记录 diff 内容到本 issue
- **累积 2 届 diff 数据后**：讨论 L1 schema，目标是把 ≥ 80% 的修改通过 YAML 完成
- **L2/L3 暂不考虑**（先验证 L1 是否真有价值）

## 状态

⏸ **Backlog** — 等待第 2 届赛事产生 diff 数据。

## 相关

- Spec: `docs/superpowers/specs/2026-04-29-hackforger-landing-page-design.md`
- Implementation PR: #108
- 关联 issue: #109 (KPI 统计接入) + #110 (赛项&命题预览同步 + 运营绑定 UI)
