# hackforger-cli Smoke Test

> **前置条件**: HackForger 实例运行中，数据库中有测试数据（hackathon, grant, credits, reputation, feed events）。

## 测试环境

- **CLI 路径**: `./hackforger-cli`（worktree 根目录构建）
- **服务器**: `http://localhost:3000`
- **认证**: 通过 `--token` 传入 API token

## 测试账号

| 角色 | 用户名 | Token 环境变量 |
|------|--------|---------------|
| Admin | hackforger | ADMIN_TOKEN |
| Hacker | hacker_eve | EVE_TOKEN |

## 构建

```bash
make hackforger-cli
```

---

## 测试用例

### TC-CLI-01: --help 输出

```bash
./hackforger-cli --help
```

**验证**:
- 列出所有子命令：hackathon, bounty, grant, credits, feed, reputation, search, assistant
- 显示全局 flags：--url, --token, --output

### TC-CLI-02: hackathon list

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN hackathon list
```

**验证**:
- 返回 JSON 数组
- 包含 "Web3 Innovation Challenge"
- 每项有 Name, Slug, Status, Description 字段

### TC-CLI-03: hackathon get

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN hackathon get <hackathon_id>
```

**验证**:
- 返回单个 hackathon 的完整 JSON
- 包含 tracks、criteria、registration 统计

### TC-CLI-04: bounty list

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN bounty list
```

**验证**:
- 返回 JSON 数组（可能为空或包含测试 bounty）
- 格式正确，无错误

### TC-CLI-05: grant round-list

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN grant round-list
```

**验证**:
- 返回 JSON 数组
- 包含 "DeFi Track Boost"（如有）

### TC-CLI-06: grant round-get

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN grant round-get <round_id>
```

**验证**:
- 返回单个 grant round 的完整 JSON

### TC-CLI-07: credits balance

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN credits balance
```

**验证**:
- 返回 `{"balance": <number>}`
- 余额与 Web UI 一致

### TC-CLI-08: credits transactions

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN credits transactions
```

**验证**:
- 返回交易记录数组
- 每项有 amount, type, source 字段

### TC-CLI-09: reputation leaderboard

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN reputation leaderboard
```

**验证**:
- 返回用户列表，按 score 降序
- 每项有 user_id, score, tier

### TC-CLI-10: reputation get

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN reputation get hackforger
```

**验证**:
- 返回指定用户的声誉详情
- 包含 score, tier, bounties_completed, hackathon_wins 等

### TC-CLI-11: search query

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN search query --q "innovation"
```

**验证**:
- 返回分组搜索结果
- hackathons 组包含 "Web3 Innovation Challenge"

### TC-CLI-12: search query (scope filter)

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN search query --q "defi" --scope "grants"
```

**验证**:
- 仅返回 grants 相关结果

### TC-CLI-13: feed list

```bash
./hackforger-cli --url $URL --token $ADMIN_TOKEN feed list
```

**验证**:
- 返回 Feed 事件列表
- 每项有 actor, action_type, content

### TC-CLI-14: 环境变量认证

```bash
HACKFORGER_URL=http://localhost:3000 HACKFORGER_TOKEN=$ADMIN_TOKEN ./hackforger-cli hackathon list
```

**验证**:
- 不使用 --url 和 --token flags 也能正常工作

### TC-CLI-15: 错误处理 — 无 token

```bash
./hackforger-cli --url $URL hackathon list
```

**验证**:
- 返回认证错误，不崩溃

### TC-CLI-16: 错误处理 — 无效 URL

```bash
./hackforger-cli --url http://nonexistent:9999 --token fake hackathon list
```

**验证**:
- 返回连接错误，不崩溃

---

## 报告模板

| 用例 | 状态 | 输出摘要 |
|------|------|---------|
| TC-CLI-01 | PASS/FAIL | |
| TC-CLI-02 | PASS/FAIL | |
| ... | | |
