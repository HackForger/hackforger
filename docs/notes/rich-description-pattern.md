# HackForger 富文本描述模式

## 概述

HackForger 所有实体的描述字段使用 Forgejo 原生 `ComboMarkdownEditor` 组件，支持 Markdown 编辑、实时预览、图片/文件拖拽上传。描述在详情页以 Markdown 渲染为 HTML 显示。

## 架构

```
创建/编辑表单                    详情/管理页面
┌─────────────────────┐        ┌──────────────────┐
│ ComboMarkdownEditor │        │ markdown.Render  │
│ ┌─撰写─┬─预览──┐    │        │ String()         │
│ │ H B I <> 🔗 │    │        │                  │
│ ├──────────────┤    │   DB   │ <div class=      │
│ │ ## Title     │────┼───→────│  "markup">       │
│ │ - item       │    │  TEXT  │  <h2>Title</h2>  │
│ │ ![img](url)  │    │        │  <ul><li>...     │
│ └──────────────┘    │        │  <img src=...    │
│ ┌拖放文件或点击上传──┐ │        │ </div>           │
│ └──────────────────┘ │        └──────────────────┘
└─────────────────────┘
         │
         ▼ POST /hackforger/attachments
  ┌──────────────┐
  │ 文件存储      │ RepoID = -1
  │ (Attachment)  │ (平台级附件)
  └──────────────┘
```

## 涉及的组件

### 1. 模板层

**编辑器（创建/编辑表单）：**
```html
{{template "shared/combomarkdowneditor" (dict
    "MarkdownPreviewUrl" (print AppSubUrl "/hackforger/markup")
    "MarkdownPreviewContext" ""
    "TextareaName" "description"
    "TextareaContent" .Entity.Description
    "TextareaPlaceholder" (ctx.Locale.TrString "...")
    "DropzoneParentContainer" "form"
)}}
{{if .IsAttachmentEnabled}}
<div class="field">{{template "repo/upload" .}}</div>
{{end}}
```

**渲染（详情/管理页面）：**
```html
{{if .DescriptionHTML}}
<div class="markup">{{.DescriptionHTML}}</div>
{{else if .Entity.Description}}
<p>{{.Entity.Description}}</p>
{{end}}
```

### 2. Go handler 层

**创建/编辑页面 GET — 设置附件上传参数：**
```go
ctx.Data["IsAttachmentEnabled"] = setting.Attachment.Enabled
ctx.Data["AttachmentAllowedTypes"] = setting.Attachment.AllowedTypes
ctx.Data["AttachmentMaxSize"] = setting.Attachment.MaxSize
ctx.Data["AttachmentMaxFiles"] = setting.Attachment.MaxFiles
ctx.Data["UploadUrl"] = setting.AppSubURL + "/hackforger/attachments"
ctx.Data["UploadRemoveUrl"] = ""
ctx.Data["UploadLinkUrl"] = ""
ctx.Data["UploadAccepts"] = strings.ReplaceAll(setting.Attachment.AllowedTypes, "|", ",")
ctx.Data["UploadMaxFiles"] = setting.Attachment.MaxFiles
ctx.Data["UploadMaxSize"] = setting.Attachment.MaxSize
```

**详情页面 GET — Markdown 渲染：**
```go
if entity.Description != "" {
    rendered, err := markdown.RenderString(&markup.RenderContext{Ctx: ctx}, entity.Description)
    if err == nil {
        ctx.Data["DescriptionHTML"] = rendered
    }
}
```

### 3. 路由

| 路由 | 用途 |
|------|------|
| `POST /hackforger/attachments` | 附件上传（RepoID=-1 平台级） |
| `POST /hackforger/markup` | Markdown 预览渲染（session auth） |

### 4. JS 初始化

`web_src/js/features/hackforger/hackathon-editor.js` — 查找 `.page-content.hackathon .combo-markdown-editor` 并调用 `initComboMarkdownEditor()`。在 `hackforger/init.js` 中注册。

**重要：** 每个使用 ComboMarkdownEditor 的页面必须有 JS 初始化。Forgejo 不会自动初始化。

### 5. 附件存储

- 使用 `RepoID = -1` 标识 HackForger 平台级附件
- Forgejo 的 `NewAttachment()` 只拒绝 `RepoID == 0`，`-1` 可通过
- 上传文件存储在 Forgejo 标准附件目录中

## 管理页面的 Preview/Edit 模式

管理页面默认显示渲染后的描述（预览模式），点击"编辑"按钮切换到编辑表单。通过简单的 DOM 显示/隐藏实现：

```html
<div id="entity-details-preview">
    <!-- 渲染后的 Markdown + 元信息 -->
    <button onclick="切换到编辑">编辑</button>
</div>
<div id="entity-details-edit" style="display:none">
    <!-- ComboMarkdownEditor 表单 -->
    <button onclick="切换到预览">取消</button>
</div>
```

## 需要应用此模式的实体

| 实体 | 描述字段 | 创建页面 | 详情页面 | 管理页面 |
|------|---------|---------|---------|---------|
| Hackathon | Description, PrizeSummary | `/hackathons/new` | `/hackathon/{slug}` | `/hackathon/{slug}/manage` |
| Bounty | Description (title) | `/bounties/new` | Issue 页面 | — |
| Grant Round | Description | `/grants/new` | `/grants/{slug}` | `/grants/{slug}/manage` |
| Grant Project | Description | `/grants/{slug}/submit` | — | — |
| Hackathon Submission | Description | `/hackathon/{slug}/submit` | — | — |
| Hackathon Track | Description | manage 页面内 | — | — |

## Checklist（新增描述页面时）

1. [ ] 模板中使用 `shared/combomarkdowneditor` 而非 `<textarea>`
2. [ ] Go handler 设置 `UploadUrl`, `UploadMaxFiles`, `UploadMaxSize`, `UploadAccepts`, `IsAttachmentEnabled`
3. [ ] 详情页 handler 调用 `markdown.RenderString()` 生成 `DescriptionHTML`
4. [ ] 详情页模板使用 `<div class="markup">{{.DescriptionHTML}}</div>`
5. [ ] JS 初始化：在 `hackforger/init.js` 或对应 JS 中调用 `initComboMarkdownEditor()`
6. [ ] 确认 `.page-content` 的 CSS class 和 JS 选择器匹配
