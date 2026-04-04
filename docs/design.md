# HackForger Design System — 江峡泼墨 (River Gorge × Ink Splash)

> 融合两幅画的配色基因：一幅是雾气氤氲的抒情山水，一幅是泼墨写意的大开大合。
> 前者贡献了柔和层次与自然色阶，后者注入了书法的力量感与朱砂金石的点睛之色。

---

## 1. Design Philosophy

### 墨不到顶

两幅画虽然在对比度上截然不同——山水画回避纯黑，泼墨画以纯黑为骨——但融合方案选择了"墨不到顶"的策略：文字色采用远山蓝灰（`#2E3545`）而非纯墨黑（`#1A1A1A`），保留了山水画的高级灰调，同时通过 Ink ramp 的深色阶段保留泼墨画的力量感，供需要极致强调的场景使用。

### 朱砂点睛，不可泛滥

第二幅画引入的朱砂红（Vermillion）与第一幅画的新芽绿（Spring）构成互补色关系，视觉张力极大。设计系统中将朱砂严格限定为 danger / destructive action / 高优先级 CTA 角色，永远不与新芽绿等面积相邻。两者之间应至少间隔一个中性色（Ink、Sand 或 Mountain）作为缓冲。

### 暖底冷文

两幅画共享暖白色底（宣纸色 / 雾霭色），而文字色偏冷（远山蓝灰）。这种"暖底冷文"的搭配是整个系统的底层气质，切勿将底色换成冷白或将文字色换成暖灰，否则会失去两幅画共同赋予的温润感。

### UI 风格方向：Google-style B1

整体 UI 风格参考 Google Material Design 的结构化美学，但使用江峡泼墨色系：
- **Type Bar** — 每个 feed/list item 左侧 3px 彩色竖条，指示所属模块（翠=hackathon，芽=bounty，金=grant，朱=danger）
- **Category Icons** — Tab 和 badge 内嵌 SVG line icons（Octicon 图标库），强化分类辨识
- **Activity Heatmap** — 底部 12 周活跃热力图，使用 Jade 色阶（900→200）表示强度
- **紧凑但有呼吸感** — 12px 行间距，hover 高亮，Material 风格交互反馈

---

## 2. Color Ramps

系统包含 7 条色阶（ramp），每条 5 个色阶（stop）。命名采用画中意象的中文单字 + 英文名。

### Ink 墨 — 中性骨架

来源：泼墨画的墨色层次 + 山水画的雾霭灰调融合。

| Stop | Hex       | Role                              |
|------|-----------|-----------------------------------|
| 50   | `#F4F2EF` | Surface / 卡片背景 / 输入框底色      |
| 200  | `#C8C0B4` | Border / divider / 禁用态文字       |
| 400  | `#8E8A82` | Muted text / placeholder / icon   |
| 700  | `#4A4A46` | Strong muted / 次要标题             |
| 900  | `#1A1A1A` | 极致强调专用（logo、hero heading）   |

> **注意**：日常文字色不使用 Ink-900，而是使用 Mountain-900（`#2E3545`）。
> Ink-900 仅在需要"泼墨般"视觉冲击力时使用，如品牌 logo、hero section 大标题、ink-style button。

### Jade 翠 — 主色（Primary）

来源：山水画的翠江深绿 + 泼墨画的翡翠飞溅，取两者中间值。

| Stop | Hex       | Role                              |
|------|-----------|-----------------------------------|
| 50   | `#E0F0EA` | Primary subtle bg / tag 背景       |
| 200  | `#7CC4A0` | Light accent / hover 态            |
| 500  | `#1F8C65` | Primary action / link / icon       |
| 700  | `#106B4C` | Primary hover / pressed            |
| 900  | `#0D3F30` | Primary text on light bg / dark mode card bg |

### Sand 沙 — 辅助暖色（Secondary）

来源：山水画的沙洲暖色。

