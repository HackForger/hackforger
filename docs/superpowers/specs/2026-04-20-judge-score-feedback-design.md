# Judge Score Feedback + Rubric Validation

**Date**: 2026-04-20
**Issues**: #27 (HF-009 评分提交没有反应), #28 (HF-010 预览结果没有显示)
**Goal**: 关闭"点击 Submit Scores 没反应"与"feed 显示已评审但没分数"两个反馈闭环缺失；同时堵住 publish-time 缺少评分标准的源头漏洞。

---

## Context

### 测试员报告

- #27: 评委点 "Submit Scores" 没反应、分数未保存
- #28: "预览结果"没有显示内容

### 数据库事实（2026-04-20 复现）

- Hackathon 84 (`黑客松提交`) 配置了 criteria 91 `创新设计`，有 2 条 `hackathon_judge_score` 记录，`updated_unix` 为本次复现当日 → **评分实际上被保存了**。
- Hackathon 72 (`测试评审`) 零 criteria，进入 Judging 后仍能打开 judge 页并点 Submit，后端 upsert 循环零次但返回 200 + 发 feed → **静默 no-op**。
- 几个已发布活动（87/71/76/79）没有 criteria，证明 publish-time 没做校验。

### 根因

| 症状 | 根因 |
|---|---|
| #27 点 Submit 无反应 | 成功路径反馈弱：仅"1/M"计数 + 右上角小徽章；无 toast、无时间戳、无摘要回显 |
| #27 feed 显示已评但无分数 | 空 rubric 情形 —— 后端 upsert 零行但 feed 照发；验证副作用数量 |
| #28 预览结果没显示 | `CalculateRanks` 对空 rubric 的 track `continue`，结果 map 空；与 #27 同源 |

### 决策

- **不做自愈 migration 修 `status_cache`** —— dev 环境直接 SQL 删违规数据
- **不做 keyboard shortcut** —— 用户明确跳过
- **不做 JudgeScoreCard 大重构** —— 维持单组件内增量改
- **前端 Vue 调 web route**（不走 `/api/v1/`）—— 遵守现有 CSRF/session 方案
- **所有新文案必须双 locale**（`hackforger.*` namespace）

---

## Non-Goals

- 允许 Judging 阶段修改或删除已有 criteria（仅允许**新增**以救济错发布的活动）
- 改变 leaderboard 的公开/私有边界（per-judge 分数仍私有）
- 评委对 submission 评分后的评论系统
- 让管理员查看 per-judge 的 score/comment

---

## Scope Summary

| # | 区块 | 优先级 | 描述 |
|---|---|---|---|
| A | 后端数据验证 | P0 | ErrNoCriteria 发布门控、SubmitScores 拒绝空 rubric、校验写入行数再发 feed |
| B | Judge UI 反馈 | P0 | Toast、摘要折叠卡、Update Scores 文案切换、Sticky 进度条、错误 banner |
| C | Judging 阶段救济 | P1 | 允许 AddCriteria 直到 Judging；Update/Delete 仍在 `< 3` 锁定 |
| D | Feed 跳转 | P1 | HackathonScored → leaderboard；个人 activity 特例跳 judge 页（P2） |
| E | 清理脚本 | P1 | scripts/cleanup-invalid-hackathons.sql 一次性执行 |

---

## A. Backend Data Validation

### A.1 PublishHackathon 增加 criteria 检查

**File**: `services/hackforger/hackathon.go` — 在 tracks/phases 校验之后、`SetIsPublished` 之前插入

**新增 typed error**（放在 `hackathon.go` 靠近 `ErrNoTracks`）：

```go
type ErrNoCriteria struct {
    HackathonID int64
    TrackID     int64  // 0 = 全局；非 0 = 具体 track 无 enabled criteria
}
func (e ErrNoCriteria) Error() string {
    if e.TrackID == 0 {
        return fmt.Sprintf("hackathon has no scoring criteria [id: %d]", e.HackathonID)
    }
    return fmt.Sprintf("track has no enabled scoring criteria [hackathon: %d, track: %d]", e.HackathonID, e.TrackID)
}
// Unwrap lets errors.Is detect invalid-argument and route to HTTP 400 — matches
// the dominant pattern in models/hackforger/ and services/hackforger/hackathon_criteria.go.
func (e ErrNoCriteria) Unwrap() error { return util.ErrInvalidArgument }
func IsErrNoCriteria(err error) bool { _, ok := err.(ErrNoCriteria); return ok }
```

