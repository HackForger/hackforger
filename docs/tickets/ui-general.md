# UI & General

## Bugs

### HF-025: JavaScript error -- Cannot read properties of null
- **Severity**: High
- **Source**: E2E testing
- **Description**: 页面出现 JS 错误：`Cannot read properties of null (reading 'style')` (index.js @ 142:18917)。需要排查是哪个组件在 DOM 元素不存在时尝试访问 `.style` 属性。
- **Likely cause**: HackForger 页面中某个 Vue 组件或 JS 初始化代码在 DOM 未就绪或元素被条件隐藏时执行。

### HF-026: "显示名称(可选)"字段应删除
- **Severity**: Low
- **Source**: E2E testing
- **Description**: 创建/编辑表单中的"显示名称(可选——默认使用用户名或组织名）"字段应该删除，这个字段多余且造成困惑。

## Feature Requests

### HF-017: 添加用户时应提供搜索选择框
- **(Cross-reference: see bounty.md HF-017)**
- **Description**: 全局性问题，所有需要输入用户名的地方都应提供搜索下拉选择组件。

## Infrastructure

### HF-027: Workflow auto-merge token 权限问题
- **Severity**: Medium (有临时方案)
- **Source**: GitHub #10 (已关闭)
- **GitHub Issue**: HackForger/hackforger#10
- **Description**: Action token 无法通过 Forgejo 内部 hook 的 merge API。当前使用 git push 直接推送作为临时方案。长期方案为使用 repo secret 传 PAT。
- **Status**: 已关闭，有临时方案运行中。