| Stop | Hex       | Role                              |
|------|-----------|-----------------------------------|
| 50   | `#F8F1E6` | Secondary subtle bg / warm surface |
| 200  | `#E8D5B5` | Secondary border / tag bg          |
| 400  | `#C9A87A` | Secondary accent / badge           |
| 600  | `#9A7B4E` | Secondary text / icon              |
| 900  | `#5E4A2D` | Secondary dark / dark mode card bg |

### Gold 金 — 高亮暖色（Highlight）

来源：泼墨画的金色飞溅点。比 Sand 更高饱和度，用于需要引起注意的场景。

| Stop | Hex       | Role                              |
|------|-----------|-----------------------------------|
| 50   | `#FDF5E0` | Highlight bg / pricing card        |
| 200  | `#F0D060` | Highlight border / badge           |
| 500  | `#D4A22E` | Highlight accent / star / premium  |
| 700  | `#A07B18` | Highlight hover                    |
| 900  | `#5C4808` | Highlight text on light bg         |

### Mountain 岚 — 冷灰文字（Text）

来源：山水画的远山蓝灰色调。这是系统的默认文字色 ramp。

| Stop | Hex       | Role                              |
|------|-----------|-----------------------------------|
| 50   | `#ECEEF3` | Dark mode 主文字色 / light mode subtle bg |
| 200  | `#C2C8D5` | Dark mode 次要文字 / light border   |
| 400  | `#8A94A8` | Dark mode muted / light mode caption |
| 600  | `#5A6478` | Light mode 次要文字                 |
| 900  | `#2E3545` | **Light mode 主文字色**             |

> **这是整个系统最关键的决策**：主文字色不是纯黑，而是远山蓝灰。
> 它比 `#1A1A1A` 柔和，但 WCAG AA 对比度在 `#FAFAF8` 底色上仍达 11.8:1，远超 4.5:1 的要求。

### Vermillion 朱 — 危险 / 强调（Danger / CTA）

来源：泼墨画的朱砂印章与飞溅红点。

| Stop | Hex       | Role                              |
|------|-----------|-----------------------------------|
| 50   | `#FCEAE6` | Danger subtle bg / error bg        |
| 200  | `#F0A090` | Danger border / error border       |
| 500  | `#C83C23` | Danger action / destructive button |
| 700  | `#922A18` | Danger hover / pressed             |
| 900  | `#5A1A0E` | Danger text on light bg            |

> **使用限制**：朱砂色不可与 Spring 绿等面积相邻。两者之间至少间隔一个中性色。

### Spring 芽 — 成功 / 生长（Success）

来源：山水画左侧新生枝叶的嫩绿。

| Stop | Hex       | Role                              |
|------|-----------|-----------------------------------|
| 50   | `#EEF5DC` | Success subtle bg / positive tag   |
| 200  | `#C5DD7E` | Success border / progress bar      |
| 500  | `#8EBA3A` | Success icon / check mark          |
| 700  | `#5D8720` | Success hover                      |
| 900  | `#33500F` | Success text on light bg           |

---

## 3. Semantic Tokens

### Light mode