**校验逻辑**：
- 查 hackathon 的 `hackathon_judge_criteria`，若为 0 → `ErrNoCriteria{HackathonID: h.ID}`
- 对每个 track 查 `GetEffectiveRubric` —— 若所有 criteria 都被 `hackathon_track_criteria.enabled=false` 覆盖 → `ErrNoCriteria{HackathonID: h.ID, TrackID: t.ID}`

**Handler** `ManagePublish` (`routers/web/hackforger/hackathon.go`) 新增 `IsErrNoCriteria` 分支：
- `TrackID == 0` → `hackforger.hackathon.error.publish_requires_criteria`
- `TrackID != 0` → `hackforger.hackathon.error.track_requires_criteria %s`（带 track 名）

### A.2 SubmitScores 拒绝空 rubric

**File**: `services/hackforger/hackathon_judge.go` — `SubmitScores` 在 step 4 `GetEffectiveRubric` 后：

```go
if len(rubric) == 0 {
    return ErrNoRubricConfigured{TrackID: sub.TrackID}
}
```

**新增 typed error**：

```go
type ErrNoRubricConfigured struct{ TrackID int64 }
func (e ErrNoRubricConfigured) Error() string {
    return fmt.Sprintf("no scoring rubric configured for track [track: %d]", e.TrackID)
}
func (e ErrNoRubricConfigured) Unwrap() error { return util.ErrInvalidArgument }
func IsErrNoRubricConfigured(err error) bool { _, ok := err.(ErrNoRubricConfigured); return ok }
```

**Handler** `JudgeScoresPost` 分支返回 `400` + `hackforger.hackathon.error.no_rubric_configured`。

### A.3 Feed 只在实际写入时发

**File**: `services/hackforger/hackathon_judge.go` — 修改 step 7 事务使之返回"实际 upsert 的行数"：

```go
var writeCount int
if err := db.WithTx(ctx, func(txCtx context.Context) error {
    for _, s := range scores {
        if _, ok := rubricMap[s.CriteriaID]; !ok {
            continue
        }
        // ... upsert ...
        writeCount++
    }
    return nil
}); err != nil { return err }

if writeCount == 0 {
    return ErrNoRubricConfigured{TrackID: sub.TrackID}  // defensive
}
// step 8 Publish feed event ← 保留原样
```

> A.2 已在服务入口拦截空 rubric，A.3 是深度防御：rubric 存在但 payload 不含任何有效 criteria 时也拒绝。

### A.4 Criteria UI gate 调整（配合 C）

**File**: `templates/hackforger/hackathon/manage.tmpl:214`

```
{{if lt .Hackathon.StatusCache 3}}  →  {{if le .Hackathon.StatusCache 3}}
```

`<=3` 让 Judging 阶段仍显示 criteria 管理区。配合 C.1 服务层放宽。

### A.5 Criteria 表"只能新增"守卫

**File**: `services/hackforger/hackathon_criteria.go:65`

```go
func checkCriteriaModifiable(ctx, hackathonID) error {
    // 当前：StatusCache >= Judging → 拒绝
    // 改为：给 3 个分支，分别 AddCriteria / UpdateCriteria / DeleteCriteria 调用时传入 op
}
```

**重构为**：

```go
type criteriaOp int
const (
    opAdd criteriaOp = iota
    opUpdate
    opDelete
)

func checkCriteriaModifiable(ctx context.Context, hackathonID int64, op criteriaOp) error {
    h, err := hackforger_model.GetHackathonByID(ctx, hackathonID)
    if err != nil { return err }
    switch {
    case h.StatusCache >= HackathonStatusFinished:
        return ErrInvalidHackathonPhase{...}  // 锁 Finalized/Cancelled
    case h.StatusCache == HackathonStatusJudging && op != opAdd:
        return ErrInvalidHackathonPhase{...}  // Judging 仅允许 Add
    }
    return nil
}
```

三个 caller 传入对应 op。

---

## B. Judge UI Feedback

### B.1 Vue 组件 `JudgeScoreCard.vue` 增强

