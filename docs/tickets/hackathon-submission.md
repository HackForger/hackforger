# Hackathon -- Submission & Judging

## Bugs

### HF-008: 提交时赛道下拉框显示不全
- **Severity**: Medium
- **Source**: E2E testing + GitHub #19
- **GitHub Issue**: HackForger/hackforger#19
- **Description**: 提交作品和评委分配时，赛道（Track）下拉框显示不完整，内容被截断。
- **Expected**: 下拉框宽度自适应或支持滚动，完整显示赛道名称。

### HF-009: 评分提交没有反应
- **Severity**: Critical
- **Source**: E2E testing
- **Description**: 评审时点击"提交评分"按钮没有反应，评分无法保存。

### HF-010: 预览结果没有显示
- **Severity**: High
- **Source**: E2E testing
- **Description**: 评审完成后"预览结果"功能不显示任何内容。

## Feature Requests

### HF-011: 同一赛道不允许重复提交
- **Priority**: Medium
- **Source**: E2E testing
- **Description**: 同一个参赛者在同一个赛道内不应该可以重复提交作品。后端需要添加唯一性校验。

### HF-012: 评审界面信息不够丰富
- **Priority**: Medium
- **Source**: E2E testing
- **Description**: 评审时应展示更多信息，包括提交者（创建人）信息和指向 repo 的链接，方便评审查看代码。

### HF-013: 评审报名应校验身份
- **Priority**: Medium
- **Source**: E2E testing
- **Description**: 评审报名（Judge signup）应该有身份校验，当前可能会报错或静默失败。

### HF-014: 创建重复活动应弹出提示
- **Priority**: Low
- **Source**: E2E testing
- **Description**: 如果创建的 Hackathon 名称与已有活动重复，应弹出确认提示而非静默创建。