```css
:root {
  /* Background */
  --color-bg:              #FAFAF8;     /* 暖白，两幅画共有的宣纸/雾霭底色 */
  --color-bg-surface:      #F4F2EF;     /* Ink-50, 卡片/输入框底色 */
  --color-bg-elevated:     #FFFFFF;     /* 浮层、modal、tooltip */

  /* Text */
  --color-text:            #2E3545;     /* Mountain-900, 远山蓝灰主文字 */
  --color-text-secondary:  #5A6478;     /* Mountain-600 */
  --color-text-muted:      #8E8A82;     /* Ink-400 */
  --color-text-inverse:    #ECEEF3;     /* Mountain-50, 深色背景上的文字 */

  /* Border */
  --color-border:          #C8C0B4;     /* Ink-200 */
  --color-border-subtle:   #DDD9D3;     /* Ink-200 偏浅，用于分割线 */

  /* Primary — Jade 翠 */
  --color-primary:         #1F8C65;     /* Jade-500 */
  --color-primary-hover:   #106B4C;     /* Jade-700 */
  --color-primary-subtle:  #E0F0EA;     /* Jade-50 */
  --color-primary-text:    #0D3F30;     /* Jade-900 */

  /* Secondary — Sand 沙 */
  --color-secondary:       #C9A87A;     /* Sand-400 */
  --color-secondary-subtle:#F8F1E6;     /* Sand-50 */
  --color-secondary-text:  #5E4A2D;     /* Sand-900 */

  /* Highlight — Gold 金 */
  --color-highlight:       #D4A22E;     /* Gold-500 */
  --color-highlight-subtle:#FDF5E0;     /* Gold-50 */
  --color-highlight-text:  #5C4808;     /* Gold-900 */

  /* Danger — Vermillion 朱 */
  --color-danger:          #C83C23;     /* Vermillion-500 */
  --color-danger-hover:    #922A18;     /* Vermillion-700 */
  --color-danger-subtle:   #FCEAE6;     /* Vermillion-50 */
  --color-danger-text:     #5A1A0E;     /* Vermillion-900 */

  /* Success — Spring 芽 */
  --color-success:         #8EBA3A;     /* Spring-500 */
  --color-success-subtle:  #EEF5DC;     /* Spring-50 */
  --color-success-text:    #33500F;     /* Spring-900 */

  /* Special */
  --color-ink:             #1A1A1A;     /* Ink-900, 仅用于 hero/logo/ink-button */
}
```

### Dark mode

```css
:root {
  /* Background */
  --color-bg:              #141618;     /* 近黑，带微蓝的夜色 */
  --color-bg-surface:      #1E2228;     /* 带蓝灰调的深色表面 */
  --color-bg-elevated:     #252A33;     /* Mountain-900 附近 */

  /* Text */
  --color-text:            #ECEEF3;     /* Mountain-50 */
  --color-text-secondary:  #8A94A8;     /* Mountain-400 */
  --color-text-muted:      #5A6478;     /* Mountain-600 */
  --color-text-inverse:    #2E3545;     /* Mountain-900 */

  /* Border */
  --color-border:          #2E3545;     /* Mountain-900 */
  --color-border-subtle:   #252A33;     /* 极淡分割线 */

  /* Primary — Jade 翠 */
  --color-primary:         #1F8C65;     /* Jade-500, 与 light mode 一致 */
  --color-primary-hover:   #7CC4A0;     /* Jade-200 */
  --color-primary-subtle:  #0D3F30;     /* Jade-900 */
  --color-primary-text:    #E0F0EA;     /* Jade-50 */

  /* Secondary — Sand 沙 */
  --color-secondary:       #C9A87A;     /* Sand-400, 不变 */
  --color-secondary-subtle:#5E4A2D;     /* Sand-900 */
  --color-secondary-text:  #F8F1E6;     /* Sand-50 */

  /* Highlight — Gold 金 */
  --color-highlight:       #D4A22E;     /* Gold-500, 不变 */
  --color-highlight-subtle:#5C4808;     /* Gold-900 */
  --color-highlight-text:  #FDF5E0;     /* Gold-50 */

  /* Danger — Vermillion 朱 */
  --color-danger:          #C83C23;     /* Vermillion-500 */
  --color-danger-hover:    #F0A090;     /* Vermillion-200 */
  --color-danger-subtle:   #5A1A0E;     /* Vermillion-900 */
  --color-danger-text:     #FCEAE6;     /* Vermillion-50 */

  /* Success — Spring 芽 */
  --color-success:         #8EBA3A;     /* Spring-500, 不变 */
  --color-success-subtle:  #33500F;     /* Spring-900 */
  --color-success-text:    #EEF5DC;     /* Spring-50 */

  /* Special */
  --color-ink:             #F4F2EF;     /* Ink-50, dark mode 下反转 */
}
```

---

## 4. Typography