**File**: `web_src/js/components/hackforger/JudgeScoreCard.vue`

变更清单：

**1. 成功/失败 toast** — 已确认 `web_src/js/modules/toast.js` 提供 `showInfoToast` / `showErrorToast` / `showWarningToast`（upstream Forgejo 模块），直接 import：

```js
import {showInfoToast, showErrorToast} from '../../modules/toast.js';
import {formatDatetime} from '../../utils/time.js';
// NOTE: Forgejo's `showInfoToast` renders **green with octicon-check** (toast.js:11-15);
// it is the canonical success toast. No `showSuccessToast` exists.
// ...
async submitScores(submissionId) {
  // ...
  if (resp.ok) {
    this.saved[submissionId] = true
    this.lastSavedAt[submissionId] = Date.now()
    showInfoToast(this.messages.scoreSaved)
  } else {
    const data = await resp.json()
    this.errors[submissionId] = data.message || this.messages.errorGeneric
    this.globalError = this.errors[submissionId]
    showErrorToast(this.globalError)
  }
}
```

**2. 已有评分摘要卡（折叠态）**：

```vue
<!-- 替代当前直接渲染表单 -->
<div v-if="saved[sub.id] && !expanded[sub.id]" class="hf-card-body hf-score-summary">
  <div class="tw-flex tw-items-center tw-gap-3">
    <svg class="svg octicon-check-circle-fill tw-text-green"/>
    <span>{{ messages.scoreSavedAt.replace('%s', formatTime(lastSavedAt[sub.id])) }}</span>
  </div>
  <div class="tw-mt-2 tw-grid" style="grid-template-columns: repeat(auto-fit, minmax(140px, 1fr))">
    <div v-for="c in activeRubric" :key="c.criteria_id">
      <span class="tw-text-gray">{{ c.name }}</span>
      <strong>{{ scores[sub.id][c.criteria_id].score }} / {{ c.max_score }}</strong>
    </div>
  </div>
  <button class="hf-btn hf-btn-outline" @click="expanded[sub.id] = true">
    {{ messages.updateScores }}
  </button>
</div>
<div v-else><!-- 现有表单 --></div>
```

**3. Button 文案动态切换**：

```vue
<button class="hf-btn hf-btn-primary" @click="submitScores(sub.id)">
  {{ saved[sub.id] ? messages.updateScores : messages.submitScores }}
</button>
```

**4. 顶部 sticky 进度条 + "下一个未评"**：

Sticky top uses a concrete pixel offset (Forgejo convention — `web_src/css/repo.css:1383,2405,2460`). No `--topbar-height` CSS variable exists in the project. Judge page has no secondary tabbar above this region, so `top: 0` is correct:

```vue
<div class="hf-judge-progress-sticky" style="position: sticky; top: 0; z-index: 10; background: var(--color-box-body);">
  <span>{{ scoredCount }} / {{ activeSubmissions.length }} {{ messages.progressLabel }}</span>
  <div class="ui indicating progress"><div class="bar" :style="{width: progressPercent + '%'}"></div></div>
  <button v-if="nextUnscoredSubId" @click="scrollToSubmission(nextUnscoredSubId)">
    {{ messages.nextUnscored }}
  </button>
</div>
```

**5. 顶部错误 banner**（不再埋在表单底部）：

```vue
<div v-if="globalError" class="ui error message visible tw-mb-4" role="alert">
  {{ globalError }}
</div>
```

**6. 空 rubric 提示**（组织者侧问题，评委可读）：

```vue
<div v-if="!activeRubric.length" class="ui warning message">
  {{ messages.rubricNotConfigured }}
</div>
```

### B.2 Vue 组件 i18n 策略（per-key data-locale-*）

当前 `JudgeScoreCard.vue` 所有字符串都硬编码英文。遵循 **Forgejo 现有惯例**（见 `templates/repo/actions/view.tmpl`、`templates/repo/contributors.tmpl`、`web_src/js/features/code-frequency.js`）：逐键 `data-locale-*` 属性，不用 JSON bag。`templates/base/head_script.tmpl:34-45` 官方文档说明 i18n 三种传递渠道，module-specific 字符串应走 data-attribute。

**`templates/hackforger/hackathon/judge.tmpl`** 逐键注入：

