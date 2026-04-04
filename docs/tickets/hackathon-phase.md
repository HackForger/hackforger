# Hackathon -- Phase Control

## Bugs

### HF-001: Phase Control UI 没有翻译
- **Severity**: Medium
- **Source**: E2E testing
- **Description**: Phase Control 界面缺少 i18n 翻译，界面文字显示为英文或 key 值。

### HF-002: 没有赛道时无法发布，但无错误提示
- **Severity**: High
- **Source**: E2E testing
- **Description**: 创建 Hackathon 后如果未添加赛道（Track），点击发布时操作失败但没有任何提示信息。用户不知道为什么无法发布。
- **Expected**: 发布前校验赛道是否存在，不满足时显示明确的错误提示。

### HF-003: Phase 时间过了仍可修改
- **Severity**: High
- **Source**: E2E testing
- **Description**: Phase Control 需要独立的时间控制逻辑：Phase 自动按设定时间推进，milestone 未到时允许修改时间，过了的 Phase 不允许再修改。当前没有这个限制。
- **Expected**: 已过的 Phase 时间锁定为只读。

### HF-004: Cancel 活动没有确认对话框
- **Severity**: Medium
- **Source**: E2E testing
- **Description**: 取消 Hackathon 时没有弹出确认对话框，误操作风险高。

## Feature Requests

### HF-005: Phase 应自动按时间转换
- **Priority**: High
- **Source**: E2E testing
- **Description**: Phase Control 应根据预设时间自动切换状态。当前需要手动触发。可通过修改时间来调整 Phase 转换点。

### HF-006: 支持自定义 Phase 阶段
- **Priority**: Medium
- **Source**: E2E testing
- **Description**: 当前 Hackathon 时间阶段应允许增加更多阶段（阶段1、阶段2...），状态区别主要是区分"是否可以提交"。

### HF-007: 禁止组织者报名自己发布的活动
- **Priority**: High
- **Source**: E2E testing
- **Description**: 创建者（组织者）不应该可以报名自己发布的 Hackathon，需要在后端添加校验。