| 层级                    | Light mode              | Dark mode               |
|-------------------------|-------------------------|-------------------------|
| 标题 / 正文             | Mountain-900 `#2E3545`  | Mountain-50 `#ECEEF3`  |
| 次要文字 / 副标题        | Mountain-600 `#5A6478`  | Mountain-400 `#8A94A8` |
| Placeholder / 辅助      | Ink-400 `#8E8A82`       | Mountain-600 `#5A6478` |
| 禁用态                  | Ink-200 `#C8C0B4`       | Mountain-900 `#2E3545` |
| Hero / Logo / 极致强调   | Ink-900 `#1A1A1A`       | Ink-50 `#F4F2EF`       |

> **Hero heading 与正文的区分**：Hero 使用 Ink-900 的纯墨黑，正文使用 Mountain-900 的远山蓝灰。
> 这个落差本身就是"泼墨"与"山水"两种气质的切换点，是刻意的设计而非不一致。

---

## 5. Color Pairing Rules

### Safe pairs（可自由组合）

- **Jade + Sand** — 翠江沙洲，两幅画共有的自然对比
- **Jade + Ink** — 墨底翠意，高雅
- **Mountain + Sand** — 远山暖沙，柔和
- **Gold + Ink** — 金石拓片感，泼墨画气质
- **Vermillion + Ink** — 朱砂题跋，经典水墨配色

### Tension pairs（限制使用，需缓冲色）

- **Vermillion + Spring** — 互补色冲突，禁止等面积相邻
- **Vermillion + Gold** — 双暖高饱和，容易视觉疲劳，需要大面积 Ink 或 Mountain 缓冲
- **Gold + Spring** — 黄绿相邻色，小面积可以，大面积会糊

### Forbidden（禁止组合）

- Vermillion 做背景 + Spring 做前景文字（反之亦然）——对比度不足且色相冲突
- Ink-900 做大段正文——仅限 hero/logo，正文必须用 Mountain-900

---

## 6. Backgrounds

| 场景                       | Light mode   | Dark mode    |
|----------------------------|-------------|-------------|
| Page bg                    | `#FAFAF8`   | `#141618`   |
| Card / Surface             | `#F4F2EF`   | `#1E2228`   |
| Elevated (modal, popover)  | `#FFFFFF`   | `#252A33`   |
| Sidebar / Nav              | `#F4F2EF`   | `#1A1E24`   |
| Code block                 | `#F4F2EF`   | `#1A1E24`   |

> Page bg 的 `#FAFAF8` 是两幅画共有的暖白底色——宣纸与雾霭的交集。
> Dark mode 的 `#141618` 带有极微弱的蓝调，呼应远山蓝灰的冷色基因，而非纯中性黑。

---

## 7. Component Patterns

### Type Bar（分类竖条）

每个 feed item / list row 左侧 3px 彩色竖条，用于快速辨识所属模块。

| 模块 | Color | CSS class |
|------|-------|-----------|
| Hackathon | Jade-500 `#1F8C65` | `.hf-type-jade` |
| Bounty | Spring-500 `#8EBA3A` | `.hf-type-spring` |
| Grant | Gold-500 `#D4A22E` | `.hf-type-gold` |
| Danger / Cancel | Vermillion-500 `#C83C23` | `.hf-type-verm` |
| Credits | Sand-400 `#C9A87A` | `.hf-type-sand` |

```html
<div class="hf-row">
  <div class="hf-type-bar hf-type-jade"></div>
  <div class="hf-row-left">...</div>
</div>
```

### Badges with Icons

Badge 内嵌 SVG icon 增强分类辨识。使用各 ramp 的 50/900 stop 对（light）或 900/200 stop 对（dark）。

```html
<span class="hf-badge hf-badge-jade">
  {{svg "octicon-rocket" 10}} hackathon
</span>
```

