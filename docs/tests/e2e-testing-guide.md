# HackForger E2E Testing Guide

## Environment Setup

### Instance Access

| Method | URL | Use When |
|--------|-----|----------|
| **localhost (推荐)** | `http://localhost:3000` | agent-browser 自动化测试 |
| **HTTPS** | `https://hackforger.inside.h2os.cloud` | 浏览器手动测试 |

**为什么用 localhost：** 本机代理 (127.0.0.1:7897) 拦截 HTTPS 到 Tailscale IP 的流量导致 TLS 握手失败。agent-browser (Chromium) 无法绕过。直连 localhost:3000 无此问题。

**CSRF 安全性：** Forgejo 使用 `CrossOriginProtection`（基于 Origin/Referer 头），不使用 CSRF token。localhost 请求的 Origin 和服务器一致，不会被拦截。

### Test Accounts

| 角色 | 用户名 | 密码 | 用途 |
|------|--------|------|------|
| Admin/Organizer | `hackforger` | `admin1234` | 创建 hackathon、管理 |
| Judge 1 | `judge_carol` | `admin1234` | 评审（需先通过 admin API 重置密码） |
| Judge 2 | `judge_dave` | `admin1234` | 评审 |
| Hacker 1 | `hacker_eve` | `admin1234` | 参赛（需先通过 admin API 重置密码） |
| Hacker 2 | `hacker_frank` | `admin1234` | 参赛 |

**密码重置：** 测试用户密码可能不是 `admin1234`，测试前通过 admin API 重置：
```bash
TOKEN="<admin-api-token>"
curl -s -X PATCH "http://localhost:3000/api/v1/admin/users/<username>" \
  -H "Authorization: token $TOKEN" -H "Content-Type: application/json" \
  -d '{"password":"admin1234","must_change_password":false}'
```

### Database Cleanup

每轮测试前清空 hackathon 数据：
```bash
sqlite3 /Users/h2oslabs/Workspace/hackforger/data/forgejo.db "
DELETE FROM hackathon_judge_score;
DELETE FROM hackathon_judge_criteria;
DELETE FROM hackathon_track_criteria;
DELETE FROM hackathon_judge;
DELETE FROM hackathon_submission;
DELETE FROM hackathon_registration;
DELETE FROM hackathon_track;
DELETE FROM hackathon;
DELETE FROM action_run; DELETE FROM action_run_job;
DELETE FROM action_task; DELETE FROM action_task_step;
"
```

### Build & Start

```bash
# In worktree directory
mkdir -p custom/conf
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini
make frontend
TAGS="bindata sqlite sqlite_unlock_notify" make build
kill $(lsof -t -i :3000) 2>/dev/null; sleep 2
rm -f /Users/h2oslabs/Workspace/hackforger/data/queues/common/LOCK
./gitea web
```

### Forgejo Actions Runner

Runner 配置在 `~/.config/forgejo-runner/`，由 launchd 管理：
```bash
launchctl list com.h2os.forgejo-runner  # 检查状态
cat ~/.config/forgejo-runner/runner.log  # 查看日志
```

Runner 使用 host 模式（非 Docker），标签：`ubuntu-latest:host`, `macos-arm64:host`。

## Testing Approach

### Web-First 原则

E2E 测试应以 Web 界面操作为核心，因为大部分用户通过 UI 交互：

1. **核心流程用 agent-browser** — 登录、表单提交、Vue 组件交互、页面导航
2. **批量操作可用 API** — 创建多个 criteria、批量评分等重复性操作
3. **通过 browser eval 调用 fetch** — Web-only 端点（如评分 POST）需要 session auth，用 `agent-browser eval` 执行 `fetch()` 调用

### 多用户并行 Session

使用 `--session` 为每个测试用户创建独立浏览器 session，避免反复登录/登出：

```bash
# 一次性登录所有用户（每个用户一个 session）
agent-browser --session admin open "http://localhost:3000/user/login"
agent-browser --session admin fill @e13 "hackforger" && fill @e14 "admin1234" && click @e17

agent-browser --session judge_carol open "http://localhost:3000/user/login"
agent-browser --session judge_carol fill @e13 "judge_carol" && fill @e14 "admin1234" && click @e17

# 之后直接用 session 名操作，无需再登录
agent-browser --session admin open "http://localhost:3000/hackathon/my-hack/manage"
agent-browser --session judge_carol open "http://localhost:3000/hackathon/my-hack/judge"
```

**`--session` vs `--profile`：**
- `--session` — 命名 session，在 daemon 生命周期内保持独立 cookie/state。适合并行多用户测试。
- `--profile` — 持久化浏览器 profile（磁盘保存）。需要在 daemon 启动时指定，无法中途切换。适合长期复用。
- **E2E 测试推荐用 `--session`**，因为需要同时操作多个用户。

### 常用操作模式

```bash
# 页面验证 + 截图
agent-browser --session admin screenshot /tmp/hackforger-e2e-tc01.png

# Vue 组件交互（tab 切换 — Vue 渲染的元素不在 accessibility tree，用 find text）
agent-browser --session judge_carol find text "Web Track" click

# Session-auth API 调用（通过浏览器 fetch）
agent-browser --session judge_carol eval --stdin <<'EVALEOF'
(async () => {
  const resp = await fetch('/hackathon/slug/judge/123/scores', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({scores: [{criteria_id: 1, score: 8.5}]})
  });
  return JSON.stringify(await resp.json());
})()
EVALEOF
```

### Screenshots

每个 TC 的关键验证点必须截图：
- 保存到 `/tmp/hackforger-e2e-<tc>-<step>.png`
- 在 E2E report 中用 markdown 引用
- 截图内容应能独立证明测试结果

### Known Issues

1. **agent-browser session 断开** — 长时间测试后 Chromium session 可能失效，需要 `agent-browser close --all` 重新开始
2. **Vue tab 不在 accessibility tree** — Vue 渲染的 tab 需要用 `agent-browser find text "Tab Name" click` 而不是 ref
3. **Web POST 需要 session auth** — 评分等 web 端点不接受 API token，需要通过 browser fetch 或 agent-browser 的已登录 session
