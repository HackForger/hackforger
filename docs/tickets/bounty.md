# Bounty

## Bugs

### HF-015: 创建悬赏不应允许修改关联的 Issue
- **Severity**: Medium
- **Source**: E2E testing
- **Description**: 创建 Bounty 后，关联的 Issue 内容不应允许被直接修改（或至少 Bounty 关键字段不应被覆盖）。需要明确 Bounty 和 Issue 的编辑权限边界。

## Feature Requests

### HF-016: Bounty 状态变更写入 Issue Feed
- **Priority**: High
- **Source**: E2E testing
- **Description**: Bounty Applied、Accept、Complete 等状态变更应写入 Issue 的 Timeline/Feed 中，并显著标明状态变化（如"Bounty Created"、"Bounty Accepted"等），让关注 Issue 的人能看到 Bounty 进展。

### HF-017: 添加用户时应提供选择框
- **Priority**: Medium
- **Source**: GitHub #17
- **GitHub Issue**: HackForger/hackforger#17
- **Description**: 在 Bounty/Hackathon 中添加用户（如指派评委、参赛者）时，应提供用户搜索下拉选择框，而非直接输入用户名。当前直接输入如果用户不存在，没有报错但添加失败。