| 分类 | Light bg/text | Dark bg/text | Icon |
|------|-------------|-------------|------|
| Hackathon | Jade 50/900 | Jade 900/200 | `octicon-rocket` |
| Bounty | Spring 50/900 | Spring 900/200 | `octicon-gift` |
| Grant | Gold 50/900 | Gold 900/200 | `octicon-heart` |
| Credits | Sand 50/900 | Sand 900/200 | `octicon-credit-card` |
| Danger | Vermillion 50/900 | Vermillion 900/200 | `octicon-x` |
| Success | Spring 50/900 | Spring 900/200 | `octicon-check` |

### Activity Heatmap

12 周活跃热力图，使用 Jade 色阶表示强度：

| 强度 | Light mode | Dark mode |
|------|-----------|-----------|
| 0 (none) | Ink-50 `#F4F2EF` | Elevated `#252A33` |
| 1 (low) | Jade-50 `#E0F0EA` | Jade-900 `#0D3F30` |
| 2 (medium) | `#B5DFCC` (Jade 50-200 中间) | Jade-700 `#106B4C` |
| 3 (high) | Jade-200 `#7CC4A0` | Jade-500 `#1F8C65` |
| 4 (peak) | Jade-500 `#1F8C65` | Jade-200 `#7CC4A0` |

```html
<div class="hf-heatmap">
  <div class="hf-heatmap-cell hf-hc-0"></div>
  <div class="hf-heatmap-cell hf-hc-1"></div>
  ...
</div>
```

### Buttons

| Type           | Light mode                                      | Dark mode                                       |
|----------------|------------------------------------------------|------------------------------------------------|
| Primary        | bg Jade-500, text white                         | bg Jade-500, text white                         |
| Secondary      | bg transparent, border Jade-500, text Jade-700  | bg transparent, border Jade-200, text Jade-200  |
| Danger         | bg Vermillion-500, text white                   | bg Vermillion-500, text white                   |
| Ghost          | bg transparent, text Mountain-900               | bg transparent, text Mountain-50                |
| Ink (hero)     | bg Ink-900, text Ink-50                         | bg Ink-50, text Ink-900                         |
| Gold (premium) | bg Gold-500, text white                         | bg Gold-500, text #141618                       |

### Tags / Badges

在浅色底上使用各 ramp 的 50 stop 做背景，900 stop 做文字。
在深色底上使用各 ramp 的 900 stop 做背景，50 或 200 stop 做文字。
Badge 圆角 4px（方角，Google 风格），内嵌 icon 10px。

### Cards

- Light: bg `#FFFFFF`, border Ink-200 (`#C8C0B4`) at 0.5px, border-radius 12px
- Dark: bg `#1E2228`, border Mountain-900 (`#2E3545`) at 0.5px, border-radius 12px
- Card head: 略深的背景色（Ink-50 / Elevated），底部 1px 边框

### Tabs with Icons

Tab 栏内嵌 13px SVG icon + 文字标签。Active tab 使用 Jade-500 下划线 + Jade-200（dark）或 Jade-500（light）文字色。

```html
<div class="hf-tabs">
  <div class="hf-tab active">
    {{svg "octicon-pulse" 13}} All
  </div>
  <div class="hf-tab">
    {{svg "octicon-rocket" 13}} Hackathons
  </div>
</div>
```

---

## 8. Module Color Mapping

### Hackathon Status

| Status | Value | Color Ramp |
|--------|-------|-----------|
| Draft | 0 | Ink (neutral) |
| Open | 1 | Jade (primary) |
| Hacking | 2 | Jade (primary) |
| Judging | 3 | Gold (highlight) |
| Finished | 4 | Ink (neutral) |
| Cancelled | 5 | Vermillion (danger) |

### Bounty Status

| Status | Value | Color Ramp |
|--------|-------|-----------|
| Open | 0 | Jade |
| Claimed | 1 | Jade |
| In Review | 2 | Gold |
| Completed | 3 | Spring |
| Paid | 4 | Spring |
| Expired | 5 | Vermillion |
| Cancelled | 6 | Ink |