```html
<div id="hackforger-judge-scorecard"
     data-hackathon-slug="{{.Hackathon.Slug}}"
     data-tracks="{{.TracksJSON}}"
     data-submissions="{{.SubmissionsJSON}}"
     data-rubrics="{{.RubricsJSON}}"
     data-locale-score-saved="{{ctx.Locale.Tr "hackforger.hackathon.judge.scores_saved"}}"
     data-locale-score-saved-at="{{ctx.Locale.Tr "hackforger.hackathon.judge.score_saved_at"}}"
     data-locale-update-scores="{{ctx.Locale.Tr "hackforger.hackathon.judge.update_scores"}}"
     data-locale-submit-scores="{{ctx.Locale.Tr "hackforger.hackathon.judge.submit_scores"}}"
     data-locale-progress-label="{{ctx.Locale.Tr "hackforger.hackathon.judge.progress_detail"}}"
     data-locale-next-unscored="{{ctx.Locale.Tr "hackforger.hackathon.judge.next_unscored"}}"
     data-locale-rubric-not-configured="{{ctx.Locale.Tr "hackforger.hackathon.judge.rubric_not_configured"}}"
     data-locale-error-generic="{{ctx.Locale.Tr "hackforger.hackathon.judge.error_generic"}}">
</div>
```

**`init.js`** 组装 messages 对象：

```js
createApp(JudgeScoreCard, {
  // ...existing
  messages: {
    scoreSaved:          judgeEl.getAttribute('data-locale-score-saved'),
    scoreSavedAt:        judgeEl.getAttribute('data-locale-score-saved-at'),
    updateScores:        judgeEl.getAttribute('data-locale-update-scores'),
    submitScores:        judgeEl.getAttribute('data-locale-submit-scores'),
    progressLabel:       judgeEl.getAttribute('data-locale-progress-label'),
    nextUnscored:        judgeEl.getAttribute('data-locale-next-unscored'),
    rubricNotConfigured: judgeEl.getAttribute('data-locale-rubric-not-configured'),
    errorGeneric:        judgeEl.getAttribute('data-locale-error-generic'),
  },
}).mount(judgeEl);
```

**Vue 组件** `props: { messages: {type: Object, default: () => ({})} }`；访问用 `this.messages.scoreSaved`（camelCase）。带参 key 用 `String.prototype.replace` 拼接（Forgejo locale 的 `%d`/`%s` 占位符保留，JS 端手动替换）。

### B.2a 新增 computed / methods

```js
computed: {
  // ...existing
  nextUnscoredSubId() {
    const s = this.activeSubmissions.find((s) => !this.saved[s.id]);
    return s ? s.id : null;
  },
},
methods: {
  // ...existing
  scrollToSubmission(subId) {
    const el = document.getElementById('submission-' + subId);
    if (el) el.scrollIntoView({behavior: 'smooth', block: 'center'});
  },
  formatTime(ms) {
    if (!ms) return '';
    // Use Forgejo's locale-aware formatter — respects user 12/24h preference.
    return formatDatetime(new Date(ms), {hour: 'numeric', minute: '2-digit'});
  },
},
```

每条 submission 卡片 root 元素必须带 `:id="'submission-' + sub.id"` 以支持 scroll 和未来的锚点跳转（配合 D.2）。

### B.3 Data state 新增

```js
data() {
  return {
    // ...existing
    lastSavedAt: {}, // {subId: timestamp_ms}
    expanded: {},    // {subId: bool} — 默认 !saved
    globalError: '', // 顶部 banner 文案；toast 之外的持续可见错误
  }
}
```

### B.4 提交成功后的交互细节

1. `saved[subId] = true`
2. `lastSavedAt[subId] = Date.now()`
3. `expanded[subId] = false`（自动折叠）
4. Toast 显示 2 秒
5. 若还有未评 submission，显示 "下一个 →" 浮动按钮滚动锚定

---

## C. Judging 阶段救济

### C.1 允许 AddCriteria 到 Judging

已在 A.5 描述（`checkCriteriaModifiable` 传 `opAdd`）。

### C.2 Manage 页面 Judging 状态的视觉告警

**File**: `templates/hackforger/hackathon/manage.tmpl:213` 附近

当 `StatusCache == 3` 且区块显示时，在表格顶部加 info message：

