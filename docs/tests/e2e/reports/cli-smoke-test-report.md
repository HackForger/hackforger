# hackforger-cli Smoke Test Report

> **日期**: 2026-04-02
> **服务器**: http://localhost:3000
> **CLI 版本**: hackforger-cli (built from worktree)
> **测试数据**: E2E full-cycle 运行后的数据（hackathon, grant, credits, reputation）

## 概要

| 结果 | 数量 |
|------|------|
| PASS | 16 |
| FAIL | 0 |
| **总计** | **16** |

## 测试结果

### TC-CLI-01: --help 输出 — PASS

```bash
$ ./hackforger-cli --help
```

**输出**: 列出 8 个子命令（hackathon, bounty, grant, credits, feed, reputation, search, assistant）+ 全局 flags（--url, --token, --output）

---

### TC-CLI-02: hackathon list — PASS

```bash
$ ./hackforger-cli --url http://localhost:3000 --token $TOKEN hackathon list
```

**输出**: JSON 数组，包含 "Web3 Innovation Challenge"
```json
[
  {
    "ID": 58,
    "Name": "Web3 Innovation Challenge",
    "Slug": "web3-innovation",
    "Status": 3,
    "Description": "## Web3 Innovation Challenge\n\nBuild the **future** of decentralized web...",
    "RegistrationStart": 1775782800,
    "RegistrationEnd": 1776700740,
    "HackingStart": 1776733200,
    "HackingEnd": 1777996740,
    "JudgingEnd": 1778860740
  }
]
```

---

### TC-CLI-03: hackathon get 58 — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN hackathon get 58
```

**输出**: 完整 hackathon JSON，含 Description, PrizeSummary, 所有日期字段

---

### TC-CLI-04: bounty list — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN bounty list
```

**输出**: `[]`（当前无 bounty，格式正确）

---

### TC-CLI-05: grant round-list — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN grant round-list
```

**输出**: JSON 数组，包含 grant round 记录

---

### TC-CLI-06: grant round-get — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN grant round-get 10
```

**输出**: 完整 grant round JSON，含 Budget, BudgetCredits, Status

---

### TC-CLI-07: credits balance — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN credits balance
```

**输出**:
```json
{
  "balance": 800
}
```

---

### TC-CLI-08: credits transactions — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN credits transactions
```

**输出**: 交易记录数组，每项含 Amount, Balance, Type, Reference
```json
[
  {
    "Amount": -200,
    "Balance": 800,
    "Reference": "GPU 算力 - 100 小时",
    "Type": "redeem"
  },
  {
    "Amount": 1000,
    "Balance": 1000,
    "Type": "admin_deposit"
  }
]
```

---

### TC-CLI-09: reputation leaderboard — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN reputation leaderboard
```

**输出**: 按 score 降序排列的用户列表
```json
[
  {"user_id": 1, "username": "hackforger", "score": 100, "tier": "Silver"},
  {"user_id": 3, "username": "hacker_eve", "score": 1, "tier": "Bronze"},
  {"user_id": 4, "username": "hacker_frank", "score": 1, "tier": "Bronze"}
]
```

---

### TC-CLI-10: reputation get hackforger — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN reputation get hackforger
```

**输出**: 用户声誉详情
```json
{
  "score": 100,
  "tier": "Silver",
  "bounties_completed": 0,
  "hackathon_wins": 0,
  "grants_received": 0,
  "total_stars": 0,
  "credits_earned": 1000
}
```

---

### TC-CLI-11: search query — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN search query --q "innovation"
```

**输出**: 分组搜索结果，hackathons 组包含 "Web3 Innovation Challenge"
```json
{
  "groups": [
    {
      "key": "hackathons",
      "items": [
        {
          "title": "Web3 Innovation Challenge",
          "status": "judging",
          "desc": "Web3 Innovation Challenge Build the future of decentralized web..."
        }
      ]
    }
  ]
}
```

---

### TC-CLI-12: search query --scope — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN search query --q "defi" --scope "grants"
```

**输出**: 返回搜索结果（scope 参数传递到 API）

---

### TC-CLI-13: feed list — PASS

```bash
$ ./hackforger-cli --url $URL --token $TOKEN feed list
```

**输出**: Feed 事件列表，每项含 actor, action_type, content
```json
{
  "items": [
    {
      "actor": {"id": 1, "username": "hackforger"},
      "created_at": 1775128535,
      "action_type": 39
    }
  ]
}
```

---

### TC-CLI-14: 环境变量认证 — PASS

```bash
$ HACKFORGER_URL=http://localhost:3000 HACKFORGER_TOKEN=$TOKEN ./hackforger-cli hackathon list
```

**输出**: 与 TC-CLI-02 相同，环境变量替代 --url/--token flags

---

### TC-CLI-15: 无 token — PASS

```bash
$ ./hackforger-cli --url http://localhost:3000 hackathon list
```

**输出**: 返回公开数据（hackathon list 不要求认证），无崩溃

---

### TC-CLI-16: 无效 URL — PASS

```bash
$ ./hackforger-cli --url http://nonexistent:9999 --token fake hackathon list
```

**输出**: `Error: HTTP 502:`（通过代理返回 502，程序正常退出，无 panic）

---

## 结论

- **16/16 PASS** — 所有 CLI 子命令正常工作
- JSON 输出格式一致，支持 jq 管道处理
- 环境变量和 flags 两种认证方式均可用
- 错误处理正常（无 token 返回公开数据，无效 URL 优雅报错）
- search 返回正确的分组结果，包含 HackForger 实体（hackathon）