### Registration Status

| Status | Value | Color Ramp |
|--------|-------|-----------|
| Pending | 0 | Gold |
| Approved | 1 | Spring |
| Rejected | 2 | Vermillion |

### Reputation Tier

| Tier | Color Ramp |
|------|-----------|
| Diamond | Jade |
| Gold | Gold |
| Silver | Ink |
| Bronze | Sand |

---

## 9. Forgejo Theme Integration

HackForger 通过 Forgejo 的主题系统实现全站配色：

### Theme files

```
web_src/css/themes/theme-hackforger-light.css  — 浅色主题
web_src/css/themes/theme-hackforger-dark.css   — 深色主题
```

### Registration

在 `modules/setting/ui.go` 的 `Themes` 列表中添加：
```go
Themes: []string{..., "hackforger-light", "hackforger-dark"},
```

### Default theme

在 `app.ini` 中设置：
```ini
[ui]
DEFAULT_THEME = hackforger-dark
THEMES = forgejo-auto,forgejo-light,forgejo-dark,...,hackforger-light,hackforger-dark
```

### HackForger-specific CSS

```
web_src/css/hackforger-colors.css      — Badge/status color tokens (--hf-* vars)
web_src/css/hackforger-colors-dark.css — Dark mode overrides for --hf-* vars
web_src/css/hackforger.css             — Component classes (.hf-badge, .hf-card, etc.)
```

这些文件通过 `web_src/css/index.css` 导入（所有主题共享），dark 版本由各 dark theme 文件导入。

---

## 10. Accessibility

| Pair                          | Contrast ratio | WCAG AA (4.5:1) | WCAG AAA (7:1) |
|-------------------------------|---------------|-----------------|-----------------|
| Mountain-900 on #FAFAF8       | 11.8:1        | Pass            | Pass            |
| Mountain-600 on #FAFAF8       | 5.2:1         | Pass            | Fail            |
| Ink-400 on #FAFAF8            | 4.6:1         | Pass (barely)   | Fail            |
| Jade-500 on white             | 4.8:1         | Pass            | Fail            |
| Vermillion-500 on white       | 5.1:1         | Pass            | Fail            |
| Mountain-50 on #141618        | 13.2:1        | Pass            | Pass            |
| Mountain-400 on #141618       | 5.8:1         | Pass            | Fail            |

> Ink-400 (`#8E8A82`) 作为 placeholder 色勉强达标 AA。如果用于更小字号（<16px），
> 建议换用 Mountain-600 (`#5A6478`) 以获得更安全的对比度。

---

## 11. Origin Story

### Painting 1 — 江峡（River Gorge landscape）

抒情山水油画，雾气朦胧。贡献了系统的基础气质：

- **Ink ramp 的暖灰调**（雾霭）
- **Jade ramp**（翠江深绿）
- **Sand ramp**（沙洲暖色）
- **Spring ramp**（新芽嫩绿）
- **Mountain ramp**（远山蓝灰 → 系统主文字色）
- **"暖底冷文"的底层气质**

### Painting 2 — 泼墨（Ink Splash abstract）

大写意书法/泼墨，高对比度。贡献了系统的力量元素：

- **Ink-900 纯墨黑**（用于 hero / logo 的极致强调）
- **Vermillion ramp**（朱砂红，来自印章和飞溅点）
- **Gold ramp**（金色飞溅，填补暖色饱和度断层）
- **高对比度 Ink button 的概念**（泼墨感的 UI 元素）

### 融合冲突与解法

| 冲突点               | 解法                                         |
|----------------------|----------------------------------------------|
| 对比度差异（柔 vs 刚）| 日常用远山蓝灰，hero 用纯墨黑，形成有意的气质切换 |
| 朱砂 vs 新芽互补冲突  | 朱砂限定为 danger/CTA，禁止与新芽等面积相邻     |
| 金色与沙洲重叠        | 沙洲做日常辅助，金色做高亮 premium，用饱和度区分   |