```
⚠️ 当前评审阶段已开始，你可以新增评分项，但无法修改或删除已有项目。
```

i18n key: `hackforger.hackathon.manage.criteria.judging_add_only`

---

## D. Feed 跳转

### D.1 Community feed 的评分事件改指 leaderboard

**File**: `modules/templates/helper.go:227` 的 `HackforgerEntityURL` 不改（其他地方有复用）；

在 `templates/hackforger/feed/community_feeds.tmpl:15` 新增 action→target 的条件：

```go
{{else if eq .OpType 33}}
  {{ctx.Locale.Tr "hackforger.feed.hackathon_scored" .EntityName
    (printf "%s/leaderboard" (HackforgerEntityURL .EntityType .EntitySlug))}}
```

（locale 字符串里 `%[2]s` 位置替换为 leaderboard URL）

### D.2 个人 profile 活动流单独处理（P2 — 本次可选实现）

**File**: `templates/hackforger/feed/community_feeds.tmpl` 或新建 `personal_feeds.tmpl`

在个人 profile 活动 tab 的模板里，对 `.OpType == 33 && .ActUser.ID == $.Doer.ID` 的行，URL 改为：

```
/hackathon/{slug}/judge#submission-{Content.Extra.submission_id}
```

**前置准备**：`JudgeScoreCard.vue` 每张 submission 卡要加 `:id="'submission-' + sub.id"` 锚点。

**风险**：如果活动已 Finalized，评委仍会被路由到 judge 页，但服务端校验 `AllowsAction("score")` 会 403。为此：handler 检测若不是 score 阶段，302 到 leaderboard。

> **实现策略**：先只做 D.1，D.2 列为 follow-up。

---

## E. 一次性清理 SQL 脚本

### E.1 文件位置

`scripts/cleanup-invalid-hackathons.sql`（新建 scripts 目录如不存在）

### E.2 清理对象（范围按用户 2026-04-20 决策）

**违规活动**（is_published=1 AND criteria 为 0）：
- id 87 (2nd-dishui-opc-w3)
- id 72 (judge / 测试评审)
- id 71 (dev / 测试开发阶段)
- id 76 (test-time / 测试发布活动)
- id 79 (123 / 123123)

**孤儿 feed**（op_type=33 HackathonScored 但无对应 score 行）：

```sql
DELETE FROM hackforger_action
 WHERE op_type = 33
   AND NOT EXISTS (
     SELECT 1 FROM hackathon_judge_score
      WHERE hackathon_id = hackforger_action.entity_id
        AND judge_id    = hackforger_action.user_id
   );
```

### E.3 级联顺序（手工执行，SQLite fk 不强制）

```sql
BEGIN;

-- 1. 收集 track ids
CREATE TEMP TABLE bad_hackathons AS
  SELECT id FROM hackathon WHERE id IN (87, 72, 71, 76, 79);

CREATE TEMP TABLE bad_tracks AS
  SELECT id FROM hackathon_track WHERE hackathon_id IN (SELECT id FROM bad_hackathons);

-- 2. 从叶子表开始删
DELETE FROM hackathon_judge_score WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_track_criteria WHERE track_id IN (SELECT id FROM bad_tracks);
DELETE FROM hackathon_judge_criteria WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_submission WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_registration WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_judge WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_track WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM phase WHERE activity_kind='hackathon'
                    AND activity_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackforger_action WHERE entity_type='hackathon'
                                AND entity_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon WHERE id IN (SELECT id FROM bad_hackathons);

-- 3. 再清一次孤儿 feed（覆盖其他活动历史残留）
DELETE FROM hackforger_action
 WHERE op_type = 33
   AND NOT EXISTS (
     SELECT 1 FROM hackathon_judge_score
      WHERE hackathon_id = hackforger_action.entity_id
        AND judge_id    = hackforger_action.user_id
   );

COMMIT;
```

**执行方式**：手工 `sqlite3 data/forgejo.db < scripts/cleanup-invalid-hackathons.sql`，**不进 migration 链**（用户决策：dev + 内部 instance 自己跑）。

**回滚**：脚本自己不做 backup；执行前手工 `cp data/forgejo.db data/forgejo.db.bak.$(date +%s)`。

---

## i18n keys

所有新 key 同时写入 `options/locale/locale_en-US.ini` 和 `locale_zh-CN.ini` 的 `[hackforger]` section。**优先复用已存在的 key**（实施时须检查 `locale_en-US.ini` 当前内容）。

### 复用已存在的 key（不要覆写、不要重复定义）

| Key | 已存在于 | 用途 |
|---|---|---|
| `hackforger.hackathon.judge.submit_scores` | locale_en-US.ini:4177 ("Submit Scores") | Button 文案（未评审时）|
| `hackforger.hackathon.judge.scores_saved` | 4178 ("Scores saved successfully") | Success toast |
| `hackforger.hackathon.judge.progress_detail` | 4176 ("%d of %d submissions scored") | Sticky 进度条文案（JS 端 `.replace('%d',N).replace('%d',M)`） |
| `hackforger.hackathon.error.incomplete_rubric` | 4193 | Deep defense 当 payload 过滤后为空 |

### 新增 key

| Key | en-US | zh-CN |
|---|---|---|
| `hackforger.hackathon.error.publish_requires_criteria` | Cannot publish: at least one scoring criterion is required | 无法发布：至少需要配置一个评分标准 |
| `hackforger.hackathon.error.track_requires_criteria` | Track "%s" has no enabled scoring criteria | 赛道 "%s" 没有启用任何评分项 |
| `hackforger.hackathon.error.no_rubric_configured` | No scoring criteria configured for this track — please contact the organizer | 该赛道尚未配置评分标准，请联系组织者 |
| `hackforger.hackathon.manage.criteria.judging_add_only` | Judging has started. You can add new criteria but cannot modify or delete existing ones. | 评审阶段已开始，你可以新增评分项，但无法修改或删除已有项目 |
| `hackforger.hackathon.judge.score_saved_at` | Saved at %s | 保存于 %s |
| `hackforger.hackathon.judge.update_scores` | Update Scores | 更新分数 |
| `hackforger.hackathon.judge.next_unscored` | Next unscored → | 下一个未评 → |
| `hackforger.hackathon.judge.rubric_not_configured` | This track has no scoring criteria configured. Ask the organizer to add criteria. | 该赛道尚未配置评分标准，请组织者补充后再评分 |
| `hackforger.hackathon.judge.error_generic` | Could not save scores. Please try again. | 保存分数失败，请重试 |

### 注意

- `hackforger.hackathon.error.no_criteria`（4196）已有、语义贴近但专指 "before starting judging" 场景。publish gate 用**新 key** `publish_requires_criteria` 而非复用 `no_criteria`，因为错误触发时机不同。
- `hackforger.hackathon.error.criteria_locked`（4195）已有、与 Judging 阶段 Update/Delete 拒绝的场景语义一致，**复用这条**；不要为 C.1 新增额外 key。

---

## Testing Strategy

### Unit tests

- `services/hackforger/hackathon_test.go`
  - `TestPublishHackathon_NoCriteria` → `ErrNoCriteria{HackathonID, TrackID=0}`
  - `TestPublishHackathon_TrackHasNoEnabledCriteria` → `ErrNoCriteria{TrackID=X}`
- `services/hackforger/hackathon_judge_test.go`
  - `TestSubmitScores_EmptyRubric` → `ErrNoRubricConfigured`
  - `TestSubmitScores_PayloadAllFiltered` → `ErrNoRubricConfigured`（深度防御）
  - `TestSubmitScores_Success_PublishesFeed`（确保正常路径未 regress）
- `services/hackforger/hackathon_criteria_test.go`
  - `TestAddCriteria_AllowedInJudging`
  - `TestUpdateCriteria_BlockedInJudging` → `ErrInvalidHackathonPhase`
  - `TestDeleteCriteria_BlockedInJudging` → `ErrInvalidHackathonPhase`

### Integration tests

- `tests/integration/hackforger_publish_test.go` 新增：尝试发布无 criteria 活动，断言 400 + flash error i18n key
- `tests/integration/hackforger_judge_test.go` 新增：空 rubric 活动 POST /judge/:sid/scores 返回 400

### E2E（web-only，agent-browser）

必须通过 UI 复现原 bug 场景（用户偏好 `feedback_e2e_web_mandatory.md`）。

---

## E2E Prompt（必须放在 `docs/tests/e2e/tasks/`）

**File**: `docs/tests/e2e/tasks/issue-27-28-judge-score-feedback-e2e.md`

```markdown
# Issue #27 + #28 Judge Score Feedback E2E

**目标**：验证评分反馈闭环与 publish 校验同时生效。
**工具**：agent-browser via http://localhost:3000
**前置**：当前 worktree 已 `make frontend && make backend`，并已执行 `sqlite3 data/forgejo.db < scripts/cleanup-invalid-hackathons.sql`。
**账号**：hackforger / admin1234（admin+organizer+judge）

## 测试流程

### TC1: Publish 拒绝无 criteria
1. 以 hackforger 登录，新建 Draft hackathon "E2E评分回归"，添加 1 个 track，不添加 criteria
2. 管理页点发布
3. 断言：顶部 flash 显示红色 "无法发布：至少需要配置一个评分标准"
4. 截图：`screenshots/issue-27-28/tc1-publish-blocked.png`

### TC2: Judging 阶段新增 criteria
1. 接 TC1，在 Draft 添加 1 个 criterion ("创新" 10/25) + 把 phase 推到 Judging（SQL 调 end_time 过期 + sync status_cache）
2. 回管理页断言 criteria 表格仍可见，顶部警告 "评审阶段已开始，你可以新增..."
3. 再加一个 criterion ("完成度" 10/20) — 成功
4. 尝试编辑第一个 criterion — 断言按钮被禁用或提示
5. 截图：`screenshots/issue-27-28/tc2-judging-add-only.png`

### TC3: SubmitScores 成功反馈
1. 在 TC2 活动中提交一条 submission（hackforger 以 hacker 身份）
2. 以 hackforger (自己也是 judge) 访问 /hackathon/{slug}/judge
3. 填入分数，点 "Submit Scores"
4. 断言：
   - 顶部绿色 toast "分数已保存"
   - 表单折叠为摘要卡，显示 "保存于 HH:MM"
   - 按钮文案变 "更新分数"
   - Sticky 进度条 1/1
5. 截图：`screenshots/issue-27-28/tc3-success-feedback.png`

### TC4: 刷新后摘要回显
1. 刷新 judge 页
2. 断言：submission 卡片默认折叠为摘要、显示之前填的分数
3. 截图：`screenshots/issue-27-28/tc4-summary-persist.png`

### TC5: 错误场景
1. 断开网络后点 Submit
2. 断言：红色 banner 在顶部显示 "保存分数失败，请重试"
3. 恢复网络、重试、成功
4. 截图：`screenshots/issue-27-28/tc5-error-banner.png`

### TC6: Feed 跳转
1. 在 dashboard feed 或 community feed 找到 "hackforger 评审了 E2E评分回归 的提交" 的条目
2. 点击活动名
3. 断言：跳到 `/hackathon/e2e评分回归/leaderboard`，看得到聚合分数
4. 截图：`screenshots/issue-27-28/tc6-feed-to-leaderboard.png`

### TC7: Finalize 预览 (#28)
1. 以 hackforger 进 manage 页，点 "预览结果"
2. 断言：显示每个 track 的排名表 + 加权总分（不再是空白）
3. 截图：`screenshots/issue-27-28/tc7-preview-rankings.png`

## 报告模板

报告保存到 `docs/tests/e2e/reports/issue-27-28-judge-score-feedback-report.md`，按 TC 列出 PASS/FAIL + 截图相对路径 + 关键 HTTP 日志片段。
```

---

## Rollout

1. 主工作区 `make frontend && make backend`
2. 执行 `scripts/cleanup-invalid-hackathons.sql`
3. 启 server，跑 E2E prompt
4. PR 合并前：closes #27 #28
5. 内部 instance 部署后，通知测试员重测"测试评审"活动：由组织者在 Draft 先配 criteria，再走完整流程

---

## Open Questions / Future Work

- **Future**: 管理员查看 per-judge score/comment 的界面（目前仅能 SQL 查）—— 对评分争议处理有价值，但超出本 spec
- **Future**: Finalize 后把 feed 从 "评审了..." 升级成 "最终排名揭晓..."（已有 op_type 51 `hackathon_finalized`，只是未利用）
- **Future**: 个人 activity 的评分 feed 跳 judge 页（D.2），等 JudgeScoreCard 加完锚点后跟进
